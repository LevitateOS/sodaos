package scripts

import (
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

func TestForgejoRepositorySettingsUseSharedLayout(t *testing.T) {
	head := readForgejoTemplate(t, "repo", "settings", "layout_head.tmpl")
	for _, marker := range []string{
		`class="page-content soda-page soda-repository-settings soda-native-forms {{.pageClass}}"`,
		`data-signed="{{if .ctxData.IsSigned}}true{{else}}false{{end}}"`,
		`class="ui container flex-container soda-settings-layout"`,
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
		`.page-content:has(> .soda-repository-header) > .ui.container:not(.fluid)`,
		`.soda-repository-header .repo-header .repo-buttons`,
		`:is(.ui.primary.button,.primary.button):not(.basic)`,
		`.ui.basic.button:not(.red)`,
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
