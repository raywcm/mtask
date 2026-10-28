package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateDirUpgrade(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv("TDOCK_STATE_DIR", "")
	t.Setenv("MTASK_STATE_DIR", "")
	check := func(want string) {
		t.Helper()
		got, err := StateDir("")
		if err != nil || got != want {
			t.Fatalf("StateDir() = %q, %v; want %q", got, err, want)
		}
	}
	preferred, legacy := filepath.Join(base, "taskdock"), filepath.Join(base, "mtask")
	check(preferred)
	if err := os.Mkdir(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	check(legacy)
	if err := os.Mkdir(preferred, 0700); err != nil {
		t.Fatal(err)
	}
	check(preferred)
	t.Setenv("MTASK_STATE_DIR", filepath.Join(base, "old-override"))
	check(filepath.Join(base, "old-override"))
	t.Setenv("TDOCK_STATE_DIR", filepath.Join(base, "new-override"))
	check(filepath.Join(base, "new-override"))
	if got, err := StateDir(base); err != nil || got != base {
		t.Fatalf("explicit path lost priority: %q %v", got, err)
	}
	t.Setenv("TDOCK_STATE_DIR", "relative")
	if _, err := StateDir(""); err == nil {
		t.Fatal("relative override accepted")
	}
}

func TestStateDirDoesNotFollowLegacySymlink(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv("TDOCK_STATE_DIR", "")
	t.Setenv("MTASK_STATE_DIR", "")
	if err := os.Symlink(t.TempDir(), filepath.Join(base, "mtask")); err != nil {
		t.Fatal(err)
	}
	got, err := StateDir("")
	if err != nil || got != filepath.Join(base, "taskdock") {
		t.Fatalf("legacy symlink selected: %q %v", got, err)
	}
}
