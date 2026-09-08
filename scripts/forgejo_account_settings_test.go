package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// The native navigation restrictions must remain true even with no enhancement.
func TestForgejoPersonalSettingsNavigationGates(t *testing.T) {
	source := readForgejoTemplate(t, "user/settings/navbar.tmpl")
	for _, test := range []struct {
		name         string
		data         map[string]any
		disabled     bool
		want, absent []string
	}{
		{name: "ordinary", data: map[string]any{}, disabled: true, want: []string{"/user/settings/account", "/user/settings/blocked_users"}, absent: []string{"/actions/runners", "/packages", "/storage_overview", "/hooks"}},
		{name: "all capabilities", data: map[string]any{"EnableActions": true, "EnablePackages": true, "EnableQuota": true, "PageIsSettingsKeys": true}, want: []string{"/actions/runners", "/actions/secrets", "/actions/variables", "/packages", "/storage_overview", "/hooks", `aria-current="page"`}},
		{name: "mandatory enrollment", data: map[string]any{"HideNavbarLinks": true, "EnableActions": true, "EnablePackages": true, "EnableQuota": true}, absent: []string{"<nav", "/user/settings"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := template.New("nav").Funcs(template.FuncMap{
				"ctx":       func() forgejoTemplateContext { return forgejoTemplateContext{Locale: forgejoTemplateLocale{}} },
				"AppSubUrl": func() string { return "" }, "svg": func(string, ...any) string { return "" }, "DisableWebhooks": func() bool { return test.disabled },
			}).Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err = parsed.Execute(&out, test.data); err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(out.String(), want) {
					t.Errorf("missing %s", want)
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(out.String(), absent) {
					t.Errorf("unexpected %s", absent)
				}
			}
		})
	}
}

func TestForgejoAccountSettingsLayoutComposesNativeSeams(t *testing.T) {
	source := readForgejoTemplate(t, "user/settings/layout_head.tmpl")
	for _, want := range []string{`template "base/head" .ctxData`, `template "base/alert" .ctxData`, `template "user/settings/navbar" .ctxData`, ".ctxData.SignedUser", "soda-settings-bar", "soda-settings-title", `{{if not .ctxData.HideNavbarLinks}}href=`} {
		if !strings.Contains(source, want) {
			t.Errorf("missing native seam %s", want)
		}
	}
	for _, old := range []string{"page_intro", "artwork", "flex-container-nav", "settings-sidebar"} {
		if strings.Contains(source, old) {
			t.Errorf("obsolete shell %s", old)
		}
	}
}

type forgejoCleanupFixtureType string

func (t forgejoCleanupFixtureType) Name() string { return string(t) }

func TestForgejoPersonalAdaptersKeepNativeRootContext(t *testing.T) {
	funcs := template.FuncMap{"dict": forgejoTemplateDict, "ctx": func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} }}
	data := map[string]any{"Link": "/native/action", "RunnersListLink": "/native/runners", "Runner": map[string]any{"Name": "runner", "Description": "details"}, "AvailableTypes": []forgejoCleanupFixtureType{"alpine"}, "CleanupRule": map[string]any{"ID": 7, "Type": forgejoCleanupFixtureType("alpine"), "KeepCount": 5, "RemoveDays": 30}}
	for _, path := range []string{"shared/actions/runner_create.tmpl", "package/shared/cleanup_rules/edit.tmpl"} {
		for _, personal := range []bool{false, true} {
			parsed, err := template.New("root").Funcs(funcs).Parse(readForgejoTemplate(t, path))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			var input any = data
			if personal {
				input = map[string]any{"PersonalSettings": true, "ctxData": data}
			}
			if err = parsed.Execute(&out, input); err != nil {
				t.Fatal(err)
			}
			html := out.String()
			if !strings.Contains(html, `action="/native/action"`) {
				t.Fatal("lost native action")
			}
			if strings.Contains(path, "cleanup") && !strings.Contains(html, `selected="selected" value="alpine"`) {
				t.Fatal("lost native selection through root context")
			}
			if strings.Contains(path, "runner") && !strings.Contains(html, `href="/native/runners"`) {
				t.Fatal("lost native root cancel link")
			}
			if strings.Contains(html, "<h4") == personal {
				t.Fatal("child title suppression leaked across caller boundary")
			}
		}
	}
}

type forgejoSettingsFixtureLocale struct{}

func (forgejoSettingsFixtureLocale) Tr(key string, args ...any) string { return key }

func TestForgejoPersonalProviderSectionAbsentWithoutProviders(t *testing.T) {
	parsed, err := template.New("providers").Funcs(template.FuncMap{"ctx": func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} }, "AppSubUrl": func() string { return "" }, "svg": func(...any) string { return "" }}).Parse(`{{define "base/modal_actions_confirm"}}{{end}}` + readForgejoTemplate(t, "user/settings/security/accountlinks.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = parsed.Execute(&out, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("disabled providers leaked structural markup: %s", out.String())
	}
}

func TestForgejoOAuthHeadingScopedByCallerNotNativeSettingsFlag(t *testing.T) {
	source := readForgejoTemplate(t, "user/settings/applications_oauth2.tmpl")
	parsed, err := template.New("oauth").Funcs(template.FuncMap{"dict": forgejoTemplateDict, "ctx": func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} }}).Parse(`{{define "user/settings/applications_oauth2_list"}}inventory{{end}}` + source)
	if err != nil {
		t.Fatal(err)
	}
	for _, personal := range []bool{false, true} {
		data := map[string]any{"PageIsSettingsApplications": true}
		var input any = data
		if personal {
			input = map[string]any{"PersonalSettings": true, "ctxData": data}
		}
		var out bytes.Buffer
		if err = parsed.Execute(&out, input); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), `<h4 class="ui top attached header">`) == personal {
			t.Fatal("organization header lost through a shared native flag")
		}
		if !strings.Contains(out.String(), "inventory") {
			t.Fatal("OAuth inventory lost")
		}
	}
}

type forgejoSettingsFixtureStrings struct{}

func (forgejoSettingsFixtureStrings) Join(values []string, sep string) string {
	return strings.Join(values, sep)
}
func TestForgejoOAuthEditorPreservesOtherCallers(t *testing.T) {
	parsed, err := template.New("oauth-edit").Funcs(template.FuncMap{"dict": forgejoTemplateDict, "ctx": func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} }, "StringUtils": func() any { return forgejoSettingsFixtureStrings{} }}).Parse(readForgejoTemplate(t, "user/settings/applications_oauth2_edit_form.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, personal := range []bool{false, true} {
		data := map[string]any{"PageIsSettingsApplications": true, "FormActionPath": "/native/oauth", "App": map[string]any{"Name": "Example", "ClientID": "fixture-id", "RedirectURIs": []string{"https://example.test/callback"}, "ConfidentialClient": true}}
		var input any = data
		if personal {
			input = map[string]any{"PersonalSettings": true, "ctxData": data}
		}
		var out bytes.Buffer
		if err = parsed.Execute(&out, input); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`action="/native/oauth"`, `action="/native/oauth/regenerate_secret"`, `https://example.test/callback`, `name="confidential_client" checked`} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("missing native OAuth contract %s", want)
			}
		}
		if strings.Contains(out.String(), "<h4") == personal {
			t.Fatal("OAuth title suppression leaked to nonpersonal caller")
		}
	}
}
