package appliancerelease

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// VerifyContent binds the shared OCI content to the payload before import or disk
// writes. Each shared blob is hashed and counted once; export tar files are not embedded.
func VerifyContent(p Payload, images string) (map[string]string, uint64, error) {
	if err := p.Validate(); err != nil {
		return nil, 0, err
	}
	if !filepath.IsAbs(images) || strings.ContainsAny(images, ":\r\n") {
		return nil, 0, errors.New("absolute local OCI directory required")
	}
	revisions := map[string]string{}
	for _, name := range Names {
		revision := p.Revision
		if name == "proxy" {
			revision = ""
		}
		ref := p.Images[name].Config
		if previous, ok := revisions[ref]; ok && previous != revision {
			return nil, 0, errors.New("conflicting OCI source bindings")
		}
		revisions[ref] = revision
	}
	layout, err := nativebuild.InspectOCILayout(images, p.Architecture, revisions)
	if err != nil {
		return nil, 0, err
	}
	for _, name := range Names {
		expected := p.Images[name]
		got := layout.Images[expected.Config]
		if got.Config != expected.Config || got.Manifest != expected.Manifest {
			return nil, 0, fmt.Errorf("%s OCI identity mismatch", name)
		}
	}
	return layout.Files, layout.Bytes, nil
}
