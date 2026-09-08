package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoAccountSettingsLayoutComposesNativeSeams(t *testing.T) {
	functions := template.FuncMap{"dict": forgejoTemplateDict}
	definition := `
		{{define "base/head"}}native-head/{{.Title}}{{end}}
		{{define "base/alert"}}native-alert/{{.Title}}{{end}}
		{{define "base/footer"}}native-footer/{{.Title}}{{end}}
		{{define "user/settings/navbar"}}native-settings-navbar/{{.Title}}{{end}}
		{{define "custom/soda/page_intro"}}soda-intro/{{.Title}}/{{.Class}}{{end}}
		{{define "settings-head"}}` + readForgejoTemplate(t, "user", "settings", "layout_head.tmpl") + `{{end}}
		{{define "settings-footer"}}` + readForgejoTemplate(t, "user", "settings", "layout_footer.tmpl") + `{{end}}`
	parsed, err := template.New("account-settings").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse account settings layout: %v", err)
	}

	page := map[string]any{
		"Title": "Security & access",
		"ctxData": map[string]any{
			"Title": "Security & access",
		},
		"pageClass": "user settings security",
	}
	var head bytes.Buffer
	if err := parsed.ExecuteTemplate(&head, "settings-head", page); err != nil {
		t.Fatalf("execute account settings head: %v", err)
	}
	output := head.String()
	for _, want := range []string{
		"native-head/Security &amp; access",
		"native-settings-navbar/Security &amp; access",
		"native-alert/Security &amp; access",
		"soda-intro/Security &amp; access/soda-page-intro--compact",
		`class="page-content soda-page soda-native-forms soda-settings user settings security"`,
		`data-signed="true"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("account settings head does not contain %q:\n%s", want, output)
		}
	}

	var footer bytes.Buffer
	if err := parsed.ExecuteTemplate(&footer, "settings-footer", page); err != nil {
		t.Fatalf("execute account settings footer: %v", err)
	}
	if !strings.Contains(footer.String(), "native-footer/Security &amp; access") {
		t.Errorf("account settings footer lost native footer:\n%s", footer.String())
	}
}

func TestForgejoAccountSettingsStylesRemainPageScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "account-settings.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	if !strings.Contains(css, ".soda-settings") {
		t.Fatal("account settings stylesheet is missing its page root")
	}
	for _, forbidden := range []string{"body ", "#navbar", ".soda-page {", ".soda-form"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("account settings stylesheet reaches outside its page contract with %q", forbidden)
		}
	}
}

func TestForgejoAccountSettingsArtworkUsesNativePageFlags(t *testing.T) {
	definition := `{{define "base/head"}}{{end}}{{define "base/alert"}}{{end}}{{define "user/settings/navbar"}}{{end}}{{define "custom/soda/page_intro"}}{{.Artwork}}{{end}}{{define "settings"}}` + readForgejoTemplate(t, "user", "settings", "layout_head.tmpl") + `{{end}}`
	parsed, err := template.New("settings-artwork").Funcs(template.FuncMap{"dict": forgejoTemplateDict}).Parse(definition)
	if err != nil {
		t.Fatal(err)
	}
	for flag, artwork := range map[string]string{
		"PageIsSettingsProfile":      "settings-profile-papercraft.png",
		"PageIsSettingsAccount":      "settings-account-papercraft.png",
		"PageIsSettingsAppearance":   "settings-appearance-papercraft.png",
		"PageIsSettingsSecurity":     "settings-security-papercraft.png",
		"PageIsSettingsKeys":         "settings-keys-papercraft.png",
		"PageIsSettingsApplications": "settings-applications-papercraft.png",
	} {
		t.Run(flag, func(t *testing.T) {
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "settings", map[string]any{"ctxData": map[string]any{flag: true}}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), artwork) {
				t.Fatalf("native %s did not select %s", flag, artwork)
			}
			if _, err := os.Stat(filepath.Join("..", "assets", "branding", "forgejo", artwork)); err != nil {
				t.Fatal(err)
			}
		})
	}
	var unrelated bytes.Buffer
	if err := parsed.ExecuteTemplate(&unrelated, "settings", map[string]any{"ctxData": map[string]any{"PageIsSettingsRepos": true}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(unrelated.String(), "papercraft.png") {
		t.Fatal("unrelated settings page borrowed a different page's artwork")
	}
}
