package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoStatusWrappersPreserveNativeDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{"404.tmpl", []string{
			`{{if .IsRepo}}{{template "repo/header" .}}{{end}}`,
			`{{if .NotFoundPrompt}}{{.NotFoundPrompt}}{{else}}{{ctx.Locale.Tr "error404"}}{{end}}`,
			`{{if .NotFoundGoBackURL}}`,
			`href="{{.NotFoundGoBackURL}}"`,
			`{{if .ShowFooterVersion}}`,
			`{{AppVerNoMetadata}}`,
		}},
		{"413.tmpl", []string{
			`{{if .IsRepo}}{{template "repo/header" .}}{{end}}`,
			`{{ctx.Locale.Tr "error413"}}`,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := readForgejoTemplate(t, "status", tt.name)
			for _, want := range tt.want {
				if !strings.Contains(source, want) {
					t.Errorf("status wrapper lost native diagnostic contract %q", want)
				}
			}
			for _, want := range []string{`{{template "base/head" .}}`, `{{template "base/footer" .}}`, `class="page-content ui {{if .IsRepo}}repository{{end}} soda-page soda-status"`} {
				if !strings.Contains(source, want) {
					t.Errorf("status wrapper missing composition %q", want)
				}
			}
		})
	}
}

func TestForgejoStatus404RendersDefaultAndEscapedCustomPrompt(t *testing.T) {
	source := readForgejoTemplate(t, "status", "404.tmpl")
	definition := `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "repo/header"}}{{end}}{{define "status"}}` + source + `{{end}}`
	parsed, err := template.New("status").Funcs(template.FuncMap{
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{"error404": "Page unavailable", "go_back": "Go back", "admin.config.app_ver": "Version"}}}
		},
		"AppVerNoMetadata": func() string { return "15.0.7" },
		"AssetUrlPrefix": func() string { return "/assets" },
	}).Parse(definition)
	if err != nil {
		t.Fatalf("parse status 404: %v", err)
	}
	for _, tt := range []struct {
		name string
		data map[string]any
		want string
	}{
		{"default", map[string]any{}, "Page unavailable"},
		{"custom escaped", map[string]any{"NotFoundPrompt": `<script>alert("x")</script>`}, `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "status", tt.data); err != nil {
				t.Fatalf("render status 404: %v", err)
			}
			if !strings.Contains(output.String(), `<h1 class="error-code">404</h1>`) || !strings.Contains(output.String(), tt.want) {
				t.Fatalf("unexpected status output: %s", output.String())
			}
			if strings.Contains(output.String(), "<script>") {
				t.Fatalf("custom not-found prompt rendered active markup: %s", output.String())
			}
		})
	}
}

func TestForgejoStatus500RetainsUpstreamPanicFallback(t *testing.T) {
	path := filepath.Join("..", "appliance", "forgejo", "templates", "status", "500.tmpl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("500 must remain an upstream fallback without a custom override; stat error: %v", err)
	}

	css := readForgejoAssetFile(t, "status.css")
	for _, want := range []string{".soda-status-card", ".soda-status-card .error-code", ".soda-status-version", "grid-template-columns: minmax(0, 1fr)"} {
		if !strings.Contains(css, want) {
			t.Errorf("status CSS missing scoped rule %q", want)
		}
	}
}
