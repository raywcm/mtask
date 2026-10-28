// Package history manages stopped-session retention independently of PTY brokers.
package history

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/raywcm/mtask/internal/ipc"
	"github.com/raywcm/mtask/internal/store"
	"github.com/raywcm/mtask/internal/task"
)

// Ensure starts at most one effective worker per state directory. A racing
// startup is harmless: the worker itself owns the exclusive history lock.
func Ensure(repository *store.JSON) error {
	retention, err := repository.HistoryRetention()
	if err != nil || retention == 0 {
		return err
	}
	records, err := repository.List(context.Background())
	if err != nil || len(records) == 0 {
		return err
	}
	release, err := repository.HistoryLock()
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return nil
	}
	if err != nil {
		return err
	}
	release()
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(repository.Root(), "history.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	null, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer null.Close()
	cmd := exec.Command(exe, "--state-dir", repository.Root(), "__history")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Run keeps timers alive after the frontend exits. No task signals are sent.
func Run(ctx context.Context, state string) error {
	repository, err := store.OpenExisting(state)
	if err != nil {
		return err
	}
	release, err := repository.HistoryLock()
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return nil
	}
	if err != nil {
		return err
	}
	release = sync.OnceFunc(release)
	defer release()
	for {
		pending, err := Sweep(ctx, repository, time.Now())
		if err != nil {
			return err
		}
		if !pending {
			// Coordinate the final empty/disabled check with task creation and setting
			// writes. Release the worker lock before the catalog lock so Ensure cannot
			// miss a worker that is about to exit after publishing a new task.
			unlock, err := repository.Lock(ctx)
			if err != nil {
				return err
			}
			retention, readErr := repository.HistoryRetention()
			records, listErr := repository.List(ctx)
			done := readErr == nil && listErr == nil && (retention == 0 || len(records) == 0)
			if done {
				release()
			}
			unlock()
			if readErr != nil {
				return readErr
			}
			if listErr != nil {
				return listErr
			}
			if done {
				return nil
			}
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Sweep rechecks policy under the catalog lock. Only terminal records whose
// broker lock is free can be removed; stale running PIDs are never consulted.
func Sweep(ctx context.Context, repository *store.JSON, now time.Time) (bool, error) {
	unlock, err := repository.Lock(ctx)
	if err != nil {
		return false, err
	}
	defer unlock()
	retention, err := repository.HistoryRetention()
	if err != nil || retention == 0 {
		return false, err
	}
	records, err := repository.List(ctx)
	if err != nil {
		return false, err
	}
	remaining := len(records)
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return true, err
		}
		ended := record.Spec.CreatedAt
		if r := record.Runtime; r != nil {
			if r.State != task.Exited && r.State != task.Failed {
				continue
			}
			ended = r.UpdatedAt
			if r.FinishedAt != nil && !r.FinishedAt.IsZero() {
				ended = *r.FinishedAt
			}
		}
		if ended.IsZero() || now.Before(ended.Add(retention)) {
			continue
		}
		release, err := repository.BrokerLock(record.Spec.ID)
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			continue
		}
		if err != nil {
			return true, err
		}
		err = func() error {
			defer release()
			socket, err := ipc.SocketPath(repository.Root(), record.Spec.ID)
			if err != nil {
				return err
			}
			if info, err := os.Lstat(socket); err == nil {
				if info.Mode()&os.ModeSocket == 0 {
					return errors.New("refusing to remove non-socket history path")
				}
				if err := os.Remove(socket); err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return repository.Remove(ctx, record.Spec.ID)
		}()
		if err != nil {
			return true, fmt.Errorf("clean stopped session %s: %w", record.Spec.ID, err)
		}
		remaining--
	}
	return remaining > 0, nil
}
