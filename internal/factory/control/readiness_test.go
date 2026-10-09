package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func TestIntakeHintValidate(t *testing.T) {
	if err := readinessHint("d1", 42, 3).Validate(); err != nil {
		t.Fatal("valid hint refused:", err)
	}
	created := IntakeHint{Delivery: "d1", Repository: 42, Issue: 3, Created: true, Creator: 5, CreatorWrite: true}
	if err := created.Validate(); err != nil {
		t.Fatal("valid creation hint refused:", err)
	}
	for name, hint := range map[string]IntakeHint{
		"delivery":  {Repository: 42, Issue: 3},
		"scope":     {Delivery: "d1", Issue: 3},
		"creator":   {Delivery: "d1", Repository: 42, Issue: 3, Created: true},
		"authority": {Delivery: "d1", Repository: 42, Issue: 3, CreatorWrite: true},
		"control":   {Delivery: "d\x001", Repository: 42, Issue: 3},
	} {
		if err := hint.Validate(); err == nil {
			t.Error("invalid hint accepted:", name)
		}
	}
}

func TestObserveIssueEventCreationQueues(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	view := readinessView("3")
	view.ContentVer, view.Lifecycle = 0, 0
	view.Verified, view.FirstCreated = true, true
	source.evidence["42/3"] = readinessEvidence(12, view, nil, nil)
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	assessed, changed, err := c.ObserveIssueEvent(context.Background(), IntakeHint{
		Delivery: "delivery-1", Repository: 42, Issue: 3, Created: true, Creator: 5, CreatorWrite: true,
	})
	if err != nil || !changed {
		t.Fatal("creation intake failed:", assessed, changed, err)
	}
	if assessed.Readiness != factory.ReadinessQueued || assessed.Reason != factory.ReasonEligible {
		t.Fatal("created issue not queued:", assessed)
	}
	head, err := c.Store.AcceptanceHead(context.Background(), 42, 3)
	if err != nil || head != factory.InitialAcceptanceID(42, "3") {
		t.Fatal("initial acceptance not recorded:", head, err)
	}
	if assessed.Acceptance != head || assessed.Revision != 1 {
		t.Fatal("control identity wrong:", assessed)
	}
}

func TestObserveIssueEventDuplicateSuppresses(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	hint := readinessHint("delivery-1", 42, 3)
	first, changed, err := c.ObserveIssueEvent(context.Background(), hint)
	if err != nil || !changed {
		t.Fatal("first intake failed:", first, changed, err)
	}
	calls := source.totalCalls()
	second, changed, err := c.ObserveIssueEvent(context.Background(), hint)
	if err != nil || changed {
		t.Fatal("duplicate intake reassessed:", second, changed, err)
	}
	if second.Revision != first.Revision || source.totalCalls() != calls {
		t.Fatal("duplicate intake re-read native state")
	}
}

func TestObserveIssueEventUnchangedBlockerSuppresses(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	endpoint := readinessDecision("d"+strings.Repeat("2", 24), 42, "9", 12)
	mustAdmit(t, c, endpoint)
	dependent := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	dependent.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 9,
		Outcome: factory.PrereqCode, PrereqAcceptance: endpoint.ID,
	}}
	mustAdmit(t, c, dependent)
	edge := []AcceptanceEdge{{Occurrence: "21", DependsOn: "8", Visible: true}}
	source.evidence["42/3"] = readinessEvidence(12, readinessView("3"), nil, edge)
	source.evidence["42/9"] = readinessEvidence(12, readinessView("9"), nil, nil)
	first, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil || !changed || first.Readiness != factory.ReadinessBlocked {
		t.Fatal("blocked intake failed:", first, changed, err)
	}
	if findBlocker(first.Blockers, factory.BlockerCodePending) == nil {
		t.Fatal("code blocker missing:", first.Blockers)
	}
	calls := source.totalCalls()
	second, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-2", 42, 3))
	if err != nil || changed {
		t.Fatal("unchanged blocker reassessed:", second, changed, err)
	}
	if second.Revision != first.Revision || second.Fingerprint != first.Fingerprint {
		t.Fatal("unchanged blocker advanced the record")
	}
	if source.totalCalls() == calls {
		t.Fatal("second observation read nothing; suppression must compare fresh evidence")
	}
}

type acceptanceEvidenceDeadline struct {
	repository, issue string
	deadline          time.Time
	observedAt        time.Time
	hasDeadline       bool
}

type deadlineRecordingAcceptanceSource struct {
	delegate *fakeEvidenceSource
	reads    []acceptanceEvidenceDeadline
}

func (s *deadlineRecordingAcceptanceSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (AcceptanceEvidence, error) {
	deadline, ok := ctx.Deadline()
	s.reads = append(s.reads, acceptanceEvidenceDeadline{
		repository: repository, issue: issue, deadline: deadline, observedAt: time.Now(), hasDeadline: ok,
	})
	return s.delegate.ReadAcceptanceEvidence(ctx, repository, issue, commentIDs)
}

func TestObserveIssueEventBoundsPrerequisiteEvidenceRead(t *testing.T) {
	baseSource := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, baseSource, &fakeObserver{revision: 12, idle: true})
	root := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	root.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 9,
		Outcome: factory.PrereqResult,
	}}
	mustAdmit(t, c, root)
	rootEdge := []AcceptanceEdge{{Occurrence: "21", DependsOn: "8", Visible: true}}
	baseSource.evidence["42/3"] = readinessEvidence(12, readinessView("3"), nil, rootEdge)
	baseSource.evidence["42/9"] = readinessEvidence(12, readinessView("9"), nil, nil)
	source := &deadlineRecordingAcceptanceSource{delegate: baseSource}
	c.AcceptanceReads = source
	grantFullAuthority(t, c)

	if _, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("deadline-1", 42, 3)); err != nil {
		t.Fatal("background intake assessment failed:", err)
	}
	if len(source.reads) != 2 {
		t.Fatalf("expected root and prerequisite evidence reads, got %+v", source.reads)
	}
	for _, key := range [][2]string{{"42", "3"}, {"42", "9"}} {
		var found bool
		for _, read := range source.reads {
			if read.repository != key[0] || read.issue != key[1] {
				continue
			}
			found = true
			if !read.hasDeadline {
				t.Errorf("evidence read %s/%s has no assessment deadline", key[0], key[1])
			} else if read.deadline.After(read.observedAt.Add(2 * time.Minute)) {
				t.Errorf("evidence read %s/%s exceeds the two-minute assessment budget: %s", key[0], key[1], read.deadline)
			}
		}
		if !found {
			t.Errorf("missing evidence read %s/%s: %+v", key[0], key[1], source.reads)
		}
	}
}

func TestObserveIssueEventRetriesDependantsAfterUnchangedRoot(t *testing.T) {
	source := &fakeEvidenceSource{
		evidence: map[string]AcceptanceEvidence{
			"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
			"42/5": readinessEvidence(12, readinessView("5"), nil, []AcceptanceEdge{{Occurrence: "21", DependsOn: "8", Visible: true}}),
		},
		errs: map[string]error{"42/5": errors.New("temporary dependant read failure")},
	}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	root := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	dependent := readinessDecision("d"+strings.Repeat("2", 24), 42, "5", 12)
	dependent.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 3,
		Outcome: factory.PrereqResult,
	}}
	mustAdmit(t, c, root)
	mustAdmit(t, c, dependent)

	hint := readinessHint("delivery-retry", 42, 3)
	first, changed, err := c.ObserveIssueEvent(context.Background(), hint)
	if err == nil || changed || first.Revision != 0 {
		t.Fatalf("downstream failure should return no successful event outcome: %+v changed=%v err=%v", first, changed, err)
	}
	storedRoot, err := c.Store.IssueControl(context.Background(), 42, 3)
	if err != nil || storedRoot.Revision != 1 {
		t.Fatalf("root assessment was not retained: %+v %v", storedRoot, err)
	}
	if _, err := c.Store.IssueControl(context.Background(), 42, 5); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("failed dependant unexpectedly recorded: %v", err)
	}

	delete(source.errs, "42/5")
	second, changed, err := c.ObserveIssueEvent(context.Background(), hint)
	if err != nil || changed {
		t.Fatalf("unchanged root retry should complete dependant traversal without changing root: %+v changed=%v err=%v", second, changed, err)
	}
	storedRoot, err = c.Store.IssueControl(context.Background(), 42, 3)
	if err != nil || storedRoot.Revision != 1 {
		t.Fatalf("unchanged root retry recorded a spurious assessment: %+v %v", storedRoot, err)
	}
	if _, err := c.Store.IssueControl(context.Background(), 42, 5); err != nil {
		t.Fatalf("unchanged root retry did not assess its dependant: %v", err)
	}
}

func TestCreationWithoutAuthorityWaitsForAdoption(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	view := readinessView("3")
	view.Verified, view.FirstCreated = true, true
	source.evidence["42/3"] = readinessEvidence(12, view, nil, nil)
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	assessed, changed, err := c.ObserveIssueEvent(context.Background(), IntakeHint{
		Delivery: "delivery-1", Repository: 42, Issue: 3, Created: true, Creator: 5,
	})
	if err != nil || !changed {
		t.Fatal("outsider creation intake failed:", assessed, changed, err)
	}
	if assessed.Readiness != factory.ReadinessNotAuthorized {
		t.Fatal("outsider creation authorized:", assessed)
	}
	if findBlocker(assessed.Blockers, factory.BlockerAcceptanceMissing) == nil {
		t.Fatal("adoption blocker missing:", assessed.Blockers)
	}
	if _, err = c.Store.AcceptanceHead(context.Background(), 42, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("outsider creation adopted:", err)
	}
}

func TestEditedCreationNeedsExplicitAdoption(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	view := readinessView("3")
	view.Verified, view.FirstCreated, view.ContentVer = true, true, 1
	source.evidence["42/3"] = readinessEvidence(12, view, nil, nil)
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	assessed, _, err := c.ObserveIssueEvent(context.Background(), IntakeHint{
		Delivery: "delivery-1", Repository: 42, Issue: 3, Created: true, Creator: 5, CreatorWrite: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessNotAuthorized ||
		findBlocker(assessed.Blockers, factory.BlockerAcceptanceMissing) == nil {
		t.Fatal("edited creation not awaiting adoption:", assessed)
	}
	if _, err = c.Store.AcceptanceHead(context.Background(), 42, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("edited creation adopted:", err)
	}
}

func TestPullHintSkips(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	view := readinessView("3")
	view.IsPull = true
	source.evidence["42/3"] = readinessEvidence(12, view, nil, nil)
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	assessed, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil || changed || assessed.Readiness != "" {
		t.Fatal("pull hint recorded:", assessed, changed, err)
	}
	if _, err = c.Store.IssueControl(context.Background(), 42, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("pull control recorded:", err)
	}
	again, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil || changed || again.Readiness != "" {
		t.Fatal("duplicate pull hint recorded:", again, changed, err)
	}
}

func TestObserveWithoutReadsRefuses(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	grantFullAuthority(t, c)
	if _, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3)); err == nil {
		t.Fatal("unwired observation assessed")
	}
	seen, err := c.Store.IntakeDeliverySeen(context.Background(), "delivery-1")
	if err != nil || seen {
		t.Fatal("failed intake logged its delivery:", seen, err)
	}
}

func TestReadinessVerdictPrecedence(t *testing.T) {
	verdict, reason := readinessVerdict(nil)
	if verdict != factory.ReadinessQueued || reason != factory.ReasonEligible {
		t.Fatal("empty verdict wrong:", verdict, reason)
	}
	verdict, reason = readinessVerdict([]factory.Blocker{
		acceptanceBlocker(factory.BlockerCodePending, ""),
		acceptanceBlocker(factory.BlockerAuthorityMissing, factory.MissingPolicy),
	})
	if verdict != factory.ReadinessNotAuthorized || reason != factory.BlockerAuthorityMissing {
		t.Fatal("authority precedence wrong:", verdict, reason)
	}
	verdict, reason = readinessVerdict([]factory.Blocker{
		acceptanceBlocker(factory.BlockerCodePending, ""),
		acceptanceBlocker(factory.BlockerCycle, "42/3 -> 42/3"),
	})
	if verdict != factory.ReadinessBlocked || reason != factory.BlockerCodePending {
		t.Fatal("blocker precedence wrong:", verdict, reason)
	}
}

func TestIntakeBudgetYieldRetainsCursorAndRedeliveryResumes(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	hint := readinessHint("delivery-budget-resume", 42, 3)
	deliveryID := readinessDeliverySourceID(hint.Delivery)
	if err := c.Store.EnqueueReadinessWork(context.Background(), deliveryID, hint.Delivery, factory.DependenceRef{Repository: 42, Issue: 3}); err != nil {
		t.Fatal(err)
	}
	work, err := c.Store.BeginReadinessWork(context.Background(), deliveryID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	node, err := c.Store.NextReadinessWorkNode(context.Background(), deliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Store.CheckpointReadinessAssessment(context.Background(), work, node, false); err != nil {
		t.Fatal(err)
	}
	passCtx, _, cancel := c.readinessPass(context.Background())
	budget := readinessBudgetFrom(passCtx)
	budget.pages.Store(readinessMaxPages)
	if result, drainErr := c.drainReadinessWork(passCtx, hint.Delivery); drainErr != nil || result.targetDone {
		t.Fatal("page-budget yield unexpectedly completed source:", result, drainErr)
	}
	cancel()
	seen, err := c.Store.IntakeDeliverySeen(context.Background(), hint.Delivery)
	if err != nil || seen {
		t.Fatal("budget yield acknowledged intake:", seen, err)
	}
	continued, err := c.Store.NextReadinessWorkNode(context.Background(), deliveryID)
	if err != nil || continued.State != "scanning" || continued.Cursor != (factory.DependenceRef{}) {
		t.Fatal("budget yield discarded checkpoint:", continued, err)
	}
	if _, _, err = c.ObserveIssueEvent(context.Background(), hint); err != nil {
		t.Fatal("redelivery did not resume saved work:", err)
	}
	seen, err = c.Store.IntakeDeliverySeen(context.Background(), hint.Delivery)
	if err != nil || !seen {
		t.Fatal("completed redelivery did not acknowledge:", seen, err)
	}
}

func TestReadinessDrainDefersFailureAndAdvancesAnotherRoot(t *testing.T) {
	source := &fakeEvidenceSource{
		evidence: map[string]AcceptanceEvidence{
			"42/4": readinessEvidence(12, readinessView("4"), nil, nil),
		},
		errs: map[string]error{"42/3": errors.New("temporary native read failure")},
	}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	ctx := context.Background()
	for _, item := range []struct {
		id    string
		issue int64
	}{{"best-effort:failed", 3}, {"best-effort:ready", 4}} {
		if err := c.Store.EnqueueReadinessWork(ctx, item.id, "", factory.DependenceRef{Repository: 42, Issue: item.issue}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.drainReadinessWork(ctx, ""); err != nil {
		t.Fatal("best-effort drain:", err)
	}
	if _, err := c.Store.IssueControl(ctx, 42, 4); err != nil {
		t.Fatal("failure on one root blocked a different due root:", err)
	}
	if _, err := c.Store.IssueControl(ctx, 42, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("failed root unexpectedly recorded:", err)
	}
	if _, err := c.Store.ReadinessWork(ctx, "best-effort:failed"); err != nil {
		t.Fatal("failed best-effort root was lost:", err)
	}
}

func TestReadinessEvidenceBudgetCountsActualBrackets(t *testing.T) {
	c := &Coordinator{}
	ctx, _, cancel := c.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if !budget.reserveEvidence(51) || !budget.reserveEvidence(1) || budget.reserveEvidence(1) {
		t.Fatal("conservative evidence reservations crossed the pass cap")
	}
	for i := 0; i < readinessMaxEvidenceRead; i++ {
		if err := c.recordReadinessEvidence(ctx); err != nil {
			t.Fatalf("actual bracket %d rejected: %v", i+1, err)
		}
	}
	if err := c.recordReadinessEvidence(ctx); !errors.Is(err, ErrReadinessPassPending) {
		t.Fatal("actual evidence counter exceeded the profile:", err)
	}
	if budget.nativeRPC.Load() != readinessMaxNativeRPC || budget.reserveNativeRPC(1) {
		t.Fatal("evidence brackets and freshness polls can exceed the 156 native-call profile")
	}
}

func TestReadinessPassPreservesShortCallerDeadline(t *testing.T) {
	callerDeadline := time.Now().Add(500 * time.Millisecond)
	callerCtx, callerCancel := context.WithDeadline(context.Background(), callerDeadline)
	defer callerCancel()
	ctx, _, cancel := (&Coordinator{}).readinessPass(callerCtx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		t.Fatalf("pass expired before its caller deadline: %v", err)
	}
	workDeadline, ok := ctx.Deadline()
	if !ok || !workDeadline.Before(callerDeadline) || !workDeadline.After(time.Now()) {
		t.Fatalf("short caller deadline was not preserved with bounded cleanup slack: %v, caller %v", workDeadline, callerDeadline)
	}
	if budget := readinessBudgetFrom(ctx); budget == nil || !budget.outerDeadline.Equal(callerDeadline) {
		t.Fatalf("pass lost original outer deadline: %+v", budget)
	}
}

func TestReadinessCleanupStaysInsideOriginalDeadline(t *testing.T) {
	outerDeadline := time.Now().Add(time.Second)
	workCtx, cancelWork := context.WithDeadline(context.Background(), outerDeadline.Add(-100*time.Millisecond))
	cancelWork()
	cleanupCtx, cancelCleanup := readinessCleanupContext(workCtx, outerDeadline)
	defer cancelCleanup()
	deadline, ok := cleanupCtx.Deadline()
	if !ok || deadline.After(outerDeadline) || deadline.After(time.Now().Add(readinessCleanupReserve)) {
		t.Fatalf("cleanup extended beyond the pass horizon: deadline=%v outer=%v", deadline, outerDeadline)
	}
	if err := cleanupCtx.Err(); err != nil {
		t.Fatalf("bounded cleanup inherited canceled work context: %v", err)
	}
}

func TestReadinessDrainExcludesConcurrentLocalPass(t *testing.T) {
	c := &Coordinator{}
	c.cascadeMu.Lock()
	defer c.cascadeMu.Unlock()
	if _, err := c.drainReadinessWork(context.Background(), ""); !errors.Is(err, ErrReadinessPassPending) {
		t.Fatalf("concurrent local pass was not bounded out: %v", err)
	}
}

func TestReserveIssueAssessmentPropagatesStoreReadFailure(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	if err := c.Store.Close(); err != nil {
		t.Fatal("close fixture store:", err)
	}
	ctx, _, cancel := c.readinessPass(context.Background())
	defer cancel()
	reserved, err := c.reserveIssueAssessment(ctx, factory.DependenceRef{Repository: 42, Issue: 3})
	if reserved || err == nil || errors.Is(err, ErrReadinessPassPending) {
		t.Fatalf("store read failure was hidden as budget yield: reserved=%t err=%v", reserved, err)
	}
}
