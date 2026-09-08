package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoAdminDetailsOverridesMatchStock1507(t *testing.T) {
	tests := []struct {
		name string
		kind string
		sha  string
	}{
		{"dashboard.tmpl", "dashboard", "66be2bd90fad0aa90981e9283c614c69d574a6fb883f1d1be9616e0db11a0f97"},
		{"config.tmpl", "config", "d44a53121631182bf6de4d77980f38915828bf114930d654824ee8a8858e7295"},
		{"auth/new.tmpl", "auth-new", "f4120ea177d7dc6569ebd47380cb7b2394606f008c29028426096061f9dd7473"},
		{"auth/edit.tmpl", "auth-edit", "d50ddd4916af1e27411d4915f37e941d868367eb53d1d939a0df0532c674a3b9"},
		{"user/new.tmpl", "user-new", "802c5ac53575e9a36989be4564d94c2f65d0b693cfddfc829f70a29191a19ff9"},
		{"user/edit.tmpl", "user-edit", "e3d1d0208fd8a6a92351789d80128bc95ab0177620d00ac16332a14b1409aa90"},
		{"applications/list.tmpl", "applications", "30c2f9040fba74ba77bcf3f17c4ef191b827a1aca11b43f190f6fec0716b8cb9"},
		{"applications/oauth2_edit.tmpl", "oauth2-edit", "4bf7bc194d224bd5093755568c72e051d0f42b9738ca1dbec2224e46db1130d5"},
		{"auth/list.tmpl", "auth-list", "3be096d353ffbbdf11eabd0d31a79968a94a1605428164825a841d1d392cf501"},
		{"user/list.tmpl", "user-list", "4bb9ad9fa86b8ea034004ef268c8b3e81243c77aba9babe2a287aae98d8f3bf3"},
		{"emails/list.tmpl", "email-list", "969d4895ff364b0a5e2c0f29e88aa06c91d2b5f41f9daa534817de85ab101eed"},
		{"repo/list.tmpl", "repo-list", "d5ec17847cadaf5ff50df1e8ccf755a6a1cb475cce26a53a57d0dcff83dacb00"},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.name, "/", "_"), func(t *testing.T) {
			contents := readForgejoTemplate(t, append([]string{"admin"}, strings.Split(tt.name, "/")...)...)
			provenance := fmt.Sprintf("{{/* Adapted from Forgejo 15.0.7 templates/admin/%s, GPL-3.0-or-later.\nUpstream: https://codeberg.org/forgejo/forgejo\nEmbedded source SHA-256: %s */}}\n", tt.name, tt.sha)
			if !strings.HasPrefix(contents, provenance) {
				t.Fatalf("%s lost exact Forgejo version, GPL attribution, or embedded-source provenance", tt.name)
			}
			restored := strings.TrimPrefix(contents, provenance)
			if tt.name == "user/new.tmpl" {
				restored = strings.Replace(restored, ` "artwork" "admin-new-account-papercraft.png"`, "", 1)
			}
			if tt.name == "dashboard.tmpl" || tt.name == "applications/oauth2_edit.tmpl" || tt.name == "auth/list.tmpl" || tt.name == "user/list.tmpl" || tt.name == "repo/list.tmpl" || tt.name == "emails/list.tmpl" {
				restored = strings.Replace(restored, ` "hideArtwork" true`, "", 1)
			}
			custom := `class="admin-setting-content soda-admin-details soda-admin-details--` + tt.kind + `"`
			if count := strings.Count(restored, custom); count != 1 {
				t.Fatalf("%s has %d scoped native content roots, want 1", tt.name, count)
			}
			restored = strings.Replace(restored, custom, `class="admin-setting-content"`, 1)
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(restored))); got != tt.sha {
				t.Errorf("%s differs from pristine Forgejo 15.0.7 outside its attributed content-root class: got SHA-256 %s, want %s", tt.name, got, tt.sha)
			}
			calls := templateCalls(contents)
			if !calls["admin/layout_head"] || !calls["admin/layout_footer"] {
				t.Errorf("%s lost the shared native administrator layout seams", tt.name)
			}
		})
	}
}

func TestForgejoAdminDetailsKeepNativeActionsAndBranches(t *testing.T) {
	tests := []struct {
		name      string
		required  []string
		delegates []string
	}{
		{
			name: "dashboard.tmpl",
			required: []string{
				`<form method="post" action="{{AppSubUrl}}/admin">`,
				`name="op" value="{{.}}"`,
				`hx-get="{{$.Link}}/system_status"`,
				`hx-trigger="every 5s"`,
				`hx-indicator=".no-loading-indicator"`,
			},
			delegates: []string{"admin/system_status"},
		},
		{
			name: "config.tmpl",
			required: []string{
				`action="{{AppSubUrl}}/admin/config/test_mail" method="post"`,
				`name="email"`,
				`action="{{AppSubUrl}}/admin/config/test_cache" method="post"`,
				`.GlobalTwoFactorRequirement`, `.DbCfg.Type`, `.SSH.Disabled`, `.LFS.StartServer`, `.Loggers`,
			},
		},
		{
			name: "auth/new.tmpl",
			required: []string{
				`action="{{.Link}}" method="post"`, `id="auth_type" name="type" value="{{.type}}"`,
				`id="auth_name" name="name" value="{{.name}}" autofocus required`,
				`name="is_active" type="checkbox"`,
			},
			delegates: []string{"base/disable_form_autofill", "admin/auth/source/ldap", "admin/auth/source/smtp", "admin/auth/source/oauth"},
		},
		{
			name: "auth/edit.tmpl",
			required: []string{
				`action="{{.Link}}" method="post"`, `id="auth_type" name="type" value="{{.Source.Type.Int}}"`,
				`{{if .Source.IsLDAP}}`, `{{if .Source.IsSMTP}}`, `{{if .Source.IsOAuth2}}`,
				`data-url="{{$.Link}}/delete"`, `data-modal-id="delete-auth-source"`, `id="delete-auth-source"`,
			},
			delegates: []string{"base/disable_form_autofill", "base/modal_actions_confirm"},
		},
		{
			name: "user/new.tmpl",
			required: []string{
				`action="{{.Link}}" method="post"`,
				`id="login_type" name="login_type" value="{{.login_type}}" data-password="required" required`,
				`name="visibility" value="{{if .visibility}}`,
				`name="user_name" value="{{.user_name}}" autofocus required maxlength="40"`,
				`name="password" type="password" autocomplete="new-password" value="{{.password}}" {{if eq .login_type "0-0"}}required{{end}}`,
				`{{if .CanSendEmail}}`, `name="send_notify" type="checkbox"`,
			},
			delegates: []string{"base/disable_form_autofill"},
		},
		{
			name: "user/edit.tmpl",
			required: []string{
				`action="./edit" method="post"`, `{{if not .DisableRegularOrgCreation}}`, `{{if .TwoFactorEnabled}}`,
				`name="prohibit_login" type="checkbox"`, `{{if (eq .User.ID .SignedUserID)}}disabled{{end}}`,
				`name="allow_git_hook" type="checkbox"`, `{{if DisableGitHooks}}disabled{{end}}`,
				`action="./avatar" method="post" enctype="multipart/form-data"`, `accept="image/png,image/jpeg,image/gif,image/webp"`,
				`data-modal="#delete-user-modal"`, `action="./delete"`, `name="purge" type="checkbox"`,
			},
			delegates: []string{"base/disable_form_autofill", "base/modal_actions_confirm"},
		},
		{
			name:      "applications/list.tmpl",
			delegates: []string{"user/settings/applications_oauth2_list"},
		},
		{
			name:      "applications/oauth2_edit.tmpl",
			delegates: []string{"user/settings/applications_oauth2_edit_form"},
		},
		{
			name: "auth/list.tmpl",
			required: []string{
				`href="{{AppSubUrl}}/admin/auths/new"`, `{{range .Sources}}`, `{{if .IsActive}}`,
				`href="{{AppSubUrl}}/admin/auths/{{.ID}}"`, `colspan="7"`,
			},
		},
		{
			name: "user/list.tmpl",
			required: []string{
				`id="user-list-search-form"`, `name="status_filter[is_admin]"`, `name="status_filter[is_active]"`,
				`name="status_filter[is_restricted]"`, `name="status_filter[is_prohibit_login]"`,
				`name="status_filter[is_2fa_enabled]"`, `name="status_filter[account_type]"`,
				`name="sort" value="recentupdate"`, `{{range .Users}}`, `{{if index $.UsersTwoFaStatus .ID}}`,
				`href="{{$.Link}}/{{.ID}}/edit"`, `class="no-results-row"`,
			},
			delegates: []string{"shared/search/combo", "base/paginate"},
		},
		{
			name: "emails/list.tmpl",
			required: []string{
				`href="?sort=reverseemail&q={{$.Keyword}}"`, `class="link-email-action"`,
				`data-primary="{{if .IsPrimary}}1{{else}}0{{end}}"`, `data-activate="{{if .IsActivated}}0{{else}}1{{end}}"`,
				`id="change-email-modal"`, `id="email-action-form" action="{{AppSubUrl}}/admin/emails/activate" method="post"`,
				`id="form-uid" name="uid"`, `id="form-email" name="email"`, `id="delete-email"`,
			},
			delegates: []string{"shared/search/combo", "base/paginate", "base/modal_actions_confirm"},
		},
		{
			name: "repo/list.tmpl",
			required: []string{
				`{{range .Repos}}`, `{{if .IsArchived}}`, `{{if .IsPrivate}}`, `{{if .IsTemplate}}`,
				`{{if eq .ObjectFormatName "sha256"}}`, `{{if .IsMirror}}`,
				`data-url="{{$.Link}}/delete?page={{$.Page.Paginater.Current}}&sort={{$.SortType}}"`,
				`data-modal-id="delete-repo"`, `id="delete-repo"`,
			},
			delegates: []string{"shared/repo_search", "base/paginate", "base/modal_actions_confirm"},
		},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.name, "/", "_"), func(t *testing.T) {
			contents := readForgejoTemplate(t, append([]string{"admin"}, strings.Split(tt.name, "/")...)...)
			for _, required := range tt.required {
				if !strings.Contains(contents, required) {
					t.Errorf("%s lost native contract %q", tt.name, required)
				}
			}
			calls := templateCalls(contents)
			for _, delegate := range tt.delegates {
				if !calls[delegate] {
					t.Errorf("%s lost native delegate %q", tt.name, delegate)
				}
			}
		})
	}
}

func TestForgejoAdminDetailsCSSIsScopedAndResponsive(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "admin-details.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, selector := range []string{
		".soda-admin-details--config .admin-dl-horizontal",
		".soda-admin-details--dashboard table .primary.button",
		".soda-admin-details--auth-edit #ldap-group-options",
		".soda-admin-details--applications .flex-list > .flex-item",
		".soda-admin-details--oauth2-edit > .ui.attached.form",
		".soda-admin-details--repo-list) > .ui.attached.table.segment",
		"tbody td[colspan]",
		"@media (max-width: 700px)",
	} {
		if !strings.Contains(css, selector) {
			t.Errorf("admin-details.css lost scoped contract %q", selector)
		}
	}
	for _, forbidden := range []string{"body {", ":root {", ".admin-setting-content {"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("admin-details.css contains unscoped selector %q", forbidden)
		}
	}
}
