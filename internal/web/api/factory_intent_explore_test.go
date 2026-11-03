package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

// Exercise the extracted service against real PostgreSQL. Product design,
// carried clarification and answered human choices cross the durable revision
// and review boundary, without a native API/session or execution provider.
func TestIntentDesignRespectsExplicitApproachAndChangesGraph(t *testing.T) {
	st, ctx := driveTestStore(t)
	requests := factoryservice.Requests{Store: st}
	chooser := &exploreFixtureChooser{design: "complete_first"}
	exploration := factoryservice.RequestExploration{Requests: requests, Chooser: chooser}
	idea := factory.ProductIdea{ID: "d" + strings.Repeat("1", 24), Description: "Accessible request entry", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}
	create := func(digit, approach, objective string, ideas []factory.ProductIdea) factory.IntentRevision {
		t.Helper()
		input := factory.IntentRevision{ID: "d" + strings.Repeat(digit, 24), Intent: "d" + strings.Repeat(digit, 23) + "e", Objective: objective, Approach: approach, State: "draft", MaxIssues: 10, PlanningCalls: 8, Ideas: ideas}
		saved, err := requests.SaveRevision(ctx, 7, input)
		if err != nil {
			t.Fatal(err)
		}
		return saved
	}
	first := create("2", "", "Build accessible request entry", []factory.ProductIdea{idea})
	combined, err := exploration.Explore(ctx, 7, first.Intent, first.ID)
	if err != nil || combined.Approach != "complete_first" || len(combined.Tasks) != 1 {
		t.Fatal("shared design did not produce combined work", combined.Approach, len(combined.Tasks), err)
	}
	explicit := create("3", "incremental", "Build accessible request entry", []factory.ProductIdea{idea})
	incremental, err := exploration.Explore(ctx, 7, explicit.Intent, explicit.ID)
	if err != nil || incremental.Approach != "incremental" || len(incremental.Tasks) != 2 {
		t.Fatal("shared design overrode an explicit human approach", incremental.Approach, len(incremental.Tasks), err)
	}
	raw, _ := json.Marshal(combined)
	var flat map[string]any
	if err := json.Unmarshal(raw, &flat); err != nil {
		t.Fatal(err)
	}
	advice, ok := flat["exploration"].(map[string]any)
	if flat["id"] != combined.ID || flat["state"] != "draft" || !ok || advice["content"] == nil || advice["challenges"] == nil || advice["settled"] == nil {
		t.Fatal("shared response lost flat revision or exploration fields", flat)
	}
	// An ordinary requested human idea that uses the same prose is not the
	// producer's material assumption and must not suppress its clarification.
	chooser.design = "clarify"
	ordinary := idea
	ordinary.Description = "Describe the first user action and the result that should count as success"
	ordinary.Consequence = "Implementation waits until this material product question is resolved."
	source := create("4", "", "Clarify the intended product behavior", []factory.ProductIdea{ordinary})
	round, err := exploration.Explore(ctx, 7, source.Intent, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	clarification := func(r factory.IntentRevision, expected string, answered bool) string {
		t.Helper()
		found, id, human := 0, "", false
		for _, current := range r.Ideas {
			if current.ID == ordinary.ID {
				human = current.Provenance == "human" && current.Source == "requested"
			}
			if current.Description != ordinary.Description || !current.Assumption || current.Source != "assumption" {
				continue
			}
			found++
			id = current.ID
			if !current.Material || (expected != "" && id != expected) {
				t.Fatal("clarification identity/basis changed", current)
			}
			if answered {
				if current.Decision != "edit" || current.Provenance != "human" || current.Actor != 7 || current.Correction != "The first action is submitting the request form" {
					t.Fatal("answered clarification was overwritten", current)
				}
			} else if current.Decision != "pending" || current.Provenance != "autonomous" || current.Actor != 0 {
				t.Fatal("clarification fabricated human approval", current)
			}
		}
		if found != 1 || !human {
			t.Fatal("clarification was duplicated or coalesced with human writing", found, human)
		}
		return id
	}
	stable := clarification(round.IntentRevision, "", false)
	second, err := exploration.Explore(ctx, 7, source.Intent, round.ID)
	if err != nil {
		t.Fatal(err)
	}
	clarification(second.IntentRevision, stable, false)
	answered := second.IntentRevision
	answered.ID, answered.Predecessor = "d"+strings.Repeat("5", 24), second.ID
	for i := range answered.Ideas {
		if answered.Ideas[i].ID == stable {
			answered.Ideas[i].Decision, answered.Ideas[i].Correction = "edit", "The first action is submitting the request form"
			answered.Ideas[i].Provenance, answered.Ideas[i].Actor = "human", 7
		}
	}
	answered, err = requests.SaveRevision(ctx, 7, answered)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := exploration.Explore(ctx, 7, source.Intent, answered.ID)
	if err != nil {
		t.Fatal(err)
	}
	clarification(resumed.IntentRevision, stable, true)
	review, err := requests.Review(ctx, 7, source.Intent, resumed.ID)
	if err != nil || review.Review == nil || review.Review.BasisRevision != answered.ID || review.Review.ResultRevision != resumed.ID {
		t.Fatal("shared exploration did not atomically save its actual review", err)
	}
	before := chooser.calls
	if _, err := exploration.Explore(ctx, 7, source.Intent, answered.ID); !errors.Is(err, store.ErrStaleRevision) || chooser.calls != before {
		t.Fatal("stale exploration consulted again", err)
	}
	if _, err := exploration.Explore(ctx, 8, source.Intent, resumed.ID); !errors.Is(err, factoryservice.ErrRequestNotFound) || chooser.calls != before {
		t.Fatal("another owner reached exploration", err)
	}
	if runs, err := st.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
		t.Fatal("private exploration launched execution", runs, err)
	}
}

// exploreFixtureChooser is the declared external judgment for explore
// boundary tests: deterministic valid answers without a provider. When fail
// is set every consultation fails like a provider outage.
type exploreFixtureChooser struct {
	design string
	fail   bool
	calls  int
}

func (f *exploreFixtureChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	if f.fail {
		return factory.PriorityAnswer{}, errors.New("fixture judgment failed")
	}
	ids := make([]string, 0, len(choices))
	for id := range choices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	probs := make(map[string]float64, len(ids))
	for _, id := range ids {
		probs[id] = 0
	}
	selected := ids[0]
	if f.design != "" && choices["complete_first"] != nil && choices[f.design] != nil {
		selected = f.design
	}
	probs[selected] = 1
	return factory.PriorityAnswer{Selected: selected, Model: "fixture", Confidence: 0.9, Probabilities: probs}, nil
}

// TestExplorePersistsAgreedReview proves the producer and its persisted
// response agree: a successful injected-judgment round stores the actual
// proposal, challenges, recommendation and consistency under the result
// revision with exact basis/result provenance.
func TestExplorePersistsAgreedReview(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	draftBody := `{"id":"d121212121212121212121212","intent":"d232323232323232323232323","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[{"id":"d343434343434343434343434","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draftBody); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	chooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = chooser
	requests := factoryservice.Requests{Store: s}
	transport := factoryhttpapi.Requests{Service: requests, Host: product.FountainIdentities.HostID(), Actor: collaboration.ActorID(1),
		Exploration: factoryservice.RequestExploration{Requests: requests, Chooser: chooser},
		Planning:    factoryservice.RequestPlanning{Requests: requests},
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			if r.Header.Get("Authorization") != "Bearer declared-exploration-client" {
				http.Error(w, "declared account admission unavailable", 401)
				return false
			}
			return true
		}}
	server := httptest.NewServer(transport.Handler())
	defer server.Close()
	direct := func(method, path, body string) (int, []byte) {
		t.Helper()
		r, err := http.NewRequestWithContext(t.Context(), method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-exploration-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, data
	}
	code, body := direct(http.MethodPost, "/api/factory/intents/d232323232323232323232323/explore", `{"revision":"d121212121212121212121212"}`)
	if code != http.StatusCreated {
		t.Fatalf("explore status=%d body=%s", code, body)
	}
	var responded factoryservice.ExplorationResult
	if err := json.Unmarshal(body, &responded); err != nil {
		t.Fatal(err)
	}
	stored, err := s.IntentReview(context.Background(), responded.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Intent != "d232323232323232323232323" || stored.BasisRevision != "d121212121212121212121212" || stored.ResultRevision != responded.ID {
		t.Fatalf("stored provenance wrong: %+v", stored)
	}
	agree := func(name string, want, got any) {
		t.Helper()
		wantRaw, _ := json.Marshal(want)
		gotRaw, _ := json.Marshal(got)
		if string(wantRaw) != string(gotRaw) {
			t.Fatalf("%s disagrees: stored=%s response=%s", name, wantRaw, gotRaw)
		}
	}
	agree("content", stored.Content, responded.Exploration.Content)
	agree("challenges", stored.Challenges, responded.Exploration.Challenges)
	agree("settled", stored.Settled, responded.Exploration.Settled)
	agree("recommendation", stored.Recommendation, responded.Exploration.Recommendation)
	agree("consistency", stored.Consistency, responded.Exploration.Consistency)
	if stored.Recommendation == nil || stored.Consistency == nil {
		t.Fatal("supported round lost its recommendation or consistency")
	}
	// Reads on a command-capable host cannot consult its configured chooser.
	beforeCalls := chooser.calls
	chooser.fail = true
	base := "/api/factory/intents/" + stored.Intent
	for _, path := range []string{base + "/plan", base + "/review", base + "/generation"} {
		if code, body := direct(http.MethodGet, path, ""); code != 200 || chooser.calls != beforeCalls {
			t.Fatal("shared HTTP read consulted or lost recorded state", path, code, string(body), chooser.calls)
		}
	}
	if code, _ := direct(http.MethodGet, base+"/plan?revision=unaccepted", ""); code != 400 || chooser.calls != beforeCalls {
		t.Fatal("status query broadened admission", code, chooser.calls)
	}
	if code, _ := direct(http.MethodPost, base+"/plan/review", `{"revision":"`+responded.ID+`"}`); code != 503 || chooser.calls != beforeCalls {
		t.Fatal("unassigned walkthrough review bypassed missing write authority", code, chooser.calls)
	}
	if code, _ := direct(http.MethodPost, base+"/generate", `{"revision":"`+responded.ID+`"}`); code != 409 || chooser.calls != beforeCalls {
		t.Fatal("draft direction generated work", code, chooser.calls)
	}
}

// TestExploreVerdictsTargetGeneratedResultIdeas proves verdict provenance:
// with empty basis ideas, Explore generates result-only ideas and every
// stored challenge/settlement targets those result ideas, evaluated using
// basis evidence. Content still binds the basis revision.
func TestExploreVerdictsTargetGeneratedResultIdeas(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	const draftID = "d5e5e5e5e5e5e5e5e5e5e5e5e"
	const intentID = "d6e6e6e6e6e6e6e6e6e6e6e6e"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[]}`
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
	if len(responded.Ideas) == 0 {
		t.Fatal("empty basis produced no result ideas")
	}
	stored, err := s.IntentReview(context.Background(), responded.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Content.Revision != draftID || stored.BasisRevision != draftID || stored.ResultRevision != responded.ID {
		t.Fatalf("content left the basis: %+v", stored)
	}
	if len(stored.Challenges)+len(stored.Settled) == 0 {
		t.Fatal("no verdict targets the generated result ideas")
	}
	proposal := map[string]bool{}
	for _, idea := range responded.Ideas {
		proposal[idea.ID] = true
	}
	for _, challenge := range stored.Challenges {
		if !proposal[challenge.Idea] {
			t.Fatalf("challenge outside the result: %+v", challenge)
		}
	}
	for _, id := range stored.Settled {
		if !proposal[id] {
			t.Fatalf("settlement outside the result: %s", id)
		}
	}
}

// scriptedExploreChooser answers each explore judgment leg without a
// provider: incremental design, the shared
// review walkthrough, and an outcome_gap consistency verdict at 0.73.
type scriptedExploreChooser struct {
	calls          int
	sawConsistency bool
}

func (c *scriptedExploreChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	c.calls++
	selected, confidence := "shared_review", 0.9
	switch {
	case choices["complete_first"] != nil:
		selected, confidence = "incremental", 0.9
	case choices["consistent"] != nil:
		selected, confidence = "outcome_gap", 0.73
		c.sawConsistency = true
	}
	probs := make(map[string]float64, len(choices))
	for id := range choices {
		probs[id] = 0
	}
	probs[selected] = 1
	return factory.PriorityAnswer{Selected: selected, Model: "scripted", Confidence: confidence, Probabilities: probs}, nil
}

// TestExploreDerivedProposalBoundsRefuseBeforeFurtherJudgments pins the
// FJ-v3 explore post-judgment trace: a draft whose judged work graph
// exceeds max_issues cannot persist, so the round refuses with the
// whitelisted invalid_proposal code after deriving the graph instead of
// burning the remaining judgments. The single spent design receipt is
// retained and replayed, never refunded or replaced: a repeated explore
// replays it without a new charge. Raising max_issues on a new revision
// completes the same round, persisting the outcome_gap verdict at 0.73.
func TestExploreDerivedProposalBoundsRefuseBeforeFurtherJudgments(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	const intentID = "d202202202202202202202202"
	draft := func(id, predecessor string, maxIssues int) string {
		body := `{"id":"` + id + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":` + fmt.Sprint(maxIssues) + `,"planning_calls":8,"ideas":[`
		body += `{"id":"d303303303303303303303303","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true,"source":"assumption","provenance":"autonomous"},`
		body += `{"id":"d304304304304304304304304","description":"Capture leave requests","consequence":"Requests are recorded","decision":"accept"},`
		body += `{"id":"d305305305305305305305305","description":"Managers review requests","consequence":"Decisions are visible","decision":"accept"},`
		body += `{"id":"d306306306306306306306306","description":"People see their own outcomes","consequence":"Outcomes stay visible","decision":"accept"}]}`
		if predecessor != "" {
			body = `{"id":"` + id + `","intent":"` + intentID + `","predecessor":"` + predecessor + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":` + fmt.Sprint(maxIssues) + `,"planning_calls":8,"ideas":[`
			body += `{"id":"d303303303303303303303303","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true,"source":"assumption","provenance":"autonomous"},`
			body += `{"id":"d304304304304304304304304","description":"Capture leave requests","consequence":"Requests are recorded","decision":"accept"},`
			body += `{"id":"d305305305305305305305305","description":"Managers review requests","consequence":"Decisions are visible","decision":"accept"},`
			body += `{"id":"d306306306306306306306306","description":"People see their own outcomes","consequence":"Outcomes stay visible","decision":"accept"}]}`
		}
		return body
	}
	const narrowID = "d201201201201201201201201"
	if code, body := call(http.MethodPost, "/api/factory/intents", draft(narrowID, "", 4)); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	chooser := &scriptedExploreChooser{}
	product.IntentExploreChooser = chooser
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+narrowID+`"}`)
	if code != http.StatusBadRequest || errorCode(body) != "invalid_proposal_task_limit" {
		t.Fatalf("over-bound explore status=%d body=%s", code, body)
	}
	if chooser.calls != 1 || chooser.sawConsistency {
		t.Fatalf("round spent judgments past the derived graph: calls=%d consistency=%v", chooser.calls, chooser.sawConsistency)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/"+intentID, "")
	if code != http.StatusOK {
		t.Fatalf("intent read status=%d body=%s", code, body)
	}
	var head factory.IntentRevision
	if err := json.Unmarshal(body, &head); err != nil {
		t.Fatal(err)
	}
	if head.ID != narrowID {
		t.Fatalf("refused round moved the head: %+v", head)
	}
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+narrowID+`"}`)
	if code != http.StatusBadRequest || errorCode(body) != "invalid_proposal_task_limit" {
		t.Fatalf("repeat explore status=%d body=%s", code, body)
	}
	if chooser.calls != 1 {
		t.Fatalf("repeat explore consulted again: calls=%d", chooser.calls)
	}

	// The same round fits after raising max_issues: the remaining judgments
	// run and the outcome_gap verdict at 0.73 persists with its review.
	const wideID = "d205205205205205205205205"
	if code, body := call(http.MethodPost, "/api/factory/intents", draft(wideID, narrowID, 64)); code != http.StatusCreated {
		t.Fatalf("wide draft save status=%d body=%s", code, body)
	}
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+wideID+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("fitting explore status=%d body=%s", code, body)
	}
	if chooser.calls != 4 || !chooser.sawConsistency {
		t.Fatalf("fitting round calls=%d consistency=%v", chooser.calls, chooser.sawConsistency)
	}
	var responded factoryservice.ExplorationResult
	if err := json.Unmarshal(body, &responded); err != nil {
		t.Fatal(err)
	}
	if len(responded.Tasks) != 6 {
		t.Fatalf("derived tasks=%d", len(responded.Tasks))
	}
	if responded.Exploration.Consistency == nil || len(responded.Exploration.Consistency.Mismatches) != 1 || responded.Exploration.Consistency.Mismatches[0].Class != "outcome_gap" || responded.Exploration.Consistency.Confidence != 0.73 {
		t.Fatalf("consistency verdict lost: %+v", responded.Exploration.Consistency)
	}
	stored, err := s.IntentReview(context.Background(), responded.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ResultRevision != responded.ID || stored.BasisRevision != wideID {
		t.Fatalf("stored provenance wrong: %+v", stored)
	}
	if stored.Consistency == nil || len(stored.Consistency.Mismatches) != 1 || stored.Consistency.Mismatches[0].Class != "outcome_gap" || stored.Consistency.Confidence != 0.73 {
		t.Fatalf("stored consistency lost: %+v", stored.Consistency)
	}
}

// TestExploreFailureCodes proves the real explore path's error codes:
// provider failure reports 503 and retries on the next explore, while stale
// head and session change keep their own codes. The saved correction head
// is untouched by the failed round.
func TestExploreFailureCodes(t *testing.T) {
	product, _, call, errorCode := q05Mux(t, "")
	saveDraft := func(id, intent, predecessor, ideas string) {
		t.Helper()
		body := `{"id":"` + id + `","intent":"` + intent + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[` + ideas + `]}`
		if predecessor != "" {
			body = `{"id":"` + id + `","intent":"` + intent + `","predecessor":"` + predecessor + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,"ideas":[` + ideas + `]}`
		}
		if code, body := call(http.MethodPost, "/api/factory/intents", body); code != http.StatusCreated {
			t.Fatalf("draft save status=%d body=%s", code, body)
		}
	}
	pendingIdea := func(id string) string {
		return `{"id":"` + id + `","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}`
	}

	var code int
	var body []byte

	const firstID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	const headID = "d1a1a1a1a1a1a1a1a1a1a1a1a"
	const correction = "Only direct managers approve; skip HR review"
	saveDraft(firstID, intentID, "", pendingIdea("d333333333333333333333333"))
	saveDraft(headID, intentID, firstID, `{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"edit","correction":"`+correction+`"},`+pendingIdea("d335333333333333333333333"))

	// Provider failure reports 503 and records nothing, so the next
	// explore asks again instead of refusing on a spent call.
	failing := &exploreFixtureChooser{fail: true}
	product.IntentExploreChooser = failing
	for i := 1; i <= 2; i++ {
		code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+headID+`"}`)
		if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" || failing.calls != i {
			t.Fatalf("provider explore %d status=%d body=%s calls=%d", i, code, body, failing.calls)
		}
	}
	code, body = call(http.MethodGet, "/api/factory/intents/"+intentID, "")
	if code != http.StatusOK {
		t.Fatalf("intent read status=%d body=%s", code, body)
	}
	var head factory.IntentRevision
	if err := json.Unmarshal(body, &head); err != nil {
		t.Fatal(err)
	}
	if head.ID != headID || len(head.Ideas) != 2 || head.Ideas[0].Correction != correction {
		t.Fatalf("saved correction moved: %+v", head)
	}

	// Stale head keeps 412.
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"d999999999999999999999999"}`)
	if code != http.StatusPreconditionFailed || errorCode(body) != "stale_revision" {
		t.Fatalf("stale explore status=%d body=%s", code, body)
	}

	// Changed session generation keeps its own authority code, not funding.
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)
	authority, err := json.Marshal(map[string]any{
		"extension_id": "soda", "instance_id": "q05-test", "session_generation": "generation-1",
		"contribution": map[string]any{"id": "spaces", "kind": "page", "scope": "global", "action": "post"},
		"actor":        map[string]any{"id": "1", "username": "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", strings.NewReader(`{"revision":"`+headID+`"}`))
	r.Header.Set(extensions.ContextHeader, string(authority))
	r.Header.Set(extensions.AdmissionHeader, "test-admission")
	r.Header.Set(extensions.SessionGenerationHeader, "generation-9")
	r.Header.Set("Origin", "https://soda.test")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusConflict || errorCode(w.Body.Bytes()) != "extension_session_changed" {
		t.Fatalf("session explore status=%d body=%s", w.Code, w.Body.String())
	}
}
