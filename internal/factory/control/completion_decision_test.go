package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func decidingCycle() factory.ReassessmentCycle {
	return factory.ReassessmentCycle{
		EventID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
		State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
	}
}

func decisionAnswer(selected string, confidence float64) factory.PriorityAnswer {
	return factory.PriorityAnswer{Selected: selected, Confidence: confidence}
}

// Every completion gets its own funded decision, including an empty
// inventory: valid confidence records, anything else holds.
func TestApplyReassessmentAnswerRecordsValidContinue(t *testing.T) {
	updated, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionContinue, 0.9))
	if err != nil || updated.Decision != factory.ReassessmentOptionContinue {
		t.Fatalf("updated=%+v err=%v, want a recorded continue", updated, err)
	}
}

func TestApplyReassessmentAnswerRecordsBoundaryConfidence(t *testing.T) {
	updated, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionReassess, 0.50))
	if err != nil || updated.Decision != factory.ReassessmentOptionReassess {
		t.Fatalf("updated=%+v err=%v, want a recorded reassess at the bound", updated, err)
	}
}

func TestApplyReassessmentAnswerRefusesLowConfidence(t *testing.T) {
	if _, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionContinue, 0.49)); err == nil {
		t.Fatal("low-confidence continue recorded instead of holding")
	}
	if _, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionReassess, 0.0)); err == nil {
		t.Fatal("zero-confidence reassess recorded instead of holding")
	}
}

func TestApplyReassessmentAnswerRefusesUnknownOption(t *testing.T) {
	if _, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer("maybe", 0.95)); err == nil {
		t.Fatal("unknown option recorded instead of holding")
	}
}

func TestApplyReassessmentAnswerRefusesOutsideDeciding(t *testing.T) {
	cycle := decidingCycle()
	cycle.State = factory.ReassessmentCollecting
	if _, err := applyReassessmentAnswer(cycle, decisionAnswer(factory.ReassessmentOptionContinue, 0.9)); err == nil {
		t.Fatal("decision recorded outside its deciding state")
	}
}

func TestReassessmentDecisionOptionsAreExactlyTwo(t *testing.T) {
	options := reassessmentDecisionOptions()
	if len(options) != 2 {
		t.Fatalf("options=%v, want exactly reassess and continue", options)
	}
	if _, ok := options[factory.ReassessmentOptionReassess]; !ok {
		t.Fatal("reassess option missing")
	}
	if _, ok := options[factory.ReassessmentOptionContinue]; !ok {
		t.Fatal("continue option missing")
	}
}

// sealedDecisionRecord seals one genuine single-issue inventory through
// the real collector, so the fixture carries the exact digest the
// decision refold verifies — never a placeholder hash.
func sealedDecisionRecord() store.ReassessmentRecord {
	title, body := "sealed title", "sealed body with substance"
	evidence, err := factory.CompletionIssueEvidence{
		Index: 4, Title: title, Body: body, Dependencies: []string{"2"},
		TitleDigest: issueDigest(title), ContentDigest: issueDigest(body),
	}.CanonicalData()
	if err != nil {
		panic(err)
	}
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"": {Issues: []NativeInventoryIssue{{Index: 4, Data: evidence}}, Complete: true},
		},
	}
	sealed, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
	if err != nil {
		panic(err)
	}
	cycle := factory.ReassessmentCycle{
		EventID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
		State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
	}
	return store.ReassessmentRecord{
		Cycle: cycle,
		Completion: factory.CompletionEvent{
			PublicationID: strings.Repeat("5", 32), MergedCommit: strings.Repeat("c", 40),
			ClosingOccurrence: "501", AcceptanceID: "d" + strings.Repeat("a", 24),
			ID: strings.Repeat("1", 32), Repository: 7, Issue: 3, Generation: 1,
		},
		Local:               sealed.Local,
		Evidence:            sealed.Bodies,
		InventoryDigest:     sealed.Digest,
		EvidenceFingerprint: SealEvidenceFingerprint(sealed.Digest, cycle.EventID, cycle.CycleID),
		Issue:               3, EvidenceIssues: sealed.Issues, EvidenceRevision: sealed.Revision,
	}
}

// The decision input carries the sealed substance: the completed
// change, the local bindings and every issue body, within the
// pre-spend bound.
func TestDecisionStateFromRecordCarriesSubstance(t *testing.T) {
	state, err := DecisionStateFromRecord(sealedDecisionRecord())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if state.CompletionID != strings.Repeat("1", 32) || state.Issues != 1 {
		t.Fatalf("state=%+v, want the sealed cycle identity and count", state)
	}
	if state.Completion.MergedCommit != strings.Repeat("c", 40) || state.Completion.Issue != 3 {
		t.Fatalf("completion=%+v, want the sealed completed change", state.Completion)
	}
	if !state.Local.Sealed() {
		t.Fatalf("local=%+v, want the sealed local bindings", state.Local)
	}
	if len(state.Evidence) != 1 || state.Evidence[0].Title != "sealed title" ||
		state.Evidence[0].Body != "sealed body with substance" || len(state.Evidence[0].Dependencies) != 1 {
		t.Fatalf("evidence=%+v, want the sealed issue substance", state.Evidence)
	}
}

func TestDecisionStateFromRecordRefusesUnsealed(t *testing.T) {
	record := sealedDecisionRecord()
	record.InventoryDigest = ""
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("unsubstantial state built from an unsealed record")
	}
	record = sealedDecisionRecord()
	record.Evidence = nil
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("hash-only state built from a bodiless record")
	}
	record = sealedDecisionRecord()
	record.EvidenceIssues = 2
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built from bodies outside their sealed count")
	}
}

// A substituted digest refuses even when it is well-formed: the
// refold must reproduce exactly the sealed digest.
func TestDecisionStateFromRecordRefusesTamperedDigest(t *testing.T) {
	record := sealedDecisionRecord()
	record.InventoryDigest = strings.Repeat("f", 64)
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built under a substituted inventory digest")
	}
}

// A shape-valid body edit refuses: the refold binds substance, not
// just shape, so a mutation under an unchanged digest never decides.
func TestDecisionStateFromRecordRefusesTamperedBody(t *testing.T) {
	record := sealedDecisionRecord()
	record.Evidence[0].Body = "edited body after the seal"
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built from bodies edited after the seal")
	}
}

func TestDecisionStateFromRecordRefusesInvalidBody(t *testing.T) {
	record := sealedDecisionRecord()
	record.Evidence[0].TitleDigest = "not-a-digest"
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built from a body outside its sealed validation")
	}
}

// A missing local refuses with zero spend: no reservation is taken and
// no provider is consulted before this refusal. The pre-077
// digest-only shape (valid four bindings, nil material body) refuses
// the same way: digests without the body never fund a decision.
func TestDecisionStateFromRecordRefusesMissingMaterial(t *testing.T) {
	record := sealedDecisionRecord()
	record.Local = factory.CompletionLocal{}
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built without its sealed material")
	}
	record = sealedDecisionRecord()
	record.Local = digestOnlyLocal()
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("digest-only state built without its material body")
	}
}

// A reordered persisted sequence refuses even under an unchanged
// digest: the refold preserves the persisted order and never sorts a
// malformed sequence into admission. Same for duplicate indexes.
func TestDecisionStateFromRecordRefusesReorderedPersistedBodies(t *testing.T) {
	seal := func() store.ReassessmentRecord {
		t.Helper()
		var issues []NativeInventoryIssue
		for _, index := range []int64{1, 2} {
			title := "title"
			body := "body"
			evidence, err := factory.CompletionIssueEvidence{
				Index: index, Title: title, Body: body,
				TitleDigest: issueDigest(title), ContentDigest: issueDigest(body),
			}.CanonicalData()
			if err != nil {
				t.Fatal(err)
			}
			issues = append(issues, NativeInventoryIssue{Index: index, Data: evidence})
		}
		enum := &fixtureEnumerator{
			revision: 41, idle: true,
			pages: map[string]NativeInventoryPage{
				"": {Issues: issues, Complete: true},
			},
		}
		sealed, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
		if err != nil {
			t.Fatal(err)
		}
		cycle := factory.ReassessmentCycle{
			EventID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
			State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
		}
		return store.ReassessmentRecord{
			Cycle: cycle,
			Completion: factory.CompletionEvent{
				PublicationID: strings.Repeat("5", 32), MergedCommit: strings.Repeat("c", 40),
				ClosingOccurrence: "501", AcceptanceID: "d" + strings.Repeat("a", 24),
				ID: strings.Repeat("1", 32), Repository: 7, Issue: 3, Generation: 1,
			},
			Local:               sealed.Local,
			Evidence:            sealed.Bodies,
			InventoryDigest:     sealed.Digest,
			EvidenceFingerprint: SealEvidenceFingerprint(sealed.Digest, cycle.EventID, cycle.CycleID),
			Issue:               3, EvidenceIssues: sealed.Issues, EvidenceRevision: sealed.Revision,
		}
	}
	record := seal()
	if _, err := DecisionStateFromRecord(record); err != nil {
		t.Fatalf("valid two-body seal precondition refused: %v", err)
	}
	reversed := seal()
	reversed.Evidence[0], reversed.Evidence[1] = reversed.Evidence[1], reversed.Evidence[0]
	if _, err := DecisionStateFromRecord(reversed); err == nil {
		t.Fatal("state built from a reordered persisted sequence under an unchanged digest")
	}
	duplicated := seal()
	duplicated.Evidence[1] = duplicated.Evidence[0]
	if _, err := DecisionStateFromRecord(duplicated); err == nil {
		t.Fatal("state built from a duplicate-index persisted sequence")
	}
}

func TestDecisionStateFromRecordRefusesCompletionCycleMismatch(t *testing.T) {
	record := sealedDecisionRecord()
	record.Completion.Repository = 8
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built from a completion outside its cycle repository")
	}
	record = sealedDecisionRecord()
	record.Completion.Generation = 2
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built from a completion outside its cycle generation")
	}
}

// A complete empty inventory decides: an explicitly sealed empty set
// is distinguishable from missing evidence, which refuses above.
func TestDecisionStateAcceptsCompleteEmpty(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages:    map[string]NativeInventoryPage{"": {Complete: true}},
	}
	sealed, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("empty collect: %v", err)
	}
	cycle := decidingCycle()
	record := store.ReassessmentRecord{
		Cycle: cycle,
		Completion: factory.CompletionEvent{
			PublicationID: strings.Repeat("5", 32), MergedCommit: strings.Repeat("c", 40),
			ClosingOccurrence: "501", AcceptanceID: "d" + strings.Repeat("a", 24),
			ID: cycle.EventID, Repository: 7, Issue: 3, Generation: 1,
		},
		Local:               sealed.Local,
		Evidence:            sealed.Bodies,
		InventoryDigest:     sealed.Digest,
		EvidenceFingerprint: SealEvidenceFingerprint(sealed.Digest, cycle.EventID, cycle.CycleID),
		Issue:               3, EvidenceIssues: 0, EvidenceRevision: sealed.Revision,
	}
	state, err := DecisionStateFromRecord(record)
	if err != nil {
		t.Fatalf("complete empty state refused: %v", err)
	}
	if state.Issues != 0 || len(state.Evidence) != 0 {
		t.Fatalf("state=%+v, want the explicitly empty set", state)
	}
}

// decisionTestCoordinator wires only identity: every refusal below
// returns before any reservation or provider call, so the empty store
// is never touched — reaching it would fail loudly instead of spending.
func decisionTestCoordinator() *Coordinator {
	coord := &Coordinator{Store: &store.Store{}}
	coord.Planning = &PlanningJudgments{
		Journal: coord.Store, Funding: coord.Store,
		Chooser: &assumptionChooser{selected: factory.ReassessmentOptionContinue, confidence: 0.9},
	}
	return coord
}

func TestDecideReassessmentRefusesMismatchedMaterial(t *testing.T) {
	record := sealedDecisionRecord()
	state, err := DecisionStateFromRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	coord := decisionTestCoordinator()
	edited := state
	edited.Local.AuthorityFingerprint = "changed-after-the-seal"
	if _, err := coord.DecideReassessment(context.Background(), record, edited); err == nil ||
		!strings.Contains(err.Error(), "material does not match") {
		t.Fatalf("err=%v, want the material mismatch refusal before spend", err)
	}
	edited = state
	edited.EvidenceFingerprint = strings.Repeat("f", 64)
	if _, err := coord.DecideReassessment(context.Background(), record, edited); err == nil ||
		!strings.Contains(err.Error(), "evidence does not match") {
		t.Fatalf("err=%v, want the evidence mismatch refusal before spend", err)
	}
	edited = state
	edited.Local = factory.CompletionLocal{}
	if _, err := coord.DecideReassessment(context.Background(), record, edited); err == nil ||
		!strings.Contains(err.Error(), "sealed material") {
		t.Fatalf("err=%v, want the missing material refusal before spend", err)
	}
	edited = state
	edited.CompletionID = strings.Repeat("9", 32)
	if _, err := coord.DecideReassessment(context.Background(), record, edited); err == nil ||
		!strings.Contains(err.Error(), "does not match its cycle") {
		t.Fatalf("err=%v, want the cycle mismatch refusal before spend", err)
	}
}

func TestDecideReassessmentRefusesOutsideDeciding(t *testing.T) {
	record := sealedDecisionRecord()
	record.Cycle.State = factory.ReassessmentCollecting
	state, err := DecisionStateFromRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decisionTestCoordinator().DecideReassessment(context.Background(), record, state); err == nil {
		t.Fatal("decision taken outside its deciding state")
	}
}

func TestDecisionFingerprintBindsBodies(t *testing.T) {
	state, err := DecisionStateFromRecord(sealedDecisionRecord())
	if err != nil {
		t.Fatal(err)
	}
	edited := state
	edited.Evidence = append([]factory.CompletionIssueEvidence(nil), state.Evidence...)
	edited.Evidence[0].Body = "edited body after the seal"
	if fingerprintOf(edited) == fingerprintOf(state) {
		t.Fatal("edited bodies share the sealed decision cache entry")
	}
}

func TestDecisionFingerprintIsStableAndPerCompletion(t *testing.T) {
	state := ReassessmentDecisionState{
		CompletionID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
		InventoryDigest: strings.Repeat("a", 64), EvidenceFingerprint: strings.Repeat("b", 64),
		Repository: 7, Generation: 1, Issues: 3,
	}
	first, second := fingerprintOf(state), fingerprintOf(state)
	if first == "" || first != second || !factory.ValidDigest(first) {
		t.Fatal("decision fingerprint unstable or malformed")
	}
	other := state
	other.CompletionID = strings.Repeat("3", 32)
	other.CycleID = strings.Repeat("4", 32)
	if fingerprintOf(other) == first {
		t.Fatal("two completions share one decision cache entry")
	}
}
