// Package terminal owns the local terminal only while attached. No Bubble Tea
// model, ANSI parser, screen buffer, or renderer belongs in this package.
package terminal

import (
	"errors"
	"os"
	"strconv"
	"strings"

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

// EscapeDecoder handles detach keys without decoding application output.
// Unknown input is returned unchanged; incomplete escape sequences are bounded.
type EscapeDecoder struct {
	Disable bool
	pending uint16
	prefix  []byte
	escape  []byte
	pasting bool
}

const inputEscapeLimit = 64

func (d *EscapeDecoder) Feed(input []byte) (output []byte, detach bool) {
	if d.Disable {
		return append([]byte(nil), input...), false
	}
	for index, b := range input {
		if len(d.escape) > 0 {
			d.escape = append(d.escape, b)
			if len(d.escape) == 2 && b == '[' {
				continue
			}
			if len(d.escape) > 2 && b < 0x40 && len(d.escape) < inputEscapeLimit {
				continue
			}
			part, done := d.consume(d.escape, escapeKey(d.escape))
			d.escape = nil
			output = append(output, part...)
			if done {
				return output, true
			}
			continue
		}
		if b == 0x1b {
			d.escape = []byte{b}
			continue
		}
		part, done := d.consume(input[index:index+1], uint16(b))
		output = append(output, part...)
		if done {
			return output, true
		}
	}
	return output, false
}

const (
	keyF12 uint16 = 256 + iota
	keyPasteStart
	keyPasteEnd
	keyRelease
)

func (d *EscapeDecoder) consume(raw []byte, key uint16) ([]byte, bool) {
	if d.pasting {
		if key == keyPasteEnd {
			d.pasting = false
		}
		return raw, false
	}
	if key == keyF12 {
		d.pending, d.prefix = 0, nil
		return nil, true
	}
	if key == keyPasteStart {
		d.pasting = true
	}
	if d.pending != 0 {
		if key == keyRelease && len(d.prefix)+len(raw) <= 256 {
			d.prefix = append(d.prefix, raw...)
			return nil, false
		}
		prefix, saved := d.pending, d.prefix
		d.pending, d.prefix = 0, nil
		if key == 'd' || (prefix == 0x02 && key == 0x04) {
			return nil, true
		}
		if key == prefix {
			return saved, false
		}
		return append(saved, raw...), false
	}
	if !d.pasting && (key == 0x02 || key == 0x1d) {
		d.pending = key
		d.prefix = append([]byte(nil), raw...)
		return nil, false
	}
	return raw, false
}

// escapeKey recognizes unmodified F12 and the existing prefix keys in CSI-u
// and modifyOtherKeys encodings. Modified F12 and release events pass through.
func escapeKey(raw []byte) uint16 {
	s := string(raw)
	if s == "\x1b[200~" {
		return keyPasteStart
	}
	if s == "\x1b[201~" {
		return keyPasteEnd
	}
	if len(s) < 4 || !strings.HasPrefix(s, "\x1b[") {
		return 0
	}
	final := s[len(s)-1]
	if final != '~' && final != 'u' {
		return 0
	}
	parts := strings.Split(s[2:len(s)-1], ";")
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	modifiers := 1
	if len(parts) > 1 {
		fields := strings.Split(parts[1], ":")
		if len(fields) > 2 {
			return 0
		}
		modifiers, err = strconv.Atoi(fields[0])
		if err != nil || modifiers < 1 {
			return 0
		}
		if len(fields) == 2 {
			if fields[1] == "3" {
				return keyRelease
			}
			if fields[1] != "1" && fields[1] != "2" {
				return 0
			}
		}
	}
	// Caps/Num lock do not change these shortcuts.
	mods := (modifiers - 1) &^ (64 | 128)
	if final == '~' && code == 24 && len(parts) <= 2 && mods == 0 {
		return keyF12
	}
	if final == '~' && code == 27 && len(parts) == 3 {
		code, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0
		}
	} else if final != 'u' || len(parts) > 2 {
		return 0
	}
	if mods == 4 {
		switch code {
		case 98:
			return 0x02
		case 100:
			return 0x04
		case 93:
			return 0x1d
		}
	}
	if code == 100 && mods == 0 {
		return 'd'
	}
	return 0
}

// flushEscape releases a standalone Esc or a truncated sequence after a short
// input deadline. Ctrl+B itself remains pending until the next key.
func (d *EscapeDecoder) flushEscape() []byte {
	if len(d.escape) == 0 {
		return nil
	}
	out, _ := d.consume(d.escape, 0)
	d.escape = nil
	return out
}

func (d *EscapeDecoder) Flush() []byte {
	out := d.flushEscape()
	if d.pending != 0 {
		out = append(out, d.prefix...)
		d.pending, d.prefix = 0, nil
	}
	return out
}
