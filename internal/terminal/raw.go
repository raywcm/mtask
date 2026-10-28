// Package terminal owns the local terminal only while attached. No Bubble Tea
// model, ANSI parser, screen buffer, or renderer belongs in this package.
package terminal

import (
	"errors"
	"os"

	"golang.org/x/term"
)

// WithRaw restores termios on normal return, errors and panic unwinding. The
// attach coordinator must handle HUP/TERM and cancel I/O before returning here.
// SIGKILL cannot run deferred cleanup. ANSI mode cleanup is a separate concern.
func WithRaw(tty *os.File, run func() error) (err error) {
	fd := int(tty.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, term.Restore(fd, state)) }()
	return run()
}

// EscapeDecoder recognizes the tmux-style Ctrl+B d detach and the legacy
// Ctrl+] d alias. Doubling either prefix sends it literally. Disable bypasses
// both prefixes for exact passthrough.
// Pending state is preserved across arbitrary read boundaries.
type EscapeDecoder struct {
	Disable bool
	pending byte
}

func (d *EscapeDecoder) Feed(input []byte) (output []byte, detach bool) {
	if d.Disable {
		return append([]byte(nil), input...), false
	}
	for _, b := range input {
		if d.pending != 0 {
			prefix := d.pending
			d.pending = 0
			if b == 'd' {
				return output, true
			}
			output = append(output, prefix)
			if b != prefix {
				output = append(output, b)
			}
		} else if b == 0x02 || b == 0x1d {
			d.pending = b
		} else {
			output = append(output, b)
		}
	}
	return output, false
}

func (d *EscapeDecoder) Flush() []byte {
	if d.pending == 0 {
		return nil
	}
	prefix := d.pending
	d.pending = 0
	return []byte{prefix}
}
