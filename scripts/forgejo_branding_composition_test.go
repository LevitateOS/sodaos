package scripts

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NFX-H03/H05 source composition: approved extension tabs stay discoverable
// in the branded settings navigation, and the repository header keeps its
// tab, switcher and native account/repository controls under both entry
// orders (native page current, extension page current). History stays with
// native anchors: no Soda branding script manipulates it.

func brandingSettingsFuncs(disabledHooks bool) template.FuncMap {
	return template.FuncMap{
		"AppSubUrl":       func() string { return "" },
		"ctx":             func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} },
		"svg":             func(string, ...any) string { return "" },
		"DisableWebhooks": func() bool { return disabledHooks },
	}
}

func renderBrandingTemplate(t *testing.T, name, source string, data map[string]any) string {
	t.Helper()
	parsed, err := template.New(name).Funcs(brandingSettingsFuncs(true)).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := parsed.Execute(&out, data); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestForgejoRepositorySettingsNavbarKeepsAdminExtensionTabs(t *testing.T) {
	source := readForgejoTemplate(t, "repo/settings/navbar.tmpl")
	data := map[string]any{
		"RepoLink": "/fixture/repository", "Title": "Settings", "Context": nil, "IsSigned": true,
		"UnitTypeCode": 1, "UnitTypeActions": 2,
		"Repository": forgejoSettingsRepository{Code: true, ID: 7},
		"SignedUser": map[string]any{"CanEditGitHook": false},
		"Permission": forgejoSettingsPermission(false),
		"ExtensionPages": map[string][]map[string]string{"repository": {
			{"Title": "Workspace settings", "URL": "/fixture/repository/settings/workspace", "Permission": "admin"},
			{"Title": "Workspace", "URL": "/fixture/repository/workspace", "Permission": "read"},
		}},
		"ExtensionPageURL": "/fixture/repository/settings/workspace",
	}
	body := renderBrandingTemplate(t, "repo-settings-nav", source, data)
	if got := strings.Count(body, `href="/fixture/repository/settings/workspace"`); got != 1 {
		t.Fatalf("admin extension tab rendered %d times, want once", got)
	}
	if !strings.Contains(body, `href="/fixture/repository/settings/workspace" aria-current="page">Workspace settings`) {
		t.Fatal("current admin extension tab lost its aria-current marker")
	}
	if strings.Contains(body, `href="/fixture/repository/workspace"`) {
		t.Fatal("non-admin repository tab leaked into settings navigation")
	}
	if !strings.Contains(body, `href="/fixture/repository/settings"`) {
		t.Fatal("native repository settings destination lost")
	}
}

func TestForgejoUserSettingsNavbarKeepsExtensionTabs(t *testing.T) {
	source := readForgejoTemplate(t, "user/settings/navbar.tmpl")
	data := map[string]any{
		"ExtensionPages": map[string][]map[string]string{"user": {
			{"Title": "Account extension", "URL": "/user/settings/extension", "Permission": "read"},
		}},
		"ExtensionPageURL": "/user/settings/extension",
	}
	body := renderBrandingTemplate(t, "user-settings-nav", source, data)
	if got := strings.Count(body, `href="/user/settings/extension"`); got != 1 {
		t.Fatalf("user extension tab rendered %d times, want once", got)
	}
	if !strings.Contains(body, `href="/user/settings/extension" aria-current="page">Account extension`) {
		t.Fatal("current user extension tab lost its aria-current marker")
	}
	if !strings.Contains(body, `href="/user/settings/account"`) {
		t.Fatal("native account settings destination lost")
	}
	data["HideNavbarLinks"] = true
	if body := renderBrandingTemplate(t, "user-settings-nav", source, data); strings.Contains(body, "/user/settings/extension") {
		t.Fatal("extension tab leaked while the native navbar is hidden")
	}
}

func TestForgejoHeaderTabsSwitcherAndControlsSurviveBothEntryOrders(t *testing.T) {
	pages := []map[string]string{
		{"Title": "Workspace", "URL": "/native-owner/native-workspace/workspace", "Permission": "read"},
	}
	for _, current := range []string{"/native-owner/native-workspace", "/native-owner/native-workspace/workspace"} {
		body := renderForgejoRepositoryHeaderAt(t, current, pages)
		if got := strings.Count(body, `href="/native-owner/native-workspace/workspace"`); got != 1 {
			t.Fatalf("entry %s: workspace tab rendered %d times, want once", current, got)
		}
		if !strings.Contains(body, `soda-repository-switcher`) {
			t.Fatalf("entry %s: repository switcher lost", current)
		}
		if !strings.Contains(body, `href="/native-owner/native-workspace/settings"`) {
			t.Fatalf("entry %s: native repository settings control lost", current)
		}
		active := strings.Contains(body, `active item" href="/native-owner/native-workspace/workspace"`)
		if want := current == "/native-owner/native-workspace/workspace"; active != want {
			t.Fatalf("entry %s: workspace tab active=%t, want %t", current, active, want)
		}
	}
}

func renderForgejoRepositoryHeaderAt(t *testing.T, pageURL string, pages []map[string]string) string {
	t.Helper()
	functions := template.FuncMap{
		"AppSubUrl":        func() string { return "" },
		"svg":              func(...any) string { return "icon" },
		"ctx":              func() repositoryHeaderContext { return repositoryHeaderContext{} },
		"RepoFlagsEnabled": func() bool { return false },
		"MirrorRemoteAddress": func(any, any) struct{ Address string } {
			return struct{ Address string }{}
		},
		"DateUtils": func() repositoryHeaderDateUtils { return repositoryHeaderDateUtils{} },
		"CountFmt":  func(any) string { return "0" },
	}
	source := `{{define "repo/header"}}` + readForgejoTemplate(t, "repo", "header.tmpl") + `{{end}}
{{define "repo/icon"}}REPOSITORY_ICON{{end}}
{{define "repo/watch_unwatch"}}WATCH_CONTROL{{end}}
{{define "repo/star_unstar"}}STAR_CONTROL{{end}}
{{define "repo/header_fork"}}FORK_CONTROL{{end}}
{{define "custom/extra_tabs"}}NATIVE_EXTRA_TABS{{end}}`
	tmpl, err := template.New("repository-header").Funcs(functions).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]any{
		"Repository": repositoryHeaderRepository{
			ID: 17, Owner: repositoryHeaderOwner{ID: 5, Name: "native-owner", HomeLink: "/native-owner"},
			Name: "native-workspace", RepoLink: "/native-owner/native-workspace", DefaultBranch: "main",
		},
		"Permission":       repositoryHeaderPermission{read: true, admin: true},
		"ExtensionPages":   map[string][]map[string]string{"repository": pages},
		"ExtensionPageURL": pageURL, "RepoLink": "/native-owner/native-workspace",
		"SignedUser": repositoryHeaderSignedUser{}, "SignedUserID": int64(7), "IsSigned": true,
		"UnitTypeCode": "code", "UnitTypeIssues": "issues", "UnitTypeExternalTracker": "external-tracker",
		"UnitTypePullRequests": "pulls", "UnitTypeProjects": "projects", "UnitTypeReleases": "releases",
		"UnitTypePackages": "packages", "UnitTypeWiki": "wiki", "UnitTypeExternalWiki": "external-wiki", "UnitTypeActions": "actions",
		"DisableStars": true, "DisableForks": true,
	}
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "repo/header", data); err != nil {
		t.Fatal(err)
	}
	return rendered.String()
}

func TestForgejoAccountControlsStayNativeAndScriptsLeaveHistoryAlone(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "frontend", "forgejo", "payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	// The global account/navbar and admin navigation keep Fountain's native
	// templates (including their global/admin extension tabs); Soda ships no
	// override that could replace the native account controls.
	for _, native := range []string{"templates/base/head_navbar.tmpl", "templates/admin/navbar.tmpl"} {
		if _, overridden := payload[native]; overridden {
			t.Errorf("payload overrides native account navigation %s", native)
		}
	}
	for _, script := range []string{"repository-switcher.ts", "repository-actions.ts", "notification-preview.ts", "personal-settings.ts", "login-theme.ts"} {
		contents, err := os.ReadFile(filepath.Join("..", "assets", "branding", "forgejo", script))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"history.pushState", "history.replaceState", "location.assign", "location.replace"} {
			if strings.Contains(string(contents), forbidden) {
				t.Errorf("%s manipulates history via %s; native anchors own it", script, forbidden)
			}
		}
	}
}
