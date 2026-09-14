package hostimage

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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
)

type mediaLock struct{ Assembler, Config, Architecture, Installer string }

// Signing inputs belong to the invoking authority, never the source snapshot or
// child build environment. Local fixture authority does not prove job isolation;
// protected worker commissioning and final qualification remain B4/B5 work.
type MediaAuthority struct {
	Trust string
	Keys  rd.SecretFiles
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
}

func mediaBaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(value, "\r\n ") {
		return errors.New("explicit public HTTP(S) rootfs base URL required")
	}
	return nil
}

func prepareAssembler(p nativebuild.Production, root string) (mediaLock, error) {
	var lock mediaLock
	if err := nativebuild.ReadJSON(filepath.Join(p.Source, "appliance/locks/media-tools.json"), &lock); err != nil {
		return lock, err
	}
	const prefix = "quay.io/coreos-assembler/coreos-assembler@sha256:"
	if lock.Architecture != p.Arch || !strings.HasPrefix(lock.Assembler, prefix) || !nativebuild.Digest(strings.TrimPrefix(lock.Assembler, prefix)) || !nativebuild.Revision(lock.Config) || lock.Installer != "coreos-installer 0.26.0" {
		return lock, errors.New("unreviewed media tools")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		return lock, err
	}
	run := func(cmd string, args ...string) error { return p.Execute(root, cmd, args...) }
	for _, args := range [][]string{{"init", "config-repo"}, {"-C", "config-repo", "fetch", "--depth=1", "https://github.com/coreos/fedora-coreos-config.git", lock.Config}, {"-C", "config-repo", "archive", "--format=tar", "--output", filepath.Join(root, "config.tar"), lock.Config}} {
		if err := run("git", args...); err != nil {
			return lock, err
		}
	}
	config := filepath.Join(root, "config")
	if err := os.Mkdir(config, 0755); err != nil {
		return lock, err
	}
	if err := run("tar", "-xf", "config.tar", "-C", config, "--no-same-owner"); err != nil {
		return lock, err
	}
	argsFile := filepath.Join(config, "build-args.conf")
	b, err := os.ReadFile(argsFile)
	if err != nil {
		return lock, err
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
		return lock, errors.New("missing upstream buildroot selection")
	}
	if err = os.WriteFile(argsFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return lock, err
	}
	if err = run("podman", "--remote=false", "pull", lock.Assembler); err != nil {
		return lock, err
	}
	if err = nativebuild.WriteNew(filepath.Join(root, "Containerfile"), []byte("FROM "+lock.Assembler+"\nUSER 0\n"), 0644); err != nil {
		return lock, err
	}
	if err = run("podman", "--remote=false", "build", "--pull=never", "--network=none", "--iidfile", "builder.iid", "."); err != nil {
		return lock, err
	}
	id, err := builderID(root)
	if err != nil {
		return lock, err
	}
	original, err := p.Capture(root, "podman", "--remote=false", "image", "inspect", "--format", "{{json .RootFS.Layers}}", lock.Assembler)
	if err != nil {
		return lock, err
	}
	wrapped, err := p.Capture(root, "podman", "--remote=false", "image", "inspect", "--format", "{{json .RootFS.Layers}}", id)
	if err != nil {
		return lock, err
	}
	if original != wrapped || original == "" {
		return lock, errors.New("assembler wrapper changed rootfs")
	}
	err = run("podman", "--remote=false", "save", "--format=oci-archive", "--output", "assembler-root.oci", id)
	return lock, err
}
func builderID(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "builder.iid"))
	id := strings.TrimSpace(string(b))
	if err != nil || !strings.HasPrefix(id, "sha256:") || !nativebuild.Digest(strings.TrimPrefix(id, "sha256:")) {
		return "", errors.New("missing exact assembler image")
	}
	return id, nil
}

func signMediaInput(ctx context.Context, authority MediaAuthority, trust rd.Trust, input, transport, repository, digest, out string) error {
	runner := rd.Native{Home: filepath.Dir(authority.Trust)}
	permit := rd.Permit{Format: 1, Repository: repository, Digest: digest, Expires: time.Now().Add(time.Hour).Unix()}
	if err := rd.Sign(ctx, runner, trust, permit, transport, input, out, authority.Keys); err != nil {
		return err
	}
	return rd.VerifyCopy(ctx, runner, trust, repository+"@"+digest, "dir:"+filepath.Join(out, "signed"), out+"-admitted")
}

func assembleMedia(ctx context.Context, p nativebuild.Production, r Request, lock mediaLock, next func(string) error) (Media, error) {
	var result Media
	root := filepath.Join(r.Out, "work/media")
	artifacts := p.Out
	var candidate rd.Candidate
	var payload appliancerelease.Payload
	var authority MediaAuthority
	var trust rd.Trust
	if err := rd.ReadJSON(r.MediaAuthority, &authority); err != nil {
		return result, err
	}
	if err := rd.ReadJSON(authority.Trust, &trust); err != nil {
		return result, err
	}
	if trust.Validate() != nil || trust.Prefix != r.RepositoryPrefix {
		return result, errors.New("media authority does not match intended repositories")
	}
	if err := rd.ReadJSON(filepath.Join(artifacts, "candidate.json"), &candidate); err != nil {
		return result, err
	}
	payload, err := appliancerelease.Load(filepath.Join(artifacts, "payload.json"))
	if err != nil {
		return result, err
	}
	payloadBytes, err := os.ReadFile(filepath.Join(artifacts, "payload.json"))
	if err != nil {
		return result, err
	}
	if err = candidate.Validate(payload, payloadBytes); err != nil {
		return result, err
	}
	archive := filepath.Join(artifacts, "host.oci")
	observed, err := nativebuild.InspectOCI(archive, p.Arch, p.Revision)
	if err != nil || observed.Manifest != candidate.Host.Manifest {
		return result, errors.New("media candidate identity mismatch")
	}
	hash, err := nativebuild.HashFile(archive)
	if err != nil || hash != candidate.HostArchiveSHA256 {
		return result, errors.New("media candidate archive mismatch")
	}
	if err = next("P7 / Authenticate packaging inputs"); err != nil {
		return result, err
	}
	if err = rd.CheckNative(ctx, rd.Native{Home: filepath.Dir(authority.Trust)}); err != nil {
		return result, err
	}
	if err = signMediaInput(ctx, authority, trust, archive, "oci-archive", r.RepositoryPrefix+"-host", observed.Manifest, filepath.Join(root, "host-signature")); err != nil {
		return result, err
	}
	// The signed public inventory authenticates exactly the auxiliary inputs used
	// by the native packager. No source script runs in this signing operation.
	inventory := map[string]string{}
	for _, dir := range []string{filepath.Join(root, "config"), filepath.Join(artifacts, "tools")} {
		if err = filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			h, e := nativebuild.HashFile(path)
			if e == nil {
				rel, _ := filepath.Rel(r.Out, path)
				inventory[rel] = h
			}
			return e
		}); err != nil {
			return result, err
		}
	}
	for _, path := range []string{filepath.Join(root, "assembler-root.oci"), filepath.Join(root, "config.tar"), filepath.Join(artifacts, "live.ign"), filepath.Join(artifacts, "destination.ign"), filepath.Join(artifacts, "candidate.json"), filepath.Join(artifacts, "payload.json")} {
		h, e := nativebuild.HashFile(path)
		if e != nil {
			return result, e
		}
		rel, _ := filepath.Rel(r.Out, path)
		inventory[rel] = h
	}
	inputDocument := struct {
		Tools mediaLock
		Files map[string]string
	}{lock, inventory}
	document := filepath.Join(root, "input-document")
	digest, err := rd.WriteDocument(document, inputDocument)
	if err != nil {
		return result, err
	}
	if err = signMediaInput(ctx, authority, trust, document, "dir", r.RepositoryPrefix+"-media", digest, filepath.Join(root, "input-signature")); err != nil {
		return result, err
	}
	for path, want := range inventory {
		got, e := nativebuild.HashFile(filepath.Join(r.Out, path))
		if e != nil || got != want {
			return result, errors.New("packaging input changed after admission")
		}
	}
	if err = next("P8 / Assemble candidate-derived native media"); err != nil {
		return result, err
	}
	work := filepath.Join(root, "work")
	if err = os.Mkdir(work, 0755); err != nil {
		return result, err
	}
	id, err := builderID(root)
	if err != nil {
		return result, err
	}
	base := []string{"--remote=false", "run", "--rm", "--cidfile=" + filepath.Join(root, "packaging.cid"), "--network=none", "--privileged", "--security-opt=label=disable", "--device=/dev/kvm", "--device=/dev/fuse", "--cpus=4", "--memory=16g", "--env=COSA_SUPERMIN_MEMORY=12288", "--env=RUNVM_NONET=1", "--volume=" + work + ":/srv:rw", "--volume=" + filepath.Join(root, "config") + ":/config:ro", "--volume=" + artifacts + ":/inputs:ro", "--volume=" + filepath.Join(root, "assembler-root.oci") + ":/assembler-root.oci:ro", "--workdir=/srv", "--entrypoint=/usr/bin/taskset", id, "-c", "0-3", "/bin/bash", "-euc"}
	script := `umask 0022; cosa init /config; cp /assembler-root.oci /srv/tmp/assembler-root.oci; cosa import --skip-prune oci-archive:/inputs/host.oci; cosa buildextend-live --build "$1"`
	if err = p.Execute(root, "podman", append(base, script, "assemble", payload.ID)...); err != nil {
		// Podman owns the helper VM's namespace, outside the CLI process group.
		// Stop only this run's container even when the build context is cancelled.
		if cid, e := os.ReadFile(filepath.Join(root, "packaging.cid")); e == nil && nativebuild.Digest(strings.TrimSpace(string(cid))) {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cmd := exec.CommandContext(cleanup, "/usr/bin/podman", "--remote=false", "stop", "--time=10", strings.TrimSpace(string(cid)))
			cmd.Env = buildEnvironment()
			_ = cmd.Run()
			cancel()
		}
		return result, err
	}
	build := filepath.Join(work, "builds", payload.ID, p.Arch)
	var meta struct {
		OSTreeCommit string `json:"ostree-commit"`
		Images       map[string]struct {
			Path, SHA256 string
			Size         int64
		}
	}
	b, err := os.ReadFile(filepath.Join(build, "meta.json"))
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(b, &meta); err != nil {
		return result, err
	}
	for _, name := range []string{"ostree", "oci-manifest", "live-iso", "live-rootfs", "live-initramfs"} {
		file, ok := meta.Images[name]
		if !ok || filepath.Base(file.Path) != file.Path {
			return result, errors.New("missing native media output")
		}
		h, e := nativebuild.HashFile(filepath.Join(build, file.Path))
		if e != nil || h != file.SHA256 {
			return result, errors.New("native output checksum mismatch")
		}
	}
	if meta.Images["ostree"].SHA256 != candidate.HostArchiveSHA256 || meta.Images["oci-manifest"].SHA256 != strings.TrimPrefix(candidate.Host.Manifest, "sha256:") {
		return result, errors.New("native import changed candidate")
	}
	mediaDir := filepath.Join(artifacts, "media")
	if err = os.Mkdir(mediaDir, 0755); err != nil {
		return result, err
	}
	rootfs := meta.Images["live-rootfs"]
	rootfsName := rootfs.SHA256 + "-rootfs.img"
	if err = os.Link(filepath.Join(build, rootfs.Path), filepath.Join(mediaDir, rootfsName)); err != nil {
		return result, err
	}
	rootfsURL := strings.TrimRight(r.RootfsBaseURL, "/") + "/" + rootfsName
	native := func(name string, args ...string) (string, error) {
		prefix := []string{"--remote=false", "run", "--rm", "--pull=never", "--network=none", "--read-only", "--cap-drop=all", "--security-opt=label=disable", "--volume=" + build + ":/build:ro", "--volume=" + artifacts + ":/out:rw", "--entrypoint=" + name, id}
		return p.Capture(root, "podman", append(prefix, args...)...)
	}
	version, err := native("/usr/bin/coreos-installer", "--version")
	if err != nil || version != lock.Installer {
		return result, errors.New("media installer version mismatch")
	}
	if _, err = native("/usr/bin/coreos-installer", "iso", "extract", "minimal-iso", "/build/"+meta.Images["live-iso"].Path, "/out/media/minimal.iso"); err != nil {
		return result, err
	}
	if _, err = native("/usr/bin/coreos-installer", "iso", "customize", "--live-ignition", "/out/live.ign", "--live-karg-append", "coreos.live.rootfs_url="+rootfsURL, "--output", "/out/media/installer.iso", "/out/media/minimal.iso"); err != nil {
		return result, err
	}
	ignition, err := native("/usr/bin/coreos-installer", "iso", "ignition", "show", "/out/media/installer.iso")
	if err != nil {
		return result, err
	}
	expected, err := os.ReadFile(filepath.Join(artifacts, "live.ign"))
	if err != nil {
		return result, err
	}
	// iso customize wraps the supplied fragment in a merge source. Require the
	// exact public fragment to survive native readback (checked below).
	if err = verifyLiveIgnition([]byte(ignition), expected); err != nil {
		return result, err
	}
	kargs, err := native("/usr/bin/coreos-installer", "iso", "kargs", "show", "/out/media/installer.iso")
	if err != nil {
		return result, err
	}
	if !strings.Contains(kargs, "coreos.live.rootfs_url="+rootfsURL) || strings.Contains(kargs, "coreos.liveiso") {
		return result, errors.New("minimal-media kernel arguments differ")
	}
	readback := filepath.Join(mediaDir, "readback")
	if err = os.Mkdir(readback, 0755); err != nil {
		return result, err
	}
	if _, err = native("/usr/bin/coreos-installer", "iso", "extract", "pxe", "--output-dir", "/out/media/readback", "/out/media/installer.iso"); err != nil {
		return result, err
	}
	initrds, err := filepath.Glob(filepath.Join(readback, "*-initrd.img"))
	if err != nil || len(initrds) != 1 {
		return result, errors.New("missing ISO initrd readback")
	}
	if _, err = native("/usr/bin/coreos-installer", "dev", "extract", "initrd", "--directory", "/out/media/readback", "/out/media/readback/"+filepath.Base(initrds[0]), "etc/coreos-live-want-rootfs"); err != nil {
		return result, err
	}
	chunks, err := os.ReadFile(filepath.Join(readback, "etc/coreos-live-want-rootfs"))
	if err != nil {
		return result, err
	}
	if err = verifyRootfsChunks(filepath.Join(mediaDir, rootfsName), string(chunks)); err != nil {
		return result, err
	}
	iso, err := mediaFile(filepath.Join(mediaDir, "installer.iso"))
	if err != nil {
		return result, err
	}
	if iso.Bytes >= 2000000000 {
		return result, errors.New("minimal ISO exceeds distribution ceiling")
	}
	rf, err := mediaFile(filepath.Join(mediaDir, rootfsName))
	if err != nil {
		return result, err
	}
	console, err := nativebuild.HashFile(filepath.Join(artifacts, "tools/soda-installer"))
	if err != nil {
		return result, err
	}
	result = Media{Scope: "candidate-derived media; native qualification and final protected release signing pending", Revision: p.Revision, Architecture: p.Arch, HostManifest: candidate.Host.Manifest, PayloadSHA256: candidate.PayloadSHA256, ConsoleSHA256: console, AssemblerImportCommit: meta.OSTreeCommit, RootfsURL: rootfsURL, ISO: iso, Rootfs: rf, Tools: lock}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	err = nativebuild.WriteNew(filepath.Join(mediaDir, "media.json"), append(data, '\n'), 0644)
	return result, err
}

func verifyLiveIgnition(data, expected []byte) error {
	var wrapper struct {
		Ignition struct {
			Config struct {
				Merge []struct{ Source, Compression string }
			}
		}
	}
	if json.Unmarshal(data, &wrapper) != nil || len(wrapper.Ignition.Config.Merge) != 1 {
		return errors.New("unexpected native live Ignition")
	}
	fragment := wrapper.Ignition.Config.Merge[0]
	if !strings.HasPrefix(fragment.Source, "data:;base64,") || fragment.Compression != "gzip" {
		return errors.New("unexpected native Ignition encoding")
	}
	compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(fragment.Source, "data:;base64,"))
	if err != nil {
		return err
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return err
	}
	defer reader.Close()
	raw, err := io.ReadAll(io.LimitReader(reader, 2<<20))
	if err != nil {
		return err
	}
	var got, want any
	if json.Unmarshal(raw, &got) != nil || json.Unmarshal(expected, &want) != nil || !reflect.DeepEqual(got, want) {
		return errors.New("embedded live Ignition differs")
	}
	return nil
}
func mediaFile(path string) (MediaFile, error) {
	st, err := os.Stat(path)
	if err != nil {
		return MediaFile{}, err
	}
	h, err := nativebuild.HashFile(path)
	return MediaFile{Path: filepath.Base(path), SHA256: h, Bytes: st.Size()}, err
}
func verifyRootfsChunks(path, text string) error {
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
		h := sha256.New()
		n, e := io.CopyN(h, f, 2<<20)
		if e != nil && e != io.EOF {
			return e
		}
		if n == 0 {
			break
		}
		if i >= len(want) || hex.EncodeToString(h.Sum(nil)) != want[i] {
			return errors.New("rootfs differs from native bootstrap hashes")
		}
		i++
		if e == io.EOF {
			break
		}
	}
	if i == 0 || i != len(want) {
		return errors.New("incomplete rootfs bootstrap hashes")
	}
	return nil
}
