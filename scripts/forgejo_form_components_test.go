package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoRepositoryCreationKeepsNativePermissionBranches(t *testing.T) {
	functions := template.FuncMap{
		"AppSubUrl": func() string { return "/forge" },
		"dict":      forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"repo.form.reach_limit_of_creation_n": "creation limit reached",
				"repo.form.cannot_create":             "cannot create",
				"repo.create_repo":                    "create repository",
			}}}
		},
	}
	definition := `
		{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "base/alert"}}native-alert{{end}}
		{{define "custom/soda/page_intro"}}intro{{end}}
		{{define "repo/create_helper"}}native-helper{{end}}
		{{define "repo/create_basic"}}native-basic{{end}}
		{{define "repo/create_from_template"}}native-template{{end}}
		{{define "repo/create_init"}}native-init{{end}}
		{{define "repo/create_advanced"}}native-advanced{{end}}
		{{define "page"}}` + readForgejoTemplate(t, "repo", "create.tmpl") + `{{end}}`
	parsed, err := template.New("repo-create").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse repository creation page: %v", err)
	}

	tests := []struct {
		name      string
		data      map[string]any
		want      []string
		forbidden []string
	}{
		{
			name:      "personal creation allowed",
			data:      map[string]any{"CanCreateRepo": true, "Link": "/repo/create"},
			want:      []string{"native-alert", "native-helper", "native-basic", "native-template", "native-init", "native-advanced", "create repository", `class="soda-form-options"`},
			forbidden: []string{"cannot create", "creation limit reached"},
		},
		{
			name:      "organization target available at personal limit",
			data:      map[string]any{"Orgs": []int{1}, "MaxCreationLimit": 3, "Link": "/repo/create"},
			want:      []string{"native-basic", "native-template", "native-init", "native-advanced", "creation limit reached", "create repository", `class="soda-form-options"`},
			forbidden: []string{"cannot create"},
		},
		{
			name:      "creation denied",
			data:      map[string]any{"Link": "/repo/create"},
			want:      []string{"cannot create"},
			forbidden: []string{"native-alert", "native-helper", "native-basic", "native-template", "native-init", "native-advanced", "create repository", `class="soda-form-options"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "page", tt.data); err != nil {
				t.Fatalf("execute repository creation page: %v", err)
			}
			output := rendered.String()
			if !strings.Contains(output, `action="/repo/create" method="post"`) {
				t.Errorf("repository creation branch lost its native POST target:\n%s", output)
			}
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("repository creation branch does not contain %q:\n%s", want, output)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(output, forbidden) {
					t.Errorf("repository creation branch unexpectedly contains %q:\n%s", forbidden, output)
				}
			}
		})
	}
}

func TestForgejoNativeFormAdapterSelectsMainFormsOnly(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "components-forms.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, required := range []string{
		".soda-form.ui.form",
		".soda-login .ui.form",
		".soda-native-forms :is(.user-setting-content, .repo-setting-content, .user-main-content, .org-setting-content, .admin-setting-content) > .ui.form:not(.ignore-dirty)",
		".soda-native-forms :is(.user-setting-content, .repo-setting-content, .user-main-content, .org-setting-content, .admin-setting-content) > .ui.attached.segment > .ui.form:not(.ignore-dirty)",
		"& .selection.dropdown > .default.text",
		"& .dropdown .menu > .item:hover",
		"& .primary.button",
	} {
		if !strings.Contains(css, required) {
			t.Errorf("shared form stylesheet lost the positive form scope %q", required)
		}
	}
	for _, forbidden := range []string{
		".soda-native-forms .ui.form",
		".soda-native-forms dialog .ui.form",
		".soda-native-forms table .ui.form",
		".soda-native-forms .flex-item .ui.form",
	} {
		if strings.Contains(css, forbidden) {
			t.Errorf("shared form stylesheet includes broad or compact-form scope %q", forbidden)
		}
	}
	if count := strings.Count(css, ".soda-login .ui.form"); count != 1 {
		t.Errorf("native form roots must have one declaration owner, found %d", count)
	}
}
