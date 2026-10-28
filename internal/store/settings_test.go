package store

import (
	"context"
	"github.com/raywcm/mtask/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTripPermissionsAndValidation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defaults, err := s.Settings()
	if err != nil || defaults != config.Defaults() {
		t.Fatalf("defaults: %+v %v", defaults, err)
	}
	changed := defaults
	changed.Language = "zh-CN"
	changed.Theme = "light"
	changed.DefaultTab = "stop"
	changed.ShowResources = false
	if err := s.SaveSettings(context.Background(), changed); err != nil {
		t.Fatal(err)
	}
	again, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := again.Settings()
	if err != nil || loaded != changed {
		t.Fatalf("preferences did not survive reopen: %+v %v", loaded, err)
	}
	info, err := os.Stat(filepath.Join(root, "settings.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("settings not private")
	}
	invalid := changed
	invalid.RefreshSeconds = 0
	if err := s.SaveSettings(context.Background(), invalid); err == nil {
		t.Fatal("accepted invalid interval")
	}
	loaded, _ = again.Settings()
	if loaded != changed {
		t.Fatal("invalid update damaged saved preferences")
	}
}

func TestOldPreferencesUseDefaultRetention(t *testing.T) {
	repository, err := Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository.Root(), "settings.json"), []byte(`{"schema_version":1,"language":"en"}`), 0600); err != nil {
		t.Fatal(err)
	}
	settings, err := repository.Settings()
	if err != nil || settings.HistoryRetentionSeconds != 3600 {
		t.Fatal("old settings lost default retention", settings, err)
	}
}
