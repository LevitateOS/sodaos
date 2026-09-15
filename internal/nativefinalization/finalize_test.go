package nativefinalization

import (
	"os"
	"path/filepath"
	"testing"

	rd "github.com/levitateos/sodaos/internal/releasedelivery"
	"github.com/stretchr/testify/require"
)

func privateDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o700))
	return root
}

func TestAdmitConfigRefusesPartialPublication(t *testing.T) {
	root := privateDir(t)
	trust := filepath.Join(root, "trust.json")
	require.NoError(t, os.WriteFile(trust, []byte(`{}`), 0o644))
	signer := filepath.Join(root, "signer.json")
	require.NoError(t, os.WriteFile(signer, []byte(`{"Key":"/x","Passphrase":"/y"}`), 0o600))
	c := Config{Trust: trust, Signer: signer, Serial: 1, Class: "normal", Notes: "fixture", AuthFile: filepath.Join(root, "auth")}
	require.Error(t, admitConfig(c))
	require.Error(t, admitConfig(Config{Trust: trust, Signer: signer, Serial: 0, Class: "normal", Notes: "n"}))
	require.Error(t, admitConfig(Config{Trust: trust, Signer: signer, Serial: 1, Class: "other", Notes: "n"}))
}

func TestBindChannelReleasesRequiresArchitectures(t *testing.T) {
	require.Error(t, bindChannelReleases(&rd.Channel{}, "ghcr.io/example/sodaos-release@sha256:"+string(make([]byte, 0))))
	offer := rd.Channel{Releases: map[string]string{"x86_64": "old"}}
	require.NoError(t, bindChannelReleases(&offer, "ghcr.io/example/sodaos-release@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	require.Equal(t, "ghcr.io/example/sodaos-release@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", offer.Releases["x86_64"])
}

func TestQualificationEvidenceBindsExactReceiptHash(t *testing.T) {
	root := privateDir(t)
	evidence := filepath.Join(root, "qualification.json")
	require.NoError(t, os.WriteFile(evidence, []byte(`{"scope":"native-install-upgrade-recovery"}`), 0o600))
	q, err := qualificationFromEvidence(Config{Serial: 3, Class: "normal", Notes: "fixture notes"}, evidence)
	require.NoError(t, err)
	require.Equal(t, "native-install-upgrade-recovery", q.Scope)
	require.Equal(t, uint64(3), q.Serial)
	require.Contains(t, q.Evidence, "qualification.json")
}
