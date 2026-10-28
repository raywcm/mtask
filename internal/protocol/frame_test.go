package protocol

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"
)

type fragmentWriter struct{ bytes.Buffer }

func (w *fragmentWriter) Write(p []byte) (int, error) {
	return w.Buffer.Write(p[:min(3, len(p))])
}

func TestOpaqueBytesWithShortWrites(t *testing.T) {
	input := []byte{0, 0xff, 0x1b, '[', '?', '1', '0', '4', '9', 'h', '\r', '\n'}
	var wire fragmentWriter
	if err := NewEncoder(&wire).Write(Frame{Kind: Output, Payload: input}); err != nil {
		t.Fatal(err)
	}
	got, err := Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != Output || !bytes.Equal(got.Payload, input) {
		t.Fatalf("changed payload: %#v", got)
	}
}

func TestRejectMalformedFrames(t *testing.T) {
	cases := map[string][]byte{
		"magic":     {'X', 'T', Version, byte(Input), 0, 0, 0, 0},
		"version":   {'M', 'T', 99, byte(Input), 0, 0, 0, 0},
		"kind":      {'M', 'T', Version, 255, 0, 0, 0, 0},
		"huge":      {'M', 'T', Version, byte(Output), 255, 255, 255, 255},
		"truncated": {'M', 'T', Version, byte(Input), 0, 0, 0, 2, 1},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Read(bytes.NewReader(raw)); err == nil {
				t.Fatal("accepted malformed frame")
			}
		})
	}
	var header [8]byte
	copy(header[:], []byte{'M', 'T', Version, byte(Hello)})
	binary.BigEndian.PutUint32(header[4:], MaxControlPayload+1)
	if _, err := Read(bytes.NewReader(header[:])); err == nil {
		t.Fatal("accepted oversized control frame")
	}
}

func TestConcurrentFramesDoNotInterleave(t *testing.T) {
	var wire bytes.Buffer
	encoder := NewEncoder(&wire)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := encoder.Write(Frame{Kind: Output, Payload: bytes.Repeat([]byte{byte(i)}, 100)}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	seen := map[byte]bool{}
	for wire.Len() > 0 {
		f, err := Read(&wire)
		if err != nil {
			t.Fatal(err)
		}
		if len(f.Payload) != 100 || !bytes.Equal(f.Payload, bytes.Repeat(f.Payload[:1], 100)) {
			t.Fatal("interleaved frames")
		}
		seen[f.Payload[0]] = true
	}
	if len(seen) != 20 {
		t.Fatalf("received %d frames", len(seen))
	}
}

func FuzzRead(f *testing.F) {
	f.Add([]byte{'M', 'T', Version, byte(Input), 0, 0, 0, 1, 0xff})
	f.Add([]byte("invalid"))
	f.Fuzz(func(t *testing.T, data []byte) {
		frame, err := Read(bytes.NewReader(data))
		if err != nil {
			return
		}
		var b bytes.Buffer
		if err := NewEncoder(&b).Write(frame); err != nil {
			t.Fatal(err)
		}
		decoded, err := Read(&b)
		if err != nil || decoded.Kind != frame.Kind || !bytes.Equal(decoded.Payload, frame.Payload) {
			t.Fatalf("round trip mismatch: %v", err)
		}
	})
}
