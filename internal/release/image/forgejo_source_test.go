package image

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractForgejoSnapshotLaysOutReplacePath(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(out, "work"), 0o700))
	var calls []string
	execute := func(dir, name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}
	revision := strings.Repeat("2", 40)
	err := extractForgejoSnapshot(Request{Out: out, ForgejoSource: "/fork", ForgejoRevision: revision}, execute)
	require.NoError(t, err)
	info, err := os.Stat(filepath.Join(out, "work/forgejo-ext"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Len(t, calls, 2)
	require.Contains(t, calls[0], "archive")
	require.Contains(t, calls[0], revision)
	require.Contains(t, calls[0], filepath.Join(out, "inputs/forgejo-source.tar"))
	require.True(t, strings.HasPrefix(calls[1], "tar "))
}

func TestExtractForgejoSnapshotRefusesBadRevision(t *testing.T) {
	out := t.TempDir()
	called := false
	execute := func(dir, name string, args ...string) error {
		called = true
		return nil
	}
	err := extractForgejoSnapshot(Request{Out: out, ForgejoSource: "/fork", ForgejoRevision: "short"}, execute)
	require.Error(t, err)
	require.False(t, called)
	_, err = os.Stat(filepath.Join(out, "work/forgejo-ext"))
	require.True(t, os.IsNotExist(err))
}

func TestForgejoSnapshotDirMatchesGoModReplace(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(sourceRoot(t), "go.mod"))
	require.NoError(t, err)
	var replaced string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		const prefix = "replace forgejo.org/extension-sdk => "
		if rest, ok := strings.CutPrefix(line, prefix); ok {
			replaced = rest
		}
	}
	require.Equal(t, "../forgejo-ext/sdk", replaced,
		"soda snapshot expects the fork at work/forgejo-ext; keep extractForgejoSnapshot aligned")
}
