package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoCodeSearchKeepsNativeSearchContext(t *testing.T) {
	definition := `
		{{define "base/head"}}head{{end}}
		{{define "base/footer"}}footer{{end}}
		{{define "custom/explore_navbar"}}navbar/{{.Route}}{{end}}
		{{define "shared/search/code/search"}}code-search/{{.Route}}/{{.CodeSearchPath}}{{end}}
		{{define "custom/soda/page_intro"}}intro/{{.Title}}{{end}}
		{{define "page"}}` + readForgejoTemplate(t, "explore", "code.tmpl") + `{{end}}`
	parsed, err := template.New("code-search").Funcs(template.FuncMap{"dict": forgejoTemplateDict}).Parse(definition)
	if err != nil {
		t.Fatalf("parse code explorer: %v", err)
	}
	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "page", map[string]any{
		"Title": "Code", "Route": "code", "CodeSearchPath": "internal/web", "IsSigned": true,
	}); err != nil {
		t.Fatalf("render code explorer: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`class="page-content explore users soda-page soda-explore soda-explore-code"`,
		`data-signed="true"`, `navbar/code`, `code-search/code/internal/web`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("code explorer does not contain %q:\n%s", want, output)
		}
	}
	if strings.Count(output, "code-search/code/internal/web") != 1 {
		t.Errorf("native code search rendered more than once:\n%s", output)
	}
}

func TestForgejoPackagesOwnerPagesRetainOrganizationAndUserBranches(t *testing.T) {
	functions := template.FuncMap{
		"dict": forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"packages.title": "Packages", "packages.versions": "Versions",
			}}}
		},
	}
	seams := `
		{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}
		{{define "org/header"}}org-header/{{.Owner}}{{end}}
		{{define "shared/user/profile_big_avatar"}}avatar/{{.Owner}}{{end}}
		{{define "user/overview/header"}}overview/{{.Owner}}{{end}}
		{{define "custom/soda/page_intro"}}intro/{{.Title}}{{end}}
		{{define "package/shared/list"}}package-list/{{.Owner}}{{end}}
		{{define "package/shared/versionlist"}}version-list/{{.Owner}}{{end}}`
	tests := []struct {
		name, file, leaf string
	}{
		{"packages", "packages.tmpl", "package-list"},
		{"versions", "package_versions.tmpl", "version-list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := seams + `{{define "page"}}` + readForgejoTemplate(t, "user", "overview", tt.file) + `{{end}}`
			parsed, err := template.New(tt.name).Funcs(functions).Parse(definition)
			if err != nil {
				t.Fatalf("parse %s owner page: %v", tt.name, err)
			}
			for _, tc := range []struct {
				name         string
				org          bool
				want, absent string
			}{
				{"organization", true, "org-header/acme", "avatar/acme"},
				{"user", false, "avatar/alice", "org-header/alice"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var rendered bytes.Buffer
					data := map[string]any{
						"Title": "Packages", "Owner": map[bool]string{true: "acme", false: "alice"}[tc.org],
						"IsSigned": true, "ContextUser": map[string]any{"IsOrganization": tc.org},
						"PackageDescriptor": map[string]any{"Package": map[string]any{"Name": "widget"}},
					}
					if err := parsed.ExecuteTemplate(&rendered, "page", data); err != nil {
						t.Fatalf("render %s %s page: %v", tt.name, tc.name, err)
					}
					output := rendered.String()
					for _, want := range []string{tc.want, tt.leaf + "/" + data["Owner"].(string), `soda-page soda-packages`} {
						if !strings.Contains(output, want) {
							t.Errorf("%s %s page lacks %q:\n%s", tt.name, tc.name, want, output)
						}
					}
					if strings.Contains(output, tc.absent) {
						t.Errorf("%s %s page crossed owner branch with %q:\n%s", tt.name, tc.name, tc.absent, output)
					}
					if got, want := strings.Contains(output, "soda-profile-card-context"), !tc.org; got != want {
						t.Errorf("%s %s shared profile-card marker presence = %t, want %t:\n%s", tt.name, tc.name, got, want, output)
					}
				})
			}
		})
	}
}

func TestForgejoPackagesNativeActionsAndProtocolPartialsRemain(t *testing.T) {
	sharedList := readForgejoTemplate(t, "package", "shared", "list.tmpl")
	for _, want := range []string{
		`name="type"`, `range $type := .AvailableTypes`, `index $.RepositoryAccessMap .Repository.ID`,
		`if and .Repository .CanWritePackages`, `URLJoin .Owner.HomeLink "-" "packages"`,
		`https://forgejo.org/docs/latest/user/packages/`, `template "base/paginate" .`,
	} {
		if !strings.Contains(sharedList, want) {
			t.Errorf("package list lost native contract %q", want)
		}
	}

	versions := readForgejoTemplate(t, "package", "shared", "versionlist.tmpl")
	for _, want := range []string{
		`name="sort"`, `eq .PackageDescriptor.Package.Type "container"`, `name="tagged"`,
		`template "shared/search/button"`, `template "base/paginate" .`,
	} {
		if !strings.Contains(versions, want) {
			t.Errorf("package versions lost native contract %q", want)
		}
	}

	view := readForgejoTemplate(t, "package", "view.tmpl")
	for _, protocol := range []string{
		"alpine", "arch", "cargo", "chef", "composer", "conan", "conda", "container", "cran", "debian",
		"generic", "go", "helm", "maven", "npm", "nuget", "pub", "pypi", "rpm", "alt", "rubygems", "swift", "vagrant",
	} {
		if !strings.Contains(view, `template "package/content/`+protocol+`" .`) {
			t.Errorf("package detail lost native %s content partial", protocol)
		}
		if protocol != "go" && !strings.Contains(view, `template "package/metadata/`+protocol+`" .`) {
			t.Errorf("package detail lost native %s metadata partial", protocol)
		}
	}
	for _, want := range []string{`<main aria-label="{{.Title}}"`, `class="ui container soda-page-container"`, `</main>`, ".HasRepositoryAccess", ".CanWritePackages", "/files/{{.File.ID}}", "/settings"} {
		if !strings.Contains(view, want) {
			t.Errorf("package detail lost native access/action contract %q", want)
		}
	}

	settings := readForgejoTemplate(t, "package", "settings.tmpl")
	for _, want := range []string{
		`<main aria-label="{{.Title}}"`, `class="ui container soda-page-container"`, `</main>`,
		`action="{{.Link}}" method="post"`, `name="action" value="link"`, `name="repo_id"`,
		`data-modal="#delete-package-modal"`, `name="action" value="delete"`, `template "base/modal_actions_confirm" .`,
	} {
		if !strings.Contains(settings, want) {
			t.Errorf("package settings lost native form/modal contract %q", want)
		}
	}
	if strings.Count(settings, `class="ui form soda-form"`) != 1 || strings.Count(settings, `class="ui form"`) != 1 {
		t.Errorf("package settings must adapt only the link form and keep the delete modal form native")
	}
}

func TestForgejoPackagesStylesStayScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "packages.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read package stylesheet: %v", err)
	}
	css := string(contents)
	for _, want := range []string{".soda-packages", ".soda-package-view", ".soda-package-settings", ".soda-package-cleanup-list", ".soda-package-cleanup-edit", ".soda-package-preview-table"} {
		if !strings.Contains(css, want) {
			t.Errorf("package stylesheet lacks scoped owner %q", want)
		}
	}
	for _, forbidden := range []string{"\nbody {", "#navbar", ".page-footer", ".soda-code-search", ".soda-project-board", ".soda-shared-project-list"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("package stylesheet reaches shared shell with %q", forbidden)
		}
	}
	for _, want := range []string{"display: block", "max-width: 100%", "overflow-x: auto"} {
		if !strings.Contains(css, want) {
			t.Errorf("package preview table lost effective overflow rule %q", want)
		}
	}
}

func TestForgejoPackagesCleanupRulesKeepNativeActionsAndData(t *testing.T) {
	functions := template.FuncMap{
		"ctx":         func() forgejoTemplateContext { return forgejoTemplateContext{} },
		"svg":         func(...any) string { return "" },
		"StringUtils": func() any { return struct{}{} },
		"DateUtils":   func() any { return struct{}{} },
	}
	for _, name := range []string{"list.tmpl", "edit.tmpl", "preview.tmpl"} {
		if _, err := template.New(name).Funcs(functions).Parse(readForgejoTemplate(t, "package", "shared", "cleanup_rules", name)); err != nil {
			t.Fatalf("parse cleanup-rule %s: %v", name, err)
		}
	}

	list := readForgejoTemplate(t, "package", "shared", "cleanup_rules", "list.tmpl")
	for _, want := range []string{
		`range .CleanupRules`, `href="{{.Link}}/rules/add"`, `href="{{$.Link}}/rules/{{.ID}}"`,
		`href="{{$.Link}}/rules/{{.ID}}/preview"`, `.KeepCount`, `.KeepPattern`, `.RemoveDays`, `.RemovePattern`,
	} {
		if !strings.Contains(list, want) {
			t.Errorf("cleanup-rule list lost native contract %q", want)
		}
	}
	if !strings.Contains(list, `class="soda-list"`) || !strings.Contains(list, `class="flex-list"`) {
		t.Error("cleanup-rule list does not compose the shared list around the native list")
	}

	edit := readForgejoTemplate(t, "package", "shared", "cleanup_rules", "edit.tmpl")
	for _, want := range []string{
		`action="{{.Link}}" method="post"`, `name="id"`, `name="enabled"`, `name="type"`,
		`name="match_full_name"`, `name="keep_count"`, `name="keep_pattern"`, `name="remove_days"`,
		`name="remove_pattern"`, `name="action" value="save"`, `name="action" value="remove"`,
		`href="{{.Link}}/preview"`, `.Err_Type`, `.Err_KeepCount`, `.Err_KeepPattern`, `.Err_RemoveDays`, `.Err_RemovePattern`,
	} {
		if !strings.Contains(edit, want) {
			t.Errorf("cleanup-rule editor lost native form contract %q", want)
		}
	}
	if strings.Count(edit, `class="ui form soda-form"`) != 1 {
		t.Error("cleanup-rule editor must adapt its single native form exactly once")
	}

	preview := readForgejoTemplate(t, "package", "shared", "cleanup_rules", "preview.tmpl")
	for _, want := range []string{
		`len .VersionsToRemove`, `range .VersionsToRemove`, `href="{{.VersionWebLink}}"`,
		`href="{{.Creator.HomeLink}}"`, `.CalculateBlobSize`, `.Version.CreatedUnix`, `colspan="6"`,
	} {
		if !strings.Contains(preview, want) {
			t.Errorf("cleanup-rule preview lost native data contract %q", want)
		}
	}
}
