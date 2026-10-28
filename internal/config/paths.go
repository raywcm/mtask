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
		override = os.Getenv("MTASK_STATE_DIR")
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
	return filepath.Join(base, "mtask"), nil
}
