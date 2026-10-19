package api

import (
	"context"
	"errors"
	"testing"

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

func (f *fakeCommentJudge) JudgeComment(ctx context.Context, input CommentJudgeInput, snapshot string) (CommentJudgment, factory.PriorityReceipt, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return CommentJudgment{}, factory.PriorityReceipt{}, f.err
	}
	selected := f.judgment.Route
	if selected == factory.CommentRouteClarify {
		selected = "clarify_result"
	}
	if f.judgment.Route == factory.CommentRouteExistingWork {
		selected = commentExistingChoice(f.judgment.Issue)
	}
	request := factory.PriorityRequest{Snapshot: snapshot, Rubric: commentRouteRubric}
	for _, key := range commentChoiceKeys(input.Candidates) {
		request.Candidates = append(request.Candidates, factory.PriorityCandidate{ID: key})
	}
	receipt := factory.PriorityReceipt{
		Snapshot: snapshot, Candidates: request.Candidates, Selected: selected,
		Outcome:     "routed",
		Calls:       []factory.PriorityCall{{Request: request, Answer: factory.PriorityAnswer{Selected: selected}}},
		CreatedUnix: 1,
	}
	return f.judgment, receipt, nil
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
	s, _, cleanup, err := store.OpenEphemeral(context.Background(), admin, nil)
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
	wired := &CommentCollaborators{
		Judge:   &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteNewIssue}},
		Drafter: &fakeCommentDrafter{draft: broken},
		Creator: &fakeCommentCreator{next: 100, issues: map[string]int64{}},
	}
	response, err := driveCommentPipeline(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Draft != factory.CommentDraftInvalid || response.Effect != factory.CommentEffectBlocked {
		t.Fatalf("malformed response: %+v", response)
	}
	if _, err := s.CommentDraft(ctx, id, 0); err == nil {
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
	receipt, err := s.PriorityJudgment(ctx, store.CommentJudgmentSnapshot(id, 0))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Selected != "existing_55" {
		t.Fatalf("receipt selected %q, want existing_55", receipt.Selected)
	}

	// Crash replay: a second comment whose receipt is already
	// recorded but whose pipeline never advanced applies the stored
	// selection without calling the judge again.
	id2, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{
		Repository: 7, Body: "also the filter work", Issue: 55,
		IdempotencyKey: "key-existing-2",
	}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordCommentIntake(ctx, id2, 0); err != nil {
		t.Fatal(err)
	}
	builder := &fakeCommentJudge{judgment: CommentJudgment{Route: factory.CommentRouteExistingWork, Issue: 55}}
	_, recorded, err := builder.JudgeComment(ctx, CommentJudgeInput{
		Candidates: []factory.CommentWorkCandidate{{Issue: 55, Origin: factory.CommentCandidateBound}},
	}, store.CommentJudgmentSnapshot(id2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if funded, err := s.ReserveCommentJudgment(ctx, id2, 0); err != nil || !funded {
		t.Fatalf("reserve replay judgment: funded=%v err=%v", funded, err)
	}
	if err := s.RecordCommentJudgmentReceipt(ctx, id2, 0, recorded); err != nil {
		t.Fatal(err)
	}
	replayJudge := &fakeCommentJudge{err: errors.New("must not be called")}
	replayed, err := driveCommentPipeline(ctx, s, &CommentCollaborators{Judge: replayJudge}, id2)
	if err != nil {
		t.Fatal(err)
	}
	if replayJudge.calls != 0 {
		t.Fatal("crash replay re-judged instead of applying the stored receipt")
	}
	if replayed.Route != factory.CommentRouteExistingWork || replayed.ExistingIssue != 55 {
		t.Fatalf("replayed response: %+v", replayed)
	}
}

// TestCommentJudgeChoiceHelpers pins the candidate-bound choice
// contract: existing-work keys exist only for listed candidates,
// selections outside the offered set refuse, stored receipts recover
// their candidate set, and the judge state carries source, parent
// and candidate evidence.
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
	recovered := commentCandidatesFromReceipt(factory.PriorityReceipt{Candidates: []factory.PriorityCandidate{
		{ID: "new_issue"}, {ID: "existing_55"}, {ID: "existing_x"},
	}})
	if len(recovered) != 1 || recovered[0].Issue != 55 {
		t.Fatalf("receipt candidates %+v", recovered)
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
