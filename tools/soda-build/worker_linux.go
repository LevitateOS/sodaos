package main

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/release/image"
)

// workerConfig is installed by the operator, not emitted by a build. Paths name
// existing, separately owned task directories; this command does not create users,
// grant sudo, install trust or adopt an existing service/VM.
type workerConfig struct {
	Executable, Source, OutputParent                   string
	BuildHome, Runtime, Tools, MediaAuthorityDirectory string
}

const (
	workerSource  = "/run/soda-build-source"
	workerHome    = "/var/lib/soda-build-worker"
	workerRuntime = "/run/soda-build-worker"
	workerTools   = "/run/soda-build-tools"
	// pinnedGoRoot is the fixed host GOROOT provisioned by
	// setup-soda-candidate.sh. Go 1.26 refuses a toolchain reached
	// through a bind mount, so it cannot live under workerTools.
	pinnedGoRoot = "/usr/local/lib/soda/pinned-go"
)

func buildWorkerIdentity() error {
	u, err := user.Lookup("soda-build-worker")
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil || uid == 0 || os.Geteuid() != uid {
		return errors.New("build stage requires the isolated soda-build-worker identity")
	}
	return nil
}

func validWorkerTaskPaths(c workerConfig, r image.Request) bool {
	return c.Source == r.Source && filepath.Dir(r.Out) == c.OutputParent && strings.HasPrefix(c.OutputParent, filepath.Join(r.Source, ".artifacts/releases")+"/")
}

func workerExecutableMatches(path string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	a, err := build.HashFile(self)
	if err != nil {
		return err
	}
	b, err := build.HashFile(path)
	if err != nil || a != b {
		return errors.New("dispatcher differs from admitted worker executable")
	}
	return nil
}

func validWorkerPath(p string) error {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, ":\n\r\t %") {
		return errors.New("explicit safe worker path required")
	}
	resolved, e := filepath.EvalSymlinks(p)
	if e != nil || resolved != p {
		return errors.New("worker paths must exist without symlink traversal")
	}
	return nil
}

func admitLoadedWorkerConfig(c workerConfig, r image.Request) error {
	if !validWorkerTaskPaths(c, r) {
		return errors.New("worker source/output differs from approved task paths")
	}
	if err := acceptance.TrustedExecutable(c.Executable); err != nil {
		return err
	}
	return workerExecutableMatches(c.Executable)
}

func workerConfigPaths(c workerConfig, r image.Request) []string {
	paths := []string{c.Source, c.OutputParent, c.BuildHome, c.Runtime, c.Tools}
	if r.WantsMedia() {
		paths = append(paths, c.MediaAuthorityDirectory)
	}
	return paths
}

func loadWorkerConfig(path string, r image.Request) (workerConfig, error) {
	var c workerConfig
	if os.Geteuid() != 0 {
		return c, errors.New("run the admitted controller as root; source commands run only as soda-build-worker")
	}
	if err := deliver.PrivateFile(path); err != nil {
		return c, errors.New("root-owned restricted worker configuration required")
	}
	if err := deliver.ReadJSON(path, &c); err != nil {
		return c, err
	}
	if err := admitLoadedWorkerConfig(c, r); err != nil {
		return c, err
	}
	for _, p := range workerConfigPaths(c, r) {
		if err := validWorkerPath(p); err != nil {
			return c, err
		}
	}
	return c, nil
}

// runBuildWorker delegates the existing producer, never another recipe. The
// service cannot see operator homes or real release custody; only its selected
// source view, caches and output parent are bound into that namespace.
func runBuildWorker(ctx context.Context, c workerConfig, r image.Request, p *build.BuildProgress) (image.Result, error) {
	var result image.Result
	boundary := "P1-P8 / Isolated source-to-media worker"
	if !r.WantsMedia() {
		boundary = "P1-P6 / Isolated development candidate worker"
	}
	if err := p.Phase(boundary); err != nil {
		return result, err
	}
	w, err := buildWorker(c, r)
	if err != nil {
		return result, err
	}
	if err = w.Run(ctx, os.Stderr, os.Stderr); err != nil {
		return result, err
	}
	// Producer output is not qualification authority. P9 independently admits it.
	if err = deliver.ReadJSON(filepath.Join(r.Out, "evidence/build.json"), &result); err != nil {
		return result, err
	}
	if err = validateWorkerResult(r, &result); err != nil {
		return result, err
	}
	return result, errors.Join(p.End(nil), p.EndPhase(nil))
}

func buildWorker(c workerConfig, r image.Request) (acceptance.Worker, error) {
	var w acceptance.Worker
	if err := r.ValidateTarget(); err != nil {
		return w, err
	}
	rel, err := filepath.Rel(c.Source, r.Out)
	if err != nil {
		return w, err
	}
	out := filepath.Join(workerSource, rel)
	parentRel, err := filepath.Rel(c.Source, c.OutputParent)
	if err != nil {
		return w, err
	}
	name := "soda-build-" + filepath.Base(r.Out)
	// Bun installs only from its HOME default cache; an explicit cache
	// directory makes it fail before touching the network.
	w = acceptance.Worker{
		Name: name, User: "soda-build-worker", Executable: c.Executable, Directory: workerSource,
		ReadOnly:    []string{c.Source + ":" + workerSource, c.Tools + ":" + workerTools},
		Writable:    []string{c.OutputParent + ":" + filepath.Join(workerSource, parentRel), c.BuildHome + ":" + workerHome, c.Runtime + ":" + workerRuntime},
		Environment: []string{"HOME=" + workerHome, "PATH=" + pinnedGoRoot + "/bin:" + workerTools + "/bin:/usr/sbin:/usr/bin:/sbin:/bin", "XDG_RUNTIME_DIR=" + workerRuntime, "GOTOOLCHAIN=go1.26.7", "GOCACHE=" + workerHome + "/go-build", "GOMODCACHE=" + workerHome + "/go-mod", "PLAYWRIGHT_BROWSERS_PATH=" + workerHome + "/browsers", "SODA_BUILD_START_NS=" + os.Getenv("SODA_BUILD_START_NS")},
		Arguments:   []string{"--worker-build", "--arch", r.Arch, "--out", out, "--repository-prefix", r.RepositoryPrefix},
	}
	if r.Development {
		w.Arguments = append(w.Arguments, "--development", "--target", r.Target)
	}
	if r.MediaCompression != "" {
		w.Arguments = append(w.Arguments, "--media-compression", r.MediaCompression)
	}
	if r.WantsMedia() {
		w.ReadOnly = append(w.ReadOnly, c.MediaAuthorityDirectory+":/run/soda-media-authority")
		w.Arguments = append(w.Arguments, "--rootfs-base-url", r.RootfsBaseURL, "--media-authority", "/run/soda-media-authority/config.json")
	}
	return w, nil
}

func workerResultMatches(r image.Request, result *image.Result, out, media, completed string) bool {
	return result.Revision == r.Revision && result.Architecture == r.Arch && result.Candidate == filepath.Join(out, "artifacts/candidate.json") && result.Media == media && result.Purpose == r.Purpose() && result.RequestedTarget == r.RequestedTarget() && result.CompletedTarget == completed && result.MediaCompression == r.MediaCompression
}

func validateWorkerResult(r image.Request, result *image.Result) error {
	rel, err := filepath.Rel(r.Source, r.Out)
	if err != nil {
		return err
	}
	out := filepath.Join(workerSource, rel)
	media, completed := "", "candidate"
	if r.WantsMedia() {
		media, completed = filepath.Join(out, "artifacts/media/media.json"), "media"
	}
	if !workerResultMatches(r, result, out, media, completed) {
		return errors.New("worker result does not match this run")
	}
	candidate := filepath.Join(r.Out, "artifacts/candidate.json")
	hash, err := build.HashFile(candidate)
	if err != nil || hash != result.CandidateSHA256 {
		return errors.New("worker candidate receipt differs")
	}
	result.Candidate = candidate
	if media != "" {
		result.Media = filepath.Join(r.Out, "artifacts/media/media.json")
	}
	return nil
}
