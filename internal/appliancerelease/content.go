package appliancerelease

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// VerifyContent binds installed content to payload identities before any import
// or disk write. V2 keeps its archive hashes; V3 verifies one standard OCI layout,
// with each shared blob hashed and counted once. Export archives remain provenance
// and delivery inputs, not additional embedded copies in V3.
func VerifyContent(p Payload, images string) (map[string]string, uint64, error) {
	if err := p.Validate(); err != nil {
		return nil, 0, err
	}
	if p.Format == 3 {
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
	if p.Format != 2 {
		return nil, 0, errors.New("candidate content requires payload v2 or v3")
	}
	files := map[string]string{}
	var total uint64
	for _, name := range Names {
		file := name + ".oci"
		path := filepath.Join(images, file)
		st, err := os.Lstat(path)
		if err != nil || !st.Mode().IsRegular() || st.Size() <= 0 || uint64(st.Size()) > ^uint64(0)-total {
			return nil, 0, errors.New("invalid local application archive")
		}
		hash, err := nativebuild.HashFile(path)
		if err != nil || hash != p.Images[name].ArchiveSHA256 {
			return nil, 0, errors.New("local application archive differs from candidate")
		}
		files[file] = hash
		total += uint64(st.Size())
	}
	return files, total, nil
}
