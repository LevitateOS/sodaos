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

type (
	BuildExec    func(string, string, ...string) error
	BuildCapture func(string, string, ...string) (string, error)
)

type Production struct {
	Source, Native, Out, Arch, Revision string
	// Vendor selects image-owned binaries and Forgejo presentation. Legacy is
	// deliberately not byte-equivalent; it still uses the writable installer.
	Vendor  bool
	Execute BuildExec
	Capture BuildCapture
	Next    func(string) error
	inputs  []ResolvedInput // admitted once; disk provenance is not execution authority
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
	if e := os.Chmod(dest, 0o755); e != nil {
		return e
	}
	return inspectELF(dest, p.Arch)
}

// Assets runs exactly once before layout-specific assembly or image production.
func (p Production) Assets(hostContext, forgejoContext string) error {
	if e := p.validate(); e != nil {
		return e
	}
	stage := []string{"python3", "scripts/stage.py", "--arch", p.Arch}
	if p.Vendor {
		if !filepath.IsAbs(hostContext) || !filepath.IsAbs(forgejoContext) {
			return errors.New("explicit vendor asset destinations required")
		}
		stage = append(stage, "--host-context", hostContext, "--forgejo-context", forgejoContext)
	} else if hostContext != "" || forgejoContext != "" {
		return errors.New("legacy assets cannot use vendor destinations")
	}
	return p.assetSteps(stage)
}

// Dependencies is run before any production compilation.
func (p Production) requirePinnedBun() error {
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
	return nil
}

func (p Production) Dependencies() error {
	if e := p.validate(); e != nil {
		return e
	}
	if e := p.step("Check frontend toolchain"); e != nil {
		return e
	}
	if e := p.requirePinnedBun(); e != nil {
		return e
	}
	if e := p.step("Verify Go dependencies"); e != nil {
		return e
	}
	if e := p.Execute(p.Source, "go", "mod", "verify"); e != nil {
		return e
	}
	if e := p.step("Install frontend dependencies"); e != nil {
		return e
	}
	return p.Execute(p.Source, "bun", "install", "--frozen-lockfile")
}

func (p Production) assetSteps(stage []string) error {
	steps := []struct {
		label string
		args  []string
	}{
		{"Build frontend assets", []string{"bun", "scripts/build-forgejo.ts", "--out", filepath.Join(p.Native, "forgejo-js")}},
		{"Fetch terminal assets", []string{"python3", "scripts/fetch-terminal.py", "--out", filepath.Join(p.Native, "terminal-assets")}},
		{"Prepare Forgejo translations", []string{"python3", "scripts/forgejo-locales.py", "--lock", "appliance/forgejo/locale.lock.json", "--out", filepath.Join(p.Native, "forgejo-locales/locale_en-US.ini")}},
		{"Fetch upstream Tea binary", []string{"python3", "scripts/fetch-tea.py", "--arch", p.Arch}},
		{"Stage appliance files", stage},
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
func (p Production) pullFrozenImage(inputs []ResolvedInput, label, ref, iidName string) (string, string, error) {
	if e := p.step("Select frozen " + label); e != nil {
		return "", "", e
	}
	for _, input := range inputs {
		if input.Requested != ref {
			continue
		}
		if !Digest(strings.TrimPrefix(input.Config, "sha256:")) || !strings.Contains(input.Reference, "@sha256:") {
			return "", "", errors.New("invalid frozen image input")
		}
		if iidName != "" {
			if e := WriteNew(filepath.Join(p.Out, iidName+".iid"), []byte(input.Config+"\n"), 0o600); e != nil {
				return "", "", e
			}
		}
		return input.Config, input.Reference, nil
	}
	return "", "", errors.New("image source was not frozen before production")
}

func (p Production) Images(forgejoContext string) (map[string]ProducedImage, error) {
	if e := p.validate(); e != nil {
		return nil, e
	}
	if p.Vendor != (forgejoContext != "") {
		return nil, errors.New("explicit Forgejo layout required")
	}
	archives := filepath.Join(p.Out, "images")
	if e := os.Mkdir(archives, 0o755); e != nil {
		return nil, e
	}
	inputs := p.inputs
	if len(inputs) != 4 {
		return nil, errors.New("image inputs must be frozen before production")
	}
	return p.exportImages(forgejoContext, archives, func(label, ref, iidName string) (string, string, error) {
		return p.pullFrozenImage(inputs, label, ref, iidName)
	}, p.buildImage)
}

// ResolveInputs records the actual upstream manifests once, before shipping work.
// This is the existing app-inputs owner, not a reuse planner or another inventory.
type ResolvedInput struct{ Requested, Reference, Config string }

func parseImageRepo(ref string) string {
	repo := strings.SplitN(ref, "@", 2)[0]
	if colon := strings.LastIndex(repo, ":"); colon > strings.LastIndex(repo, "/") {
		repo = repo[:colon]
	}
	return repo
}

func (p *Production) admitResolvedInputRecord(inputs []ResolvedInput) error {
	b, e := json.MarshalIndent(inputs, "", "  ")
	if e != nil {
		return e
	}
	// This private, fresh production attempt owns this growing provenance record.
	return os.WriteFile(filepath.Join(p.Out, "app-inputs.json"), append(b, '\n'), 0o600)
}

func (p *Production) pullResolvedInput(label, ref, iidName, platform string, inputs *[]ResolvedInput) (string, string, error) {
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
	pinned := parseImageRepo(ref) + "@" + digest
	if iidName != "" {
		if e = WriteNew(filepath.Join(p.Out, iidName+".iid"), []byte(id+"\n"), 0o600); e != nil {
			return "", "", e
		}
	}
	*inputs = append(*inputs, ResolvedInput{ref, pinned, id})
	return id, pinned, p.admitResolvedInputRecord(*inputs)
}

func (p *Production) recipeImageRefs() (rocky, forgejo, proxy string, err error) {
	rocky, err = recipeBase(filepath.Join(p.Source, "project-os/Containerfile"))
	if err != nil {
		return "", "", "", err
	}
	dashboard, err := recipeBase(filepath.Join(p.Source, "appliance/dashboard.Containerfile"))
	if err != nil {
		return "", "", "", err
	}
	if rocky != dashboard {
		return "", "", "", errors.New("dashboard and Project OS base owners disagree")
	}
	forgejo, err = unitImage(filepath.Join(p.Source, "appliance/services/forgejo.container"))
	if err != nil {
		return "", "", "", err
	}
	proxy, err = unitImage(filepath.Join(p.Source, "appliance/services/soda-proxy.container"))
	return rocky, forgejo, proxy, err
}

func (p *Production) lockedTailnetBase(platform string) (string, error) {
	var tail struct {
		Version, Base string
		SHA256        map[string]string
	}
	if e := ReadJSON(filepath.Join(p.Source, "appliance/locks/tailscale-image.json"), &tail); e != nil {
		return "", e
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tail.Version) || !regexp.MustCompile(`^docker.io/tailscale/alpine-base@sha256:[0-9a-f]{64}$`).MatchString(tail.Base) || !Digest(tail.SHA256[platform]) {
		return "", errors.New("invalid locked Tailnet input")
	}
	return tail.Base, nil
}

func (p *Production) ResolveInputs() error {
	if e := p.validate(); e != nil {
		return e
	}
	if _, e := os.Lstat(filepath.Join(p.Out, "app-inputs.json")); !os.IsNotExist(e) {
		return errors.New("occupied image input record")
	}
	platform, _ := OCIArchitecture(p.Arch)
	rocky, forgejo, proxy, e := p.recipeImageRefs()
	if e != nil {
		return e
	}
	tailBase, e := p.lockedTailnetBase(platform)
	if e != nil {
		return e
	}
	var inputs []ResolvedInput
	for _, ref := range []string{rocky, forgejo, proxy, tailBase} {
		if _, _, e = p.pullResolvedInput(ref, ref, "", platform, &inputs); e != nil {
			return e
		}
	}
	p.inputs = inputs
	return nil
}

func (p Production) buildImage(name, dir, file, pinned string, args ...string) (string, error) {
	platform, _ := OCIArchitecture(p.Arch)
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

func (p Production) exportImageArchive(archives, name, id, revision string) (ProducedImage, error) {
	if err := p.step("Export and verify image: " + name); err != nil {
		return ProducedImage{}, err
	}
	file := filepath.Join(archives, name+".oci")
	if err := p.Execute(p.Source, "podman", "--remote=false", "save", "--format=oci-archive", "--output", file, id); err != nil {
		return ProducedImage{}, err
	}
	im, err := InspectOCI(file, p.Arch, revision)
	if err != nil {
		return ProducedImage{}, err
	}
	if im.Config != id {
		return ProducedImage{}, errors.New("app archive/config mismatch")
	}
	hash, err := HashFile(file)
	if err != nil {
		return ProducedImage{}, err
	}
	return ProducedImage{im, hash}, nil
}

func (p Production) resolveRockyBase(pull func(string, string, string) (string, string, error)) (string, string, error) {
	rocky, err := recipeBase(filepath.Join(p.Source, "project-os/Containerfile"))
	if err != nil {
		return "", "", err
	}
	dashboard, err := recipeBase(filepath.Join(p.Source, "appliance/dashboard.Containerfile"))
	if err != nil {
		return "", "", err
	}
	if rocky != dashboard {
		return "", "", errors.New("dashboard and Project OS base owners disagree")
	}
	_, pinned, err := pull("Rocky base", rocky, "base")
	if err != nil {
		return "", "", err
	}
	nativeRel, err := filepath.Rel(p.Source, p.Native)
	if err != nil || strings.HasPrefix(nativeRel, "..") {
		return "", "", errors.New("native assets must be inside source context")
	}
	return pinned, nativeRel, nil
}

func (p Production) exportAppImages(
	nativeRel, rocky string,
	build func(string, string, string, string, ...string) (string, error),
	export func(string, string, string) error,
) error {
	for _, name := range []string{"dashboard", "project-os"} {
		file := "appliance/dashboard.Containerfile"
		if name == "project-os" {
			file = "project-os/Containerfile"
		}
		id, err := build(name, p.Source, file, rocky, "--build-arg=ARTIFACT_DIR="+filepath.ToSlash(nativeRel))
		if err != nil {
			return err
		}
		if err = export(name, id, p.Revision); err != nil {
			return err
		}
	}
	return nil
}

func (p Production) exportForgejoImage(
	forgejoContext string,
	pull func(string, string, string) (string, string, error),
	build func(string, string, string, string, ...string) (string, error),
	export func(string, string, string) error,
) error {
	forgejo, err := unitImage(filepath.Join(p.Source, "appliance/services/forgejo.container"))
	if err != nil {
		return err
	}
	iid := "forgejo"
	if p.Vendor {
		iid = "forgejo-base"
	}
	id, pinned, err := pull("Forgejo", forgejo, iid)
	if err != nil {
		return err
	}
	revision := ""
	if p.Vendor {
		id, err = build("forgejo", forgejoContext, "Containerfile", pinned)
		if err != nil {
			return err
		}
		revision = p.Revision
	}
	return export("forgejo", id, revision)
}

func (p Production) exportProxyImage(
	pull func(string, string, string) (string, string, error),
	export func(string, string, string) error,
) error {
	proxy, err := unitImage(filepath.Join(p.Source, "appliance/services/soda-proxy.container"))
	if err != nil {
		return err
	}
	name := "caddy"
	if p.Vendor {
		name = "proxy"
	}
	id, _, err := pull("Proxy", proxy, name)
	if err != nil {
		return err
	}
	return export(name, id, "")
}

func (p Production) exportTailnetImage(
	platform string,
	pull func(string, string, string) (string, string, error),
	build func(string, string, string, string, ...string) (string, error),
	export func(string, string, string) error,
) error {
	var tail struct {
		Version, Base string
		SHA256        map[string]string
	}
	if err := ReadJSON(filepath.Join(p.Source, "appliance/locks/tailscale-image.json"), &tail); err != nil {
		return err
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tail.Version) || !regexp.MustCompile(`^docker.io/tailscale/alpine-base@sha256:[0-9a-f]{64}$`).MatchString(tail.Base) || !Digest(tail.SHA256[platform]) {
		return errors.New("invalid locked Tailscale input")
	}
	_, pinned, err := pull("Tailnet base", tail.Base, "tailnet-base")
	if err != nil {
		return err
	}
	id, err := build("tailnet", p.Source, "appliance/tailnet.Containerfile", pinned, "--build-arg=TAILSCALE_VERSION="+tail.Version, "--build-arg=TARGETARCH="+platform, "--build-arg=ARCHIVE_SHA256="+tail.SHA256[platform])
	if err != nil {
		return err
	}
	return export("tailnet", id, p.Revision)
}

func (p Production) exportImages(forgejoContext, archives string, pull func(string, string, string) (string, string, error), build func(string, string, string, string, ...string) (string, error)) (map[string]ProducedImage, error) {
	platform, _ := OCIArchitecture(p.Arch)
	result := map[string]ProducedImage{}
	export := func(name, id, revision string) error {
		produced, err := p.exportImageArchive(archives, name, id, revision)
		if err != nil {
			return err
		}
		result[name] = produced
		return nil
	}
	rocky, nativeRel, err := p.resolveRockyBase(pull)
	if err != nil {
		return nil, err
	}
	if err = p.exportAppImages(nativeRel, rocky, build, export); err != nil {
		return nil, err
	}
	if err = p.exportForgejoImage(forgejoContext, pull, build, export); err != nil {
		return nil, err
	}
	if err = p.exportProxyImage(pull, export); err != nil {
		return nil, err
	}
	if err = p.exportTailnetImage(platform, pull, build, export); err != nil {
		return nil, err
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
