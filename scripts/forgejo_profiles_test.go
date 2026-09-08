package scripts

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"
)

type forgejoProfilesLocale struct{}

func (forgejoProfilesLocale) Tr(key string, _ ...any) string { return key }

type forgejoProfilesContext struct{ Locale forgejoProfilesLocale }

func TestForgejoProfilesKeepNativePageWorkflows(t *testing.T) {
	profile := readForgejoTemplate(t, "user", "profile.tmpl")
	for _, marker := range []string{
		`class="page-content user profile soda-page soda-profile"`,
		`data-signed="{{.IsSigned}}"`,
		`template "shared/user/profile_big_avatar" .`,
		`template "user/overview/header" .`,
		`template "user/heatmap" .`,
		`template "user/dashboard/feeds" .`,
		`template "shared/repo_search" .`,
		`template "explore/repo_list" .`,
		`template "repo/user_cards" .`,
		`{{.ProfileReadme}}`,
		`id="block-user"`,
		`template "base/modal_actions_confirm" .`,
	} {
		if !strings.Contains(profile, marker) {
			t.Errorf("profile override lost native boundary %q", marker)
		}
	}
	if strings.Count(profile, `{{.ProfileReadme}}`) != 2 {
		t.Error("profile override must preserve both native plain and rendered readme branches")
	}
	if strings.Index(profile, `</main>`) > strings.Index(profile, `id="block-user"`) {
		t.Error("native block-user modal must remain outside the branded page root")
	}
}

func TestForgejoProfilesComposeNativeBranchesWithOriginalContext(t *testing.T) {
	profile := readForgejoTemplate(t, "user", "profile.tmpl")
	seams := `
		{{define "base/head"}}head{{end}}{{define "base/footer"}}footer{{end}}{{define "base/alert"}}alert{{end}}
		{{define "shared/user/profile_big_avatar"}}avatar/{{.ContextUser.HomeLink}}{{end}}
		{{define "user/overview/header"}}header/{{.ContextUser.HomeLink}}{{end}}
		{{define "user/heatmap"}}heatmap{{end}}{{define "user/dashboard/feeds"}}feeds{{end}}
		{{define "shared/repo_search"}}search/{{.ContextUser.HomeLink}}{{end}}
		{{define "explore/repo_list"}}repos/{{.ContextUser.HomeLink}}{{end}}
		{{define "base/paginate"}}paginate{{end}}{{define "repo/user_cards"}}cards{{end}}
		{{define "base/modal_actions_confirm"}}modal-actions{{end}}`
	parsed, err := template.New("profile").Funcs(template.FuncMap{
		"AppSubUrl": func() string { return "/forge" },
		"ctx":       func() forgejoProfilesContext { return forgejoProfilesContext{} },
	}).Parse(seams + `{{define "page"}}` + profile + `{{end}}`)
	if err != nil {
		t.Fatalf("parse profile override: %v", err)
	}
	contextUser := map[string]any{"HomeLink": "/forge/alice", "ID": int64(7), "KeepActivityPrivate": false, "Visibility": 0}
	base := map[string]any{"Title": "Alice", "IsSigned": true, "SignedUserID": int64(7), "ContextUser": contextUser}

	base["TabName"] = "repositories"
	var repositories bytes.Buffer
	if err := parsed.ExecuteTemplate(&repositories, "page", base); err != nil {
		t.Fatalf("render repository profile branch: %v", err)
	}
	for _, want := range []string{`data-signed="true"`, "avatar//forge/alice", "header//forge/alice", "search//forge/alice", "repos//forge/alice", "modal-actions"} {
		if !strings.Contains(repositories.String(), want) {
			t.Errorf("repository profile branch lost %q:\n%s", want, repositories.String())
		}
	}

	base["TabName"] = "overview"
	base["IsProfileReadmePlain"] = true
	base["ProfileReadme"] = `<script>alert("profile")</script>`
	var overview bytes.Buffer
	if err := parsed.ExecuteTemplate(&overview, "page", base); err != nil {
		t.Fatalf("render overview profile branch: %v", err)
	}
	if !strings.Contains(overview.String(), `&lt;script&gt;alert(&#34;profile&#34;)&lt;/script&gt;`) || strings.Contains(overview.String(), `<script>`) {
		t.Errorf("plain profile readme was not escaped:\n%s", overview.String())
	}
	if strings.Contains(overview.String(), "repos//forge/alice") {
		t.Errorf("overview profile branch rendered repository list:\n%s", overview.String())
	}
}

func TestForgejoProfilesHeaderComposesNativeRoutes(t *testing.T) {
	header := readForgejoTemplate(t, "user", "overview", "header.tmpl")
	parsed, err := template.New("header").Funcs(template.FuncMap{
		"ctx": func() forgejoProfilesContext { return forgejoProfilesContext{} },
		"svg": func(name string, _ ...any) template.HTML {
			return template.HTML("<svg>" + template.HTMLEscapeString(name) + "</svg>")
		},
	}).Parse(`{{define "header"}}` + header + `{{end}}`)
	if err != nil {
		t.Fatalf("parse profile header: %v", err)
	}
	data := map[string]any{
		"HasProfileReadme": true, "TabName": "repositories", "SignedUserID": int64(2),
		"IsPackageEnabled": true, "IsRepoIndexerEnabled": true,
		"ContextUser": map[string]any{"ID": int64(2), "HomeLink": "/forge/alice", "IsIndividual": true, "KeepActivityPrivate": false, "NumStars": 3},
	}
	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "header", data); err != nil {
		t.Fatalf("render profile header: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{`<div class="soda-tabs">`, `class="ui secondary pointing tabular borderless menu secondary-nav"`, `href="/forge/alice?tab=repositories"`} {
		if !strings.Contains(output, want) {
			t.Errorf("profile header lost %q:\n%s", want, output)
		}
	}
	if strings.Count(output, `href="/forge/alice/-/packages"`) != 1 {
		t.Errorf("profile header must render one native package route from ContextUser.HomeLink:\n%s", output)
	}
}

func TestForgejoProfilesStylesStayInsideProfileRoot(t *testing.T) {
	contents, err := os.ReadFile("../assets/branding/forgejo/profiles.css")
	if err != nil {
		t.Fatalf("read profile styles: %v", err)
	}
	css := string(contents)
	for _, selector := range []string{"#profile-avatar-card", "#visibility-hint", "#activity-feed", ".user-cards", ".soda-profile-readme"} {
		for _, line := range strings.Split(css, "\n") {
			if strings.Contains(line, selector) && !strings.Contains(line, ".soda-profile") {
				t.Errorf("profile selector %q escapes .soda-profile scope: %s", selector, line)
			}
		}
	}
}
