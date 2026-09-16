package image

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
)

func readHostImageID(iid string) (string, error) {
	data, err := os.ReadFile(iid)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(data))
	if !strings.HasPrefix(id, "sha256:") || !build.Digest(strings.TrimPrefix(id, "sha256:")) {
		return "", errors.New("invalid built host image ID")
	}
	return id, nil
}

func inspectHostIdentity(context, platform, revision, scope, id string, p build.Production) error {
	observed, err := p.Capture(context, "podman", "--remote=false", "image", "inspect", "--format", `{{.Os}}/{{.Architecture}} {{index .Labels "org.opencontainers.image.revision"}} {{index .Labels "io.soda.host-image.scope"}}`, id)
	if err != nil {
		return err
	}
	if observed != "linux/"+platform+" "+revision+" "+scope {
		return errors.New("built host identity/platform mismatch")
	}
	return nil
}

// recordHostPackages files the built host inventory as the bill-of-materials.
// The install floats on bare names, so nothing precedes the observation: the
// image's own inventory is validated for shape and recorded, and its SHA256
// becomes the payload fingerprint.
func recordHostPackages(context, out, id string, p build.Production) (string, error) {
	packages, err := p.Capture(context, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "inspect.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--entrypoint=/bin/sh", id, "-ec", `test "$(stat -c %a /usr/libexec/soda/soda-host)" = 755
 test -L /usr/sbin
 test "$(readlink /usr/sbin)" = bin
 for name in grub2-install soda-setup soda-activate; do test -x /usr/sbin/$name; done
 test -f /usr/lib/systemd/system/soda-project@.service
 test -f /usr/share/containers/systemd/forgejo.container
 for file in /etc/subuid /etc/subgid; do test "$(cat "$file")" = 'containers:1000000:268435456'; done
 test ! -e /usr/local/libexec/soda/soda-host
 test ! -e /etc/soda/host.json
 test ! -e /etc/soda/dashboard.json
 test ! -e /etc/zincati/config.d/90-soda-image.toml
 test ! -e /usr/lib/bootc/bound-images.d/forgejo.container
 rpm -q rpm-ostree zincati ignition cockpit-ostree tailscale >/dev/null
 cat /usr/share/soda/host-image/packages.txt`)
	if err != nil {
		return "", fmt.Errorf("read-only host inspection failed; retain inspect.cid: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(packages), "\n")
	if err := validRPMInventory(lines); err != nil {
		return "", err
	}
	record := []byte(strings.Join(lines, "\n") + "\n")
	if err := ownedWrite(filepath.Join(context, "packages.recorded"), record, 0o644); err != nil {
		return "", err
	}
	if err := build.WriteNew(filepath.Join(out, "packages.txt"), record, 0o600); err != nil {
		return "", err
	}
	return hashBytes(record), nil
}

func exportHostArchive(context, out, arch, revision, prefix, pinned, id string, p build.Production) error {
	if err := p.Next("P6 / Export and verify host OCI"); err != nil {
		return err
	}
	archive := filepath.Join(out, "host.oci")
	if err := p.Execute(context, "podman", "--remote=false", "save", "--format=oci-archive", "--output", archive, id); err != nil {
		return err
	}
	host, err := build.InspectOCI(archive, arch, revision)
	if err != nil {
		return err
	}
	if host.Config != id || host.BaseName != pinned || host.BaseDigest != strings.SplitN(pinned, "@", 2)[1] {
		return errors.New("host export identity/base mismatch")
	}
	hash, err := build.HashFile(archive)
	if err != nil {
		return err
	}
	return recordCandidate(out, prefix, host, hash)
}

// buildHostImage bakes one host image from the context as staged. The first
// pass stages no release metadata: its only job is package observation for
// the bill-of-materials. The final pass builds the sealed context.
func buildHostImage(context, out, arch, revision, prefix string, base Base, p build.Production) (string, error) {
	platform, _ := build.OCIArchitecture(arch)
	pinned := base.Images[arch]
	iid := filepath.Join(out, "host.iid")
	const scope = "complete-local-payload"
	if err := p.Execute(context, "podman", "--remote=false", "build", "--pull=never", "--rm=false", "--platform=linux/"+platform, "--build-arg=BASE_IMAGE="+pinned, "--build-arg=PAYLOAD_SCOPE="+scope, "--label=org.opencontainers.image.revision="+revision, "--label=org.opencontainers.image.base.name="+pinned, "--label=org.opencontainers.image.base.digest="+strings.SplitN(pinned, "@", 2)[1], "--label=org.opencontainers.image.version="+base.Release+".soda-"+revision[:12], "--iidfile", iid, "--file", "Containerfile", "."); err != nil {
		return "", err
	}
	return readHostImageID(iid)
}

// verifyBuiltHost inspects the final sealed-context image: identity, a
// package inventory that must match the sealed fingerprint, image config,
// embedded payload/content/Quadlets, then OCI export.
func verifyBuiltHost(context, out, arch, revision, prefix, id, packageHash string, base Base, p build.Production, phase func(string) error) error {
	platform, _ := build.OCIArchitecture(arch)
	const scope = "complete-local-payload"
	if err := phase("P6 / Verify native host identity and content"); err != nil {
		return err
	}
	if err := inspectHostIdentity(context, platform, revision, scope, id, p); err != nil {
		return err
	}
	observed, err := recordHostPackages(context, out, id, p)
	if err != nil {
		return err
	}
	if observed != packageHash {
		return errors.New("host inventory shifted during build; refusing mismatched fingerprint")
	}
	imageConfig, err := p.Capture(context, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "image-config-inspect.cid"), "--network=none", "--read-only", "--cap-drop=all", "--entrypoint=/usr/bin/cat", id, "/"+imageConfigPath)
	if err != nil {
		return err
	}
	if err = recordImageConfig(context, out, imageConfig); err != nil {
		return err
	}
	if err = inspectComplete(context, out, id, p.Capture); err != nil {
		return err
	}
	return exportHostArchive(context, out, arch, revision, prefix, base.Images[arch], id, p)
}
