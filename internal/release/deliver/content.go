package deliver

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
)

// VerifyContent binds the shared OCI content to the payload before import or disk
// writes. Each shared blob is hashed and counted once; export tar files are not embedded.
func bindImageRevisions(p Payload) (map[string]string, error) {
	revisions := map[string]string{}
	for _, name := range Names {
		revision := p.Revision
		if name == "proxy" {
			revision = ""
		}
		ref := p.Images[name].Config
		if previous, ok := revisions[ref]; ok && previous != revision {
			return nil, errors.New("conflicting OCI source bindings")
		}
		revisions[ref] = revision
	}
	return revisions, nil
}

func matchLayoutIdentities(p Payload, layout build.OCILayout) error {
	for _, name := range Names {
		expected := p.Images[name]
		got := layout.Images[expected.Config]
		if got.Config != expected.Config || got.Manifest != expected.Manifest {
			return fmt.Errorf("%s OCI identity mismatch", name)
		}
	}
	return nil
}

func VerifyContent(p Payload, images string) (map[string]string, uint64, error) {
	if err := p.Validate(); err != nil {
		return nil, 0, err
	}
	if !filepath.IsAbs(images) || strings.ContainsAny(images, ":\r\n") {
		return nil, 0, errors.New("absolute local OCI directory required")
	}
	revisions, err := bindImageRevisions(p)
	if err != nil {
		return nil, 0, err
	}
	layout, err := build.InspectOCILayout(images, p.Architecture, revisions)
	if err != nil {
		return nil, 0, err
	}
	if err = matchLayoutIdentities(p, layout); err != nil {
		return nil, 0, err
	}
	return layout.Files, layout.Bytes, nil
}
