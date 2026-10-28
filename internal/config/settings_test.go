package config

import "testing"

func TestLanguageDetectionAndOverrides(t *testing.T) {
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LANG", "en_US.UTF-8")
	if Language("auto") != "zh-CN" || Language("en") != "en" {
		t.Fatal("locale priority or override failed")
	}
	t.Setenv("LC_ALL", "")
	if Language("auto") != "en" {
		t.Fatal("English fallback failed")
	}
}

func TestAllLanguagesValidateAndDetectRegionalLocales(t *testing.T) {
	for _, option := range Languages() {
		s := Defaults()
		s.Language = option.Code
		if err := s.Validate(); err != nil {
			t.Fatalf("%s rejected: %v", option.Code, err)
		}
		if LanguageName(option.Code) == "" {
			t.Fatal("missing native language name")
		}
	}
	for input, want := range map[string]string{"ja_JP.UTF-8": "ja", "es_MX.UTF-8": "es", "fr_CA.UTF-8": "fr", "en_GB.UTF-8": "en", "zh_CN.UTF-8": "zh-CN", "it:fr:en": "fr", "C.UTF-8": "en", "it_IT.UTF-8": "en"} {
		t.Setenv("LC_ALL", input)
		if got := Language("auto"); got != want {
			t.Fatalf("%s detected as %s, want %s", input, got, want)
		}
	}
	s := Defaults()
	s.Language = "unknown"
	if s.Validate() == nil {
		t.Fatal("unsupported persisted language accepted")
	}
}

func TestHistoryRetentionDefaultsAndDurations(t *testing.T) {
	if Defaults().HistoryRetentionSeconds != 3600 {
		t.Fatal("default must be one hour")
	}
	for value, want := range map[string]int64{"off": 0, "0": 0, "30m": 1800, "1h": 3600, "24h": 86400, "1s": 1} {
		got, err := With(Defaults(), "history-retention", value)
		if err != nil || got.HistoryRetentionSeconds != want {
			t.Fatal(value, got, err)
		}
	}
	for _, value := range []string{"-1h", "0.5s", "10000h", "invalid"} {
		if _, err := With(Defaults(), "history-retention", value); err == nil {
			t.Fatal("accepted", value)
		}
	}
}
