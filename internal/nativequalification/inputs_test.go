package nativequalification

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

func TestSameBaseScenarioBoundary(t *testing.T) {
	var a Artifact
	a.Payload.Architecture = "x86_64"
	a.Candidate.Host.Manifest = "baseline"
	b := a
	b.Candidate.Host.Manifest = "candidate"
	require.NoError(t, SameBaseScenario(a, b))
	b.Payload.Schema++
	require.Error(t, SameBaseScenario(a, b))
	b = a
	b.Candidate.Host.Manifest = "candidate"
	b.Payload.Base = "different-base"
	require.Error(t, SameBaseScenario(a, b))
	b = a
	b.Payload.Architecture = "aarch64"
	require.Error(t, SameBaseScenario(a, b))
	require.Error(t, SameBaseScenario(a, a))
}

func TestStateSchemaRefusesMigrationWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := store.Open(path)
	require.NoError(t, err)
	require.NoError(t, s.Close())
	require.NoError(t, checkStateSchema(t.Context(), path))
	before, err := nativebuild.HashFile(path)
	require.NoError(t, err)
	require.Error(t, checkStateSchema(t.Context(), filepath.Join(t.TempDir(), "missing.db")))
	after, err := nativebuild.HashFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestInputMutationBlocksQualification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "candidate.json")
	require.NoError(t, os.WriteFile(path, []byte("admitted"), 0o600))
	sum, err := nativebuild.HashFile(path)
	require.NoError(t, err)
	a := Artifact{Files: map[string]string{"candidate.json": sum}}
	require.NoError(t, Unchanged(dir, a))
	require.NoError(t, os.WriteFile(path, []byte("changed"), 0o600))
	require.Error(t, Unchanged(dir, a))
}
