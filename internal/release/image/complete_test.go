package image

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/testoci"
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
	require.True(t, build.Digest(hash))
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
	forgejo := filepath.Join(work, "forgejo")
	archives := filepath.Join(work, "images")
	base, err := Prepare(source, context, "x86_64", strings.Repeat("a", 40))
	require.NoError(t, err)
	for name, data := range map[string]string{
		"etc/cockpit/branding/branding.css": "fixture branding", "usr/share/soda/fastfetch/sodaos.txt": "fixture logo",
		"etc/motd": "fixture MOTD", "etc/fastfetch/config.jsonc": "/usr/share/soda/fastfetch/sodaos.txt",
		"etc/soda/forgejo.env": "FORGEJO____APP_NAME=Soda OS\n", "etc/soda/proxy.Caddyfile": "fixture proxy",
	} {
		require.NoError(t, ownedWrite(filepath.Join(context, "rootfs", name), []byte(data), 0o644))
	}
	for name, data := range map[string]string{
		"templates/custom/header.tmpl":                   "fixture header",
		"public/assets/soda/forgejo/soda-native-page.js": "fixture module",
	} {
		require.NoError(t, ownedWrite(filepath.Join(forgejo, "forgejo", name), []byte(data), 0o644))
	}
	// Match the asset leaf's explicit public directory modes under private umask.
	for _, root := range []string{filepath.Join(forgejo, "forgejo"), filepath.Join(context, "rootfs")} {
		require.NoError(t, filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				err = os.Chmod(path, 0o755)
			}
			return err
		}))
	}
	presentation, err := StagePresentation(forgejo, context)
	require.NoError(t, err)
	p := deliver.Payload{Format: 3, ID: base.Release + ".soda-" + strings.Repeat("a", 12), Revision: strings.Repeat("a", 40), Architecture: "x86_64", CoreOS: base.Release, Base: base.Images["x86_64"], RepositoryPrefix: "ghcr.io/example/sodaos", Schema: 10, PresentationSHA256: presentation, HostPackagesSHA256: strings.Repeat("b", 64), Images: map[string]deliver.Image{}}
	for _, name := range deliver.Names {
		im := testoci.Archive(t, filepath.Join(archives, name+".oci"), "amd64", p.Revision)
		p.Images[name] = deliver.Image{Reference: p.RepositoryPrefix + "-" + name + "@" + im.Manifest, Manifest: im.Manifest, Config: im.Config, ArchiveSHA256: im.ArchiveSHA256}
	}
	copies := 0
	run := func(dir, cmd string, args ...string) error {
		require.Equal(t, archives, dir)
		require.Equal(t, "skopeo", cmd)
		name := deliver.Names[copies]
		dest := filepath.Join(context, "rootfs/usr/share/soda/images")
		require.Equal(t, []string{"copy", "--preserve-digests", "--dest-oci-accept-uncompressed-layers", "oci-archive:" + filepath.Join(archives, name+".oci"), "oci:" + dest + ":" + p.Images[name].Config}, args)
		testoci.Add(t, filepath.Join(archives, name+".oci"), dest)
		copies++
		return nil
	}
	require.NoError(t, Complete(source, context, archives, p, run))
	require.Equal(t, 5, copies)
	layout := filepath.Join(context, "rootfs/usr/share/soda/images")
	files, _, err := deliver.VerifyContent(p, layout)
	require.NoError(t, err)
	require.Len(t, files, 13) // five configs/manifests, one shared layer, index and layout
	require.Error(t, stageImages(archives, layout, p, run))
	require.Equal(t, 5, copies, "occupied layout must refuse before another copy")
	read := func(path string) string {
		t.Helper()
		b, e := os.ReadFile(filepath.Join(context, "rootfs", path))
		require.NoError(t, e)
		return string(b)
	}
	for name, unit := range map[string]string{"forgejo": "forgejo.container", "dashboard": "soda-dashboard.container", "proxy": "soda-proxy.container"} {
		body := read("usr/share/containers/systemd/" + unit)
		require.Contains(t, body, "Image="+p.Images[name].Config)
		require.Equal(t, 1, strings.Count(body, "Pull=never"))
		require.NotContains(t, body, "additionalimagestore")
		require.Contains(t, body, "Requires=soda-image-import.service")
	}
	require.NoDirExists(t, filepath.Join(context, "rootfs/usr/lib/bootc/bound-images.d"))
	for _, name := range deliver.Names {
		require.NoFileExists(t, filepath.Join(context, "rootfs/usr/share/soda/images", name+".oci"))
		require.FileExists(t, filepath.Join(archives, name+".oci"))
	}
	require.Contains(t, read("usr/lib/systemd/system/soda-project@.service"), "Requires=soda-image-import.service")
	require.NoDirExists(t, filepath.Join(context, "rootfs/usr/lib/systemd/system/soda-project@.service.d"))
	require.Contains(t, read("usr/lib/systemd/system/soda-image-import.service"), "ExecStart=/usr/libexec/soda/soda-image-import")
	for _, path := range []string{"etc/containers/storage.conf", "var", "etc/soda/host.json", "etc/soda/dashboard.json"} {
		_, e := os.Lstat(filepath.Join(context, "rootfs", path))
		require.True(t, os.IsNotExist(e), path)
	}
	var example map[string]any
	require.NoError(t, json.Unmarshal([]byte(read("usr/share/soda/defaults/host.example.json")), &example))
	require.NotContains(t, example, "image")
	require.NotContains(t, example, "tailnet_image")
	require.Equal(t, "", example["subnet"])
	loaded, e := deliver.Load(filepath.Join(context, "rootfs/usr/share/soda/release.json"))
	require.NoError(t, e)
	require.Equal(t, p, loaded)
	require.Contains(t, read("etc/fastfetch/config.jsonc"), "/usr/share/soda/fastfetch")
	require.Equal(t, read("etc/soda/forgejo.env"), read("usr/share/soda/defaults/forgejo.env"))
	require.Equal(t, read("etc/soda/proxy.Caddyfile"), read("usr/share/soda/defaults/proxy.Caddyfile"))
}

func TestDirectPresentationDoesNotNormalizeOrCopyUntrustedInputs(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o755} {
		source := filepath.Join(t.TempDir(), "public")
		file := filepath.Join(source, "asset")
		require.NoError(t, ownedWrite(file, []byte("fixture bytes"), mode))
		require.NoError(t, os.Chmod(source, 0o755))
		_, err := publicFiles(source)
		require.ErrorContains(t, err, "file must be 0644")
		info, err := os.Stat(file)
		require.NoError(t, err)
		require.Equal(t, mode, info.Mode().Perm())
	}
	source := t.TempDir()
	_, err := publicFiles(source)
	require.ErrorContains(t, err, "directory must be 0755")
	require.NoError(t, os.Chmod(source, 0o755))
	require.NoError(t, ownedWrite(filepath.Join(source, "asset"), []byte("fixture bytes"), 0o644))
	files, err := publicFiles(source)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"asset": hashBytes([]byte("fixture bytes"))}, files)
}

func TestPublicStageAndQuadletRefuseAmbiguity(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.Chmod(source, 0o755))
	require.NoError(t, os.Symlink("/etc/shadow", filepath.Join(source, "not-public")))
	_, err := publicFiles(source)
	require.ErrorContains(t, err, "symlink/special")
	for _, body := range []string{"[Container]\n", "[Container]\nImage=a\nImage=b\n", "[Container]\nImage=a\nGlobalArgs=--root=/other"} {
		_, err = LocalQuadlet(body, "ghcr.io/example/image@sha256:"+strings.Repeat("a", 64))
		require.Error(t, err)
	}
}
