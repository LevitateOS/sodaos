package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestDispatchInputsBindNativeTitleOccurrence(t *testing.T) {
	decision := acceptanceDecision()
	decision.Sources, decision.Resolutions, decision.Prerequisites = nil, nil, nil
	for name, tc := range map[string]struct {
		occurrence, reason string
	}{
		"unchanged":       {factory.OriginalOccurrence, ""},
		"title_reverted":  {"902", WaitInputsChanged},
		"unknown_history": {"", WaitInputsIncomplete},
	} {
		t.Run(name, func(t *testing.T) {
			reads := &fakeDispatchReads{inputs: map[string]DispatchInputs{"42/3": {
				Revision: 41, TipRef: "refs/heads/main", Tip: strings.Repeat("e", 40),
				Issue: DispatchIssue{
					Visible: true, TitleDigest: decision.TitleDigest,
					TitleOccurrence: tc.occurrence, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion,
				},
			}}}
			plan := &attemptPlan{acceptance: decision, policy: factory.RepositoryPolicy{TargetBranch: "refs/heads/main"}}
			wait, failed := readAttemptInputs(context.Background(), DispatchDeps{Reads: reads}, plan, 42, 3)
			if failed != nil {
				t.Fatalf("unexpected failure: %+v", failed)
			}
			if tc.reason == "" {
				if wait != nil || plan.inputs.Issue.TitleOccurrence != factory.OriginalOccurrence {
					t.Fatalf("unchanged native input refused: %+v", wait)
				}
			} else if wait == nil || wait.Reason != tc.reason {
				t.Fatalf("stale title authorized: %+v", wait)
			}
		})
	}
}
