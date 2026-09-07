package nativebuild

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Inventory identifies bytes, not product acceptance or a signed release.
type File struct {
	SHA256    string `json:"sha256,omitempty"`
	Mode      uint32 `json:"mode"`
	Link      string `json:"link,omitempty"`
	Directory bool   `json:"directory,omitempty"`
}
type Inventory struct {
	Revision     string
	Architecture string
	Files        map[string]File
	Images       map[string]Image
}

const inventoryName = "build-info.json"

func allowedPayload(p string) bool {
	// Retired React output is not a valid payload for new Soda bundles.
	if p == "rootfs/usr/local/share/soda/dashboard" || strings.HasPrefix(p, "rootfs/usr/local/share/soda/dashboard/") {
		return false
	}
	if p == "rootfs" || strings.HasPrefix(p, "rootfs/") {
		if p == "rootfs/etc" || strings.HasPrefix(p, "rootfs/etc/") {
			return publicEtcPath(p)
		}
		// The core owns the stage. Reject known runtime/private content rather than
		// interpreting application config or inventing a React payload.
		for _, bad := range []string{"/dashboard.json", "/host.json", "/install-started", "/shadow", "/gshadow", "/machine-id", "/tailscale/", "/browser-home/", "/soda-artifacts", "/soda-acceptance", "/ssh_host_", "/authorized_keys"} {
			if strings.Contains(p, bad) {
				return false
			}
		}
		switch filepath.Ext(p) {
		case ".ign", ".bu", ".key", ".pem", ".sqlite", ".db":
			return false
		}
		return p == "rootfs" || p == "rootfs/usr" || p == "rootfs/var" || p == "rootfs/var/lib" || p == "rootfs/var/lib/soda" || p == "rootfs/var/lib/soda/forgejo" || p == "rootfs/var/lib/soda/forgejo/gitea" || p == "rootfs/usr/local" || strings.HasPrefix(p, "rootfs/usr/local/") || p == "rootfs/var/lib/soda/forgejo/gitea/public" || strings.HasPrefix(p, "rootfs/var/lib/soda/forgejo/gitea/public/")
	}
	switch p {
	case "images", "tools", "tools/soda-artifacts", "install-native.sh", "images/project-os.oci", "images/dashboard.oci", "images/forgejo.oci", "images/caddy.oci", "inputs", "inputs/go.mod", "inputs/go.sum", "inputs/tea-source.toml", "inputs/github-runner-source.toml", "inputs/coreos-qemu.json", "inputs/cockpit-package.json", "inputs/cockpit-pnpm-lock.yaml", "inputs/native-build.json", "notices", "notices/README.md", "notices/tea-LICENSE":
		return true
	}
	return false
}

// P04's export boundary follows the actual /etc outputs in scripts/stage.py.
// Runtime config/credentials and unexpected files are never admitted merely
// because their filename has no private extension.
func publicEtcPath(p string) bool {
	for _, name := range []string{
		"containers/systemd/forgejo.container", "containers/systemd/soda-dashboard.container", "containers/systemd/soda-proxy.container",
		"systemd/system/soda-host.service", "systemd/system/soda-host.socket", "systemd/system/soda-project@.service", "systemd/system/soda-runner@.service", "systemd/system/cockpit.socket.d/10-soda.conf",
		"sysusers.d/soda.conf", "sysusers.d/soda-runners.conf", "tmpfiles.d/soda.conf", "tmpfiles.d/soda-runners.conf", "sysctl.d/90-soda-routing.conf",
		"pam.d/cockpit", "cockpit/cockpit.conf", "cockpit/users.override.json", "cockpit/disallowed-users", "profile.d/soda-console-welcome.sh", "motd", "soda/forgejo.env", "soda/proxy.Caddyfile",
		"cockpit/branding/branding.css", "cockpit/branding/theme.css", "cockpit/branding/palette.css", "cockpit/branding/login-background-light.svg", "cockpit/branding/login-background-dark.svg", "cockpit/branding/favicon.ico", "cockpit/branding/apple-touch-icon.png", "cockpit/branding/soda-logo-horizontal.svg", "cockpit/branding/soda-logo-horizontal-dark.svg", "cockpit/branding/soda-symbol.svg",
	} {
		full := "rootfs/etc/" + name
		if p == full || strings.HasPrefix(full, p+"/") {
			return true
		}
	}
	return false
}

func validLink(name, target string) bool {
	fixed := map[string]string{"rootfs/usr/local/bin/soda-tailnet": "/usr/local/libexec/soda/soda-tailnet", "rootfs/usr/local/sbin/soda-setup": "/usr/local/libexec/soda/soda-setup"}
	if expected, ok := fixed[name]; ok {
		return target == expected
	}
	prefix := "rootfs/usr/local/lib/soda/github-actions-runner/"
	return !filepath.IsAbs(target) && strings.HasPrefix(name, prefix) && strings.HasPrefix(filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(name), target))), prefix)
}
func tree(root string) (map[string]File, error) {
	cap, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer cap.Close()
	st, err := cap.Lstat("tools")
	if err != nil || !st.IsDir() {
		return nil, errors.New("real tools directory required")
	}
	result := map[string]File{}
	for _, top := range []string{"rootfs", "images", "tools/soda-artifacts", "install-native.sh", "inputs", "notices"} {
		err := fs.WalkDir(cap.FS(), top, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel := p
			if !allowedPayload(rel) {
				return fmt.Errorf("non-payload path refused: %s", rel)
			}
			info, err := cap.Lstat(p)
			if err != nil {
				return err
			}
			if info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
				return errors.New("set-ID payload refused")
			}
			entry := File{Mode: uint32(info.Mode().Perm()), Directory: info.IsDir()}
			switch {
			case info.IsDir():
			case info.Mode().IsRegular():
				entry.SHA256, err = HashAt(cap, p)
			case info.Mode()&os.ModeSymlink != 0:
				entry.Link, err = cap.Readlink(p)
				if err == nil && !validLink(rel, entry.Link) {
					err = errors.New("unsafe payload symlink")
				}
			default:
				err = errors.New("unsupported payload file")
			}
			if err != nil {
				return err
			}
			result[rel] = entry
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, required := range []string{"rootfs/etc/containers/systemd/forgejo.container", "rootfs/etc/containers/systemd/soda-dashboard.container", "rootfs/etc/containers/systemd/soda-proxy.container", "rootfs/etc/systemd/system/soda-host.service", "rootfs/etc/systemd/system/soda-host.socket", "rootfs/usr/local/libexec/soda/soda-dashboard", "rootfs/usr/local/libexec/soda/soda-host", "rootfs/usr/local/share/cockpit/soda-tailscale/index.html", "rootfs/usr/local/share/cockpit/soda-runners/index.html", "inputs/native-build.json", "inputs/go.mod", "inputs/go.sum", "notices/README.md", "notices/tea-LICENSE"} {
		entry, ok := result[required]
		if !ok || entry.SHA256 == "" {
			return nil, fmt.Errorf("missing core/support payload: %s", required)
		}
	}
	return result, nil
}

func verifyBuildInputs(root, arch, revision string, images map[string]Image) error {
	var record struct {
		Revision, Architecture string
		Tools                  map[string]string
		Images                 map[string]struct {
			ID, RegistryDigest string
			RepositoryDigests  []string
			RPMs               []string
			CLIs               map[string]string
		}
	}
	if err := ReadJSON(filepath.Join(root, "inputs/native-build.json"), &record); err != nil {
		return err
	}
	if record.Revision != revision || record.Architecture != arch {
		return errors.New("build inputs revision/platform mismatch")
	}
	for _, tool := range []string{"go", "node", "pnpm", "podman", "python", "kernel"} {
		if record.Tools[tool] == "" {
			return fmt.Errorf("missing build tool observation: %s", tool)
		}
	}
	for name, image := range images {
		id := strings.TrimPrefix(record.Images[name].ID, "sha256:")
		if !Digest(id) || "sha256:"+id != image.Config {
			return fmt.Errorf("build input image mismatch: %s", name)
		}
	}
	return nil
}

func Seal(root, arch, revision string) error {
	if err := RequireNative(arch); err != nil {
		return err
	}
	if !Revision(revision) {
		return errors.New("full revision required")
	}
	files, err := tree(root)
	if err != nil {
		return err
	}
	if err = inspectBinaries(root, arch, files); err != nil {
		return err
	}
	images := map[string]Image{}
	for _, name := range []string{"project-os", "dashboard", "forgejo", "caddy"} {
		rev := ""
		if name == "project-os" || name == "dashboard" {
			rev = revision
		}
		img, err := InspectOCI(filepath.Join(root, "images", name+".oci"), arch, rev)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", name, err)
		}
		images[name] = img
	}
	if err = verifyBuildInputs(root, arch, revision, images); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Inventory{revision, arch, files, images}, "", "  ")
	if err != nil {
		return err
	}
	if err = WriteNew(filepath.Join(root, inventoryName), append(data, '\n'), 0644); err != nil {
		return err
	}
	return checksums(root)
}
func Verify(root, arch, revision string) (Inventory, error) {
	var inv Inventory
	if err := ReadJSON(filepath.Join(root, inventoryName), &inv); err != nil {
		return inv, err
	}
	sum, err := HashFile(filepath.Join(root, inventoryName))
	if err != nil {
		return inv, err
	}
	sumPath := filepath.Join(root, "SHA256SUMS")
	st, err := os.Lstat(sumPath)
	if err != nil || !st.Mode().IsRegular() || st.Size() > 256 {
		return inv, errors.New("regular bounded manifest checksum required")
	}
	checks, err := os.ReadFile(sumPath)
	if err != nil || string(checks) != sum+"  "+inventoryName+"\n" {
		return inv, errors.New("missing or mismatched manifest checksum")
	}
	if inv.Architecture != arch || inv.Revision != revision || !Revision(revision) || len(inv.Images) != 4 {
		return inv, errors.New("bundle revision/platform mismatch")
	}
	files, err := tree(root)
	if err != nil {
		return inv, err
	}
	if len(files) != len(inv.Files) {
		return inv, errors.New("payload file set changed")
	}
	for name, entry := range files {
		if entry != inv.Files[name] {
			return inv, fmt.Errorf("payload changed: %s", name)
		}
	}
	if err = inspectBinaries(root, arch, files); err != nil {
		return inv, err
	}
	for _, name := range []string{"project-os", "dashboard", "forgejo", "caddy"} {
		rev := ""
		if name == "project-os" || name == "dashboard" {
			rev = revision
		}
		img, err := InspectOCI(filepath.Join(root, "images", name+".oci"), arch, rev)
		if err != nil {
			return inv, err
		}
		if img != inv.Images[name] {
			return inv, errors.New("image identity changed")
		}
	}
	if err = verifyBuildInputs(root, arch, revision, inv.Images); err != nil {
		return inv, err
	}
	return inv, nil
}

// Bundle copies only the sealed payload, not private state or the build tree.
func Bundle(source, dest, arch, revision string) error {
	inv, err := Verify(source, arch, revision)
	if err != nil {
		return err
	}
	srcRoot, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer srcRoot.Close()
	if err = FreshDirectory(dest); err != nil {
		return err
	}
	destRoot, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer destRoot.Close()
	files := inv.Files
	for name, entry := range files {
		if entry.Directory {
			if err = destRoot.MkdirAll(name, 0755); err != nil {
				return err
			}
		}
	}
	for name, entry := range files {
		target := name
		if entry.Directory {
			continue
		}
		if err = destRoot.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if entry.Link != "" {
			err = destRoot.Symlink(entry.Link, target)
		} else {
			err = copyExclusive(srcRoot, destRoot, name, entry)
		}
		if err != nil {
			return err
		}
	}
	for name, entry := range files {
		if entry.Directory {
			if err = destRoot.Chmod(name, os.FileMode(entry.Mode)); err != nil {
				return err
			}
		}
	}
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return err
	}
	if err = WriteNew(filepath.Join(dest, inventoryName), append(data, '\n'), 0644); err != nil {
		return err
	}
	if err = checksums(dest); err != nil {
		return err
	}
	_, err = Verify(dest, arch, revision)
	return err
}
func checksums(root string) error {
	sum, err := HashFile(filepath.Join(root, inventoryName))
	if err != nil {
		return err
	}
	return WriteNew(filepath.Join(root, "SHA256SUMS"), []byte(sum+"  "+inventoryName+"\n"), 0644)
}
func copyExclusive(src, dest *os.Root, name string, entry File) error {
	st, err := src.Lstat(name)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || uint32(st.Mode().Perm()) != entry.Mode {
		return errors.New("payload type/mode changed before copying")
	}
	in, err := src.Open(name)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := dest.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(entry.Mode))
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(out, hash), in)
	if hex.EncodeToString(hash.Sum(nil)) != entry.SHA256 {
		err = errors.Join(err, errors.New("payload changed during copying"))
	}
	return errors.Join(err, out.Chmod(os.FileMode(entry.Mode)), out.Close())
}
