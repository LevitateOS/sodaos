package project

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	domain "github.com/levitateos/sodaos/internal/project"
)

const (
	execTestProject   = "p0123456789abcdef01234567"
	execTestID        = "f0123456789abcdef01234567"
	execTestContainer = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	execTestDigest    = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	execTestCommit    = "abcdef0123456789abcdef0123456789abcdef01"
)

func execTestPreparation() domain.Preparation {
	setup := execTestSetup()
	return domain.Preparation{
		ID: execTestID, Project: execTestProject, Role: domain.RoleCoder,
		Requirements: domain.RequirementAcceptance{ID: "d0123456789abcdef01234567", Revision: 1, Approver: 7, SourceCommit: execTestCommit, Digest: execTestDigest},
		Approval:     domain.AdminApproval{ID: "d123456789abcdef012345678", Revision: 1, Approver: 9, EffectsDigest: execTestDigest},
		SourceCommit: execTestCommit, SetupDigest: domain.SetupDigestOf(setup.Files), Tools: []string{"python3"},
	}
}

func execTestSetup() domain.ApprovedSetup {
	return domain.ApprovedSetup{
		Files:  map[string][]byte{"setup.sh": []byte("true\n"), "check.sh": []byte("true\n")},
		Bundle: []byte("bundle-bytes"),
	}
}

type scriptedExec struct {
	calls  []string
	ops    []string
	handle func(args []string, stdin []byte) ([]byte, error)
}

func (s *scriptedExec) Run(_ context.Context, stdin []byte, executable string, args ...string) ([]byte, error) {
	if executable != "/usr/bin/podman" {
		return nil, errors.New("unexpected executable")
	}
	s.calls = append(s.calls, strings.Join(args, " "))
	return s.handle(args, stdin)
}

func execInspection(running bool) []byte {
	return []byte(`{"id":"` + execTestContainer + `","running":` + map[bool]string{true: "true", false: "false"}[running] +
		`,"project":"` + execTestProject + `","owner":"3","privileged":false,"userns":"private",` +
		`"mappings":{"UidMap":["0:524288:262144"],"GidMap":["0:524288:262144"]}}`)
}

func execHelperState(phase, missing string) []byte {
	out, _ := json.Marshal(map[string]any{
		"hold": map[string]any{"active": false, "revision": -1}, "known": true, "phase": phase,
		"role": domain.RoleCoder, "setup_digest": domain.SetupDigestOf(execTestSetup().Files), "source_commit": execTestCommit,
		"tools": []any{}, "verified": map[string]any{"uid": "1001", "login": domain.RoleCoder, "groups": domain.RoleCoder},
		"missing": missing, "stopped": phase == domain.PrepareStopped,
		"ready": phase == domain.PrepareReady, "setup_log": "", "check_log": "",
	})
	return out
}

func execApproveResponse() []byte {
	out, _ := json.Marshal(map[string]any{
		"approved": execTestID, "repeated": false,
		"checkout": "/home/soda-coder/checkouts/" + execTestID, "credential_file": "",
	})
	return out
}

// prepareScript answers the fixed Prepare argv sequence. Missing tools fail
// resolution; groupsOverride changes the launcher group observation.
func prepareScript(exec *scriptedExec, missing string, groupsOverride string, phaseAfterStart string) {
	groups := domain.RoleCoder
	if groupsOverride != "" {
		groups = groupsOverride
	}
	exec.handle = func(args []string, stdin []byte) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "--format") && strings.Contains(joined, "inspect"):
			return execInspection(true), nil
		case strings.HasSuffix(joined, domain.FactoryHelper):
			var payload map[string]any
			if err := json.Unmarshal(stdin, &payload); err != nil {
				return nil, err
			}
			op, _ := payload["op"].(string)
			exec.ops = append(exec.ops, op)
			switch op {
			case "ensure":
				return []byte(`{"roles":["soda-coder","soda-reviewer"]}`), nil
			case "approve":
				files, _ := payload["files"].(map[string]any)
				if payload["id"] != execTestID || payload["role"] != domain.RoleCoder || files["setup.sh"] == nil || files["check.sh"] == nil || payload["bundle"] == nil {
					return nil, errors.New("approve payload lost bounded inputs")
				}
				return execApproveResponse(), nil
			case "record", "start":
				return []byte(`{"started":"` + execTestID + `"}`), nil
			case "inspect":
				phase := domain.PrepareApproved
				for _, seen := range exec.ops {
					if seen == "start" {
						phase = phaseAfterStart
					}
					if seen == "record" && phase == domain.PrepareApproved {
						if missing != "" {
							phase = domain.PrepareWaiting
						} else if groups != domain.RoleCoder {
							phase = domain.PrepareFailed
						}
					}
				}
				return execHelperState(phase, missing), nil
			}
			return nil, errors.New("unexpected helper op " + op)
		case strings.Contains(joined, "/usr/bin/test -d ") && strings.HasSuffix(joined, "/.git"):
			return nil, errors.New("no git tree yet")
		case strings.Contains(joined, "/usr/bin/ls -A "):
			return []byte(""), nil
		case strings.Contains(joined, "/usr/bin/git clone "):
			if !strings.Contains(joined, "/var/lib/soda/factory/preparations/"+execTestID+"/snapshot/source.bundle") ||
				!strings.Contains(joined, "/home/soda-coder/checkouts/"+execTestID) {
				return nil, errors.New("clone escaped fixed paths")
			}
			return []byte(""), nil
		case strings.Contains(joined, "rev-parse HEAD"):
			return []byte(execTestCommit + "\n"), nil
		case strings.Contains(joined, "/usr/bin/mkdir -m 700 -p ") && strings.HasSuffix(joined, "/.soda-home"):
			return []byte(""), nil
		case strings.HasSuffix(joined, "/usr/bin/id -u"):
			return []byte("1001\n"), nil
		case strings.HasSuffix(joined, "/usr/bin/id -un"):
			return []byte(domain.RoleCoder + "\n"), nil
		case strings.HasSuffix(joined, "/usr/bin/id -Gn"):
			return []byte(groups + "\n"), nil
		case strings.Contains(joined, "/usr/bin/test -r ") && strings.Contains(joined, "setup.sh"):
			return []byte(""), nil
		case strings.Contains(joined, "/usr/bin/test -w ") && strings.Contains(joined, "setup.sh"):
			return nil, errors.New("not writable")
		case strings.Contains(joined, "/usr/bin/test -w /srv/project/shared"):
			return nil, errors.New("not writable")
		case strings.Contains(joined, "/usr/bin/test -e /run/"):
			return nil, errors.New("absent")
		case strings.Contains(joined, "/usr/bin/test -x /usr/bin/sudo"):
			return nil, errors.New("absent")
		case strings.Contains(joined, "/usr/bin/test -w ") && strings.Contains(joined, ".soda-home"):
			return []byte(""), nil
		case strings.Contains(joined, `command -v`):
			name := args[len(args)-1]
			if name == missing {
				return nil, errors.New("not installed")
			}
			return []byte("/usr/bin/" + name + "\n"), nil
		case strings.HasSuffix(joined, "--version"):
			return []byte("9.9-test\n"), nil
		}
		return nil, errors.New("unexpected argv: " + joined)
	}
}

func TestPrepareWaitsOnMissingToolWithoutStarting(t *testing.T) {
	exec := &scriptedExec{}
	prepareScript(exec, "python3", "", domain.PrepareRunning)
	r := &Runtime{Exec: exec}
	state, err := r.Prepare(context.Background(), domain.Prepare{Preparation: execTestPreparation(), Setup: execTestSetup()})
	if err != nil {
		t.Fatalf("waiting prepare errored: %v", err)
	}
	if state.Phase != domain.PrepareWaiting || state.Missing != "python3" || state.Ready || state.Container != execTestContainer {
		t.Fatalf("waiting state: %+v", state)
	}
	for _, op := range exec.ops {
		if op == "start" {
			t.Fatal("setup started despite the missing prerequisite")
		}
	}
}

func TestPrepareRefusesUnexpectedRoleGroups(t *testing.T) {
	exec := &scriptedExec{}
	prepareScript(exec, "", "soda-coder wheel", domain.PrepareRunning)
	r := &Runtime{Exec: exec}
	state, err := r.Prepare(context.Background(), domain.Prepare{Preparation: execTestPreparation(), Setup: execTestSetup()})
	if err != nil {
		t.Fatalf("refused prepare errored: %v", err)
	}
	if state.Phase != domain.PrepareFailed || state.Ready {
		t.Fatalf("refusal state: %+v", state)
	}
	for _, op := range exec.ops {
		if op == "start" {
			t.Fatal("setup started despite launcher refusal")
		}
	}
}

func TestPrepareStartsDetachedSetup(t *testing.T) {
	exec := &scriptedExec{}
	prepareScript(exec, "", "", domain.PrepareRunning)
	r := &Runtime{Exec: exec}
	state, err := r.Prepare(context.Background(), domain.Prepare{Preparation: execTestPreparation(), Setup: execTestSetup()})
	if err != nil {
		t.Fatalf("prepare errored: %v", err)
	}
	if state.Phase != domain.PrepareRunning || state.ID != execTestID || state.Role != domain.RoleCoder {
		t.Fatalf("running state: %+v", state)
	}
	seen := map[string]bool{}
	for _, op := range exec.ops {
		seen[op] = true
	}
	for _, op := range []string{"ensure", "approve", "record", "start", "inspect"} {
		if !seen[op] {
			t.Fatalf("missing helper op %q in %q", op, exec.ops)
		}
	}
}

func TestPrepareStopsOnRacingTombstone(t *testing.T) {
	exec := &scriptedExec{}
	exec.handle = func(args []string, stdin []byte) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--format") && strings.Contains(joined, "inspect") {
			return execInspection(true), nil
		}
		if strings.HasSuffix(joined, domain.FactoryHelper) {
			var payload map[string]any
			_ = json.Unmarshal(stdin, &payload)
			exec.ops = append(exec.ops, payload["op"].(string))
			if payload["op"] == "ensure" {
				return []byte(`{}`), nil
			}
			if payload["op"] == "approve" {
				return nil, errors.New("preparation was stopped; use a new identity")
			}
			return execHelperState(domain.PrepareStopped, ""), nil
		}
		return nil, errors.New("unexpected argv: " + joined)
	}
	r := &Runtime{Exec: exec}
	state, err := r.Prepare(context.Background(), domain.Prepare{Preparation: execTestPreparation(), Setup: execTestSetup()})
	if err != nil {
		t.Fatalf("stopped prepare errored: %v", err)
	}
	if !state.Stopped || state.Phase != domain.PrepareStopped {
		t.Fatalf("stopped state: %+v", state)
	}
}

func TestStopPreparationConfirmsRetirement(t *testing.T) {
	exec := &scriptedExec{}
	exec.handle = func(args []string, stdin []byte) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--format") && strings.Contains(joined, "inspect") {
			return execInspection(false), nil
		}
		var payload map[string]any
		_ = json.Unmarshal(stdin, &payload)
		if payload["op"] == "stop" {
			return []byte(`{"stopped":"` + execTestID + `","retirement":"confirmed","known":true}`), nil
		}
		return execHelperState(domain.PrepareStopped, ""), nil
	}
	r := &Runtime{Exec: exec}
	state, err := r.StopPreparation(context.Background(), domain.PrepareStop{Project: execTestProject, ID: execTestID})
	if err != nil {
		t.Fatalf("stop errored: %v", err)
	}
	if !state.Stopped || state.Retirement != "confirmed" {
		t.Fatalf("stop state: %+v", state)
	}
}

func TestHoldPreparationConfirmsMarker(t *testing.T) {
	exec := &scriptedExec{}
	exec.handle = func(args []string, stdin []byte) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--format") && strings.Contains(joined, "inspect") {
			return execInspection(true), nil
		}
		var payload map[string]any
		_ = json.Unmarshal(stdin, &payload)
		if payload["op"] != "hold" {
			return nil, errors.New("unexpected helper op")
		}
		return []byte(`{"hold":{"active":true,"revision":2}}`), nil
	}
	r := &Runtime{Exec: exec}
	held, err := r.HoldPreparation(context.Background(), domain.PrepareHold{Project: execTestProject, Hold: true, Revision: 2})
	if err != nil {
		t.Fatalf("hold errored: %v", err)
	}
	if !held.Active || held.Revision != 2 {
		t.Fatalf("hold state: %+v", held)
	}
}

func TestPrepareIDMapKeepsHostRootOut(t *testing.T) {
	if !prepareIDMap([]string{"0:524288:262144"}) {
		t.Fatal("shifted production mapping rejected")
	}
	for _, values := range [][]string{
		{},
		{"0:0:262144"},
		{"0:1:1"},
		{"1:1000:262144", "0:0:65536"},
		{"bogus"},
	} {
		if prepareIDMap(values) {
			t.Fatalf("unsafe mapping accepted: %q", values)
		}
	}
}

func TestMapPreparationStateRejectsInconsistentObservations(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"hold": map[string]any{"active": false, "revision": -1}, "known": true, "phase": "ready",
		"role": domain.RoleCoder, "setup_digest": execTestDigest, "source_commit": execTestCommit,
		"tools": []any{}, "missing": "", "stopped": false, "ready": false,
		"setup_log": "", "check_log": "",
	})
	if _, err := mapPreparationState(execTestContainer, raw); err == nil {
		t.Fatal("inconsistent ready flag accepted")
	}
	if _, err := mapPreparationState(execTestContainer, []byte(`{"known":true,"phase":"resumed"}`)); err == nil {
		t.Fatal("unknown phase accepted")
	}
}

func TestInvalidPreparationNeverExecutes(t *testing.T) {
	exec := &scriptedExec{handle: func([]string, []byte) ([]byte, error) { return nil, errors.New("unexpected call") }}
	r := &Runtime{Exec: exec}
	bad := execTestPreparation()
	bad.ID = "../escape"
	if _, err := r.Prepare(context.Background(), domain.Prepare{Preparation: bad, Setup: execTestSetup()}); err == nil {
		t.Fatal("untrusted preparation reached the executor")
	}
	if len(exec.calls) != 0 {
		t.Fatal("untrusted preparation executed native commands")
	}
}
