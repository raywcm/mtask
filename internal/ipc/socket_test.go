package ipc

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/raywcm/mtask/internal/protocol"
	"github.com/raywcm/mtask/internal/task"
)

func TestOldBrokerCannotSilentlyGrantReadOnlyInputControl(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	state, err := os.MkdirTemp("/tmp", "mtask-old-broker-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(state)
	id, _ := task.NewID()
	instance, _ := task.NewID()
	r := task.Runtime{TaskID: id, InstanceID: instance}
	path, err := SocketPath(filepath.Join(state, "state"), id)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, err := listener.AcceptUnix()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		frame, err := protocol.Read(conn)
		if err != nil {
			done <- err
			return
		}
		var hello protocol.HelloMessage
		if err = Decode(frame.Payload, &hello); err != nil {
			done <- err
			return
		}
		if hello.Mode != "control" {
			done <- errors.New("requested attachment before negotiating safety")
			return
		}
		done <- protocol.NewEncoder(conn).Write(JSON(protocol.Welcome, protocol.WelcomeMessage{TaskID: id, InstanceID: instance}))
	}()
	session, err := ConnectAttachment(context.Background(), filepath.Join(state, "state"), r, &task.Size{Rows: 24, Cols: 80}, protocol.AttachReadOnly)
	if session != nil {
		session.Close()
		t.Fatal("old broker was allowed to attach")
	}
	var remote *RemoteError
	if !errors.As(err, &remote) || remote.Code != "unsupported_attach_mode" {
		t.Fatalf("wrong error: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Dir(path))
}
