package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
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
