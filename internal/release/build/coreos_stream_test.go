package build

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureDisk(base, file string, uncompressed bool) map[string]string {
	loc := base + "/" + file
	disk := map[string]string{"location": loc, "sha256": strings.Repeat("a", 64), "signature": loc + ".sig"}
	if uncompressed {
		disk["uncompressed-sha256"] = strings.Repeat("b", 64)
	}
	return disk
}

// fixtureStreamDoc builds a minimal stable-stream document; mutate can
// corrupt one entry to prove fail-closed parsing.
func fixtureStreamDoc(t *testing.T, mutate func(arch, kind string, disk map[string]string)) string {
	t.Helper()
	arches := map[string]any{}
	for _, arch := range []string{"x86_64", "aarch64"} {
		base := fmt.Sprintf("https://builds.test/prod/streams/stable/builds/44.20260901.1.0/%s/fedora-coreos-44.20260901.1.0", arch)
		iso := fixtureDisk(base, "fedora-coreos-44.20260901.1.0-live."+arch+".iso", false)
		qemu := fixtureDisk(base, "fedora-coreos-44.20260901.1.0-qemu."+arch+".qcow2.xz", true)
		if mutate != nil {
			mutate(arch, "iso", iso)
			mutate(arch, "qcow2.xz", qemu)
		}
		arches[arch] = map[string]any{"artifacts": map[string]any{
			"metal": map[string]any{"formats": map[string]any{"iso": map[string]any{"disk": iso}}},
			"qemu":  map[string]any{"formats": map[string]any{"qcow2.xz": map[string]any{"disk": qemu}}},
		}}
	}
	data, err := json.Marshal(map[string]any{"architectures": arches})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func fixtureIndexDoc(t *testing.T, mutate func(entries []any) []any) string {
	t.Helper()
	entries := []any{
		map[string]any{"digest": "sha256:" + strings.Repeat("c", 64), "platform": map[string]any{"architecture": "amd64"}},
		map[string]any{"digest": "sha256:" + strings.Repeat("d", 64), "platform": map[string]any{"architecture": "arm64"}},
	}
	if mutate != nil {
		entries = mutate(entries)
	}
	data, err := json.Marshal(map[string]any{"mediaType": "application/vnd.oci.image.index.v1+json", "manifests": entries})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// streamFixtureServer serves a stream document and registry index over TLS
// the resolver client trusts, with both endpoints overridden by environment.
func streamFixtureServer(t *testing.T, streamBody, indexBody string, indexStatus int) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/streams/stable.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, streamBody)
	})
	mux.HandleFunc("/v2/fedora/fedora-coreos/manifests/stable", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.oci.image.index.v1+json" {
			t.Errorf("registry request wants an image index, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(indexStatus)
		fmt.Fprint(w, indexBody)
	})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	t.Setenv("SODA_COREOS_STREAM_URL", server.URL+"/streams/stable.json")
	t.Setenv("SODA_COREOS_REGISTRY", server.URL)
	previous := streamHTTPTransport
	streamHTTPTransport = server.Client().Transport
	t.Cleanup(func() { streamHTTPTransport = previous })
}

func TestResolveCoreOSFindsLiveBuild(t *testing.T) {
	streamFixtureServer(t, fixtureStreamDoc(t, nil), fixtureIndexDoc(t, nil), http.StatusOK)
	resolved, err := ResolveCoreOS(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Release != "44.20260901.1.0" {
		t.Fatalf("release parsed as %q", resolved.Release)
	}
	if !strings.HasSuffix(resolved.Container["x86_64"], "@sha256:"+strings.Repeat("c", 64)) {
		t.Fatalf("x86_64 container resolved as %q", resolved.Container["x86_64"])
	}
	if !strings.HasSuffix(resolved.Container["aarch64"], "@sha256:"+strings.Repeat("d", 64)) {
		t.Fatalf("aarch64 container resolved as %q", resolved.Container["aarch64"])
	}
	for _, arch := range []string{"x86_64", "aarch64"} {
		if !strings.HasSuffix(resolved.ISO[arch].URL, ".iso") || resolved.ISO[arch].SHA256 != strings.Repeat("a", 64) {
			t.Fatalf("%s ISO triple malformed: %+v", arch, resolved.ISO[arch])
		}
		if resolved.QEMU[arch].UncompressedSHA256 != strings.Repeat("b", 64) {
			t.Fatalf("%s qemu triple malformed: %+v", arch, resolved.QEMU[arch])
		}
	}
}

func TestResolveCoreOSRefusesBadStream(t *testing.T) {
	for name, mutate := range map[string]func(string, string, map[string]string){
		"bad digest": func(_, _ string, disk map[string]string) { disk["sha256"] = "nope" },
		"bad release": func(_, _ string, disk map[string]string) {
			disk["location"] = strings.Replace(disk["location"], "44.20260901.1.0", "yesterday", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			streamFixtureServer(t, fixtureStreamDoc(t, mutate), fixtureIndexDoc(t, nil), http.StatusOK)
			if _, err := ResolveCoreOS(context.Background()); err == nil {
				t.Fatal("malformed stream admitted")
			}
		})
	}
}

func TestResolveCoreOSRefusesRegistryFailure(t *testing.T) {
	streamFixtureServer(t, fixtureStreamDoc(t, nil), "{}", http.StatusUnauthorized)
	if _, err := ResolveCoreOS(context.Background()); err == nil {
		t.Fatal("refused registry admitted")
	}
}

func TestResolveCoreOSRequiresHTTPSStream(t *testing.T) {
	t.Setenv("SODA_COREOS_STREAM_URL", "http://builds.test/streams/stable.json")
	if _, err := ResolveCoreOS(context.Background()); err == nil {
		t.Fatal("plain-HTTP stream admitted")
	}
}

func liveInputsFixture(t *testing.T) LiveInputs {
	t.Helper()
	streamFixtureServer(t, fixtureStreamDoc(t, nil), fixtureIndexDoc(t, nil), http.StatusOK)
	resolved, err := ResolveCoreOS(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return LiveInputs{CoreOS: resolved, Tailnet: TailnetInputs{
		Version: "1.2.3",
		SHA256:  strings.Repeat("a", 64),
		Base:    "docker.io/tailscale/alpine-base:3.22",
	}}
}

func TestLiveInputsFileRoundTrip(t *testing.T) {
	inputs := liveInputsFixture(t)
	path := filepath.Join(t.TempDir(), "live-inputs.json")
	if err := WriteLiveInputs(path, inputs); err != nil {
		t.Fatal(err)
	}
	back, err := ReadLiveInputs(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.CoreOS.Release != inputs.CoreOS.Release || back.CoreOS.MetadataURL != inputs.CoreOS.MetadataURL {
		t.Fatal("inputs file lost the resolved release")
	}
	if back.CoreOS.Container["x86_64"] != inputs.CoreOS.Container["x86_64"] {
		t.Fatal("inputs file lost base digests")
	}
	if back.CoreOS.ISO["x86_64"] != inputs.CoreOS.ISO["x86_64"] || back.CoreOS.QEMU["aarch64"] != inputs.CoreOS.QEMU["aarch64"] {
		t.Fatal("inputs file lost media triples")
	}
	if back.Tailnet != inputs.Tailnet {
		t.Fatal("inputs file lost Tailnet inputs")
	}
}

func TestLiveInputsFileRefusesTampering(t *testing.T) {
	inputs := liveInputsFixture(t)
	inputs.CoreOS.Container["x86_64"] = "quay.io/fedora/fedora-coreos@sha256:nope"
	if err := WriteLiveInputs(filepath.Join(t.TempDir(), "bad.json"), inputs); err == nil {
		t.Fatal("undigest-pinned base admitted")
	}
	inputs = liveInputsFixture(t)
	inputs.Tailnet.Version = "yesterday"
	if err := WriteLiveInputs(filepath.Join(t.TempDir(), "bad-tailnet.json"), inputs); err == nil {
		t.Fatal("unversioned Tailnet admitted")
	}
	path := filepath.Join(t.TempDir(), "live-inputs.json")
	if err := WriteLiveInputs(path, liveInputsFixture(t)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), strings.Repeat("c", 64), strings.Repeat("z", 64), 1)
	if tampered == string(raw) {
		t.Fatal("fixture digest not found for tampering")
	}
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLiveInputs(path); err == nil {
		t.Fatal("on-disk digest swap admitted")
	}
	if err := os.WriteFile(path, []byte("{truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLiveInputs(path); err == nil {
		t.Fatal("corrupt inputs file admitted")
	}
}

func tailnetFixtureServer(t *testing.T) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/idx/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/idx/")
		if strings.HasSuffix(name, ".sha256") {
			fmt.Fprint(w, strings.Repeat("c", 64)+"  "+strings.TrimSuffix(name, ".sha256")+"\n")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
<a href="tailscale_1.9.9_amd64.tgz">old</a>
<a href="tailscale_1.10.2_arm64.tgz">new-arm</a>
<a href="tailscale_1.10.2_amd64.tgz">new</a>
<a href="tailscale_1.10.10_amd64.tgz">newest</a>
</body></html>`)
	})
	mux.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[{"name":"latest"},{"name":"3.16"},{"name":"3.22"},{"name":"edge"}]}`)
	})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	t.Setenv("SODA_TAILSCALE_INDEX_URL", server.URL+"/idx/")
	t.Setenv("SODA_TAILSCALE_BASE_TAGS_URL", server.URL+"/tags")
	previous := streamHTTPTransport
	streamHTTPTransport = server.Client().Transport
	t.Cleanup(func() { streamHTTPTransport = previous })
}

func TestResolveTailnetInputsFloats(t *testing.T) {
	tailnetFixtureServer(t)
	for _, arch := range []string{"x86_64", "aarch64"} {
		got, err := ResolveTailnetInputs(context.Background(), arch)
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != "1.10.10" {
			t.Fatalf("newest stable not selected for %s: %s", arch, got.Version)
		}
		if got.SHA256 != strings.Repeat("c", 64) {
			t.Fatalf("checksum not read for %s", arch)
		}
		if got.Base != "docker.io/tailscale/alpine-base:3.22" {
			t.Fatalf("newest base tag not selected for %s: %s", arch, got.Base)
		}
	}
	if _, err := ResolveTailnetInputs(context.Background(), "armv7"); err == nil {
		t.Fatal("unknown architecture admitted")
	}
}
