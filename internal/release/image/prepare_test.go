package image

import (
	"context"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/stretchr/testify/require"
)

func sourceRoot(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs("../../..")
	require.NoError(t, err)
	return source
}

func TestPrepareVendorContextFromActualOwners(t *testing.T) {
	streamFixtures(t, goodStreamDoc(), goodIndexDoc())
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
			require.Contains(t, read("rootfs/usr/lib/systemd/system/soda-console.service"), "/usr/libexec/soda/soda-console-welcome")
			require.Contains(t, read("rootfs/usr/lib/systemd/system/soda-console.service"), "Before=getty@tty1.service")
			require.Contains(t, read("rootfs/usr/lib/systemd/system/soda-console.service"), "Wants=forgejo.service soda-dashboard.service soda-proxy.service")
			require.Contains(t, read("rootfs/usr/share/containers/systemd/soda-dashboard.container"), "Image=localhost/soda-dashboard:dev") // Explicitly not yet bound app delivery.
			require.NoFileExists(t, filepath.Join(out, "rootfs/etc/zincati/config.d/90-soda-image.toml"))
			for _, path := range []string{"rootfs/var", "rootfs/usr/sbin", "rootfs/usr/local", "rootfs/etc/soda", "rootfs/etc/systemd/system", "rootfs/etc/containers/systemd"} {
				require.NoDirExists(t, filepath.Join(out, path))
			}
			for _, path := range []string{"rootfs/usr/libexec/soda/soda-console-welcome", "rootfs/usr/bin/soda-activate"} {
				info, e := os.Stat(filepath.Join(out, path))
				require.NoError(t, e)
				require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
			}
			target, e := os.Readlink(filepath.Join(out, "rootfs/usr/bin/soda-tailnet"))
			require.NoError(t, e)
			require.Equal(t, "../libexec/soda/soda-tailnet", target)
			target, e = os.Readlink(filepath.Join(out, "rootfs/usr/bin/soda-setup"))
			require.NoError(t, e)
			require.Equal(t, "../libexec/soda/soda-setup", target)
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

// streamFixtures serves a minimal stable-stream document and registry
// index over local TLS the resolver trusts via SSL_CERT_FILE, then points
// the resolver endpoints at it. Every LoadBase path in these tests resolves
// live fixtures; production endpoints are never contacted.
func streamFixtures(t *testing.T, streamDoc, indexDoc string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/streams/stable.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, streamDoc)
	})
	mux.HandleFunc("/v2/fedora/fedora-coreos/manifests/stable", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, indexDoc)
	})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	cert := filepath.Join(t.TempDir(), "cert.pem")
	require.NoError(t, os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600))
	t.Setenv("SSL_CERT_FILE", cert)
	t.Setenv("SODA_COREOS_STREAM_URL", server.URL+"/streams/stable.json")
	t.Setenv("SODA_COREOS_REGISTRY", server.URL)
}

func fixtureDisk(base, file string, uncompressed bool) string {
	loc := base + "/" + file
	disk := fmt.Sprintf(`{"location":%q,"sha256":%q,"signature":%q`, loc, strings.Repeat("a", 64), loc+".sig")
	if uncompressed {
		disk += fmt.Sprintf(`,"uncompressed-sha256":%q`, strings.Repeat("b", 64))
	}
	return disk + "}"
}

// goodStreamDoc builds a minimal valid stream document for both arches.
func goodStreamDoc() string {
	arches := []string{}
	for _, arch := range []string{"x86_64", "aarch64"} {
		base := fmt.Sprintf("https://builds.test/prod/streams/stable/builds/44.20260901.1.0/%s/fedora-coreos-44.20260901.1.0", arch)
		arches = append(arches, fmt.Sprintf(`%q:{"artifacts":{"metal":{"formats":{"iso":{"disk":%s}}},"qemu":{"formats":{"qcow2.xz":{"disk":%s}}}}}`, arch,
			fixtureDisk(base, "fedora-coreos-44.20260901.1.0-live."+arch+".iso", false),
			fixtureDisk(base, "fedora-coreos-44.20260901.1.0-qemu."+arch+".qcow2.xz", true)))
	}
	return `{"architectures":{` + strings.Join(arches, ",") + `}}`
}

func goodIndexDoc() string {
	return `{"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[` +
		`{"digest":"sha256:` + strings.Repeat("c", 64) + `","platform":{"architecture":"amd64"}},` +
		`{"digest":"sha256:` + strings.Repeat("d", 64) + `","platform":{"architecture":"arm64"}}]}`
}

func TestBaseStreamAndUnsafeOutputRefusal(t *testing.T) {
	streamFixtures(t, goodStreamDoc(), goodIndexDoc())
	_, err := LoadBase("armv7")
	require.Error(t, err)
	base, err := LoadBase("x86_64")
	require.NoError(t, err)
	require.Equal(t, "44.20260901.1.0", base.Release)
	require.Contains(t, base.Images["x86_64"], "@sha256:"+strings.Repeat("c", 64))
	require.Contains(t, base.Images["aarch64"], "@sha256:"+strings.Repeat("d", 64))
	require.Contains(t, base.MetadataURL, "/builds/44.20260901.1.0/release.json")
	t.Run("admitted file", func(t *testing.T) {
		resolved, err := build.ResolveCoreOS(context.Background())
		require.NoError(t, err)
		inputs := build.LiveInputs{CoreOS: resolved, Tailnet: build.TailnetInputs{
			Version: "1.2.3",
			SHA256:  strings.Repeat("a", 64),
			Base:    "docker.io/tailscale/alpine-base:3.22",
		}}
		path := filepath.Join(t.TempDir(), "live-inputs.json")
		require.NoError(t, build.WriteLiveInputs(path, inputs))
		filed, err := LoadBaseFromFile(path, "x86_64")
		require.NoError(t, err)
		require.Equal(t, base, filed)
		require.NoError(t, os.WriteFile(path, []byte(`{"release":"yesterday"}`), 0o600))
		_, err = LoadBaseFromFile(path, "x86_64")
		require.Error(t, err)
	})
	for name, doc := range map[string]string{
		"missing arch":   strings.Replace(goodStreamDoc(), `"aarch64":{"artifacts"`, `"ppc64le":{"artifacts"`, 1),
		"bad digest":     strings.Replace(goodStreamDoc(), strings.Repeat("a", 64), "zz", 1),
		"bad release":    strings.Replace(goodStreamDoc(), "44.20260901.1.0", "yesterday", -1),
		"bad image ref":  `{"architectures":{}}`,
		"untrusted host": strings.Replace(goodStreamDoc(), "https://builds.test/", "http://builds.test/", -1),
	} {
		t.Run(name, func(t *testing.T) {
			streamFixtures(t, doc, goodIndexDoc())
			_, err := LoadBase("x86_64")
			require.Error(t, err)
		})
	}
	t.Run("registry failure", func(t *testing.T) {
		streamFixtures(t, goodStreamDoc(), `{"manifests":[]}`)
		_, err := LoadBase("x86_64")
		require.Error(t, err)
	})
	source := sourceRoot(t)
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
	require.Contains(t, string(b), "&& \\\n    ostree container commit")
	require.Contains(t, string(b), "test ! -s /etc/subuid && test ! -s /etc/subgid")
	require.Contains(t, string(b), "printf 'containers:1000000:268435456\\n' > /etc/subuid")
	require.Contains(t, string(b), "cp /etc/subuid /etc/subgid")
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
