package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsPersistLocalizeHelpAndPreserveJSON(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	run := func(args ...string) string {
		t.Helper()
		root := NewRoot()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs(append([]string{"--state-dir", state}, args...))
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	run("settings", "set", "language", "zh-CN")
	if text := run("--help"); !strings.Contains(text, "用法") || !strings.Contains(text, "MTasK") {
		t.Fatalf("help was not localized/branded: %s", text)
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(run("settings", "--json")), &settings); err != nil {
		t.Fatal(err)
	}
	if settings["language"] != "zh-CN" {
		t.Fatal("language did not persist")
	}
	run("settings", "set", "language", "en")
	if text := run("--help"); !strings.Contains(text, "Usage:") || !strings.Contains(text, "Settings") {
		if !strings.Contains(text, "interface preferences") {
			t.Fatalf("English help missing: %s", text)
		}
	}
	if text := run("--version"); text != "MTasK v"+Version+"\n" {
		t.Fatalf("version branding: %q", text)
	}
	if text := run("list", "--json"); text != "[]\n" {
		t.Fatal("localized machine JSON")
	}
}

func TestAdditionalLanguagesPersistAndLocalizeCLI(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	run := func(args ...string) string {
		t.Helper()
		root := NewRoot()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs(append([]string{"--state-dir", state}, args...))
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	for code, usage := range map[string]string{"ja": "使い方", "es": "Uso:", "fr": "Utilisation"} {
		run("settings", "set", "language", code)
		help := run("--help")
		if !strings.Contains(help, usage) || strings.Contains(help, "Independent terminal task manager") {
			t.Fatalf("%s CLI did not localize: %s", code, help)
		}
		var settings map[string]any
		if err := json.Unmarshal([]byte(run("settings", "--json")), &settings); err != nil {
			t.Fatal(err)
		}
		if settings["language"] != code {
			t.Fatal("language did not persist", code)
		}
		if text := run("list", "--json"); text != "[]\n" {
			t.Fatal("translated JSON schema", text)
		}
	}
}
