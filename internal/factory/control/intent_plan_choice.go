package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// ErrPlanChoiceStale reports a recorded decomposition plan that no longer
// answers the accepted direction: the world moved (another draft reviewed,
// revised scope, another source commit or preparation ready, another head
// accepted) after the run completed. The caller revises the direction and
// plans again; the stale answer never funds a choice.
var ErrPlanChoiceStale = errors.New("recorded plan no longer answers the accepted direction")

// ErrPlanChoiceAttention reports a recorded decomposition answer that
// needs its product decision before native work. The attention text
// travels in the wrapped message; no choice is funded.
var ErrPlanChoiceAttention = errors.New("recorded plan needs a product decision first")

// ErrPlanChoiceStore reports a planning read the choice binding could
// not complete. Serving maps it to an infrastructure failure, never
// to a product refusal.
var ErrPlanChoiceStore = errors.New("planning store unavailable")

// GenerateAcceptedIntentFromRecordedPlan is the bound R13 serving
// connector: it resolves the current accepted head's own recorded
// decomposition choice and generates through the choice-bound pipeline
// in one step, so no caller can resolve a choice and adopt it later
// across an unchecked gap. A still-running plan or a volatile
// preparation waits truthfully through the returned wait.
func (c *Coordinator) GenerateAcceptedIntentFromRecordedPlan(ctx context.Context, intentID string, owner int64) (factory.IntentGeneration, *DispatchWait, error) {
	var empty factory.IntentGeneration
	if c.Store == nil {
		return empty, nil, fmt.Errorf("%w: intent planning store is nil", ErrPlanChoiceStore)
	}
	r, err := c.Store.IntentRevision(ctx, intentID, owner)
	if err != nil {
		return empty, nil, err
	}
	if r.State != "accepted" {
		return empty, nil, errors.New("planned choice serves the accepted direction")
	}
	chosen, wait, err := c.ResolveAcceptedPlanChoice(ctx, owner, r)
	if err != nil || wait != nil {
		return empty, wait, err
	}
	g, err := c.GenerateAcceptedIntentFromChoice(ctx, intentID, owner, chosen)
	if err != nil {
		return empty, nil, err
	}
	return g, nil, nil
}

// ResolveAcceptedPlanChoice binds the current accepted head back to its
// own recorded plan: the completed decomposition_options run the exact
// reviewed draft (this acceptance's predecessor) settled through the
// shared finish boundary. It returns the single funded comparative
// choice for one multi-candidate answer, or the lone candidate with
// zero comparative calls. Anything else refuses before any funded
// choice: a missing attempt, a corrupt staged input, a failed or
// cancelled run, an attention answer, revised scope, a stale source,
// preparation or head — each with the stored assignment, staged input
// and result as evidence. A still-running attempt waits truthfully
// through the returned wait. The funded comparison takes time, so
// source, preparation and head currency are rechecked after the choice
// before adoption.
func (c *Coordinator) ResolveAcceptedPlanChoice(ctx context.Context, owner int64, r factory.IntentRevision) (ChosenIntentPlan, *DispatchWait, error) {
	var empty ChosenIntentPlan
	if c.Store == nil {
		return empty, nil, fmt.Errorf("%w: intent planning store is nil", ErrPlanChoiceStore)
	}
	if err := admitAcceptedChoiceHead(r, owner); err != nil {
		return empty, nil, err
	}
	a, err := c.Store.IntentPlanAttempt(ctx, r.Intent, r.Predecessor)
	if errors.Is(err, store.ErrNotFound) {
		return empty, nil, errors.New("the reviewed draft has no recorded decomposition plan")
	}
	if err != nil {
		return empty, nil, fmt.Errorf("%w: plan attempt unreadable", ErrPlanChoiceStore)
	}
	in, err := bindRecordedPlanInput(a, owner, r)
	if err != nil {
		return empty, nil, err
	}
	if in.Operation != factory.IntentPlanningOptions {
		return empty, nil, errors.New("accepted direction carries no recorded decomposition choice")
	}
	out, wait, err := settlePlanChoiceStatus(a)
	if err != nil || wait != nil {
		return empty, wait, err
	}
	if err := bindRecordedScopeCoverage(r, out); err != nil {
		return empty, nil, err
	}
	if wait, err := c.recheckPlanChoiceSource(ctx, r, a, in); err != nil || wait != nil {
		return empty, wait, err
	}
	if err := c.currentAcceptedIntent(ctx, r); err != nil {
		return empty, nil, fmt.Errorf("%w: accepted direction changed during choice", ErrPlanChoiceStale)
	}
	chosen, err := c.ChoosePlannedCandidate(ctx, r, in, *out)
	if err != nil {
		return empty, nil, err
	}
	chosen.Attempt = a.ID
	if wait, err := c.recheckPlanChoiceAfterComparison(ctx, r, a, in); err != nil || wait != nil {
		return empty, wait, err
	}
	if chosen.Intent != r.Intent {
		return empty, nil, errors.New("planned choice answers another intent")
	}
	return chosen, nil, nil
}

// admitAcceptedChoiceHead proves the caller serves the owning accepted
// direction carrying a reviewed draft before any recorded-plan read.
func admitAcceptedChoiceHead(r factory.IntentRevision, owner int64) error {
	if r.State != "accepted" {
		return errors.New("planned choice serves the accepted direction")
	}
	if r.Owner != owner || r.Owner <= 0 {
		return errors.New("planned choice answers its owning reviewer")
	}
	if r.Predecessor == "" {
		return errors.New("accepted direction carries no reviewed draft plan")
	}
	return nil
}

// bindRecordedPlanInput recovers the exact staged input a plan attempt
// recorded and proves it answers the funded operation behind this
// acceptance: the staged head must equal the assignment subject triple
// the shared finish boundary settled, its revision must be the exact
// reviewed draft this acceptance carries as predecessor, and the
// accepted direction must preserve the reviewed substantive scope.
func bindRecordedPlanInput(a factory.Assignment, owner int64, r factory.IntentRevision) (factory.SemanticPlannerInput, error) {
	var empty factory.SemanticPlannerInput
	subj := a.Subject.Intent
	if subj == nil || subj.OwnerID != owner {
		return empty, errors.New("recorded plan answers another owner")
	}
	in, staged, err := ParseStagedPlanningInput(a.Prompt)
	if err != nil || !staged {
		return empty, errors.New("recorded planning input is missing or corrupt")
	}
	if in.Head.Digest() != subj.ScopeDigest || in.Head.ID != subj.RevisionID ||
		in.Head.Intent != subj.IntentID || in.Head.Owner != subj.OwnerID {
		return empty, fmt.Errorf("%w: recorded plan answers another funded operation", ErrPlanChoiceStale)
	}
	if in.Head.ID != r.Predecessor {
		return empty, fmt.Errorf("%w: accepted direction reviewed another draft", ErrPlanChoiceStale)
	}
	if err := bindAcceptedScope(r, in.Head); err != nil {
		return empty, err
	}
	return in, nil
}

// bindAcceptedScope freezes the reviewed substantive planning content
// across draft to accepted: the objective, the repository, and every
// reviewed product choice's description, correction, consequence,
// provenance and decision. Only lifecycle revision, state and
// acceptance metadata plus planner-generated task differences may
// change; a revised product direction refuses before any funded
// comparison instead of spending on an answer to another intent.
func bindAcceptedScope(r, draft factory.IntentRevision) error {
	if r.Objective != draft.Objective {
		return fmt.Errorf("%w: accepted direction revised its reviewed objective", ErrPlanChoiceStale)
	}
	if r.Repository != draft.Repository {
		return fmt.Errorf("%w: accepted direction moved repositories", ErrPlanChoiceStale)
	}
	reviewed := make(map[string]factory.ProductIdea, len(draft.Ideas))
	for _, idea := range draft.Ideas {
		reviewed[idea.ID] = idea
	}
	if len(r.Ideas) != len(reviewed) {
		return fmt.Errorf("%w: accepted direction changed its reviewed product choices", ErrPlanChoiceStale)
	}
	for _, idea := range r.Ideas {
		prior, ok := reviewed[idea.ID]
		if !ok {
			return fmt.Errorf("%w: accepted direction adds an unreviewed product choice", ErrPlanChoiceStale)
		}
		if idea.Description != prior.Description || idea.Correction != prior.Correction ||
			idea.Consequence != prior.Consequence || idea.Provenance != prior.Provenance ||
			idea.Decision != prior.Decision {
			return fmt.Errorf("%w: accepted direction revised its reviewed product choice", ErrPlanChoiceStale)
		}
	}
	return nil
}

// bindRecordedScopeCoverage proves every recorded candidate fits inside
// the current accepted scope before any funded comparison, mirroring
// the adoption boundary's own rule. Scope that the acceptance no
// longer authorizes refuses here, not after spending on a choice the
// graph validation would reject.
func bindRecordedScopeCoverage(r factory.IntentRevision, out *factory.SemanticPlannerOutput) error {
	for _, candidate := range out.Candidates {
		work := r
		work.Tasks = append([]factory.IntentTask(nil), candidate.Tasks...)
		if err := work.Validate(); err != nil {
			return fmt.Errorf("%w: recorded choice exceeds accepted scope", ErrPlanChoiceStale)
		}
	}
	return nil
}

// settlePlanChoiceStatus reads one recorded attempt the way the draft
// status view does: a still-open run waits, a failed or cancelled run
// refuses with its recorded summary, a reported completed run yields
// its stored answer, and any other reported outcome carries the
// planner's attention text back to the product decision.
func settlePlanChoiceStatus(a factory.Assignment) (*factory.SemanticPlannerOutput, *DispatchWait, error) {
	switch {
	case a.Stage != factory.AssignmentFinished || a.Result == nil:
		return nil, &DispatchWait{Repository: a.Repository, Reason: WaitAssigned, Detail: "recorded decomposition plan is still running"}, nil
	case a.Outcome == factory.Failed:
		return nil, nil, fmt.Errorf("recorded decomposition failed: %s", a.Result.Summary)
	case a.Outcome == factory.Cancelled:
		return nil, nil, fmt.Errorf("recorded decomposition was cancelled: %s", a.Result.Summary)
	case a.Result.Reported && a.Result.PhaseResult != nil && a.Result.PhaseResult.Status == "completed":
		if a.Result.PhaseResult.Output == nil || a.Result.PhaseResult.Output.Decomposition == nil {
			return nil, nil, errors.New("recorded plan carries no decomposition result")
		}
		return a.Result.PhaseResult.Output.Decomposition, nil, nil
	default:
		attention := ""
		if pr := a.Result.PhaseResult; pr != nil && pr.Output != nil && pr.Output.Decomposition != nil {
			attention = pr.Output.Decomposition.Attention
		}
		if attention == "" {
			attention = "the recorded plan needs its product decision before native work"
		}
		return nil, nil, fmt.Errorf("%w: %s", ErrPlanChoiceAttention, attention)
	}
}

// recheckPlanChoiceSource proves the recorded plan still inspects the
// current source: the ready role preparation behind this repository
// must be the exact preparation the run settled — same identity, not
// merely the same commit — and every staged evidence reference must
// answer that preparation's source commit. A volatile preparation
// waits truthfully; a moved source or a replacement preparation
// refuses as stale.
func (c *Coordinator) recheckPlanChoiceSource(ctx context.Context, r factory.IntentRevision, a factory.Assignment, in factory.SemanticPlannerInput) (*DispatchWait, error) {
	proj, err := c.Store.ProjectByRepository(ctx, r.Repository)
	if errors.Is(err, store.ErrNotFound) {
		return &DispatchWait{Repository: r.Repository, Reason: WaitAuthority, Detail: "project changed during accepted planning choice"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: project unreadable", ErrPlanChoiceStore)
	}
	plan := &attemptPlan{projectID: proj.ID}
	first, second := selectPreparationFor(ctx, c.dispatchDeps(), plan, project.RoleCoder, "")
	if second != nil {
		return nil, fmt.Errorf("%w: %s", ErrPlanChoiceStore, second.Detail)
	}
	if first != nil {
		wait := timeOnlyWait(r.Repository, first)
		return &wait, nil
	}
	if a.Preparation == "" {
		return nil, errors.New("recorded plan carries no preparation binding")
	}
	if plan.prep.Preparation.ID != a.Preparation {
		return nil, fmt.Errorf("%w: recorded plan answers another preparation", ErrPlanChoiceStale)
	}
	if plan.prep.Preparation.SourceCommit != a.SourceCommit {
		return nil, fmt.Errorf("%w: recorded plan inspects another source commit", ErrPlanChoiceStale)
	}
	for _, ref := range in.Inspected {
		if ref.Commit != a.SourceCommit {
			return nil, fmt.Errorf("%w: recorded evidence answers another source commit", ErrPlanChoiceStale)
		}
	}
	return nil, nil
}

// recheckPlanChoiceAfterComparison closes the funded-comparison gap: a
// comparative choice takes time, so source, preparation and head
// currency are proven again after ChoosePlannedCandidate returns and
// before the choice may reach native effects. Reads only; no provider
// call runs inside a transaction here.
func (c *Coordinator) recheckPlanChoiceAfterComparison(ctx context.Context, r factory.IntentRevision, a factory.Assignment, in factory.SemanticPlannerInput) (*DispatchWait, error) {
	if wait, err := c.recheckPlanChoiceSource(ctx, r, a, in); err != nil || wait != nil {
		return wait, err
	}
	if err := c.currentAcceptedIntent(ctx, r); err != nil {
		return nil, fmt.Errorf("%w: accepted direction changed during choice", ErrPlanChoiceStale)
	}
	return nil, nil
}

// validateGenerationBinding proves a generated operation's stamped
// recorded plan still answers the current direction: the exact trusted
// assignment row, its staged input against the current head, the
// accepted scope freeze, per-candidate coverage, and exact
// preparation/source/head currency. Pure reads; no comparative choice
// is funded and no provider is called, so submission and adoption can
// re-prove the binding after slow external reads. Generations without
// a stamped binding predate this contract and fail closed here;
// callers bind them before resuming.
func (c *Coordinator) validateGenerationBinding(ctx context.Context, g factory.IntentGeneration) (*DispatchWait, error) {
	if g.PlanAssignment == "" {
		return nil, errors.New("generation carries no recorded plan binding")
	}
	a, err := c.Store.Assignment(ctx, g.PlanAssignment)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("%w: recorded plan is unavailable", ErrPlanChoiceStale)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: recorded plan unreadable", ErrPlanChoiceStore)
	}
	subj := a.Subject.Intent
	if subj == nil || subj.OwnerID <= 0 {
		return nil, errors.New("recorded plan answers another owner")
	}
	r, err := c.Store.IntentRevision(ctx, subj.IntentID, subj.OwnerID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("%w: accepted direction is unavailable", ErrPlanChoiceStale)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: accepted head unreadable", ErrPlanChoiceStore)
	}
	in, err := bindRecordedPlanInput(a, subj.OwnerID, r)
	if err != nil {
		return nil, err
	}
	out, wait, err := settlePlanChoiceStatus(a)
	if err != nil || wait != nil {
		return wait, err
	}
	if err := bindRecordedScopeCoverage(r, out); err != nil {
		return nil, err
	}
	if wait, err := c.recheckPlanChoiceSource(ctx, r, a, in); err != nil || wait != nil {
		return wait, err
	}
	if err := c.currentAcceptedIntent(ctx, r); err != nil {
		return nil, fmt.Errorf("%w: accepted direction changed", ErrPlanChoiceStale)
	}
	return nil, nil
}
