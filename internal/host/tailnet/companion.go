// Package tailnet runs the Tailnet companion container for a project. It
// does not own Tailnet policy (domain package `tailnet`) or HTTP admission.
package tailnet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	domain "github.com/levitateos/sodaos/internal/tailnet"
)

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}

type Companion struct {
	Exec         Executor
	Tailnet      *domain.Control
	Image        string // sha256:... companion image
	EnabledCheck func(ctx context.Context, project, cid string) (bool, error)
}

var (
	projectID   = regexp.MustCompile(`^p[0-9a-f]{24}$`)
	containerID = regexp.MustCompile(`^[0-9a-f]{64}$`)
	imageID     = regexp.MustCompile(`^(?:sha256:)?[0-9a-f]{64}$`)
)

func isHostNative(exec Executor) bool {
	_, ok := exec.(interface{ HostNative() })
	return ok
}

func (c *Companion) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return c.Exec.Run(ctx, in, "/usr/bin/podman", args...)
}

const (
	runtimeRoot      = "/run/soda-tailnet"
	companionInspect = `{"id":{{json .ID}},"image":{{json .Image}},"command":{{json .Config.CreateCommand}},"running":{{json .State.Running}},"pid":{{json .State.Pid}},"started":{{json .State.StartedAt}},"execs":{{json .ExecIDs}}}`
)

type companion struct {
	ID, Image string
	Command   []string
	Running   bool
	PID       int
	Started   string
	Execs     []string
}

func (c *Companion) runtimeCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	// The ordinary host executor predates secret-bearing native protocols. Never
	// collect its stderr for the companion, and never return native error text.
	if isHostNative(c.Exec) {
		return domain.RunNative(ctx, command, args...)
	}
	b, e := c.Exec.Run(ctx, nil, command, args...)
	if e != nil || len(b) > 65536 {
		return nil, domain.ErrUnconfirmed
	}
	return b, nil
}

func (c *Companion) runtimePodman(ctx context.Context, args ...string) ([]byte, error) {
	return c.runtimeCommand(ctx, "/usr/bin/podman", append([]string{"--remote=false"}, args...)...)
}

func companionIdentityMatches(out companion, run projectRun, image, id string) bool {
	return containerID.MatchString(id) && out.ID == id && strings.TrimPrefix(out.Image, "sha256:") == strings.TrimPrefix(image, "sha256:")
}

func companionCommandMatches(out companion, args []string) bool {
	return len(out.Command) == len(args)+1 && (out.Command[0] == "/usr/bin/podman" || out.Command[0] == "podman") && slices.Equal(out.Command[1:], args)
}

func companionExecsValid(execs []string) error {
	if len(execs) > 16 {
		return domain.ErrUnavailable
	}
	for _, id := range execs {
		if !containerID.MatchString(id) {
			return domain.ErrUnavailable
		}
	}
	return nil
}

func validateCompanionRecord(out companion, run projectRun, image, id string) error {
	args, e := companionCreateArgs(run, image)
	if e != nil {
		return e
	}
	if !companionIdentityMatches(out, run, image, id) {
		return domain.ErrConflict
	}
	if out.Running {
		started, e := time.Parse(time.RFC3339Nano, out.Started)
		if e != nil || started.IsZero() || out.PID <= 1 {
			return domain.ErrConflict
		}
	}
	// CreateCommand is retained by Podman. Admit exactly our immutable recipe,
	// allowing only the native argv[0] spelling, never labels/name alone.
	if !companionCommandMatches(out, args) {
		return domain.ErrConflict
	}
	return companionExecsValid(out.Execs)
}

func matchCompanionNamespaces(out companion, run projectRun) error {
	if !out.Running {
		return nil
	}
	identity, e := processRunIdentity(out.PID, os.ReadFile, os.Readlink)
	if e != nil || identity.UserNS != run.UserNS || identity.NetNS != run.NetNS || identity.UID != run.UID || identity.GID != run.GID {
		return domain.ErrConflict
	}
	return nil
}

func (c *Companion) inspectCompanion(ctx context.Context, run projectRun) (companion, error) {
	var out companion
	id, e := readCompanionID(runtimeRoot, run, 0, 0)
	if e != nil {
		return out, e
	}
	b, e := c.runtimePodman(ctx, "inspect", "--format", companionInspect, id)
	if e != nil || strictjson.Decode(bytes.NewReader(b), &out) != nil {
		return out, domain.ErrConflict
	}
	if e = validateCompanionRecord(out, run, c.Image, id); e != nil {
		return out, e
	}
	if e = matchCompanionNamespaces(out, run); e != nil {
		return out, e
	}
	return out, nil
}

// Podman's generated ResolvConfPath is absent before first start and does not
// describe every explicit mount. Enrollment/observations use the actual inode.
// Logout/stop must still reach an exact owned old companion after the parent has
// restarted and Podman has generated a new resolver inode for that new run.
func companionResolver(run projectRun, pid int, stat func(string) (os.FileInfo, error)) error {
	original, e := stat(run.Resolver)
	actual, other := stat("/proc/" + strconv.Itoa(pid) + "/root/etc/resolv.conf")
	if e != nil || other != nil || !original.Mode().IsRegular() || !os.SameFile(original, actual) {
		return domain.ErrConflict
	}
	return nil
}

func companionStillRunning(before, after companion, err, other error) bool {
	return err == nil && other == nil && after.ID == before.ID && after.PID == before.PID && after.Started == before.Started && after.Running
}

func (c *Companion) companionCLI(ctx context.Context, run projectRun, args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, domain.ErrInvalid
	}
	rec, e := c.inspectCompanion(ctx, run)
	if e != nil || !rec.Running {
		return nil, domain.ErrUnavailable
	}
	if args[0] != "logout" {
		if e = companionResolver(run, rec.PID, os.Stat); e != nil {
			return nil, e
		}
	}
	command := []string{"exec", "--user=0:0", rec.ID, "/usr/local/bin/tailscale", "--socket=/run/tailscale/tailscaled.sock"}
	b, e := c.runtimePodman(ctx, append(command, args...)...)
	after, other := c.inspectCompanion(ctx, run)
	if !companionStillRunning(rec, after, e, other) {
		return nil, domain.ErrUnconfirmed
	}
	if args[0] != "logout" {
		if e = companionResolver(run, after.PID, os.Stat); e != nil {
			return nil, domain.ErrUnconfirmed
		}
	}
	return b, nil
}

func (f *runFiles) current() (projectRun, error) {
	var run projectRun
	file, e := f.root.OpenFile("current.json", os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return run, e
	}
	defer file.Close()
	info, e := file.Stat()
	if e != nil || !runtimeFile(info, f.uid, f.gid, 0o600) || info.Size() > 8192 {
		return run, domain.ErrUnavailable
	}
	if strictjson.Decode(file, &run) != nil {
		return run, domain.ErrUnavailable
	}
	// The recipe validates all path-bearing identity fields. The caller separately
	// checks this record's project, parent CID and native namespace incarnation.
	if _, e = companionCreateArgs(run, "sha256:"+strings.Repeat("0", 64)); e != nil {
		return run, e
	}
	return run, nil
}

func (f *runFiles) saveCurrent(run projectRun) error {
	name := "current-" + run.Target.Run + ".json"
	b, e := json.Marshal(run)
	if e != nil {
		return domain.ErrUnavailable
	}
	file, e := f.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if e != nil {
		return domain.ErrConflict
	}
	defer file.Close()
	if _, e = file.Write(b); e != nil {
		return domain.ErrUnconfirmed
	}
	if file.Sync() != nil || f.root.Rename(name, "current.json") != nil {
		return domain.ErrUnconfirmed
	}
	dir, e := f.root.Open(".")
	if e != nil {
		return domain.ErrUnconfirmed
	}
	defer dir.Close()
	if dir.Sync() != nil {
		return domain.ErrUnconfirmed
	}
	return nil
}

func (c *Companion) consumeRunKey(ctx context.Context, run projectRun, root *os.Root, key string) error {
	file, e := writeRunKey(root, run, key)
	if e != nil {
		return e
	}
	defer file.Close()
	_, e = c.companionCLI(ctx, run, "up", "--json", "--timeout=5s", "--auth-key=file:/run/soda-enrollment/key", "--hostname=soda-"+run.Target.Project, "--accept-dns=true", "--accept-routes=false", "--ssh=false", "--advertise-exit-node=false")
	// A cancelled Podman observer does not cancel the native exec. The CLI has its
	// own five-second deadline. Do not retire its input until all execs have ended.
	finish, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		rec, other := c.inspectCompanion(finish, run)
		if other == nil && len(rec.Execs) == 0 {
			if retireRunKey(root, file) != nil {
				return domain.ErrUnconfirmed
			}
			return e
		}
		select {
		case <-finish.Done():
			return domain.ErrUnconfirmed
		case <-tick.C:
		}
	}
}

// projectTailnetEnabled reports saved policy intent without requiring
// companion-specific runtime readiness. Missing policy means Off; malformed or
// ambiguous configured state fails safely instead of enrolling with defaults.
// Enabled operations still go through the full exact-incarnation validation
// below, so this never bypasses authorization or native identity checks.
func (c *Companion) projectTailnetEnabled(ctx context.Context, project, cid string) (bool, error) {
	if c.EnabledCheck != nil {
		return c.EnabledCheck(ctx, project, cid)
	}
	view, e := c.Tailnet.Project(ctx, domain.ProjectRequest{Project: project, Action: "inspect"}, cid)
	if e != nil {
		return false, e
	}
	return view.Enabled, nil
}

// preparationError tags the actual failure stage. Messages are fixed labels;
// native/provider output never enters the chain (see runtimeCommand/RunNative
// and the key/status sanitizers), so wrapping preserves errors.Is safely.
func preparationError(stage string, e error) error {
	if e == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", stage, e)
}

func (c *Companion) shouldStartTailnet(ctx context.Context, id string) (bool, error) {
	if c.Tailnet == nil || c.Image == "" {
		return false, nil
	}
	// Off/unconfigured policy is a clean no-op before companion-specific
	// runtime readiness. A resolvable container with definitively disabled or
	// missing policy never waits on the bounded run validation below.
	// Malformed policy fails safely; an unresolvable container falls through
	// to that loop so enabled startup races still apply.
	if cid, cidErr := c.projectContainer(ctx, id, false); cidErr == nil {
		enabled, pErr := c.projectTailnetEnabled(ctx, id, cid)
		if pErr != nil {
			return false, preparationError("Tailnet policy unconfirmed", pErr)
		}
		if !enabled {
			return false, nil
		}
	}
	return true, nil
}

func (c *Companion) waitProjectRuntime(ctx context.Context, id string) (projectRun, error) {
	var run projectRun
	readyParent, cancelParent := context.WithTimeout(ctx, 10*time.Second)
	parentTick := time.NewTicker(100 * time.Millisecond)
	defer parentTick.Stop()
	defer cancelParent()
	var e error
	for {
		run, e = c.projectRun(readyParent, id)
		if e == nil {
			return run, nil
		}
		select {
		case <-readyParent.Done():
			return projectRun{}, preparationError("project runtime not ready", e)
		case <-parentTick.C:
		}
	}
}

func (c *Companion) admitTailnetRun(ctx context.Context, id string) (projectRun, domain.RunBinding, error) {
	run, err := c.waitProjectRuntime(ctx, id)
	if err != nil {
		return projectRun{}, domain.RunBinding{}, err
	}
	binding, err := c.Tailnet.RunBinding(ctx, run.Target)
	if err != nil {
		return projectRun{}, domain.RunBinding{}, preparationError("Tailnet policy unconfirmed", err)
	}
	return run, binding, nil
}

func isCompanionRunFresh(currentErr error, previous, run projectRun) bool {
	if errors.Is(currentErr, os.ErrNotExist) {
		return true
	}
	return currentErr == nil && previous.Target.Run != run.Target.Run
}

func (c *Companion) retirePreviousCompanion(ctx context.Context, id string, run, previous projectRun) error {
	if previous.Target.Run == "" {
		return nil
	}
	if previous.Target.Project != id || previous.Target.Container != run.Target.Container {
		return preparationError("project runtime changed", domain.ErrConflict)
	}
	if err := c.stopTailnetRun(ctx, previous); err != nil {
		return preparationError("companion stop unconfirmed", err)
	}
	return nil
}

func (c *Companion) reconcilePreviousRun(ctx context.Context, id string, run, previous projectRun, currentErr error) (bool, error) {
	if currentErr != nil && !errors.Is(currentErr, os.ErrNotExist) {
		return false, preparationError("companion runtime unconfirmed", currentErr)
	}
	fresh := isCompanionRunFresh(currentErr, previous, run)
	if !fresh && previous != run {
		return false, preparationError("project runtime changed", domain.ErrConflict)
	}
	if fresh {
		if err := c.retirePreviousCompanion(ctx, id, run, previous); err != nil {
			return false, err
		}
	}
	return fresh, nil
}

func (c *Companion) prepareCompanionState(ctx context.Context, id string, run projectRun) (*runFiles, *os.Root, bool, error) {
	f, err := openRuntimeProject(ctx, runtimeRoot, id)
	if err != nil {
		return nil, nil, false, preparationError("companion runtime unconfirmed", err)
	}
	previous, currentErr := f.current()
	fresh, err := c.reconcilePreviousRun(ctx, id, run, previous, currentErr)
	if err != nil {
		f.Close()
		return nil, nil, false, err
	}
	if err := validateRunResolver(run, os.ReadFile, os.Stat, fresh); err != nil {
		f.Close()
		return nil, nil, false, preparationError("project runtime not ready", err)
	}
	root, err := f.prepare(run, fresh)
	if err != nil {
		f.Close()
		return nil, nil, false, preparationError("companion runtime unconfirmed", err)
	}
	return f, root, fresh, nil
}

func (c *Companion) createFreshCompanion(ctx context.Context, f *runFiles, root *os.Root, run projectRun) error {
	if err := f.saveCurrent(run); err != nil {
		return preparationError("companion runtime unconfirmed", err)
	}
	if err := c.recheckProjectRun(ctx, run); err != nil {
		return preparationError("project runtime changed", err)
	}
	args, err := companionCreateArgs(run, c.Image)
	if err != nil {
		return preparationError("companion runtime unconfirmed", err)
	}
	created, err := c.runtimeCommand(ctx, "/usr/bin/podman", args...)
	if err != nil {
		return preparationError("companion startup unconfirmed", err)
	}
	id := strings.TrimSpace(string(created))
	if err := writeCompanionID(root, id); err != nil {
		return preparationError("companion runtime unconfirmed", err)
	}
	return nil
}

func (c *Companion) startCompanionIfStopped(ctx context.Context, run projectRun) error {
	rec, err := c.inspectCompanion(ctx, run)
	if err != nil {
		return preparationError("companion startup unconfirmed", err)
	}
	if !rec.Running {
		if err := c.recheckProjectRun(ctx, run); err != nil {
			return preparationError("project runtime changed", err)
		}
		if _, err := c.runtimePodman(ctx, "start", rec.ID); err != nil {
			return preparationError("companion startup unconfirmed", err)
		}
	}
	return nil
}

func (c *Companion) activateCompanionContainer(ctx context.Context, f *runFiles, root *os.Root, run projectRun, fresh bool) error {
	if fresh {
		if err := c.createFreshCompanion(ctx, f, root, run); err != nil {
			return err
		}
	}
	return c.startCompanionIfStopped(ctx, run)
}

func (c *Companion) waitCompanionNode(ctx context.Context, run projectRun) (bool, error) {
	ready, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		b, other := c.companionCLI(ready, run, "status", "--json", "--peers=false")
		if other == nil {
			hasNode, err := domain.ProjectHasNode(b)
			if err != nil {
				return false, preparationError("companion status unavailable", err)
			}
			return hasNode, nil
		}
		select {
		case <-ready.Done():
			return false, preparationError("companion status unavailable", domain.ErrUnavailable)
		case <-tick.C:
		}
	}
}

func (c *Companion) enrollCompanion(ctx context.Context, run projectRun, root *os.Root) error {
	rec, err := c.inspectCompanion(ctx, run)
	if err != nil || len(rec.Execs) != 0 {
		return preparationError("enrollment unconfirmed", domain.ErrUnconfirmed)
	}
	if err := retirePendingRunKey(root, run); err != nil {
		return preparationError("enrollment unconfirmed", err)
	}
	recheck := func(ctx context.Context) error { return c.recheckProjectRun(ctx, run) }
	consume := func(ctx context.Context, key string) error { return c.consumeRunKey(ctx, run, root, key) }
	if err := c.Tailnet.EnrollRun(ctx, run.Target, recheck, consume); err != nil {
		return preparationError("enrollment unconfirmed", err)
	}
	return nil
}

func (c *Companion) ensureCompanionEnrolled(ctx context.Context, run projectRun, root *os.Root, admission bool) error {
	hasNode, err := c.waitCompanionNode(ctx, run)
	if err != nil {
		return err
	}
	if hasNode || !admission {
		return nil
	}
	return c.enrollCompanion(ctx, run, root)
}

func (c *Companion) finalizeCompanionRun(ctx context.Context, run projectRun) (string, error) {
	if err := c.recheckProjectRun(ctx, run); err != nil {
		return "", preparationError("project runtime changed", err)
	}
	rec, err := c.inspectCompanion(ctx, run)
	if err != nil {
		return "", preparationError("companion runtime unconfirmed", err)
	}
	return rec.ID, nil
}

// StartTailnet prepares one exact run. Only systemd/native explicit activation
// calls it; HTTP reads and browser focus never enter this path.
func (c *Companion) StartTailnet(ctx context.Context, id string) (string, error) {
	start, err := c.shouldStartTailnet(ctx, id)
	if err != nil || !start {
		return "", err
	}
	run, binding, err := c.admitTailnetRun(ctx, id)
	if err != nil || !binding.Enabled {
		return "", err
	}
	f, root, fresh, err := c.prepareCompanionState(ctx, id, run)
	if err != nil {
		return "", err
	}
	defer f.Close()
	defer root.Close()

	if err := c.activateCompanionContainer(ctx, f, root, run, fresh); err != nil {
		return "", err
	}
	if err := c.ensureCompanionEnrolled(ctx, run, root, binding.Admission); err != nil {
		return "", err
	}
	return c.finalizeCompanionRun(ctx, run)
}

// WaitTailnet delegates lifetime to native Podman/systemd, not a reenrollment
// worker. Each daemon activation has its own in-memory ephemeral identity.
func (c *Companion) WaitTailnet(ctx context.Context, cid string) error {
	if cid == "" {
		return nil
	}
	if !containerID.MatchString(cid) {
		return domain.ErrInvalid
	}
	b, e := c.runtimePodman(ctx, "wait", "--condition=stopped", cid)
	if ctx.Err() != nil {
		return nil
	}
	if e != nil {
		return domain.ErrUnconfirmed
	}
	if strings.TrimSpace(string(b)) != "0" {
		return domain.ErrUnconfirmed
	}
	// Unexpected successful daemon exit is also a supervision failure, not a reason
	// to silently leave an enabled project without its daemon.
	return domain.ErrUnavailable
}

func (c *Companion) StopTailnet(ctx context.Context, id string) error {
	if !projectID.MatchString(id) {
		return domain.ErrInvalid
	}
	// The unit is conditioned on an existing runtime record; no scan or repair.
	f, e := openRuntimeProject(ctx, runtimeRoot, id)
	if e != nil {
		return e
	}
	defer f.Close()
	run, e := f.current()
	if e != nil {
		return e
	}
	if run.Target.Project != id {
		return domain.ErrConflict
	}
	return c.stopTailnetRun(ctx, run)
}

func confirmStoppedResolver(ctx context.Context, c *Companion, run projectRun) error {
	current, runErr := c.projectRun(ctx, run.Target.Project)
	if runErr != nil || current != run {
		return nil
	}
	resolver, e := os.ReadFile(run.Resolver)
	if e != nil || len(resolver) > 16384 || strings.Contains(strings.ToLower(string(resolver)), "tailscale") {
		return domain.ErrUnconfirmed
	}
	return nil
}

func (c *Companion) stopTailnetRun(ctx context.Context, run projectRun) error {
	id := run.Target.Project
	cid, e := c.projectContainer(ctx, id, false)
	if e != nil || cid != run.Target.Container {
		return domain.ErrConflict
	}
	rec, e := c.inspectCompanion(ctx, run)
	if e != nil {
		return e
	}
	if !rec.Running {
		return nil
	}
	result := c.logoutAndStopCompanion(ctx, run, rec.ID)
	after, e := c.inspectCompanion(ctx, run)
	if e != nil || after.ID != rec.ID || after.Running {
		return domain.ErrUnconfirmed
	}
	if e = confirmStoppedResolver(ctx, c, run); e != nil {
		return e
	}
	return result
}

func (c *Companion) logoutAndStopCompanion(ctx context.Context, run projectRun, id string) error {
	logout, done := context.WithTimeout(ctx, 5*time.Second)
	_, logoutErr := c.companionCLI(logout, run, "logout")
	done()
	if _, e := c.runtimePodman(ctx, "stop", "--time=8", id); e != nil {
		return domain.ErrUnconfirmed
	}
	if logoutErr != nil {
		return domain.ErrUnconfirmed
	}
	return nil
}

func (c *Companion) queueProjectTailnetDisable(ctx context.Context, project string, view *domain.ProjectView) {
	// Queue cancellation first. Also handle an owned orphan whose unit is already
	// inactive: stopping an inactive systemd unit does not execute ExecStop.
	if _, e := c.runtimeCommand(ctx, "/usr/bin/systemctl", "stop", "--no-block", "soda-tailnet@"+project+".service"); e == nil {
		view.Outcome = "queued"
	}
	stop, done := context.WithTimeout(ctx, 15*time.Second)
	if e := c.StopTailnet(stop, project); e != nil {
		view.Outcome = "runtime-unconfirmed"
	}
	done()
}

func (c *Companion) markStoppedProject(ctx context.Context, project string, view *domain.ProjectView) {
	// Distinguish a confirmed stopped parent from failed runtime observation.
	running, other := c.projectRunning(ctx, project)
	if other == nil && !running {
		view.State = "stopped"
	}
}

func (c *Companion) queueProjectTailnetStart(ctx context.Context, in domain.ProjectRequest, view *domain.ProjectView) bool {
	if in.Action == "inspect" || !view.Enabled {
		return true
	}
	args := []string{"start", "--no-block", "soda-tailnet@" + in.Project + ".service"}
	if _, e := c.runtimeCommand(ctx, "/usr/bin/systemctl", args...); e != nil {
		return false
	}
	view.Outcome = "queued"
	return true
}

func applyCompanionIdleState(view *domain.ProjectView, running bool) bool {
	if running {
		return view.Enabled
	}
	if !view.Enabled {
		view.State = "off"
	}
	return false // Off intent is not confirmed disconnection.
}

func (c *Companion) observeCompanionStatus(ctx context.Context, run projectRun, view *domain.ProjectView) {
	binding, e := c.Tailnet.RunBinding(ctx, run.Target)
	if e != nil {
		return
	}
	b, e := c.companionCLI(ctx, run, "status", "--json", "--peers=false")
	if e != nil {
		return
	}
	prefs, e := c.companionCLI(ctx, run, "debug", "prefs")
	if e != nil {
		return
	}
	state, addresses, dns, e := domain.ProjectStatus(b, prefs, binding)
	if e != nil || c.recheckProjectRun(ctx, run) != nil {
		return
	}
	view.State = state
	view.Addresses = addresses
	view.DNSName = dns
}

func (c *Companion) ObserveProjectTailnet(ctx context.Context, in domain.ProjectRequest, cid string) (domain.ProjectView, error) {
	view, e := c.Tailnet.Project(ctx, in, cid)
	if e != nil {
		return view, e
	}
	if c.Image == "" {
		return view, nil
	}
	if in.Action == "disable" {
		c.queueProjectTailnetDisable(ctx, in.Project, &view)
	}
	run, e := c.projectRun(ctx, in.Project)
	if e != nil {
		c.markStoppedProject(ctx, in.Project, &view)
		return view, nil
	}
	if !c.queueProjectTailnetStart(ctx, in, &view) {
		return view, nil
	}
	rec, e := c.inspectCompanion(ctx, run)
	if e != nil || !applyCompanionIdleState(&view, rec.Running) {
		return view, nil
	}
	c.observeCompanionStatus(ctx, run, &view)
	return view, nil
}
