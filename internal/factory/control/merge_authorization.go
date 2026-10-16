package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// AuthorizeMerge binds one ready first-generation authorization to a
// recorded, still-undriven merge row (D09 registration). The binding
// enforces the common decision gate — a valid generation, an effective
// ready_to_merge decision over its exact fingerprint, confirmed accepted
// resolutions and matching authority — plus the exact row tuple, so a
// generation can never authorize another candidate. Registration happens
// before the first drive: accepted fixes resolve and read back before any
// intent is recorded. Later generations need the bounded ledger history
// in store rows and stay refused here; recovery of recorded effects never
// enters this path. The future Jev producer calls this seam once funded
// judgment accepts the admitted correction; until then no production
// caller binds, and new submits wait honestly instead of merging
// ungated.
func (c *Coordinator) AuthorizeMerge(ctx context.Context, publicationID string, auth factory.MergeAuthorization) error {
	m, err := c.Store.MergeByPublication(ctx, publicationID)
	if err != nil {
		return err
	}
	if m.Stage != factory.MergeOpen {
		return errors.New("merge authorization needs its open row")
	}
	if m.Authorization.Bound() {
		return errors.New("merge authorization already bound")
	}
	if err := auth.Validate(); err != nil {
		return err
	}
	if auth.Generation.PublicationID != m.PublicationID ||
		auth.Generation.Head != m.HeadOID || auth.Generation.Base != m.BaseOID {
		return errors.New("merge authorization answers another candidate")
	}
	if auth.Generation.Authority != m.Authority {
		return errors.New("merge authority changed under its authorization")
	}
	if m.Operation.Work != nil || m.Operation.Attempts != 0 {
		return errors.New("merge authorization precedes its first drive")
	}
	m.Authorization = auth
	m.Revision++
	return c.Store.UpdateMerge(ctx, m)
}

// mergeAuthorizationGate enforces the common decision gate before any new
// native mutation on the merge path. Publication progress, MergePass,
// mergeOne and reconciliation all pass through it; recovery of an
// already-recorded operation (lookup, receipt adoption, completion and
// cancellation) never enters it. An unbound row waits for its
// authorization instead of submitting; a stale binding waits instead of
// merging another candidate. No branch spends or mutates.
func mergeAuthorizationGate(m factory.Merge) error {
	if !m.Authorization.Bound() {
		return &factory.PublicationWait{Reason: "merge_judgment_unavailable"}
	}
	auth := m.Authorization
	if auth.Generation.Head != m.HeadOID {
		return &factory.PublicationWait{Reason: "superseded_head"}
	}
	if auth.Generation.Base != m.BaseOID {
		return &factory.PublicationWait{Reason: "superseded_base"}
	}
	// The row authority was just verified current, so this re-verifies
	// the immutable binding against it: a ready decision over the exact
	// generation fingerprint with confirmed accepted resolutions.
	if err := factory.CheckMergeDecisionGate(factory.MergeDecisionInput{
		Generation: auth.Generation, Decision: auth.Decision,
		CurrentAuthority: m.Authority, ResolutionsConfirmed: auth.ResolutionsConfirmed,
	}); err != nil {
		return &factory.PublicationWait{Reason: "merge_authorization_stale"}
	}
	return nil
}
