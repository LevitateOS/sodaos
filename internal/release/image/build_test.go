package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/stretchr/testify/require"
)

func TestControllerAdmissionRefusesBeforeProduction(t *testing.T) {
	for _, mode := range []string{"dirty", "revision", "occupied", "outside", "worktree", "architecture", "forgejo-dirty", "forgejo-revision", "forgejo-worktree"} {
		t.Run(mode, func(t *testing.T) {
			source := t.TempDir()
			forgejo := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(source, ".git"), 0o700))
			require.NoError(t, os.Mkdir(filepath.Join(forgejo, ".git"), 0o700))
			parent := filepath.Join(source, ".artifacts/releases")
			require.NoError(t, os.MkdirAll(parent, 0o700))
			out := filepath.Join(parent, "run")
			if mode == "occupied" {
				require.NoError(t, os.Mkdir(out, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(out, "retain"), []byte("later writes"), 0o600))
			}
			if mode == "worktree" {
				require.NoError(t, os.Rename(filepath.Join(source, ".git"), filepath.Join(source, "retained-git")))
				require.NoError(t, os.WriteFile(filepath.Join(source, ".git"), []byte("gitdir: elsewhere"), 0o600))
			}
			if mode == "forgejo-worktree" {
				require.NoError(t, os.Rename(filepath.Join(forgejo, ".git"), filepath.Join(forgejo, "retained-git")))
				require.NoError(t, os.WriteFile(filepath.Join(forgejo, ".git"), []byte("gitdir: elsewhere"), 0o600))
			}
			authority := filepath.Join(source, "fixture-authority.json")
			require.NoError(t, os.WriteFile(authority, []byte("{}"), 0o600))
			r := Request{Source: source, ForgejoSource: forgejo, ForgejoRevision: strings.Repeat("f", 40), Out: out, Arch: "x86_64", RepositoryPrefix: "ghcr.io/example/sodaos", RootfsBaseURL: "https://example.invalid", MediaAuthority: authority}
			if mode == "outside" {
				r.Out = filepath.Join(source, "outside")
			}
			if mode == "architecture" {
				r.Arch = "not-native"
			}
			if mode == "revision" {
				r.Revision = strings.Repeat("b", 40)
			}
			if mode == "forgejo-revision" {
				r.ForgejoRevision = strings.Repeat("b", 40)
			}
			capture := func(dir, name string, args ...string) (string, error) {
				if name == "go" {
					require.Equal(t, []string{"env", "GOVERSION"}, args)
					return runtime.Version(), nil
				}
				require.Equal(t, "git", name)
				require.Equal(t, []string{"-c", "safe.directory=" + dir}, args[:2])
				switch strings.Join(args[2:], " ") {
				case "rev-parse --show-toplevel":
					return dir, nil
				case "status --porcelain --untracked-files=normal":
					if mode == "dirty" && dir == source || mode == "forgejo-dirty" && dir == forgejo {
						return " M source.go", nil
					}
					return "", nil
				case "rev-parse HEAD":
					if dir == forgejo {
						return strings.Repeat("f", 40), nil
					}
					return strings.Repeat("a", 40), nil
				}
				t.Fatalf("unexpected preflight %v", args)
				return "", nil
			}
			p := &build.BuildProgress{Now: func() time.Duration { return time.Second }, Stderr: io.Discard}
			_, err := runBuild(t.Context(), r, p, func(string, string, ...string) error { t.Fatal("refused input dispatched production"); return nil }, capture, func(string) (func() error, error) { t.Fatal("refused input opened logs"); return nil, nil })
			require.Error(t, err)
			if mode == "occupied" {
				data, e := os.ReadFile(filepath.Join(out, "retain"))
				require.NoError(t, e)
				require.Equal(t, "later writes", string(data))
			} else {
				require.NoDirExists(t, out)
			}
		})
	}
}

func TestDevelopmentTargetAdmission(t *testing.T) {
	for _, tc := range []struct {
		r     Request
		valid bool
	}{
		{Request{RootfsBaseURL: "https://example.invalid"}, true},
		{Request{Development: true, Target: "candidate"}, true},
		{Request{Development: true, Target: "media", RootfsBaseURL: "https://example.invalid"}, true},
		{Request{}, false},
		{Request{Development: true}, false},
		{Request{Target: "candidate"}, false},
		{Request{Target: "media", RootfsBaseURL: "https://example.invalid"}, false},
		{Request{Development: true, Target: "release"}, false},
		{Request{Development: true, Target: "candidate", RootfsBaseURL: "https://example.invalid"}, false},
		{Request{Development: true, Target: "candidate", MediaAuthority: "/private/unused"}, false},
		{Request{Development: true, Target: "media"}, false},
	} {
		if tc.valid {
			require.NoError(t, tc.r.ValidateTarget())
		} else {
			require.Error(t, tc.r.ValidateTarget())
		}
	}
}

func TestCandidateBoundaryDoesNotAdmitOrDispatchMedia(t *testing.T) {
	root := t.TempDir()
	r := Request{Development: true, Target: "candidate", Out: root}
	p := build.Production{
		Source: filepath.Join(root, "absent-source"), Out: filepath.Join(root, "artifacts"), Arch: "x86_64", Revision: strings.Repeat("a", 40),
		Next:    func(string) error { t.Fatal("candidate dispatched media progress"); return nil },
		Execute: func(string, string, ...string) error { t.Fatal("candidate dispatched media command"); return nil },
		Capture: func(string, string, ...string) (string, error) {
			t.Fatal("candidate dispatched media capture")
			return "", nil
		},
	}
	tools, lock, err := prepareBuildMedia(p, r)
	require.NoError(t, err)
	require.NoError(t, finishBuildMedia(t.Context(), p, r, tools, lock, p.Next))
	require.NoDirExists(t, filepath.Join(root, "work/media"))
	require.NoDirExists(t, p.Out)
	// A successful fixed candidate path records only its own checks/identities.
	require.NoError(t, os.Mkdir(p.Out, 0o700))
	require.NoError(t, os.Mkdir(filepath.Join(root, "evidence"), 0o700))
	candidate := deliver.Candidate{Host: build.Image{Manifest: "sha256:" + strings.Repeat("b", 64)}, PayloadSHA256: strings.Repeat("c", 64)}
	data, err := json.Marshal(candidate)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(p.Out, "candidate.json"), data, 0o600))
	result, err := recordBuildResult(p, r)
	require.NoError(t, err)
	require.Equal(t, "development", result.Purpose)
	require.Equal(t, "candidate", result.RequestedTarget)
	require.Equal(t, "candidate", result.CompletedTarget)
	require.Empty(t, result.Media)
	require.Equal(t, hashBytes(data), result.CandidateSHA256)
	require.Equal(t, candidate.Host.Manifest, result.HostManifest)
	require.Equal(t, candidate.PayloadSHA256, result.PayloadSHA256)
	require.Contains(t, result.Checks, "Application OCI identities")
	require.NotContains(t, strings.Join(result.Checks, " "), "Prepared")
	require.NotContains(t, strings.Join(result.Checks, " "), "media")
	require.Equal(t, "development-only; not release-qualified", result.Scope)
}

func TestCandidateRecordsExactForkArchive(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "artifacts")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "inputs"), 0o700))
	require.NoError(t, os.Mkdir(out, 0o700))
	archive := []byte("archived fork source")
	require.NoError(t, os.WriteFile(filepath.Join(root, "inputs/forgejo-source.tar"), archive, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(out, "payload.json"), []byte("payload"), 0o600))
	toolchain, err := json.MarshalIndent(build.ForgejoToolchain{CompilerImage: build.ForgejoCompilerImage, APKPackages: []string{"build-base-0.5-r4", "gcc-14.2.0-r6", "musl-dev-1.2.5-r10"}}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(out, "forgejo-toolchain.json"), append(toolchain, '\n'), 0o600))
	for path, content := range map[string]string{
		"artifacts/forgejo-context/forgejo-bin":                                          "fork binary",
		"artifacts/extension-context/extension/extension.json":                           "package manifest",
		"artifacts/extension-context/extension/backend":                                  "Soda backend",
		"artifacts/extension-context/extension/run":                                      "package runner",
		"artifacts/extension-context/extension/assets/entry.js":                          "browser asset",
		"work/host-context/rootfs/usr/share/containers/systemd/forgejo.container":        "fork service",
		"work/host-context/rootfs/usr/lib/systemd/system/soda-extension-install.service": "package service",
	} {
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o700))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}
	revision := strings.Repeat("a", 40)
	host := build.Image{Manifest: "sha256:" + strings.Repeat("b", 64), Architecture: "amd64"}
	require.NoError(t, writeContentInventory(filepath.Join(root, "work/host-context"), out))
	require.NoError(t, recordCandidate(out, "ghcr.io/example/sodaos", host, strings.Repeat("c", 64), revision, "x86_64"))
	var candidate deliver.Candidate
	require.NoError(t, build.ReadJSON(filepath.Join(out, "candidate.json"), &candidate))
	require.Equal(t, revision, candidate.ForgejoRevision)
	require.Equal(t, hashBytes(archive), candidate.ForgejoSourceSHA256)
	require.Equal(t, "x86_64", candidate.Architecture)
	require.Len(t, candidate.ContentSHA256, 8)
	require.Equal(t, hashBytes([]byte("fork binary")), candidate.ContentSHA256["forgejo:/usr/local/bin/gitea"])
	require.Equal(t, candidate.ContentSHA256["forgejo:/usr/local/bin/gitea"], candidate.ContentSHA256["extension:/usr/local/bin/gitea"])
	require.NoError(t, candidate.ForgejoToolchain.Validate())
	require.Error(t, recordCandidate(out, "ghcr.io/example/sodaos", build.Image{}, "", "", "x86_64"))
	require.NoError(t, os.Remove(filepath.Join(root, "inputs/forgejo-source.tar")))
	require.Error(t, recordCandidate(out, "ghcr.io/example/sodaos", host, "", revision, "x86_64"))
}

func TestMediaBoundaryStillStopsOnFailure(t *testing.T) {
	sentinel := errors.New("media prerequisite failed")
	p := build.Production{Next: func(string) error { return sentinel }}
	for _, r := range []Request{{}, {Development: true, Target: "media"}} {
		_, _, err := prepareBuildMedia(p, r)
		require.ErrorIs(t, err, sentinel)
		require.ErrorIs(t, finishBuildMedia(t.Context(), p, r, mediaTools{}, mediaLock{}, p.Next), sentinel)
	}
}

func TestLinkPreparedAssetsRunsNoCommands(t *testing.T) {
	root := t.TempDir()
	p := build.Production{Source: root, Native: filepath.Join(root, ".artifacts/native/x86_64"), Next: func(string) error {
		t.Fatal("asset linking emits no progress phases")
		return nil
	}, Execute: func(_, name string, _ ...string) error {
		t.Fatal("asset linking runs no commands: " + name)
		return nil
	}}
	require.NoError(t, linkPreparedAssets(p))
	for link, target := range map[string]string{
		".artifacts/forgejo-js":              "forgejo-js",
		".artifacts/browser-terminal/vendor": "terminal-assets",
	} {
		resolved, err := os.Readlink(filepath.Join(root, link))
		require.NoError(t, err)
		require.Equal(t, filepath.Join(p.Native, target), resolved)
	}
}

func TestRecallLogReasonSkipsMarkers(t *testing.T) {
	ring := newRecallLog()
	_, err := ring.Write([]byte("\nCOMMAND go\n"))
	require.NoError(t, err)
	_, err = ring.Write([]byte("go: first trouble\n$ go mod verify\n"))
	require.NoError(t, err)
	_, err = ring.Write([]byte("go: last word: permission denied"))
	require.NoError(t, err)
	require.Equal(t, "go: last word: permission denied", ring.reason())
}

func TestFailureReasonPrefersLocalErrorOverStaleRing(t *testing.T) {
	ring := newRecallLog()
	_, err := ring.Write([]byte("Writing manifest to image destination\n"))
	require.NoError(t, err)
	local := errors.New("decode request: json: cannot unmarshal string into Go struct field Trust.Keys of type []string")
	require.Equal(t, local.Error(), failureReason(ring, local))
	tool := errors.New("podman observation failed; retain attempt and inspect build.log: exit status 1")
	require.Equal(t, "Writing manifest to image destination", failureReason(ring, tool))
	require.False(t, isToolFailure(local))
	require.True(t, isToolFailure(tool))
	require.False(t, isToolFailure(nil))
}

func TestRecallLogAttachFlushesAdmission(t *testing.T) {
	ring := newRecallLog()
	_, err := ring.Write([]byte("early voice\n"))
	require.NoError(t, err)
	var file bytes.Buffer
	ring.attach(&file)
	_, err = ring.Write([]byte("later voice\n"))
	require.NoError(t, err)
	require.Contains(t, file.String(), "early voice\nlater voice\n")
	empty := newRecallLog()
	require.Equal(t, "", empty.reason())
}

func TestBuildCommandCaptureEnvironmentAndFailure(t *testing.T) {
	t.Setenv("SODA_RELEASE_NATIVE_OUT", "private-signing-sentinel")
	t.Setenv("SODA_FORGEJO_NATIVE_PAGES", "private-fixture-sentinel")
	t.Setenv("GITHUB_TOKEN", "private-token-sentinel")
	var log bytes.Buffer
	got, err := runBuildCommand(t.Context(), &log, nil, t.TempDir(), "sh", "-c", "printf captured-id; test -z \"$GITHUB_TOKEN$SODA_RELEASE_NATIVE_OUT$SODA_FORGEJO_NATIVE_PAGES\"")
	require.NoError(t, err)
	require.Equal(t, "captured-id", got)
	require.NotContains(t, log.String(), "printf")
	_, err = runBuildCommand(t.Context(), &log, nil, t.TempDir(), "sh", "-c", "exit 7")
	require.Equal(t, 7, build.BuildExitCode(err))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = runBuildCommand(ctx, &log, nil, t.TempDir(), "sh", "-c", "exit 0")
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, log.String(), "private-")
}
