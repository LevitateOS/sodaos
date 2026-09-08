package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type forgejoSharedProjectsLocale struct{}

func (forgejoSharedProjectsLocale) Tr(key string, _ ...any) string { return key }
func (forgejoSharedProjectsLocale) PrettyNumber(value any) any     { return value }

type forgejoSharedProjectsContext struct{ Locale forgejoSharedProjectsLocale }

func TestForgejoSharedProjectsPreserveNativeOwnershipAndForms(t *testing.T) {
	list := readForgejoTemplate(t, "projects", "list.tmpl")
	for _, marker := range []string{
		`and $.CanWriteProjects (not $.Repository.IsArchived)`,
		`class="switch list-header-toggle soda-tabs"`,
		`class="primary button soda-toolbar-action soda-toolbar-action--primary"`,
		`template "shared/search/combo"`,
		`class="link-action flex-text-inline"`,
		`data-url="{{.Link ctx}}/open"`,
		`data-url="{{.Link ctx}}/close"`,
		`data-url="{{.Link ctx}}/delete"`,
		`data-modal-id="delete-project"`,
		`template "base/modal_actions_confirm" .`,
		`class="milestone-list soda-shared-project-list"`,
	} {
		if !strings.Contains(list, marker) {
			t.Errorf("shared project list lost native contract %q", marker)
		}
	}
	ownerList := readForgejoTemplate(t, "org", "projects", "list.tmpl")
	if !strings.Contains(ownerList, "soda-profile-card-context") {
		t.Error("user project list lost the shared profile-card component marker")
	}

	form := readForgejoTemplate(t, "projects", "new.tmpl")
	for _, marker := range []string{
		`class="ui form soda-form" action="{{.Link}}" method="post"`,
		`class="soda-form-section"`,
		`if .PageIsEditProjects`,
		`if not .PageIsEditProjects`,
		`name="template_type"`,
		`range $element := .TemplateConfigs`,
		`name="card_type"`,
		`range $element := .CardTypes`,
		`href="{{$.CancelLink}}"`,
		`ctx.Locale.Tr "repo.projects.modify"`,
		`ctx.Locale.Tr "repo.projects.create"`,
	} {
		if !strings.Contains(form, marker) {
			t.Errorf("shared project form lost native contract %q", marker)
		}
	}
}

func TestForgejoSharedProjectsHaveOneStyleOwner(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "assets", "branding", "forgejo", "projects.css"))
	if err != nil {
		t.Fatalf("read project styles: %v", err)
	}
	css := string(contents)
	for _, want := range []string{
		".soda-shared-project-list", ".soda-projects-list", ".soda-project-board-page",
		".soda-project-board .project-column", ".soda-project-board .issue-card",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("project stylesheet lacks scoped owner %q", want)
		}
	}
	for _, name := range []string{"packages.css", "profiles.css", "repository-content.css"} {
		other, err := os.ReadFile(filepath.Join("..", "assets", "branding", "forgejo", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, escaped := range []string{".soda-shared-project-list", ".soda-projects-list", ".soda-project-board"} {
			if strings.Contains(string(other), escaped) {
				t.Errorf("%s still owns project selector %q", name, escaped)
			}
		}
	}
}

func TestForgejoSharedProjectsFormExecutesNewAndEditBranches(t *testing.T) {
	form := readForgejoTemplate(t, "projects", "new.tmpl")
	parsed, err := template.New("project-form").Funcs(template.FuncMap{
		"ctx": func() forgejoSharedProjectsContext { return forgejoSharedProjectsContext{} },
		"svg": func(string, ...any) template.HTML { return "<svg></svg>" },
	}).Parse(`{{define "base/alert"}}alert{{end}}{{define "form"}}` + form + `{{end}}`)
	if err != nil {
		t.Fatalf("parse shared project form: %v", err)
	}
	data := map[string]any{
		"Link": "/forge/alice&projects/new", "CancelLink": "/forge/alice/projects",
		"TemplateConfigs": []any{}, "CardTypes": []any{}, "card_type": 1,
	}
	for _, test := range []struct {
		name             string
		edit             bool
		wantAction       string
		wantTemplateType bool
	}{
		{name: "new", wantAction: "repo.projects.create", wantTemplateType: true},
		{name: "edit", edit: true, wantAction: "repo.projects.modify"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data["PageIsEditProjects"] = test.edit
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "form", data); err != nil {
				t.Fatalf("render %s project form: %v", test.name, err)
			}
			output := rendered.String()
			for _, want := range []string{`action="/forge/alice&amp;projects/new" method="post"`, `class="soda-form-section"`, test.wantAction} {
				if !strings.Contains(output, want) {
					t.Errorf("%s project form lost %q:\n%s", test.name, want, output)
				}
			}
			if got := strings.Contains(output, `name="template_type"`); got != test.wantTemplateType {
				t.Errorf("%s project form template selector presence = %t, want %t", test.name, got, test.wantTemplateType)
			}
		})
	}
}
