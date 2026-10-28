package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// StateDir follows the same convention on Linux and macOS. Runtime socket paths
// and their ownership checks are managed by the ipc package.
func StateDir(override string) (string, error) {
	if override == "" {
		override = os.Getenv("TDOCK_STATE_DIR")
	}
	if override == "" {
		override = os.Getenv("MTASK_STATE_DIR") // Existing installations keep their state.
	}
	if override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("state directory must be absolute")
		}
		return filepath.Clean(override), nil
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("XDG_STATE_HOME must be absolute")
	}
	preferred := filepath.Join(base, "taskdock")
	if _, err := os.Lstat(preferred); err == nil {
		return preferred, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	// Live brokers retain absolute storage paths. Reuse their directory in
	// place; moving it would interrupt their log and metadata writes.
	legacy := filepath.Join(base, "mtask")
	if info, err := os.Lstat(legacy); err == nil && info.IsDir() {
		return legacy, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return preferred, nil
}
