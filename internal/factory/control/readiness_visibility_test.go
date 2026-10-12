package control

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func TestHiddenEndpointBlocksWithoutLeaking(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	setupCodePrereq(t, c, source)
	hidden := readinessView("9")
	hidden.Visible = false
	hidden.TitleDigest, hidden.ContentDigest, hidden.PosterID = "", "", ""
	source.evidence["42/9"] = readinessEvidence(12, hidden, nil, nil)
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessBlocked {
		t.Fatal("hidden endpoint not blocked:", assessed)
	}
	blocker := findBlocker(assessed.Blockers, factory.BlockerEndpointHidden)
	if blocker == nil || blocker.EndpointRepo != 42 || blocker.EndpointIssue != 9 {
		t.Fatal("hidden endpoint blocker wrong:", assessed.Blockers)
	}
}

func TestHiddenIssueDeniesAuthorization(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	hidden := readinessView("3")
	hidden.Visible = false
	hidden.TitleDigest, hidden.ContentDigest, hidden.PosterID = "", "", ""
	source.evidence["42/3"] = readinessEvidence(12, hidden, nil, nil)
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessNotAuthorized ||
		findBlocker(assessed.Blockers, factory.BlockerIssueInaccessible) == nil {
		t.Fatal("hidden issue not inaccessible:", assessed)
	}
}

func TestIncompleteEvidenceBlocks(t *testing.T) {
	source := &fakeEvidenceSource{
		evidence: map[string]AcceptanceEvidence{},
		errs:     map[string]error{"42/3": refuseAcceptance(RefusalIncompleteEvidence)},
	}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	decision := readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12)
	mustAdmit(t, c, decision)
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessBlocked ||
		findBlocker(assessed.Blockers, factory.BlockerEvidenceIncomplete) == nil {
		t.Fatal("over-bound issue not blocked:", assessed)
	}
}

func TestLongerCycleBlocks(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	issues := []string{"3", "4", "5"}
	for i, issue := range issues {
		next := issues[(i+1)%len(issues)]
		decision := readinessDecision("d"+strings.Repeat(strconv.Itoa(i+1), 24)[:24], 42, issue, 12)
		endpoint, _ := strconv.ParseInt(next, 10, 64)
		decision.Prerequisites = []factory.AcceptedPrerequisite{{
			Occurrence: "2" + issue, DependsOn: "8", EndpointRepo: 42, EndpointIssue: endpoint,
			Outcome: factory.PrereqResult,
		}}
		decision.Resolutions = []factory.SelectedSource{{ID: "3" + issue, Digest: readinessCommentDigest}}
		mustAdmit(t, c, decision)
		view := readinessView(issue)
		view.Closed, view.ClosedUnix = true, 400
		view.ClosingOccurrence, view.LifecycleOccurrence = "94", "94"
		edge := []AcceptanceEdge{{Occurrence: "2" + issue, DependsOn: "8", Visible: true}}
		comment := []AcceptanceComment{{ID: "3" + issue, Digest: readinessCommentDigest, Visible: true}}
		source.evidence["42/"+issue] = readinessEvidence(12, view, comment, edge)
	}
	// Setup admissions queued one root source each. Drain them across
	// passes first, as serving reconciliation would: the bounded page
	// budget cannot settle setup residue plus the exercise cascade in
	// one pass, and the cycle verdict below must not depend on it.
	drained := false
	for pass := 0; pass < 10 && !drained; pass++ {
		if _, err := c.drainReadinessWork(context.Background(), ""); err != nil && !errors.Is(err, ErrReadinessPassPending) {
			t.Fatal("setup drain:", err)
		}
		if _, err := c.Store.NextReadinessWork(context.Background(), time.Now()); errors.Is(err, store.ErrNotFound) {
			drained = true
		} else if err != nil {
			t.Fatal("setup work poll:", err)
		}
	}
	if !drained {
		t.Fatal("setup roots never drained")
	}
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessBlocked {
		t.Fatal("cycle member not blocked:", assessed)
	}
	blocker := findBlocker(assessed.Blockers, factory.BlockerCycle)
	if blocker == nil || blocker.Detail != "42/3 -> 42/4 -> 42/5 -> 42/3" {
		t.Fatal("cycle blocker wrong:", assessed.Blockers)
	}
}

func TestWithdrawnEndpointInvalidatesDependent(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	setupCodePrereq(t, c, source)
	ctx := context.Background()
	endpointID := "d" + strings.Repeat("2", 24)
	if err := c.Store.WithdrawAcceptanceDecision(ctx, 42, 9, endpointID, 5); err != nil {
		t.Fatal(err)
	}
	assessed, _, err := c.ObserveIssueEvent(ctx, readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessNotAuthorized {
		t.Fatal("withdrawn endpoint route authorized:", assessed)
	}
	blocker := findBlocker(assessed.Blockers, factory.BlockerAcceptanceInvalid)
	if blocker == nil || blocker.Detail != factory.ResultDetailAcceptanceWithdrawn ||
		blocker.EndpointIssue != 9 {
		t.Fatal("withdrawn route blocker wrong:", assessed.Blockers)
	}
}

func TestEditedEndpointInvalidatesDependent(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	setupCodePrereq(t, c, source)
	edited := readinessView("9")
	edited.ContentDigest = strings.Repeat("f", 64)
	source.evidence["42/9"] = readinessEvidence(13, edited, nil, nil)
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessNotAuthorized {
		t.Fatal("edited endpoint route authorized:", assessed)
	}
	blocker := findBlocker(assessed.Blockers, factory.BlockerAcceptanceInvalid)
	if blocker == nil || blocker.Detail != factory.DetailPrereqAcceptanceInvalid {
		t.Fatal("edited route blocker wrong:", assessed.Blockers)
	}
}

func TestMissingAuthorityStaysUnauthorized(t *testing.T) {
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	mustAdmit(t, c, readinessDecision("d"+strings.Repeat("1", 24), 42, "3", 12))
	assessed, _, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-1", 42, 3))
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Readiness != factory.ReadinessNotAuthorized {
		t.Fatal("grantless issue authorized:", assessed)
	}
	var missing []string
	for _, blocker := range assessed.Blockers {
		if blocker.Code == factory.BlockerAuthorityMissing {
			missing = append(missing, blocker.Detail)
		}
	}
	if len(missing) != 5 {
		t.Fatal("authority gaps underreported:", assessed.Blockers)
	}
}
