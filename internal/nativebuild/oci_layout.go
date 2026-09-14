package nativebuild

import (
	"errors"
	"os"
	"strings"
)

// OCILayout describes the standard shared-blob directory, not a second image store.
// Files includes the index/layout hashes; Bytes counts each blob exactly once.
type OCILayout struct {
	Images map[string]Image
	Files  map[string]string
	Bytes  uint64
}

// InspectOCILayout verifies the exact named image set and its local blobs without
// running image code. References are immutable config IDs, as in our OCI exports.
// Archive inspection retains its single-image contract; it does not accept this
// multi-image index merely because a file happens to have an .oci suffix.
func InspectOCILayout(dir, arch string, revisions map[string]string) (OCILayout, error) {
	var result OCILayout
	want, err := OCIArchitecture(arch)
	if err != nil {
		return result, err
	}
	if len(revisions) == 0 {
		return result, errors.New("explicit OCI image set required")
	}
	for ref, rev := range revisions {
		if !strings.HasPrefix(ref, "sha256:") || !Digest(strings.TrimPrefix(ref, "sha256:")) || (rev != "" && !Revision(rev)) {
			return result, errors.New("invalid OCI image selection")
		}
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return result, errors.New("real OCI layout directory required")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return result, err
	}
	defer root.Close()
	entries := map[string]blob{}
	jsonBytes := 0
	load := func(name string) error {
		if _, ok := entries[name]; ok {
			return nil
		}
		st, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() {
			return errors.New("non-regular OCI layout entry")
		}
		f, err := root.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		opened, err := f.Stat()
		if err != nil {
			return err
		}
		if !os.SameFile(st, opened) {
			return errors.New("OCI layout entry changed")
		}
		return readOCIBlob(entries, name, st.Size(), f, &jsonBytes)
	}
	for _, name := range []string{"index.json", "oci-layout"} {
		if err := load(name); err != nil {
			return result, err
		}
	}
	index, err := readOCIIndex(entries)
	if err != nil {
		return result, err
	}
	if len(index) != len(revisions) {
		return result, errors.New("exact OCI image set required")
	}
	result.Images = map[string]Image{}
	for _, descriptor := range index {
		ref := descriptor.Annotations["org.opencontainers.image.ref.name"]
		revision, ok := revisions[ref]
		if !ok || result.Images[ref].Config != "" || descriptor.MediaType != "application/vnd.oci.image.manifest.v1+json" {
			return result, errors.New("unexpected or duplicate OCI image reference")
		}
		image, err := inspectOCIImage(entries, descriptor, want, revision, load)
		if err != nil {
			return result, err
		}
		if image.Config != ref {
			return result, errors.New("OCI reference differs from config identity")
		}
		result.Images[ref] = image
	}
	result.Files = map[string]string{}
	for name, entry := range entries {
		result.Files[name] = entry.hash
		result.Bytes += uint64(entry.size)
	}
	return result, nil
}
