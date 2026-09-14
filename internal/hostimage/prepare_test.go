package hostimage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func sourceRoot(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs("../..")
	require.NoError(t, err)
	return source
}

func TestPrepareVendorContextFromActualOwners(t *testing.T) {
	source := sourceRoot(t)
	for _, arch := range []string{"x86_64", "aarch64"} {
		t.Run(arch, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "context")
			base, err := Prepare(source, out, arch, strings.Repeat("a", 40))
			require.NoError(t, err)
			require.Contains(t, base.Images[arch], "@sha256:")
			read := func(name string) string {
				t.Helper()
				data, e := os.ReadFile(filepath.Join(out, name))
				require.NoError(t, e)
				return string(data)
			}
			require.Contains(t, read("rootfs/usr/lib/systemd/system/soda-host.service"), "ExecStart=/usr/libexec/soda/soda-host")
			require.Contains(t, read("rootfs/usr/lib/systemd/system/soda-tailnet@.service"), "/usr/libexec/soda/soda-host --tailnet-action=run")
			require.Contains(t, read("rootfs/etc/profile.d/soda-console-welcome.sh"), "/usr/libexec/soda/soda-console-welcome")
			require.Contains(t, read("rootfs/usr/share/containers/systemd/soda-dashboard.container"), "Image=localhost/soda-dashboard:dev") // Explicitly not yet bound app delivery.
			require.NoFileExists(t, filepath.Join(out, "rootfs/etc/zincati/config.d/90-soda-image.toml"))
			for _, path := range []string{"rootfs/var", "rootfs/usr/local", "rootfs/etc/soda", "rootfs/etc/systemd/system", "rootfs/etc/containers/systemd"} {
				require.NoDirExists(t, filepath.Join(out, path))
			}
			for _, path := range []string{"rootfs/usr/libexec/soda/soda-console-welcome", "rootfs/usr/sbin/soda-activate"} {
				info, e := os.Stat(filepath.Join(out, path))
				require.NoError(t, e)
				require.Equal(t, os.FileMode(0755), info.Mode().Perm())
			}
			target, e := os.Readlink(filepath.Join(out, "rootfs/usr/bin/soda-tailnet"))
			require.NoError(t, e)
			require.Equal(t, "../libexec/soda/soda-tailnet", target)
			packages := read("packages.list")
			original, e := os.ReadFile(filepath.Join(source, "appliance/provisioning/base.json"))
			require.NoError(t, e)
			expected, repo, e := PackageInputs(original)
			require.NoError(t, e)
			require.Equal(t, strings.Join(expected, "\n")+"\n", packages)
			require.Equal(t, repo+"\n", read("tailscale-repo.url"))
			require.Contains(t, packages, "cockpit-ostree\n")
			require.Contains(t, packages, "tailscale\n")
			require.NoError(t, Inventory(out))
			inventory, e := os.ReadFile(filepath.Join(filepath.Dir(out), "context-inventory.json"))
			require.NoError(t, e)
			require.Contains(t, string(inventory), "usr/lib/systemd/system/soda-host.service")
			require.Error(t, Inventory(out)) // Never overwrite previous evidence.
			_, e = Prepare(source, out, arch, strings.Repeat("a", 40))
			require.Error(t, e)
		})
	}
	// The live first-install sources are not rewritten by vendor context generation.
	b, err := os.ReadFile(filepath.Join(source, "appliance/services/soda-host.service"))
	require.NoError(t, err)
	require.Contains(t, string(b), "/usr/local/libexec/soda/soda-host")
}

func TestPackageInputFailures(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(sourceRoot(t), "appliance/provisioning/base.json"))
	require.NoError(t, err)
	for _, data := range []string{
		"{", "{}",
		strings.Replace(string(b), "cockpit-system cockpit-ws", "cockpit-system cockpit-system", 1),
		strings.Replace(string(b), "cockpit-system cockpit-ws", "cockpit-system --uninstall", 1),
		strings.Replace(string(b), "cockpit-system cockpit-ws", "cockpit-system ;touch /host", 1),
		strings.Replace(string(b), "rpm-ostree install -y --allow-inactive", "dnf install -y", 1),
		strings.Replace(string(b), "https://pkgs.tailscale.com/", "https://untrusted.test/", 1),
	} {
		_, _, err = PackageInputs([]byte(data))
		require.Error(t, err)
	}
}

func TestBaseLockAndUnsafeOutputRefusal(t *testing.T) {
	source := sourceRoot(t)
	_, err := LoadBase(source, "armv7")
	require.Error(t, err)
	data, err := os.ReadFile(filepath.Join(source, "appliance/locks/coreos-host.json"))
	require.NoError(t, err)
	var base Base
	require.NoError(t, json.Unmarshal(data, &base))
	for _, mutate := range []func(*Base){
		func(b *Base) { b.Images["x86_64"] = "quay.io/fedora/fedora-coreos:stable" },
		func(b *Base) { delete(b.Images, "aarch64") },
		func(b *Base) { b.MetadataURL = "https://untrusted.test/release.json" },
	} {
		var invalid Base
		require.NoError(t, json.Unmarshal(data, &invalid))
		mutate(&invalid)
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "appliance/locks"), 0755))
		raw, e := json.Marshal(invalid)
		require.NoError(t, e)
		require.NoError(t, os.WriteFile(filepath.Join(root, "appliance/locks/coreos-host.json"), raw, 0644))
		_, e = LoadBase(root, "x86_64")
		require.Error(t, e)
	}
	parent := t.TempDir()
	link := filepath.Join(parent, "link")
	require.NoError(t, os.Symlink(t.TempDir(), link))
	_, err = Prepare(source, filepath.Join(link, "context"), "x86_64", strings.Repeat("a", 40))
	require.Error(t, err)
	out := filepath.Join(parent, "invalid-revision")
	_, err = Prepare(source, out, "x86_64", "dirty")
	require.Error(t, err)
	require.NoDirExists(t, out)
}

func TestCandidateUsesOnlyNativeOSTreeFinalization(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(sourceRoot(t), "appliance/host.Containerfile"))
	require.NoError(t, err)
	require.NotContains(t, string(b), "bootc")
	require.NotContains(t, string(b), "RUN sed")
	require.Contains(t, string(b), "RUN ostree container commit")
}

func TestRecipeDoesNotInstallOrPublishOnBuilder(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(sourceRoot(t), "appliance/host.Containerfile"))
	require.NoError(t, err)
	recipe := string(b)
	for _, required := range []string{"ARG BASE_IMAGE\nFROM ${BASE_IMAGE}", "rpm-ostree install $(cat /run/soda-build/packages.list)", "ostree container commit", "host-content-only"} {
		require.Contains(t, recipe, required)
	}
	for _, bad := range []string{"FROM quay.io/fedora/fedora-coreos:stable", "COPY . ", "install-native.sh", "podman push", "systemctl enable", "tailscale up"} {
		require.NotContains(t, recipe, bad)
	}
}
