package scripts

import (
	"strings"
	"testing"
)

func TestForgejoOrgProjectsPreserveNativeContextsAndProjectPartials(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{"list.tmpl", []string{`{{if .ContextUser.IsOrganization}}`, `{{template "org/header" .}}`, `{{template "shared/user/profile_big_avatar" .}}`, `{{template "user/overview/header" .}}`, `{{template "projects/list" .}}`}},
		{"new.tmpl", []string{`{{template "shared/user/org_profile_avatar" .}}`, `{{template "user/overview/header" .}}`, `{{template "projects/new" .}}`}},
		{"view.tmpl", []string{`{{if .ContextUser.IsOrganization}}`, `{{template "org/header" .}}`, `{{template "shared/user/org_profile_avatar" .}}`, `{{template "user/overview/header" .}}`, `class="ui container fluid padded"`, `{{template "projects/view" .}}`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := readForgejoTemplate(t, "org", "projects", tt.name)
			for _, want := range tt.want {
				if !strings.Contains(source, want) {
					t.Errorf("organization project wrapper lost native contract %q", want)
				}
			}
			for _, want := range []string{`soda-page soda-org-projects`, `data-signed="{{.IsSigned}}"`, `{{template "custom/soda/page_intro"`} {
				if !strings.Contains(source, want) {
					t.Errorf("organization project wrapper missing page composition %q", want)
				}
			}
		})
	}
}
