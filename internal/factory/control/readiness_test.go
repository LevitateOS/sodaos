package control

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

const (
	readinessTitleDigest   = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	readinessContentDigest = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	readinessCommentDigest = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

type fakeEvidenceSource struct {
	evidence map[string]AcceptanceEvidence
	errs     map[string]error
	calls    map[string]int
}

func (f *fakeEvidenceSource) ReadAcceptanceEvidence(_ context.Context, repository, issue string, _ []string) (AcceptanceEvidence, error) {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	key := repository + "/" + issue
	f.calls[key]++
	if err, ok := f.errs[key]; ok {
		return AcceptanceEvidence{}, err
	}
	evidence, ok := f.evidence[key]
	if !ok {
		return AcceptanceEvidence{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	return evidence, nil
}

func (f *fakeEvidenceSource) totalCalls() int {
	total := 0
	for _, n := range f.calls {
		total += n
	}
	return total
}

type fakeObserver struct {
	revision int64
	idle     bool
	revErr   error
	pages    map[int]fakeIssuePage
	byRepo   map[int64][]int64
}

type fakeIssuePage struct {
	indexes []int64
	hasMore bool
}

func (f *fakeObserver) ObserveNativeRevision(context.Context) (int64, bool, error) {
	if f.revErr != nil {
		return 0, false, f.revErr
	}
	return f.revision, f.idle, nil
}

func (f *fakeObserver) ListRepositoryIssues(_ context.Context, repository int64, page int) ([]int64, bool, error) {
	if f.byRepo != nil {
		if page != 1 {
			return nil, false, nil
		}
		return f.byRepo[repository], false, nil
	}
	listed, ok := f.pages[page]
	if !ok {
		return nil, false, nil
	}
	return listed.indexes, listed.hasMore, nil
}

func readinessView(issue string) AcceptanceIssueView {
	return AcceptanceIssueView{
		Index: issue, TitleDigest: readinessTitleDigest, ContentDigest: readinessContentDigest,
		PosterID: "5", Visible: true,
	}
}

func readinessEvidence(revision int64, view AcceptanceIssueView, comments []AcceptanceComment, edges []AcceptanceEdge) AcceptanceEvidence {
	return AcceptanceEvidence{Issue: view, Comments: comments, Dependencies: edges, Revision: revision}
}

func readinessDecision(id string, repository int64, issue string, revision int64) factory.Acceptance {
	return factory.Acceptance{
		ID: id, Repository: repository, IssueIndex: issue,
		Approver: 5, NativeRev: revision,
		TitleDigest: readinessTitleDigest, ContentDigest: readinessContentDigest,
	}
}

func readinessCoordinator(t *testing.T, source *fakeEvidenceSource, observer *fakeObserver) *Coordinator {
	t.Helper()
	c := coordinatorFixture(t, nil, nil)
	c.AcceptanceReads = source
	c.Readiness = observer
	return c
}

func readinessHint(delivery string, repository, issue int64) IntakeHint {
	return IntakeHint{Delivery: delivery, Repository: repository, Issue: issue}
}

func mustAdmit(t *testing.T, c *Coordinator, decision factory.Acceptance) {
	t.Helper()
	if err := c.Store.AdmitAcceptanceDecision(context.Background(), decision); err != nil {
		t.Fatal(err)
	}
}

func findBlocker(blockers []factory.Blocker, code string) *factory.Blocker {
	for i := range blockers {
		if blockers[i].Code == code {
			return &blockers[i]
		}
	}
	return nil
}

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

func setupCodePrereq(t *testing.T, c *Coordinator, source *fakeEvidenceSource) {
	t.Helper()
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
	endpointView := readinessView("9")
	source.evidence["42/9"] = readinessEvidence(12, endpointView, nil, nil)
}

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
	closed.Closed, closed.ClosedUnix, closed.Lifecycle = true, 400, 5
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
	closed.Closed, closed.ClosedUnix, closed.Lifecycle = true, 400, 5
	source.evidence["42/9"] = readinessEvidence(13, closed, nil, nil)
	satisfied, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-2", 42, 3))
	if err != nil || !changed || satisfied.Readiness != factory.ReadinessQueued {
		t.Fatal("closed result prerequisite not queued:", satisfied, changed, err)
	}
	reopened := readinessView("9")
	reopened.Lifecycle = 6
	source.evidence["42/9"] = readinessEvidence(14, reopened, nil, nil)
	withdrawn, changed, err := c.ObserveIssueEvent(context.Background(), readinessHint("delivery-3", 42, 3))
	if err != nil || !changed || withdrawn.Readiness != factory.ReadinessBlocked {
		t.Fatal("reopened prerequisite not blocked:", withdrawn, changed, err)
	}
	reclosed := readinessView("9")
	reclosed.Closed, reclosed.ClosedUnix, reclosed.Lifecycle = true, 500, 7
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
	closed.Closed, closed.ClosedUnix, closed.Lifecycle = true, 400, 5
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
		view.Closed, view.ClosedUnix, view.Lifecycle = true, 400, 5
		edge := []AcceptanceEdge{{Occurrence: "2" + issue, DependsOn: "8", Visible: true}}
		comment := []AcceptanceComment{{ID: "3" + issue, Digest: readinessCommentDigest, Visible: true}}
		source.evidence["42/"+issue] = readinessEvidence(12, view, comment, edge)
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
