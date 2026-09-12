package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestSodaOperatorNavigationDoesNotDependOnNativeSiteAdmin(t *testing.T) {
	for _, signed := range []bool{false, true} {
		tmpl, err := template.New("extra").Funcs(template.FuncMap{"AppSubUrl": func() string { return "/native" }, "dict": forgejoTemplateDict, "ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} }, "svg": func(...any) string { return "icon" }}).Parse(readForgejoTemplate(t, "custom/extra_links.tmpl"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tmpl.New("custom/soda/guest_theme").Parse(""); err != nil {
			t.Fatal(err)
		}
		var body bytes.Buffer
		if err = tmpl.Execute(&body, map[string]any{"IsSigned": signed, "SignedUserID": int64(9007199254740993), "IsAdmin": false}); err != nil {
			t.Fatal(err)
		}
		source := body.String()
		if strings.Contains(source, `id="soda-settings-link"`) != signed {
			t.Fatal("wrong native gate")
		}
		if signed && !strings.Contains(source, `data-actor="9007199254740993"`) {
			t.Fatal("rounded actor")
		}
		if strings.Contains(source, "/settings/runners") {
			t.Fatal("unconditional operator navigation")
		}
		if strings.Count(source, `id="soda-spaces-link"`) != 1 {
			t.Fatal("Spaces navigation needs one stable presentation target")
		}
		if !strings.Contains(source, map[bool]string{true: `/native/?soda-view=spaces`, false: `/native/-/soda/spaces`}[signed]) {
			t.Fatal("lost Spaces")
		}
	}
}
