package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestForgejoOwnerCodePreservesNativeSearchContext(t *testing.T) {
	definition := `
		{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}
		{{define "org/header"}}org/{{.ContextUser.Name}}{{end}}
		{{define "shared/user/profile_big_avatar"}}avatar/{{.ContextUser.Name}}{{end}}
		{{define "user/overview/header"}}tabs/{{.ContextUser.Name}}{{end}}
		{{define "shared/search/code/search"}}search/{{.CodeSearchPath}}/{{.Keyword}}{{end}}
		{{define "custom/soda/page_intro"}}intro/{{.Eyebrow}}/{{.Title}}{{end}}
		{{define "page"}}` + readForgejoTemplate(t, "user", "code.tmpl") + `{{end}}`
	parsed, err := template.New("owner-code").Funcs(template.FuncMap{
		"dict": forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{"user.code": "Code"}}}
		},
	}).Parse(definition)
	if err != nil {
		t.Fatal(err)
	}
	for _, organization := range []bool{false, true} {
		var output bytes.Buffer
		data := map[string]any{
			"IsSigned": organization, "CodeSearchPath": "/forge/alice/-/code", "Keyword": "<query>",
			"ContextUser": map[string]any{"IsOrganization": organization, "Name": "<owner>"},
		}
		if err := parsed.ExecuteTemplate(&output, "page", data); err != nil {
			t.Fatal(err)
		}
		html := output.String()
		for _, want := range []string{"search//forge/alice/-/code/&lt;query&gt;", "intro/&lt;owner&gt;/Code", "soda-code-search"} {
			if !strings.Contains(html, want) {
				t.Errorf("owner code page lacks %q: %s", want, html)
			}
		}
		if strings.Count(html, "search//forge/alice/-/code/") != 1 || strings.Count(html, "<main ") != 1 || strings.Count(html, "</main>") != 1 {
			t.Errorf("owner code page must render one main root and native search: %s", html)
		}
		if organization {
			if !strings.Contains(html, "org/&lt;owner&gt;") || strings.Contains(html, "avatar/") || !strings.Contains(html, `data-signed="true"`) {
				t.Errorf("organization context crossed profile branch: %s", html)
			}
		} else if !strings.Contains(html, "avatar/&lt;owner&gt;") || !strings.Contains(html, "tabs/&lt;owner&gt;") || strings.Contains(html, "org/") || !strings.Contains(html, `data-signed="false"`) {
			t.Errorf("individual context crossed organization branch: %s", html)
		}
	}
}
