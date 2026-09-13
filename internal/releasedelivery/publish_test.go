package releasedelivery

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This is a registry/process double for ordering/failure/state tests, NOT native
// cryptographic or GHCR evidence. Native crypto has a separate opt-in fixture.
type registryDouble struct {
	t                        *testing.T
	images                   map[string]string
	tags                     map[string]map[string]string
	writes                   int
	failPromotionAfterCommit bool
	calls                    [][]string
}

func (f *registryDouble) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{}, args...))
	has := func(s string) bool {
		for _, a := range args {
			if a == s {
				return true
			}
		}
		return false
	}
	if has("copy") {
		src, dst := args[len(args)-2], args[len(args)-1]
		path := strings.TrimPrefix(src, "dir:")
		if strings.HasPrefix(src, "docker://") {
			ref := strings.TrimPrefix(src, "docker://")
			path = f.images[ref]
			if path == "" {
				return nil, ErrUnavailable
			}
		}
		if strings.HasPrefix(dst, "docker://") {
			target := strings.TrimPrefix(dst, "docker://")
			repo, tag, ok := strings.Cut(target, ":")
			require.True(f.t, ok)
			mb, e := os.ReadFile(filepath.Join(path, "manifest.json"))
			require.NoError(f.t, e)
			d := Hash(mb)
			f.writes++
			f.images[repo+"@"+d] = path
			if f.tags[repo] == nil {
				f.tags[repo] = map[string]string{}
			}
			f.tags[repo][tag] = d
			if channel(tag) && f.failPromotionAfterCommit {
				return nil, ErrUnavailable
			}
			return nil, nil
		}
		require.True(f.t, strings.HasPrefix(dst, "dir:"))
		dest := strings.TrimPrefix(dst, "dir:")
		require.NoError(f.t, os.Mkdir(dest, 0700))
		entries, e := os.ReadDir(path)
		require.NoError(f.t, e)
		for _, entry := range entries {
			require.False(f.t, entry.IsDir())
			b, e := os.ReadFile(filepath.Join(path, entry.Name()))
			require.NoError(f.t, e)
			require.NoError(f.t, os.WriteFile(filepath.Join(dest, entry.Name()), b, 0600))
		}
		return nil, nil
	}
	if has("list-tags") {
		repo := strings.TrimPrefix(args[len(args)-1], "docker://")
		tags := []string{}
		for tag := range f.tags[repo] {
			tags = append(tags, tag)
		}
		return json.Marshal(map[string]any{"Repository": repo, "Tags": tags})
	}
	if has("inspect") && has("--raw") {
		target := strings.TrimPrefix(args[len(args)-1], "docker://")
		repo, tag, _ := strings.Cut(target, ":")
		d := f.tags[repo][tag]
		path := f.images[repo+"@"+d]
		if path == "" {
			return nil, ErrUnavailable
		}
		return os.ReadFile(filepath.Join(path, "manifest.json"))
	}
	if has("inspect") && has("--config") {
		path := strings.TrimPrefix(args[len(args)-1], "dir:")
		mb, e := os.ReadFile(filepath.Join(path, "manifest.json"))
		require.NoError(f.t, e)
		var m struct{ Config descriptor }
		require.NoError(f.t, json.Unmarshal(mb, &m))
		return os.ReadFile(filepath.Join(path, strings.TrimPrefix(m.Config.Digest, "sha256:")))
	}
	return nil, errors.New("unexpected test-double operation")
}
func documentDir(t *testing.T, root, name string, value any) (string, string) {
	t.Helper()
	oci := filepath.Join(root, name+"-oci")
	d, e := WriteDocument(oci, value)
	require.NoError(t, e)
	dest := filepath.Join(root, name+"-dir")
	require.NoError(t, os.Mkdir(dest, 0700))
	mb, e := os.ReadFile(filepath.Join(oci, "blobs/sha256", strings.TrimPrefix(d, "sha256:")))
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(filepath.Join(dest, "manifest.json"), mb, 0600))
	entries, e := os.ReadDir(filepath.Join(oci, "blobs/sha256"))
	require.NoError(t, e)
	for _, entry := range entries {
		b, e := os.ReadFile(filepath.Join(oci, "blobs/sha256", entry.Name()))
		require.NoError(t, e)
		require.NoError(t, os.WriteFile(filepath.Join(dest, entry.Name()), b, 0600))
	}
	return dest, d
}
func TestPublicationCommitLastAndUncertainResultIsObservedNotReplayed(t *testing.T) {
	root := t.TempDir()
	tr := testTrust(t)
	c := testChannel(tr, "candidate")
	c.Withdrawn = true
	c.Releases = nil
	signed, d := documentDir(t, root, "channel", c)
	p := Permit{Format: 1, Repository: tr.Prefix + "-channel-candidate", Digest: d, Previous: "absent", Expires: time.Now().Add(time.Hour).Unix()}
	auth := filepath.Join(root, "auth.json")
	require.NoError(t, os.WriteFile(auth, []byte(`{}`), 0600))
	ledger := filepath.Join(root, "ledger.json")
	require.NoError(t, InitLedger(ledger, tr, p.Repository))
	f := &registryDouble{t: t, images: map[string]string{}, tags: map[string]map[string]string{}, failPromotionAfterCommit: true}
	require.Error(t, Publish(t.Context(), f, tr, p, signed, auth, ledger, filepath.Join(root, "first"), false))
	require.Equal(t, 2, f.writes)
	var l Ledger
	require.NoError(t, ReadJSON(ledger, &l))
	require.Equal(t, "pending", l.Phase)
	require.ErrorContains(t, Publish(t.Context(), f, tr, p, signed, auth, ledger, filepath.Join(root, "repeat"), false), "held")
	require.Equal(t, 2, f.writes)
	// Even an expired permit can observe a previously recorded commit, not write it.
	p.Expires = 1
	require.NoError(t, Publish(t.Context(), f, tr, p, "", "", ledger, filepath.Join(root, "observe"), true))
	require.Equal(t, 2, f.writes)
	require.NoError(t, ReadJSON(ledger, &l))
	require.Equal(t, "complete", l.Phase)
	require.NoError(t, Publish(t.Context(), f, tr, p, "", "", ledger, filepath.Join(root, "duplicate"), false))
	require.Equal(t, 2, f.writes)
	firstWrite, lastWrite := -1, -1
	roundTrip := -1
	for i, args := range f.calls {
		dst := args[len(args)-1]
		copy := false
		for _, arg := range args {
			if arg == "copy" {
				copy = true
			}
		}
		if copy && strings.HasPrefix(dst, "docker://") {
			if firstWrite < 0 {
				firstWrite = i
			}
			lastWrite = i
		}
		if strings.Contains(strings.Join(args, " "), "registry-roundtrip/image") {
			roundTrip = i
		}
	}
	require.Greater(t, roundTrip, firstWrite)
	require.Greater(t, lastWrite, roundTrip, "immutable content/signature round-trip must precede mutable channel tag")
}
func TestMissingArtifactAndUnqualifiedStableNeverWriteRegistry(t *testing.T) {
	for _, kind := range []string{"missing-release", "missing-image", "unqualified-stable", "missing-architecture"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			tr := testTrust(t)
			c := testChannel(tr, "candidate")
			f := &registryDouble{t: t, images: map[string]string{}, tags: map[string]map[string]string{}}
			if kind != "missing-release" {
				release := testRelease(t, tr)
				rd, hash := documentDir(t, root, "release", release)
				ref := tr.Prefix + "-release@" + hash
				f.images[ref] = rd
				c.Releases["x86_64"] = ref
				if kind == "unqualified-stable" {
					c.Name = "stable"
				}
				if kind == "missing-architecture" {
					c.Releases["aarch64"] = ref
				}
			}
			signed, d := documentDir(t, root, "channel", c)
			p := Permit{Format: 1, Repository: tr.Prefix + "-channel-" + c.Name, Digest: d, Previous: "absent", Expires: time.Now().Add(time.Hour).Unix()}
			auth := filepath.Join(root, "auth.json")
			require.NoError(t, os.WriteFile(auth, []byte(`{}`), 0600))
			ledger := filepath.Join(root, "ledger.json")
			require.NoError(t, InitLedger(ledger, tr, p.Repository))
			require.Error(t, Publish(t.Context(), f, tr, p, signed, auth, ledger, filepath.Join(root, "attempt"), false))
			require.Zero(t, f.writes)
		})
	}
}
func TestFetchAdvancesObservedAuthorityEvenWhenImageUnavailable(t *testing.T) {
	root := t.TempDir()
	tr := testTrust(t)
	release := testRelease(t, tr)
	rd, d := documentDir(t, root, "release", release)
	c := testChannel(tr, "candidate")
	ref := tr.Prefix + "-release@" + d
	c.Releases["x86_64"] = ref
	cd, ch := documentDir(t, root, "channel", c)
	repo := tr.Prefix + "-channel-candidate"
	f := &registryDouble{t: t, images: map[string]string{ref: rd, repo + "@" + ch: cd}, tags: map[string]map[string]string{repo: {"candidate": ch}}}
	state := filepath.Join(root, "state.json")
	require.NoError(t, InitState(state, tr))
	require.Error(t, Fetch(t.Context(), f, tr, "candidate", "x86_64", state, filepath.Join(root, "fetch"), time.Now()))
	var observed Highwater
	require.NoError(t, ReadJSON(state, &observed))
	require.Equal(t, c.Sequence, observed.Channels["candidate"].Sequence)
	require.Equal(t, release.Serial, observed.Serials["x86_64"])
	require.Zero(t, f.writes)
	require.NoFileExists(t, filepath.Join(root, "fetch/verified.json"))
	// A different architecture is an honest unavailable result, never fallback.
	require.ErrorContains(t, Fetch(t.Context(), f, tr, "candidate", "aarch64", state, filepath.Join(root, "arm"), time.Now()), "architecture unavailable")
}
