package store

import (
	"context"
	"errors"
	"github.com/raywcm/taskdock/internal/config"
	"os"
	"path/filepath"
)

func ReadSettings(root string) (config.Settings, error) {
	settings := config.Defaults()
	err := readJSON(filepath.Join(root, "settings.json"), &settings)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return config.Defaults(), err
	}
	return settings, settings.Validate()
}
func (s *JSON) Settings() (config.Settings, error) { return ReadSettings(s.root) }
func (s *JSON) SaveSettings(ctx context.Context, settings config.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	release, err := s.Lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	return writeJSON(filepath.Join(s.root, "settings.json"), settings)
}
