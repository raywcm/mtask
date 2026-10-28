package config

import (
	"errors"
	"fmt"
	"github.com/raywcm/taskdock/internal/i18n"
	"strings"
	"time"
)

type Settings struct {
	SchemaVersion           int    `json:"schema_version"`
	Language                string `json:"language"`
	Theme                   string `json:"theme"`
	RefreshSeconds          int    `json:"refresh_seconds"`
	ShowResources           bool   `json:"show_resources"`
	DefaultTab              string `json:"default_tab"`
	HistoryRetentionSeconds int64  `json:"history_retention_seconds"`
}

func Defaults() Settings {
	return Settings{SchemaVersion: 1, Language: "auto", Theme: "auto", RefreshSeconds: 2, ShowResources: true, DefaultTab: "running", HistoryRetentionSeconds: 3600}
}
func (s Settings) Validate() error {
	if s.SchemaVersion != 1 {
		return errors.New("unsupported settings schema")
	}
	if !SupportedLanguage(s.Language) {
		return fmt.Errorf("language must be one of %s", strings.Join(LanguageCodes(), ", "))
	}
	if s.Theme != "auto" && s.Theme != "dark" && s.Theme != "light" {
		return errors.New("theme must be auto, dark or light")
	}
	if s.RefreshSeconds != 1 && s.RefreshSeconds != 2 && s.RefreshSeconds != 5 {
		return errors.New("refresh interval must be 1, 2 or 5 seconds")
	}
	if s.DefaultTab != "running" && s.DefaultTab != "stop" {
		return errors.New("default tab must be running or stop")
	}
	if _, err := s.Retention(); err != nil {
		return err
	}
	return nil
}

func (s Settings) Retention() (time.Duration, error) {
	if s.HistoryRetentionSeconds < 0 || s.HistoryRetentionSeconds > 365*24*3600 {
		return 0, errors.New("history retention must be between 0 and 365 days")
	}
	return time.Duration(s.HistoryRetentionSeconds) * time.Second, nil
}

type LanguageOption = i18n.LanguageOption

func Languages() []LanguageOption        { return i18n.Languages() }
func LanguageCodes() []string            { return i18n.LanguageCodes() }
func SupportedLanguage(code string) bool { return i18n.SupportedLanguage(code) }
func LanguageName(code string) string    { return i18n.LanguageName(code) }
func Language(setting string) string     { return i18n.Language(setting) }

func With(s Settings, key, value string) (Settings, error) {
	switch key {
	case "language":
		s.Language = value
	case "theme":
		s.Theme = value
	case "refresh":
		switch value {
		case "1":
			s.RefreshSeconds = 1
		case "2":
			s.RefreshSeconds = 2
		case "5":
			s.RefreshSeconds = 5
		default:
			return s, errors.New("refresh must be 1, 2 or 5")
		}
	case "resources":
		switch value {
		case "on", "true":
			s.ShowResources = true
		case "off", "false":
			s.ShowResources = false
		default:
			return s, errors.New("resources must be on or off")
		}
	case "history-retention":
		if value == "off" || value == "never" || value == "0" {
			s.HistoryRetentionSeconds = 0
		} else {
			duration, err := time.ParseDuration(value)
			if err != nil || duration < time.Second || duration%time.Second != 0 {
				return s, errors.New("history-retention must be a whole duration such as 30m, 1h, 24h, or off")
			}
			s.HistoryRetentionSeconds = int64(duration / time.Second)
		}
	case "default-tab":
		s.DefaultTab = value
	default:
		return s, fmt.Errorf("unknown setting %q", key)
	}
	return s, s.Validate()
}
