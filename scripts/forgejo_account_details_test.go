package scripts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type forgejoAccountDetailFixture struct {
	path string
	slug string
	sha  string
}

var forgejoAccountDetailFixtures = []forgejoAccountDetailFixture{
	{path: "profile.tmpl", slug: "profile", sha: "877b9dc97e0c7ee3769bf00765959009e91f30d064777387d4ac9433c5713e75"},
	{path: "account.tmpl", slug: "account", sha: "4e418544763f6a3d79fce08aa84ddca216bf390060fcb2d60b54d63a39c392ca"},
	{path: "appearance.tmpl", slug: "appearance", sha: "f59f51eafb0a9e9884765a95b182906dc3884cb53cab79918dde50e4c6f9014f"},
	{path: "applications.tmpl", slug: "applications", sha: "113a38dd2baa98b0bcf9bb3ecb252b486fc5c6441807a7364687bda399d3930d"},
	{path: "access_token_edit.tmpl", slug: "access-token", sha: "0168c31a3d728e74d4c9e3756c9c863457aeaca55ef17c7433999d89a3703b42"},
	{path: "applications_oauth2_edit.tmpl", slug: "oauth2-edit", sha: "1119c308d9540bb725559d41b9e747daad6e619e67ddbb86f274571ab9be2886"},
	{path: "keys.tmpl", slug: "keys", sha: "e587ae7db2beecccd7af83b42a64b85b417c8157cd70726c38b8093e23638740"},
	{path: "security/security.tmpl", slug: "security", sha: "a5fbae07bc6d439c451bcf2eaaa333d4dcfc358025918158c1f7bbc60fb7db8c"},
	{path: "security/twofa_enroll.tmpl", slug: "twofa-enroll", sha: "fdd8187cd87d5c5ed07f78a49a7b5402c69b6f5c08fac5aa75f3ce307f94ea70"},
}

func TestForgejoAccountDetailOverridesMatchStock1507(t *testing.T) {
	for _, fixture := range forgejoAccountDetailFixtures {
		t.Run(fixture.slug, func(t *testing.T) {
			contents := readForgejoAccountDetailTemplate(t, fixture.path)
			notice := fmt.Sprintf("{{/* Adapted from Forgejo 15.0.7 templates/user/settings/%s (GPL-3.0-or-later); upstream SHA-256 %s. */}}\n", fixture.path, fixture.sha)
			if !strings.HasPrefix(contents, notice) {
				t.Fatalf("%s lost exact Forgejo version, license, or pristine-source attribution", fixture.path)
			}
			restored := strings.TrimPrefix(contents, notice)
			edits := [][2]string{
				{
					fmt.Sprintf(`"pageClass" "user settings soda-account-details-page soda-account-details-page--%s `, fixture.slug),
					`"pageClass" "user settings `,
				},
				{
					fmt.Sprintf(`<div class="user-setting-content soda-account-details soda-account-details--%s">`, fixture.slug),
					`<div class="user-setting-content">`,
				},
			}
			if fixture.path == "appearance.tmpl" {
				edits = append(edits, [2]string{
					`<form class="ui form soda-form" action="{{.Link}}/theme" method="post">`,
					`<form class="ui form" action="{{.Link}}/theme" method="post">`,
				})
			}
			for _, edit := range edits {
				if count := strings.Count(restored, edit[0]); count != 1 {
					t.Fatalf("%s has %d occurrences of presentation edit %q, want 1", fixture.path, count, edit[0])
				}
				restored = strings.Replace(restored, edit[0], edit[1], 1)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(restored))); got != fixture.sha {
				t.Errorf("%s differs from pristine Forgejo 15.0.7 outside attributed page/content classes: got %s, want %s", fixture.path, got, fixture.sha)
			}
		})
	}
}

func TestForgejoOAuthApplicationListKeepsNativeFormsAndActions(t *testing.T) {
	const upstreamHash = "fef3276344cd83d2df9a074ff37eafb320b2835de59d951add2601672d53c761"
	const notice = "{{/* Adapted from Forgejo 15.0.7 templates/user/settings/applications_oauth2_list.tmpl (GPL-3.0-or-later); upstream SHA-256 " + upstreamHash + ". */}}\n"
	contents := readForgejoAccountDetailTemplate(t, "applications_oauth2_list.tmpl")
	if !strings.HasPrefix(contents, notice) {
		t.Fatal("OAuth application list lost exact Forgejo version, license, or pristine-source attribution")
	}
	const branded = `<form class="ui form soda-form ignore-dirty" action="{{.Link}}/oauth2" method="post">`
	const stock = `<form class="ui form ignore-dirty" action="{{.Link}}/oauth2" method="post">`
	restored := strings.TrimPrefix(contents, notice)
	if count := strings.Count(restored, branded); count != 1 {
		t.Fatalf("OAuth application list has %d principal form adapters, want 1", count)
	}
	restored = strings.Replace(restored, branded, stock, 1)
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(restored))); got != upstreamHash {
		t.Fatalf("OAuth application list differs from pristine Forgejo 15.0.7 beyond its principal form class: got %s, want %s", got, upstreamHash)
	}
	for _, marker := range []string{
		`{{range .Applications}}`, `{{if $isBuiltin}}`, `data-modal-id="remove-gitea-oauth2-application"`,
		`data-url="{{$.Link}}/oauth2/{{.ID}}/delete"`, `{{template "base/modal_actions_confirm" .}}`,
		`name="application_name"`, `name="redirect_uris"`, `name="confidential_client"`,
	} {
		if !strings.Contains(contents, marker) {
			t.Errorf("OAuth application list lost native marker %q", marker)
		}
	}
}

func TestForgejoAccountDetailsPreserveNativeSecurityAndActions(t *testing.T) {
	wants := map[string][]string{
		"profile.tmpl": {
			`action="{{.Link}}" method="post"`, `{{if or .UserRenameDisabled .IsReverseProxy}}disabled{{end}}`,
			`name="visibility"`, `name="keep_email_private"`, `action="{{.Link}}/avatar" method="post" enctype="multipart/form-data"`,
			`data-url="{{.Link}}/avatar/delete"`,
		},
		"account.tmpl": {
			`UserDisabledFeatures.Contains "manage_password"`, `{{if .SignedUser.IsPasswordSet}}`,
			`action="{{AppSubUrl}}/user/settings/account/email"`, `{{if $.ActivationsPending}}`, `{{if not .CanAddEmails}}disabled{{end}}`,
			`UserDisabledFeatures.Contains "deletion"`, `id="delete-form"`, `id="delete-account"`, `id="delete-email"`,
		},
		"appearance.tmpl": {
			`action="{{.Link}}/theme" method="post"`, `name="theme"`, `action="{{.Link}}/language" method="post"`,
			`action="{{.Link}}/hints" method="post"`, `action="{{.Link}}/hidden_comments" method="post"`, `IsCommentTypeGroupChecked`,
		},
		"applications.tmpl": {
			`data-modal-id="regenerate-token"`, `data-url="{{$.Link}}/tokens/regenerate"`, `data-modal-id="delete-token"`,
			`{{if .EnableOAuth2}}`, `{{template "user/settings/grants_oauth2" .}}`, `{{template "user/settings/applications_oauth2" .}}`,
			`{{template "base/modal_actions_confirm" (dict "ModalButtonColors" "primary")}}`,
		},
		"access_token_edit.tmpl": {
			`id="scoped-access-form"`, `action="{{.Link}}" method="post"`, `id="resource-all"`, `id="resource-public-only"`, `id="resource-repo-specific"`,
			`id="repo-selector-wrapper" role="group"`, `type="search" name="repo_search"`, `formnovalidate="true" formmethod="get"`,
			`type="hidden" name="selected_repo"`, `SliceUtils.Contains $.scope`, `id="scoped-access-submit"`,
		},
		"applications_oauth2_edit.tmpl": {
			`{{template "user/settings/applications_oauth2_edit_form" .}}`,
		},
		"keys.tmpl": {
			`UserDisabledFeatures.Contains "manage_ssh_keys"`, `{{template "user/settings/keys_ssh" .}}`,
			`{{template "user/settings/keys_principal" .}}`, `UserDisabledFeatures.Contains "manage_gpg_keys"`, `{{template "user/settings/keys_gpg" .}}`,
		},
		"security/security.tmpl": {
			`{{if .MustEnableTwoFactor}}`, `{{template "user/settings/security/twofa" .}}`, `{{template "user/settings/security/webauthn" .}}`,
			`{{if not .MustEnableTwoFactor}}`, `{{template "user/settings/security/accountlinks" .}}`, `{{if .EnableOpenIDSignIn}}`,
			`{{template "user/settings/security/openid" .}}`,
		},
		"security/twofa_enroll.tmpl": {
			`{{if .ReenrollTwofa}}`, `src="{{.QrUri}}" alt="{{.TwofaSecret}}"`, `{{ctx.Locale.Tr "settings.or_enter_secret" .TwofaSecret}}`,
			`action="{{.Link}}" method="post"`, `class="inline required field {{if .Err_Passcode}}error{{end}}"`, `id="passcode" name="passcode" autofocus required`,
		},
	}

	for name, markers := range wants {
		contents := readForgejoAccountDetailTemplate(t, name)
		for _, marker := range markers {
			if !strings.Contains(contents, marker) {
				t.Errorf("%s lost native marker %q", name, marker)
			}
		}
		calls := templateCalls(contents)
		if !calls["user/settings/layout_head"] || !calls["user/settings/layout_footer"] {
			t.Errorf("%s no longer delegates its native settings layout", name)
		}
	}
}

type forgejoAccountDisabledFeatures map[string]bool

func (features forgejoAccountDisabledFeatures) Contains(name string) bool {
	return features[name]
}

func TestForgejoAccountKeysKeepNativeCapabilityGates(t *testing.T) {
	definition := `
		{{define "user/settings/layout_head"}}{{end}}
		{{define "user/settings/layout_footer"}}{{end}}
		{{define "user/settings/keys_ssh"}}native-ssh{{end}}
		{{define "user/settings/keys_principal"}}native-principal{{end}}
		{{define "user/settings/keys_gpg"}}native-gpg{{end}}
		{{define "page"}}` + readForgejoAccountDetailTemplate(t, "keys.tmpl") + `{{end}}`
	parsed, err := template.New("keys").Funcs(template.FuncMap{"dict": forgejoTemplateDict}).Parse(definition)
	if err != nil {
		t.Fatalf("parse keys settings: %v", err)
	}

	for _, test := range []struct {
		name      string
		disabled  forgejoAccountDisabledFeatures
		want      []string
		forbidden []string
	}{
		{name: "all key types enabled", disabled: forgejoAccountDisabledFeatures{}, want: []string{"native-ssh", "native-principal", "native-gpg"}},
		{name: "managed key types disabled", disabled: forgejoAccountDisabledFeatures{"manage_ssh_keys": true, "manage_gpg_keys": true}, want: []string{"native-principal"}, forbidden: []string{"native-ssh", "native-gpg"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "page", map[string]any{"UserDisabledFeatures": test.disabled}); err != nil {
				t.Fatalf("render keys settings: %v", err)
			}
			for _, want := range test.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("keys branch lacks %q: %s", want, output.String())
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(output.String(), forbidden) {
					t.Errorf("keys branch unexpectedly contains %q: %s", forbidden, output.String())
				}
			}
		})
	}
}

func TestForgejoAccountSecurityKeepsRequiredTwoFactorBoundary(t *testing.T) {
	functions := template.FuncMap{
		"dict": forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{"settings.must_enable_2fa": "Two-factor required"}}}
		},
	}
	definition := `
		{{define "user/settings/layout_head"}}{{end}}
		{{define "user/settings/layout_footer"}}{{end}}
		{{define "user/settings/security/twofa"}}native-twofa{{end}}
		{{define "user/settings/security/webauthn"}}native-webauthn{{end}}
		{{define "user/settings/security/accountlinks"}}native-accountlinks{{end}}
		{{define "user/settings/security/openid"}}native-openid{{end}}
		{{define "page"}}` + readForgejoAccountDetailTemplate(t, "security/security.tmpl") + `{{end}}`
	parsed, err := template.New("security").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse security settings: %v", err)
	}

	for _, test := range []struct {
		name      string
		data      map[string]any
		want      []string
		forbidden []string
	}{
		{name: "two-factor required", data: map[string]any{"MustEnableTwoFactor": true, "EnableOpenIDSignIn": true}, want: []string{"Two-factor required", "native-twofa", "native-webauthn"}, forbidden: []string{"native-accountlinks", "native-openid"}},
		{name: "normal security", data: map[string]any{"EnableOpenIDSignIn": true}, want: []string{"native-twofa", "native-webauthn", "native-accountlinks", "native-openid"}, forbidden: []string{"Two-factor required"}},
		{name: "openid disabled", data: map[string]any{}, want: []string{"native-twofa", "native-webauthn", "native-accountlinks"}, forbidden: []string{"native-openid"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "page", test.data); err != nil {
				t.Fatalf("render security settings: %v", err)
			}
			for _, want := range test.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("security branch lacks %q: %s", want, output.String())
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(output.String(), forbidden) {
					t.Errorf("security branch unexpectedly contains %q: %s", forbidden, output.String())
				}
			}
		})
	}
}

func TestForgejoAccountDetailsCSSIsScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "account-details.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, marker := range []string{"Forgejo 15.0.7", "GPL-3.0-or-later", ".soda-account-details--profile", ".soda-account-details--account", ".soda-account-details--applications", ".soda-account-details--access-token", ".soda-account-details--oauth2-edit", ".soda-account-details--keys", ".soda-account-details--security", ".soda-account-details--twofa-enroll"} {
		if !strings.Contains(css, marker) {
			t.Errorf("account detail stylesheet lost %q", marker)
		}
	}
	for _, forbidden := range []string{"display: none", "overflow: hidden", ".ui.red", ".danger.button", "button {", "input:disabled"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("account detail stylesheet contains behavior-sensitive rule %q", forbidden)
		}
	}
}

func readForgejoAccountDetailTemplate(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "appliance", "forgejo", "templates", "user", "settings", filepath.FromSlash(name))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
