package releasedelivery

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

const manifestType = "application/vnd.oci.image.manifest.v1+json"
const layerType = "application/vnd.oci.image.layer.v1.tar"

type descriptor struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}
type manifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Config        descriptor   `json:"config"`
	Layers        []descriptor `json:"layers"`
}

// WriteDocument packages one bounded JSON record using the standard OCI image
// layout. Native skopeo owns transport and signatures; this image is never run.
func WriteDocument(path string, value any) (string, error) {
	data, e := marshal(value)
	if e != nil || len(data) > 1<<20 {
		return "", ErrRefused
	}
	if e = nativebuild.FreshDirectory(path); e != nil {
		return "", e
	}
	blobs := filepath.Join(path, "blobs/sha256")
	if e = os.MkdirAll(blobs, 0700); e != nil {
		return "", e
	}
	put := func(b []byte, media string) (descriptor, error) {
		h := Hash(b)
		e := nativebuild.WriteNew(filepath.Join(blobs, strings.TrimPrefix(h, "sha256:")), b, 0600)
		return descriptor{media, h, int64(len(b))}, e
	}
	var layer bytes.Buffer
	tw := tar.NewWriter(&layer)
	if e = tw.WriteHeader(&tar.Header{Name: "record.json", Mode: 0444, Size: int64(len(data)), Typeflag: tar.TypeReg}); e != nil {
		return "", e
	}
	if _, e = tw.Write(data); e != nil {
		return "", e
	}
	if e = tw.Close(); e != nil {
		return "", e
	}
	ld, e := put(layer.Bytes(), layerType)
	if e != nil {
		return "", e
	}
	config, _ := json.Marshal(map[string]any{"architecture": "unknown", "os": "unknown", "rootfs": map[string]any{"type": "layers", "diff_ids": []string{ld.Digest}}})
	cd, e := put(config, "application/vnd.oci.image.config.v1+json")
	if e != nil {
		return "", e
	}
	mb, _ := json.Marshal(manifest{2, manifestType, cd, []descriptor{ld}})
	md, e := put(mb, manifestType)
	if e != nil {
		return "", e
	}
	index, _ := json.Marshal(map[string]any{"schemaVersion": 2, "manifests": []descriptor{md}})
	if e = nativebuild.WriteNew(filepath.Join(path, "index.json"), index, 0600); e != nil {
		return "", e
	}
	e = nativebuild.WriteNew(filepath.Join(path, "oci-layout"), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0600)
	return md.Digest, e
}

func ReadFile(path string, maximum int64) ([]byte, error) {
	root, e := os.OpenRoot(filepath.Dir(path))
	if e != nil {
		return nil, e
	}
	defer root.Close()
	return readAt(root, filepath.Base(path), maximum)
}
func readAt(root *os.Root, path string, maximum int64) ([]byte, error) {
	st, e := root.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > maximum {
		return nil, ErrRefused
	}
	f, e := root.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !os.SameFile(st, actual) {
		return nil, ErrRefused
	}
	b, e := io.ReadAll(io.LimitReader(f, maximum+1))
	if e != nil || int64(len(b)) > maximum {
		return nil, ErrRefused
	}
	return b, nil
}
func ReadJSON(path string, v any) error {
	b, e := ReadFile(path, 1<<20)
	if e != nil {
		return e
	}
	return decode(b, v)
}

// ReadDocument operates only on a fresh dir: copy already verified by native
// signature policy. Verify every descriptor and reject extraction/path tricks.
// Calling this hash reader alone is deliberately NOT signature verification.
func ReadDocument(path, want string, v any) error {
	root, e := os.OpenRoot(path)
	if e != nil {
		return e
	}
	defer root.Close()
	mb, e := readAt(root, "manifest.json", 64<<10)
	if e != nil || Hash(mb) != want {
		return ErrRefused
	}
	var m manifest
	if e = decode(mb, &m); e != nil || m.SchemaVersion != 2 || m.MediaType != manifestType || len(m.Layers) != 1 || m.Layers[0].MediaType != layerType || m.Config.MediaType != "application/vnd.oci.image.config.v1+json" {
		return ErrRefused
	}
	blob := func(d descriptor) ([]byte, error) {
		if !Digest(d.Digest) || d.Size < 0 || d.Size > 2<<20 {
			return nil, ErrRefused
		}
		b, e := readAt(root, strings.TrimPrefix(d.Digest, "sha256:"), 2<<20)
		if e != nil || int64(len(b)) != d.Size || Hash(b) != d.Digest {
			return nil, ErrRefused
		}
		return b, nil
	}
	if _, e = blob(m.Config); e != nil {
		return e
	}
	data, e := blob(m.Layers[0])
	if e != nil {
		return e
	}
	tr := tar.NewReader(bytes.NewReader(data))
	h, e := tr.Next()
	if e != nil || h.Name != "record.json" || h.Typeflag != tar.TypeReg || h.Size > 1<<20 {
		return ErrRefused
	}
	record, e := io.ReadAll(io.LimitReader(tr, (1<<20)+1))
	if e != nil {
		return e
	}
	if _, e = tr.Next(); !errors.Is(e, io.EOF) {
		return ErrRefused
	}
	return decode(record, v)
}
