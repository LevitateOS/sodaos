package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoHomeRedesign(t *testing.T) {
	for _, prefix := range []string{"", "/forge"} {
		for _, registration := range []bool{false, true} {
			ctx := forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: map[string]string{"home": "Home", "sign_in": "Sign in"}}}
			funcs := template.FuncMap{"ctx": func() forgejoTemplateContext { return ctx }, "AppSubUrl": func() string { return prefix }, "AssetUrlPrefix": func() string { return prefix + "/assets" }, "dict": func() map[string]any { return map[string]any{} }}
			tmpl := template.New("home").Funcs(funcs)
			// Only the base shell is a fixture; render the authored front page and theme control.
			shell := `{{define "base/head"}}<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Soda front-page template preview</title><link rel="stylesheet" href="/assets/soda/fonts/fonts.css"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-guest.css"><link rel="stylesheet" href="/assets/soda/forgejo/home.css"><style>*{box-sizing:border-box}body{margin:0}a{text-decoration:none}footer{padding:24px 16px;font:14px Barlow,sans-serif}</style></head><body>{{end}}{{define "base/footer"}}<footer>Soda OS · Local template preview</footer></body></html>{{end}}`
			var err error
			tmpl, err = tmpl.Parse(shell)
			if err != nil {
				t.Fatal(err)
			}
			_, err = tmpl.New("custom/soda/theme_toggle").Parse(readForgejoTemplate(t, "custom/soda/theme_toggle.tmpl"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = tmpl.Parse(readForgejoTemplate(t, "home.tmpl"))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err = tmpl.Execute(&out, map[string]any{"ShowRegistrationButton": registration}); err != nil {
				t.Fatal(err)
			}
			html := out.String()
			if strings.Contains(html, "papercraft") || strings.Contains(html, "robot") {
				t.Fatal("decorative hero returned")
			}
			for _, route := range []string{"/user/login", "/explore/repos"} {
				if !strings.Contains(html, `href="`+prefix+route+`"`) {
					t.Fatalf("missing subpath-aware %s", route)
				}
			}
			if strings.Contains(html, `href="`+prefix+`/user/sign_up"`) != registration {
				t.Fatal("registration gate changed")
			}
			if !registration && !strings.Contains(html, "Ask your administrator") {
				t.Fatal("missing invitation guidance")
			}
			if prefix == "" && registration {
				dir := filepath.Join("..", ".artifacts", "forgejo-presentation")
				if err = os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				for _, theme := range []string{"light", "dark"} {
					preview := strings.Replace(html, "<html>", `<html data-soda-login-theme="`+theme+`" style="color-scheme: `+theme+`">`, 1)
					if err = os.WriteFile(filepath.Join(dir, "home-"+theme+".html"), []byte(preview), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
			if dir := os.Getenv("SODA_HOME_PREVIEW"); dir != "" && prefix == "" && registration {
				if err = os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, "index.html"), out.Bytes(), 0644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
