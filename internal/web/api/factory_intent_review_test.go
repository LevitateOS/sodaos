package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
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
		if method != http.MethodGet && method != http.MethodHead {
			r.Header.Set("Origin", product.Config.ForgejoURL)
		}
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
	product, db, _, call, _ := q05MuxFull(t, "")
	ctx := context.Background()
	accepted := factory.IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 7,
		Objective: "Employees request equipment and Operations records delivery", State: "accepted", MaxIssues: 10, PlanningCalls: 8, CreatedUnix: 1729000000,
		Ideas:             []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Keep request and approval history", Consequence: "Original records remain readable", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		EffectiveOutcomes: []string{"d333333333333333333333333"}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Deliver equipment workflow", Outcomes: []string{"Recorded request history"}, IdeaIDs: []string{"d333333333333333333333333"}}},
	}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	posts, _, _, err := db.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10})
	if err != nil || len(posts) != 1 {
		t.Fatalf("accepted source posts=%d err=%v", len(posts), err)
	}
	code, body := call(http.MethodPost, "/api/factory/feed/posts", `{"repository":"7","body":"New requests above EUR 1000 also need Finance approval. Existing requests keep their route.","parent_id":"`+posts[0].ID+`","idempotency_key":"finance-change"}`)
	if code != http.StatusOK {
		t.Fatalf("comment save=%d %s", code, body)
	}
	var saved struct {
		SourcePost string `json:"post_id"`
	}
	if err := json.Unmarshal(body, &saved); err != nil || saved.SourcePost == "" {
		t.Fatalf("comment identity=%s err=%v", body, err)
	}
	pipe, err := db.PostIntake(ctx, saved.SourcePost)
	if err != nil {
		t.Fatal(err)
	}
	// The external routing judgment is a declared fixture; the HTTP and
	// transaction boundaries below use the real mux and PostgreSQL owners.
	routed := pipe
	routed.Judgment, routed.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
	if err := db.AdvancePostIntake(ctx, pipe, routed); err != nil {
		t.Fatal(err)
	}
	drafter := &fakePostDrafter{draft: validTestDraft(saved.SourcePost)}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	response, err := factoryservice.DrivePostIntake(ctx, db, &factoryservice.PostCollaborators{Drafter: drafter, Creator: creator}, saved.SourcePost)
	if err != nil || response.Issue != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatalf("product proposal created detached work: response=%+v drafts=%d creates=%d err=%v", response, drafter.calls, creator.creates, err)
	}
	path := "/api/factory/feed/posts/" + saved.SourcePost
	decode := func(code int, body []byte) store.PostProductChange {
		t.Helper()
		var response struct {
			Change store.PostProductChange `json:"product_change"`
			Issue  int64                   `json:"issue"`
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
	shared := factoryservice.Posts{Store: db, Visibility: func(context.Context, collaboration.ActorID, int64) error { return nil }}
	if _, err := shared.ReviewProduct(ctx, 1, saved.SourcePost, 0); !errors.Is(err, factoryservice.ErrRequestWriteUnavailable) {
		t.Fatal("shared product review invented write authority", err)
	}
	denied := errors.New("declared current product-review authority withdrawn")
	shared.WriteAccess = func(context.Context, collaboration.ActorID, int64) error { return denied }
	if _, err := shared.ReviewProduct(ctx, 1, saved.SourcePost, 0); !errors.Is(err, factoryservice.ErrRequestWriteAccess) || !errors.Is(err, denied) {
		t.Fatal("shared product review bypassed withdrawn write authority", err)
	}
	if current, err := db.IntentRevision(ctx, accepted.Intent, 1); err != nil || current.ID != accepted.ID {
		t.Fatal("refused shared product review advanced the accepted request", current.ID, err)
	}
	other := actorCall(t, product, "2", "bob")
	view = decode(other(http.MethodGet, path, ""))
	if view.CanReview {
		t.Fatal("another native repository admin can review the owner's private intent")
	}
	if status, _ := other(http.MethodPost, path+"/review", `{"revision":0}`); status != http.StatusNotFound {
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
	firstDraft := view.DraftRevision
	view = decode(call(http.MethodPost, path+"/review", `{"revision":0}`))
	if view.DraftRevision != firstDraft {
		t.Fatal("replayed review made a second draft")
	}
	view = decode(other(http.MethodGet, path, ""))
	if view.CanReview {
		t.Fatal("another actor can continue the owner's draft")
	}
	status, replyBody := call(http.MethodGet, "/api/factory/feed/posts/"+posts[0].ID+"/replies?repository=7", "")
	var replies struct {
		Items []struct {
			Change *store.PostProductChange `json:"product_change"`
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
	// Adopt the reviewed change: the owner may open the accepted direction,
	// a collaborator sees it read-only.
	commentIdea := factory.PostDecisionID("product-idea", saved.SourcePost, 0)
	adopted := head
	financeTask := factory.IntentTask{Key: "d666666666666666666666666", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{commentIdea}}
	adopted.ID, adopted.Predecessor, adopted.State = "d555555555555555555555555", head.ID, "accepted"
	adopted.Tasks = append(append([]factory.IntentTask(nil), adopted.Tasks...), financeTask)
	if err := db.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	view = decode(call(http.MethodGet, path, ""))
	if view.State != "accepted" || !view.CanReview {
		t.Fatalf("adopted change over HTTP=%+v", view)
	}
	if collaboratorView := decode(other(http.MethodGet, path, "")); collaboratorView.State != "accepted" || collaboratorView.CanReview {
		t.Fatalf("collaborator authority=%+v", collaboratorView)
	}
	// A stale review against the adopted head stays a 412 and replays keep
	// the adopted head in place.
	if status, _ := call(http.MethodPost, path+"/review", `{"revision":1}`); status != http.StatusPreconditionFailed {
		t.Fatalf("stale source revision after adoption=%d", status)
	}
	if status, _ := other(http.MethodPost, path+"/review", `{"revision":0}`); status != http.StatusNotFound {
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
	brokenSource, err := db.ResolvePostSource(ctx, 7, factory.PostSourcePost, posts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	broken := *brokenSource
	broken.Product = &factory.PostProductSource{Intent: "d999999999999999999999999", AcceptedRevision: "d999999999999999999999999"}
	brokenID, _, err := db.SaveFeedPostWithSource(ctx, factory.PostRequest{Repository: 7, Body: "Unknown product follow-up.", SourceKind: broken.Kind, SourceID: broken.ID, IdempotencyKey: "finance-unknown"}, "1", "Ada", 2, &broken)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecordPostIntake(ctx, brokenID, 0); err != nil {
		t.Fatal(err)
	}
	brokenPipe, err := db.PostIntake(ctx, brokenID)
	if err != nil {
		t.Fatal(err)
	}
	brokenRouted := brokenPipe
	brokenRouted.Judgment, brokenRouted.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
	if err := db.AdvancePostIntake(ctx, brokenPipe, brokenRouted); err != nil {
		t.Fatal(err)
	}
	decodeBroken := func(code int, body []byte) store.PostProductChange {
		t.Helper()
		var response struct {
			Change store.PostProductChange `json:"product_change"`
			Issue  int64                   `json:"issue"`
		}
		if code != http.StatusOK || json.Unmarshal(body, &response) != nil || response.Change.State != store.PostProductStateUnavailable {
			t.Fatalf("unavailable projection=%d %s", code, body)
		}
		if response.Issue != 0 {
			t.Fatalf("unavailable lane left a native issue: %d", response.Issue)
		}
		return response.Change
	}
	brokenPath := "/api/factory/feed/posts/" + brokenID
	unavailable := decodeBroken(call(http.MethodGet, brokenPath, ""))
	if unavailable.Intent != broken.Product.Intent || unavailable.AcceptedRevision != broken.Product.AcceptedRevision || unavailable.CanReview {
		t.Fatalf("unavailable projection claims work: %+v", unavailable)
	}
	unavailableOther := decodeBroken(other(http.MethodGet, brokenPath, ""))
	if unavailableOther.CanReview {
		t.Fatal("collaborator gained unavailable review control")
	}
	status, replyBody = call(http.MethodGet, "/api/factory/feed/posts/"+posts[0].ID+"/replies?repository=7", "")
	if status != http.StatusOK || json.Unmarshal(replyBody, &replies) != nil || len(replies.Items) != 2 {
		t.Fatalf("unavailable thread over HTTP=%d %s", status, replyBody)
	}
}

// TestIntentReviewReadCurrentStaleMissing proves owner-private exact-revision
// reads: current at the result head, stale past it, honestly missing without
// a stored presentation. Reads perform no judgments and consume no funding.
func TestIntentReviewReadCurrentStaleMissing(t *testing.T) {
	product, st, dsn, call, errorCode := q05MuxFull(t, "")
	const draftID = "d565656565656565656565656"
	const intentID = "d676767676767676767676767"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d787878787878787878787878","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}]}`
	originalCode, originalReceipt := call(http.MethodPost, "/api/factory/intents", draft)
	if originalCode != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", originalCode, originalReceipt)
	}
	chooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = chooser
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("explore status=%d body=%s", code, body)
	}
	var responded factoryservice.ExplorationResult
	if err := json.Unmarshal(body, &responded); err != nil {
		t.Fatal(err)
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
		parsed, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		requests := factoryservice.Requests{Store: st}
		direct, err := requests.Review(t.Context(), 1, intentID, parsed.Query().Get("revision"))
		directBytes, _ := json.Marshal(direct)
		httpBytes, _ := json.Marshal(out)
		if err != nil || string(directBytes) != string(httpBytes) {
			t.Fatal("standalone saved-review read changed the recorded answer", err)
		}
		view, err := requests.Overview(t.Context(), 1, intentID)
		if err != nil || view.Setup.State != factoryservice.RequestSetupUnassigned || view.Revision != out.HeadRevision {
			t.Fatal("private unbound request required a native setup host", view, err)
		}
		return code, out
	}
	calls := chooser.calls
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
	if chooser.calls != calls {
		t.Fatalf("reads judged: calls=%d", chooser.calls)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/review?revision=nope", ""); code != http.StatusBadRequest || errorCode(body) != "invalid_revision" {
		t.Fatalf("bad revision status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/d999999999999999999999999/review", ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("unknown intent status=%d body=%s", code, body)
	}
	// A direct HTTP host consumes the same request transport without an API or
	// native session. Its declared principal owns private, unassigned direction.
	ctx := t.Context()
	inspection, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer inspection.Close()
	workSnapshot := func() string {
		var value string
		if err := inspection.QueryRowContext(ctx, `SELECT jsonb_build_array((SELECT count(*) FROM factory_runs),(SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_usage),(SELECT count(*) FROM factory_priority_judgments))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	beforeWork := workSnapshot()
	const nativeActor collaboration.ActorID = 9007199254740993
	var checks, revokeAt atomic.Int32
	shared := factoryhttpapi.Requests{Service: factoryservice.Requests{Store: st},
		Host: product.FountainIdentities.HostID(), Actor: nativeActor,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			count := checks.Add(1)
			if r.Header.Get("Authorization") != "Bearer declared-request-client" || revokeAt.Load() > 0 && count >= revokeAt.Load() {
				http.Error(w, "declared account admission unavailable", 401)
				return false
			}
			return true
		}}
	server := httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		r, err := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-request-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != status {
			t.Fatalf("shared request HTTP %s %s: %d %s %v", method, path, response.StatusCode, data, err)
		}
		if status < 300 && response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("private request response was cacheable")
		}
		return data
	}
	department := "d" + strings.Repeat("e", 24)
	created := request(http.MethodPost, "/api/factory/departments", `{"id":"`+department+`","owner":"1","name":"Native requests","description":"Private request grouping"}`, 201)
	if !strings.Contains(string(created), `"owner":"9007199254740993"`) || !strings.Contains(string(request(http.MethodGet, "/api/factory/departments", "", 200)), department) {
		t.Fatal("request HTTP did not bind department ownership", string(created))
	}
	privateIntent := "d" + strings.Repeat("a", 24)
	privateRevision := "d" + strings.Repeat("b", 24)
	input := `{"id":"` + privateRevision + `","intent":"` + privateIntent + `","owner":"1","repository":"0","department":"` + department + `","created_unix":1,"objective":"Private native direction","state":"draft","max_issues":10,"planning_calls":8,"ideas":[{"id":"dcccccccccccccccccccccccc","description":"Keep private request history","consequence":"Owner can revise direction","decision":"pending","assumption":true,"material":true}]}`
	var recorded factory.IntentRevision
	if err := json.Unmarshal(request(http.MethodPost, "/api/factory/intents", input, 201), &recorded); err != nil || recorded.Owner != int64(nativeActor) || recorded.CreatedUnix <= 1 {
		t.Fatal("request HTTP body selected ownership/time", recorded, err)
	}
	request(http.MethodPost, "/api/factory/intents", input, 201)
	requestPath := "/api/factory/intents/" + privateIntent
	request(http.MethodGet, requestPath, "", 200)
	request(http.MethodGet, requestPath+"/review", "", 200)
	if body := request(http.MethodGet, "/api/factory/intents?department="+department+"&owner=1", "", 200); !strings.Contains(string(body), privateIntent) || strings.Contains(string(body), intentID) {
		t.Fatal("request catalogue ignored its bound owner", string(body))
	}
	updated := recorded
	updated.ID, updated.Predecessor, updated.Objective = "d"+strings.Repeat("f", 24), recorded.ID, "Revised private native direction"
	updated.Owner, updated.CreatedUnix = 1, 1
	updateBody, _ := json.Marshal(updated)
	request(http.MethodPost, "/api/factory/intents", string(updateBody), 201)
	staleRevision := updated
	staleRevision.ID = "d" + strings.Repeat("1", 24)
	staleBody, _ := json.Marshal(staleRevision)
	request(http.MethodPost, "/api/factory/intents", string(staleBody), 412)
	request(http.MethodPost, "/api/factory/intents", input+` {}`, 400)
	var history struct {
		Revisions []factory.IntentRevision `json:"revisions"`
	}
	if err := json.Unmarshal(request(http.MethodGet, requestPath+"/history", "", 200), &history); err != nil || len(history.Revisions) != 2 {
		t.Fatal("HTTP replay/stale write forked revision history", len(history.Revisions), err)
	}
	var overview factoryservice.RequestOverview
	if err := json.Unmarshal(request(http.MethodGet, requestPath+"/overview", "", 200), &overview); err != nil || overview.Revision != updated.ID || overview.Setup.State != factoryservice.RequestSetupUnassigned {
		t.Fatal("unassigned HTTP request required native Project setup", overview, err)
	}
	// Advance beyond the history page's maximum without changing the immutable
	// saved revision. A pending save must be recoverable without replaying a stale
	// predecessor or walking an unbounded history chain.
	historical := updated
	for index := int64(0); index < 54; index++ {
		predecessor := historical.ID
		suffix := strconv.FormatInt(4096+index, 16)
		historical.ID = "d" + strings.Repeat("0", 24-len(suffix)) + suffix
		historical.Predecessor = predecessor
		historical.Owner = int64(nativeActor)
		historical.Objective = "Later private native direction " + suffix
		historical.CreatedUnix = int64(2000 + index)
		if err := st.SaveIntentRevision(ctx, historical); err != nil {
			t.Fatal(err)
		}
	}
	var page struct {
		Revisions []factory.IntentRevision `json:"revisions"`
	}
	if err := json.Unmarshal(request(http.MethodGet, requestPath+"/history?limit=50", "", 200), &page); err != nil || len(page.Revisions) != 50 {
		t.Fatal("bounded history did not retain its page limit", len(page.Revisions), err)
	}
	for _, row := range page.Revisions {
		if row.ID == recorded.ID {
			t.Fatal("fixture did not put the original save beyond bounded history")
		}
	}
	snapshot := func() string {
		var value string
		if err := inspection.QueryRowContext(ctx, `SELECT jsonb_build_object('head',
   (SELECT revision FROM factory_intent_heads WHERE id=$1), 'revisions',
   (SELECT jsonb_agg(to_jsonb(r) ORDER BY r.id) FROM factory_intent_revisions r WHERE r.intent=$1))::text`, privateIntent).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	savedState := snapshot()
	receiptPath := requestPath + "/revisions/" + recorded.ID
	var exact factory.IntentRevision
	if err := json.Unmarshal(request(http.MethodGet, receiptPath, "", 200), &exact); err != nil {
		t.Fatal(err)
	}
	expected, _ := json.Marshal(recorded)
	actual, _ := json.Marshal(exact)
	if string(expected) != string(actual) {
		t.Fatal("saved revision lookup returned another input or attribution", string(actual))
	}
	// Owner and intent are current serving bindings, never query/body selectors.
	request(http.MethodGet, receiptPath+"?owner=1", "", 400)
	request(http.MethodGet, receiptPath+"?", "", 400)
	request(http.MethodGet, requestPath+"/revisions/not-a-revision", "", 404)
	request(http.MethodGet, "/api/factory/intents/not-an-intent/revisions/"+recorded.ID, "", 404)
	request(http.MethodGet, requestPath+"/revisions/d999999999999999999999999", "", 404)
	request(http.MethodGet, "/api/factory/intents/"+intentID+"/revisions/"+recorded.ID, "", 404)
	request(http.MethodGet, requestPath+"/revisions/"+draftID, "", 404)
	request(http.MethodPost, receiptPath, "{}", 405)
	checks.Store(0)
	revokeAt.Store(2)
	if denied := request(http.MethodGet, receiptPath, "", 401); strings.Contains(string(denied), recorded.Objective) {
		t.Fatal("revoked account disclosed an immutable receipt")
	}
	revokeAt.Store(0)
	if snapshot() != savedState {
		t.Fatal("saved revision reads changed the head or immutable records")
	}
	// Exercise the actual Fountain private proxy and its native product route,
	// rather than substituting a route-predicate assertion for the wire path.
	nativeMux := http.NewServeMux()
	product.registerExtensionProductRoutes(nativeMux)
	socket := "@soda-saved-revision-" + strconv.Itoa(os.Getpid())
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	native := httptest.NewUnstartedServer(nativeMux)
	native.Listener.Close()
	native.Listener = listener
	native.Start()
	t.Cleanup(native.Close)
	proxy, closeProxy := Extension(socket)
	t.Cleanup(closeProxy)
	authority, err := json.Marshal(extensions.Authority{
		ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1",
		Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "get"},
		Actor:        extensions.Actor{ID: "1", Username: "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	proxyPath := "/factory/intents/" + intentID + "/revisions/" + draftID
	for _, probe := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, proxyPath, 200}, {http.MethodPost, proxyPath, 404},
		{http.MethodGet, proxyPath + "?owner=1", 404}, {http.MethodGet, proxyPath + "/extra", 404},
		{http.MethodGet, "/factory/intents/" + intentID + "/revisions/bad", 404},
	} {
		r := httptest.NewRequest(probe.method, probe.path, nil)
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, r)
		if w.Code != probe.status {
			t.Fatalf("saved revision proxy %s %s: %d %s", probe.method, probe.path, w.Code, w.Body.String())
		}
		if probe.status == 200 && (w.Body.String() != string(originalReceipt) || w.Header().Get("Cache-Control") != "no-store") {
			t.Fatal("Fountain proxy changed or cached the saved revision receipt", w.Body.String())
		}
	}
	native.Close()
	checks.Store(0)
	revokeAt.Store(2)
	if denied := request(http.MethodGet, requestPath, "", 401); strings.Contains(string(denied), recorded.Objective) {
		t.Fatal("revoked admission disclosed private request", string(denied))
	}
	revokeAt.Store(0)
	server.Close()
	shared.Actor = 1
	server = httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	request(http.MethodGet, requestPath, "", 404)
	request(http.MethodGet, requestPath+"/history", "", 404)
	request(http.MethodGet, requestPath+"/review", "", 404)
	request(http.MethodGet, receiptPath, "", 404)
	server.Close()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	shared.Actor, shared.Service.Store = nativeActor, st
	server = httptest.NewServer(shared.Handler())
	t.Cleanup(server.Close)
	var retained factory.IntentRevision
	if err := json.Unmarshal(request(http.MethodGet, requestPath, "", 200), &retained); err != nil || retained.ID != historical.ID || retained.Owner != int64(nativeActor) || retained.CreatedUnix <= 1 {
		t.Fatal("fresh request HTTP handler lost canonical revision", retained, err)
	}
	if err := json.Unmarshal(request(http.MethodGet, receiptPath, "", 200), &exact); err != nil {
		t.Fatal(err)
	}
	actual, _ = json.Marshal(exact)
	if string(expected) != string(actual) || snapshot() != savedState {
		t.Fatal("reopened store/HTTP lost the exact historical save or changed current writing")
	}
	if after := workSnapshot(); after != beforeWork || chooser.calls != calls {
		t.Fatal("request transport drove planning/execution/model work", after, beforeWork, chooser.calls)
	}
}

// TestIntentReviewCrossIntentRevisionDenied proves a stored review never
// answers under another intent, even for the same owner.
func TestIntentReviewCrossIntentRevisionDenied(t *testing.T) {
	product, _, call, errorCode := q05Mux(t, "")
	explore := func(draftID, intentID, ideaID string) string {
		t.Helper()
		draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
			`"ideas":[{"id":"` + ideaID + `","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}]}`
		if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
			t.Fatalf("draft save status=%d body=%s", code, body)
		}
		product.IntentExploreChooser = &exploreFixtureChooser{}
		code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
		if code != http.StatusCreated {
			t.Fatalf("explore status=%d body=%s", code, body)
		}
		var responded factoryservice.ExplorationResult
		if err := json.Unmarshal(body, &responded); err != nil {
			t.Fatal(err)
		}
		return responded.ID
	}
	const intentA = "daaaaaaaaaaaaaaaaaaaaaaaa"
	resultB := explore("dabababababababababababab", "dbbcbbbbbbbbbbbbbbbbbbbbb", "dacacacacacacacacacacacac")
	explore("dbcbababababababababababa", intentA, "dbacacacacacacacacacacaca")
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
	product, s, dsn, call, errorCode := q05MuxFull(t, "")
	ctx := context.Background()
	const draftID = "d717171717171717171717171"
	const intentID = "d727272727272727272727272"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[` +
		`{"id":"d737373737373737373737373","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true},` +
		`{"id":"d747474747474747474747474","description":"Keep history visible","consequence":"Past requests stay listed","decision":"pending","source":"autonomous","provenance":"autonomous"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}

	receiptPath := "/api/factory/intents/" + intentID + "/revisions/" + draftID + "/exploration"
	if status, raw := call(http.MethodGet, receiptPath, ""); status != 200 || strings.TrimSpace(string(raw)) != `{"intent":"`+intentID+`","basis_revision":"`+draftID+`","state":"none"}` {
		t.Fatalf("unrecorded exploration: %d %s", status, raw)
	}
	chooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = chooser
	code, _ := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("explore status=%d", code)
	}
	calls := chooser.calls
	if calls == 0 {
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
	if chooser.calls != calls {
		t.Fatalf("reads consumed judgments: calls=%d", chooser.calls)
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
	t.Run("original_input_receipt_after_head_changes", func(t *testing.T) {
		terminal := after
		terminal.ID, terminal.Predecessor, terminal.State = "d989898989898989898989898", after.ID, "declined"
		if err := s.SaveIntentRevision(ctx, terminal); err != nil {
			t.Fatal(err)
		}
		if status, data := call(http.MethodGet, "/api/factory/intents/"+intentID+"/revisions/"+terminal.ID+"/exploration", ""); status != 404 {
			t.Fatalf("nondraft input receipt: %d %s", status, data)
		}
		page, err := s.IntentHistory(ctx, intentID, 1, 1)
		if err != nil || len(page) != 1 || page[0].ID != terminal.ID {
			t.Fatal("one-row history fixture", page, err)
		}

		receipt := func(call func(string, string, string) (int, []byte), path string) []byte {
			t.Helper()
			status, raw := call(http.MethodGet, path, "")
			if status != 200 {
				t.Fatalf("exploration receipt %d %s", status, raw)
			}
			return raw
		}
		raw := receipt(call, receiptPath)
		var recorded factoryservice.ExplorationReceipt
		if err := json.Unmarshal(raw, &recorded); err != nil {
			t.Fatal(err)
		}
		if recorded.Intent != intentID || recorded.BasisRevision != draftID || recorded.State != "recorded" || recorded.Result == nil || recorded.Result.ID != head.ID || recorded.Result.Predecessor != draftID || recorded.Result.State != "draft" {
			t.Fatalf("wrong original transition: %+v", recorded)
		}
		if recorded.Result.Exploration.Recommendation == nil || len(recorded.Result.Exploration.Settled) != 1 || recorded.Result.Exploration.Settled[0] != "d747474747474747474747474" {
			t.Fatal("receipt lost saved advice/verdicts")
		}
		originalAdvice, _ := json.Marshal(factoryservice.Exploration{Content: review.Content, Challenges: review.Challenges, Settled: review.Settled, Recommendation: review.Recommendation, Consistency: review.Consistency})
		receiptAdvice, _ := json.Marshal(recorded.Result.Exploration)
		if string(originalAdvice) != string(receiptAdvice) {
			t.Fatal("historical receipt changed recorded presentation")
		}
		// An ordinary successor without a review is not an exploration receipt.
		absent := receipt(call, "/api/factory/intents/"+intentID+"/revisions/"+head.ID+"/exploration")
		var missing factoryservice.ExplorationReceipt
		if json.Unmarshal(absent, &missing) != nil || missing.State != "none" || missing.Result != nil {
			t.Fatalf("ordinary save confirmed exploration: %s", absent)
		}
		// A result remains private independently of the current head and history.
		if status, data := other(http.MethodGet, receiptPath, ""); status != 404 || strings.Contains(string(data), head.ID) {
			t.Fatalf("foreign receipt: %d %s", status, data)
		}
		nativeMux := http.NewServeMux()
		product.registerExtensionProductRoutes(nativeMux)
		socket := "@soda-exploration-receipt-" + strconv.Itoa(os.Getpid())
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		native := httptest.NewUnstartedServer(nativeMux)
		native.Listener.Close()
		native.Listener = listener
		native.Start()
		defer native.Close()
		proxy, closeProxy := Extension(socket)
		defer closeProxy()
		authority, err := json.Marshal(extensions.Authority{ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1", Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "get"}, Actor: extensions.Actor{ID: "1", Username: "alice"}})
		if err != nil {
			t.Fatal(err)
		}
		proxyPath := strings.TrimPrefix(receiptPath, "/api")
		for _, probe := range []struct {
			method, path string
			status       int
		}{
			{http.MethodGet, proxyPath, 200}, {http.MethodPost, proxyPath, 404}, {http.MethodGet, proxyPath + "?", 404}, {http.MethodGet, proxyPath + "?owner=1", 404}, {http.MethodGet, proxyPath + "/extra", 404}, {http.MethodGet, "/factory/intents/" + intentID + "/revisions/bad/exploration", 404},
		} {
			r := httptest.NewRequest(probe.method, probe.path, nil)
			r.Header.Set(extensions.ContextHeader, string(authority))
			r.Header.Set(extensions.AdmissionHeader, "test-admission")
			r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
			w := httptest.NewRecorder()
			proxy.ServeHTTP(w, r)
			if w.Code != probe.status {
				t.Fatalf("private exploration proxy %s: %d %s", probe.path, w.Code, w.Body.String())
			}
			if probe.status == 200 && (w.Body.String() != string(raw) || w.Header().Get("Cache-Control") != "no-store") {
				t.Fatal("proxy changed/cached receipt")
			}
		}
		native.Close()
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		inspect, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer inspect.Close()
		snapshot := func() string {
			t.Helper()
			var digest string
			err := inspect.QueryRowContext(ctx, `SELECT md5(jsonb_build_array(
        (SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM factory_intent_heads h),
        (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_intent_revisions r),
        (SELECT jsonb_agg(to_jsonb(v) ORDER BY result_revision) FROM factory_intent_reviews v),
        (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM factory_assignments a),
        (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),
        (SELECT jsonb_agg(to_jsonb(u) ORDER BY run) FROM factory_usage u),
        (SELECT jsonb_agg(to_jsonb(j) ORDER BY snapshot) FROM factory_priority_judgments j),
        (SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c))::text)`).Scan(&digest)
			if err != nil {
				t.Fatal(err)
			}
			return digest
		}
		before := snapshot()
		checks, revokeAt := 0, 0
		transport := factoryhttpapi.Requests{Service: factoryservice.Requests{Store: reopened}, Actor: 1, Host: product.FountainIdentities.HostID(), Current: func(w http.ResponseWriter, r *http.Request) bool {
			checks++
			if r.Header.Get("Authorization") != "Bearer declared-exploration-reader" || (revokeAt > 0 && checks == revokeAt) {
				http.Error(w, "declared account withdrawn", 401)
				return false
			}
			return true
		}}
		server := httptest.NewServer(transport.Handler())
		defer func() { server.Close() }()
		read := func(method, path string, status int) []byte {
			t.Helper()
			r, err := http.NewRequestWithContext(ctx, method, server.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Authorization", "Bearer declared-exploration-reader")
			response, err := server.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != status {
				t.Fatalf("receipt HTTP %s: %d %s %v", path, response.StatusCode, data, err)
			}
			if status == 200 && response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("cached receipt")
			}
			return data
		}
		if data := read(http.MethodGet, receiptPath, 200); string(data) != string(raw) {
			t.Fatal("reopen/recomposition changed original result")
		}
		for _, query := range []string{"?", "?revision=" + nextID, "?owner=1"} {
			read(http.MethodGet, receiptPath+query, 400)
		}
		read(http.MethodPost, receiptPath, 405)
		read(http.MethodGet, "/api/factory/intents/"+intentID+"/revisions/d999999999999999999999999/exploration", 404)
		read(http.MethodGet, "/api/factory/intents/d999999999999999999999999/revisions/"+draftID+"/exploration", 404)
		checks, revokeAt = 0, 2
		if data := read(http.MethodGet, receiptPath, 401); strings.Contains(string(data), head.ID) {
			t.Fatal("retired read disclosed historical result")
		}
		revokeAt = 0
		server.Close()
		transport.Actor = 9007199254740993
		server = httptest.NewServer(transport.Handler())
		read(http.MethodGet, receiptPath, 404)
		if snapshot() != before || chooser.calls != calls {
			t.Fatal("receipt reads moved state, consumed credit or consulted")
		}
	})

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
	accepted := factory.IntentRevision{
		ID: "d595959595959595959595959", Intent: "d505050505050505050505050", Owner: 2, Repository: 7,
		Objective: "Employees request equipment and Operations records delivery", State: "accepted", MaxIssues: 10, PlanningCalls: 8, CreatedUnix: 1729000000,
		Ideas:             []factory.ProductIdea{idea},
		EffectiveOutcomes: []string{idea.ID},
		Tasks:             []factory.IntentTask{{Key: "d515151515151515151515151", Objective: "Deliver equipment workflow", Outcomes: []string{"Recorded request history"}, IdeaIDs: []string{idea.ID}}},
	}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	posts, _, _, err := db.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10})
	if err != nil || len(posts) != 1 {
		t.Fatalf("accepted source posts=%d err=%v", len(posts), err)
	}
	code, body := owner(http.MethodPost, "/api/factory/feed/posts", `{"repository":"7","body":"New requests above EUR 1000 also need Finance approval. Existing requests keep their route.","parent_id":"`+posts[0].ID+`","idempotency_key":"owner-read-change"}`)
	if code != http.StatusOK {
		t.Fatalf("comment save=%d %s", code, body)
	}
	var saved struct {
		SourcePost string `json:"post_id"`
	}
	if err := json.Unmarshal(body, &saved); err != nil || saved.SourcePost == "" {
		t.Fatalf("comment identity=%s err=%v", body, err)
	}
	pipe, err := db.PostIntake(ctx, saved.SourcePost)
	if err != nil {
		t.Fatal(err)
	}
	routed := pipe
	routed.Judgment, routed.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
	if err := db.AdvancePostIntake(ctx, pipe, routed); err != nil {
		t.Fatal(err)
	}
	drafter := &fakePostDrafter{draft: validTestDraft(saved.SourcePost)}
	creator := &fakePostCreator{next: 100, issues: map[string]int64{}}
	response, err := factoryservice.DrivePostIntake(ctx, db, &factoryservice.PostCollaborators{Drafter: drafter, Creator: creator}, saved.SourcePost)
	if err != nil || response.Issue != 0 || drafter.calls != 0 || creator.creates != 0 {
		t.Fatalf("product proposal created detached work: response=%+v drafts=%d creates=%d err=%v", response, drafter.calls, creator.creates, err)
	}
	path := "/api/factory/feed/posts/" + saved.SourcePost
	decode := func(code int, body []byte) store.PostProductChange {
		t.Helper()
		var response struct {
			Change store.PostProductChange `json:"product_change"`
			Issue  int64                   `json:"issue"`
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
	status, replyBody := owner(http.MethodGet, "/api/factory/feed/posts/"+posts[0].ID+"/replies?repository=7", "")
	var replies struct {
		Items []struct {
			Change *store.PostProductChange `json:"product_change"`
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
	if code, body := collaborator(http.MethodGet, "/api/factory/intents/"+accepted.Intent+"/history", ""); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("collaborator private history=%d %s", code, body)
	}
}
