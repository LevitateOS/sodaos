package scripts

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"testing"
)

type forgejoThemeStrings struct{}

func (forgejoThemeStrings) HasPrefix(s, prefix string) bool { return strings.HasPrefix(s, prefix) }

func TestForgejoThemeToggleIsSingletonAtEachPlacement(t *testing.T) {
	selectedTheme := "soda-auto"
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ThemeName":      func(any) string { return selectedTheme },
		"StringUtils":    func() forgejoThemeStrings { return forgejoThemeStrings{} },
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
	navbar := `{{define "custom/soda/guest_theme"}}` + readForgejoTemplate(t, "custom", "soda", "guest_theme.tmpl") + `{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "custom/extra_links"}}` + readForgejoTemplate(t, "custom", "extra_links.tmpl") + `{{end}}`
	tests := []struct {
		name       string
		templateID string
		definition string
		data       map[string]any
		wantClass  string
		theme      string
		wantCount  int
	}{
		{
			name:       "home",
			wantCount:  1,
			templateID: "home",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "home"}}` + readForgejoTemplate(t, "home.tmpl") + `{{end}}`,
			data:       map[string]any{"ShowRegistrationButton": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "sign in",
			wantCount:  1,
			templateID: "sign-in",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "user/auth/signin_inner"}}native sign in{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "sign-in"}}` + readForgejoTemplate(t, "user", "auth", "signin.tmpl") + `{{end}}`,
			data:       map[string]any{},
			wantClass:  "soda-theme-toggle",
		},
		{
			name:       "prohibited sign in",
			wantCount:  1,
			templateID: "prohibit-login",
			definition: `{{define "base/head"}}{{end}}{{define "base/footer"}}{{end}}{{define "custom/soda/theme_toggle"}}` + theme + `{{end}}{{define "prohibit-login"}}` + readForgejoTemplate(t, "user", "auth", "prohibit_login.tmpl") + `{{end}}`,
			data:       map[string]any{"PageIsSignIn": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "guest explorer hook",
			wantCount:  1,
			templateID: "custom/extra_links",
			definition: navbar,
			data:       map[string]any{"PageIsExploreRepositories": true},
			wantClass:  "soda-guest-theme-toggle",
		},
		{
			name:       "guest native page without presentation markers",
			templateID: "custom/extra_links",
			definition: navbar,
			data:       map[string]any{},
			wantClass:  "soda-guest-theme-toggle",
			wantCount:  1,
		},
		{
			name:       "guest native theme",
			templateID: "custom/extra_links",
			definition: navbar,
			data:       map[string]any{"PageIsExploreRepositories": true},
			theme:      "forgejo-auto",
		},
		{
			name:       "signed native page without presentation markers",
			templateID: "custom/extra_links",
			definition: navbar,
			data:       map[string]any{"IsSigned": true},
			wantClass:  "soda-account-appearance",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedTheme = tt.theme
			if selectedTheme == "" {
				selectedTheme = "soda-auto"
			}
			parsed, err := template.New(tt.name).Funcs(functions).Parse(tt.definition)
			if err != nil {
				t.Fatalf("parse theme placement: %v", err)
			}
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, tt.templateID, tt.data); err != nil {
				t.Fatalf("execute theme placement: %v", err)
			}
			output := rendered.String()
			if count := strings.Count(output, `id="soda-theme-toggle"`); count != tt.wantCount {
				t.Errorf("theme placement rendered %d singleton hooks, want %d:\n%s", count, tt.wantCount, output)
			}
			if tt.wantClass != "" && !strings.Contains(output, tt.wantClass) {
				t.Errorf("theme placement does not contain class %q:\n%s", tt.wantClass, output)
			}
			if strings.Contains(output, "soda-account-appearance") != (tt.data["IsSigned"] == true) {
				t.Errorf("account appearance control does not match sign-in state:\n%s", output)
			}
		})
	}
}

func TestForgejoHeaderLoadsGuestThemeScriptForSodaGuests(t *testing.T) {
	selectedTheme := "soda-auto"
	functions := template.FuncMap{
		"dict":           forgejoTemplateDict,
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ThemeName":      func(any) string { return selectedTheme },
		"StringUtils":    func() forgejoThemeStrings { return forgejoThemeStrings{} },
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
		theme      string
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
		{name: "guest recovery independent of route", data: map[string]any{"Link": "/user/forgot_password"}, wantScript: true},
		{name: "signed recovery", data: map[string]any{"IsSigned": true, "Link": "/forge/user/reset_password"}},
		{name: "signed explorer", data: map[string]any{"IsSigned": true, "PageIsExploreRepositories": true}},
		{name: "unrelated guest page", data: map[string]any{}, wantScript: true},
		{name: "guest extension", data: map[string]any{"PageIsExtension": true}, wantScript: true},
		{name: "guest light theme", data: map[string]any{}, theme: "soda-light", wantScript: true},
		{name: "guest dark theme", data: map[string]any{}, theme: "soda-dark", wantScript: true},
		{name: "guest native theme", data: map[string]any{"PageIsExploreRepositories": true}, theme: "forgejo-auto"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedTheme = tt.theme
			if selectedTheme == "" {
				selectedTheme = "soda-auto"
			}
			var rendered bytes.Buffer
			if err := parsed.ExecuteTemplate(&rendered, "custom/header", tt.data); err != nil {
				t.Fatalf("execute custom header: %v", err)
			}
			output := rendered.String()
			hasScript := strings.Contains(output, `/soda/forgejo/login-theme.js?`)
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
