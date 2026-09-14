package hostimage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

func buildHost(context, out, arch, revision, prefix string, base Base, p nativebuild.Production, phase func(string) error) error {
	platform, _ := nativebuild.OCIArchitecture(arch)
	pinned := base.Images[arch]
	iid := filepath.Join(out, "host.iid")
	const scope = "complete-local-payload"
	if err := p.Execute(context, "podman", "--remote=false", "build", "--pull=never", "--rm=false", "--platform=linux/"+platform, "--build-arg=BASE_IMAGE="+pinned, "--build-arg=PAYLOAD_SCOPE="+scope, "--label=org.opencontainers.image.revision="+revision, "--label=org.opencontainers.image.base.name="+pinned, "--label=org.opencontainers.image.base.digest="+strings.SplitN(pinned, "@", 2)[1], "--label=org.opencontainers.image.version="+base.Release+".soda-"+revision[:12], "--iidfile", iid, "--file", "Containerfile", "."); err != nil {
		return err
	}
	data, err := os.ReadFile(iid)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(string(data))
	if !strings.HasPrefix(id, "sha256:") || !nativebuild.Digest(strings.TrimPrefix(id, "sha256:")) {
		return errors.New("invalid built host image ID")
	}
	if err = phase("P6 / Verify native host identity and content"); err != nil {
		return err
	}
	observed, err := p.Capture(context, "podman", "--remote=false", "image", "inspect", "--format", `{{.Os}}/{{.Architecture}} {{index .Labels "org.opencontainers.image.revision"}} {{index .Labels "io.soda.host-image.scope"}}`, id)
	if err != nil {
		return err
	}
	if observed != "linux/"+platform+" "+revision+" "+scope {
		return errors.New("built host identity/platform mismatch")
	}
	packages, err := p.Capture(context, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "inspect.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--entrypoint=/bin/sh", id, "-ec", `test "$(stat -c %a /usr/libexec/soda/soda-host)" = 755
 test -L /usr/sbin
 test "$(readlink /usr/sbin)" = bin
 for name in grub2-install soda-setup soda-activate; do test -x /usr/sbin/$name; done
 test -f /usr/lib/systemd/system/soda-project@.service
 test -f /usr/share/containers/systemd/forgejo.container
 test ! -e /usr/local/libexec/soda/soda-host
 test ! -e /etc/soda/host.json
 test ! -e /etc/soda/dashboard.json
 test ! -e /etc/zincati/config.d/90-soda-image.toml
 test ! -e /usr/lib/bootc/bound-images.d/forgejo.container
 for name in dashboard forgejo proxy project-os tailnet; do test -s /usr/share/soda/images/$name.oci; done
 rpm -q rpm-ostree zincati ignition cockpit-ostree tailscale >/dev/null
 cat /usr/share/soda/host-image/packages.txt`)
	if err != nil {
		return fmt.Errorf("read-only host inspection failed; retain inspect.cid: %w", err)
	}
	expected, err := os.ReadFile(filepath.Join(context, "packages.expected"))
	if err != nil {
		return err
	}
	if packages == "" || packages != strings.TrimSpace(string(expected)) {
		return errors.New("built RPM inventory differs from the admitted transaction")
	}
	if err = nativebuild.WriteNew(filepath.Join(out, "packages.txt"), []byte(packages+"\n"), 0600); err != nil {
		return err
	}
	if err = inspectComplete(context, out, id, p.Capture); err != nil {
		return err
	}
	if err = p.Next("P6 / Export and verify host OCI"); err != nil {
		return err
	}
	archive := filepath.Join(out, "host.oci")
	if err = p.Execute(context, "podman", "--remote=false", "save", "--format=oci-archive", "--output", archive, id); err != nil {
		return err
	}
	host, err := nativebuild.InspectOCI(archive, arch, revision)
	if err != nil {
		return err
	}
	if host.Config != id || host.BaseName != pinned || host.BaseDigest != strings.SplitN(pinned, "@", 2)[1] {
		return errors.New("host export identity/base mismatch")
	}
	hash, err := nativebuild.HashFile(archive)
	if err != nil {
		return err
	}
	return recordCandidate(out, prefix, host, hash)
}
