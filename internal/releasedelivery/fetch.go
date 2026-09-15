package releasedelivery

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

func discover(ctx context.Context, r Runner, t Trust, name string) (string, error) {
	if !channel(name) {
		return "", ErrRefused
	}
	ref := t.Prefix + "-channel-" + name + ":" + name
	b, e := r.Run(ctx, "--command-timeout=2m", "inspect", "--raw", "--no-creds", "docker://"+ref)
	if e != nil || len(b) > 1<<20 {
		return "", ErrUnavailable
	}
	// An unauthenticated discovery digest is only a locator. Fetch/verify that
	// immutable manifest with the appropriate channel key before using its data.
	return t.Prefix + "-channel-" + name + "@" + Hash(b), nil
}

// State is explicit, private and durable. Missing/corrupt state never becomes an
// implicit fresh bootstrap. The installed client will own a fixed state path in
// M3; these build-side tools have no root update/status API.
func InitState(path string, t Trust) error {
	if t.Validate() != nil {
		return ErrRefused
	}
	if e := nativebuild.PrivateDestination(path); e != nil {
		return e
	}
	s := EmptyState()
	s.TrustEpoch = t.Epoch
	s.CheckedAt = nowUTC().Unix()
	return writeJSON(path, s)
}

func lockState(path string) (*os.File, error) {
	if e := PrivateFile(path); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(path+".lock", os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, errors.New("release operation already active")
	}
	return f, nil
}
func unlock(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }
func saveState(path string, value any) error {
	b, e := marshal(value)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".delivery-state-")
	if e != nil {
		return e
	}
	// Preserve incomplete write attempts rather than repairing them over later state.
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(f.Name(), path); e != nil {
		return e
	}
	parent, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer parent.Close()
	return parent.Sync()
}

func fetchDocument(ctx context.Context, r Runner, t Trust, ref, out string, v any) error {
	if e := VerifyCopy(ctx, r, t, ref, "docker://"+ref, out); e != nil {
		return e
	}
	_, d, _ := strings.Cut(ref, "@")
	return ReadDocument(filepath.Join(out, "image"), d, v)
}

func resolveVerificationArchitectures(c Channel, arch string) ([]string, error) {
	if arch == "" {
		return []string{"x86_64", "aarch64"}, nil
	}
	if _, err := nativebuild.OCIArchitecture(arch); err != nil {
		return nil, err
	}
	if c.Releases[arch] == "" {
		return nil, errors.New("architecture unavailable")
	}
	return []string{arch}, nil
}

func checkVerifiedImageMetadata(config []byte, arch string) error {
	var metadata struct {
		Architecture string `json:"architecture"`
		OS           string `json:"os"`
	}
	if json.Unmarshal(config, &metadata) != nil {
		return ErrRefused
	}
	want, _ := nativebuild.OCIArchitecture(arch)
	if metadata.OS != "linux" || metadata.Architecture != want {
		return ErrRefused
	}
	return nil
}

func checkVerifiedImageManifest(manifestPath, expectedConfig string) error {
	mb, err := ReadFile(manifestPath, 1<<20)
	if err != nil {
		return err
	}
	var m struct {
		Config descriptor `json:"config"`
	}
	if json.Unmarshal(mb, &m) != nil || m.Config.Digest != expectedConfig {
		return ErrRefused
	}
	return nil
}

func verifyReleaseImageCopy(ctx context.Context, r Runner, t Trust, ref, arch, path string, expectedConfig string) error {
	if err := VerifyCopy(ctx, r, t, ref, "docker://"+ref, path); err != nil {
		return err
	}
	// inspect alone isn't verification. Here it reads only the just-verified
	// private copy, checking the actual config bytes/platform against the payload.
	config, err := r.Run(ctx, "--command-timeout=2m", "inspect", "--config", "dir:"+filepath.Join(path, "image"))
	if err != nil {
		return err
	}
	if err := checkVerifiedImageMetadata(config, arch); err != nil {
		return err
	}
	return checkVerifiedImageManifest(filepath.Join(path, "image/manifest.json"), expectedConfig)
}

func buildExpectedReleaseConfigs(candidate Candidate, p appliancerelease.Payload) map[string]string {
	expected := map[string]string{candidate.HostReference: candidate.Host.Config}
	for _, im := range p.Images {
		expected[im.Reference] = im.Config
	}
	return expected
}

func verifyReleaseImages(ctx context.Context, r Runner, t Trust, arch, out string, refs []string, expected map[string]string, refsSeen map[string]string) error {
	for i, ref := range refs {
		if prior, ok := refsSeen[ref]; ok {
			if prior != arch {
				return errors.New("one platform manifest advertised for two architectures")
			}
			continue
		}
		path := filepath.Join(out, arch+"-image-"+string(rune('0'+i)))
		if err := verifyReleaseImageCopy(ctx, r, t, ref, arch, path, expected[ref]); err != nil {
			return err
		}
		refsSeen[ref] = arch
	}
	return nil
}

type releaseIdentityTracker struct {
	releaseID string
	serial    uint64
	class     string
}

func (t *releaseIdentityTracker) check(id string, serial uint64, class string) error {
	if t.releaseID != "" && (id != t.releaseID || serial != t.serial || class != t.class) {
		return errors.New("mixed architecture release identities")
	}
	t.releaseID = id
	t.serial = serial
	t.class = class
	return nil
}

func verifyArchitectureRelease(ctx context.Context, r Runner, t Trust, s Highwater, c Channel, arch, ref, out string, tracker *releaseIdentityTracker, refsSeen map[string]string) (Highwater, error) {
	var release Release
	if err := fetchDocument(ctx, r, t, ref, filepath.Join(out, arch+"-release"), &release); err != nil {
		return s, err
	}
	next, err := AdmitRelease(t, s, c, arch, ref, release)
	if err != nil {
		return s, err
	}
	p, candidate, err := release.Validate(t)
	if err != nil {
		return s, err
	}
	if err := tracker.check(p.ID, release.Serial, release.Class); err != nil {
		return s, err
	}
	refs, err := release.References(t)
	if err != nil {
		return s, err
	}
	expected := buildExpectedReleaseConfigs(candidate, p)
	if err := verifyReleaseImages(ctx, r, t, arch, out, refs, expected, refsSeen); err != nil {
		return next, err
	}
	return next, nil
}

// verifyReleases checks the selected architecture for a client, or every
// advertised architecture before promotion. No architecture falls back to another.
func verifyReleases(ctx context.Context, r Runner, t Trust, s Highwater, c Channel, arch, out string) (Highwater, error) {
	if c.Withdrawn {
		return s, nil
	}
	arches, err := resolveVerificationArchitectures(c, arch)
	if err != nil {
		return s, err
	}
	refsSeen := map[string]string{}
	var tracker releaseIdentityTracker
	for _, a := range arches {
		ref := c.Releases[a]
		if ref == "" {
			continue
		}
		next, err := verifyArchitectureRelease(ctx, r, t, s, c, a, ref, out, &tracker, refsSeen)
		if err != nil {
			return next, err
		}
		s = next // highest authenticated release, even if subsequent content is missing
	}
	return s, nil
}

// Fetch verifies an approved channel and its images but never installs/imports,
// switches bootc, changes policy or reboots. Withdrawal advances the high-water
// mark without rewinding any installed release or application database.
func admitFetchRequest(t Trust, name, arch string) error {
	if t.Validate() != nil || !channel(name) {
		return ErrRefused
	}
	_, e := nativebuild.OCIArchitecture(arch)
	return e
}

func completeFetch(ctx context.Context, r Runner, t Trust, state Highwater, offer Channel, ref, arch, statePath, out string) error {
	next, verificationErr := verifyReleases(ctx, r, t, state, offer, arch, out)
	if e := saveState(statePath, next); e != nil {
		return e
	}
	if verificationErr != nil {
		return verificationErr
	}
	return writeJSON(filepath.Join(out, "verified.json"), map[string]any{"Channel": ref, "Architecture": arch, "Withdrawn": offer.Withdrawn, "Scope": "native signature/digest verification only; no installation or activation"})
}

func Fetch(ctx context.Context, r Runner, t Trust, name, arch, statePath, out string, now time.Time) error {
	if e := admitFetchRequest(t, name, arch); e != nil {
		return e
	}
	lock, e := lockState(statePath)
	if e != nil {
		return e
	}
	defer unlock(lock)
	var state Highwater
	if e = ReadJSON(statePath, &state); e != nil {
		return e
	}
	if e = nativebuild.FreshDirectory(out); e != nil {
		return e
	}
	ref, e := discover(ctx, r, t, name)
	if e != nil {
		return e
	}
	var offer Channel
	if e = fetchDocument(ctx, r, t, ref, filepath.Join(out, "channel"), &offer); e != nil {
		return e
	}
	_, digest, _ := strings.Cut(ref, "@")
	state, e = AdmitChannel(t, state, offer, digest, name, now)
	if e != nil {
		return e
	}
	if e = saveState(statePath, state); e != nil {
		return e
	}
	return completeFetch(ctx, r, t, state, offer, ref, arch, statePath, out)
}
