package releasedelivery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/stretchr/testify/require"
)

func completeRegistry(t *testing.T, root string, tr Trust) (*registryDouble, string, string, Channel) {
	t.Helper()
	f := &registryDouble{t: t, images: map[string]string{}, tags: map[string]map[string]string{}}
	release := testRelease(t, tr)
	var p appliancerelease.Payload
	var c Candidate
	require.NoError(t, decode(release.Payload, &p))
	require.NoError(t, decode(release.Candidate, &c))
	for _, n := range append([]string{"host"}, appliancerelease.Names...) {
		path := filepath.Join(root, "artifact-"+n)
		require.NoError(t, os.Mkdir(path, 0700))
		config := []byte(`{"os":"linux","architecture":"amd64"}`)
		configHash := Hash(config)
		mb, e := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": manifestType, "config": descriptor{MediaType: "application/vnd.oci.image.config.v1+json", Digest: configHash, Size: int64(len(config))}, "layers": []descriptor{}, "annotations": map[string]string{"synthetic-fixture": n}})
		require.NoError(t, e)
		require.NoError(t, os.WriteFile(filepath.Join(path, "manifest.json"), mb, 0600))
		require.NoError(t, os.WriteFile(filepath.Join(path, strings.TrimPrefix(configHash, "sha256:")), config, 0600))
		d := Hash(mb)
		ref := tr.Prefix + "-" + n + "@" + d
		f.images[ref] = path
		if n == "host" {
			c.HostReference = ref
			c.Host.Config = configHash
			c.Host.Manifest = d
		} else {
			im := p.Images[n]
			im.Reference = ref
			im.Manifest = d
			im.Config = configHash
			p.Images[n] = im
		}
	}
	release.Payload, _ = marshal(p)
	c.PayloadSHA256 = strings.TrimPrefix(Hash(release.Payload), "sha256:")
	release.Candidate, _ = marshal(c)
	rd, d := documentDir(t, root, "release", release)
	ref := tr.Prefix + "-release@" + d
	f.images[ref] = rd
	offer := testChannel(tr, "candidate")
	offer.Releases["x86_64"] = ref
	signed, d := documentDir(t, root, "offer", offer)
	return f, signed, d, offer
}
func TestWrongSnapshotDigestNeverInvokesSigning(t *testing.T) {
	root := t.TempDir()
	tr := testTrust(t)
	input, _ := documentDir(t, root, "input", map[string]string{"Scope": "fixture"})
	key := filepath.Join(root, "key")
	pass := filepath.Join(root, "pass")
	require.NoError(t, os.WriteFile(key, []byte("must not be consumed"), 0600))
	require.NoError(t, os.WriteFile(pass, []byte("must not be consumed"), 0600))
	f := &registryDouble{t: t, images: map[string]string{}, tags: map[string]map[string]string{}}
	p := Permit{Format: 1, Repository: tr.Prefix + "-host", Digest: Hash([]byte("not this input")), Expires: time.Now().Add(time.Hour).Unix()}
	out := filepath.Join(root, "attempt")
	require.Error(t, Sign(t.Context(), f, tr, p, "dir", input, out, SecretFiles{Key: key, Passphrase: pass}))
	require.Len(t, f.calls, 1)
	require.NotContains(t, strings.Join(f.calls[0], " "), "--sign-by")
	require.NoDirExists(t, filepath.Join(out, "signed"))
}

func TestAdvertisingAnUnqualifiedSiblingRefusesBeforePublication(t *testing.T) {
	root := t.TempDir()
	tr := testTrust(t)
	f, _, _, c := completeRegistry(t, root, tr)
	c.Releases["aarch64"] = c.Releases["x86_64"]
	signed, d := documentDir(t, root, "mixed-offer", c)
	p := Permit{Format: 1, Repository: tr.Prefix + "-channel-candidate", Digest: d, Previous: "absent", Expires: time.Now().Add(time.Hour).Unix()}
	auth := filepath.Join(root, "auth.json")
	require.NoError(t, os.WriteFile(auth, []byte(`{}`), 0600))
	ledger := filepath.Join(root, "ledger.json")
	require.NoError(t, InitLedger(ledger, tr, p.Repository))
	require.Error(t, Publish(t.Context(), f, tr, p, signed, auth, ledger, filepath.Join(root, "attempt"), false))
	require.Zero(t, f.writes)
}

func TestCompletePublishAndVerifiedFetchHaveNoActivation(t *testing.T) {
	root := t.TempDir()
	tr := testTrust(t)
	f, signed, d, c := completeRegistry(t, root, tr)
	p := Permit{Format: 1, Repository: tr.Prefix + "-channel-candidate", Digest: d, Previous: "absent", Expires: time.Now().Add(time.Hour).Unix()}
	auth := filepath.Join(root, "auth.json")
	require.NoError(t, os.WriteFile(auth, []byte(`{}`), 0600))
	ledger := filepath.Join(root, "ledger.json")
	require.NoError(t, InitLedger(ledger, tr, p.Repository))
	require.NoError(t, Publish(t.Context(), f, tr, p, signed, auth, ledger, filepath.Join(root, "publication"), false))
	require.Equal(t, 2, f.writes)
	state := filepath.Join(root, "state.json")
	require.NoError(t, InitState(state, tr))
	fetch := filepath.Join(root, "fetch")
	require.NoError(t, Fetch(t.Context(), f, tr, c.Name, "x86_64", state, fetch, time.Now()))
	require.FileExists(t, filepath.Join(fetch, "verified.json"))
	require.Equal(t, 2, f.writes, "fetch never publishes or activates")
	var s Highwater
	require.NoError(t, ReadJSON(state, &s))
	require.Equal(t, uint64(10), s.Serials["x86_64"])
	for _, args := range f.calls {
		all := strings.Join(args, " ")
		require.NotContains(t, all, "--insecure-policy")
		require.NotContains(t, all, "--retry-times")
		require.NotContains(t, all, "--src-creds")
		require.NotContains(t, all, "--dest-creds")
		require.NotContains(t, all, "bootc")
		require.NotContains(t, all, "podman")
	}
}
