# 本地构建、安装与打包

用户当前要求暂不公开发布。以下流程只创建本地文件；GitHub Actions 只运行检查并上传构建附件，不创建公开 Release、不上传 APT 源、不使用发布令牌。

## 单文件安装

```sh
make build
./scripts/install.sh --prefix "$HOME/.local" ./bin/mtask
```

指定 `/usr/local` 等系统目录时，由用户选择合适的权限运行安装命令。安装器不自动使用 sudo，不改 shell 配置，不修改或终止已有任务。

## GoReleaser

开发工具版本固定为 GoReleaser v2.19.0 / nFPM v2.48.0，不是 mtask 运行依赖。

```sh
go install github.com/goreleaser/goreleaser/v2@v2.19.0
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

GoReleaser 需要含 commit 的 Git checkout；不要为了打包修改用户全局 Git 配置。模块路径和计划仓库为 github.com/raywcm/mtask；本地仓库已初始化，同名私有远端已创建，暂不公开。

输出包括 Linux/macOS × amd64/arm64 的单二进制归档、Linux 两架构的 Deb/RPM 包与 SHA-256 校验清单。`CGO_ENABLED=0`，Linux 无 libc 运行时依赖。

也可以对已有 Linux 二进制单独使用 nFPM：

```sh
MTASK_VERSION=0.1.0 MTASK_ARCH=amd64 MTASK_BINARY=./bin/mtask-linux-amd64 \
  nfpm package --config packaging/nfpm.yaml --packager deb --target dist/
```

包只安装 `/usr/bin/mtask` 和文档，不安装全局 systemd 服务，不在升级/卸载钩子中终止 Broker。项目采用 MIT 许可证，归档和安装包包含 LICENSE、THIRD_PARTY_NOTICES.md 与依赖许可证原文；维护者联系入口及公开下载需在远端仓库创建后接通。

## 本地 APT 软件源

```sh
python3 scripts/apt_repo.py dist dist/apt
```

脚本从 Deb 的控制元数据生成 `Packages`、`Packages.gz`、`Release` 及包校验和，支持 amd64/arm64。不会监听端口、上传文件或生成签名密钥。

仅在本地测试仓库时，可在 Debian/Ubuntu 的 sources 中添加实际绝对目录，例如：

```text
deb [trusted=yes] file:/absolute/path/to/dist/apt ./
```

随后执行 `apt update` 和 `apt install mtask`。`trusted=yes` 只适用于自己生成并控制的本地验证目录。正式软件源需要确定托管地址、维护者、许可证和签名密钥，并生成 InRelease/Release.gpg；本次按要求不公开发布，也未配置这些外部资源。

## 自动化验证

`.github/workflows/ci.yml` 在 Ubuntu 24.04/macOS 14 运行单元测试、竞态检查及真实 PTY 集成测试，然后在 Ubuntu 生成安装包和本地 APT 索引。此处存在配置不代表已在 GitHub 执行；实际执行需要将项目放入相应仓库。

开发机为 macOS；现已通过用户提供的 SSH 主机，在 Ubuntu 24.04.3 LTS / aarch64 上实际安装并验证单文件版本。CLI/TUI、真实 PTY、中文输入输出、resize/Ctrl+C、CPU/RSS、SSH 意外断开重连和后台历史清理通过。Linux amd64 实体运行、Deb/RPM 的包管理器安装卸载及系统软件源更新尚未实测；二进制安装不等同于这些验收。Deb/RPM 内容、架构、索引与校验和已在本机验证。

0.3.0 的多终端连接功能按用户要求仅更新本地，未连接或升级云端。该版本的证据为 macOS 本地真实 PTY/界面回归及 Linux/macOS 四目标交叉编译；上述 Ubuntu 实机记录对应此前安装版本，不代表 0.3.0 已完成 Linux 实机验收。

0.4.0 加入前端监控、确认弹窗和重名建议。用户随后授权云端更新，并要求保留现有会话；安装继续使用原子替换，不重启既有 Broker。旧会话可使用监控，旧 Broker 尺寸查询的回退规则见 usage.md。新增 monitor_dialog_regression.py 覆盖监控、弹窗与命名竞争，已接入 CI。

## 当前验收入口

`make check` 校验单元测试、竞态及依赖声明；`make acceptance PYTHON=.venv/bin/python` 运行真实进程/PTY 回归并写入 dist/acceptance/local.json。已验证范围见 [compatibility.md](compatibility.md)，完整复现见 [stability.md](stability.md)。上述旧版本段落保留历史证据，不能作为当前版本完整实机结果。
