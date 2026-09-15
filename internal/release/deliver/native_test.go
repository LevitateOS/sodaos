package deliver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/stretchr/testify/require"
)

// Explicit local filesystem-only native proof. No registry listener, container,
// installed policy, provider, real signing identity or cleanup is involved.
type offlineNative struct{ Native }

func (n offlineNative) Run(ctx context.Context, args ...string) ([]byte, error) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "docker://") {
			return nil, ErrRefused
		}
	}
	return n.Native.Run(ctx, args...)
}

func TestNativeSigstoreDirectoryRoundTrip(t *testing.T) {
	out := os.Getenv("SODA_RELEASE_NATIVE_OUT")
	if out == "" {
		t.Skip("explicit fresh local native proof output required")
	}
	require.NoError(t, build.FreshDirectory(out))
	n := offlineNative{Native{Home: out}}
	require.NoError(t, CheckNative(t.Context(), n))
	pass := filepath.Join(out, "synthetic-passphrase")
	var random [32]byte
	_, e := rand.Read(random[:])
	require.NoError(t, e)
	require.NoError(t, build.WriteNew(pass, []byte(hex.EncodeToString(random[:])), 0o600))
	tr := testTrust(t)
	keys := map[string]SecretFiles{}
	for _, role := range []string{"artifact", "candidate", "preview", "stable", "wrong"} {
		prefix := filepath.Join(out, "synthetic-"+role)
		_, e = n.Run(t.Context(), "generate-sigstore-key", "--output-prefix", prefix, "--passphrase-file", pass)
		require.NoError(t, e)
		pub, e := ReadFile(prefix+".pub", 16384)
		require.NoError(t, e)
		keys[role] = SecretFiles{Key: prefix + ".private", Passphrase: pass}
		if role != "wrong" {
			tr.Keys[role] = []string{string(pub)}
		}
	}
	require.NoError(t, writeJSON(filepath.Join(out, "synthetic-trust.json"), tr))
	doc := filepath.Join(out, "document")
	digest, e := WriteDocument(doc, map[string]string{"Scope": "synthetic native signature fixture; not an appliance"})
	require.NoError(t, e)
	permit := Permit{Format: 1, Repository: tr.Prefix + "-forgejo", Digest: digest, Expires: time.Now().Add(time.Hour).Unix()}
	signed := filepath.Join(out, "signed-artifact")
	require.NoError(t, Sign(t.Context(), n, tr, permit, "oci", doc, signed, keys["artifact"]))
	ref := permit.Repository + "@" + digest
	input := "dir:" + filepath.Join(signed, "signed")
	require.NoError(t, VerifyCopy(t.Context(), n, tr, ref, input, filepath.Join(out, "valid")))
	var value map[string]string
	require.NoError(t, ReadDocument(filepath.Join(out, "valid/image"), digest, &value))
	require.Contains(t, value["Scope"], "synthetic")
	require.Error(t, VerifyCopy(t.Context(), n, tr, ref, "dir:"+filepath.Join(signed, "snapshot"), filepath.Join(out, "unsigned")))
	require.Error(t, VerifyCopy(t.Context(), n, tr, tr.Prefix+"-host@"+digest, input, filepath.Join(out, "wrong-repository")))
	// Existing correct signatures cannot mask use of the wrong signing key.
	require.Error(t, Sign(t.Context(), n, tr, permit, "dir", filepath.Join(signed, "signed"), filepath.Join(out, "wrong-signer"), keys["wrong"]))
	// Rotation overlap accepts the existing signature; removing its key refuses it.
	old := tr.Keys["artifact"][0]
	// Use the otherwise-unused wrong key as a synthetic replacement, not a role-shared key.
	replacement, e := ReadFile(keys["wrong"].Key[:len(keys["wrong"].Key)-len(".private")]+".pub", 16384)
	require.NoError(t, e)
	tr.Keys["artifact"] = []string{old, string(replacement)}
	require.NoError(t, tr.Validate())
	require.NoError(t, VerifyCopy(t.Context(), n, tr, ref, input, filepath.Join(out, "overlap")))
	tr.Keys["artifact"] = []string{string(replacement)}
	require.Error(t, VerifyCopy(t.Context(), n, tr, ref, input, filepath.Join(out, "revoked")))
	tr.Keys["artifact"] = []string{old}
	c := testChannel(tr, "preview")
	c.Withdrawn = true
	c.Releases = nil
	channelOCI := filepath.Join(out, "preview-document")
	cd, e := WriteDocument(channelOCI, c)
	require.NoError(t, e)
	cp := Permit{Format: 1, Repository: tr.Prefix + "-channel-preview", Digest: cd, Expires: permit.Expires}
	preview := filepath.Join(out, "signed-preview")
	require.NoError(t, Sign(t.Context(), n, tr, cp, "oci", channelOCI, preview, keys["preview"]))
	require.Error(t, VerifyCopy(t.Context(), n, tr, tr.Prefix+"-channel-stable@"+cd, "dir:"+filepath.Join(preview, "signed"), filepath.Join(out, "preview-as-stable")))
	// Tamper only the separately owned verification copy, preserving original evidence.
	tamper := filepath.Join(out, "valid/image/manifest.json")
	b, e := ReadFile(tamper, 1<<20)
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(tamper, append([]byte(" "), b...), 0o600))
	require.Error(t, VerifyCopy(t.Context(), n, tr, ref, "dir:"+filepath.Join(out, "valid/image"), filepath.Join(out, "tampered")))
	require.NoError(t, writeJSON(filepath.Join(out, "receipt.json"), map[string]any{"Skopeo": "1.22.2", "Positive": []string{"native local signing", "native signature/digest verification", "rotation overlap"}, "Refused": []string{"unsigned", "wrong repository", "wrong signer despite preexisting valid signature", "revoked key", "preview as stable", "tampered manifest"}, "Scope": "filesystem transports only; no GHCR, bootc deployment or cache/import proof"}))
}
