package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Structural presentation is reviewed in the test-only inventory; the pinned
// controls, gates and script hooks are checked by repository-settings-source.test.mjs.
func TestForgejoRepositorySettingsLeavesUseSharedShell(t *testing.T) {
	for _, name := range []string{"actions", "branches", "collaboration", "deploy_keys", "githooks", "githook_edit", "lfs", "lfs_file", "lfs_file_find", "lfs_locks", "lfs_pointers", "options", "protected_branch", "runner_create", "runner_details", "runner_edit", "runner_setup", "secrets", "tags", "units", "webhook/base", "webhook/new"} {
		path := "repo/settings/" + name + ".tmpl"
		requireForgejoTemplateCalls(t, path, "repo/settings/layout_head", "repo/settings/layout_footer")
		source := readForgejoTemplate(t, strings.Split(path, "/")...)
		if strings.Count(source, "repo-setting-content") != 1 {
			t.Errorf("%s must select exactly one shared content canvas", path)
		}
		if !strings.Contains(source, `"title"`) {
			t.Errorf("%s must provide its task title to the shell", path)
		}
	}
}

func TestForgejoWebhookPartialsRetain1507Source(t *testing.T) {
	partials := []struct{ name, hash string }{
		{"base_list.tmpl", "4f9799349c043b5ece5757c8fd1929b13304b033916d667c624de73064c705cb"},
		{"history.tmpl", "ebb007fd23ec9cd8e8ffbe057f78bb516674d245d7c8672b27d29558ef2a3432"},
	}
	for _, tt := range partials {
		page := readForgejoTemplate(t, "repo", "settings", "webhook", tt.name)
		page = strings.ReplaceAll(page, " soda-webhook-list-header", "")
		page = strings.ReplaceAll(page, " soda-webhook-list", "")
		page = strings.ReplaceAll(page, " soda-webhook-history-header", "")
		page = strings.ReplaceAll(page, " soda-webhook-history", "")
		if tt.name == "base_list.tmpl" {
			_, page, _ = strings.Cut(page, `{{$settings := .SettingsPresentation}}{{$ := .ctxData}}{{with .ctxData}}`+"\n")
			page = strings.TrimSuffix(page, "\n{{end}}{{end}}\n")
			page = strings.Replace(page, `{{if not $settings}}{{.Title}}{{end}}`, `{{.Title}}`, 1)
			page = strings.Replace(page, `{{if $settings}}<div class="soda-toolbar soda-settings-inventory-actions">{{else}}<h4 class="ui top attached header">{{end}}`, `<h4 class="ui top attached header">`, 1)
			page = strings.Replace(page, `{{if $settings}}</div>{{else}}</h4>{{end}}`, `</h4>`, 1)
			page = strings.Replace(page, `{{if and $settings (not .Webhooks)}}<div class="soda-empty soda-empty--page">{{template "custom/soda/empty_content" dict "Icon" "octicon-webhook" "Title" (ctx.Locale.Tr "repo.issues.filter_no_results") "Description" .Description}}</div>{{else}}`, ``, 1)
			page = strings.Replace(page, "</div>{{end}}", "</div>", 1)
		}

		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(page))); got != tt.hash {
			t.Errorf("%s diverges from exact Forgejo 15.0.7 beyond presentation classes: got %s, want %s", tt.name, got, tt.hash)
		}
	}
}

func TestForgejoRepositorySettingsDetailStylesStayScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "repository-settings-details.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, required := range []string{
		".soda-repository-settings .soda-settings-layout",
		".soda-repo-settings-detail",
		".soda-repo-settings-branches",
	} {
		if !strings.Contains(css, required) {
			t.Errorf("detail stylesheet lost scoped family %q", required)
		}
	}
	for _, forbidden := range []string{"body ", "#navbar", ".ui.form {", ".ui.red.button"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("detail stylesheet contains broad selector %q", forbidden)
		}
	}
}

func TestForgejoSharedRunnerStylesStayWithSharedPartial(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "runners.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, required := range []string{".soda-shared-runner", ".soda-shared-runner .runner-list", ".soda-shared-runner dl > .item"} {
		if !strings.Contains(css, required) {
			t.Errorf("runner stylesheet lost shared partial scope %q", required)
		}
	}
	for _, forbidden := range []string{"body ", "#navbar", ".ui.form {", ".ui.red.button"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("runner stylesheet contains broad selector %q", forbidden)
		}
	}
}

func TestForgejoSharedRunnerDetailsRetain1507Source(t *testing.T) {
	hashes := map[string]string{
		"runner_create.tmpl":  "e05805b4f7885dc082c586ecea87db0fae758b8fa269a4c4a63c07c730d0cc6a",
		"runner_details.tmpl": "1ecf2232c0d2c434bad3fdbb7049f31ec6399fe959322d00fbe2dce5b0eca73e",
		"runner_edit.tmpl":    "7d4e39ed9b99fad63834fd1bff7cf13a679cbf1e15edae95084b75a0075df0bd",
		"runner_list.tmpl":    "93fc7a1fa402d9a6ac7f7f896271e41ad14365c6540daee3b730dac8cbbf429e",
		"runner_setup.tmpl":   "21a5b973b08357eabf61af30cce642e80a4eb2c9cb293ddd5fbca9c84c3f30e5",
	}
	for name, upstreamHash := range hashes {
		t.Run(name, func(t *testing.T) {
			page := readForgejoTemplate(t, "shared", "actions", name)
			slug := strings.ReplaceAll(strings.TrimSuffix(name, ".tmpl"), "_", "-")
			branded := `class="runner-container soda-shared-runner soda-shared-` + slug + `"`
			if strings.Count(page, branded) != 1 {
				t.Fatalf("%s must expose exactly one shared runner styling root", name)
			}
			recovered := strings.Replace(page, branded, `class="runner-container"`, 1)
			if strings.Contains(recovered, `{{$settings := .SettingsPresentation}}`) {
				// Reviewed opted-in settings title suppression. Native root data is
				// explicitly rebound before entering the original body.
				prefix := `{{$settings := .SettingsPresentation}}{{$ := .ctxData}}{{with .ctxData}}` + "\n"
				_, recovered, _ = strings.Cut(recovered, prefix)
				recovered = strings.TrimSuffix(recovered, "\n{{end}}{{end}}\n")
				recovered = strings.Replace(recovered, `{{if not $settings}}{{ctx.Locale.Tr "actions.runners.runner_title" .Runner.Name}}{{end}}`, `{{ctx.Locale.Tr "actions.runners.runner_title" .Runner.Name}}`, 1)
				if name != "runner_details.tmpl" {
					recovered = strings.Replace(recovered, `{{if not $settings}}`, "", 1)
					recovered = strings.Replace(recovered, `</h4>{{end}}`, `</h4>`, 1)
				}
			}
			if name == "runner_details.tmpl" {
				recovered = strings.Replace(recovered, `{{if $settings}}<div class="soda-toolbar soda-settings-inventory-actions">{{else}}<h4 class="ui top attached header">{{end}}`, `<h4 class="ui top attached header">`, 1)
				recovered = strings.Replace(recovered, `{{if $settings}}</div>{{else}}</h4>{{end}}`, `</h4>`, 1)
			}
			if name == "runner_list.tmpl" {
				recovered = strings.Replace(recovered, `{{if $settings}}<div class="soda-empty soda-empty--compact">{{template "custom/soda/empty_content" dict "Icon" "octicon-play" "Title" (ctx.Locale.Tr "actions.runners.none")}}</div>{{else}}<div class="tw-flex tw-p-4">
			{{ctx.Locale.Tr "actions.runners.none"}}
		</div>{{end}}`, `<div class="tw-flex tw-p-4">
			{{ctx.Locale.Tr "actions.runners.none"}}
		</div>`, 1)
			}
			if name == "runner_setup.tmpl" {
				recovered = strings.Replace(recovered, `<p{{if $settings}} class="ui warning message soda-notice"{{end}}>`, `<p>`, 1)
			}
			recovered = strings.ReplaceAll(recovered, `<fieldset{{if $settings}} class="soda-form-section"{{end}}>`, `<fieldset>`)
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(recovered)))
			if got != upstreamHash {
				t.Fatalf("%s diverges from exact Forgejo 15.0.7 beyond its presentation class: got %s, want %s", name, got, upstreamHash)
			}
		})
	}
}
