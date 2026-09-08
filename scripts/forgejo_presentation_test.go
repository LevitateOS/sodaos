package scripts

import (
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Render production partials and the production registry. This is a local
// component reference, never a replacement for native route evidence.
func TestForgejoPresentationGallery(t *testing.T) {
	if os.Getenv("SODA_FORGEJO_GALLERY") != "1" {
		t.Skip("explicit local gallery output")
	}
	header := readForgejoTemplate(t, "custom/header.tmpl")
	links := regexp.MustCompile(`<link[^>]+>`).FindAllString(header, -1)
	registry := strings.ReplaceAll(strings.Join(links, "\n"), "{{AssetUrlPrefix}}", "http://localhost:3300/assets")
	fixtures, err := os.ReadFile("../tests/forgejo/presentation/gallery.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	funcs := template.FuncMap{"dict": forgejoTemplateDict, "AssetUrlPrefix": func() string { return "http://localhost:3300/assets" }, "svg": func(string, ...any) string { return "" }, "AppSubUrl": func() string { return "" }, "DisableWebhooks": func() bool { return false }, "ctx": func() forgejoTemplateContext {
		return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{"soda.nav_personal": "Personal", "soda.nav_access": "Access & integrations", "soda.nav_resources": "Resources", "settings.profile": "Profile", "settings.account": "Account", "settings.appearance": "Appearance", "settings.security": "Security", "settings.applications": "Applications", "settings.ssh_gpg_keys": "SSH/GPG keys", "settings.repos": "Repositories", "settings.organization": "Organizations", "settings.blocked_users": "Blocked users", "repo.settings.hooks": "Webhooks"}}}
	}}
	source := `{{define "custom/soda/page_intro"}}` + readForgejoTemplate(t, "custom/soda/page_intro.tmpl") + `{{end}}`
	source += `{{define "custom/soda/empty_content"}}` + readForgejoTemplate(t, "custom/soda/empty_content.tmpl") + `{{end}}`
	source += `{{define "user/settings/navbar"}}` + readForgejoTemplate(t, "user/settings/navbar.tmpl") + `{{end}}`
	source += `{{define "gallery"}}` + string(fixtures) + `{{end}}`
	parsed, err := template.New("gallery").Funcs(funcs).Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	dir := "../.artifacts/forgejo-presentation"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		f, err := os.Create(filepath.Join(dir, "gallery-"+theme+".html"))
		if err != nil {
			t.Fatal(err)
		}
		err = parsed.ExecuteTemplate(f, "gallery", map[string]any{"Theme": theme, "Registry": template.HTML(registry)})
		closeErr := f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
}
