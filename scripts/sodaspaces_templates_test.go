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

func TestForgejoFooterKeepsLiveFeatures(t *testing.T) {
	// The retired drawer no longer owns the footer. Repository actions and
	// notifications keep their native Forgejo visibility gates.
	type repository struct {
		ID                       int64
		IsBroken, IsBeingCreated bool
	}
	for _, tc := range []struct {
		name    string
		repo    *repository
		signed  bool
		actions bool
	}{
		{"signed", &repository{ID: 9223372036854775807}, true, true},
		{"anonymous", &repository{ID: 42}, false, true},
		{"no repository", nil, false, false},
		{"signed non-repository", nil, true, false},
		{"broken", &repository{ID: 42, IsBroken: true}, true, false},
		{"creating", &repository{ID: 42, IsBeingCreated: true}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := template.New("hooks").Funcs(template.FuncMap{"AppSubUrl": func() string { return "" }, "AssetUrlPrefix": func() string { return "/assets" }, "ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} }, "svg": func(...any) string { return "icon" }}).ParseFiles(
				"../appliance/forgejo/templates/custom/footer.tmpl")
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			ctx := map[string]any{"Repository": tc.repo, "IsSigned": tc.signed, "SignedUserID": int64(9007199254740993), "Link": ""}
			if err := tmpl.ExecuteTemplate(&out, "footer.tmpl", ctx); err != nil {
				t.Fatal(err)
			}
			html := out.String()
			if strings.Contains(html, `src="/assets/soda/forgejo/repository-actions.js?v=4"`) != tc.actions {
				t.Fatal("repository disclosure script escaped its native repository boundary")
			}
			if strings.Contains(html, `id="sodaspaces-root"`) || strings.Contains(html, `/assets/sodaspaces.js`) || strings.Contains(html, `data-user-id=`) {
				t.Fatal("retired drawer bootstrap returned")
			}
			if strings.Contains(html, `id="soda-notification-preview"`) != tc.signed {
				t.Fatal("notification preview changed its signed-session gate")
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

func TestForgejoFooterUsesPrefixedNativeAssets(t *testing.T) {
	tmpl, err := template.New("footer.tmpl").Funcs(template.FuncMap{"AppSubUrl": func() string { return "/native" }, "AssetUrlPrefix": func() string { return "/native/assets" }, "ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} }, "svg": func(...any) string { return "icon" }}).ParseFiles("../appliance/forgejo/templates/custom/footer.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ctx := map[string]any{"Repository": map[string]any{"ID": int64(42), "IsBroken": false, "IsBeingCreated": false}, "IsSigned": true}
	if err := tmpl.Execute(&out, ctx); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"repository-actions.js?v=4", "notification-preview.js?v=1", "repository-switcher.js?v=3-repository-only"} {
		if !strings.Contains(out.String(), `src="/native/assets/soda/forgejo/`+path+`"`) {
			t.Fatal("lost native asset subpath", path)
		}
	}
}
