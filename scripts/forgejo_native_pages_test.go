package scripts

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Match the selected upstream scalar request methods. This checks our template
// branches, not Forgejo's authentication; the opt-in browser test uses Forgejo.
type nativePageRequest struct{ Req *http.Request }

func (c nativePageRequest) FormString(key string) string { return c.Req.FormValue(key) }
func (c nativePageRequest) FormStrings(key string) []string {
	_ = c.Req.FormValue(key)
	return c.Req.Form[key]
}
func (c nativePageRequest) FormInt64(key string) int64 {
	value, _ := strconv.ParseInt(c.Req.FormValue(key), 10, 64)
	return value
}

type nativePageTemplateContext struct {
	Context nativePageRequest
	Locale  forgejoTemplateLocale
}
type nativePageUser struct{ IsOrganization bool }

func (nativePageUser) ShortName(int) string { return "Fixture user" }

func renderNativePageHost(t *testing.T, prefix, query, link string, signed, news bool) string {
	t.Helper()
	ctx := nativePageTemplateContext{Context: nativePageRequest{Req: httptest.NewRequest("GET", "http://forge.test/"+query, nil)}}
	functions := template.FuncMap{
		"ctx":            func() nativePageTemplateContext { return ctx },
		"AppSubUrl":      func() string { return prefix },
		"AssetUrlPrefix": func() string { return prefix + "/assets" },
		"AppDisplayName": func() string { return "Forgejo" },
		"svg":            func(...any) string { return "icon" },
		"dict":           forgejoTemplateDict,
	}
	source := `{{define "base/head"}}<header>NATIVE_HEAD</header>{{end}}
{{define "base/footer"}}<footer>NATIVE_FOOTER</footer>{{template "custom/footer" .}}{{end}}
{{define "custom/footer"}}` + readForgejoTemplate(t, "custom/footer.tmpl") + `{{end}}
{{define "custom/soda/page_intro"}}` + readForgejoTemplate(t, "custom/soda/page_intro.tmpl") + `{{end}}
{{define "base/alert"}}{{end}}{{define "user/dashboard/navbar"}}NATIVE_CONTEXT{{end}}
{{define "user/heatmap"}}NATIVE_HEATMAP{{end}}{{define "user/dashboard/feeds"}}NATIVE_FEED{{end}}
{{define "user/dashboard/guide"}}NATIVE_GUIDE{{end}}{{define "user/dashboard/repolist"}}NATIVE_REPOSITORIES{{end}}
{{define "page"}}` + readForgejoTemplate(t, "user/dashboard/dashboard.tmpl") + `{{end}}`
	tmpl, err := template.New("native-page-host").Funcs(functions).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var result bytes.Buffer
	err = tmpl.ExecuteTemplate(&result, "page", map[string]any{
		"Title": "Dashboard", "IsSigned": signed, "PageIsNews": news, "Link": link,
		"SignedUserID": int64(9007199254740993), "SignedUser": nativePageUser{}, "ContextUser": nativePageUser{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.String()
}

// Match the build's epoch source, rather than duplicating a version constant.
func sodaPresentationVersion(t *testing.T) string {
	t.Helper()
	match := regexp.MustCompile(`name="soda-presentation-revision" content="([a-zA-Z0-9.-]+)"`).FindStringSubmatch(readForgejoTemplate(t, "custom/header.tmpl"))
	if len(match) != 2 {
		t.Fatal("missing presentation epoch")
	}
	return match[1]
}

func TestNativeSodaPageHost(t *testing.T) {
	for _, prefix := range []string{"", "/forge"} {
		// Spaces views stay on the dashboard host. Runners and Tailnet render
		// inside the real administration layout; see TestAdminSodaSettingsHost.
		for _, tc := range []struct{ query, view, destination, repository string }{
			{"?soda-view=spaces", "spaces", "/spaces", ""},
			{"?soda-view=repository-spaces&repository_id=9223372036854775807", "repository-spaces", "/repositories/9223372036854775807/settings/spaces", "9223372036854775807"},
		} {
			t.Run(prefix+tc.query, func(t *testing.T) {
				body := renderNativePageHost(t, prefix, tc.query, prefix, true, true)
				for _, want := range []string{
					`<header>NATIVE_HEAD</header>`, `<footer>NATIVE_FOOTER</footer>`,
					`id="soda-native-content"`, `data-view="` + tc.view + `"`,
					`data-actor="9007199254740993"`, `data-repository-id="` + tc.repository + `"`,
					`src="` + prefix + `/assets/soda/forgejo/soda-native-page.js?v=` + sodaPresentationVersion(t) + `"`,
					`id="soda-notification-preview"`, `soda-settings-link.js`,
				} {
					if !strings.Contains(body, want) {
						t.Fatalf("missing %s", want)
					}
				}
				if strings.Contains(body, `/assets/soda-tailnet.css?v=`+sodaPresentationVersion(t)) != (tc.view == "tailnet") {
					t.Fatal("Tailnet stylesheet escaped its native selector")
				}
				if tc.view == "tailnet" && (!strings.Contains(body, `data-appliance-label=`) || !strings.Contains(body, `data-enrollment-label=`)) {
					t.Fatal("missing Tailnet localized section labels")
				}
				if strings.Count(body, `<main `) != 1 || strings.Count(body, `id="soda-native-content"`) != 1 {
					t.Fatal("expected one main and one content owner")
				}
				for _, forbidden := range []string{`id="sodaspaces-root"`, `/assets/sodaspaces.js`, `NATIVE_GUIDE`, `NATIVE_REPOSITORIES`, `<form`, `<soda-runners`, `<soda-tailnet`, `data-csrf`} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("native scaffold must not contain %s", forbidden)
					}
				}
			})
		}
	}
}

func TestNativeSodaPageRejectsInvalidRepositoryLocators(t *testing.T) {
	for _, query := range []string{
		"?soda-view=spaces&repository_id=1",
		"?soda-view=repository-spaces", "?soda-view=repository-spaces&repository_id=",
		"?soda-view=repository-spaces&repository_id=0", "?soda-view=repository-spaces&repository_id=-1",
		"?soda-view=repository-spaces&repository_id=01", "?soda-view=repository-spaces&repository_id=%2B1",
		"?soda-view=repository-spaces&repository_id=1&repository_id=1",
		"?soda-view=repository-spaces&repository_id=9223372036854775808",
		"?soda-view=repository-spaces&repository_id=%3Cscript%3E",
	} {
		t.Run(query, func(t *testing.T) {
			body := renderNativePageHost(t, "", query, "", true, true)
			if !strings.Contains(body, `role="alert">Invalid Soda page destination.`) {
				t.Fatal("missing bounded invalid-destination feedback")
			}
			for _, forbidden := range []string{`id="soda-native-content"`, `data-actor=`, `data-repository-id=`, `soda-native-page.js`, `id="sodaspaces-root"`} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("invalid locator mounted %s", forbidden)
				}
			}
		})
	}
}

func TestNativeSodaPagePreservesOrdinaryDashboardAndOtherRoutes(t *testing.T) {
	baseline := strings.Join(strings.Fields(renderNativePageHost(t, "", "", "", true, true)), " ")
	// Runners and Tailnet left the dashboard host for /admin?soda-view=; those
	// legacy dashboard queries must render the ordinary dashboard unchanged.
	for _, query := range []string{"?soda-view=unknown", "?soda-view=spaces&soda-view=runners", "?soda-view=tailnet&soda-view=tailnet", "?soda-view=", "?repository_id=1", "?soda-view=runners", "?soda-view=tailnet", "?soda-view=runners&repository_id=1", "?soda-view=tailnet&repository_id=1"} {
		if got := strings.Join(strings.Fields(renderNativePageHost(t, "", query, "", true, true)), " "); got != baseline {
			t.Fatalf("ordinary dashboard changed for %s", query)
		}
	}
	for _, tc := range []struct {
		name, link   string
		signed, news bool
	}{
		{"guest cannot select a native Soda body", "", false, true},
		{"other dashboard route", "/org/team/dashboard", true, true},
		{"non-dashboard gate page", "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := renderNativePageHost(t, "", "?soda-view=spaces", tc.link, tc.signed, tc.news)
			if strings.Contains(body, `id="soda-native-content"`) || !strings.Contains(body, "NATIVE_REPOSITORIES") {
				t.Fatal("Soda presentation escaped its signed root-dashboard boundary")
			}
			if strings.Contains(body, `id="sodaspaces-root"`) != tc.signed {
				t.Fatal("ordinary drawer changed on an unrelated page")
			}
		})
	}
}

// Runners and Tailnet render inside Forgejo's real administration layout on
// the existing /admin route. This checks that presentation branch against the
// actual admin shell partials, not the dashboard host above.
func renderAdminSettingsHost(t *testing.T, prefix, query string, data map[string]any) string {
	t.Helper()
	ctx := nativePageTemplateContext{Context: nativePageRequest{Req: httptest.NewRequest("GET", "http://forge.test/admin"+query, nil)}}
	functions := template.FuncMap{
		"ctx":             func() nativePageTemplateContext { return ctx },
		"AppSubUrl":       func() string { return prefix },
		"AssetUrlPrefix":  func() string { return prefix + "/assets" },
		"AppDisplayName":  func() string { return "Forgejo" },
		"AppVer":          func() string { return "15.0.7" },
		"svg":             func(...any) string { return "icon" },
		"dict":            forgejoTemplateDict,
		"DisableWebhooks": func() bool { return false },
	}
	source := `{{define "base/head"}}<header>NATIVE_HEAD</header>{{end}}
{{define "base/footer"}}<footer>NATIVE_FOOTER</footer>{{end}}
{{define "base/alert"}}NATIVE_ALERT{{end}}
{{define "admin/system_status"}}NATIVE_STATUS{{end}}
{{define "admin/layout_head"}}` + readForgejoTemplate(t, "admin/layout_head.tmpl") + `{{end}}
{{define "admin/navbar"}}` + readForgejoTemplate(t, "admin/navbar.tmpl") + `{{end}}
{{define "admin/layout_footer"}}ADMIN_FOOTER{{end}}
{{define "custom/soda/page_intro"}}` + readForgejoTemplate(t, "custom/soda/page_intro.tmpl") + `{{end}}
{{define "page"}}` + readForgejoTemplate(t, "admin/dashboard.tmpl") + `{{end}}`
	tmpl, err := template.New("admin-settings-host").Funcs(functions).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var result bytes.Buffer
	if err := tmpl.ExecuteTemplate(&result, "page", data); err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func adminSettingsHostData(prefix string, signed bool) map[string]any {
	return map[string]any{
		"Title": "Dashboard", "IsSigned": signed, "IsAdmin": signed, "SignedUserID": int64(1),
		"PageIsAdminDashboard": true, "Link": prefix + "/admin",
		"DatabaseType": map[string]bool{"IsMySQL": false}, "EnableActions": true,
	}
}

func TestAdminSodaSettingsHost(t *testing.T) {
	for _, prefix := range []string{"", "/forge"} {
		for _, view := range []string{"runners", "tailnet"} {
			t.Run(prefix+"?soda-view="+view, func(t *testing.T) {
				body := renderAdminSettingsHost(t, prefix, "?soda-view="+view, adminSettingsHostData(prefix, true))
				assertSodaSidebarEntries(t, body)
				for _, want := range []string{
					`<header>NATIVE_HEAD</header>`, `NATIVE_ALERT`, `ADMIN_FOOTER`, `<p class="soda-page-eyebrow">Soda administration</p>`,
					`id="soda-native-content"`, `data-view="` + view + `"`,
					`data-actor="1"`, `data-repository-id=""`,
					`href="` + prefix + `/-/soda/settings/runners"`, `href="` + prefix + `/-/soda/settings/tailnet"`,
					`src="` + prefix + `/assets/soda/forgejo/soda-native-page.js?v=` + sodaPresentationVersion(t) + `"`,
					`/assets/soda-settings.css?v=` + sodaPresentationVersion(t),
				} {
					if !strings.Contains(body, want) {
						t.Fatalf("missing %s", want)
					}
				}
				if strings.Contains(body, `/assets/soda-tailnet.css?v=`+sodaPresentationVersion(t)) != (view == "tailnet") {
					t.Fatal("Tailnet stylesheet escaped its native selector")
				}
				if view == "tailnet" && (!strings.Contains(body, `data-appliance-label=`) || !strings.Contains(body, `data-enrollment-label=`)) {
					t.Fatal("missing Tailnet localized section labels")
				}
				// The real admin shell owns the landmark: one role=main from the
				// layout, one content mount, and no second main element.
				if strings.Count(body, `role="main"`) != 1 || strings.Count(body, `id="soda-native-content"`) != 1 || strings.Contains(body, `<main `) {
					t.Fatal("expected one admin landmark and one content owner")
				}
				// The native operations/status branch must not render alongside.
				for _, forbidden := range []string{`hx-get=`, `no-loading-indicator`, `<form`, `<soda-runners`, `<soda-tailnet`, `data-csrf`} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("native admin content leaked into the Soda branch: %s", forbidden)
					}
				}
			})
		}
	}
}

func TestAdminSodaSettingsHostKeepsNativeDashboard(t *testing.T) {
	for _, tc := range []struct {
		query  string
		signed bool
	}{
		{"", true}, {"?soda-view=spaces", true}, {"?soda-view=unknown", true},
		{"?soda-view=runners&soda-view=runners", true}, {"?soda-view=tailnet&soda-view=runners", true},
		{"?soda-view=runners&repository_id=1", true}, {"?soda-view=tailnet&repository_id=", true},
		{"?soda-view=tailnet&repository_id=1&repository_id=2", true},
		{"?soda-view=runners", false},
	} {
		t.Run(tc.query, func(t *testing.T) {
			body := renderAdminSettingsHost(t, "", tc.query, adminSettingsHostData("", tc.signed))
			if !strings.Contains(body, `hx-get=`) || !strings.Contains(body, `ADMIN_FOOTER`) {
				t.Fatal("native admin dashboard lost its operations branch")
			}
			if strings.Contains(body, `id="soda-native-content"`) || strings.Contains(body, `soda-native-page.js`) {
				t.Fatal("Soda mount escaped its validated admin selector")
			}
		})
	}
}

func TestAdminSodaSettingsHostSuppressesDrawer(t *testing.T) {
	renderFooter := func(t *testing.T, query string, signed bool) string {
		t.Helper()
		ctx := nativePageTemplateContext{Context: nativePageRequest{Req: httptest.NewRequest("GET", "http://forge.test/admin"+query, nil)}}
		functions := template.FuncMap{
			"ctx":            func() nativePageTemplateContext { return ctx },
			"AppSubUrl":      func() string { return "" },
			"AssetUrlPrefix": func() string { return "/assets" },
			"svg":            func(...any) string { return "icon" },
		}
		tmpl, err := template.New("admin-footer").Funcs(functions).Parse(`{{define "page"}}` + readForgejoTemplate(t, "custom/footer.tmpl") + `{{end}}`)
		if err != nil {
			t.Fatal(err)
		}
		var result bytes.Buffer
		if err := tmpl.ExecuteTemplate(&result, "page", map[string]any{"IsSigned": signed, "PageIsAdminDashboard": true, "SignedUserID": int64(1)}); err != nil {
			t.Fatal(err)
		}
		return result.String()
	}
	for _, query := range []string{"?soda-view=runners", "?soda-view=tailnet"} {
		if strings.Contains(renderFooter(t, query, true), `id="sodaspaces-root"`) {
			t.Fatal("drawer owner leaked onto the admin Soda branch", query)
		}
	}
	for _, query := range []string{"", "?soda-view=spaces", "?soda-view=unknown", "?soda-view=runners&soda-view=runners", "?soda-view=tailnet&soda-view=runners", "?soda-view=runners&repository_id=1", "?soda-view=tailnet&repository_id=", "?soda-view=tailnet&repository_id=1&repository_id=2"} {
		if !strings.Contains(renderFooter(t, query, true), `id="sodaspaces-root"`) {
			t.Fatal("ordinary admin drawer changed", query)
		}
	}
	if body := renderFooter(t, "?soda-view=runners", false); strings.Contains(body, `id="sodaspaces-root"`) || strings.Contains(body, `id="soda-settings-link"`) {
		t.Fatal("signed footer metadata rendered without a signed session")
	}
}
