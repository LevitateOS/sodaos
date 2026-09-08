package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoRepositorySettingsDetailsRetain1507Source(t *testing.T) {
	tests := []struct {
		path string
		hash string
		root string
	}{
		{"branches.tmpl", "9afd3d56b0955a61da42bb73ea694905e8ada8b91109ea0e79fa8c8cbfd4ee8b", "repo-setting-content"},
		{"collaboration.tmpl", "b7b188a81097a11734b7f49ec4a2837267d199e2f072783d3bec3634df60b722", "repo-setting-content"},
		{"deploy_keys.tmpl", "177a5bb6ddb85c0a1967668fc9752ad6fa71c38536fe66f773e14c6e1b6f2718", "repo-setting-content"},
		{"githooks.tmpl", "54fb05e758cdf71321f319217d835dd3731d04d2a79c0c277d1bef1729e35305", "repo-setting-content"},
		{"githook_edit.tmpl", "1d5cc5470570d49bf879fdec0f665445b5b08942b5fef92f8a18ef66c17ea1b6", "repo-setting-content"},
		{"lfs.tmpl", "c6baa3e2f2759d103dc2c64d243822ec5fdc491ffe2de7f25a38c9d7a72a37ed", "repo-setting-content"},
		{"lfs_file.tmpl", "67a95626e03603d4200d2c1715ed4e1a42567564ff8e4bf43aefdc634be12485", "user-main-content twelve wide column content repository file list"},
		{"lfs_file_find.tmpl", "420b1be2d2c599922591fc9ac088876274dbc9c088a79edc4edf0538783dbd34", "user-main-content twelve wide column content repository file list"},
		{"lfs_locks.tmpl", "13c1d120e38cbb740af397a95daa0f48af4a9353812268cc09f45bf8ab93b0d4", "user-main-content twelve wide column content repository file list"},
		{"lfs_pointers.tmpl", "e968aa7c928c10066898fa71ddef4848cecbf93d6283b89ff075527aeff01839", "repo-setting-content"},
		{"options.tmpl", "343a90479c4cdcbf33965a2e0cd934f6f8be5465a0076559276342cd5a36d5ac", "user-main-content twelve wide column"},
		{"protected_branch.tmpl", "d4f16ae03ed440196dae2d119a7b0eb1d0d4af6d7953ee4576df1c711edb79be", "repo-setting-content"},
		{"tags.tmpl", "5e6c0f84d41088958f7aabaaaf27e9ed25370467e427be9cf1279ae5374cdd42", "repo-setting-content"},
		{"units.tmpl", "d06730797f3938c7b1a50eb1ebe66ae753c114c62ce14e6fd943d8b229accdb6", "user-main-content twelve wide column"},
		{"webhook/new.tmpl", "56f40561c4f892a3c468e8e3820a58564a19286f226e2832d2011e3bccf611a8", "repo-setting-content"},
		{"webhook/base.tmpl", "d210d3e6c5802fbf3ecefdc7a329e788ad7fae0fd04813874f10d236fd80340e", "repo-setting-content"},
		{"runner_create.tmpl", "d483965d625f53b0447d2a986635e09add3baf5d3074264adf300728042877ae", "repo-setting-content"},
		{"runner_details.tmpl", "cde4a1c36cb335036e84f4ac8c563205b705eac7e81d134bfeb24150028aa78f", "repo-setting-content"},
		{"runner_edit.tmpl", "33000c8a309b2138b9ebf951b6cb0d5ef940810f3c3cd619cb644bb3ef15f179", "repo-setting-content"},
		{"runner_setup.tmpl", "db2a1d581212b9d6e17578ca5aa1485f66b1279fdf94881c51d9c0bf9e95c397", "repo-setting-content"},
		{"actions.tmpl", "c3d3cba18e10f5d813970415b67270b8612829d936d2c61d9d74d03ea69ff2a0", "repo-setting-content"},
		{"secrets.tmpl", "bfda544f16c80cb6472eb3332f7b55ef1653a6a8a22dfbac71b2d89dd83f65d9", "repo-setting-content"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			page := readForgejoTemplate(t, append([]string{"repo", "settings"}, strings.Split(tt.path, "/")...)...)
			slug := strings.ReplaceAll(strings.TrimSuffix(tt.path, ".tmpl"), "/", "-")
			slug = strings.ReplaceAll(slug, "_", "-")
			stockRoot := `class="` + tt.root + `"`
			brandedRoot := `class="` + tt.root + ` soda-repo-settings-detail soda-repo-settings-` + slug + `"`
			if strings.Count(page, brandedRoot) != 1 {
				t.Fatalf("%s must expose exactly one detail styling root", tt.path)
			}
			recovered := strings.Replace(page, brandedRoot, stockRoot, 1)
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(recovered)))
			if got != tt.hash {
				t.Fatalf("%s diverges from exact Forgejo 15.0.7 beyond its presentation class: got %s, want %s", tt.path, got, tt.hash)
			}
		})
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
			_, page, _ = strings.Cut(page, `{{$personal := .PersonalSettings}}{{$ := .ctxData}}{{with .ctxData}}`+"\n")
			page = strings.TrimSuffix(page, "\n{{end}}{{end}}\n")
			page = strings.Replace(page, `{{if not $personal}}{{.Title}}{{end}}`, `{{.Title}}`, 1)
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
		".soda-repo-settings-units",
		".soda-repo-settings-protected-branch",
		".soda-repo-settings-tags",
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
			if strings.Contains(recovered, `{{$personal := .PersonalSettings}}`) {
				// Reviewed personal-only title suppression. Native root data is
				// explicitly rebound before entering the original body.
				prefix := `{{$personal := .PersonalSettings}}{{$ := .ctxData}}{{with .ctxData}}` + "\n"
				_, recovered, _ = strings.Cut(recovered, prefix)
				recovered = strings.TrimSuffix(recovered, "\n{{end}}{{end}}\n")
				recovered = strings.Replace(recovered, `{{if not $personal}}{{ctx.Locale.Tr "actions.runners.runner_title" .Runner.Name}}{{end}}`, `{{ctx.Locale.Tr "actions.runners.runner_title" .Runner.Name}}`, 1)
				recovered = strings.Replace(recovered, `{{if not $personal}}`, "", 1)
				recovered = strings.Replace(recovered, `</h4>{{end}}`, `</h4>`, 1)
			}
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(recovered)))
			if got != upstreamHash {
				t.Fatalf("%s diverges from exact Forgejo 15.0.7 beyond its presentation class: got %s, want %s", name, got, upstreamHash)
			}
		})
	}
}
