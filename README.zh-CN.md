# mtask

轻量、独立的 Linux/macOS 终端任务管理器。交互程序在断开连接后继续运行，通过任务列表重新进入同一进程。

[English](README.md) · [完整使用指南](docs/usage.md) · [兼容性与限制](docs/compatibility.md)

每个任务由独立 PTY Broker 保持运行，提供完整 CLI 和 TUI。运行时只需单个二进制文件，不依赖 tmux、screen、zmx、数据库或 Go；正常连接时原样转发应用终端字节。

**当前为预发行版本 v0.4.3-dev，尚未提供公开下载或 APT 软件源。** 重连画面恢复存在限制，重要任务使用前请阅读兼容性说明。

![mtask 实际操作录制](docs/assets/demo.gif)

演示在临时状态目录运行 Shell，展示列表、进入/返回、监控、停止确认和输出历史；没有模拟 AI 应用，也不代表 SSH 重连验收。

## 构建与安装

在 Linux/macOS 准备 Go 1.26+、Git 和 make：

```sh
git clone https://github.com/raywcm/mtask.git
cd mtask
make build
./scripts/install.sh --prefix "$HOME/.local" ./bin/mtask
"$HOME/.local/bin/mtask" --version
```

仓库目前为私有，克隆需要 raywcm/mtask 的访问权限。已有本地源码时从 `make build` 开始。
如果 `~/.local/bin` 未加入 PATH，可以继续使用完整路径或 `./bin/mtask`。安装器原子替换程序，不重启已有任务。

## 第一个会话

```sh
mtask new --name work -- bash
# 在任务内按 Ctrl+B，再按 d，断开并保留任务
mtask
# 选择 work，按 Enter 回到同一进程
```

通过 SSH 使用时，应在远端主机运行 mtask。断线后重新登录同一主机和账号，再执行 `mtask`。服务器需允许后台用户进程存活；机器重启会终止进程。

列表中 ↑/↓ 选择，Enter 进入，n 新建 Shell，v 监控，s 停止确认，F2 重命名，Delete 删除已停止记录，Tab 切换标签，F1/? 查看帮助。**列表中的 Ctrl+C 退出界面；进入应用后 Ctrl+C 发送给应用。** 停止、删除和接管弹窗默认取消，左右键/Tab 选按钮，Enter 执行，y 可直接确认。

“已停止”按 Enter 查看输出历史，支持滚动、翻页和长行横向查看，Esc/q 返回。**已停止记录及日志默认一小时自动删除**；需要长期保留时提前关闭清理：

```sh
mtask settings set history-retention off
```

## CLI 与多终端

```sh
mtask new -d --name build -- make test
mtask list --json
mtask show build
mtask logs --follow build
mtask stop build
mtask list --history --json
mtask delete build

mtask attach work                # 默认独占
mtask attach work --read-only    # 只读观看
mtask attach work --shared       # 共享输入
mtask attach work --takeover     # 断开原控制端并接管
```

三个连接选项互斥，需要 0.3.0 及以后的 Broker。旧 Broker 继续运行并保留普通独占连接。共享模式最早加入且仍在线的控制端决定 PTY 尺寸。当前所有连接要求同一 Unix UID，没有跨系统账号授权。

设置支持简体中文、英语、日语、西班牙语、法语，以及主题、资源信息、默认标签和历史保留时间；新增语言只需补充语言包。详情见[设置与语言](docs/usage.md#设置与语言)。

## 兼容性边界

**重连恢复已知终端模式并请求全屏程序重绘，尚不恢复完整屏幕快照或宿主终端滚动历史。** 实际画面取决于应用是否支持重绘，以及所使用的终端扩展。

监控按右侧面板改变任务尺寸，离开后保留该尺寸；完整进入时使用完整终端尺寸。占用、失联或输出不可用时明确显示“无法监控”，不自动接管。监控主界面左右键不响应；弹窗、输入框保留方向键操作。

Broker 崩溃后的 PTY 无法由新进程接管，`recover` 只协调失联元数据。关机、OOM 和主机登录策略清理进程也可能结束任务。已实测和仅交叉编译的平台分别列在[兼容性表](docs/compatibility.md)，不以构建成功代替运行验收。

## 开发与验收

```sh
make check
make integration
python3 -m venv .venv
.venv/bin/pip install -r scripts/requirements-screen.txt
make acceptance PYTHON=.venv/bin/python
python3 scripts/third_party_notices.py --check
```

验收使用独立临时状态，不连接用户原有任务。具体范围见[稳定性验收](docs/stability.md)。CI 已接通 [GitHub Actions](https://github.com/raywcm/mtask/actions)，以实际运行结果为准；本地验收与托管 CI 分别记录。

默认状态目录为 `$XDG_STATE_HOME/mtask` 或 `~/.local/state/mtask`，可通过 `--state-dir` / `MTASK_STATE_DIR` 覆盖。目录 0700，元数据和日志 0600，Socket 双向检查 UID。日志只记录输出，保留两个约 8 MiB 文件；参数和输出可能包含业务信息，`logs --raw` 会输出原始终端控制序列。

## 更多文档

- [使用指南](docs/usage.md)、[安装与打包](docs/installation.md)
- [架构设计](docs/architecture.md)、[通信协议](docs/protocol.md)
- [语言包扩展](docs/i18n.md)、[开发日志](docs/development-log.md)

采用 [MIT 许可证](LICENSE)。依赖保留各自许可证，原文见[第三方声明](THIRD_PARTY_NOTICES.md)。
