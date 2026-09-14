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
	"math"
	"os"
	"path"
	"strings"
)

type (
	Image      struct{ Manifest, Config, Architecture, Revision, Source, BaseName, BaseDigest string }
	descriptor struct {
		Digest      string            `json:"digest"`
		Size        int64             `json:"size"`
		MediaType   string            `json:"mediaType"`
		URLs        []string          `json:"urls,omitempty"`
		Annotations map[string]string `json:"annotations,omitempty"`
	}
)

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
		if h.Typeflag != tar.TypeReg {
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
		if err := readOCIBlob(entries, n, h.Size, tr, &jsonBytes); err != nil {
			return Image{}, err
		}
	}
	index, err := readOCIIndex(entries)
	if err != nil {
		return Image{}, err
	}
	if len(index) != 1 || index[0].MediaType != "application/vnd.oci.image.manifest.v1+json" {
		return Image{}, errors.New("single-platform OCI index required")
	}
	return inspectOCIImage(entries, index[0], want, revision, nil)
}

func readOCIBlob(entries map[string]blob, name string, length int64, r io.Reader, jsonBytes *int) error {
	if length < 0 || length == math.MaxInt64 {
		return errors.New("invalid OCI blob size")
	}
	hash := sha256.New()
	var data bytes.Buffer
	var w io.Writer = hash
	if length <= 4<<20 {
		w = io.MultiWriter(hash, &data)
	}
	size, err := io.Copy(w, io.LimitReader(r, length+1))
	if err != nil {
		return err
	}
	if size != length {
		return errors.New("OCI blob size changed")
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	if strings.HasPrefix(name, "blobs/") && name != "blobs/sha256/"+sum {
		return errors.New("OCI blob checksum mismatch")
	}
	body := data.Bytes()
	if !json.Valid(body) {
		body = nil
	}
	*jsonBytes += len(body)
	if *jsonBytes > 32<<20 {
		return errors.New("OCI JSON metadata limit exceeded")
	}
	entries[name] = blob{sum, size, body}
	return nil
}

func readOCIIndex(entries map[string]blob) ([]descriptor, error) {
	var layout struct {
		Version string `json:"imageLayoutVersion"`
	}
	if json.Unmarshal(entries["oci-layout"].data, &layout) != nil || layout.Version != "1.0.0" {
		return nil, errors.New("missing OCI layout")
	}
	var index struct {
		SchemaVersion int          `json:"schemaVersion"`
		MediaType     string       `json:"mediaType"`
		Manifests     []descriptor `json:"manifests"`
	}
	if json.Unmarshal(entries["index.json"].data, &index) != nil || index.SchemaVersion != 2 || (index.MediaType != "" && index.MediaType != "application/vnd.oci.image.index.v1+json") {
		return nil, errors.New("valid OCI index required")
	}
	return index.Manifests, nil
}

type ociManifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Config        descriptor   `json:"config"`
	Layers        []descriptor `json:"layers"`
}

type ociConfig struct {
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

func fetchOCIBlob(entries map[string]blob, d descriptor, load func(string) error) (blob, error) {
	if d.Size < 0 || len(d.URLs) != 0 {
		return blob{}, errors.New("local bounded OCI descriptor required")
	}
	hexDigest := strings.TrimPrefix(d.Digest, "sha256:")
	if !strings.HasPrefix(d.Digest, "sha256:") || !Digest(hexDigest) {
		return blob{}, errors.New("invalid OCI digest")
	}
	name := "blobs/sha256/" + hexDigest
	if load != nil {
		if err := load(name); err != nil {
			return blob{}, err
		}
	}
	b, ok := entries[name]
	if !ok || b.size != d.Size {
		return blob{}, errors.New("missing or wrong-size OCI blob")
	}
	return b, nil
}

func parseOCIManifest(data []byte) (ociManifest, error) {
	var manifest ociManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ociManifest{}, err
	}
	if manifest.SchemaVersion != 2 || (manifest.MediaType != "" && manifest.MediaType != "application/vnd.oci.image.manifest.v1+json") || manifest.Config.MediaType != "application/vnd.oci.image.config.v1+json" {
		return ociManifest{}, errors.New("invalid OCI image manifest")
	}
	return manifest, nil
}

func validateOCILayers(entries map[string]blob, layers []descriptor, load func(string) error) error {
	for _, layer := range layers {
		switch layer.MediaType {
		case "application/vnd.oci.image.layer.v1.tar", "application/vnd.oci.image.layer.v1.tar+gzip", "application/vnd.oci.image.layer.v1.tar+zstd":
		default:
			return errors.New("unsupported OCI layer media type")
		}
		if _, err := fetchOCIBlob(entries, layer, load); err != nil {
			return err
		}
	}
	return nil
}

func validateOCIRootFS(diffIDs []string, layers []descriptor) error {
	if len(diffIDs) != len(layers) {
		return errors.New("OCI rootfs/layer count mismatch")
	}
	for i, id := range diffIDs {
		hexDigest := strings.TrimPrefix(id, "sha256:")
		if !strings.HasPrefix(id, "sha256:") || !Digest(hexDigest) {
			return errors.New("invalid OCI diff ID")
		}
		if layers[i].MediaType == "application/vnd.oci.image.layer.v1.tar" && id != layers[i].Digest {
			return errors.New("uncompressed OCI layer identity mismatch")
		}
	}
	return nil
}

func validateOCIAttribution(labels map[string]string, wantRevision string) error {
	rev := labels["org.opencontainers.image.revision"]
	if wantRevision != "" && rev != wantRevision {
		return errors.New("OCI source revision mismatch")
	}
	if wantRevision == "" {
		return nil
	}
	baseDigest := strings.TrimPrefix(labels["org.opencontainers.image.base.digest"], "sha256:")
	if labels["org.opencontainers.image.source"] != "https://github.com/LevitateOS/sodaos" ||
		labels["org.opencontainers.image.base.name"] == "" ||
		!strings.HasPrefix(labels["org.opencontainers.image.base.digest"], "sha256:") ||
		!Digest(baseDigest) {
		return errors.New("soda image lacks source/base attribution")
	}
	return nil
}

func inspectOCIConfig(configBlob blob, layers []descriptor, want, revision string, imageDigest, configDigest string) (Image, error) {
	var cfg ociConfig
	if err := json.Unmarshal(configBlob.data, &cfg); err != nil {
		return Image{}, err
	}
	if cfg.RootFS.Type != "layers" {
		return Image{}, errors.New("OCI rootfs/layer count mismatch")
	}
	if err := validateOCIRootFS(cfg.RootFS.DiffIDs, layers); err != nil {
		return Image{}, err
	}
	// Compressed layer contents are not extracted here. Their blob identities
	// are checked; native import remains the proof of decompression/rootfs use.
	if cfg.OS != "linux" || cfg.Arch != want {
		return Image{}, fmt.Errorf("OCI must be linux/%s", want)
	}
	labels := cfg.Config.Labels
	if err := validateOCIAttribution(labels, revision); err != nil {
		return Image{}, err
	}
	rev := labels["org.opencontainers.image.revision"]
	return Image{
		Manifest:     imageDigest,
		Config:       configDigest,
		Architecture: cfg.Arch,
		Revision:     rev,
		Source:       labels["org.opencontainers.image.source"],
		BaseName:     labels["org.opencontainers.image.base.name"],
		BaseDigest:   labels["org.opencontainers.image.base.digest"],
	}, nil
}

func inspectOCIImage(entries map[string]blob, image descriptor, want, revision string, load func(string) error) (Image, error) {
	m, err := fetchOCIBlob(entries, image, load)
	if err != nil {
		return Image{}, err
	}
	manifest, err := parseOCIManifest(m.data)
	if err != nil {
		return Image{}, err
	}
	if err := validateOCILayers(entries, manifest.Layers, load); err != nil {
		return Image{}, err
	}
	configBlob, err := fetchOCIBlob(entries, manifest.Config, load)
	if err != nil {
		return Image{}, err
	}
	return inspectOCIConfig(configBlob, manifest.Layers, want, revision, image.Digest, manifest.Config.Digest)
}
