package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
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
}

func (f *fakeObserver) ObserveNativeRevision(context.Context) (int64, bool, error) {
	if f.revErr != nil {
		return 0, false, f.revErr
	}
	return f.revision, f.idle, nil
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
