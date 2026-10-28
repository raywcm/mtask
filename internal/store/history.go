package store

import (
	"context"
	"errors"
	"github.com/raywcm/taskdock/internal/config"
	"os"
	"path/filepath"
	"time"
)

// OpenExisting lets maintenance exit if its state directory was removed, without
// recreating a deleted test directory or uninstalled state in the background.
func OpenExisting(root string) (*JSON, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("state directory must be absolute")
	}
	if err := checkPrivateDir(root); err != nil {
		return nil, err
	}
	if err := checkPrivateDir(filepath.Join(root, "tasks")); err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	return &JSON{root: canonical}, nil
}
func (s *JSON) HistoryLock() (func(), error) {
	return lock(context.Background(), filepath.Join(s.root, "history.lock"), false)
}

// Read only retention policy: an older worker must not fail just because a newer
// frontend selected a language or another preference it does not recognize.
func (s *JSON) HistoryRetention() (time.Duration, error) {
	settings := config.Defaults()
	err := readJSON(filepath.Join(s.root, "settings.json"), &settings)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	return settings.Retention()
}
