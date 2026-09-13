package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/store"
)

type buildExec func(string, string, ...string) error
type buildCapture func(string, string, ...string) (string, error)

// completeCandidate reuses the current app recipes, asset builders and native
// staging owner. It builds local artifacts with intended, unpublished GHCR refs;
// nothing in this function authenticates to or writes a registry.
func completeCandidate(source, context, out, arch, revision, prefix string, base hostimage.Base, execute buildExec, capture buildCapture) (appliancerelease.Payload, error) {
	p := appliancerelease.Payload{Format: 1, ID: base.Release + ".soda-" + revision[:12], Revision: revision, Architecture: arch, CoreOS: base.Release, Base: base.Images[arch], RepositoryPrefix: prefix, Schema: store.SchemaVersion(), Images: map[string]appliancerelease.Image{}, UpgradeFrom: []string{}}
	var err error
	p.HostPackagesSHA256, err = hostimage.LockHostPackages(source, context, arch, base)
	if err != nil {
		return p, err
	}
	manifest, err := os.ReadFile(filepath.Join(source, "package.json"))
	if err != nil {
		return p, err
	}
	var workspace struct {
		PackageManager string `json:"packageManager"`
	}
	if err = json.Unmarshal(manifest, &workspace); err != nil {
		return p, err
	}
	bun, err := capture(source, "bun", "--version")
	if err != nil || "bun@"+bun != workspace.PackageManager {
		return p, errors.New("workspace-pinned Bun required")
	}
	nativeRel := ".artifacts/native/" + arch
	native := filepath.Join(source, filepath.FromSlash(nativeRel))
	if err = os.MkdirAll(filepath.Join(native, "bin"), 0755); err != nil {
		return p, err
	}
	commands, err := hostimage.Commands(source)
	if err != nil {
		return p, err
	}
	for _, name := range commands {
		b, err := os.ReadFile(filepath.Join(context, "rootfs/usr/libexec/soda", name))
		if err != nil {
			return p, err
		}
		if err = os.WriteFile(filepath.Join(native, "bin", name), b, 0755); err != nil {
			return p, err
		}
	}
	for _, cmd := range [][]string{
		{"bun", "install", "--frozen-lockfile"},
		{"bun", "scripts/build-forgejo.ts", "--out", filepath.Join(native, "forgejo-js")},
		{"python3", "scripts/fetch-terminal.py", "--out", filepath.Join(native, "terminal-assets")},
		{"python3", "scripts/forgejo-locales.py", "--lock", "appliance/forgejo/locale.lock.json", "--out", filepath.Join(native, "forgejo-locales/locale_en-US.ini")},
		{"python3", "scripts/build-project-tools.py", "--arch", arch},
		{"python3", "scripts/stage.py", "--arch", arch},
	} {
		if err = execute(source, cmd[0], cmd[1:]...); err != nil {
			return p, err
		}
	}
	nativeRoot := filepath.Join(native, "rootfs")
	forgejoContext := filepath.Join(out, "forgejo-context")
	if err = os.Mkdir(forgejoContext, 0700); err != nil {
		return p, err
	}
	p.PresentationSHA256, err = hostimage.StagePresentation(nativeRoot, forgejoContext, context)
	if err != nil {
		return p, err
	}
	recipe, err := os.ReadFile(filepath.Join(source, "appliance/forgejo.Containerfile"))
	if err != nil {
		return p, err
	}
	if err = os.WriteFile(filepath.Join(forgejoContext, "Containerfile"), recipe, 0644); err != nil {
		return p, err
	}
	archives := filepath.Join(out, "images")
	if err = os.Mkdir(archives, 0700); err != nil {
		return p, err
	}
	platform, _ := nativebuild.OCIArchitecture(arch)
	var inputs []struct{ Requested, Reference, Config string }
	pull := func(ref string) (string, string, error) {
		id, e := capture(source, "podman", "--remote=false", "pull", "--quiet", "--platform=linux/"+platform, ref)
		if e != nil {
			return "", "", e
		}
		if !nativebuild.Digest(strings.TrimPrefix(id, "sha256:")) {
			return "", "", errors.New("invalid pulled image ID")
		}
		digest, e := capture(source, "podman", "--remote=false", "image", "inspect", "--format", "{{.Digest}}", id)
		if e != nil {
			return "", "", e
		}
		if !strings.HasPrefix(digest, "sha256:") || !nativebuild.Digest(strings.TrimPrefix(digest, "sha256:")) {
			return "", "", errors.New("invalid registry digest")
		}
		repository := strings.SplitN(ref, "@", 2)[0]
		if colon := strings.LastIndex(repository, ":"); colon > strings.LastIndex(repository, "/") {
			repository = repository[:colon]
		}
		pinned := repository + "@" + digest
		inputs = append(inputs, struct{ Requested, Reference, Config string }{ref, pinned, id})
		b, e := json.MarshalIndent(inputs, "", "  ")
		if e != nil {
			return "", "", e
		}
		if e = os.WriteFile(filepath.Join(out, "app-inputs.json"), append(b, '\n'), 0600); e != nil {
			return "", "", e
		}
		return id, pinned, nil
	}
	imageRef := func(unit string) (string, error) {
		b, e := os.ReadFile(filepath.Join(source, "appliance/services", unit))
		if e != nil {
			return "", e
		}
		var ref string
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "Image=") {
				if ref != "" {
					return "", errors.New("duplicate service image")
				}
				ref = strings.TrimPrefix(line, "Image=")
			}
		}
		if ref == "" {
			return "", errors.New("missing service image")
		}
		return ref, nil
	}
	build := func(name, dir, file, pinned string, args ...string) (string, error) {
		iid := filepath.Join(out, name+".iid")
		cmd := []string{"--remote=false", "build", "--pull=never", "--rm=false", "--platform=linux/" + platform, "--build-arg=BASE_IMAGE=" + pinned, "--label=org.opencontainers.image.revision=" + revision, "--label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos", "--label=org.opencontainers.image.base.name=" + pinned, "--label=org.opencontainers.image.base.digest=" + strings.SplitN(pinned, "@", 2)[1], "--iidfile", iid, "--file", file}
		cmd = append(cmd, args...)
		cmd = append(cmd, ".")
		if e := execute(dir, "podman", cmd...); e != nil {
			return "", e
		}
		b, e := os.ReadFile(iid)
		if e != nil {
			return "", e
		}
		id := strings.TrimSpace(string(b))
		if !strings.HasPrefix(id, "sha256:") || !nativebuild.Digest(strings.TrimPrefix(id, "sha256:")) {
			return "", errors.New("invalid built image ID")
		}
		return id, nil
	}
	export := func(name, id, expectedRevision string) error {
		archive := filepath.Join(archives, name+".oci")
		if e := execute(source, "podman", "--remote=false", "save", "--format=oci-archive", "--output", archive, id); e != nil {
			return e
		}
		im, e := nativebuild.InspectOCI(archive, arch, expectedRevision)
		if e != nil {
			return e
		}
		if im.Config != "sha256:"+strings.TrimPrefix(id, "sha256:") {
			return errors.New("app archive/config mismatch")
		}
		hash, e := nativebuild.HashFile(archive)
		if e != nil {
			return e
		}
		storage := "bound"
		if name == "project-os" || name == "tailnet" {
			storage = "retained"
		}
		p.Images[name] = appliancerelease.Image{Reference: prefix + "-" + name + "@" + im.Manifest, Manifest: im.Manifest, Config: im.Config, ArchiveSHA256: hash, Storage: storage}
		return nil
	}
	rockyRecipe, err := os.ReadFile(filepath.Join(source, "project-os/Containerfile"))
	if err != nil {
		return p, err
	}
	var rocky string
	for _, line := range strings.Split(string(rockyRecipe), "\n") {
		if strings.HasPrefix(line, "ARG BASE_IMAGE=") {
			rocky = strings.TrimPrefix(line, "ARG BASE_IMAGE=")
		}
	}
	if rocky == "" {
		return p, errors.New("missing Project OS base owner")
	}
	_, rocky, err = pull(rocky)
	if err != nil {
		return p, err
	}
	for _, name := range []string{"dashboard", "project-os"} {
		file := "appliance/dashboard.Containerfile"
		if name == "project-os" {
			file = "project-os/Containerfile"
		}
		id, e := build(name, source, file, rocky, "--build-arg=ARTIFACT_DIR="+nativeRel)
		if e != nil {
			return p, e
		}
		if e = export(name, id, revision); e != nil {
			return p, e
		}
	}
	forgejo, err := imageRef("forgejo.container")
	if err != nil {
		return p, err
	}
	_, forgejo, err = pull(forgejo)
	if err != nil {
		return p, err
	}
	id, err := build("forgejo", forgejoContext, "Containerfile", forgejo)
	if err != nil {
		return p, err
	}
	if err = export("forgejo", id, revision); err != nil {
		return p, err
	}
	// Read-only inspection invokes the actual upstream binary, not its bootstrap
	// entrypoint. No Forgejo instance, account, database or listener is created.
	result, err := capture(source, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "forgejo-inspect.cid"), "--network=none", "--read-only", "--entrypoint=/bin/sh", id, "-ec", `test "$GITEA_CUSTOM" = /usr/share/soda/forgejo
 test "$FORGEJO_CUSTOM" = "$GITEA_CUSTOM"
 test "$(readlink "$GITEA_CUSTOM/conf")" = /data/gitea/conf
 test "$(stat -c '%u:%g:%a' "$GITEA_CUSTOM/templates/custom/header.tmpl")" = 0:0:444
 test -s "$GITEA_CUSTOM/public/assets/soda/forgejo/soda-native-page.js"
 /usr/local/bin/gitea --version`)
	if err != nil {
		return p, fmt.Errorf("Forgejo payload image inspection failed: %w", err)
	}
	if err = os.WriteFile(filepath.Join(out, "forgejo-inspection.txt"), []byte(result+"\n"), 0600); err != nil {
		return p, err
	}
	proxy, err := imageRef("soda-proxy.container")
	if err != nil {
		return p, err
	}
	id, _, err = pull(proxy)
	if err != nil {
		return p, err
	}
	if err = export("proxy", id, ""); err != nil {
		return p, err
	}
	var tail struct {
		Version string            `json:"version"`
		Base    string            `json:"base"`
		SHA256  map[string]string `json:"sha256"`
	}
	if err = nativebuild.ReadJSON(filepath.Join(source, "appliance/locks/tailscale-image.json"), &tail); err != nil {
		return p, err
	}
	if !nativebuild.Digest(tail.SHA256[platform]) {
		return p, errors.New("invalid locked Tailscale archive")
	}
	_, tailBase, err := pull(tail.Base)
	if err != nil {
		return p, err
	}
	id, err = build("tailnet", source, "appliance/tailnet.Containerfile", tailBase, "--build-arg=TAILSCALE_VERSION="+tail.Version, "--build-arg=TARGETARCH="+platform, "--build-arg=ARCHIVE_SHA256="+tail.SHA256[platform])
	if err != nil {
		return p, err
	}
	if err = export("tailnet", id, revision); err != nil {
		return p, err
	}
	if err = p.Validate(); err != nil {
		return p, err
	}
	if err = hostimage.Complete(source, nativeRoot, context, archives, p); err != nil {
		return p, err
	}
	record, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return p, err
	}
	err = nativebuild.WriteNew(filepath.Join(out, "payload.json"), append(record, '\n'), 0600)
	return p, err
}
