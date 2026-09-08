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
				input = map[string]any{"PersonalSettings": true, "SettingsPresentation": true, "ctxData": data}
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

func TestForgejoPersonalActionsUseScopedEmptyStatesAndWarning(t *testing.T) {
	personalActions := readForgejoTemplate(t, "user/settings/actions.tmpl")
	for _, call := range []string{
		`{{template "shared/secrets/add_list" (dict "SettingsPresentation" true "ctxData" .)}}`,
		`{{template "shared/actions/runner_list" (dict "SettingsPresentation" true "ctxData" .)}}`,
		`{{template "shared/variables/variable_list" (dict "SettingsPresentation" true "ctxData" .)}}`,
	} {
		if !strings.Contains(personalActions, call) {
			t.Errorf("personal Actions caller lacks scoped adapter %q", call)
		}
	}

	for _, fixture := range []struct {
		path, body, empty string
	}{
		{"shared/secrets/add_list.tmpl", "shared/secrets/add_list_body", `ctx.Locale.Tr "secrets.none"`},
		{"shared/actions/runner_list.tmpl", "shared/actions/runner_list_body", `ctx.Locale.Tr "actions.runners.none"`},
		{"shared/variables/variable_list.tmpl", "shared/variables/variable_list_body", `ctx.Locale.Tr "actions.variables.none"`},
	} {
		source := readForgejoTemplate(t, fixture.path)
		for _, marker := range []string{
			`{{if .SettingsPresentation}}`,
			`{{define "` + fixture.body + `"}}`,
			`{{if $settings}}<div class="soda-empty soda-empty--compact">`,
			fixture.empty,
		} {
			if !strings.Contains(source, marker) {
				t.Errorf("%s lacks personal-only empty-state marker %q", fixture.path, marker)
			}
		}
	}

	runnerSetup := readForgejoTemplate(t, "shared/actions/runner_setup.tmpl")
	if !strings.Contains(runnerSetup, `<p{{if $settings}} class="ui warning message soda-notice"{{end}}>{{ctx.Locale.Tr "actions.runners.runner_setup.last_chance_copying_token"}}</p>`) {
		t.Error("personal runner setup lacks its static last-chance warning")
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
			input = map[string]any{"PersonalSettings": true, "SettingsPresentation": true, "ctxData": data}
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
			input = map[string]any{"PersonalSettings": true, "SettingsPresentation": true, "ctxData": data}
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

func (forgejoSettingsFixtureLocale) TrSize(size any) string { return "1 MiB" }

func TestForgejoAvatarSourceChoices(t *testing.T) {
	source := readForgejoTemplate(t, "user/settings/profile.tmpl")
	start := strings.Index(source, "<form ")
	source = source[start : start+strings.Index(source[start:], "</form>")+len("</form>")]
	parsed, err := template.New("avatar").Funcs(template.FuncMap{"ctx": func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} }}).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, disabled := range []bool{true, false} {
		for _, custom := range []bool{true, false} {
			var out bytes.Buffer
			err := parsed.Execute(&out, map[string]any{"DisableGravatar": disabled, "SignedUser": map[string]any{"UseCustomAvatar": custom}, "Link": "/user/settings"})
			if err != nil {
				t.Fatal(err)
			}
			html := out.String()
			if disabled {
				if strings.Contains(html, `type="radio"`) || strings.Contains(html, `id="gravatar"`) || !strings.Contains(html, `name="source" value="local" type="hidden"`) {
					t.Fatal("upload-only mode must submit local without a false choice")
				}
			} else {
				if strings.Count(html, `type="radio"`) != 2 || strings.Contains(html, `type="hidden"`) {
					t.Fatal("lookup-enabled mode must retain both source choices")
				}
				selected := `value="lookup" type="radio" checked`
				if custom {
					selected = `value="local" type="radio" checked`
				}
				if !strings.Contains(html, selected) {
					t.Fatal("lost native avatar source selection")
				}
			}
			for _, contract := range []string{`action="/user/settings/avatar"`, `enctype="multipart/form-data"`, `name="avatar" type="file"`, `data-url="/user/settings/avatar/delete"`} {
				if !strings.Contains(html, contract) {
					t.Fatalf("lost avatar contract %s", contract)
				}
			}
		}
	}
}
