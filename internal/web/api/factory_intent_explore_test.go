package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

func TestIntentDesignRespectsExplicitApproachAndChangesGraph(t *testing.T) {
	r := factory.IntentRevision{Objective: "Build accessible request entry", Ideas: []factory.ProductIdea{{ID: "d111111111111111111111111", Description: "Accessible request entry", Decision: "accept"}}}
	ids := make([]string, 104)
	for i := range ids {
		ids[i] = fmt.Sprintf("d%024x", 1000+i)
	}
	proposal := factory.ProposeIntent(r, ids)
	complete := applyIntentDesign(r, ids, proposal, factory.PriorityAnswer{Selected: "complete_first", Confidence: 1})
	if complete.Approach != "complete_first" || len(complete.Tasks) != 1 {
		t.Fatal("design advice did not produce real combined work")
	}
	r.Approach = "incremental"
	candidates := intentDesignCandidates(r)
	if len(candidates) != 2 || candidates[0].ID != "incremental" || candidates[1].ID != "clarify" {
		t.Fatal("explicit human choice was overridden or scope review removed")
	}
	explicit := applyIntentDesign(r, ids, proposal, factory.PriorityAnswer{Selected: "complete_first", Confidence: 1})
	if explicit.Approach != "incremental" || len(explicit.Tasks) != 2 {
		t.Fatal("human choice lost")
	}
}

func TestClarifyIdeaCarriesAssumptionBasis(t *testing.T) {
	idea := clarifyIdea("d111111111111111111111111")
	if !idea.Assumption || idea.Decision != "pending" || idea.Source != "assumption" || idea.Provenance != "autonomous" || !idea.Material || idea.Actor != 0 {
		t.Fatalf("clarify basis=%+v", idea)
	}
	r := factory.IntentRevision{ID: "d222222222222222222222222", Intent: "d333333333333333333333333", Owner: 7, Objective: "Build a tracker", State: "draft", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestExploreResponseKeepsRevisionFlatWithExploration(t *testing.T) {
	proposal := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", State: "draft"}
	content := factory.DescribeAlternatives("Time-off requests my manager can approve")
	challenges := []factory.AssumptionChallenge{{Idea: "d333333333333333333333333", Assumption: "Everyone can see each request", Consequence: "Dates are exposed", Recommendation: "Confirm it"}}
	raw, err := json.Marshal(attachExploration(proposal, factory.ProposalContent{Producer: content.Producer, Supported: content.Supported, Unknowns: content.Unknowns}, challenges, []string{"d333333333333333333333333"}, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	var merged map[string]any
	if err := json.Unmarshal(raw, &merged); err != nil {
		t.Fatal(err)
	}
	if merged["id"] != proposal.ID || merged["state"] != "draft" {
		t.Fatalf("revision fields not flat: %v", merged)
	}
	exploration, ok := merged["exploration"].(map[string]any)
	if !ok || exploration["recommendation"] != nil {
		t.Fatalf("exploration missing: %v", merged)
	}
	if _, ok := exploration["content"].(map[string]any); !ok {
		t.Fatal("content missing from exploration")
	}
	if list, ok := exploration["challenges"].([]any); !ok || len(list) != 1 {
		t.Fatalf("challenges=%v", exploration["challenges"])
	}
	if list, ok := exploration["settled"].([]any); !ok || len(list) != 1 || list[0] != "d333333333333333333333333" {
		t.Fatalf("settled=%v", exploration["settled"])
	}
}

// exploreFixtureChooser is the declared external judgment for explore
// boundary tests: deterministic valid answers without a provider. When fail
// is set every consultation fails like a provider outage.
type exploreFixtureChooser struct {
	fail  bool
	calls int
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
	probs[ids[0]] = 1
	return factory.PriorityAnswer{Selected: ids[0], Model: "fixture", Confidence: 0.9, Probabilities: probs}, nil
}

// TestExploreMidRoundExhaustionNeedsFunding proves the U04 budget
// distinction on the real explore path: finite allowance exhaustion
// mid-round reports 409 planning_funding_required (never 503), while
// provider failure, operator ceiling, stale head and session change keep
// their own codes. Charged calls stay charged and the saved correction
// head is untouched by the failed round.
func TestExploreMidRoundExhaustionNeedsFunding(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
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
	saveAllowance := func(id, intent string, amount int) {
		t.Helper()
		shape := fmt.Sprintf(`{"id":"%s","intent":"%s","amount":%d,"ceiling":%d,"unit":"planning_calls"}`, id, intent, amount, amount)
		if code, body := call(http.MethodPost, "/api/factory/intent-allowances", shape); code != http.StatusCreated {
			t.Fatalf("allowance save status=%d body=%s", code, body)
		}
	}
	allowanceUsage := func(intent string) (reserved, consumed int) {
		t.Helper()
		code, body := call(http.MethodGet, "/api/factory/intents/"+intent+"/allowance", "")
		if code != http.StatusOK {
			t.Fatalf("allowance read status=%d body=%s", code, body)
		}
		var a factory.PlanningAllowance
		if err := json.Unmarshal(body, &a); err != nil {
			t.Fatal(err)
		}
		return a.Reserved, a.Consumed
	}
	pendingIdea := func(id string) string {
		return `{"id":"` + id + `","description":"Track leave requests","consequence":"Managers review requests","decision":"pending","assumption":true,"material":true}`
	}

	// Operator ceiling first: the global budget is fresh, so the purpose
	// call reserves and the assumptions leg refuses on the ceiling with
	// 503, never the funding code.
	const ceilingDraft = "d555555555555555555555555"
	const ceilingIntent = "d666666666666666666666666"
	saveDraft(ceilingDraft, ceilingIntent, "", pendingIdea("d777777777777777777777777"))
	saveAllowance("d888888888888888888888888", ceilingIntent, 8)
	s.PriorityCallBudget = 1
	ceilingChooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = ceilingChooser
	code, body := call(http.MethodPost, "/api/factory/intents/"+ceilingIntent+"/explore", `{"revision":"`+ceilingDraft+`"}`)
	s.PriorityCallBudget = 32
	if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" || ceilingChooser.calls != 1 {
		t.Fatalf("ceiling explore status=%d body=%s calls=%d", code, body, ceilingChooser.calls)
	}

	// Mid-round exhaustion: amount 1 funds the purpose call; the
	// assumptions leg then refuses on the exhausted allowance with 409.
	const firstID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	const headID = "d1a1a1a1a1a1a1a1a1a1a1a1a"
	const correction = "Only direct managers approve; skip HR review"
	saveDraft(firstID, intentID, "", pendingIdea("d333333333333333333333333"))
	saveDraft(headID, intentID, firstID, `{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"edit","correction":"`+correction+`"},`+pendingIdea("d335333333333333333333333"))
	saveAllowance("d444444444444444444444444", intentID, 1)
	midChooser := &exploreFixtureChooser{}
	product.IntentExploreChooser = midChooser
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+headID+`"}`)
	if code != http.StatusConflict || errorCode(body) != "planning_funding_required" || midChooser.calls != 1 {
		t.Fatalf("mid-round explore status=%d body=%s calls=%d", code, body, midChooser.calls)
	}
	if reserved, consumed := allowanceUsage(intentID); reserved != 0 || consumed != 1 {
		t.Fatalf("charged call not retained: reserved=%d consumed=%d", reserved, consumed)
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
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+headID+`"}`)
	if code != http.StatusConflict || errorCode(body) != "planning_funding_required" {
		t.Fatalf("repeat explore status=%d body=%s", code, body)
	}

	// Provider failure keeps 503 with the call charged, never the funding code.
	const providerDraft = "d777777777777777777777778"
	const providerIntent = "d888888888888888888888889"
	saveDraft(providerDraft, providerIntent, "", pendingIdea("d999999999999999999999991"))
	saveAllowance("d999999999999999999999992", providerIntent, 8)
	product.IntentExploreChooser = &exploreFixtureChooser{fail: true}
	code, body = call(http.MethodPost, "/api/factory/intents/"+providerIntent+"/explore", `{"revision":"`+providerDraft+`"}`)
	if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" {
		t.Fatalf("provider explore status=%d body=%s", code, body)
	}
	if reserved, consumed := allowanceUsage(providerIntent); reserved != 0 || consumed != 1 {
		t.Fatalf("failed call not charged: reserved=%d consumed=%d", reserved, consumed)
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
