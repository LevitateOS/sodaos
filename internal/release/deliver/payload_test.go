package deliver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func fixture() Payload {
	p := Payload{Format: 3, CoreOS: "44.20260817.3.2", Revision: strings.Repeat("a", 40), Architecture: "x86_64", Schema: 10, RepositoryPrefix: "ghcr.io/example/sodaos", Base: "quay.io/fedora/fedora-coreos@sha256:" + strings.Repeat("b", 64), PresentationSHA256: strings.Repeat("c", 64), HostPackagesSHA256: strings.Repeat("d", 64), Images: map[string]Image{}}
	p.ID = p.CoreOS + ".soda-" + p.Revision[:12]
	for _, n := range Names {
		p.Images[n] = Image{Reference: p.RepositoryPrefix + "-" + n + "@sha256:" + strings.Repeat("e", 64), Manifest: "sha256:" + strings.Repeat("e", 64), Config: "sha256:" + strings.Repeat("f", 64), ArchiveSHA256: strings.Repeat("1", 64)}
	}
	return p
}
func TestPayloadValidation(t *testing.T) {
	require.NoError(t, fixture().Validate())
	for _, mutate := range []func(*Payload){
		func(p *Payload) { p.Format = 2 }, func(p *Payload) { p.Revision = "dirty" }, func(p *Payload) { p.Architecture = "armv7" }, func(p *Payload) { p.PresentationSHA256 = "" }, func(p *Payload) { p.UpgradeFrom = []string{"unproved"} }, func(p *Payload) { p.RepositoryPrefix = "ghcr.io/example/soda\nImage=untrusted" }, func(p *Payload) { delete(p.Images, "proxy") }, func(p *Payload) {
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
	require.NoError(t, os.WriteFile(path, append(b, []byte(` {"unexpected":true}`)...), 0o644))
	_, e = Load(path)
	require.Error(t, e)
}
func writeArchive(t *testing.T, path string, p Payload) Image {
	t.Helper()
	im := testoci.Archive(t, path, "amd64", p.Revision)
	return Image{Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256}
}

type statusError int

func (e statusError) Error() string { return "synthetic native status" }
func (e statusError) ExitCode() int { return int(e) }
