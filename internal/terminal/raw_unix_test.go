//go:build linux || darwin

package terminal

import (
	"errors"
	"reflect"
	"testing"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func TestRawStateRestoredOnError(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	before, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("attach failed")
	err = WithRaw(slave, func() error {
		during, err := term.GetState(int(slave.Fd()))
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(before, during) {
			t.Fatal("terminal did not enter raw mode")
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("lost callback error: %v", err)
	}
	after, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("terminal settings were not restored")
	}
}
