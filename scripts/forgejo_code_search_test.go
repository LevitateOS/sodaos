package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoCodeSearchStylesHaveOneScopedOwner(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "assets", "branding", "forgejo", "code-search.css"))
	if err != nil {
		t.Fatalf("read code-search styles: %v", err)
	}
	css := string(contents)
	for _, want := range []string{".soda-explore-code", ".soda-code-search", ".soda-code-search > .ui.form", ".soda-code-search > .ui.user.list"} {
		if !strings.Contains(css, want) {
			t.Errorf("code-search stylesheet lacks scoped owner %q", want)
		}
	}
	for _, forbidden := range []string{"body {", "#navbar", ".soda-packages", ".soda-project"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("code-search stylesheet crosses its family boundary with %q", forbidden)
		}
	}
	packages, err := os.ReadFile(filepath.Join("..", "assets", "branding", "forgejo", "packages.css"))
	if err != nil {
		t.Fatalf("read package styles: %v", err)
	}
	if strings.Contains(string(packages), ".soda-code-search") {
		t.Error("package stylesheet still owns code-search selectors")
	}
}
