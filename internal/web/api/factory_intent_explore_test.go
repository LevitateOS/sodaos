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

func TestAppendClarifyOnceReusesStableIdentity(t *testing.T) {
	base := factory.IntentRevision{Ideas: []factory.ProductIdea{{ID: "d111111111111111111111111", Description: "Track leave requests", Decision: "accept"}}}
	first := appendClarifyOnce(base, "d222222222222222222222222")
	clarified := 0
	for _, idea := range first.Ideas {
		if idea.Description == "Describe the first user action and the result that should count as success" {
			clarified++
		}
	}
	if clarified != 1 {
		t.Fatalf("round one clarifications=%d", clarified)
	}
	second := appendClarifyOnce(first, "d333333333333333333333333")
	clarified = 0
	for _, idea := range second.Ideas {
		if idea.Description == "Describe the first user action and the result that should count as success" {
			clarified++
			if idea.ID != "d222222222222222222222222" || idea.Decision != "pending" {
				t.Fatalf("stable clarification lost=%+v", idea)
			}
		}
	}
	if clarified != 1 {
		t.Fatalf("round two clarifications=%d, want exactly one stable idea", clarified)
	}
}

func TestAppendClarifyOncePreservesAnsweredClarification(t *testing.T) {
	answered := clarifyIdea("d222222222222222222222222")
	answered.Decision = "edit"
	answered.Correction = "The first action is submitting the request form"
	answered.Provenance = "human"
	answered.Actor = 7
	r := factory.IntentRevision{Ideas: []factory.ProductIdea{
		{ID: "d111111111111111111111111", Description: "Track leave requests", Decision: "accept"},
		answered,
	}}
	next := appendClarifyOnce(r, "d333333333333333333333333")
	if len(next.Ideas) != 2 {
		t.Fatalf("answered clarification duplicated: %+v", next.Ideas)
	}
	if next.Ideas[1].Correction != "The first action is submitting the request form" || next.Ideas[1].Provenance != "human" {
		t.Fatalf("answer not preserved=%+v", next.Ideas[1])
	}
}

func TestAppendClarifyOnceIgnoresOrdinaryIdeas(t *testing.T) {
	r := factory.IntentRevision{Ideas: []factory.ProductIdea{
		{ID: "d111111111111111111111111", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
	}}
	next := appendClarifyOnce(r, "d222222222222222222222222")
	if len(next.Ideas) != 2 || next.Ideas[0].ID != "d111111111111111111111111" {
		t.Fatalf("ordinary idea touched=%+v", next.Ideas)
	}
	if next.Ideas[1].ID != "d222222222222222222222222" || next.Ideas[1].Decision != "pending" {
		t.Fatalf("clarification missing=%+v", next.Ideas)
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
	allowanceBody := `{"id":"d454545454545454545454545","intent":"d232323232323232323232323","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", allowanceBody); code != http.StatusCreated {
		t.Fatalf("allowance save status=%d body=%s", code, body)
	}
	product.IntentExploreChooser = &exploreFixtureChooser{}
	code, body := call(http.MethodPost, "/api/factory/intents/d232323232323232323232323/explore", `{"revision":"d121212121212121212121212"}`)
	if code != http.StatusCreated {
		t.Fatalf("explore status=%d body=%s", code, body)
	}
	var responded exploreResponse
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
	allowance := `{"id":"d8e8e8e8e8e8e8e8e8e8e8e8e","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
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
// provider: incremental design, default assumption handling, the shared
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
	case choices["apply_default"] != nil:
		selected, confidence = "apply_default", 0.9
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
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", `{"id":"d404404404404404404404404","intent":"`+intentID+`","amount":8,"ceiling":8,"unit":"planning_calls"}`); code != http.StatusCreated {
		t.Fatalf("allowance save status=%d body=%s", code, body)
	}
	allowanceUsage := func() (reserved, consumed int) {
		t.Helper()
		code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", "")
		if code != http.StatusOK {
			t.Fatalf("allowance read status=%d body=%s", code, body)
		}
		var a factory.PlanningAllowance
		if err := json.Unmarshal(body, &a); err != nil {
			t.Fatal(err)
		}
		return a.Reserved, a.Consumed
	}
	chooser := &scriptedExploreChooser{}
	product.IntentExploreChooser = chooser
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+narrowID+`"}`)
	if code != http.StatusBadRequest || errorCode(body) != "invalid_proposal_task_limit" {
		t.Fatalf("over-bound explore status=%d body=%s", code, body)
	}
	if reserved, consumed := allowanceUsage(); reserved != 0 || consumed != 1 {
		t.Fatalf("bound refusal kept %d charges: reserved=%d consumed=%d calls=%d", consumed, reserved, consumed, chooser.calls)
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
	if reserved, consumed := allowanceUsage(); reserved != 0 || consumed != 1 {
		t.Fatalf("repeat explore charged again: reserved=%d consumed=%d", reserved, consumed)
	}
	if chooser.calls != 1 {
		t.Fatalf("repeat explore consulted again: calls=%d", chooser.calls)
	}

	// The same round fits after raising max_issues: all four judgments
	// run and the outcome_gap verdict at 0.73 persists with its review.
	const wideID = "d205205205205205205205205"
	if code, body := call(http.MethodPost, "/api/factory/intents", draft(wideID, narrowID, 64)); code != http.StatusCreated {
		t.Fatalf("wide draft save status=%d body=%s", code, body)
	}
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+wideID+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("fitting explore status=%d body=%s", code, body)
	}
	if chooser.calls != 5 || !chooser.sawConsistency {
		t.Fatalf("fitting round calls=%d consistency=%v", chooser.calls, chooser.sawConsistency)
	}
	var responded exploreResponse
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
	if reserved, consumed := allowanceUsage(); reserved != 0 || consumed != 5 {
		t.Fatalf("fitting round charges: reserved=%d consumed=%d", reserved, consumed)
	}
}

// TestExploreMidRoundExhaustionNeedsFunding proves the U04 budget
// distinction on the real explore path: finite allowance exhaustion
// mid-round reports 409 planning_funding_required (never 503), while
// provider failure, stale head and session change keep
// their own codes. Charged calls stay charged and the saved correction
// head is untouched by the failed round.
func TestExploreMidRoundExhaustionNeedsFunding(t *testing.T) {
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

	var code int
	var body []byte

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
