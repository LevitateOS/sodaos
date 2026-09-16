package build

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

// One reviewed source/destination inventory drives staging and the compiled
// verifier. It admits these exact customization files, never a mutable data tree.
//
//go:embed forgejo-payload.json
var forgejoPayload []byte

var forgejoFiles = func() []string {
	var entries map[string]string
	if err := json.Unmarshal(forgejoPayload, &entries); err != nil {
		panic(err)
	}
	names := make([]string, 0, len(entries))
	for dest, source := range entries {
		if !fs.ValidPath(dest) || !fs.ValidPath(strings.TrimPrefix(source, "@build/")) {
			panic("invalid Forgejo payload path")
		}
		names = append(names, "rootfs/var/lib/soda/forgejo/gitea/"+dest)
	}
	slices.Sort(names)
	return names
}()

func isForgejoPayload(p string) bool {
	for _, name := range forgejoFiles {
		if p == name || strings.HasPrefix(name, p+"/") {
			return true
		}
	}
	return false
}

func isRetiredPresentationPayload(p string) bool {
	for _, retired := range []string{"rootfs/usr/local/share/soda/dashboard", "rootfs/usr/local/share/cockpit"} {
		if p == retired || strings.HasPrefix(p, retired+"/") {
			return true
		}
	}
	return false
}

func isPrivateRootfsPayload(p string) bool {
	// The core owns the stage. Reject known runtime/private content rather than
	// interpreting application config or inventing a React payload.
	for _, bad := range []string{"/dashboard.json", "/host.json", "/install-started", "/shadow", "/gshadow", "/machine-id", "/tailscale/", "/browser-home/", "/soda-artifacts", "/soda-acceptance", "/ssh_host_", "/authorized_keys"} {
		if strings.Contains(p, bad) {
			return true
		}
	}
	switch filepath.Ext(p) {
	case ".ign", ".bu", ".key", ".pem", ".sqlite", ".db":
		return true
	}
	return false
}

func isSodaRootfsDir(p string) bool {
	switch p {
	case "rootfs", "rootfs/usr", "rootfs/var", "rootfs/var/lib", "rootfs/var/lib/soda":
		return true
	}
	return false
}

func isForgejoRootfsDir(p string) bool {
	return p == "rootfs/var/lib/soda/forgejo" || p == "rootfs/var/lib/soda/forgejo/gitea" || p == "rootfs/var/lib/soda/forgejo/gitea/public" || strings.HasPrefix(p, "rootfs/var/lib/soda/forgejo/gitea/public/")
}

func isUsrLocalPayload(p string) bool {
	return p == "rootfs/usr/local" || strings.HasPrefix(p, "rootfs/usr/local/")
}

func isRootfsPayload(p string) bool {
	if p != "rootfs" && !strings.HasPrefix(p, "rootfs/") {
		return false
	}
	if p == "rootfs/etc" || strings.HasPrefix(p, "rootfs/etc/") {
		return publicEtcPath(p)
	}
	if isPrivateRootfsPayload(p) {
		return false
	}
	return isSodaRootfsDir(p) || isForgejoRootfsDir(p) || isUsrLocalPayload(p)
}

func isImagesPayload(p string) bool {
	switch p {
	case "images", "images/project-os.oci", "images/dashboard.oci", "images/forgejo.oci", "images/caddy.oci", "images/tailnet.oci":
		return true
	}
	return false
}

func isToolsPayload(p string) bool {
	switch p {
	case "tools", "tools/soda-artifacts", "install-native.sh":
		return true
	}
	return false
}

func isLockInputsPayload(p string) bool {
	switch p {
	case "inputs", "inputs/go.mod", "inputs/go.sum":
		return true
	}
	return false
}

func isFrontendInputsPayload(p string) bool {
	switch p {
	case "inputs/package.json", "inputs/lit-check-package.json", "inputs/bun.lock", "inputs/bunfig.toml", "inputs/native-build.json":
		return true
	}
	return false
}

func isNoticesPayload(p string) bool {
	switch p {
	case "notices", "notices/README.md", "notices/tea-LICENSE", "notices/avatar-dependencies.txt", "notices/soda-LICENSE", "notices/soda-NOTICE":
		return true
	}
	return false
}

func isBundleManifestPayload(p string) bool {
	return isImagesPayload(p) || isToolsPayload(p) || isLockInputsPayload(p) || isFrontendInputsPayload(p) || isNoticesPayload(p)
}

func allowedPayload(p string) bool {
	if isForgejoPayload(p) {
		return true
	}
	// Retired presentation output is not a valid payload for new Soda bundles.
	if isRetiredPresentationPayload(p) {
		return false
	}
	if isRootfsPayload(p) {
		return true
	}
	return isBundleManifestPayload(p)
}

// P04's export boundary follows the actual /etc outputs in scripts/stage.py.
// Runtime config/credentials and unexpected files are never admitted merely
// because their filename has no private extension.
func publicEtcPath(p string) bool {
	const fontPrefix = "rootfs/var/lib/soda/forgejo/gitea/public/assets/soda/fonts/"
	for _, source := range forgejoFiles {
		if strings.HasPrefix(source, fontPrefix) {
			full := "rootfs/etc/cockpit/branding/fonts/" + strings.TrimPrefix(source, fontPrefix)
			if p == full || strings.HasPrefix(full, p+"/") {
				return true
			}
		}
	}
	for _, name := range []string{
		"containers/systemd/forgejo.container", "containers/systemd/soda-dashboard.container", "containers/systemd/soda-proxy.container",
		"systemd/system/soda-host.service", "systemd/system/soda-host.socket", "systemd/system/soda-project@.service", "systemd/system/soda-tailnet@.service", "systemd/system/soda-runner@.service", "systemd/system/cockpit.socket.d/10-soda.conf",
		"sysusers.d/soda.conf", "sysusers.d/soda-runners.conf", "tmpfiles.d/soda.conf", "tmpfiles.d/soda-runners.conf", "sysctl.d/90-soda-routing.conf",
		"pam.d/cockpit", "cockpit/cockpit.conf", "cockpit/disallowed-users", "profile.d/soda-console-welcome.sh", "motd", "fastfetch/config.jsonc", "soda/forgejo.env", "soda/proxy.Caddyfile",
		"cockpit/branding/branding.css", "cockpit/branding/theme.css", "cockpit/branding/palette.css", "cockpit/branding/favicon.ico", "cockpit/branding/apple-touch-icon.png", "cockpit/branding/soda-symbol-brutalist.svg", "cockpit/branding/soda-symbol-brutalist-dark.svg",
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
	return false
}

func openBundleRoot(root string) (*os.Root, error) {
	cap, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	st, err := cap.Lstat("tools")
	if err != nil || !st.IsDir() {
		cap.Close()
		return nil, errors.New("real tools directory required")
	}
	return cap, nil
}

func resolveTreeFileContent(cap *os.Root, p, rel string, mode os.FileMode) (string, string, error) {
	switch {
	case mode.IsDir():
		return "", "", nil
	case mode.IsRegular():
		hash, err := HashAt(cap, p)
		return hash, "", err
	case mode&os.ModeSymlink != 0:
		link, err := cap.Readlink(p)
		if err != nil {
			return "", "", err
		}
		if !validLink(rel, link) {
			return "", "", errors.New("unsafe payload symlink")
		}
		return "", link, nil
	default:
		return "", "", errors.New("unsupported payload file")
	}
}

func processTreeEntry(cap *os.Root, p string) (File, error) {
	rel := p
	if !allowedPayload(rel) {
		return File{}, fmt.Errorf("non-payload path refused: %s", rel)
	}
	info, err := cap.Lstat(p)
	if err != nil {
		return File{}, err
	}
	if info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return File{}, errors.New("set-ID payload refused")
	}
	hash, link, err := resolveTreeFileContent(cap, p, rel, info.Mode())
	if err != nil {
		return File{}, err
	}
	return File{
		Mode:      uint32(info.Mode().Perm()),
		Directory: info.IsDir(),
		SHA256:    hash,
		Link:      link,
	}, nil
}

func walkTreePayload(cap *os.Root) (map[string]File, error) {
	result := map[string]File{}
	for _, top := range []string{"rootfs", "images", "tools/soda-artifacts", "install-native.sh", "inputs", "notices"} {
		err := fs.WalkDir(cap.FS(), top, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			entry, err := processTreeEntry(cap, p)
			if err != nil {
				return err
			}
			result[p] = entry
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func verifySodaspacesPayload(result map[string]File) error {
	for _, name := range forgejoFiles {
		entry, ok := result[name]
		if !ok || entry.SHA256 == "" || entry.Mode != 0o644 {
			return fmt.Errorf("missing or unreadable Sodaspaces payload: %s", name)
		}
	}
	for _, name := range []string{"rootfs/var/lib/soda/forgejo/gitea/templates", "rootfs/var/lib/soda/forgejo/gitea/templates/custom"} {
		entry := result[name]
		if !entry.Directory || entry.Mode != 0o755 {
			return fmt.Errorf("unreadable Sodaspaces directory: %s", name)
		}
	}
	return nil
}

func verifyRequiredPayload(result map[string]File) error {
	for _, required := range []string{"rootfs/etc/containers/systemd/forgejo.container", "rootfs/etc/containers/systemd/soda-dashboard.container", "rootfs/etc/containers/systemd/soda-proxy.container", "rootfs/etc/systemd/system/soda-host.service", "rootfs/etc/systemd/system/soda-host.socket", "rootfs/etc/systemd/system/soda-project@.service", "rootfs/etc/systemd/system/soda-tailnet@.service", "rootfs/etc/fastfetch/config.jsonc", "rootfs/usr/local/share/soda/fastfetch/sodaos.txt", "rootfs/usr/local/libexec/soda/soda-dashboard", "rootfs/usr/local/libexec/soda/soda-host", "inputs/native-build.json", "inputs/go.mod", "inputs/go.sum", "notices/README.md", "notices/tea-LICENSE", "notices/avatar-dependencies.txt", "notices/soda-LICENSE", "notices/soda-NOTICE"} {
		entry, ok := result[required]
		if !ok || entry.SHA256 == "" {
			return fmt.Errorf("missing core/support payload: %s", required)
		}
	}
	return verifySodaspacesPayload(result)
}

func tree(root string) (map[string]File, error) {
	cap, err := openBundleRoot(root)
	if err != nil {
		return nil, err
	}
	defer cap.Close()
	result, err := walkTreePayload(cap)
	if err != nil {
		return nil, err
	}
	if err := verifyRequiredPayload(result); err != nil {
		return nil, err
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
	for _, tool := range []string{"go", "bun", "podman", "python", "kernel"} {
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

func inspectSealImages(root, arch, revision string) (map[string]Image, error) {
	images := map[string]Image{}
	for _, name := range []string{"project-os", "dashboard", "forgejo", "caddy", "tailnet"} {
		rev := ""
		if name == "project-os" || name == "dashboard" {
			rev = revision
		}
		img, err := InspectOCI(filepath.Join(root, "images", name+".oci"), arch, rev)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", name, err)
		}
		images[name] = img
	}
	return images, nil
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
	images, err := inspectSealImages(root, arch, revision)
	if err != nil {
		return err
	}
	if err = verifyBuildInputs(root, arch, revision, images); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Inventory{revision, arch, files, images}, "", "  ")
	if err != nil {
		return err
	}
	if err = WriteNew(filepath.Join(root, inventoryName), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return checksums(root)
}

func verifyManifestChecksumFile(root, sum string) error {
	sumPath := filepath.Join(root, "SHA256SUMS")
	st, err := os.Lstat(sumPath)
	if err != nil || !st.Mode().IsRegular() || st.Size() > 256 {
		return errors.New("regular bounded manifest checksum required")
	}
	checks, err := os.ReadFile(sumPath)
	if err != nil || string(checks) != sum+"  "+inventoryName+"\n" {
		return errors.New("missing or mismatched manifest checksum")
	}
	return nil
}

func verifyInventoryChecksum(root, arch, revision string, inv Inventory) error {
	sum, err := HashFile(filepath.Join(root, inventoryName))
	if err != nil {
		return err
	}
	if err := verifyManifestChecksumFile(root, sum); err != nil {
		return err
	}
	if inv.Architecture != arch || inv.Revision != revision || !Revision(revision) || len(inv.Images) != 5 {
		return errors.New("bundle revision/platform mismatch")
	}
	return nil
}

func verifyInventoryFiles(root string, inv Inventory) (map[string]File, error) {
	files, err := tree(root)
	if err != nil {
		return nil, err
	}
	if len(files) != len(inv.Files) {
		return nil, errors.New("payload file set changed")
	}
	for name, entry := range files {
		if entry != inv.Files[name] {
			return nil, fmt.Errorf("payload changed: %s", name)
		}
	}
	return files, inspectBinaries(root, inv.Architecture, files)
}

func verifyInventoryImages(root, arch, revision string, inv Inventory) error {
	for _, name := range []string{"project-os", "dashboard", "forgejo", "caddy", "tailnet"} {
		rev := ""
		if name == "project-os" || name == "dashboard" {
			rev = revision
		}
		img, err := InspectOCI(filepath.Join(root, "images", name+".oci"), arch, rev)
		if err != nil {
			return err
		}
		if img != inv.Images[name] {
			return errors.New("image identity changed")
		}
	}
	return verifyBuildInputs(root, arch, revision, inv.Images)
}

func Verify(root, arch, revision string) (Inventory, error) {
	var inv Inventory
	if err := ReadJSON(filepath.Join(root, inventoryName), &inv); err != nil {
		return inv, err
	}
	if err := verifyInventoryChecksum(root, arch, revision, inv); err != nil {
		return inv, err
	}
	if _, err := verifyInventoryFiles(root, inv); err != nil {
		return inv, err
	}
	return inv, verifyInventoryImages(root, arch, revision, inv)
}

// Bundle copies only the sealed payload, not private state or the build tree.
func mkdirBundleDirectories(destRoot *os.Root, files map[string]File) error {
	for name, entry := range files {
		if entry.Directory {
			if err := destRoot.MkdirAll(name, 0o755); err != nil {
				return err
			}
		}
	}
	return nil
}

func chmodBundleDirectories(destRoot *os.Root, files map[string]File) error {
	for name, entry := range files {
		if entry.Directory {
			if err := destRoot.Chmod(name, os.FileMode(entry.Mode)); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyBundleFiles(srcRoot, destRoot *os.Root, files map[string]File) error {
	for name, entry := range files {
		if entry.Directory {
			continue
		}
		if err := destRoot.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			return err
		}
		var err error
		if entry.Link != "" {
			err = destRoot.Symlink(entry.Link, name)
		} else {
			err = copyExclusive(srcRoot, destRoot, name, entry)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func copyBundleEntries(srcRoot, destRoot *os.Root, files map[string]File) error {
	if err := mkdirBundleDirectories(destRoot, files); err != nil {
		return err
	}
	if err := copyBundleFiles(srcRoot, destRoot, files); err != nil {
		return err
	}
	return chmodBundleDirectories(destRoot, files)
}

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
	if err = copyBundleEntries(srcRoot, destRoot, inv.Files); err != nil {
		return err
	}
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return err
	}
	if err = WriteNew(filepath.Join(dest, inventoryName), append(data, '\n'), 0o644); err != nil {
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
	return WriteNew(filepath.Join(root, "SHA256SUMS"), []byte(sum+"  "+inventoryName+"\n"), 0o644)
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
