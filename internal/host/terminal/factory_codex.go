package terminal

import (
	"bytes"
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// FactoryCodexPaths are the fixed container paths for one supervised Codex
// run, derived from its validated identities. No caller-supplied path is
// accepted; the guest codex binary is shared per proved harness version.
type FactoryCodexPaths struct {
	Checkout string
	RunDir   string
	Home     string
	Codex    string
	Prompt   string
	Marker   string
	Started  string
	Stop     string
	PIDFile  string
	Output   string
	Stdout   string
	Auth     string
	Guest    string
}

func factoryCodexPaths(run domain.FactoryRun) (FactoryCodexPaths, error) {
	var p FactoryCodexPaths
	if err := run.Validate(); err != nil {
		return p, err
	}
	checkout, runDir, home, codex := domain.FactoryRunPaths(run.Role, run.Preparation, run.ID)
	guest := domain.FactoryCodexGuest(run.HarnessVers)
	if checkout == "" || guest == "" {
		return p, identity.ErrDenied
	}
	return FactoryCodexPaths{
		Checkout: checkout, RunDir: runDir, Home: home, Codex: codex,
		Prompt: runDir + "/prompt", Marker: runDir + "/marker", Started: runDir + "/started",
		Stop: runDir + "/stop", PIDFile: runDir + "/supervisor.pid",
		Output: runDir + "/last-message.txt", Stdout: runDir + "/stdout.log",
		Auth: codex + "/auth.json", Guest: guest,
	}, nil
}

// FactoryCodexOutputSlice is one observed byte slice of a run's recorded
// CLI output with its cursor placement.
type FactoryCodexOutputSlice struct {
	Data      []byte
	Total     int64
	Offset    int64
	Truncated bool
	Gap       bool
}

// systemdEscape doubles every dollar for command text transported through
// systemd-run: the manager reduces $$ to a literal $ and would otherwise
// erase $VAR references and collapse $$ before the shell ever runs. Only
// the supervisor travels this path; the retire script runs through a direct
// exec and keeps single dollars.
func systemdEscape(script string) string {
	return strings.ReplaceAll(script, "$", "$$")
}

// factoryCodexBinding checks that a lease carries a supervised factory-Codex
// binding: exact run/container/role/unit/generation fields. It returns the
// derived run paths; the recorded credential root must equal the derived run
// directory rather than selecting another location.
func factoryCodexBinding(l identity.Lease) (FactoryCodexPaths, error) {
	var p FactoryCodexPaths
	b := l.Binding
	if b == nil || l.ProviderID != identity.Codex || l.Kind != identity.Factory {
		return p, identity.ErrDenied
	}
	if b.Kind != identity.Factory || b.Scope != domain.FactoryScopeCodex || !terminalID.MatchString(b.ID) {
		return p, identity.ErrDenied
	}
	if !containerID.MatchString(b.Project) || !loginName.MatchString(b.Login) || b.Login == "root" {
		return p, identity.ErrDenied
	}
	if b.UID <= 0 || b.GID <= 0 || b.Generation != l.Generation || b.Generation <= 0 {
		return p, identity.ErrDenied
	}
	if !terminalID.MatchString(b.InvocationID) || !domain.ValidPreparationID(b.ChildID) {
		return p, identity.ErrDenied
	}
	run := domain.FactoryRun{ID: b.ID, Project: l.ProjectID, Role: b.Login, Preparation: b.ChildID}
	// Harness fields are not in the binding; paths need only role/prep/run.
	checkout, runDir, home, codex := domain.FactoryRunPaths(run.Role, run.Preparation, run.ID)
	if checkout == "" || filepath.Clean(b.CredentialRoot) != runDir {
		return p, identity.ErrDenied
	}
	p = FactoryCodexPaths{
		Checkout: checkout, RunDir: runDir, Home: home, Codex: codex,
		Prompt: runDir + "/prompt", Marker: runDir + "/marker", Started: runDir + "/started",
		Stop: runDir + "/stop", PIDFile: runDir + "/supervisor.pid",
		Output: runDir + "/last-message.txt", Stdout: runDir + "/stdout.log",
		Auth: codex + "/auth.json",
	}
	return p, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// factorySupervisor waits for the staged start marker, consumes it exactly
// once, then runs the fixed Codex entrypoint and records its exit. A stop
// file ends the wait promptly so retirement never waits out the bounded
// gate. Every outcome lands in the exit file: the collecting unit may vanish
// before the exit status can be re-read from systemd.
func factorySupervisor(p FactoryCodexPaths, guest string) string {
	steps := []string{
		"RUNDIR=" + shellQuote(p.RunDir),
		"exec >\"$RUNDIR/stdout.log\" 2>&1",
		`STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo "$$ ${20}" >"$RUNDIR/supervisor.pid"`,
		`fail() { echo "$1" >"$RUNDIR/exit"; exit "$1"; }`,
		`i=0; while [ ! -f "$RUNDIR/marker" ]; do [ -f "$RUNDIR/stop" ] && fail 44; i=$((i+1)); [ "$i" -gt 600 ] && fail 42; sleep 1; done`,
		`mv "$RUNDIR/marker" "$RUNDIR/started" || fail 43`,
		shellQuote(guest) + " exec --color never --sandbox danger-full-access --skip-git-repo-check --config " + shellQuote(`model_reasoning_effort="low"`) + " --output-last-message " + shellQuote(p.Output) + " - <" + shellQuote(p.Prompt),
		`CODE=$?; echo "$CODE" >"$RUNDIR/exit"; exit "$CODE"`,
	}
	return strings.Join(steps, "\n") + "\n"
}

// factoryRetire kills only the recorded supervisor process group after
// verifying its identity, then scans for lingering members. The stop file is
// planted first so a late-starting supervisor exits instead of launching.
func factoryRetire(p FactoryCodexPaths) string {
	script := []string{
		"RUNDIR=" + shellQuote(p.RunDir),
		`: >"$RUNDIR/stop"`,
		`[ -f "$RUNDIR/supervisor.pid" ] || exit 0`,
		`read PID START <"$RUNDIR/supervisor.pid"`,
		`case "$PID" in ''|*[!0-9]*) exit 0;; esac`,
		`case "$START" in ''|*[!0-9]*) exit 0;; esac`,
		`if [ -d "/proc/$PID" ]; then`,
		`  if STAT=$(cat "/proc/$PID/stat" 2>/dev/null); then`,
		`    REST=${STAT##*)}; set -- $REST`,
		`    if [ "${20}" = "$START" ]; then`,
		`      CMDLINE=$(tr '\000' ' ' <"/proc/$PID/cmdline" 2>/dev/null) || CMDLINE=""`,
		`      case "$CMDLINE" in *"$RUNDIR"*)`,
		`        if [ "$3" = "$PID" ]; then kill -KILL -- "-$PID" 2>/dev/null || true; else kill -KILL -- "$PID" 2>/dev/null || true; fi`,
		`      ;; esac`,
		`    fi`,
		`  fi`,
		`fi`,
		`sleep 1`,
		`for S in /proc/[0-9]*/stat; do`,
		`  STAT=$(cat "$S" 2>/dev/null) || continue`,
		`  REST=${STAT##*)}; set -- $REST`,
		`  if [ "$3" = "$PID" ]; then echo "lingering: $S"; exit 1; fi`,
		`done`,
		`exit 0`,
	}
	return strings.Join(script, "\n") + "\n"
}

func factoryUnitName(runID string) (string, error) {
	unit := domain.FactoryUnitName(runID)
	if unit == "" {
		return "", identity.ErrDenied
	}
	return unit, nil
}

// factoryProjectContainer resolves a Soda project to its exact container ID:
// fixed container name, Soda labels, unprivileged private user namespace and
// required running state. This is the same gate family as ST01's proved
// preparation boundary. The human-terminal 262144-mapping check stays
// specific to interactive sessions; supervised factory runs bind the exact
// container incarnation instead of the creation profile's ID-range size.
func (s *Service) factoryProjectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	if !projectID.MatchString(id) {
		return "", identity.ErrDenied
	}
	data, err := s.podman(ctx, nil, "--remote=false", "inspect", "--format", terminalInspect, "soda-"+id)
	if err != nil || len(data) == 0 || len(data) > 16384 {
		return "", identity.ErrStale
	}
	var v terminalInspection
	if err = strictjson.Decode(bytes.NewReader(data), &v); err != nil {
		return "", identity.ErrStale
	}
	if !containerID.MatchString(v.ID) || v.Project != id || v.Privileged || v.Userns != "private" {
		return "", identity.ErrDenied
	}
	owner, err := strconv.ParseInt(v.Owner, 10, 64)
	if err != nil || owner <= 0 {
		return "", identity.ErrDenied
	}
	if requireRunning && !v.Running {
		return "", identity.ErrStale
	}
	return v.ID, nil
}
