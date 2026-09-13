package nativebuild

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func productionFixture(t *testing.T, vendor bool) (Production, *[]string) {
	t.Helper()
	root := t.TempDir()
	out := filepath.Join(root, ".artifacts/native/x86_64")
	if e := os.MkdirAll(out, 0755); e != nil {
		t.Fatal(e)
	}
	files := map[string]string{
		"package.json":                            `{"packageManager":"bun@1.4.2","unrelated":true}`,
		"project-os/Containerfile":                "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
		"appliance/dashboard.Containerfile":       "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
		"appliance/services/forgejo.container":    "[Container]\nImage=codeberg.org/forgejo/forgejo:15.0.7\n",
		"appliance/services/soda-proxy.container": "[Container]\nImage=docker.io/library/caddy:2\n",
		"appliance/locks/tailscale-image.json":    `{"Version":"1.98.2","Base":"docker.io/tailscale/alpine-base@sha256:` + strings.Repeat("b", 64) + `","SHA256":{"amd64":"` + strings.Repeat("c", 64) + `"}}`,
	}
	for n, b := range files {
		path := filepath.Join(root, n)
		if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(path, []byte(b), 0644); e != nil {
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
	var calls []string
	p := Production{Source: root, Native: out, Out: out, Arch: "x86_64", Revision: fixtureRevision, Vendor: vendor}
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
	p.Execute = func(dir, name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "go" && args[0] == "build" {
			header := make([]byte, 64)
			copy(header, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
			binary.LittleEndian.PutUint16(header[16:], 2)
			binary.LittleEndian.PutUint16(header[18:], 62)
			binary.LittleEndian.PutUint32(header[20:], 1)
			binary.LittleEndian.PutUint16(header[52:], 64)
			for i, a := range args {
				if a == "-o" {
					return os.WriteFile(args[i+1], header, 0755)
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
				return WriteNew(args[i+1], []byte(image.Config), 0600)
			}
			if a == "--output" {
				return WriteNew(args[i+1], bytes, 0600)
			}
		}
		t.Fatalf("unexpected execution: %v", args)
		return nil
	}
	return p, &calls
}

func TestProductionBothLayoutsUseOneAssetAndImageSequence(t *testing.T) {
	for _, vendor := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "vendor"}[vendor], func(t *testing.T) {
			p, calls := productionFixture(t, vendor)
			host, forgejo := "", ""
			if vendor {
				host = filepath.Join(p.Out, "context")
				forgejo = filepath.Join(p.Out, "forgejo-context")
			}
			if e := p.Assets(host, forgejo); e != nil {
				t.Fatal(e)
			}
			images, e := p.Images(forgejo)
			if e != nil {
				t.Fatal(e)
			}
			if len(images) != 5 {
				t.Fatal(images)
			}
			text := strings.Join(*calls, "\n")
			for _, needle := range []string{"bun install --frozen-lockfile", "bun scripts/build-forgejo.ts", "python3 scripts/fetch-tea.py", "python3 scripts/stage.py", "STEP Build image: dashboard\n", "STEP Build image: project-os\n", "STEP Build image: tailnet\n"} {
				if strings.Count(text, needle) != 1 {
					t.Fatalf("not produced exactly once: %s\n%s", needle, text)
				}
			}
			if strings.Contains(text, "--host-context "+host+" --forgejo-context "+forgejo) != vendor {
				t.Fatal("asset destinations do not match the selected layout", text)
			}
			if strings.Count(text, " save --format=oci-archive") != 5 {
				t.Fatal(text)
			}
			if strings.Contains(text, " push ") || strings.Contains(text, " --rm ") || strings.Contains(text, "--replace") {
				t.Fatal("unapproved publication/cleanup")
			}
			if strings.Contains(text, "STEP Build image: forgejo\n") != vendor {
				t.Fatal("Forgejo layout confused")
			}
			proxy := "caddy"
			if vendor {
				proxy = "proxy"
			}
			if _, ok := images[proxy]; !ok {
				t.Fatal("legacy filename or vendor identity lost")
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
		})
	}
}

func TestProductionFailureStopsBeforeLaterImages(t *testing.T) {
	p, calls := productionFixture(t, true)
	execute := p.Execute
	sentinel := errors.New("fixture build refused")
	p.Execute = func(dir, name string, args ...string) error {
		if name == "podman" && strings.Contains(strings.Join(args, " "), "dashboard.Containerfile") {
			return sentinel
		}
		return execute(dir, name, args...)
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
			p, _ := productionFixture(t, true)
			switch mode {
			case "bun":
				p.Capture = func(string, string, ...string) (string, error) { return "different", nil }
				if e := p.Assets(filepath.Join(p.Out, "context"), filepath.Join(p.Out, "forgejo-context")); e == nil {
					t.Fatal("wrong Bun accepted")
				}
				return
			case "base":
				os.WriteFile(filepath.Join(p.Source, "appliance/dashboard.Containerfile"), []byte("ARG BASE_IMAGE=unrelated\n"), 0644)
			case "tailnet":
				os.WriteFile(filepath.Join(p.Source, "appliance/locks/tailscale-image.json"), []byte(`{"Version":"1.0.0","Base":"mutable:latest","SHA256":{}}`), 0644)
			case "forgejo":
				if _, e := p.Images(""); e == nil {
					t.Fatal("legacy Forgejo entered host payload")
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
	for _, vendor := range []bool{false, true} {
		p, calls := productionFixture(t, vendor)
		host, forgejo := "/host-context", "/forgejo-context"
		if vendor {
			forgejo = ""
		}
		if err := p.Assets(host, forgejo); err == nil || len(*calls) != 0 {
			t.Fatal("invalid asset layout had downstream effects", err, *calls)
		}
	}
}

func TestProductionCompileKeepsLayoutAndVerifiesELF(t *testing.T) {
	for _, vendor := range []bool{false, true} {
		p, calls := productionFixture(t, vendor)
		dest := filepath.Join(p.Out, "program")
		if e := p.Compile("soda-host", "./cmd/soda-host", dest); e != nil {
			t.Fatal(e)
		}
		text := strings.Join(*calls, "\n")
		if strings.Contains(text, "-tags=soda_host_image") != vendor || strings.Contains(text, "-buildvcs=false") != vendor {
			t.Fatal(text)
		}
		if strings.Count(text, "go build") != 1 {
			t.Fatal("duplicate compilation")
		}
		p.Execute = func(string, string, ...string) error { return os.WriteFile(dest, []byte("not ELF"), 0755) }
		if e := p.Compile("soda-host", "./cmd/soda-host", dest); e == nil {
			t.Fatal("invalid program accepted")
		}
	}
}

func TestSodaCommandsRefusesSupportToolsInRuntime(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"soda-host", "soda-dashboard"} {
		if e := os.MkdirAll(filepath.Join(root, "cmd", name), 0755); e != nil {
			t.Fatal(e)
		}
	}
	names, e := SodaCommands(root)
	if e != nil || strings.Join(names, ",") != "soda-dashboard,soda-host" {
		t.Fatal(names, e)
	}
	os.Mkdir(filepath.Join(root, "cmd/soda-artifacts"), 0755)
	if _, e = SodaCommands(root); e == nil {
		t.Fatal("support tool admitted")
	}
}
