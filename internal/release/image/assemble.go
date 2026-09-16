package image

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

type mediaLock struct{ Assembler, Config, Architecture, Installer string }

// Signing inputs belong to the invoking authority, never the source snapshot or
// child build environment. Local fixture authority does not prove job isolation;
// protected worker commissioning and final qualification remain B4/B5 work.
type MediaAuthority struct {
	Trust string
	Keys  deliver.SecretFiles
}
type MediaFile struct {
	Path, SHA256 string
	Bytes        int64
}
type Media struct {
	Scope, Revision, Architecture, HostManifest, PayloadSHA256, ConsoleSHA256 string
	AssemblerImportCommit, RootfsURL                                          string
	ISO, Rootfs                                                               MediaFile
	Tools                                                                     mediaLock
	CompressionMode, RootfsFilesystem, RootfsOptions                          string
}

func mediaBaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(value, "\r\n ") {
		return errors.New("explicit public HTTP(S) rootfs base URL required")
	}
	// The installing machine fetches this address: a loopback URL always
	// points at the guest itself, never at the machine serving the rootfs.
	host := strings.ToLower(u.Hostname())
	if host == "localhost" {
		return errors.New("rootfs base URL must be reachable from the installing machine, not loopback")
	}
	if addr, err := netip.ParseAddr(host); err == nil && addr.IsLoopback() {
		return errors.New("rootfs base URL must be reachable from the installing machine, not loopback")
	}
	return nil
}

// assemblerImage and assemblerConfigBranch float on upstream: the latest
// assembler release and the stable config branch matching the stable base.
// (Upstream publishes no stable assembler tag.) The resolved digest and
// fetched revision are recorded per build in media.json; no pinned digest,
// revision or installer version precedes the run.
const assemblerImage = "quay.io/coreos-assembler/coreos-assembler:latest"
const assemblerConfigBranch = "stable"

func fetchAssemblerConfig(p build.Production, run func(string, ...string) error, root string) (string, error) {
	for _, args := range [][]string{{"init", "config-repo"}, {"-C", "config-repo", "fetch", "--depth=1", "https://github.com/coreos/fedora-coreos-config.git", assemblerConfigBranch}} {
		if err := run("git", args...); err != nil {
			return "", err
		}
	}
	sha, err := p.Capture(root, "git", "-C", "config-repo", "rev-parse", "FETCH_HEAD")
	if err != nil {
		return "", err
	}
	sha = strings.TrimSpace(sha)
	if !build.Revision(sha) {
		return "", errors.New("unresolved assembler config revision")
	}
	if err := run("git", "-C", "config-repo", "archive", "--format=tar", "--output", filepath.Join(root, "config.tar"), sha); err != nil {
		return "", err
	}
	config := filepath.Join(root, "config")
	if err := os.Mkdir(config, 0o755); err != nil {
		return "", err
	}
	if err := run("tar", "-xf", "config.tar", "-C", config, "--no-same-owner"); err != nil {
		return "", err
	}
	return sha, nil
}

func pinAssemblerBuildArgs(root string) error {
	argsFile := filepath.Join(root, "config", "build-args.conf")
	b, err := os.ReadFile(argsFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "BUILDER_IMG=") {
			lines[i] = "BUILDER_IMG=oci-archive:/srv/tmp/assembler-root.oci"
			found = true
		}
	}
	if !found {
		return errors.New("missing upstream buildroot selection")
	}
	return os.WriteFile(argsFile, []byte(strings.Join(lines, "\n")), 0o644)
}

func verifyAssemblerLayers(p build.Production, root, assembler, id string) error {
	original, err := p.Capture(root, "podman", "--remote=false", "image", "inspect", "--format", "{{json .RootFS.Layers}}", assembler)
	if err != nil {
		return err
	}
	wrapped, err := p.Capture(root, "podman", "--remote=false", "image", "inspect", "--format", "{{json .RootFS.Layers}}", id)
	if err != nil {
		return err
	}
	if original != wrapped || original == "" {
		return errors.New("assembler wrapper changed rootfs")
	}
	return nil
}

func wrapAssemblerImage(p build.Production, run func(string, ...string) error, root string) (string, error) {
	// No --policy=missing: the tag floats, so every build re-resolves it.
	if err := run("podman", "--remote=false", "pull", assemblerImage); err != nil {
		return "", err
	}
	digest, err := p.Capture(root, "podman", "--remote=false", "image", "inspect", "--format", "{{.Digest}}", assemblerImage)
	if err != nil {
		return "", err
	}
	digest = strings.TrimSpace(digest)
	const prefix = "quay.io/coreos-assembler/coreos-assembler@sha256:"
	if !strings.HasPrefix(digest, "sha256:") || !build.Digest(strings.TrimPrefix(digest, "sha256:")) {
		return "", errors.New("unresolved assembler digest")
	}
	ref := prefix + strings.TrimPrefix(digest, "sha256:")
	if err := build.WriteNew(filepath.Join(root, "Containerfile"), []byte("FROM "+ref+"\nUSER 0\n"), 0o644); err != nil {
		return "", err
	}
	if err := run("podman", "--remote=false", "build", "--pull=never", "--network=none", "--iidfile", "builder.iid", "."); err != nil {
		return "", err
	}
	id, err := builderID(root)
	if err != nil {
		return "", err
	}
	if err = verifyAssemblerLayers(p, root, ref, id); err != nil {
		return "", err
	}
	if err := run("podman", "--remote=false", "save", "--format=oci-archive", "--output", "assembler-root.oci", id); err != nil {
		return "", err
	}
	return ref, nil
}

func prepareAssembler(p build.Production, root string) (mediaLock, error) {
	lock := mediaLock{Architecture: p.Arch}
	if err := os.Mkdir(root, 0o700); err != nil {
		return lock, err
	}
	run := func(cmd string, args ...string) error { return p.Execute(root, cmd, args...) }
	sha, err := fetchAssemblerConfig(p, run, root)
	if err != nil {
		return lock, err
	}
	lock.Config = sha
	if err := pinAssemblerBuildArgs(root); err != nil {
		return lock, err
	}
	digest, err := wrapAssemblerImage(p, run, root)
	if err != nil {
		return lock, err
	}
	lock.Assembler = digest
	return lock, nil
}

func builderID(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "builder.iid"))
	id := strings.TrimSpace(string(b))
	if err != nil || !strings.HasPrefix(id, "sha256:") || !build.Digest(strings.TrimPrefix(id, "sha256:")) {
		return "", errors.New("missing exact assembler image")
	}
	return id, nil
}

func signMediaInput(ctx context.Context, authority MediaAuthority, trust deliver.Trust, input, transport, repository, digest, out string) error {
	runner := deliver.Native{Home: filepath.Dir(authority.Trust)}
	permit := deliver.Permit{Format: 1, Repository: repository, Digest: digest, Expires: time.Now().Add(time.Hour).Unix()}
	if err := deliver.Sign(ctx, runner, trust, permit, transport, input, out, authority.Keys); err != nil {
		return err
	}
	return deliver.VerifyCopy(ctx, runner, trust, repository+"@"+digest, "dir:"+filepath.Join(out, "signed"), out+"-admitted")
}

type mediaInputs struct {
	authority  MediaAuthority
	trust      deliver.Trust
	candidate  deliver.Candidate
	payload    deliver.Payload
	observed   build.Image
	archive    string
	filesystem string
	fsoptions  string
}

func admitMediaAuthority(authorityPath, prefix string) (MediaAuthority, deliver.Trust, error) {
	var authority MediaAuthority
	var trust deliver.Trust
	if err := deliver.ReadJSON(authorityPath, &authority); err != nil {
		return authority, trust, err
	}
	if err := deliver.ReadJSON(authority.Trust, &trust); err != nil {
		return authority, trust, err
	}
	if trust.Validate() != nil || trust.Prefix != prefix {
		return authority, trust, errors.New("media authority does not match intended repositories")
	}
	return authority, trust, nil
}

func admitMediaCandidate(artifacts, arch, revision string) (deliver.Candidate, deliver.Payload, build.Image, string, error) {
	var candidate deliver.Candidate
	if err := deliver.ReadJSON(filepath.Join(artifacts, "candidate.json"), &candidate); err != nil {
		return candidate, deliver.Payload{}, build.Image{}, "", err
	}
	payload, err := deliver.Load(filepath.Join(artifacts, "payload.json"))
	if err != nil {
		return candidate, payload, build.Image{}, "", err
	}
	payloadBytes, err := os.ReadFile(filepath.Join(artifacts, "payload.json"))
	if err != nil {
		return candidate, payload, build.Image{}, "", err
	}
	if err = candidate.Validate(payload, payloadBytes); err != nil {
		return candidate, payload, build.Image{}, "", err
	}
	archive := filepath.Join(artifacts, "host.oci")
	observed, err := build.InspectOCI(archive, arch, revision)
	if err != nil || observed.Manifest != candidate.Host.Manifest {
		return candidate, payload, observed, archive, errors.New("media candidate identity mismatch")
	}
	hash, err := build.HashFile(archive)
	if err != nil || hash != candidate.HostArchiveSHA256 {
		return candidate, payload, observed, archive, errors.New("media candidate archive mismatch")
	}
	return candidate, payload, observed, archive, nil
}

func admitMediaCompression(artifacts, mediaCompression string) (string, string, error) {
	filesystem, fsoptions, err := rootfsSettings(artifacts)
	if err != nil {
		return "", "", err
	}
	if mediaCompression == "fast" && (filesystem != "erofs" || fsoptions != fastRootfsOptions) {
		return "", "", errors.New("fast media metadata differs from selected setting")
	}
	return filesystem, fsoptions, nil
}

func admitMediaInputs(r Request, artifacts, arch, revision string) (mediaInputs, error) {
	var in mediaInputs
	var err error
	in.authority, in.trust, err = admitMediaAuthority(r.MediaAuthority, r.RepositoryPrefix)
	if err != nil {
		return in, err
	}
	in.candidate, in.payload, in.observed, in.archive, err = admitMediaCandidate(artifacts, arch, revision)
	if err != nil {
		return in, err
	}
	in.filesystem, in.fsoptions, err = admitMediaCompression(artifacts, r.MediaCompression)
	return in, err
}

func collectMediaInventory(root, artifacts, outDir string) (map[string]string, error) {
	inventory := map[string]string{}
	for _, dir := range []string{filepath.Join(root, "config"), filepath.Join(artifacts, "tools")} {
		if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			h, e := build.HashFile(path)
			if e == nil {
				rel, _ := filepath.Rel(outDir, path)
				inventory[rel] = h
			}
			return e
		}); err != nil {
			return nil, err
		}
	}
	for _, path := range []string{
		filepath.Join(root, "assembler-root.oci"),
		filepath.Join(root, "config.tar"),
		filepath.Join(artifacts, "live.ign"),
		filepath.Join(artifacts, "destination.ign"),
		filepath.Join(artifacts, "candidate.json"),
		filepath.Join(artifacts, "payload.json"),
		filepath.Join(artifacts, "image-config.json"),
	} {
		h, err := build.HashFile(path)
		if err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(outDir, path)
		inventory[rel] = h
	}
	return inventory, nil
}

func verifyMediaInventory(outDir string, inventory map[string]string) error {
	for path, want := range inventory {
		got, err := build.HashFile(filepath.Join(outDir, path))
		if err != nil || got != want {
			return errors.New("packaging input changed after admission")
		}
	}
	return nil
}

func authenticatePackagingInputs(ctx context.Context, r Request, root, artifacts, archive, manifest string, authority MediaAuthority, trust deliver.Trust, lock mediaLock) error {
	if err := deliver.CheckNative(ctx, deliver.Native{Home: filepath.Dir(authority.Trust)}); err != nil {
		return err
	}
	if err := signMediaInput(ctx, authority, trust, archive, "oci-archive", r.RepositoryPrefix+"-host", manifest, filepath.Join(root, "host-signature")); err != nil {
		return err
	}
	// The signed public inventory authenticates exactly the auxiliary inputs used
	// by the native packager. No source script runs in this signing operation.
	inventory, err := collectMediaInventory(root, artifacts, r.Out)
	if err != nil {
		return err
	}
	inputDocument := struct {
		Tools mediaLock
		Files map[string]string
	}{lock, inventory}
	document := filepath.Join(root, "input-document")
	digest, err := deliver.WriteDocument(document, inputDocument)
	if err != nil {
		return err
	}
	if err = signMediaInput(ctx, authority, trust, document, "oci", r.RepositoryPrefix+"-media", digest, filepath.Join(root, "input-signature")); err != nil {
		return err
	}
	return verifyMediaInventory(r.Out, inventory)
}

func stopPackagingContainer(root string) {
	if cid, err := os.ReadFile(filepath.Join(root, "packaging.cid")); err == nil && build.Digest(strings.TrimSpace(string(cid))) {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cleanup, "/usr/bin/podman", "--remote=false", "stop", "--time=10", strings.TrimSpace(string(cid)))
		cmd.Env = buildEnvironment()
		_ = cmd.Run()
	}
}

func buildMediaContainer(p build.Production, root, artifacts, payloadID, work, id string) error {
	base := []string{"--remote=false", "run", "--rm", "--cidfile=" + filepath.Join(root, "packaging.cid"), "--network=none", "--privileged", "--security-opt=label=disable", "--device=/dev/kvm", "--device=/dev/fuse", "--cpus=4", "--memory=16g", "--env=COSA_SUPERMIN_MEMORY=12288", "--env=RUNVM_NONET=1", "--volume=" + work + ":/srv:rw", "--volume=" + filepath.Join(root, "config") + ":/config:ro", "--volume=" + artifacts + ":/inputs:ro", "--volume=" + filepath.Join(root, "assembler-root.oci") + ":/assembler-root.oci:ro", "--workdir=/srv", "--entrypoint=/usr/bin/taskset", id, "-c", "0-3", "/bin/bash", "-euc"}
	script := `umask 0022; cosa init /config; cp /assembler-root.oci /srv/tmp/assembler-root.oci; cosa import --skip-prune oci-archive:/inputs/host.oci; cosa buildextend-live --build "$1"`
	if err := p.Execute(root, "podman", append(base, script, "assemble", payloadID)...); err != nil {
		// Podman owns the helper VM's namespace, outside the CLI process group.
		// Stop only this run's container even when the build context is cancelled.
		stopPackagingContainer(root)
		return err
	}
	return nil
}

func assembleNativeMedia(p build.Production, root, artifacts, payloadID string) (string, string, error) {
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		return "", "", err
	}
	id, err := builderID(root)
	if err != nil {
		return "", "", err
	}
	if err = buildMediaContainer(p, root, artifacts, payloadID, work, id); err != nil {
		return "", "", err
	}
	buildDir := filepath.Join(work, "builds", payloadID, p.Arch)
	return buildDir, id, nil
}

type mediaMeta struct {
	OSTreeCommit string `json:"ostree-commit"`
	Images       map[string]struct {
		Path, SHA256 string
		Size         int64
	}
}

func verifyMetaImages(buildDir string, images map[string]struct {
	Path, SHA256 string
	Size         int64
},
) error {
	for _, name := range []string{"ostree", "oci-manifest", "live-iso", "live-rootfs", "live-initramfs"} {
		file, ok := images[name]
		if !ok || filepath.Base(file.Path) != file.Path {
			return errors.New("missing native media output")
		}
		h, err := build.HashFile(filepath.Join(buildDir, file.Path))
		if err != nil || h != file.SHA256 {
			return errors.New("native output checksum mismatch")
		}
	}
	return nil
}

func verifyBuildMeta(buildDir string, candidate deliver.Candidate) (mediaMeta, error) {
	var meta mediaMeta
	b, err := os.ReadFile(filepath.Join(buildDir, "meta.json"))
	if err != nil {
		return meta, err
	}
	if err = json.Unmarshal(b, &meta); err != nil {
		return meta, err
	}
	if err = verifyMetaImages(buildDir, meta.Images); err != nil {
		return meta, err
	}
	if meta.Images["ostree"].SHA256 != candidate.HostArchiveSHA256 || meta.Images["oci-manifest"].SHA256 != strings.TrimPrefix(candidate.Host.Manifest, "sha256:") {
		return meta, errors.New("native import changed candidate")
	}
	return meta, nil
}

func setupMediaRootfs(artifacts, buildDir, rootfsPath, rootfsSHA string) (string, string, error) {
	mediaDir := filepath.Join(artifacts, "media")
	if err := os.Mkdir(mediaDir, 0o755); err != nil {
		return "", "", err
	}
	rootfsName := rootfsSHA + "-rootfs.img"
	if err := os.Link(filepath.Join(buildDir, rootfsPath), filepath.Join(mediaDir, rootfsName)); err != nil {
		return "", "", err
	}
	return mediaDir, rootfsName, nil
}

func verifyCustomizedISO(native func(string, ...string) (string, error), artifacts, rootfsURL string) error {
	ignition, err := native("/usr/bin/coreos-installer", "iso", "ignition", "show", "/out/media/installer.iso")
	if err != nil {
		return err
	}
	expected, err := os.ReadFile(filepath.Join(artifacts, "live.ign"))
	if err != nil {
		return err
	}
	// iso customize wraps the supplied fragment in a merge source. Require the
	// exact public fragment to survive native readback (checked below).
	if err = VerifyLiveIgnition([]byte(ignition), expected); err != nil {
		return err
	}
	kargs, err := native("/usr/bin/coreos-installer", "iso", "kargs", "show", "/out/media/installer.iso")
	if err != nil {
		return err
	}
	if !strings.Contains(kargs, "coreos.live.rootfs_url="+rootfsURL) || strings.Contains(kargs, "coreos.liveiso") {
		return errors.New("minimal-media kernel arguments differ")
	}
	return nil
}

func customizeInstallerISO(p build.Production, root, artifacts, buildDir, id, rootfsURL, liveISOPath string) (string, error) {
	native := func(name string, args ...string) (string, error) {
		prefix := []string{"--remote=false", "run", "--rm", "--pull=never", "--network=none", "--read-only", "--cap-drop=all", "--security-opt=label=disable", "--volume=" + buildDir + ":/build:ro", "--volume=" + artifacts + ":/out:rw", "--entrypoint=" + name, id}
		return p.Capture(root, "podman", append(prefix, args...)...)
	}
	version, err := native("/usr/bin/coreos-installer", "--version")
	if err != nil {
		return "", err
	}
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "coreos-installer ") || strings.ContainsAny(version, "\r\n") {
		return "", errors.New("unrecognized installer version output")
	}
	if _, err = native("/usr/bin/coreos-installer", "iso", "extract", "minimal-iso", "/build/"+liveISOPath, "/out/media/minimal.iso"); err != nil {
		return "", err
	}
	if _, err = native("/usr/bin/coreos-installer", "iso", "customize", "--live-ignition", "/out/live.ign", "--live-karg-append", "coreos.live.rootfs_url="+rootfsURL, "--output", "/out/media/installer.iso", "/out/media/minimal.iso"); err != nil {
		return "", err
	}
	if err = verifyCustomizedISO(native, artifacts, rootfsURL); err != nil {
		return "", err
	}
	return version, nil
}

func verifyMediaReadback(p build.Production, root, artifacts, buildDir, id, mediaDir, rootfsName string) error {
	native := func(name string, args ...string) (string, error) {
		prefix := []string{"--remote=false", "run", "--rm", "--pull=never", "--network=none", "--read-only", "--cap-drop=all", "--security-opt=label=disable", "--volume=" + buildDir + ":/build:ro", "--volume=" + artifacts + ":/out:rw", "--entrypoint=" + name, id}
		return p.Capture(root, "podman", append(prefix, args...)...)
	}
	readback := filepath.Join(mediaDir, "readback")
	if err := os.Mkdir(readback, 0o755); err != nil {
		return err
	}
	if _, err := native("/usr/bin/coreos-installer", "iso", "extract", "pxe", "--output-dir", "/out/media/readback", "/out/media/installer.iso"); err != nil {
		return err
	}
	initrds, err := filepath.Glob(filepath.Join(readback, "*-initrd.img"))
	if err != nil || len(initrds) != 1 {
		return errors.New("missing ISO initrd readback")
	}
	if _, err = native("/usr/bin/coreos-installer", "dev", "extract", "initrd", "--directory", "/out/media/readback", "/out/media/readback/"+filepath.Base(initrds[0]), "etc/coreos-live-want-rootfs"); err != nil {
		return err
	}
	chunks, err := os.ReadFile(filepath.Join(readback, "etc/coreos-live-want-rootfs"))
	if err != nil {
		return err
	}
	return VerifyRootfsChunks(filepath.Join(mediaDir, rootfsName), string(chunks))
}

func prepareAndVerifyMedia(p build.Production, root, artifacts, buildDir, id, mediaDir string, meta mediaMeta, rootfsURL, rootfsName string) (string, error) {
	version, err := customizeInstallerISO(p, root, artifacts, buildDir, id, rootfsURL, meta.Images["live-iso"].Path)
	if err != nil {
		return "", err
	}
	if err = verifyMediaReadback(p, root, artifacts, buildDir, id, mediaDir, rootfsName); err != nil {
		return "", err
	}
	return version, nil
}

func sealMedia(artifacts, mediaDir, rootfsName, revision, arch, manifest, payloadSHA256, ostreeCommit, rootfsURL, compression, filesystem, fsoptions string, lock mediaLock) (Media, error) {
	iso, err := mediaFile(filepath.Join(mediaDir, "installer.iso"))
	if err != nil {
		return Media{}, err
	}
	if iso.Bytes >= 2000000000 {
		return Media{}, errors.New("minimal ISO exceeds distribution ceiling")
	}
	rf, err := mediaFile(filepath.Join(mediaDir, rootfsName))
	if err != nil {
		return Media{}, err
	}
	console, err := build.HashFile(filepath.Join(artifacts, "tools/soda-installer"))
	if err != nil {
		return Media{}, err
	}
	result := Media{
		Scope:                 "candidate-derived media; native qualification and final protected release signing pending",
		Revision:              revision,
		Architecture:          arch,
		HostManifest:          manifest,
		PayloadSHA256:         payloadSHA256,
		ConsoleSHA256:         console,
		AssemblerImportCommit: ostreeCommit,
		RootfsURL:             rootfsURL,
		ISO:                   iso,
		Rootfs:                rf,
		Tools:                 lock,
		CompressionMode:       compression,
		RootfsFilesystem:      filesystem,
		RootfsOptions:         fsoptions,
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return Media{}, err
	}
	err = build.WriteNew(filepath.Join(mediaDir, "media.json"), append(data, '\n'), 0o644)
	return result, err
}

func assembleMedia(ctx context.Context, p build.Production, r Request, lock mediaLock, next func(string) error) (Media, error) {
	root := filepath.Join(r.Out, "work/media")
	artifacts := p.Out
	in, err := admitMediaInputs(r, artifacts, p.Arch, p.Revision)
	if err != nil {
		return Media{}, err
	}
	if err = next("P7 / Authenticate packaging inputs"); err != nil {
		return Media{}, err
	}
	if err = authenticatePackagingInputs(ctx, r, root, artifacts, in.archive, in.observed.Manifest, in.authority, in.trust, lock); err != nil {
		return Media{}, err
	}
	if err = next("P8 / Assemble candidate-derived native media"); err != nil {
		return Media{}, err
	}
	buildDir, id, err := assembleNativeMedia(p, root, artifacts, in.payload.ID)
	if err != nil {
		return Media{}, err
	}
	meta, err := verifyBuildMeta(buildDir, in.candidate)
	if err != nil {
		return Media{}, err
	}
	mediaDir, rootfsName, err := setupMediaRootfs(artifacts, buildDir, meta.Images["live-rootfs"].Path, meta.Images["live-rootfs"].SHA256)
	if err != nil {
		return Media{}, err
	}
	rootfsURL := strings.TrimRight(r.RootfsBaseURL, "/") + "/" + rootfsName
	installerVersion, err := prepareAndVerifyMedia(p, root, artifacts, buildDir, id, mediaDir, meta, rootfsURL, rootfsName)
	if err != nil {
		return Media{}, err
	}
	lock.Installer = installerVersion
	return sealMedia(artifacts, mediaDir, rootfsName, p.Revision, p.Arch, in.candidate.Host.Manifest, in.candidate.PayloadSHA256, meta.OSTreeCommit, rootfsURL, r.MediaCompression, in.filesystem, in.fsoptions, lock)
}

// VerifyLiveIgnition checks the native customization readback against the exact
// fragment supplied by this build. Ignition's serializer emits absent optionals as null.
func liveIgnitionBytes(data []byte) ([]byte, error) {
	var wrapper struct {
		Ignition struct {
			Config struct {
				Merge []struct{ Source, Compression string }
			}
		}
	}
	if json.Unmarshal(data, &wrapper) != nil || len(wrapper.Ignition.Config.Merge) != 1 {
		return nil, errors.New("unexpected native live Ignition")
	}
	fragment := wrapper.Ignition.Config.Merge[0]
	if !strings.HasPrefix(fragment.Source, "data:;base64,") || fragment.Compression != "gzip" {
		return nil, errors.New("unexpected native Ignition encoding")
	}
	compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(fragment.Source, "data:;base64,"))
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, 2<<20))
}

func VerifyLiveIgnition(data, expected []byte) error {
	raw, err := liveIgnitionBytes(data)
	if err != nil {
		return err
	}
	var got, want any
	if json.Unmarshal(raw, &got) != nil || json.Unmarshal(expected, &want) != nil {
		return errors.New("invalid Ignition readback")
	}
	omitNullFields(got)
	omitNullFields(want)
	if !reflect.DeepEqual(got, want) {
		return errors.New("embedded live Ignition differs")
	}
	return nil
}

func omitNullFields(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if child == nil {
				delete(v, key)
			} else {
				omitNullFields(child)
			}
		}
	case []any:
		for _, child := range v {
			omitNullFields(child)
		}
	}
}

func mediaFile(path string) (MediaFile, error) {
	st, err := os.Stat(path)
	if err != nil {
		return MediaFile{}, err
	}
	h, err := build.HashFile(path)
	return MediaFile{Path: filepath.Base(path), SHA256: h, Bytes: st.Size()}, err
}

// VerifyRootfsChunks checks the bootstrap's native chunk list against the download.
func matchRootfsChunk(f *os.File, want []string, i int) (int, bool, error) {
	h := sha256.New()
	n, e := io.CopyN(h, f, 2<<20)
	if e != nil && e != io.EOF {
		return i, false, e
	}
	if n == 0 {
		return i, false, nil
	}
	if i >= len(want) || hex.EncodeToString(h.Sum(nil)) != want[i] {
		return i, false, errors.New("rootfs differs from native bootstrap hashes")
	}
	i++
	return i, e != io.EOF, nil
}

func VerifyRootfsChunks(path, text string) error {
	if !strings.HasPrefix(text, "stream-hash sha256 2097152\n") {
		return errors.New("unexpected native rootfs hash format")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	want := strings.Fields(strings.TrimPrefix(text, "stream-hash sha256 2097152\n"))
	i := 0
	for {
		var more bool
		i, more, err = matchRootfsChunk(f, want, i)
		if err != nil {
			return err
		}
		if !more {
			break
		}
	}
	if i == 0 || i != len(want) {
		return errors.New("incomplete rootfs bootstrap hashes")
	}
	return nil
}
