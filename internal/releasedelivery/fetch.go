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
	f, e := os.OpenFile(path+".lock", os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600)
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

// verifyReleases checks the selected architecture for a client, or every
// advertised architecture before promotion. No architecture falls back to another.
func verifyReleases(ctx context.Context, r Runner, t Trust, s Highwater, c Channel, arch, out string) (Highwater, error) {
	if c.Withdrawn {
		return s, nil
	}
	arches := []string{"x86_64", "aarch64"}
	if arch != "" {
		if _, e := nativebuild.OCIArchitecture(arch); e != nil {
			return s, e
		}
		if c.Releases[arch] == "" {
			return s, errors.New("architecture unavailable")
		}
		arches = []string{arch}
	}
	refsSeen := map[string]string{}
	releaseID := ""
	var serial uint64
	class := ""
	for _, a := range arches {
		ref := c.Releases[a]
		if ref == "" {
			continue
		}
		var release Release
		if e := fetchDocument(ctx, r, t, ref, filepath.Join(out, a+"-release"), &release); e != nil {
			return s, e
		}
		next, e := AdmitRelease(t, s, c, a, ref, release)
		if e != nil {
			return s, e
		}
		p, candidate, e := release.Validate(t)
		if e != nil {
			return s, e
		}
		if releaseID != "" && (p.ID != releaseID || serial != release.Serial || class != release.Class) {
			return s, errors.New("mixed architecture release identities")
		}
		releaseID = p.ID
		serial = release.Serial
		class = release.Class
		s = next // highest authenticated release, even if subsequent content is missing
		expected := map[string]string{candidate.HostReference: candidate.Host.Config}
		for _, im := range p.Images {
			expected[im.Reference] = im.Config
		}
		refs, e := release.References(t)
		if e != nil {
			return s, e
		}
		for i, ref := range refs {
			if prior, ok := refsSeen[ref]; ok {
				if prior != a {
					return s, errors.New("one platform manifest advertised for two architectures")
				}
				continue
			}
			path := filepath.Join(out, a+"-image-"+string(rune('0'+i)))
			if e = VerifyCopy(ctx, r, t, ref, "docker://"+ref, path); e != nil {
				return s, e
			}
			// inspect alone isn't verification. Here it reads only the just-verified
			// private copy, checking the actual config bytes/platform against the payload.
			config, e := r.Run(ctx, "--command-timeout=2m", "inspect", "--config", "dir:"+filepath.Join(path, "image"))
			if e != nil {
				return s, e
			}
			// skopeo may pretty-print config, so use the manifest's actual config digest,
			// and use inspect only for architecture (never hash its reformatted output).
			var metadata struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			}
			if json.Unmarshal(config, &metadata) != nil {
				return s, ErrRefused
			}
			want, _ := nativebuild.OCIArchitecture(a)
			if metadata.OS != "linux" || metadata.Architecture != want {
				return s, ErrRefused
			}
			mb, e := ReadFile(filepath.Join(path, "image/manifest.json"), 1<<20)
			if e != nil {
				return s, e
			}
			var m struct {
				Config descriptor `json:"config"`
			}
			if json.Unmarshal(mb, &m) != nil || m.Config.Digest != expected[ref] {
				return s, ErrRefused
			}
			refsSeen[ref] = a
		}
	}
	return s, nil
}

// Fetch verifies an approved channel and its images but never installs/imports,
// switches bootc, changes policy or reboots. Withdrawal advances the high-water
// mark without rewinding any installed release or application database.
func Fetch(ctx context.Context, r Runner, t Trust, name, arch, statePath, out string, now time.Time) error {
	if t.Validate() != nil || !channel(name) {
		return ErrRefused
	}
	if _, e := nativebuild.OCIArchitecture(arch); e != nil {
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
	next, verificationErr := verifyReleases(ctx, r, t, state, offer, arch, out)
	if e = saveState(statePath, next); e != nil {
		return e
	}
	if verificationErr != nil {
		return verificationErr
	}
	return writeJSON(filepath.Join(out, "verified.json"), map[string]any{"Channel": ref, "Architecture": arch, "Withdrawn": offer.Withdrawn, "Scope": "native signature/digest verification only; no installation or activation"})
}
