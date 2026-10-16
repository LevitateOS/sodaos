package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// AuthorizeMerge binds one ready first-generation authorization to a
// recorded, still-undriven merge row (D09 registration). The binding
// enforces the FW37 union gate — the journaled current applicable D07
// delivery acceptance behind the generation, confirmed accepted
// resolutions and matching authority — plus the exact row tuple, so a
// generation can never authorize another candidate. Registration happens
// before the first drive: accepted fixes resolve and read back before any
// intent is recorded. A binding without its union refuses: stale or
// absent acceptance blocks every new registration. Later generations need
// the bounded ledger history in store rows and stay refused here;
// recovery of recorded effects never enters this path. The future Jev
// producer calls this seam once funded judgment accepts the admitted
// correction; until then no production caller binds, and new submits
// wait honestly instead of merging ungated.
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
	if !auth.Generation.UnionBound() {
		return errors.New("merge authorization lacks its acceptance binding")
	}
	p, err := c.Store.PublicationByAssignment(ctx, m.AssignmentID)
	if err != nil {
		return errors.New("merge authorization lacks its publication")
	}
	acceptance, ok := journaledAcceptance(p, auth.Generation.AcceptanceID)
	if !ok {
		return errors.New("delivery acceptance unrecorded for its merge binding")
	}
	if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
		Publication: p, Acceptance: acceptance, Generation: auth.Generation,
		Decision: auth.Decision, ResolutionsConfirmed: auth.ResolutionsConfirmed,
		CurrentAuthority: m.Authority,
	}); err != nil {
		return err
	}
	m.Authorization = auth
	m.Revision++
	return c.Store.UpdateMerge(ctx, m)
}

// journaledAcceptance returns the publication's journaled acceptance with
// the given identity. Journals are append-only, so a missing identity
// never appears later for the same binding.
func journaledAcceptance(p factory.Publication, id string) (factory.DeliveryAcceptance, bool) {
	for _, acceptance := range p.DeliveryAcceptances {
		if acceptance.ID == id {
			return acceptance, true
		}
	}
	return factory.DeliveryAcceptance{}, false
}

// mergeAuthorizationGate enforces the common union gate before any new
// native mutation on the merge path. Publication progress, MergePass,
// mergeOne and reconciliation all pass through it; recovery of an
// already-recorded operation (lookup, receipt adoption, completion and
// cancellation) never enters it. An unbound row waits for its
// authorization instead of submitting; a stale binding waits instead of
// merging another candidate. Union-bound rows re-verify their journaled
// acceptance on every pass; earlier funded-decision-only bindings keep
// their legacy gate so committed effects reconcile untouched. No branch
// spends or mutates.
func mergeAuthorizationGate(m factory.Merge, p factory.Publication) error {
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
	if auth.Generation.UnionBound() {
		acceptance, ok := journaledAcceptance(p, auth.Generation.AcceptanceID)
		if !ok {
			return &factory.PublicationWait{Reason: "merge_authorization_stale"}
		}
		// The row authority was just verified current, so this
		// re-verifies the immutable union binding against it: the
		// journaled current applicable acceptance behind the exact
		// generation with confirmed accepted resolutions.
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: p, Acceptance: acceptance, Generation: auth.Generation,
			Decision: auth.Decision, ResolutionsConfirmed: auth.ResolutionsConfirmed,
			CurrentAuthority: m.Authority,
		}); err != nil {
			return &factory.PublicationWait{Reason: "merge_authorization_stale"}
		}
		return nil
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

// MergeUnionReadyInput gathers the FW37 union submit gate: the eligible
// publication, its current applicable journaled acceptance, the union
// generation seeking admission with its decision evidence and confirmed
// resolutions, the current authority, and the re-evaluated native
// mergeable + reviewable expression. Both acceptance variants pass these
// identical gates; native required checks, approvals and blocking human
// review still apply to either, so a clear review never waives them.
type MergeUnionReadyInput struct {
	Publication          factory.Publication
	Acceptance           factory.DeliveryAcceptance
	Generation           factory.MergeGeneration
	Decision             factory.DeliveryDecision
	ResolutionsConfirmed bool
	CurrentAuthority     factory.AuthorityRef
	Native               MergeNativeGate
}

// CheckMergeUnionReady enforces the deterministic T45/T46 gate before any
// new mutation on the union path. Every branch below is cheap: malformed
// inputs refuse, superseded or protected merges refuse, and transient or
// incomplete evidence waits. A nil error means the caller may proceed to
// its native mutation; any error means it must not.
func CheckMergeUnionReady(in MergeUnionReadyInput) error {
	if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
		Publication: in.Publication, Acceptance: in.Acceptance, Generation: in.Generation,
		Decision: in.Decision, ResolutionsConfirmed: in.ResolutionsConfirmed,
		CurrentAuthority: in.CurrentAuthority,
	}); err != nil {
		return err
	}
	if err := in.Native.Validate(); err != nil {
		return err
	}
	// Mergeable + reviewable re-evaluated: fresh native tips must still
	// match the generation; hidden or incomplete evidence blocks. A
	// blocking independent review waits; revoked or protected merges
	// refuse before any mutation. Neither variant waives this section.
	if !in.Native.Visible || !in.Native.Complete {
		return &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
	}
	if in.Native.HeadOID != in.Generation.Head {
		return &factory.PublicationRefusal{Reason: "superseded_head"}
	}
	if in.Native.BaseOID != in.Generation.Base {
		return &factory.PublicationRefusal{Reason: "superseded_base"}
	}
	if in.Native.ChangesRequested {
		return &factory.PublicationWait{Reason: "changes_requested"}
	}
	if !in.Native.Approved {
		return &factory.PublicationWait{Reason: "approval_missing"}
	}
	if !in.Native.Eligible {
		if mergeGateProtected(in.Native.Blocker) {
			return &factory.PublicationRefusal{Reason: in.Native.Blocker}
		}
		return &factory.PublicationWait{Reason: in.Native.Blocker}
	}
	return nil
}
