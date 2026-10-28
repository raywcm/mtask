package broker

import (
	"context"
	"io"

	"github.com/raywcm/taskdock/internal/task"
)

// Process is the OS boundary. Only the supervisor calls Wait, exactly once.
// Close closes the PTY; it is not a substitute for terminating/reaping the child.
type Process interface {
	io.ReadWriteCloser
	PID() int
	Resize(task.Size) error
	Wait() (ExitStatus, error)
}

type ExitStatus struct {
	Code   int
	Signal string
}

type SpawnOptions struct {
	Spec task.Spec
	Size task.Size
	Env  []string // explicit launch-time environment, never stored in spec.json
}

type Spawner interface {
	Start(SpawnOptions) (Process, error)
}

// Launcher describes re-executing the same binary as __broker in a new session
// with a readiness pipe. Canceling launch only cancels waiting;
// callers must reconcile the task ID before retrying after an ambiguous result.
type Launcher interface {
	Launch(context.Context, task.ID) (task.Runtime, error)
}

// Control describes authenticated Unix-socket operations. It never signals a PID
// read from disk. Peer UID, task ID, instance ID, and protocol version must match.
type Control interface {
	Probe(context.Context, task.ID) (task.Runtime, error)
	Stop(context.Context, task.ID) error
}
