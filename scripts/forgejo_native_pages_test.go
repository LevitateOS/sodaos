package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

type dashboardFixtureUser struct{ IsOrganization bool }

func (dashboardFixtureUser) ShortName(int) string { return "Fixture user" }

func renderForgejoDashboard(t *testing.T, prefix string) string {
	t.Helper()
	functions := template.FuncMap{
		"AppSubUrl": func() string { return prefix },
		"svg":       func(...any) string { return "icon" },
		"dict":      forgejoTemplateDict,
	}
	source := `{{define "base/head"}}NATIVE_HEAD{{end}}
{{define "base/footer"}}NATIVE_FOOTER{{end}}
{{define "base/alert"}}{{end}}
{{define "custom/soda/page_intro"}}NATIVE_INTRO{{end}}
{{define "user/dashboard/navbar"}}NATIVE_CONTEXT{{end}}
{{define "user/heatmap"}}NATIVE_HEATMAP{{end}}
{{define "user/dashboard/feeds"}}NATIVE_FEED{{end}}
{{define "user/dashboard/guide"}}NATIVE_GUIDE{{end}}
{{define "user/dashboard/repolist"}}NATIVE_REPOSITORIES{{end}}
{{define "page"}}` + readForgejoTemplate(t, "user/dashboard/dashboard.tmpl") + `{{end}}`
	tmpl, err := template.New("dashboard").Funcs(functions).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var result bytes.Buffer
	if err := tmpl.ExecuteTemplate(&result, "page", map[string]any{
		"Title": "Dashboard", "IsSigned": true, "SignedUser": dashboardFixtureUser{}, "ContextUser": dashboardFixtureUser{},
	}); err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func TestForgejoDashboardKeepsNativeContent(t *testing.T) {
	for _, prefix := range []string{"", "/forge"} {
		body := renderForgejoDashboard(t, prefix)
		for _, want := range []string{"NATIVE_HEAD", "NATIVE_FOOTER", "NATIVE_INTRO", "NATIVE_CONTEXT", "NATIVE_HEATMAP", "NATIVE_GUIDE", "NATIVE_REPOSITORIES"} {
			if !strings.Contains(body, want) {
				t.Fatal("lost dashboard content", want)
			}
		}
	}
}

func renderForgejoAdminDashboard(t *testing.T, prefix string, data map[string]any) string {
	t.Helper()
	locale := forgejoTemplateLocale{translations: map[string]string{
		"admin.dashboard.new_major_version_hint": "admin.dashboard.new_major_version_hint",
		"admin.dashboard.new_minor_version_hint": "admin.dashboard.new_minor_version_hint",
		"admin.dashboard.update_checker_error":   "admin.dashboard.update_checker_error",
	}}
	functions := template.FuncMap{
		"ctx":             func() forgejoTemplateContext { return forgejoTemplateContext{Locale: locale} },
		"AppSubUrl":       func() string { return prefix },
		"AppVer":          func() string { return "15.0.9" },
		"svg":             func(...any) string { return "icon" },
		"dict":            forgejoTemplateDict,
		"DisableWebhooks": func() bool { return false },
	}
	source := `{{define "base/head"}}NATIVE_HEAD{{end}}
{{define "base/alert"}}NATIVE_ALERT{{end}}
{{define "admin/system_status"}}NATIVE_STATUS{{end}}
{{define "admin/layout_head"}}` + readForgejoTemplate(t, "admin/layout_head.tmpl") + `{{end}}
{{define "admin/navbar"}}FOUNTAIN_EXTENSION_NAVIGATION{{end}}
{{define "admin/layout_footer"}}ADMIN_FOOTER{{end}}
{{define "custom/soda/page_intro"}}NATIVE_INTRO{{end}}
{{define "page"}}` + readForgejoTemplate(t, "admin/dashboard.tmpl") + `{{end}}`
	tmpl, err := template.New("admin-dashboard").Funcs(functions).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var result bytes.Buffer
	if err := tmpl.ExecuteTemplate(&result, "page", data); err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func TestForgejoAdminDashboardKeepsNativeOperations(t *testing.T) {
	for _, prefix := range []string{"", "/forge"} {
		data := map[string]any{
			"Title": "Dashboard", "IsSigned": true, "IsAdmin": true, "Link": prefix + "/admin",
			"DatabaseType": map[string]bool{"IsMySQL": false}, "EnableActions": true,
		}
		body := renderForgejoAdminDashboard(t, prefix, data)
		for _, want := range []string{"NATIVE_HEAD", "NATIVE_ALERT", "NATIVE_STATUS", "ADMIN_FOOTER", "FOUNTAIN_EXTENSION_NAVIGATION", "hx-get="} {
			if !strings.Contains(body, want) {
				t.Fatal("lost native administrator content", want)
			}
		}
		for field, key := range map[string]string{
			"NeedMajorUpdate":    "admin.dashboard.new_major_version_hint",
			"NeedMinorUpdate":    "admin.dashboard.new_minor_version_hint",
			"UpdateCheckerError": "admin.dashboard.update_checker_error",
		} {
			updated := map[string]any{}
			for name, value := range data {
				updated[name] = value
			}
			updated[field] = true
			if !strings.Contains(renderForgejoAdminDashboard(t, prefix, updated), key) {
				t.Fatal("lost native update notice", field)
			}
		}
	}
}
