package scripts

import (
	"bytes"
	"context"
	"html/template"
	"os"
	"strings"
	"testing"
)

type previewRequestContext struct {
	context.Context
	flags map[string]bool
}

func (c previewRequestContext) FormBool(key string) bool { return c.flags[key] }

type previewTemplateContext struct {
	context.Context
	Locale forgejoTemplateLocale
}
type previewDates struct{}

func (previewDates) TimeSince(value any) string { return "recently" }

type previewNotification struct {
	ID          int
	Status      int
	Issue       any
	Repository  map[string]any
	UpdatedUnix int
}

func (previewNotification) Link(any) string { return "/forge/team/repo/issues/1" }

func TestForgejoNotificationPreviewRendering(t *testing.T) {
	for _, count := range []int{0, 1, 5} {
		for _, flags := range []map[string]bool{{}, {"soda-preview": true}, {"div-only": true}, {"soda-preview": true, "div-only": true}} {
			ctx := previewTemplateContext{Context: previewRequestContext{Context: context.Background(), flags: flags}, Locale: forgejoTemplateLocale{}}
			funcs := template.FuncMap{
				"ctx":                 func() previewTemplateContext { return ctx },
				"AppSubUrl":           func() string { return "/forge" },
				"AssetUrlPrefix":      func() string { return "/forge/assets" },
				"svg":                 func(...any) string { return "icon" },
				"DateUtils":           func() previewDates { return previewDates{} },
				"RenderRefIssueTitle": func(_ any, s string) string { return s },
				"dict":                forgejoTemplateDict,
			}
			source := `{{define "custom/soda/notification_preview"}}` + readForgejoTemplate(t, "custom", "soda", "notification_preview.tmpl") + `{{end}}
{{define "shared/issueicon"}}issue-icon{{end}}
{{define "custom/soda/page_intro"}}{{end}}
{{define "custom/soda/empty_content"}}{{end}}
{{define "base/paginate"}}{{end}}
{{define "fragment"}}` + readForgejoTemplate(t, "user", "notification", "notification_div.tmpl") + `{{end}}`
			tmpl, err := template.New("preview").Funcs(funcs).Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			rows := make([]previewNotification, count)
			for i := range rows {
				rows[i] = previewNotification{ID: i + 1, Status: 1, Repository: map[string]any{"FullName": "team/<repo>"}}
				if i%2 == 0 {
					rows[i].Issue = map[string]any{"Index": 1, "Title": "<script>alert(1)</script>"}
				}
				if i == 0 {
					rows[i].Status = 3
				}
			}
			data := map[string]any{"Notifications": rows, "Context": context.Background(), "Title": "Notifications", "Status": 1, "NotificationUnreadCount": func() int { return count }, "Page": map[string]any{"Paginater": map[string]any{"Current": 1}}}
			var out bytes.Buffer
			if err := tmpl.ExecuteTemplate(&out, "fragment", data); err != nil {
				t.Fatal(err)
			}
			if os.Getenv("SODA_FORGEJO_NOTIFICATION_GALLERY") == "1" && count == 5 && len(flags) == 0 {
				if err := os.MkdirAll("../.artifacts/forgejo-notification-refinement", 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile("../.artifacts/forgejo-notification-refinement/populated.html", out.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			compact := flags["soda-preview"] && flags["div-only"]
			if strings.Contains(out.String(), "data-soda-notification-fragment") != compact {
				t.Fatalf("wrong branch for %v", flags)
			}
			if compact {
				for _, forbidden := range []string{`id="notification_`, `role="main"`, `<form`, `<script>`} {
					if strings.Contains(out.String(), forbidden) {
						t.Fatalf("compact contains %q", forbidden)
					}
				}
				if strings.Count(out.String(), `href="/forge/team/repo/issues/1"`) != count {
					t.Fatal("lost native links")
				}
				if count > 0 && (!strings.Contains(out.String(), "&lt;script&gt;") || !strings.Contains(out.String(), "Pinned")) {
					t.Fatal("escaping or pinned label missing")
				}
				if count == 0 && !strings.Contains(out.String(), "No unread or pinned") {
					t.Fatal("empty state missing")
				}
			} else if !strings.Contains(out.String(), `id="notification_div"`) {
				t.Fatal("native full page lost")
			}
		}
	}
}

func TestForgejoNotificationPreviewSignedInHook(t *testing.T) {
	funcs := template.FuncMap{"ctx": func() forgejoTemplateContext { return forgejoTemplateContext{} }, "AppSubUrl": func() string { return "/forge" }, "AssetUrlPrefix": func() string { return "/forge/assets" }, "svg": func(...any) string { return "icon" }}
	tmpl, err := template.New("footer").Funcs(funcs).Parse(readForgejoTemplate(t, "custom", "footer.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, signed := range []bool{false, true} {
		var out bytes.Buffer
		if err := tmpl.Execute(&out, map[string]any{"IsSigned": signed}); err != nil {
			t.Fatal(err)
		}
		if !signed && strings.TrimSpace(out.String()) != "" {
			t.Fatal("guest loads preview")
		}
		if signed {
			for _, want := range []string{`hx-get="/forge/notifications?`, `soda-preview=true`, `perPage=5`, `/forge/assets/soda/forgejo/notification-preview.js`, `View all notifications`} {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("missing %q", want)
				}
			}
		}
	}
}
