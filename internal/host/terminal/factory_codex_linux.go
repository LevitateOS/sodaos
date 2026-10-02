//go:build linux

package terminal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

func factoryUserBus() string {
	return fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%d/bus", os.Geteuid())
}

func (s *Service) factorySystemctl(ctx context.Context, args ...string) ([]byte, error) {
	full := []string{factoryUserBus(), "/usr/bin/systemctl", "--user"}
	return s.Exec.Run(ctx, nil, "/usr/bin/env", append(full, args...)...)
}

func (s *Service) factorySystemdRun(ctx context.Context, args ...string) ([]byte, error) {
	full := []string{factoryUserBus(), "/usr/bin/systemd-run", "--user"}
	return s.Exec.Run(ctx, nil, "/usr/bin/env", append(full, args...)...)
}

type factoryUnitShow struct {
	active     bool
	invocation string
}

func parseFactoryUnitShow(body []byte) factoryUnitShow {
	var out factoryUnitShow
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if value, ok := strings.CutPrefix(line, "ActiveState="); ok {
			out.active = value == "active"
		}
		if value, ok := strings.CutPrefix(line, "InvocationID="); ok {
			out.invocation = strings.TrimSpace(value)
		}
	}
	return out
}

func (s *Service) factoryUnitState(ctx context.Context, unit string) (factoryUnitShow, error) {
	body, err := s.factorySystemctl(ctx, "show", "--property=ActiveState,InvocationID", unit)
	if err != nil || len(body) > 4096 {
		return factoryUnitShow{}, errors.New("factory unit observation unavailable")
	}
	return parseFactoryUnitShow(body), nil
}

func (s *Service) factoryActiveInvocation(ctx context.Context, unit string, wait time.Duration) (string, error) {
	deadline := time.Now().Add(wait)
	for {
		show, err := s.factoryUnitState(ctx, unit)
		if err == nil && show.active && terminalID.MatchString(show.invocation) {
			return show.invocation, nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return "", identity.ErrStale
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func factoryRoleID(out []byte) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid role identity")
	}
	return id, nil
}

func (s *Service) factoryRoleIDs(ctx context.Context, container, role string) (int, int, error) {
	uidOut, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/id", "-u", role)
	if err != nil || len(uidOut) > 64 {
		return 0, 0, errors.New("factory role is not resolvable")
	}
	uid, err := factoryRoleID(uidOut)
	if err != nil {
		return 0, 0, err
	}
	gidOut, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/id", "-g", role)
	if err != nil || len(gidOut) > 64 {
		return 0, 0, errors.New("factory role is not resolvable")
	}
	gid, err := factoryRoleID(gidOut)
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}

// FactoryCodexReserve prepares run directories, stages the verified harness
// bytes, resolves role identity and starts the waiting supervisor unit. No
// credentials are staged; Register delivers them after attestation.
func (s *Service) FactoryCodexReserve(ctx context.Context, run domain.FactoryRun, lease identity.Lease, pinSHA256 string, maxSecs int64) (identity.Binding, FactoryCodexPaths, error) {
	var binding identity.Binding
	p, err := factoryCodexPaths(run)
	if err != nil {
		return binding, p, err
	}
	if lease.Kind != identity.Factory || lease.ExecutionID != run.ID || lease.Generation <= 0 {
		return binding, p, identity.ErrDenied
	}
	if pinSHA256 == "" || pinSHA256 != s.CodexHarnessSHA256 || !filepath.IsAbs(s.CodexHarness) {
		return binding, p, identity.ErrDenied
	}
	if maxSecs < 60 || maxSecs > 3*3600 {
		return binding, p, identity.ErrDenied
	}
	if err = s.verifyIdentityHarness(); err != nil {
		return binding, p, err
	}
	container, err := s.factoryProjectContainer(ctx, run.Project, true)
	if err != nil {
		return binding, p, err
	}
	uid, gid, err := s.factoryRoleIDs(ctx, container, run.Role)
	if err != nil {
		return binding, p, err
	}
	if err = s.factoryCodexSetup(ctx, container, run, p, uid, gid); err != nil {
		return binding, p, err
	}
	guest, err := s.factoryCodexStage(ctx, container, run.HarnessVers)
	if err != nil {
		return binding, p, err
	}
	unit, err := factoryUnitName(run.ID)
	if err != nil {
		return binding, p, err
	}
	execArgs := []string{
		"--remote=false", "exec", "--user", run.Role, "--workdir", p.Checkout,
		"--env", "HOME=" + p.Home, "--env", "CODEX_HOME=" + p.Codex, "--env", "TERM=dumb",
		// Factory runs need a default git author identity so coder
		// commits succeed; an explicit git -c user.name/email still wins.
		"--env", "GIT_AUTHOR_NAME=" + run.Role, "--env", "GIT_AUTHOR_EMAIL=" + run.Role + "@localhost",
		"--env", "GIT_COMMITTER_NAME=" + run.Role, "--env", "GIT_COMMITTER_EMAIL=" + run.Role + "@localhost",
		container, "/usr/bin/setsid", "--wait", "/usr/bin/sh", "-c", systemdEscape(factorySupervisor(p, guest, run.Model)),
	}
	runArgs := []string{
		"--unit=" + unit, "--collect", "--property=KillMode=control-group",
		"--property=RuntimeMaxSec=" + strconv.FormatInt(maxSecs, 10), "--property=TimeoutStopSec=10",
		"--", "/usr/bin/podman",
	}
	if _, err = s.factorySystemdRun(ctx, append(runArgs, execArgs...)...); err != nil {
		return binding, p, errors.New("factory unit start unconfirmed")
	}
	invocation, err := s.factoryActiveInvocation(ctx, unit, 10*time.Second)
	if err != nil {
		// A started-but-unattested unit must not linger: stop it before
		// reporting, so Reserve either fully succeeds or leaves nothing
		// behind for the orchestrator to adopt.
		_, _ = s.factorySystemctl(ctx, "stop", unit)
		return binding, p, err
	}
	return identity.Binding{
		Kind: identity.Factory, ID: run.ID, Project: container, Login: run.Role,
		UID: uid, GID: gid, Scope: domain.FactoryScopeCodex, InvocationID: invocation,
		CredentialRoot: p.RunDir, Generation: lease.Generation, ChildID: run.Preparation,
	}, p, nil
}

func (s *Service) factoryCodexSetup(ctx context.Context, container string, run domain.FactoryRun, p FactoryCodexPaths, uid, gid int) error {
	setup := strings.Join([]string{
		"set -u",
		"mkdir -p -m 700 " + shellQuote(p.Home) + " " + shellQuote(p.Codex),
		"chown " + strconv.Itoa(uid) + ":" + strconv.Itoa(gid) + " " + shellQuote(p.RunDir) + " " + shellQuote(p.Home) + " " + shellQuote(p.Codex),
		"chmod 700 " + shellQuote(p.RunDir) + " " + shellQuote(p.Home) + " " + shellQuote(p.Codex),
	}, "\n") + "\n"
	if _, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sh", "-c", setup); err != nil {
		return errors.New("factory run directories unconfirmed")
	}
	head, err := s.podman(ctx, nil, "--remote=false", "exec", "--user", run.Role, container,
		"/usr/bin/git", "-C", p.Checkout, "rev-parse", "HEAD")
	if err != nil || len(head) > 1024 {
		return errors.New("factory checkout identity unconfirmed")
	}
	if strings.TrimSpace(string(head)) != run.SourceCommit {
		return errors.New("factory checkout is not the assigned commit")
	}
	return nil
}

func (s *Service) factoryCodexStage(ctx context.Context, container, version string) (string, error) {
	guest := domain.FactoryCodexGuest(version)
	if guest == "" {
		return "", identity.ErrDenied
	}
	if out, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sha256sum", guest); err != nil {
		hostBin := filepath.Join(s.CodexHarness, "bin", "codex")
		if _, err := s.podman(ctx, nil, "--remote=false", "cp", hostBin, container+":"+guest+".new"); err != nil {
			return "", errors.New("factory harness staging unconfirmed")
		}
		install := "mv " + shellQuote(guest+".new") + " " + shellQuote(guest) + " && chmod 755 " + shellQuote(guest) + " && /usr/bin/sha256sum " + shellQuote(guest) + "\n"
		out, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sh", "-c", install)
		if err != nil {
			return "", errors.New("factory harness install unconfirmed")
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) != 2 || fields[0] != s.CodexHarnessSHA256 {
			return "", errors.New("guest harness digest differs")
		}
	} else if fields := strings.Fields(strings.TrimSpace(string(out))); len(fields) != 2 || fields[0] != s.CodexHarnessSHA256 {
		return "", errors.New("guest harness digest differs")
	}
	// The CLI resolves its code-mode host as a fixed sibling; without it
	// tool execution fails closed. The host bytes ride the same trusted
	// harness directory with transfer verification; the version pin selects
	// the CLI whose digest the executor admits.
	if err := s.factoryCodexStageHost(ctx, container); err != nil {
		return "", err
	}
	return guest, nil
}

func (s *Service) factoryCodexStageHost(ctx context.Context, container string) error {
	const hostGuest = "/usr/local/bin/codex-code-mode-host"
	hostBin := filepath.Join(s.CodexHarness, "bin", "codex-code-mode-host")
	want, err := os.ReadFile(hostBin)
	if err != nil {
		return errors.New("factory code host is not staged")
	}
	sum := sha256.Sum256(want)
	digest := hex.EncodeToString(sum[:])
	if out, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sha256sum", hostGuest); err == nil {
		if fields := strings.Fields(strings.TrimSpace(string(out))); len(fields) == 2 && fields[0] == digest {
			return nil
		}
	}
	if _, err := s.podman(ctx, nil, "--remote=false", "cp", hostBin, container+":"+hostGuest+".new"); err != nil {
		return errors.New("factory code host staging unconfirmed")
	}
	install := "mv " + shellQuote(hostGuest+".new") + " " + shellQuote(hostGuest) + " && chmod 755 " + shellQuote(hostGuest) + " && /usr/bin/sha256sum " + shellQuote(hostGuest) + "\n"
	out, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sh", "-c", install)
	if err != nil {
		return errors.New("factory code host install unconfirmed")
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) != 2 || fields[0] != digest {
		return errors.New("guest code host digest differs")
	}
	return nil
}

// FactoryCodexStart stages the delivered credential, the exact prompt bytes
// and finally the start marker the waiting supervisor gates on. Marker order
// is load-bearing: the CLI can never start without staged delivery.
func (s *Service) FactoryCodexStart(ctx context.Context, l identity.Lease, credential, prompt []byte) error {
	p, err := factoryCodexBinding(l)
	if err != nil {
		return err
	}
	if !identity.CredentialValid(credential) {
		return identity.ErrDenied
	}
	if len(prompt) == 0 || len(prompt) > domain.MaxFactoryPrompt {
		return identity.ErrDenied
	}
	container, err := s.factoryProjectContainer(ctx, l.ProjectID, true)
	if err != nil || container != l.Binding.Project {
		return identity.ErrStale
	}
	if err = s.factoryStageFile(ctx, container, l.Binding, p.Auth, credential); err != nil {
		return err
	}
	if err = s.factoryStageFile(ctx, container, l.Binding, p.Prompt, prompt); err != nil {
		return err
	}
	if err = s.factoryStageFile(ctx, container, l.Binding, p.Marker, []byte{}); err != nil {
		return err
	}
	// The supervisor may consume the marker the instant it lands; either
	// the marker or its consumed form proves staging reached the gate.
	out, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sh", "-c",
		"test -s "+shellQuote(p.Auth)+" && test -s "+shellQuote(p.Prompt)+" && { test -f "+shellQuote(p.Marker)+" || test -f "+shellQuote(p.Started)+"; }\n")
	if err != nil || len(out) > 1024 {
		return errors.New("factory start staging unconfirmed")
	}
	return nil
}

func (s *Service) factoryStageFile(ctx context.Context, container string, b *identity.Binding, path string, data []byte) error {
	install := "/usr/bin/install -m 600 /dev/stdin " + shellQuote(path) + " && chown " + strconv.Itoa(b.UID) + ":" + strconv.Itoa(b.GID) + " " + shellQuote(path) + "\n"
	if _, err := s.podman(ctx, data, "--remote=false", "exec", "--interactive", container, "/usr/bin/sh", "-c", install); err != nil {
		return errors.New("factory file staging unconfirmed")
	}
	return nil
}

// FactoryCodexWait blocks until the supervised unit leaves active state, then
// reads the recorded exit and bounded last-message output. A missing exit
// record means the unit never reached the CLI; the caller treats that as a
// launch failure rather than guessing an outcome.
func (s *Service) FactoryCodexWait(ctx context.Context, l identity.Lease) (int, string, error) {
	p, err := factoryCodexBinding(l)
	if err != nil {
		return -1, "", err
	}
	unit, err := factoryUnitName(l.ExecutionID)
	if err != nil {
		return -1, "", err
	}
	for {
		show, err := s.factoryUnitState(ctx, unit)
		if err != nil {
			return -1, "", err
		}
		if !show.active {
			break
		}
		select {
		case <-ctx.Done():
			return -1, "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	exit := -1
	if out, err := s.podman(ctx, nil, "--remote=false", "exec", l.Binding.Project, "/usr/bin/cat", p.RunDir+"/exit"); err == nil {
		if code, perr := strconv.Atoi(strings.TrimSpace(string(out))); perr == nil && code >= 0 && code <= 255 {
			exit = code
		}
	}
	output := ""
	if out, err := s.podman(ctx, nil, "--remote=false", "exec", l.Binding.Project, "/usr/bin/head", "-c", "65537", p.Output); err == nil {
		output = string(out)
	}
	return exit, output, nil
}

// FactoryCodexValidate attests the live supervised boundary before the broker
// releases credential bytes: exact container incarnation, role identity and
// unit invocation. Anything else refuses.
func (s *Service) FactoryCodexValidate(ctx context.Context, l identity.Lease) error {
	if _, err := factoryCodexBinding(l); err != nil {
		return err
	}
	container, err := s.factoryProjectContainer(ctx, l.ProjectID, true)
	if err != nil || container != l.Binding.Project {
		return identity.ErrDenied
	}
	uid, gid, err := s.factoryRoleIDs(ctx, container, l.Binding.Login)
	if err != nil || uid != l.Binding.UID || gid != l.Binding.GID {
		return identity.ErrDenied
	}
	unit, err := factoryUnitName(l.ExecutionID)
	if err != nil {
		return err
	}
	show, err := s.factoryUnitState(ctx, unit)
	if err != nil {
		return identity.ErrDenied
	}
	if !show.active || show.invocation != l.Binding.InvocationID {
		return identity.ErrDenied
	}
	return nil
}

// FactoryCodexStop retires only the recorded run boundary: its host unit and
// its container process group. It is idempotent: an already retired run
// succeeds. Human sessions and sibling runs are never targeted.
func (s *Service) FactoryCodexStop(ctx context.Context, l identity.Lease) error {
	p, err := factoryCodexBinding(l)
	if err != nil {
		return err
	}
	unit, err := factoryUnitName(l.ExecutionID)
	if err != nil {
		return err
	}
	before := s.factoryReadPID(ctx, l.Binding.Project, p)
	_, _ = s.factorySystemctl(ctx, "stop", unit)
	deadline := time.Now().Add(15 * time.Second)
	for {
		show, err := s.factoryUnitState(ctx, unit)
		if err == nil && !show.active {
			break
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return identity.ErrUncertain
		}
		select {
		case <-ctx.Done():
			return identity.ErrUncertain
		case <-time.After(250 * time.Millisecond):
		}
	}
	if exists, err := s.factoryContainerExists(ctx, l.Binding.Project); err != nil || !exists {
		if err != nil {
			return identity.ErrUncertain
		}
		return nil
	}
	if _, err = s.podman(ctx, nil, "--remote=false", "exec", l.Binding.Project, "/usr/bin/sh", "-c", factoryRetire(p)); err != nil {
		return identity.ErrUncertain
	}
	after := s.factoryReadPID(ctx, l.Binding.Project, p)
	if after != "" && after != before {
		if _, err = s.podman(ctx, nil, "--remote=false", "exec", l.Binding.Project, "/usr/bin/sh", "-c", factoryRetire(p)); err != nil {
			return identity.ErrUncertain
		}
		if again := s.factoryReadPID(ctx, l.Binding.Project, p); again != after {
			return identity.ErrUncertain
		}
	}
	return nil
}

func (s *Service) factoryReadPID(ctx context.Context, container string, p FactoryCodexPaths) string {
	out, err := s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/cat", p.PIDFile)
	if err != nil || len(out) > 256 || !bytes.Contains(out, []byte(" ")) {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (s *Service) factoryContainerExists(ctx context.Context, container string) (bool, error) {
	_, err := s.podman(ctx, nil, "--remote=false", "container", "exists", container)
	if err == nil {
		return true, nil
	}
	var exited interface{ ExitCode() int }
	if errors.As(err, &exited) && exited.ExitCode() == 1 {
		return false, nil
	}
	return false, identity.ErrUncertain
}

// FactoryCodexCapture reads back the maintained credential after confirmed
// retirement. It never invents bytes: a missing or invalid file is uncertain.
func (s *Service) FactoryCodexCapture(ctx context.Context, l identity.Lease) ([]byte, error) {
	p, err := factoryCodexBinding(l)
	if err != nil {
		return nil, err
	}
	out, err := s.podman(ctx, nil, "--remote=false", "exec", l.Binding.Project, "/usr/bin/head", "-c", "262145", p.Auth)
	if err != nil {
		return nil, identity.ErrUncertain
	}
	if !identity.CredentialValid(out) {
		return nil, identity.ErrUncertain
	}
	return out, nil
}

// FactoryCodexFinish stops the recorded boundary and captures the maintained
// credential. Capture runs only after retirement is confirmed.
func (s *Service) FactoryCodexFinish(ctx context.Context, l identity.Lease) ([]byte, error) {
	if err := s.FactoryCodexStop(ctx, l); err != nil {
		return nil, err
	}
	return s.FactoryCodexCapture(ctx, l)
}

// FactoryCodexStopUnbound retires a run whose binding was never recorded:
// the deterministic unit by name and the derived run process group. It is
// the recovery path for a reservation that never reached registration; it
// never touches another run's unit or processes.
func (s *Service) FactoryCodexStopUnbound(ctx context.Context, run domain.FactoryRun) error {
	p, err := factoryCodexPaths(run)
	if err != nil {
		return err
	}
	unit, err := factoryUnitName(run.ID)
	if err != nil {
		return err
	}
	_, _ = s.factorySystemctl(ctx, "stop", unit)
	deadline := time.Now().Add(15 * time.Second)
	for {
		show, err := s.factoryUnitState(ctx, unit)
		if err == nil && !show.active {
			break
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return identity.ErrUncertain
		}
		select {
		case <-ctx.Done():
			return identity.ErrUncertain
		case <-time.After(250 * time.Millisecond):
		}
	}
	container, err := s.factoryProjectContainer(ctx, run.Project, false)
	if err != nil {
		if exists, existsErr := s.factoryContainerExists(ctx, "soda-"+run.Project); existsErr != nil || !exists {
			if existsErr != nil {
				return identity.ErrUncertain
			}
			return nil
		}
		return identity.ErrUncertain
	}
	before := s.factoryReadPID(ctx, container, p)
	if _, err = s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sh", "-c", factoryRetire(p)); err != nil {
		return identity.ErrUncertain
	}
	if after := s.factoryReadPID(ctx, container, p); after != "" && after != before {
		if _, err = s.podman(ctx, nil, "--remote=false", "exec", container, "/usr/bin/sh", "-c", factoryRetire(p)); err != nil {
			return identity.ErrUncertain
		}
		if again := s.factoryReadPID(ctx, container, p); again != after {
			return identity.ErrUncertain
		}
	}
	return nil
}

// FactoryCodexLive reports whether the recorded unit is currently active with
// the recorded invocation. It is an observation for inspection, never proof
// of credential delivery or completion.
func (s *Service) FactoryCodexLive(ctx context.Context, b identity.Binding) bool {
	if b.Kind != identity.Factory || b.Scope != domain.FactoryScopeCodex || !terminalID.MatchString(b.ID) || !terminalID.MatchString(b.InvocationID) {
		return false
	}
	unit, err := factoryUnitName(b.ID)
	if err != nil {
		return false
	}
	show, err := s.factoryUnitState(ctx, unit)
	return err == nil && show.active && show.invocation == b.InvocationID
}
