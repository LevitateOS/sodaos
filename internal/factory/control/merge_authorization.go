package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// AuthorizeMerge binds one ready authorization generation to a recorded
// merge row (D09 registration). The first generation binds to a
// recorded, still-undriven open row; later generations replace a
// finalized predecessor through the bounded ledger history: the exact
// finalized predecessor snapshots once, the generation increments once,
// and only the generation-owned tuple, authority, deadline and
// admission-withdrawal move. Every binding enforces the FW37 union gate
// — the journaled current applicable D07 delivery acceptance behind the
// generation, confirmed accepted resolutions and matching authority —
// so a generation can never authorize another candidate. Registration
// happens before the first drive: accepted fixes resolve and read back
// before any intent is recorded. A binding without its union refuses:
// stale or absent acceptance blocks every new registration. A pending,
// uncertain or committed predecessor forbids replacement: committed
// effects recover and adopt instead of replacing. Recovery of recorded
// effects never enters this path. The future Jev producer calls this
// seam once funded judgment accepts the admitted correction; until then
// no production caller binds, and new submits wait honestly instead of
// merging ungated.
func (c *Coordinator) AuthorizeMerge(ctx context.Context, publicationID string, auth factory.MergeAuthorization) error {
	m, err := c.Store.MergeByPublication(ctx, publicationID)
	if err != nil {
		return err
	}
	if err := auth.Validate(); err != nil {
		return err
	}
	// Ledger position: the binding generation is always the next in
	// line — one on an unbound row, the bound current plus one behind
	// its history. A second generation without its predecessor
	// history, a rebinding of the current generation and any
	// generation gap all refuse here.
	want := len(m.History) + 1
	if m.Authorization.Bound() {
		if m.Authorization.Generation.Generation != len(m.History)+1 {
			return errors.New("merge authorization generation needs its ledger history")
		}
		want = m.Authorization.Generation.Generation + 1
	}
	if auth.Generation.Generation != want {
		return errors.New("merge authorization generation needs its ledger history")
	}
	if auth.Generation.Generation == 1 {
		return c.authorizeFirstGeneration(ctx, m, auth)
	}
	return c.authorizeReplacementGeneration(ctx, m, auth)
}

// authorizeFirstGeneration binds the ready first-generation
// authorization to a recorded, still-undriven open row.
func (c *Coordinator) authorizeFirstGeneration(ctx context.Context, m factory.Merge, auth factory.MergeAuthorization) error {
	if m.Stage != factory.MergeOpen {
		return errors.New("merge authorization needs its open row")
	}
	if m.Authorization.Bound() {
		return errors.New("merge authorization already bound")
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

// authorizeReplacementGeneration binds a fresh authorization generation
// after its predecessor finalized with proven no-effect: a confirmed
// cancellation or a deterministic no-effect refusal. The exact
// finalized predecessor snapshots once into the bounded history, the
// generation increments once, and the live fields become the fresh
// generation: its exact tuple and authority, its undriven operation, no
// observations and a cleared admission. The PR identity never moves, no
// receipt is discarded and no allowance resets. The fresh binding
// re-verifies the journaled current applicable D07 acceptance, the
// current union authority and confirmed accepted resolutions; the store
// transaction rechecks the union binding on commit, and the row refuses
// anything the predecessor proof does not justify.
func (c *Coordinator) authorizeReplacementGeneration(ctx context.Context, m factory.Merge, auth factory.MergeAuthorization) error {
	if auth.Generation.PublicationID != m.PublicationID {
		return errors.New("merge authorization answers another candidate")
	}
	if !m.Authorization.Bound() {
		return errors.New("merge replacement lacks its finalized predecessor")
	}
	if m.Operation.Effect == factory.OpEffectCommitted {
		return errors.New("committed merge forbids replacement")
	}
	if _, err := factory.MergePredecessorProofOf(m.Authorization.Generation.Generation, m.Operation); err != nil {
		return err
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
		CurrentAuthority: auth.Generation.Authority,
	}); err != nil {
		return err
	}
	next := m
	next.History = append(append([]factory.MergeGenerationSnapshot(nil), m.History...),
		factory.SnapshotMergeGeneration(m.Authorization, m.Operation, m.Stage, m.Outcome, m.Reason, m.FinishedUnix))
	next.Authorization = auth
	next.Operation = factory.MergeOperation{Kind: factory.OpMerge}
	next.Authority = auth.Generation.Authority
	next.HeadOID, next.BaseOID = auth.Generation.Head, auth.Generation.Base
	next.WithdrawRequested = false
	next.Stage, next.Outcome, next.Reason, next.FinishedUnix = factory.MergeOpen, "", "", 0
	next.MergedCommit, next.MergedUnix, next.ClosedUnix = "", 0, 0
	next.OriginalClosingOccurrence, next.OriginalClosedUnix = "", 0
	next.NativeRev, next.ObservedUnix = 0, 0
	next.Revision++
	return c.Store.UpdateMerge(ctx, next)
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
