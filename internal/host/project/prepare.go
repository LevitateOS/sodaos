package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path"
	"strconv"
	"strings"

	domain "github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// Factory helper exchange bounds. Inspect responses carry bounded setup and
// check logs; nothing here accepts caller-supplied paths or commands.
const factoryInspectLimit = 262144

func (r *Runtime) factoryHelper(ctx context.Context, id string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	out, err := r.podman(ctx, body, "exec", "--interactive", "soda-"+id, domain.FactoryHelper)
	if err != nil {
		return nil, err
	}
	if len(out) > factoryInspectLimit {
		return nil, errors.New("factory helper response exceeds the bounded size")
	}
	return out, nil
}

// prepareIDMap requires every mapping to keep host root out of the container
// and container root shifted onto a usable range. Creation-time mapping size
// stays owned by Create; preparation binds the target identity without
// re-auditing those flags.
func prepareIDMap(values []string) bool {
	if len(values) == 0 {
		return false
	}
	shifted := false
	for _, value := range values {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return false
		}
		container, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil || strconv.FormatUint(container, 10) != parts[0] {
			return false
		}
		base, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil || strconv.FormatUint(base, 10) != parts[1] || base == 0 {
			return false
		}
		size, err := strconv.ParseUint(parts[2], 10, 32)
		if err != nil || strconv.FormatUint(size, 10) != parts[2] {
			return false
		}
		if container == 0 && size >= 65536 && base+size <= 4294967295 {
			shifted = true
		}
	}
	return shifted
}

func prepareTargetReady(v projectInspection, id string, requireRunning bool) bool {
	owner, err := strconv.ParseInt(v.Owner, 10, 64)
	if err != nil || owner <= 0 {
		return false
	}
	if requireRunning && !v.Running {
		return false
	}
	if !domain.ValidContainerID(v.ID) || v.Project != id || v.Privileged || v.Userns != "private" {
		return false
	}
	return prepareIDMap(v.Mappings.UIDMap) && prepareIDMap(v.Mappings.GIDMap)
}

// prepareContainer binds the exact container identity for one preparation:
// label ownership, isolation mode and liveness. It never starts a container.
func (r *Runtime) prepareContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	if !domain.ValidID(id) {
		return "", errors.New("invalid project")
	}
	data, err := r.podman(ctx, nil, "--remote=false", "inspect", "--format", projectInspect, "soda-"+id)
	if err != nil || len(data) > 4096 {
		return "", errors.New("preparation target unavailable")
	}
	var v projectInspection
	if err = strictjson.Decode(bytes.NewReader(data), &v); err != nil {
		return "", errors.New("invalid preparation target")
	}
	if !prepareTargetReady(v, id, requireRunning) {
		return "", errors.New("preparation target not ready or isolated")
	}
	return v.ID, nil
}

func preparationPaths(role, id string) (checkout, snapshot, bundle, home string) {
	checkout = "/home/" + role + "/checkouts/" + id
	snapshot = domain.FactoryPreparationsDir + "/" + id + "/snapshot"
	bundle = snapshot + "/source.bundle"
	home = checkout + "/.soda-home"
	return checkout, snapshot, bundle, home
}

type helperInspection struct {
	Hold struct {
		Active   bool  `json:"active"`
		Revision int64 `json:"revision"`
	} `json:"hold"`
	Known        bool                  `json:"known"`
	Phase        string                `json:"phase"`
	Role         string                `json:"role"`
	SetupDigest  string                `json:"setup_digest"`
	SourceCommit string                `json:"source_commit"`
	Tools        []domain.ResolvedTool `json:"tools"`
	Verified     struct {
		UID     string `json:"uid"`
		Login   string `json:"login"`
		Groups  string `json:"groups"`
		Refusal string `json:"refusal"`
	} `json:"verified"`
	Missing   string `json:"missing"`
	Stopped   bool   `json:"stopped"`
	Ready     bool   `json:"ready"`
	SetupExit *int   `json:"setup_exit"`
	CheckExit *int   `json:"check_exit"`
	SetupLog  string `json:"setup_log"`
	CheckLog  string `json:"check_log"`
}

func mapPreparationState(container string, raw []byte) (domain.PrepareState, error) {
	var state domain.PrepareState
	var in helperInspection
	if err := strictjson.Decode(bytes.NewReader(raw), &in); err != nil {
		return state, errors.New("invalid preparation observation")
	}
	if !in.Known || !domain.ValidPreparePhase(in.Phase) || !domain.ValidFactoryRole(in.Role) ||
		!domain.ValidDigest(in.SetupDigest) || !domain.ValidCommit(in.SourceCommit) {
		return state, errors.New("invalid preparation observation")
	}
	if len(in.SetupLog) > 66560 || len(in.CheckLog) > 66560 {
		return state, errors.New("preparation observation exceeds the bounded size")
	}
	state.Phase = in.Phase
	state.Role = in.Role
	state.Container = container
	state.SourceCommit = in.SourceCommit
	state.SetupDigest = in.SetupDigest
	state.Tools = in.Tools
	state.Missing = in.Missing
	state.SetupExit = in.SetupExit
	state.CheckExit = in.CheckExit
	state.Ready = in.Ready
	state.Stopped = in.Stopped
	if in.SetupLog != "" || in.CheckLog != "" {
		state.Output = "--- setup ---\n" + in.SetupLog + "\n--- check ---\n" + in.CheckLog
	}
	if state.Ready != (state.Phase == domain.PrepareReady) {
		return domain.PrepareState{}, errors.New("preparation observation is inconsistent")
	}
	return state, nil
}

func (r *Runtime) inspectPreparationState(ctx context.Context, in domain.PrepareInspect, container string) (domain.PrepareState, error) {
	raw, err := r.factoryHelper(ctx, in.Project, map[string]any{"op": "inspect", "id": in.ID})
	if err != nil {
		return domain.PrepareState{}, err
	}
	state, err := mapPreparationState(container, raw)
	if err != nil {
		return domain.PrepareState{}, err
	}
	state.ID = in.ID
	state.Project = in.Project
	return state, nil
}

// Prepare records approved inputs, resolves tools, verifies the launcher
// environment and starts setup detached under the recorded identity. It
// returns the authoritative observed state, including waiting, failed,
// stopped and resumed states; only unconfirmed operations return an error.
func (r *Runtime) Prepare(ctx context.Context, in domain.Prepare) (domain.PrepareState, error) {
	var state domain.PrepareState
	if err := in.Validate(); err != nil {
		return state, err
	}
	prep := in.Preparation
	container, err := r.prepareContainer(ctx, prep.Project, true)
	if err != nil {
		return state, err
	}
	address := domain.PrepareInspect{Project: prep.Project, ID: prep.ID}
	if _, err = r.factoryHelper(ctx, prep.Project, map[string]any{"op": "ensure"}); err != nil {
		return state, err
	}
	if err = r.approvePreparation(ctx, prep, in.Setup); err != nil {
		// A stop tombstone racing approval surfaces as the stopped state;
		// every other refusal stays an error for a new identity or action.
		if stopped, ierr := r.inspectPreparationState(ctx, address, container); ierr == nil && stopped.Stopped {
			return stopped, nil
		}
		return state, err
	}
	state, err = r.inspectPreparationState(ctx, address, container)
	if err != nil {
		return state, err
	}
	if state.Stopped || state.Ready || state.Phase == domain.PrepareFailed {
		return state, nil
	}
	if err = r.clonePreparationSource(ctx, prep); err != nil {
		return state, err
	}
	verified, err := r.verifyLauncherEnvironment(ctx, prep)
	if err != nil {
		return state, err
	}
	tools, missing, err := r.resolvePreparationTools(ctx, prep)
	if err != nil {
		return state, err
	}
	if err = r.recordPreparationTools(ctx, prep, tools, missing, verified); err != nil {
		return state, err
	}
	if missing != "" || verified.Refusal != "" {
		return r.inspectPreparationState(ctx, address, container)
	}
	if _, err = r.factoryHelper(ctx, prep.Project, map[string]any{"op": "start", "id": prep.ID}); err != nil {
		state, ierr := r.inspectPreparationState(ctx, address, container)
		if ierr != nil {
			return domain.PrepareState{}, err
		}
		return state, err
	}
	return r.inspectPreparationState(ctx, address, container)
}

func (r *Runtime) approvePreparation(ctx context.Context, prep domain.Preparation, setup domain.ApprovedSetup) error {
	checkout, _, _, _ := preparationPaths(prep.Role, prep.ID)
	files := map[string][]byte{}
	for name, contents := range setup.Files {
		files[name] = contents
	}
	raw, err := r.factoryHelper(ctx, prep.Project, map[string]any{
		"op": "approve", "id": prep.ID, "role": prep.Role, "setup_digest": prep.SetupDigest,
		"source_commit": prep.SourceCommit, "files": files, "bundle": setup.Bundle, "credential": prep.Credential,
	})
	if err != nil {
		return err
	}
	var approved struct {
		Approved       string `json:"approved"`
		Repeated       bool   `json:"repeated"`
		Checkout       string `json:"checkout"`
		CredentialFile string `json:"credential_file"`
	}
	if err = strictjson.Decode(bytes.NewReader(raw), &approved); err != nil || approved.Approved != prep.ID {
		return errors.New("preparation approval unconfirmed")
	}
	if approved.Repeated {
		return nil
	}
	wantCredential := ""
	if prep.Credential != "" {
		wantCredential = domain.FactoryCredentialsDir + "/" + prep.Role + "/" + prep.Credential
	}
	if approved.Checkout != checkout || approved.CredentialFile != wantCredential {
		return errors.New("preparation approval resolved unexpected paths")
	}
	return nil
}

func (r *Runtime) roleExec(ctx context.Context, prep domain.Preparation, args ...string) ([]byte, error) {
	full := append([]string{"exec", "--user", prep.Role, "soda-" + prep.Project}, args...)
	return r.podman(ctx, nil, full...)
}

func (r *Runtime) clonePreparationSource(ctx context.Context, prep domain.Preparation) error {
	checkout, _, bundle, home := preparationPaths(prep.Role, prep.ID)
	if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-d", checkout+"/.git"); err == nil {
		if err = r.confirmPreparationHead(ctx, prep, checkout); err != nil {
			return err
		}
	} else {
		out, err := r.roleExec(ctx, prep, "/usr/bin/ls", "-A", checkout)
		if err != nil {
			return errors.New("preparation checkout is not inspectable")
		}
		if len(bytes.TrimSpace(out)) != 0 {
			return errors.New("preparation checkout holds unknown partial effects; use a new identity")
		}
		if _, err = r.roleExec(ctx, prep, "/usr/bin/env", "-i", "PATH=/usr/bin:/bin", "HOME="+home,
			"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0",
			"/usr/bin/git", "clone", "--template=", "--config", "core.hooksPath=/dev/null", bundle, checkout); err != nil {
			return errors.New("preparation source clone unconfirmed")
		}
		if err = r.confirmPreparationHead(ctx, prep, checkout); err != nil {
			return err
		}
	}
	// The private role home is role-created after the clone, never shared.
	if _, err := r.roleExec(ctx, prep, "/usr/bin/mkdir", "-m", "700", "-p", home); err != nil {
		return errors.New("private role home unconfirmed")
	}
	return nil
}

func (r *Runtime) confirmPreparationHead(ctx context.Context, prep domain.Preparation, checkout string) error {
	out, err := r.roleExec(ctx, prep, "/usr/bin/git", "-C", checkout, "rev-parse", "HEAD")
	if err != nil || len(out) > 1024 {
		return errors.New("preparation source identity unconfirmed")
	}
	if strings.TrimSpace(string(out)) != prep.SourceCommit {
		return errors.New("preparation checkout is not the approved commit")
	}
	return nil
}

type launcherEvidence struct {
	UID     string
	Login   string
	Groups  string
	Refusal string
}

func singleLine(out []byte, limit int) (string, bool) {
	if len(out) == 0 || len(out) > limit {
		return "", false
	}
	line := strings.TrimSpace(string(out))
	if line == "" || strings.Contains(line, "\n") {
		return "", false
	}
	return line, true
}

func (r *Runtime) verifyLauncherEnvironment(ctx context.Context, prep domain.Preparation) (launcherEvidence, error) {
	var evidence launcherEvidence
	refuse := func(reason string) (launcherEvidence, error) {
		evidence.Refusal = reason
		return evidence, nil
	}
	uid, ok := singleLine(mustRoleExec(r, ctx, prep, "/usr/bin/id", "-u"), 64)
	if !ok {
		return refuse("role uid is not observable")
	}
	evidence.UID = uid
	login, ok := singleLine(mustRoleExec(r, ctx, prep, "/usr/bin/id", "-un"), 64)
	if !ok || login != prep.Role {
		return refuse("launcher is not the assigned role")
	}
	evidence.Login = login
	groups, ok := singleLine(mustRoleExec(r, ctx, prep, "/usr/bin/id", "-Gn"), 256)
	if !ok || groups != prep.Role {
		return refuse("role holds unexpected groups")
	}
	evidence.Groups = groups
	_, snapshot, _, home := preparationPaths(prep.Role, prep.ID)
	if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-r", snapshot+"/"+domain.FactorySetupEntry); err != nil {
		return refuse("approved setup is not readable")
	}
	if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-w", snapshot+"/"+domain.FactorySetupEntry); err == nil {
		return refuse("approved setup is writable by the role")
	}
	if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-w", "/srv/project/shared"); err == nil {
		return refuse("shared project data is writable by the role")
	}
	for _, socket := range []string{"/run/podman/podman.sock", "/run/docker.sock"} {
		if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-e", socket); err == nil {
			return refuse("engine socket is visible to the role")
		}
	}
	if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-x", "/usr/bin/sudo"); err == nil {
		if _, err = r.roleExec(ctx, prep, "/usr/bin/sudo", "-n", "true"); err == nil {
			return refuse("role holds unexpected privilege")
		}
	}
	if _, err := r.roleExec(ctx, prep, "/usr/bin/test", "-w", home); err != nil {
		return refuse("private role home is not writable")
	}
	return evidence, nil
}

// mustRoleExec returns empty output on failure; callers treat that as refusal.
func mustRoleExec(r *Runtime, ctx context.Context, prep domain.Preparation, args ...string) []byte {
	out, err := r.roleExec(ctx, prep, args...)
	if err != nil {
		return nil
	}
	return out
}

func validResolvedToolPath(value string) bool {
	if value == "" || len(value) > 256 || !path.IsAbs(value) || path.Clean(value) != value {
		return false
	}
	return strings.HasPrefix(value, "/usr/bin/") || strings.HasPrefix(value, "/usr/local/bin/")
}

func (r *Runtime) resolvePreparationTools(ctx context.Context, prep domain.Preparation) ([]domain.ResolvedTool, string, error) {
	tools := []domain.ResolvedTool{}
	for _, name := range prep.Tools {
		out, err := r.podman(ctx, nil, "exec", "soda-"+prep.Project, "/bin/bash", "-c", `command -v "$0"`, name)
		resolved, ok := "", false
		if err == nil {
			resolved, ok = singleLine(out, 256)
		}
		if !ok || !validResolvedToolPath(resolved) {
			return tools, name, nil
		}
		version, err := r.podman(ctx, nil, "exec", "soda-"+prep.Project, resolved, "--version")
		observed, ok := "", false
		if err == nil {
			observed, ok = singleLine(bytes.SplitN(version, []byte("\n"), 2)[0], 256)
		}
		if !ok {
			return tools, name, nil
		}
		tools = append(tools, domain.ResolvedTool{Name: name, Path: resolved, Version: observed})
	}
	return tools, "", nil
}

func (r *Runtime) recordPreparationTools(ctx context.Context, prep domain.Preparation, tools []domain.ResolvedTool, missing string, verified launcherEvidence) error {
	verifiedPayload := map[string]any{"uid": verified.UID, "login": verified.Login, "groups": verified.Groups}
	if verified.Refusal != "" {
		verifiedPayload["refusal"] = verified.Refusal
	}
	_, err := r.factoryHelper(ctx, prep.Project, map[string]any{
		"op": "record", "id": prep.ID, "tools": tools, "missing": missing, "verified": verifiedPayload,
	})
	return err
}

// InspectPreparation reports the authoritative observed state for one
// preparation identity. It never starts, resumes or mutates the preparation.
func (r *Runtime) InspectPreparation(ctx context.Context, in domain.PrepareInspect) (domain.PrepareState, error) {
	var state domain.PrepareState
	if err := in.Validate(); err != nil {
		return state, err
	}
	container, err := r.prepareContainer(ctx, in.Project, false)
	if err != nil {
		return state, err
	}
	return r.inspectPreparationState(ctx, in, container)
}

// StopPreparation persists the stop tombstone for one identity and retires
// its setup scope. It works before the identity is ever observed.
func (r *Runtime) StopPreparation(ctx context.Context, in domain.PrepareStop) (domain.PrepareState, error) {
	var state domain.PrepareState
	if err := in.Validate(); err != nil {
		return state, err
	}
	container, err := r.prepareContainer(ctx, in.Project, false)
	if err != nil {
		return state, err
	}
	raw, err := r.factoryHelper(ctx, in.Project, map[string]any{"op": "stop", "id": in.ID})
	if err != nil {
		return state, err
	}
	var stopped struct {
		Stopped    string `json:"stopped"`
		Retirement string `json:"retirement"`
		Known      bool   `json:"known"`
	}
	if err = strictjson.Decode(bytes.NewReader(raw), &stopped); err != nil || stopped.Stopped != in.ID ||
		(stopped.Retirement != "confirmed" && stopped.Retirement != "uncertain") {
		return state, errors.New("preparation stop unconfirmed")
	}
	state, err = r.inspectPreparationState(ctx, domain.PrepareInspect(in), container)
	if err != nil {
		return state, err
	}
	state.Retirement = stopped.Retirement
	return state, nil
}

// HoldPreparation sets or clears the native maintenance hold marker. The
// store record stays the source of truth; this enforces it in the Project.
func (r *Runtime) HoldPreparation(ctx context.Context, in domain.PrepareHold) (domain.HoldState, error) {
	var state domain.HoldState
	if err := in.Validate(); err != nil {
		return state, err
	}
	if _, err := r.prepareContainer(ctx, in.Project, true); err != nil {
		return state, err
	}
	op := "release"
	if in.Hold {
		op = "hold"
	}
	raw, err := r.factoryHelper(ctx, in.Project, map[string]any{"op": op, "revision": in.Revision})
	if err != nil {
		return state, err
	}
	var observed struct {
		Hold domain.HoldState `json:"hold"`
	}
	if err = strictjson.Decode(bytes.NewReader(raw), &observed); err != nil {
		return state, errors.New("maintenance hold unconfirmed")
	}
	if observed.Hold.Active != in.Hold {
		return state, errors.New("maintenance hold outcome not confirmed")
	}
	return observed.Hold, nil
}
