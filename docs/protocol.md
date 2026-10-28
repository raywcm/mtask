# 本地 Broker 协议 v1

更新：2026-10-27。帧、握手和连接处理均已实现。初始草案保留在 [A0 存档](archive/protocol-a0.md)。

## 帧格式

```text
0       2     ASCII "MT"
2       1     version = 1
3       1     kind
4       4     payload_length，uint32 big endian
8       N     payload
```

Input/Output 载荷为原始字节，最多 64 KiB。其他载荷为 UTF-8 JSON 或空字节，最多 8 KiB。解码先验证长度再分配；完整帧的编码用同一写锁，支持短写。非法魔数/版本/类型、超限和截断使连接关闭，不扫描重新同步。写入出错后不能重试半帧。

## 消息

| kind | 名称 | 方向 | 载荷 |
| --- | --- | --- | --- |
| 1 | Hello | C → B | task_id、instance_id、mode、可选 size/attach_mode/capabilities |
| 2 | Welcome | B → C | task_id、instance_id、replay；可选 attach_mode/size_owner/size/capabilities |
| 3 | Input | C → B | 原始输入，仅仍持有控制权的 attach |
| 4 | Output | B → C | 原始 PTY 输出，仅 attach |
| 5 | Resize | C → B | rows/cols，非零 uint16，仅尺寸主控生效 |
| 6 | Detach | C → B | 空，释放连接 |
| 7 | Exit | B → C | code、可选 signal |
| 8 | Error | B → C | code、message；随后关闭 |
| 9 | Ping | C → B | 空 |
| 10 | Pong | B → C | 当前 Runtime JSON |
| 11 | Stop | C → B | 空，幂等停止请求，只读 attach 不可用 |
| 12 | Notice | B → C | type: role/refresh，role 带 size_owner；仅协商后的新客户端 |

控制 JSON 允许新增未知字段以兼容扩展；拒绝第二个 JSON 值。消息必需字段、身份、模式和尺寸另外验证。应用输出绝不进入 JSON 字符串。

## 握手与状态机

1. 接受连接后验证 peer UID 与当前 UID 相同；客户端也验证服务端 UID。
2. 首帧必须在 2 秒内为有效 Hello。task_id 和 instance_id 必须匹配 Broker，避免旧快照误连新实例。
3. mode 为 control 或 attach。attach 提供有效尺寸；attach_mode 缺省 exclusive，已有控制端时返回 busy。takeover 撤销其他全部控制端并保留只读端；readonly 只订阅输出；shared 加入输入控制。非默认方式必须声明 attach-modes-v1。control 不取得输入权。
4. 只有新尺寸主控会同步尺寸；Welcome 告知实际尺寸、主控归属与能力，再启动 Output/Notice 转发。新旁观端请求主控协作重绘。失败时释放该连接的租约。
5. 可输入 attach 发送 Input/Resize/Detach/Ping/Stop；非主控的合法 Resize 忽略。readonly 仅允许 Detach/Ping，Input/Resize/Stop 返回 read_only。control 只允许 Ping/Stop/Detach。其他类型、方向不合法和重复 Hello 返回 invalid_request。
6. Stop 将请求交给 supervisor 并返回 Pong。Pong 可能仍为 running，因为请求与状态转换并发；客户端继续查询或等待 Exit。
7. EOF 与 Detach 只释放客户端，不停止任务。mtask 前端的主动 detach 通过关闭连接完成，避免服务器 EOF 抢在本地 detach 结果之前。
8. 正常退出时先提交状态，再尽量排空输出并发送 Exit，关闭连接。没收到 Exit 的 EOF 代表连接丢失，不能推断应用成功结束。

每 Broker 总连接数上限 32；control 空闲读取上限 30 秒；写入截止时间 2 秒。慢 attach 输出队列溢出会主动关闭连接，不能为发送错误消息再无限等待。

错误码使用 busy、identity_mismatch、invalid_request、internal、slow_client、read_only、taken_over。客户端能力探测可返回本地错误 unsupported_attach_mode。帧版本错误可能直接关闭，客户端呈现协议/连接错误。错误文本不主动包含环境变量。

## 版本与恢复

协议版本和磁盘 schema 版本独立，当前均为 1。新的 spec.size、spec.capture_initial 和 record.health 字段不改变原有字段含义；旧任务缺少初始尺寸时使用 24×80。

重新连接的 replay 为 none：得到原进程后续输出，尺寸和 SIGWINCH 帮助支持的应用重绘。新建前台任务可在首次连接时使用 initial，交付最多 2 MiB 的启动字节，确保提示符或终端查询不会因启动握手而丢失；首次连接之后不再回放。客户端可在连接边界从日志提取白名单模式并生成有限的恢复控制；不把历史日志原样注入终端。全屏重绘使用现有 v1 Resize 帧，临时缩小后恢复尺寸，因此兼容已有 Broker。未实现远程 TCP、多输入客户端、自动历史 ANSI 回放或完整 VT 屏幕恢复。


新前端在请求非默认方式前，先建立 control 连接，检查 Welcome.capabilities 包含 attach-modes-v1 后关闭，再建立指定方式的 attach，并核对 Welcome.attach_mode 的回执。探测失败不尝试连接降级。旧 v1 前端的 Hello 没有该能力声明，新 Broker 不向它发送 Notice，保留原独占连接和帧行为。尺寸主控断开后按仍在线控制端的加入顺序转交，向新客户端发送 role；只读端从不自动提升为控制端。
