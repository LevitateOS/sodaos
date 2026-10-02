package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
)

type museArtifact struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type museRelease struct {
	Version   string                  `json:"version"`
	Artifacts map[string]museArtifact `json:"artifacts"`
}

// FetchMuse stages upstream bytes only after checking the selected release pin.
func FetchMuse(ctx context.Context, manifest, arch, dest string) error {
	if !filepath.IsAbs(dest) {
		return errors.New("absolute Muse destination required")
	}
	release, artifact, err := loadMuseRelease(manifest, arch)
	if err != nil {
		return err
	}
	if digest, err := HashFile(dest); err == nil && digest == artifact.SHA256 {
		return os.Chmod(dest, 0o755)
	}
	return fetchMuseArtifact(ctx, release.Version, artifact, dest)
}

func loadMuseRelease(manifest, arch string) (museRelease, museArtifact, error) {
	var release museRelease
	if _, err := OCIArchitecture(arch); err != nil {
		return release, museArtifact{}, err
	}
	f, err := os.Open(manifest)
	if err != nil {
		return release, museArtifact{}, err
	}
	decoder := json.NewDecoder(io.LimitReader(f, 8192))
	decoder.DisallowUnknownFields()
	err = errors.Join(decoder.Decode(&release), f.Close())
	if err != nil {
		return release, museArtifact{}, err
	}
	artifact, ok := release.Artifacts[arch]
	if !ok || !validMuseArtifact(release.Version, arch, artifact) {
		return release, artifact, errors.New("invalid Muse release pin")
	}
	return release, artifact, nil
}

func validMuseArtifact(version, arch string, artifact museArtifact) bool {
	expected := map[string]string{"x86_64": "muse-x86-linux"}[arch]
	return artifact.File == expected && Digest(artifact.SHA256) && artifact.Size > 0 && regexp.MustCompile(`^\d+\.\d+\.\d+-R\d+\.\d+$`).MatchString(version)
}

func fetchMuseArtifact(ctx context.Context, version string, artifact museArtifact, dest string) error {
	query := url.Values{"channel": {"muse"}, "version": {version}, "file": {artifact.File}}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://lookaside.facebook.com/lookaside/muse/download/?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download Muse: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return errors.New("muse download failed")
	}
	return stageMuseArtifact(response.Body, artifact, dest)
}

func stageMuseArtifact(body io.Reader, artifact museArtifact, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(dest), ".muse-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(body, artifact.Size+1))
	err = errors.Join(copyErr, temporary.Close())
	if err != nil {
		return err
	}
	if size != artifact.Size || hex.EncodeToString(hash.Sum(nil)) != artifact.SHA256 {
		return errors.New("muse release checksum or size mismatch")
	}
	if err = os.Chmod(temporary.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), dest)
}
