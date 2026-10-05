package control

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

type stubAcceptanceSource struct {
	evidence AcceptanceEvidence
	err      error
	requests [][]string
}

func (s *stubAcceptanceSource) ReadAcceptanceEvidence(_ context.Context, repository, issue string, commentIDs []string) (AcceptanceEvidence, error) {
	s.requests = append(s.requests, append([]string{repository, issue}, commentIDs...))
	return s.evidence, s.err
}

func acceptanceDigests() (title, body, source, resolution string) {
	return strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
}

func acceptanceDecision() factory.Acceptance {
	title, body, source, resolution := acceptanceDigests()
	return factory.Acceptance{
		ID: "d0123456789abcdef01234567", Repository: 42, IssueIndex: "3", Approver: 7, NativeRev: 41,
		TitleDigest: title, ContentDigest: body, ContentVersion: 2,
		Sources:       []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: source}},
		Prerequisites: []factory.AcceptedPrerequisite{{Occurrence: "21", DependsOn: "9", EndpointRepo: 42, EndpointIssue: 2, Outcome: factory.PrereqResult}},
		Resolutions:   []factory.SelectedSource{{ID: "12", ContentVersion: 1, Digest: resolution}},
	}
}

func acceptanceEvidence() AcceptanceEvidence {
	title, body, source, resolution := acceptanceDigests()
	return AcceptanceEvidence{
		Revision: 41,
		Issue: AcceptanceIssueView{Index: "3", TitleDigest: title, ContentDigest: body, ContentVer: 2,
			PosterID: "7", Verified: true, FirstCreated: true, Visible: true},
		Comments: []AcceptanceComment{
			{ID: "11", Digest: source, ContentVer: 0, Visible: true},
			{ID: "12", Digest: resolution, ContentVer: 1, Visible: true},
		},
		Dependencies: []AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
	}
}

func acceptanceHarness(t *testing.T, evidence AcceptanceEvidence, err error) (*Coordinator, *stubAcceptanceSource) {
	t.Helper()
	c := coordinatorFixture(t, nil, nil)
	source := &stubAcceptanceSource{evidence: evidence, err: err}
	c.AcceptanceReads = source
	return c, source
}

func acceptanceRefusal(t *testing.T, err error) string {
	t.Helper()
	var refusal *AcceptanceRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("expected refusal, got %v", err)
	}
	return refusal.Reason
}

func TestAdmitAcceptanceVerifiesAndRecords(t *testing.T) {
	c, source := acceptanceHarness(t, acceptanceEvidence(), nil)
	decision := acceptanceDecision()
	receipt, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", decision)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.DecisionID != decision.ID || receipt.Head != decision.ID || receipt.Depth != 1 {
		t.Fatalf("receipt: %+v", receipt)
	}
	if len(source.requests) != 1 {
		t.Fatalf("requests: %d", len(source.requests))
	}
	// Only the objective, selected IDs and the edge set are read:
	// unselected discussion never enters the bracket.
	got := source.requests[0]
	if len(got) != 4 || got[0] != "42" || got[1] != "3" || got[2] != "11" || got[3] != "12" {
		t.Fatalf("request: %v", got)
	}
	stored, err := c.Store.AcceptanceDecision(context.Background(), decision.ID)
	if err != nil || stored.Approver != 7 || stored.NativeRev != 41 || len(stored.Prerequisites) != 1 {
		t.Fatalf("stored: %+v %v", stored, err)
	}
	second := acceptanceDecision()
	second.ID, second.Predecessor = "d123456789abcdef012345678", decision.ID
	command := factory.NewID()
	first, err := c.AdmitAcceptance(context.Background(), command, "native:7", second)
	if err != nil {
		t.Fatal(err)
	}
	again, err := c.AdmitAcceptance(context.Background(), command, "native:7", second)
	if err != nil || !reflect.DeepEqual(again, first) || again.Depth != 2 {
		t.Fatalf("replay: %+v %v", again, err)
	}
	changed := second
	changed.ID = "d999999999999999999999999"
	if _, err = c.AdmitAcceptance(context.Background(), command, "native:7", changed); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatalf("changed content: %v", err)
	}
	stale := acceptanceDecision()
	stale.ID = "d23456789abcdef0123456789"
	if _, err = c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", stale); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("stale predecessor: %v", err)
	}
}

func TestAdmitAcceptanceReplaysAfterNativeRevisionAdvance(t *testing.T) {
	c, source := acceptanceHarness(t, acceptanceEvidence(), nil)
	decision := acceptanceDecision()
	command := factory.NewID()
	first, err := c.AdmitAcceptance(context.Background(), command, "native:7", decision)
	if err != nil {
		t.Fatal(err)
	}
	// Native revision advances; the admitted decision is unchanged.
	advanced := acceptanceEvidence()
	advanced.Revision = 42
	source.evidence = advanced
	again, err := c.AdmitAcceptance(context.Background(), command, "native:7", decision)
	if err != nil || !reflect.DeepEqual(again, first) {
		t.Fatalf("replay: %+v %v", again, err)
	}
}

func TestAdmitAcceptanceReplayEnforcesCurrentVisibility(t *testing.T) {
	c, source := acceptanceHarness(t, acceptanceEvidence(), nil)
	decision := acceptanceDecision()
	command := factory.NewID()
	if _, err := c.AdmitAcceptance(context.Background(), command, "native:7", decision); err != nil {
		t.Fatal(err)
	}
	hidden := acceptanceEvidence()
	hidden.Issue.Visible = false
	source.evidence = hidden
	_, err := c.AdmitAcceptance(context.Background(), command, "native:7", decision)
	if reason := acceptanceRefusal(t, err); reason != RefusalIssueHidden {
		t.Fatalf("reason %q", reason)
	}
}

func TestAdmitAcceptanceRefusesWithoutRecording(t *testing.T) {
	cases := map[string]struct {
		mutate  func(*AcceptanceEvidence)
		err     error
		unwired bool
		reason  string
	}{
		"stale-screen": {mutate: func(e *AcceptanceEvidence) { e.Revision = 42 }, reason: RefusalStaleEvidence},
		"stale":        {err: &AcceptanceRefusal{Reason: RefusalStaleEvidence}, reason: RefusalStaleEvidence},
		"incomplete":   {err: &AcceptanceRefusal{Reason: RefusalIncompleteEvidence}, reason: RefusalIncompleteEvidence},
		"busy":         {err: &AcceptanceRefusal{Reason: RefusalNativeBusy}, reason: RefusalNativeBusy},
		"hidden-issue": {mutate: func(e *AcceptanceEvidence) {
			e.Issue.Visible = false
			e.Issue.TitleDigest, e.Issue.ContentDigest = "", ""
		}, reason: RefusalIssueHidden},
		"objective": {mutate: func(e *AcceptanceEvidence) { e.Issue.ContentVer = 3 }, reason: RefusalObjectiveChanged},
		"title": {mutate: func(e *AcceptanceEvidence) {
			e.Issue.TitleDigest = strings.Repeat("f", 64)
		}, reason: RefusalObjectiveChanged},
		"source-gone": {mutate: func(e *AcceptanceEvidence) {
			e.Comments = e.Comments[:1]
		}, reason: RefusalSourceMissing},
		"source-hidden": {mutate: func(e *AcceptanceEvidence) {
			e.Comments[0].Visible = false
			e.Comments[0].Digest = ""
		}, reason: RefusalSourceHidden},
		"source-edit": {mutate: func(e *AcceptanceEvidence) { e.Comments[0].ContentVer = 1 }, reason: RefusalSourceChanged},
		"edge-hidden": {mutate: func(e *AcceptanceEvidence) {
			e.Dependencies[0].Visible = false
			e.Dependencies[0].DependsOn = ""
		}, reason: RefusalEdgeHidden},
		"edge-added": {mutate: func(e *AcceptanceEvidence) {
			e.Dependencies = append(e.Dependencies, AcceptanceEdge{Occurrence: "22", DependsOn: "10", Visible: true})
		}, reason: RefusalEdgeChanged},
		"edge-recreated": {mutate: func(e *AcceptanceEvidence) {
			e.Dependencies[0].Occurrence = "29"
		}, reason: RefusalEdgeChanged},
		"edge-retarget": {mutate: func(e *AcceptanceEvidence) {
			e.Dependencies[0].DependsOn = "10"
		}, reason: RefusalEdgeChanged},
		"unwired": {unwired: true, reason: RefusalSnapshotUnavailable},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			evidence := acceptanceEvidence()
			if tc.mutate != nil {
				tc.mutate(&evidence)
			}
			c, _ := acceptanceHarness(t, evidence, tc.err)
			if tc.unwired {
				c.AcceptanceReads = nil
			}
			_, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", acceptanceDecision())
			if reason := acceptanceRefusal(t, err); reason != tc.reason {
				t.Fatalf("reason %q", reason)
			}
			if _, err := c.Store.AcceptanceHead(context.Background(), 42, 3); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("refusal recorded: %v", err)
			}
		})
	}
}

func TestAdmitAcceptanceGuardsCodeRoutes(t *testing.T) {
	setup := func(t *testing.T) *Coordinator {
		t.Helper()
		c, _ := acceptanceHarness(t, acceptanceEvidence(), nil)
		return c
	}
	t.Run("unknown", func(t *testing.T) {
		c := setup(t)
		decision := acceptanceDecision()
		decision.Prerequisites[0].Outcome = factory.PrereqCode
		decision.Prerequisites[0].PrereqAcceptance = "d999999999999999999999999"
		_, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", decision)
		if reason := acceptanceRefusal(t, err); reason != RefusalPrereqUnknown {
			t.Fatalf("reason %q", reason)
		}
	})
	t.Run("stale", func(t *testing.T) {
		c := setup(t)
		prereq := factory.Acceptance{ID: "d111111111111111111111111", Repository: 42, IssueIndex: "2", Approver: 7, NativeRev: 40,
			TitleDigest: strings.Repeat("e", 64), ContentDigest: strings.Repeat("f", 64)}
		if err := c.Store.AdmitAcceptanceDecision(context.Background(), prereq); err != nil {
			t.Fatal(err)
		}
		newer := prereq
		newer.ID, newer.Predecessor, newer.NativeRev = "d222222222222222222222222", prereq.ID, 41
		if err := c.Store.AdmitAcceptanceDecision(context.Background(), newer); err != nil {
			t.Fatal(err)
		}
		decision := acceptanceDecision()
		decision.Prerequisites[0].Outcome = factory.PrereqCode
		decision.Prerequisites[0].PrereqAcceptance = prereq.ID
		_, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", decision)
		if reason := acceptanceRefusal(t, err); reason != RefusalPrereqStale {
			t.Fatalf("reason %q", reason)
		}
		decision.Prerequisites[0].PrereqAcceptance = newer.ID
		if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", decision); err != nil {
			t.Fatalf("current route: %v", err)
		}
	})
}
