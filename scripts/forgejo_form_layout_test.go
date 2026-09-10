package scripts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Only exact-upstream parity tests reconstruct historical presentation markup.
// Behavior/branch tests use readForgejoTemplate, which returns authored bytes.
func readForgejoTemplateForUpstreamParity(t *testing.T, parts ...string) string {
	t.Helper()
	source := readForgejoTemplate(t, parts...)
	path := filepath.ToSlash(filepath.Join(parts...))
	data, err := os.ReadFile("../tests/forgejo/presentation/form-presentation-deltas.json")
	if err != nil {
		t.Fatal(err)
	}
	var deltas map[string][][2]string
	if err := json.Unmarshal(data, &deltas); err != nil {
		t.Fatal(err)
	}
	for _, delta := range deltas[path] {
		if !strings.Contains(source, delta[0]) {
			t.Fatalf("%s: reviewed form-layout fragment is missing", path)
		}
		source = strings.Replace(source, delta[0], delta[1], 1)
	}
	return regexp.MustCompile(` soda-p-(editor-container|form-host|form|title|heading|section|gap|toolbar|control|compact)\b`).ReplaceAllString(source, "")
}
