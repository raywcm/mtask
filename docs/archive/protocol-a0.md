# 本地 Broker 协议 v1 草案

A0 协议草案存档。该阶段仅实现帧编解码和载荷类型，Socket、握手及客户端尚待实现。现行定义见[协议文档](../protocol.md)。

## 传输层

每任务一个 Unix stream Socket，连接建立后先验证 peer UID，再读取 Hello。第一个帧有握手截止时间；未完成认证和握手前，不转发任何 PTY 数据。

```text
offset  size  field
0       2     ASCII "MT"
2       1     version = 1
3       1     kind
4       4     payload_length，uint32 big endian
8       N     payload
```

输入/输出载荷最多 64 KiB；其他类型最多 8 KiB。Hub 以最多 32 KiB 的块提供输出。stream 的任意分包/合包不影响语义，解码用 `io.ReadFull`；编码器处理短写。

魔数错误、未知版本、未知类型、超长或截断帧使连接失败。任何写入错误都要求关闭连接，因为已经发送的部分帧不能安全重试。无帧边界时不尝试扫描恢复。一个连接只有一个读取方，所有写方共享一个 Encoder。

## 消息表

| kind | 名称 | 方向 | 载荷/约束 |
| --- | --- | --- | --- |
| 1 | Hello | client → broker | JSON：task_id、instance_id、mode、可选 size |
| 2 | Welcome | broker → client | JSON：task_id、instance_id、replay |
| 3 | Input | client → broker | 原始输入字节；仅活动 attach 控制端 |
| 4 | Output | broker → client | 原始 PTY 输出字节；仅 attach |
| 5 | Resize | client → broker | JSON：rows、cols；均为非零 uint16；仅控制端 |
| 6 | Detach | client → broker | 空载荷，释放控制权并关闭连接 |
| 7 | Exit | broker → client | JSON：code、可选 signal；终止事件 |
| 8 | Error | broker → client | JSON：code、message；错误后关闭连接 |
| 9 | Ping | client → broker | 空载荷，查询 Broker |
| 10 | Pong | broker → client | JSON：当前 Runtime 快照 |
| 11 | Stop | client → broker | 空载荷，幂等请求受管理任务停止 |

JSON 采用 UTF-8；v1 允许未知对象字段以便添加可选字段，但必须拒绝第二个 JSON 值及错误的必要字段。该阶段的 codec 只校验帧头和大小；消息语义由后续处理器校验。

## 握手与连接状态机

Hello 的 `mode` 为 `attach` 或 `control`，不允许其他值。attach 必须包含有效 size，control 不取得终端控制权。匹配磁盘 runtime 的 instance_id 用于检测旧客户端误连到新实例；PID 不参与认证。

计划中的顺序：

1. 验证 UID、协议、任务身份、Broker 就绪状态。
2. attach 原子预留控制权；已有控制端则返回 `busy`。验证尺寸并调用 resize。
3. 先写 Welcome，再转发队列中的 Output，不能让应用输出出现在欢迎帧之前。握手中断须释放已预留的控制权。
4. Welcome 的 `replay` 在首版固定为 `none`，明确不提供历史屏幕恢复。
5. attach 允许 Input/Resize/Detach/Ping/Stop；control 只允许 Ping/Stop/Detach。其他帧、重复 Hello 或方向不合法均返回错误并关闭。
6. EOF 或 Detach 只释放客户端，不终止应用。Stop 请求由 supervisor 幂等处理，并返回 Pong 状态（通常是 stopping，可能已 exited）。客户端可继续等待 Exit；必要时重连 control 查询。
7. 正常任务退出：在有界时间内排空输出，发送 Exit，关闭连接。慢客户端不能阻碍 Broker 落盘退出状态或回收任务。

第二个客户端的 control 查询可以与 attach 并存；“单控制端”特指唯一终端输入所有者。listener 必须独立限制控制连接数，避免无界 goroutine。

## 错误码约定

计划使用 `busy`、`not_ready`、`not_found`、`identity_mismatch`、`protocol_mismatch`、`invalid_request`、`slow_client`、`internal`。权限失败直接关闭或返回最少信息。错误消息不包含环境变量、令牌或完整命令行。

对于慢客户端，通知 `slow_client` 是尽力而为；不能为发送这个错误继续阻塞。客户端遇到 EOF 而没有 Exit 时，结论是“连接结束，任务状态待查询”，不是“任务成功退出”。

## 兼容性与未定事项

- 帧版本不兼容时拒绝连接，不尝试误解码。
- 磁盘 `schema_version` 独立演进，不由帧版本隐式推断。
- 暂不引入压缩、加密、远程 TCP、断点偏移、帧 ACK 或多客户端广播。
- 快照恢复、日志读取、强制接管、额外观察者需单独扩展协议并测试。
- Unix credential API、Socket 路径长度检查、请求/写超时和协议状态机仍属 A1；当前只完成底层帧实现。
