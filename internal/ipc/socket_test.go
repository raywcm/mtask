package ipc

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/raywcm/taskdock/internal/protocol"
	"github.com/raywcm/taskdock/internal/task"
)

func TestOldBrokerCannotSilentlyGrantReadOnlyInputControl(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	state, err := os.MkdirTemp("/tmp", "tdock-old-broker-")
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

func TestLegacySocketDiscovery(t *testing.T) {
	// Keep Unix paths short on both macOS and Linux.
	runtime, err := os.MkdirTemp("/tmp", "tdock-ipc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(runtime)
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	id, _ := task.NewID()
	state := filepath.Join(runtime, "state")
	preferred, err := SocketPath(state, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(runtime, "mtask")); !os.IsNotExist(err) {
		t.Fatal("fresh installation created the legacy namespace")
	}
	legacy, err := socketPath(state, id, "mtask", true)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: legacy, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := os.Chmod(legacy, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := SocketPath(state, id); err != nil || got != legacy {
		t.Fatalf("old live socket not found: %q %v", got, err)
	}
	if err := os.Chmod(legacy, 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := SocketPath(state, id); err == nil {
		t.Fatal("unsafe legacy socket accepted")
	}
	if err := os.Remove(legacy); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(preferred, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := SocketPath(state, id); err == nil {
		t.Fatal("legacy socket symlink accepted")
	}
	if err := os.WriteFile(preferred, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := SocketPath(state, id); err != nil || got != preferred {
		t.Fatalf("preferred namespace lost priority: %q %v", got, err)
	}
}
