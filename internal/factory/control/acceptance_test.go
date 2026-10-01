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

func creationEvidence() AcceptanceEvidence {
	title, body, _, _ := acceptanceDigests()
	return AcceptanceEvidence{
		Revision: 41,
		Issue: AcceptanceIssueView{Index: "3", TitleDigest: title, ContentDigest: body, ContentVer: 0,
			Lifecycle: 0, PosterID: "5", Verified: true, FirstCreated: true, Visible: true},
	}
}

func creationHarness(t *testing.T, evidence AcceptanceEvidence) *Coordinator {
	t.Helper()
	c, _ := acceptanceHarness(t, evidence, nil)
	if _, err := c.ApplyPolicy(context.Background(), factory.NewID(), "native:7", 0, grantPolicy()); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAdmitInitialAcceptanceVerifiesCreation(t *testing.T) {
	c := creationHarness(t, creationEvidence())
	got, err := c.AdmitInitialAcceptance(context.Background(), 42, "3", "5", true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Initial || got.Approver != 5 || got.NativeRev != 41 || got.ID != factory.InitialAcceptanceID(42, "3") {
		t.Fatalf("decision: %+v", got)
	}
	head, err := c.Store.AcceptanceHead(context.Background(), 42, 3)
	if err != nil || head != got.ID {
		t.Fatalf("head: %q %v", head, err)
	}
	// A duplicate creation observation replays the one decision.
	again, err := c.AdmitInitialAcceptance(context.Background(), 42, "3", "5", true)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("replay: %+v %v", again, err)
	}
}

func TestAdmitInitialAcceptanceRefusesIneligible(t *testing.T) {
	cases := map[string]struct {
		mutate    func(*AcceptanceEvidence)
		creator   string
		authorize bool
		policy    bool
		reason    string
	}{
		"imported":  {mutate: func(e *AcceptanceEvidence) { e.Issue.Verified = false }, creator: "5", authorize: true, policy: true, reason: RefusalCreationUnverified},
		"recreated": {mutate: func(e *AcceptanceEvidence) { e.Issue.FirstCreated = false }, creator: "5", authorize: true, policy: true, reason: RefusalCreationUnverified},
		"poster":    {creator: "6", authorize: true, policy: true, reason: RefusalCreationUnverified},
		"hidden":    {mutate: func(e *AcceptanceEvidence) { e.Issue.Visible = false }, creator: "5", authorize: true, policy: true, reason: RefusalIssueHidden},
		"edited":    {mutate: func(e *AcceptanceEvidence) { e.Issue.ContentVer = 1 }, creator: "5", authorize: true, policy: true, reason: RefusalCreationEdited},
		"retitled":  {mutate: func(e *AcceptanceEvidence) { e.Issue.Lifecycle = 1 }, creator: "5", authorize: true, policy: true, reason: RefusalCreationEdited},
		"edges": {mutate: func(e *AcceptanceEvidence) {
			e.Dependencies = []AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}}
		}, creator: "5", authorize: true, policy: true, reason: RefusalCreationBlocked},
		"factory":   {creator: "12", authorize: true, policy: true, reason: RefusalCreationUnverified},
		"outsider":  {creator: "5", authorize: false, policy: true, reason: RefusalCreationUnauthorized},
		"no-policy": {creator: "5", authorize: true, policy: false, reason: RefusalCreationUnauthorized},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			evidence := creationEvidence()
			if tc.mutate != nil {
				tc.mutate(&evidence)
			}
			// The factory-poster case needs provenance to match the actor.
			if name == "factory" {
				evidence.Issue.PosterID = "12"
			}
			var c *Coordinator
			if tc.policy {
				c = creationHarness(t, evidence)
			} else {
				c, _ = acceptanceHarness(t, evidence, nil)
			}
			_, err := c.AdmitInitialAcceptance(context.Background(), 42, "3", tc.creator, tc.authorize)
			want := tc.reason
			if name == "factory" {
				want = RefusalCreationFactory
			}
			if reason := acceptanceRefusal(t, err); reason != want {
				t.Fatalf("reason %q", reason)
			}
			if _, err := c.Store.AcceptanceHead(context.Background(), 42, 3); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("refusal recorded: %v", err)
			}
		})
	}
}

func TestAdmitInitialAcceptanceCannotOverwrite(t *testing.T) {
	c := creationHarness(t, creationEvidence())
	explicit := acceptanceDecision()
	explicit.NativeRev = 41
	explicit.Sources, explicit.Prerequisites, explicit.Resolutions = nil, nil, nil
	explicit.TitleDigest, explicit.ContentDigest, explicit.ContentVersion =
		creationEvidence().Issue.TitleDigest, creationEvidence().Issue.ContentDigest, 0
	source := c.AcceptanceReads.(*stubAcceptanceSource)
	source.evidence = creationEvidence()
	if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", explicit); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdmitInitialAcceptance(context.Background(), 42, "3", "5", true); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("overwrite: %v", err)
	}
}

func TestAcceptanceStatusAssessesValidity(t *testing.T) {
	admit := func(t *testing.T, c *Coordinator) {
		t.Helper()
		if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", acceptanceDecision()); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("none", func(t *testing.T) {
		c, _ := acceptanceHarness(t, acceptanceEvidence(), nil)
		status, err := c.AcceptanceStatus(context.Background(), 42, "3")
		if err != nil || status.Valid || status.Acceptance != nil || len(status.Reasons) != 1 || status.Reasons[0] != "no_acceptance" {
			t.Fatalf("status: %+v %v", status, err)
		}
	})
	t.Run("valid", func(t *testing.T) {
		c, _ := acceptanceHarness(t, acceptanceEvidence(), nil)
		admit(t, c)
		status, err := c.AcceptanceStatus(context.Background(), 42, "3")
		if err != nil || !status.Valid || status.Revision != 41 || status.Acceptance == nil {
			t.Fatalf("status: %+v %v", status, err)
		}
	})
	t.Run("revision-alone", func(t *testing.T) {
		c, source := acceptanceHarness(t, acceptanceEvidence(), nil)
		admit(t, c)
		evidence := acceptanceEvidence()
		evidence.Revision = 87
		source.evidence = evidence
		status, err := c.AcceptanceStatus(context.Background(), 42, "3")
		if err != nil || !status.Valid || status.Revision != 87 {
			t.Fatalf("status: %+v %v", status, err)
		}
	})
	t.Run("revert", func(t *testing.T) {
		c, source := acceptanceHarness(t, acceptanceEvidence(), nil)
		admit(t, c)
		// Restored text at a new version does not reactivate.
		evidence := acceptanceEvidence()
		evidence.Issue.ContentVer = 4
		source.evidence = evidence
		status, err := c.AcceptanceStatus(context.Background(), 42, "3")
		if err != nil || status.Valid {
			t.Fatalf("status: %+v %v", status, err)
		}
	})
	changes := map[string]struct {
		mutate  func(*AcceptanceEvidence)
		reasons []string
	}{
		"objective": {mutate: func(e *AcceptanceEvidence) { e.Issue.ContentDigest = strings.Repeat("f", 64) },
			reasons: []string{RefusalObjectiveChanged}},
		"source": {mutate: func(e *AcceptanceEvidence) { e.Comments[1].Digest = strings.Repeat("e", 64) },
			reasons: []string{RefusalSourceChanged}},
		"source-gone": {mutate: func(e *AcceptanceEvidence) { e.Comments = e.Comments[:1] },
			reasons: []string{RefusalSourceMissing}},
		"edge-replaced": {mutate: func(e *AcceptanceEvidence) { e.Dependencies[0].Occurrence = "29" },
			reasons: []string{RefusalEdgeChanged}},
		"edge-removed": {mutate: func(e *AcceptanceEvidence) { e.Dependencies = nil },
			reasons: []string{RefusalEdgeChanged}},
		"hidden": {mutate: func(e *AcceptanceEvidence) { e.Issue.Visible = false }, reasons: []string{RefusalIssueHidden}},
	}
	for name, tc := range changes {
		t.Run(name, func(t *testing.T) {
			c, source := acceptanceHarness(t, acceptanceEvidence(), nil)
			admit(t, c)
			evidence := acceptanceEvidence()
			tc.mutate(&evidence)
			source.evidence = evidence
			status, err := c.AcceptanceStatus(context.Background(), 42, "3")
			if err != nil || status.Valid {
				t.Fatalf("status: %+v %v", status, err)
			}
			for _, want := range tc.reasons {
				found := false
				for _, reason := range status.Reasons {
					found = found || reason == want
				}
				if !found {
					t.Fatalf("reasons %v", status.Reasons)
				}
			}
		})
	}
	t.Run("prereq", func(t *testing.T) {
		c, _ := acceptanceHarness(t, acceptanceEvidence(), nil)
		prereq := factory.Acceptance{ID: "d111111111111111111111111", Repository: 42, IssueIndex: "2", Approver: 7, NativeRev: 40,
			TitleDigest: strings.Repeat("e", 64), ContentDigest: strings.Repeat("f", 64)}
		if err := c.Store.AdmitAcceptanceDecision(context.Background(), prereq); err != nil {
			t.Fatal(err)
		}
		decision := acceptanceDecision()
		decision.Prerequisites[0].Outcome = factory.PrereqCode
		decision.Prerequisites[0].PrereqAcceptance = prereq.ID
		if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", decision); err != nil {
			t.Fatal(err)
		}
		status, err := c.AcceptanceStatus(context.Background(), 42, "3")
		if err != nil || !status.Valid {
			t.Fatalf("status: %+v %v", status, err)
		}
		newer := prereq
		newer.ID, newer.Predecessor, newer.NativeRev = "d222222222222222222222222", prereq.ID, 41
		if err := c.Store.AdmitAcceptanceDecision(context.Background(), newer); err != nil {
			t.Fatal(err)
		}
		status, err = c.AcceptanceStatus(context.Background(), 42, "3")
		if err != nil || status.Valid {
			t.Fatalf("stale route valid: %+v %v", status, err)
		}
	})
}

func TestWithdrawAcceptanceLatchesAndReplays(t *testing.T) {
	c, _ := acceptanceHarness(t, acceptanceEvidence(), nil)
	decision := acceptanceDecision()
	if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", decision); err != nil {
		t.Fatal(err)
	}
	command := factory.NewID()
	receipt, err := c.WithdrawAcceptance(context.Background(), command, "native:7", 42, 3, decision.ID, 7)
	if err != nil || !receipt.Withdrawn {
		t.Fatal(receipt, err)
	}
	again, err := c.WithdrawAcceptance(context.Background(), command, "native:7", 42, 3, decision.ID, 7)
	if err != nil || !reflect.DeepEqual(again, receipt) {
		t.Fatalf("replay: %+v %v", again, err)
	}
	status, err := c.AcceptanceStatus(context.Background(), 42, "3")
	if err != nil || status.Valid || !status.Withdrawn {
		t.Fatalf("status: %+v %v", status, err)
	}
	second := acceptanceDecision()
	second.ID, second.Predecessor = "d123456789abcdef012345678", decision.ID
	if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", second); err != nil {
		t.Fatal(err)
	}
	if _, err := c.WithdrawAcceptance(context.Background(), factory.NewID(), "native:7", 42, 3, decision.ID, 7); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("superseded: %v", err)
	}
	status, err = c.AcceptanceStatus(context.Background(), 42, "3")
	if err != nil || !status.Valid || status.Withdrawn {
		t.Fatalf("fresh head: %+v %v", status, err)
	}
}
