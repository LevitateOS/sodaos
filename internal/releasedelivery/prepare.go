package releasedelivery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

func nowUTC() time.Time { return time.Now().UTC() }

type Qualification struct {
	Serial              uint64
	Class, Scope, Notes string
	Evidence            map[string]string
}

// Prepare reuses the M1 payload and native OCI verifier. Qualification assertions
// do not authorize signing: the protected worker must independently admit this
// exact resulting document digest based on its retained test evidence.
func matchingReleaseProvenance(release Release, p appliancerelease.Payload) bool {
	return release.Provenance["packages.txt"] == "sha256:"+p.HostPackagesSHA256 && release.Provenance["presentation.json"] == "sha256:"+p.PresentationSHA256
}

func hashReleaseProvenance(root *os.Root, release *Release) error {
	paths := map[string]string{"source.tar": "source.tar", "app-inputs.json": "app-inputs.json", "packages.txt": "packages.txt", "presentation.json": "forgejo-context/presentation.json"}
	for n, path := range paths {
		h, e := nativebuild.HashAt(root, path)
		if e != nil {
			return e
		}
		release.Provenance[n] = "sha256:" + h
	}
	return nil
}

func readMediaBinding(path string) ([]byte, error) {
	b, err := ReadFile(path, 1<<20)
	if err != nil {
		return nil, err
	}
	var m MediaBinding
	if json.Unmarshal(b, &m) != nil || !validMediaFile(m.ISO) || !validMediaFile(m.Rootfs) {
		return nil, ErrRefused
	}
	return b, nil
}

func buildReleaseDocument(t Trust, candidate string, media []byte, q Qualification) (Release, error) {
	root, e := os.OpenRoot(candidate)
	if e != nil {
		return Release{}, e
	}
	defer root.Close()
	pb, e := readAt(root, "payload.json", 1<<20)
	if e != nil {
		return Release{}, e
	}
	cb, e := readAt(root, "candidate.json", 1<<20)
	if e != nil {
		return Release{}, e
	}
	release := Release{Format: 1, Serial: q.Serial, Class: q.Class, Payload: pb, Candidate: cb, Media: media, Qualification: q.Scope, Evidence: q.Evidence, Notes: q.Notes, Provenance: map[string]string{}}
	if e = hashReleaseProvenance(root, &release); e != nil {
		return Release{}, e
	}
	p, c, e := release.Validate(t)
	if e != nil {
		return Release{}, e
	}
	if !matchingReleaseProvenance(release, p) {
		return Release{}, ErrRefused
	}
	if _, e = VerifyCandidateImages(root, candidate, p, c); e != nil {
		return Release{}, e
	}
	return release, nil
}

func Prepare(t Trust, candidate, media string, q Qualification, out string) (string, error) {
	if e := t.Validate(); e != nil {
		return "", e
	}
	mb, e := readMediaBinding(media)
	if e != nil {
		return "", e
	}
	release, e := buildReleaseDocument(t, candidate, mb, q)
	if e != nil {
		return "", e
	}
	return WriteDocument(out, release)
}

// VerifyCandidateImages is the shared native archive/identity check used by
// qualification admission and release preparation. The caller validates payload
// and candidate metadata first; observed archive hashes can bind later evidence.
func verifyCandidateImage(root *os.Root, candidate, n, arch string, expected appliancerelease.Image, rev string) (string, string, error) {
	path := n + ".oci"
	if n != "host" {
		path = "images/" + path
	}
	hash, e := nativebuild.HashAt(root, path)
	if e != nil || hash != expected.ArchiveSHA256 {
		return "", "", errorAt(n + " archive")
	}
	im, e := nativebuild.InspectOCI(filepath.Join(candidate, path), arch, rev)
	if e != nil || im.Manifest != expected.Manifest || im.Config != expected.Config {
		return "", "", errorAt(n + " identity")
	}
	return path, hash, nil
}

func VerifyCandidateImages(root *os.Root, candidate string, p appliancerelease.Payload, c Candidate) (map[string]string, error) {
	files := map[string]string{}
	inputs := map[string]appliancerelease.Image{"host": {Config: c.Host.Config, Manifest: c.Host.Manifest, ArchiveSHA256: c.HostArchiveSHA256}}
	for n, im := range p.Images {
		inputs[n] = im
	}
	for _, n := range append([]string{"host"}, appliancerelease.Names...) {
		rev := p.Revision
		if n == "proxy" {
			rev = ""
		}
		path, hash, e := verifyCandidateImage(root, candidate, n, p.Architecture, inputs[n], rev)
		if e != nil {
			return nil, e
		}
		files[path] = hash
	}
	return files, nil
}

func ReferenceForDocument(t Trust, kind, digest string) (string, error) {
	if !Digest(digest) {
		return "", ErrRefused
	}
	repo := t.Prefix + "-release"
	if channel(kind) {
		repo = t.Prefix + "-channel-" + kind
	} else if kind != "release" {
		return "", ErrRefused
	}
	return repo + "@" + digest, nil
}

func immutableTag(ref string) string {
	repo, d, _ := strings.Cut(ref, "@")
	return repo + ":sha256-" + strings.TrimPrefix(d, "sha256:")
}
