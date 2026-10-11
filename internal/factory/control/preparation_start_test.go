package control

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

type standingHost struct {
	stubHost
	prepare func(context.Context, project.Prepare) (project.PrepareState, error)
	inspect func(context.Context, project.PrepareInspect) (project.PrepareState, error)
}

func (h *standingHost) Prepare(ctx context.Context, in project.Prepare) (project.PrepareState, error) {
	return h.prepare(ctx, in)
}

func (h *standingHost) InspectPreparation(ctx context.Context, in project.PrepareInspect) (project.PrepareState, error) {
	return h.inspect(ctx, in)
}

func standingFixture(t *testing.T) (*Coordinator, project.Prepare, *standingHost) {
	t.Helper()
	db, _ := postgresFixture(t, nil)
	host := &standingHost{}
	c := &Coordinator{Store: db, Host: host}
	grantProject(t, c, "p"+strings.Repeat("a", 24))
	files := map[string][]byte{"setup.sh": []byte("#!/bin/sh\nexit 0\n"), "check.sh": []byte("#!/bin/sh\ngit --version\n")}
	selected := map[string]string{"setup.sh": strings.Repeat("c", 64)}
	requirement := project.RequirementDecision{
		ID: "d" + strings.Repeat("1", 24), Project: "p" + strings.Repeat("a", 24), Approver: 7,
		SourceCommit: strings.Repeat("a", 40), SetupDigest: project.SetupDigestOf(files), InputsDigest: project.PreparationInputsDigestOf([]string{"setup.sh", "check.sh"}, selected, []string{"git"}),
		SourceInputs: selected,
	}
	if _, err := c.AdmitRequirement(t.Context(), factory.NewID(), "native:7", requirement); err != nil {
		t.Fatal(err)
	}
	approval := project.ApprovalDecision{
		ID: "d" + strings.Repeat("2", 24), Project: requirement.Project, Requirement: requirement.ID, Approver: 9,
		EffectsDigest: strings.Repeat("d", 64), ReadinessDigest: strings.Repeat("e", 64), Verified: true,
	}
	if _, err := c.AdmitApproval(t.Context(), factory.NewID(), "native:9", approval); err != nil {
		t.Fatal(err)
	}
	in := project.Prepare{
		Preparation: project.Preparation{
			ID: "f" + strings.Repeat("1", 24), Project: requirement.Project, Role: project.RoleCoder,
			Requirements: requirement.Ref(1), Approval: approval.Ref(1), SourceCommit: requirement.SourceCommit,
			SetupDigest: requirement.SetupDigest, Tools: []string{"git"},
		},
		Setup: project.ApprovedSetup{Files: files, Bundle: []byte("native source producer fixture")},
	}
	return c, in, host
}

func standingState(in project.Prepare, phase string) project.PrepareState {
	return project.PrepareState{
		ID: in.Preparation.ID, Project: in.Preparation.Project, Role: in.Preparation.Role,
		SourceCommit: in.Preparation.SourceCommit, SetupDigest: in.Preparation.SetupDigest,
		Container: strings.Repeat("c", 64), Phase: phase, Ready: phase == project.PrepareReady,
	}
}

func standingAllow(context.Context) error { return nil }

func TestStartPreparationAdmitsBeforeNativeEffectAndAdoptsLostReply(t *testing.T) {
	c, in, host := standingFixture(t)
	request, cancel := context.WithCancel(context.Background())
	defer cancel()
	var native project.PrepareState
	prepares, inspections, authority := 0, 0, 0
	host.inspect = func(ctx context.Context, _ project.PrepareInspect) (project.PrepareState, error) {
		inspections++
		if native.ID == "" {
			return project.PrepareState{}, project.ErrPreparationNotFound
		}
		if ctx.Err() != nil {
			t.Fatal("lost-reply observation inherited request cancellation")
		}
		return native, nil
	}
	host.prepare = func(ctx context.Context, got project.Prepare) (project.PrepareState, error) {
		prepares++
		stored, err := c.Store.Preparation(ctx, got.Preparation.ID)
		if err != nil || stored.State.ID != "" || stored.Preparation.ID != in.Preparation.ID {
			t.Fatalf("native effect preceded durable admission: %+v %v", stored, err)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 25*time.Second {
			t.Fatalf("unbounded preparation observation: %v %t", deadline, ok)
		}
		native = standingState(got, project.PrepareRunning)
		cancel()
		return project.PrepareState{}, request.Err()
	}
	stored, err := c.StartPreparation(request, in, func(context.Context) error { authority++; return nil })
	if err != nil || stored.State.Phase != project.PrepareRunning || authority != 2 || prepares != 1 || inspections != 2 {
		t.Fatalf("lost reply was not adopted: %+v %v auth=%d prepare=%d inspect=%d", stored, err, authority, prepares, inspections)
	}
	native = standingState(in, project.PrepareReady)
	// A later serving owner uses the same durable identity after a lost reply.
	restarted := &Coordinator{Store: c.Store, Host: host}
	stored, err = restarted.StartPreparation(t.Context(), in, standingAllow)
	if err != nil || !stored.State.Ready || prepares != 1 {
		t.Fatalf("ready replay restarted native setup: %+v %v prepare=%d", stored, err, prepares)
	}
}

func TestStartPreparationKeepsUnconfirmedAdmissionAndInspectsBeforeRetry(t *testing.T) {
	c, in, host := standingFixture(t)
	prepares, nativeKnown := 0, false
	unavailable := errors.New("native observation unavailable")
	host.inspect = func(context.Context, project.PrepareInspect) (project.PrepareState, error) {
		if !nativeKnown {
			if prepares == 0 {
				return project.PrepareState{}, project.ErrPreparationNotFound
			}
			return project.PrepareState{}, unavailable
		}
		return standingState(in, project.PrepareRunning), nil
	}
	host.prepare = func(context.Context, project.Prepare) (project.PrepareState, error) {
		prepares++
		return project.PrepareState{}, errors.New("lost native reply")
	}
	if _, err := c.StartPreparation(t.Context(), in, standingAllow); !errors.Is(err, unavailable) {
		t.Fatalf("unconfirmed outcome was hidden: %v", err)
	}
	stored, err := c.Store.Preparation(t.Context(), in.Preparation.ID)
	if err != nil || stored.State.ID != "" || stored.Preparation.ID != in.Preparation.ID {
		t.Fatalf("uncertain admission was removed or marked ready: %+v %v", stored, err)
	}
	nativeKnown = true
	stored, err = c.StartPreparation(t.Context(), in, standingAllow)
	if err != nil || stored.State.Phase != project.PrepareRunning || prepares != 1 {
		t.Fatalf("uncertain retry launched twice: %+v %v prepares=%d", stored, err, prepares)
	}
}

func TestStartPreparationResumesSameWaitingOrApprovedIdentity(t *testing.T) {
	for _, phase := range []string{project.PrepareWaiting, project.PrepareApproved} {
		t.Run(phase, func(t *testing.T) {
			c, in, host := standingFixture(t)
			var native project.PrepareState
			prepares := 0
			host.inspect = func(context.Context, project.PrepareInspect) (project.PrepareState, error) {
				if native.ID == "" {
					return project.PrepareState{}, project.ErrPreparationNotFound
				}
				return native, nil
			}
			host.prepare = func(_ context.Context, got project.Prepare) (project.PrepareState, error) {
				prepares++
				if !reflect.DeepEqual(got, in) {
					t.Fatal("resume changed the immutable native request")
				}
				if prepares == 1 {
					native = standingState(in, phase)
					native.Missing = "git"
				} else {
					// Only the native prerequisite observation changes after the
					// administrator installs the tool outside this control.
					native = standingState(in, project.PrepareReady)
				}
				return native, nil
			}
			stored, err := c.StartPreparation(t.Context(), in, standingAllow)
			if err != nil || stored.State.Phase != phase || stored.State.Missing != "git" {
				t.Fatalf("native prerequisite wait: %+v %v", stored, err)
			}
			stored, err = c.StartPreparation(t.Context(), in, standingAllow)
			if err != nil || !stored.State.Ready || prepares != 2 {
				t.Fatalf("same identity did not resume: %+v %v prepares=%d", stored, err, prepares)
			}
			records, err := c.Store.StandingProjectPreparations(t.Context(), in.Preparation.Project)
			if err != nil || len(records) != 1 || records[0].Preparation.ID != in.Preparation.ID {
				t.Fatalf("resume duplicated admission: %+v %v", records, err)
			}
		})
	}
}

func TestStartPreparationLateReadyCannotOverwriteTerminalObservation(t *testing.T) {
	for _, phase := range []string{project.PrepareFailed, project.PrepareStopped, project.PrepareInterrupted} {
		t.Run(phase, func(t *testing.T) {
			c, in, host := standingFixture(t)
			host.prepare = func(context.Context, project.Prepare) (project.PrepareState, error) {
				t.Fatal("late observation reached native prepare")
				return project.PrepareState{}, nil
			}
			host.inspect = func(ctx context.Context, _ project.PrepareInspect) (project.PrepareState, error) {
				ready := standingState(in, project.PrepareReady)
				current, err := c.Store.Preparation(ctx, in.Preparation.ID)
				if err != nil {
					t.Fatal(err)
				}
				// A newer exact native observation commits while this older
				// Ready response remains in flight.
				current.State = standingState(in, phase)
				if phase == project.PrepareStopped {
					current.State.Stopped = true
				}
				if err := c.Store.ObservePreparation(ctx, current); err != nil {
					t.Fatal(err)
				}
				return ready, nil
			}
			returned, err := c.StartPreparation(t.Context(), in, standingAllow)
			if err == nil || returned.State.Ready || returned.State.Phase != phase {
				t.Fatalf("late Ready replaced terminal observation: %+v %v", returned, err)
			}
			stored, err := c.Store.Preparation(t.Context(), in.Preparation.ID)
			if err != nil || stored.State.Ready || stored.State.Phase != phase {
				t.Fatalf("terminal custody was not retained: %+v %v", stored, err)
			}
			if _, err := c.StartPreparation(t.Context(), in, standingAllow); !errors.Is(err, ErrPreparationIntervention) {
				t.Fatalf("terminal identity resumed: %v", err)
			}
		})
	}
}

func TestStartPreparationRefusesChangedCurrentAuthorityBeforeEffect(t *testing.T) {
	for _, change := range []string{"requirement", "approval", "hold", "authority"} {
		t.Run(change, func(t *testing.T) {
			c, in, host := standingFixture(t)
			calls := 0
			host.inspect = func(context.Context, project.PrepareInspect) (project.PrepareState, error) {
				return project.PrepareState{}, project.ErrPreparationNotFound
			}
			host.prepare = func(context.Context, project.Prepare) (project.PrepareState, error) {
				t.Fatal("invalidated authority reached native effect")
				return project.PrepareState{}, nil
			}
			refused := errors.New("current administrator revoked")
			_, err := c.StartPreparation(t.Context(), in, func(ctx context.Context) error {
				calls++
				if calls != 2 {
					return nil
				}
				switch change {
				case "requirement":
					previous, err := c.Store.RequirementDecision(ctx, in.Preparation.Requirements.ID)
					if err != nil {
						return err
					}
					previous.Predecessor, previous.ID = previous.ID, "d"+strings.Repeat("3", 24)
					return c.Store.AdmitRequirementDecision(ctx, previous)
				case "approval":
					previous, err := c.Store.ApprovalDecision(ctx, in.Preparation.Approval.ID)
					if err != nil {
						return err
					}
					previous.Predecessor, previous.ID = previous.ID, "d"+strings.Repeat("3", 24)
					return c.Store.AdmitApprovalDecision(ctx, previous)
				case "hold":
					return c.Store.SaveMaintenanceHold(ctx, project.MaintenanceHold{Project: in.Preparation.Project, Hold: true})
				default:
					return refused
				}
			})
			if err == nil {
				t.Fatal("changed current admission was accepted")
			}
			stored, readErr := c.Store.Preparation(t.Context(), in.Preparation.ID)
			if readErr != nil || stored.State.ID != "" {
				t.Fatalf("refusal lost durable pending identity: %+v %v", stored, readErr)
			}
		})
	}
}

func TestStartPreparationRefusesUnacceptedInputsAndRetiredIdentity(t *testing.T) {
	for _, change := range []string{"source", "selected_inputs", "tools", "depth", "setup", "unverified", "missing_authority", "failed", "interrupted", "stopped", "foreign_native", "false_ready", "stopped_ready"} {
		t.Run(change, func(t *testing.T) {
			c, in, host := standingFixture(t)
			host.prepare = func(context.Context, project.Prepare) (project.PrepareState, error) {
				t.Fatal("refused input reached native prepare")
				return project.PrepareState{}, nil
			}
			host.inspect = func(context.Context, project.PrepareInspect) (project.PrepareState, error) {
				phase := project.PrepareReady
				switch change {
				case "failed":
					phase = project.PrepareFailed
				case "interrupted":
					phase = project.PrepareInterrupted
				case "stopped":
					phase = project.PrepareStopped
				}
				state := standingState(in, phase)
				if change == "foreign_native" {
					state.Role = project.RoleReviewer
				}
				if change == "false_ready" {
					state.Phase = project.PrepareRunning
				}
				if change == "stopped_ready" {
					state.Stopped = true
				}
				return state, nil
			}
			authorize := standingAllow
			switch change {
			case "source":
				in.Preparation.SourceCommit = strings.Repeat("c", 40)
			case "selected_inputs":
				in.Preparation.Requirements.SourceInputs["lockfile"] = strings.Repeat("b", 64)
			case "tools":
				in.Preparation.Tools = []string{"curl"}
			case "depth":
				in.Preparation.Requirements.Revision++
			case "setup":
				in.Setup.Files["setup.sh"] = []byte("#!/bin/sh\nfalse\n")
				in.Preparation.SetupDigest = project.SetupDigestOf(in.Setup.Files)
			case "unverified":
				approval, err := c.Store.ApprovalDecision(t.Context(), in.Preparation.Approval.ID)
				if err != nil {
					t.Fatal(err)
				}
				approval.Predecessor, approval.ID, approval.Verified = approval.ID, "d"+strings.Repeat("3", 24), false
				if err := c.Store.AdmitApprovalDecision(t.Context(), approval); err != nil {
					t.Fatal(err)
				}
				in.Preparation.Approval = approval.Ref(2)
			case "missing_authority":
				authorize = nil
			}
			_, err := c.StartPreparation(t.Context(), in, authorize)
			if err == nil {
				t.Fatal("unaccepted or retired preparation was accepted")
			}
			if change == "failed" || change == "interrupted" || change == "stopped" {
				if !errors.Is(err, ErrPreparationIntervention) {
					t.Fatalf("terminal native state did not preserve intervention: %v", err)
				}
				if _, err := c.StartPreparation(t.Context(), in, standingAllow); !errors.Is(err, ErrPreparationIntervention) {
					t.Fatalf("retired identity was reused: %v", err)
				}
			}
		})
	}
}

func TestStandingPreparationReplayBindsEntireImmutableIdentity(t *testing.T) {
	c, in, _ := standingFixture(t)
	if _, _, err := c.Store.AdmitPreparation(t.Context(), project.StoredPreparation{Preparation: in.Preparation}); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"project", "role", "tools", "credential"} {
		t.Run(change, func(t *testing.T) {
			p := in.Preparation
			switch change {
			case "project":
				p.Project = "p" + strings.Repeat("b", 24)
			case "role":
				p.Role = project.RoleReviewer
			case "tools":
				p.Tools = []string{"curl"}
			case "credential":
				p.Credential = "different"
			}
			if _, _, err := c.Store.AdmitPreparation(t.Context(), project.StoredPreparation{Preparation: p}); err == nil {
				t.Fatal("changed immutable identity replayed")
			}
		})
	}
	stored, err := c.Store.Preparation(t.Context(), in.Preparation.ID)
	if err != nil || stored.Preparation.Role != project.RoleCoder || stored.Preparation.Credential != "" || len(stored.Preparation.Tools) != 1 || stored.Preparation.Tools[0] != "git" {
		t.Fatalf("refused replay changed stored identity: %+v %v", stored, err)
	}
}

func TestStandingPreparationReplayPreservesAbsentSourceSelection(t *testing.T) {
	c, in, _ := standingFixture(t)
	in.Preparation.Requirements.SourceInputs = map[string]string{}
	if _, created, err := c.Store.AdmitPreparation(t.Context(), project.StoredPreparation{Preparation: in.Preparation}); err != nil || !created {
		t.Fatalf("initial exact-source-only admission: %t %v", created, err)
	}
	stored, created, err := c.Store.AdmitPreparation(t.Context(), project.StoredPreparation{Preparation: in.Preparation})
	if err != nil || created || len(stored.Preparation.Requirements.SourceInputs) != 0 {
		t.Fatalf("empty selection did not replay after JSON persistence: %+v %t %v", stored, created, err)
	}
}
