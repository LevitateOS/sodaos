package scripts

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	texttemplate "text/template"
)

func TestSodaspacesTemplates(t *testing.T) {
	// Only Soda's hooks and the native scalar context contract, not a Forgejo renderer.
	type repository struct {
		ID                       int64
		IsBroken, IsBeingCreated bool
	}
	for _, tc := range []struct {
		name   string
		repo   *repository
		signed bool
		want   bool
	}{
		{"signed", &repository{ID: 9223372036854775807}, true, true},
		{"anonymous", &repository{ID: 42}, false, true},
		{"no repository", nil, false, false},
		{"signed non-repository resume hook", nil, true, true},
		{"broken", &repository{ID: 42, IsBroken: true}, true, false},
		{"creating", &repository{ID: 42, IsBeingCreated: true}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := template.New("hooks").Funcs(template.FuncMap{"AppSubUrl": func() string { return "" }, "AssetUrlPrefix": func() string { return "/assets" }, "dict": forgejoTemplateDict, "ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} }, "svg": func(...any) string { return "icon" }}).ParseFiles(
				"../appliance/forgejo/templates/custom/header.tmpl", "../appliance/forgejo/templates/custom/footer.tmpl")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = tmpl.New("custom/soda/guest_theme").Parse(readForgejoTemplate(t, "custom/soda/guest_theme.tmpl")); err != nil {
				t.Fatal(err)
			}
			if _, err = tmpl.New("custom/soda/theme_toggle").Parse(readForgejoTemplate(t, "custom/soda/theme_toggle.tmpl")); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			ctx := map[string]any{"Repository": tc.repo, "IsSigned": tc.signed, "SignedUserID": int64(9007199254740993), "Link": ""}
			if err := tmpl.ExecuteTemplate(&out, "footer.tmpl", ctx); err != nil {
				t.Fatal(err)
			}
			html := out.String()
			if strings.Contains(html, `id="sodaspaces-root"`) != tc.want {
				t.Fatal("incorrect repository guard")
			}
			if tc.want {
				for _, value := range []string{`type="button"`, `aria-labelledby="sodaspaces-title"`, `src="/assets/sodaspaces.js"`, `data-sub-url=""`} {
					if !strings.Contains(html, value) {
						t.Fatalf("missing %s", value)
					}
				}
				if tc.signed && !strings.Contains(html, `data-user-id="9007199254740993"`) {
					t.Fatal("rounded actor")
				}
				if !tc.signed && !strings.Contains(html, `data-user-id=""`) {
					t.Fatal("anonymous actor emitted")
				}
			}
			out.Reset()
			if err := tmpl.ExecuteTemplate(&out, "header.tmpl", ctx); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), `href="/assets/sodaspaces.css"`) {
				t.Fatal("missing local stylesheet")
			}
		})
	}
}

func TestSodaspacesRequestLoggingOmitsQueries(t *testing.T) {
	body, err := os.ReadFile("../appliance/config/forgejo.env")
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		if key, value, found := strings.Cut(line, "="); found {
			values[key] = value
		}
	}
	if mode, exists := values["FORGEJO__log__LOGGER_ROUTER_MODE"]; !exists || mode != "" {
		t.Fatal("native query-bearing router log enabled")
	}
	if values["FORGEJO__log__LOGGER_ACCESS_MODE"] != "console" {
		t.Fatal("lost native request diagnostics")
	}
	tmpl, err := texttemplate.New("native-access").Parse(values["FORGEJO__log__ACCESS_LOG_TEMPLATE"])
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "https://forge.test/login/oauth/authorize%0A?state=SYNTHETIC_PRIVATE_MARKER", nil)
	r.Header.Set("Referer", "https://forge.test/?code=SYNTHETIC_PRIVATE_MARKER")
	var out bytes.Buffer
	if err = tmpl.Execute(&out, map[string]any{"Ctx": map[string]any{"Req": r}, "ResponseWriter": map[string]any{"Status": 200}}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "GET /login/oauth/authorize%0A 200" {
		t.Fatal("unexpected request log fields")
	}
}

func TestSodaspacesTemplateEscapesContext(t *testing.T) {
	tmpl, err := template.New("footer.tmpl").Funcs(template.FuncMap{"AppSubUrl": func() string { return "/native" }, "AssetUrlPrefix": func() string { return "/native/assets" }, "ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} }, "svg": func(...any) string { return "icon" }}).ParseFiles("../appliance/forgejo/templates/custom/footer.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ctx := map[string]any{"Repository": map[string]any{"ID": `42" onmouseover="evil`, "IsBroken": false, "IsBeingCreated": false}, "IsSigned": true, "SignedUserID": `1"><script>evil</script>`}
	if err := tmpl.Execute(&out, ctx); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `<script>evil`) || strings.Contains(out.String(), `data-repository-id="42" onmouseover`) {
		t.Fatal("unescaped context")
	}
	if !strings.Contains(out.String(), `src="/native/assets/sodaspaces.js"`) {
		t.Fatal("lost native asset subpath")
	}
}
