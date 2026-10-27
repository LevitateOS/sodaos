package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// actorCall serves the product mux as one fixed native actor through a
// dedicated callback. The SDK reads the callback address per request, so
// each call temporarily points CallbackEnv at this actor's socket.
func actorCall(t *testing.T, product *API, actorID, username string) func(method, path, body string) (int, []byte) {
	t.Helper()
	callbackPath := "@soda-review-" + actorID + "-" + strconv.Itoa(os.Getpid())
	listener, err := net.Listen("unix", callbackPath)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		switch in.Operation {
		case extensions.OperationCurrentActor:
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: actorID, Username: username}})
		case extensions.OperationRepository:
			if in.RepositoryID != "7" {
				http.Error(w, "not visible", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "Alpha", Permission: "admin"}})
		default:
			http.Error(w, "denied", http.StatusForbidden)
		}
	}))
	callback.Listener.Close()
	callback.Listener = listener
	callback.Start()
	t.Cleanup(callback.Close)
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)
	return func(method, path, body string) (int, []byte) {
		t.Helper()
		authority, err := json.Marshal(extensions.Authority{
			ExtensionID: "soda", InstanceID: "review-test", SessionGeneration: "generation-1",
			Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)},
			Actor:        extensions.Actor{ID: actorID, Username: username},
		})
		if err != nil {
			t.Fatal(err)
		}
		previous, existed := os.LookupEnv(extensions.CallbackEnv)
		if err := os.Setenv(extensions.CallbackEnv, callbackPath); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if existed {
				_ = os.Setenv(extensions.CallbackEnv, previous)
			} else {
				_ = os.Unsetenv(extensions.CallbackEnv)
			}
		}()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code, w.Body.Bytes()
	}
}

// Real HTTP and PostgreSQL exercise the same source-derived review contract
// used by nested Finance comments; external routing is a declared fixture.
func TestCommentProductReviewHTTPPreservesOwnerAndExactSource(t *testing.T) {
	product, db, dsn, call, _ := q05MuxFull(t, "")
	ctx := context.Background()
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 7,
		Objective: "Employees request equipment and Operations records delivery", State: "accepted", MaxIssues: 10, PlanningCalls: 8, CreatedUnix: 1729000000,
		Ideas:             []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Keep request and approval history", Consequence: "Original records remain readable", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		EffectiveOutcomes: []string{"d333333333333333333333333"}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Deliver equipment workflow", Outcomes: []string{"Recorded request history"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	posts, _, _, err := db.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10})
	if err != nil || len(posts) != 1 {
		t.Fatalf("accepted source posts=%d err=%v", len(posts), err)
	}
	code, body := call(http.MethodPost, "/api/factory/feed/comments", `{"repository":7,"body":"New requests above EUR 1000 also need Finance approval. Existing requests keep their route.","source_kind":"post","source_id":"`+posts[0].ID+`","reply_to":"`+posts[0].ID+`","idempotency_key":"finance-change"}`)
	if code != http.StatusOK {
		t.Fatalf("comment save=%d %s", code, body)
	}
	var saved struct {
		Comment string `json:"comment_id"`
	}
	if err := json.Unmarshal(body, &saved); err != nil || saved.Comment == "" {
		t.Fatalf("comment identity=%s err=%v", body, err)
	}
	pipe, err := db.CommentPipeline(ctx, saved.Comment)
	if err != nil {
		t.Fatal(err)
	}
	// The external routing judgment is a declared fixture; the HTTP and
	// transaction boundaries below use the real mux and PostgreSQL owners.
	routed := pipe
	routed.Judgment, routed.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err := db.AdvanceCommentPipeline(ctx, pipe, routed); err != nil {
		t.Fatal(err)
	}
	drafter := &fakeCommentDrafter{draft: validTestDraft(saved.Comment)}
	creator := &fakeCommentCreator{next: 100, issues: map[string]int64{}}
	response, err := driveCommentPipeline(ctx, db, &CommentCollaborators{Drafter: drafter, Creator: creator}, saved.Comment)
	if err != nil || response.Issue != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatalf("product proposal created detached work: response=%+v drafts=%d creates=%d err=%v", response, drafter.calls, creator.creates, err)
	}
	path := "/api/factory/feed/comments/" + saved.Comment
	decode := func(code int, body []byte) store.CommentProductChange {
		t.Helper()
		var response struct {
			Change store.CommentProductChange `json:"product_change"`
			Issue  int64                      `json:"issue"`
		}
		if code != http.StatusOK || json.Unmarshal(body, &response) != nil || response.Change.Intent != accepted.Intent || response.Change.AcceptedRevision != accepted.ID {
			t.Fatalf("change projection=%d %s", code, body)
		}
		if response.Issue != 0 {
			t.Fatalf("same-product pipeline left its zero issue: %d", response.Issue)
		}
		return response.Change
	}
	view := decode(call(http.MethodGet, path, ""))
	if view.State != "review_required" || !view.CanReview || view.DraftRevision != "" {
		t.Fatalf("owner proposal=%+v", view)
	}
	if view.EvidenceStatus != store.CommentProductEvidenceUnavailable || view.Verified || len(view.Evidence) != 0 || view.AdoptedRevision != "" {
		t.Fatalf("unreviewed proposal claims work evidence: %+v", view)
	}
	other := actorCall(t, product, "2", "bob")
	view = decode(other(http.MethodGet, path, ""))
	if view.CanReview {
		t.Fatal("another native repository admin can review the owner's private intent")
	}
	if status, _ := other(http.MethodPost, path+"/review", `{"revision":0}`); status != http.StatusForbidden {
		t.Fatalf("other owner review=%d", status)
	}
	if status, _ := call(http.MethodPost, path+"/review", `{"revision":1}`); status != http.StatusPreconditionFailed {
		t.Fatalf("stale source revision=%d", status)
	}
	if status, _ := call(http.MethodPost, path+"/review", `{"revision":0,"intent":"d999999999999999999999999"}`); status != http.StatusBadRequest {
		t.Fatalf("caller asserted target=%d", status)
	}
	view = decode(call(http.MethodPost, path+"/review", `{"revision":0}`))
	if view.State != "draft" || !view.CanReview || view.DraftRevision == "" {
		t.Fatalf("draft continuation=%+v", view)
	}
	if view.EvidenceStatus != store.CommentProductEvidenceUnavailable || view.Verified || len(view.Evidence) != 0 || view.AdoptedRevision != "" {
		t.Fatalf("unadopted draft claims work evidence: %+v", view)
	}
	firstDraft := view.DraftRevision
	view = decode(call(http.MethodPost, path+"/review", `{"revision":0}`))
	if view.DraftRevision != firstDraft {
		t.Fatal("replayed review made a second draft")
	}
	view = decode(other(http.MethodGet, path, ""))
	if view.CanReview {
		t.Fatal("another actor can continue the owner's draft")
	}
	status, replyBody := call(http.MethodGet, "/api/factory/feed/replies?repository=7&source_kind=post&source_id="+posts[0].ID, "")
	var replies struct {
		Items []struct {
			Change *store.CommentProductChange `json:"product_change"`
		} `json:"items"`
	}
	if status != http.StatusOK || json.Unmarshal(replyBody, &replies) != nil || len(replies.Items) != 1 || replies.Items[0].Change == nil || !replies.Items[0].Change.CanReview {
		t.Fatalf("thread projection=%d %s", status, replyBody)
	}
	history, err := db.IntentHistory(ctx, accepted.Intent, 1, 20)
	if err != nil || len(history) != 2 {
		t.Fatalf("history after replay=%d err=%v", len(history), err)
	}
	head, err := db.IntentRevision(ctx, accepted.Intent, 1)
	if err != nil || head.State != "draft" || head.Predecessor != accepted.ID || head.Tasks[0].Key != accepted.Tasks[0].Key || len(head.Ideas) != 2 {
		t.Fatalf("preserved draft=%+v err=%v", head, err)
	}
	// Adopt the reviewed change, then seed the delivery evidence behind
	// it: a completed generation with native bindings, one canonical
	// completion and the controller-owned F10 fact. The comment and
	// replies HTTP projections must carry the recorded lineage to the
	// owner and read-only evidence to a collaborator; this proves
	// registered-HTTP readback of seeded fixtures, not a producer chain.
	commentIdea := factory.CommentDecisionID("product-idea", saved.Comment, 0)
	adopted := head
	financeTask := factory.IntentTask{Key: "d666666666666666666666666", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{commentIdea}}
	adopted.ID, adopted.Predecessor, adopted.State = "d555555555555555555555555", head.ID, "accepted"
	adopted.Tasks = append(append([]factory.IntentTask(nil), adopted.Tasks...), financeTask)
	if err := db.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	generation := factory.IntentGeneration{RevisionID: adopted.ID, IntentID: adopted.Intent, IntentDigest: adopted.Digest(), OperationID: "lineage-http", ActorID: 1, NativeRev: 1, NotAfter: 3, Payload: []byte(`{}`), State: "pending", SubmittedTasks: adopted.Tasks}
	if err := db.BeginIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	generation.State = "completed"
	if err := db.RecordIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	originalBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: adopted.Tasks[0].Key, IssueID: 100, IssueIndex: 11, AcceptanceID: "d777777777777777777777777", CreatedUnix: 4}
	matchedBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: financeTask.Key, IssueID: 101, IssueIndex: 12, AcceptanceID: "d888888888888888888888888", CreatedUnix: 4}
	if err := db.BindIntentTaskAcceptance(ctx, originalBinding); err != nil {
		t.Fatal(err)
	}
	if err := db.BindIntentTaskAcceptance(ctx, matchedBinding); err != nil {
		t.Fatal(err)
	}
	rawDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	write := func(query string, args ...any) {
		t.Helper()
		if _, err := rawDB.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: matchedBinding.IssueIndex, Generation: 2, PublicationID: factory.NewID(), AcceptanceID: matchedBinding.AcceptanceID, MergedCommit: strings.Repeat("a", 40), ClosingOccurrence: "1"}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	write(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,$1,$2,$3,'1',$4)`, 2, event.ID, event.PublicationID, string(raw))
	delivery := factory.NewID()
	assignmentID := factory.NewID()
	assignment, err := json.Marshal(map[string]any{
		"id": assignmentID, "attempt_root": assignmentID, "repository": "7", "issue": 0, "run": factory.NewID(), "stage": "finished",
		"phase": "verify", "role": "soda-reviewer", "outcome": "succeeded", "source_commit": strings.Repeat("b", 40),
		"subject": map[string]any{"kind": "delivery", "delivery": map[string]any{"delivery_id": delivery, "intent_id": adopted.Intent, "accepted_revision": adopted.ID, "scope_digest": adopted.Digest()}},
		"result":  map[string]any{"phase_result": map[string]any{"output": map[string]any{"verification": map[string]any{"outcome": "verified"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	write(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,0,$2,'finished',0,$3)`, assignmentID, factory.NewID(), string(assignment))
	postRaw, err := json.Marshal(map[string]any{"actor": "factory", "event_identity": delivery, "assignment": assignmentID, "communication": map[string]any{"component_id": "F10"}})
	if err != nil {
		t.Fatal(err)
	}
	write(`INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data) VALUES($1,7,0,'factory','verified',$2,$3)`, factory.NewID(), "f10-http-"+delivery, string(postRaw))
	view = decode(call(http.MethodGet, path, ""))
	if view.State != "accepted" || !view.CanReview || view.AdoptedRevision != adopted.ID || view.ScopeDigest != adopted.Digest() {
		t.Fatalf("adopted lineage over HTTP=%+v", view)
	}
	if !view.Verified || view.EvidenceStatus != store.CommentProductEvidenceRecorded || len(view.Evidence) != 1 || view.Evidence[0].TaskKey != financeTask.Key || view.Evidence[0].Completion == "" {
		t.Fatalf("recorded delivery over HTTP=%+v", view)
	}
	status, replyBody = call(http.MethodGet, "/api/factory/feed/replies?repository=7&source_kind=post&source_id="+posts[0].ID, "")
	if status != http.StatusOK || json.Unmarshal(replyBody, &replies) != nil || len(replies.Items) != 1 || replies.Items[0].Change == nil || !replies.Items[0].Change.Verified {
		t.Fatalf("recorded thread over HTTP=%d %s", status, replyBody)
	}
	decodeOther := func(code int, body []byte) store.CommentProductChange {
		t.Helper()
		var response struct {
			Change store.CommentProductChange `json:"product_change"`
			Issue  int64                      `json:"issue"`
		}
		if code != http.StatusOK || json.Unmarshal(body, &response) != nil || response.Change.AdoptedRevision != adopted.ID {
			t.Fatalf("collaborator lineage=%d %s", code, body)
		}
		return response.Change
	}
	collaboratorView := decodeOther(other(http.MethodGet, path, ""))
	if collaboratorView.State != "accepted" || collaboratorView.CanReview || !collaboratorView.Verified || len(collaboratorView.Evidence) != 1 {
		t.Fatalf("collaborator lineage authority=%+v", collaboratorView)
	}
	// A stale review against the adopted head stays a 412 and replays keep
	// the adopted head in place.
	if status, _ := call(http.MethodPost, path+"/review", `{"revision":1}`); status != http.StatusPreconditionFailed {
		t.Fatalf("stale source revision after adoption=%d", status)
	}
	if status, _ := other(http.MethodPost, path+"/review", `{"revision":0}`); status != http.StatusForbidden {
		t.Fatalf("collaborator review after adoption=%d", status)
	}
	view = decode(call(http.MethodPost, path+"/review", `{"revision":0}`))
	if view.DraftRevision != firstDraft {
		t.Fatal("replayed review moved the adopted head")
	}
	adoptedHead, err := db.IntentRevision(ctx, accepted.Intent, 1)
	if err != nil || adoptedHead.ID != adopted.ID {
		t.Fatalf("replay rolled back adoption: %+v %v", adoptedHead, err)
	}
	// A same-product comment whose product read misses its intent head
	// projects the lane explicitly: state unavailable with the accepted
	// source intact, never a detached generic pipeline.
	brokenSource, err := db.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, posts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	broken := *brokenSource
	broken.Product = &factory.CommentProductSource{Intent: "d999999999999999999999999", AcceptedRevision: "d999999999999999999999999"}
	brokenID, _, err := db.SaveFeedCommentWithSource(ctx, factory.CommentRequest{Repository: 7, Body: "Unknown product follow-up.", SourceKind: broken.Kind, SourceID: broken.ID, IdempotencyKey: "finance-unknown"}, "1", "Ada", 2, &broken)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecordCommentIntake(ctx, brokenID, 0); err != nil {
		t.Fatal(err)
	}
	brokenPipe, err := db.CommentPipeline(ctx, brokenID)
	if err != nil {
		t.Fatal(err)
	}
	brokenRouted := brokenPipe
	brokenRouted.Judgment, brokenRouted.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err := db.AdvanceCommentPipeline(ctx, brokenPipe, brokenRouted); err != nil {
		t.Fatal(err)
	}
	decodeBroken := func(code int, body []byte) store.CommentProductChange {
		t.Helper()
		var response struct {
			Change store.CommentProductChange `json:"product_change"`
			Issue  int64                      `json:"issue"`
		}
		if code != http.StatusOK || json.Unmarshal(body, &response) != nil || response.Change.State != store.CommentProductStateUnavailable {
			t.Fatalf("unavailable projection=%d %s", code, body)
		}
		if response.Issue != 0 {
			t.Fatalf("unavailable lane left a native issue: %d", response.Issue)
		}
		return response.Change
	}
	brokenPath := "/api/factory/feed/comments/" + brokenID
	unavailable := decodeBroken(call(http.MethodGet, brokenPath, ""))
	if unavailable.Intent != broken.Product.Intent || unavailable.AcceptedRevision != broken.Product.AcceptedRevision || unavailable.CanReview || unavailable.Verified || unavailable.EvidenceStatus != store.CommentProductEvidenceUnavailable || len(unavailable.Evidence) != 0 {
		t.Fatalf("unavailable projection claims work: %+v", unavailable)
	}
	unavailableOther := decodeBroken(other(http.MethodGet, brokenPath, ""))
	if unavailableOther.CanReview {
		t.Fatal("collaborator gained unavailable review control")
	}
	status, replyBody = call(http.MethodGet, "/api/factory/feed/replies?repository=7&source_kind=post&source_id="+posts[0].ID, "")
	if status != http.StatusOK || json.Unmarshal(replyBody, &replies) != nil || len(replies.Items) != 2 {
		t.Fatalf("unavailable thread over HTTP=%d %s", status, replyBody)
	}
}

// TestIntentReviewReadCurrentStaleMissing proves owner-private exact-revision
// reads: current at the result head, stale past it, honestly missing without
// a stored presentation. Reads perform no judgments and consume no funding.
func TestIntentReviewReadCurrentStaleMissing(t *testing.T) {
	product, _, call, errorCode := q05Mux(t, "")
	const draftID = "d565656565656565656565656"
	const intentID = "d676767676767676767676767"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d787878787878787878787878","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	allowance := `{"id":"d898989898989898989898989","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", allowance); code != http.StatusCreated {
		t.Fatalf("allowance save status=%d body=%s", code, body)
	}
	chooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = chooser
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("explore status=%d body=%s", code, body)
	}
	var responded exploreResponse
	if err := json.Unmarshal(body, &responded); err != nil {
		t.Fatal(err)
	}
	consumed := func() int {
		t.Helper()
		code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", "")
		if code != http.StatusOK {
			t.Fatalf("allowance read status=%d body=%s", code, body)
		}
		var a factory.PlanningAllowance
		if err := json.Unmarshal(body, &a); err != nil {
			t.Fatal(err)
		}
		return a.Consumed
	}
	read := func(path string) (int, factory.SavedReviewRead) {
		t.Helper()
		code, body := call(http.MethodGet, path, "")
		if code != http.StatusOK {
			t.Fatalf("review read status=%d body=%s", code, body)
		}
		var out factory.SavedReviewRead
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatal(err)
		}
		if err := out.Validate(); err != nil {
			t.Fatalf("review read invalid: %v", err)
		}
		return code, out
	}
	calls, spent := chooser.calls, consumed()
	_, current := read("/api/factory/intents/" + intentID + "/review")
	if current.Status != factory.ReviewCurrent || current.Review == nil || current.Review.ResultRevision != responded.ID || current.Review.BasisRevision != draftID {
		t.Fatalf("current read wrong: %+v", current)
	}
	if current.Review.Content.Objective != "Time-off requests my manager can approve" || current.Review.Recommendation == nil {
		t.Fatalf("current review lost content: %+v", current.Review)
	}
	_, explicit := read("/api/factory/intents/" + intentID + "/review?revision=" + responded.ID)
	if explicit.Status != factory.ReviewCurrent || explicit.Review.ResultRevision != responded.ID {
		t.Fatalf("explicit current read wrong: %+v", explicit)
	}
	const nextID = "d909090909090909090909090"
	next := `{"id":"` + nextID + `","intent":"` + intentID + `","predecessor":"` + responded.ID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", next); code != http.StatusCreated {
		t.Fatalf("next save status=%d body=%s", code, body)
	}
	_, stale := read("/api/factory/intents/" + intentID + "/review?revision=" + responded.ID)
	if stale.Status != factory.ReviewStale || stale.HeadRevision != nextID || stale.Review.ResultRevision != responded.ID {
		t.Fatalf("stale read wrong: %+v", stale)
	}
	_, missing := read("/api/factory/intents/" + intentID + "/review")
	if missing.Status != factory.ReviewMissing || missing.HeadRevision != nextID || missing.Review != nil {
		t.Fatalf("missing read wrong: %+v", missing)
	}
	_, basis := read("/api/factory/intents/" + intentID + "/review?revision=" + draftID)
	if basis.Status != factory.ReviewMissing {
		t.Fatalf("basis read wrong: %+v", basis)
	}
	if chooser.calls != calls || consumed() != spent {
		t.Fatalf("reads judged or spent: calls=%d spent=%d", chooser.calls, consumed())
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/review?revision=nope", ""); code != http.StatusBadRequest || errorCode(body) != "invalid_revision" {
		t.Fatalf("bad revision status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/d999999999999999999999999/review", ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("unknown intent status=%d body=%s", code, body)
	}
}

// TestIntentReviewCrossIntentRevisionDenied proves a stored review never
// answers under another intent, even for the same owner.
func TestIntentReviewCrossIntentRevisionDenied(t *testing.T) {
	product, _, call, errorCode := q05Mux(t, "")
	explore := func(draftID, intentID, ideaID, allowanceID string) string {
		t.Helper()
		draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
			`"ideas":[{"id":"` + ideaID + `","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}]}`
		if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
			t.Fatalf("draft save status=%d body=%s", code, body)
		}
		allowance := `{"id":"` + allowanceID + `","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
		if code, body := call(http.MethodPost, "/api/factory/intent-allowances", allowance); code != http.StatusCreated {
			t.Fatalf("allowance save status=%d body=%s", code, body)
		}
		product.IntentExploreChooser = &exploreFixtureChooser{}
		code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
		if code != http.StatusCreated {
			t.Fatalf("explore status=%d body=%s", code, body)
		}
		var responded exploreResponse
		if err := json.Unmarshal(body, &responded); err != nil {
			t.Fatal(err)
		}
		return responded.ID
	}
	const intentA = "daaaaaaaaaaaaaaaaaaaaaaaa"
	resultB := explore("dabababababababababababab", "dbbcbbbbbbbbbbbbbbbbbbbbb", "dacacacacacacacacacacacac", "dadadadadadadadadadadadad")
	explore("dbcbababababababababababa", intentA, "dbacacacacacacacacacacaca", "dbadadadadadadadadadadada")
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentA+"/review?revision="+resultB, ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("cross-intent status=%d body=%s", code, body)
	}
}

// TestSavedReviewJoinExploreSaveRead proves the joined round with injected
// judgments: Explore atomically saves, a lost response recovers through a
// fresh private read, stale and missing cases stay honest, another actor is
// denied, and reads consume no judgments, move no heads and replay no
// generation or effects.
func TestSavedReviewJoinExploreSaveRead(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	ctx := context.Background()
	const draftID = "d717171717171717171717171"
	const intentID = "d727272727272727272727272"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[` +
		`{"id":"d737373737373737373737373","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true},` +
		`{"id":"d747474747474747474747474","description":"Keep history visible","consequence":"Past requests stay listed","decision":"pending","source":"autonomous","provenance":"autonomous"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	allowance := `{"id":"d757575757575757575757575","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", allowance); code != http.StatusCreated {
		t.Fatalf("allowance save status=%d body=%s", code, body)
	}
	chooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = chooser
	code, _ := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("explore status=%d", code)
	}
	consumed := func() (int, int) {
		t.Helper()
		code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", "")
		if code != http.StatusOK {
			t.Fatalf("allowance read status=%d body=%s", code, body)
		}
		var a factory.PlanningAllowance
		if err := json.Unmarshal(body, &a); err != nil {
			t.Fatal(err)
		}
		return a.Consumed, chooser.calls
	}
	spent, calls := consumed()
	if spent == 0 || calls == 0 {
		t.Fatal("round consumed no judgments")
	}
	head, err := s.IntentRevision(ctx, intentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	history, err := s.IntentHistory(ctx, intentID, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) factory.SavedReviewRead {
		t.Helper()
		code, body := call(http.MethodGet, path, "")
		if code != http.StatusOK {
			t.Fatalf("review read status=%d body=%s", code, body)
		}
		var out factory.SavedReviewRead
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatal(err)
		}
		if err := out.Validate(); err != nil {
			t.Fatalf("live read invalid: %v", err)
		}
		return out
	}
	// The lost Explore response recovers through a fresh private read.
	current := read("/api/factory/intents/" + intentID + "/review")
	if current.Status != factory.ReviewCurrent || current.Review == nil {
		t.Fatalf("lost response did not recover: %+v", current)
	}
	review := current.Review
	if review.BasisRevision != draftID || review.ResultRevision != head.ID || current.HeadRevision != head.ID {
		t.Fatalf("recovered provenance wrong: %+v", current)
	}
	if review.Recommendation == nil || len(review.Recommendation.Milestones) == 0 || len(review.Recommendation.Alternative.Steps) == 0 {
		t.Fatal("recovered review lost the illustrated recommendation")
	}
	if len(review.Content.Unknowns) == 0 || len(review.Settled) != 1 || review.Settled[0] != "d747474747474747474747474" {
		t.Fatalf("recovered review lost content or verdicts: %+v", review)
	}
	currentRaw, _ := json.Marshal(review)
	const nextID = "d767676767676767676767676"
	next := `{"id":"` + nextID + `","intent":"` + intentID + `","predecessor":"` + head.ID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", next); code != http.StatusCreated {
		t.Fatalf("next save status=%d body=%s", code, body)
	}
	stale := read("/api/factory/intents/" + intentID + "/review?revision=" + head.ID)
	if stale.Status != factory.ReviewStale {
		t.Fatalf("moved head read wrong: %+v", stale)
	}
	staleRaw, _ := json.Marshal(stale.Review)
	if string(staleRaw) != string(currentRaw) {
		t.Fatal("stored presentation changed across the head move")
	}
	if missing := read("/api/factory/intents/" + intentID + "/review"); missing.Status != factory.ReviewMissing || missing.Review != nil {
		t.Fatalf("new head read wrong: %+v", missing)
	}
	if historical := read("/api/factory/intents/" + intentID + "/review?revision=" + draftID); historical.Status != factory.ReviewMissing {
		t.Fatalf("historical read wrong: %+v", historical)
	}
	other := actorCall(t, product, "2", "bob")
	if code, body := other(http.MethodGet, "/api/factory/intents/"+intentID+"/review", ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("other actor status=%d body=%s", code, body)
	}
	if spentAfter, callsAfter := consumed(); spentAfter != spent || callsAfter != calls {
		t.Fatalf("reads consumed judgments: spent=%d calls=%d", spentAfter, callsAfter)
	}
	after, err := s.IntentRevision(ctx, intentID, 1)
	if err != nil || after.ID != nextID {
		t.Fatalf("reads moved the head: %+v %v", after, err)
	}
	historyAfter, err := s.IntentHistory(ctx, intentID, 1, 50)
	if err != nil || len(historyAfter) != len(history)+1 {
		t.Fatalf("reads changed history: %d vs %d", len(historyAfter), len(history))
	}
	if _, err := s.IntentGeneration(ctx, head.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("read replayed generation: %v", err)
	}
}

// TestSavedReviewJoinCaptureRound records one live Explore→save→read round as
// ignored JSON for the production Lit restoration fixture. The files hold API
// response bodies only: synthetic IDs, no requests, no auth material.
func TestSavedReviewJoinCaptureRound(t *testing.T) {
	product, _, call, _ := q05Mux(t, "")
	const draftID = "d818181818181818181818181"
	const intentID = "d828282828282828282828282"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[` +
		`{"id":"d838383838383838383838383","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true},` +
		`{"id":"d848484848484848484848484","description":"Keep history visible","consequence":"Past requests stay listed","decision":"pending","source":"autonomous","provenance":"autonomous"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	allowance := `{"id":"d858585858585858585858585","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", allowance); code != http.StatusCreated {
		t.Fatalf("allowance save status=%d body=%s", code, body)
	}
	product.IntentExploreChooser = &exploreFixtureChooser{}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`); code != http.StatusCreated {
		t.Fatalf("explore status=%d body=%s", code, body)
	}
	get := func(path string) []byte {
		t.Helper()
		code, body := call(http.MethodGet, path, "")
		if code != http.StatusOK {
			t.Fatalf("capture read %s status=%d body=%s", path, code, body)
		}
		return body
	}
	revision := get("/api/factory/intents/" + intentID)
	review := get("/api/factory/intents/" + intentID + "/review")
	var read factory.SavedReviewRead
	if err := json.Unmarshal(review, &read); err != nil {
		t.Fatal(err)
	}
	if err := read.Validate(); err != nil {
		t.Fatalf("captured read invalid: %v", err)
	}
	if read.Status != factory.ReviewCurrent || read.Review == nil || read.Review.Recommendation == nil ||
		len(read.Review.Recommendation.Milestones) == 0 || len(read.Review.Recommendation.Alternative.Steps) == 0 ||
		len(read.Review.Challenges)+len(read.Review.Settled) == 0 {
		t.Fatalf("captured round has nothing to illustrate: %+v", read)
	}
	dir := "../../../.artifacts/g07-join"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "revision.json"), revision, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review.json"), review, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestIntentReviewOtherActorDenied proves reviews stay owner-private: another
// actor cannot read them.
func TestIntentReviewOtherActorDenied(t *testing.T) {
	product, _, call, errorCode := q05Mux(t, "")
	const draftID = "d565656565656565656565657"
	const intentID = "d676767676767676767676768"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d787878787878787878787879","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	allowance := `{"id":"d898989898989898989898980","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", allowance); code != http.StatusCreated {
		t.Fatalf("allowance save status=%d body=%s", code, body)
	}
	product.IntentExploreChooser = &exploreFixtureChooser{}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`); code != http.StatusCreated {
		t.Fatalf("explore status=%d body=%s", code, body)
	}
	other := actorCall(t, product, "2", "bob")
	if code, body := other(http.MethodGet, "/api/factory/intents/"+intentID+"/review", ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("other actor status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/review", ""); code != http.StatusOK {
		t.Fatalf("owner read after other actor status=%d body=%s", code, body)
	}
}

// downgradableActorCall serves the product mux as one fixed native actor
// with a shared mutable repository permission, so a test can move the actor
// between admin and read through the real visibility path mid-test.
func downgradableActorCall(t *testing.T, product *API, actorID, username string, permission *atomic.Value) func(method, path, body string) (int, []byte) {
	t.Helper()
	callbackPath := "@soda-downgrade-" + actorID + "-" + strconv.Itoa(os.Getpid())
	listener, err := net.Listen("unix", callbackPath)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		switch in.Operation {
		case extensions.OperationCurrentActor:
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: actorID, Username: username}})
		case extensions.OperationRepository:
			if in.RepositoryID != "7" {
				http.Error(w, "not visible", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "Alpha", Permission: permission.Load().(string)}})
		default:
			http.Error(w, "denied", http.StatusForbidden)
		}
	}))
	callback.Listener.Close()
	callback.Listener = listener
	callback.Start()
	t.Cleanup(callback.Close)
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)
	return func(method, path, body string) (int, []byte) {
		t.Helper()
		authority, err := json.Marshal(extensions.Authority{
			ExtensionID: "soda", InstanceID: "downgrade-test", SessionGeneration: "generation-1",
			Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)},
			Actor:        extensions.Actor{ID: actorID, Username: username},
		})
		if err != nil {
			t.Fatal(err)
		}
		previous, existed := os.LookupEnv(extensions.CallbackEnv)
		if err := os.Setenv(extensions.CallbackEnv, callbackPath); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if existed {
				_ = os.Setenv(extensions.CallbackEnv, previous)
			} else {
				_ = os.Unsetenv(extensions.CallbackEnv)
			}
		}()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		if method != http.MethodGet && method != http.MethodHead {
			r.Header.Set("Origin", "https://soda.test")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code, w.Body.Bytes()
	}
}

// TestCommentProductAcceptedOwnerReadKeepsAction proves the owner-read
// capability separation on the real accepted API projection path. The intent
// owner (bob, not the repository owner) is downgraded to repository Read
// after the accepted descendant lands: the private GET
// /factory/intents/{id} still reads 200 without admin, and the accepted Feed
// action stays on through the comment and replies projections. A
// collaborator stays denied both the action and the private request.
func TestCommentProductAcceptedOwnerReadKeepsAction(t *testing.T) {
	product, db, _, errorCode := q05Mux(t, "")
	ctx := context.Background()
	permission := &atomic.Value{}
	permission.Store("admin")
	owner := downgradableActorCall(t, product, "2", "bob", permission)
	collaborator := downgradableActorCall(t, product, "3", "carol", permission)
	idea := factory.ProductIdea{ID: "d585858585858585858585858", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 2}
	accepted := factory.IntentRevision{ID: "d595959595959595959595959", Intent: "d505050505050505050505050", Owner: 2, Repository: 7,
		Objective: "Employees request equipment and Operations records delivery", State: "accepted", MaxIssues: 10, PlanningCalls: 8, CreatedUnix: 1729000000,
		Ideas:             []factory.ProductIdea{idea},
		EffectiveOutcomes: []string{idea.ID},
		Tasks:             []factory.IntentTask{{Key: "d515151515151515151515151", Objective: "Deliver equipment workflow", Outcomes: []string{"Recorded request history"}, IdeaIDs: []string{idea.ID}}}}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	posts, _, _, err := db.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10})
	if err != nil || len(posts) != 1 {
		t.Fatalf("accepted source posts=%d err=%v", len(posts), err)
	}
	code, body := owner(http.MethodPost, "/api/factory/feed/comments", `{"repository":7,"body":"New requests above EUR 1000 also need Finance approval. Existing requests keep their route.","source_kind":"post","source_id":"`+posts[0].ID+`","reply_to":"`+posts[0].ID+`","idempotency_key":"owner-read-change"}`)
	if code != http.StatusOK {
		t.Fatalf("comment save=%d %s", code, body)
	}
	var saved struct {
		Comment string `json:"comment_id"`
	}
	if err := json.Unmarshal(body, &saved); err != nil || saved.Comment == "" {
		t.Fatalf("comment identity=%s err=%v", body, err)
	}
	pipe, err := db.CommentPipeline(ctx, saved.Comment)
	if err != nil {
		t.Fatal(err)
	}
	routed := pipe
	routed.Judgment, routed.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err := db.AdvanceCommentPipeline(ctx, pipe, routed); err != nil {
		t.Fatal(err)
	}
	drafter := &fakeCommentDrafter{draft: validTestDraft(saved.Comment)}
	creator := &fakeCommentCreator{next: 100, issues: map[string]int64{}}
	response, err := driveCommentPipeline(ctx, db, &CommentCollaborators{Drafter: drafter, Creator: creator}, saved.Comment)
	if err != nil || response.Issue != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatalf("product proposal created detached work: response=%+v drafts=%d creates=%d err=%v", response, drafter.calls, creator.creates, err)
	}
	path := "/api/factory/feed/comments/" + saved.Comment
	decode := func(code int, body []byte) store.CommentProductChange {
		t.Helper()
		var response struct {
			Change store.CommentProductChange `json:"product_change"`
			Issue  int64                      `json:"issue"`
		}
		if code != http.StatusOK || json.Unmarshal(body, &response) != nil || response.Change.Intent != accepted.Intent || response.Change.AcceptedRevision != accepted.ID {
			t.Fatalf("change projection=%d %s", code, body)
		}
		if response.Issue != 0 {
			t.Fatalf("same-product pipeline left its zero issue: %d", response.Issue)
		}
		return response.Change
	}
	view := decode(owner(http.MethodPost, path+"/review", `{"revision":0}`))
	if view.State != "draft" || !view.CanReview || view.DraftRevision == "" {
		t.Fatalf("draft continuation=%+v", view)
	}
	head, err := db.IntentRevision(ctx, accepted.Intent, 2)
	if err != nil || head.State != "draft" {
		t.Fatalf("review head=%+v err=%v", head, err)
	}
	adopted := head
	adopted.ID, adopted.Predecessor, adopted.State = "d525252525252525252525252", head.ID, "accepted"
	if err := db.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	view = decode(owner(http.MethodGet, path, ""))
	if view.State != "accepted" || !view.CanReview {
		t.Fatalf("accepted owner action while admin=%+v", view)
	}
	// Actual role downgrade: repository Read, no admin anywhere now.
	permission.Store("read")
	if code, body := owner(http.MethodGet, "/api/factory/intents/"+accepted.Intent, ""); code != http.StatusOK {
		t.Fatalf("owner read of private request after downgrade=%d %s", code, body)
	}
	view = decode(owner(http.MethodGet, path, ""))
	if view.State != "accepted" || !view.CanReview {
		t.Fatalf("downgraded owner lost accepted action=%+v", view)
	}
	status, replyBody := owner(http.MethodGet, "/api/factory/feed/replies?repository=7&source_kind=post&source_id="+posts[0].ID, "")
	var replies struct {
		Items []struct {
			Change *store.CommentProductChange `json:"product_change"`
		} `json:"items"`
	}
	if status != http.StatusOK || json.Unmarshal(replyBody, &replies) != nil || len(replies.Items) != 1 || replies.Items[0].Change == nil || replies.Items[0].Change.State != "accepted" || !replies.Items[0].Change.CanReview {
		t.Fatalf("downgraded owner thread projection=%d %s", status, replyBody)
	}
	view = decode(collaborator(http.MethodGet, path, ""))
	if view.State != "accepted" || view.CanReview {
		t.Fatalf("collaborator gained accepted action=%+v", view)
	}
	if code, body := collaborator(http.MethodGet, "/api/factory/intents/"+accepted.Intent, ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("collaborator private read=%d %s", code, body)
	}
}
