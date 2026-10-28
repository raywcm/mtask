# 构建、安装与打包

目前从源码构建。仓库为私有，尚未提供公开下载或 APT 软件源。

## 单文件安装

构建需要 Go 1.26+ 和 make，运行时只需生成的二进制文件：

```sh
make build
./scripts/install.sh --prefix "$HOME/.local" ./bin/mtask
"$HOME/.local/bin/mtask" --version
```

如果 `~/.local/bin` 不在 PATH 中，使用完整路径运行，或将该目录加入 PATH。

安装到 `/usr/local/bin`：

```sh
sudo ./scripts/install.sh --prefix /usr/local ./bin/mtask
```

安装器不修改 Shell 配置。新文件写入完成后原子替换目标路径，已有 Broker 继续运行。升级时也使用同一命令。

## GoReleaser

打包工具版本为 GoReleaser v2.19.0、nFPM v2.48.0：

```sh
go install github.com/goreleaser/goreleaser/v2@v2.19.0
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

在已有提交的 Git checkout 中运行。`--clean` 会删除原有 `dist` 目录，需保留的测试报告或安装包应先移出。

输出包括：

- Linux/macOS × amd64/arm64 的四份归档。
- Linux amd64/arm64 的 Deb 和 RPM 包。
- SHA-256 校验清单。

构建使用 `CGO_ENABLED=0`，Linux 二进制不依赖 libc。归档和安装包包含 MIT 许可证、第三方许可证原文和文档。

也可以对已有二进制单独使用 nFPM：

```sh
MTASK_VERSION=0.4.3-dev MTASK_ARCH=amd64 MTASK_BINARY=./bin/mtask-linux-amd64 \
  nfpm package --config packaging/nfpm.yaml --packager deb --target dist/
```

包安装 `/usr/bin/mtask` 和 `/usr/share/doc/mtask/`，不安装全局 systemd 服务。升级和卸载钩子不停止任务。

## 本地 APT 软件源

```sh
python3 scripts/apt_repo.py dist dist/apt
```

脚本从 Deb 元数据生成 `Packages`、`Packages.gz`、`Release` 及校验和，支持 amd64/arm64。

在 Debian/Ubuntu 上测试本地索引时，可添加以下 source，将路径换为生成目录的绝对路径：

```text
deb [trusted=yes] file:/absolute/path/to/dist/apt ./
```

然后执行 `apt update` 和 `apt install mtask`。`trusted=yes` 仅用于可信的本地测试目录。对外提供软件源前，需要配置托管地址、签名密钥和 `InRelease` / `Release.gpg`。

## 检查构建结果

```sh
python3 scripts/check_packages.py dist
make check
```

`check_packages.py` 检查八个安装产物的架构、负载、文件权限、许可证和校验和。

[GitHub Actions](https://github.com/raywcm/mtask/actions) 运行单元测试、竞态检查、PTY 集成测试和打包流程。APT 安装、重装、卸载测试仅在一次性 CI 环境中运行；`scripts/ci_install_test.py` 通过 `CI=true` 检查防止误用。

已测试的平台和安装方式见[兼容性表](compatibility.md)，终端回归脚本见[测试说明](stability.md)。
