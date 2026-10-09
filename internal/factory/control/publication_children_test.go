package control

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type producerFlowHost struct {
	*fakePublishHost
	launches       []project.FactoryLaunch
	preparations   []project.FactoryCandidate
	prepInspects   []project.PrepareInspect
	prepPhase      string
	stopRetirement string
	prepStops      []project.PrepareStop
	contextReads   []project.FactoryPreparationContextRequest
	contextRoles   []string
}

func (h *producerFlowHost) FactoryLaunch(_ context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	h.launches = append(h.launches, in)
	return project.FactoryState{
		ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role,
		Phase: project.FactoryCompleted, Container: strings.Repeat("c", 64),
	}, nil
}

func (h *producerFlowHost) ReadPreparationContext(_ context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
	h.contextReads = append(h.contextReads, in)
	h.contextRoles = append(h.contextRoles, role)
	return fixturePreparationContext(in, role), nil
}

func (*producerFlowHost) FactoryHarness(_ context.Context, family string) (project.FactoryHarnessPin, error) {
	return project.FactoryHarnessPin{Harness: family, Version: "1.2.3", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
}

func (h *producerFlowHost) PrepareCandidate(_ context.Context, in project.FactoryCandidate) (project.PrepareState, error) {
	h.preparations = append(h.preparations, in)
	phase := h.prepPhase
	if phase == "" {
		phase = project.PrepareReady
	}
	return project.PrepareState{
		ID: in.Preparation.ID, Project: in.Preparation.Project, Role: in.Preparation.Role,
		Phase: phase, Container: strings.Repeat("d", 64), SourceCommit: in.Preparation.SourceCommit,
		SetupDigest: in.Preparation.SetupDigest, Ready: phase == project.PrepareReady,
	}, nil
}

func (h *producerFlowHost) InspectPreparation(_ context.Context, in project.PrepareInspect) (project.PrepareState, error) {
	h.prepInspects = append(h.prepInspects, in)
	return project.PrepareState{}, project.ErrPreparationNotFound
}

func (h *producerFlowHost) StopPreparation(_ context.Context, in project.PrepareStop) (project.PrepareState, error) {
	h.prepStops = append(h.prepStops, in)
	retirement := h.stopRetirement
	if retirement == "" {
		retirement = "confirmed"
	}
	return project.PrepareState{
		ID: in.ID, Project: in.Project, Phase: project.PrepareStopped,
		Stopped: true, Retirement: retirement,
	}, nil
}

func publishedChildProducerSeed(t *testing.T, issue int64, preparationPhase ...string) (*publishFixture, *producerFlowHost, factory.Assignment, factory.Publication) {
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
	if len(preparationPhase) > 0 {
		host.prepPhase = preparationPhase[0]
	}
	fx.coord.Host = host
	fx.coord.DispatchReads = fx.seed.reads
	fx.coord.Reviews = &fakeReviewer{}
	owner := fx.finishReported(t, issue)
	fx.wire(happyPublisher())
	if report := fx.coord.PublishPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("initial production publication pass: %+v", report)
	}
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || p.Stage != factory.PublicationPublished {
		t.Fatalf("publication not linked: %+v %v", p, err)
	}
	return fx, host, owner, p
}

func settleProducedChild(t *testing.T, fx *publishFixture, runID, output string) (factory.Assignment, factory.Run) {
	t.Helper()
	ctx := context.Background()
	run, err := fx.db.FactoryRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "child result reconciled", true
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	assignment, accounted := AccountSettledRun(ctx, fx.db, run, output, time.Now())
	if !accounted || assignment.Stage != factory.AssignmentFinished {
		t.Fatalf("produced child result was not accounted: %+v", assignment)
	}
	return assignment, run
}

func TestPublishPassProducesExactReviewerChildAndReplays(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication := publishedChildProducerSeed(t, 23, project.PrepareRunning)
	admissions, admissionErr := fx.db.CandidatePreparations(ctx, owner.Repository, owner.ProjectID, owner.Issue, owner.AttemptRoot, "", 64, 0)
	if admissionErr != nil || len(admissions) != 1 {
		t.Fatalf("running review preparation admission missing: %+v %v", admissions, admissionErr)
	}
	changed := admissions[0].Admission
	changed.ActorID++
	if _, _, err := fx.db.AdmitCandidatePreparation(ctx, project.StoredPreparation{Preparation: admissions[0].Preparation.Preparation}, changed); !errors.Is(err, store.ErrAdmissionChanged) {
		t.Fatalf("candidate preparation replay changed execution actor: %v", err)
	}
	host.prepPhase = project.PrepareReady
	first := fx.coord.PublishPass(ctx)
	if len(first.Errors) != 0 || len(host.launches) != 1 || len(host.preparations) != 2 {
		t.Fatalf("review child production: report=%+v launches=%d preparations=%d", first, len(host.launches), len(host.preparations))
	}
	launch := host.launches[0]
	child, err := fx.db.AssignmentByRun(ctx, launch.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	prep := host.preparations[len(host.preparations)-1]
	if len(host.contextReads) != 1 || len(host.contextRoles) != 1 {
		t.Fatalf("review context admission reads=%+v roles=%+v", host.contextReads, host.contextRoles)
	}
	contextRead := host.contextReads[0]
	if host.contextRoles[0] != project.RoleReviewer || contextRead.Project != owner.ProjectID ||
		contextRead.ID != prep.Preparation.ID || contextRead.SourceCommit != publication.Candidate ||
		contextRead.ApprovedBase != owner.SourceCommit || contextRead.DiffBase != publication.PRCreate.BaseOID ||
		contextRead.Candidate != publication.Candidate || !contextRead.NotAfter.Equal(launch.Run.Deadline) ||
		!strings.Contains(string(child.Prompt), "synthetic approved setup instructions") ||
		!strings.Contains(string(child.Prompt), "synthetic selected source: README.md") ||
		!strings.Contains(string(child.Prompt), "synthetic exact candidate diff") {
		t.Fatalf("review context lost exact candidate/base/role/deadline or prompt bytes: request=%+v role=%q", contextRead, host.contextRoles[0])
	}
	sourcePrep, err := fx.db.Preparation(ctx, prep.SourcePreparation)
	if err != nil {
		t.Fatal(err)
	}
	if child.Role != project.RoleReviewer || child.PublicationAssignment != owner.ID || child.AttemptRoot != owner.AttemptRoot ||
		child.ActorID != owner.ActorID || launch.Run.Actor != child.ActorID ||
		child.SourceCommit != publication.Candidate || launch.Run.SourceCommit != publication.Candidate || launch.Run.Preparation != child.Preparation ||
		prep.Preparation.ID != publicationPreparationID(publication.ID, publication.Candidate) ||
		prep.Preparation.Role != project.RoleReviewer || prep.Preparation.SourceCommit != publication.Candidate ||
		sourcePrep.Preparation.Role != project.RoleReviewer || sourcePrep.Preparation.Requirements.ID != child.Authority.RequirementsID ||
		sourcePrep.Preparation.Approval.ID != child.Authority.ApprovalID || !sourcePrep.State.Ready ||
		!strings.Contains(string(child.Prompt), "Approved repository base: "+owner.SourceCommit) ||
		!strings.Contains(string(child.Prompt), "Candidate: "+publication.Candidate+" verified base: "+publication.PRCreate.BaseOID) {
		t.Fatalf("review child lost exact publication linkage: assignment=%+v launch=%+v prep=%+v", child, launch.Run, prep)
	}
	second := fx.coord.PublishPass(ctx)
	if len(second.Errors) != 0 || len(host.launches) != 1 || len(host.preparations) != 2 {
		t.Fatalf("review child replay repeated preparation or launch: report=%+v launches=%d preparations=%d", second, len(host.launches), len(host.preparations))
	}
}

func TestPublishPassProducesCorrectionThenFreshReviewForChangedHead(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, publication := publishedChildProducerSeed(t, 24)
	if len(host.launches) != 1 {
		t.Fatalf("initial reviewer launch count = %d", len(host.launches))
	}
	reviewerLaunch := host.launches[0]
	reviewOutput := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"empty input panics\",\"findings\":[\"empty input\"]}\n```"
	_, reviewerRun := settleProducedChild(t, fx, reviewerLaunch.Run.ID, reviewOutput)
	reviewer := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":31}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 31, CommentID: 32, ReviewerID: 6, PRID: publication.PRID,
			PRNumber: publication.PRNumber, IssueID: publication.PRCreate.IssueID,
			HeadOID: publication.Candidate, BaseOID: publication.PRCreate.BaseOID,
			CommitID: publication.Candidate, Event: "REQUEST_CHANGES",
		},
	}
	fx.coord.Reviews = reviewer
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
	checkReport := fx.coord.CheckPass(ctx)
	if len(checkReport.Errors) != 0 || len(host.launches) != 2 {
		t.Fatalf("failed CI plus request-changes did not produce one correction: report=%+v launches=%d", checkReport, len(host.launches))
	}
	correctionLaunch := host.launches[1]
	if len(host.contextReads) != 2 || len(host.contextRoles) != 2 {
		t.Fatalf("correction did not read bounded repository context: reads=%+v roles=%+v", host.contextReads, host.contextRoles)
	}
	correctionContext := host.contextReads[1]
	if host.contextRoles[1] != project.RoleCoder || correctionContext.ID != owner.Preparation ||
		correctionContext.SourceCommit != owner.SourceCommit || correctionContext.ApprovedBase != owner.SourceCommit ||
		correctionContext.DiffBase != publication.PRCreate.BaseOID || correctionContext.Candidate != publication.Candidate ||
		!correctionContext.NotAfter.Equal(correctionLaunch.Run.Deadline) {
		t.Fatalf("correction context changed base, role, candidate or deadline: request=%+v role=%q run=%+v", correctionContext, host.contextRoles[1], correctionLaunch.Run)
	}
	correction, err := fx.db.AssignmentByRun(ctx, correctionLaunch.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Recorded check evidence", "## Independent review findings", "empty input panics", "verdict: failed"} {
		if !strings.Contains(string(correction.Prompt), want) {
			t.Errorf("correction prompt lacks %q", want)
		}
	}
	if correction.Role != project.RoleCoder || correction.PublicationAssignment != owner.ID || correction.AttemptRoot != owner.AttemptRoot ||
		correction.SourceCommit != publication.Candidate {
		t.Fatalf("correction child lost parent attempt identity: %+v", correction)
	}

	newHead := strings.Repeat("d", 40)
	_, correctionRun := settleProducedChild(t, fx, correction.Run, correctionOutput(newHead))
	corrected := fx.coord.PublishCorrection(ctx, correctionRun.ID)
	if len(corrected.Errors) != 0 || len(corrected.Corrected) != 1 || corrected.Corrected[0].HeadOID != newHead {
		t.Fatalf("correction did not advance the publication head: %+v", corrected)
	}
	if report := fx.coord.PublishPass(ctx); len(report.Errors) != 0 || len(host.launches) != 3 || len(host.preparations) != 2 {
		t.Fatalf("changed head did not receive a fresh reviewer: report=%+v launches=%d preparations=%d", report, len(host.launches), len(host.preparations))
	}
	newReview, err := fx.db.AssignmentByRun(ctx, host.launches[2].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newReview.Role != project.RoleReviewer || newReview.PublicationAssignment != owner.ID || newReview.AttemptRoot != owner.AttemptRoot ||
		newReview.SourceCommit != newHead || strings.Contains(string(newReview.Prompt), "empty input panics") {
		t.Fatalf("new-head review reused stale evidence or changed its root: %+v", newReview)
	}
	if len(host.contextReads) != 3 || len(host.contextRoles) != 3 {
		t.Fatalf("changed head did not read fresh context: reads=%+v roles=%+v", host.contextReads, host.contextRoles)
	}
	newReviewContext := host.contextReads[2]
	if host.contextRoles[2] != project.RoleReviewer || newReviewContext.ID != host.preparations[1].Preparation.ID ||
		newReviewContext.ApprovedBase != owner.SourceCommit || newReviewContext.DiffBase != publication.PRCreate.BaseOID ||
		newReviewContext.Candidate != newHead || !newReviewContext.NotAfter.Equal(host.launches[2].Run.Deadline) {
		t.Fatalf("fresh review context lost changed head/base/deadline: request=%+v role=%q", newReviewContext, host.contextRoles[2])
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || allowance.RootAssignment != owner.AttemptRoot || !slices.Contains(allowance.Corrections, correction.ID) {
		t.Fatalf("correction was not charged to the owner's cumulative allowance: %+v %v", allowance, err)
	}
}

func TestPublishPassFencesAfterThreeCorrectionCycles(t *testing.T) {
	ctx := context.Background()
	fx, host, owner, _ := publishedChildProducerSeed(t, 28)
	checks := happyCheckObserver()
	checks.observe = func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, err := happyCheckObserver().observe(target)
		observed.Checks = []factory.ObservedCheck{{Context: "ci", State: "failure"}}
		return observed, err
	}
	fx.coord.Checks = checks
	requestChanges := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"cycle finding\",\"findings\":[\"cycle finding\"]}\n```"
	correctionIDs := make([]string, 0, factory.DefaultAttemptLimits().CorrectionCycles)
	priorActiveSeconds := int64(0)

	for cycle := 0; cycle < factory.DefaultAttemptLimits().CorrectionCycles; cycle++ {
		current, err := fx.db.PublicationByAssignment(ctx, owner.ID)
		if err != nil || current.Stage != factory.PublicationPublished {
			t.Fatalf("cycle %d publication = %+v, %v", cycle+1, current, err)
		}
		reviewerLaunch := host.launches[len(host.launches)-1]
		_, reviewerRun := settleProducedChild(t, fx, reviewerLaunch.Run.ID, requestChanges)
		fx.coord.Reviews = &fakeReviewer{
			observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
			outcome:  committedOutcome("review-receipt-" + string(rune('1'+cycle))),
			adopted: factory.ReviewOutcome{
				ReviewID: int64(31 + cycle), CommentID: int64(41 + cycle), ReviewerID: 6, PRID: current.PRID,
				PRNumber: current.PRNumber, IssueID: current.PRCreate.IssueID,
				HeadOID: current.Candidate, BaseOID: current.PRCreate.BaseOID,
				CommitID: current.Candidate, Event: "REQUEST_CHANGES",
			},
		}
		if _, err := fx.coord.SubmitReviewForRun(ctx, reviewerRun.ID); err != nil {
			t.Fatalf("cycle %d review submission: %v", cycle+1, err)
		}

		launchesBefore := len(host.launches)
		checkReport := fx.coord.CheckPass(ctx)
		if len(checkReport.Errors) != 0 || len(host.launches) != launchesBefore+1 {
			t.Fatalf("cycle %d did not admit exactly one correction: report=%+v launches=%d before=%d", cycle+1, checkReport, len(host.launches), launchesBefore)
		}
		correctionLaunch := host.launches[len(host.launches)-1]
		correction, err := fx.db.AssignmentByRun(ctx, correctionLaunch.Run.ID)
		if err != nil || correction.Role != project.RoleCoder || correction.AttemptRoot != owner.AttemptRoot ||
			correction.PublicationAssignment != owner.ID || correction.SourceCommit != current.Candidate {
			t.Fatalf("cycle %d correction identity = %+v, %v", cycle+1, correction, err)
		}
		correctionIDs = append(correctionIDs, correction.ID)
		newHead := strings.Repeat(string(rune('d'+cycle)), 40)
		_, correctionRun := settleProducedChild(t, fx, correction.Run, correctionOutput(newHead))
		corrected := fx.coord.PublishCorrection(ctx, correctionRun.ID)
		if len(corrected.Errors) != 0 || len(corrected.Corrected) != 1 || corrected.Corrected[0].HeadOID != newHead {
			t.Fatalf("cycle %d correction receipt = %+v", cycle+1, corrected)
		}

		allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
		if err != nil || allowance.RootAssignment != owner.AttemptRoot || allowance.Limits != factory.DefaultAttemptLimits() ||
			!allowance.Active || allowance.Closed || len(allowance.Corrections) != cycle+1 ||
			!slices.Equal(allowance.Corrections, correctionIDs) || allowance.ActiveSeconds < priorActiveSeconds {
			t.Fatalf("cycle %d allowance changed root/time/cycle custody: %+v, %v", cycle+1, allowance, err)
		}
		priorActiveSeconds = allowance.ActiveSeconds
		pass := fx.coord.PublishPass(ctx)
		if len(pass.Errors) != 0 || len(host.launches) != launchesBefore+2 {
			t.Fatalf("cycle %d changed head did not get one reviewer: report=%+v launches=%d", cycle+1, pass, len(host.launches))
		}
	}

	current, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || current.Candidate != strings.Repeat("f", 40) || len(host.launches) != 7 || len(host.preparations) != 4 {
		t.Fatalf("three-cycle pre-exhaustion state: publication=%+v launches=%d preparations=%d err=%v", current, len(host.launches), len(host.preparations), err)
	}
	finalReviewerLaunch := host.launches[len(host.launches)-1]
	_, finalReviewerRun := settleProducedChild(t, fx, finalReviewerLaunch.Run.ID, requestChanges)
	fx.coord.Reviews = &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome("review-receipt-exhausted"),
		adopted: factory.ReviewOutcome{
			ReviewID: 34, CommentID: 44, ReviewerID: 6, PRID: current.PRID,
			PRNumber: current.PRNumber, IssueID: current.PRCreate.IssueID,
			HeadOID: current.Candidate, BaseOID: current.PRCreate.BaseOID,
			CommitID: current.Candidate, Event: "REQUEST_CHANGES",
		},
	}
	if _, err := fx.coord.SubmitReviewForRun(ctx, finalReviewerRun.ID); err != nil {
		t.Fatalf("exhausted-head review submission: %v", err)
	}
	exhaustedBefore, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || exhaustedBefore.RootAssignment != owner.AttemptRoot || exhaustedBefore.Limits != factory.DefaultAttemptLimits() ||
		!exhaustedBefore.Active || exhaustedBefore.Closed || len(exhaustedBefore.Corrections) != 3 ||
		!slices.Equal(exhaustedBefore.Corrections, correctionIDs) || exhaustedBefore.ActiveSeconds < priorActiveSeconds {
		t.Fatalf("pre-exhaustion allowance = %+v, %v", exhaustedBefore, err)
	}
	runsBefore, err := fx.db.ProjectFactoryRuns(ctx, owner.ProjectID, 100)
	if err != nil {
		t.Fatal(err)
	}
	launchesBefore, preparationsBefore := len(host.launches), len(host.preparations)
	exhaustedPass := fx.coord.CheckPass(ctx)
	if len(exhaustedPass.Errors) != 0 || len(host.launches) != launchesBefore || len(host.preparations) != preparationsBefore {
		t.Fatalf("exhausted correction created child work: report=%+v launches=%d preparations=%d", exhaustedPass, len(host.launches), len(host.preparations))
	}
	exhaustedChildID := publicationChildID(current.ID, current.Candidate, project.RoleCoder)
	if _, err := fx.db.Assignment(ctx, exhaustedChildID); !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("exhausted correction assignment exists: %v", err)
	}
	runsAfter, err := fx.db.ProjectFactoryRuns(ctx, owner.ProjectID, 100)
	if err != nil || len(runsAfter) != len(runsBefore) {
		t.Fatalf("exhausted correction run was recorded: before=%d after=%d err=%v", len(runsBefore), len(runsAfter), err)
	}
	fenced, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || fenced.Stage != factory.PublicationFenced || fenced.Outcome != factory.NeedsHuman {
		t.Fatalf("exhaustion did not request maintainer intervention: publication=%+v err=%v", fenced, err)
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || allowance.RootAssignment != owner.AttemptRoot || allowance.Limits != factory.DefaultAttemptLimits() ||
		allowance.Active || !allowance.Closed || len(allowance.Corrections) != 3 ||
		!slices.Equal(allowance.Corrections, correctionIDs) || allowance.CheckpointUnix < exhaustedBefore.CheckpointUnix ||
		allowance.ActiveSeconds != exhaustedBefore.ActiveSeconds+allowance.CheckpointUnix-exhaustedBefore.CheckpointUnix {
		t.Fatalf("exhaustion did not close the settled root while accounting for elapsed time: before=%+v after=%+v err=%v", exhaustedBefore, allowance, err)
	}
}

func TestPauseKeepsRunningCandidatePreparationUntilStopReceiptConfirms(t *testing.T) {
	for _, tc := range []struct {
		name       string
		retirement string
		failed     bool
	}{
		{name: "uncertain", retirement: "uncertain"},
		{name: "confirmed", retirement: "confirmed"},
		{name: "failed-uncertain", retirement: "uncertain", failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			retirement := tc.retirement
			ctx := context.Background()
			fx, host, owner, publication := publishedChildProducerSeed(t, 25, project.PrepareRunning)
			if len(host.preparations) != 1 || len(host.launches) != 0 {
				t.Fatalf("running preparation should remain pending before pause: preparations=%d launches=%d", len(host.preparations), len(host.launches))
			}
			prep := host.preparations[0]
			wantPreparationID := publicationPreparationID(publication.ID, publication.Candidate)
			if prep.Preparation.ID != wantPreparationID {
				t.Fatalf("candidate preparation ID = %q, want deterministic %q", prep.Preparation.ID, wantPreparationID)
			}
			stored, err := fx.db.Preparation(ctx, wantPreparationID)
			if err != nil || stored.State.Phase != project.PrepareRunning || stored.State.Ready {
				t.Fatalf("running native receipt not persisted: %+v %v", stored, err)
			}
			pendingRecords, err := fx.db.CandidatePreparations(ctx, owner.Repository, owner.ProjectID, owner.Issue, owner.AttemptRoot, "", 64, 0)
			if err != nil || len(pendingRecords) != 1 || pendingRecords[0].Preparation.Preparation.ID != wantPreparationID ||
				pendingRecords[0].NotAfter <= 0 {
				t.Fatalf("running preparation is not held as pending: %+v %v", pendingRecords, err)
			}
			originalDeadline := pendingRecords[0].NotAfter
			replayPreparation := pendingRecords[0].Preparation
			replayPreparation.State = project.PrepareState{}
			replayed, created, err := fx.db.AdmitCandidatePreparation(ctx, replayPreparation, pendingRecords[0].Admission)
			if err != nil || created || replayed.NotAfter != originalDeadline ||
				replayed.Admission.Authority != pendingRecords[0].Admission.Authority ||
				replayed.Admission.OwnerAssignment != owner.ID || replayed.Admission.AttemptRoot != owner.AttemptRoot ||
				replayed.Preparation.Preparation.Role != project.RoleReviewer {
				t.Fatalf("candidate preparation replay changed its deadline or owner authority: created=%t replay=%+v err=%v", created, replayed, err)
			}
			if deadline, err := fx.db.AuthorizeCandidatePreparation(ctx, wantPreparationID); err != nil || deadline != originalDeadline {
				t.Fatalf("candidate preparation authorization changed its first deadline: got=%d want=%d err=%v", deadline, originalDeadline, err)
			}
			if tc.failed {
				// A failed native receipt may retain a child or capture owner.
				// Only its subsequent confirmed stop can release that custody.
				stored.State.Phase = project.PrepareFailed
				if err := fx.db.ObservePreparation(ctx, stored); err != nil {
					t.Fatal(err)
				}
				if _, err := fx.db.AuthorizeCandidatePreparation(ctx, wantPreparationID); err == nil {
					t.Fatal("failed preparation authorized another native effect")
				}
			}
			host.stopRetirement = retirement
			_, pauseErr := fx.coord.PauseRepository(ctx, factory.NewID(), "native:7", owner.Repository)
			if len(host.prepStops) != 1 || host.prepStops[0] != (project.PrepareStop{Project: owner.ProjectID, ID: wantPreparationID}) || len(host.launches) != 0 {
				t.Fatalf("pause did not stop the exact pending preparation before launch: stops=%+v launches=%d", host.prepStops, len(host.launches))
			}
			pendingRecords, recordsErr := fx.db.CandidatePreparations(ctx, owner.Repository, owner.ProjectID, owner.Issue, owner.AttemptRoot, "", 64, 0)
			allowance, allowanceErr := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
			if recordsErr != nil || allowanceErr != nil || allowance.RootAssignment != owner.AttemptRoot {
				t.Fatalf("read pending/root custody after pause: records=%+v/%v allowance=%+v/%v", pendingRecords, recordsErr, allowance, allowanceErr)
			}
			if retirement == "uncertain" {
				if !errors.Is(pauseErr, store.ErrCandidatePreparationPending) || len(pendingRecords) != 1 || !allowance.Active ||
					!pendingRecords[0].Stopped || pendingRecords[0].Retirement != "uncertain" {
					t.Fatalf("uncertain preparation stop released its root: records=%+v active=%t pauseErr=%v", pendingRecords, allowance.Active, pauseErr)
				}
			} else if pauseErr != nil || len(pendingRecords) != 0 || allowance.Active || allowance.Closed {
				t.Fatalf("confirmed preparation stop did not finish the pause/freeze: records=%+v allowance=%+v err=%v", pendingRecords, allowance, pauseErr)
			}
			if _, err := fx.db.AuthorizeCandidatePreparation(ctx, wantPreparationID); !errors.Is(err, store.ErrDispatchClosed) {
				t.Fatalf("withdrawn gate authorized a candidate preparation: %v", err)
			}
		})
	}
}
