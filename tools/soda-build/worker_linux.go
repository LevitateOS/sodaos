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
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
)

// workerConfig is installed by the operator, not emitted by a build. Paths name
// existing, separately owned task directories; this command does not create users,
// grant sudo, install trust or adopt an existing service/VM.
type workerConfig struct {
	Executable, Source, OutputParent                   string
	BuildHome, Runtime, Tools, MediaAuthorityDirectory string
}

const workerSource = "/run/soda-build-source"
const workerHome = "/var/lib/soda-build-worker"
const workerRuntime = "/run/soda-build-worker"
const workerTools = "/run/soda-build-tools"

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

func loadWorkerConfig(path, source, out string) (workerConfig, error) {
	var c workerConfig
	if os.Geteuid() != 0 {
		return c, errors.New("run the admitted controller as root; source commands run only as soda-build-worker")
	}
	if err := rd.PrivateFile(path); err != nil {
		return c, errors.New("root-owned restricted worker configuration required")
	}
	if err := rd.ReadJSON(path, &c); err != nil {
		return c, err
	}
	if c.Source != source || filepath.Dir(out) != c.OutputParent || !strings.HasPrefix(c.OutputParent, filepath.Join(source, ".artifacts/releases")+"/") {
		return c, errors.New("worker source/output differs from approved task paths")
	}
	if err := acceptance.TrustedExecutable(c.Executable); err != nil {
		return c, err
	}
	self, err := os.Executable()
	if err != nil {
		return c, err
	}
	a, err := nativebuild.HashFile(self)
	if err != nil {
		return c, err
	}
	b, err := nativebuild.HashFile(c.Executable)
	if err != nil || a != b {
		return c, errors.New("dispatcher differs from admitted worker executable")
	}
	for _, p := range []string{c.Source, c.OutputParent, c.BuildHome, c.Runtime, c.Tools, c.MediaAuthorityDirectory} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, ":\n\r\t %") {
			return c, errors.New("explicit safe worker path required")
		}
		resolved, e := filepath.EvalSymlinks(p)
		if e != nil || resolved != p {
			return c, errors.New("worker paths must exist without symlink traversal")
		}
	}
	return c, nil
}

// runBuildWorker delegates the existing producer, never another recipe. The
// service cannot see operator homes or real release custody; only its selected
// source view, caches and output parent are bound into that namespace.
func runBuildWorker(ctx context.Context, c workerConfig, r hostimage.Request, p *nativebuild.BuildProgress) (hostimage.Result, error) {
	var result hostimage.Result
	if err := p.Phase("P1-P8 / Isolated source-to-media worker"); err != nil {
		return result, err
	}
	rel, err := filepath.Rel(c.Source, r.Out)
	if err != nil {
		return result, err
	}
	out := filepath.Join(workerSource, rel)
	parentRel, err := filepath.Rel(c.Source, c.OutputParent)
	if err != nil {
		return result, err
	}
	name := "soda-build-" + filepath.Base(r.Out)
	w := acceptance.Worker{Name: name, User: "soda-build-worker", Executable: c.Executable, Directory: workerSource,
		ReadOnly:    []string{c.Source + ":" + workerSource, c.Tools + ":" + workerTools, c.MediaAuthorityDirectory + ":/run/soda-media-authority"},
		Writable:    []string{c.OutputParent + ":" + filepath.Join(workerSource, parentRel), c.BuildHome + ":" + workerHome, c.Runtime + ":" + workerRuntime},
		Environment: []string{"HOME=" + workerHome, "PATH=" + workerTools + "/go/bin:" + workerTools + "/bin:/usr/sbin:/usr/bin:/sbin:/bin", "XDG_RUNTIME_DIR=" + workerRuntime, "GOTOOLCHAIN=go1.26.7", "GOCACHE=" + workerHome + "/go-build", "GOMODCACHE=" + workerHome + "/go-mod", "BUN_INSTALL_CACHE_DIR=" + workerHome + "/bun-cache", "PLAYWRIGHT_BROWSERS_PATH=" + workerHome + "/browsers", "SODA_BUILD_START_NS=" + os.Getenv("SODA_BUILD_START_NS")},
		Arguments:   []string{"--worker-build", "--arch", r.Arch, "--out", out, "--repository-prefix", r.RepositoryPrefix, "--rootfs-base-url", r.RootfsBaseURL, "--media-authority", "/run/soda-media-authority/config.json"}}
	if err = w.Run(ctx, os.Stderr, os.Stderr); err != nil {
		return result, err
	}
	// This is a producer result, not qualification authority. P9 must independently
	// admit/snapshot the candidate rather than trusting this success document.
	if err = rd.ReadJSON(filepath.Join(r.Out, "evidence/build.json"), &result); err != nil {
		return result, err
	}
	if result.Revision != r.Revision || result.Architecture != r.Arch || result.Candidate != filepath.Join(out, "artifacts/candidate.json") || result.Media != filepath.Join(out, "artifacts/media/media.json") {
		return result, errors.New("worker result does not match this run")
	}
	result.Candidate = filepath.Join(r.Out, "artifacts/candidate.json")
	result.Media = filepath.Join(r.Out, "artifacts/media/media.json")
	return result, errors.Join(p.End(nil), p.EndPhase(nil))
}
