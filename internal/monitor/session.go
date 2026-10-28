// Package monitor reconstructs a screen from bounded task output logs.
// Its exclusive connection carries keyboard input and owns the pane size.
// Occupied sessions are refused without displacing their clients.
package monitor

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/x/vt"
	"github.com/raywcm/mtask/internal/ipc"
	"github.com/raywcm/mtask/internal/protocol"
	"github.com/raywcm/mtask/internal/store"
	"github.com/raywcm/mtask/internal/task"
	"golang.org/x/sys/unix"
)

type Frame struct {
	Rows                            []string
	Width, Height, CursorX, CursorY int
	CursorHidden                    bool
	Alternate                       bool
	Ready, Ended                    bool
	Err                             error
}
type Session struct {
	mu         sync.RWMutex
	frame      Frame
	cancel     context.CancelFunc
	done       chan struct{}
	resize     chan task.Size
	connection *ipc.Session
	connected  chan struct{}
}

func Open(ctx context.Context, repository *store.JSON, record task.Record, size task.Size) *Session {
	ctx, cancel := context.WithCancel(ctx)
	s := &Session{cancel: cancel, done: make(chan struct{}), connected: make(chan struct{}), resize: make(chan task.Size, 1)}
	go func() { defer close(s.done); s.run(ctx, repository, record, size) }()
	return s
}
func (s *Session) Snapshot() Frame { s.mu.RLock(); defer s.mu.RUnlock(); return s.frame }
func (s *Session) Resize(size task.Size) {
	if size.Validate() != nil {
		return
	}
	select {
	case s.resize <- size:
	default:
		select {
		case <-s.resize:
		default:
		}
		select {
		case s.resize <- size:
		default:
		}
	}
}

// Input uses the monitor's existing exclusive display lease. Switching between
// preview and input never reconnects or changes the PTY dimensions.
func (s *Session) Input(data []byte) error {
	if s.connected == nil {
		return errors.New("session is unavailable for input")
	}
	select {
	case <-s.connected:
	case <-s.done:
		return errors.New("session is unavailable for input")
	}
	s.mu.RLock()
	conn, frame := s.connection, s.frame
	s.mu.RUnlock()
	if conn == nil || frame.Ended || frame.Err != nil {
		return errors.New("session is unavailable for input")
	}
	for len(data) > 0 {
		n := min(len(data), protocol.MaxPayload)
		if err := conn.Send(protocol.Frame{Kind: protocol.Input, Payload: data[:n]}); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
func (s *Session) Close() { s.cancel(); <-s.done }

func dimensions(size task.Size) (int, int) {
	if size.Validate() != nil {
		return 80, 24
	}
	return min(512, int(size.Cols)), min(256, int(size.Rows))
}
func newScreen(width, height int) *vt.Emulator {
	e := vt.NewEmulator(width, height)
	e.SetScrollbackSize(0)
	// VT device queries can write synchronously to the input pipe. Closing it
	// discards those replies without a reader goroutine or any application input.
	if closer, ok := e.InputPipe().(io.Closer); ok {
		_ = closer.Close()
	}
	return e
}
func screenFrame(e *vt.Emulator) Frame {
	rows := make([]string, e.Height())
	for y := 0; y < e.Height(); y++ {
		var line strings.Builder
		style := ""
		for x := 0; x < e.Width(); x++ {
			cell := e.CellAt(x, y)
			if cell == nil {
				if style != "" {
					line.WriteString("\x1b[0m")
					style = ""
				}
				line.WriteByte(' ')
				continue
			}
			if cell.Width == 0 {
				continue
			}
			next := cell.Style.String()
			if next != style {
				line.WriteString("\x1b[0m")
				line.WriteString(next)
				style = next
			}
			// Only generated SGR plus printable cell content leave this renderer.
			// OSC title/clipboard/hyperlinks, DCS and terminal queries stay in the emulator.
			content := strings.Map(func(r rune) rune {
				if unicode.IsControl(r) {
					return -1
				}
				return r
			}, cell.Content)
			if content == "" {
				content = " "
			}
			line.WriteString(content)
		}
		line.WriteString("\x1b[0m")
		rows[y] = line.String()
	}
	return Frame{Rows: rows, Width: e.Width(), Height: e.Height(), CursorX: e.CursorPosition().X, CursorY: e.CursorPosition().Y, CursorHidden: e.CursorHidden(), Alternate: e.IsAltScreen(), Ready: true}
}
func (s *Session) publish(e *vt.Emulator, err error, ended bool) {
	frame := screenFrame(e)
	frame.Err = err
	frame.Ended = ended
	s.mu.Lock()
	s.frame = frame
	s.mu.Unlock()
}
func openLog(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	var stat unix.Stat_t
	if err = unix.Fstat(fd, &stat); err == nil && (stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&0077 != 0 || int(stat.Uid) != os.Getuid()) {
		err = errors.New("unsafe monitor output log")
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
func feed(ctx context.Context, e *vt.Emulator, f *os.File, limit int64) (bool, error) {
	buf := make([]byte, 32*1024)
	for limit > 0 {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, err := f.Read(buf[:min(int64(len(buf)), limit)])
		if n > 0 {
			if _, writeErr := e.Write(buf[:n]); writeErr != nil {
				return false, writeErr
			}
			limit -= int64(n)
		}
		if errors.Is(err, io.EOF) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}
func actualSize(ctx context.Context, repository *store.JSON, r task.Record) task.Size {
	if r.Runtime == nil {
		return r.Spec.Size
	}
	probe, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	conn, err := ipc.Connect(probe, repository.Root(), *r.Runtime, "control", nil)
	if err != nil {
		return r.Spec.Size
	}
	defer conn.Close()
	if conn.Welcome.Size != nil && conn.Welcome.Size.Validate() == nil {
		return *conn.Welcome.Size
	}
	return r.Spec.Size
}
func (s *Session) run(ctx context.Context, repository *store.JSON, r task.Record, wanted task.Size) {
	original := actualSize(ctx, repository, r)
	width, height := dimensions(original)
	wantedWidth, wantedHeight := dimensions(wanted)
	wanted = task.Size{Rows: uint16(wantedHeight), Cols: uint16(wantedWidth)}
	if ctx.Err() != nil {
		return
	}
	e := newScreen(width, height)
	defer e.Close()
	if r.Runtime == nil {
		s.publish(e, errors.New("task has not started"), false)
		return
	}
	// Enter directly at the pane size, including with old brokers whose
	// Welcome cannot report the current size. Never bounce through Spec.Size.
	conn, err := ipc.ConnectAttachment(ctx, repository.Root(), *r.Runtime, &wanted, protocol.AttachExclusive)
	if err != nil {
		if ctx.Err() == nil {
			s.publish(e, err, false)
		}
		return
	}
	s.mu.Lock()
	s.connection = conn
	close(s.connected)
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.connection = nil; s.mu.Unlock() }()
	type result struct {
		err   error
		ended bool
	}
	readDone := make(chan result, 1)
	go func() {
		for {
			frame, err := protocol.Read(conn.Conn)
			if err != nil {
				readDone <- result{err: err}
				return
			}
			switch frame.Kind {
			case protocol.Exit:
				readDone <- result{ended: true}
				return
			case protocol.Error:
				readDone <- result{err: ipc.FrameError(frame)}
				return
			}
		}
	}()
	readerFinished := false
	defer func() {
		if !readerFinished {
			// Detach is acknowledged by EOF only after the broker releases the lease.
			// This lets Enter move from monitor to ordinary attach without a busy race.
			_ = conn.Send(protocol.Frame{Kind: protocol.Detach})
			_ = conn.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			<-readDone
		}
		_ = conn.Close()
	}()

	current, _ := repository.Path(r.Spec.ID, "output.log")
	previous, _ := repository.Path(r.Spec.ID, "output.log.1")
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	// Prime both retained generations, through the offscreen emulator only.
	if first, err := openLog(previous); err == nil {
		_, err = feed(ctx, e, first, 8*1024*1024)
		file = first
		if err != nil {
			if ctx.Err() == nil {
				s.publish(e, err, false)
			}
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		s.publish(e, err, false)
		return
	}
	// Seed at the task's original dimensions before resizing the live PTY.
	if first, err := openLog(current); err == nil {
		if file != nil {
			file.Close()
		}
		file = first
		if _, err = feed(ctx, e, file, 8*1024*1024); err != nil {
			if ctx.Err() == nil {
				s.publish(e, err, false)
			}
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		s.publish(e, err, false)
		return
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	needsSize := true
	appliedSize := wanted // The attachment handshake already applied this size.
	endSeen := false
	var missingSince time.Time

	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case result := <-readDone:
			readerFinished = true
			if result.err != nil {
				s.publish(e, result.err, false)
				return
			}
			// Final bytes are committed to the log before Exit; drain them below.
		default:
		}
		select {
		case wanted = <-s.resize:
			needsSize = true
		default:
		}
		if needsSize {
			w, h := dimensions(wanted)
			wanted = task.Size{Rows: uint16(h), Cols: uint16(w)}
			if wanted != appliedSize {
				if err := conn.Send(ipc.JSON(protocol.Resize, wanted)); err != nil {
					s.publish(e, err, false)
					return
				}
				appliedSize = wanted
			}
			if e.Width() != w || e.Height() != h {
				e.Resize(w, h)
			}
			needsSize = false
		}
		if file == nil {
			var err error
			file, err = openLog(current)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				s.publish(e, err, false)
				return
			}
			if errors.Is(err, os.ErrNotExist) {
				if missingSince.IsZero() {
					missingSince = time.Now()
				}
				if time.Since(missingSince) >= 2*time.Second {
					s.publish(e, errors.New("task output is unavailable"), false)
					return
				}
			} else {
				missingSince = time.Time{}
			}
		}
		atEnd := true
		if file != nil {
			var err error
			atEnd, err = feed(ctx, e, file, 2*1024*1024)
			if err != nil {
				if ctx.Err() == nil {
					s.publish(e, err, false)
				}
				return
			}
			if atEnd {
				opened, openErr := file.Stat()
				latest, statErr := os.Stat(current)
				if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
					s.publish(e, statErr, false)
					return
				}
				if openErr != nil {
					s.publish(e, openErr, false)
					return
				}
				if statErr == nil && !os.SameFile(opened, latest) {
					// If several generations passed while behind, read the latest retained
					// predecessor before current; never loop over a generation already read.
					candidate := current
					if older, err := os.Stat(previous); err == nil && !os.SameFile(opened, older) {
						candidate = previous
					}
					next, err := openLog(candidate)
					if err == nil {
						file.Close()
						file = next
						s.publish(e, nil, false)
						continue
					}
					if !errors.Is(err, os.ErrNotExist) {
						s.publish(e, err, false)
						return
					}
				}
			}
		}
		latest, err := repository.Load(ctx, r.Spec.ID)
		if err != nil {
			if ctx.Err() == nil {
				s.publish(e, err, false)
			}
			return
		}
		ended := latest.Runtime != nil && (latest.Runtime.State == task.Exited || latest.Runtime.State == task.Failed)
		if file != nil || ended {
			s.publish(e, nil, ended)
		}
		if ended && atEnd {
			if endSeen {
				return
			}
			endSeen = true
			continue
		}
		endSeen = false

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
