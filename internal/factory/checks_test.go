package factory

import (
	"strings"
	"testing"
)

func checkTargetFixture() CheckTarget {
	return CheckTarget{
		Repository: 7, PRNumber: 3, PRID: 11, IssueID: 13,
		HeadRef: "refs/heads/soda/factory/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("a", 40), BaseOID: strings.Repeat("b", 40),
	}
}

func adoptedChecksFixture(checks ...string) AdoptedChecks {
	if checks == nil {
		checks = []string{"st11-build", "st11-review-gate"}
	}
	return AdoptedChecks{Checks: checks, PolicyRevision: 4, Digest: ChecksDigest(checks)}
}

func checkPolicyFixture(checks []string, revision int64) RepositoryPolicy {
	roles := map[string]RoleSelection{"soda-coder": {Harness: "codex-0.1.0", Model: "m"}, "soda-reviewer": {Harness: "codex-0.1.0", Model: "m"}}
	return RepositoryPolicy{
		Roles:      roles,
		Publish:    ActorBindingRef{TokenID: 1, ActorID: 2, Kind: OpRefPublish},
		Create:     ActorBindingRef{TokenID: 1, ActorID: 2, Kind: OpPRCreate},
		Review:     ActorBindingRef{TokenID: 3, ActorID: 4, Kind: OpReviewSubmit},
		Merge:      ActorBindingRef{TokenID: 1, ActorID: 2, Kind: OpMerge},
		Repository: 7, Revision: revision, GrantedBy: 1,
		TargetBranch: "refs/heads/main", Checks: checks,
		MergeMethod: MergeFastForward, MaxConcurrent: 1, Enabled: true,
	}
}

func observedChecksFixture(states ...ObservedCheck) ObservedChecks {
	return ObservedChecks{
		Checks: states, NativeRev: 41, ObservedContexts: len(states),
		HeadTip: strings.Repeat("a", 40), BaseTip: strings.Repeat("b", 40),
		Complete: true,
	}
}

func verifyCheckVerdict(t *testing.T, target CheckTarget, adopted AdoptedChecks, current RepositoryPolicy, observed ObservedChecks) CheckAssessment {
	t.Helper()
	assessment, err := VerifyChecks(target, adopted, current, observed, 1700000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := assessment.Validate(); err != nil {
		t.Fatalf("recorded verdict is invalid: %v", err)
	}
	return assessment
}

func TestVerifyChecksPass(t *testing.T) {
	adopted := adoptedChecksFixture()
	assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted,
		checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision),
		observedChecksFixture(
			ObservedCheck{Context: "st11-build", State: "success"},
			ObservedCheck{Context: "st11-review-gate", State: "success"},
			// Extra non-required evidence never authorizes or blocks.
			ObservedCheck{Context: "unrelated/context", State: "success"},
		))
	if assessment.Verdict != CheckPass || assessment.Reason != CheckReasonPass {
		t.Fatalf("verdict: %+v", assessment)
	}
	if len(assessment.Results) != 2 || !assessment.Results[0].Passed || !assessment.Results[1].Passed {
		t.Fatalf("results: %+v", assessment.Results)
	}
	if assessment.NativeRev != 41 || assessment.PolicyRevision != 4 || assessment.ChecksDigest != adopted.Digest {
		t.Fatalf("bindings: %+v", assessment)
	}
}

func TestVerifyChecksWaitOnPendingOrMissing(t *testing.T) {
	adopted := adoptedChecksFixture()
	for name, states := range map[string][]ObservedCheck{
		"pending": {{Context: "st11-build", State: "success"}, {Context: "st11-review-gate", State: "pending"}},
		"missing": {{Context: "st11-build", State: "success"}},
		"unrequired success cannot satisfy": {
			{Context: "st11-build", State: "success"},
			{Context: "unrelated/context", State: "success"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted,
				checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision),
				observedChecksFixture(states...))
			if assessment.Verdict != CheckPending || assessment.Reason != CheckReasonPending {
				t.Fatalf("verdict: %+v", assessment)
			}
		})
	}
}

func TestVerifyChecksFailClosed(t *testing.T) {
	adopted := adoptedChecksFixture("st11-only")
	cases := map[string]string{
		"failure":   CheckReasonFailed,
		"error":     CheckReasonError,
		"cancelled": CheckReasonCancelled,
		"skipped":   CheckReasonSkipped,
		"warning":   CheckReasonWarning,
		"bogus":     CheckReasonUnknownState,
		"SUCCESS":   CheckReasonUnknownState,
		" success":  CheckReasonUnknownState,
	}
	for state, reason := range cases {
		t.Run("state_"+state, func(t *testing.T) {
			assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted,
				checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision),
				observedChecksFixture(ObservedCheck{Context: "st11-only", State: state}))
			if assessment.Verdict != CheckFailed || assessment.Reason != reason {
				t.Fatalf("verdict: %+v", assessment)
			}
			if assessment.Results[0].Passed {
				t.Fatalf("non-success passed: %+v", assessment.Results)
			}
		})
	}
}

func TestVerifyChecksWorstStateWins(t *testing.T) {
	adopted := adoptedChecksFixture("st11-a", "st11-b", "st11-c")
	assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted,
		checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision),
		observedChecksFixture(
			ObservedCheck{Context: "st11-a", State: "pending"},
			ObservedCheck{Context: "st11-b", State: "failure"},
			ObservedCheck{Context: "st11-c", State: "success"},
		))
	if assessment.Verdict != CheckFailed || assessment.Reason != CheckReasonFailed {
		t.Fatalf("verdict: %+v", assessment)
	}
}

func TestVerifyChecksRefusals(t *testing.T) {
	adopted := adoptedChecksFixture()
	policy := checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision)
	observed := observedChecksFixture(
		ObservedCheck{Context: "st11-build", State: "success"},
		ObservedCheck{Context: "st11-review-gate", State: "success"},
	)
	t.Run("stale head", func(t *testing.T) {
		moved := observed
		moved.HeadTip = strings.Repeat("c", 40)
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, policy, moved)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonStaleHead {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
	t.Run("stale base", func(t *testing.T) {
		moved := observed
		moved.BaseTip = strings.Repeat("c", 40)
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, policy, moved)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonStaleBase {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
	t.Run("changed definitions", func(t *testing.T) {
		changed := checkPolicyFixture([]string{"st11-build", "st11-review-gate", "st11-added"}, adopted.PolicyRevision+1)
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, changed, observed)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonDefinitionsChanged {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
	t.Run("same set new revision still needs adoption", func(t *testing.T) {
		changed := checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision+1)
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, changed, observed)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonDefinitionsChanged {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
	t.Run("zero configured checks", func(t *testing.T) {
		var missing RepositoryPolicy
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, missing, observed)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonPolicyEmpty {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
	t.Run("incomplete evidence", func(t *testing.T) {
		partial := observed
		partial.Complete = false
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, policy, partial)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonEvidenceIncomplete {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
	t.Run("hidden evidence", func(t *testing.T) {
		hidden := observed
		hidden.Hidden = true
		assessment := verifyCheckVerdict(t, checkTargetFixture(), adopted, policy, hidden)
		if assessment.Verdict != CheckRefused || assessment.Reason != CheckReasonHidden {
			t.Fatalf("verdict: %+v", assessment)
		}
	})
}

func TestVerifyChecksRejectsMalformedInputs(t *testing.T) {
	adopted := adoptedChecksFixture()
	policy := checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision)
	observed := observedChecksFixture(ObservedCheck{Context: "st11-build", State: "success"})
	badTarget := checkTargetFixture()
	badTarget.HeadOID = "short"
	if _, err := VerifyChecks(badTarget, adopted, policy, observed, 1700000000); err == nil {
		t.Fatal("bad target verified")
	}
	badAdopted := adopted
	badAdopted.Digest = strings.Repeat("0", 64)
	if _, err := VerifyChecks(checkTargetFixture(), badAdopted, policy, observed, 1700000000); err == nil {
		t.Fatal("bad adoption verified")
	}
	if _, err := VerifyChecks(checkTargetFixture(), adopted, policy, observed, 0); err == nil {
		t.Fatal("missing record time verified")
	}
}

func TestChecksDigestIgnoresOrder(t *testing.T) {
	if ChecksDigest([]string{"b", "a"}) != ChecksDigest([]string{"a", "b"}) {
		t.Fatal("digest depends on order")
	}
	if ChecksDigest([]string{"a"}) == ChecksDigest([]string{"a", "b"}) {
		t.Fatal("digest ignores set membership")
	}
}

func TestCheckAssessmentValidation(t *testing.T) {
	adopted := adoptedChecksFixture()
	policy := checkPolicyFixture(append([]string(nil), adopted.Checks...), adopted.PolicyRevision)
	good := verifyCheckVerdict(t, checkTargetFixture(), adopted, policy,
		observedChecksFixture(
			ObservedCheck{Context: "st11-build", State: "success"},
			ObservedCheck{Context: "st11-review-gate", State: "success"},
		))
	bad := good
	bad.Reason = CheckReasonPending
	if bad.Validate() == nil {
		t.Fatal("pass with pending reason validates")
	}
	bad = good
	bad.Results[0].Passed = false
	if bad.Validate() == nil {
		t.Fatal("pass with failing result validates")
	}
	bad = good
	bad.ChecksDigest = strings.Repeat("0", 64)
	if bad.Validate() == nil {
		t.Fatal("pass with foreign digest validates")
	}
}
