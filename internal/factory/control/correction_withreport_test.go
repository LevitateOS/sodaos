package control

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// checkSeedWithReviewer seeds a published publication exactly like
// checkSeed, but admits reviewer sponsorship before any admission
// freezes its authority. The publication row refuses authority rewrites
// afterwards, so a later grant save would stale every serving advance;
// the grant must predate the publish.
func checkSeedWithReviewer(t *testing.T, issue int64) (*publishFixture, factory.Publication) {
	t.Helper()
	fx := publishSeed(t)
	sponsorship, err := fx.db.Sponsorship(context.Background(), fx.seed.repo, "conn")
	if err != nil {
		t.Fatalf("reviewer fixture sponsorship: %v", err)
	}
	sponsorship.Roles = append(sponsorship.Roles, project.RoleReviewer)
	if err := fx.db.SaveSponsorship(context.Background(), sponsorship); err != nil {
		t.Fatalf("reviewer fixture sponsorship: %v", err)
	}
	a := fx.finishReported(t, issue)
	fx.wire(happyPublisher())
	fx.pass(t)
	p, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stage != factory.PublicationPublished {
		t.Fatalf("check seed publication not published: %+v", p)
	}
	return fx, p
}

// withReportReviewer records one finished phased reviewer round carrying
// the given findings and assessments, then returns the refreshed
// publication. First admission binds the owning history — reviewer row,
// reconciled run and recorded round — instead of a fabricated correction
// cycle alone, so the canonical carried set reconstructs exactly the
// submitted findings.
func withReportReviewer(t *testing.T, ctx context.Context, fx *publishFixture, p factory.Publication, owner factory.Assignment, round int, findings []factory.Finding, assessments []factory.FindingAssessment) factory.Publication {
	t.Helper()
	// Reviewer sponsorship is admitted before this helper runs (see
	// checkSeedWithReviewer): the publication freezes its authority at
	// publish time and UpdatePublication refuses later rewrites, so the
	// grant must predate every admission.
	authority, err := fx.coord.EffectiveAuthority(ctx, owner.Repository)
	if err != nil || !authority.Effective {
		t.Fatalf("reviewer fixture authority: %+v", authority)
	}
	policy, err := fx.db.RepositoryPolicy(ctx, owner.Repository)
	if err != nil {
		t.Fatalf("reviewer fixture policy: %v", err)
	}
	selection, ok := policy.Roles[project.RoleReviewer]
	if !ok {
		t.Fatalf("reviewer fixture role %q has no policy selection", project.RoleReviewer)
	}
	// The reviewer packet guard requires its round already recorded, so
	// the round names the reviewer identity before the packet exists.
	reviewerID := factory.NewID()
	roundID := factory.ReviewCycleID(p.ID, owner.AttemptRoot, round, "review")
	p.ReviewCycles = append(p.ReviewCycles, factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID: roundID, Phase: "review",
		TriggerDecision: "initial-review", TriggerReason: "first review of published head",
		Head: p.Candidate, BaseOID: p.PRCreate.BaseOID,
		FindingDigest:      reviewCycleInputDigest(p.Candidate, p.PRCreate.BaseOID, nil, nil),
		ReviewerAssignment: reviewerID,
		Round:              round,
	})
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatalf("reviewer fixture round: %v", err)
	}
	reviewer := factory.Assignment{
		Authority: authority.Authority,
		ID:        reviewerID, AttemptRoot: owner.AttemptRoot, PublicationAssignment: owner.ID,
		Subject: factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: owner.Issue, Acceptance: owner.Acceptance}},
		Phase:   project.ExecutionReview, ParentAssignment: owner.ID,
		ProjectID: owner.ProjectID, ActorID: owner.ActorID, Role: project.RoleReviewer,
		Repository: owner.Repository, Issue: owner.Issue,
		NativeRev: owner.NativeRev, Acceptance: owner.Acceptance, Preparation: "f" + strings.Repeat("e", 24),
		Harness: selection.Harness + "-" + selection.HarnessVers, HarnessVers: selection.HarnessVers, Model: selection.Model,
		Connection: owner.Connection, SourceCommit: p.Candidate,
		Prompt: []byte("reviewer prompt"), PromptSHA: dispatchDigest("reviewer prompt"),
		Run: factory.NewID(), RunHistory: []string{}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: time.Now().Unix(),
	}
	reviewer.Authority.RequirementsID, err = fx.db.RequirementHead(ctx, reviewer.ProjectID)
	if err != nil {
		t.Fatalf("reviewer fixture requirement head: %v", err)
	}
	reviewer.Authority.ApprovalID, err = fx.db.ApprovalHead(ctx, reviewer.ProjectID)
	if err != nil {
		t.Fatalf("reviewer fixture approval head: %v", err)
	}
	// The registration captures the headed authority, so the heads land
	// before any packet struct below.
	reviewer.RunHistory = []string{reviewer.Run}
	now := time.Now().Truncate(time.Second)
	run := factory.Run{
		ID: reviewer.Run, ProjectID: reviewer.ProjectID, Role: project.RoleReviewer, InputSHA: p.Candidate,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: reviewer.Harness, Model: reviewer.Model,
	}
	reservation := factory.Reservation{
		AssignmentID: reviewer.ID, Repository: reviewer.Repository, Connection: reviewer.Connection,
		State: factory.ReservationHeld, PlannedMinutes: 30,
	}
	view := factory.RunView{RunID: run.ID, Repository: reviewer.Repository, Issue: reviewer.Issue, Attempt: reviewer.ID}
	registration := factory.DispatchRegistration{ID: reviewer.ID, Repository: reviewer.Repository, Authority: reviewer.Authority}
	if err := fx.db.RecordDispatchPacket(ctx, registration, dispatchControlForAssignment(t, fx.db, reviewer), reviewer, reservation, run, view); err != nil {
		t.Fatalf("reviewer fixture packet: %v", err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "reviewer finished", true
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatalf("reviewer fixture run: %v", err)
	}
	output := phasedFenceOutput(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
		Summary: "reviewed",
		Output:  &factory.PhaseOutput{Review: &factory.ReviewCycleReport{CycleID: roundID, NewFindings: findings, Assessments: assessments}},
	})
	reviewer = accountChildResult(t, fx, run, output)
	if reviewer.ID == "" || reviewer.Result == nil || reviewer.Result.PhaseResult == nil ||
		reviewer.Result.PhaseResult.Output == nil || reviewer.Result.PhaseResult.Output.Review == nil {
		t.Fatalf("reviewer settlement is not typed: %+v", reviewer.Result)
	}
	fresh, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func TestPublishCorrectionWithReportDisputedReplayStaysHeld(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 45)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(1, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report := correctionTestReport(newHead, []factory.Finding{finding}, []factory.CorrectionDisposition{{
		Key: finding.Key, Disposition: factory.DispositionDisputed,
	}})
	first := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, nil, nil)
	if len(first.Errors) != 0 || len(first.Corrected) != 0 || len(first.Waits) != 1 || first.Waits[0].Reason != "delivery_judgment_required" {
		t.Fatalf("first disputed report not held: %+v", first)
	}
	journaled, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	exports, pushes := len(fx.host.exports), len(fx.exec.pushes)
	replay := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, nil, nil)
	if len(replay.Errors) != 0 || len(replay.Corrected) != 0 || len(replay.Waits) != 1 || replay.Waits[0].Reason != "delivery_judgment_required" {
		t.Fatalf("disputed replay escaped its hold: %+v", replay)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(held, journaled) || len(fx.host.exports) != exports || len(fx.exec.pushes) != pushes {
		t.Fatal("disputed replay changed publication state or invoked export/push")
	}
}

func TestPublishCorrectionWithReportReplayRefusesChangedReport(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 46)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, checks := withReportFixed(t, p, newHead)
	if first := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks); len(first.Corrected) != 1 {
		t.Fatalf("fixed report did not publish: %+v", first)
	}
	journaled, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*factory.CorrectionReport){
		"candidate":  func(r *factory.CorrectionReport) { r.NewCandidate = strings.Repeat("e", 40) },
		"set digest": func(r *factory.CorrectionReport) { r.SetDigest = strings.Repeat("a", 64) },
		"disposition": func(r *factory.CorrectionReport) {
			r.Dispositions = []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionDisputed}}
		},
		"evidence": func(r *factory.CorrectionReport) {
			r.Evidence = []factory.CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"another.go"}}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := report
			mutate(&changed)
			exports, pushes := len(fx.host.exports), len(fx.exec.pushes)
			replay := fx.coord.PublishCorrectionWithReport(ctx, run.ID, changed, evidence, checks)
			if len(replay.Errors) != 0 || len(replay.Corrected) != 0 || len(replay.Waits) != 1 || replay.Waits[0].Reason != "correction_refused" {
				t.Fatalf("changed %s replay admitted: %+v", name, replay)
			}
			held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(held, journaled) || len(fx.host.exports) != exports || len(fx.exec.pushes) != pushes {
				t.Fatal("changed replay mutated publication state or invoked export/push")
			}
		})
	}
}

// withReportCycle appends one finding-set-digest correction cycle for a
// corrector child to a stored publication.
func withReportCycle(t *testing.T, ctx context.Context, fx *publishFixture, p factory.Publication, child factory.Assignment, digest string) factory.ReviewCycle {
	t.Helper()
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	round := 1
	for _, cycle := range p.ReviewCycles {
		if cycle.Round >= round {
			round = cycle.Round + 1
		}
	}
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                   factory.ReviewCycleID(p.ID, owner.AttemptRoot, round, "review"),
		Phase:                "review",
		TriggerDecision:      "correction-after-review",
		TriggerReason:        "address request-changes on the published head",
		Head:                 p.Candidate,
		BaseOID:              p.PRCreate.BaseOID,
		FindingDigest:        digest,
		ReviewerAssignment:   factory.NewID(),
		CorrectionAssignment: child.ID,
		Round:                round,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatal(err)
	}
	return cycle
}

// withReportFixed builds one fixed code-finding report with its mechanical
// evidence and a check assessment on the new candidate.
func withReportFixed(t *testing.T, p factory.Publication, newHead string) (factory.CorrectionReport, []factory.CorrectionFixEvidence, *factory.CheckAssessment) {
	t.Helper()
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	report := correctionTestReport(newHead, []factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"service.go"}, Commits: []string{newHead}}}
	checks := correctionTestChecks(newHead, p.PRCreate.BaseOID)
	return report, evidence, checks
}

// TestPublishCorrectionWithReportJournalsFixed (real pg) proves the
// canonical serving correction entry: a fixed report journals through
// the canonical gate and advances the head, exactly once per run.
func TestPublishCorrectionWithReportJournalsFixed(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 42)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, checks := withReportFixed(t, p, newHead)
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks)
	if len(served.Errors) != 0 || len(served.Waits) != 0 || len(served.Corrected) != 1 || served.Corrected[0].HeadOID != newHead {
		t.Fatalf("served correction: %+v", served)
	}
	after, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.CorrectionJournals) != 1 || after.CorrectionJournals[0].RunID != run.ID {
		t.Fatalf("correction journal missing: %+v", after.CorrectionJournals)
	}
	if after.Candidate != newHead {
		t.Fatalf("head did not advance: %q", after.Candidate)
	}
	replay := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks)
	if len(replay.Corrected) != 0 {
		t.Fatalf("replay re-advanced: %+v", replay)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 1 {
		t.Fatalf("replay duplicated its journal: %+v", held.CorrectionJournals)
	}
}

// TestPublishCorrectionWithReportWithholdsDisputed (real pg) proves the
// disputed-journal shape now arises through serving: a disputed report
// journals, the head does not advance on contested feedback, and the
// next child-production pass withholds the run pending the funded
// decision instead of launching an automatic second coder run.
func TestPublishCorrectionWithReportWithholdsDisputed(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 43)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, p.Candidate)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report := correctionTestReport(p.Candidate, []factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionDisputed}})
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, nil, nil)
	if len(served.Errors) != 0 || len(served.Corrected) != 0 || len(served.Waits) != 1 || served.Waits[0].Reason != "delivery_judgment_required" {
		t.Fatalf("disputed correction mis-served: %+v", served)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 1 {
		t.Fatalf("disputed journal missing: %+v", held.CorrectionJournals)
	}
	if held.Candidate != p.Candidate {
		t.Fatalf("contested feedback advanced the head: %q", held.Candidate)
	}
	if got := newestCurrentHeadDispositions(held); len(got) != 1 || got[0] != factory.DispositionDisputed {
		t.Fatalf("routing cannot read its journal: %+v", got)
	}
}

// TestPublishCorrectionWithReportRefusesRewrite (real pg) proves the
// canonical binding on the serving entry: a replay that rewrites its
// recorded findings refuses instead of replacing the canonical set, and
// records no second journal.
func TestPublishCorrectionWithReportRefusesRewrite(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 44)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, checks := withReportFixed(t, p, newHead)
	if served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks); len(served.Corrected) != 1 {
		t.Fatalf("fixed correction mis-served: %+v", served)
	}
	rewritten := finding
	rewritten.Defect = "rewritten defect"
	bad := correctionTestReport(newHead, []factory.Finding{rewritten}, []factory.CorrectionDisposition{{Key: rewritten.Key, Disposition: factory.DispositionFixed}})
	replay := fx.coord.PublishCorrectionWithReport(ctx, run.ID, bad, evidence, checks)
	if len(replay.Waits) != 1 || replay.Waits[0].Reason != "correction_refused" {
		t.Fatalf("rewritten replay mis-served: %+v", replay)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 1 {
		t.Fatalf("rewrite recorded a second journal: %+v", held.CorrectionJournals)
	}
	if held.CorrectionJournals[0].Report.Findings[0].Defect != finding.Defect {
		t.Fatalf("canonical set rewritten: %+v", held.CorrectionJournals[0].Report.Findings)
	}
}

// TestPublishCorrectionWithReportRefusesFirstRewrite (real pg) proves the
// first-admission canonical binding: a first report that rewrites its
// finding content while retaining keys refuses before any journal effect,
// instead of journaling under the round's finding-set digest.
func TestPublishCorrectionWithReportRefusesFirstRewrite(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 47)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	rewritten := finding
	rewritten.Defect = "rewritten defect"
	evidence := []factory.CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"service.go"}, Commits: []string{newHead}}}
	checks := correctionTestChecks(newHead, p.PRCreate.BaseOID)
	bad := correctionTestReport(newHead, []factory.Finding{rewritten}, []factory.CorrectionDisposition{{Key: rewritten.Key, Disposition: factory.DispositionFixed}})
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, bad, evidence, checks)
	if len(served.Errors) != 0 || len(served.Corrected) != 0 || len(served.Waits) != 1 || served.Waits[0].Reason != "correction_refused" {
		t.Fatalf("rewritten first report mis-served: %+v", served)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 0 {
		t.Fatalf("rewritten first report journaled: %+v", held.CorrectionJournals)
	}
}

// TestPublishCorrectionWithReportRefusesStaleChecks (real pg) proves stale
// check inputs never admit: a fixed report with check evidence bound to
// another head refuses before any journal effect.
func TestPublishCorrectionWithReportRefusesStaleChecks(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 48)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, _ := withReportFixed(t, p, newHead)
	stale := correctionTestChecks(strings.Repeat("e", 40), p.PRCreate.BaseOID)
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, stale)
	if len(served.Errors) != 0 || len(served.Corrected) != 0 || len(served.Waits) != 1 || served.Waits[0].Reason != "correction_refused" {
		t.Fatalf("stale-check report mis-served: %+v", served)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 0 {
		t.Fatalf("stale-check report journaled: %+v", held.CorrectionJournals)
	}
}

// TestPublishCorrectionWithReportAdmitsRetiredVerifiedCoverage (real pg)
// proves coverage derives from the canonical outstanding set: a key the
// history retired on bound verification evidence stays out of the
// correction round, and the report answering only the outstanding
// finding journals through first admission.
func TestPublishCorrectionWithReportAdmitsRetiredVerifiedCoverage(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 49)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	retired := correctionTestFinding(9, true, factory.FindingFixCode)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{retired},
		[]factory.FindingAssessment{{Key: retired.Key, Status: factory.FindingStillOpen, Evidence: "still open"}})
	p = withReportReviewer(t, ctx, fx, p, owner, 2, []factory.Finding{finding},
		[]factory.FindingAssessment{
			{Key: retired.Key, Status: factory.FindingVerified, Evidence: "fixed and checked"},
			{Key: finding.Key, Status: factory.FindingStillOpen, Evidence: "still open"},
		})
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, checks := withReportFixed(t, p, newHead)
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks)
	if len(served.Errors) != 0 || len(served.Waits) != 0 || len(served.Corrected) != 1 || served.Corrected[0].HeadOID != newHead {
		t.Fatalf("retired-coverage correction mis-served: %+v", served)
	}
	after, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.CorrectionJournals) != 1 {
		t.Fatalf("retired-coverage journal missing: %+v", after.CorrectionJournals)
	}
}

// TestPublishCorrectionWithReportRefusesSupersededCycle (real pg) proves a
// report for a superseded cycle never journals: when the cycle answers
// another head than the live publication — the head moved on before this
// report arrived — the stale report waits without appending a journal,
// advancing the head, or invoking export/push.
func TestPublishCorrectionWithReportRefusesSupersededCycle(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeedWithReviewer(t, 50)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	p = withReportReviewer(t, ctx, fx, p, owner, 1, []factory.Finding{finding}, nil)
	run := correctionRun(t, fx, owner, p.Candidate)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	// Another correction advances the head first through the legacy
	// entry; the recorded cycle now answers a superseded tree, exactly
	// the late settler this gate must wait without journaling.
	advance := correctionRun(t, fx, owner, newHead)
	advanced := fx.coord.PublishCorrection(ctx, advance.ID)
	if len(advanced.Errors) != 0 || len(advanced.Waits) != 0 || len(advanced.Corrected) != 1 {
		t.Fatalf("legacy advance mis-served: %+v", advanced)
	}
	moved, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Candidate != newHead {
		t.Fatalf("legacy advance did not move the head: %+v", moved)
	}
	exports, pushes := len(fx.host.exports), len(fx.exec.pushes)
	report := correctionTestReport(p.Candidate, []factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"service.go"}, Commits: []string{p.Candidate}}}
	checks := correctionTestChecks(p.Candidate, p.PRCreate.BaseOID)
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks)
	if len(served.Errors) != 0 || len(served.Corrected) != 0 || len(served.Waits) != 1 || served.Waits[0].Reason != "correction_cycle_superseded" {
		t.Fatalf("superseded correction mis-served: %+v", served)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 0 || held.Candidate != newHead {
		t.Fatalf("superseded correction mutated publication state: %+v", held)
	}
	if len(fx.host.exports) != exports || len(fx.exec.pushes) != pushes {
		t.Fatal("superseded correction invoked export/push")
	}
}
