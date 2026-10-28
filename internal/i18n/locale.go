package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Every embedded catalog registers itself. Adding a locale needs no Go/UI edits.
//
//go:embed locales/*.json
var files embed.FS

type catalog struct {
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	Aliases  []string          `json:"aliases"`
	Order    int               `json:"order"`
	Messages map[string]string `json:"messages"`
	CLI      map[string]string `json:"cli"`
}

var catalogs = loadCatalogs()

func loadCatalogs() map[string]catalog {
	result := map[string]catalog{}
	entries, err := files.ReadDir("locales")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := files.ReadFile("locales/" + entry.Name())
		if err != nil {
			panic(err)
		}
		var c catalog
		if err := json.Unmarshal(data, &c); err != nil {
			panic(fmt.Errorf("decode locale %s: %w", entry.Name(), err))
		}
		if c.Code == "" || c.Name == "" || c.Code+".json" != entry.Name() {
			panic("invalid locale metadata: " + entry.Name())
		}
		if _, duplicate := result[c.Code]; duplicate {
			panic("duplicate locale: " + c.Code)
		}
		result[c.Code] = c
	}
	aliases := map[string]string{}
	for _, c := range result {
		for _, alias := range append([]string{c.Code}, c.Aliases...) {
			key := normalize(alias)
			if key == "" || key == "auto" {
				panic("invalid locale alias: " + alias)
			}
			if old, ok := aliases[key]; ok && old != c.Code {
				panic("ambiguous locale alias: " + alias)
			}
			aliases[key] = c.Code
		}
	}
	if _, ok := result["en"]; !ok {
		panic("English fallback catalog is required")
	}
	return result
}

type LanguageOption struct{ Code, Name string }

func Languages() []LanguageOption {
	result := make([]LanguageOption, 0, len(catalogs))
	for _, c := range catalogs {
		result = append(result, LanguageOption{c.Code, c.Name})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := catalogs[result[i].Code], catalogs[result[j].Code]
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Code < b.Code
	})
	return result
}
func LanguageCodes() []string {
	result := []string{"auto"}
	for _, language := range Languages() {
		result = append(result, language.Code)
	}
	return result
}
func SupportedLanguage(code string) bool { _, ok := catalogs[code]; return code == "auto" || ok }
func LanguageName(code string) string {
	if c, ok := catalogs[code]; ok {
		return c.Name
	}
	return code
}
func normalize(value string) string {
	value = strings.ReplaceAll(strings.ToLower(value), "_", "-")
	return strings.Split(strings.Split(value, ".")[0], "@")[0]
}
func detectLanguage(value string) string {
	for _, item := range strings.Split(value, ":") {
		item = normalize(item)
		if item == "c" || item == "posix" {
			return "en"
		}
		best, bestLength := "", -1
		for _, c := range catalogs {
			for _, alias := range append([]string{c.Code}, c.Aliases...) {
				alias = normalize(alias)
				if (item == alias || strings.HasPrefix(item, alias+"-")) && len(alias) > bestLength {
					best, bestLength = c.Code, len(alias)
				}
			}
		}
		if best != "" {
			return best
		}
	}
	return "en"
}
func Language(setting string) string {
	if setting != "auto" && setting != "" {
		if SupportedLanguage(setting) {
			return setting
		}
		return "en"
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return detectLanguage(value)
		}
	}
	return "en"
}

type Locale struct{ Language string }

func New(language string) Locale { return Locale{Language(language)} }
func (l Locale) Text(key string) string {
	if value := catalogs[l.Language].Messages[key]; value != "" {
		return value
	}
	if value := catalogs["en"].Messages[key]; value != "" {
		return value
	}
	return key
}
func (l Locale) CLI(text string) string {
	if value := catalogs[l.Language].CLI[text]; value != "" {
		return value
	}
	return text
}
