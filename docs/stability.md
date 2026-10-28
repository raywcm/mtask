# Testing

Tests create temporary tasks and state directories. Existing tasks are left
alone. Build to a new file or use atomic replacement when updating an executable
that may still be used by a broker.

## Run the suite

Requirements: Linux/macOS, Go 1.26+, Python 3, and the test dependency pyte.

```sh
make check
python3 -m venv .venv
.venv/bin/pip install -r scripts/requirements-screen.txt
make acceptance PYTHON=.venv/bin/python
```

The runner saves per-suite output, exit status, platform, version and elapsed time
to `dist/acceptance/local.json`. It stops on failure. Reports and binaries are
excluded from Git.

For race-enabled terminal acceptance, build to a new path first:

```sh
go build -race -o /tmp/mtask-acceptance ./cmd/mtask
.venv/bin/python scripts/acceptance.py --binary /tmp/mtask-acceptance \
  --report dist/acceptance/race.json
```

Use a fresh binary path for each build or atomically rename a completed build.
The `make build` target already uses a temporary file and atomic rename.

## What each suite verifies

| Suite | Assertions |
| --- | --- |
| integration.py | Lifecycle, task identity, raw byte forwarding, Unicode, resize, application Ctrl+C, malformed/slow clients, bounded logs, atomic upgrade, shell/editor behavior |
| screen_regression.py | Main/alternate screen boundaries, static incremental full-screen content after reconnect, terminal modes, shell history preservation; real htop if available |
| ui_regression.py | Five locales, tabs/settings, resource display, stop confirmation, persisted preferences |
| connection_modes_regression.py | Broker-enforced read-only, takeover, shared input, size ownership, capability negotiation, busy/confirmation UI |
| monitor_dialog_regression.py | Preview, size retention/deduplication, horizontal-key suppression, busy/retry, attach transition, dialogs and competing name suggestions |
| history_view_regression.py | Five-language stopped Enter viewer, scrolling, safe OSC filtering, empty output, unchanged tasks |
| history_regression.py | Default one-hour retention, background cleanup, policy changes and protection of active tasks |
| tui_exit_regression.py | Ctrl+C and INT/TERM/HUP in five pages, shell cells/cursor/termios preservation, monitor lease release and no task stop |
| stress.py | Bounded 16-task/5-second concurrent health observation, broker RSS sampling and cleanup |

The default soak runs for five seconds. For a longer run, use
`scripts/stress.py --tasks 32 --seconds 3600`. Record task count, duration, RSS
and any failures when comparing results.

## Optional installed-app smoke

```sh
.venv/bin/python scripts/app_smoke.py ./bin/mtask
```

This starts installed Codex/OpenCode, collects startup output and checks PIDs
across attach/detach rounds. It submits no model requests and skips missing
executables. Screen layout and interactive workflows need separate testing.

## Old-broker compatibility

Keep an immutable older binary outside the checkout and run:

```sh
.venv/bin/python scripts/screen_regression.py ./bin/mtask /path/to/old-mtask
.venv/bin/python scripts/monitor_dialog_regression.py ./bin/mtask /path/to/old-mtask
```

`connection_modes_regression.py` also accepts an old binary, but its old-broker
negative checks specifically require a **pre-v0.3.0** broker without the new-mode
capability. A recent broker is not a suitable negative fixture.

## Run on another host

Upload a completed binary and these test scripts to a temporary remote directory.
Run against an isolated state directory as an unprivileged user. Record the
OS/architecture, executable SHA-256, version, exact suites and results. For an
installation upgrade, compare existing task instance IDs, broker/child PIDs and
process birth identities before and after atomic replacement; verification must
not attach to or resize those user tasks.

macOS and Ubuntu ARM64 execution, build-only targets, and pending native package
manager/GUI-terminal checks are separated in the [compatibility matrix](compatibility.md).
Hosted runs are listed in [GitHub Actions](https://github.com/raywcm/mtask/actions).

## Demo recording

The GIF and asciinema recording show a temporary shell session. Regenerate them
with Pillow:

```sh
.venv/bin/pip install -r scripts/requirements-demo.txt
.venv/bin/python scripts/record_demo.py --binary ./bin/mtask --font /path/to/monospace.ttf
```

The default font is Menlo on macOS. On Linux, pass the path to a local monospace
font.
