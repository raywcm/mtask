# Usage

## Create and reconnect

```sh
mtask new --name work -- bash
mtask new -d --name build -- make test
mtask new --cwd /absolute/project --name editor -- vim README.md
mtask attach work
```

Interactive `new` enters the task immediately. `-d`, `--json` or non-terminal input
starts it in the background. Omit the command to use `$SHELL` or `/bin/sh`.
Commands run as argv; use `bash -lc '...'` explicitly for shell syntax.
Tasks accept names, full IDs or unique ID prefixes of at least four characters.

## Detach keys

- **F12**: detach; Mac keyboards may require Fn+F12.
- **Ctrl+B, d**: release Ctrl before pressing d.

Compatibility shortcuts are documented here rather than in the interface:
**Ctrl+B, Ctrl+D** (keep Ctrl held while pressing B then D) and **Ctrl+], d**.

From the manager, detach returns to the list. A direct `mtask attach` returns to
the shell. The task keeps running. Standalone Ctrl+D and Ctrl+C retain the
application's behavior. Press a prefix twice to send it literally.

`mtask attach --no-escape TASK` disables all detach shortcuts, including F12.
Close the terminal or end the frontend to disconnect in that mode. Bracketed
paste passes through without interpreting detach keys. Incomplete escape
sequences wait up to 50 ms before being forwarded.

## Manage tasks

```sh
mtask list --json
mtask show work
mtask rename work work-old
mtask stop work-old
mtask list --history --json
mtask logs work-old
mtask logs --follow work-old
mtask logs --tail 4096 work-old
mtask delete work-old
```

`stop` sends TERM, then KILL after one second if necessary. Deletion requires a
finished task with no active broker. `show` reports exit code and signal.
Detach returns 0; an application exit is returned by `attach`.

Logs retain two approximately 8 MiB files. Normal viewing strips control
sequences; `logs --raw` writes the original bytes. `--tail` counts bytes.
Stopped records and logs expire after one hour by default:

```sh
mtask settings set history-retention 24h
mtask settings set history-retention off
```

In the Stop tab, Enter opens retained output. Use arrows, PgUp/PgDn, Home/End to
navigate; Esc/q returns. Viewing history does not restart the task.

## Multiple clients

```sh
mtask attach work                # Exclusive; returns busy if occupied
mtask attach work --read-only    # No application input or resize
mtask attach work --shared       # All controllers may type
mtask attach work --takeover     # Disconnect existing controllers; keep readers
```

The flags are mutually exclusive and require v0.3.0+ brokers. The TUI's `c` menu
provides the same modes, with read-only selected by default. Takeover requires
confirmation in the TUI; the CLI flag acts immediately.

The oldest connected controller owns the PTY size; ownership transfers on
leaving. Shared inputs can interfere. Read-only restricts that connection, not
the account's ability to open another management client. All peers use the same
Unix UID. Older brokers retain ordinary exclusive attach.

## Monitor and settings

Press `v` in Running to preview a task. ↑/↓ switches tasks, F5 retries, Enter
attaches, and v/Esc returns. ←/→ and h/l do nothing in the monitor's main view;
dialogs and inputs keep their normal arrow keys. Monitoring uses an exclusive
connection and the pane's size. Leaving retains that size. Occupied or unavailable
tasks display “Unable to monitor.”

The Settings tab opens for editing with Enter, or directly with `,`.
↑/↓ selects a field, ←/→ changes it, Enter saves, Esc cancels. Tab switches tabs
and discards unsaved changes.

```sh
mtask settings --json
mtask settings set language en
mtask settings set theme dark
mtask settings set refresh 2
mtask settings set resources off
mtask settings set default-tab stop
mtask settings reset
```

Languages: auto, en, zh-CN, ja, es, fr. Themes: auto, dark, light. Refresh: 1, 2 or
5 seconds. CPU/RSS cover the task process tree; shared memory can be counted more
than once.

## Storage and recovery

State directory precedence: `--state-dir`, `MTASK_STATE_DIR`,
`$XDG_STATE_HOME/mtask`, then `~/.local/state/mtask`. Directories are 0700; metadata
and logs are 0600. Socket peers verify UID. Arguments and output may contain
sensitive data.

After SSH loss, reconnect to the same host/account and run mtask again. Reboot,
OOM or host login-session cleanup can end tasks. A crashed broker's PTY cannot
be recovered; `mtask recover TASK` reconciles stale records without signalling
old PIDs. Installation replaces the executable atomically and leaves existing
brokers running. See [compatibility](compatibility.md) for terminal limitations.
