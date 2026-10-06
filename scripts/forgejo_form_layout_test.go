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
	source = expandAuthLeafInvocations(t, source)
	return stripSodaPClasses(reverseFormPresentationDeltas(t, path, source))
}

func reverseFormPresentationDeltas(t *testing.T, path, source string) string {
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

func stripSodaPClasses(source string) string {
	return regexp.MustCompile(` soda-p-(editor-container|form-host|form|title|heading|section|gap|toolbar|control|compact)\b`).ReplaceAllString(source, "")
}

// expandAuthLeafInvocations splices the ACTUAL tracked provider-leaf bodies
// into a split entry at their invocation seams, restoring the lexical context
// the split moved out (the $cfg binding and the ctxData dot). Sources without
// auth-leaf invocations pass through untouched.
func expandAuthLeafInvocations(t *testing.T, source string) string {
	t.Helper()
	return expandAuthLeafInvocationsWithReader(t, source, func(name string) string {
		parts := strings.Split(name, "/")
		parts[len(parts)-1] += ".tmpl"
		return readForgejoTemplate(t, parts...)
	})
}

func expandAuthLeafInvocationsWithReader(t *testing.T, source string, readLeaf func(name string) string) string {
	t.Helper()
	if !strings.Contains(source, "admin/auth/edit_") {
		return source
	}
	var out []string
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		name, ok := strings.CutPrefix(trimmed, `{{template "`)
		if !ok || !strings.HasPrefix(name, "admin/auth/edit_") || !strings.HasSuffix(trimmed, `(dict "ctxData" . "cfg" .Source.Cfg)}}`) {
			out = append(out, line)
			continue
		}
		name = name[:strings.Index(name, `"`)]
		out = append(out, reconstructAuthLeaf(t, line[:len(line)-len(trimmed)], name, readLeaf)...)
	}
	return strings.Join(out, "\n")
}

func reconstructAuthLeaf(t *testing.T, indent, name string, readLeaf func(name string) string) []string {
	t.Helper()
	lines := strings.Split(readLeaf(name), "\n")
	if len(lines) < 5 || !strings.HasPrefix(lines[0], "{{/*") || lines[1] != "{{$cfg:=.cfg}}" || lines[2] != "{{with .ctxData}}" || lines[len(lines)-2] != "{{end}}" || lines[len(lines)-1] != "" {
		t.Fatalf("auth leaf %s lost its parity reconstruction structure", name)
	}
	out := []string{indent + "{{$cfg:=.Source.Cfg}}"}
	return append(out, lines[3:len(lines)-2]...)
}
