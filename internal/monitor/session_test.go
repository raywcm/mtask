package monitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/raywcm/mtask/internal/store"
	"github.com/raywcm/mtask/internal/task"
)

func TestOffscreenEmulationStylesCJKAndAlternateScreen(t *testing.T) {
	e := newScreen(40, 10)
	defer e.Close()
	_, _ = e.Write([]byte("SHELL\r\n"))
	_, _ = e.Write([]byte("\x1b[?1049h\x1b[2J\x1b[H\x1b[31m标题\x1b[3;1HSTATIC\x1b[5;1Hcounter:1"))
	_, _ = e.Write([]byte("\x1b[5;9H2"))
	frame := screenFrame(e)
	if !frame.Alternate || !strings.Contains(ansi.Strip(frame.Rows[0]), "标题") || !strings.Contains(ansi.Strip(frame.Rows[2]), "STATIC") || !strings.Contains(ansi.Strip(frame.Rows[4]), "counter:2") {
		t.Fatalf("screen was not reconstructed: %#v", frame.Rows)
	}
	for _, row := range frame.Rows {
		if !utf8.ValidString(row) || ansi.StringWidth(row) != 40 {
			t.Fatalf("invalid cell row: %q", row)
		}
	}
	_, _ = e.Write([]byte("\x1b[?1049l"))
	if got := screenFrame(e); got.Alternate || !strings.Contains(ansi.Strip(got.Rows[0]), "SHELL") {
		t.Fatal("normal buffer was lost")
	}
}
func TestQueriesAndOSCStayInsideEmulator(t *testing.T) {
	e := newScreen(40, 8)
	defer e.Close()
	done := make(chan struct{})
	go func() {
		_, _ = e.Write([]byte("\x1b]2;HOST-TITLE\a\x1b]52;c;SECRET\a\x1b[6n\x1b[c\x1b]8;;https://example.invalid\aLINK\x1b]8;;\a"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("device query blocked the preview")
	}
	frame := screenFrame(e)
	for _, row := range frame.Rows {
		if strings.Contains(row, "\x1b]") || strings.Contains(row, "HOST-TITLE") || strings.Contains(row, "SECRET") || strings.Contains(row, "\x1b[6n") {
			t.Fatalf("unsafe terminal commands escaped the virtual screen: %q", row)
		}
	}
	if !strings.Contains(ansi.Strip(frame.Rows[0]), "LINK") {
		t.Fatal("safe printable content was lost")
	}
}
func TestMonitorLogsArePrivateRegularFiles(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "output.log")
	if err := os.WriteFile(target, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := openLog(target)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	link := filepath.Join(root, "symlink")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if f, err := openLog(link); err == nil {
		f.Close()
		t.Fatal("followed a symlink")
	}
	if err := os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	if f, err := openLog(target); err == nil {
		f.Close()
		t.Fatal("accepted a public log file")
	}
}
func TestCancelledSessionExitsAndDimensionsAreBounded(t *testing.T) {
	repository, err := store.Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	id, _ := task.NewID()
	s := Open(ctx, repository, task.Record{Spec: task.Spec{ID: id, Size: task.Size{Rows: 24, Cols: 80}}}, task.Size{Rows: 10, Cols: 40})
	done := make(chan struct{})
	go func() { s.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled monitor leaked")
	}
	w, h := dimensions(task.Size{Rows: 65535, Cols: 65535})
	if w > 512 || h > 256 {
		t.Fatal("unbounded virtual screen")
	}
}
