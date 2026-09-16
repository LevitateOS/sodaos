// Payload defines immutable appliance payload metadata, not an update
// authority. Signatures, channel freshness and upgrade admission are owned by
// the signing and publication work in this package; an unsigned candidate is
// never approval.
package deliver

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
)

const (
	Path       = "/usr/share/soda/release.json"
	ImagesPath = "/usr/share/soda/images"
)

var Names = []string{"dashboard", "forgejo", "proxy", "project-os", "tailnet"}

type Image struct {
	Reference     string
	Config        string
	Manifest      string
	ArchiveSHA256 string // Detached export provenance; the host embeds shared OCI blobs.
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

func (p Payload) validIdentity() bool {
	return p.Format == 3 && build.Revision(p.Revision) && p.ID == p.CoreOS+".soda-"+p.Revision[:12] && regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(p.CoreOS)
}

func (p Payload) validBase() bool {
	// The base is the fedora-coreos repository at a digest; the registry
	// host floats (production quay.io, fixture servers in tests) while the
	// repository path and digest shape are fixed. Digests, not hosts, carry
	// integrity: provenance matching compares hashes, never names.
	host, digest, ok := strings.Cut(p.Base, "/fedora/fedora-coreos@sha256:")
	return ok && host != "" && !strings.Contains(host, "/") && build.Digest(digest) && p.Schema >= 1 && build.Digest(p.PresentationSHA256) && build.Digest(p.HostPackagesSHA256)
}

func (p Payload) validImages() error {
	if len(p.Images) != len(Names) {
		return errors.New("complete image set required")
	}
	for _, name := range Names {
		im, ok := p.Images[name]
		if !ok || !digest(im.Config) || !digest(im.Manifest) || !build.Digest(im.ArchiveSHA256) || im.Reference != p.RepositoryPrefix+"-"+name+"@"+im.Manifest {
			return fmt.Errorf("invalid %s image binding", name)
		}
	}
	return nil
}

func (p Payload) Validate() error {
	if !p.validIdentity() {
		return errors.New("invalid appliance payload identity")
	}
	if _, err := build.OCIArchitecture(p.Architecture); err != nil {
		return err
	}
	if !ValidRepositoryPrefix(p.RepositoryPrefix) {
		return errors.New("explicit GHCR repository prefix required")
	}
	if !p.validBase() {
		return errors.New("incomplete appliance payload")
	}
	if err := p.validImages(); err != nil {
		return err
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
	return strings.HasPrefix(s, "sha256:") && build.Digest(strings.TrimPrefix(s, "sha256:"))
}

func Load(path string) (Payload, error) {
	var p Payload
	if err := build.ReadJSON(path, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}
