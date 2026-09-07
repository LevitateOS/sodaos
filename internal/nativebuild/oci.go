// Identity checks adapted from soda-os bc1d3e0 release/inspection.go.
// Stream real OCI archives without extracting layers or importing release code.
package nativebuild

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

type Image struct{ Manifest, Config, Architecture, Revision, Source, BaseName, BaseDigest string }
type descriptor struct {
	Digest    string   `json:"digest"`
	Size      int64    `json:"size"`
	MediaType string   `json:"mediaType"`
	URLs      []string `json:"urls,omitempty"`
}
type blob struct {
	hash string
	size int64
	data []byte
}

func InspectOCI(file, arch, revision string) (Image, error) {
	want, err := OCIArchitecture(arch)
	if err != nil {
		return Image{}, err
	}
	if revision != "" && !Revision(revision) {
		return Image{}, errors.New("full source revision required")
	}
	st, err := os.Lstat(file)
	if err != nil {
		return Image{}, err
	}
	if !st.Mode().IsRegular() {
		return Image{}, errors.New("OCI archive must be regular")
	}
	f, err := os.Open(file)
	if err != nil {
		return Image{}, err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	entries := map[string]blob{}
	seen := map[string]bool{}
	jsonBytes := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Image{}, err
		}
		n := path.Clean(h.Name)
		if n == "." && h.Typeflag == tar.TypeDir {
			continue
		}
		if path.IsAbs(n) || n == ".." || strings.HasPrefix(n, "../") {
			return Image{}, errors.New("unsafe OCI path")
		}
		if seen[n] {
			return Image{}, errors.New("duplicate OCI entry")
		}
		seen[n] = true
		if len(seen) > 100000 {
			return Image{}, errors.New("too many OCI entries")
		}
		if h.Typeflag == tar.TypeDir {
			if n != "blobs" && n != "blobs/sha256" {
				return Image{}, errors.New("unexpected OCI directory")
			}
			continue
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
			return Image{}, errors.New("non-regular OCI entry")
		}
		if _, ok := entries[n]; ok {
			return Image{}, errors.New("duplicate OCI entry")
		}
		if len(entries) > 100000 {
			return Image{}, errors.New("too many OCI entries")
		}
		if n != "index.json" && n != "oci-layout" && !(strings.HasPrefix(n, "blobs/sha256/") && Digest(strings.TrimPrefix(n, "blobs/sha256/"))) {
			return Image{}, errors.New("not an OCI archive")
		}
		hash := sha256.New()
		var data bytes.Buffer
		var w io.Writer = hash
		if h.Size <= 4<<20 {
			w = io.MultiWriter(hash, &data)
		}
		size, err := io.Copy(w, tr)
		if err != nil {
			return Image{}, err
		}
		sum := hex.EncodeToString(hash.Sum(nil))
		if strings.HasPrefix(n, "blobs/") && n != "blobs/sha256/"+sum {
			return Image{}, errors.New("OCI blob checksum mismatch")
		}
		body := data.Bytes()
		if !json.Valid(body) {
			body = nil
		}
		jsonBytes += len(body)
		if jsonBytes > 32<<20 {
			return Image{}, errors.New("OCI JSON metadata limit exceeded")
		}
		entries[n] = blob{sum, size, body}
	}
	var layout struct {
		Version string `json:"imageLayoutVersion"`
	}
	if json.Unmarshal(entries["oci-layout"].data, &layout) != nil || layout.Version != "1.0.0" {
		return Image{}, errors.New("missing OCI layout")
	}
	var index struct {
		SchemaVersion int          `json:"schemaVersion"`
		MediaType     string       `json:"mediaType"`
		Manifests     []descriptor `json:"manifests"`
	}
	if json.Unmarshal(entries["index.json"].data, &index) != nil || index.SchemaVersion != 2 || (index.MediaType != "" && index.MediaType != "application/vnd.oci.image.index.v1+json") || len(index.Manifests) != 1 || index.Manifests[0].MediaType != "application/vnd.oci.image.manifest.v1+json" {
		return Image{}, errors.New("single-platform OCI index required")
	}
	get := func(d descriptor) (blob, error) {
		if d.Size < 0 || len(d.URLs) != 0 {
			return blob{}, errors.New("local bounded OCI descriptor required")
		}
		if !strings.HasPrefix(d.Digest, "sha256:") || !Digest(strings.TrimPrefix(d.Digest, "sha256:")) {
			return blob{}, errors.New("invalid OCI digest")
		}
		b, ok := entries["blobs/sha256/"+strings.TrimPrefix(d.Digest, "sha256:")]
		if !ok || b.size != d.Size {
			return blob{}, errors.New("missing or wrong-size OCI blob")
		}
		return b, nil
	}
	m, err := get(index.Manifests[0])
	if err != nil {
		return Image{}, err
	}
	var manifest struct {
		SchemaVersion int          `json:"schemaVersion"`
		MediaType     string       `json:"mediaType"`
		Config        descriptor   `json:"config"`
		Layers        []descriptor `json:"layers"`
	}
	if err = json.Unmarshal(m.data, &manifest); err != nil {
		return Image{}, err
	}
	if manifest.SchemaVersion != 2 || (manifest.MediaType != "" && manifest.MediaType != "application/vnd.oci.image.manifest.v1+json") || manifest.Config.MediaType != "application/vnd.oci.image.config.v1+json" {
		return Image{}, errors.New("invalid OCI image manifest")
	}
	for _, layer := range manifest.Layers {
		switch layer.MediaType {
		case "application/vnd.oci.image.layer.v1.tar", "application/vnd.oci.image.layer.v1.tar+gzip", "application/vnd.oci.image.layer.v1.tar+zstd":
		default:
			return Image{}, errors.New("unsupported OCI layer media type")
		}
		if _, err = get(layer); err != nil {
			return Image{}, err
		}
	}
	config, err := get(manifest.Config)
	if err != nil {
		return Image{}, err
	}
	var cfg struct {
		OS     string `json:"os"`
		Arch   string `json:"architecture"`
		RootFS struct {
			Type    string   `json:"type"`
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"config"`
	}
	if err = json.Unmarshal(config.data, &cfg); err != nil {
		return Image{}, err
	}
	if cfg.RootFS.Type != "layers" || len(cfg.RootFS.DiffIDs) != len(manifest.Layers) {
		return Image{}, errors.New("OCI rootfs/layer count mismatch")
	}
	for i, id := range cfg.RootFS.DiffIDs {
		if !strings.HasPrefix(id, "sha256:") || !Digest(strings.TrimPrefix(id, "sha256:")) {
			return Image{}, errors.New("invalid OCI diff ID")
		}
		if manifest.Layers[i].MediaType == "application/vnd.oci.image.layer.v1.tar" && id != manifest.Layers[i].Digest {
			return Image{}, errors.New("uncompressed OCI layer identity mismatch")
		}
	}
	// Compressed layer contents are not extracted here. Their blob identities
	// are checked; native import remains the proof of decompression/rootfs use.
	if cfg.OS != "linux" || cfg.Arch != want {
		return Image{}, fmt.Errorf("OCI must be linux/%s", want)
	}
	rev := cfg.Config.Labels["org.opencontainers.image.revision"]
	if revision != "" && rev != revision {
		return Image{}, errors.New("OCI source revision mismatch")
	}
	labels := cfg.Config.Labels
	if revision != "" && (labels["org.opencontainers.image.source"] != "https://github.com/LevitateOS/sodaos" || labels["org.opencontainers.image.base.name"] == "" || !strings.HasPrefix(labels["org.opencontainers.image.base.digest"], "sha256:") || !Digest(strings.TrimPrefix(labels["org.opencontainers.image.base.digest"], "sha256:"))) {
		return Image{}, errors.New("Soda image lacks source/base attribution")
	}
	return Image{index.Manifests[0].Digest, manifest.Config.Digest, cfg.Arch, rev, labels["org.opencontainers.image.source"], labels["org.opencontainers.image.base.name"], labels["org.opencontainers.image.base.digest"]}, nil
}
