package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func sodaNavigationFuncs() template.FuncMap {
	return template.FuncMap{
		"AppSubUrl": func() string { return "/native" }, "dict": forgejoTemplateDict,
		"ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} },
		"svg": func(...any) string { return "icon" },
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
			if strings.Count(source, `id="soda-spaces-link"`) != 1 || !strings.Contains(source, map[bool]string{true: `/native/?soda-view=spaces`, false: `/native/-/soda/spaces`}[signed]) {
				t.Fatal("lost native Spaces entry")
			}
		}
	}
}

func TestSodaSettingsRenderInNativeAdministrationWithoutSodaSession(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, admin := range []bool{false, true} {
			tmpl := template.Must(template.New("admin").Funcs(sodaNavigationFuncs()).Parse(readForgejoTemplate(t, "admin/layout_head.tmpl")))
			for _, name := range []string{"base/head", "admin/navbar", "base/alert", "custom/soda/page_intro"} {
				template.Must(tmpl.New(name).Parse("native-seam:" + name))
			}
			var body bytes.Buffer
			// No Soda session, operator discovery result or provider data exists in
			// native template context. Forgejo owns this area; Soda APIs own writes.
			if err := tmpl.Execute(&body, map[string]any{"ctxData": map[string]any{"IsSigned": signed, "IsAdmin": admin}}); err != nil {
				t.Fatal(err)
			}
			source := body.String()
			for _, marker := range []string{`id="soda-admin-settings"`, `href="/native/-/soda/settings/runners"`, `href="/native/-/soda/settings/tailnet"`} {
				want := 0
				if signed && admin {
					want = 1
				}
				if strings.Count(source, marker) != want {
					t.Fatal("wrong native administration visibility", signed, admin, marker)
				}
			}
			for _, seam := range []string{"base/head", "admin/navbar", "base/alert"} {
				if strings.Count(source, "native-seam:"+seam) != 1 {
					t.Fatal("lost native administration", seam)
				}
			}
		}
	}
}
