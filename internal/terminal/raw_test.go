package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestEscapeAcrossReadBoundaries(t *testing.T) {
	d := EscapeDecoder{}
	first, detached := d.Feed([]byte{'a', 0x1d})
	if detached || string(first) != "a" {
		t.Fatal("incorrect partial prefix")
	}
	second, detached := d.Feed([]byte{'d', 'x'})
	if !detached || len(second) != 0 {
		t.Fatal("detach or discard suffix failed")
	}
}

func TestOrdinaryInputUnchanged(t *testing.T) {
	input := []byte("\x03\x1b[<0;10;12M\x1b[200~中文\x1b[201~\x00\xff")
	d := EscapeDecoder{}
	var output []byte
	for _, b := range input {
		part, detach := d.Feed([]byte{b})
		if detach {
			t.Fatal("unexpected detach")
		}
		output = append(output, part...)
	}
	if !bytes.Equal(output, input) {
		t.Fatalf("input changed: %q", output)
	}
}

func TestLiteralPrefixAndBypass(t *testing.T) {
	d := EscapeDecoder{}
	out, detached := d.Feed([]byte{0x1d, 0x1d, 0x1d, 'x', 0x1d})
	out = append(out, d.Flush()...)
	if detached || !bytes.Equal(out, []byte{0x1d, 0x1d, 'x', 0x1d}) {
		t.Fatalf("literal prefix: %v", out)
	}
	d = EscapeDecoder{Disable: true}
	input := []byte{0x1d, 'd'}
	out, detached = d.Feed(input)
	if detached || !bytes.Equal(out, input) {
		t.Fatal("bypass transformed bytes")
	}
}

func TestBothDetachPrefixesAcrossBoundaries(t *testing.T) {
	for _, prefix := range []byte{0x02, 0x1d} {
		d := EscapeDecoder{}
		first, detached := d.Feed([]byte{'a', prefix})
		if detached || string(first) != "a" {
			t.Fatalf("prefix %x: lost input", prefix)
		}
		last, detached := d.Feed([]byte{'d', 'x'})
		if !detached || len(last) != 0 {
			t.Fatalf("prefix %x: did not detach", prefix)
		}
		d = EscapeDecoder{}
		input := []byte{prefix, prefix, prefix, 'x', prefix}
		out, detached := d.Feed(input)
		out = append(out, d.Flush()...)
		if detached || !bytes.Equal(out, []byte{prefix, prefix, 'x', prefix}) || d.Flush() != nil {
			t.Fatalf("prefix %x: literal or flush changed: %v", prefix, out)
		}
	}
}

func TestMixedPrefixesAndDisabledInputRemainLiteral(t *testing.T) {
	input := []byte{0x02, 0x1d, 'd', 0x1d, 0x02, 'x'}
	d := EscapeDecoder{}
	out, detached := d.Feed(input)
	if detached || !bytes.Equal(out, input) {
		t.Fatalf("unrecognized mixed prefix changed: %v", out)
	}
	d = EscapeDecoder{Disable: true}
	input = []byte{0x02, 'd', 0x1d, 'd'}
	for _, b := range input {
		part, detach := d.Feed([]byte{b})
		if detach || !bytes.Equal(part, []byte{b}) {
			t.Fatal("no-escape transformed a byte")
		}
	}
}

func TestDetachKeysAtEveryReadBoundary(t *testing.T) {
	for _, key := range []string{
		"\x1b[24~", "\x1b[24;1~", "\x1b[24;1:1~", "\x1b[24;1:2~", "\x1b[24;65~",
		"\x02d", "\x02\x04", "\x1dd",
		"\x1b[98;5ud", "\x1b[98;5u\x1b[100;5u",
		"\x1b[98;5u\x1b[98;5:3u\x1b[57442;1:3ud",
		"\x1b[27;5;98~\x1b[27;5;100~",
	} {
		for split := 0; split <= len(key); split++ {
			d := EscapeDecoder{}
			out, detached := d.Feed([]byte(key[:split]))
			if !detached {
				part, done := d.Feed([]byte(key[split:]))
				out = append(out, part...)
				detached = done
			}
			if !detached || len(out) != 0 {
				t.Fatalf("key %q split %d: detached=%v output=%q", key, split, detached, out)
			}
		}
	}
}

func TestDetachDecoderPreservesOtherInput(t *testing.T) {
	for _, input := range []string{
		"\x04", "\x1b[100;5u", "\x1b[27;5;100~", // Ctrl+D without a prefix.
		"\x1b[24;2~\x1b[24;5~\x1b[24;1:3~",       // Modified/released F12.
		"\x1b[A\x1b[1;5D\x1b[<0;10;12M\x1b[1;2R", // Other terminal input.
		"中文🙂\x00\xff\xf0\xf1\xf2\xf3", "\x1bx", "\x1b\x1b", "\x1b",
		"\x1b[200~pasted:\x1b[24~\x02d\x02\x04\x1dd\x04\x1b[201~",
		"\x1b[" + strings.Repeat("1", 1000) + "~",
	} {
		for width := 1; width <= len(input); width++ {
			d := EscapeDecoder{}
			var output []byte
			for start := 0; start < len(input); start += width {
				part, detached := d.Feed([]byte(input[start:min(start+width, len(input))]))
				if detached {
					t.Fatalf("unexpected detach for %q", input)
				}
				output = append(output, part...)
			}
			output = append(output, d.Flush()...)
			if string(output) != input {
				t.Fatalf("changed %q to %q", input, output)
			}
		}
	}
}

func TestEscapeTimeoutAndPasteExit(t *testing.T) {
	d := EscapeDecoder{}
	if out, done := d.Feed([]byte("\x1b")); len(out) != 0 || done {
		t.Fatal("incomplete escape")
	}
	if string(d.flushEscape()) != "\x1b" || d.Flush() != nil {
		t.Fatal("standalone Esc lost")
	}
	out, done := d.Feed([]byte("[24~\x04"))
	if done || string(out) != "[24~\x04" {
		t.Fatal("expired prefix became a shortcut")
	}
	d = EscapeDecoder{}
	d.Feed([]byte{0x02})
	if d.flushEscape() != nil {
		t.Fatal("Ctrl+B should wait for its second key")
	}
	if _, done = d.Feed([]byte{0x04}); !done {
		t.Fatal("held Ctrl did not detach")
	}
	d = EscapeDecoder{}
	input := "\x1b[200~\x1b[24~\x02d\x1b[201~"
	out, done = d.Feed([]byte(input))
	if done || string(out) != input {
		t.Fatal("paste intercepted")
	}
	if _, done = d.Feed([]byte("\x1b[24~")); !done {
		t.Fatal("paste mode did not end")
	}
}

func TestNoEscapePassesAllDetachKeys(t *testing.T) {
	d := EscapeDecoder{Disable: true}
	input := "\x1b[24~\x02d\x02\x04\x1dd\x04\x1b[98;5u\x1b[100;5u"
	out, done := d.Feed([]byte(input))
	if done || string(out) != input || d.Flush() != nil {
		t.Fatal("no-escape changed input")
	}
}

func TestPendingReleaseBufferIsBounded(t *testing.T) {
	d := EscapeDecoder{}
	input := "\x1b[98;5u" + strings.Repeat("\x1b[98;5:3u", 1000)
	out, done := d.Feed([]byte(input))
	out = append(out, d.Flush()...)
	if done || string(out) != input {
		t.Fatal("release flood lost bytes")
	}
	if len(d.prefix) > 256 || len(d.escape) > inputEscapeLimit {
		t.Fatal("unbounded input state")
	}
}
