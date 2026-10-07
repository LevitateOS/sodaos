package control

import (
	"context"
	"errors"
	"strings"
	"testing"

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
