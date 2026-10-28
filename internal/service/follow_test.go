package service

import (
	"bytes"
	"testing"
)

func TestLogFilterAcrossReadBoundaries(t *testing.T) {
	input := []byte("hello\x1b[31m中文\x1b[0m\x1b]52;c;SECRET\x1b\\\x1bPPRIVATE\x1b\\\n")
	var f logFilter
	var output []byte
	for _, b := range input {
		output = append(output, f.clean([]byte{b})...)
	}
	if !bytes.Equal(output, []byte("hello中文\n")) {
		t.Fatalf("unsafe or broken log: %q", output)
	}
}
