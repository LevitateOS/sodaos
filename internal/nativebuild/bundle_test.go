package nativebuild

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func fixtureBundle(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "source")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		"rootfs/etc/containers/systemd/forgejo.container", "rootfs/etc/containers/systemd/soda-dashboard.container", "rootfs/etc/containers/systemd/soda-proxy.container",
		"rootfs/etc/systemd/system/soda-host.service", "rootfs/etc/systemd/system/soda-host.socket",
		"rootfs/usr/local/libexec/soda/soda-dashboard", "rootfs/usr/local/libexec/soda/soda-host",
		"rootfs/usr/local/share/cockpit/soda-tailscale/index.html", "rootfs/usr/local/share/cockpit/soda-runners/index.html",
		"rootfs/var/lib/soda/forgejo/gitea/public/assets/img/logo.svg",
		"inputs/native-build.json", "inputs/go.mod", "inputs/go.sum", "notices/README.md", "notices/tea-LICENSE", "notices/avatar-dependencies.txt", "notices/soda-LICENSE", "notices/soda-NOTICE", "tools/soda-artifacts", "install-native.sh",
	}
	paths = append(paths, sodaspacesFiles...)
	for _, name := range paths {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("synthetic payload; never executed\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range sodaspacesFiles {
		if err := os.Chmod(filepath.Join(root, name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"templates", "templates/custom"} {
		if err := os.Chmod(filepath.Join(root, "rootfs/var/lib/soda/forgejo/gitea", name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "images"), 0755); err != nil {
		t.Fatal(err)
	}
	images := map[string]Image{}
	for _, name := range []string{"project-os", "dashboard", "forgejo", "caddy"} {
		p := filepath.Join(root, "images", name+".oci")
		fixtureOCI(t, p, "amd64", false)
		image, err := InspectOCI(p, "x86_64", fixtureRevision)
		if err != nil {
			t.Fatal(err)
		}
		images[name] = image
	}
	// Header-only synthetic ELF fixtures: architecture inspection, never execution.
	for _, name := range []string{"tools/soda-artifacts", "rootfs/usr/local/libexec/soda/soda-dashboard", "rootfs/usr/local/libexec/soda/soda-host"} {
		p := filepath.Join(root, name)
		header := make([]byte, 64)
		copy(header, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
		binary.LittleEndian.PutUint16(header[16:], 2)
		binary.LittleEndian.PutUint16(header[18:], 62)
		binary.LittleEndian.PutUint32(header[20:], 1)
		binary.LittleEndian.PutUint16(header[52:], 64)
		binary.LittleEndian.PutUint16(header[54:], 56)
		binary.LittleEndian.PutUint16(header[58:], 64)
		if err := os.WriteFile(p, header, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	tools := map[string]string{}
	for _, name := range []string{"go", "node", "pnpm", "podman", "python", "kernel"} {
		tools[name] = "synthetic fixture; not executed"
	}
	ids := map[string]any{}
	for name, image := range images {
		ids[name] = map[string]string{"ID": image.Config}
	}
	metadata, _ := json.Marshal(map[string]any{"Revision": fixtureRevision, "Architecture": "x86_64", "Tools": tools, "Images": ids})
	if err := os.WriteFile(filepath.Join(root, "inputs/native-build.json"), metadata, 0644); err != nil {
		t.Fatal(err)
	}
	files, err := tree(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(Inventory{Revision: fixtureRevision, Architecture: "x86_64", Files: files, Images: images})
	if err = os.WriteFile(filepath.Join(root, inventoryName), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err = checksums(root); err != nil {
		t.Fatal(err)
	}
	return root
}
func TestBundleAllowlistIntegrityAndNoOverwrite(t *testing.T) {
	source := fixtureBundle(t)
	if err := os.WriteFile(filepath.Join(source, "private-operator-key"), []byte("synthetic excluded data"), 0600); err != nil {
		t.Fatal(err)
	}
	oldMask := syscall.Umask(0077)
	defer syscall.Umask(oldMask)
	dest := filepath.Join(t.TempDir(), "x86_64")
	if err := Bundle(source, dest, "x86_64", fixtureRevision); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "private-operator-key")); !os.IsNotExist(err) {
		t.Fatal("copied outside allowlist")
	}
	if _, err := Verify(dest, "x86_64", fixtureRevision); err != nil {
		t.Fatal(err)
	}
	if err := Bundle(source, dest, "x86_64", fixtureRevision); err == nil {
		t.Fatal("overwrote bundle")
	}
	p := filepath.Join(dest, "rootfs/usr/local/libexec/soda/soda-host")
	if err := os.WriteFile(p, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(dest, "x86_64", fixtureRevision); err == nil {
		t.Fatal("accepted changed bytes")
	}
}
func TestBundleRejectsPrivateFilesAndMissingPayload(t *testing.T) {
	for _, name := range []string{"rootfs/etc/soda/operator.key", "rootfs/usr/local/libexec/soda/soda-artifacts", "rootfs/etc/soda/oauth-secret", "rootfs/etc/soda/admin-token", "rootfs/etc/soda/grant-key", "rootfs/etc/soda/dashboard.json", "rootfs/etc/private/credentials", "rootfs/etc/soda/unknown-input"} {
		root := fixtureBundle(t)
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("synthetic private/support file"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := tree(root); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	root := fixtureBundle(t)
	if err := os.Remove(filepath.Join(root, "rootfs/usr/local/share/cockpit/soda-runners/index.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := tree(root); err == nil {
		t.Fatal("accepted missing core-owned page")
	}
}

func TestBundleRequiresAvatarDependencyNotices(t *testing.T) {
	root := fixtureBundle(t)
	if err := os.Remove(filepath.Join(root, "notices/avatar-dependencies.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := tree(root); err == nil {
		t.Fatal("accepted a bundle without avatar dependency notices")
	}
}
func TestRunnerInstalledUtilityIsNotAnApplianceStateMarker(t *testing.T) {
	if !allowedPayload("rootfs/usr/local/lib/soda/github-actions-runner/externals/node20/lib/node_modules/npm/lib/utils/installed-deep.js") {
		t.Fatal("vendor npm utility mistaken for appliance runtime state")
	}
	if allowedPayload("rootfs/etc/soda/installed") || allowedPayload("rootfs/etc/soda/install-started") {
		t.Fatal("appliance installation state admitted to payload")
	}
}

func TestLinkAndPathBoundaries(t *testing.T) {
	if !validLink("rootfs/usr/local/bin/soda-tailnet", "/usr/local/libexec/soda/soda-tailnet") {
		t.Fatal("lost delivered CLI link")
	}
	if validLink("rootfs/usr/local/bin/evil", "/etc/shadow") {
		t.Fatal("unsafe absolute link")
	}
	if allowedPayload("rootfs/usr/local-escape/anything") || allowedPayload("inputs/private.ign") {
		t.Fatal("allowlist prefix escape")
	}
	if _, err := OCIArchitecture("arm64"); err == nil {
		t.Fatal("mixed native/OCI vocabulary")
	}
	if Digest(strings.Repeat("g", 64)) || !Digest(strings.Repeat("a", 64)) {
		t.Fatal("digest validation")
	}
}
func TestSelectedSignerAndMalformedStatus(t *testing.T) {
	signer := strings.Repeat("A", 40)
	good := "[GNUPG:] VALIDSIG " + signer + " 2026-09-06 1 0 4 0 1 10 00\n"
	if !validSignature([]byte(good), signer) {
		t.Fatal("valid selected fingerprint not recognized")
	}
	for _, status := range []string{"", "[GNUPG:] VALIDSIG " + signer, strings.ReplaceAll(good, signer, strings.Repeat("B", 40)), good + "[GNUPG:] BADSIG bad\n", good + "[GNUPG:] EXPKEYSIG bad\n"} {
		if validSignature([]byte(status), signer) {
			t.Fatal("accepted invalid signature status")
		}
	}
}
func TestPrivateOutputRefusesExistingAndSymlinkedParents(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "private.json")
	if err := PrivateDestination(out); err != nil {
		t.Fatal(err)
	}
	if err := WriteNew(out, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := PrivateDestination(out); err == nil {
		t.Fatal("accepted existing output")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	if err := PrivateDestination(filepath.Join(link, "out")); err == nil {
		t.Fatal("accepted symlink parent")
	}
}
