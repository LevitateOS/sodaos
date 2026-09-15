// Package archcheck guards the internal package topology: which directories
// may exist and which internal dependencies may cross ownership boundaries.
// It parses only production (non-test) imports, so test fakes may still use
// whatever they need. See docs/go.md and docs/go-packages.md for the rules.
package archcheck

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/levitateos/sodaos"

// retired lists top-level internal packages removed by the architectural
// rewrite. Recreating any of them is a regression, not a shortcut.
var retired = []string{
	"internal/projectos",
	"internal/linuxhost",
	"internal/installlayout",
	"internal/hostproject",
	"internal/hostterminal",
	"internal/hosttailnet",
	"internal/nativebuild",
	"internal/nativequalification",
	"internal/nativefinalization",
	"internal/releasedelivery",
	"internal/appliancerelease",
	"internal/webapp",
	"internal/webauth",
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above working directory")
		}
		dir = parent
	}
}

// productionImports maps internal package path -> internal imports used by
// non-test Go files in that directory.
func productionImports(t *testing.T, root string) map[string][]string {
	t.Helper()
	edges := map[string][]string{}
	roots := []string{"internal", "cmd", "tools"}
	for _, base := range roots {
		entries, err := os.ReadDir(filepath.Join(root, base))
		if err != nil {
			continue
		}
		var dirs []string
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(base, e.Name()))
			}
		}
		// Nested subpackages (host/*, release/*, web/*).
		for _, d := range append([]string{}, dirs...) {
			sub, err := os.ReadDir(filepath.Join(root, d))
			if err != nil {
				continue
			}
			for _, s := range sub {
				if s.IsDir() {
					dirs = append(dirs, filepath.Join(d, s.Name()))
				}
			}
		}
		for _, dir := range dirs {
			files, err := os.ReadDir(filepath.Join(root, dir))
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, f := range files {
				name := f.Name()
				if f.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
					continue
				}
				pf, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, dir, name), nil, parser.ImportsOnly)
				if err != nil {
					t.Fatal(err)
				}
				for _, imp := range pf.Imports {
					path, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						t.Fatal(err)
					}
					if rest, ok := strings.CutPrefix(path, modulePath+"/internal/"); ok {
						seen["internal/"+rest] = true
					}
				}
			}
			if len(seen) > 0 {
				pkg := modulePath + "/" + filepath.ToSlash(dir)
				for imp := range seen {
					edges[pkg] = append(edges[pkg], imp)
				}
			}
		}
	}
	return edges
}

func importsOf(edges map[string][]string, pkg string) []string {
	return edges[modulePath+"/"+pkg]
}

func forbid(t *testing.T, edges map[string][]string, pkg string, banned ...string) {
	t.Helper()
	for _, imp := range importsOf(edges, pkg) {
		for _, b := range banned {
			if imp == b || strings.HasPrefix(imp, b+"/") {
				t.Errorf("%s must not import %s (banned dependency)", pkg, imp)
			}
		}
	}
}

func allowOnly(t *testing.T, edges map[string][]string, pkg string, allowed ...string) {
	t.Helper()
loop:
	for _, imp := range importsOf(edges, pkg) {
		for _, a := range allowed {
			if imp == a {
				continue loop
			}
		}
		t.Errorf("%s must not import %s (allowed: %v)", pkg, imp, allowed)
	}
}

func TestRetiredPackagesStayDeleted(t *testing.T) {
	root := moduleRoot(t)
	for _, dir := range retired {
		if _, err := os.Stat(filepath.Join(root, dir)); !os.IsNotExist(err) {
			t.Errorf("retired package %s was recreated", dir)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "internal/web/aliases.go")); !os.IsNotExist(err) {
		t.Errorf("internal/web/aliases.go facade was recreated")
	}
}

func TestDependencyDirection(t *testing.T) {
	edges := productionImports(t, moduleRoot(t))

	// Pure domain: project validates; store persists. Neither reaches
	// transport, privilege or release machinery.
	allowOnly(t, edges, "internal/project", "internal/strictjson")
	allowOnly(t, edges, "internal/store", "internal/project")

	// Dashboard transport never executes privilege or builds releases
	// directly; it goes through the host client and domain types.
	for _, pkg := range []string{"internal/web", "internal/web/api", "internal/web/auth"} {
		forbid(t, edges, pkg,
			"internal/host/project", "internal/host/terminal", "internal/host/tailnet",
			"internal/release")
	}

	// Release construction never calls back into HTTP transport or the
	// privileged daemon; image assembly shells out instead of importing.
	for _, pkg := range []string{"internal/release/build", "internal/release/image", "internal/release/qualify", "internal/release/deliver"} {
		forbid(t, edges, pkg, "internal/web", "internal/host")
	}

	// Privileged executors stay leaves: no upward import of the daemon,
	// no outward reach into transport, storage or release. The single
	// exception is host/project -> host/terminal: key operations attach
	// through the terminal executor.
	for _, pkg := range []string{"internal/host/project", "internal/host/terminal", "internal/host/tailnet"} {
		forbid(t, edges, pkg, "internal/web", "internal/release", "internal/store")
		for _, imp := range importsOf(edges, pkg) {
			if imp == "internal/host" {
				t.Errorf("%s must not import the host daemon", pkg)
			}
			if imp != "internal/host/terminal" && strings.HasPrefix(imp, "internal/host/") {
				t.Errorf("%s must not import sibling executor %s", pkg, imp)
			}
		}
	}

	// Leaf capabilities never reach up.
	forbid(t, edges, "internal/runners", "internal/web", "internal/host", "internal/store", "internal/release")
	forbid(t, edges, "internal/tailnet", "internal/web", "internal/host", "internal/store", "internal/release", "internal/runners")

	// The dashboard binary wires the web facade, never the privileged
	// daemon directly.
	forbid(t, edges, "cmd/soda-dashboard", "internal/host")
}
