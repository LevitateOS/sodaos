//go:build linux

package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/image"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestDevelopmentWorkerInputsAndCompletion(t *testing.T) {
	c := workerConfig{Source: "/source", OutputParent: "/source/.artifacts/releases/isolated", Tools: "/tools", MediaAuthorityDirectory: "/authority"}
	for _, target := range []string{"candidate", "media", ""} {
		r := image.Request{Source: c.Source, Out: c.OutputParent + "/test", Development: target != "", Target: target}
		if r.WantsMedia() {
			r.RootfsBaseURL = "https://example.invalid"
		}
		w, err := buildWorker(c, r)
		require.NoError(t, err)
		require.Equal(t, "soda-build-worker", w.User)
		if target == "candidate" {
			require.NotContains(t, strings.Join(w.ReadOnly, " "), "authority")
			require.NotContains(t, w.Arguments, "--rootfs-base-url")
			require.NotContains(t, w.Arguments, "--media-authority")
		} else {
			require.Contains(t, w.ReadOnly, "/authority:/run/soda-media-authority")
			require.Contains(t, w.Arguments, "--rootfs-base-url")
		}
		if target == "" {
			require.NotContains(t, w.Arguments, "--development")
			require.Equal(t, 2, build.BuildExitCode(interrupted(2)))
		} else {
			require.Contains(t, w.Arguments, "--development")
			require.Contains(t, w.Arguments, target)
		}
	}
}

func TestWorkerForwardsControllerLiveInputs(t *testing.T) {
	c := workerConfig{Source: "/source", OutputParent: "/source/.artifacts/releases/isolated", Tools: "/tools", MediaAuthorityDirectory: "/authority"}
	r := image.Request{Source: c.Source, Out: c.OutputParent + "/test", Development: true, Target: "candidate", LiveInputs: "/run/soda-build-source/.artifacts/releases/isolated/soda-live-inputs-test.json"}
	w, err := buildWorker(c, r)
	require.NoError(t, err)
	require.Contains(t, w.Arguments, "--live-inputs")
	require.Contains(t, w.Arguments, r.LiveInputs)
	r.LiveInputs = ""
	w, err = buildWorker(c, r)
	require.NoError(t, err)
	require.NotContains(t, w.Arguments, "--live-inputs")
}

func TestLiveInputsAdmissionBoundary(t *testing.T) {
	worker := buildFlags{WorkerBuild: true, Request: image.Request{Out: "/o", Arch: "x86_64"}}
	require.ErrorContains(t, admitWorkerBuild(worker), "requires controller-resolved live inputs")
	parent := buildFlags{WorkerConfig: "/cfg", Request: image.Request{Out: "/o", Arch: "x86_64", LiveInputs: "/inputs.json"}}
	require.ErrorContains(t, admitParentDispatch(parent), "operator selection refused")
}

func TestWorkerEnvUsesDefaultBunCache(t *testing.T) {
	c := workerConfig{Source: "/source", OutputParent: "/source/.artifacts/releases/isolated", Tools: "/tools", MediaAuthorityDirectory: "/authority"}
	r := image.Request{Source: c.Source, Out: c.OutputParent + "/test", Development: true, Target: "media", RootfsBaseURL: "https://example.invalid"}
	w, err := buildWorker(c, r)
	require.NoError(t, err)
	for _, env := range w.Environment {
		require.NotContains(t, env, "BUN_INSTALL_CACHE_DIR")
	}
	require.Contains(t, w.Environment, "HOME="+workerHome)
}

func TestWorkerPathLeadsWithFixedGoRoot(t *testing.T) {
	c := workerConfig{Source: "/source", OutputParent: "/source/.artifacts/releases/isolated", Tools: "/tools", MediaAuthorityDirectory: "/authority"}
	r := image.Request{Source: c.Source, Out: c.OutputParent + "/test", Development: true, Target: "media", RootfsBaseURL: "https://example.invalid"}
	w, err := buildWorker(c, r)
	require.NoError(t, err)
	for _, env := range w.Environment {
		if strings.HasPrefix(env, "PATH=") {
			require.True(t, strings.HasPrefix(env, "PATH="+pinnedGoRoot+"/bin:"), env)
			require.NotContains(t, env, "soda-build-tools/go")
			return
		}
	}
	t.Fatal("worker PATH missing")
}

func TestFastMediaWorkerSelection(t *testing.T) {
	c := workerConfig{Source: "/source", OutputParent: "/source/.artifacts/releases/isolated", Tools: "/tools", MediaAuthorityDirectory: "/authority"}
	r := image.Request{Source: c.Source, Out: c.OutputParent + "/test", Development: true, Target: "media", MediaCompression: "fast", RootfsBaseURL: "https://example.invalid"}
	w, err := buildWorker(c, r)
	require.NoError(t, err)
	require.Contains(t, strings.Join(w.Arguments, " "), "--media-compression fast")
	r.Development = false
	r.Target = ""
	_, err = buildWorker(c, r)
	require.Error(t, err)
}

func TestWorkerResultBindsTargetAndCandidate(t *testing.T) {
	source := t.TempDir()
	out := filepath.Join(source, ".artifacts/releases/isolated/test")
	require.NoError(t, os.MkdirAll(filepath.Join(out, "artifacts"), 0o700))
	candidate := filepath.Join(out, "artifacts/candidate.json")
	require.NoError(t, os.WriteFile(candidate, []byte("fixture"), 0o600))
	hash, err := build.HashFile(candidate)
	require.NoError(t, err)
	r := image.Request{Source: source, Out: out, Revision: strings.Repeat("a", 40), Arch: "x86_64", Development: true, Target: "candidate"}
	original := image.Result{Revision: r.Revision, Architecture: r.Arch, Candidate: workerSource + "/.artifacts/releases/isolated/test/artifacts/candidate.json", CandidateSHA256: hash, Purpose: "development", RequestedTarget: "candidate", CompletedTarget: "candidate"}
	for _, mode := range []string{"valid", "media", "purpose", "requested", "completed", "hash", "compression"} {
		result := original
		switch mode {
		case "media":
			result.Media = "unexpected-media.json"
		case "purpose":
			result.Purpose = "production"
		case "requested":
			result.RequestedTarget = "release"
		case "completed":
			result.CompletedTarget = "media"
		case "compression":
			result.MediaCompression = "fast"
		case "hash":
			result.CandidateSHA256 = strings.Repeat("b", 64)
		}
		err := validateWorkerResult(r, &result)
		if mode == "valid" {
			require.NoError(t, err)
			require.Equal(t, candidate, result.Candidate)
			require.Empty(t, result.Media)
		} else {
			require.Error(t, err, mode)
		}
	}
}

func TestAdmitInternalDispatchBindsAuthority(t *testing.T) {
	// The worker stage refuses operator configuration; the controller
	// refuses anything but development dispatch. Failures land before
	// identity is consulted.
	for _, f := range []buildFlags{
		{WorkerBuild: true, WorkerConfig: "/restricted/worker.json"},
		{WorkerBuild: true},
		{WorkerConfig: ""},
		{WorkerConfig: "/cfg", Request: image.Request{MediaAuthority: "x"}},
		{WorkerConfig: "/cfg", Request: image.Request{LiveInputs: "x"}},
		{WorkerConfig: "/cfg"},
	} {
		require.Error(t, admitBuildDispatch(f), "%+v", f)
	}
	for _, f := range []buildFlags{
		{WorkerConfig: "/cfg", Request: image.Request{Development: true}},
	} {
		require.NoError(t, admitBuildDispatch(f), "%+v", f)
	}
}

func TestControllerSubreaperReapsLeaderFirstDescendant(t *testing.T) {
	// Isolate the process-wide subreaper setting from the parent test runner.
	if os.Getenv("SODA_BUILD_REAPER_FIXTURE") != "1" {
		self, err := os.Executable()
		require.NoError(t, err)
		cmd := exec.CommandContext(t.Context(), self, "-test.run=^TestControllerSubreaperReapsLeaderFirstDescendant$")
		cmd.Env = append(os.Environ(), "SODA_BUILD_REAPER_FIXTURE=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		return
	}
	require.NoError(t, unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0))
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	cmd := exec.Command("sh", "-c", `sleep 60 & printf '%s' "$!" > "$1"`, "fixture", pidFile)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	process, err := acceptance.StartCommand(t.Context(), cmd)
	require.NoError(t, err)
	require.NoError(t, process.Wait(t.Context()))
	data, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	// ESRCH, not an orphaned zombie waiting for init. Signal 0 only observes.
	require.ErrorIs(t, unix.Kill(pid, 0), unix.ESRCH)
}
