package factory

import (
	"strings"
	"testing"
)

// historyTestOperation builds one driven merge operation with its
// terminal effect for snapshot tests.
func historyTestOperation(publicationID, operationID, effect, cancellation, reason string) MergeOperation {
	return MergeOperation{
		Work: testMergeIntent(operationID),
		Kind: OpMerge, Attempts: 1, OperationID: operationID,
		Effect: effect, Cancellation: cancellation, Reason: reason,
		Receipt: `{"pr_number":9}`, UpdatedUnix: 1200,
		InstallationID: "8e81334d-94c3-4b96-8f34-19f2160ff4f8", ActorID: 8, RepositoryID: 7,
	}
}

// historyTestAuth binds one valid non-union generation authorization for
// the test row's exact head and base.
func historyTestAuth(publicationID string, n int, head string) MergeAuthorization {
	generation := testMergeGeneration(publicationID, n, head)
	return MergeAuthorization{
		Generation: generation, Decision: testReadyDecision(generation.Fingerprint),
		ResolutionsConfirmed: true,
	}
}

func TestMergeOperationIDForGeneration(t *testing.T) {
	publicationID := strings.Repeat("f", 32)
	if got, want := MergeOperationIDForGeneration(publicationID, 1), MergeOperationID(publicationID, 1); got != want {
		t.Fatalf("generation one identity %q differs from its historic derivation %q", got, want)
	}
	if got, want := MergeOperationIDForGeneration(publicationID, 0), MergeOperationID(publicationID, 1); got != want {
		t.Fatalf("unbound identity %q differs from its historic derivation %q", got, want)
	}
	for _, n := range []int{2, 3, 32} {
		if got, want := MergeOperationIDForGeneration(publicationID, n), MergeGenerationOperationID(publicationID, n); got != want {
			t.Fatalf("generation %d identity %q differs from its derivation %q", n, got, want)
		}
	}
	if MergeOperationIDForGeneration(publicationID, 2) == MergeOperationIDForGeneration(publicationID, 1) {
		t.Fatal("replacement generation reuses its finalized identity")
	}
}

func TestMergePredecessorProofOf(t *testing.T) {
	newOp := func() MergeOperation {
		return historyTestOperation(strings.Repeat("f", 32), "soda-x-merge-1", OpEffectNotCommitted, OpCancelCancelled, "cancelled_before_admission")
	}
	if _, err := MergePredecessorProofOf(1, newOp()); err != nil {
		t.Fatalf("confirmed cancel refused: %v", err)
	}
	refused := newOp()
	refused.Cancellation, refused.Reason = OpCancelNone, "native_refused"
	if _, err := MergePredecessorProofOf(1, refused); err != nil {
		t.Fatalf("deterministic refusal refused: %v", err)
	}
	idle := newOp()
	idle.Cancellation, idle.Reason = "", "native_refused"
	if _, err := MergePredecessorProofOf(1, idle); err != nil {
		t.Fatalf("idle-marked refusal refused: %v", err)
	}
	cases := map[string]func(*MergeOperation){
		"pending":     func(o *MergeOperation) { o.Effect, o.Cancellation, o.Reason = OpEffectPending, OpCancelNone, "" },
		"uncertain":   func(o *MergeOperation) { o.Effect, o.Cancellation = OpEffectIndeterminate, OpCancelIndeterminate },
		"unobserved":  func(o *MergeOperation) { o.Effect, o.Cancellation, o.Reason = "", "", "" },
		"reasonless":  func(o *MergeOperation) { o.Effect, o.Cancellation, o.Reason = OpEffectNotCommitted, OpCancelNone, "" },
		"committed":   func(o *MergeOperation) { o.Effect, o.Cancellation = OpEffectCommitted, OpCancelNone },
		"late-cancel": func(o *MergeOperation) { o.Effect, o.Cancellation = OpEffectCommitted, OpCancelTooLate },
		"pristine":    func(o *MergeOperation) { *o = MergeOperation{Kind: OpMerge} },
	}
	for name, mutate := range cases {
		op := newOp()
		mutate(&op)
		if _, err := MergePredecessorProofOf(1, op); err == nil {
			t.Errorf("case %s proved replaceability", name)
		}
	}
	if _, err := MergePredecessorProofOf(0, newOp()); err == nil {
		t.Error("proof for no generation accepted")
	}
}

func TestMergeGenerationSnapshotValidation(t *testing.T) {
	publicationID := strings.Repeat("f", 32)
	auth := historyTestAuth(publicationID, 1, strings.Repeat("2", 40))
	op := historyTestOperation(publicationID, "soda-x-merge-1", OpEffectNotCommitted, OpCancelCancelled, "cancelled_before_admission")
	snapshot := SnapshotMergeGeneration(auth, op, MergeWithdrawn, Cancelled, MergeReasonWithdrawn, 1300)
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("finalized snapshot refused: %v", err)
	}
	refusedOp := historyTestOperation(publicationID, "soda-x-merge-1", OpEffectNotCommitted, OpCancelNone, "native_refused")
	refused := SnapshotMergeGeneration(auth, refusedOp, MergeFailed, Failed, MergeReasonRefused, 1300)
	if err := refused.Validate(); err != nil {
		t.Fatalf("refused snapshot refused: %v", err)
	}
	cases := map[string]func(*MergeGenerationSnapshot){
		"unbound": func(s *MergeGenerationSnapshot) { s.Authorization = MergeAuthorization{} },
		"open":    func(s *MergeGenerationSnapshot) { s.Stage, s.Outcome, s.Reason, s.FinishedUnix = MergeOpen, "", "", 0 },
		"fenced": func(s *MergeGenerationSnapshot) {
			s.Stage, s.Outcome, s.Reason = MergeFenced, NeedsHuman, MergeReasonFenced
		},
		"merged": func(s *MergeGenerationSnapshot) {
			s.Stage, s.Outcome, s.Reason = MergeMerged, Succeeded, MergeReasonMerged
		},
		"outcome":  func(s *MergeGenerationSnapshot) { s.Outcome = Failed },
		"reason":   func(s *MergeGenerationSnapshot) { s.Reason = MergeReasonRefused },
		"unproven": func(s *MergeGenerationSnapshot) { s.Operation.Effect = OpEffectPending },
	}
	for name, mutate := range cases {
		next := snapshot
		mutate(&next)
		if err := next.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func TestMergeRowLedgerPosition(t *testing.T) {
	publicationID := strings.Repeat("f", 32)
	newRow := func() Merge {
		m := testMerge()
		m.PublicationID = publicationID
		m.HeadOID = strings.Repeat("2", 40)
		return m
	}
	m := newRow()
	m.Authorization = historyTestAuth(publicationID, 1, m.HeadOID)
	if err := m.Validate(); err != nil {
		t.Fatalf("first-generation row refused: %v", err)
	}
	// A second generation without its predecessor history refuses: the
	// binding generation is always history length plus one.
	m.Authorization = historyTestAuth(publicationID, 2, m.HeadOID)
	if err := m.Validate(); err == nil {
		t.Fatal("second generation without history accepted")
	}
	// The exact finalized predecessor snapshots once; the live fields
	// become generation two.
	op := historyTestOperation(publicationID, "soda-x-merge-1", OpEffectNotCommitted, OpCancelCancelled, "cancelled_before_admission")
	predecessor := SnapshotMergeGeneration(historyTestAuth(publicationID, 1, m.HeadOID), op, MergeWithdrawn, Cancelled, MergeReasonWithdrawn, 1300)
	replaced := newRow()
	replaced.History = []MergeGenerationSnapshot{predecessor}
	replaced.Authorization = historyTestAuth(publicationID, 2, replaced.HeadOID)
	if err := replaced.Validate(); err != nil {
		t.Fatalf("replacement row refused: %v", err)
	}
	ledger, err := replaced.Ledger()
	if err != nil {
		t.Fatalf("replacement ledger refused: %v", err)
	}
	if len(ledger.History) != 1 || ledger.Current.Generation != 2 {
		t.Fatalf("replacement ledger misplaced: %+v", ledger)
	}
	// Prefix immutability (no rewrite, no deletion) is the store's
	// compare-and-swap comparison, not a single-row shape check: a
	// rewritten but well-formed snapshot still validates alone.
	cases := map[string]func(*Merge){
		"drop": func(m *Merge) { m.History = nil },
		"gap": func(m *Merge) {
			m.History = append(m.History, SnapshotMergeGeneration(
				historyTestAuth(publicationID, 3, strings.Repeat("4", 40)),
				historyTestOperation(publicationID, "soda-x-merge-3", OpEffectNotCommitted, OpCancelCancelled, "cancelled_before_admission"),
				MergeWithdrawn, Cancelled, MergeReasonWithdrawn, 1400))
		},
		"foreign": func(m *Merge) { m.History[0].Authorization.Generation.PublicationID = strings.Repeat("9", 32) },
	}
	for name, mutate := range cases {
		next := replaced
		next.History = append([]MergeGenerationSnapshot(nil), replaced.History...)
		mutate(&next)
		if err := next.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func TestMergeRowLedgerCeiling(t *testing.T) {
	publicationID := strings.Repeat("f", 32)
	m := testMerge()
	m.PublicationID = publicationID
	m.HeadOID = strings.Repeat("2", 40)
	for n := 1; n <= MaxMergeGenerations; n++ {
		head := strings.Repeat("2", 39) + string(rune('0'+n%10))
		if n == 1 {
			head = strings.Repeat("2", 40)
		}
		auth := historyTestAuth(publicationID, n, head)
		op := historyTestOperation(publicationID, "soda-x-merge-1", OpEffectNotCommitted, OpCancelCancelled, "cancelled_before_admission")
		m.History = append(m.History, SnapshotMergeGeneration(auth, op, MergeWithdrawn, Cancelled, MergeReasonWithdrawn, 1300))
	}
	m.Authorization = MergeAuthorization{}
	if err := m.Validate(); err != nil {
		t.Fatalf("ceiling history refused: %v", err)
	}
	// One more generation above the ceiling is a visible intervention,
	// never a silent drop.
	m.Authorization = historyTestAuth(publicationID, MaxMergeGenerations+1, strings.Repeat("8", 40))
	if err := m.Validate(); err == nil {
		t.Fatal("generation above the history ceiling accepted")
	}
}
