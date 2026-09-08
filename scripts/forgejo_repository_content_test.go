package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoRepositoryContentOverridesRetain1507Source(t *testing.T) {
	tests := []struct {
		name         string
		path         []string
		upstreamHash string
		stockClass   string
		sodaClass    string
	}{
		{"wiki start", []string{"repo", "wiki", "start.tmpl"}, "df9bba8dd8bd9a8b7f24ac4fe958ea15a37cfc8aba93a9d8784953f1a7b58877", "page-content repository wiki start", "page-content repository wiki start soda-repository-content soda-wiki-start"},
		{"wiki pages", []string{"repo", "wiki", "pages.tmpl"}, "f51549e3a26f4047022a0ea036feba2facc8f6dde285eb691874a7a7ce2c5c30", "page-content repository wiki pages", "page-content repository wiki pages soda-repository-content soda-wiki-pages"},
		{"wiki editor", []string{"repo", "wiki", "new.tmpl"}, "41ccd691cd117d62d760f8d0b3e64c4e36b84a7c4960218897cc60554ea67ded", "page-content repository wiki new", "page-content repository wiki new soda-repository-content soda-wiki-editor"},
		{"wiki revisions", []string{"repo", "wiki", "revision.tmpl"}, "cb35be04efc2167a55f190294507804e7b1dbb659f59e9a324d820aea8450164", "page-content repository wiki revisions", "page-content repository wiki revisions soda-repository-content soda-wiki-revisions"},
		{"project list", []string{"repo", "projects", "list.tmpl"}, "393a0457150580cab1c757a361a7e0f999203c23ba94f017139f56518ef01fc7", "page-content repository projects milestones", "page-content repository projects milestones soda-repository-content soda-projects-list"},
		{"project editor", []string{"repo", "projects", "new.tmpl"}, "e41a6fc08613492002ef37ea4c46123aa64dd77a87193144df9921bf2626663f", "page-content repository projects edit-project new milestone", "page-content repository projects edit-project new milestone soda-repository-content soda-projects-new"},
		{"project board", []string{"repo", "projects", "view.tmpl"}, "9263f45e6652fb23c6100c72da27a069c04cb2b936dc1b5cda54f59dda46f0b5", "page-content repository projects view-project", "page-content repository projects view-project soda-repository-content soda-projects-view"},
		{"release list", []string{"repo", "release", "list.tmpl"}, "4225a3f054e68ea9d69772bbaa230d8de625c88a17da415bcbcb571d3e765794", "page-content repository releases", "page-content repository releases soda-repository-content soda-release-list"},
		{"release editor", []string{"repo", "release", "new.tmpl"}, "7e58e9fd1283d4fd10362b9fa53b39510c902d8d7943503b0e23cb9acf6472cb", "page-content repository new release", "page-content repository new release soda-repository-content soda-release-editor"},
		{"wiki view", []string{"repo", "wiki", "view.tmpl"}, "781d4d2a4dc174b7c00e96db8bba77dfcad110546d3a6575c4bdde3e135a3758", "page-content repository wiki view", "page-content repository wiki view soda-repository-content soda-wiki-view"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := readForgejoTemplate(t, tt.path...)
			normalized := strings.Replace(contents, `class="`+tt.sodaClass+`"`, `class="`+tt.stockClass+`"`, 1)
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
			if got != tt.upstreamHash {
				t.Fatalf("override differs from pinned Forgejo 15.0.7 source beyond its page-class delta: got %s, want %s", got, tt.upstreamHash)
			}
		})
	}
}

func TestForgejoRepositoryContentKeepsNativeGatesAndPartials(t *testing.T) {
	tests := []struct {
		name    string
		path    []string
		markers []string
	}{
		{"wiki start", []string{"repo", "wiki", "start.tmpl"}, []string{`{{if and .CanWriteWiki (not .Repository.IsMirror)}}`, `{{template "repo/header" .}}`}},
		{"wiki pages", []string{"repo", "wiki", "pages.tmpl"}, []string{`{{if and .CanWriteWiki (not .Repository.IsMirror)}}`, `data-tooltip-content=`, `{{template "repo/header" .}}`}},
		{"wiki editor", []string{"repo", "wiki", "new.tmpl"}, []string{`{{if .PageIsWikiEdit}}`, `{{template "shared/combomarkdowneditor"`, `method="post"`}},
		{"wiki revisions", []string{"repo", "wiki", "revision.tmpl"}, []string{`{{if and .Commits (gt .CommitCount 0)}}`, `{{template "repo/commits_list" .}}`, `{{template "base/paginate" .}}`}},
		{"project list", []string{"repo", "projects", "list.tmpl"}, []string{`{{template "projects/list" .}}`, `{{template "repo/header" .}}`}},
		{"project editor", []string{"repo", "projects", "new.tmpl"}, []string{`{{template "projects/new" .}}`, `{{template "repo/header" .}}`}},
		{"project board", []string{"repo", "projects", "view.tmpl"}, []string{`{{if (not $.Repository.IsArchived)}}`, `{{template "repo/issue/navbar" .}}`, `{{template "projects/view" .}}`}},
		{"release list", []string{"repo", "release", "list.tmpl"}, []string{`$.Permission.CanRead $.UnitTypeCode`, `{{if and $.CanCreateRelease (not $release.IsTag)}}`, `{{template "repo/release_tag_header" .}}`}},
		{"release editor", []string{"repo", "release", "new.tmpl"}, []string{`{{if .PageIsEditRelease}}`, `{{template "shared/combomarkdowneditor"`, `id="attachment-template"`, `id="delete-release"`}},
		{"wiki view", []string{"repo", "wiki", "view.tmpl"}, []string{`{{if and .CanWriteWiki (not .Repository.IsMirror)}}`, `hx-get="{{$.RepoLink}}/wiki/search"`, `id="delete-wiki-page"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := readForgejoTemplate(t, tt.path...)
			for _, marker := range tt.markers {
				if !strings.Contains(contents, marker) {
					t.Errorf("override lost native gate or partial %q", marker)
				}
			}
		})
	}
}

func TestForgejoWikiSearchFragmentRetains1507Structure(t *testing.T) {
	contents := readForgejoTemplate(t, "repo", "wiki", "search.tmpl")
	normalized := strings.Replace(contents, "item soda-wiki-search-result tw-max-w", "item tw-max-w", 1)
	normalized = strings.Replace(normalized, "item muted soda-wiki-search-empty", "item muted", 1)
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
	const upstreamHash = "a1bc2b8774b415885f65ae031f74daddc1b003c48b8226528d8ad2d5044c4007"
	if got != upstreamHash {
		t.Fatalf("wiki search fragment differs from pinned Forgejo 15.0.7 source beyond presentation classes: got %s, want %s", got, upstreamHash)
	}
	if templateCalls(contents)["base/head"] || templateCalls(contents)["base/footer"] {
		t.Fatal("wiki search must remain a fragment for native HTMX replacement")
	}
}

func TestForgejoReleaseTagHeaderRetainsNativePolicy(t *testing.T) {
	contents := readForgejoTemplate(t, "repo", "release_tag_header.tmpl")
	normalized := strings.Replace(contents, `class="list-header soda-toolbar tw-justify-between"`, `class="list-header tw-justify-between"`, 1)
	normalized = strings.Replace(normalized, `class="switch soda-tabs"`, `class="switch"`, 1)
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
	const upstreamHash = "a3ab4d2a9892bad246b88e9f05f1d90835cd2958104e77cadcaeb3a0933067cc"
	if got != upstreamHash {
		t.Fatalf("release/tag header differs from pinned Forgejo 15.0.7 source beyond toolbar classes: got %s, want %s", got, upstreamHash)
	}
	for _, marker := range []string{
		`$.Permission.CanRead $.UnitTypeReleases`,
		`$.Permission.CanRead $.UnitTypeCode`,
		`{{if .ShowReleaseSearch}}`,
		`{{if and (not .PageIsTagList) .CanCreateRelease}}`,
		`{{template "repo/sub_menu" .}}`,
	} {
		if !strings.Contains(contents, marker) {
			t.Errorf("release/tag header lost native policy marker %q", marker)
		}
	}
}

func TestForgejoRepositoryContentStylesStayScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "repository-content.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, marker := range []string{
		".soda-wiki-start",
		".soda-wiki-pages",
		".soda-wiki-editor",
		".soda-wiki-revisions",
		".soda-release-list",
		".soda-release-editor",
		".soda-wiki-view",
	} {
		if !strings.Contains(css, marker) {
			t.Errorf("repository content stylesheet lost route scope %q", marker)
		}
	}
	for _, forbidden := range []string{"body:has(", "#navbar", ".delete-button {", ".danger.button {"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("repository content stylesheet exceeds presentation scope with %q", forbidden)
		}
	}
	if strings.Contains(css, ".soda-repository-content > .ui.container") {
		t.Error("repository content stylesheet must leave repository container width to the shell owner")
	}
}
