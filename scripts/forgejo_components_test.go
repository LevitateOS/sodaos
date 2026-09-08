package scripts

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type forgejoTemplateLocale struct {
	translations map[string]string
}

func (l forgejoTemplateLocale) Tr(key string) string {
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

func TestForgejoPageIntroComposition(t *testing.T) {
	partial := readForgejoTemplate(t, "custom", "soda", "page_intro.tmpl")
	translations := map[string]string{
		"new_repo.title": "Créer & partager <ensemble>",
	}
	functions := template.FuncMap{
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: translations}}
		},
		"dict": forgejoTemplateDict,
	}
	definition := `{{define "custom/soda/page_intro"}}` + partial + `{{end}}
		{{define "caller"}}{{template "custom/soda/page_intro" dict
			"TitleID" .TitleID
			"Eyebrow" .Eyebrow
			"Title" (ctx.Locale.Tr "new_repo.title")
			"Description" .Description
			"Artwork" .Artwork
			"Class" .Class}}{{end}}`
	parsed, err := template.New("components").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse shared page intro: %v", err)
	}

	data := map[string]string{
		"TitleID":     `intro" onclick="alert(1)`,
		"Eyebrow":     `<script>eyebrow</script>`,
		"Description": `Make & share <strong>things</strong>.`,
		"Artwork":     `diagram" onerror="alert(1).png`,
		"Class":       `compact" data-owned="true`,
	}
	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "caller", data); err != nil {
		t.Fatalf("execute shared page intro: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`Créer &amp; partager &lt;ensemble&gt;`,
		`&lt;script&gt;eyebrow&lt;/script&gt;`,
		`Make &amp; share &lt;strong&gt;things&lt;/strong&gt;.`,
		`src="/forge/assets/soda/forgejo/diagram%22%20onerror=%22alert%281%29.png"`,
		`soda-page-intro compact&#34; data-owned=&#34;true`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered intro does not contain escaped %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{`<script>`, `onclick="alert(1)"`, `onerror="alert(1)`} {
		if strings.Contains(output, forbidden) {
			t.Errorf("rendered intro contains active caller markup %q:\n%s", forbidden, output)
		}
	}

	var minimal bytes.Buffer
	if err := parsed.ExecuteTemplate(&minimal, "custom/soda/page_intro", map[string]any{
		"TitleID": "plain-title",
		"Title":   "Plain title",
	}); err != nil {
		t.Fatalf("execute minimal shared page intro: %v", err)
	}
	minimalOutput := minimal.String()
	for _, absent := range []string{"soda-page-eyebrow", "soda-page-description", "soda-page-art", "<img"} {
		if strings.Contains(minimalOutput, absent) {
			t.Errorf("minimal intro unexpectedly contains optional %q:\n%s", absent, minimalOutput)
		}
	}
}

func TestForgejoEmptyContentComposition(t *testing.T) {
	partial := readForgejoTemplate(t, "custom", "soda", "empty_content.tmpl")
	functions := template.FuncMap{
		"dict": forgejoTemplateDict,
		"svg": func(name string, _ int) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	definition := `{{define "custom/soda/empty_content"}}` + partial + `{{end}}
		{{define "caller"}}{{template "custom/soda/empty_content" dict
			"Icon" .Icon
			"Eyebrow" .Eyebrow
			"TitleID" .TitleID
			"Title" .Title
			"Description" .Description}}{{end}}`
	parsed, err := template.New("components").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse shared empty content: %v", err)
	}

	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "caller", map[string]string{
		"Icon":        "octicon-inbox",
		"Eyebrow":     "Nothing & nowhere",
		"TitleID":     `empty" data-owned="true`,
		"Title":       `<script>empty</script>`,
		"Description": `Try <strong>again</strong>.`,
	}); err != nil {
		t.Fatalf("execute shared empty content: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`data-icon="octicon-inbox"`,
		`Nothing &amp; nowhere`,
		`id="empty&#34; data-owned=&#34;true"`,
		`&lt;script&gt;empty&lt;/script&gt;`,
		`Try &lt;strong&gt;again&lt;/strong&gt;.`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered empty content does not contain escaped %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "<script>") {
		t.Errorf("rendered empty content contains active caller markup:\n%s", output)
	}

	var minimal bytes.Buffer
	if err := parsed.ExecuteTemplate(&minimal, "custom/soda/empty_content", map[string]any{
		"Title": "No results",
	}); err != nil {
		t.Fatalf("execute minimal shared empty content: %v", err)
	}
	minimalOutput := minimal.String()
	for _, absent := range []string{"soda-empty-symbol", "soda-empty-eyebrow", "soda-empty-description", " id="} {
		if strings.Contains(minimalOutput, absent) {
			t.Errorf("minimal empty content unexpectedly contains optional %q:\n%s", absent, minimalOutput)
		}
	}
}

func TestForgejoExploreNavbarDelegatesNativePolicyAndOverflow(t *testing.T) {
	partial := readForgejoTemplate(t, "custom", "explore_navbar.tmpl")
	definition := `{{define "explore/navbar"}}<overflow-menu data-native-context="{{.Context}}"><a class="item">native tabs</a></overflow-menu>{{end}}
		{{define "custom/explore_navbar"}}` + partial + `{{end}}`
	parsed, err := template.New("explore-navbar").Parse(definition)
	if err != nil {
		t.Fatalf("parse explore navbar adapter: %v", err)
	}

	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "custom/explore_navbar", map[string]string{"Context": "organization"}); err != nil {
		t.Fatalf("execute explore navbar adapter: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`class="soda-tabs"`,
		`<overflow-menu data-native-context="organization">`,
		`native tabs`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("explore navbar adapter does not contain %q:\n%s", want, output)
		}
	}
	if strings.Count(output, "<overflow-menu") != 1 {
		t.Errorf("explore navbar adapter rendered the native navigation %d times:\n%s", strings.Count(output, "<overflow-menu"), output)
	}
}

func TestForgejoExplorePagesComposeNativeControlsWithOriginalContext(t *testing.T) {
	functions := template.FuncMap{
		"dict": forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{}}}
		},
	}
	nativeSeams := `
		{{define "base/head"}}head{{end}}
		{{define "base/footer"}}footer{{end}}
		{{define "base/paginate"}}paginate/{{.View}}{{end}}
		{{define "explore/navbar"}}<overflow-menu>navbar/{{.View}}</overflow-menu>{{end}}
		{{define "shared/repo_search"}}repo-search/{{.View}}{{end}}
		{{define "explore/repo_list"}}repo-list/{{.View}}{{end}}
		{{define "explore/search"}}people-search/{{.View}}{{end}}
		{{define "explore/user_list"}}people-list/{{.View}}{{end}}
		{{define "custom/explore_empty"}}empty/{{.View}}{{end}}
		{{define "custom/soda/page_intro"}}intro/{{.Title}}{{end}}`
	navbar := `{{define "custom/explore_navbar"}}` + readForgejoTemplate(t, "custom", "explore_navbar.tmpl") + `{{end}}`
	tests := []struct {
		name     string
		file     []string
		data     map[string]any
		expected []string
	}{
		{
			name:     "repositories",
			file:     []string{"explore", "repos.tmpl"},
			data:     map[string]any{"View": "repositories", "Title": "Repositories", "Repos": []int{1}},
			expected: []string{"navbar/repositories", "repo-search/repositories", "repo-list/repositories", "paginate/repositories"},
		},
		{
			name:     "people",
			file:     []string{"explore", "users.tmpl"},
			data:     map[string]any{"View": "people", "Title": "People", "Users": []int{1}},
			expected: []string{"navbar/people", "people-search/people", "people-list/people", "paginate/people"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := nativeSeams + navbar + `{{define "page"}}` + readForgejoTemplate(t, tt.file...) + `{{end}}`
			parsed, err := template.New(tt.name).Funcs(functions).Parse(definition)
			if err != nil {
				t.Fatalf("parse explore page: %v", err)
			}
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "page", tt.data); err != nil {
				t.Fatalf("execute explore page: %v", err)
			}
			output := rendered.String()
			for _, want := range tt.expected {
				if !strings.Contains(output, want) {
					t.Errorf("explore page does not contain native seam %q:\n%s", want, output)
				}
			}
			if strings.Count(output, "navbar/"+tt.name) != 1 {
				t.Errorf("explore page did not render exactly one native navbar with its page context:\n%s", output)
			}
		})
	}
}

func TestForgejoExploreEmptyOwnsOnlyVisibleActions(t *testing.T) {
	functions := template.FuncMap{
		"AppSubUrl": func() string { return "/forge" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"new_repo.link": "New repository",
				"new_org.link":  "New organization",
			}}}
		},
		"dict": forgejoTemplateDict,
		"svg": func(name string, _ ...any) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	definition := `{{define "custom/soda/empty_content"}}` + readForgejoTemplate(t, "custom", "soda", "empty_content.tmpl") + `{{end}}
		{{define "custom/explore_empty"}}` + readForgejoTemplate(t, "custom", "explore_empty.tmpl") + `{{end}}`
	parsed, err := template.New("explore-empty").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse explore empty state: %v", err)
	}

	tests := []struct {
		name      string
		data      map[string]any
		want      []string
		forbidden []string
	}{
		{
			name:      "guest organization search",
			data:      map[string]any{"PageIsExploreOrganizations": true, "Keyword": "hidden <team>"},
			want:      []string{`No matches this time.`, `hidden &lt;team&gt;`, `href="/forge/explore/organizations"`},
			forbidden: []string{"New organization", "New repository"},
		},
		{
			name: "signed organization creator",
			data: map[string]any{
				"PageIsExploreOrganizations": true,
				"IsSigned":                   true,
				"SignedUser":                 map[string]any{"CanCreateOrganization": true},
			},
			want:      []string{`href="/forge/org/create"`, "New organization"},
			forbidden: []string{"New repository"},
		},
		{
			name: "signed organization non-creator",
			data: map[string]any{
				"PageIsExploreOrganizations": true,
				"IsSigned":                   true,
				"SignedUser":                 map[string]any{"CanCreateOrganization": false},
			},
			want:      []string{`href="/forge/explore/repos"`},
			forbidden: []string{"New organization", "New repository"},
		},
		{
			name:      "signed repository explorer",
			data:      map[string]any{"PageIsExploreRepositories": true, "IsSigned": true},
			want:      []string{`href="/forge/explore/repos?only_show_relevant=false"`, `href="/forge/repo/create"`, "New repository"},
			forbidden: []string{"New organization"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "custom/explore_empty", tt.data); err != nil {
				t.Fatalf("execute explore empty state: %v", err)
			}
			output := rendered.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("empty state does not contain %q:\n%s", want, output)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(output, forbidden) {
					t.Errorf("empty state unexpectedly contains %q:\n%s", forbidden, output)
				}
			}
		})
	}
}

func TestForgejoThemeToggleIsSingletonAtEachPlacement(t *testing.T) {
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"home":                   "Home",
				"sign_in":                "Sign in",
				"settings.manage_themes": "Manage themes",
			}}}
		},
		"dict": forgejoTemplateDict,
		"svg": func(name string, _ ...any) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	theme := readForgejoTemplate(t, "custom", "soda", "theme_toggle.tmpl")
	tests := []struct {
		name       string
		templateID string
		definition string
		data       map[string]any
		wantClass  string
	}{
		{
			name:       "home",
			templateID: "home",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "home"}}` + readForgejoTemplate(t, "home.tmpl") + `{{end}}`,
			data:       map[string]any{"ShowRegistrationButton": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "sign in",
			templateID: "sign-in",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "user/auth/signin_inner"}}native sign in{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "sign-in"}}` + readForgejoTemplate(t, "user", "auth", "signin.tmpl") + `{{end}}`,
			data:       map[string]any{},
			wantClass:  "soda-theme-toggle",
		},
		{
			name:       "prohibited sign in",
			templateID: "prohibit-login",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "prohibit-login"}}` + readForgejoTemplate(t, "user", "auth", "prohibit_login.tmpl") + `{{end}}`,
			data:       map[string]any{"PageIsSignIn": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "guest explorer hook",
			templateID: "custom/extra_links",
			definition: `{{define "custom/soda/guest_theme"}}` + readForgejoTemplate(t, "custom", "soda", "guest_theme.tmpl") + `{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "custom/extra_links"}}` + readForgejoTemplate(t, "custom", "extra_links.tmpl") + `{{end}}`,
			data:       map[string]any{"PageIsExploreRepositories": true},
			wantClass:  "soda-guest-theme-toggle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := template.New(tt.name).Funcs(functions).Parse(tt.definition)
			if err != nil {
				t.Fatalf("parse theme placement: %v", err)
			}
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, tt.templateID, tt.data); err != nil {
				t.Fatalf("execute theme placement: %v", err)
			}
			output := rendered.String()
			if count := strings.Count(output, `id="soda-theme-toggle"`); count != 1 {
				t.Errorf("theme placement rendered %d singleton hooks:\n%s", count, output)
			}
			if !strings.Contains(output, tt.wantClass) {
				t.Errorf("theme placement does not contain class %q:\n%s", tt.wantClass, output)
			}
		})
	}
}

func TestForgejoHeaderLoadsGuestThemeScriptOnlyForToggleRoutes(t *testing.T) {
	functions := template.FuncMap{
		"dict":           forgejoTemplateDict,
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
	}
	definition := `{{define "custom/soda/theme_toggle"}}{{end}}{{define "custom/soda/guest_theme"}}` + readForgejoTemplate(t, "custom", "soda", "guest_theme.tmpl") + `{{end}}{{define "custom/header"}}` + readForgejoTemplate(t, "custom", "header.tmpl") + `{{end}}`
	parsed, err := template.New("custom-header").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse custom header: %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantScript bool
	}{
		{name: "guest home", data: map[string]any{"PageIsHome": true}, wantScript: true},
		{name: "guest sign in", data: map[string]any{"PageIsSignIn": true}, wantScript: true},
		{name: "guest account link", data: map[string]any{"LinkAccountMode": true}, wantScript: true},
		{name: "guest repositories", data: map[string]any{"PageIsExploreRepositories": true}, wantScript: true},
		{name: "guest people", data: map[string]any{"PageIsExploreUsers": true}, wantScript: true},
		{name: "guest organizations", data: map[string]any{"PageIsExploreOrganizations": true}, wantScript: true},
		{name: "guest repository", data: map[string]any{"Repository": true}, wantScript: true},
		{name: "guest organization", data: map[string]any{"Org": true}, wantScript: true},
		{name: "guest recovery under subpath", data: map[string]any{"Link": "/forge/user/forgot_password"}, wantScript: true},
		{name: "recovery on wrong subpath", data: map[string]any{"Link": "/user/forgot_password"}},
		{name: "signed recovery", data: map[string]any{"IsSigned": true, "Link": "/forge/user/reset_password"}},
		{name: "signed explorer", data: map[string]any{"IsSigned": true, "PageIsExploreRepositories": true}},
		{name: "unrelated guest page", data: map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "custom/header", tt.data); err != nil {
				t.Fatalf("execute custom header: %v", err)
			}
			output := rendered.String()
			hasScript := strings.Contains(output, `/soda/forgejo/login-theme.js?v=2`)
			if hasScript != tt.wantScript {
				t.Errorf("guest theme script presence = %t, want %t:\n%s", hasScript, tt.wantScript, output)
			}
			if count := strings.Count(output, `login-theme.js`); count > 1 {
				t.Errorf("custom header rendered guest theme script %d times:\n%s", count, output)
			}
			if !regexp.MustCompile(`/soda/forgejo/components\.css\?v=[1-9][0-9]*"`).MatchString(output) {
				t.Errorf("custom header lost the shared component stylesheet:\n%s", output)
			}
		})
	}
}

func TestForgejoRepositoryCreationKeepsNativePermissionBranches(t *testing.T) {
	functions := template.FuncMap{
		"dict": forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"repo.form.reach_limit_of_creation_n": "creation limit reached",
				"repo.form.cannot_create":             "cannot create",
				"repo.create_repo":                    "create repository",
			}}}
		},
	}
	definition := `
		{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "base/alert"}}native-alert{{end}}
		{{define "custom/soda/page_intro"}}intro{{end}}
		{{define "repo/create_helper"}}native-helper{{end}}
		{{define "repo/create_basic"}}native-basic{{end}}
		{{define "repo/create_from_template"}}native-template{{end}}
		{{define "repo/create_init"}}native-init{{end}}
		{{define "repo/create_advanced"}}native-advanced{{end}}
		{{define "page"}}` + readForgejoTemplate(t, "repo", "create.tmpl") + `{{end}}`
	parsed, err := template.New("repo-create").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse repository creation page: %v", err)
	}

	tests := []struct {
		name      string
		data      map[string]any
		want      []string
		forbidden []string
	}{
		{
			name:      "personal creation allowed",
			data:      map[string]any{"CanCreateRepo": true, "Link": "/repo/create"},
			want:      []string{"native-alert", "native-helper", "native-basic", "native-template", "native-init", "native-advanced", "create repository"},
			forbidden: []string{"cannot create", "creation limit reached"},
		},
		{
			name:      "organization target available at personal limit",
			data:      map[string]any{"Orgs": []int{1}, "MaxCreationLimit": 3, "Link": "/repo/create"},
			want:      []string{"native-basic", "native-template", "native-init", "native-advanced", "creation limit reached", "create repository"},
			forbidden: []string{"cannot create"},
		},
		{
			name:      "creation denied",
			data:      map[string]any{"Link": "/repo/create"},
			want:      []string{"cannot create"},
			forbidden: []string{"native-alert", "native-helper", "native-basic", "native-template", "native-init", "native-advanced", "create repository"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "page", tt.data); err != nil {
				t.Fatalf("execute repository creation page: %v", err)
			}
			output := rendered.String()
			if !strings.Contains(output, `action="/repo/create" method="post"`) {
				t.Errorf("repository creation branch lost its native POST target:\n%s", output)
			}
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("repository creation branch does not contain %q:\n%s", want, output)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(output, forbidden) {
					t.Errorf("repository creation branch unexpectedly contains %q:\n%s", forbidden, output)
				}
			}
		})
	}
}

func TestForgejoPagesComposeSharedPresentationWithNativeBoundaries(t *testing.T) {
	introPages := []string{
		"explore/repos.tmpl",
		"explore/users.tmpl",
		"org/create.tmpl",
		"repo/create.tmpl",
		"user/dashboard/dashboard.tmpl",
		"user/dashboard/issues.tmpl",
		"user/dashboard/milestones.tmpl",
		"user/notification/notification_div.tmpl",
		"user/notification/notification_subscriptions.tmpl",
	}
	for _, name := range introPages {
		t.Run("intro/"+name, func(t *testing.T) {
			requireForgejoTemplateCalls(t, name, "custom/soda/page_intro")
		})
	}

	boundaries := []struct {
		name  string
		calls []string
	}{
		{
			name: "user/dashboard/dashboard.tmpl",
			calls: []string{
				"base/alert",
				"user/dashboard/navbar",
				"user/heatmap",
				"user/dashboard/feeds",
				"user/dashboard/guide",
				"user/dashboard/repolist",
			},
		},
		{
			name: "user/dashboard/issues.tmpl",
			calls: []string{
				"base/alert",
				"shared/search/combo",
				"shared/search/issue/syntax",
				"user/dashboard/navbar",
				"shared/issuelist",
			},
		},
		{
			name: "user/notification/notification_subscriptions.tmpl",
			calls: []string{
				"shared/issuelist",
				"shared/repo_search",
				"explore/repo_list",
				"base/paginate",
			},
		},
	}
	for _, boundary := range boundaries {
		t.Run("native/"+boundary.name, func(t *testing.T) {
			requireForgejoTemplateCalls(t, boundary.name, boundary.calls...)
		})
	}
	orgCreate := readForgejoTemplate(t, "org", "create.tmpl")
	for _, gate := range []string{".Err_OrgName", ".Err_OrgVisibility", ".visibility", ".repo_admin_change_team_access"} {
		if !strings.Contains(orgCreate, gate) {
			t.Errorf("org/create.tmpl lost native form state %s", gate)
		}
	}
	if !strings.Contains(orgCreate, `action="{{.Link}}" method="post"`) {
		t.Error("org/create.tmpl lost its native POST target")
	}

	notifications := readForgejoTemplate(t, "user", "notification", "notification_div.tmpl")
	for _, marker := range []string{`id="notification_div"`, `data-sequence-number="{{.SequenceNumber}}"`} {
		if !strings.Contains(notifications, marker) {
			t.Errorf("notification replacement fragment lost %s", marker)
		}
	}
	if templateCalls(notifications)["base/head"] || templateCalls(notifications)["base/footer"] {
		t.Error("notification_div.tmpl must remain a fragment for native notification swaps")
	}
}

func TestForgejoNativeFormAdapterSelectsMainFormsOnly(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "components-forms.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, required := range []string{
		".soda-form.ui.form",
		".soda-login .ui.form",
		".soda-native-forms :is(.user-setting-content, .repo-setting-content, .user-main-content, .org-setting-content, .admin-setting-content) > .ui.form:not(.ignore-dirty)",
		".soda-native-forms :is(.user-setting-content, .repo-setting-content, .user-main-content, .org-setting-content, .admin-setting-content) > .ui.attached.segment > .ui.form:not(.ignore-dirty)",
		"& .selection.dropdown > .default.text",
		"& .dropdown .menu > .item:hover",
		"& .primary.button",
	} {
		if !strings.Contains(css, required) {
			t.Errorf("shared form stylesheet lost the positive form scope %q", required)
		}
	}
	for _, forbidden := range []string{
		".soda-native-forms .ui.form",
		".soda-native-forms dialog .ui.form",
		".soda-native-forms table .ui.form",
		".soda-native-forms .flex-item .ui.form",
	} {
		if strings.Contains(css, forbidden) {
			t.Errorf("shared form stylesheet includes broad or compact-form scope %q", forbidden)
		}
	}
	if count := strings.Count(css, ".soda-login .ui.form"); count != 1 {
		t.Errorf("native form roots must have one declaration owner, found %d", count)
	}
}

func readForgejoTemplate(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "appliance", "forgejo", "templates"}, parts...)...)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	// Presentation roles are checked independently against the pre-migration bytes.
	// These legacy tests continue to recover their exact pinned upstream source.
	return withoutForgejoPresentationRoles(string(contents))
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

func withoutForgejoPresentationRoles(source string) string {
	return regexp.MustCompile(` soda-p-(form-host|form|title|heading|section|gap|toolbar)\b`).ReplaceAllString(source, "")
}
