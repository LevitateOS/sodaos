package scripts

import (
	"bytes"
	"html"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

func TestForgejoOnboardingMigrationProvidersUseSharedFormLayout(t *testing.T) {
	dir := filepath.Join("..", "appliance", "forgejo", "templates", "repo", "migrate")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migration overrides: %v", err)
	}
	if len(entries) != 12 {
		t.Fatalf("expected chooser, progress, options and nine provider bodies (Forgejo delegates to Gitea); found %v", entryNames(entries))
	}
	for _, provider := range []string{"git", "github", "gitlab", "gitea", "gogs", "onedev", "gitbucket", "codebase", "pagure"} {
		source := readForgejoTemplate(t, "repo", "migrate", provider+".tmpl")
		for _, required := range []string{"soda-form-layout", "soda-form-content", `{{template "repo/migrate/options" .}}`, `name="service"`, `name="clone_addr"`, `method="post"`} {
			if !strings.Contains(source, required) {
				t.Errorf("%s provider lost %s", provider, required)
			}
		}
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
		`.soda-migrate-chooser .soda-migrate-provider`,
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

type migrationChooserService struct {
	ID    int
	Name  string
	Title string
}

func (s migrationChooserService) String() string { return strconv.Itoa(s.ID) }

func TestForgejoMigrationChooserRendersOnlyAvailableSourcesAndPreservesContext(t *testing.T) {
	seams := `{{define "base/head"}}head{{end}}{{define "base/footer"}}footer{{end}}{{define "repo/migrate/helper"}}native-migration-helper{{end}}`
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"svg":            func(_ string, _ ...any) string { return "icon" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"migrate.git.description":    "native Git description",
				"migrate.github.description": "native GitHub description",
				"migrate.custom.description": "native custom description",
			}}}
		},
	}
	parsed, err := template.New("page").Funcs(functions).Parse(seams + readForgejoTemplate(t, "repo", "migrate", "migrate.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	git := migrationChooserService{1, "git", "Git"}
	github := migrationChooserService{2, "github", "GitHub"}
	custom := migrationChooserService{91, "custom", "Custom <source>"}
	for _, tc := range []struct {
		name      string
		services  []migrationChooserService
		wantIDs   []string
		providers bool
	}{
		{"git in middle", []migrationChooserService{github, git, custom}, []string{"1", "2", "91"}, true},
		{"git unavailable", []migrationChooserService{custom, github}, []string{"91", "2"}, true},
		{"only git", []migrationChooserService{git}, []string{"1"}, false},
		{"none available", nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			data := map[string]any{"Title": "Migrate", "Services": tc.services, "Org": "42&mirror=false", "Mirror": "true"}
			if err := parsed.ExecuteTemplate(&output, "page", data); err != nil {
				t.Fatal(err)
			}
			body := output.String()
			links := regexp.MustCompile(`<a class="soda-migrate-provider[^\"]*" href="([^\"]+)"`).FindAllStringSubmatch(body, -1)
			if len(links) != len(tc.wantIDs) {
				t.Fatalf("rendered %d sources, want %d", len(links), len(tc.wantIDs))
			}
			for i, link := range links {
				u, err := url.Parse(html.UnescapeString(link[1]))
				if err != nil {
					t.Fatal(err)
				}
				if u.Path != "/forge/repo/migrate" || u.Query().Get("service_type") != tc.wantIDs[i] || u.Query().Get("org") != "42&mirror=false" || u.Query().Get("mirror") != "true" || len(u.Query()) != 3 {
					t.Errorf("native route/context changed: %s", u)
				}
			}
			if strings.Contains(body, "Hosting services") != tc.providers {
				t.Error("empty or missing hosting-services group")
			}
			if strings.Contains(body, "No migration sources are available.") != (len(tc.services) == 0) {
				t.Error("incorrect unavailable state")
			}
			for _, s := range tc.services {
				if !strings.Contains(body, "native "+map[string]string{"git": "Git", "github": "GitHub", "custom": "custom"}[s.Name]+" description") {
					t.Errorf("lost native %s description", s.Name)
				}
				if s.Name == "custom" && !strings.Contains(body, "Custom &lt;source&gt;") {
					t.Error("provider title was not escaped")
				}
			}
			for _, want := range []string{"native-migration-helper", `/forge/assets/soda/forgejo/migrate-papercraft.png`, `aria-labelledby="soda-migrate-source-title"`} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(body, "<form") || strings.Contains(body, "<script") {
				t.Error("chooser must remain ordinary native navigation")
			}
		})
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
