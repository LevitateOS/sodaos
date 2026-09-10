package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

type forgejoCargoLocale struct{}

func (forgejoCargoLocale) Lang() string { return "en-US" }

type forgejoCargoStrings struct{}

func (forgejoCargoStrings) Split(s, sep string) []string { return strings.Split(s, sep) }

func (forgejoCargoLocale) Tr(key string, args ...any) string {
	if strings.HasSuffix(key, ".description") {
		return "A special index Git repository is needed to use the Cargo registry. Using this option will (re-)create the repository and configure it automatically."
	}
	if len(args) == 0 {
		return key
	}
	return key + "/" + args[0].(string) + "/" + args[1].(string)
}

type forgejoCargoContext struct{ Locale forgejoCargoLocale }

func TestForgejoCargoComposesNativeIndexBranches(t *testing.T) {
	cargo := readForgejoTemplate(t, "package", "shared", "cargo.tmpl")
	parsed, err := template.New("cargo").Funcs(template.FuncMap{
		"StringUtils": func() forgejoCargoStrings { return forgejoCargoStrings{} },
		"ctx":         func() forgejoCargoContext { return forgejoCargoContext{} },
	}).Parse(`{{define "cargo"}}` + cargo + `{{end}}`)
	if err != nil {
		t.Fatalf("parse Cargo settings override: %v", err)
	}

	for _, test := range []struct {
		name       string
		exists     bool
		wantPath   string
		wantAction string
		absent     string
	}{
		{name: "existing index", exists: true, wantPath: "/cargo/rebuild", wantAction: "packages.owner.settings.cargo.rebuild", absent: "/cargo/initialize"},
		{name: "missing index", exists: false, wantPath: "/cargo/initialize", wantAction: "packages.owner.settings.cargo.initialize", absent: "/cargo/rebuild"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "cargo", map[string]any{
				"CargoIndexExists": test.exists,
				"Link":             "/forge/alice&tools",
			}); err != nil {
				t.Fatalf("render Cargo settings: %v", err)
			}
			output := rendered.String()
			for _, want := range []string{
				`class="ui form soda-form"`,
				`<fieldset class="soda-form-section soda-p-section">`,
				`method="post"`,
				`class="ui info message soda-notice"`, `class="ui warning message soda-notice"`,
				`action="/forge/alice&amp;tools` + test.wantPath + `"`,
				test.wantAction,
				`packages.registry.documentation/Cargo/https://forgejo.org/docs/latest/user/packages/cargo/`,
			} {
				if !strings.Contains(output, want) {
					t.Errorf("Cargo %s branch lost %q:\n%s", test.name, want, output)
				}
			}
			if strings.Contains(output, test.absent) {
				t.Errorf("Cargo %s branch rendered inactive action %q:\n%s", test.name, test.absent, output)
			}
		})
	}
}
