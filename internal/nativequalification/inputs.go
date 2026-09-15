// Package nativequalification provides artifact admission and fixed guest-state
// checks. It does not build images, orchestrate releases or provide an update client.
package nativequalification

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
)

type Artifact struct {
	Candidate rd.Candidate
	Payload   appliancerelease.Payload
	Files     map[string]string
}

func loadArtifactMetadata(root string) (Artifact, error) {
	var a Artifact
	if !filepath.IsAbs(root) {
		return a, errors.New("absolute artifact root required")
	}
	pb, err := os.ReadFile(filepath.Join(root, "payload.json"))
	if err != nil {
		return a, err
	}
	a.Payload, err = appliancerelease.Load(filepath.Join(root, "payload.json"))
	if err != nil {
		return a, err
	}
	if err = rd.ReadJSON(filepath.Join(root, "candidate.json"), &a.Candidate); err != nil {
		return a, err
	}
	if err = a.Candidate.Validate(a.Payload, pb); err != nil {
		return a, err
	}
	return a, nil
}

func hashArtifactSidecars(opened *os.Root, files map[string]string) error {
	for _, path := range []string{"payload.json", "candidate.json"} {
		sum, e := nativebuild.HashAt(opened, path)
		if e != nil {
			return e
		}
		files[path] = sum
	}
	return nil
}

func ReadArtifact(root string) (Artifact, error) {
	a, err := loadArtifactMetadata(root)
	if err != nil {
		return a, err
	}
	opened, err := os.OpenRoot(root)
	if err != nil {
		return a, err
	}
	defer opened.Close()
	a.Files, err = rd.VerifyCandidateImages(opened, root, a.Payload, a.Candidate)
	if err != nil {
		return a, err
	}
	if err = hashArtifactSidecars(opened, a.Files); err != nil {
		return a, err
	}
	return a, nil
}

func mediaBindingMatches(m hostimage.Media, a Artifact) bool {
	return m.Revision == a.Payload.Revision && m.Architecture == a.Payload.Architecture && m.HostManifest == a.Candidate.Host.Manifest && m.PayloadSHA256 == a.Candidate.PayloadSHA256
}

func verifyMediaFile(root string, file hostimage.MediaFile) error {
	if file.Path == "" || filepath.Base(file.Path) != file.Path {
		return errors.New("local media basename required")
	}
	path := filepath.Join(root, "media", file.Path)
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() != file.Bytes {
		return errors.New("media file/size differs")
	}
	sum, err := nativebuild.HashFile(path)
	if err != nil || sum != file.SHA256 {
		return errors.New("media digest differs")
	}
	return nil
}

func ReadMedia(root string, a Artifact) (hostimage.Media, error) {
	var m hostimage.Media
	if err := rd.ReadJSON(filepath.Join(root, "media/media.json"), &m); err != nil {
		return m, err
	}
	if !mediaBindingMatches(m, a) {
		return m, errors.New("media candidate binding differs")
	}
	if m.ISO.Bytes <= 0 || m.ISO.Bytes >= 2_000_000_000 {
		return m, errors.New("minimal installer size contract exceeded")
	}
	for _, file := range []hostimage.MediaFile{m.ISO, m.Rootfs} {
		if err := verifyMediaFile(root, file); err != nil {
			return m, err
		}
	}
	console, err := nativebuild.HashFile(filepath.Join(root, "tools/soda-installer"))
	if err != nil || console != m.ConsoleSHA256 {
		return m, errors.New("installer binding differs")
	}
	return m, nil
}

func Unchanged(root string, a Artifact) error {
	for path, want := range a.Files {
		got, err := nativebuild.HashFile(filepath.Join(root, path))
		if err != nil || got != want {
			return errors.New("qualified input changed")
		}
	}
	return nil
}

func SameBaseScenario(a, b Artifact) error {
	if a.Payload.Architecture != "x86_64" || b.Payload.Architecture != a.Payload.Architecture || a.Payload.Base != b.Payload.Base || a.Payload.Schema != b.Payload.Schema || a.Payload.RepositoryPrefix != b.Payload.RepositoryPrefix || a.Candidate.Host.Manifest == b.Candidate.Host.Manifest {
		return errors.New("distinct same-base x86_64 candidates with matching schema/repository required")
	}
	return nil
}
