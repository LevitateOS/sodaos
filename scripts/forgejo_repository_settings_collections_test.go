package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readRepositorySettingsCollectionTemplate(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "appliance", "forgejo", "templates", "repo", "settings", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

func TestForgejoRepositorySettingsCollectionsUseOpenSettingsComposition(t *testing.T) {
	landing := map[string]string{
		"branches.tmpl":      `"title" (ctx.Locale.Tr "repo.settings.branches")`,
		"collaboration.tmpl": `"title" (ctx.Locale.Tr "repo.settings.collaboration")`,
		"deploy_keys.tmpl":   `"title" (ctx.Locale.Tr "repo.settings.deploy_keys")`,
		"githooks.tmpl":      `"title" (ctx.Locale.Tr "repo.settings.githooks")`,
		"lfs.tmpl":           `"title" (ctx.Locale.Tr "repo.settings.lfs_filelist")`,
		"tags.tmpl":          `"title" (ctx.Locale.Tr "repo.settings.tags.protection")`,
	}
	for name, title := range landing {
		t.Run(name, func(t *testing.T) {
			page := readRepositorySettingsCollectionTemplate(t, name)
			for _, required := range []string{`class="repo-setting-content`, title} {
				if !strings.Contains(page, required) {
					t.Errorf("%s lost repository settings composition marker %q", name, required)
				}
			}
			if strings.Contains(page, `<h4 class="ui top attached header`) {
				t.Errorf("%s restores a duplicate attached task heading", name)
			}
		})
	}

	children := map[string][]string{
		"protected_branch.tmpl": {`"parentURL"`, `"parentTitle" (ctx.Locale.Tr "repo.settings.branches")`, `soda-settings-editor`},
		"githook_edit.tmpl":     {`"parentURL"`, `"parentTitle" (ctx.Locale.Tr "repo.settings.githooks")`, `soda-settings-editor`},
		"lfs_file.tmpl":         {`"parentURL" .LFSFilesLink`, `"parentTitle" (ctx.Locale.Tr "repo.settings.lfs")`, `repo-setting-content`},
		"lfs_file_find.tmpl":    {`"parentURL" .LFSFilesLink`, `"parentTitle" (ctx.Locale.Tr "repo.settings.lfs")`, `repo-setting-content`},
		"lfs_locks.tmpl":        {`"parentURL" .LFSFilesLink`, `"parentTitle" (ctx.Locale.Tr "repo.settings.lfs")`, `repo-setting-content`},
		"lfs_pointers.tmpl":     {`"parentURL" .LFSFilesLink`, `"parentTitle" (ctx.Locale.Tr "repo.settings.lfs")`, `repo-setting-content`},
	}
	for name, required := range children {
		t.Run(name, func(t *testing.T) {
			page := readRepositorySettingsCollectionTemplate(t, name)
			for _, marker := range required {
				if !strings.Contains(page, marker) {
					t.Errorf("%s lost child settings marker %q", name, marker)
				}
			}
			if strings.Contains(page, `<h4 class="ui top attached header`) {
				t.Errorf("%s restores the native breadcrumb-shaped duplicate heading", name)
			}
		})
	}
}

func TestForgejoRepositorySettingsCollectionsRetainNativeInteractionContracts(t *testing.T) {
	checks := map[string][]string{
		"branches.tmpl": {
			`name="action" value="default_branch"`, `name="branch"`, `data-modal-id="delete-protected-branch"`,
		},
		"collaboration.tmpl": {
			`class="ui form soda-p-form" id="repo-collab-form"`, `id="search-user-box"`, `data-url="{{$.Link}}/access_mode"`, `id="repo-collab-team-form"`, `id="delete-collaborator"`,
		},
		"deploy_keys.tmpl": {
			`data-panel="#add-deploy-key-panel"`, `id="ssh-key-title"`, `id="ssh-key-content"`, `name="is_writable"`, `id="delete-deploy-key"`,
		},
		"githook_edit.tmpl": {
			`id="content"`, `name="content"`, `template "shared/codemirror_container"`,
		},
		"tags.tmpl": {
			`id="search-tag-box"`, `{{if .PageIsEditProtectedTag}}autofocus {{end}}required`, `name="allowlist_users"`, `name="allowlist_teams"`, `class="ui button" href="{{$.RepoLink}}/settings/tags"`, `name="id" value="{{.ID}}"`,
		},
		"protected_branch.tmpl": {
			`name="rule_id"`, `id="whitelist_box"`, `id="approvals_whitelist_box"`, `id="statuscheck_contexts_box"`, `id="merge_whitelist_box"`,
		},
		"lfs_file.tmpl": {
			`unescape-button`, `escape-button`, `template "repo/unicode_escape_prompt"`, `class="file-view`,
		},
		"lfs_locks.tmpl": {
			`ctx.Locale.Tr "admin.total" .Total`, `class="ui form ignore-dirty soda-p-form"`, `name="path"`, `/unlock`, `id="lfs-files-locks-table"`,
		},
		"lfs_pointers.tmpl": {
			`action="{{$.Link}}/associate"`, `name="oid"`, `id="lfs-files-table"`,
		},
		"lfs.tmpl": {
			`ctx.Locale.Tr "admin.total" .Total`, `id="lfs-files-table"`,
		},
	}
	for name, markers := range checks {
		t.Run(name, func(t *testing.T) {
			page := readRepositorySettingsCollectionTemplate(t, name)
			for _, marker := range markers {
				if !strings.Contains(page, marker) {
					t.Errorf("%s lost native interaction contract %q", name, marker)
				}
			}
		})
	}
	for _, name := range []string{"collaboration.tmpl"} {
		page := readRepositorySettingsCollectionTemplate(t, name)
		if strings.Contains(page, "autofocus") {
			t.Errorf("%s must not steal focus when its landing collection opens", name)
		}
	}
}

func TestForgejoRepositorySettingsCollectionsUseSharedEmptyAndNoticeRoles(t *testing.T) {
	emptyPages := []string{"collaboration.tmpl", "deploy_keys.tmpl", "githooks.tmpl", "lfs.tmpl", "lfs_file_find.tmpl", "lfs_pointers.tmpl"}
	for _, name := range emptyPages {
		page := readRepositorySettingsCollectionTemplate(t, name)
		for _, required := range []string{`soda-empty--page`, `template "custom/soda/empty_content"`} {
			if !strings.Contains(page, required) {
				t.Errorf("%s lost empty-page role %q", name, required)
			}
		}
	}

	emptySections := []string{"branches.tmpl", "collaboration.tmpl", "tags.tmpl", "lfs_locks.tmpl"}
	for _, name := range emptySections {
		page := readRepositorySettingsCollectionTemplate(t, name)
		if !strings.Contains(page, `soda-empty--compact`) || !strings.Contains(page, `template "custom/soda/empty_content"`) {
			t.Errorf("%s lost compact section-empty composition", name)
		}
	}

	for _, name := range []string{"branches.tmpl", "deploy_keys.tmpl", "githooks.tmpl", "githook_edit.tmpl", "tags.tmpl"} {
		page := readRepositorySettingsCollectionTemplate(t, name)
		if !strings.Contains(page, `message soda-notice`) {
			t.Errorf("%s lost semantic static guidance", name)
		}
	}
}
