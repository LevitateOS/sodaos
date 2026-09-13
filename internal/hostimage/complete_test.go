package hostimage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/stretchr/testify/require"
)

func TestLockedHostTransactionMatchesActualPackageOwner(t *testing.T) {
	source := sourceRoot(t)
	root := t.TempDir()
	context := filepath.Join(root, "context")
	base, err := Prepare(source, context, "x86_64", strings.Repeat("a", 40))
	require.NoError(t, err)
	hash, err := LockHostPackages(source, context, "x86_64", base)
	require.NoError(t, err)
	require.True(t, nativebuild.Digest(hash))
	b, err := os.ReadFile(filepath.Join(context, "packages.list"))
	require.NoError(t, err)
	require.Contains(t, string(b), "cockpit-ostree-1:225-1.fc44.noarch")
	b, err = os.ReadFile(filepath.Join(context, "packages.expected"))
	require.NoError(t, err)
	require.Equal(t, 625, len(strings.FieldsFunc(string(b), func(r rune) bool { return r == '\n' })))
	require.Equal(t, hash, hashBytes(b))
	// Do not infer native ARM packages or qualification from the x86 lock.
	_, err = LockHostPackages(source, context, "aarch64", base)
	require.ErrorContains(t, err, "architecture package lock")
	base.Release = "44.0.0.0"
	_, err = LockHostPackages(source, context, "x86_64", base)
	require.ErrorContains(t, err, "does not match")
}

func TestCompleteCandidateBindingsAndStateOwnership(t *testing.T) {
	source := sourceRoot(t)
	work := t.TempDir()
	context := filepath.Join(work, "context")
	native := filepath.Join(work, "native")
	forgejo := filepath.Join(work, "forgejo")
	archives := filepath.Join(work, "images")
	base, err := Prepare(source, context, "x86_64", strings.Repeat("a", 40))
	require.NoError(t, err)
	for name, data := range map[string]string{
		"var/lib/soda/forgejo/gitea/templates/custom/header.tmpl":                   "fixture header",
		"var/lib/soda/forgejo/gitea/public/assets/soda/forgejo/soda-native-page.js": "fixture module",
		"etc/cockpit/branding/branding.css":                                         "fixture branding", "usr/local/share/soda/fastfetch/sodaos.txt": "fixture logo",
		"etc/motd": "fixture MOTD", "etc/fastfetch/config.jsonc": "/usr/local/share/soda/fastfetch/sodaos.txt",
		"etc/soda/forgejo.env": "FORGEJO____APP_NAME=Soda OS\n", "etc/soda/proxy.Caddyfile": "fixture proxy",
	} {
		require.NoError(t, ownedWrite(filepath.Join(native, name), []byte(data), 0644))
	}
	presentation, err := StagePresentation(native, forgejo, context)
	require.NoError(t, err)
	p := appliancerelease.Payload{Format: 1, ID: base.Release + ".soda-" + strings.Repeat("a", 12), Revision: strings.Repeat("a", 40), Architecture: "x86_64", CoreOS: base.Release, Base: base.Images["x86_64"], RepositoryPrefix: "ghcr.io/example/sodaos", Schema: 10, PresentationSHA256: presentation, HostPackagesSHA256: strings.Repeat("b", 64), Images: map[string]appliancerelease.Image{}}
	for _, name := range appliancerelease.Names {
		storage := "bound"
		if name == "project-os" || name == "tailnet" {
			storage = "retained"
			require.NoError(t, ownedWrite(filepath.Join(archives, name+".oci"), []byte("archive fixture"), 0644))
		}
		p.Images[name] = appliancerelease.Image{Reference: p.RepositoryPrefix + "-" + name + "@sha256:" + strings.Repeat("c", 64), Manifest: "sha256:" + strings.Repeat("c", 64), Config: "sha256:" + strings.Repeat("d", 64), ArchiveSHA256: hashBytes([]byte("archive fixture")), Storage: storage}
	}
	require.NoError(t, Complete(source, native, context, archives, p))
	read := func(path string) string {
		t.Helper()
		b, e := os.ReadFile(filepath.Join(context, "rootfs", path))
		require.NoError(t, e)
		return string(b)
	}
	for name, unit := range map[string]string{"forgejo": "forgejo.container", "dashboard": "soda-dashboard.container", "proxy": "soda-proxy.container"} {
		body := read("usr/share/containers/systemd/" + unit)
		require.Contains(t, body, "Image="+p.Images[name].Reference)
		require.Equal(t, 1, strings.Count(body, "Pull=never"))
		require.Contains(t, body, "GlobalArgs=--storage-opt=additionalimagestore=/usr/lib/bootc/storage")
		link, e := os.Readlink(filepath.Join(context, "rootfs/usr/lib/bootc/bound-images.d", unit))
		require.NoError(t, e)
		require.Equal(t, "/usr/share/containers/systemd/"+unit, link)
	}
	files, e := os.ReadDir(filepath.Join(context, "rootfs/usr/lib/bootc/bound-images.d"))
	require.NoError(t, e)
	require.Len(t, files, 3)
	require.Contains(t, read("usr/lib/systemd/system/soda-project@.service"), "Requires=soda-image-import.service")
	require.NoDirExists(t, filepath.Join(context, "rootfs/usr/lib/systemd/system/soda-project@.service.d"))
	require.Contains(t, read("usr/lib/systemd/system/soda-image-import.service"), "ExecStart=/usr/libexec/soda/soda-image-import")
	for _, path := range []string{"etc/containers/storage.conf", "var", "etc/soda/host.json", "etc/soda/dashboard.json"} {
		_, e = os.Lstat(filepath.Join(context, "rootfs", path))
		require.True(t, os.IsNotExist(e), path)
	}
	var example map[string]any
	require.NoError(t, json.Unmarshal([]byte(read("usr/share/soda/defaults/host.example.json")), &example))
	require.NotContains(t, example, "image")
	require.NotContains(t, example, "tailnet_image")
	require.Equal(t, "", example["subnet"])
	loaded, e := appliancerelease.Load(filepath.Join(context, "rootfs/usr/share/soda/release.json"))
	require.NoError(t, e)
	require.Equal(t, p, loaded)
	require.Contains(t, read("etc/fastfetch/config.jsonc"), "/usr/share/soda/fastfetch")
}

func TestPublicStageAndQuadletRefuseAmbiguity(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.Symlink("/etc/shadow", filepath.Join(source, "not-public")))
	_, err := CopyPublicTree(source, filepath.Join(t.TempDir(), "out"))
	require.ErrorContains(t, err, "symlink/special")
	for _, body := range []string{"[Container]\n", "[Container]\nImage=a\nImage=b\n", "[Container]\nImage=a\nGlobalArgs=--root=/other"} {
		_, err = BoundQuadlet(body, "ghcr.io/example/image@sha256:"+strings.Repeat("a", 64))
		require.Error(t, err)
	}
}
