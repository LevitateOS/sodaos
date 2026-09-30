// Identity checks adapted from soda-os bc1d3e0 release/inspection.go.
// Stream real OCI archives without extracting layers or importing release code.
package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"path"
	"slices"
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

type layerMember struct {
	hash    string
	present bool
	blocked bool
}

func openOCIArchive(file, arch, revision string) (string, *os.File, error) {
	want, err := OCIArchitecture(arch)
	if err != nil {
		return "", nil, err
	}
	if revision != "" && !Revision(revision) {
		return "", nil, errors.New("full source revision required")
	}
	st, err := os.Lstat(file)
	if err != nil {
		return "", nil, err
	}
	if !st.Mode().IsRegular() {
		return "", nil, errors.New("OCI archive must be regular")
	}
	f, err := os.Open(file)
	if err != nil {
		return "", nil, err
	}
	return want, f, nil
}

func checkTarEntryName(n string, typeflag byte) (bool, error) {
	if n == "." && typeflag == tar.TypeDir {
		return true, nil
	}
	if path.IsAbs(n) || n == ".." || strings.HasPrefix(n, "../") {
		return false, errors.New("unsafe OCI path")
	}
	return false, nil
}

func checkTarDirectory(n string, typeflag byte) (bool, error) {
	if typeflag != tar.TypeDir {
		return false, nil
	}
	if n != "blobs" && n != "blobs/sha256" {
		return false, errors.New("unexpected OCI directory")
	}
	return true, nil
}

func isValidOCIRegularEntry(n string) bool {
	if n == "index.json" || n == "oci-layout" {
		return true
	}
	return strings.HasPrefix(n, "blobs/sha256/") && Digest(strings.TrimPrefix(n, "blobs/sha256/"))
}

func recordTarEntry(seen map[string]bool, n string) error {
	if seen[n] {
		return errors.New("duplicate OCI entry")
	}
	seen[n] = true
	if len(seen) > 100000 {
		return errors.New("too many OCI entries")
	}
	return nil
}

func readArchiveBlobEntry(entries map[string]blob, n string, h *tar.Header, tr io.Reader, jsonBytes *int) error {
	if h.Typeflag != tar.TypeReg {
		return errors.New("non-regular OCI entry")
	}
	if _, ok := entries[n]; ok {
		return errors.New("duplicate OCI entry")
	}
	if len(entries) > 100000 {
		return errors.New("too many OCI entries")
	}
	if !isValidOCIRegularEntry(n) {
		return errors.New("not an OCI archive")
	}
	return readOCIBlob(entries, n, h.Size, tr, jsonBytes)
}

func processArchiveTarHeader(h *tar.Header, tr io.Reader, entries map[string]blob, seen map[string]bool, jsonBytes *int) error {
	n := path.Clean(h.Name)
	skip, err := checkTarEntryName(n, h.Typeflag)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}
	if err := recordTarEntry(seen, n); err != nil {
		return err
	}
	isDir, err := checkTarDirectory(n, h.Typeflag)
	if err != nil {
		return err
	}
	if isDir {
		return nil
	}
	return readArchiveBlobEntry(entries, n, h, tr, jsonBytes)
}

func readOCIArchiveEntries(f io.Reader) (map[string]blob, error) {
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
			return nil, err
		}
		if err := processArchiveTarHeader(h, tr, entries, seen, &jsonBytes); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func inspectArchiveIndex(entries map[string]blob, want, revision string) (Image, error) {
	index, err := readOCIIndex(entries)
	if err != nil {
		return Image{}, err
	}
	if len(index) != 1 || index[0].MediaType != "application/vnd.oci.image.manifest.v1+json" {
		return Image{}, errors.New("single-platform OCI index required")
	}
	return inspectOCIImage(entries, index[0], want, revision, nil)
}

func InspectOCI(file, arch, revision string) (Image, error) {
	want, f, err := openOCIArchive(file, arch, revision)
	if err != nil {
		return Image{}, err
	}
	defer f.Close()

	entries, err := readOCIArchiveEntries(f)
	if err != nil {
		return Image{}, err
	}
	return inspectArchiveIndex(entries, want, revision)
}

func requestedOCIPaths(paths []string) (map[string]string, error) {
	wanted := make(map[string]string, len(paths))
	for _, requested := range paths {
		name := strings.TrimPrefix(requested, "/")
		if requested == name || name == "" || path.Clean(name) != name || strings.ContainsAny(name, "\\\n\r\x00") {
			return nil, errors.New("absolute clean OCI member paths required")
		}
		if _, duplicate := wanted[name]; duplicate {
			return nil, errors.New("duplicate OCI member request")
		}
		wanted[name] = requested
	}
	if len(wanted) == 0 {
		return nil, errors.New("explicit OCI members required")
	}
	return wanted, nil
}

func cleanLayerName(name string) (string, error) {
	name = strings.TrimPrefix(name, "./")
	clean := path.Clean(name)
	segments := strings.Split(name, "/")
	if clean == "." || path.IsAbs(clean) || slices.Contains(segments, "..") || strings.ContainsAny(clean, "\\\n\r\x00") {
		return "", errors.New("unsafe OCI layer path")
	}
	return clean, nil
}

func whiteoutTarget(name string) (string, bool) {
	base := path.Base(name)
	if !strings.HasPrefix(base, ".wh.") || base == ".wh..wh..opq" {
		return "", false
	}
	return path.Join(path.Dir(name), strings.TrimPrefix(base, ".wh.")), true
}

func recordLayerDeletion(found map[string]layerMember, wanted map[string]string, removed string) error {
	for name := range wanted {
		if name != removed && !strings.HasPrefix(name, removed+"/") {
			continue
		}
		if existing, ok := found[name]; ok && (existing.present || existing.blocked) {
			continue
		}
		found[name] = layerMember{}
	}
	return nil
}

func recordAncestorReplacement(found map[string]layerMember, wanted map[string]string, name string, typeflag byte) {
	if typeflag == tar.TypeDir {
		return
	}
	for requested := range wanted {
		if strings.HasPrefix(requested, name+"/") {
			found[requested] = layerMember{blocked: true}
		}
	}
}

func recordOpaqueDirectory(found map[string]layerMember, wanted map[string]string, dir string) {
	for requested := range wanted {
		if dir != "." && !strings.HasPrefix(requested, dir+"/") {
			continue
		}
		if existing, ok := found[requested]; ok && (existing.present || existing.blocked) {
			continue
		}
		found[requested] = layerMember{}
	}
}

func recordLayerEntry(tr io.Reader, h *tar.Header, name string, found map[string]layerMember, wanted map[string]string) error {
	if removed, whiteout := whiteoutTarget(name); whiteout {
		return recordLayerDeletion(found, wanted, removed)
	}
	if path.Base(name) == ".wh..wh..opq" {
		recordOpaqueDirectory(found, wanted, path.Dir(name))
		return nil
	}
	recordAncestorReplacement(found, wanted, name, h.Typeflag)
	if _, ok := wanted[name]; !ok || found[name].blocked {
		return nil
	}
	if h.Typeflag != tar.TypeReg {
		return errors.New("requested OCI member is non-regular")
	}
	hash := sha256.New()
	if copied, copyErr := io.Copy(hash, tr); copyErr != nil || copied != h.Size {
		return errors.New("OCI layer member size changed")
	}
	found[name] = layerMember{hash: hex.EncodeToString(hash.Sum(nil)), present: true}
	return nil
}

func scanLayerHeader(tr io.Reader, h *tar.Header, found map[string]layerMember, seen map[string]bool, wanted map[string]string) error {
	name, err := cleanLayerName(h.Name)
	if err != nil {
		return err
	}
	if seen[name] {
		return errors.New("duplicate OCI layer entry")
	}
	seen[name] = true
	if len(seen) > 1000000 {
		return errors.New("too many OCI layer entries")
	}
	return recordLayerEntry(tr, h, name, found, wanted)
}

func scanOCILayer(r io.Reader, wanted map[string]string) (map[string]layerMember, error) {
	found := map[string]layerMember{}
	seen := map[string]bool{}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return found, nil
		}
		if err != nil {
			return nil, err
		}
		if err := scanLayerHeader(tr, h, found, seen, wanted); err != nil {
			return nil, err
		}
	}
}

func layerArchiveIndexes(layers []descriptor) map[string][]int {
	indexes := map[string][]int{}
	for i, layer := range layers {
		name := "blobs/sha256/" + strings.TrimPrefix(layer.Digest, "sha256:")
		indexes[name] = append(indexes[name], i)
	}
	return indexes
}

func verifyLayerDigest(raw io.Reader, checksum hash.Hash, digest string) error {
	if _, err := io.Copy(io.Discard, raw); err != nil {
		return errors.New("OCI layer changed during member verification")
	}
	if "sha256:"+hex.EncodeToString(checksum.Sum(nil)) != digest {
		return errors.New("OCI layer changed during member verification")
	}
	return nil
}

func openLayerReader(raw io.Reader, mediaType string) (io.Reader, func() error, error) {
	switch mediaType {
	case "application/vnd.oci.image.layer.v1.tar":
		return raw, nil, nil
	case "application/vnd.oci.image.layer.v1.tar+gzip":
		gz, err := gzip.NewReader(raw)
		if err != nil {
			return nil, nil, err
		}
		return gz, gz.Close, nil
	default:
		return nil, nil, errors.New("unsupported OCI layer media type")
	}
}

func scanLayerReader(layer io.Reader, closeLayer func() error, wanted map[string]string) (map[string]layerMember, error) {
	members, scanErr := scanOCILayer(layer, wanted)
	if closeLayer != nil {
		if closeErr := closeLayer(); scanErr == nil {
			scanErr = closeErr
		}
	}
	return members, scanErr
}

func scanArchiveLayer(tr io.Reader, descriptor descriptor, wanted map[string]string) (map[string]layerMember, bool, error) {
	rawHash := sha256.New()
	raw := io.TeeReader(tr, rawHash)
	if descriptor.MediaType == "application/vnd.oci.image.layer.v1.tar+zstd" {
		return nil, true, verifyLayerDigest(raw, rawHash, descriptor.Digest)
	}
	layer, closeLayer, err := openLayerReader(raw, descriptor.MediaType)
	if err != nil {
		return nil, false, err
	}
	members, scanErr := scanLayerReader(layer, closeLayer, wanted)
	if _, copyErr := io.Copy(io.Discard, raw); scanErr == nil && copyErr != nil {
		scanErr = copyErr
	}
	if scanErr == nil && "sha256:"+hex.EncodeToString(rawHash.Sum(nil)) != descriptor.Digest {
		scanErr = errors.New("OCI layer changed during member verification")
	}
	return members, false, scanErr
}

func scanOCIArchiveLayers(f io.Reader, layers []descriptor, wanted map[string]string) ([]map[string]layerMember, []bool, error) {
	indexes := layerArchiveIndexes(layers)
	found := make([]map[string]layerMember, len(layers))
	unsupported := make([]bool, len(layers))
	tr := tar.NewReader(f)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return found, unsupported, nil
		}
		if err != nil {
			return nil, nil, err
		}
		positions := indexes[path.Clean(h.Name)]
		if len(positions) == 0 {
			continue
		}
		members, blocked, err := scanArchiveLayer(tr, layers[positions[0]], wanted)
		if err != nil {
			return nil, nil, err
		}
		for _, position := range positions {
			found[position], unsupported[position] = members, blocked
		}
	}
}

func resolveOCIMembers(layers []map[string]layerMember, unsupported []bool, wanted map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(wanted))
	for name, requested := range wanted {
		for i := len(layers) - 1; i >= 0; i-- {
			if unsupported[i] {
				return nil, errors.New("zstd OCI layer blocks member verification")
			}
			member, ok := layers[i][name]
			if !ok {
				continue
			}
			if member.blocked {
				return nil, errors.New("requested OCI member has a non-directory ancestor")
			}
			if !member.present {
				return nil, errors.New("requested OCI member was removed")
			}
			resolved[requested] = member.hash
			break
		}
		if resolved[requested] == "" {
			return nil, errors.New("requested OCI member missing")
		}
	}
	return resolved, nil
}

func inspectContentManifest(entries map[string]blob, want, revision string) (Image, ociManifest, error) {
	index, err := readOCIIndex(entries)
	if err != nil || len(index) != 1 || index[0].MediaType != "application/vnd.oci.image.manifest.v1+json" {
		return Image{}, ociManifest{}, errors.New("single-platform OCI index required")
	}
	image, err := inspectOCIImage(entries, index[0], want, revision, nil)
	if err != nil {
		return Image{}, ociManifest{}, err
	}
	manifestBlob, err := fetchOCIBlob(entries, index[0], nil)
	if err != nil {
		return Image{}, ociManifest{}, err
	}
	manifest, err := parseOCIManifest(manifestBlob.data)
	return image, manifest, err
}

// InspectOCIContent extends the ordinary identity check with hashes of exact
// regular files as they appear in the verified image rootfs overlay.
func InspectOCIContent(file, arch, revision string, paths []string) (Image, map[string]string, error) {
	wanted, err := requestedOCIPaths(paths)
	if err != nil {
		return Image{}, nil, err
	}
	want, f, err := openOCIArchive(file, arch, revision)
	if err != nil {
		return Image{}, nil, err
	}
	defer f.Close()
	entries, err := readOCIArchiveEntries(f)
	if err != nil {
		return Image{}, nil, err
	}
	image, manifest, err := inspectContentManifest(entries, want, revision)
	if err != nil {
		return Image{}, nil, err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return Image{}, nil, err
	}
	layers, unsupported, err := scanOCIArchiveLayers(f, manifest.Layers, wanted)
	if err != nil {
		return Image{}, nil, err
	}
	content, err := resolveOCIMembers(layers, unsupported, wanted)
	return image, content, err
}

func copyOCIBlob(name string, length int64, r io.Reader) (sum string, size int64, body []byte, err error) {
	hash := sha256.New()
	var data bytes.Buffer
	var w io.Writer = hash
	if length <= 4<<20 {
		w = io.MultiWriter(hash, &data)
	}
	size, err = io.Copy(w, io.LimitReader(r, length+1))
	if err != nil {
		return "", 0, nil, err
	}
	if size != length {
		return "", 0, nil, errors.New("OCI blob size changed")
	}
	sum = hex.EncodeToString(hash.Sum(nil))
	if strings.HasPrefix(name, "blobs/") && name != "blobs/sha256/"+sum {
		return "", 0, nil, errors.New("OCI blob checksum mismatch")
	}
	body = data.Bytes()
	if !json.Valid(body) {
		body = nil
	}
	return sum, size, body, nil
}

func readOCIBlob(entries map[string]blob, name string, length int64, r io.Reader, jsonBytes *int) error {
	if length < 0 || length == math.MaxInt64 {
		return errors.New("invalid OCI blob size")
	}
	sum, size, body, err := copyOCIBlob(name, length, r)
	if err != nil {
		return err
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
