package scripts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	forgejoActivity1507SHA = "60cce79800498066b21ec48a2dfc317ee593761d24945c20c481a8572806c9e3"
	forgejoGraph1507SHA    = "fb938c350bbb9718ab6d0b289ff9156a4d184644581ec83a1ef01c8ebc6ddd0d"
)

func TestForgejoInsightsOverridesMatchStock1507(t *testing.T) {
	tests := []struct {
		name         string
		sha          string
		provenance   string
		stockSuffix  string
		replacements [][2]string
	}{
		{
			name:        "activity.tmpl",
			sha:         forgejoActivity1507SHA,
			provenance:  `{{/* Adapted from Forgejo 15.0.7 templates/repo/activity.tmpl (GPL-3.0-or-later); upstream SHA-256 ` + forgejoActivity1507SHA + `. */}}` + "\n",
			stockSuffix: "\n",
			replacements: [][2]string{
				{` class="page-content repository commits soda-page soda-insights soda-insights-activity" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`, ` class="page-content repository commits"`},
				{` class="ui container flex-container soda-page-container"`, ` class="ui container flex-container"`},
			},
		},
		{
			name:       "graph.tmpl",
			sha:        forgejoGraph1507SHA,
			provenance: `{{/* Adapted from Forgejo 15.0.7 templates/repo/graph.tmpl (GPL-3.0-or-later); upstream SHA-256 ` + forgejoGraph1507SHA + `. */}}` + "\n",
			replacements: [][2]string{
				{` class="page-content repository commits soda-page soda-insights soda-insights-graph" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`, ` class="page-content repository commits"`},
				{` class="ui container soda-page-container"`, ` class="ui container"`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := readForgejoTemplateForUpstreamParity(t, "repo", tt.name)
			if !strings.HasPrefix(contents, tt.provenance) {
				t.Fatalf("%s lost exact Forgejo version, license, or pristine-source attribution", tt.name)
			}
			restored := strings.TrimPrefix(contents, tt.provenance)
			for _, replacement := range tt.replacements {
				if count := strings.Count(restored, replacement[0]); count != 1 {
					t.Fatalf("%s has %d occurrences of Soda wrapper change %q, want 1", tt.name, count, replacement[0])
				}
				restored = strings.Replace(restored, replacement[0], replacement[1], 1)
			}
			restored += tt.stockSuffix
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(restored))); got != tt.sha {
				t.Errorf("%s differs from pristine Forgejo 15.0.7 outside the attributed Soda wrapper changes: got SHA-256 %s, want %s", tt.name, got, tt.sha)
			}
		})
	}
}

type forgejoInsightsPermission struct {
	code bool
}

func (p forgejoInsightsPermission) CanRead(unit any) bool {
	return p.code && unit == "code"
}

func TestForgejoActivityDelegatesNativeInsightBranches(t *testing.T) {
	definition := `
		{{define "base/head"}}native-head{{end}}
		{{define "base/footer"}}native-footer{{end}}
		{{define "repo/header"}}native-repo-header{{end}}
		{{define "repo/navbar"}}native-repo-navbar{{end}}
		{{define "repo/pulse"}}native-pulse{{end}}
		{{define "repo/contributors"}}native-contributors{{end}}
		{{define "repo/code_frequency"}}native-code-frequency{{end}}
		{{define "repo/recent_commits"}}native-recent-commits{{end}}
		{{define "page"}}` + readForgejoTemplate(t, "repo", "activity.tmpl") + `{{end}}`
	parsed, err := template.New("activity").Parse(definition)
	if err != nil {
		t.Fatalf("parse activity override: %v", err)
	}

	tests := []struct {
		name      string
		data      map[string]any
		want      []string
		forbidden []string
	}{
		{
			name:      "pulse with code access",
			data:      map[string]any{"Title": "Activity", "IsSigned": true, "Permission": forgejoInsightsPermission{code: true}, "UnitTypeCode": "code", "PageIsPulse": true},
			want:      []string{`data-signed="true"`, "native-repo-header", "native-repo-navbar", "native-pulse"},
			forbidden: []string{"native-contributors", "native-code-frequency", "native-recent-commits"},
		},
		{
			name:      "contributors with code access",
			data:      map[string]any{"Title": "Contributors", "IsSigned": false, "Permission": forgejoInsightsPermission{code: true}, "UnitTypeCode": "code", "PageIsContributors": true},
			want:      []string{`data-signed="false"`, "native-repo-navbar", "native-contributors"},
			forbidden: []string{"native-pulse", "native-code-frequency", "native-recent-commits"},
		},
		{
			name:      "code frequency keeps native access boundary",
			data:      map[string]any{"Title": "Code frequency", "Permission": forgejoInsightsPermission{}, "UnitTypeCode": "code", "PageIsCodeFrequency": true},
			want:      []string{"native-repo-header", "native-code-frequency"},
			forbidden: []string{"native-repo-navbar", "native-pulse", "native-contributors", "native-recent-commits"},
		},
		{
			name:      "recent commits in empty repository",
			data:      map[string]any{"Title": "Recent commits", "IsEmptyRepo": true, "Permission": forgejoInsightsPermission{code: true}, "UnitTypeCode": "code", "PageIsRecentCommits": true},
			want:      []string{"native-repo-header", "native-recent-commits"},
			forbidden: []string{"native-repo-navbar", "native-pulse", "native-contributors", "native-code-frequency"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "page", tt.data); err != nil {
				t.Fatalf("execute activity override: %v", err)
			}
			output := rendered.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("activity branch does not contain %q:\n%s", want, output)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(output, forbidden) {
					t.Errorf("activity branch unexpectedly contains %q:\n%s", forbidden, output)
				}
			}
		})
	}
}

type forgejoGraphRef struct {
	RefGroup  string
	Name      string
	ShortName string
}

func TestForgejoGraphPreservesNativeModesRefsAndContent(t *testing.T) {
	functions := template.FuncMap{
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"repo.commit_graph":              "Commit graph",
				"repo.commit_graph.select":       "Select refs",
				"repo.commit_graph.hide_pr_refs": "Hide pull refs",
				"repo.commit_graph.monochrome":   "Monochrome",
				"repo.commit_graph.color":        "Color",
			}}}
		},
		"svg": func(name string, _ ...any) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	definition := `
		{{define "base/head"}}native-head{{end}}
		{{define "base/footer"}}native-footer{{end}}
		{{define "base/paginate"}}native-pagination{{end}}
		{{define "repo/header"}}native-repo-header{{end}}
		{{define "repo/graph/svgcontainer"}}<svg id="native-graph"></svg>{{end}}
		{{define "repo/graph/commits"}}<div id="native-commits"></div>{{end}}
		{{define "page"}}` + readForgejoTemplate(t, "repo", "graph.tmpl") + `{{end}}`
	parsed, err := template.New("graph").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse graph override: %v", err)
	}

	for _, tt := range []struct {
		mode   string
		signed bool
	}{{mode: "color", signed: true}, {mode: "monochrome", signed: false}} {
		t.Run(tt.mode, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "page", map[string]any{
				"Title": "Commit graph", "IsSigned": tt.signed, "Mode": tt.mode,
				"AllRefs": []forgejoGraphRef{{RefGroup: "heads", Name: "refs/heads/main", ShortName: "main"}, {RefGroup: "tags", Name: "refs/tags/v1", ShortName: "v1"}},
			}); err != nil {
				t.Fatalf("execute graph override: %v", err)
			}
			output := rendered.String()
			for _, want := range []string{
				`id="flow-select-refs-dropdown"`, `data-value="refs/heads/main"`, `data-value="refs/tags/v1"`,
				`id="flow-color-monochrome"`, `id="flow-color-colored"`, `id="loading-indicator"`,
				`id="native-graph"`, `id="native-commits"`, "native-pagination",
			} {
				if !strings.Contains(output, want) {
					t.Errorf("graph mode %s lost native marker %q:\n%s", tt.mode, want, output)
				}
			}
			wantMonochrome := tt.mode == "monochrome"
			if got := strings.Contains(output, `id="git-graph-container" class="ui segment monochrome"`); got != wantMonochrome {
				t.Errorf("graph mode %s monochrome container = %t, want %t", tt.mode, got, wantMonochrome)
			}
			wantSigned := fmt.Sprintf(`data-signed="%t"`, tt.signed)
			if !strings.Contains(output, wantSigned) {
				t.Errorf("graph mode %s lost explicit signed state %q:\n%s", tt.mode, wantSigned, output)
			}
		})
	}
}

func TestForgejoInsightsCSSIsPageScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "insights.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, marker := range []string{".soda-insights", "#repo-contributors-chart", "#repo-code-frequency-chart", "#repo-recent-commits-chart", "#git-graph-container", "var(--soda-page-surface)"} {
		if !strings.Contains(css, marker) {
			t.Errorf("insights stylesheet lost %q", marker)
		}
	}
	for _, forbidden := range []string{"body {", ":root {"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("insights stylesheet contains unscoped selector %q", forbidden)
		}
	}
}
