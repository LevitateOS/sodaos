package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// admitSecondAcceptance journals a round-two funded acceptance on a
// stored serving publication: the fresh review and judgment behind a
// second merge generation on the same candidate. It mirrors
// unionPgAdmitJev with the round advanced; the tuple stays exact
// because the H4 tuple refresh joint is separate and absent here.
func admitSecondAcceptance(t *testing.T, ctx context.Context, db *store.Store, p factory.Publication, reviewer string) (factory.Publication, factory.DeliveryAcceptance, factory.DeliveryDecision, string) {
	t.Helper()
	owner, err := db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	digest := correctionTestDigest(t, finding.Key)
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                 factory.ReviewCycleID(p.ID, owner.AttemptRoot, 2, "review"),
		Phase:              "review",
		TriggerDecision:    "second review of the published head",
		TriggerReason:      "second review of the published head",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              2,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintInput := factory.DeliveryFingerprintInput{
		Reviews:       []string{reviewer},
		Conversations: []string{strings.Repeat("3", 64)},
		AttemptRoot:   owner.AttemptRoot,
		PublicationID: p.ID,
		ScopeDigest:   scope,
		FixDigest:     scope,
		DiffDigest:    scope,
		CheckDigest:   scope,
		TreeDigest:    scope,
		Head:          p.Candidate,
		Base:          p.PRCreate.BaseOID,
		Model:         "jev-test",
		Rubric:        factory.DeliveryRubric,
		Round:         2,
	}
	fingerprint, err := factory.DeliveryFingerprint(fingerprintInput)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := readyDeliveryAnswer(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	decision := factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitJevAcceptance(JevAdmissionInput{
		Publication:       p,
		Cycle:             cycle,
		Carried:           []factory.Finding{finding},
		FingerprintInput:  fingerprintInput,
		Fingerprint:       fingerprint,
		Decision:          decision,
		DecisionID:        strings.Repeat("8", 32),
		ScopeDigest:       scope,
		PolicyRevision:    "policy-revision-9",
		CheckDigest:       scope,
		DiffBase:          p.PRCreate.BaseOID,
		ResolutionIntents: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	published.Revision = p.Revision + 1
	if err := db.UpdatePublication(ctx, published); err != nil {
		t.Fatal(err)
	}
	return published, acceptance, decision, fingerprint
}

// authForGeneration builds one union authorization for an open row at
// an arbitrary ledger position behind a journaled acceptance.
func authForGeneration(t *testing.T, m factory.Merge, acceptance factory.DeliveryAcceptance, decision factory.DeliveryDecision, fingerprint string, n int, confirmed bool) factory.MergeAuthorization {
	t.Helper()
	auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, confirmed)
	auth.Generation.Generation = n
	auth.Generation.NativeOperationID = factory.MergeGenerationOperationID(m.PublicationID, n)
	auth.Generation.ReviewRound = acceptance.Round
	return auth
}

// recordGenerationIntent stores one generation's undriven intent behind
// its derived operation identity. It re-reads the row so the intent
// lands on the current revision behind the bound authorization.
func recordGenerationIntent(t *testing.T, ctx context.Context, db *store.Store, m factory.Merge) factory.Merge {
	t.Helper()
	current, err := db.MergeByPublication(ctx, m.PublicationID)
	if err != nil {
		t.Fatal(err)
	}
	m = current
	n := 1
	if m.Authorization.Bound() {
		n = m.Authorization.Generation.Generation
	}
	operationID := factory.MergeOperationIDForGeneration(m.PublicationID, n)
	intent := &factory.MergeIntent{
		OperationID: operationID, AuthRevision: "accepted-revision",
		HeadRef: m.HeadRef, BaseRef: m.BaseRef,
		HeadOID: m.HeadOID, BaseOID: m.BaseOID,
		Repository: m.Repository, ActorID: 8, PRNumber: m.PRNumber, PRID: m.PRID, IssueID: m.IssueID,
		PRAuthorID: m.PRAuthorID, ReviewerID: m.ReviewerID,
		NativeRev: 12, NotAfter: time.Now().Add(10 * time.Minute).Unix(), AssessmentRevision: 2, ReviewID: 21,
	}
	m.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: operationID, UpdatedUnix: 1200, Work: intent}
	m.Revision++
	if err := db.UpdateMerge(ctx, m); err != nil {
		t.Fatal(err)
	}
	stored, err := db.MergeByPublication(ctx, m.PublicationID)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

// TestAuthorizeMergeSecondGeneration proves the same-PR two-generation
// flow through the registration seam on PostgreSQL: intent, confirmed
// cancel, withdrawal, then a fresh generation on fresh review and
// judgment. The predecessor snapshots exactly once, the PR identity
// never moves, no receipt is discarded and the fresh generation drives
// its own derived operation identity.
func TestAuthorizeMergeSecondGeneration(t *testing.T) {
	ctx := context.Background()
	fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 61)
	db := fx.publish.db
	exec := happyMerger()
	fx.wire(exec)
	auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
	if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err != nil {
		t.Fatalf("first generation refused: %v", err)
	}
	driven := recordGenerationIntent(t, ctx, db, m)
	if driven.Operation.Work.OperationID != factory.MergeOperationID(p.ID, 1) {
		t.Fatalf("generation one submit identity %q differs from its historic derivation", driven.Operation.Work.OperationID)
	}
	report := &MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.withdrawMerge(ctx, &driven, report)
	if len(report.Errors) != 0 {
		t.Fatalf("withdrawal errored: %+v", report)
	}
	finalized, err := db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Stage != factory.MergeWithdrawn || finalized.Operation.Effect != factory.OpEffectNotCommitted ||
		finalized.Operation.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("predecessor not finalized behind its confirmed cancel: %+v", finalized)
	}
	_, acceptance2, decision2, fingerprint2 := admitSecondAcceptance(t, ctx, db, p, strings.Repeat("5", 32))
	auth2 := authForGeneration(t, finalized, acceptance2, decision2, fingerprint2, 2, true)
	if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth2); err != nil {
		t.Fatalf("fresh generation refused: %v", err)
	}
	replaced, err := db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(replaced.History) != 1 || replaced.Authorization.Generation.Generation != 2 {
		t.Fatalf("replacement misplaced: history=%d current=%+v", len(replaced.History), replaced.Authorization.Generation)
	}
	snapshot := replaced.History[0]
	if snapshot.Authorization.Generation.Generation != 1 ||
		snapshot.Operation.OperationID != finalized.Operation.OperationID ||
		snapshot.Operation.Receipt != finalized.Operation.Receipt ||
		snapshot.Stage != factory.MergeWithdrawn || snapshot.FinishedUnix != finalized.FinishedUnix {
		t.Fatalf("predecessor not snapshotted exactly: %+v", snapshot)
	}
	if replaced.PRNumber != m.PRNumber || replaced.PRID != m.PRID || replaced.IssueID != m.IssueID {
		t.Fatal("replacement moved its PR identity")
	}
	if replaced.HeadOID != m.HeadOID || replaced.Stage != factory.MergeOpen || replaced.Operation.Attempts != 0 {
		t.Fatalf("live fields are not the fresh generation: %+v", replaced)
	}
	fresh, err := db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := mergeAuthorizationGate(replaced, fresh); err != nil {
		t.Fatalf("fresh generation refused its submit gate: %v", err)
	}
	work := fx.publish.coord.mergeWork(replaced, factory.RepositoryPolicy{})
	if work.OperationID != factory.MergeGenerationOperationID(p.ID, 2) {
		t.Fatalf("fresh generation submit identity %q differs from its derivation", work.OperationID)
	}
	if len(exec.submits) != 0 {
		t.Fatalf("registration submitted a native effect: %d", len(exec.submits))
	}
}

// TestAuthorizeMergeSecondGenerationRefusals proves every unjustified
// replacement refuses through the seam with no new native effect: a
// pending or committed predecessor, a stale generation number, another
// candidate and unconfirmed resolutions.
func TestAuthorizeMergeSecondGenerationRefusals(t *testing.T) {
	ctx := context.Background()
	t.Run("pending predecessor", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 62)
		exec := happyMerger()
		fx.wire(exec)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, unionAuthForBind(t, m, acceptance, decision, fingerprint, true)); err != nil {
			t.Fatal(err)
		}
		driven := recordGenerationIntent(t, ctx, fx.publish.db, m)
		_ = driven
		_, acceptance2, decision2, fingerprint2 := admitSecondAcceptance(t, ctx, fx.publish.db, p, strings.Repeat("5", 32))
		stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, authForGeneration(t, stored, acceptance2, decision2, fingerprint2, 2, true)); err == nil {
			t.Fatal("pending predecessor authorized replacement")
		}
		if got, _ := fx.publish.db.MergeByPublication(ctx, p.ID); len(got.History) != 0 || got.Authorization.Generation.Generation != 1 {
			t.Fatal("refused replacement changed its ledger")
		}
		if len(exec.submits) != 0 || len(exec.cancels) != 0 {
			t.Fatal("refused replacement touched a native effect")
		}
	})
	t.Run("committed predecessor", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 63)
		exec := happyMerger()
		fx.wire(exec)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, unionAuthForBind(t, m, acceptance, decision, fingerprint, true)); err != nil {
			t.Fatal(err)
		}
		driven := recordGenerationIntent(t, ctx, fx.publish.db, m)
		driven.Operation.Effect, driven.Operation.Cancellation = factory.OpEffectCommitted, factory.OpCancelNone
		driven.Operation.Completion = factory.OpCompletionComplete
		driven.Operation.MergedCommit = driven.HeadOID
		driven.Operation.PRNumber, driven.Operation.PRID, driven.Operation.IssueID = driven.PRNumber, driven.PRID, driven.IssueID
		driven.Operation.HeadRef, driven.Operation.BaseRef = driven.HeadRef, driven.BaseRef
		driven.Operation.HeadOID, driven.Operation.BaseOID = driven.HeadOID, driven.BaseOID
		driven.Revision++
		if err := fx.publish.db.UpdateMerge(ctx, driven); err != nil {
			t.Fatal("record commit:", err)
		}
		_, acceptance2, decision2, fingerprint2 := admitSecondAcceptance(t, ctx, fx.publish.db, p, strings.Repeat("5", 32))
		stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, authForGeneration(t, stored, acceptance2, decision2, fingerprint2, 2, true)); err == nil {
			t.Fatal("committed predecessor authorized replacement")
		} else if err.Error() != "committed merge forbids replacement" {
			t.Fatalf("committed predecessor gave %v, want an honest committed refusal", err)
		}
		if got, _ := fx.publish.db.MergeByPublication(ctx, p.ID); len(got.History) != 0 || got.Operation.Effect != factory.OpEffectCommitted {
			t.Fatal("refused replacement touched its committed effect")
		}
		if len(exec.submits) != 0 || len(exec.cancels) != 0 {
			t.Fatal("refused replacement touched a native effect")
		}
	})
	t.Run("stale generation and candidate", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 64)
		exec := happyMerger()
		fx.wire(exec)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, unionAuthForBind(t, m, acceptance, decision, fingerprint, true)); err != nil {
			t.Fatal(err)
		}
		driven := recordGenerationIntent(t, ctx, fx.publish.db, m)
		report := &MergeReport{Merged: []MergeLink{}}
		fx.publish.coord.withdrawMerge(ctx, &driven, report)
		_, acceptance2, decision2, fingerprint2 := admitSecondAcceptance(t, ctx, fx.publish.db, p, strings.Repeat("5", 32))
		stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		before := len(exec.cancels)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, authForGeneration(t, stored, acceptance2, decision2, fingerprint2, 3, true)); err == nil {
			t.Fatal("generation gap authorized replacement")
		}
		moved := authForGeneration(t, stored, acceptance2, decision2, fingerprint2, 2, true)
		moved.Generation.Head = strings.Repeat("4", 40)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, moved); err == nil {
			t.Fatal("another candidate authorized replacement")
		}
		if got, _ := fx.publish.db.MergeByPublication(ctx, p.ID); len(got.History) != 0 {
			t.Fatal("refused replacement changed its ledger")
		}
		if len(exec.submits) != 0 || len(exec.cancels) != before {
			t.Fatal("refused replacement touched a native effect")
		}
	})
	t.Run("unconfirmed resolutions", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 65)
		exec := happyMerger()
		fx.wire(exec)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, unionAuthForBind(t, m, acceptance, decision, fingerprint, true)); err != nil {
			t.Fatal(err)
		}
		driven := recordGenerationIntent(t, ctx, fx.publish.db, m)
		report := &MergeReport{Merged: []MergeLink{}}
		fx.publish.coord.withdrawMerge(ctx, &driven, report)
		_, acceptance2, decision2, fingerprint2 := admitSecondAcceptance(t, ctx, fx.publish.db, p, strings.Repeat("5", 32))
		stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		before := len(exec.cancels)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, authForGeneration(t, stored, acceptance2, decision2, fingerprint2, 2, false)); err == nil {
			t.Fatal("unconfirmed resolutions authorized replacement")
		}
		if got, _ := fx.publish.db.MergeByPublication(ctx, p.ID); len(got.History) != 0 {
			t.Fatal("refused replacement changed its ledger")
		}
		if len(exec.submits) != 0 || len(exec.cancels) != before {
			t.Fatal("refused replacement touched a native effect")
		}
	})
}

// TestUpdateMergeRechecksUnionBindingInTx proves the registration
// transaction re-verifies the journaled D07 acceptance independently of
// the seam: a union binding for unjournaled acceptance refuses on
// commit, while the journaled binding commits through the same path.
func TestUpdateMergeRechecksUnionBindingInTx(t *testing.T) {
	ctx := context.Background()
	fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 66)
	db := fx.publish.db
	stray := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
	stray.Generation.AcceptanceID = strings.Repeat("7", 32)
	m.Revision++
	bound := m
	bound.Authorization = stray
	if err := db.UpdateMerge(ctx, bound); err == nil {
		t.Fatal("unjournaled union binding committed")
	}
	if got, _ := db.MergeByPublication(ctx, p.ID); got.Authorization.Bound() {
		t.Fatal("refused binding persisted")
	}
	bound.Authorization = unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
	if err := db.UpdateMerge(ctx, bound); err != nil {
		t.Fatalf("journaled union binding refused in its transaction: %v", err)
	}
}

// TestCommittedMergeRecoversDespiteWithdrawal proves a recorded
// committed merge still confirms after a later invalidation — here a
// dispatch withdrawal closing every new submit. Recovery reuses the
// exact recorded operation by lookup: one lookup, no resubmit, no new
// judgment, and no replacement generation.
func TestCommittedMergeRecoversDespiteWithdrawal(t *testing.T) {
	ctx := context.Background()
	fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 67)
	db := fx.publish.db
	exec := happyMerger()
	fx.wire(exec)
	if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, unionAuthForBind(t, m, acceptance, decision, fingerprint, true)); err != nil {
		t.Fatal(err)
	}
	driven := recordGenerationIntent(t, ctx, db, m)
	operationID := driven.Operation.OperationID
	exec.ledger[operationID] = factory.OperationOutcome{
		OperationID: operationID, InstallationID: "test-installation", Kind: factory.OpMerge,
		ActorID: 8, RepositoryID: driven.Repository, Effect: factory.OpEffectCommitted,
		Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Reason: "merged",
		Receipt: []byte(`{"pr_number":9}`),
	}
	if _, err := db.WithdrawDispatch(ctx, driven.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	report := &MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.reconcileMerge(ctx, driven, report)
	if len(report.Errors) != 0 {
		t.Fatalf("recovery errored: %+v", report)
	}
	recovered, err := db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Stage != factory.MergeMerged || recovered.MergedCommit != recovered.HeadOID {
		t.Fatalf("committed merge not recovered: %+v", recovered)
	}
	if recovered.Operation.OperationID != operationID || len(recovered.History) != 0 {
		t.Fatalf("recovery replaced its exact operation: %+v", recovered.Operation)
	}
	if len(exec.submits) != 0 {
		t.Fatalf("recovery resubmitted: %d", len(exec.submits))
	}
	// Recovery reuses the exact recorded operation by identity: the
	// lookup path, the cancellation check, or both — never another
	// identity and never a resubmission.
	seen := map[string]bool{}
	for _, id := range append(append([]string{}, exec.lookups...), exec.cancels...) {
		if id != operationID {
			t.Fatalf("recovery touched another operation: %+v", append(exec.lookups, exec.cancels...))
		}
		seen[id] = true
	}
	if !seen[operationID] {
		t.Fatalf("recovery bypassed its exact operation: lookups=%+v cancels=%+v", exec.lookups, exec.cancels)
	}
	if len(report.Merged) != 1 || report.Merged[0].MergeID != recovered.ID {
		t.Fatalf("recovery unlinked: %+v", report)
	}
}
