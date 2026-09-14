package appliancerelease

import (
	"context"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

func TestPayloadVersionsKeepStorageMeaning(t *testing.T) {
	p := fixture()
	require.NoError(t, p.Validate())
	p.Format = 2
	require.Error(t, p.Validate())
	for name, im := range p.Images {
		im.Storage = "podman"
		p.Images[name] = im
	}
	require.NoError(t, p.Validate())
	p.Format = 1
	require.Error(t, p.Validate())
	p.Format = 3
	require.Error(t, p.Validate())
}
func TestV2ImportsAllFiveExactImagesWithoutLifecycle(t *testing.T) {
	p := fixture()
	p.Format = 2
	root := t.TempDir()
	for _, name := range Names {
		im := writeArchive(t, filepath.Join(root, name+".oci"), p)
		im.Storage = "podman"
		im.Reference = p.RepositoryPrefix + "-" + name + "@" + im.Manifest
		p.Images[name] = im
	}
	require.NoError(t, p.Validate())
	queries, loads := 0, 0
	err := ImportRetained(t.Context(), p, root, func(_ context.Context, cmd string, args ...string) error {
		require.Equal(t, "/usr/bin/podman", cmd)
		require.Equal(t, "--remote=false", args[0])
		require.NotContains(t, strings.Join(args, " "), "additionalimagestore")
		switch args[1] {
		case "load":
			require.Equal(t, filepath.Join(root, Names[loads]+".oci"), args[3])
			loads++
			return nil
		case "image":
			require.Equal(t, "exists", args[2])
			queries++
			if queries%2 == 1 {
				return statusError(1)
			}
			return nil
		default:
			t.Fatalf("unexpected lifecycle command: %v", args)
			return nil
		}
	})
	require.NoError(t, err)
	require.Equal(t, 5, loads)
	require.Equal(t, 10, queries)
}
