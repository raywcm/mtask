# mtask

A terminal task manager for Linux and macOS. Keep programs running after an SSH
disconnect, then return to them from a task list. No tmux or screen required.

![mtask demo](docs/assets/demo.gif)

## Install

```sh
curl -fsSL https://github.com/raywcm/mtask/releases/latest/download/install.sh | sh
```

Installs to `~/.local/bin` after SHA-256 verification. Supports Linux and macOS
on amd64/arm64. Add `~/.local/bin` to PATH. Existing tasks keep running on upgrade.
See [releases](https://github.com/raywcm/mtask/releases) for archives and Deb/RPM
packages, or [installation options](docs/installation.md) for a specific version
or installation directory.

### From source

Requires Go 1.26+, Git and make. Go is not needed at runtime.

```sh
git clone https://github.com/raywcm/mtask.git
cd mtask
make build
./scripts/install.sh --prefix "$HOME/.local" ./bin/mtask
```

Add `~/.local/bin` to PATH, or run `./bin/mtask` directly.

## Use

```sh
mtask new --name work -- bash
# Press F12 to detach. The task keeps running.
mtask
# Select work and press Enter to reconnect.
```

When using SSH, run mtask on the remote host. Reconnect to the same host and
account after a dropped connection. Tasks do not survive a host reboot.

| In the task list | Action |
| --- | --- |
| ↑/↓, Enter | Select and enter a task; stopped tasks open output history |
| Tab / Shift+Tab | Switch Running, Stop and Settings tabs |
| n, F2, s, Delete | Create, rename, stop or delete; stop/delete require confirmation |
| v | Monitor the selected running task |
| c | Choose takeover, read-only or shared control |
| F5, F1/? | Refresh or show help |
| q / Ctrl+C | Exit the manager without stopping tasks |

Inside a task, **F12** or **Ctrl+B, d** returns to the list.
For compatibility, **Ctrl+B, Ctrl+D** (hold Ctrl while pressing B then D) and
**Ctrl+], d** also work. Standalone **Ctrl+D** and **Ctrl+C** go to the application.
On Mac keyboards, F12 may require Fn. Use `mtask attach --no-escape TASK` to send
all keys, including F12, to the application.

## CLI

```sh
mtask new -d --name build -- make test
mtask list --json
mtask logs --follow build
mtask stop build
mtask delete build
mtask attach work --read-only     # Or --shared / --takeover
mtask settings set history-retention off
```

Stopped records and logs expire after **one hour** by default. Each task retains
two output files of about 8 MiB. All connections use the same Unix account.

Reattach restores known terminal modes and requests an application redraw; it
cannot restore a full screen snapshot or host-terminal scrollback. Monitoring
uses the preview pane's PTY size and retains it when leaving.

- [Usage](docs/usage.md) · [Compatibility](docs/compatibility.md)
- [Build and packaging](docs/installation.md) · [Testing](docs/stability.md)
- [Architecture](docs/architecture.md) · [Protocol](docs/protocol.md)
- [Language packs](docs/i18n.md) · [Roadmap](docs/roadmap.md)
- [GitHub Actions](https://github.com/raywcm/mtask/actions)

[MIT License](LICENSE). Dependency licenses are included in
[third-party notices](THIRD_PARTY_NOTICES.md).
