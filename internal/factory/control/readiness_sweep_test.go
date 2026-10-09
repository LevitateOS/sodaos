package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func TestReconcileDiscoversMissedIssue(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	observer := &fakeObserver{revision: 12, idle: true, pages: map[int]fakeIssuePage{
		1: {indexes: []int64{3}},
	}}
	c := readinessCoordinator(t, source, observer)
	grantFullAuthority(t, c)
	sweep, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if sweep.Seen != 1 || sweep.Assessed != 1 || sweep.Changed != 1 || sweep.Skipped != 0 {
		t.Fatal("discovery sweep wrong:", sweep)
	}
	control, err := c.Store.IssueControl(context.Background(), 42, 3)
	if err != nil || control.Readiness != factory.ReadinessNotAuthorized {
		t.Fatal("discovered issue not assessed:", control, err)
	}
	if findBlocker(control.Blockers, factory.BlockerAcceptanceMissing) == nil {
		t.Fatal("discovery blocker wrong:", control.Blockers)
	}
	revision, err := c.Store.ReadinessSweepRevision(context.Background(), 42)
	if err != nil || revision != 12 {
		t.Fatal("sweep revision not saved:", revision, err)
	}
	// Scans assess only: no initial acceptance is adopted without an
	// authenticated creation observation.
	if _, err = c.Store.AcceptanceHead(context.Background(), 42, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("sweep adopted an initial acceptance:", err)
	}
}

func TestReconcileSkipsUnchanged(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	observer := &fakeObserver{revision: 12, idle: true, pages: map[int]fakeIssuePage{
		1: {indexes: []int64{3}},
	}}
	c := readinessCoordinator(t, source, observer)
	grantFullAuthority(t, c)
	if _, err := c.ReconcileReadiness(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	calls := source.totalCalls()
	sweep, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if sweep.Seen != 1 || sweep.Skipped != 1 || sweep.Assessed != 0 {
		t.Fatal("unchanged sweep not skipped:", sweep)
	}
	if source.totalCalls() != calls {
		t.Fatal("skipped sweep re-read native state")
	}
}

func TestReconcileRetriesDependantsFromPrecheckedRoot(t *testing.T) {
	source := &fakeEvidenceSource{
		evidence: map[string]AcceptanceEvidence{
			"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
			"42/5": readinessEvidence(12, readinessView("5"), nil, []AcceptanceEdge{{Occurrence: "21", DependsOn: "8", Visible: true}}),
		},
		errs: map[string]error{"42/5": errors.New("temporary dependant read failure")},
	}
	observer := &fakeObserver{revision: 12, idle: true, pages: map[int]fakeIssuePage{1: {indexes: []int64{3}}}}
	c := readinessCoordinator(t, source, observer)
	grantFullAuthority(t, c)
	root := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	dependent := readinessDecision("d"+strings.Repeat("2", 24), 42, "5", 12)
	dependent.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 3,
		Outcome: factory.PrereqResult,
	}}
	mustAdmit(t, c, root)
	mustAdmit(t, c, dependent)

	first, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil || first.Failed != 1 || first.Changed != 0 {
		t.Fatalf("first sweep should record root then report downstream failure: %+v %v", first, err)
	}
	rootControl, err := c.Store.IssueControl(context.Background(), 42, 3)
	if err != nil || rootControl.Revision != 1 {
		t.Fatalf("root assessment was not retained: %+v %v", rootControl, err)
	}
	if _, err := c.Store.IssueControl(context.Background(), 42, 5); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("failed dependant unexpectedly recorded: %v", err)
	}

	delete(source.errs, "42/5")
	second, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil || second.Failed != 0 || second.Skipped != 1 || second.Assessed != 0 {
		t.Fatalf("prechecked root should retry descendants without reassessing itself: %+v %v", second, err)
	}
	rootControl, err = c.Store.IssueControl(context.Background(), 42, 3)
	if err != nil || rootControl.Revision != 1 {
		t.Fatalf("unchanged root retry recorded a spurious assessment: %+v %v", rootControl, err)
	}
	if _, err := c.Store.IssueControl(context.Background(), 42, 5); err != nil {
		t.Fatalf("prechecked root retry did not assess its dependant: %v", err)
	}
}

func TestReconcileDetectsNativeChange(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	observer := &fakeObserver{revision: 12, idle: true, pages: map[int]fakeIssuePage{
		1: {indexes: []int64{3}},
	}}
	c := readinessCoordinator(t, source, observer)
	grantFullAuthority(t, c)
	if _, err := c.ReconcileReadiness(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	observer.revision = 13
	source.evidence["42/3"] = readinessEvidence(13, readinessView("3"), nil, nil)
	sweep, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if sweep.Assessed != 1 || sweep.Changed != 0 {
		t.Fatal("revision change not reassessed:", sweep)
	}
	revision, err := c.Store.ReadinessSweepRevision(context.Background(), 42)
	if err != nil || revision != 13 {
		t.Fatal("sweep revision not advanced:", revision, err)
	}
}

func TestReconcileSkipsDisabledPolicy(t *testing.T) {
	c := readinessCoordinator(t, &fakeEvidenceSource{}, &fakeObserver{revision: 12, idle: true})
	sweep, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil || sweep.SkippedReason != factory.MissingPolicy {
		t.Fatal("policy-less sweep wrong:", sweep, err)
	}
	paused := grantPolicy()
	paused.Paused = true
	if err := c.Store.SaveRepositoryPolicy(context.Background(), paused); err != nil {
		t.Fatal(err)
	}
	sweep, err = c.ReconcileReadiness(context.Background(), 42)
	if err != nil || sweep.SkippedReason != factory.MissingPolicyPaused {
		t.Fatal("paused sweep wrong:", sweep, err)
	}
}

func TestReconcileBusyAndUnavailable(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12})
	grantFullAuthority(t, c)
	sweep, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil || !sweep.Busy || sweep.Seen != 0 {
		t.Fatal("busy sweep wrong:", sweep, err)
	}
	c.Readiness = &fakeObserver{revErr: errors.New("boom")}
	sweep, err = c.ReconcileReadiness(context.Background(), 42)
	if err != nil || sweep.Reason != ReadinessSweepUnavailable {
		t.Fatal("unavailable sweep wrong:", sweep, err)
	}
	unwired := coordinatorFixture(t, nil, nil)
	grantFullAuthority(t, unwired)
	if _, err = unwired.ReconcileReadiness(context.Background(), 42); err == nil {
		t.Fatal("unwired sweep ran")
	}
}

func TestReconcileRecordsIssueFailures(t *testing.T) {
	source := &fakeEvidenceSource{
		evidence: map[string]AcceptanceEvidence{
			"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
		},
		errs: map[string]error{"42/4": refuseAcceptance(RefusalNativeBusy)},
	}
	observer := &fakeObserver{revision: 12, idle: true, pages: map[int]fakeIssuePage{
		1: {indexes: []int64{3, 4}},
	}}
	c := readinessCoordinator(t, source, observer)
	grantFullAuthority(t, c)
	sweep, err := c.ReconcileReadiness(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if sweep.Failed != 1 || len(sweep.Failures) != 1 || sweep.Failures[0].Issue != 4 ||
		sweep.Failures[0].Reason != ReadinessSweepUnavailable {
		t.Fatal("issue failure not recorded:", sweep)
	}
	if sweep.Assessed != 1 || sweep.Changed != 1 {
		t.Fatal("sweep abandoned its healthy issue:", sweep)
	}
	if _, err = c.Store.ReadinessSweepRevision(context.Background(), 42); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("dirty sweep advanced its revision:", err)
	}
}

func TestSweepPrecheckCatchesWithdrawal(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	observer := &fakeObserver{revision: 12, idle: true, pages: map[int]fakeIssuePage{
		1: {indexes: []int64{3}},
	}}
	c := readinessCoordinator(t, source, observer)
	grantFullAuthority(t, c)
	setupCodePrereq(t, c, source)
	ctx := context.Background()
	first, err := c.ReconcileReadiness(ctx, 42)
	if err != nil || first.Changed != 1 {
		t.Fatal("first sweep wrong:", first, err)
	}
	endpointID := "d" + strings.Repeat("2", 24)
	if err := c.Store.WithdrawAcceptanceDecision(ctx, 42, 9, endpointID, 5); err != nil {
		t.Fatal(err)
	}
	second, err := c.ReconcileReadiness(ctx, 42)
	if err != nil {
		t.Fatal(err)
	}
	if second.Skipped != 0 || second.Assessed != 1 || second.Changed != 1 {
		t.Fatal("withdrawal skipped by precheck:", second)
	}
	control, err := c.Store.IssueControl(ctx, 42, 3)
	if err != nil || control.Readiness != factory.ReadinessNotAuthorized {
		t.Fatal("withdrawn route still blocked:", control, err)
	}
}

func TestFindPrereqCycle(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	chain := func(id, issue string, endpoint int64) {
		decision := readinessDecision(id, 42, issue, 12)
		decision.Prerequisites = []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: endpoint,
			Outcome: factory.PrereqResult,
		}}
		mustAdmit(t, c, decision)
	}
	chain("d"+strings.Repeat("1", 24), "3", 4)
	chain("d"+strings.Repeat("2", 24), "4", 5)
	if cycle, exceeded, err := c.findPrereqCycle(ctx, 42, 3); err != nil || exceeded || cycle != "" {
		t.Fatal("acyclic walk reported a cycle:", cycle, exceeded, err)
	}
	self := readinessDecision("d"+strings.Repeat("3", 24), 42, "6", 12)
	self.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 6,
		Outcome: factory.PrereqResult,
	}}
	mustAdmit(t, c, self)
	if cycle, _, err := c.findPrereqCycle(ctx, 42, 6); err != nil || cycle != "42/6 -> 42/6" {
		t.Fatal("self cycle wrong:", cycle, err)
	}
}

func TestFindPrereqCycleExceedsBound(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	for i := int64(0); i <= MaxCycleNodes+1; i++ {
		decision := readinessDecision(fmt.Sprintf("d%024x", i+1), 42, fmt.Sprint(1000+i), 12)
		decision.Prerequisites = []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 1000 + i + 1,
			Outcome: factory.PrereqResult,
		}}
		mustAdmit(t, c, decision)
	}
	if _, exceeded, err := c.findPrereqCycle(ctx, 42, 1000); err != nil || !exceeded {
		t.Fatal("over-bound walk not exceeded:", exceeded, err)
	}
}
