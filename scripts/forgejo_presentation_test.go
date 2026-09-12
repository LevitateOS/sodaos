package scripts

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Render production partials and the production registry. This is a local
// component reference, never a replacement for native route evidence.
func TestForgejoPresentationGallery(t *testing.T) {
	if os.Getenv("SODA_FORGEJO_GALLERY") != "1" {
		t.Skip("explicit local gallery output")
	}
	header := readForgejoTemplate(t, "custom/header.tmpl")
	links := regexp.MustCompile(`<link[^>]+>`).FindAllString(header, -1)
	registry := strings.NewReplacer("{{AssetUrlPrefix}}", "http://localhost:3300/assets", "{{AppSubUrl}}", "http://localhost:3300").Replace(strings.Join(links, "\n"))
	fixtures, err := os.ReadFile("../tests/forgejo/presentation/gallery.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	// Use the exact preview's native icons too; an empty SVG stub disguises
	// icon spacing and leaves the empty-state reference visually incomplete.
	icons := map[string]string{}
	client := &http.Client{Timeout: 3 * time.Second}
	svg := func(name string, dimensions ...any) template.HTML {
		if _, exists := icons[name]; !exists {
			response, err := client.Get("http://localhost:3300/assets/img/svg/" + name + ".svg")
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil || response.StatusCode != http.StatusOK || !strings.HasPrefix(string(body), "<svg ") {
				t.Fatalf("native icon %s: status %d, error %v", name, response.StatusCode, err)
			}
			icons[name] = string(body)
		}
		icon := icons[name]
		if len(dimensions) > 0 {
			size := fmt.Sprint(dimensions[0])
			icon = regexp.MustCompile(`\b(width|height)="\d+"`).ReplaceAllString(icon, `${1}="`+size+`"`)
		}
		return template.HTML(icon)
	}
	funcs := template.FuncMap{"dict": forgejoTemplateDict, "AssetUrlPrefix": func() string { return "http://localhost:3300/assets" }, "svg": svg, "AppSubUrl": func() string { return "" }, "DisableWebhooks": func() bool { return false }, "ctx": func() forgejoTemplateContext {
		return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
			"migrate.git.description":      "Import code from any Git repository URL.",
			"migrate.github.description":   "Bring repositories and available history from GitHub.",
			"migrate.gitlab.description":   "Bring repositories and available history from GitLab.",
			"migrate.gitea.description":    "Move repositories from another Gitea instance.",
			"sign_in":                      "Sign in",
			"repo.settings":                "Settings",
			"repo.settings.options":        "Repository",
			"repo.settings.units.units":    "Repository units",
			"repo.settings.units.overview": "Overview",
			"repo.issues":                  "Issues",
			"repo.pulls":                   "Pull requests",
			"repo.wiki":                    "Wiki",
			"repo.settings.branches":       "Branches",
			"repo.settings.tags":           "Tags",
			"repo.settings.lfs":            "LFS",
			"repo.settings.collaboration":  "Collaborators",
			"repo.settings.deploy_keys":    "Deploy keys",
			"repo.settings.githooks":       "Git hooks",
			"actions.actions":              "Actions",
			"actions.runners":              "Runners",
			"secrets.secrets":              "Secrets",
			"actions.variables":            "Variables",
			"actions.runners.create_runner.properties_fieldset": "Properties",
			"actions.runners.create_runner.name_label":          "Name",
			"actions.runners.create_runner.description_label":   "Description",
			"actions.runners.create_runner.create_button":       "Create runner",
			"actions.runners.create_runner.cancel_button":       "Cancel",
			"soda.nav_personal":                                 "Personal",
			"soda.nav_access":                                   "Access & integrations",
			"soda.nav_resources":                                "Resources",
			"settings.profile":                                  "Profile",
			"settings.account":                                  "Account",
			"settings.appearance":                               "Appearance",
			"settings.security":                                 "Security",
			"settings.applications":                             "Applications",
			"settings.ssh_gpg_keys":                             "SSH/GPG keys",
			"settings.repos":                                    "Repositories",
			"settings.organization":                             "Organizations",
			"settings.blocked_users":                            "Blocked users",
			"repo.settings.hooks":                               "Webhooks",
		}}}
	}}
	source := `{{define "custom/soda/page_intro"}}` + readForgejoTemplate(t, "custom/soda/page_intro.tmpl") + `{{end}}`
	source += `{{define "custom/soda/empty_content"}}` + readForgejoTemplate(t, "custom/soda/empty_content.tmpl") + `{{end}}`
	source += `{{define "user/settings/navbar"}}` + readForgejoTemplate(t, "user/settings/navbar.tmpl") + `{{end}}`
	source += `{{define "gallery"}}` + string(fixtures) + `{{end}}`
	parsed, err := template.New("gallery").Funcs(funcs).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"repo/settings/navbar", "shared/actions/runner_create"} {
		contents, readErr := os.ReadFile("../appliance/forgejo/templates/" + name + ".tmpl")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = parsed.New(name).Parse(string(contents)); err != nil {
			t.Fatal(err)
		}
	}
	repositoryFixture, err := os.ReadFile("../tests/forgejo/presentation/repository-settings-gallery.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parsed.New("repository-settings-gallery").Parse(string(repositoryFixture)); err != nil {
		t.Fatal(err)
	}
	// Expand visual coverage with actual migration/auth page templates. The base
	// shell and sign-in fields remain explicit fixtures, not installed route proof.
	refinement, err := os.ReadFile("../tests/forgejo/presentation/refinement-gallery.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parsed.New("refinement-gallery").Parse(string(refinement)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"custom/soda/theme_toggle", "repo/migrate/migrate", "user/auth/signin"} {
		if _, err = parsed.New(name).Parse(readForgejoTemplate(t, name+".tmpl")); err != nil {
			t.Fatal(err)
		}
	}
	fixtureShell := `{{define "base/head"}}<!doctype html><html lang="en" data-theme="soda-{{.Theme}}" data-soda-login-theme="{{.Theme}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Soda page refinement</title><link rel="stylesheet" href="http://localhost:3300/assets/css/index.css"><link rel="stylesheet" href="http://localhost:3300/assets/soda/forgejo/css/theme-soda-{{.Theme}}.css">{{.Registry}}</head><body>{{end}}{{define "base/footer"}}<footer class="page-footer">Soda OS · Rendered template reference</footer></body></html>{{end}}{{define "repo/migrate/helper"}}{{end}}{{define "user/auth/signin_inner"}}<div class="ui container fluid"><form class="ui form" onsubmit="return false"><div class="field"><label for="user_name">Username or email</label><input id="user_name" autocomplete="username"></div><div class="field"><label for="password">Password</label><input id="password" type="password" autocomplete="current-password"></div><div class="field"><label><input type="checkbox"> Remember this device</label></div><button class="ui primary button">Sign in</button><a href="#password">Forgot your password?</a></form></div>{{end}}`
	if _, err = parsed.Parse(fixtureShell); err != nil {
		t.Fatal(err)
	}
	dir := "../.artifacts/forgejo-presentation"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		f, err := os.Create(filepath.Join(dir, "gallery-"+theme+".html"))
		if err != nil {
			t.Fatal(err)
		}
		err = parsed.ExecuteTemplate(f, "gallery", map[string]any{"Theme": theme, "Registry": template.HTML(registry)})
		closeErr := f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		f, err = os.Create(filepath.Join(dir, "repository-settings-"+theme+".html"))
		if err != nil {
			t.Fatal(err)
		}
		err = parsed.ExecuteTemplate(f, "repository-settings-gallery", map[string]any{
			"Theme": theme, "Registry": template.HTML(registry),
			"RepositoryFixture": map[string]any{"Title": "Repository", "RepoLink": "/fixture/repository", "IsSigned": true, "Repository": forgejoSettingsRepository{Code: true, ID: 42}, "SignedUser": map[string]any{"CanEditGitHook": true}, "LFSStartServer": true, "EnableActions": true, "Permission": forgejoSettingsPermission(true), "UnitTypeCode": 1, "UnitTypeActions": 2, "PageIsSettingsOptions": true},
			"RunnerFixture":     map[string]any{"Link": "#runner-fixture", "RunnersListLink": "#runner-fixture", "Runner": map[string]any{"Name": "Fixture runner", "Description": "Production fields; no credential generation or submission."}},
		})
		closeErr = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	for _, theme := range []string{"light", "dark"} {
		data := map[string]any{"Theme": theme, "Registry": template.HTML(registry), "Title": "Migration reference", "ShowRegistrationButton": false, "Services": []migrationChooserService{{1, "git", "Git"}, {2, "github", "GitHub"}, {3, "gitlab", "GitLab"}, {4, "gitea", "Gitea"}}}
		for _, page := range []struct{ name, source string }{{"refinement", "refinement-gallery"}, {"migration", "repo/migrate/migrate"}, {"signin", "user/auth/signin"}} {
			f, err := os.Create(filepath.Join(dir, page.name+"-"+theme+".html"))
			if err != nil {
				t.Fatal(err)
			}
			err = parsed.ExecuteTemplate(f, page.source, data)
			closeErr := f.Close()
			if err != nil {
				t.Fatal(err)
			}
			if closeErr != nil {
				t.Fatal(closeErr)
			}
		}
	}

}
