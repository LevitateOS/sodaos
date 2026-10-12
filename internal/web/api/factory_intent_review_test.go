package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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
		if in.Operation != extensions.OperationCurrentActor {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: actorID, Username: username}})
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
