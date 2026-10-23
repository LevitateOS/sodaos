package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

func currentReviewTx(ctx context.Context, tx *sql.Tx, a factory.Assignment, out *factory.CurrentContext) error {
	if a.ReviewContext == nil {
		out.Evidence.MissingReasons = append(out.Evidence.MissingReasons, "required_review_context_unavailable")
		return nil
	}
	if err := a.ReviewContext.ValidateFor(a); err != nil {
		return err
	}
	var owner factory.Assignment
	if err := contextJSONRow(ctx, tx, &owner, `SELECT data FROM factory_assignments WHERE id=$1`, a.PublicationAssignment); err != nil {
		return err
	}
	// A publication link grants only this registered review/correction
	// context; it does not grant the owner's prompt or submissions.
	if a.ActorID <= 0 || owner.ActorID != a.ActorID || owner.Repository != a.Repository || owner.ProjectID != a.ProjectID || owner.AttemptRoot != a.AttemptRoot || owner.Acceptance != a.Acceptance {
		return errors.New("review publication is outside current context grant")
	}
	var p factory.Publication
	if err := contextJSONRow(ctx, tx, &p, `SELECT data FROM factory_publications WHERE assignment=$1 AND repository=$2 AND issue=$3`, a.PublicationAssignment, a.Repository, a.Issue); err != nil {
		return err
	}
	if p.ID != a.ReviewContext.Publication.ID || p.ProjectID != a.ProjectID || p.Acceptance != a.Acceptance || p.AssignmentID != owner.ID {
		return errors.New("review publication differs from assigned cycle")
	}
	found := false
	for _, cycle := range p.ReviewCycles {
		if cycle.ID != a.ReviewContext.Cycle.ID {
			continue
		}
		if cycle != a.ReviewContext.Cycle {
			return errors.New("review context cycle changed")
		}
		found = true
	}
	if !found {
		return errors.New("review context cycle is unavailable")
	}
	tuple := factory.CandidateTuple{Head: p.Candidate, Base: p.BaseSHA, DiffBase: p.CurrentDiffBase(), Revision: p.NativeRev}
	if p.Tuple != nil {
		tuple = *p.Tuple
	}
	if tuple.Validate() != nil {
		return errors.New("current review candidate tuple is unavailable")
	}
	current := *a.ReviewContext
	current.Assessments = []factory.CurrentFindingAssessment{}
	current.Corrections = []factory.CurrentCorrectionFact{}
	current.Heads = []factory.CurrentReviewHead{}
	current.CurrentTuple = &tuple
	current.Publication = factory.CurrentPublicationContext{ID: p.ID, AssignmentID: p.AssignmentID, Revision: p.Revision, PRID: p.PRID, PRNumber: p.PRNumber, Stage: p.Stage}
	if err := currentReviewFactsTx(ctx, tx, a, p, &current); err != nil {
		return err
	}
	out.Role.Review = &current
	if tuple.Head != current.AdmittedTuple.Head || tuple.Base != current.AdmittedTuple.Base || tuple.DiffBase != current.AdmittedTuple.DiffBase {
		out.Evidence.MissingReasons = append(out.Evidence.MissingReasons, "review_candidate_inputs_changed")
	}
	return nil
}
