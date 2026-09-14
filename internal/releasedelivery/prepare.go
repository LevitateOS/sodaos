package releasedelivery

import (
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
func Prepare(t Trust, candidate string, q Qualification, out string) (string, error) {
	if e := t.Validate(); e != nil {
		return "", e
	}
	root, e := os.OpenRoot(candidate)
	if e != nil {
		return "", e
	}
	defer root.Close()
	pb, e := readAt(root, "payload.json", 1<<20)
	if e != nil {
		return "", e
	}
	cb, e := readAt(root, "candidate.json", 1<<20)
	if e != nil {
		return "", e
	}
	release := Release{Format: 1, Serial: q.Serial, Class: q.Class, Payload: pb, Candidate: cb, Qualification: q.Scope, Evidence: q.Evidence, Notes: q.Notes, Provenance: map[string]string{}}
	paths := map[string]string{"source.tar": "source.tar", "app-inputs.json": "app-inputs.json", "packages.txt": "packages.txt", "presentation.json": "forgejo-context/presentation.json"}
	for n, path := range paths {
		h, e := nativebuild.HashAt(root, path)
		if e != nil {
			return "", e
		}
		release.Provenance[n] = "sha256:" + h
	}
	p, c, e := release.Validate(t)
	if e != nil {
		return "", e
	}
	if release.Provenance["packages.txt"] != "sha256:"+p.HostPackagesSHA256 || release.Provenance["presentation.json"] != "sha256:"+p.PresentationSHA256 {
		return "", ErrRefused
	}
	if _, e = VerifyCandidateImages(root, candidate, p, c); e != nil {
		return "", e
	}
	return WriteDocument(out, release)
}

// VerifyCandidateImages is the shared native archive/identity check used by
// qualification admission and release preparation. The caller validates payload
// and candidate metadata first; observed archive hashes can bind later evidence.
func VerifyCandidateImages(root *os.Root, candidate string, p appliancerelease.Payload, c Candidate) (map[string]string, error) {
	files := map[string]string{}
	inputs := map[string]appliancerelease.Image{"host": {Config: c.Host.Config, Manifest: c.Host.Manifest, ArchiveSHA256: c.HostArchiveSHA256}}
	for n, im := range p.Images {
		inputs[n] = im
	}
	for _, n := range append([]string{"host"}, appliancerelease.Names...) {
		path := n + ".oci"
		if n != "host" {
			path = "images/" + path
		}
		hash, e := nativebuild.HashAt(root, path)
		if e != nil || hash != inputs[n].ArchiveSHA256 {
			return nil, errorAt(n + " archive")
		}
		rev := p.Revision
		if n == "proxy" {
			rev = ""
		}
		im, e := nativebuild.InspectOCI(filepath.Join(candidate, path), p.Architecture, rev)
		if e != nil || im.Manifest != inputs[n].Manifest || im.Config != inputs[n].Config {
			return nil, errorAt(n + " identity")
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
