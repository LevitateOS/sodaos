package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoRepositoryHeaderKeepsNativeAuthority(t *testing.T) {
	header := readForgejoTemplate(t, "repo", "header.tmpl")
	for _, marker := range []string{
		`class="secondary-nav soda-page-marker soda-repository-header"`,
		`data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
		`class="repo-header"`,
		`class="repo-buttons button-row"`,
		`class="ui dropdown custom soda-repository-switcher"`,
		`{{if $.RepoTransfer}}`,
		`{{if not $.DisableStars}}`,
		`{{if not $.DisableForks}}`,
		`{{if .Permission.CanRead $.UnitTypeCode}}`,
		`{{if .Permission.CanRead $.UnitTypeIssues}}`,
		`{{if .Permission.IsAdmin}}`,
		`id="settings-btn"`,
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("repository header lost native marker %q", marker)
		}
	}

	for _, partial := range []string{
		"repo/icon",
		"repo/watch_unwatch",
		"repo/star_unstar",
		"repo/header_fork",
		"custom/extra_tabs",
	} {
		if !templateCalls(header)[partial] {
			t.Errorf("repository header no longer delegates to native template %q", partial)
		}
	}
}

type repositoryHeaderPermission struct {
	read  bool
	admin bool
}

func (p repositoryHeaderPermission) CanRead(unit any) bool {
	return p.read && unit == "code"
}

func (p repositoryHeaderPermission) CanReadAny(...any) bool { return false }
func (p repositoryHeaderPermission) IsAdmin() bool          { return p.admin }

type repositoryHeaderVisibility struct{}

func (repositoryHeaderVisibility) IsPrivate() bool { return false }

type repositoryHeaderOwner struct {
	ID         int64
	Name       string
	HomeLink   string
	Visibility repositoryHeaderVisibility
}

type repositoryHeaderRepository struct {
	ID                int64
	Owner             repositoryHeaderOwner
	Name              string
	RepoLink          string
	DefaultBranch     string
	ObjectFormatName  string
	IsBeingCreated    bool
	IsBroken          bool
	IsArchived        bool
	IsPrivate         bool
	IsTemplate        bool
	IsFork            bool
	IsGenerated       bool
	NumOpenProjects   int
	NumOpenActionRuns int
	CanPullRequests   bool
}

func (repositoryHeaderRepository) CanEnablePulls() bool  { return false }
func (repositoryHeaderRepository) NumOpenIssues(any) int { return 0 }
func (repositoryHeaderRepository) NumOpenPulls(any) int  { return 0 }

type repositoryHeaderSignedUser struct {
	IsAdmin             bool
	EnableRepoUnitHints bool
}

type repositoryHeaderLocale struct{}

func (repositoryHeaderLocale) Tr(key string, _ ...any) string { return key }

type repositoryHeaderContext struct{ Locale repositoryHeaderLocale }

type repositoryHeaderDateUtils struct{}

func (repositoryHeaderDateUtils) TimeSince(any) string { return "" }

func renderForgejoRepositoryHeader(t *testing.T, permissions repositoryHeaderPermission, approvedExtensionPages []map[string]string) string {
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
		"Permission": permissions, "ExtensionPages": map[string][]map[string]string{"repository": approvedExtensionPages},
		"ExtensionPageURL": "/native-owner/native-workspace/workspace", "RepoLink": "/native-owner/native-workspace",
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

func TestForgejoRepositoryHeaderRendersWorkspaceOnceAndKeepsAdminSettings(t *testing.T) {
	approvedExtensionPages := []map[string]string{
		{"Title": "Workspace", "URL": "/native-owner/native-workspace/workspace", "Permission": "read"},
		{"Title": "Repository settings contribution", "URL": "/native-owner/native-workspace/settings/workspace", "Permission": "admin"},
	}
	body := renderForgejoRepositoryHeader(t, repositoryHeaderPermission{read: true, admin: true}, approvedExtensionPages)
	feedBody := renderForgejoRepositoryHeader(t, repositoryHeaderPermission{read: true, admin: true}, []map[string]string{
		{"ID": "repository-feed", "Title": "Feed", "URL": "/native-owner/native-workspace/workspace", "Permission": "read"},
	})
	if !strings.Contains(feedBody, "icon Feed") {
		t.Fatal("repository feed tab rendered without its icon")
	}
	if got := strings.Count(body, `href="/native-owner/native-workspace/workspace"`); got != 1 {
		t.Fatalf("native repository workspace link rendered %d times, want once", got)
	}
	if strings.Contains(body, "Repository settings contribution") || strings.Contains(body, `href="/native-owner/native-workspace/settings/workspace"`) {
		t.Fatal("admin-only native contribution appeared as a repository header tab")
	}
	workspaceTab := strings.Index(body, `href="/native-owner/native-workspace/workspace"`)
	firstNativeTab := strings.Index(body, "repo.code")
	if workspaceTab < 0 || firstNativeTab < 0 || workspaceTab > firstNativeTab {
		t.Fatalf("extension tab at %d, first native tab content at %d, want extension first", workspaceTab, firstNativeTab)
	}
	if !strings.Contains(body, `href="/native-owner/native-workspace/settings"`) {
		t.Fatal("repository admin lost the existing settings entry point")
	}
}

func TestForgejoRepositorySettingsUseSharedLayout(t *testing.T) {
	head := readForgejoTemplate(t, "repo", "settings", "layout_head.tmpl")
	for _, marker := range []string{
		`class="page-content soda-page soda-settings-shell soda-repository-settings soda-native-forms {{.pageClass}}"`,
		`data-signed="{{if .ctxData.IsSigned}}true{{else}}false{{end}}"`,
		`class="ui container soda-page-container"`,
	} {
		if !strings.Contains(head, marker) {
			t.Errorf("repository settings layout lost %q", marker)
		}
	}
	requireForgejoTemplateCalls(t, "repo/settings/layout_head.tmpl", "base/head", "repo/header", "repo/settings/navbar", "base/alert")
}

func TestForgejoRepositoryStylesStayScopedToNativePages(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "repository.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, scope := range []string{
		`.page-content:has(> .soda-repository-header)`,
		`:where(.page-content:has(> .soda-repository-header)) > .ui.container:not(.fluid)`,
		`.soda-repository-header .repo-header .repo-buttons`,
	} {
		if !strings.Contains(css, scope) {
			t.Errorf("repository stylesheet lost page-family scope %q", scope)
		}
	}
	for _, forbidden := range []string{
		"body:has(", "#navbar", ".page-footer",
		".page-content.repository.file", ".page-content.repository.issue-list",
		".page-content.repository.view.issue", ".page-content.repository.commits",
		".page-content.repository.branches", ".page-content.repository.tags",
		".page-content.repository.releases", ".page-content.repository.wiki",
		".page-content.repository.projects", ".page-content.repository.actions",
		":not(.soda-repository-settings)",
		"background: var(--soda-page-action)", "background: var(--soda-page-action-hover)",
	} {
		if strings.Contains(css, forbidden) {
			t.Errorf("repository stylesheet took over shared shell selector %q", forbidden)
		}
	}
}
