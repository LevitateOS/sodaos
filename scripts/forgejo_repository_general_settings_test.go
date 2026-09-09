package scripts

import (
	"html/template"
	"strings"
	"testing"
)

func TestForgejoRepositoryGeneralSettingsUseOpenSections(t *testing.T) {
	page := readForgejoTemplate(t, "repo", "settings", "options.tmpl")

	for _, required := range []string{
		`"title" (ctx.Locale.Tr "repo.settings.options")`,
		`repo-setting-content soda-repo-settings-detail soda-repo-settings-options`,
		`<section class="soda-settings-section">`,
		`<section class="soda-settings-section soda-settings-danger">`,
		`<div class="soda-settings-section-body">`,
		`<h2 class="soda-p-heading">`,
		`class="ui info message soda-notice"`,
		`class="ui warning message soda-notice tw-text-center"`,
		`class="ui error danger segment"`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("repository general settings lost %q", required)
		}
	}
	for _, forbidden := range []string{
		`<h4 class="ui top attached header soda-p-heading">`,
		`<div class="ui attached segment soda-p-section">`,
		` autofocus`,
	} {
		if strings.Contains(page, forbidden) {
			t.Errorf("repository general settings retains closed-card markup %q", forbidden)
		}
	}

	for _, action := range []string{
		"update", "federation", "mirror-sync", "mirror", "push-mirror-sync",
		"push-mirror-remove", "push-mirror-add", "signing", "admin", "admin_index",
		"cancel_transfer", "convert", "convert_fork", "transfer", "delete",
		"delete-wiki", "rename-wiki-branch",
	} {
		if !strings.Contains(page, `value="`+action+`"`) {
			t.Errorf("repository general settings lost action %q", action)
		}
	}
	if !strings.Contains(page, `value="{{if .Repository.IsArchived}}unarchive{{else}}archive{{end}}"`) {
		t.Error("repository general settings lost conditional archive action")
	}
	for _, modal := range []string{
		"convert-mirror-repo-modal", "convert-fork-repo-modal", "transfer-repo-modal",
		"delete-repo-modal", "delete-wiki-modal", "rename-wiki-branch-modal",
		"archive-repo-modal",
	} {
		if !strings.Contains(page, `id="`+modal+`"`) {
			t.Errorf("repository general settings lost modal %q", modal)
		}
	}
}

func TestForgejoRepositoryUnitsRetainNativeControlContracts(t *testing.T) {
	page := readForgejoTemplate(t, "repo", "settings", "units.tmpl")
	for _, required := range []string{
		`"title" (ctx.Locale.Tr "repo.settings.units.units")`,
		`repo-setting-content soda-repo-settings-detail soda-repo-settings-units`,
		`action="{{.RepoLink}}/settings/units"`,
		`{{if not .IsMirror}}`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("repository units page lost %q", required)
		}
	}

	partials := []struct {
		name     string
		heading  string
		controls []string
	}{
		{"overview.tmpl", "overview", []string{"enable_code", "enable_projects", "enable_releases", "enable_packages", "enable_actions"}},
		{"issues.tmpl", "issues", []string{"enable_issues", "issue_box", "internal_issue_box", "external_issue_box", "only_contributors", "tracker-issue-style-regex-box"}},
		{"pulls.tmpl", "pulls", []string{"enable_pulls", "pull_box", "pulls_default_merge_style", "pulls_default_update_style", "default_delete_branch_after_merge", "pulls_ignore_whitespace"}},
		{"wiki.tmpl", "wiki", []string{"enable_wiki", "wiki_box", "globally_writeable_checkbox", "external_wiki_box", "external_wiki_url"}},
	}
	for _, partial := range partials {
		t.Run(partial.name, func(t *testing.T) {
			contents := readForgejoTemplate(t, "repo", "settings", "units", partial.name)
			if _, err := template.New(partial.name).Funcs(template.FuncMap{
				"ctx": func() any { return nil },
				"svg": func(...any) string { return "" },
			}).Parse(contents); err != nil {
				t.Fatalf("parse unit partial: %v", err)
			}
			if strings.Count(contents, `<section class="soda-settings-section">`) != 1 || strings.Count(contents, `<div class="soda-settings-section-body">`) != 1 {
				t.Fatal("unit partial must expose one open settings section and body")
			}
			if !strings.Contains(contents, `class="soda-p-heading" id="`+partial.heading+`"`) {
				t.Errorf("unit partial lost anchored heading %q", partial.heading)
			}
			if strings.Contains(contents, "ui attached segment") || strings.Contains(contents, `<div class="divider">`) {
				t.Error("unit partial retains closed-card or decorative-divider markup")
			}
			for _, control := range partial.controls {
				if !strings.Contains(contents, control) {
					t.Errorf("unit partial lost native control hook %q", control)
				}
			}
		})
	}
}
