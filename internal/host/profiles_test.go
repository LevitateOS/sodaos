package host

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/levitateos/sodaos/internal/projectos"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func testProfile() projectos.Profile {
	return projectos.Profile{ID: projectos.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: runtime.GOARCH, Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
}
func testImage(p projectos.Profile) []byte {
	raw, _ := json.Marshal(map[string]any{"Id": strings.TrimPrefix(p.Image, "sha256:"), "Os": "linux", "Architecture": p.Architecture, "Labels": map[string]string{"org.soda.profile": p.ID, "org.soda.distribution": p.Distribution, "org.soda.distribution.version": p.Version, "org.soda.interface": p.Interface, "org.opencontainers.image.revision": p.Revision}})
	return raw
}

type profileExec struct {
	raw   []byte
	calls int
}

func (e *profileExec) Run(_ context.Context, _ []byte, name string, args ...string) ([]byte, error) {
	e.calls++
	if name != "/usr/bin/podman" || !reflect.DeepEqual(args, []string{"image", "inspect", "--format", `{"Id":{{json .ID}},"Architecture":{{json .Architecture}},"Os":{{json .Os}},"Labels":{{json .Labels}}}`, "configured-image"}) {
		return nil, errors.New("unexpected command")
	}
	return e.raw, nil
}

type profileInspection struct{ raw []byte }

func (e profileInspection) Run(_ context.Context, _ []byte, name string, args ...string) ([]byte, error) {
	if name != "/usr/bin/podman" || !reflect.DeepEqual(args, []string{"inspect", "soda-p0123456789abcdef01234567"}) {
		return nil, errors.New("unexpected inspection command")
	}
	return e.raw, nil
}
func TestNativeCreationProfileIsObservedNotInferredFromDefault(t *testing.T) {
	for _, kind := range []string{"legacy", "matching", "image drift", "label drift", "invalid metadata"} {
		t.Run(kind, func(t *testing.T) {
			p := testProfile()
			raw, _ := json.Marshal(p)
			labels := map[string]string{"org.soda.project": "p0123456789abcdef01234567", "org.soda.owner": "1"}
			if kind != "legacy" {
				labels["org.soda.profile"] = p.ID
				labels["org.soda.creation-profile"] = string(raw)
			}
			image := p.Image
			switch kind {
			case "image drift":
				image = "sha256:" + strings.Repeat("c", 64)
			case "label drift":
				labels["org.soda.profile"] = "fedora-kde"
			case "invalid metadata":
				labels["org.soda.creation-profile"] = "null"
			}
			observation, _ := json.Marshal([]any{map[string]any{"Image": image, "Config": map[string]any{"Labels": labels}, "State": map[string]bool{"Running": false}}})
			d := Daemon{Exec: profileInspection{observation}, Config: Config{Image: "never-read-this-default"}}
			env, _, err := d.inspect(t.Context(), "p0123456789abcdef01234567")
			switch kind {
			case "legacy":
				if err != nil || env.Profile != nil {
					t.Fatal("legacy image default inferred", env, err)
				}
			case "matching":
				if err != nil || env.Profile == nil || *env.Profile != p {
					t.Fatal(env, err)
				}
			default:
				if err == nil {
					t.Fatal("drift accepted", env)
				}
			}
		})
	}
}

func TestOnlyCompleteNativeInstalledProfileAndNoTagCreation(t *testing.T) {
	for _, kind := range []string{"valid", "foreign", "missing revision", "unsupported", "changed image"} {
		t.Run(kind, func(t *testing.T) {
			p := testProfile()
			installed := p
			switch kind {
			case "foreign":
				installed.Architecture = "arm64"
				if runtime.GOARCH == "arm64" {
					installed.Architecture = "amd64"
				}
			case "missing revision":
				installed.Revision = ""
			case "unsupported":
				installed.ID = "fedora-kde"
			case "changed image":
				installed.Image = "sha256:" + strings.Repeat("c", 64)
			}
			e := &profileExec{raw: testImage(installed)}
			d := Daemon{Exec: e, Config: Config{Image: "configured-image"}}
			result, err := d.resolveProfile(t.Context())
			if kind == "valid" || kind == "changed image" {
				if err != nil || result != installed {
					t.Fatal(result, err)
				}
			} else if err == nil {
				t.Fatal("invalid profile accepted")
			}
			e.calls = 0
			if kind != "valid" {
				if _, err := d.create(t.Context(), Create{ID: "p0123456789abcdef01234567", Owner: 1, Profile: &p}); err == nil || e.calls != 1 {
					t.Fatal("preflight reached mutations", e.calls, err)
				}
			}
		})
	}
	n := &noExec{}
	d := Daemon{Exec: n}
	if _, err := d.create(t.Context(), Create{ID: "p0123456789abcdef01234567", Owner: 1}); err == nil || n.called {
		t.Fatal("missing profile reached native executor")
	}
}
