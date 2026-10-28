package tui

import (
	"bytes"
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestMonitorTypeaheadAndFailureRelease(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	defer write.Close()
	route := &monitorInput{File: read}
	route.Arm(true)
	input := []byte("vinput\x03中文")
	if _, err := write.Write(input); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1024)
	if n, err := route.Read(buffer); err != nil || n != 1 || buffer[0] != 'v' {
		t.Fatalf("activation: %d %v", n, err)
	}
	route.Begin()
	if !bytes.Equal(route.pending, input[1:]) {
		t.Fatalf("typeahead lost: %q", route.pending)
	}
	m := readyModel(t)
	m.monitorInput = route
	m, cmd := press(m, 'v')
	if cmd == nil {
		t.Fatal("monitor command missing")
	}
	next, _ := m.Update(cmd()) // The backend reports a connection error.
	m = next.(model)
	if route.enabled || len(route.pending) > 0 || m.monitorFrame.Err == nil {
		t.Fatal("failed monitor retained the keyboard or buffered application input")
	}
	m, _ = press(m, tea.KeyF12)
	if m.monitoring {
		t.Fatal("F12 did not return from failed monitoring")
	}
}

func TestMonitorKeysCannotTriggerListActions(t *testing.T) {
	m := readyModel(t)
	route := &monitorInput{}
	m.monitorInput = route
	m, _ = press(m, 'v')
	defer m.closeMonitor()
	for _, key := range []rune{'n', 's', 'q', 'v', tea.KeyLeft, tea.KeyDown, tea.KeyEnter, tea.KeyF12} {
		msg := tea.KeyPressMsg{Code: key}
		if key >= 32 && key < 127 {
			msg.Text = string(key)
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}
	if !m.monitoring || m.mode != "" || m.selected.Kind != "" || m.cursors[0] != 0 {
		t.Fatal("application input triggered a list action")
	}
	if !bytes.Contains(route.pending, []byte("nsqv\x1b[D\x1b[B\r\x1b[24~")) {
		t.Fatalf("prefetched input lost: %q", route.pending)
	}
}
