package deliver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func sharedFixture(t *testing.T) (Payload, string) {
	t.Helper()
	p := fixture()
	root := t.TempDir()
	layout := filepath.Join(root, "layout")
	for _, name := range Names {
		archive := filepath.Join(root, name+".oci")
		im := writeArchive(t, archive, p)
		im.Reference = p.RepositoryPrefix + "-" + name + "@" + im.Manifest
		p.Images[name] = im
		testoci.Add(t, archive, layout)
	}
	return p, layout
}

func TestV3ImportsExactLocalReferencesWithoutLifecycle(t *testing.T) {
	for _, alreadyPresent := range []bool{false, true} {
		p, root := sharedFixture(t)
		files, size, err := VerifyContent(p, root)
		require.NoError(t, err)
		require.Len(t, files, 13)
		require.Positive(t, size)
		present := map[string]bool{}
		queries, pulls := 0, 0
		err = ImportImages(t.Context(), p, root, func(_ context.Context, cmd string, args ...string) error {
			require.Equal(t, "/usr/bin/podman", cmd)
			require.Equal(t, "--remote=false", args[0])
			switch args[1] {
			case "image":
				require.Equal(t, "exists", args[2])
				require.Len(t, args, 4)
				queries++
				if alreadyPresent || present[args[3]] {
					return nil
				}
				return statusError(1)
			case "pull":
				im := p.Images[Names[pulls]]
				require.Equal(t, []string{"--remote=false", "pull", "--retry=0", "oci:" + root + ":" + im.Config}, args)
				present[im.Config] = true
				pulls++
				return nil
			default:
				t.Fatalf("unexpected native operation: %v", args)
				return nil
			}
		})
		require.NoError(t, err)
		if alreadyPresent {
			require.Equal(t, 5, queries)
			require.Zero(t, pulls)
		} else {
			require.Equal(t, 10, queries)
			require.Equal(t, 5, pulls)
		}
	}
}

func TestV3RefusesWholeLayoutBeforeAnyImport(t *testing.T) {
	for _, kind := range []string{"bad-last-config", "wrong-manifest", "relative-path", "transport-separator", "wrong-format"} {
		t.Run(kind, func(t *testing.T) {
			p, root := sharedFixture(t)
			switch kind {
			case "bad-last-config":
				require.NoError(t, os.WriteFile(filepath.Join(root, "blobs/sha256", strings.TrimPrefix(p.Images[Names[len(Names)-1]].Config, "sha256:")), []byte("bad"), 0o644))
			case "wrong-manifest":
				im := p.Images["dashboard"]
				im.Manifest = "sha256:" + strings.Repeat("9", 64)
				im.Reference = p.RepositoryPrefix + "-dashboard@" + im.Manifest
				p.Images["dashboard"] = im
			case "relative-path":
				root = "relative/layout"
			case "transport-separator":
				root += "ignored:selector"
			case "wrong-format":
				p.Format = 2
			}
			calls := 0
			err := ImportImages(t.Context(), p, root, func(context.Context, string, ...string) error { calls++; return nil })
			require.Error(t, err)
			require.Zero(t, calls)
		})
	}
}

func TestV3NativeFailuresRemainUnconfirmedWithoutReplay(t *testing.T) {
	for _, kind := range []string{"observation", "pull", "postcheck", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			p, root := sharedFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if kind == "cancelled" {
				cancel()
			}
			queries, pulls := 0, 0
			err := ImportImages(ctx, p, root, func(_ context.Context, _ string, args ...string) error {
				if args[1] == "image" {
					queries++
					if kind == "observation" {
						return statusError(125)
					}
					return statusError(1)
				}
				require.Equal(t, "pull", args[1])
				pulls++
				if kind == "pull" {
					return errors.New("DO_NOT_LOG native diagnostic")
				}
				return nil
			})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "DO_NOT_LOG")
			switch kind {
			case "observation":
				require.Equal(t, 1, queries)
				require.Zero(t, pulls)
			case "pull":
				require.Equal(t, 1, queries)
				require.Equal(t, 1, pulls)
			case "postcheck":
				require.Equal(t, 2, queries)
				require.Equal(t, 1, pulls)
			case "cancelled":
				require.ErrorIs(t, err, context.Canceled)
				require.Zero(t, queries)
				require.Zero(t, pulls)
			}
		})
	}
}
