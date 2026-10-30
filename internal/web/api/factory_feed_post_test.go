package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Real PostgreSQL drives the intake, role post, native-effect journal and
// replies. This native peer supplies explicit conditional operation receipts.
type intakeNativePeer struct {
	issues     map[int64]*sdk.SnapshotIssue
	operations map[string]sdk.OperationRecord
	lostReply  bool
}

func seedIntakeGrant(t *testing.T, s *store.Store, ctx context.Context, id string) {
	t.Helper()
	comment, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	grant := sdk.IssueIntakeGrant{Token: id, Claims: sdk.IssueIntakeClaims{Installation: "declared-native-peer", Source: id, Revision: comment.Record.Revision, Actor: comment.Record.Author, Repository: strconv.FormatInt(comment.Record.Repository, 10), Digest: factory.MCPInputDigest([]byte(comment.Body)), Expires: time.Now().Add(time.Hour).Unix()}}
	if err = s.SaveIssueIntakeGrant(ctx, id, comment.Record.Revision, grant); err != nil {
		t.Fatal(err)
	}
}

func (g *intakeNativePeer) ApplyIssueIntake(ctx context.Context, request sdk.IssueIntakeMutation) (sdk.IssueIntakeReceipt, error) {
	id := request.Grant + "/" + request.Action + "/" + strconv.Itoa(request.Slot)
	if request.Action == "refine" {
		id = request.Grant + "/refine/" + strconv.Itoa(request.Step)
	}
	if row, ok := g.operations[id]; ok {
		var receipt sdk.IssueGraphReceipt
		_ = json.Unmarshal(row.Receipt, &receipt)
		return sdk.IssueIntakeReceipt{Index: receipt.Issues[0].IssueIndex, IssueID: receipt.Issues[0].IssueID, Slot: request.Slot, Title: request.Title, BodyDigest: factory.MCPInputDigest([]byte(request.Body))}, nil
	}
	payload := sdk.IssueGraphPayload{}
	if request.Action == "create" {
		payload.Creates = []sdk.IssueGraphCreate{{Key: strconv.Itoa(request.Slot), Title: request.Title, Body: request.Body}}
	} else {
		issue := g.issues[int64(request.Slot+1)]
		if issue == nil || issue.Title != request.ExpectedTitle || factory.MCPInputDigest([]byte(issue.Content)) != request.ExpectedBodyDigest {
			return sdk.IssueIntakeReceipt{}, errors.New("human edit is preserved")
		}
		payload.Edits = []sdk.IssueGraphEdit{{IssueID: int64(request.Slot + 1), ExpectedTitle: issue.Title, ExpectedBody: issue.Content, ExpectedContentVersion: issue.ContentVersion, Title: request.Title, Body: request.Body}}
	}
	raw, _ := json.Marshal(payload)
	record, err := g.SubmitGraph(ctx, factory.IntentGeneration{OperationID: id, Payload: raw}, 7)
	if err != nil {
		return sdk.IssueIntakeReceipt{}, err
	}
	var receipt sdk.IssueGraphReceipt
	_ = json.Unmarshal(record.Receipt, &receipt)
	return sdk.IssueIntakeReceipt{Index: receipt.Issues[0].IssueIndex, IssueID: receipt.Issues[0].IssueID, Slot: request.Slot, Title: request.Title, BodyDigest: factory.MCPInputDigest([]byte(request.Body))}, nil
}

func (g *intakeNativePeer) CheckIntentOwner(context.Context, int64, int64, int64) (int64, error) {
	return 1, nil
}
func (g *intakeNativePeer) LookupGraph(_ context.Context, in factory.IntentGeneration, _ int64) (sdk.OperationLookup, error) {
	if row, ok := g.operations[in.OperationID]; ok {
		return sdk.OperationLookup{Record: &row}, nil
	}
	return sdk.OperationLookup{}, nil
}
func (g *intakeNativePeer) CancelGraph(context.Context, string) (sdk.OperationRecord, error) {
	return sdk.OperationRecord{}, nil
}
func (g *intakeNativePeer) ReadGeneratedIssue(_ context.Context, _ int64, index, _ int64) (sdk.NativeSnapshot, int64, error) {
	issue := g.issues[index]
	if issue == nil {
		return sdk.NativeSnapshot{}, 0, errors.New("missing issue")
	}
	copy := *issue
	return sdk.NativeSnapshot{Issue: &copy}, 1, nil
}
func (g *intakeNativePeer) SubmitGraph(_ context.Context, in factory.IntentGeneration, _ int64) (sdk.OperationRecord, error) {
	if row, ok := g.operations[in.OperationID]; ok {
		return row, nil
	}
	var payload sdk.IssueGraphPayload
	if err := json.Unmarshal(in.Payload, &payload); err != nil {
		return sdk.OperationRecord{}, err
	}
	if g.issues == nil {
		g.issues = map[int64]*sdk.SnapshotIssue{}
		g.operations = map[string]sdk.OperationRecord{}
	}
	var receipt sdk.IssueGraphReceipt
	for _, create := range payload.Creates {
		index := int64(len(g.issues) + 1)
		g.issues[index] = &sdk.SnapshotIssue{ID: strconv.FormatInt(index, 10), Index: strconv.FormatInt(index, 10), Title: create.Title, Content: create.Body, ContentVersion: 1, Visible: true, Complete: true}
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{Key: create.Key, IssueID: index, IssueIndex: index, ContentVersion: 1})
	}
	for _, edit := range payload.Edits {
		issue := g.issues[edit.IssueID]
		if issue == nil || issue.Title != edit.ExpectedTitle || issue.Content != edit.ExpectedBody || issue.ContentVersion != edit.ExpectedContentVersion {
			return sdk.OperationRecord{}, errors.New("conditional native edit refused")
		}
		issue.Title, issue.Content = edit.Title, edit.Body
		issue.ContentVersion++
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{IssueID: edit.IssueID, IssueIndex: edit.IssueID, ContentVersion: issue.ContentVersion})
	}
	raw, _ := json.Marshal(receipt)
	row := sdk.OperationRecord{OperationID: in.OperationID, EffectState: sdk.EffectCommitted, Receipt: raw}
	g.operations[in.OperationID] = row
	if g.lostReply {
		g.lostReply = false
		return sdk.OperationRecord{}, errors.New("declared lost native reply")
	}
	return row, nil
}

func TestIssueMakerCapturesBeforePlanningAndPublishesOnePostPerIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "Add a login screen and notifications", "issue-maker-intake", true)
	legacy := saveDrivePost(t, s, ctx, "Saved request left for its author", "legacy-request")
	if s.PostUsesIssueMaker(ctx, legacy, 0) {
		t.Fatal("older intake was opted into new work")
	}
	if !s.PostUsesIssueMaker(ctx, id, 0) {
		t.Fatal("new intake did not atomically select Issue Maker")
	}
	seedIntakeGrant(t, s, ctx, id)
	graph := &intakeNativePeer{}
	app := &API{Store: s, Config: &config.Config{}, IssueMakerNative: graph, Coordinator: &control.Coordinator{Store: s, Graph: graph, GraphActorID: 9}}
	wired := &PostCollaborators{Judge: &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}}, IssueMaker: app.drivePostIssueMaker}
	result, err := drivePostIntake(ctx, s, wired, id)
	if err != nil || result.Effect != factory.PostEffectVerified || result.Issue != 1 {
		t.Fatalf("initial issue waited for planning: %+v %v", result, err)
	}
	post, err := s.PostIssueMaker(ctx, id, 0)
	if err != nil || post.State != "queued" || post.Run != "" {
		t.Fatalf("initial role state: %+v %v", post, err)
	}
	rolePost, err := s.FeedPost(ctx, post.Scope.Post)
	if err != nil || rolePost.Record.ParentID != id {
		t.Fatalf("role post lost its source parent: %+v %v", rolePost, err)
	}
	parentContext := []factory.FeedEntry{{Kind: factory.FeedKindPost, ID: post.Scope.Post, Repository: 7, ParentID: id}}
	enrichFeedParents(ctx, s, parentContext)
	if parentContext[0].ParentAuthor == "" {
		t.Fatal("off-page parent author missing")
	}
	humanParent, err := s.FeedPost(ctx, id)
	if err != nil || parentContext[0].ParentActor != humanParent.Record.Author {
		t.Fatalf("off-page human parent actor mismatch: %+v %v", parentContext[0], err)
	}
	if _, err = s.IntentRevision(ctx, factory.PostDecisionID("intent", id, 0), 42); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("planning started before the first issue receipt: %v", err)
	}
	replies, _, err := s.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil || len(replies) != 1 {
		t.Fatal("initial receipt is missing from the role thread", replies, err)
	}
	// A later progress pass can wait on AI preparation without hiding the issue.
	if _, err = drivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	post, err = s.PostIssueMaker(ctx, id, 0)
	if err != nil || post.State != "waiting" || post.Scope.Issue != 1 || post.Run != "" {
		t.Fatalf("refinement wait lost its first issue: %+v %v", post, err)
	}
	pending, err := s.PendingPostIntakes(ctx, 16)
	queued := false
	for _, request := range pending {
		queued = queued || request.SourcePost == id
	}
	if err != nil || !queued {
		t.Fatal("service continuation lost the captured issue", pending, err)
	}
	graph.lostReply = true
	if _, err = app.mutateIssueMakerIssue(ctx, post.Scope, "create", "notifications", "Add notifications", "Notify on important changes.", "create-notifications"); err == nil {
		t.Fatal("lost native response was not surfaced")
	}
	for range 2 {
		index, err := app.mutateIssueMakerIssue(ctx, post.Scope, "create", "notifications", "Add notifications", "Notify on important changes.", "create-notifications")
		if err != nil || index != 2 {
			t.Fatal("native replay did not reconcile", index, err)
		}
	}
	replies, _, err = s.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil || len(replies) != 2 || len(graph.issues) != 2 {
		t.Fatal("replay duplicated or lost issue comments", replies, len(graph.issues), err)
	}
	for _, id := range replies {
		detail, err := s.FeedPost(ctx, id)
		if err != nil || detail.Actor != "issue_maker" || detail.Record.ParentID != post.Scope.Post || detail.Record.Issue <= 0 {
			t.Fatal("reply lost native issue provenance", detail, err)
		}
	}
	entries, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	if !seen[post.Scope.Post] || !seen[replies[0]] || !seen[replies[1]] {
		t.Fatal("feed omitted the role post or its live receipts", seen)
	}
	graph.issues[1].Content = "A human changed this issue."
	if _, err = app.mutateIssueMakerIssue(ctx, post.Scope, "refine", "request", "Refined request", "Replacement body", "human-conflict"); err == nil {
		t.Fatal("refinement overwrote human content")
	}
	if graph.issues[1].Content != "A human changed this issue." {
		t.Fatal("human content was not retained")
	}
	if _, err = app.mutateIssueMakerIssue(ctx, post.Scope, "refine", "unrelated", "Unrelated", "Must refuse", "cross-scope"); err == nil {
		t.Fatal("role edited an issue outside its own created set")
	}
}

type waitingPostPlanner struct{ dispatches int }

func (p *waitingPostPlanner) AssembleIntentPlanningInput(context.Context, factory.IntentRevision, string) (factory.SemanticPlannerInput, project.Preparation, *control.DispatchWait, error) {
	return factory.SemanticPlannerInput{}, project.Preparation{}, &control.DispatchWait{Reason: control.WaitPreparation, Detail: "project has no accepted requirements"}, nil
}
func (p *waitingPostPlanner) DispatchIntentPlan(context.Context, string, int64, string, factory.SemanticPlannerInput, string, string) control.DispatchReport {
	p.dispatches++
	return control.DispatchReport{}
}

func TestPostAuthoringWaitIsPersistedWithoutPretendingToRun(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "Please add a login flow", "authoring-wait")
	planner := &waitingPostPlanner{}
	wired := &PostCollaborators{Judge: &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}}, Drafter: &IntentPostDrafter{Store: s, Transport: planner}}
	if _, err := drivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	status, err := s.PostAuthoring(ctx, id)
	if err != nil || status == nil {
		t.Fatal("authoring status missing", err)
	}
	if status.State != factory.PostAuthoringWaiting || status.Reason != control.WaitPreparation || status.Detail != "project has no accepted requirements" || status.Run != "" || status.Intent == "" || status.CheckedUnix == 0 {
		t.Fatal("waiting state lost its actual cause", status)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil || pipe.Attempts != 0 || planner.dispatches != 0 {
		t.Fatal("wait dispatched or spent an attempt", pipe, err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	application := &API{Store: s, Config: &config.Config{}}
	view := application.feedPostWithProduct(httptest.NewRequest("GET", "/", nil), store.Session{User: store.User{ID: 42}}, detail, pipe, "")
	projected, ok := view["authoring"].(factory.PostAuthoring)
	if !ok || projected.State != factory.PostAuthoringWaiting || projected.Reason != control.WaitPreparation {
		t.Fatal("read projection lost authoring state", view)
	}
	if err := s.NotePostAuthoring(ctx, id, pipe.Revision, *status, status.CheckedUnix+5); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.PostAuthoring(ctx, id)
	if err != nil || fresh.SinceUnix != status.SinceUnix || fresh.CheckedUnix <= status.CheckedUnix {
		t.Fatal("wait timestamps drifted", fresh, err)
	}
	if err := s.NotePostAuthoring(ctx, id, pipe.Revision, factory.PostAuthoring{State: factory.PostAuthoringRunning, Run: "recorded-run"}, status.CheckedUnix+10); err != nil {
		t.Fatal(err)
	}
	fresh, err = s.PostAuthoring(ctx, id)
	if err != nil || fresh.State != factory.PostAuthoringRunning || fresh.SinceUnix != status.CheckedUnix+10 {
		t.Fatal("run transition was not recorded", fresh, err)
	}
}

// Fake Lane B collaborators: deterministic routing, authoring and
// native effects without spending, dispatching or network calls.

type fakePostJudge struct {
	judgment PostJudgment
	err      error
	calls    int
	input    PostJudgeInput
}

func (f *fakePostJudge) JudgePost(ctx context.Context, input PostJudgeInput) (PostJudgment, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return PostJudgment{}, f.err
	}
	return f.judgment, nil
}

func TestPostRoutingUsesSavedHumanContextWithoutCoordinator(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "Just discussing the existing login flow", "human-context-routing")
	judge := &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteDiscussion}}
	application := &API{Store: s}
	wired := &PostCollaborators{Judge: judge, Prepare: application.preparePostJudgeInput}
	// Older requests wait without spending until their author refreshes context.
	if _, err := drivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil || pipe.Attempts != 0 || judge.calls != 0 {
		t.Fatal("missing human context spent a judgment", pipe, err)
	}
	if err := s.SavePostIssueContext(ctx, id, pipe.Revision, []factory.PostWorkCandidate{{Issue: 55, Title: "Login flow", Objective: "Existing login behavior", Origin: factory.PostCandidateRecent, Status: "open"}}); err != nil {
		t.Fatal(err)
	}
	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil || response.Route != factory.PostRouteDiscussion || judge.calls != 1 {
		t.Fatal("routing required Factory setup", response, err)
	}
	if len(judge.input.Candidates) != 1 || judge.input.Candidates[0].Title != "Login flow" {
		t.Fatal("routing lost native issue meaning", judge.input.Candidates)
	}
}

type fakePostDrafter struct {
	draft factory.IssueDraft
	err   error
	errs  []error
	calls int
}

func (f *fakePostDrafter) DraftPostIssue(ctx context.Context, comment store.FeedPostDetail) (factory.IssueDraft, error) {
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

type fakePostCreator struct {
	issues    map[string]int64
	next      int64
	creates   int
	createErr error
}

type feedbackPostCreator struct {
	fakePostCreator
	deliveries int
}

func TestClarificationQuestionAndAnswerResumeTheSameRequest(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Change that page", IdempotencyKey: "clarification"}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	judgment, err := postJudgmentFromSelection("clarify_target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = applyPostJudgment(ctx, s, pipe, judgment); err != nil {
		t.Fatal(err)
	}
	question, err := s.PostProgress(ctx, id)
	if err != nil || question != postClarificationQuestions["clarify_target"] {
		t.Fatalf("missing specific question: %q %v", question, err)
	}
	if err = s.RetryFeedPost(ctx, id, "42", 0, "Change that page\n\nClarification: the onboarding page", 1729000001); err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil || detail.Record.Revision != 1 || detail.Body != "Change that page\n\nClarification: the onboarding page" {
		t.Fatalf("answer lost original request: %+v %v", detail, err)
	}
	resumed, err := s.PostIntake(ctx, id)
	if err != nil || resumed.Judgment != factory.PostJudgmentPending {
		t.Fatalf("answer did not resume assessment: %+v %v", resumed, err)
	}
}

func (f *feedbackPostCreator) DeliverPostFeedback(_ context.Context, detail store.FeedPostDetail, issue int64) error {
	if issue != 55 || detail.Body != "Fix the filter" {
		return errors.New("feedback changed source or issue")
	}
	f.deliveries++
	return nil
}

func TestExistingWorkFeedbackActuallyReachesTheSelectedIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Fix the filter", Issue: 55, IdempotencyKey: "feedback"}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	creator := &feedbackPostCreator{}
	wired := &PostCollaborators{Judge: &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteExistingWork, Issue: 55}}, Creator: creator}
	for range 2 {
		response, err := drivePostIntake(ctx, s, wired, id)
		if err != nil || response.Effect != factory.PostEffectApplied {
			t.Fatalf("feedback not delivered: %+v %v", response, err)
		}
	}
	if creator.deliveries != 1 || creator.creates != 0 {
		t.Fatalf("feedback duplicated or created new work: %+v", creator)
	}
}

func (f *fakePostCreator) CreatePostIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.creates++
	f.next++
	if f.issues == nil {
		f.issues = make(map[string]int64)
	}
	f.issues[draft.SourcePost] = f.next
	return f.next, nil
}

func (f *fakePostCreator) LookupPostIssue(ctx context.Context, repository int64, postID string) (int64, error) {
	return f.issues[postID], nil
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

func saveDrivePost(t *testing.T, s *store.Store, ctx context.Context, body, key string, issueMaker ...bool) string {
	t.Helper()
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: body, IdempotencyKey: key}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	var intakeErr error
	if len(issueMaker) > 0 && issueMaker[0] {
		intakeErr = s.RecordIssueMakerIntake(ctx, id, 0)
	} else {
		intakeErr = s.RecordPostIntake(ctx, id, 0)
	}
	if intakeErr != nil {
		t.Fatal(intakeErr)
	}
	return id
}

func validTestDraft(comment string) factory.IssueDraft {
	return factory.IssueDraft{SourcePost: comment, Title: "Fix login redirect", Body: "Change the callback.", Objective: "Logins land back in the app.", Acceptance: "Redirect returns 302.", Metadata: `{"work":"soda"}`}
}

func TestDrivePostNewIssueToVerified(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "please fix the login redirect", "key-1")
	judge := &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}}
	drafter := &fakePostDrafter{draft: validTestDraft(id)}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &PostCollaborators{Judge: judge, Drafter: drafter, Creator: creator}

	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.SourcePost != id || response.Judgment != factory.PostJudgmentApplied || response.Route != factory.PostRouteNewIssue ||
		response.Draft != factory.PostDraftApplied || response.Effect != factory.PostEffectVerified || response.Issue != 101 {
		t.Fatalf("drove response: %+v", response)
	}
	if judge.calls != 1 || drafter.calls != 1 || creator.creates != 1 {
		t.Fatalf("calls judge=%d draft=%d creates=%d", judge.calls, drafter.calls, creator.creates)
	}
	// A replay recovers the same identity and verified link without any
	// new call, draft or native create.
	replayed, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.SourcePost != id || replayed.Effect != factory.PostEffectVerified || replayed.Issue != 101 {
		t.Fatalf("replayed response: %+v", replayed)
	}
	if judge.calls != 1 || drafter.calls != 1 || creator.creates != 1 {
		t.Fatalf("replay spent: judge=%d draft=%d creates=%d", judge.calls, drafter.calls, creator.creates)
	}
}

func TestDrivePostDiscussionCreatesNothing(t *testing.T) {
	s, ctx := driveTestStore(t)
	if err := s.RecordRepositoryOpening(ctx, 7, 1729000000); err != nil {
		t.Fatal(err)
	}
	entries, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 20})
	if err != nil || len(entries) != 1 {
		t.Fatalf("opening question missing: %+v %v", entries, err)
	}
	id, revision, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, ParentID: entries[0].ID, Body: "thanks, acknowledged", IdempotencyKey: "key-1"}, "42", "ada", 1729000001)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordIssueMakerIntake(ctx, id, revision); err != nil {
		t.Fatal(err)
	}
	judge := &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteDiscussion}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &PostCollaborators{Judge: judge, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator}

	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Route != factory.PostRouteDiscussion || response.Draft != factory.PostDraftPending || response.Effect != factory.PostEffectPending || response.Issue != 0 {
		t.Fatalf("discussion response: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("discussion created an issue")
	}
	if judge.calls != 1 || judge.input.Parent == nil || judge.input.Parent.Body != store.RepositoryOpeningQuestion {
		t.Fatalf("human reply was not evaluated with its Factory question: %+v", judge.input)
	}
}

func TestDrivePostMalformedDraftBlocksEffect(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	broken := validTestDraft(id)
	broken.Acceptance = ""
	drafter := &fakePostDrafter{err: errors.New("source assembly unavailable")}
	wired := &PostCollaborators{
		Judge:   &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}},
		Drafter: drafter,
		Creator: &fakePostCreator{next: 100, issues: map[string]int64{}},
	}
	for range factory.PostAttemptLimit {
		if _, err := drivePostIntake(ctx, s, wired, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RetryFeedPost(ctx, id, "42", 0, "", 1729000050); err != nil {
		t.Fatal(err)
	}
	drafter.err, drafter.draft = nil, broken
	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	// The retry re-routes the request as a new source revision.
	if response.Draft != factory.PostDraftInvalid || response.Effect != factory.PostEffectBlocked || response.Revision != 1 {
		t.Fatalf("malformed response: %+v", response)
	}
	if _, err := s.PostDraft(ctx, id, 1); err == nil {
		t.Fatal("refused draft stays servable")
	}
}

func TestDrivePostEffectReplyLossReconciles(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	// The native issue 101 already exists from a lost reply: the
	// creator knows it, the pipeline does not.
	creator := &fakePostCreator{next: 101, issues: map[string]int64{id: 101}}
	wired := &PostCollaborators{
		Judge:   &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}},
		Drafter: &fakePostDrafter{draft: validTestDraft(id)},
		Creator: creator,
	}
	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Effect != factory.PostEffectVerified || response.Issue != 101 {
		t.Fatalf("reconciled response: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("lost reply duplicated the issue")
	}
}

func TestDrivePostDeletedSourceSpendsNothing(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	judge := &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}}
	wired := &PostCollaborators{
		Judge:   judge,
		Drafter: &fakePostDrafter{draft: validTestDraft(id)},
		Creator: &fakePostCreator{next: 100, issues: map[string]int64{}},
	}
	if _, err := s.DeleteFeedPost(ctx, id, "42", 0); err != nil {
		t.Fatal(err)
	}
	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Effect != factory.PostEffectBlocked {
		t.Fatalf("deleted response: %+v", response)
	}
	if judge.calls != 0 {
		t.Fatal("deleted source spent a judgment call")
	}
}

func TestDrivePostJudgmentFailureExhaustsHonestly(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	wired := &PostCollaborators{Judge: &fakePostJudge{err: errors.New("jev down")}}
	var response factory.PostResponse
	var err error
	for range factory.PostAttemptLimit {
		response, err = drivePostIntake(ctx, s, wired, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if response.Judgment != factory.PostJudgmentFailed {
		t.Fatalf("exhausted judgment = %q", response.Judgment)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Attempts != factory.PostAttemptLimit {
		t.Fatalf("attempts = %d", pipe.Attempts)
	}
}

func TestDrivePostUnwiredStagesWaitInPending(t *testing.T) {
	s, ctx := driveTestStore(t)
	id := saveDrivePost(t, s, ctx, "fix it", "key-1")
	response, err := drivePostIntake(ctx, s, &PostCollaborators{}, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Intake != factory.PostIntakeRecorded || response.Judgment != factory.PostJudgmentPending {
		t.Fatalf("unwired response: %+v", response)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Attempts != 0 {
		t.Fatalf("unwired drive recorded %d attempts", pipe.Attempts)
	}
}

// TestDrivePendingPostsResumesAfterRequestEnd proves the durable
// continuation: the POST's two durable writes (comment + intake) are
// all that survive the request, and service passes alone carry the
// stored request to its verified issue. A still-settling planner
// keeps the draft pending with attempts intact; the next pass
// finishes it, and the issue appears exactly once in readback.
func TestDrivePendingPostsResumesAfterRequestEnd(t *testing.T) {
	s, ctx := driveTestStore(t)
	// Exactly what the POST persists before answering promptly.
	id := saveDrivePost(t, s, ctx, "please fix the login redirect", "key-1")
	judge := &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteNewIssue}}
	drafter := &fakePostDrafter{draft: validTestDraft(id), errs: []error{ErrPostDraftPending}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &PostCollaborators{Judge: judge, Drafter: drafter, Creator: creator}

	if driven := drivePendingPosts(ctx, s, wired, postDriveLimit); driven != 1 {
		t.Fatalf("first pass driven %d, want 1", driven)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Judgment != factory.PostJudgmentApplied || pipe.Route != factory.PostRouteNewIssue ||
		pipe.Draft != factory.PostDraftPending || pipe.Attempts != 0 {
		t.Fatalf("pending planner burned state: %+v", pipe)
	}
	if creator.creates != 0 {
		t.Fatal("pending planner created an issue")
	}

	if driven := drivePendingPosts(ctx, s, wired, postDriveLimit); driven != 1 {
		t.Fatalf("second pass driven %d, want 1", driven)
	}
	pipe, err = s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Draft != factory.PostDraftApplied || pipe.Effect != factory.PostEffectVerified || pipe.Issue != 101 {
		t.Fatalf("resumed pipeline: %+v", pipe)
	}
	if judge.calls != 1 || creator.creates != 1 {
		t.Fatalf("resume spent twice: judge=%d creates=%d", judge.calls, creator.creates)
	}
	if status := store.PostStatus(&pipe); status != "issue:101" {
		t.Fatalf("readback status %q, want issue:101", status)
	}
	// Settled requests leave the service work list entirely.
	if driven := drivePendingPosts(ctx, s, wired, postDriveLimit); driven != 0 {
		t.Fatalf("settled request drove %d more times", driven)
	}
	if creator.creates != 1 {
		t.Fatal("settled request created a duplicate issue")
	}
}

// TestDrivePostExistingWorkPersistsSelectedIssue proves an
// existing_work judgment persists WHICH tracked work it selected,
// the judge saw the candidate it picked, and a receipt recorded
// before a crash replays without a second judgment call.
func TestDrivePostExistingWorkPersistsSelectedIssue(t *testing.T) {
	s, ctx := driveTestStore(t)
	id, _, err := s.SaveFeedPost(ctx, factory.PostRequest{
		Repository: 7, Body: "this belongs with the filter work", Issue: 55,
		IdempotencyKey: "key-existing",
	}, "42", "ada", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	judge := &fakePostJudge{judgment: PostJudgment{Route: factory.PostRouteExistingWork, Issue: 55}}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	wired := &PostCollaborators{Judge: judge, Drafter: &fakePostDrafter{draft: validTestDraft(id)}, Creator: creator}

	response, err := drivePostIntake(ctx, s, wired, id)
	if err != nil {
		t.Fatal(err)
	}
	if response.Route != factory.PostRouteExistingWork || response.Issue != 55 || response.ExistingIssue != 55 {
		t.Fatalf("existing work response lost its selection: %+v", response)
	}
	if creator.creates != 0 {
		t.Fatal("existing work created an issue")
	}
	found := false
	for _, candidate := range judge.input.Candidates {
		if candidate.Issue == 55 && candidate.Origin == factory.PostCandidateBound {
			found = true
		}
	}
	if !found {
		t.Fatalf("judge never saw bound candidate 55: %+v", judge.input.Candidates)
	}
}

// TestPostJudgeChoiceHelpers pins the candidate-bound choice
// contract: existing-work keys exist only for listed candidates,
// selections outside the offered set refuse, and the judge state
// carries source, parent and candidate evidence.
func TestPostJudgeChoiceHelpers(t *testing.T) {
	candidates := []factory.PostWorkCandidate{
		{Issue: 55, Origin: factory.PostCandidateBound, Summary: "filter work"},
		{Issue: 9, Origin: factory.PostCandidateRecent},
	}
	keys := postChoiceKeys(candidates)
	want := []string{"new_issue", "discussion", "clarify_target", "clarify_result", "clarify_conflict", "existing_55", "existing_9"}
	if len(keys) != len(want) {
		t.Fatalf("keys %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys %v, want %v", keys, want)
		}
	}
	if keys := postChoiceKeys(nil); len(keys) != 5 {
		t.Fatalf("candidate-less keys %v must exclude existing work", keys)
	}
	judgment, err := postJudgmentFromSelection("existing_9", candidates)
	if err != nil || judgment.Route != factory.PostRouteExistingWork || judgment.Issue != 9 {
		t.Fatalf("selection parse: %+v err=%v", judgment, err)
	}
	if _, err := postJudgmentFromSelection("existing_404", candidates); err == nil {
		t.Fatal("unlisted existing work admitted")
	}
	if _, err := postJudgmentFromSelection("delete_everything", candidates); err == nil {
		t.Fatal("unknown selection admitted")
	}
	input := PostJudgeInput{
		SourcePost: store.FeedPostDetail{
			Record: factory.PostRecord{Repository: 7, Issue: 55, ParentID: "parent"},
			Body:   "more filter",
			Source: &factory.PostSource{Kind: factory.PostSourceRecord, ID: "rec", Issue: 55, Summary: "built filter"},
		},
		Parent:     &PostParentEvidence{SourcePost: "parent", Body: "start filter", Issue: 55},
		Candidates: candidates,
	}
	state := postJudgeState(input)
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
// original record thread behind the existing ResolvePostSource and
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
	if err != nil || review.SourceKind != factory.PostSourceRecord || len(review.Roles) != 1 {
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
	resolved, err := s.ResolvePostSource(ctx, 7, factory.PostSourceRecord, sourceID)
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
		receipt, err := s.RecordMCPSubmission(ctx, submission)
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	receipt := submit(factory.MCPToolReply, []byte(`{"title":"Approval boundary","summary":"I checked the delivery guard and the self-approval case still fails before changing code.","parent_id":"`+invitation+`"}`))
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
		PayloadHash: factory.MCPInputDigest([]byte(`{"title":"Again","summary":"Second reply.","parent_id":"` + invitation + `"}`)),
		Payload:     []byte(`{"title":"Again","summary":"Second reply.","parent_id":"` + invitation + `"}`),
	}
	if _, err := s.RecordMCPSubmission(ctx, duplicate); err == nil {
		t.Fatal("record invitation admitted a second reply")
	}
}
