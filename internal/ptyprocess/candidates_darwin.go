package ptyprocess

import (
	"golang.org/x/sys/unix"
	"os"
)

func sessionCandidates() ([]int, error) {
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.uid", os.Getuid())
	if err != nil {
		return nil, err
	}
	pids := make([]int, 0, len(processes))
	for _, p := range processes {
		pids = append(pids, int(p.Proc.P_pid))
	}
	return pids, nil
}
