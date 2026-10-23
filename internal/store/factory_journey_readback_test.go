package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// journeySQL opens the ephemeral database for direct fixture inserts;
// the Store owns reads, raw SQL owns only test seeding.
func journeySQL(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// journeyCycleRecord builds a minimal valid reassessment record naming
// one issue at one generation in the given state.
func journeyCycleRecord(cycle, event string, repository, issue, generation int64, state string) ReassessmentRecord {
	return ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: event, CycleID: cycle,
			State: state, Repository: repository, Generation: generation,
		},
		Issue: issue,
	}
}

// insertJourneyCompletionParent satisfies the cycle table's
// (repository,generation) foreign key; the parent carries no journey
// semantics beyond ordering the fence.
func insertJourneyCompletionParent(t *testing.T, db *sql.DB, repository, generation int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(repository,generation) DO NOTHING`,
		repository, generation, strings.Repeat("e", 32),
		fmt.Sprintf("%032x", repository*1000003+generation),
		fmt.Sprintf("occ-%d-%d", repository, generation), `{}`); err != nil {
		t.Fatal(err)
	}
}

func insertJourneyCycle(t *testing.T, db *sql.DB, record ReassessmentRecord) {
	t.Helper()
	insertJourneyCompletionParent(t, db, record.Cycle.Repository, record.Cycle.Generation)
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,$2,$3,$4,$5,$6)`,
		record.Cycle.CycleID, record.Cycle.Repository, record.Cycle.Generation, record.Cycle.EventID, record.Cycle.State, string(raw)); err != nil {
		t.Fatal(err)
	}
}

// insertJourneyAssignment stores one assignment row with routing columns
// mirroring its JSON, matching production FinishAssignment.
func insertJourneyAssignment(t *testing.T, db *sql.DB, a factory.Assignment, stage string, revision int64) {
	t.Helper()
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, stage, revision, string(raw)); err != nil {
		t.Fatal(err)
	}
}

func journeyAssignment(id, root, acceptance string, repository, issue int64, owner bool) factory.Assignment {
	publication := id
	parent := ""
	if !owner {
		publication = root
		parent = root
	}
	return factory.Assignment{
		ID: id, AttemptRoot: root, ParentAssignment: parent, PublicationAssignment: publication,
		ProjectID: "p" + strings.Repeat("1", 24), Role: "coder", Acceptance: acceptance,
		Preparation: "f" + strings.Repeat("b", 24),
		Run:         "run-" + id[:8], RunHistory: []string{"run-" + id[:8]},
		Repository: repository, Issue: issue, CreatedUnix: 1700000000,
	}
}

func TestJourneyReassignmentCyclesScopesPerIssue(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	// Other-issue rows land before the target rows: seq and generation
	// order must not leak them into the target read.
	insertJourneyCycle(t, raw, journeyCycleRecord(strings.Repeat("1", 32), strings.Repeat("a", 32), 7, 888, 1, factory.ReassessmentCollecting))
	insertJourneyCycle(t, raw, journeyCycleRecord(strings.Repeat("2", 32), strings.Repeat("b", 32), 8, 3, 1, factory.ReassessmentCollecting))
	first := journeyCycleRecord(strings.Repeat("3", 32), strings.Repeat("c", 32), 7, 3, 2, factory.ReassessmentCollecting)
	second := journeyCycleRecord(strings.Repeat("4", 32), strings.Repeat("d", 32), 7, 3, 3, factory.ReassessmentAttention)
	insertJourneyCycle(t, raw, first)
	insertJourneyCycle(t, raw, second)
	// The read includes terminal decision history without calling it work.
	insertJourneyCycle(t, raw, journeyCycleRecord(strings.Repeat("5", 32), strings.Repeat("e", 32), 7, 3, 4, factory.ReassessmentReleased))
	insertJourneyCycle(t, raw, journeyCycleRecord(strings.Repeat("6", 32), strings.Repeat("f", 32), 7, 3, 5, factory.ReassessmentSuperseded))

	records, truncated, err := s.JourneyReassignmentCycles(ctx, 7, 3, 8)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Fatalf("bounded read reported truncation: %+v", records)
	}
	if len(records) != 4 || records[0].Cycle.CycleID != first.Cycle.CycleID || records[1].Cycle.CycleID != second.Cycle.CycleID ||
		records[2].Cycle.State != factory.ReassessmentReleased || records[3].Cycle.State != factory.ReassessmentSuperseded {
		t.Fatalf("per-issue cycles: %+v", records)
	}
	// The extra record past the render cap is the truncation signal.
	records, truncated, err = s.JourneyReassignmentCycles(ctx, 7, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(records) != 1 || records[0].Cycle.CycleID != second.Cycle.CycleID {
		t.Fatalf("truncation signal: truncated=%v %+v", truncated, records)
	}
	other, _, err := s.JourneyReassignmentCycles(ctx, 7, 888, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 {
		t.Fatalf("other-issue cycles: %+v", other)
	}
	for _, limit := range []int{0, -1, 257} {
		if _, _, err := s.JourneyReassignmentCycles(ctx, 7, 3, limit); err == nil {
			t.Fatalf("listing accepted limit %d", limit)
		}
	}
	if _, _, err := s.JourneyReassignmentCycles(ctx, 0, 3, 8); err == nil {
		t.Fatal("listing accepted empty scope")
	}
}

func TestJourneyReassignmentCyclesRefuseCorruptRows(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	// Malformed JSON text cannot land: the JSONB column refuses it at
	// write. A row that decodes but fails record validation is the
	// corrupt case, and it must fail the read explicitly.
	insertJourneyCompletionParent(t, raw, 7, 1)
	if _, err := raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,$2,$3,$4,$5,$6)`,
		strings.Repeat("9", 32), 7, 1, strings.Repeat("8", 32), factory.ReassessmentCollecting, `{"issue": 7}`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.JourneyReassignmentCycles(ctx, 7, 7, 8); err == nil {
		t.Fatal("corrupt row read without an error")
	}
}

func TestLatestPhaseAttemptsSelectedRoot(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	root := strings.Repeat("7", 32)
	sibling := strings.Repeat("b", 32)
	phaseOf := func(id, groupRoot, phase string) factory.Assignment {
		a := journeyAssignment(id, groupRoot, acceptance, 7, 3, id == groupRoot)
		a.Phase = phase
		return a
	}
	// Out-of-scope rows land first: another issue, another acceptance
	// and a sibling root sharing the acceptance must never leak into
	// the selected root's groups.
	other := phaseOf(strings.Repeat("0", 32), strings.Repeat("0", 32), "implement")
	other.Issue = 888
	insertJourneyAssignment(t, raw, other, factory.AssignmentFinished, 1)
	stale := phaseOf(strings.Repeat("1", 32), strings.Repeat("1", 32), "implement")
	stale.Acceptance = "d" + strings.Repeat("9", 24)
	insertJourneyAssignment(t, raw, stale, factory.AssignmentFinished, 1)
	siblingOwner := phaseOf(sibling, sibling, "review")
	insertJourneyAssignment(t, raw, siblingOwner, factory.AssignmentFinished, 1)
	// Three implement attempts on the selected root: the owner first,
	// then an older success, then a newer failure. Newest wins by seq
	// order regardless of the owner/child role.
	owner := phaseOf(root, root, "implement")
	insertJourneyAssignment(t, raw, owner, factory.AssignmentFinished, 1)
	old := phaseOf(strings.Repeat("2", 32), root, "implement")
	insertJourneyAssignment(t, raw, old, factory.AssignmentFinished, 1)
	failed := phaseOf(strings.Repeat("3", 32), root, "implement")
	insertJourneyAssignment(t, raw, failed, factory.AssignmentFinished, 1)
	review := phaseOf(strings.Repeat("4", 32), root, "review")
	insertJourneyAssignment(t, raw, review, factory.AssignmentFinished, 1)

	groups, truncated, err := s.LatestPhaseAttempts(ctx, 7, 3, acceptance, root)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Fatalf("bounded read reported truncation: %+v", groups)
	}
	byPhase := make(map[string]PhaseAttempt, len(groups))
	for _, group := range groups {
		byPhase[group.Assignment.Phase] = group
	}
	if len(byPhase) != 2 {
		t.Fatalf("selected-root groups: %+v", groups)
	}
	// The failed child is the newest implement row: it wins over both
	// the older success and the owner row behind it.
	implement, ok := byPhase["implement"]
	if !ok || implement.Assignment.ID != failed.ID {
		t.Fatalf("implement group: %+v", groups)
	}
	reviewGroup, ok := byPhase["review"]
	if !ok || reviewGroup.Assignment.ID != review.ID {
		t.Fatalf("review group: %+v", groups)
	}
	if implement.Seq >= reviewGroup.Seq {
		t.Fatalf("group order not newest: %+v", groups)
	}
	siblingGroups, _, err := s.LatestPhaseAttempts(ctx, 7, 3, acceptance, sibling)
	if err != nil {
		t.Fatal(err)
	}
	if len(siblingGroups) != 1 || siblingGroups[0].Assignment.ID != sibling {
		t.Fatalf("sibling-root groups: %+v", siblingGroups)
	}
}

func TestLatestPhaseAttemptsScopeValidation(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, _, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	root := strings.Repeat("7", 32)
	acceptance := "d" + strings.Repeat("a", 24)
	if _, _, err := s.LatestPhaseAttempts(ctx, 0, 3, acceptance, root); err == nil {
		t.Fatal("latest phases accepted empty repository")
	}
	if _, _, err := s.LatestPhaseAttempts(ctx, 7, 0, acceptance, root); err == nil {
		t.Fatal("latest phases accepted empty issue")
	}
	if _, _, err := s.LatestPhaseAttempts(ctx, 7, 3, "", root); err == nil {
		t.Fatal("latest phases accepted empty acceptance")
	}
	if _, _, err := s.LatestPhaseAttempts(ctx, 7, 3, acceptance, ""); err == nil {
		t.Fatal("latest phases accepted empty root")
	}
	// A scope with no rows is an empty group set, never an error: the
	// caller renders no phases for it without failing the journey.
	empty, truncated, err := s.LatestPhaseAttempts(ctx, 7, 99999, acceptance, root)
	if err != nil || truncated || len(empty) != 0 {
		t.Fatalf("missing scope groups=%v truncated=%v err=%v", empty, truncated, err)
	}
}

func TestLatestPhaseAttemptsRefuseCorruptRows(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	root := strings.Repeat("7", 32)
	// Valid routing scope but undecodable JSON: the scope filter
	// selects the row, the reader fails, and the read reports an
	// explicit storage failure, never a silent cut.
	if _, err := raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		strings.Repeat("6", 32), 7, 3, "run-poison", factory.AssignmentFinished, 1,
		`{"acceptance": "`+acceptance+`", "attempt_root": "`+root+`", "repository": "seven"}`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LatestPhaseAttempts(ctx, 7, 3, acceptance, root); err == nil {
		t.Fatal("corrupt row read without an error")
	}
}

func TestLatestPhaseAttemptsTruncationSignal(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	root := strings.Repeat("7", 32)
	// More distinct phase values than the group bound carries: the
	// extra group past the bound is the truncation signal, so the
	// caller marks its read incomplete instead of passing a cut set
	// as the whole lineage.
	for i := 0; i <= journeyPhaseGroupLimit; i++ {
		a := journeyAssignment(fmt.Sprintf("%032x", 30000+i), root, acceptance, 7, 3, false)
		a.Phase = fmt.Sprintf("junk-%02d", i)
		insertJourneyAssignment(t, raw, a, factory.AssignmentFinished, 1)
	}
	groups, truncated, err := s.LatestPhaseAttempts(ctx, 7, 3, acceptance, root)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(groups) != journeyPhaseGroupLimit {
		t.Fatalf("truncation signal: truncated=%v groups=%d", truncated, len(groups))
	}
}

func TestLatestIssueAttemptBindsNewestRow(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	// Other-issue rows before the target rows prove the read is scoped.
	otherRoot := strings.Repeat("0", 32)
	insertJourneyAssignment(t, raw, journeyAssignment(otherRoot, otherRoot, acceptance, 7, 888, true), factory.AssignmentFinished, 1)
	root := strings.Repeat("7", 32)
	child := strings.Repeat("8", 32)
	insertJourneyAssignment(t, raw, journeyAssignment(root, root, acceptance, 7, 3, true), factory.AssignmentFinished, 1)
	insertJourneyAssignment(t, raw, journeyAssignment(child, root, acceptance, 7, 3, false), factory.AssignmentFinished, 1)
	latest, err := s.LatestIssueAttempt(ctx, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != child || latest.AttemptRoot != root {
		t.Fatalf("latest attempt: %+v", latest)
	}
	other, err := s.LatestIssueAttempt(ctx, 7, 888)
	if err != nil || other.ID != otherRoot {
		t.Fatalf("other-issue latest: %+v %v", other, err)
	}
	if _, err := s.LatestIssueAttempt(ctx, 7, 99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing attempt err=%v", err)
	}
	if _, err := s.LatestIssueAttempt(ctx, 0, 3); err == nil {
		t.Fatal("latest attempt accepted empty scope")
	}
}

func TestJourneyAdoptionsScopePerIssue(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, _, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	revision := factory.IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222",
		Owner: 1, Repository: 7, Objective: "Build a request form",
		State: "accepted", MaxIssues: 4, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}},
	}
	if err := s.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatal(err)
	}
	binding := factory.IntentTaskBinding{
		RevisionID: revision.ID, TaskKey: "d444444444444444444444444",
		IssueID: 99, IssueIndex: 3, AcceptanceID: "d555555555555555555555555", CreatedUnix: 1,
	}
	if err := s.BindIntentTaskAcceptance(ctx, binding); err != nil {
		t.Fatal(err)
	}
	adoptions, truncated, err := s.JourneyAdoptions(ctx, 7, 3, 16)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(adoptions) != 1 || adoptions[0].Intent != revision.Intent ||
		adoptions[0].Revision != revision.ID || adoptions[0].Binding.AcceptanceID != binding.AcceptanceID {
		t.Fatalf("adoptions: %+v truncated=%v", adoptions, truncated)
	}
	// Wrong repository or issue scope never leaks the binding.
	for _, scope := range [][2]int64{{8, 3}, {7, 4}} {
		none, _, err := s.JourneyAdoptions(ctx, scope[0], scope[1], 16)
		if err != nil || len(none) != 0 {
			t.Fatalf("scope %v adoptions=%v err=%v", scope, none, err)
		}
	}
	if _, _, err := s.JourneyAdoptions(ctx, 0, 3, 16); err == nil {
		t.Fatal("adoptions accepted empty scope")
	}
}

func TestDeliveryJudgmentsForPublicationScope(t *testing.T) {
	ctx := context.Background()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw := journeySQL(t, dsn)
	publicationID := strings.Repeat("9", 32)
	root := strings.Repeat("7", 32)
	in := factory.DeliveryFingerprintInput{
		Reviews:     []string{strings.Repeat("6", 32)},
		AttemptRoot: root, PublicationID: publicationID,
		ScopeDigest: strings.Repeat("3", 64), FixDigest: strings.Repeat("7", 64),
		DiffDigest: strings.Repeat("8", 64), CheckDigest: strings.Repeat("6", 64),
		TreeDigest: strings.Repeat("9", 64),
		Head:       strings.Repeat("a1", 20), Base: strings.Repeat("b2", 20),
		Model: "test-delivery-model", Rubric: factory.DeliveryRubric,
		PolicyRevision: 1, Round: 1,
	}
	fingerprint, err := factory.DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	answer := factory.DeliveryAnswer{
		Selected: factory.DeliveryReadyToMerge, Probability: 0.9, Confidence: 0.95,
		Probabilities: map[string]float64{factory.DeliveryReadyToMerge: 0.9, factory.DeliveryReviewAgain: 0.1},
		Model:         "test-delivery-model", Fingerprint: fingerprint,
	}
	receipt, err := factory.NewDeliveryReceipt(in, fingerprint, answer, true, 41)
	if err != nil {
		t.Fatal(err)
	}
	rawReceipt, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_priority_judgments(snapshot,max_calls,data) VALUES($1,1,$2)`,
		fingerprint, string(rawReceipt)); err != nil {
		t.Fatal(err)
	}
	// A non-delivery journal row in the same table never joins.
	if _, err := raw.Exec(`INSERT INTO factory_priority_judgments(snapshot,max_calls,data) VALUES($1,1,$2)`,
		strings.Repeat("b", 64), `{"snapshot":"`+strings.Repeat("b", 64)+`"}`); err != nil {
		t.Fatal(err)
	}
	receipts, truncated, err := s.DeliveryJudgmentsForPublication(ctx, publicationID, root, 8)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(receipts) != 1 || receipts[0].Fingerprint != fingerprint {
		t.Fatalf("receipts: %+v truncated=%v", receipts, truncated)
	}
	none, _, err := s.DeliveryJudgmentsForPublication(ctx, strings.Repeat("8", 32), root, 8)
	if err != nil || len(none) != 0 {
		t.Fatalf("wrong publication receipts=%v err=%v", none, err)
	}
	if _, _, err := s.DeliveryJudgmentsForPublication(ctx, "bad", root, 8); err == nil {
		t.Fatal("delivery listing accepted invalid scope")
	}
}
