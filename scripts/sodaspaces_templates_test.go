package scripts

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
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
		{"broken", &repository{ID: 42, IsBroken: true}, true, false},
		{"creating", &repository{ID: 42, IsBeingCreated: true}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := template.New("hooks").Funcs(template.FuncMap{"AppSubUrl": func() string { return "" }}).ParseFiles(
				"../appliance/forgejo/templates/custom/header.tmpl", "../appliance/forgejo/templates/custom/footer.tmpl")
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			ctx := struct {
				Repository   *repository
				IsSigned     bool
				SignedUserID int64
			}{tc.repo, tc.signed, 9007199254740993}
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

func TestSodaspacesTemplateEscapesContext(t *testing.T) {
	tmpl, err := template.New("footer.tmpl").Funcs(template.FuncMap{"AppSubUrl": func() string { return "/native" }}).ParseFiles("../appliance/forgejo/templates/custom/footer.tmpl")
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
