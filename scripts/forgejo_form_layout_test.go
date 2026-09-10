package scripts

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The older exact-upstream tests undo their historical presentation changes.
// Undo only the reviewed form-layout delta first, with ordered exact matches.
// form-source.test.ts checks the real (un-normalized) submission controls and
// gates, while the native browser checks exercise the actual redesigned pages.
func withoutForgejoFormLayout(t *testing.T, path, source string) string {
	t.Helper()
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
	return source
}
