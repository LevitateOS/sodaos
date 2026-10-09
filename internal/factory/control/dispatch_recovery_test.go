package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
)

func TestRecoveryConsumesSettledRunWithoutHook(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 {
		t.Fatalf("first pass = %+v", first.Launched)
	}
	// The run settles without the accounting hook, as if the process died
	// between the run save and the hook.
	run, err := db.FactoryRun(ctx, first.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Failed, "boom", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Recovered) != 1 || second.Recovered[0] != first.Launched[0].AssignmentID {
		t.Fatalf("recovered = %+v errors = %+v", second.Recovered, second.Errors)
	}
	a, err := db.Assignment(ctx, first.Launched[0].AssignmentID)
	if err != nil || a.Stage != factory.AssignmentFinished || a.Outcome != factory.Failed {
		t.Fatalf("assignment = %+v %v", a, err)
	}
	if r, _ := db.Reservation(ctx, a.ID); r.State != factory.ReservationConsumed {
		t.Fatalf("reservation = %+v", r)
	}
	if total, _ := db.UsageTotal(ctx, 7, "conn"); total < 1 {
		t.Fatalf("usage = %d", total)
	}
}

func TestReleasedReviewerChildRetryKeepsRecordedPlan(t *testing.T) {
	ctx := context.Background()
	fx, parent, _, run := reviewRunSeed(t, false)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	var launches []project.FactoryLaunch
	host := &fakeDispatchHost{
		launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
			launches = append(launches, in)
			return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
		},
		harness: func(family string) (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: family, Version: fx.seed.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	deps := fx.seed.deps()
	deps.Host = host
	deps.Broker = &fakeDispatchBroker{}
	reservation, err := fx.db.Reservation(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unused, fenced := confirmUnused(ctx, deps, child, run); !unused || fenced {
		t.Fatalf("child run was not confirmed unused: unused=%t fenced=%t", unused, fenced)
	}
	report := DispatchReport{}
	recoverUnused(ctx, deps, child, reservation, run, &report)
	if len(launches) != 1 || launches[0].Run.Role != project.RoleReviewer ||
		launches[0].Run.Preparation != child.Preparation || launches[0].Run.HarnessVers != child.HarnessVers ||
		launches[0].Run.Model != child.Model || launches[0].Run.SourceCommit != child.SourceCommit || launches[0].Run.Connection != child.Connection ||
		!bytes.Equal(launches[0].Prompt, child.Prompt) || launches[0].Run.Assignment != child.PromptSHA {
		t.Fatalf("review retry changed its recorded plan: launches=%+v errors=%+v", launches, report.Errors)
	}
	after, err := fx.db.Assignment(ctx, child.ID)
	if err != nil || after.Attempts != 2 || after.AttemptRoot != parent.AttemptRoot ||
		after.PublicationAssignment != parent.ID || after.Role != project.RoleReviewer ||
		!bytes.Equal(after.Prompt, child.Prompt) || after.PromptSHA != child.PromptSHA {
		t.Fatalf("review retry changed child identity: %+v err=%v", after, err)
	}
}

func TestReleasedCorrectionChildRetryKeepsRecordedPlan(t *testing.T) {
	ctx := context.Background()
	fx, publication := checkSeed(t, 16)
	parent, err := fx.db.Assignment(ctx, publication.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	_, run := fx.childRun(t, parent, publication, project.RoleCoder, publication.Candidate, parent.Preparation, false)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	var launches []project.FactoryLaunch
	host := &fakeDispatchHost{
		launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
			launches = append(launches, in)
			return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
		},
		harness: func(family string) (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: family, Version: fx.seed.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	deps := fx.seed.deps()
	deps.Host = host
	deps.Broker = &fakeDispatchBroker{}
	reservation, err := fx.db.Reservation(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unused, fenced := confirmUnused(ctx, deps, child, run); !unused || fenced {
		t.Fatalf("correction child run was not confirmed unused: unused=%t fenced=%t", unused, fenced)
	}
	report := DispatchReport{}
	recoverUnused(ctx, deps, child, reservation, run, &report)
	if len(launches) != 1 || launches[0].Run.Role != project.RoleCoder ||
		launches[0].Run.Preparation != child.Preparation || launches[0].Run.Model != child.Model ||
		launches[0].Run.SourceCommit != child.SourceCommit || launches[0].Run.Connection != child.Connection ||
		!bytes.Equal(launches[0].Prompt, child.Prompt) || launches[0].Run.Assignment != child.PromptSHA {
		t.Fatalf("correction retry changed its recorded plan: launches=%+v errors=%+v", launches, report.Errors)
	}
	after, err := fx.db.Assignment(ctx, child.ID)
	if err != nil || after.Attempts != 2 || after.AttemptRoot != parent.AttemptRoot ||
		after.PublicationAssignment != parent.ID || after.Role != project.RoleCoder || !bytes.Equal(after.Prompt, child.Prompt) {
		t.Fatalf("correction retry changed child identity: %+v err=%v", after, err)
	}
}

func TestReleasedReviewerRetryAcceptsCurrentAuthorityAndNativeRevision(t *testing.T) {
	ctx := context.Background()
	fx, _, _, run := reviewRunSeed(t, false)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	capacity, err := fx.db.Capacity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	capacity.MaxConcurrentRuns++
	if err := fx.db.SaveCapacity(ctx, capacity); err != nil {
		t.Fatal(err)
	}
	budget, err := fx.db.ConnectionUsageBudget(ctx, child.Connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := fx.db.SaveConnectionUsageBudget(ctx, budget); err != nil {
		t.Fatal(err)
	}
	control, err := fx.db.IssueControl(ctx, child.Repository, child.Issue)
	if err != nil {
		t.Fatal(err)
	}
	control.NativeRev++
	control.Fingerprint = strings.Repeat("4", 64)
	if _, _, err := fx.db.RecordIssueAssessment(ctx, control, time.Now()); err != nil {
		t.Fatal(err)
	}
	inputKey := fmt.Sprintf("%d/%d", child.Repository, child.Issue)
	inputs := fx.seed.reads.inputs[inputKey]
	inputs.Revision = control.NativeRev
	fx.seed.reads.inputs[inputKey] = inputs
	var launches []project.FactoryLaunch
	host := &fakeDispatchHost{
		launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
			launches = append(launches, in)
			return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
		},
		harness: func(family string) (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: family, Version: fx.seed.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	deps := fx.seed.deps()
	deps.Host = host
	deps.Broker = &fakeDispatchBroker{}
	reservation, err := fx.db.Reservation(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unused, fenced := confirmUnused(ctx, deps, child, run); !unused || fenced {
		t.Fatalf("reviewer run was not confirmed unused: unused=%t fenced=%t", unused, fenced)
	}
	report := DispatchReport{}
	recoverUnused(ctx, deps, child, reservation, run, &report)
	if len(launches) != 1 || launches[0].Run.Role != project.RoleReviewer || !bytes.Equal(launches[0].Prompt, child.Prompt) {
		t.Fatalf("retry did not retain the current-authorized child plan: launches=%+v errors=%+v", launches, report.Errors)
	}
	after, err := fx.db.Assignment(ctx, child.ID)
	if err != nil || after.Attempts != child.Attempts+1 || after.Authority != child.Authority ||
		after.AttemptRoot != child.AttemptRoot || after.PublicationAssignment != child.PublicationAssignment ||
		after.SourceCommit != child.SourceCommit || after.PromptSHA != child.PromptSHA || !bytes.Equal(after.Prompt, child.Prompt) ||
		len(after.RunHistory) != len(child.RunHistory)+1 || !slices.Equal(after.RunHistory[:len(child.RunHistory)], child.RunHistory) {
		t.Fatalf("retry changed the child assignment plan: before=%+v after=%+v err=%v", child, after, err)
	}
	retryRun, err := fx.db.FactoryRun(ctx, after.Run)
	if err != nil || retryRun.Admission == nil {
		t.Fatalf("retry run admission is missing: %+v err=%v", retryRun.Admission, err)
	}
	currentCapacity, err := fx.db.Capacity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	currentBudget, err := fx.db.ConnectionUsageBudget(ctx, child.Connection)
	if err != nil {
		t.Fatal(err)
	}
	if retryRun.Admission.Authority.Capacity != currentCapacity.Revision ||
		retryRun.Admission.Authority.ConnectionUsageBudget != currentBudget.Revision {
		t.Fatalf("retry run did not capture current capacity/budget revisions: admission=%+v capacity=%d budget=%d", retryRun.Admission.Authority, currentCapacity.Revision, currentBudget.Revision)
	}
}

func TestReleasedReviewerRetryRefusesLostSponsorship(t *testing.T) {
	for name, change := range map[string]func(*factory.Sponsorship){
		"withdrawn":             func(s *factory.Sponsorship) { s.Active = false },
		"reviewer role removed": func(s *factory.Sponsorship) { s.Roles = []string{project.RoleCoder} },
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			fx, _, _, run := reviewRunSeed(t, false)
			child, err := fx.db.AssignmentByRun(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			sponsorship, err := fx.db.Sponsorship(ctx, child.Repository, child.Connection)
			if err != nil {
				t.Fatal(err)
			}
			change(&sponsorship)
			if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
				t.Fatal(err)
			}
			var launches []project.FactoryLaunch
			host := &fakeDispatchHost{
				launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
					launches = append(launches, in)
					return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
				},
				harness: func(family string) (project.FactoryHarnessPin, error) {
					return project.FactoryHarnessPin{Harness: family, Version: fx.seed.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
				},
			}
			deps := fx.seed.deps()
			deps.Host = host
			deps.Broker = &fakeDispatchBroker{}
			reservation, err := fx.db.Reservation(ctx, child.ID)
			if err != nil {
				t.Fatal(err)
			}
			if unused, fenced := confirmUnused(ctx, deps, child, run); !unused || fenced {
				t.Fatalf("reviewer run was not confirmed unused: unused=%t fenced=%t", unused, fenced)
			}
			report := DispatchReport{}
			recoverUnused(ctx, deps, child, reservation, run, &report)
			if len(launches) != 0 {
				t.Fatalf("child launched after sponsorship change: %+v", launches)
			}
			after, err := fx.db.Assignment(ctx, child.ID)
			if err != nil || after.Stage != factory.AssignmentFinished || after.Reason != factory.AssignReasonWithdrawn {
				t.Fatalf("unauthorized child was not retired: %+v err=%v", after, err)
			}
		})
	}
}

func TestReleasedReviewerRetryRefusesChangedRequiredChecks(t *testing.T) {
	ctx := context.Background()
	fx, _, _, run := reviewRunSeed(t, false)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := fx.db.RepositoryPolicy(ctx, child.Repository)
	if err != nil {
		t.Fatal(err)
	}
	policy.Checks = []string{"new-required-check"}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	var launches []project.FactoryLaunch
	host := &fakeDispatchHost{
		launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
			launches = append(launches, in)
			return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
		},
		harness: func(family string) (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: family, Version: fx.seed.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	deps := fx.seed.deps()
	deps.Host = host
	deps.Broker = &fakeDispatchBroker{}
	reservation, err := fx.db.Reservation(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unused, fenced := confirmUnused(ctx, deps, child, run); !unused || fenced {
		t.Fatalf("reviewer run was not confirmed unused: unused=%t fenced=%t", unused, fenced)
	}
	report := DispatchReport{}
	recoverUnused(ctx, deps, child, reservation, run, &report)
	if len(launches) != 0 {
		t.Fatalf("child launched with obsolete required checks: %+v", launches)
	}
	after, err := fx.db.Assignment(ctx, child.ID)
	if err != nil || after.Stage != factory.AssignmentFinished || after.Reason != factory.AssignReasonSuperseded {
		t.Fatalf("obsolete child was not retired: %+v err=%v", after, err)
	}
}

func TestReleasedChildRetryRefusesSupersededPublication(t *testing.T) {
	ctx := context.Background()
	fx, publication := checkSeed(t, 16)
	parent, err := fx.db.Assignment(ctx, publication.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	correction := correctionRun(t, fx, parent, strings.Repeat("d", 40))
	// Keep the publisher whose target-tip ledger was populated by checkSeed's
	// initial publication; a fresh happyPublisher would observe an empty ref.
	exec := fx.exec
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	pending := fx.coord.PublishCorrection(ctx, correction.ID)
	if len(pending.Waits) != 1 || pending.Waits[0].Reason != "push_unconfirmed" {
		t.Fatalf("correction intent was not left pending: %+v", pending)
	}
	stored, err := fx.db.PublicationByAssignment(ctx, parent.ID)
	if err != nil || len(stored.Corrections) != 1 || stored.Candidate != publication.Candidate ||
		stored.Corrections[0].Effect != factory.OpEffectPending || stored.Corrections[0].Work == nil {
		t.Fatalf("pending correction intent is not recorded against the old head: %+v %v", stored, err)
	}
	sponsorship, err := fx.db.Sponsorship(ctx, parent.Repository, parent.Connection)
	if err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = append(sponsorship.Roles, project.RoleReviewer)
	if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
		t.Fatal(err)
	}
	_, reviewRun := fx.childRun(t, parent, publication, project.RoleReviewer, publication.Candidate, "f222222222222222222222222", false)
	staleChild, err := fx.db.AssignmentByRun(ctx, reviewRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	work := exec.pushes[len(exec.pushes)-1]
	exec.ledger[work.OperationID] = publicationTestOutcome(work, factory.OpRefPublish, committedOutcome("correction-receipt"))
	advanced := fx.coord.PublishCorrection(ctx, correction.ID)
	if len(advanced.Corrected) != 1 || advanced.Corrected[0].HeadOID != strings.Repeat("d", 40) {
		t.Fatalf("recorded correction was not reconciled: %+v", advanced)
	}
	var launches []project.FactoryLaunch
	host := &fakeDispatchHost{
		launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
			launches = append(launches, in)
			return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
		},
		harness: func(family string) (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: family, Version: fx.seed.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	deps := fx.seed.deps()
	deps.Host = host
	deps.Broker = &fakeDispatchBroker{}
	reservation, err := fx.db.Reservation(ctx, staleChild.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unused, fenced := confirmUnused(ctx, deps, staleChild, reviewRun); !unused || fenced {
		t.Fatalf("stale child run was not confirmed unused: unused=%t fenced=%t", unused, fenced)
	}
	report := DispatchReport{}
	recoverUnused(ctx, deps, staleChild, reservation, reviewRun, &report)
	if len(launches) != 0 {
		t.Fatalf("stale reviewer child launched: %+v", launches)
	}
	after, err := fx.db.Assignment(ctx, staleChild.ID)
	if err != nil || after.Stage != factory.AssignmentFinished || after.Reason != factory.AssignReasonSuperseded {
		t.Fatalf("stale child was not retired: %+v err=%v", after, err)
	}
	if len(report.Errors) != 0 || len(report.Waits) != 0 {
		t.Fatalf("stale child retry report: %+v", report)
	}
}

func TestCompletionTriggersDependantReassessment(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	endpointHead := fx.accept(t, 3, "d333333333333333333333333")
	endpoint := endpointHead
	// The dependant adopts the endpoint as a code prerequisite.
	dependent := factory.Acceptance{
		ID: "d555555555555555555555555", Repository: fx.repo, IssueIndex: "5", Approver: 5, NativeRev: 41,
		TitleDigest: dispatchDigest("dep title"), ContentDigest: dispatchDigest("dep body"), ContentVersion: 1,
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: "100", DependsOn: "200", PrereqAcceptance: endpointHead.ID,
			EndpointRepo: fx.repo, EndpointIssue: 3, Outcome: factory.PrereqCode,
		}},
	}
	if err := db.AdmitAcceptanceDecision(ctx, dependent); err != nil {
		t.Fatal(err)
	}
	source := &fakeAcceptanceSource{evidence: map[string]AcceptanceEvidence{
		"7/5": {
			Issue:        AcceptanceIssueView{Index: "5", TitleDigest: dependent.TitleDigest, ContentDigest: dependent.ContentDigest, ContentVer: 1, Visible: true},
			Dependencies: []AcceptanceEdge{{Occurrence: "100", DependsOn: "200", Visible: true}},
			Revision:     41,
		},
		"7/3": {
			Issue: AcceptanceIssueView{
				Index: "3", TitleDigest: endpoint.TitleDigest, ContentDigest: endpoint.ContentDigest,
				ContentVer: endpoint.ContentVersion, Visible: true,
			},
			Comments: []AcceptanceComment{{ID: "11", Digest: endpoint.Sources[0].Digest, ContentVer: 0, Visible: true}},
			Revision: 41,
		},
	}}
	coord := &Coordinator{Store: db, AcceptanceReads: source}
	outcome, err := coord.assessCascade(ctx, fx.repo, 5, map[factory.DependenceRef]bool{})
	if err != nil || !outcome.changed || outcome.control.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant assessment = %+v %v", outcome, err)
	}
	revision := outcome.control.Revision
	reads := source.reads

	fx.queue(t, 3, endpointHead.ID)
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 1 {
		t.Fatalf("launched = %+v", report.Launched)
	}
	run, err := db.FactoryRun(ctx, report.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
	if !ok || finished.Outcome != factory.NeedsHuman {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	coord.assessDispatchDependants(ctx, fx.repo, 3)
	if source.reads <= reads {
		t.Fatal("dependant was not reassessed after completion")
	}
	after, err := db.IssueControl(ctx, fx.repo, 5)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != revision || after.Readiness != factory.ReadinessBlocked || after.Reason != factory.BlockerCodePending {
		t.Fatalf("dependant after completion = %+v", after)
	}
}

func TestHostNotFoundMatcherPinsHostSentinel(t *testing.T) {
	if !isHostNotFound(host.ErrRunNotFound) {
		t.Fatal("host sentinel not recognized")
	}
	if !isHostNotFound(fmt.Errorf("inspect: %w", host.ErrRunNotFound)) {
		t.Fatal("wrapped host sentinel not recognized")
	}
	if isHostNotFound(errors.New("boom")) || isHostNotFound(nil) {
		t.Fatal("unrelated error recognized as host miss")
	}
}

func TestIntakeTriggersAutomaticDispatch(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	decision := fx.accept(t, 3, "d333333333333333333333333")
	source := &fakeAcceptanceSource{evidence: map[string]AcceptanceEvidence{
		"7/3": {
			Issue: AcceptanceIssueView{
				Index: "3", TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest,
				ContentVer: decision.ContentVersion, Visible: true,
			},
			Comments: []AcceptanceComment{{
				ID: "11", Digest: decision.Sources[0].Digest,
				ContentVer: decision.Sources[0].ContentVersion, Visible: true,
			}},
			Revision: 41,
		},
	}}
	coord := &Coordinator{Store: db, Host: fx.host, Broker: fx.broker, AcceptanceReads: source, DispatchReads: fx.reads}
	control, changed, err := coord.ObserveIssueEvent(ctx, IntakeHint{Delivery: "st08-intake-1", Repository: fx.repo, Issue: 3})
	if err != nil || !changed || control.Readiness != factory.ReadinessQueued {
		t.Fatalf("intake = %+v %v %v", control, changed, err)
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("automatic launches = %d", len(fx.host.launches))
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil || assignment.Run != fx.host.launches[0].Run.ID || assignment.Stage != factory.AssignmentAssigned {
		t.Fatalf("latest assignment = %+v %v", assignment, err)
	}
	// A duplicate delivery replays without dispatching again.
	again, changed, err := coord.ObserveIssueEvent(ctx, IntakeHint{Delivery: "st08-intake-1", Repository: fx.repo, Issue: 3})
	if err != nil || changed || again.Revision != control.Revision {
		t.Fatalf("replay = %+v %v %v", again, changed, err)
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("launches after replay = %d", len(fx.host.launches))
	}
}

func TestAssessedResultPrerequisiteAppearsInRecordedDispatchPrompt(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	endpoint := fx.accept(t, 3, "d333333333333333333333333")
	dependent := fx.accept(t, 5, "d555555555555555555555555")
	dependent.Predecessor = dependent.ID
	dependent.ID = "d666666666666666666666666"
	dependent.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "9", EndpointRepo: fx.repo, EndpointIssue: 3,
		Outcome: factory.PrereqResult,
	}}
	resolution := "accepted resolution"
	dependent.Resolutions = []factory.SelectedSource{{ID: "31", ContentVersion: 0, Digest: dispatchDigest(resolution)}}
	if err := db.AdmitAcceptanceDecision(ctx, dependent); err != nil {
		t.Fatal(err)
	}
	fx.decision[5] = dependent
	dependentInput := fx.reads.inputs["7/5"]
	dependentInput.Comments = append(dependentInput.Comments, DispatchComment{ID: "31", Content: resolution, Digest: dispatchDigest(resolution), Visible: true})
	fx.reads.inputs["7/5"] = dependentInput
	endpointIssue := AcceptanceIssueView{
		Index: "3", TitleDigest: endpoint.TitleDigest, ContentDigest: endpoint.ContentDigest,
		ContentVer: endpoint.ContentVersion, Lifecycle: 4, ClosedUnix: 1_800_000_000,
		Closed: true, Visible: true,
	}
	source := &fakeAcceptanceSource{evidence: map[string]AcceptanceEvidence{
		"7/3": {
			Issue:    endpointIssue,
			Comments: []AcceptanceComment{{ID: "11", Digest: endpoint.Sources[0].Digest, Visible: true}},
			Revision: 41,
		},
		"7/5": {
			Issue: AcceptanceIssueView{
				Index: "5", TitleDigest: dependent.TitleDigest, ContentDigest: dependent.ContentDigest,
				ContentVer: dependent.ContentVersion, Visible: true,
			},
			Comments: []AcceptanceComment{
				{ID: "11", Digest: dependent.Sources[0].Digest, Visible: true},
				{ID: "31", Digest: dispatchDigest(resolution), Visible: true},
			},
			Dependencies: []AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
			Revision:     41,
		},
	}}
	coord := &Coordinator{
		Store: db, Host: fx.host, Broker: fx.broker,
		AcceptanceReads: source, DispatchReads: fx.reads,
	}
	control, changed, err := coord.ObserveIssueEvent(ctx, IntakeHint{Delivery: "result-dependency-dispatch", Repository: fx.repo, Issue: 5})
	if err != nil || !changed || control.Readiness != factory.ReadinessQueued {
		t.Fatalf("dependent readiness = %+v changed=%v err=%v", control, changed, err)
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("dependency dispatch launches = %d", len(fx.host.launches))
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 5)
	if err != nil || assignment.Stage != factory.AssignmentAssigned {
		t.Fatalf("dependent assignment = %+v err=%v", assignment, err)
	}
	for _, want := range []string{
		"## Accepted prerequisites", "Occurrence 21: depends on issue 9 at repository 7, issue 3; outcome result",
		"satisfied as of queued readiness revision " + fmt.Sprint(control.Revision),
		"lifecycle 4 closed at 1800000000 under acceptance " + dependent.ID,
	} {
		if !strings.Contains(string(assignment.Prompt), want) {
			t.Errorf("recorded dependent prompt lacks %q", want)
		}
	}
}
