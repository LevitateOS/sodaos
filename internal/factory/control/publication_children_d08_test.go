package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// publishPassWaits returns every wait reason in a publish report.
func publishPassWaits(report PublishReport) []string {
	reasons := make([]string, 0, len(report.Waits))
	for _, wait := range report.Waits {
		reasons = append(reasons, wait.Reason)
	}
	return reasons
}

func waitRecorded(report PublishReport, id, reason string) bool {
	for _, wait := range report.Waits {
		if wait.ID == id && wait.Reason == reason {
			return true
		}
	}
	return false
}

// TestPublishPassChargesReviewRoundOnce proves the T44 dispatch leg on
// the serving path: the first reviewer dispatch charges exactly one
// ReviewRound by the immutable reviewer assignment identity without
// touching the extra pass, and the replay pass recharges nothing while
// launching nothing new.
func TestPublishPassChargesReviewRoundOnce(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication := publishedChildProducerSeed(t, 27)
	if len(host.launches) != 1 {
		t.Fatalf("initial reviewer launch count = %d", len(host.launches))
	}
	child, err := fx.db.AssignmentByRun(ctx, host.launches[0].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if len(allowance.Reviews) != 1 || allowance.Reviews[0] != child.ID {
		t.Fatalf("review dispatch charge = %+v", allowance.Reviews)
	}
	if len(allowance.UnchangedHeadExtra) != 0 {
		t.Fatalf("first inspection spent the extra pass: %+v", allowance)
	}
	if got := allowance.ReviewedHeads[publication.Candidate]; got != child.ID {
		t.Fatalf("first inspection did not record its head: %+v", allowance.ReviewedHeads)
	}
	second := fx.coord.PublishPass(ctx)
	if len(second.Errors) != 0 || len(host.launches) != 1 {
		t.Fatalf("replay relaunched: report=%+v launches=%d", second, len(host.launches))
	}
	again, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Reviews) != 1 || again.Revision != allowance.Revision {
		t.Fatalf("replay recharged: %+v", again)
	}
}

// policyReviewSeed publishes with the reviewer path parked (no reviewer
// executor, no checks), so the first real child pass can observe passing
// checks before any review exists.
func policyReviewSeed(t *testing.T, issue int64) (*publishFixture, *producerFlowHost, factory.Assignment) {
	t.Helper()
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	sponsorship, err := fx.db.Sponsorship(ctx, fx.seed.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = append(sponsorship.Roles, project.RoleReviewer)
	if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
		t.Fatal(err)
	}
	host := &producerFlowHost{fakePublishHost: fx.host}
	fx.coord.Host = host
	fx.coord.DispatchReads = fx.seed.reads
	owner := fx.finishReported(t, issue)
	fx.wire(happyPublisher())
	if report := fx.coord.PublishPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("initial publication pass: %+v", report)
	}
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || p.Stage != factory.PublicationPublished {
		t.Fatalf("publication not linked: %+v %v", p, err)
	}
	if len(p.ReviewCycles) != 0 {
		t.Fatalf("parked reviewer recorded a cycle: %+v", p.ReviewCycles)
	}
	return fx, host, owner
}

// TestPublishPassPolicyReviewCarriesSeparateCause proves the T44 policy
// leg on the serving path: passing checks with a missing approval the
// configured reviewer can supply dispatches exactly one bounded
// policy-required review with its separately recorded cause, charging
// only its ReviewRound.
func TestPublishPassPolicyReviewCarriesSeparateCause(t *testing.T) {
	ctx := context.Background()
	fx, host, owner := policyReviewSeed(t, 28)
	fx.coord.Reviews = &fakeReviewer{}
	fx.coord.Checks = happyCheckObserver()
	// CheckPass produces children by committed design, so the
	// policy-required pass is recorded here: exactly one bounded pass
	// with its separately recorded cause, one launch, one charged round.
	if report := fx.coord.CheckPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("check pass: %+v", report)
	}
	recorded, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(recorded.ReviewCycles) != 1 {
		t.Fatalf("policy pass recorded %d cycles", len(recorded.ReviewCycles))
	}
	if len(host.launches) != 1 {
		t.Fatalf("policy review production launches=%d", len(host.launches))
	}
	cycle := recorded.ReviewCycles[0]
	if cycle.TriggerDecision != policyReviewDecision {
		t.Fatalf("policy pass recorded cause %q", cycle.TriggerDecision)
	}
	if !policyReviewRecorded(recorded, recorded.Candidate, 6) {
		t.Fatal("recorded policy pass is not deduplicated")
	}
	child, err := fx.db.AssignmentByRun(ctx, host.launches[0].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.ID != cycle.ReviewerAssignment {
		t.Fatalf("policy child %q is not the recorded reviewer assignment %q", child.ID, cycle.ReviewerAssignment)
	}
	host.prepPhase = project.PrepareReady
	report := fx.coord.PublishPass(ctx)
	if len(report.Errors) != 0 || len(host.launches) != 1 {
		t.Fatalf("policy replay relaunched: report=%+v launches=%d", report, len(host.launches))
	}
	after, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.ReviewCycles) != 1 {
		t.Fatalf("policy replay recorded %d cycles", len(after.ReviewCycles))
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if len(allowance.Reviews) != 1 || allowance.Reviews[0] != child.ID {
		t.Fatalf("policy review did not charge its round: %+v", allowance.Reviews)
	}
	if len(allowance.UnchangedHeadExtra) != 0 {
		t.Fatalf("policy review spent the extra pass: %+v", allowance)
	}
	// A further pass replays instead of dispatching a second bounded pass.
	third := fx.coord.PublishPass(ctx)
	if len(third.Errors) != 0 || len(host.launches) != 1 {
		t.Fatalf("policy replay relaunched: report=%+v launches=%d", third, len(host.launches))
	}
	final, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(final.ReviewCycles) != 1 {
		t.Fatalf("policy second replay recorded %d cycles", len(final.ReviewCycles))
	}
}

// TestPublishPassWithholdsDisputedFeedbackForDecision proves the T44
// routing leg on the serving path: disputed feedback on the current head
// withholds the correction run pending the funded D08 decision instead of
// launching an automatic second coder run.
func TestPublishPassWithholdsDisputedFeedbackForDecision(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication := publishedChildProducerSeed(t, 29)
	if len(host.launches) != 1 {
		t.Fatalf("initial reviewer launch count = %d", len(host.launches))
	}
	reviewerLaunch := host.launches[0]
	reviewOutput := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"empty input panics\",\"findings\":[\"empty input\"]}\n```"
	_, reviewerRun := settleProducedChild(t, fx, reviewerLaunch.Run.ID, reviewOutput)
	fx.coord.Reviews = &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":31}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 31, CommentID: 32, ReviewerID: 6, PRID: publication.PRID,
			PRNumber: publication.PRNumber, IssueID: publication.PRCreate.IssueID,
			HeadOID: publication.Candidate, BaseOID: publication.PRCreate.BaseOID,
			CommitID: publication.Candidate, Event: "REQUEST_CHANGES",
		},
	}
	if _, err := fx.coord.SubmitReviewForRun(ctx, reviewerRun.ID); err != nil {
		t.Fatalf("submit produced reviewer result: %v", err)
	}
	fx.coord.Checks = happyCheckObserver()
	if report := fx.coord.CheckPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("check pass: %+v", report)
	}
	// The disputed correction journal answers the current head: it uses
	// the FW37-populated input shape (a recorded cycle plus its exact
	// finding-set journal entry), which the store validates structurally.
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	round := 1
	for _, cycle := range p.ReviewCycles {
		if cycle.Round > round {
			round = cycle.Round
		}
	}
	round++
	disputed := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID: factory.ReviewCycleID(p.ID, owner.AttemptRoot, round, "review"), Phase: "review",
		TriggerDecision: "correction-after-review", TriggerReason: "address request-changes with pass checks",
		Head: p.Candidate, BaseOID: p.PRCreate.BaseOID,
		FindingDigest:      mustFindingSetDigest(t, correctionTestKey(1)),
		ReviewerAssignment: factory.NewID(), CorrectionAssignment: factory.NewID(), Round: round,
	}
	p.ReviewCycles = append(p.ReviewCycles, disputed)
	p.CorrectionJournals = append(p.CorrectionJournals, factory.CorrectionJournal{
		CycleID: disputed.ID, RunID: factory.NewID(),
		Report: factory.CorrectionReport{
			SetDigest: disputed.FindingDigest,
			Findings:  []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)},
			Dispositions: []factory.CorrectionDisposition{
				{Key: correctionTestKey(1), Disposition: factory.DispositionDisputed},
			},
		},
	})
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatalf("disputed journal unrecorded: %v", err)
	}
	launchesBefore := len(host.launches)
	cyclesBefore := len(p.ReviewCycles)
	allowanceBefore, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	report := fx.coord.PublishPass(ctx)
	if len(report.Errors) != 0 {
		t.Fatalf("disputed pass errored: %+v", report)
	}
	if len(host.launches) != launchesBefore {
		t.Fatalf("disputed feedback launched another run: launches=%d", len(host.launches))
	}
	if !waitRecorded(report, p.ID, "delivery_judgment_required") {
		t.Fatalf("disputed feedback did not wait for its decision: %+v", publishPassWaits(report))
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.ReviewCycles) != cyclesBefore {
		t.Fatalf("withheld dispute recorded cycles: %d", len(held.ReviewCycles))
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if len(allowance.Corrections) != len(allowanceBefore.Corrections) {
		t.Fatalf("withheld dispute charged a correction: %+v", allowance.Corrections)
	}
}

func mustFindingSetDigest(t *testing.T, keys ...string) string {
	t.Helper()
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

// TestPublishPassWithholdsReviewWhenRoundsExhaust proves the finite
// counter leg end to end: one ReviewRound covers the first inspection,
// and the next head's first inspection waits with
// review_allowance_exhausted instead of launching without cover.
func TestPublishPassWithholdsReviewWhenRoundsExhaust(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	policy.AttemptLimits.ReviewRounds = 1
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	sponsorship, err := fx.db.Sponsorship(ctx, fx.seed.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = append(sponsorship.Roles, project.RoleReviewer)
	if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
		t.Fatal(err)
	}
	host := &producerFlowHost{fakePublishHost: fx.host}
	fx.coord.Host = host
	fx.coord.DispatchReads = fx.seed.reads
	fx.coord.Reviews = &fakeReviewer{}
	owner := fx.finishReported(t, 30)
	fx.wire(happyPublisher())
	if report := fx.coord.PublishPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("initial publication pass: %+v", report)
	}
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || p.Stage != factory.PublicationPublished {
		t.Fatalf("publication not linked: %+v %v", p, err)
	}
	if len(host.launches) != 1 {
		t.Fatalf("first inspection launch count = %d", len(host.launches))
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if len(allowance.Reviews) != 1 {
		t.Fatalf("first inspection charge = %+v", allowance.Reviews)
	}
	// The first inspection reports request-changes over failing checks,
	// which still earns its bounded correction (correction cycles are a
	// separate allowance).
	reviewerLaunch := host.launches[0]
	reviewOutput := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"empty input panics\",\"findings\":[\"empty input\"]}\n```"
	_, reviewerRun := settleProducedChild(t, fx, reviewerLaunch.Run.ID, reviewOutput)
	fx.coord.Reviews = &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":31}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 31, CommentID: 32, ReviewerID: 6, PRID: p.PRID,
			PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID,
			HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID,
			CommitID: p.Candidate, Event: "REQUEST_CHANGES",
		},
	}
	if _, err := fx.coord.SubmitReviewForRun(ctx, reviewerRun.ID); err != nil {
		t.Fatalf("submit produced reviewer result: %v", err)
	}
	checks := happyCheckObserver()
	checks.observe = func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, err := happyCheckObserver().observe(target)
		observed.Checks = []factory.ObservedCheck{{Context: "ci", State: "failure"}}
		return observed, err
	}
	fx.coord.Checks = checks
	if report := fx.coord.CheckPass(ctx); len(report.Errors) != 0 || len(host.launches) != 2 {
		t.Fatalf("failed CI plus request-changes did not produce one correction: report=%+v launches=%d", report, len(host.launches))
	}
	correction, err := fx.db.AssignmentByRun(ctx, host.launches[1].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	_, correctionRun := settleProducedChild(t, fx, correction.Run, correctionOutput(newHead))
	corrected := fx.coord.PublishCorrection(ctx, correctionRun.ID)
	if len(corrected.Errors) != 0 || len(corrected.Corrected) != 1 || corrected.Corrected[0].HeadOID != newHead {
		t.Fatalf("correction did not advance the publication head: %+v", corrected)
	}
	// The single ReviewRound is spent: the corrected head's first
	// inspection waits instead of launching without cover, and the
	// attempt still holds its spent round exactly once.
	report := fx.coord.PublishPass(ctx)
	if len(report.Errors) != 0 {
		t.Fatalf("exhausted pass errored: %+v", report)
	}
	if len(host.launches) != 2 {
		t.Fatalf("exhausted allowance launched another review: launches=%d", len(host.launches))
	}
	fresh, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !waitRecorded(report, fresh.ID, "review_allowance_exhausted") {
		t.Fatalf("exhausted allowance did not wait: %+v", publishPassWaits(report))
	}
	spent, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if len(spent.Reviews) != 1 || len(spent.Corrections) != 1 {
		t.Fatalf("exhausted ledger changed: %+v", spent)
	}
}
