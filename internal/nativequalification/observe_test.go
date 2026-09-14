package nativequalification

import (
	"strings"
	"testing"

	rd "github.com/levitateos/sodaos/internal/releasedelivery"
	"github.com/stretchr/testify/require"
)

func TestReceiptRequiresWholeNativeScenario(t *testing.T) {
	b := Artifact{Candidate: rd.Candidate{PayloadSHA256: strings.Repeat("a", 64)}}
	r := Receipt{Scope: "native-install-upgrade-recovery", HostManifest: "sha256:" + strings.Repeat("b", 64), PayloadSHA256: b.Candidate.PayloadSHA256, Checks: append([]string(nil), requiredChecks...), InstallCommit: strings.Repeat("c", 64), UpdatedCommit: strings.Repeat("d", 64), RecoveredCommit: strings.Repeat("e", 64)}
	require.NoError(t, r.Validate(b, r.HostManifest))
	missing := r
	missing.Checks = missing.Checks[:len(missing.Checks)-1]
	require.Error(t, missing.Validate(b, r.HostManifest))
	other := r
	other.PayloadSHA256 = strings.Repeat("f", 64)
	require.Error(t, other.Validate(b, r.HostManifest))
	buildOnly := r
	buildOnly.Scope = "development-only"
	require.Error(t, buildOnly.Validate(b, r.HostManifest))
	noBoot := r
	noBoot.UpdatedCommit = ""
	require.Error(t, noBoot.Validate(b, r.HostManifest))
}

func TestLaterStatePreservesAAndRequiresB(t *testing.T) {
	a := map[string]any{"project": "original-container", "forgejo_repository": "original-repository", "machine_settings_public_keys": "original-identity", "schema": 10, "project_files": map[string]any{"a.txt": "a-hash"}, "forgejo_ref": "a-commit", "user": "a-user"}
	b := map[string]any{}
	for k, v := range a {
		b[k] = v
	}
	require.Error(t, laterState(a, b))
	b["project_files"] = map[string]any{"a.txt": "a-hash", "b.txt": "b-hash"}
	b["forgejo_ref"] = "b-commit"
	b["user"] = "b-user"
	require.NoError(t, laterState(a, b))
	require.NoError(t, sameState(b, b))
	require.Error(t, sameState(a, b))
	b["project"] = "recreated-container"
	require.Error(t, laterState(a, b))
}
