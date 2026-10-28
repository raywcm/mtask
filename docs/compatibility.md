# Compatibility and known limitations

Updated: 2026-10-28 (Asia/Shanghai). Current version: **0.4.3-dev**.
This matrix separates execution evidence from build-only targets. A successful
PTY/cell-model test does not certify every terminal emulator or every feature
of an application.

## Operating systems

| Target | Evidence | Scope / remaining work |
| --- | --- | --- |
| macOS arm64 | Local execution and automated real-PTY acceptance | CLI/TUI, task lifecycle, Unicode, terminal-cell rendering, resize, detach/reconnect, connection modes, retention, interruption and atomic upgrades |
| Ubuntu 24.04.3 LTS arm64 | SSH deployment and isolated real-PTY regression | Previous full lifecycle/SSH-loss acceptance; current-version targeted regression is recorded in the development log. Distribution package installation is a separate test. |
| Linux amd64 | Cross-build | No native machine execution claimed. Ubuntu CI is configured but hosted runs are pending. |
| macOS amd64 | Cross-build | No native Intel Mac execution claimed. |
| Debian / RPM-based distributions | Packages and payload checks | Native install/remove/upgrade execution remains pending. |

Builds disable CGO. The declared minimum Go version is 1.26; the local development
toolchain used in this acceptance run is Go 1.27.3. Runtime requires neither Go
nor tmux/screen/zmx. Minimum-version Go execution still needs a dedicated runner.

## Applications and terminals

| Application / environment | Evidence | What it does not establish |
| --- | --- | --- |
| POSIX shell / Bash | Real process input/output, exit codes, Ctrl+C and job handling | All shell customizations and plugins |
| Vim | Real PTY lifecycle and editor exit checks | Every Vim plugin or terminal extension |
| htop, when installed | Visible static content across repeated reconnects | Every ncurses program |
| Simulated incremental full-screen application | Visible-cell and mode assertions across reconnects | A complete VT snapshot implementation |
| Codex / OpenCode | 0.4.3-dev local mixed-session PID smoke: Codex 0.164.0-alpha.11.1 / OpenCode 1.18.30; no model prompts | End-to-end coding, approvals, authentication/network reliability or all mouse/keyboard features |
| xterm-256color protocol, real PTY + pyte model | Automated main/alternate buffer, cursor, Unicode and redraw checks | Native GUI terminal behavior; pyte is a test decoder, not a desktop terminal |
| SSH | Prior Ubuntu ARM64 disconnect/reconnect process-identity test | Host reboot persistence or hosts configured to kill all user background processes |
| Tabby and other GUI terminal emulators | User reports and manual observations | No blanket certification; fixes require reproduction and focused regression |

Codex/OpenCode retained six mixed task PIDs across two attach rounds. OpenCode
emitted no direct-startup bytes during the short observation window, so this is
process-lifecycle evidence only, not a visible-startup equivalence pass. Exact
results are recorded in the development log or acceptance artifacts. Installed executables alone do not
constitute an application compatibility pass.

## Behavior to understand before using mtask

1. **Disconnect persistence is process persistence.** Terminal closure or SSH loss
   leaves brokers running if the host permits it. Reboot, OOM, and login-session
   cleanup can terminate them. mtask does not resume dead applications after reboot.
2. **Reattach is not a full screen/scrollback restore.** It restores known modes
   and triggers an application redraw using a temporary size change. Shell
   scrollback and output lost to log rotation cannot be recreated. Applications
   that do not redraw or use unsupported terminal extensions can display incompletely.
3. **Monitoring changes the task size.** Preview acquires an exclusive display
   lease and uses the pane size. Leaving retains that size; a full attach applies
   the full terminal size. Occupied/unreachable tasks show an explicit failure
   instead of automatic takeover. Preview is reconstructed from bounded logs.
4. **Multi-client roles share one Unix account.** Read-only restricts that
   connection, not the account's ability to open another management client.
   Shared controllers can interfere through input; the oldest connected
   controller owns the PTY size. There is no cross-account sharing service.
5. **Stopped output expires.** Records and logs are removed after one hour by
   default. Set `mtask settings set history-retention off` for indefinite retention.
   Each task's output is bounded to two approximately 8 MiB files.
6. **Broker loss cannot recover the PTY.** `recover` updates stale records without
   signalling old PIDs. Deliberately detached descendant processes may need manual
   inspection after a broker failure.
7. **Upgrades preserve existing broker code.** Updating the executable starts new
   tasks with the new version; it does not hot-patch old brokers. Ordinary attach
   remains available for protocol-v1 brokers; v0.3.0+ is required for explicit
   takeover/read-only/shared modes.
8. **TUI exit and application interruption differ.** Ctrl+C in the list exits
   cleanly and leaves tasks running. In a full attach it is application input.
   SIGKILL cannot restore frontend terminal settings; normal Ctrl+C/INT/TERM/HUP
   cleanup is covered by the exit regression.

See [stability acceptance](stability.md) for reproduction commands and
[usage](usage.md) for operational details.
