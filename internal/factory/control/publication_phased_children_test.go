package control

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func phasedFenceOutput(t *testing.T, result factory.PhaseResult) string {
	t.Helper()
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return "work\n```" + factory.ResultFence + "\n" + string(payload) + "\n```\n"
}

func phasedReviewFinding(child factory.Assignment) factory.Finding {
	return factory.Finding{
		Key: correctionTestKey(1), OriginRun: child.Run,
		Severity: factory.FindingSeverityMajor, Defect: "missing bound",
		Evidence: "line 42 lacks a check", RequiredOutcome: "add the check",
		FixKind: factory.FindingFixCode, Actionable: true,
	}
}

func phasedReviewOutput(t *testing.T, cycleID string, child factory.Assignment) string {
	t.Helper()
	return phasedFenceOutput(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
		Summary: "reviewed",
		Output: &factory.PhaseOutput{Review: &factory.ReviewCycleReport{
			CycleID: cycleID, NewFindings: []factory.Finding{phasedReviewFinding(child)},
		}},
	})
}

func phasedCorrectionReport(t *testing.T, child factory.Assignment) factory.CorrectionReport {
	t.Helper()
	finding := phasedReviewFinding(child)
	keys := []string{finding.Key}
	setDigest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	return factory.CorrectionReport{
		SetDigest: setDigest, Findings: []factory.Finding{finding},
		Dispositions: []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}},
	}
}

func phasedCorrectionOutput(t *testing.T, child factory.Assignment) string {
	t.Helper()
	report := phasedCorrectionReport(t, child)
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	// The corrector emits both envelopes the wire consumes: the typed
	// phase result for settlement and the fenced correction report for
	// the canonical report gate.
	return phasedFenceOutput(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionCorrect, Status: "completed",
		Summary: "corrected",
		Output:  &factory.PhaseOutput{Correction: &report},
	}) + "```" + factory.CorrectionReportFence + "\n" + string(payload) + "\n```\n"
}

func phasedReviewCycle(t *testing.T, ctx context.Context, fx *publishFixture, owner factory.Assignment, childID string) factory.ReviewCycle {
	t.Helper()
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, cycle := range p.ReviewCycles {
		if cycle.ReviewerAssignment == childID {
			return cycle
		}
	}
	t.Fatalf("no review cycle names reviewer %s: %+v", childID, p.ReviewCycles)
	return factory.ReviewCycle{}
}

func phasedCorrectionCycle(t *testing.T, ctx context.Context, fx *publishFixture, owner factory.Assignment, childID string) factory.ReviewCycle {
	t.Helper()
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, cycle := range p.ReviewCycles {
		if cycle.CorrectionAssignment == childID {
			return cycle
		}
	}
	t.Fatalf("no review cycle names corrector %s: %+v", childID, p.ReviewCycles)
	return factory.ReviewCycle{}
}

func failingPhasedChecks(fx *publishFixture) {
	checks := happyCheckObserver()
	observe := checks.observe
	checks.observe = func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, err := observe(target)
		if err != nil {
			return observed, err
		}
		observed.Checks = []factory.ObservedCheck{{Context: "ci", State: "failure"}}
		return observed, err
	}
	fx.coord.Checks = checks
}

func requestChangesReviewer(p factory.Publication) *fakeReviewer {
	return &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":31}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 31, CommentID: 32, ReviewerID: 6, PRID: p.PRID,
			PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID,
			HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID,
			CommitID: p.Candidate, Event: "REQUEST_CHANGES",
		},
	}
}

func TestPhasedProducerLaunchesReviewThroughTypedWire(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication, _ := phasedChildProducerSeed(t, 41)
	if len(host.launches) != 1 {
		t.Fatalf("phased seed launch count = %d", len(host.launches))
	}
	launch := host.launches[0]
	if launch.Run.ExecutionPhase != project.ExecutionReview || len(launch.ExecutionManifest) != 0 {
		t.Fatalf("review launch carries no manifest but its phase: %+v", launch.Run)
	}
	child, err := fx.db.AssignmentByRun(ctx, launch.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Phase != project.ExecutionReview || child.Role != project.RoleReviewer ||
		child.ParentAssignment != owner.ID || child.PublicationAssignment != owner.ID ||
		child.AttemptRoot != owner.AttemptRoot || child.SourceCommit != publication.Candidate ||
		child.Subject.Kind != factory.SubjectIssue || child.Subject.Issue == nil ||
		child.Subject.Issue.Issue != publication.Issue || child.Subject.Issue.Acceptance != publication.Acceptance {
		t.Fatalf("review child lost its phased lineage: %+v", child)
	}
	if child.Preparation != publicationPreparationID(publication.ID, publication.Candidate) ||
		child.Preparation == owner.Preparation {
		t.Fatalf("reviewer session is not distinct: child=%+v owner=%q", child, owner.Preparation)
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(allowance.Reviews) != 1 || allowance.Reviews[0] != child.ID {
		t.Fatalf("review round not charged once by assignment identity: %+v %v", allowance, err)
	}
	cycle := phasedReviewCycle(t, ctx, fx, owner, child.ID)
	if cycle.Head != publication.Candidate || cycle.BaseOID != publication.PRCreate.BaseOID {
		t.Fatalf("review cycle lost its candidate lineage: %+v", cycle)
	}
	_, reviewerRun := settleProducedChild(t, fx, launch.Run.ID, phasedReviewOutput(t, cycle.ID, child))
	reviewer, err := fx.db.AssignmentByRun(ctx, reviewerRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reviewer.Result == nil || reviewer.Result.PhaseResult == nil ||
		reviewer.Result.PhaseResult.Output == nil || reviewer.Result.PhaseResult.Output.Review == nil {
		t.Fatalf("review settlement is not typed: %+v", reviewer.Result)
	}
	fx.coord.Reviews = requestChangesReviewer(publication)
	adopted, err := fx.coord.SubmitReviewForRun(ctx, reviewerRun.ID)
	if err != nil {
		t.Fatalf("typed review verdict was not consumed: %v", err)
	}
	if adopted.Event != "REQUEST_CHANGES" || adopted.HeadOID != publication.Candidate {
		t.Fatalf("bound verdict lost its submission: %+v", adopted)
	}
}

func TestPhasedProducerLaunchesCorrectionWithSingletonManifest(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication, seedPlan := phasedChildProducerSeed(t, 42)
	if len(host.launches) != 1 {
		t.Fatalf("phased seed launch count = %d", len(host.launches))
	}
	reviewerLaunch := host.launches[0]
	reviewer, err := fx.db.AssignmentByRun(ctx, reviewerLaunch.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	cycle := phasedReviewCycle(t, ctx, fx, owner, reviewer.ID)
	_, reviewerRun := settleProducedChild(t, fx, reviewerLaunch.Run.ID, phasedReviewOutput(t, cycle.ID, reviewer))
	fx.coord.Reviews = requestChangesReviewer(publication)
	if _, err := fx.coord.SubmitReviewForRun(ctx, reviewerRun.ID); err != nil {
		t.Fatalf("submit produced reviewer result: %v", err)
	}
	failingPhasedChecks(fx)
	checkReport := fx.coord.CheckPass(ctx)
	if len(checkReport.Errors) != 0 || len(host.launches) != 2 {
		t.Fatalf("request-changes plus failed CI did not produce one phased correction: report=%+v launches=%d", checkReport, len(host.launches))
	}
	correctionLaunch := host.launches[1]
	if correctionLaunch.Run.ExecutionPhase != project.ExecutionCorrect {
		t.Fatalf("correction launch lost its phase: %+v", correctionLaunch.Run)
	}
	manifest, err := project.ParseExecutionManifest(correctionLaunch.ExecutionManifest)
	if err != nil {
		t.Fatalf("correction stages no valid manifest: %v", err)
	}
	if manifest.Phase != project.ExecutionCorrect || manifest.MaxChildren != 0 || manifest.Run != correctionLaunch.Run.ID {
		t.Fatalf("correction manifest is not the singleton correct envelope: %+v", manifest)
	}
	if correctionLaunch.Run.ManifestDigest != project.ExecutionManifestDigest(correctionLaunch.ExecutionManifest) {
		t.Fatal("correction manifest does not match its digest")
	}
	var staged factory.InvestigationPlan
	if err := json.Unmarshal(manifest.Plan, &staged); err != nil {
		t.Fatalf("correction manifest plan undecodable: %v", err)
	}
	if err := staged.Validate(); err != nil || len(staged.Tasks) != 1 || len(staged.Lanes) != 1 {
		t.Fatalf("correction plan is not a valid singleton: %+v %v", staged, err)
	}
	task := staged.Tasks[0]
	if len(task.DependsOn) != 0 || len(task.InterfaceWrite) != 0 || len(task.CheckIDs) == 0 {
		t.Fatalf("correction task carries deps, handoffs or no checks: %+v", task)
	}
	if len(staged.Checks) != len(seedPlan.Checks) {
		t.Fatalf("correction checks are not the derived investigation policy: %+v", staged.Checks)
	}
	corrector, err := fx.db.AssignmentByRun(ctx, correctionLaunch.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if corrector.Phase != project.ExecutionCorrect || corrector.Role != project.RoleCoder ||
		corrector.ParentAssignment != reviewer.ID || corrector.PublicationAssignment != owner.ID ||
		corrector.AttemptRoot != owner.AttemptRoot || corrector.SourceCommit != publication.Candidate ||
		corrector.Preparation != owner.Preparation ||
		corrector.Subject.Kind != factory.SubjectIssue || corrector.Subject.Issue == nil ||
		corrector.Subject.Issue.Acceptance != publication.Acceptance {
		t.Fatalf("corrector lost its phased lineage: %+v", corrector)
	}
	correctionCycle := phasedCorrectionCycle(t, ctx, fx, owner, corrector.ID)
	if correctionCycle.Head != publication.Candidate {
		t.Fatalf("correction cycle lost its candidate: %+v", correctionCycle)
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(allowance.Corrections) != 1 || allowance.Corrections[0] != corrector.ID {
		t.Fatalf("correction cycle not charged once by assignment identity: %+v %v", allowance, err)
	}
	output := phasedCorrectionOutput(t, corrector)
	settled, _ := settleProducedChild(t, fx, corrector.Run, output)
	if settled.Result == nil || settled.Result.PhaseResult == nil ||
		settled.Result.PhaseResult.Output == nil || settled.Result.PhaseResult.Output.Correction == nil {
		t.Fatalf("correction settlement is not typed: %+v", settled.Result)
	}
	parsed, ok := ParseCorrectionReport(output)
	if !ok || len(parsed.Dispositions) != 1 || parsed.Dispositions[0].Disposition != factory.DispositionFixed {
		t.Fatalf("correction output does not parse through the canonical report gate: %+v %v", parsed, ok)
	}
	if err := parsed.Validate(); err != nil {
		t.Fatalf("parsed correction report invalid: %v", err)
	}
}

func TestPhasedProducerReplaysWithoutRelaunchOrRecharge(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication, _ := phasedChildProducerSeed(t, 43)
	if len(host.launches) != 1 {
		t.Fatalf("phased seed launch count = %d", len(host.launches))
	}
	before, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(before.Reviews) != 1 {
		t.Fatalf("seed review uncharged: %+v %v", before, err)
	}
	again := fx.coord.PublishPass(ctx)
	if len(again.Errors) != 0 || len(host.launches) != 1 {
		t.Fatalf("review redelivery relaunched: report=%+v launches=%d", again, len(host.launches))
	}
	after, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(after.Reviews) != 1 || after.Reviews[0] != before.Reviews[0] {
		t.Fatalf("review redelivery recharged: %+v %v", after, err)
	}
	reviewer, err := fx.db.AssignmentByRun(ctx, host.launches[0].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	cycle := phasedReviewCycle(t, ctx, fx, owner, reviewer.ID)
	_, reviewerRun := settleProducedChild(t, fx, reviewer.Run, phasedReviewOutput(t, cycle.ID, reviewer))
	fx.coord.Reviews = requestChangesReviewer(publication)
	if _, err := fx.coord.SubmitReviewForRun(ctx, reviewerRun.ID); err != nil {
		t.Fatal(err)
	}
	failingPhasedChecks(fx)
	if report := fx.coord.CheckPass(ctx); len(report.Errors) != 0 || len(host.launches) != 2 {
		t.Fatalf("correction undispatched: report=%+v launches=%d", report, len(host.launches))
	}
	charged, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(charged.Corrections) != 1 {
		t.Fatalf("correction uncharged: %+v %v", charged, err)
	}
	redelivered := fx.coord.PublishPass(ctx)
	if len(redelivered.Errors) != 0 || len(host.launches) != 2 {
		t.Fatalf("correction redelivery relaunched: report=%+v launches=%d", redelivered, len(host.launches))
	}
	recharged, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(recharged.Reviews) != 1 || len(recharged.Corrections) != 1 ||
		recharged.Reviews[0] != charged.Reviews[0] || recharged.Corrections[0] != charged.Corrections[0] {
		t.Fatalf("redelivery recharged: %+v %v", recharged, err)
	}
}

func TestPhasedProducerRetryRelaunchesWithoutRecharge(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, _, _ := phasedChildProducerSeed(t, 44)
	if len(host.launches) != 1 {
		t.Fatalf("phased seed launch count = %d", len(host.launches))
	}
	assigned, err := fx.db.AssignmentByRun(ctx, host.launches[0].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Stage != factory.AssignmentAssigned || assigned.Phase != project.ExecutionReview {
		t.Fatalf("seed reviewer not awaiting retry: %+v", assigned)
	}
	// Recovery settles the superseded run and releases its reservation
	// before the next attempt, so the retry plans against a free
	// repository session and re-holds its own slot.
	superseded, err := fx.db.FactoryRun(ctx, assigned.Run)
	if err != nil {
		t.Fatal(err)
	}
	superseded.Outcome, superseded.Reconciled, superseded.Summary = factory.Failed, true, "superseded"
	if err := fx.db.SaveFactoryRun(ctx, superseded); err != nil {
		t.Fatal(err)
	}
	if err := fx.db.ReleaseReservation(ctx, assigned.ID); err != nil {
		t.Fatal(err)
	}
	report := DispatchReport{}
	retryAttempt(ctx, fx.coord.dispatchDeps(), assigned, &report)
	if len(report.Errors) != 0 || len(report.Launched) != 1 {
		t.Fatalf("phased review retry did not relaunch: %+v", report)
	}
	if report.Launched[0].AssignmentID != assigned.ID || report.Launched[0].RunID == assigned.Run {
		t.Fatalf("retry launched another identity: %+v", report.Launched)
	}
	relaunched, err := fx.db.Assignment(ctx, assigned.ID)
	if err != nil {
		t.Fatal(err)
	}
	if relaunched.Attempts != 2 || len(relaunched.RunHistory) != 2 {
		t.Fatalf("retry did not advance its run: %+v", relaunched)
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || len(allowance.Reviews) != 1 || allowance.Reviews[0] != assigned.ID {
		t.Fatalf("retry recharged its round: %+v %v", allowance, err)
	}
}

func TestDeriveCorrectionPlanSingletonAndBlockers(t *testing.T) {
	acceptance := factory.Acceptance{ID: "d" + strings.Repeat("a", 24)}
	root := factory.NewID()
	investigation := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: root,
		ScopeDigest: dispatchDigest("correction scope"), InputDigest: dispatchDigest("correction input"),
		SourceCommit: strings.Repeat("1", 40), IntegrationCommit: strings.Repeat("1", 40),
		Preparation:        "f" + strings.Repeat("b", 24),
		Approach:           "inspect the selected source, then implement behind checks",
		AcceptanceCoverage: []string{"accept-one"},
		Inspected: []factory.InspectedSource{{
			Path: "README.md", Commit: strings.Repeat("1", 40), Evidence: "selected source",
		}},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "inspect the selected source",
			Outputs: []string{"notes"}, AcceptanceRefs: []string{"accept-one"},
			Lane: "lane-one", OwnedPaths: []string{"README.md", "internal/"}, CheckIDs: []string{"check-one"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
		Checks: []factory.CheckSpec{{
			ID: "check-one", Argv: []string{"make", "check"}, Cwd: ".",
			TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true,
		}},
	}
	digest, err := factory.PlanDigest(investigation)
	if err != nil {
		t.Fatal(err)
	}
	investigation.Digest = digest
	head, base, cycle := strings.Repeat("2", 40), strings.Repeat("1", 40), factory.NewID()
	keys := []string{correctionTestKey(1)}
	derived, err := deriveCorrectionPlan(acceptance, investigation, root, head, base, investigation.Preparation, cycle, dispatchDigest("findings"), keys, []string{"native-ci/build"})
	if err != nil {
		t.Fatalf("singleton correction plan refused: %v", err)
	}
	if err := derived.Validate(); err != nil {
		t.Fatalf("derived plan invalid: %v", err)
	}
	if err := derived.Executable(); err != nil {
		t.Fatalf("derived plan unexecutable: %v", err)
	}
	if len(derived.Tasks) != 1 || len(derived.Lanes) != 1 || derived.Tasks[0].ID != "correct" || derived.Lanes[0].ID != "correct" {
		t.Fatalf("derived plan is not a singleton: %+v", derived)
	}
	single := derived.Tasks[0]
	if len(single.DependsOn) != 0 || len(single.InterfaceWrite) != 0 || len(single.CheckIDs) != 1 {
		t.Fatalf("singleton task carries deps, handoffs or missing checks: %+v", single)
	}
	if len(derived.Checks) != 1 || derived.Checks[0].ID != "check-one" {
		t.Fatalf("singleton checks are not derived: %+v", derived.Checks)
	}
	manifest, err := stageExecutionManifest(derived, factory.NewID(), time.Now().Add(time.Hour), factory.ExecutionProfile{Effort: factory.EffortXHigh, Contributor: true}, project.ExecutionCorrect, 0)
	if err != nil {
		t.Fatalf("singleton manifest unstaged: %v", err)
	}
	parsed, err := project.ParseExecutionManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Phase != project.ExecutionCorrect || parsed.MaxChildren != 0 {
		t.Fatalf("manifest is not the correct-phase singleton envelope: %+v", parsed)
	}
	blocked := []struct {
		name       string
		investment factory.InvestigationPlan
		checks     []string
		root       string
	}{
		{"plan without checks", withoutChecks(investigation), []string{"native-ci/build"}, root},
		{"policy without checks", investigation, nil, root},
		{"foreign attempt root", investigation, []string{"native-ci/build"}, factory.NewID()},
	}
	for _, tc := range blocked {
		if _, err := deriveCorrectionPlan(acceptance, tc.investment, tc.root, head, base, investigation.Preparation, cycle, dispatchDigest("findings"), keys, tc.checks); err == nil {
			t.Fatalf("%s derived instead of blocking", tc.name)
		} else if reason, ok := correctionPlanWaitReason(err); !ok || reason != "correction_plan_unavailable" {
			t.Fatalf("%s blocked without the visible planning blocker: %v", tc.name, err)
		}
	}
	unresolved := investigation
	unresolved.Unresolved = []string{"which line panics"}
	if _, err := deriveCorrectionPlan(acceptance, unresolved, root, head, base, investigation.Preparation, cycle, dispatchDigest("findings"), keys, []string{"native-ci/build"}); err == nil {
		t.Fatal("unexecutable plan derived instead of blocking")
	} else if reason, ok := correctionPlanWaitReason(err); !ok || reason != "correction_plan_unavailable" {
		t.Fatalf("unexecutable plan blocked without the visible planning blocker: %v", err)
	}
}

func withoutChecks(plan factory.InvestigationPlan) factory.InvestigationPlan {
	plan.Checks = nil
	plan.Tasks[0].CheckIDs = nil
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		return factory.InvestigationPlan{}
	}
	plan.Digest = digest
	return plan
}

// B16 phased publication-children tests: the real producer admits
// phase=review/correct children through the dispatch packet, launches
// them through the fake host, settles their typed outputs, and serves
// the review verdict through the canonical consumer. The fake host is
// orchestration evidence only: every admission, charge, lineage and
// settlement check runs against the real store and coordinator.

const (
	phasedSeedCandidate = "2222222222222222222222222222222222222222"
	phasedSeedBase      = "cccccccccccccccccccccccccccccccccccccccc"
)

func phasedSeedPlan(t *testing.T, root factory.Assignment) factory.InvestigationPlan {
	t.Helper()
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: root.AttemptRoot,
		ScopeDigest: dispatchDigest("phased scope"), InputDigest: dispatchDigest("phased input"),
		SourceCommit: root.SourceCommit, IntegrationCommit: root.SourceCommit,
		Preparation:        root.Preparation,
		Approach:           "inspect the selected source, then implement behind checks",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "objective 31; inspect README.md",
			Outputs: []string{"notes"}, AcceptanceRefs: []string{"accept-one"},
			Lane: "lane-one", OwnedPaths: []string{"README.md"}, CheckIDs: []string{"check-one"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
		Checks: []factory.CheckSpec{{
			ID: "check-one", Argv: []string{"make", "check"}, Cwd: ".",
			TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true,
		}},
	}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = digest
	if err := plan.Validate(); err != nil {
		t.Fatalf("phased seed plan invalid: %v", err)
	}
	if err := plan.Executable(); err != nil {
		t.Fatalf("phased seed plan unexecutable: %v", err)
	}
	return plan
}

// phasedChildProducerSeed publishes one phased journey's candidate under
// its investigate root: a real dispatch packet admits the investigate
// root, a typed plan settles it, a real advancement packet admits the
// implement child, a typed receipt settles it, persisted plan evidence
// opens the FW15 gate, and a real publish pass links the PR. The
// publication owner is the phased investigate root.
func phasedChildProducerSeed(t *testing.T, issue int64) (*publishFixture, *producerFlowHost, factory.Assignment, factory.Publication, factory.InvestigationPlan) {
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
	fx.coord.Reviews = &fakeReviewer{}

	decision := fx.seed.accept(t, issue, "d"+strings.Repeat("a", 23)+string(rune('0'+issue%10)))
	now := time.Now().Truncate(time.Second)
	prompt := []byte("phased investigate prompt")
	assignmentID := factory.NewID()
	owner := factory.Assignment{
		ID: assignmentID, AttemptRoot: assignmentID, PublicationAssignment: assignmentID,
		Subject: factory.ExecutionSubject{
			Kind:  factory.SubjectIssue,
			Issue: &factory.IssueSubject{Issue: issue, Acceptance: decision.ID},
		},
		Phase:     project.ExecutionInvestigate,
		ProjectID: fx.seed.proj, ActorID: 7, Role: project.RoleCoder,
		Repository: fx.seed.repo, Issue: issue, Revision: 0, NativeRev: 41,
		Acceptance: decision.ID, Preparation: "f111111111111111111111111",
		Connection: "conn", SourceCommit: phasedSeedBase,
		Prompt: prompt, PromptSHA: dispatchDigest("phased investigate prompt"),
		Run: factory.NewID(), RunHistory: []string{}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: now.Unix(),
	}
	authority, err := fx.coord.EffectiveAuthority(ctx, owner.Repository)
	if err != nil || !authority.Effective {
		t.Fatalf("fixture authority: %+v %v", authority, err)
	}
	selection := policy.Roles[owner.Role]
	owner.Harness, owner.HarnessVers, owner.Model = selection.Harness+"-"+selection.HarnessVers, selection.HarnessVers, selection.Model
	owner.Authority = authority.Authority
	owner.Authority.RequirementsID, err = fx.db.RequirementHead(ctx, owner.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	owner.Authority.ApprovalID, err = fx.db.ApprovalHead(ctx, owner.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	fx.reads.evidence[strconv.FormatInt(owner.Repository, 10)+"/"+strconv.FormatInt(issue, 10)] = AcceptanceEvidence{
		Revision: 9,
		Issue:    AcceptanceIssueView{Index: decision.IssueIndex, TitleDigest: decision.TitleDigest, TitleOccurrence: decision.TitleOccurrence, LifecycleOccurrence: factory.OriginalOccurrence, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: decision.Sources[0].ID, Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	owner.RunHistory = []string{owner.Run}
	reservation := factory.Reservation{AssignmentID: owner.ID, Repository: fx.seed.repo, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 30}
	run := factory.Run{
		ID: owner.Run, ProjectID: owner.ProjectID, Role: project.RoleCoder, InputSHA: phasedSeedBase,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: owner.Harness, Model: owner.Model,
	}
	view := factory.RunView{RunID: owner.Run, Repository: fx.seed.repo, Issue: issue, Attempt: owner.ID}
	if err := fx.db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: owner.ID, Repository: owner.Repository, Authority: owner.Authority}, dispatchControlForAssignment(t, fx.db, owner), owner, reservation, run, view); err != nil {
		t.Fatalf("investigate root refused: %v", err)
	}
	plan := phasedSeedPlan(t, owner)
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "plan ready", true
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := fx.db.ConsumeReservation(ctx, owner.ID); err != nil {
		t.Fatal(err)
	}
	phasedResult := factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	}
	owner.Stage, owner.Outcome, owner.Reason, owner.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, now.Unix()
	owner.Result = &factory.AssignmentResult{
		AssignmentID: owner.ID, RunID: owner.Run, Phase: project.ExecutionInvestigate,
		PhaseResult: &phasedResult, Status: "completed", Summary: "plan ready",
		Findings: []string{}, Reported: true, RecordedUnix: now.Unix(),
	}
	if err := fx.db.FinishAssignment(ctx, owner); err != nil {
		t.Fatalf("investigate finish refused: %v", err)
	}
	storedOwner, err := fx.db.Assignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fx.db.ReplacePlanHead(ctx, plan, ""); err != nil {
		t.Fatalf("plan head unrecorded: %v", err)
	}

	childID, err := factory.AdvancementID(storedOwner.ID, factory.ResultDigest(*storedOwner.Result), project.ExecutionImplement)
	if err != nil {
		t.Fatal(err)
	}
	childPrompt := []byte("phased implement prompt")
	childRunID := factory.NewID()
	child := factory.Assignment{
		Authority: storedOwner.Authority,
		ID:        childID, AttemptRoot: storedOwner.AttemptRoot, ParentAssignment: storedOwner.ID,
		PublicationAssignment: storedOwner.PublicationAssignment,
		Subject:               storedOwner.Subject, Phase: project.ExecutionImplement,
		ProjectID: storedOwner.ProjectID, Role: project.RoleCoder,
		Repository: storedOwner.Repository, Issue: storedOwner.Issue, NativeRev: storedOwner.NativeRev,
		Acceptance: storedOwner.Acceptance, Preparation: storedOwner.Preparation,
		Harness: storedOwner.Harness, HarnessVers: storedOwner.HarnessVers, Model: storedOwner.Model,
		Connection: storedOwner.Connection, SourceCommit: storedOwner.SourceCommit,
		Prompt: childPrompt, PromptSHA: dispatchDigest("phased implement prompt"),
		Run: childRunID, RunHistory: []string{childRunID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(), ActorID: storedOwner.ActorID,
	}
	childReservation := factory.Reservation{AssignmentID: child.ID, Repository: child.Repository, Connection: child.Connection, State: factory.ReservationHeld, PlannedMinutes: 30}
	childRun := factory.Run{
		ID: childRunID, ProjectID: child.ProjectID, Role: project.RoleCoder, InputSHA: child.SourceCommit,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: child.Harness, Model: child.Model,
	}
	childView := factory.RunView{RunID: childRunID, Repository: child.Repository, Issue: child.Issue, Attempt: child.ID}
	if err := fx.db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: child.ID, Repository: child.Repository, Authority: child.Authority}, dispatchControlForAssignment(t, fx.db, child), child, childReservation, childRun, childView); err != nil {
		t.Fatalf("implement child refused: %v", err)
	}
	childRun.Outcome, childRun.Summary, childRun.Reconciled = factory.Succeeded, "implemented", true
	if err := fx.db.SaveFactoryRun(ctx, childRun); err != nil {
		t.Fatal(err)
	}
	if err := fx.db.ConsumeReservation(ctx, child.ID); err != nil {
		t.Fatal(err)
	}
	finishedChild, err := fx.db.Assignment(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	finishedChild.Stage, finishedChild.Outcome, finishedChild.Reason, finishedChild.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, now.Unix()
	finishedChild.Result = &factory.AssignmentResult{
		AssignmentID: child.ID, RunID: childRunID, Phase: project.ExecutionImplement,
		PhaseResult: &factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionImplement, Status: "completed",
			Summary: "implemented",
			Output:  &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: phasedSeedCandidate}},
		},
		Status: "completed", Summary: "implemented", Candidate: phasedSeedCandidate,
		Findings: []string{}, Reported: true, RecordedUnix: now.Unix(),
	}
	if err := fx.db.FinishAssignment(ctx, finishedChild); err != nil {
		t.Fatalf("implement finish refused: %v", err)
	}
	pending := store.PlanTaskReceipt{AttemptRoot: storedOwner.AttemptRoot, PlanID: plan.ID, TaskID: "task-one", State: store.PlanTaskPending}
	if err := fx.db.SavePlanTaskReceipt(ctx, pending); err != nil {
		t.Fatalf("pending receipt unrecorded: %v", err)
	}
	running := pending
	running.Revision, running.State, running.Child = 0, store.PlanTaskRunning, "child-task-one"
	if err := fx.db.SavePlanTaskReceipt(ctx, running); err != nil {
		t.Fatalf("running receipt unrecorded: %v", err)
	}
	ready := running
	ready.Revision, ready.State, ready.OutputDigest = 1, store.PlanTaskOutputReady, dispatchDigest("task output")
	if err := fx.db.SavePlanTaskReceipt(ctx, ready); err != nil {
		t.Fatalf("ready receipt unrecorded: %v", err)
	}
	completed := ready
	completed.Revision, completed.State, completed.Commit, completed.Tree = 2, store.PlanTaskCompleted, phasedSeedCandidate, phasedSeedCandidate
	completed.Reason = "planned task finished with its checkpoint commit"
	if err := fx.db.SavePlanTaskReceipt(ctx, completed); err != nil {
		t.Fatalf("completed receipt unrecorded: %v", err)
	}

	fx.wire(happyPublisher())
	if report := fx.coord.PublishPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("phased publication pass: %+v", report)
	}
	p, err := fx.db.PublicationByAssignment(ctx, storedOwner.ID)
	if err != nil || p.Stage != factory.PublicationPublished {
		t.Fatalf("phased publication not linked: %+v %v", p, err)
	}
	return fx, host, storedOwner, p, plan
}
