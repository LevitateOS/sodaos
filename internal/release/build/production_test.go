package build

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureLiveInputs builds a valid live-inputs file: a well-formed stable
// CoreOS section plus the floating Tailnet toolchain under test.
func fixtureLiveInputs() LiveInputs {
	img := CoreOSImage{
		URL:                "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso",
		SignatureURL:       "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso.sig",
		SHA256:             strings.Repeat("a", 64),
		UncompressedSHA256: strings.Repeat("b", 64),
	}
	return LiveInputs{
		CoreOS: ResolvedCoreOS{
			Release:     "44.20260901.1.0",
			MetadataURL: "https://builds.test/prod/streams/stable/builds/44.20260901.1.0/release.json",
			Container: map[string]string{
				"x86_64": "quay.io/fedora/fedora-coreos@sha256:" + strings.Repeat("c", 64),
			},
			ISO:  map[string]CoreOSImage{"x86_64": img},
			QEMU: map[string]CoreOSImage{"x86_64": img},
		},
		Tailnet: TailnetInputs{
			Version: "1.98.2",
			SHA256:  strings.Repeat("e", 64),
			Base:    "docker.io/tailscale/alpine-base:3.22",
		},
	}
}

func productionFixture(t *testing.T) (Production, *[]string) {
	t.Helper()
	root := t.TempDir()
	out := filepath.Join(root, ".artifacts/native/x86_64")
	if e := os.MkdirAll(out, 0o755); e != nil {
		t.Fatal(e)
	}
	files := map[string]string{
		"package.json":                            `{"packageManager":"bun@1.4.2","unrelated":true}`,
		"project-os/Containerfile":                "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
		"appliance/dashboard.Containerfile":       "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
		"appliance/services/forgejo.container":    "[Container]\nImage=codeberg.org/forgejo/forgejo:15.0.9\n",
		"appliance/services/soda-proxy.container": "[Container]\nImage=docker.io/library/caddy:2\n",
	}
	for n, b := range files {
		path := filepath.Join(root, n)
		if e := os.MkdirAll(filepath.Dir(path), 0o755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(path, []byte(b), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	seed := filepath.Join(root, "seed.oci")
	fixtureOCI(t, seed, "amd64", false)
	image, e := InspectOCI(seed, "x86_64", fixtureRevision)
	if e != nil {
		t.Fatal(e)
	}
	bytes, e := os.ReadFile(seed)
	if e != nil {
		t.Fatal(e)
	}
	livePath := filepath.Join(root, "live-inputs.json")
	if e := WriteLiveInputs(livePath, fixtureLiveInputs()); e != nil {
		t.Fatal(e)
	}
	var calls []string
	p := Production{Source: root, Native: out, Out: out, Arch: "x86_64", Revision: fixtureRevision, LiveInputs: livePath}
	p.Next = func(s string) error { calls = append(calls, "STEP "+s); return nil }
	p.Capture = func(dir, name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "bun" {
			return "1.4.2", nil
		}
		if name != "podman" || args[0] != "--remote=false" {
			t.Fatalf("unexpected observation: %s %v", name, args)
		}
		if args[1] == "pull" {
			return image.Config, nil
		}
		if args[1] == "image" {
			return "sha256:" + strings.Repeat("b", 64), nil
		}
		t.Fatalf("unexpected observation: %v", args)
		return "", nil
	}
	writeELF := func(dest string) error {
		header := make([]byte, 64)
		copy(header, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
		binary.LittleEndian.PutUint16(header[16:], 2)
		binary.LittleEndian.PutUint16(header[18:], 62)
		binary.LittleEndian.PutUint32(header[20:], 1)
		binary.LittleEndian.PutUint16(header[52:], 64)
		if e := os.MkdirAll(filepath.Dir(dest), 0o755); e != nil {
			return e
		}
		return os.WriteFile(dest, header, 0o700)
	}
	p.Execute = func(dir, name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "go" && args[0] == "build" {
			for i, a := range args {
				if a == "-o" {
					return writeELF(args[i+1])
				}
			}
		}
		if name == "cargo" && args[0] == "build" {
			for i, a := range args {
				if a == "-p" && i+1 < len(args) {
					bin := args[i+1]
					if renamed, ok := map[string]string{"soda-project-terminal": "project-terminal", "soda-project-account": "project-account"}[bin]; ok {
						bin = renamed
					}
					return writeELF(filepath.Join(dir, "target", "release", bin))
				}
			}
		}
		if name != "podman" {
			return nil
		}
		if args[0] != "--remote=false" {
			t.Fatal("remote engine inherited")
		}
		for i, a := range args {
			if a == "--iidfile" {
				return WriteNew(args[i+1], []byte(image.Config), 0o600)
			}
			if a == "--output" {
				return WriteNew(args[i+1], bytes, 0o600)
			}
		}
		t.Fatalf("unexpected execution: %v", args)
		return nil
	}
	return p, &calls
}

func TestProductionUsesOneAssetAndImageSequence(t *testing.T) {
	p, calls := productionFixture(t)
	host := filepath.Join(p.Out, "context")
	forgejo := filepath.Join(p.Out, "forgejo-context")
	if e := p.Dependencies(); e != nil {
		t.Fatal(e)
	}
	if e := p.ResolveInputs(); e != nil {
		t.Fatal(e)
	}
	if e := p.Assets(host, forgejo); e != nil {
		t.Fatal(e)
	}
	for _, tool := range []string{"muse", "soda-identity-compose", "project-terminal", "project-account"} {
		info, err := os.Stat(filepath.Join(p.Native, "project-tools/bin", tool))
		if err != nil || info.Mode().Perm() != 0o755 {
			t.Fatalf("public tool %s must be executable by project accounts: %v", tool, err)
		}
	}
	images, e := p.Images(forgejo)
	if e != nil {
		t.Fatal(e)
	}
	if len(images) != 6 {
		t.Fatal(images)
	}
	text := strings.Join(*calls, "\n")
	for _, needle := range []string{"bun install --frozen-lockfile", "bun scripts/build-forgejo.ts", "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-terminal -- --out ", "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-muse -- --arch x86_64 --out ", "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-tea -- --arch x86_64 --out ", "cargo run --release --locked -p soda-forgejo-locales --bin soda-forgejo-locales -- --lock appliance/forgejo/locale.lock.json --out ", "cargo run --release --locked -p soda-stage-render --bin soda-stage -- --arch x86_64 --host-context ", "STEP Build image: dashboard\n", "STEP Build image: project-os\n", "STEP Build image: tailnet\n", "STEP Build image: forgejo\n", "bun scripts/build-soda-extension.ts --out "} {
		if strings.Count(text, needle) != 1 {
			t.Fatalf("not produced exactly once: %s\n%s", needle, text)
		}
	}
	if strings.Contains(text, "python3 scripts/fetch-") || strings.Contains(text, "tools/soda-fetch-muse") || strings.Contains(text, "python3 scripts/stage.py") || strings.Contains(text, "python3 scripts/forgejo-locales.py") {
		t.Fatal("retired fetcher/stage/locales invocation retained", text)
	}
	if !strings.Contains(text, "--host-context "+host+" --forgejo-context "+forgejo) {
		t.Fatal("asset destinations do not match the selected layout", text)
	}
	if strings.Count(text, " save --format=oci-archive") != 6 {
		t.Fatal(text)
	}
	if strings.Contains(text, " push ") || strings.Contains(text, " --rm ") || strings.Contains(text, "--replace") {
		t.Fatal("unapproved publication/cleanup")
	}
	if _, ok := images["proxy"]; !ok {
		t.Fatal("proxy identity lost")
	}
	if _, ok := images["caddy"]; ok {
		t.Fatal("retired proxy filename retained")
	}
	for name, im := range images {
		if !Digest(im.ArchiveSHA256) || im.Manifest == "" || im.Config == "" {
			t.Fatal(im)
		}
		if _, e := os.Stat(filepath.Join(p.Out, "images", name+".oci")); e != nil {
			t.Fatal(e)
		}
	}
	before := len(*calls)
	if _, e = p.Images(forgejo); e == nil || len(*calls) != before {
		t.Fatal("replayed production over retained outputs")
	}
}

func TestProductionFailureStopsBeforeLaterImages(t *testing.T) {
	p, calls := productionFixture(t)
	execute := p.Execute
	sentinel := errors.New("fixture build refused")
	p.Execute = func(dir, name string, args ...string) error {
		if name == "podman" && strings.Contains(strings.Join(args, " "), "dashboard.Containerfile") {
			return sentinel
		}
		return execute(dir, name, args...)
	}
	if e := p.ResolveInputs(); e != nil {
		t.Fatal(e)
	}
	if _, e := p.Images("forgejo-context"); !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	text := strings.Join(*calls, "\n")
	if strings.Contains(text, "STEP Build image: project-os") || strings.Contains(text, "STEP Export and verify") {
		t.Fatal("continued after failed image")
	}
	if _, e := os.Stat(filepath.Join(p.Out, "app-inputs.json")); e != nil {
		t.Fatal("failed attempt evidence lost")
	}
}

func TestProductionRefusesWrongToolchainInputsAndLayout(t *testing.T) {
	for _, mode := range []string{"bun", "base", "tailnet", "forgejo", "platform", "revision"} {
		t.Run(mode, func(t *testing.T) {
			p, _ := productionFixture(t)
			switch mode {
			case "bun":
				p.Capture = func(string, string, ...string) (string, error) { return "different", nil }
				if e := p.Dependencies(); e == nil {
					t.Fatal("wrong Bun accepted")
				}
				return
			case "base":
				if err := os.WriteFile(filepath.Join(p.Source, "appliance/dashboard.Containerfile"), []byte("ARG BASE_IMAGE=unrelated\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "tailnet":
				raw, e := json.Marshal(fixtureLiveInputs())
				if e != nil {
					t.Fatal(e)
				}
				raw = []byte(strings.Replace(string(raw), `"Version":"1.98.2"`, `"Version":"yesterday"`, 1))
				if e := os.WriteFile(filepath.Join(p.Source, "live-inputs.json"), raw, 0o644); e != nil {
					t.Fatal(e)
				}
			case "forgejo":
				if _, e := p.Images(""); e == nil {
					t.Fatal("unstaged Forgejo entered host payload")
				}
				return
			case "platform":
				p.Arch = "other"
			case "revision":
				p.Revision = "dirty"
			}
			if _, e := p.Images("forgejo-context"); e == nil {
				t.Fatal("invalid inputs accepted")
			}
		})
	}
}

func TestProductionAssetDestinationsRefuseBeforeCommands(t *testing.T) {
	for _, destinations := range [][2]string{{"/host-context", ""}, {"", "/forgejo-context"}, {"relative-host", "/forgejo-context"}, {"/host-context", "relative-forgejo"}} {
		p, calls := productionFixture(t)
		if err := p.Assets(destinations[0], destinations[1]); err == nil || len(*calls) != 0 {
			t.Fatal("invalid asset layout had downstream effects", err, *calls)
		}
	}
}

func TestProductionCompileKeepsLayoutAndVerifiesELF(t *testing.T) {
	p, calls := productionFixture(t)
	dest := filepath.Join(p.Out, "program")
	if e := p.Compile("soda-host", "./cmd/soda-host", dest); e != nil {
		t.Fatal(e)
	}
	text := strings.Join(*calls, "\n")
	if !strings.Contains(text, "-buildvcs=false") || strings.Contains(text, "-tags=") || strings.Contains(text, "-buildvcs=true") {
		t.Fatal(text)
	}
	if strings.Count(text, "go build") != 1 {
		t.Fatal("duplicate compilation")
	}
	p.Execute = func(string, string, ...string) error { return os.WriteFile(dest, []byte("not ELF"), 0o755) }
	if e := p.Compile("soda-host", "./cmd/soda-host", dest); e == nil {
		t.Fatal("invalid program accepted")
	}
}

func TestProductionCompileRustKeepsLayoutAndVerifiesELF(t *testing.T) {
	p, calls := productionFixture(t)
	dest := filepath.Join(p.Out, "soda-identity-compose")
	if e := p.CompileRust("soda-identity-compose", "soda-identity-compose", dest); e != nil {
		t.Fatal(e)
	}
	text := strings.Join(*calls, "\n")
	if !strings.Contains(text, "cargo build --release --locked") || !strings.Contains(text, "-p soda-identity-compose") {
		t.Fatal(text)
	}
	if strings.Count(text, "cargo build") != 1 {
		t.Fatal("duplicate compilation")
	}
	for _, bad := range [][2]string{{"", "bin"}, {"crate", ""}, {"a/b", "bin"}, {"crate", "a/bin"}} {
		if e := p.CompileRust(bad[0], bad[1], dest); e == nil {
			t.Fatal("invalid Rust selection accepted", bad)
		}
	}
	p.Execute = func(string, string, ...string) error { return nil }
	if e := p.CompileRust("missing-crate", "missing-crate", dest); e == nil {
		t.Fatal("missing Rust binary accepted")
	}
}

func TestSodaCommandsRefusesSupportToolsInRuntime(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"soda-host", "soda-dashboard"} {
		if e := os.MkdirAll(filepath.Join(root, "cmd", name), 0o755); e != nil {
			t.Fatal(e)
		}
	}
	names, e := SodaCommands(root)
	if e != nil || strings.Join(names, ",") != "soda-dashboard,soda-host" {
		t.Fatal(names, e)
	}
	if err := os.Mkdir(filepath.Join(root, "cmd/soda-artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, e = SodaCommands(root); e == nil {
		t.Fatal("support tool admitted")
	}
}
