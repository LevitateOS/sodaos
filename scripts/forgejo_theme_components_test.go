package scripts

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"testing"
)

func TestForgejoThemeToggleIsSingletonAtEachPlacement(t *testing.T) {
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{
				"home":                   "Home",
				"sign_in":                "Sign in",
				"settings.manage_themes": "Manage themes",
			}}}
		},
		"dict": forgejoTemplateDict,
		"svg": func(name string, _ ...any) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	theme := readForgejoTemplate(t, "custom", "soda", "theme_toggle.tmpl")
	tests := []struct {
		name       string
		templateID string
		definition string
		data       map[string]any
		wantClass  string
	}{
		{
			name:       "home",
			templateID: "home",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "home"}}` + readForgejoTemplate(t, "home.tmpl") + `{{end}}`,
			data:       map[string]any{"ShowRegistrationButton": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "sign in",
			templateID: "sign-in",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "user/auth/signin_inner"}}native sign in{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "sign-in"}}` + readForgejoTemplate(t, "user", "auth", "signin.tmpl") + `{{end}}`,
			data:       map[string]any{},
			wantClass:  "soda-theme-toggle",
		},
		{
			name:       "prohibited sign in",
			templateID: "prohibit-login",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "prohibit-login"}}` + readForgejoTemplate(t, "user", "auth", "prohibit_login.tmpl") + `{{end}}`,
			data:       map[string]any{"PageIsSignIn": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "guest explorer hook",
			templateID: "custom/extra_links",
			definition: `{{define "custom/soda/guest_theme"}}` + readForgejoTemplate(t, "custom", "soda", "guest_theme.tmpl") + `{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "custom/extra_links"}}` + readForgejoTemplate(t, "custom", "extra_links.tmpl") + `{{end}}`,
			data:       map[string]any{"PageIsExploreRepositories": true},
			wantClass:  "soda-guest-theme-toggle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := template.New(tt.name).Funcs(functions).Parse(tt.definition)
			if err != nil {
				t.Fatalf("parse theme placement: %v", err)
			}
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, tt.templateID, tt.data); err != nil {
				t.Fatalf("execute theme placement: %v", err)
			}
			output := rendered.String()
			if count := strings.Count(output, `id="soda-theme-toggle"`); count != 1 {
				t.Errorf("theme placement rendered %d singleton hooks:\n%s", count, output)
			}
			if !strings.Contains(output, tt.wantClass) {
				t.Errorf("theme placement does not contain class %q:\n%s", tt.wantClass, output)
			}
		})
	}
}

func TestForgejoHeaderLoadsGuestThemeScriptOnlyForToggleRoutes(t *testing.T) {
	functions := template.FuncMap{
		"dict":           forgejoTemplateDict,
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
	}
	definition := `{{define "custom/soda/theme_toggle"}}{{end}}{{define "custom/soda/guest_theme"}}` + readForgejoTemplate(t, "custom", "soda", "guest_theme.tmpl") + `{{end}}{{define "custom/header"}}` + readForgejoTemplate(t, "custom", "header.tmpl") + `{{end}}`
	parsed, err := template.New("custom-header").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse custom header: %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantScript bool
	}{
		{name: "guest home", data: map[string]any{"PageIsHome": true}, wantScript: true},
		{name: "guest sign in", data: map[string]any{"PageIsSignIn": true}, wantScript: true},
		{name: "guest account link", data: map[string]any{"LinkAccountMode": true}, wantScript: true},
		{name: "guest repositories", data: map[string]any{"PageIsExploreRepositories": true}, wantScript: true},
		{name: "guest people", data: map[string]any{"PageIsExploreUsers": true}, wantScript: true},
		{name: "guest organizations", data: map[string]any{"PageIsExploreOrganizations": true}, wantScript: true},
		{name: "guest repository", data: map[string]any{"Repository": true}, wantScript: true},
		{name: "guest organization", data: map[string]any{"Org": true}, wantScript: true},
		{name: "guest recovery under subpath", data: map[string]any{"Link": "/forge/user/forgot_password"}, wantScript: true},
		{name: "recovery on wrong subpath", data: map[string]any{"Link": "/user/forgot_password"}},
		{name: "signed recovery", data: map[string]any{"IsSigned": true, "Link": "/forge/user/reset_password"}},
		{name: "signed explorer", data: map[string]any{"IsSigned": true, "PageIsExploreRepositories": true}},
		{name: "unrelated guest page", data: map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "custom/header", tt.data); err != nil {
				t.Fatalf("execute custom header: %v", err)
			}
			output := rendered.String()
			hasScript := strings.Contains(output, `/soda/forgejo/login-theme.js?v=2`)
			if hasScript != tt.wantScript {
				t.Errorf("guest theme script presence = %t, want %t:\n%s", hasScript, tt.wantScript, output)
			}
			if count := strings.Count(output, `login-theme.js`); count > 1 {
				t.Errorf("custom header rendered guest theme script %d times:\n%s", count, output)
			}
			if !regexp.MustCompile(`/soda/forgejo/components\.css\?v=[1-9][0-9]*(?:-[a-z0-9]+)*"`).MatchString(output) {
				t.Errorf("custom header lost the shared component stylesheet:\n%s", output)
			}
		})
	}
}
