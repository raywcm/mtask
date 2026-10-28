package terminal

import (
	"bytes"
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
