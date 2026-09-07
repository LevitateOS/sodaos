package scripts

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type forgejoTemplateLocale struct {
	translations map[string]string
}

func (l forgejoTemplateLocale) Tr(key string) string {
	return l.translations[key]
}

type forgejoTemplateContext struct {
	Locale forgejoTemplateLocale
}

// forgejoDict mirrors the alternating string-key/value and dot-merge contract
// exposed as `dict` by Forgejo 15.0.7's modules/templates.NewFuncMap.
func forgejoDict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires key/value pairs")
	}
	result := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict key %d is not a string", i/2)
		}
		if key != "." {
			result[key] = values[i+1]
			continue
		}
		merged := reflect.ValueOf(values[i+1])
		if !merged.IsValid() {
			continue
		}
		if merged.Kind() != reflect.Map {
			return nil, fmt.Errorf("dict dot value %d is not a map", i/2)
		}
		for _, mergedKey := range merged.MapKeys() {
			result[mergedKey.String()] = merged.MapIndex(mergedKey).Interface()
		}
	}
	return result, nil
}

func TestForgejoPageIntroComposition(t *testing.T) {
	partial := readForgejoTemplate(t, "custom", "soda", "page_intro.tmpl")
	translations := map[string]string{
		"new_repo.title": "Créer & partager <ensemble>",
	}
	functions := template.FuncMap{
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx": func() forgejoTemplateContext {
			return forgejoTemplateContext{Locale: forgejoTemplateLocale{translations: translations}}
		},
		"dict": forgejoDict,
	}
	definition := `{{define "custom/soda/page_intro"}}` + partial + `{{end}}
		{{define "caller"}}{{template "custom/soda/page_intro" dict
			"TitleID" .TitleID
			"Eyebrow" .Eyebrow
			"Title" (ctx.Locale.Tr "new_repo.title")
			"Description" .Description
			"Artwork" .Artwork
			"Class" .Class}}{{end}}`
	parsed, err := template.New("components").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse shared page intro: %v", err)
	}

	data := map[string]string{
		"TitleID":     `intro" onclick="alert(1)`,
		"Eyebrow":     `<script>eyebrow</script>`,
		"Description": `Make & share <strong>things</strong>.`,
		"Artwork":     `diagram" onerror="alert(1).png`,
		"Class":       `compact" data-owned="true`,
	}
	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "caller", data); err != nil {
		t.Fatalf("execute shared page intro: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`Créer &amp; partager &lt;ensemble&gt;`,
		`&lt;script&gt;eyebrow&lt;/script&gt;`,
		`Make &amp; share &lt;strong&gt;things&lt;/strong&gt;.`,
		`src="/forge/assets/soda/forgejo/diagram%22%20onerror=%22alert%281%29.png"`,
		`soda-page-intro compact&#34; data-owned=&#34;true`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered intro does not contain escaped %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{`<script>`, `onclick="alert(1)"`, `onerror="alert(1)`} {
		if strings.Contains(output, forbidden) {
			t.Errorf("rendered intro contains active caller markup %q:\n%s", forbidden, output)
		}
	}

	var minimal bytes.Buffer
	if err := parsed.ExecuteTemplate(&minimal, "custom/soda/page_intro", map[string]any{
		"TitleID": "plain-title",
		"Title":   "Plain title",
	}); err != nil {
		t.Fatalf("execute minimal shared page intro: %v", err)
	}
	minimalOutput := minimal.String()
	for _, absent := range []string{"soda-page-eyebrow", "soda-page-description", "soda-page-art", "<img"} {
		if strings.Contains(minimalOutput, absent) {
			t.Errorf("minimal intro unexpectedly contains optional %q:\n%s", absent, minimalOutput)
		}
	}
}

func TestForgejoEmptyContentComposition(t *testing.T) {
	partial := readForgejoTemplate(t, "custom", "soda", "empty_content.tmpl")
	functions := template.FuncMap{
		"dict": forgejoDict,
		"svg": func(name string, _ int) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	definition := `{{define "custom/soda/empty_content"}}` + partial + `{{end}}
		{{define "caller"}}{{template "custom/soda/empty_content" dict
			"Icon" .Icon
			"Eyebrow" .Eyebrow
			"TitleID" .TitleID
			"Title" .Title
			"Description" .Description}}{{end}}`
	parsed, err := template.New("components").Funcs(functions).Parse(definition)
	if err != nil {
		t.Fatalf("parse shared empty content: %v", err)
	}

	var rendered bytes.Buffer
	if err := parsed.ExecuteTemplate(&rendered, "caller", map[string]string{
		"Icon":        "octicon-inbox",
		"Eyebrow":     "Nothing & nowhere",
		"TitleID":     `empty" data-owned="true`,
		"Title":       `<script>empty</script>`,
		"Description": `Try <strong>again</strong>.`,
	}); err != nil {
		t.Fatalf("execute shared empty content: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`data-icon="octicon-inbox"`,
		`Nothing &amp; nowhere`,
		`id="empty&#34; data-owned=&#34;true"`,
		`&lt;script&gt;empty&lt;/script&gt;`,
		`Try &lt;strong&gt;again&lt;/strong&gt;.`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered empty content does not contain escaped %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "<script>") {
		t.Errorf("rendered empty content contains active caller markup:\n%s", output)
	}

	var minimal bytes.Buffer
	if err := parsed.ExecuteTemplate(&minimal, "custom/soda/empty_content", map[string]any{
		"Title": "No results",
	}); err != nil {
		t.Fatalf("execute minimal shared empty content: %v", err)
	}
	minimalOutput := minimal.String()
	for _, absent := range []string{"soda-empty-symbol", "soda-empty-eyebrow", "soda-empty-description", " id="} {
		if strings.Contains(minimalOutput, absent) {
			t.Errorf("minimal empty content unexpectedly contains optional %q:\n%s", absent, minimalOutput)
		}
	}
}

func TestForgejoPagesComposeSharedPresentationWithNativeBoundaries(t *testing.T) {
	introPages := []string{
		"explore/repos.tmpl",
		"explore/users.tmpl",
		"org/create.tmpl",
		"repo/create.tmpl",
		"user/dashboard/dashboard.tmpl",
		"user/dashboard/issues.tmpl",
		"user/dashboard/milestones.tmpl",
		"user/notification/notification_div.tmpl",
		"user/notification/notification_subscriptions.tmpl",
	}
	for _, name := range introPages {
		t.Run("intro/"+name, func(t *testing.T) {
			requireForgejoTemplateCalls(t, name, "custom/soda/page_intro")
		})
	}

	boundaries := []struct {
		name  string
		calls []string
	}{
		{
			name:  "user/auth/signin.tmpl",
			calls: []string{"user/auth/signin_inner"},
		},
		{
			name: "repo/create.tmpl",
			calls: []string{
				"base/alert",
				"repo/create_helper",
				"repo/create_basic",
				"repo/create_from_template",
				"repo/create_init",
				"repo/create_advanced",
			},
		},
		{
			name: "user/dashboard/dashboard.tmpl",
			calls: []string{
				"base/alert",
				"user/dashboard/navbar",
				"user/heatmap",
				"user/dashboard/feeds",
				"user/dashboard/guide",
				"user/dashboard/repolist",
			},
		},
		{
			name: "user/dashboard/issues.tmpl",
			calls: []string{
				"base/alert",
				"shared/search/combo",
				"shared/search/issue/syntax",
				"user/dashboard/navbar",
				"shared/issuelist",
			},
		},
		{
			name: "user/notification/notification_subscriptions.tmpl",
			calls: []string{
				"shared/issuelist",
				"shared/repo_search",
				"explore/repo_list",
				"base/paginate",
			},
		},
	}
	for _, boundary := range boundaries {
		t.Run("native/"+boundary.name, func(t *testing.T) {
			requireForgejoTemplateCalls(t, boundary.name, boundary.calls...)
		})
	}

	repoCreate := readForgejoTemplate(t, "repo", "create.tmpl")
	for _, gate := range []string{".CanCreateRepo", ".Orgs", ".MaxCreationLimit"} {
		if !strings.Contains(repoCreate, gate) {
			t.Errorf("repo/create.tmpl lost native creation gate %s", gate)
		}
	}
	orgCreate := readForgejoTemplate(t, "org", "create.tmpl")
	for _, gate := range []string{".Err_OrgName", ".Err_OrgVisibility", ".visibility", ".repo_admin_change_team_access"} {
		if !strings.Contains(orgCreate, gate) {
			t.Errorf("org/create.tmpl lost native form state %s", gate)
		}
	}
	for _, name := range []string{"repo/create.tmpl", "org/create.tmpl"} {
		contents := readForgejoTemplate(t, strings.Split(name, "/")...)
		if !strings.Contains(contents, `action="{{.Link}}" method="post"`) {
			t.Errorf("%s lost its native POST target", name)
		}
	}

	notifications := readForgejoTemplate(t, "user", "notification", "notification_div.tmpl")
	for _, marker := range []string{`id="notification_div"`, `data-sequence-number="{{.SequenceNumber}}"`} {
		if !strings.Contains(notifications, marker) {
			t.Errorf("notification replacement fragment lost %s", marker)
		}
	}
	if templateCalls(notifications)["base/head"] || templateCalls(notifications)["base/footer"] {
		t.Error("notification_div.tmpl must remain a fragment for native notification swaps")
	}
}

func readForgejoTemplate(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "appliance", "forgejo", "templates"}, parts...)...)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

var forgejoTemplateCallPattern = regexp.MustCompile(`\{\{\s*template\s+"([^"]+)"`)

func templateCalls(contents string) map[string]bool {
	calls := make(map[string]bool)
	for _, match := range forgejoTemplateCallPattern.FindAllStringSubmatch(contents, -1) {
		calls[match[1]] = true
	}
	return calls
}

func requireForgejoTemplateCalls(t *testing.T, name string, required ...string) {
	t.Helper()
	contents := readForgejoTemplate(t, strings.Split(name, "/")...)
	calls := templateCalls(contents)
	for _, requiredName := range required {
		if !calls[requiredName] {
			t.Errorf("%s no longer composes native template %q", name, requiredName)
		}
	}
}
