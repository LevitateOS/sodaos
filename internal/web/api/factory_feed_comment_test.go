package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// Fake Lane B collaborators: deterministic routing, authoring and
// native effects without spending, dispatching or network calls.

type fakeCommentJudge struct {
	judgment CommentJudgment
	err      error
	calls    int
	input    CommentJudgeInput
}

func (f *fakeCommentJudge) JudgeComment(ctx context.Context, input CommentJudgeInput) (CommentJudgment, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return CommentJudgment{}, f.err
	}
	return f.judgment, nil
}

type fakeCommentDrafter struct {
	draft factory.IssueDraft
	err   error
	errs  []error
	calls int
}

func (f *fakeCommentDrafter) DraftCommentIssue(ctx context.Context, comment store.FeedCommentDetail) (factory.IssueDraft, error) {
	f.calls++
	if len(f.errs) != 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		if err != nil {
			return factory.IssueDraft{}, err
		}
		return f.draft, nil
	}
	if f.err != nil {
		return factory.IssueDraft{}, f.err
	}
	return f.draft, nil
}

type fakeCommentCreator struct {
	issues    map[string]int64
	next      int64
	creates   int
	createErr error
}

type feedbackCommentCreator struct {
	fakeCommentCreator
	deliveries int
}

func TestClarificationQuestionAndAnswerResumeTheSameRequest(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "Change that page", IdempotencyKey: "clarification"}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	judgment, err := commentJudgmentFromSelection("clarify_target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = applyCommentJudgment(ctx, s, pipe, judgment); err != nil {
		t.Fatal(err)
	}
	question, err := s.CommentProgress(ctx, id)
	if err != nil || question != commentClarificationQuestions["clarify_target"] {
		t.Fatalf("missing specific question: %q %v", question, err)
	}
	if err = s.RetryFeedComment(ctx, id, "42", 0, "Change that page\n\nClarification: the onboarding page", 1729000001); err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil || detail.Record.Revision != 1 || detail.Body != "Change that page\n\nClarification: the onboarding page" {
		t.Fatalf("answer lost original request: %+v %v", detail, err)
	}
	resumed, err := s.CommentPipeline(ctx, id)
	if err != nil || resumed.Judgment != factory.CommentJudgmentPending {
		t.Fatalf("answer did not resume assessment: %+v %v", resumed, err)
	}
}

func (f *feedbackCommentCreator) DeliverCommentFeedback(_ context.Context, detail store.FeedCommentDetail, issue int64) error {
	if issue != 55 || detail.Body != "Fix the filter" {
		return errors.New("feedback changed source or issue")
	}
	f.deliveries++
	return nil
}

func TestExistingWorkFeedbackActuallyReachesTheSelectedIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "Fix the filter", Issue: 55, IdempotencyKey: "feedback"}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	creator := &feedbackCommentCreator{}
	wired := &CommentCollaborators{Judge: &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteExistingWork, Issue: 55}}, Creator: creator}
	for range 2 {
		response, err := driveCommentPipeline(ctx, s, wired, id)
		if err != nil || response.Effect != factory.CommentEffectApplied {
			t.Fatalf("feedback not delivered: %+v %v", response, err)
		}
	}
	if creator.deliveries != 1 || creator.creates != 0 {
		t.Fatalf("feedback duplicated or created new work: %+v", creator)
	}
}

func (f *fakeCommentCreator) CreateCommentIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.creates++
	f.next++
	if f.issues == nil {
		f.issues = make(map[string]int64)
	}
	f.issues[draft.Comment] = f.next
	return f.next, nil
}

func (f *fakeCommentCreator) LookupCommentIssue(ctx context.Context, repository int64, commentID string) (int64, error) {
	return f.issues[commentID], nil
}

func driveTestStore(t *testing.T) (*store.Store, context.Context) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, _, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return s, context.Background()
}

func saveDriveComment(t *testing.T, s *store.Store, ctx context.Context, body, key string) string {
	t.Helper()
	id, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: body, IdempotencyKey: key}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	return id
}

func validTestDraft(comment string) factory.IssueDraft {
	return factory.IssueDraft{Comment: comment, Title: "Fix login redirect", Body: "Change the callback.", Objective: "Logins land back in the app.", Acceptance: "Redirect returns 302.", Metadata: `{"work":"soda"}`}
}

func TestDriveCommentNewIssueToVerified(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "please fix the login redirect", "key-1")
	judge := &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteNewIssue}}
	drafter := &fakeCommentDrafter{draft: validTestDraft(id)}
	creator := &fakeCommentCreator{next: 100, issues: map[string]int64{}}
	wired := &CommentCollaborators{Judge: judge, Drafter: drafter, Creator: creator}

	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Comment != id || response.Judgment != factory.CommentJudgmentApplied || response.Route != factory.CommentRouteNewIssue ||
		response.Draft != factory.CommentDraftApplied || response.Effect != factory.CommentEffectVerified || response.Issue != 101 {
		t.Fatalf("drove response: %+v", response)
	}
	if judge.calls != 1 || drafter.calls != 1 || creator.creates != 1 {
		t.Fatalf("calls judge=%d draft=%d creates=%d", judge.calls, drafter.calls, creator.creates)
	}
	// A replay recovers the same identity and verified link without any
	// new call, draft or native create.
	replayed, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Comment != id || replayed.Effect != factory.CommentEffectVerified || replayed.Issue != 101 {
		t.Fatalf("replayed response: %+v", replayed)
	}
	if judge.calls != 1 || drafter.calls != 1 || creator.creates != 1 {
		t.Fatalf("replay spent: judge=%d draft=%d creates=%d", judge.calls, drafter.calls, creator.creates)
	}
}

func TestDriveCommentDiscussionCreatesNothing(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "thanks, acknowledged", "key-1")
	judge := &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteDiscussion}}
	creator := &fakeCommentCreator{next: 100, issues: map[string]int64{}}
	wired := &CommentCollaborators{Judge: judge, Drafter: &fakeCommentDrafter{draft: validTestDraft(id)}, Creator: creator}

	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Route != factory.CommentRouteDiscussion || response.Draft != factory.CommentDraftPending || response.Effect != factory.CommentEffectPending || response.Issue != 0 {
		t.Fatalf("discussion response: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("discussion created an issue")
	}
}

func TestDriveCommentMalformedDraftBlocksEffect(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "fix it", "key-1")
	broken := validTestDraft(id)
	broken.Acceptance = ""
	drafter := &fakeCommentDrafter{err: errors.New("source assembly unavailable")}
	wired := &CommentCollaborators{
		Judge:   &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteNewIssue}},
		Drafter: drafter,
		Creator: &fakeCommentCreator{next: 100, issues: map[string]int64{}},
	}
	for range factory.CommentAttemptLimit {
		if _, err := driveCommentPipeline(ctx, s, wired, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RetryFeedComment(ctx, id, "42", 0, "", 1729000050); err != nil {
		t.Fatal(err)
	}
	drafter.err, drafter.draft = nil, broken
	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	// The retry re-routes the request as a new source revision.
	if response.Draft != factory.CommentDraftInvalid || response.Effect != factory.CommentEffectBlocked || response.Revision != 1 {
		t.Fatalf("malformed response: %+v", response)
	}
	if _, err := s.CommentDraft(ctx, id, 1); err == nil {
		t.Fatal("refused draft stays servable")
	}
}

func TestDriveCommentEffectReplyLossReconciles(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "fix it", "key-1")
	// The native issue 101 already exists from a lost reply: the
	// creator knows it, the pipeline does not.
	creator := &fakeCommentCreator{next: 101, issues: map[string]int64{id: 101}}
	wired := &CommentCollaborators{
		Judge:   &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteNewIssue}},
		Drafter: &fakeCommentDrafter{draft: validTestDraft(id)},
		Creator: creator,
	}
	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Effect != factory.CommentEffectVerified || response.Issue != 101 {
		t.Fatalf("reconciled response: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("lost reply duplicated the issue")
	}
}

func TestDriveCommentDeletedSourceSpendsNothing(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "fix it", "key-1")
	judge := &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteNewIssue}}
	wired := &CommentCollaborators{
		Judge:   judge,
		Drafter: &fakeCommentDrafter{draft: validTestDraft(id)},
		Creator: &fakeCommentCreator{next: 100, issues: map[string]int64{}},
	}
	if _, err := s.DeleteFeedComment(ctx, id, "42", 0); err != nil {
		t.Fatal(err)
	}
	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Effect != factory.CommentEffectBlocked {
		t.Fatalf("deleted response: %+v", response)
	}
	if judge.calls != 0 {
		t.Fatal("deleted source spent a judgment call")
	}
}

func TestDriveCommentJudgmentFailureExhaustsHonestly(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "fix it", "key-1")
	wired := &CommentCollaborators{Judge: &fakeCommentJudge{err: errors.New("jev down")}}
	var response factory.CommentResponse
	var err error
	for range factory.CommentAttemptLimit {
		response, err = driveCommentPipeline(ctx, s, wired, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if response.Judgment != factory.CommentJudgmentFailed {
		t.Fatalf("exhausted judgment = %q", response.Judgment)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Attempts != factory.CommentAttemptLimit {
		t.Fatalf("attempts = %d", pipe.Attempts)
	}
}

func TestDriveCommentUnwiredStagesWaitInPending(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDriveComment(t, s, ctx, "fix it", "key-1")
	response, err := driveCommentPipeline(ctx, s, &CommentCollaborators{}, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Intake != factory.CommentIntakeRecorded || response.Judgment != factory.CommentJudgmentPending {
		t.Fatalf("unwired response: %+v", response)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Attempts != 0 {
		t.Fatalf("unwired drive recorded %d attempts", pipe.Attempts)
	}
}

// TestDrivePendingCommentsResumesAfterRequestEnd proves the durable
// continuation: the POST's two durable writes (comment + intake) are
// all that survive the request, and service passes alone carry the
// stored request to its verified issue. A still-settling planner
// keeps the draft pending with attempts intact; the next pass
// finishes it, and the issue appears exactly once in readback.
func TestDrivePendingCommentsResumesAfterRequestEnd(t *testing.T) {
	s, ctx := driveTestStore(t)
	// Exactly what the POST persists before answering promptly.
	id := saveDriveComment(t, s, ctx, "please fix the login redirect", "key-1")
	judge := &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteNewIssue}}
	drafter := &fakeCommentDrafter{draft: validTestDraft(id), errs: []error{ErrCommentDraftPending}}
	creator := &fakeCommentCreator{next: 100, issues: map[string]int64{}}
	wired := &CommentCollaborators{Judge: judge, Drafter: drafter, Creator: creator}

	if driven := drivePendingComments(ctx, s, wired, commentDriveLimit); driven != 1 {
		t.Fatalf("first pass driven %d, want 1", driven)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Judgment != factory.CommentJudgmentApplied || pipe.Route != factory.CommentRouteNewIssue ||
		pipe.Draft != factory.CommentDraftPending || pipe.Attempts != 0 {
		t.Fatalf("pending planner burned state: %+v", pipe)
	}
	if creator.creates != 0 {
		t.Fatal("pending planner created an issue")
	}

	if driven := drivePendingComments(ctx, s, wired, commentDriveLimit); driven != 1 {
		t.Fatalf("second pass driven %d, want 1", driven)
	}
	pipe, err = s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Draft != factory.CommentDraftApplied || pipe.Effect != factory.CommentEffectVerified || pipe.Issue != 101 {
		t.Fatalf("resumed pipeline: %+v", pipe)
	}
	if judge.calls != 1 || creator.creates != 1 {
		t.Fatalf("resume spent twice: judge=%d creates=%d", judge.calls, creator.creates)
	}
	if status := store.PostStatus(&pipe); status != "issue:101" {
		t.Fatalf("readback status %q, want issue:101", status)
	}
	// Settled requests leave the service work list entirely.
	if driven := drivePendingComments(ctx, s, wired, commentDriveLimit); driven != 0 {
		t.Fatalf("settled request drove %d more times", driven)
	}
	if creator.creates != 1 {
		t.Fatal("settled request created a duplicate issue")
	}
}

// TestDriveCommentExistingWorkPersistsSelectedIssue proves an
// existing_work judgment persists WHICH tracked work it selected,
// the judge saw the candidate it picked, and a receipt recorded
// before a crash replays without a second judgment call.
func TestDriveCommentExistingWorkPersistsSelectedIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{
		Repository: 7, Body: "this belongs with the filter work", Issue: 55,
		IdempotencyKey: "key-existing",
	}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	judge := &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteExistingWork, Issue: 55}}
	creator := &fakeCommentCreator{next: 100, issues: map[string]int64{}}
	wired := &CommentCollaborators{Judge: judge, Drafter: &fakeCommentDrafter{draft: validTestDraft(id)}, Creator: creator}

	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Route != factory.CommentRouteExistingWork || response.Issue != 55 || response.ExistingIssue != 55 {
		t.Fatalf("existing work response lost its selection: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("existing work created an issue")
	}
	found := false
	for _, candidate := range judge.input.Candidates {
		if candidate.Issue == 55 && candidate.Origin == factory.CommentCandidateBound {
			found = true
		}
	}
	if !found {
		t.Fatalf("judge never saw bound candidate 55: %+v", judge.input.Candidates)
	}
}

// TestCommentJudgeChoiceHelpers pins the candidate-bound choice
// contract: existing-work keys exist only for listed candidates,
// selections outside the offered set refuse, and the judge state
// carries source, parent and candidate evidence.
func TestCommentJudgeChoiceHelpers(t *testing.T) {
	candidates := []factory.CommentWorkCandidate{
		{Issue: 55, Origin: factory.CommentCandidateBound, Summary: "filter work"},
		{Issue: 9, Origin: factory.CommentCandidateRecent},
	}
	keys := commentChoiceKeys(candidates)
	want := []string{"new_issue", "discussion", "clarify_target", "clarify_result", "clarify_conflict", "existing_55", "existing_9"}
	if len(keys) != len(want) {
		t.Fatalf("keys %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys %v, want %v", keys, want)
		}
	}
	if keys := commentChoiceKeys(nil); len(keys) != 5 {
		t.Fatalf("candidate-less keys %v must exclude existing work", keys)
	}
	judgment, err := commentJudgmentFromSelection("existing_9", candidates)
	if err != nil || judgment.Route != factory.CommentRouteExistingWork || judgment.Issue != 9 {
		t.Fatalf("selection parse: %+v err=%v", judgment, err)
	}
	if _, err := commentJudgmentFromSelection("existing_404", candidates); err == nil {
		t.Fatal("unlisted existing work admitted")
	}
	if _, err := commentJudgmentFromSelection("delete_everything", candidates); err == nil {
		t.Fatal("unknown selection admitted")
	}
	input := CommentJudgeInput{
		Comment: store.FeedCommentDetail{
			Record: factory.CommentRecord{Repository: 7, Issue: 55, ReplyTo: "parent"},
			Body:   "more filter",
			Source: &factory.CommentSource{Kind: factory.CommentSourceRecord, ID: "rec", Issue: 55, Summary: "built filter"},
		},
		Parent:     &CommentParentEvidence{Comment: "parent", Body: "start filter", Issue: 55},
		Candidates: candidates,
	}
	state := commentJudgeState(input)
	source, ok := state["source"].(map[string]any)
	if !ok || source["summary"] != "built filter" || source["issue"] != int64(55) {
		t.Fatalf("state source missing evidence: %+v", state["source"])
	}
	parent, ok := state["parent"].(map[string]any)
	if !ok || parent["body"] != "start filter" {
		t.Fatalf("state parent missing evidence: %+v", state["parent"])
	}
	listed, ok := state["candidates"].([]map[string]any)
	if !ok || len(listed) != 2 || listed[0]["summary"] != "filter work" {
		t.Fatalf("state candidates missing evidence: %+v", state["candidates"])
	}
}

// TestRecordInvitationThreadReachesOriginalAssignment proves the FUX14
// API-level leg: an eligible role is invited on a real finished assignment
// record (never a separate authored post), and its worker reply lands in the
// original record thread behind the existing ResolveCommentSource and
// FeedReplyIDs owners. JEV threshold, per-revision idempotency and the
// receipt/consultation/reply loop exclusion are preserved.
func TestRecordInvitationThreadReachesOriginalAssignment(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	s, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	now := time.Now().Unix()
	digest := strings.Repeat("a", 64)
	capability := factory.MCPCapability{
		ID: factory.NewID(), AssignmentID: factory.NewID(), RunID: "run-1", Generation: 1,
		Profile: factory.MCPProfileInspector, Subject: "review:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityStaged,
	}
	eligible := factory.Assignment{
		ID: capability.AssignmentID, AttemptRoot: capability.AssignmentID,
		PublicationAssignment: capability.AssignmentID, ProjectID: capability.Project,
		Role: "reviewer", Acceptance: "d" + strings.Repeat("a", 24),
		Preparation: "f" + strings.Repeat("b", 24),
		Run:         capability.RunID, RunHistory: []string{capability.RunID},
		Repository: 7, Issue: 42, CreatedUnix: now,
		AcceptedContext: &factory.AcceptedContext{Title: "Equipment approval policy", Body: "Implement protected manager and Finance approvals without self-approval."},
	}
	eligibleRaw, err := json.Marshal(eligible)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,42,$2,'assigned',1,$3)`,
		eligible.ID, eligible.Run, string(eligibleRaw)); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,'{}')`, capability.RunID); err != nil {
		t.Fatal(err)
	}
	if err := s.StageMCPCapability(ctx, capability); err != nil {
		t.Fatal(err)
	}
	if err := s.ActivateMCPCapability(ctx, capability.ID, capability.Generation, capability.RunID, capability.AssignmentID, capability.Authority, time.Now().Unix(), false); err != nil {
		t.Fatal(err)
	}
	capability, err = s.GetMCPCapability(ctx, capability.ID)
	if err != nil {
		t.Fatal(err)
	}
	sourceID := factory.NewID()
	source := factory.Assignment{
		ID: sourceID, AttemptRoot: sourceID, PublicationAssignment: sourceID, ProjectID: capability.Project,
		Role: "coder", Phase: "implement", Acceptance: "d" + strings.Repeat("a", 24),
		Preparation: "f" + strings.Repeat("b", 24),
		Run:         "run-" + sourceID[:8], RunHistory: []string{"run-" + sourceID[:8]},
		Repository: 7, Issue: 42, CreatedUnix: now, FinishedUnix: now,
	}
	result := factory.ResultSynthesized(sourceID, source.Run, "succeeded", "Implemented the delivery guard without self-approval; candidate abc123.", now)
	source.Result = &result
	sourceRaw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,42,$2,'finished',1,$3)`,
		sourceID, source.Run, string(sourceRaw)); err != nil {
		t.Fatal(err)
	}
	sources, err := s.ContributionSources(ctx)
	if err != nil || len(sources) != 1 || sources[0] != sourceID {
		t.Fatalf("record undiscovered via API owners: %v %v", sources, err)
	}
	review, err := s.ContributionContext(ctx, sourceID)
	if err != nil || review.SourceKind != factory.CommentSourceRecord || len(review.Roles) != 1 {
		t.Fatalf("record context via API owners: %+v %v", review, err)
	}
	if claimed, err := s.ClaimContributionReview(ctx, review); err != nil || !claimed {
		t.Fatalf("record claim: %v %v", claimed, err)
	}
	for _, question := range factory.ContributionQuestions {
		review.Calls = append(review.Calls, factory.ContributionCall{Assignment: capability.AssignmentID, Question: question, Answer: factory.PriorityAnswer{Model: "fixture-jev", Selected: "contribute", Probabilities: map[string]float64{"contribute": .95, "no_contribution": .05}, Confidence: .9}})
	}
	factory.SelectContribution(&review)
	if review.State != "invited" {
		t.Fatalf("record not invited: %+v", review)
	}
	if err := s.SettleContributionReview(ctx, review); err != nil {
		t.Fatal(err)
	}
	invitation := factory.ContributionReviewInvitationID(review)
	resolved, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourceRecord, sourceID)
	if err != nil || resolved.ID != sourceID {
		t.Fatalf("record source unresolved: %+v %v", resolved, err)
	}
	submit := func(tool factory.MCPTool, payload []byte) factory.MCPReceipt {
		t.Helper()
		submission := factory.MCPSubmission{
			AssignmentID: capability.AssignmentID, SubmissionID: factory.NewID(),
			CapabilityID: capability.ID, Tool: tool,
			PayloadHash: factory.MCPInputDigest(payload), Payload: payload,
		}
		receipt, err := s.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{
			SubmissionID: submission.SubmissionID, Tool: tool,
			State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
		})
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	receipt := submit(factory.MCPToolReply, []byte(`{"title":"Approval boundary","summary":"I checked the delivery guard and the self-approval case still fails before changing code.","reply_to":"`+invitation+`"}`))
	thread, _, err := s.FeedReplyIDs(ctx, 7, "record", sourceID, 0)
	if err != nil || len(thread) != 2 || thread[0] != receipt.FeedPostID || thread[1] != invitation {
		t.Fatalf("reply missed original record thread: %v %v", thread, err)
	}
	page, err := s.MCPDiscussion(ctx, capability, invitation, "")
	if err != nil || len(page.Items) != 2 || page.Items[0].ID != receipt.FeedPostID {
		t.Fatalf("invited thread unreadable: %+v %v", page, err)
	}
	duplicate := factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: factory.NewID(),
		CapabilityID: capability.ID, Tool: factory.MCPToolReply,
		PayloadHash: factory.MCPInputDigest([]byte(`{"title":"Again","summary":"Second reply.","reply_to":"` + invitation + `"}`)),
		Payload:     []byte(`{"title":"Again","summary":"Second reply.","reply_to":"` + invitation + `"}`),
	}
	if _, err := s.RecordMCPSubmission(ctx, duplicate, factory.MCPReceipt{
		SubmissionID: duplicate.SubmissionID, Tool: duplicate.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: duplicate.PayloadHash,
	}); err == nil {
		t.Fatal("record invitation admitted a second reply")
	}
}
