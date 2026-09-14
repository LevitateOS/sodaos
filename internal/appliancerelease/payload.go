// Package appliancerelease defines immutable appliance payload metadata, not an
// update authority. Signatures, channel freshness and upgrade admission are owned
// by the later release-delivery work; an unsigned candidate is never approval.
package appliancerelease

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

const Path = "/usr/share/soda/release.json"
const ImagesPath = "/usr/share/soda/images"

var Names = []string{"dashboard", "forgejo", "proxy", "project-os", "tailnet"}

type Image struct {
	Reference     string
	Config        string
	Manifest      string
	ArchiveSHA256 string
	Storage       string // v1: historical bound/retained; v2: ordinary Podman
}

type Payload struct {
	Format             int
	ID                 string
	Revision           string
	Architecture       string
	CoreOS             string
	Base               string
	RepositoryPrefix   string
	Schema             int
	PresentationSHA256 string
	HostPackagesSHA256 string
	Images             map[string]Image
	// Empty until native upgrade qualification selects actual supported sources.
	UpgradeFrom []string
}

func (p Payload) Validate() error {
	if (p.Format != 1 && p.Format != 2) || !nativebuild.Revision(p.Revision) || p.ID != p.CoreOS+".soda-"+p.Revision[:12] || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(p.CoreOS) {
		return errors.New("invalid appliance payload identity")
	}
	if _, err := nativebuild.OCIArchitecture(p.Architecture); err != nil {
		return err
	}
	if !ValidRepositoryPrefix(p.RepositoryPrefix) {
		return errors.New("explicit GHCR repository prefix required")
	}
	const basePrefix = "quay.io/fedora/fedora-coreos@sha256:"
	if !strings.HasPrefix(p.Base, basePrefix) || !nativebuild.Digest(strings.TrimPrefix(p.Base, basePrefix)) || p.Schema < 1 || !nativebuild.Digest(p.PresentationSHA256) || !nativebuild.Digest(p.HostPackagesSHA256) {
		return errors.New("incomplete appliance payload")
	}
	if len(p.Images) != len(Names) {
		return errors.New("complete image set required")
	}
	for _, name := range Names {
		im, ok := p.Images[name]
		want := "bound"
		if name == "project-os" || name == "tailnet" {
			want = "retained"
		}
		if p.Format == 2 {
			want = "podman"
		}
		if !ok || im.Storage != want || !digest(im.Config) || !digest(im.Manifest) || !nativebuild.Digest(im.ArchiveSHA256) || im.Reference != p.RepositoryPrefix+"-"+name+"@"+im.Manifest {
			return fmt.Errorf("invalid %s image binding", name)
		}
	}
	// Native admission is not implemented in this milestone. Do not publish a
	// guessed compatibility promise simply because schemas happen to match.
	if len(p.UpgradeFrom) != 0 {
		return errors.New("candidate has no qualified upgrade paths")
	}
	return nil
}
func ValidRepositoryPrefix(s string) bool {
	return len(s) < 200 && regexp.MustCompile(`^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`).MatchString(s)
}
func digest(s string) bool {
	return strings.HasPrefix(s, "sha256:") && nativebuild.Digest(strings.TrimPrefix(s, "sha256:"))
}
func Load(path string) (Payload, error) {
	var p Payload
	if err := nativebuild.ReadJSON(path, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}
