package i18n

import (
	"testing"
)

func TestAllSupportedCatalogsAreComplete(t *testing.T) {
	if len(Languages()) < 5 {
		t.Fatal("at least five languages required")
	}
	base := catalogs["en"]
	for _, option := range Languages() {
		c, ok := catalogs[option.Code]
		if !ok {
			t.Fatalf("missing catalog %s", option.Code)
		}
		for key := range base.Messages {
			if c.Messages[key] == "" {
				t.Fatalf("%s missing message %s", option.Code, key)
			}
		}
		for key := range base.CLI {
			if c.CLI[key] == "" {
				t.Fatalf("%s missing CLI text %s", option.Code, key)
			}
		}
	}
	for code, label := range map[string]string{"en": "Settings", "zh-CN": "设置", "ja": "設定", "es": "Ajustes", "fr": "Paramètres"} {
		if New(code).Text("settings") != label {
			t.Fatalf("%s locale selection failed", code)
		}
	}
	if New("unknown").Text("settings") != "Settings" {
		t.Fatal("missing English fallback")
	}
}

func TestFutureLanguageRegistrationUsesMetadata(t *testing.T) {
	catalogs["zz"] = catalog{Code: "zz", Name: "Future language", Aliases: []string{"future"}, Order: 1000, Messages: map[string]string{"settings": "Future settings"}}
	defer delete(catalogs, "zz")
	if !SupportedLanguage("zz") || LanguageName("zz") != "Future language" {
		t.Fatal("registry was hardcoded")
	}
	t.Setenv("LC_ALL", "future_REGION.UTF-8")
	if Language("auto") != "zz" || New("zz").Text("settings") != "Future settings" {
		t.Fatal("future locale did not auto-detect")
	}
	if New("zz").Text("help") != "help" {
		t.Fatal("English key fallback failed")
	}
	found := false
	for _, code := range LanguageCodes() {
		if code == "zz" {
			found = true
		}
	}
	if !found {
		t.Fatal("future language missing from automatic menu")
	}
}
