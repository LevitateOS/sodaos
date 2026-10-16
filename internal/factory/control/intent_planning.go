package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Pre-issue semantic planning errors. Exhaustion and cancellation stay
// visible to the caller; a missing funded capability is a precise external
// gate, never a substitute output.
var (
	ErrIntentPlanningExhausted   = errors.New("intent planning allowance is exhausted")
	ErrIntentPlanningExternal    = errors.New("funded semantic planning capability is unavailable")
	ErrIntentPlanningCancelled   = errors.New("intent planning was cancelled")
	ErrIntentPlanningInvestigate = errors.New("intent planning needs source investigation first")
)

// SemanticPlanner produces candidate issue graphs from planner outputs.
// The funded implementation calls the enrolled model account; tests and
// admission use deterministic stubs. A nil planner means no funded
// account or capability is enrolled, which is an explicit external gate.
type SemanticPlanner interface {
	PlanSemantic(ctx context.Context, in factory.SemanticPlannerInput) (factory.SemanticPlannerOutput, error)
}

// AdmitIntentPlanning is the pure FW11 decompose admission for pre-issue
// semantic planning. Checks run in fixed order so the same state reports
// the same reason: subject shape, decompose phase with the coder role,
// revision/owner/scope binding against the authoritative head, draft
// planning scope, repository association, then planner input (including
// its source investigation) and finite allowance. No Store, Host or Graph
// is touched: planning launches no repository session, writes no native
// state, invents no fake issue and grants no publication capability. The dispatch boundary loads
// the head and allowance availability before calling; recording the
// assignment and spending the allowance stay with that boundary (FW11
// dispatch files), not this handler.
func AdmitIntentPlanning(subject factory.ExecutionSubject, phase, role string, head factory.IntentRevision, allowanceAvailable bool, in factory.SemanticPlannerInput) error {
	if err := subject.Validate(); err != nil {
		return err
	}
	if subject.Kind != factory.SubjectIntent || subject.Intent == nil {
		return errors.New("intent planning needs its intent subject")
	}
	if phase != project.ExecutionDecompose || !factory.SubjectKindAllowed(subject.Kind, phase) {
		return errors.New("intent planning enters only through decompose")
	}
	if !factory.ValidPhaseRole(phase, role) {
		return errors.New("intent planning needs the coder role")
	}
	if subject.Intent.IntentID != head.Intent || subject.Intent.RevisionID != head.ID {
		return store.ErrStaleRevision
	}
	if subject.Intent.OwnerID != head.Owner || head.Owner <= 0 {
		return errors.New("intent planning owner mismatch")
	}
	if subject.Intent.ScopeDigest != head.Digest() {
		return store.ErrStaleRevision
	}
	if err := factory.ValidateDraftPlanningScope(head); err != nil {
		return err
	}
	// Pre-issue planning runs only after repository association: an
	// unassociated draft stays a private editable draft and cannot enter
	// the generative run. Full repository authority stays with the FW11
	// dispatch packet boundary; this check keeps the planner
	// repository-bound without consulting any accepted graph or history.
	if head.Repository <= 0 {
		return errors.New("intent planning needs its repository association")
	}
	// Investigation before planning is enforced ahead of other input
	// checks so missing inspection surfaces distinctly and directs work
	// to source inspection rather than another planning attempt.
	if len(in.Inspected) == 0 {
		return ErrIntentPlanningInvestigate
	}
	if err := in.Validate(); err != nil {
		return err
	}
	if in.Head.Digest() != head.Digest() {
		return store.ErrStaleRevision
	}
	if !allowanceAvailable {
		return ErrIntentPlanningExhausted
	}
	return nil
}

// PlanIntentDecomposition admits one pre-issue semantic planning run and
// assesses its answer. The handler consumes planner outputs only: no
// accepted graph or history is consulted at any step, and a first
// repository-bound draft with no tasks plans without prerequisites beyond
// its association, attributed effective scope and source investigation. A nil planner reports the
// precise external gate (missing funded account or capability) instead of
// substituting canned output. Provider failures and invalid answers are
// returned for the caller to record; an explicit Retry can start a
// reconciled run, and neither edits nor replay refill the allowance here.
// An attention answer (decomposition_needs_attention) returns without
// error and without candidates: the product decision it names needs human
// review, not another automatic batch.
func PlanIntentDecomposition(ctx context.Context, subject factory.ExecutionSubject, phase, role string, head factory.IntentRevision, allowanceAvailable bool, in factory.SemanticPlannerInput, planner SemanticPlanner) (factory.SemanticPlannerOutput, error) {
	if err := ctx.Err(); err != nil {
		return factory.SemanticPlannerOutput{}, fmt.Errorf("%w: %v", ErrIntentPlanningCancelled, err)
	}
	if err := AdmitIntentPlanning(subject, phase, role, head, allowanceAvailable, in); err != nil {
		return factory.SemanticPlannerOutput{}, err
	}
	if planner == nil {
		return factory.SemanticPlannerOutput{}, fmt.Errorf("%w: no funded semantic account is enrolled for intent %s", ErrIntentPlanningExternal, head.Intent)
	}
	out, err := planner.PlanSemantic(ctx, in)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return factory.SemanticPlannerOutput{}, fmt.Errorf("%w: %v", ErrIntentPlanningCancelled, err)
		}
		return factory.SemanticPlannerOutput{}, err
	}
	if err := factory.AssessSemanticOutput(in, out); err != nil {
		return factory.SemanticPlannerOutput{}, err
	}
	return out, nil
}
