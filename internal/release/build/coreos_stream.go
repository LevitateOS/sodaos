package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// Live CoreOS stable inputs. Pre-release builds resolve the current stable
// build at build time and record it; no stored version is consulted.
// Reproducibility returns after release as qualification-time snapshots.

const (
	defaultCoreOSStreamURL = "https://builds.coreos.fedoraproject.org/streams/stable.json"
	defaultCoreOSRegistry  = "https://quay.io"
	coreOSContainerRepo    = "fedora/fedora-coreos"
	coreOSContainerTag     = "stable"
)

func coreOSStreamURL() string {
	if u := strings.TrimSpace(os.Getenv("SODA_COREOS_STREAM_URL")); u != "" {
		return u
	}
	return defaultCoreOSStreamURL
}

func coreOSRegistry() string {
	if u := strings.TrimSpace(os.Getenv("SODA_COREOS_REGISTRY")); u != "" {
		return strings.TrimSuffix(u, "/")
	}
	return defaultCoreOSRegistry
}

// ResolvedCoreOS is one stable build as found right now: release and
// live-ISO locations from the stream, container digests from the registry.
type ResolvedCoreOS struct {
	Release     string
	MetadataURL string
	Container   map[string]string      // soda arch -> repo@digest
	ISO         map[string]CoreOSImage // arch -> live ISO triple
	QEMU        map[string]CoreOSImage // arch -> qemu triple
}

type streamDisk struct {
	Location           string `json:"location"`
	SHA256             string `json:"sha256"`
	Signature          string `json:"signature"`
	UncompressedSHA256 string `json:"uncompressed-sha256"`
}

// streamHTTPTransport defaults to the standard transport; tests point it at
// a local TLS fixture server. Production behavior is unchanged.
var streamHTTPTransport http.RoundTripper = http.DefaultTransport

func cappedClient() *http.Client {
	return &http.Client{Transport: streamHTTPTransport, Timeout: time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || !httpsURL(req.URL.String()) {
			return errors.New("unsafe metadata redirect")
		}
		return nil
	}}
}

func fetchCappedJSON(ctx context.Context, url string, maxBytes int64) ([]byte, error) {
	if !httpsURL(url) || maxBytes <= 0 {
		return nil, errors.New("bounded HTTPS fetch required")
	}
	client := cappedClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CoreOS metadata fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoreOS metadata HTTP failure: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("CoreOS metadata exceeds size limit")
	}
	return data, nil
}

var streamReleasePattern = regexp.MustCompile(`/builds/([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)/`)

type streamFormat struct {
	Disk streamDisk `json:"disk"`
}

type streamArch struct {
	Artifacts map[string]struct {
		Formats map[string]streamFormat `json:"formats"`
	} `json:"artifacts"`
}

// resolveStreamBuild parses one stable-stream document into release and
// per-arch ISO/QEMU triples. The stream carries no release field on the
// entries themselves, so the release parses out of the ISO location path
// and must agree across architectures.
func resolveStreamBuild(data []byte) (release string, iso, qemu map[string]CoreOSImage, err error) {
	var doc struct {
		Architectures map[string]streamArch `json:"architectures"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", nil, nil, err
	}
	iso, qemu = map[string]CoreOSImage{}, map[string]CoreOSImage{}
	for _, arch := range []string{"x86_64", "aarch64"} {
		entry, ok := doc.Architectures[arch]
		if !ok {
			return "", nil, nil, fmt.Errorf("stable stream lacks architecture %s", arch)
		}
		isoFormat, ok := entry.Artifacts["metal"].Formats["iso"]
		if !ok {
			return "", nil, nil, fmt.Errorf("stable stream lacks %s live ISO", arch)
		}
		m := streamReleasePattern.FindStringSubmatch(isoFormat.Disk.Location)
		if len(m) != 2 {
			return "", nil, nil, fmt.Errorf("stable stream %s ISO location names no release", arch)
		}
		if release == "" {
			release = m[1]
		} else if m[1] != release {
			return "", nil, nil, fmt.Errorf("stable stream architectures disagree on release (%s vs %s)", release, m[1])
		}
		qemuFormat, ok := entry.Artifacts["qemu"].Formats["qcow2.xz"]
		if !ok {
			return "", nil, nil, fmt.Errorf("stable stream lacks %s qemu image", arch)
		}
		iso[arch] = CoreOSImage{URL: isoFormat.Disk.Location, SignatureURL: isoFormat.Disk.Signature, SHA256: isoFormat.Disk.SHA256}
		qemu[arch] = CoreOSImage{URL: qemuFormat.Disk.Location, SignatureURL: qemuFormat.Disk.Signature, SHA256: qemuFormat.Disk.SHA256, UncompressedSHA256: qemuFormat.Disk.UncompressedSHA256}
	}
	if err := validStreamImages(release, iso, qemu); err != nil {
		return "", nil, nil, err
	}
	return release, iso, qemu, nil
}

// validStreamImages applies the lock-era shape rules to stream-resolved
// triples: the ISO rules mirror validCoreOSISO and the QEMU rules mirror
// validCoreOSLock, minus the file they used to come from.
func validStreamImages(release string, iso, qemu map[string]CoreOSImage) error {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(release) {
		return errors.New("stable stream release is malformed")
	}
	for _, arch := range []string{"x86_64", "aarch64"} {
		img, ok := iso[arch]
		if !ok || !httpsURL(img.URL) || !strings.HasSuffix(img.URL, ".iso") || img.SignatureURL != img.URL+".sig" || !Digest(img.SHA256) {
			return fmt.Errorf("stable stream %s live ISO is malformed", arch)
		}
		q, ok := qemu[arch]
		if !ok || !httpsURL(q.URL) || !httpsURL(q.SignatureURL) || !Digest(q.SHA256) || !Digest(q.UncompressedSHA256) {
			return fmt.Errorf("stable stream %s qemu image is malformed", arch)
		}
	}
	return nil
}

// resolveRegistryDigests reads the container manifest index for the stable
// tag and returns per-soda-arch manifest digests as scheme-free repo@digest
// refs, the same shape the build already pulls and builds FROM. Pulling and
// building use these digest-pinned refs, so the tag can roll without
// changing the build.
func resolveRegistryDigests(ctx context.Context, registry string) (map[string]string, error) {
	if !httpsURL(registry) {
		return nil, errors.New("container registry URL must be HTTPS")
	}
	parsed, err := url.Parse(registry)
	if err != nil || parsed.Host == "" || strings.Contains(parsed.Host, "/") {
		return nil, errors.New("container registry URL is malformed")
	}
	endpoint := registry + "/v2/" + coreOSContainerRepo + "/manifests/" + coreOSContainerTag
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json")
	client := cappedClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("container registry fetch failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("container registry refused anonymous manifest access")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("container registry HTTP failure: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, errors.New("container index exceeds size limit")
	}
	var index struct {
		Manifests []struct {
			Digest   string `json:"digest"`
			Platform struct {
				Architecture string `json:"architecture"`
			} `json:"platform"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	digests := map[string]string{}
	for _, arch := range []string{"x86_64", "aarch64"} {
		ociArch, err := OCIArchitecture(arch)
		if err != nil {
			return nil, err
		}
		found := ""
		for _, m := range index.Manifests {
			if m.Platform.Architecture != ociArch {
				continue
			}
			d, ok := strings.CutPrefix(m.Digest, "sha256:")
			if !ok || !Digest(d) {
				return nil, fmt.Errorf("container index %s digest is malformed", arch)
			}
			found = m.Digest
		}
		if found == "" {
			return nil, fmt.Errorf("container index lacks architecture %s", arch)
		}
		digests[arch] = parsed.Host + "/" + coreOSContainerRepo + "@" + found
	}
	return digests, nil
}

// ResolveCoreOSISO returns the current stable live ISO for one
// architecture: release plus the verified download triple.
func ResolveCoreOSISO(ctx context.Context, arch string) (string, CoreOSImage, error) {
	if _, err := OCIArchitecture(arch); err != nil {
		return "", CoreOSImage{}, err
	}
	resolved, err := ResolveCoreOS(ctx)
	if err != nil {
		return "", CoreOSImage{}, err
	}
	img, ok := resolved.ISO[arch]
	if !ok {
		return "", CoreOSImage{}, fmt.Errorf("stable stream lacks %s live ISO", arch)
	}
	return resolved.Release, img, nil
}

// ResolveCoreOSQEMU returns the current stable qemu image for one
// architecture: release plus the verified download triple.
func ResolveCoreOSQEMU(ctx context.Context, arch string) (string, CoreOSImage, error) {
	if _, err := OCIArchitecture(arch); err != nil {
		return "", CoreOSImage{}, err
	}
	resolved, err := ResolveCoreOS(ctx)
	if err != nil {
		return "", CoreOSImage{}, err
	}
	img, ok := resolved.QEMU[arch]
	if !ok {
		return "", CoreOSImage{}, fmt.Errorf("stable stream lacks %s qemu image", arch)
	}
	return resolved.Release, img, nil
}

// ResolveCoreOS finds the current stable build: release, live-ISO, and
// qemu locations from the stream, container digests from the registry.
// Both endpoints default to production and override by environment for
// fixture-based tests.
func ResolveCoreOS(ctx context.Context) (ResolvedCoreOS, error) {
	var resolved ResolvedCoreOS
	streamURL := coreOSStreamURL()
	if !httpsURL(streamURL) {
		return resolved, errors.New("CoreOS stream URL must be HTTPS")
	}
	data, err := fetchCappedJSON(ctx, streamURL, 8<<20)
	if err != nil {
		return resolved, err
	}
	release, iso, qemu, err := resolveStreamBuild(data)
	if err != nil {
		return resolved, err
	}
	digests, err := resolveRegistryDigests(ctx, coreOSRegistry())
	if err != nil {
		return resolved, err
	}
	meta, err := streamReleaseURL(streamURL, release)
	if err != nil {
		return resolved, err
	}
	return ResolvedCoreOS{Release: release, MetadataURL: meta, Container: digests, ISO: iso, QEMU: qemu}, nil
}

// WriteResolvedCoreOS records one resolved stable build for an isolated
// consumer that cannot fetch it: the controller resolves where network is
// admitted and the worker consumes its own attempt's file. Public metadata
// only (release, locations, digests); the file carries no authority beyond
// the digest-pinned pulls that verify content downstream.
func WriteResolvedCoreOS(path string, resolved ResolvedCoreOS) error {
	if err := ValidResolvedCoreOS(resolved); err != nil {
		return err
	}
	data, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		return err
	}
	return WriteNew(path, append(data, '\n'), 0o644)
}

// ReadResolvedCoreOS admits controller-resolved inputs for the isolated
// worker. Validation mirrors the live path; a missing or tampered file
// fails here.
func ReadResolvedCoreOS(path string) (ResolvedCoreOS, error) {
	var resolved ResolvedCoreOS
	if err := ReadJSON(path, &resolved); err != nil {
		return resolved, err
	}
	if err := ValidResolvedCoreOS(resolved); err != nil {
		return resolved, err
	}
	return resolved, nil
}

// ValidResolvedCoreOS applies the live resolution shape rules to admitted
// inputs: release form and cross-arch agreement, verified ISO/QEMU triples,
// release metadata URL, and digest-pinned container refs for both arches.
// The registry host itself is not allowlisted: digest-pinned pulls verify
// content downstream, so a retargeted host cannot substitute bytes.
func ValidResolvedCoreOS(resolved ResolvedCoreOS) error {
	if err := validStreamImages(resolved.Release, resolved.ISO, resolved.QEMU); err != nil {
		return err
	}
	if !httpsURL(resolved.MetadataURL) || !strings.HasSuffix(resolved.MetadataURL, "/builds/"+resolved.Release+"/release.json") {
		return errors.New("resolved CoreOS metadata URL is malformed")
	}
	if len(resolved.Container) != 2 {
		return errors.New("both architecture base digests required")
	}
	for _, arch := range []string{"x86_64", "aarch64"} {
		host, digest, ok := strings.Cut(resolved.Container[arch], "/fedora/fedora-coreos@sha256:")
		if !ok || host == "" || strings.Contains(host, "/") || !Digest(digest) {
			return errors.New("digest-pinned CoreOS base required")
		}
	}
	return nil
}

// streamReleaseURL derives the build's release.json URL from the stream
// endpoint and release: <stream-prefix>/builds/<release>/release.json.
func streamReleaseURL(streamURL, release string) (string, error) {
	idx := strings.Index(streamURL, "/streams/")
	if idx < 0 {
		return "", errors.New("CoreOS stream URL names no stream")
	}
	meta := streamURL[:idx] + "/prod/streams/stable/builds/" + release + "/release.json"
	if !httpsURL(meta) {
		return "", errors.New("CoreOS release metadata URL is malformed")
	}
	return meta, nil
}
