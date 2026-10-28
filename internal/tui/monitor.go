package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/raywcm/mtask/internal/ipc"
	"github.com/raywcm/mtask/internal/monitor"
	"github.com/raywcm/mtask/internal/task"
)

type monitorTick struct{ generation uint64 }
type monitorOpened struct {
	generation uint64
	session    *monitor.Session
	err        error
}

func monitorTickCmd(generation uint64) tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return monitorTick{generation: generation} })
}
func (m model) monitorGeometry() (left, right, height int) {
	width := max(1, m.width-4)
	left = min(28, max(16, width/4))
	right = max(1, width-left-3)
	height = max(1, m.height-7)
	return
}
func (m model) monitorSize() task.Size {
	_, width, height := m.monitorGeometry()
	return task.Size{Rows: uint16(min(256, max(4, height))), Cols: uint16(min(512, max(20, width)))}
}
func (m *model) closeMonitor() {
	if m.monitorCancel != nil {
		m.monitorCancel()
		m.monitorCancel = nil
	}
	if m.monitorSession != nil {
		m.monitorSession.Close()
		m.monitorSession = nil
	}
	m.monitorID = ""
	m.monitorGeneration++
	m.monitorFrame = monitor.Frame{}
}
func (m *model) syncMonitor() tea.Cmd {
	if !m.monitoring || m.tab != tabRunning {
		return nil
	}
	r, ok := m.current()
	if !ok || m.width < 50 || m.height < 12 {
		m.closeMonitor()
		return nil
	}
	if r.Spec.ID == m.monitorID {
		return nil
	}
	m.closeMonitor()
	m.monitorID = r.Spec.ID
	ctx, cancel := context.WithCancel(m.ctx)
	m.monitorCancel = cancel
	generation := m.monitorGeneration
	size := m.monitorSize()
	catalog := m.catalog
	return func() tea.Msg {
		session, err := catalog.Monitor(ctx, r.Spec.ID, size)
		if ctx.Err() != nil && session != nil {
			session.Close()
			session = nil
		}
		return monitorOpened{generation: generation, session: session, err: err}
	}
}
func (m model) monitorView(width int) []string {
	t, p := m.locale().Text, m.palette()
	if m.width < 50 || m.height < 12 {
		return []string{m.header(width), t("monitor"), t("small_window"), t("monitor_hint")}
	}
	left, right, height := m.monitorGeometry()
	rows := m.rows(tabRunning)
	title := t("monitor_select")
	if r, ok := m.current(); ok {
		title = safe(r.Spec.Name) + " [" + shortID(r.Spec.ID, m.tasks) + "] · " + t("mode_readonly")
	}
	lines := []string{m.header(width), "", m.tabs(width), p.muted.Render(strings.Repeat("─", width)), fit(t("session"), left) + " │ " + ansi.Truncate(p.accent.Bold(true).Render(title), right, "…")}
	start := max(0, m.cursors[0]-height+1)
	preview := m.monitorFrame
	for y := 0; y < height; y++ {
		label := ""
		if i := start + y; i < len(rows) {
			label = fit("● "+safe(rows[i].Spec.Name), left)
			if i == m.cursors[0] {
				label = lipgloss.NewStyle().Reverse(true).Bold(true).Render(label)
			}
		}
		content := ""
		switch {
		case preview.Err != nil:
			if y == height/2 {
				content = t("monitor_unavailable")
			}
			if y == height/2+1 {
				var remote *ipc.RemoteError
				if errors.As(preview.Err, &remote) && remote.Code == "busy" {
					content = t("monitor_busy")
				} else {
					content = safe(preview.Err.Error())
				}
			}
		case !preview.Ready:
			if y == height/2 {
				if len(rows) == 0 {
					content = t("empty_running")
				} else {
					content = t("loading")
				}
			}
		case y < len(preview.Rows):
			content = preview.Rows[y]
		}
		lines = append(lines, fit(label, left)+" │ "+fit(ansi.Truncate(content, right, ""), right))
	}
	lines = append(lines, m.status(width), p.muted.Render(ansi.Truncate(t("monitor_hint"), width, "…")))
	return lines
}
