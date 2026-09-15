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

type ociLayoutLoader struct {
	root      *os.Root
	entries   map[string]blob
	jsonBytes int
}

func (l *ociLayoutLoader) load(name string) error {
	if _, ok := l.entries[name]; ok {
		return nil
	}
	st, err := l.root.Lstat(name)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return errors.New("non-regular OCI layout entry")
	}
	f, err := l.root.Open(name)
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
	return readOCIBlob(l.entries, name, st.Size(), f, &l.jsonBytes)
}

func validateOCILayoutInputs(arch string, revisions map[string]string) (string, error) {
	want, err := OCIArchitecture(arch)
	if err != nil {
		return "", err
	}
	if len(revisions) == 0 {
		return "", errors.New("explicit OCI image set required")
	}
	for ref, rev := range revisions {
		if !strings.HasPrefix(ref, "sha256:") || !Digest(strings.TrimPrefix(ref, "sha256:")) || (rev != "" && !Revision(rev)) {
			return "", errors.New("invalid OCI image selection")
		}
	}
	return want, nil
}

func openOCILayoutRoot(dir string) (*os.Root, error) {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return nil, errors.New("real OCI layout directory required")
	}
	return os.OpenRoot(dir)
}

func inspectOCILayoutDescriptor(entries map[string]blob, desc descriptor, want string, revisions map[string]string, images map[string]Image, load func(string) error) (string, Image, error) {
	ref := desc.Annotations["org.opencontainers.image.ref.name"]
	revision, ok := revisions[ref]
	if !ok || images[ref].Config != "" || desc.MediaType != "application/vnd.oci.image.manifest.v1+json" {
		return "", Image{}, errors.New("unexpected or duplicate OCI image reference")
	}
	image, err := inspectOCIImage(entries, desc, want, revision, load)
	if err != nil {
		return "", Image{}, err
	}
	if image.Config != ref {
		return "", Image{}, errors.New("OCI reference differs from config identity")
	}
	return ref, image, nil
}

func inspectOCILayoutImages(entries map[string]blob, index []descriptor, want string, revisions map[string]string, load func(string) error) (map[string]Image, error) {
	if len(index) != len(revisions) {
		return nil, errors.New("exact OCI image set required")
	}
	images := make(map[string]Image, len(index))
	for _, descriptor := range index {
		ref, image, err := inspectOCILayoutDescriptor(entries, descriptor, want, revisions, images, load)
		if err != nil {
			return nil, err
		}
		images[ref] = image
	}
	return images, nil
}

func tallyOCILayoutFiles(entries map[string]blob) (map[string]string, uint64) {
	files := make(map[string]string, len(entries))
	var totalBytes uint64
	for name, entry := range entries {
		files[name] = entry.hash
		totalBytes += uint64(entry.size)
	}
	return files, totalBytes
}

// InspectOCILayout verifies the exact named image set and its local blobs without
// running image code. References are immutable config IDs, as in our OCI exports.
// Archive inspection retains its single-image contract; it does not accept this
// multi-image index merely because a file happens to have an .oci suffix.
func InspectOCILayout(dir, arch string, revisions map[string]string) (OCILayout, error) {
	want, err := validateOCILayoutInputs(arch, revisions)
	if err != nil {
		return OCILayout{}, err
	}
	root, err := openOCILayoutRoot(dir)
	if err != nil {
		return OCILayout{}, err
	}
	defer root.Close()
	loader := &ociLayoutLoader{
		root:    root,
		entries: map[string]blob{},
	}
	for _, name := range []string{"index.json", "oci-layout"} {
		if err := loader.load(name); err != nil {
			return OCILayout{}, err
		}
	}
	index, err := readOCIIndex(loader.entries)
	if err != nil {
		return OCILayout{}, err
	}
	images, err := inspectOCILayoutImages(loader.entries, index, want, revisions, loader.load)
	if err != nil {
		return OCILayout{}, err
	}
	files, totalBytes := tallyOCILayoutFiles(loader.entries)
	return OCILayout{
		Images: images,
		Files:  files,
		Bytes:  totalBytes,
	}, nil
}
