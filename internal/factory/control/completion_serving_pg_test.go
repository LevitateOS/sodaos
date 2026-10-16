package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// This supplement proves the Store-backed completion drive on the
// disposable PG fixture (SODA_PG_*; skips without it): a merged
// completion seals real bodies through the production local capture,
// the asserting chooser judges that substance and continues, and the
// release re-observes the native set before releasing. The
// file-backed serving proof runs the same joints here; this variant
// adds charging, fencing, refusal, retry and custody through real
// SQL. It is the supplement, not the proof.

// continueChooser asserts the funded decision input carries sealed
// substance, then answers continue once per consultation.
type continueChooser struct {
	calls int
}

func (f *continueChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	decision, ok := state.(ReassessmentDecisionState)
	if !ok {
		return factory.PriorityAnswer{}, errReassessmentTestState
	}
	if len(decision.Evidence) == 0 || !decision.Local.Sealed() || decision.Completion.ID == "" {
		return factory.PriorityAnswer{}, errReassessmentTestBodiless
	}
	for _, body := range decision.Evidence {
		if err := body.Validate(); err != nil {
			return factory.PriorityAnswer{}, err
		}
	}
	share := 0.9
	answer := factory.PriorityAnswer{Selected: factory.ReassessmentOptionContinue, Model: "supplement", Confidence: 0.9, Probabilities: map[string]float64{}}
	for id := range choices {
		answer.Probabilities[id] = (1 - share) / float64(len(choices)-1)
	}
	answer.Probabilities[factory.ReassessmentOptionContinue] = share
	return answer, nil
}

var (
	errReassessmentTestState    = errorString("decision state outside its substance shape")
	errReassessmentTestBodiless = errorString("decision judges hashes without bodies")
)

type errorString string

func (e errorString) Error() string { return string(e) }

// drivePGFixture merges one completion on an isolated PG database and
// wires the drive with the production store local capture plus the
// scripted native pages.
func drivePGFixture(t *testing.T) (*mergeFixture, *continueChooser, CompletionDriveDeps) {
	t.Helper()
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	fx.wire(exec)
	if report := fx.pass(t); len(report.Merged) != 1 {
		t.Fatalf("merge report: %+v", report)
	}
	m := fx.merge(t, p)
	if m.Stage != factory.MergeMerged {
		t.Fatalf("merge stage: %+v", m)
	}
	db := fx.publish.db
	db.PriorityCallBudget = 100
	chooser := &continueChooser{}
	fx.publish.coord.Planning = &PlanningJudgments{Journal: db, Funding: db, Chooser: chooser}
	deps := CompletionDriveDeps{Enumerator: twoPageEnumerator(t), CaptureLocal: db.CaptureCompletionLocal}
	return fx, chooser, deps
}

func pendingCycleID(t *testing.T, fx *mergeFixture) string {
	t.Helper()
	cycles, err := fx.publish.db.PendingReassessmentCycles(context.Background(), fx.publish.seed.repo)
	if err != nil || len(cycles) != 1 {
		t.Fatalf("cycles=%v err=%v, want the one collecting cycle", cycles, err)
	}
	return cycles[0].Cycle.CycleID
}

// A merged completion seals bodies, judges substance, continues and
// releases, opening the fence with its decision receipt retaining
// the exact bodies.
func TestCompletionDriveReleasesContinuePG(t *testing.T) {
	ctx := context.Background()
	fx, chooser, deps := drivePGFixture(t)
	db := fx.publish.db
	cycleID := pendingCycleID(t, fx)

	report := fx.publish.coord.DriveCompletions(ctx, deps)
	if len(report.Sealed) != 1 || report.Sealed[0] != cycleID {
		t.Fatalf("report=%+v, want the sealed collecting cycle", report)
	}
	sealed, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if sealed.Cycle.State != factory.ReassessmentDeciding || len(sealed.Evidence) != 3 || sealed.EvidenceIssues != 3 || sealed.EvidenceRevision != 41 {
		t.Fatalf("record=%+v, want the deciding seal with three bodies", sealed)
	}
	if sealed.Evidence[1].Title != "title 2" || sealed.Completion.MergedCommit == "" || !sealed.Local.Sealed() {
		t.Fatalf("record=%+v, want the sealed substance and bindings", sealed)
	}

	report = fx.publish.coord.DriveCompletions(ctx, deps)
	if len(report.Released) != 1 || report.Released[0] != cycleID {
		t.Fatalf("report=%+v, want the fresh continue release", report)
	}
	if chooser.calls != 1 {
		t.Fatalf("calls=%d, want one charged consultation for decide plus a free release", chooser.calls)
	}
	released, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if released.Cycle.State != factory.ReassessmentReleased || released.Cycle.Decision != factory.ReassessmentOptionContinue {
		t.Fatalf("record=%+v, want the released continue", released)
	}
	if pending, err := db.CompletionPending(ctx, fx.publish.seed.repo); err != nil || len(pending) != 0 {
		t.Fatalf("pending=%v err=%v, want the open fence after release", pending, err)
	}
	// The funded receipt retains the exact judged bodies: custody,
	// not just the fingerprint.
	state, err := DecisionStateFromRecord(released)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := db.PriorityJudgment(ctx, fingerprintOf(state))
	if err != nil || len(receipt.Calls) != 1 || receipt.Calls[0].Failure != "" {
		t.Fatalf("receipt=%+v err=%v, want the charged continue receipt", receipt, err)
	}
	if !strings.Contains(string(receipt.Calls[0].Request.State), "title 2") {
		t.Fatalf("receipt state=%s, want the retained issue bodies", receipt.Calls[0].Request.State)
	}
}

// An insertion after the seal refuses the stale release with
// attention; explicit Retry reseals the successor over the new set
// and releases it.
func TestCompletionDriveRefusesStaleReleasePG(t *testing.T) {
	ctx := context.Background()
	fx, chooser, deps := drivePGFixture(t)
	db := fx.publish.db
	coord := fx.publish.coord
	cycleID := pendingCycleID(t, fx)

	if report := coord.DriveCompletions(ctx, deps); len(report.Sealed) != 1 {
		t.Fatalf("report=%+v, want the seal over three issues", report)
	}
	enum, ok := deps.Enumerator.(*fixtureEnumerator)
	if !ok {
		t.Fatal("scripted pages lost their enumerator")
	}
	sealed, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	eventID := sealed.Cycle.EventID
	enum.pages["2"] = NativeInventoryPage{Issues: []NativeInventoryIssue{inventoryIssue(t, 3), inventoryIssue(t, 4)}, Complete: true}

	report := coord.DriveCompletions(ctx, deps)
	want := cycleID + ":completion evidence changed during reassessment"
	found := false
	for _, held := range report.Held {
		found = found || held == want
	}
	if !found {
		t.Fatalf("report=%+v, want %q", report, want)
	}
	attended, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if attended.Cycle.State != factory.ReassessmentAttention || attended.Cycle.Decision != factory.ReassessmentOptionContinue {
		t.Fatalf("record=%+v, want the recorded continue held with attention", attended)
	}

	successor, err := coord.StartCompletionRetry(ctx, factory.ReassessmentRetry{
		SuccessorCycleID: factory.NewID(), ExpectedPriorCycle: cycleID,
		CommandID: factory.NewID(), EventID: eventID,
	})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if again, err := coord.StartCompletionRetry(ctx, factory.ReassessmentRetry{
		SuccessorCycleID: successor, ExpectedPriorCycle: cycleID,
		CommandID: factory.NewID(), EventID: eventID,
	}); err != nil || again != successor {
		t.Fatalf("replay=%q err=%v, want the recorded successor without another cycle", again, err)
	}
	if report := coord.DriveCompletions(ctx, deps); len(report.Sealed) != 1 || report.Sealed[0] != successor {
		t.Fatalf("report=%+v, want the successor seal over four issues", report)
	}
	resealed, err := db.ReassessmentRecord(ctx, successor)
	if err != nil {
		t.Fatal(err)
	}
	if len(resealed.Evidence) != 4 {
		t.Fatalf("evidence=%+v, want the four resealed bodies", resealed.Evidence)
	}
	report = coord.DriveCompletions(ctx, deps)
	if len(report.Released) != 1 || report.Released[0] != successor {
		t.Fatalf("report=%+v, want the successor release", report)
	}
	if chooser.calls != 2 {
		t.Fatalf("calls=%d, want one charged consultation per cycle", chooser.calls)
	}
}

// A deferred release retries without charging again: the recorded
// continue decides once, and later drives only re-observe.
func TestCompletionDriveChargedReplayPG(t *testing.T) {
	ctx := context.Background()
	fx, chooser, deps := drivePGFixture(t)
	coord := fx.publish.coord
	cycleID := pendingCycleID(t, fx)

	if report := coord.DriveCompletions(ctx, deps); len(report.Sealed) != 1 {
		t.Fatalf("report=%+v, want the seal", report)
	}
	enum, ok := deps.Enumerator.(*fixtureEnumerator)
	if !ok {
		t.Fatal("scripted pages lost their enumerator")
	}
	enum.idle = false
	report := coord.DriveCompletions(ctx, deps)
	found := false
	for _, held := range report.Held {
		found = found || held == cycleID+":release_deferred"
	}
	if !found {
		t.Fatalf("report=%+v, want the deferred release with the decision standing", report)
	}
	if chooser.calls != 1 {
		t.Fatalf("calls=%d, want the single charged decision", chooser.calls)
	}
	enum.idle = true
	report = coord.DriveCompletions(ctx, deps)
	if len(report.Released) != 1 || chooser.calls != 1 {
		t.Fatalf("report=%+v calls=%d, want the free release without recharge", report, chooser.calls)
	}
}
