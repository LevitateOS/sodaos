package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

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
		`soda-page-intro compact&#34; data-owned=&#34;true`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered intro does not contain escaped %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{`<img`, `<script>`, `onclick="alert(1)"`, `onerror="alert(1)`} {
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
