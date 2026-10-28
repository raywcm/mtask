# Roadmap

The core task lifecycle, CLI/TUI, shared connections, monitoring, history and
four-target builds are implemented. Remaining work:

- Broader terminal testing: scrollback, selection, clipboard, mouse and enhanced
  keyboard modes in Codex, OpenCode, Vim and shells.
- More reliable screen recovery for applications that do not redraw on resize.
- Longer concurrent-task and high-output runs, including broker failures.
- Native Linux amd64/Intel Mac and distribution package-manager testing.
- Public binary releases and a signed APT repository.

Compatibility results belong in [compatibility.md](compatibility.md). The
reproducible test commands are in [stability.md](stability.md).
