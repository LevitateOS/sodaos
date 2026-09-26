package factory

import (
	"strings"
	"testing"
	"time"
)

func admitted(t *testing.T) (Attempt, time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	a, err := New(WorkItem{RepositoryID: 7, Issue: 1, HumanID: 3, Objective: "fix the failing check", BaseSHA: strings.Repeat("a", 40), PolicySHA: strings.Repeat("a", 64)}, "delivery-1", now)
	if err != nil {
		t.Fatal(err)
	}
	return a, now
}

func candidate(t *testing.T, a *Attempt, role Role, sha string, now time.Time) Run {
	t.Helper()
	r, err := a.BeginRun(role, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CandidateFrom(r, sha, now); err != nil {
		t.Fatal(err)
	}
	return r
}

func evidence(t *testing.T, a *Attempt, passed bool, now time.Time) {
	t.Helper()
	if err := a.BeginCI(now); err != nil {
		t.Fatal(err)
	}
	if err := a.BeginCI(now); err == nil {
		t.Fatal("duplicate CI consumed another evaluation")
	}
	if err := a.RecordCI(Evidence{ID: "actions-1", Commit: a.Candidate, Passed: passed}, now); err != nil {
		t.Fatal(err)
	}
	r, err := a.BeginRun(Review, now)
	if err != nil {
		t.Fatal(err)
	}
	r.Outcome, r.CleanupComplete = Succeeded, true
	if err := a.RecordReview(r, passed, now); err != nil {
		t.Fatal(err)
	}
	if err := a.Evaluate(now); err != nil {
		t.Fatal(err)
	}
}

func TestOneRepairEndsWithNewEvidenceOrHuman(t *testing.T) {
	for _, finalPass := range []bool{true, false} {
		a, now := admitted(t)
		candidate(t, &a, Implementation, strings.Repeat("b", 40), now)
		evidence(t, &a, false, now)
		if a.Phase != Fix {
			t.Fatal(a.Phase)
		}
		candidate(t, &a, Repair, strings.Repeat("c", 40), now)
		if a.CI != nil || a.Review != nil {
			t.Fatal("old evidence survived candidate change")
		}
		evidence(t, &a, finalPass, now)
		want := NeedsHuman
		if finalPass {
			want = Succeeded
		}
		if a.Outcome != want || a.Executions != 4 || a.CIEvaluations != 2 {
			t.Fatal(a)
		}
		if _, err := a.BeginRun(Repair, now); err == nil {
			t.Fatal("terminal attempt started a second repair")
		}
	}
}

func TestCandidateRequiresCurrentActiveImplementation(t *testing.T) {
	a, now := admitted(t)
	r, err := a.BeginRun(Implementation, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.BeginRun(Implementation, now); err == nil {
		t.Fatal("duplicate launch")
	}
	r.Role = Review
	if err := a.CandidateFrom(r, strings.Repeat("b", 40), now); err == nil {
		t.Fatal("reviewer published code")
	}
	r.Role = Implementation
	r.Outcome = Cancelled
	if err := a.CandidateFrom(r, strings.Repeat("b", 40), now); err == nil {
		t.Fatal("cancelled run published code")
	}
	r.Outcome = ""
	if err := a.CandidateFrom(r, strings.Repeat("b", 40), r.Deadline); err == nil {
		t.Fatal("expired run published code")
	}
	a.Finish(Cancelled, "cancelled by human")
	if err := a.CandidateFrom(r, strings.Repeat("b", 40), now); err == nil {
		t.Fatal("cancelled attempt published code")
	}
}

func TestWrongCommitAndIncompleteReviewCannotComplete(t *testing.T) {
	a, now := admitted(t)
	candidate(t, &a, Implementation, strings.Repeat("b", 40), now)
	if err := a.BeginCI(now); err != nil {
		t.Fatal(err)
	}
	if err := a.RecordCI(Evidence{ID: "actions-old", Commit: a.Work.BaseSHA, Passed: true}, now); err == nil {
		t.Fatal("stale CI accepted")
	}
	if err := a.Evaluate(now); err == nil {
		t.Fatal("candidate completed without evidence")
	}
	r, err := a.BeginRun(Review, now)
	if err != nil {
		t.Fatal(err)
	}
	r.Outcome = Succeeded
	if err := a.RecordReview(r, true, now); err == nil {
		t.Fatal("review accepted with leaked workspace")
	}
	r.CleanupComplete = true
	r.InputSHA = a.Work.BaseSHA
	if err := a.RecordReview(r, true, now); err == nil {
		t.Fatal("stale review accepted")
	}
}

func TestRoleDeadlinesAreCappedByAttemptDeadline(t *testing.T) {
	a, now := admitted(t)
	r, err := a.BeginRun(Implementation, now)
	if err != nil || r.Deadline.Sub(now) != 90*time.Minute {
		t.Fatal(r, err)
	}
	if err := a.CandidateFrom(r, strings.Repeat("b", 40), now); err != nil {
		t.Fatal(err)
	}
	late := a.Deadline.Add(-time.Minute)
	r, err = a.BeginRun(Review, late)
	if err != nil || !r.Deadline.Equal(a.Deadline) {
		t.Fatal(r, err)
	}
	if _, err := a.BeginRun(Repair, a.Deadline); err == nil {
		t.Fatal("attempt deadline did not withdraw authority")
	}
}
