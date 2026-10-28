package terminal

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestTTYWaitTimeoutReadinessAndInvalidDescriptor(t *testing.T) {
	var pipe [2]int
	if err := unix.Pipe(pipe[:]); err != nil {
		t.Fatal(err)
	}
	defer unix.Close(pipe[0])
	defer unix.Close(pipe[1])
	start := time.Now()
	ready, err := waitTTY(pipe[0], unix.POLLIN, 25)
	if err != nil || ready || time.Since(start) < 20*time.Millisecond {
		t.Fatalf("empty input did not wait: ready=%v err=%v elapsed=%v", ready, err, time.Since(start))
	}
	if _, err := unix.Write(pipe[1], []byte("x")); err != nil {
		t.Fatal(err)
	}
	if ready, err = waitTTY(pipe[0], unix.POLLIN, 100); err != nil || !ready {
		t.Fatalf("input not readable: %v %v", ready, err)
	}
	if ready, err = waitTTY(-1, unix.POLLIN, 0); err == nil || ready {
		t.Fatalf("invalid FD accepted: %v %v", ready, err)
	}
	if !errors.Is(err, unix.EBADF) {
		t.Fatalf("invalid FD error: %v", err)
	}
}
