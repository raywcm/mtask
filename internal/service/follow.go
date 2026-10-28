package service

import (
	"context"
	"errors"
	"io"
	"os"
	"time"
)

// logFilter retains ANSI parsing state only for safe log display, never for the
// live terminal stream. It handles sequences split across follow reads.
type logFilter struct{ state byte }

func (f *logFilter) clean(p []byte) []byte {
	out := make([]byte, 0, len(p))
	for _, b := range p {
		switch f.state {
		case 0:
			if b == 27 {
				f.state = 1
			} else if b >= 32 || b == '\n' || b == '\r' || b == '\t' {
				out = append(out, b)
			}
		case 1:
			switch b {
			case '[':
				f.state = 2
			case ']':
				f.state = 3
			case 'P', 'X', '^', '_':
				f.state = 4
			default:
				if b >= 0x20 && b <= 0x2f {
					f.state = 6
				} else {
					f.state = 0
				}
			}
		case 2:
			if b >= 0x40 && b <= 0x7e {
				f.state = 0
			}
		case 3:
			if b == 7 {
				f.state = 0
			} else if b == 27 {
				f.state = 5
			}
		case 4:
			if b == 27 {
				f.state = 5
			}
		case 5:
			if b == '\\' {
				f.state = 0
			} else if b != 27 {
				f.state = 4
			}
		case 6:
			if b >= 0x30 && b <= 0x7e {
				f.state = 0
			}
		}
	}
	return out
}

func (m *Manager) FollowLogs(ctx context.Context, ref string, raw bool, tail int, w io.Writer) error {
	if tail < 0 {
		return errors.New("tail must be non-negative")
	}
	r, err := m.Resolve(ctx, ref)
	if err != nil {
		return err
	}
	path, _ := m.Store.Path(r.Spec.ID, "output.log")
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	first := true
	filter := logFilter{}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if file == nil {
			info, statErr := os.Lstat(path)
			if statErr == nil {
				if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
					return errors.New("unsafe output log")
				}
				file, err = os.Open(path)
				if err != nil {
					return err
				}
				if first && tail > 0 && info.Size() > int64(tail) {
					if _, err := file.Seek(-int64(tail), io.SeekEnd); err != nil {
						return err
					}
				}
				first = false
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return statErr
			}
		}
		if file != nil {
			data, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			if !raw {
				data = filter.clean(data)
			}
			if len(data) > 0 {
				if _, err := w.Write(data); err != nil {
					return err
				}
			}
			current, statErr := os.Stat(path)
			opened, openErr := file.Stat()
			if statErr == nil && openErr == nil && !os.SameFile(current, opened) {
				file.Close()
				file = nil
				continue
			}
		}
		latest, err := m.Store.Load(ctx, r.Spec.ID)
		if err != nil {
			return err
		}
		if latest.Runtime != nil && (latest.Runtime.State == "exited" || latest.Runtime.State == "failed") {
			// Runtime is committed after the output pump drains; read once more before returning.
			if file != nil {
				data, err := io.ReadAll(file)
				if err != nil {
					return err
				}
				if !raw {
					data = filter.clean(data)
				}
				_, err = w.Write(data)
				return err
			}
			return nil
		}
		observed := m.Observe(ctx, latest)
		if observed.Health == "unreachable" {
			return errors.New("broker unreachable; follow stopped")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
