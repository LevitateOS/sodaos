package control

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func creationEvidence() AcceptanceEvidence {
	title, body, _, _ := acceptanceDigests()
	return AcceptanceEvidence{
		Revision: 41,
		Issue: AcceptanceIssueView{
			ID: "103", CreatedUnix: 1700000000, Index: "3", TitleDigest: title, ContentDigest: body, ContentVer: 0,
			LifecycleOccurrence: factory.OriginalOccurrence, TitleOccurrence: factory.OriginalOccurrence, PosterID: "5", Verified: true, FirstCreated: true, Visible: true,
		},
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
	got, err := c.AdmitInitialAcceptance(context.Background(), creationHint("delivery-1"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Initial || got.Approver != 5 || got.NativeRev != 41 || got.ID != factory.InitialAcceptanceID(42, "103") {
		t.Fatalf("decision: %+v", got)
	}
	head, err := c.Store.AcceptanceHead(context.Background(), 42, 3)
	if err != nil || head != got.ID {
		t.Fatalf("head: %q %v", head, err)
	}
	// A replay after unrelated native activity returns the original decision.
	c.AcceptanceReads.(*stubAcceptanceSource).evidence.Revision++
	again, err := c.AdmitInitialAcceptance(context.Background(), creationHint("delivery-1"))
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
		"retitled":  {mutate: func(e *AcceptanceEvidence) { e.Issue.LifecycleOccurrence = "91" }, creator: "5", authorize: true, policy: true, reason: RefusalCreationEdited},
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
			hint := creationHint("delivery-1")
			if tc.creator == "6" {
				hint.Creator = 6
			}
			if tc.creator == "12" {
				hint.Creator = 12
			}
			hint.CreatorWrite = tc.authorize
			if name == "imported" || name == "recreated" {
				hint.CreationVerified = false
			}
			_, err := c.AdmitInitialAcceptance(context.Background(), hint)
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
	explicit.TitleDigest, explicit.ContentDigest, explicit.ContentVersion = creationEvidence().Issue.TitleDigest, creationEvidence().Issue.ContentDigest, 0
	source := c.AcceptanceReads.(*stubAcceptanceSource)
	source.evidence = creationEvidence()
	if _, err := c.AdmitAcceptance(context.Background(), factory.NewID(), "native:7", explicit); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AdmitInitialAcceptance(context.Background(), creationHint("delivery-1")); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("overwrite: %v", err)
	}
}

func TestAdmitInitialAcceptanceNormalNativeCreationHasNoFirstHistory(t *testing.T) {
	evidence := creationEvidence()
	evidence.Issue.ID, evidence.Issue.CreatedUnix = "103", 1700000000
	evidence.Issue.Verified, evidence.Issue.FirstCreated = false, false
	c := creationHarness(t, evidence)
	decision, err := c.AdmitInitialAcceptance(context.Background(), creationHint("delivery-1"))
	if err != nil || decision.ID != factory.InitialAcceptanceID(42, "103") {
		t.Fatalf("original native creation must admit without a fabricated history row: %+v %v", decision, err)
	}
}

func creationHint(delivery string) IntakeHint {
	e := creationEvidence().Issue
	return IntakeHint{
		Delivery: delivery, Repository: 42, Issue: 3, Created: true, Creator: 5, CreatorWrite: true,
		CreationVerified: true, NativeIssueID: 103, CreatedUnix: e.CreatedUnix, TitleDigest: e.TitleDigest, ContentDigest: e.ContentDigest,
	}
}
