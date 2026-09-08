package scripts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"testing"
)

func TestForgejoOrganizationHomeSourceParity(t *testing.T) {
	src := readForgejoTemplate(t, "org", "home.tmpl")
	provenance := "{{/* Soda presentation of Forgejo 15.0.7 templates/org/home.tmpl; GPL-3.0-or-later. Upstream SHA-256: a8b8f935f72758355efa25bdb53d376f77c39a0984f2d1a0b875c9bbb8fea32f. */}}\n"
	if !strings.HasPrefix(src, provenance) {
		t.Fatal("missing source attribution")
	}
	src = strings.TrimPrefix(src, provenance)
	for _, r := range [][2]string{{"class=\"page-content organization profile soda-page soda-org-home\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\"", "class=\"page-content organization profile\""},
		{"class=\"ui {{if .ShowMemberAndTeamTab}}eleven wide{{end}} column soda-org-home-main\"", "class=\"ui {{if .ShowMemberAndTeamTab}}eleven wide{{end}} column\""},
		{"class=\"ui five wide column soda-org-home-sidebar\"", "class=\"ui five wide column\""},
		{"<div class=\"soda-toolbar\">{{template \"shared/repo_search\" .}}</div>", "{{template \"shared/repo_search\" .}}"},
		{"<div class=\"soda-list\">{{template \"explore/repo_list\" .}}</div>", "{{template \"explore/repo_list\" .}}"}} {
		if strings.Count(src, r[0]) != 1 {
			t.Fatalf("unexpected presentation delta %q", r[0])
		}
		src = strings.Replace(src, r[0], r[1], 1)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(src))); got != "a8b8f935f72758355efa25bdb53d376f77c39a0984f2d1a0b875c9bbb8fea32f" {
		t.Fatalf("native organization content changed: %s", got)
	}
}
func TestForgejoOrganizationHomeNativeCreationAndVisibility(t *testing.T) {
	seams := `{{define "base/head"}}head{{end}}{{define "base/footer"}}footer{{end}}{{define "org/header"}}org-header{{end}}{{define "base/alert"}}alert{{end}}{{define "shared/repo_search"}}native-search{{end}}{{define "explore/repo_list"}}native-repositories{{end}}{{define "base/paginate"}}native-pagination{{end}}`
	functions := template.FuncMap{"ctx": func() forgejoTemplateContext {
		return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{}}}
	}, "AppSubUrl": func() string { return "/forge" }, "svg": func(_ string, _ ...any) string { return "icon" }, "PathEscape": func(s string) string { return s }}
	parsed, err := template.New("page").Funcs(functions).Parse(seams + readForgejoTemplate(t, "org", "home.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, allow := range []bool{false, true} {
		data := map[string]any{"IsSigned": allow, "ShowMemberAndTeamTab": allow, "CanCreateOrgRepo": allow, "Org": map[string]any{"ID": 42}, "ProfileReadme": "<script>untrusted</script>", "IsProfileReadmePlain": true}
		var output bytes.Buffer
		if err := parsed.ExecuteTemplate(&output, "page", data); err != nil {
			t.Fatal(err)
		}
		for _, part := range []string{"native-search", "native-repositories", "native-pagination", "&lt;script&gt;untrusted&lt;/script&gt;"} {
			if !strings.Contains(output.String(), part) {
				t.Errorf("missing native output %q", part)
			}
		}
		if got := strings.Contains(output.String(), "/forge/repo/create?org=42"); got != allow {
			t.Errorf("native create permission =%v want%v", got, allow)
		}
		if got := strings.Contains(output.String(), "/forge/repo/migrate?org=42"); got != allow {
			t.Errorf("native migration permission =%v want%v", got, allow)
		}
	}
}
