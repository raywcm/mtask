# Initial architecture (A0)

Historical design, 2026-10-27. See [current architecture](../architecture.md).

The initial prototype established a broker per task, framed Unix-socket
communication, separate task/runtime snapshots and a shared CLI/TUI service.
It contained PTY, storage, input-prefix and frame primitives plus a list UI.
Detached startup, authenticated sockets and management commands followed later.

The main constraints remain: the broker owns task lifetime, frontends own only
their connections, and attached application output bypasses the TUI renderer.
