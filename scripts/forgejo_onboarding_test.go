package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoOnboardingPreservesNativeRoutesAndFields(t *testing.T) {
	migrate := readForgejoTemplate(t, "repo", "migrate", "migrate.tmpl")
	for _, want := range []string{
		`{{template "repo/migrate/helper" .}}`,
		`range .Services`,
		`/repo/migrate?service_type={{.}}&org={{$.Org}}&mirror={{$.Mirror}}`,
		`ctx.Locale.Tr (printf "migrate.%s.description" .Name)`,
		`class="page-content repository new migrate soda-page soda-onboarding soda-migrate-chooser"`,
	} {
		if !strings.Contains(migrate, want) {
			t.Errorf("migration chooser lost native contract %q", want)
		}
	}

	fork := readForgejoTemplate(t, "repo", "pulls", "fork.tmpl")
	for _, want := range []string{
		`action="{{.Link}}" method="post"`,
		`name="uid" value="{{.ContextUser.ID}}" required`,
		`{{if .CanForkToUser}}`,
		`{{range .Orgs}}`,
		`href="{{.ForkRepo.Link}}"`,
		`name="repo_name" value="{{.repo_name}}" required`,
		`disabled {{if .IsPrivate}}checked{{end}}`,
		`name="fork_single_branch" value="" required`,
		`{{range .Branches}}`,
		`name="description">{{.description}}</textarea>`,
		`primary button{{if not .CanForkRepo}} disabled{{end}}`,
	} {
		if !strings.Contains(fork, want) {
			t.Errorf("fork form lost native contract %q", want)
		}
	}
	if strings.Contains(fork, `name="private"`) {
		t.Error("fork override turned inherited visibility into a mutable field")
	}
}

func TestForgejoOnboardingMigrationProvidersRemainNativeFallbacks(t *testing.T) {
	dir := filepath.Join("..", "appliance", "forgejo", "templates", "repo", "migrate")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migration overrides: %v", err)
	}
	if len(entries) != 3 || entries[0].Name() != "migrate.tmpl" || entries[1].Name() != "migrating.tmpl" || entries[2].Name() != "options.tmpl" {
		t.Fatalf("provider forms must remain native fallbacks; found %v", entryNames(entries))
	}
	options := readForgejoTemplate(t, "repo", "migrate", "options.tmpl")
	if strings.Count(options, `class="soda-page-marker"`) != 1 || !strings.Contains(options, `data-signed="{{.IsSigned}}" hidden`) {
		t.Fatalf("migration options must emit one inert native-theme marker: %s", options)
	}
	parsed, err := template.New("repo/migrate/options").Funcs(template.FuncMap{
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{}}}
		},
	}).Parse(options)
	if err != nil {
		t.Fatalf("parse migration options: %v", err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, map[string]any{"IsSigned": true}); err != nil {
		t.Fatalf("render migration options: %v", err)
	}
	if strings.Count(rendered.String(), `class="soda-page-marker"`) != 1 || !strings.Contains(rendered.String(), `data-signed="true" hidden`) {
		t.Fatalf("migration options did not render one signed marker: %s", rendered.String())
	}
	for _, want := range []string{`name="mirror"`, `name="lfs"`, `id="lfs_settings_show"`, `name="lfs_endpoint"`, `.Err_LFSEndpoint`, `.ContextUser.CanImportLocal`} {
		if !strings.Contains(options, want) {
			t.Errorf("migration options lost native field or branch %q", want)
		}
	}

	css := readForgejoAssetFile(t, "onboarding.css")
	for _, want := range []string{
		`.repository.new.migrate:not(.soda-migrate-chooser)`,
		`.soda-migrate-chooser .migrate-entry`,
		`.soda-fork .soda-form`,
		`.page-content.repository:has(#repo_migrating)`,
		`#repo_migrating_progress_message`,
		`#repo_migrating_failed_error`,
		`#repo_migrating_failed_image svg`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("onboarding CSS missing scoped native seam %q", want)
		}
	}
	for _, forbidden := range []string{"input[name=", "form[action=", "body .ui.form", ".tw-hidden", "#repo_migrating_retry {"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("onboarding CSS reaches behavior-sensitive native markup with %q", forbidden)
		}
	}
}

func TestForgejoOnboardingMigratingPreservesNativeRuntimeHooks(t *testing.T) {
	source := readForgejoTemplate(t, "repo", "migrate", "migrating.tmpl")
	for _, want := range []string{
		`data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
		`{{template "repo/header" .}}`,
		`id="repo_migrating"`,
		`data-migrating-task-id="{{.MigrateTask.ID}}"`,
		`/img/forgejo-loading.svg`,
		`id="repo_migrating_failed_image"`,
		`id="repo_migrating_progress"`,
		`id="repo_migrating_progress_message"`,
		`id="repo_migrating_failed"`,
		`id="repo_migrating_failed_error"`,
		`{{if .Permission.IsAdmin}}`,
		`{{if .Failed}}`,
		`data-modal="#delete-repo-modal"`,
		`data-modal="#cancel-repo-modal"`,
		`id="repo_migrating_retry"`,
		`data-migrating-task-retry-url="{{.Link}}/settings/migrate/retry"`,
		`id="delete-repo-modal"`,
		`action="{{.Link}}/settings" method="post"`,
		`name="action" value="delete"`,
		`id="repo_name_to_delete" name="repo_name" required`,
		`id="cancel-repo-modal"`,
		`action="{{.Link}}/settings/migrate/cancel" method="post"`,
		`{{template "base/modal_actions_confirm" .}}`,
	} {
		if !strings.Contains(source, want) {
			t.Errorf("migrating page lost native runtime contract %q", want)
		}
	}
	if strings.Contains(source, `data-signed="true"`) {
		t.Error("migrating page assumes authentication even though public migrating repositories use the anonymous repository route")
	}
	for _, id := range []string{"repo_migrating", "repo_migrating_failed_image", "repo_migrating_progress", "repo_migrating_progress_message", "repo_migrating_failed", "repo_migrating_failed_error", "repo_migrating_retry", "delete-repo-modal", "cancel-repo-modal"} {
		if strings.Count(source, `id="`+id+`"`) != 1 {
			t.Errorf("migrating page must retain exactly one #%s hook", id)
		}
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func readForgejoAssetFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "assets", "branding", "forgejo", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
