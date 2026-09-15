package scripts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"testing"
)

func sodaNavigationFuncs() template.FuncMap {
	return template.FuncMap{
		"AppSubUrl": func() string { return "/native" }, "dict": forgejoTemplateDict,
		"ctx":             func() forgejoTemplateContext { return forgejoTemplateContext{} },
		"svg":             func(...any) string { return "icon" },
		"DisableWebhooks": func() bool { return false },
	}
}

func TestSodaSettingsStayOutOfGlobalNavigation(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, admin := range []bool{false, true} {
			tmpl := template.Must(template.New("extra").Funcs(sodaNavigationFuncs()).Parse(readForgejoTemplate(t, "custom/extra_links.tmpl")))
			template.Must(tmpl.New("custom/soda/guest_theme").Parse(""))
			var body bytes.Buffer
			if err := tmpl.Execute(&body, map[string]any{"IsSigned": signed, "SignedUserID": int64(9007199254740993), "IsAdmin": admin}); err != nil {
				t.Fatal(err)
			}
			source := body.String()
			if strings.Contains(source, `id="soda-settings-link"`) != signed {
				t.Fatal("lost native actor/logout marker gate")
			}
			if signed && (!strings.Contains(source, `data-actor="9007199254740993"`) || !strings.Contains(source, `data-sub-url="/native"`)) {
				t.Fatal("lost exact actor or prefixed logout context")
			}
			for _, forbidden := range []string{"soda-view=runners", "soda-view=tailnet", "/settings/runners", "/settings/tailnet"} {
				if strings.Contains(source, forbidden) {
					t.Fatal("operator settings in global navigation", forbidden)
				}
			}
			if strings.Count(source, `id="soda-spaces-link"`) != 1 || !strings.Contains(source, map[bool]string{true: `/native/-/soda/workspace`, false: `/native/-/soda/spaces`}[signed]) {
				t.Fatal("lost native Spaces entry")
			}
		}
	}
}

func TestSodaSettingsRenderInNativeAdministrationWithoutSodaSession(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, admin := range []bool{false, true} {
			tmpl := template.Must(template.New("admin").Funcs(sodaNavigationFuncs()).Parse(readForgejoTemplate(t, "admin/layout_head.tmpl")))
			template.Must(tmpl.New("admin/navbar").Parse(readForgejoTemplate(t, "admin/navbar.tmpl")))
			for _, name := range []string{"base/head", "base/alert", "custom/soda/page_intro"} {
				template.Must(tmpl.New(name).Parse("native-seam:" + name))
			}
			var body bytes.Buffer
			// No Soda session, operator discovery result or provider data exists in
			// native template context. Forgejo owns this area; Soda APIs own writes.
			if err := tmpl.Execute(&body, map[string]any{"ctxData": map[string]any{"IsSigned": signed, "IsAdmin": admin, "DatabaseType": map[string]bool{"IsMySQL": false}, "EnableActions": true}}); err != nil {
				t.Fatal(err)
			}
			source := body.String()
			if strings.Contains(source, `id="soda-admin-settings"`) {
				t.Fatal("retired content toolbar returned")
			}
			assertSodaSidebarEntries(t, source)
			for _, marker := range []string{`href="/native/-/soda/settings/runners"`, `href="/native/-/soda/settings/tailnet"`} {
				want := 0
				if signed && admin {
					want = 1
				}
				if strings.Count(source, marker) != want {
					t.Fatal("wrong native administration visibility", signed, admin, marker)
				}
			}
			if !strings.Contains(source, `href="/native/admin/actions/runners"`) {
				t.Fatal("lost Forgejo's separate native Actions runners")
			}
			for _, seam := range []string{"base/head", "base/alert"} {
				if strings.Count(source, "native-seam:"+seam) != 1 {
					t.Fatal("lost native administration", seam)
				}
			}
		}
	}
}

func assertSodaSidebarEntries(t *testing.T, source string) {
	t.Helper()
	// These exact balanced outer delimiters are pinned by the upstream-source
	// test below. Check the rendered sidebar, not just link existence in a page.
	start := strings.Index(source, "<div class=\"flex-container-nav\">\n\t<div class=\"ui fluid vertical menu\">")
	if start < 0 {
		t.Fatal("native sidebar missing")
	}
	end := strings.Index(source[start:], "\n\t</div>\n</div>")
	if end < 0 {
		t.Fatal("native sidebar end missing")
	}
	sidebar := source[start : start+end]
	for _, id := range []string{`id="soda-runners-link"`, `id="soda-tailnet-link"`} {
		if strings.Count(sidebar, id) != strings.Count(source, id) {
			t.Error("settings entry outside the native sidebar", id)
		}
	}
}

func TestSodaAdminNavbarRetainsExact1507Source(t *testing.T) {
	const provenance = `{{/* Soda navigation addition to Forgejo 15.0.7 templates/admin/navbar.tmpl; GPL-3.0-or-later.
Upstream: https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/templates/admin/navbar.tmpl
Embedded source SHA-256: b0298e1f0850ce38bea744ea6a65a16853820b27ce52208cb93737a5c1bd71ac */}}
`
	const addition = `		{{if and .IsSigned .IsAdmin}}
		<div class="header item">Soda</div>
		<a id="soda-runners-link" class="item" href="{{AppSubUrl}}/-/soda/settings/runners">{{ctx.Locale.Tr "actions.runners"}}</a>
		<a id="soda-tailnet-link" class="item" href="{{AppSubUrl}}/-/soda/settings/tailnet">{{ctx.Locale.Tr "soda.tailnet_title"}}</a>
		{{end}}
`
	source := readForgejoTemplate(t, "admin/navbar.tmpl")
	if !strings.HasPrefix(source, provenance) || strings.Count(source, addition) != 1 {
		t.Fatal("unreviewed sidebar addition")
	}
	original := strings.Replace(strings.TrimPrefix(source, provenance), addition, "", 1)
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(original))); got != "b0298e1f0850ce38bea744ea6a65a16853820b27ce52208cb93737a5c1bd71ac" {
		t.Fatal("native admin navbar changed beyond the reviewed addition", got)
	}
}
