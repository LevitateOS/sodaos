package scripts

import (
	"bytes"
	"html/template"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type forgejoSettingsRepository struct{ Code, IsEmpty bool }

func (r forgejoSettingsRepository) UnitEnabled(_, _ any) bool { return r.Code }

type forgejoSettingsPermission bool

func (p forgejoSettingsPermission) CanRead(any) bool { return bool(p) }

func TestForgejoRepositorySettingsNavigationMatchesNativeGates(t *testing.T) {
	native, err := os.ReadFile("../.artifacts/forgejo-presentation/upstream/templates/repo/settings/navbar.tmpl")
	if os.IsNotExist(err) {
		t.Skip("requires the retained exact 15.0.7 template export")
	}
	if err != nil {
		t.Fatal(err)
	}
	candidate := readForgejoTemplate(t, "repo/settings/navbar.tmpl")
	disabledHooks := false
	funcs := template.FuncMap{
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
		result := []string{}
		for _, m := range links.FindAllStringSubmatch(out.String(), -1) {
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
			"RepoLink": "/fixture/repository", "Title": "Settings", "Context": nil,
			"UnitTypeCode": 1, "UnitTypeActions": 2,
			"Repository":     forgejoSettingsRepository{Code: flag(0), IsEmpty: flag(1)},
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
