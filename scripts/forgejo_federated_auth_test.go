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

// These hashes bind the presentation-only overrides to the embedded templates in
// stock Forgejo v15.0.7. The upstream templates are GPL-3.0-or-later:
// https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/templates/user/auth
var forgejo1507FederatedTemplateHashes = map[string]string{
	"grant.tmpl":                  "25d0995c17b53e1e80e4fa48f5621e4d944ae75995d97dc64ae3139163631ffa",
	"grant_error.tmpl":            "0b5bc97171fcbb17d5845fe746740ccc7d4d5d31769c50790b3f16f64dffb775",
	"link_account.tmpl":           "b6e0a3fa0a8863728d12d4ab423058e5afd48e40f6b59152fed3be56c3ef9382",
	"signin_openid.tmpl":          "87e3143f01da17f6d94df704af1b0955a97be44f2705ce4d9448a05f59492234",
	"signup_openid_connect.tmpl":  "64aa53ed32f1143f59504282951655ef21690730b7c86fbe56a11d91803b9b3c",
	"signup_openid_register.tmpl": "73acde87ce6e08565278aecb064ed78fbdd5d87e809610458ed2b03df6149488",
}

var forgejoFederatedPresentationEdits = map[string][][2]string{
	"grant.tmpl": {
		{`{{/* Adapted presentation wrapper from stock Forgejo 15.0.7; GPL-3.0-or-later. */}}
`, ``},
		{
			`class="page-content ui one column stackable center aligned page grid oauth2-authorize-application-box soda-page soda-federated-auth soda-federated-auth--grant" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
			`class="page-content ui one column stackable center aligned page grid oauth2-authorize-application-box"`,
		},
	},
	"grant_error.tmpl": {
		{`{{/* Adapted presentation wrapper from stock Forgejo 15.0.7; GPL-3.0-or-later. */}}
`, ``},
		{
			`class="page-content ui one column stackable center aligned page grid oauth2-authorize-application-box soda-page soda-federated-auth soda-federated-auth--error {{if .IsRepo}}repository{{end}}" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
			`class="page-content ui one column stackable center aligned page grid oauth2-authorize-application-box {{if .IsRepo}}repository{{end}}"`,
		},
	},
	"link_account.tmpl": {
		{`{{/* Adapted presentation wrapper from stock Forgejo 15.0.7; GPL-3.0-or-later. */}}
`, ``},
		{
			`class="page-content user link-account soda-page soda-federated-auth soda-federated-auth--link-account" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
			`class="page-content user link-account"`,
		},
		{
			`	{{if and (not .IsSigned) (or .PageIsSignIn .LinkAccountMode)}}{{template "custom/soda/theme_toggle" (dict "Class" "soda-theme-toggle")}}{{end}}
`,
			``,
		},
	},
	"signin_openid.tmpl": {
		{`{{/* Adapted presentation wrapper from stock Forgejo 15.0.7; GPL-3.0-or-later. */}}
`, ``},
		{
			`class="page-content user signin openid soda-page soda-federated-auth soda-federated-auth--openid-signin" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
			`class="page-content user signin openid"`,
		},
		{
			`	{{if and (not .IsSigned) (or .PageIsSignIn .LinkAccountMode)}}{{template "custom/soda/theme_toggle" (dict "Class" "soda-theme-toggle")}}{{end}}
`,
			``,
		},
	},
	"signup_openid_connect.tmpl": {
		{`{{/* Adapted presentation wrapper from stock Forgejo 15.0.7; GPL-3.0-or-later. */}}
`, ``},
		{
			`class="page-content user signup soda-page soda-federated-auth soda-federated-auth--openid-connect" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
			`class="page-content user signup"`,
		},
		{
			`	{{if and (not .IsSigned) (or .PageIsSignIn .LinkAccountMode)}}{{template "custom/soda/theme_toggle" (dict "Class" "soda-theme-toggle")}}{{end}}
`,
			``,
		},
	},
	"signup_openid_register.tmpl": {
		{`{{/* Adapted presentation wrapper from stock Forgejo 15.0.7; GPL-3.0-or-later. */}}
`, ``},
		{
			`class="page-content user signup soda-page soda-federated-auth soda-federated-auth--openid-register" data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`,
			`class="page-content user signup"`,
		},
		{
			`	{{if and (not .IsSigned) (or .PageIsSignIn .LinkAccountMode)}}{{template "custom/soda/theme_toggle" (dict "Class" "soda-theme-toggle")}}{{end}}
`,
			``,
		},
	},
}

func TestForgejoFederatedAuthOverridesMatchStock1507ApartFromPresentation(t *testing.T) {
	for name, wantHash := range forgejo1507FederatedTemplateHashes {
		t.Run(name, func(t *testing.T) {
			contents := readForgejoFederatedTemplate(t, name)
			for _, edit := range forgejoFederatedPresentationEdits[name] {
				if count := strings.Count(contents, edit[0]); count != 1 {
					t.Fatalf("presentation marker occurs %d times, want 1", count)
				}
				contents = strings.Replace(contents, edit[0], edit[1], 1)
			}
			gotHash := fmt.Sprintf("%x", sha256.Sum256([]byte(contents)))
			if gotHash != wantHash {
				t.Errorf("normalized override hash = %s, stock 15.0.7 hash = %s", gotHash, wantHash)
			}
		})
	}
}

func TestForgejoFederatedAuthPreservesNativeSecurityAndDelegation(t *testing.T) {
	wants := map[string][]string{
		"grant.tmpl": {
			`action="{{AppSubUrl}}/login/oauth/grant"`,
			`name="client_id" value="{{.Application.ClientID}}"`,
			`name="state" value="{{.State}}"`,
			`name="scope" value="{{.Scope}}"`,
			`name="nonce" value="{{.Nonce}}"`,
			`name="redirect_uri" value="{{.RedirectURI}}"`,
			`id="authorize-app" name="granted" value="true" class="ui red inline button"`,
			`name="granted" value="false" class="ui basic primary inline button"`,
		},
		"grant_error.tmpl":            {`{{if .IsRepo}}{{template "repo/header" .}}{{end}}`},
		"link_account.tmpl":           {`{{template "user/auth/signup_inner" .}}`, `{{template "user/auth/signin_inner" .}}`},
		"signin_openid.tmpl":          {`action="{{.Link}}" method="post"`, `{{template "user/auth/webauthn_error" .}}`},
		"signup_openid_connect.tmpl":  {`{{template "user/auth/signup_openid_navbar" .}}`, `action="{{.Link}}" method="post"`},
		"signup_openid_register.tmpl": {`{{template "user/auth/signup_openid_navbar" .}}`, `{{template "user/auth/captcha" .}}`, `action="{{.Link}}" method="post"`},
	}
	for name, markers := range wants {
		contents := readForgejoFederatedTemplate(t, name)
		for _, marker := range markers {
			if !strings.Contains(contents, marker) {
				t.Errorf("%s lost native marker %q", name, marker)
			}
		}
	}
}

func TestForgejoFederatedAuthOverridesParseWithNativeSeams(t *testing.T) {
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{}}}
		},
		"dict": forgejoTemplateDict,
		"svg":  func(name string, _ ...any) template.HTML { return template.HTML(name) },
	}
	nativeSeams := `
		{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "base/alert"}}{{end}}
		{{define "repo/header"}}{{end}}{{define "custom/soda/theme_toggle"}}{{end}}
		{{define "user/auth/signup_inner"}}{{end}}{{define "user/auth/signin_inner"}}{{end}}
		{{define "user/auth/webauthn_error"}}{{end}}
		{{define "user/auth/signup_openid_navbar"}}{{end}}{{define "user/auth/captcha"}}{{end}}`
	for name := range forgejo1507FederatedTemplateHashes {
		t.Run(name, func(t *testing.T) {
			definition := nativeSeams + `{{define "page"}}` + readForgejoFederatedTemplate(t, name) + `{{end}}`
			if _, err := template.New(name).Funcs(functions).Parse(definition); err != nil {
				t.Fatalf("parse federated-auth override with native seams: %v", err)
			}
		})
	}
}

func TestForgejoFederatedAuthGuestToggleFollowsNativeRouteFlags(t *testing.T) {
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{}}}
		},
		"dict": forgejoTemplateDict,
		"svg":  func(name string, _ ...any) template.HTML { return template.HTML(name) },
	}
	nativeSeams := `
		{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "base/alert"}}{{end}}
		{{define "custom/soda/theme_toggle"}}guest-toggle{{end}}
		{{define "user/auth/signup_inner"}}{{end}}{{define "user/auth/signin_inner"}}{{end}}
		{{define "user/auth/webauthn_error"}}{{end}}{{define "user/auth/signup_openid_navbar"}}{{end}}
		{{define "user/auth/captcha"}}{{end}}`
	for _, test := range []struct {
		name      string
		file      string
		data      map[string]any
		wantCount int
	}{
		{name: "account link guest", file: "link_account.tmpl", data: map[string]any{"LinkAccountMode": true}, wantCount: 1},
		{name: "OpenID sign-in guest", file: "signin_openid.tmpl", data: map[string]any{"PageIsSignIn": true}, wantCount: 1},
		{name: "OpenID connect guest", file: "signup_openid_connect.tmpl", data: map[string]any{"PageIsSignIn": true}, wantCount: 1},
		{name: "OpenID register guest", file: "signup_openid_register.tmpl", data: map[string]any{"PageIsSignIn": true}, wantCount: 1},
		{name: "signed route", file: "signin_openid.tmpl", data: map[string]any{"IsSigned": true, "PageIsSignIn": true}},
		{name: "unflagged guest", file: "signin_openid.tmpl", data: map[string]any{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			definition := nativeSeams + `{{define "page"}}` + readForgejoFederatedTemplate(t, test.file) + `{{end}}`
			parsed, err := template.New(test.name).Funcs(functions).Parse(definition)
			if err != nil {
				t.Fatalf("parse federated-auth route: %v", err)
			}
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "page", test.data); err != nil {
				t.Fatalf("render federated-auth route: %v", err)
			}
			if count := strings.Count(output.String(), "guest-toggle"); count != test.wantCount {
				t.Errorf("guest toggle count = %d, want %d:\n%s", count, test.wantCount, output.String())
			}
		})
	}
}

func TestForgejoFederatedAuthNativeAccountLinkBranches(t *testing.T) {
	functions := template.FuncMap{
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"auth.oauth_signup_tab": "Register", "auth.oauth_signin_tab": "Sign in",
			}}}
		},
		"dict": forgejoTemplateDict,
	}
	definition := `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}
		{{define "custom/soda/theme_toggle"}}{{end}}
		{{define "user/auth/signup_inner"}}native-signup{{end}}
		{{define "user/auth/signin_inner"}}native-signin{{end}}
		{{define "page"}}` + readForgejoFederatedTemplate(t, "link_account.tmpl") + `{{end}}`
	parsed, err := template.New("link-account").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse link-account override: %v", err)
	}

	for _, test := range []struct {
		name   string
		data   map[string]any
		want   []string
		forbid []string
	}{
		{name: "new account", data: map[string]any{"user_exists": false}, want: []string{`class="item active"`, `data-tab="auth-link-signup-tab"`, "native-signup", "native-signin"}},
		{name: "existing account", data: map[string]any{"user_exists": true}, want: []string{`class="item active"`, `data-tab="auth-link-signin-tab"`, "native-signup", "native-signin"}},
		{name: "internal registration only", data: map[string]any{"AllowOnlyInternalRegistration": true}, want: []string{`data-tab="auth-link-signin-tab"`, "native-signup", "native-signin"}, forbid: []string{"Register"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "page", test.data); err != nil {
				t.Fatalf("render link-account override: %v", err)
			}
			for _, want := range test.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("rendered branch lacks %q:\n%s", want, output.String())
				}
			}
			for _, forbidden := range test.forbid {
				if strings.Contains(output.String(), forbidden) {
					t.Errorf("rendered branch contains %q:\n%s", forbidden, output.String())
				}
			}
		})
	}
}

func TestForgejoFederatedAuthGrantErrorKeepsRepositoryHeaderBranch(t *testing.T) {
	functions := template.FuncMap{"ctx": func() forgejoTemplateContext {
		return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{}}}
	}}
	definition := `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}
		{{define "repo/header"}}native-repository-header{{end}}
		{{define "page"}}` + readForgejoFederatedTemplate(t, "grant_error.tmpl") + `{{end}}`
	parsed, err := template.New("grant-error").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse grant-error override: %v", err)
	}
	for _, test := range []struct {
		name      string
		isRepo    bool
		wantCount int
	}{
		{name: "repository context", isRepo: true, wantCount: 1},
		{name: "application context", isRepo: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			data := map[string]any{"IsRepo": test.isRepo, "Error": map[string]any{"ErrorDescription": "denied"}}
			if err := parsed.ExecuteTemplate(&output, "page", data); err != nil {
				t.Fatalf("render grant-error override: %v", err)
			}
			if count := strings.Count(output.String(), "native-repository-header"); count != test.wantCount {
				t.Errorf("repository header count = %d, want %d:\n%s", count, test.wantCount, output.String())
			}
		})
	}
}

func TestForgejoFederatedAuthCSSIsScopedAndAttributed(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "federated-auth.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, want := range []string{"Forgejo 15.0.7", "GPL-3.0-or-later", ".soda-federated-auth", ".secondary-nav", ".ui.red.button"} {
		if !strings.Contains(css, want) {
			t.Errorf("federated-auth CSS lacks %q", want)
		}
	}
}

func readForgejoFederatedTemplate(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "appliance", "forgejo", "templates", "user", "auth", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
