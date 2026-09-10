package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type forgejoSettingsRepository struct {
	Code, IsEmpty bool
	ID            int64
}

func (r forgejoSettingsRepository) UnitEnabled(_, _ any) bool { return r.Code }

type forgejoSettingsPermission bool

func (p forgejoSettingsPermission) CanRead(any) bool { return bool(p) }

func TestForgejoRepositorySettingsNavigationMatchesNativeGates(t *testing.T) {
	root := os.Getenv("SODA_FORGEJO_TEMPLATES")
	if root == "" {
		root = "../.artifacts/forgejo-presentation/upstream/templates"
	}
	native, err := os.ReadFile(filepath.Join(root, "repo/settings/navbar.tmpl"))
	if os.IsNotExist(err) {
		t.Skip("requires the retained exact 15.0.7 template export")
	}
	if err != nil {
		t.Fatal(err)
	}
	candidate := readForgejoTemplate(t, "repo/settings/navbar.tmpl")
	disabledHooks := false
	funcs := template.FuncMap{
		"AppSubUrl":       func() string { return "" },
		"ctx":             func() any { return struct{ Locale forgejoSettingsFixtureLocale }{} },
		"svg":             func(string, ...any) string { return "" },
		"DisableWebhooks": func() bool { return disabledHooks },
	}
	old, err := template.New("native").Funcs(funcs).Parse(string(native))
	if err != nil {
		t.Fatal(err)
	}
	current, err := template.New("current").Funcs(funcs).Parse(candidate)
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile(`href="([^"]+)"`)
	render := func(tpl *template.Template, data map[string]any) []string {
		var out bytes.Buffer
		if err := tpl.Execute(&out, data); err != nil {
			t.Fatal(err)
		}
		const sodaLink = "/-/soda/repositories/9223372036854775807/settings/spaces"
		if tpl.Name() == "current" && strings.Count(out.String(), `href="`+sodaLink+`"`) != 1 {
			t.Fatal("missing/duplicate bounded Soda settings link")
		}
		result := []string{}
		for _, m := range links.FindAllStringSubmatch(out.String(), -1) {
			if m[1] == sodaLink {
				continue
			} // Only the deliberately added Soda destination differs.
			result = append(result, m[1])
		}
		slices.Sort(result)
		return result
	}
	// Independently vary every native gate, including Actions permission and
	// global disablement. Rendering menus never grants backend authority.
	for mask := 0; mask < 256; mask++ {
		flag := func(bit int) bool { return mask&(1<<bit) != 0 }
		disabledHooks = flag(2)
		data := map[string]any{
			"RepoLink": "/fixture/repository", "Title": "Settings", "Context": nil, "IsSigned": true,
			"UnitTypeCode": 1, "UnitTypeActions": 2,
			"Repository":     forgejoSettingsRepository{Code: flag(0), IsEmpty: flag(1), ID: 9223372036854775807},
			"SignedUser":     map[string]any{"CanEditGitHook": flag(3)},
			"LFSStartServer": flag(4), "EnableActions": flag(5),
			"UnitActionsGlobalDisabled": flag(6), "Permission": forgejoSettingsPermission(flag(7)),
			"PageIsSettingsOptions": true,
		}
		if want, got := render(old, data), render(current, data); !slices.Equal(want, got) {
			t.Fatalf("native visibility changed for gate matrix %08b: want %v, got %v", mask, want, got)
		}
	}
	if strings.Contains(candidate, "flex-container-nav") || strings.Contains(candidate, ` hidden`) {
		t.Fatal("navigation must remain expanded without enhancement")
	}
}
