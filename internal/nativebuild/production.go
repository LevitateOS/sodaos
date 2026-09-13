package nativebuild

// The two installation layouts share these concrete production steps, not two
// copies of their command sequences. Containerfiles, locks and stage.py remain
// the owners of content. This code never signs, publishes or installs anything.
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type BuildExec func(string, string, ...string) error
type BuildCapture func(string, string, ...string) (string, error)

type Production struct {
	Source, Native, Out, Arch, Revision string
	// Vendor selects image-owned binaries and Forgejo presentation. Legacy is
	// deliberately not byte-equivalent; it still uses the writable installer.
	Vendor  bool
	Execute BuildExec
	Capture BuildCapture
	Next    func(string) error
}

func (p Production) step(label string) error {
	if p.Next != nil {
		return p.Next(label)
	}
	return nil
}
func (p Production) validate() error {
	if _, e := OCIArchitecture(p.Arch); e != nil {
		return e
	}
	if !Revision(p.Revision) || !filepath.IsAbs(p.Source) || p.Native != filepath.Join(p.Source, ".artifacts/native", p.Arch) || !filepath.IsAbs(p.Out) || p.Execute == nil || p.Capture == nil {
		return errors.New("explicit native production inputs required")
	}
	return nil
}

func SodaCommands(source string) ([]string, error) {
	entries, e := os.ReadDir(filepath.Join(source, "cmd"))
	if e != nil {
		return nil, e
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			n := entry.Name()
			if !regexp.MustCompile(`^soda-[a-z0-9-]+$`).MatchString(n) || n == "soda-artifacts" || n == "soda-acceptance" {
				return nil, errors.New("support tools must remain outside appliance commands")
			}
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return nil, errors.New("missing Soda commands")
	}
	return names, nil // ReadDir is sorted.
}

// Compile is the sole Go command recipe for runtime programs and legacy support
// tools. Vendor binaries intentionally use their layout tag and archive-safe VCS
// mode; callers cannot silently substitute legacy binaries in a host image.
func (p Production) Compile(name, pkg, dest string) error {
	if e := p.validate(); e != nil {
		return e
	}
	if e := p.step("Compile " + name); e != nil {
		return e
	}
	args := []string{"build", "-mod=readonly", "-trimpath"}
	if p.Vendor {
		args = append(args, "-buildvcs=false", "-tags=soda_host_image")
	} else {
		args = append(args, "-buildvcs=true")
	}
	args = append(args, "-o", dest, pkg)
	if e := p.Execute(p.Source, "go", args...); e != nil {
		return e
	}
	if e := os.Chmod(dest, 0755); e != nil {
		return e
	}
	return inspectELF(dest, p.Arch)
}

// Assets runs exactly once before layout-specific assembly or image production.
func (p Production) Assets() error {
	if e := p.validate(); e != nil {
		return e
	}
	if e := p.step("Check frontend toolchain"); e != nil {
		return e
	}
	var workspace struct{ PackageManager string }
	// The workspace has unrelated fields; read the pin from its owning manifest.
	data, e := os.ReadFile(filepath.Join(p.Source, "package.json"))
	if e != nil {
		return e
	}
	if e = json.Unmarshal(data, &workspace); e != nil {
		return e
	}
	bun, e := p.Capture(p.Source, "bun", "--version")
	if e != nil || "bun@"+bun != workspace.PackageManager {
		return errors.New("workspace-pinned Bun required")
	}
	steps := []struct {
		label string
		args  []string
	}{
		{"Verify Go dependencies", []string{"go", "mod", "verify"}},
		{"Install frontend dependencies", []string{"bun", "install", "--frozen-lockfile"}},
		{"Build frontend assets", []string{"bun", "scripts/build-forgejo.ts", "--out", filepath.Join(p.Native, "forgejo-js")}},
		{"Fetch terminal assets", []string{"python3", "scripts/fetch-terminal.py", "--out", filepath.Join(p.Native, "terminal-assets")}},
		{"Prepare Forgejo translations", []string{"python3", "scripts/forgejo-locales.py", "--lock", "appliance/forgejo/locale.lock.json", "--out", filepath.Join(p.Native, "forgejo-locales/locale_en-US.ini")}},
		{"Fetch upstream Tea binary", []string{"python3", "scripts/fetch-tea.py", "--arch", p.Arch}},
		{"Stage appliance files", []string{"python3", "scripts/stage.py", "--arch", p.Arch}},
	}
	for _, s := range steps {
		if e := p.step(s.label); e != nil {
			return e
		}
		if e := p.Execute(p.Source, s.args[0], s.args[1:]...); e != nil {
			return e
		}
	}
	return nil
}

type ProducedImage struct {
	Image
	ArchiveSHA256 string
}

// Images produces each selected app archive once. The only layout-specific app
// recipe is Forgejo: vendor builds immutable presentation; legacy pulls upstream
// and stages presentation into writable paths at installation. Proxy/caddy is one
// component with the legacy archive filename retained for bundle compatibility.
func (p Production) Images(forgejoContext string) (map[string]ProducedImage, error) {
	if e := p.validate(); e != nil {
		return nil, e
	}
	if p.Vendor != (forgejoContext != "") {
		return nil, errors.New("explicit Forgejo layout required")
	}
	platform, _ := OCIArchitecture(p.Arch)
	archives := filepath.Join(p.Out, "images")
	if e := os.Mkdir(archives, 0755); e != nil {
		return nil, e
	}
	var inputs []struct{ Requested, Reference, Config string }
	pull := func(label, ref, iidName string) (string, string, error) {
		if e := p.step("Pull and resolve " + label); e != nil {
			return "", "", e
		}
		id, e := p.Capture(p.Source, "podman", "--remote=false", "pull", "--quiet", "--platform=linux/"+platform, ref)
		if e != nil {
			return "", "", e
		}
		if !Digest(strings.TrimPrefix(id, "sha256:")) {
			return "", "", errors.New("invalid pulled image ID")
		}
		id = "sha256:" + strings.TrimPrefix(id, "sha256:")
		digest, e := p.Capture(p.Source, "podman", "--remote=false", "image", "inspect", "--format", "{{.Digest}}", id)
		if e != nil {
			return "", "", e
		}
		if !strings.HasPrefix(digest, "sha256:") || !Digest(strings.TrimPrefix(digest, "sha256:")) {
			return "", "", errors.New("invalid registry digest")
		}
		repo := strings.SplitN(ref, "@", 2)[0]
		if colon := strings.LastIndex(repo, ":"); colon > strings.LastIndex(repo, "/") {
			repo = repo[:colon]
		}
		pinned := repo + "@" + digest
		if iidName != "" {
			if e = WriteNew(filepath.Join(p.Out, iidName+".iid"), []byte(id+"\n"), 0600); e != nil {
				return "", "", e
			}
		}
		inputs = append(inputs, struct{ Requested, Reference, Config string }{ref, pinned, id})
		b, e := json.MarshalIndent(inputs, "", "  ")
		if e != nil {
			return "", "", e
		}
		// This private, fresh production attempt owns this growing provenance record.
		if e = os.WriteFile(filepath.Join(p.Out, "app-inputs.json"), append(b, '\n'), 0600); e != nil {
			return "", "", e
		}
		return id, pinned, nil
	}
	build := func(name, dir, file, pinned string, args ...string) (string, error) {
		if e := p.step("Build image: " + name); e != nil {
			return "", e
		}
		iid := filepath.Join(p.Out, name+".iid")
		cmd := []string{"--remote=false", "build", "--pull=never", "--rm=false", "--platform=linux/" + platform, "--build-arg=BASE_IMAGE=" + pinned, "--label=org.opencontainers.image.revision=" + p.Revision, "--label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos", "--label=org.opencontainers.image.base.name=" + pinned, "--label=org.opencontainers.image.base.digest=" + strings.SplitN(pinned, "@", 2)[1], "--iidfile", iid, "--file", file}
		cmd = append(cmd, args...)
		cmd = append(cmd, ".")
		if e := p.Execute(dir, "podman", cmd...); e != nil {
			return "", e
		}
		b, e := os.ReadFile(iid)
		if e != nil {
			return "", e
		}
		id := strings.TrimSpace(string(b))
		if !strings.HasPrefix(id, "sha256:") || !Digest(strings.TrimPrefix(id, "sha256:")) {
			return "", errors.New("invalid built image ID")
		}
		return id, nil
	}
	result := map[string]ProducedImage{}
	export := func(name, id, revision string) error {
		if e := p.step("Export and verify image: " + name); e != nil {
			return e
		}
		file := filepath.Join(archives, name+".oci")
		if e := p.Execute(p.Source, "podman", "--remote=false", "save", "--format=oci-archive", "--output", file, id); e != nil {
			return e
		}
		im, e := InspectOCI(file, p.Arch, revision)
		if e != nil {
			return e
		}
		if im.Config != id {
			return errors.New("app archive/config mismatch")
		}
		hash, e := HashFile(file)
		if e != nil {
			return e
		}
		result[name] = ProducedImage{im, hash}
		return nil
	}
	rocky, e := recipeBase(filepath.Join(p.Source, "project-os/Containerfile"))
	if e != nil {
		return nil, e
	}
	dashboard, e := recipeBase(filepath.Join(p.Source, "appliance/dashboard.Containerfile"))
	if e != nil {
		return nil, e
	}
	if rocky != dashboard {
		return nil, errors.New("dashboard and Project OS base owners disagree")
	}
	_, rocky, e = pull("Rocky base", rocky, "base")
	if e != nil {
		return nil, e
	}
	nativeRel, e := filepath.Rel(p.Source, p.Native)
	if e != nil || strings.HasPrefix(nativeRel, "..") {
		return nil, errors.New("native assets must be inside source context")
	}
	for _, name := range []string{"dashboard", "project-os"} {
		file := "appliance/dashboard.Containerfile"
		if name == "project-os" {
			file = "project-os/Containerfile"
		}
		id, e := build(name, p.Source, file, rocky, "--build-arg=ARTIFACT_DIR="+filepath.ToSlash(nativeRel))
		if e != nil {
			return nil, e
		}
		if e = export(name, id, p.Revision); e != nil {
			return nil, e
		}
	}
	forgejo, e := unitImage(filepath.Join(p.Source, "appliance/services/forgejo.container"))
	if e != nil {
		return nil, e
	}
	iid := "forgejo"
	if p.Vendor {
		iid = "forgejo-base"
	}
	id, pinned, e := pull("Forgejo", forgejo, iid)
	if e != nil {
		return nil, e
	}
	revision := ""
	if p.Vendor {
		id, e = build("forgejo", forgejoContext, "Containerfile", pinned)
		if e != nil {
			return nil, e
		}
		revision = p.Revision
	}
	if e = export("forgejo", id, revision); e != nil {
		return nil, e
	}
	proxy, e := unitImage(filepath.Join(p.Source, "appliance/services/soda-proxy.container"))
	if e != nil {
		return nil, e
	}
	name := "caddy"
	if p.Vendor {
		name = "proxy"
	}
	id, _, e = pull("Proxy", proxy, name)
	if e != nil {
		return nil, e
	}
	if e = export(name, id, ""); e != nil {
		return nil, e
	}
	var tail struct {
		Version, Base string
		SHA256        map[string]string
	}
	if e = ReadJSON(filepath.Join(p.Source, "appliance/locks/tailscale-image.json"), &tail); e != nil {
		return nil, e
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tail.Version) || !regexp.MustCompile(`^docker.io/tailscale/alpine-base@sha256:[0-9a-f]{64}$`).MatchString(tail.Base) || !Digest(tail.SHA256[platform]) {
		return nil, errors.New("invalid locked Tailscale input")
	}
	_, pinned, e = pull("Tailnet base", tail.Base, "tailnet-base")
	if e != nil {
		return nil, e
	}
	id, e = build("tailnet", p.Source, "appliance/tailnet.Containerfile", pinned, "--build-arg=TAILSCALE_VERSION="+tail.Version, "--build-arg=TARGETARCH="+platform, "--build-arg=ARCHIVE_SHA256="+tail.SHA256[platform])
	if e != nil {
		return nil, e
	}
	if e = export("tailnet", id, p.Revision); e != nil {
		return nil, e
	}
	return result, nil
}

func recipeBase(path string) (string, error) { return singleSetting(path, "ARG BASE_IMAGE=") }
func unitImage(path string) (string, error)  { return singleSetting(path, "Image=") }
func singleSetting(path, prefix string) (string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	value := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, prefix) {
			if value != "" {
				return "", fmt.Errorf("duplicate %s input", prefix)
			}
			value = strings.TrimPrefix(line, prefix)
		}
	}
	if value == "" {
		return "", fmt.Errorf("missing %s input", prefix)
	}
	return value, nil
}
