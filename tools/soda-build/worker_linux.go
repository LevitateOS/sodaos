package main

import (
	"context"
	"errors"
	"fmt"
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
	Executable, Source, ForgejoSource, OutputParent    string
	BuildHome, Runtime, Tools, MediaAuthorityDirectory string
}

const (
	workerSource  = "/run/soda-build-source"
	workerHome    = "/var/lib/soda-build-worker"
	workerRuntime = "/run/soda-build-worker"
	workerTools   = "/run/soda-build-tools"
	workerForgejo = "/run/soda-build-forgejo-source"
	// pinnedGoRoot is the fixed host GOROOT provisioned by
	// setup-soda-candidate.sh. It carries lib_t there so the worker
	// domain can execute it, so it cannot live under workerTools.
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
	return c.Source == r.Source && c.ForgejoSource == r.ForgejoSource && filepath.Dir(r.Out) == c.OutputParent && strings.HasPrefix(c.OutputParent, filepath.Join(r.Source, ".artifacts/releases")+"/")
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
	paths := []string{c.Source, c.ForgejoSource, c.OutputParent, c.BuildHome, c.Runtime, c.Tools}
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

// workerRuntimeIDs resolves the isolated identity that owns per-attempt
// runtime state. The controller runs as root and hands each attempt
// directory to the worker; nothing is ever shared or re-owned mid-build.
func workerRuntimeIDs() (uid, gid int, err error) {
	u, err := user.Lookup("soda-build-worker")
	if err != nil {
		return 0, 0, err
	}
	uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err = strconv.Atoi(u.Gid)
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}

// claimAttemptRuntime creates a fresh unique runtime directory for one
// dispatch and hands it to the worker identity. Attempts never share
// runtime state, so no attempt prepares or deletes another's sockets,
// locks, or namespaces; stale locks from a killed run cannot poison the
// next one because the next one never reuses the directory. The directory
// inherits the parent's SELinux type. Release removes only this directory.
func claimAttemptRuntime(parent, out string, uid, gid int) (dir string, release func() error, err error) {
	leaf := filepath.Base(out)
	if leaf == "" || leaf == "." || leaf == ".." {
		return "", nil, errors.New("explicit safe output leaf required")
	}
	if st, err := os.Stat(parent); err != nil || !st.IsDir() {
		return "", nil, errors.New("worker runtime directory missing; rerun the setup script")
	}
	dir, err = os.MkdirTemp(parent, "soda-build-"+leaf+"-")
	if err != nil {
		return "", nil, fmt.Errorf("cannot claim attempt runtime: %w", err)
	}
	if err := os.Chown(dir, uid, gid); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("cannot hand attempt runtime to the worker: %w", err)
	}
	return dir, func() error { return releaseAttemptRuntime(parent, dir) }, nil
}

// releaseAttemptRuntime removes one claimed attempt directory. It refuses
// anything that is not a direct child directory of the runtime parent,
// never the parent itself, so a confused caller cannot delete shared state.
func releaseAttemptRuntime(parent, dir string) error {
	rel, err := filepath.Rel(parent, dir)
	if err != nil || rel == "." || rel == ".." || strings.ContainsRune(rel, '/') || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing to release %q outside runtime %q", dir, parent)
	}
	st, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("cannot release attempt runtime: %w", err)
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to release non-directory %q", dir)
	}
	return os.RemoveAll(dir)
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
	if err := resolveWorkerLiveInputs(ctx, c, &r); err != nil {
		return result, err
	}
	uid, gid, err := workerRuntimeIDs()
	if err != nil {
		return result, err
	}
	attempt, release, err := claimAttemptRuntime(c.Runtime, r.Out, uid, gid)
	if err != nil {
		return result, err
	}
	defer func() {
		if err := release(); err != nil {
			fmt.Fprintln(os.Stderr, "warning: cannot release attempt runtime:", err)
		}
	}()
	c.Runtime = attempt
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

// resolveWorkerLiveInputs resolves the live inputs on the controller,
// where outbound HTTPS is admitted, and records them beside the attempt
// output for the isolated worker. The worker still consumes only its own
// attempt's file and validates it like a live resolution; digest-pinned
// pulls verify content downstream. Live inputs stay controller-resolved
// even though the worker policy admits one narrow exception: P2 stages
// pinned, checksummed inputs over HTTPS from inside the worker (see
// scripts/selinux/soda-build-worker.te), so the worker is no longer
// fully denied outbound HTTPS. Keep this comment and the policy rule in
// sync; if P2 moves to the controller, remove the http_port_t grant.
func resolveWorkerLiveInputs(ctx context.Context, c workerConfig, r *image.Request) error {
	coreOS, err := build.ResolveCoreOS(ctx)
	if err != nil {
		return err
	}
	name := "soda-live-inputs-" + filepath.Base(r.Out) + ".json"
	tailnet, err := build.ResolveTailnetInputs(ctx, r.Arch)
	if err != nil {
		return err
	}
	if err := build.WriteLiveInputs(filepath.Join(c.OutputParent, name), build.LiveInputs{CoreOS: coreOS, Tailnet: tailnet}); err != nil {
		return err
	}
	parentRel, err := filepath.Rel(c.Source, c.OutputParent)
	if err != nil {
		return err
	}
	r.LiveInputs = filepath.Join(workerSource, parentRel, name)
	return nil
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
		ReadOnly:    []string{c.Source + ":" + workerSource, c.ForgejoSource + ":" + workerForgejo, c.Tools + ":" + workerTools},
		Writable:    []string{c.OutputParent + ":" + filepath.Join(workerSource, parentRel), c.BuildHome + ":" + workerHome, c.Runtime + ":" + workerRuntime},
		Environment: []string{"HOME=" + workerHome, "PATH=" + pinnedGoRoot + "/bin:" + workerTools + "/bin:/usr/sbin:/usr/bin:/sbin:/bin", "XDG_RUNTIME_DIR=" + workerRuntime, "GOTOOLCHAIN=go1.26.7", "GOCACHE=" + workerHome + "/go-build", "GOMODCACHE=" + workerHome + "/go-mod", "PLAYWRIGHT_BROWSERS_PATH=" + workerHome + "/browsers", "SODA_BUILD_START_NS=" + os.Getenv("SODA_BUILD_START_NS")},
		Arguments:   []string{"--worker-build", "--arch", r.Arch, "--out", out, "--repository-prefix", r.RepositoryPrefix, "--forgejo-source", workerForgejo, "--forgejo-revision", r.ForgejoRevision},
	}
	if r.Development {
		w.Arguments = append(w.Arguments, "--development", "--target", r.Target)
	}
	if r.MediaCompression != "" {
		w.Arguments = append(w.Arguments, "--media-compression", r.MediaCompression)
	}
	if r.LiveInputs != "" {
		w.Arguments = append(w.Arguments, "--live-inputs", r.LiveInputs)
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
