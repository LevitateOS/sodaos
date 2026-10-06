package scripts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type forgejoTemplateLocale struct {
	translations map[string]string
}

func (l forgejoTemplateLocale) Tr(key string, _ ...any) string {
	return l.translations[key]
}

func (l forgejoTemplateLocale) TrN(_ any, singular, plural string, _ ...any) string {
	if translated := l.translations[plural]; translated != "" {
		return translated
	}
	return singular
}

type forgejoTemplateContext struct {
	Locale forgejoTemplateLocale
}

func forgejoTemplateDict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires key/value pairs")
	}
	result := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict key %d is not a string", i/2)
		}
		result[key] = values[i+1]
	}
	return result, nil
}

func readForgejoTemplate(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "frontend", "forgejo", "templates"}, parts...)...)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

var forgejoTemplateCallPattern = regexp.MustCompile(`\{\{\s*template\s+"([^"]+)"`)

func templateCalls(contents string) map[string]bool {
	calls := make(map[string]bool)
	for _, match := range forgejoTemplateCallPattern.FindAllStringSubmatch(contents, -1) {
		calls[match[1]] = true
	}
	return calls
}

func requireForgejoTemplateCalls(t *testing.T, name string, required ...string) {
	t.Helper()
	contents := readForgejoTemplate(t, strings.Split(name, "/")...)
	calls := templateCalls(contents)
	for _, requiredName := range required {
		if !calls[requiredName] {
			t.Errorf("%s no longer composes native template %q", name, requiredName)
		}
	}
}
