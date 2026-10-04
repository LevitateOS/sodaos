// System-format pins for the language port: dashboard configuration schema,
// host daemon configuration bytes, systemd wiring, installed helper paths,
// and Tier-1 CLI surfaces that Rust replacements must preserve. Fixtures
// under fixtures/portcontracts are frozen shared contracts; port lanes may
// change this harness to invoke a new binary but must keep the fixture bytes
// identical.
package scripts

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
)

// buildGoPortBinary builds a not-yet-ported Go CLI. Ported binaries use
// buildRustPortBinary (Rust) from the wire-contracts harness instead.
func buildGoPortBinary(t *testing.T, pkg string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "soda-format-probe")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, pkg)
	cmd.Dir = root
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, combined)
	}
	return out
}

type configVector struct {
	Name          string         `json:"name"`
	Object        map[string]any `json:"object,omitempty"`
	Raw           string         `json:"raw,omitempty"`
	OK            bool           `json:"ok"`
	ErrorContains string         `json:"error_contains,omitempty"`
	Expect        map[string]any `json:"expect,omitempty"`
}

func TestDashboardConfigVectors(t *testing.T) {
	var fixture struct {
		Vectors []configVector `json:"vectors"`
	}
	portContractFixture(t, "dashboard_config_vectors.json", &fixture)
	if len(fixture.Vectors) == 0 {
		t.Fatal("dashboard config fixture is empty")
	}
	for _, v := range fixture.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			var contents []byte
			if v.Raw != "" || v.Object == nil {
				contents = []byte(v.Raw)
			} else {
				var err error
				contents, err = json.Marshal(v.Object)
				if err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(t.TempDir(), "dashboard.json")
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			loaded, err := config.Load(path)
			if v.OK {
				if err != nil {
					t.Fatalf("valid config rejected: %v", err)
				}
				round, err := json.Marshal(loaded)
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				if err := json.Unmarshal(round, &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, v.Expect) {
					want, _ := json.Marshal(v.Expect)
					t.Fatalf("loaded %s, want %s", round, want)
				}
				return
			}
			if err == nil {
				t.Fatalf("invalid config accepted: %s", contents)
			}
			if v.ErrorContains != "" && !strings.Contains(err.Error(), v.ErrorContains) {
				t.Fatalf("error %q lacks %q", err, v.ErrorContains)
			}
		})
	}
}

func TestDashboardConfigLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 70000)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(path)
	if err == nil || !strings.Contains(err.Error(), "configuration exceeds 64 KiB") {
		t.Fatalf("oversized config not refused: %v", err)
	}
}

func TestHostConfigBytes(t *testing.T) {
	var fixture struct {
		Full        map[string]any `json:"full"`
		FullJSON    string         `json:"full_json"`
		Minimal     map[string]any `json:"minimal"`
		MinimalJSON string         `json:"minimal_json"`
	}
	portContractFixture(t, "host_config_bytes.json", &fixture)
	for name, tc := range map[string]struct {
		object map[string]any
		want   string
	}{
		"full":    {object: fixture.Full, want: fixture.FullJSON},
		"minimal": {object: fixture.Minimal, want: fixture.MinimalJSON},
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(tc.object)
			if err != nil {
				t.Fatal(err)
			}
			var decoded host.Config
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("bytes %s, want %s", encoded, tc.want)
			}
		})
	}
}

func TestSystemdWiring(t *testing.T) {
	var fixture struct {
		Units []struct {
			File  string   `json:"file"`
			Lines []string `json:"lines"`
		} `json:"units"`
		InstalledPaths []string `json:"installed_paths"`
	}
	portContractFixture(t, "systemd_wiring.json", &fixture)
	if len(fixture.Units) == 0 {
		t.Fatal("systemd wiring fixture is empty")
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range fixture.Units {
		t.Run(unit.File, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, unit.File))
			if err != nil {
				t.Fatal(err)
			}
			present := map[string]bool{}
			for _, line := range strings.Split(string(raw), "\n") {
				present[strings.TrimRight(line, "\r")] = true
			}
			for _, want := range unit.Lines {
				if !present[want] {
					t.Fatalf("%s lacks line %q", unit.File, want)
				}
			}
		})
	}
	for _, path := range fixture.InstalledPaths {
		t.Run(path, func(t *testing.T) {
			info, err := os.Stat(filepath.Join(root, path))
			if err != nil {
				t.Fatal(err)
			}
			if info.IsDir() {
				t.Fatalf("%s is not a file", path)
			}
		})
	}
}

// portedSourcePin resolves a retired Go source pin to its Rust replacement.
// The fixture keeps the original Go lines; the port must carry the same
// flag names, defaults, and usage text, so each pinned line contributes its
// quoted fragments as required substrings of the Rust source.
func portedSourcePin(file string, lines []string) (string, []string) {
	if file != "cmd/soda-setup/main.go" {
		return file, lines
	}
	var wants []string
	for _, line := range lines {
		for _, fragment := range strings.Split(line, "\"") {
			if fragment == "" || strings.ContainsAny(fragment, "(),") || strings.HasPrefix(fragment, "flag.") {
				continue
			}
			wants = append(wants, fragment)
		}
	}
	return "rust/soda-setup/src/main.rs", wants
}

func TestCLISurface(t *testing.T) {
	var fixture struct {
		Binaries []struct {
			Name           string   `json:"name"`
			Package        string   `json:"package"`
			Argv           []string `json:"argv"`
			Exit           int      `json:"exit"`
			StdoutContains []string `json:"stdout_contains,omitempty"`
			StderrContains []string `json:"stderr_contains,omitempty"`
			StderrExact    string   `json:"stderr_exact,omitempty"`
			When           string   `json:"when,omitempty"`
		} `json:"binaries"`
		SourceFlags []struct {
			File  string   `json:"file"`
			Lines []string `json:"lines"`
		} `json:"source_flags"`
	}
	portContractFixture(t, "cli_surface.json", &fixture)
	if len(fixture.Binaries) == 0 {
		t.Fatal("CLI surface fixture is empty")
	}
	built := map[string]string{}
	for _, b := range fixture.Binaries {
		t.Run(b.Name, func(t *testing.T) {
			if b.When == "nonroot" && os.Geteuid() == 0 {
				t.Skip("privileged refusal differs for root")
			}
			binary, ok := built[b.Package]
			if !ok {
				if strings.HasPrefix(b.Package, "rust/") {
					binary = buildRustPortBinary(t, strings.TrimPrefix(b.Package, "rust/"))
				} else if b.Package == "./cmd/soda-setup" {
					// PR07 ported setup to Rust; the fixture stays frozen.
					binary = buildRustPortBinary(t, "soda-setup")
				} else if b.Package == "./cmd/soda-identity-compose" {
					// PR10 ported identity-compose to Rust; the fixture stays frozen.
					binary = buildRustPortBinary(t, "soda-identity-compose")
				} else {
					binary = buildGoPortBinary(t, b.Package)
				}
				built[b.Package] = binary
			}
			stdout, stderr, code := runPortBinary(t, binary, b.Argv...)
			if code != b.Exit {
				t.Fatalf("exit %d, want %d: stdout %q stderr %q", code, b.Exit, stdout, stderr)
			}
			for _, want := range b.StdoutContains {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout %q lacks %q", stdout, want)
				}
			}
			for _, want := range b.StderrContains {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr %q lacks %q", stderr, want)
				}
			}
			if b.StderrExact != "" && stderr != b.StderrExact {
				t.Fatalf("stderr %q, want %q", stderr, b.StderrExact)
			}
		})
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, pinned := range fixture.SourceFlags {
		t.Run(pinned.File, func(t *testing.T) {
			file, lines := portedSourcePin(pinned.File, pinned.Lines)
			raw, err := os.ReadFile(filepath.Join(root, file))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range lines {
				if !strings.Contains(string(raw), want) {
					t.Fatalf("%s lacks %q", file, want)
				}
			}
		})
	}
}
