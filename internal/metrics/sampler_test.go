package metrics

import (
	"context"
	"github.com/raywcm/mtask/internal/task"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestLiveProcessSamplesMemoryAndCPU(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "(while :; do :; done) & wait")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); _ = cmd.Wait() }()
	id, _ := task.NewID()
	records := []task.Record{{Spec: task.Spec{ID: id}, Health: "online", Runtime: &task.Runtime{ChildPID: cmd.Process.Pid, BrokerPID: os.Getpid()}}}
	sampler := New()
	first := sampler.Sample(context.Background(), records)
	if first[id].MemoryBytes == 0 || first[id].CPUAvailable {
		t.Fatalf("missing memory or fabricated first CPU sample: %+v", first[id])
	}
	if first[id].ProcessCount < 2 {
		t.Fatalf("did not include shell's child process: %+v", first[id])
	}
	time.Sleep(200 * time.Millisecond)
	second := sampler.Sample(context.Background(), records)
	if !second[id].CPUAvailable || second[id].CPUPercent <= 0 {
		t.Fatalf("busy process CPU not sampled: %+v", second[id])
	}
	records[0].Runtime.BrokerPID = -1
	if len(sampler.Sample(context.Background(), records)) != 0 {
		t.Fatal("reported reused/unrelated process")
	}
}
