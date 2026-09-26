package factory

import (
	"errors"
	"time"
)

func (a Attempt) Authority(now time.Time) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Phase == Finished || !now.Before(a.Deadline) {
		return errors.New("factory authority is inactive or expired")
	}
	return nil
}

func (a *Attempt) Finish(outcome Outcome, summary string) {
	a.Phase, a.Outcome, a.Summary = Finished, outcome, summary
}

// BeginRun consumes an execution before launching it. The caller persists the
// returned run and updated attempt together before allocating any resources.
func (a *Attempt) BeginRun(role Role, now time.Time) (Run, error) {
	if err := a.Authority(now); err != nil {
		return Run{}, err
	}
	want, expected := a.expectedRun()
	if role != want || a.Executions != expected {
		return Run{}, errors.New("execution role or attempt budget does not permit launch")
	}
	limit := 30 * time.Minute
	input := a.Candidate
	if role == Implementation {
		limit, input = 90*time.Minute, a.Work.BaseSHA
	}
	deadline := now.Add(limit)
	if a.Deadline.Before(deadline) {
		deadline = a.Deadline
	}
	r := Run{ID: NewID(), AttemptID: a.ID, Role: role, InputSHA: input, Started: now, Deadline: deadline}
	a.Executions++
	a.CleanupComplete = false
	return r, nil
}

func (a Attempt) expectedRun() (Role, int) {
	switch a.Phase {
	case Implement:
		return Implementation, 0
	case Verify:
		return Review, 1
	case Fix:
		return Repair, 2
	case Reverify:
		return Review, 3
	default:
		return "", -1
	}
}

func (a *Attempt) CandidateFrom(r Run, sha string, now time.Time) error {
	if err := a.Authority(now); err != nil {
		return err
	}
	if err := r.Authority(a.ID, now); err != nil {
		return err
	}
	if !ValidCommit(sha) || sha == r.InputSHA {
		return errors.New("invalid or unchanged candidate")
	}
	role, input, next := a.publicationInput()
	if role == "" || r.Role != role || r.InputSHA != input {
		return errors.New("run cannot publish in this phase")
	}
	a.Phase, a.Candidate, a.CI, a.Review = next, sha, nil, nil
	return nil
}

func (a Attempt) publicationInput() (Role, string, Phase) {
	switch a.Phase {
	case Implement:
		return Implementation, a.Work.BaseSHA, Verify
	case Fix:
		return Repair, a.Candidate, Reverify
	default:
		return "", "", ""
	}
}

// BeginCI accounts for one evaluation before requesting it from Forgejo Actions.
// A candidate cannot silently reset its CI budget or request a third evaluation.
func (a *Attempt) BeginCI(now time.Time) error {
	if err := a.Authority(now); err != nil {
		return err
	}
	if (a.Phase != Verify && a.Phase != Reverify) || a.Candidate == "" || a.CIEvaluations >= 2 || a.CI != nil {
		return errors.New("CI evaluation is not permitted")
	}
	want := 0
	if a.Phase == Reverify {
		want = 1
	}
	if a.CIEvaluations != want {
		return errors.New("candidate already consumed its CI evaluation")
	}
	a.CIEvaluations++
	return nil
}

func (a *Attempt) RecordCI(e Evidence, now time.Time) error {
	if err := a.Authority(now); err != nil {
		return err
	}
	if (a.Phase != Verify && a.Phase != Reverify) || e.ID == "" || e.Commit != a.Candidate || a.CI != nil {
		return errors.New("CI evidence does not bind to the current candidate")
	}
	want := 1
	if a.Phase == Reverify {
		want = 2
	}
	if a.CIEvaluations != want {
		return errors.New("CI evidence has no admitted evaluation")
	}
	a.CI = &e
	return nil
}

func (a *Attempt) RecordReview(r Run, passed bool, now time.Time) error {
	if err := a.Authority(now); err != nil {
		return err
	}
	if (a.Phase != Verify && a.Phase != Reverify) || a.Review != nil {
		return errors.New("review is not permitted in this phase")
	}
	e, err := r.reviewEvidence(a.ID, a.Candidate)
	if err != nil {
		return err
	}
	e.Passed = passed
	a.Review = &e
	return nil
}

func (a *Attempt) Evaluate(now time.Time) error {
	if err := a.Authority(now); err != nil {
		return err
	}
	if a.CI == nil || a.Review == nil || a.CI.Commit != a.Candidate || a.Review.Commit != a.Candidate {
		return errors.New("candidate needs both CI and independent review")
	}
	if a.CI.Passed && a.Review.Passed {
		a.Finish(Succeeded, "candidate verified; ready for human merge")
		return nil
	}
	if a.Phase == Verify {
		a.Phase = Fix
	} else {
		a.Finish(NeedsHuman, "candidate still fails after the single permitted repair")
	}
	return nil
}
