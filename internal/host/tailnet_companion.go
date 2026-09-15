package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/tailnet"
)

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

func (d *Daemon) runtimeCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	// The ordinary host executor predates secret-bearing native protocols. Never
	// collect its stderr for the companion, and never return native error text.
	if _, native := d.Exec.(Native); native {
		return tailnet.RunNative(ctx, command, args...)
	}
	b, e := d.Exec.Run(ctx, nil, command, args...)
	if e != nil || len(b) > 65536 {
		return nil, tailnet.ErrUnconfirmed
	}
	return b, nil
}

func (d *Daemon) runtimePodman(ctx context.Context, args ...string) ([]byte, error) {
	return d.runtimeCommand(ctx, "/usr/bin/podman", append([]string{"--remote=false"}, args...)...)
}

func companionIdentityMatches(out companion, run projectRun, image, id string) bool {
	return containerID.MatchString(id) && out.ID == id && strings.TrimPrefix(out.Image, "sha256:") == strings.TrimPrefix(image, "sha256:")
}

func companionCommandMatches(out companion, args []string) bool {
	return len(out.Command) == len(args)+1 && (out.Command[0] == "/usr/bin/podman" || out.Command[0] == "podman") && slices.Equal(out.Command[1:], args)
}

func companionExecsValid(execs []string) error {
	if len(execs) > 16 {
		return tailnet.ErrUnavailable
	}
	for _, id := range execs {
		if !containerID.MatchString(id) {
			return tailnet.ErrUnavailable
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
		return tailnet.ErrConflict
	}
	if out.Running {
		started, e := time.Parse(time.RFC3339Nano, out.Started)
		if e != nil || started.IsZero() || out.PID <= 1 {
			return tailnet.ErrConflict
		}
	}
	// CreateCommand is retained by Podman. Admit exactly our immutable recipe,
	// allowing only the native argv[0] spelling, never labels/name alone.
	if !companionCommandMatches(out, args) {
		return tailnet.ErrConflict
	}
	return companionExecsValid(out.Execs)
}

func matchCompanionNamespaces(out companion, run projectRun) error {
	if !out.Running {
		return nil
	}
	identity, e := processRunIdentity(out.PID, os.ReadFile, os.Readlink)
	if e != nil || identity.UserNS != run.UserNS || identity.NetNS != run.NetNS || identity.UID != run.UID || identity.GID != run.GID {
		return tailnet.ErrConflict
	}
	return nil
}

func (d *Daemon) inspectCompanion(ctx context.Context, run projectRun) (companion, error) {
	var out companion
	id, e := readCompanionID(runtimeRoot, run, 0, 0)
	if e != nil {
		return out, e
	}
	b, e := d.runtimePodman(ctx, "inspect", "--format", companionInspect, id)
	if e != nil || strictjson.Decode(bytes.NewReader(b), &out) != nil {
		return out, tailnet.ErrConflict
	}
	if e = validateCompanionRecord(out, run, d.Config.TailnetImage, id); e != nil {
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
		return tailnet.ErrConflict
	}
	return nil
}

func companionStillRunning(before, after companion, err, other error) bool {
	return err == nil && other == nil && after.ID == before.ID && after.PID == before.PID && after.Started == before.Started && after.Running
}

func (d *Daemon) companionCLI(ctx context.Context, run projectRun, args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, tailnet.ErrInvalid
	}
	c, e := d.inspectCompanion(ctx, run)
	if e != nil || !c.Running {
		return nil, tailnet.ErrUnavailable
	}
	if args[0] != "logout" {
		if e = companionResolver(run, c.PID, os.Stat); e != nil {
			return nil, e
		}
	}
	command := []string{"exec", "--user=0:0", c.ID, "/usr/local/bin/tailscale", "--socket=/run/tailscale/tailscaled.sock"}
	b, e := d.runtimePodman(ctx, append(command, args...)...)
	after, other := d.inspectCompanion(ctx, run)
	if !companionStillRunning(c, after, e, other) {
		return nil, tailnet.ErrUnconfirmed
	}
	if args[0] != "logout" {
		if e = companionResolver(run, after.PID, os.Stat); e != nil {
			return nil, tailnet.ErrUnconfirmed
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
		return run, tailnet.ErrUnavailable
	}
	if strictjson.Decode(file, &run) != nil {
		return run, tailnet.ErrUnavailable
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
		return tailnet.ErrUnavailable
	}
	file, e := f.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if e != nil {
		return tailnet.ErrConflict
	}
	defer file.Close()
	if _, e = file.Write(b); e != nil {
		return tailnet.ErrUnconfirmed
	}
	if file.Sync() != nil || f.root.Rename(name, "current.json") != nil {
		return tailnet.ErrUnconfirmed
	}
	dir, e := f.root.Open(".")
	if e != nil {
		return tailnet.ErrUnconfirmed
	}
	defer dir.Close()
	if dir.Sync() != nil {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

func (d *Daemon) consumeRunKey(ctx context.Context, run projectRun, root *os.Root, key string) error {
	file, e := writeRunKey(root, run, key)
	if e != nil {
		return e
	}
	defer file.Close()
	_, e = d.companionCLI(ctx, run, "up", "--json", "--timeout=5s", "--auth-key=file:/run/soda-enrollment/key", "--hostname=soda-"+run.Target.Project, "--accept-dns=true", "--accept-routes=false", "--ssh=false", "--advertise-exit-node=false")
	// A cancelled Podman observer does not cancel the native exec. The CLI has its
	// own five-second deadline. Do not retire its input until all execs have ended.
	finish, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		c, other := d.inspectCompanion(finish, run)
		if other == nil && len(c.Execs) == 0 {
			if retireRunKey(root, file) != nil {
				return tailnet.ErrUnconfirmed
			}
			return e
		}
		select {
		case <-finish.Done():
			return tailnet.ErrUnconfirmed
		case <-tick.C:
		}
	}
}

// projectTailnetEnabled reports saved policy intent without requiring
// companion-specific runtime readiness. Missing policy means Off; malformed or
// ambiguous configured state fails safely instead of enrolling with defaults.
// Enabled operations still go through the full exact-incarnation validation
// below, so this never bypasses authorization or native identity checks.
func (d *Daemon) projectTailnetEnabled(ctx context.Context, project, cid string) (bool, error) {
	if d.tailnetEnabledCheck != nil {
		return d.tailnetEnabledCheck(ctx, project, cid)
	}
	view, e := d.Tailnet.Project(ctx, tailnet.ProjectRequest{Project: project, Action: "inspect"}, cid)
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

func (d *Daemon) shouldStartTailnet(ctx context.Context, id string) (bool, error) {
	if d.Tailnet == nil || d.Config.TailnetImage == "" {
		return false, nil
	}
	// Off/unconfigured policy is a clean no-op before companion-specific
	// runtime readiness. A resolvable container with definitively disabled or
	// missing policy never waits on the bounded run validation below.
	// Malformed policy fails safely; an unresolvable container falls through
	// to that loop so enabled startup races still apply.
	if cid, cidErr := d.projectContainer(ctx, id, false); cidErr == nil {
		enabled, pErr := d.projectTailnetEnabled(ctx, id, cid)
		if pErr != nil {
			return false, preparationError("Tailnet policy unconfirmed", pErr)
		}
		if !enabled {
			return false, nil
		}
	}
	return true, nil
}

func (d *Daemon) waitProjectRuntime(ctx context.Context, id string) (projectRun, error) {
	var run projectRun
	readyParent, cancelParent := context.WithTimeout(ctx, 10*time.Second)
	parentTick := time.NewTicker(100 * time.Millisecond)
	defer parentTick.Stop()
	defer cancelParent()
	var e error
	for {
		run, e = d.projectRun(readyParent, id)
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

func (d *Daemon) admitTailnetRun(ctx context.Context, id string) (projectRun, tailnet.RunBinding, error) {
	run, err := d.waitProjectRuntime(ctx, id)
	if err != nil {
		return projectRun{}, tailnet.RunBinding{}, err
	}
	binding, err := d.Tailnet.RunBinding(ctx, run.Target)
	if err != nil {
		return projectRun{}, tailnet.RunBinding{}, preparationError("Tailnet policy unconfirmed", err)
	}
	return run, binding, nil
}

func isCompanionRunFresh(currentErr error, previous, run projectRun) bool {
	if errors.Is(currentErr, os.ErrNotExist) {
		return true
	}
	return currentErr == nil && previous.Target.Run != run.Target.Run
}

func (d *Daemon) retirePreviousCompanion(ctx context.Context, id string, run, previous projectRun) error {
	if previous.Target.Run == "" {
		return nil
	}
	if previous.Target.Project != id || previous.Target.Container != run.Target.Container {
		return preparationError("project runtime changed", tailnet.ErrConflict)
	}
	if err := d.stopTailnetRun(ctx, previous); err != nil {
		return preparationError("companion stop unconfirmed", err)
	}
	return nil
}

func (d *Daemon) reconcilePreviousRun(ctx context.Context, id string, run, previous projectRun, currentErr error) (bool, error) {
	if currentErr != nil && !errors.Is(currentErr, os.ErrNotExist) {
		return false, preparationError("companion runtime unconfirmed", currentErr)
	}
	fresh := isCompanionRunFresh(currentErr, previous, run)
	if !fresh && previous != run {
		return false, preparationError("project runtime changed", tailnet.ErrConflict)
	}
	if fresh {
		if err := d.retirePreviousCompanion(ctx, id, run, previous); err != nil {
			return false, err
		}
	}
	return fresh, nil
}

func (d *Daemon) prepareCompanionState(ctx context.Context, id string, run projectRun) (*runFiles, *os.Root, bool, error) {
	f, err := openRuntimeProject(ctx, runtimeRoot, id)
	if err != nil {
		return nil, nil, false, preparationError("companion runtime unconfirmed", err)
	}
	previous, currentErr := f.current()
	fresh, err := d.reconcilePreviousRun(ctx, id, run, previous, currentErr)
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

func (d *Daemon) createFreshCompanion(ctx context.Context, f *runFiles, root *os.Root, run projectRun) error {
	if err := f.saveCurrent(run); err != nil {
		return preparationError("companion runtime unconfirmed", err)
	}
	if err := d.recheckProjectRun(ctx, run); err != nil {
		return preparationError("project runtime changed", err)
	}
	args, err := companionCreateArgs(run, d.Config.TailnetImage)
	if err != nil {
		return preparationError("companion runtime unconfirmed", err)
	}
	created, err := d.runtimeCommand(ctx, "/usr/bin/podman", args...)
	if err != nil {
		return preparationError("companion startup unconfirmed", err)
	}
	id := strings.TrimSpace(string(created))
	if err := writeCompanionID(root, id); err != nil {
		return preparationError("companion runtime unconfirmed", err)
	}
	return nil
}

func (d *Daemon) startCompanionIfStopped(ctx context.Context, run projectRun) error {
	c, err := d.inspectCompanion(ctx, run)
	if err != nil {
		return preparationError("companion startup unconfirmed", err)
	}
	if !c.Running {
		if err := d.recheckProjectRun(ctx, run); err != nil {
			return preparationError("project runtime changed", err)
		}
		if _, err := d.runtimePodman(ctx, "start", c.ID); err != nil {
			return preparationError("companion startup unconfirmed", err)
		}
	}
	return nil
}

func (d *Daemon) activateCompanionContainer(ctx context.Context, f *runFiles, root *os.Root, run projectRun, fresh bool) error {
	if fresh {
		if err := d.createFreshCompanion(ctx, f, root, run); err != nil {
			return err
		}
	}
	return d.startCompanionIfStopped(ctx, run)
}

func (d *Daemon) waitCompanionNode(ctx context.Context, run projectRun) (bool, error) {
	ready, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		b, other := d.companionCLI(ready, run, "status", "--json", "--peers=false")
		if other == nil {
			hasNode, err := tailnet.ProjectHasNode(b)
			if err != nil {
				return false, preparationError("companion status unavailable", err)
			}
			return hasNode, nil
		}
		select {
		case <-ready.Done():
			return false, preparationError("companion status unavailable", tailnet.ErrUnavailable)
		case <-tick.C:
		}
	}
}

func (d *Daemon) enrollCompanion(ctx context.Context, run projectRun, root *os.Root) error {
	c, err := d.inspectCompanion(ctx, run)
	if err != nil || len(c.Execs) != 0 {
		return preparationError("enrollment unconfirmed", tailnet.ErrUnconfirmed)
	}
	if err := retirePendingRunKey(root, run); err != nil {
		return preparationError("enrollment unconfirmed", err)
	}
	recheck := func(c context.Context) error { return d.recheckProjectRun(c, run) }
	consume := func(c context.Context, key string) error { return d.consumeRunKey(c, run, root, key) }
	if err := d.Tailnet.EnrollRun(ctx, run.Target, recheck, consume); err != nil {
		return preparationError("enrollment unconfirmed", err)
	}
	return nil
}

func (d *Daemon) ensureCompanionEnrolled(ctx context.Context, run projectRun, root *os.Root, admission bool) error {
	hasNode, err := d.waitCompanionNode(ctx, run)
	if err != nil {
		return err
	}
	if hasNode || !admission {
		return nil
	}
	return d.enrollCompanion(ctx, run, root)
}

func (d *Daemon) finalizeCompanionRun(ctx context.Context, run projectRun) (string, error) {
	if err := d.recheckProjectRun(ctx, run); err != nil {
		return "", preparationError("project runtime changed", err)
	}
	c, err := d.inspectCompanion(ctx, run)
	if err != nil {
		return "", preparationError("companion runtime unconfirmed", err)
	}
	return c.ID, nil
}

// StartTailnet prepares one exact run. Only systemd/native explicit activation
// calls it; HTTP reads and browser focus never enter this path.
func (d *Daemon) StartTailnet(ctx context.Context, id string) (string, error) {
	start, err := d.shouldStartTailnet(ctx, id)
	if err != nil || !start {
		return "", err
	}
	run, binding, err := d.admitTailnetRun(ctx, id)
	if err != nil || !binding.Enabled {
		return "", err
	}
	f, root, fresh, err := d.prepareCompanionState(ctx, id, run)
	if err != nil {
		return "", err
	}
	defer f.Close()
	defer root.Close()

	if err := d.activateCompanionContainer(ctx, f, root, run, fresh); err != nil {
		return "", err
	}
	if err := d.ensureCompanionEnrolled(ctx, run, root, binding.Admission); err != nil {
		return "", err
	}
	return d.finalizeCompanionRun(ctx, run)
}

// WaitTailnet delegates lifetime to native Podman/systemd, not a reenrollment
// worker. Each daemon activation has its own in-memory ephemeral identity.
func (d *Daemon) WaitTailnet(ctx context.Context, cid string) error {
	if cid == "" {
		return nil
	}
	if !containerID.MatchString(cid) {
		return tailnet.ErrInvalid
	}
	b, e := d.runtimePodman(ctx, "wait", "--condition=stopped", cid)
	if ctx.Err() != nil {
		return nil
	}
	if e != nil {
		return tailnet.ErrUnconfirmed
	}
	if strings.TrimSpace(string(b)) != "0" {
		return tailnet.ErrUnconfirmed
	}
	// Unexpected successful daemon exit is also a supervision failure, not a reason
	// to silently leave an enabled project without its daemon.
	return tailnet.ErrUnavailable
}

func (d *Daemon) StopTailnet(ctx context.Context, id string) error {
	if !projectID.MatchString(id) {
		return tailnet.ErrInvalid
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
		return tailnet.ErrConflict
	}
	return d.stopTailnetRun(ctx, run)
}

func confirmStoppedResolver(ctx context.Context, d *Daemon, run projectRun) error {
	current, runErr := d.projectRun(ctx, run.Target.Project)
	if runErr != nil || current != run {
		return nil
	}
	resolver, e := os.ReadFile(run.Resolver)
	if e != nil || len(resolver) > 16384 || strings.Contains(strings.ToLower(string(resolver)), "tailscale") {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

func (d *Daemon) stopTailnetRun(ctx context.Context, run projectRun) error {
	id := run.Target.Project
	cid, e := d.projectContainer(ctx, id, false)
	if e != nil || cid != run.Target.Container {
		return tailnet.ErrConflict
	}
	c, e := d.inspectCompanion(ctx, run)
	if e != nil {
		return e
	}
	if !c.Running {
		return nil
	}
	result := d.logoutAndStopCompanion(ctx, run, c.ID)
	after, e := d.inspectCompanion(ctx, run)
	if e != nil || after.ID != c.ID || after.Running {
		return tailnet.ErrUnconfirmed
	}
	if e = confirmStoppedResolver(ctx, d, run); e != nil {
		return e
	}
	return result
}

func (d *Daemon) logoutAndStopCompanion(ctx context.Context, run projectRun, id string) error {
	logout, done := context.WithTimeout(ctx, 5*time.Second)
	_, logoutErr := d.companionCLI(logout, run, "logout")
	done()
	if _, e := d.runtimePodman(ctx, "stop", "--time=8", id); e != nil {
		return tailnet.ErrUnconfirmed
	}
	if logoutErr != nil {
		return tailnet.ErrUnconfirmed
	}
	return nil
}

func (d *Daemon) queueProjectTailnetDisable(ctx context.Context, project string, view *tailnet.ProjectView) {
	// Queue cancellation first. Also handle an owned orphan whose unit is already
	// inactive: stopping an inactive systemd unit does not execute ExecStop.
	if _, e := d.runtimeCommand(ctx, "/usr/bin/systemctl", "stop", "--no-block", "soda-tailnet@"+project+".service"); e == nil {
		view.Outcome = "queued"
	}
	stop, done := context.WithTimeout(ctx, 15*time.Second)
	if e := d.StopTailnet(stop, project); e != nil {
		view.Outcome = "runtime-unconfirmed"
	}
	done()
}

func (d *Daemon) markStoppedProject(ctx context.Context, project string, view *tailnet.ProjectView) {
	// Distinguish a confirmed stopped parent from failed runtime observation.
	env, _, other := d.inspect(ctx, project)
	if other == nil && !env.Running {
		view.State = "stopped"
	}
}

func (d *Daemon) queueProjectTailnetStart(ctx context.Context, in tailnet.ProjectRequest, view *tailnet.ProjectView) bool {
	if in.Action == "inspect" || !view.Enabled {
		return true
	}
	args := []string{"start", "--no-block", "soda-tailnet@" + in.Project + ".service"}
	if _, e := d.runtimeCommand(ctx, "/usr/bin/systemctl", args...); e != nil {
		return false
	}
	view.Outcome = "queued"
	return true
}

func applyCompanionIdleState(view *tailnet.ProjectView, running bool) bool {
	if running {
		return view.Enabled
	}
	if !view.Enabled {
		view.State = "off"
	}
	return false // Off intent is not confirmed disconnection.
}

func (d *Daemon) observeCompanionStatus(ctx context.Context, run projectRun, view *tailnet.ProjectView) {
	binding, e := d.Tailnet.RunBinding(ctx, run.Target)
	if e != nil {
		return
	}
	b, e := d.companionCLI(ctx, run, "status", "--json", "--peers=false")
	if e != nil {
		return
	}
	prefs, e := d.companionCLI(ctx, run, "debug", "prefs")
	if e != nil {
		return
	}
	state, addresses, dns, e := tailnet.ProjectStatus(b, prefs, binding)
	if e != nil || d.recheckProjectRun(ctx, run) != nil {
		return
	}
	view.State = state
	view.Addresses = addresses
	view.DNSName = dns
}

func (d *Daemon) observeProjectTailnet(ctx context.Context, in tailnet.ProjectRequest, cid string) (tailnet.ProjectView, error) {
	view, e := d.Tailnet.Project(ctx, in, cid)
	if e != nil {
		return view, e
	}
	if d.Config.TailnetImage == "" {
		return view, nil
	}
	if in.Action == "disable" {
		d.queueProjectTailnetDisable(ctx, in.Project, &view)
	}
	run, e := d.projectRun(ctx, in.Project)
	if e != nil {
		d.markStoppedProject(ctx, in.Project, &view)
		return view, nil
	}
	if !d.queueProjectTailnetStart(ctx, in, &view) {
		return view, nil
	}
	c, e := d.inspectCompanion(ctx, run)
	if e != nil || !applyCompanionIdleState(&view, c.Running) {
		return view, nil
	}
	d.observeCompanionStatus(ctx, run, &view)
	return view, nil
}
