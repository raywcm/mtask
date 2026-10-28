# Usage

## Create and reconnect

```sh
mtask new -s work -- bash
mtask new -ds build -- make test
mtask new -c /absolute/project -s editor -- vim README.md
mtask attach work
```

Interactive `new` enters the task immediately. `-d`, `--json` or non-terminal input
starts it in the background. Omit the command to use `$SHELL` or `/bin/sh`.
A single quoted command runs through `$SHELL -c`, for example
`mtask new -ds build 'make test && make build'`. Multiple arguments run directly
as argv. Use `--` before a command to keep even a single argument literal.
Place mtask flags before the command; remaining flags belong to the application.
Tasks accept names, full IDs or unique ID prefixes of at least four characters.

## tmux session commands

| Command | Supported flags / aliases |
| --- | --- |
| `new-session` | `new`, `-s NAME`, `-c DIRECTORY`, `-d`, `-A` |
| `list-sessions` | `ls`, `--json`, `--history` |
| `attach-session` | `attach`, `a`, `-t SESSION`, `-r`, `--shared`, `--takeover` |
| `rename-session` | `rename`, `-t SESSION NAME` |
| `has-session` | `has`, `-t SESSION`; exit 0 if running, 1 otherwise |
| `kill-session` | `-t SESSION`; stops the task and removes its history |

`new -As work` attaches to a running named session or creates it. Add `-d` to
leave it detached. An ended record keeps its name; inspect or delete it first.
Targets accept names, IDs or unique ID prefixes; positional targets still work.
Targets are explicit: tmux window/pane syntax, formats, implicit current sessions
and `attach -d` are unsupported. Listing includes retained stopped records.

Existing scripts should replace `list` with `ls`, `status` with `show`, `rm` with
`delete`, and `new -n` with `new -s`. `show` always prints JSON; its redundant
`--json` flag was removed. Use `stop` when you want to retain output history;
`delete` removes a stopped record.

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

The title row shows the main shortcuts; F1/? opens the full list.

```sh
mtask ls --json
mtask show work
mtask rename work work-old
mtask stop work-old
mtask ls --history --json
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

In the TUI, new sessions open selected in the monitor. In the task list, Enter
opens a session fullscreen; `v` opens the monitor. Ordinary input, Enter, ←/→,
Ctrl+C/D and pasted text in the monitor go to the application. ↑/↓ switches
sessions; Esc, F12 or Ctrl+B, d returns to the list. To enter a session, return
to the list first. Ctrl+Enter is ignored throughout the manager and monitor.

The monitor keeps a narrow session list beside a wide control pane. ↑/↓ and Esc
are manager keys and never reach the application; bracketed paste stays literal.
The monitor owns an exclusive connection and uses the pane's size. Leaving
retains that size. Occupied or unavailable tasks display “Unable to monitor”;
↑/↓ selects another task, F5 retries, and F12/Esc returns to the list. Embedded
control supports keyboard input; use fullscreen attach for native terminal mouse
behavior and terminal extensions.

Press **Ctrl+F** in the list or monitor to search running and stopped sessions.
Words match name, ID, directory and command without case sensitivity. ↑/↓ and
PgUp/PgDn navigate results; Enter confirms and Esc cancels. From the list, a
running result opens fullscreen; from the monitor, it changes the monitored
session. Stopped results open retained output. Search input stays out of the application.
Cancelling resumes the same monitor without resizing. The list scrolls with the
selection and supports PgUp/PgDn and Home/End.

Settings shows the installed version. Open the editor with Enter or `,`.
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
