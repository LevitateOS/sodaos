package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestClosureAloneCannotSatisfyCodeOutcome(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	setupCodePrereq(t, c, source)
	open, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil || !changed || open.Readiness != factory.ReadinessBlocked {
		t.Fatal("open code prerequisite not blocked:", open, changed, err)
	}
	closed := readinessView("9")
	closed.Closed, closed.ClosedUnix = true, 400
	closed.ClosingOccurrence, closed.LifecycleOccurrence = "901", "901"
	source.evidence["42/9"] = readinessEvidence(13, closed, nil, nil)
	// Closure alone must neither satisfy the code outcome nor retrigger
	// assessment: the blocker inputs are unchanged.
	again, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-2", 42, 3))
	if err != nil || changed {
		t.Fatal("endpoint closure retriggered code assessment:", again, changed, err)
	}
	if again.Readiness != factory.ReadinessBlocked ||
		findBlocker(again.Blockers, factory.BlockerCodePending) == nil {
		t.Fatal("closed endpoint satisfied a code outcome:", again)
	}
}

func setupResultPrereq(t *testing.T, c *Coordinator, source *fakeEvidenceSource) {
	t.Helper()
	endpoint := readinessDecision("d"+strings.Repeat("2", 24), 42, "9", 12)
	mustAdmit(t, c, endpoint)
	dependent := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	dependent.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 9,
		Outcome: factory.PrereqResult,
	}}
	dependent.Resolutions = []factory.SelectedSource{{ID: "31", Digest: readinessCommentDigest}}
	mustAdmit(t, c, dependent)
	edge := []AcceptanceEdge{{Occurrence: "21", DependsOn: "8", Visible: true}}
	comment := []AcceptanceComment{{ID: "31", Digest: readinessCommentDigest, Visible: true}}
	source.evidence["42/3"] = readinessEvidence(12, readinessView("3"), comment, edge)
	source.evidence["42/9"] = readinessEvidence(12, readinessView("9"), nil, nil)
}

func TestResultPrereqClosureOccurrence(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	setupResultPrereq(t, c, source)
	awaiting, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil || !changed || awaiting.Readiness != factory.ReadinessBlocked {
		t.Fatal("open result prerequisite not blocked:", awaiting, changed, err)
	}
	if blocker := findBlocker(awaiting.Blockers, factory.BlockerResultPending); blocker == nil ||
		blocker.Detail != factory.ResultDetailAwaitingClosure {
		t.Fatal("awaiting detail wrong:", awaiting.Blockers)
	}
	closed := readinessView("9")
	closed.Closed, closed.ClosedUnix = true, 400
	closed.ClosingOccurrence, closed.LifecycleOccurrence = "901", "901"
	source.evidence["42/9"] = readinessEvidence(13, closed, nil, nil)
	satisfied, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-2", 42, 3))
	if err != nil || !changed || satisfied.Readiness != factory.ReadinessQueued {
		t.Fatal("closed result prerequisite not queued:", satisfied, changed, err)
	}
	reopened := readinessView("9")
	reopened.ClosingOccurrence, reopened.LifecycleOccurrence = "901", "902"
	source.evidence["42/9"] = readinessEvidence(14, reopened, nil, nil)
	withdrawn, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-3", 42, 3))
	if err != nil || !changed || withdrawn.Readiness != factory.ReadinessBlocked {
		t.Fatal("reopened prerequisite not blocked:", withdrawn, changed, err)
	}
	reclosed := readinessView("9")
	reclosed.Closed, reclosed.ClosedUnix = true, 400
	reclosed.ClosingOccurrence, reclosed.LifecycleOccurrence = "903", "903"
	source.evidence["42/9"] = readinessEvidence(15, reclosed, nil, nil)
	// Closing again cannot revive the ended occurrence without a new
	// dependent acceptance.
	revived, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-4", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if revived.Readiness != factory.ReadinessBlocked {
		t.Fatal("reclosed prerequisite revived:", revived)
	}
	if blocker := findBlocker(revived.Blockers, factory.BlockerResultPending); blocker == nil ||
		blocker.Detail != factory.ResultDetailReopened {
		t.Fatal("reopened detail wrong:", revived.Blockers)
	}
}

func TestResultPrereqNeedsAcceptedResolution(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	endpoint := readinessDecision("d"+strings.Repeat("2", 24), 42, "9", 12)
	mustAdmit(t, c, endpoint)
	dependent := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	dependent.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 9,
		Outcome: factory.PrereqResult,
	}}
	mustAdmit(t, c, dependent)
	edge := []AcceptanceEdge{{Occurrence: "21", DependsOn: "8", Visible: true}}
	source.evidence["42/3"] = readinessEvidence(12, readinessView("3"), nil, edge)
	closed := readinessView("9")
	closed.Closed, closed.ClosedUnix = true, 400
	closed.ClosingOccurrence, closed.LifecycleOccurrence = "901", "901"
	source.evidence["42/9"] = readinessEvidence(12, closed, nil, nil)
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessBlocked {
		t.Fatal("resolutionless result prerequisite queued:", assessed)
	}
	if blocker := findBlocker(assessed.Blockers, factory.BlockerResultPending); blocker == nil ||
		blocker.Detail != factory.ResultDetailNoResolution {
		t.Fatal("resolution detail wrong:", assessed.Blockers)
	}
}

func TestResultPrereqUsesNativeClosingIdentityWithoutClockOrCount(t *testing.T) {
	a := &issueAssessor{head: "d" + strings.Repeat("1", 24)}
	prereq := factory.AcceptedPrerequisite{Occurrence: "21", Outcome: factory.PrereqResult}
	assessed := prereqAssessment{satisfied: map[string]factory.PrereqSatisfaction{}}
	evidence := AcceptanceEvidence{Issue: AcceptanceIssueView{
		Closed: true, ClosedUnix: 400, ClosingOccurrence: "901",
	}}
	a.assessResultPrereq(prereq, "", evidence, true, &assessed, &factory.FingerprintInput{})
	if len(assessed.blockers) != 0 {
		t.Fatalf("first native closure did not satisfy: %+v", assessed)
	}
	// Reclose has another immutable native comment identity. Neither a
	// coincident timestamp nor a same-size retained list can revive it.
	evidence.Issue.ClosingOccurrence = "903"
	a.assessResultPrereq(prereq, "", evidence, true, &assessed, &factory.FingerprintInput{})
	if len(assessed.blockers) != 1 || assessed.blockers[0].Detail != factory.ResultDetailReopened {
		t.Fatalf("different native close revived old satisfaction: %+v", assessed)
	}
}

func TestResultPrereqMissingClosingIdentityBlocks(t *testing.T) {
	a := &issueAssessor{head: "d" + strings.Repeat("1", 24)}
	prereq := factory.AcceptedPrerequisite{Occurrence: "21", Outcome: factory.PrereqResult}
	assessed := prereqAssessment{satisfied: map[string]factory.PrereqSatisfaction{}}
	evidence := AcceptanceEvidence{Issue: AcceptanceIssueView{Closed: true, ClosedUnix: 400}}
	a.assessResultPrereq(prereq, "", evidence, true, &assessed, &factory.FingerprintInput{})
	if len(assessed.blockers) != 1 || assessed.blockers[0].Code != factory.BlockerEvidenceIncomplete || len(assessed.satisfied) != 0 {
		t.Fatalf("closed state without native occurrence satisfied: %+v", assessed)
	}
}
