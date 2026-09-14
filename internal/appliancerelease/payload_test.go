package appliancerelease

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func fixture() Payload {
	p := Payload{Format: 1, CoreOS: "44.20260817.3.2", Revision: strings.Repeat("a", 40), Architecture: "x86_64", Schema: 10, RepositoryPrefix: "ghcr.io/example/sodaos", Base: "quay.io/fedora/fedora-coreos@sha256:" + strings.Repeat("b", 64), PresentationSHA256: strings.Repeat("c", 64), HostPackagesSHA256: strings.Repeat("d", 64), Images: map[string]Image{}}
	p.ID = p.CoreOS + ".soda-" + p.Revision[:12]
	for _, n := range Names {
		s := "bound"
		if n == "project-os" || n == "tailnet" {
			s = "retained"
		}
		p.Images[n] = Image{Reference: p.RepositoryPrefix + "-" + n + "@sha256:" + strings.Repeat("e", 64), Manifest: "sha256:" + strings.Repeat("e", 64), Config: "sha256:" + strings.Repeat("f", 64), ArchiveSHA256: strings.Repeat("1", 64), Storage: s}
	}
	return p
}
func TestPayloadValidation(t *testing.T) {
	require.NoError(t, fixture().Validate())
	for _, mutate := range []func(*Payload){
		func(p *Payload) { p.Format = 2 }, func(p *Payload) { p.Revision = "dirty" }, func(p *Payload) { p.Architecture = "armv7" },
		func(p *Payload) { p.PresentationSHA256 = "" }, func(p *Payload) { p.UpgradeFrom = []string{"unproved"} },
		func(p *Payload) { p.RepositoryPrefix = "ghcr.io/example/soda\nImage=untrusted" },
		func(p *Payload) { delete(p.Images, "proxy") },
		func(p *Payload) { im := p.Images["project-os"]; im.Storage = "bound"; p.Images["project-os"] = im },
		func(p *Payload) {
			im := p.Images["dashboard"]
			im.Reference = "ghcr.io/example/dashboard:latest"
			p.Images["dashboard"] = im
		},
	} {
		p := fixture()
		mutate(&p)
		require.Error(t, p.Validate())
	}
	path := filepath.Join(t.TempDir(), "release.json")
	b, e := json.Marshal(fixture())
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(path, append(b, []byte(` {"unexpected":true}`)...), 0644))
	_, e = Load(path)
	require.Error(t, e)
}

// Tiny standard OCI fixtures exercise the existing streaming verifier, not a
// replacement verifier or an actual Podman engine/provider.
func writeArchive(t *testing.T, path string, p Payload) Image {
	t.Helper()
	im := testoci.Archive(t, path, "amd64", p.Revision)
	return Image{Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256, Storage: "retained"}
}

type statusError int

func (e statusError) Error() string { return "synthetic native status" }
func (e statusError) ExitCode() int { return int(e) }

func TestRetainedImportUsesOnlyExactArchivesAndNeverLifecycle(t *testing.T) {
	p := fixture()
	root := t.TempDir()
	for _, n := range []string{"project-os", "tailnet"} {
		im := writeArchive(t, filepath.Join(root, n+".oci"), p)
		im.Reference = p.RepositoryPrefix + "-" + n + "@" + im.Manifest
		p.Images[n] = im
	}
	for _, status := range []int{0, 1, 125} {
		calls := []string{}
		loaded := false
		err := ImportRetained(t.Context(), p, root, func(_ context.Context, cmd string, args ...string) error {
			require.Equal(t, "/usr/bin/podman", cmd)
			require.Equal(t, "--remote=false", args[0])
			calls = append(calls, strings.Join(args, " "))
			if args[1] == "load" {
				require.Equal(t, []string{"load", "--input"}, args[1:3])
				require.True(t, strings.HasPrefix(args[3], root+"/"))
				loaded = true
				return nil
			}
			require.Equal(t, []string{"image", "exists"}, args[1:3])
			if status == 0 || loaded {
				return nil
			}
			return statusError(status)
		})
		if status == 125 {
			require.Error(t, err)
			require.Len(t, calls, 1)
		} else {
			require.NoError(t, err)
		}
		require.NotContains(t, strings.Join(calls, "\n"), "additionalimagestore")
		require.NotContains(t, strings.Join(calls, "\n"), " start ")
		require.NotContains(t, strings.Join(calls, "\n"), " rm ")
	}
	for _, failLoad := range []bool{true, false} {
		queries, loads := 0, 0
		err := ImportRetained(t.Context(), p, root, func(_ context.Context, _ string, args ...string) error {
			if args[1] == "load" {
				loads++
				if failLoad {
					return errors.New("DO_NOT_LOG raw native diagnostic")
				}
				return nil
			}
			queries++
			return statusError(1)
		})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "DO_NOT_LOG")
		require.Equal(t, 1, loads, "never replay a failed/unconfirmed import")
		if failLoad {
			require.Equal(t, 1, queries)
			require.ErrorContains(t, err, "import unconfirmed")
		} else {
			require.Equal(t, 2, queries)
			require.ErrorContains(t, err, "imported image unavailable")
		}
	}
	wrong := fixture()
	wrong.Images = make(map[string]Image)
	for k, v := range p.Images {
		wrong.Images[k] = v
	}
	im := wrong.Images["project-os"]
	im.Manifest = "sha256:" + strings.Repeat("0", 64)
	im.Reference = wrong.RepositoryPrefix + "-project-os@" + im.Manifest
	wrong.Images["project-os"] = im
	require.ErrorContains(t, ImportRetained(t.Context(), wrong, root, func(context.Context, string, ...string) error { t.Fatal("wrong image identity dispatched"); return nil }), "identity mismatch")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, ImportRetained(ctx, p, root, func(context.Context, string, ...string) error { t.Fatal("cancelled import dispatched"); return nil }), context.Canceled)
	require.NoError(t, os.WriteFile(filepath.Join(root, "project-os.oci"), []byte("tampered"), 0644))
	require.Error(t, ImportRetained(t.Context(), p, root, func(context.Context, string, ...string) error {
		t.Fatal("tampered archive dispatched")
		return errors.New("unreachable")
	}))
}
