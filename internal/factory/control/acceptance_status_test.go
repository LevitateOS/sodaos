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
