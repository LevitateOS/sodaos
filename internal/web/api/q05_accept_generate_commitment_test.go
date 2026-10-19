package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// q05KeepChooser is the declared test-local decomposition judgment: the
// recorded "keep" candidate stands. No live model call.
type q05KeepChooser struct{}

func (q05KeepChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	probabilities := map[string]float64{}
	for id := range choices {
		probabilities[id] = 0
	}
	if _, ok := choices["keep"]; !ok {
		return factory.PriorityAnswer{}, errors.New("keep is not an option")
	}
	probabilities["keep"] = 1
	return factory.PriorityAnswer{Selected: "keep", Confidence: 0.9, Probabilities: probabilities, Model: "q05-declared"}, nil
}

// q05RefusingChooser fails any judgment call: a passing run behind it
// proves the serving path refused before any funded comparison.
type q05RefusingChooser struct{}

func (q05RefusingChooser) Select(_ context.Context, _ any, _ string, _ map[string]any) (factory.PriorityAnswer, error) {
	return factory.PriorityAnswer{}, errors.New("refused path must not evaluate a comparison")
}

// q05Graph is a test-local control.IssueGraphExecutor that returns declared
// native facts faithfully from the exact submitted payload: receipts bind
// every submitted key, and read-back echoes submitted titles/bodies.
type q05Graph struct {
	t       *testing.T
	store   *store.Store
	rev     int64
	submits int
	creates []sdk.IssueGraphCreate
	checks  [][3]int64
}

func (f *q05Graph) CheckIntentOwner(_ context.Context, repo, owner, actor int64) (int64, error) {
	f.checks = append(f.checks, [3]int64{repo, owner, actor})
	return f.rev, nil
}

func (f *q05Graph) SubmitGraph(ctx context.Context, g factory.IntentGeneration, repo int64) (sdk.OperationRecord, error) {
	f.t.Helper()
	row, err := f.store.IntentGeneration(ctx, g.RevisionID)
	if err != nil {
		f.t.Fatalf("generation not durable before submit: %v", err)
	}
	if row.State != "pending" || row.OperationID != g.OperationID {
		f.t.Fatalf("pre-submit row state=%s op=%s", row.State, row.OperationID)
	}
	var payload sdk.IssueGraphPayload
	if err := json.Unmarshal(g.Payload, &payload); err != nil {
		f.t.Fatal(err)
	}
	f.creates = payload.Creates
	receipt := sdk.IssueGraphReceipt{ActorID: g.ActorID, RepositoryID: repo}
	for i, create := range payload.Creates {
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{Key: create.Key, IssueID: int64(100 + i), IssueIndex: int64(5 + i), ContentVersion: 1})
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		f.t.Fatal(err)
	}
	f.submits++
	return sdk.OperationRecord{OperationID: g.OperationID, EffectState: sdk.EffectCommitted, Receipt: raw}, nil
}

func (f *q05Graph) LookupGraph(_ context.Context, _ factory.IntentGeneration, _ int64) (sdk.OperationLookup, error) {
	return sdk.OperationLookup{Status: sdk.BackgroundOutcomeNotObserved}, nil
}

func (f *q05Graph) CancelGraph(_ context.Context, _ string) (sdk.OperationRecord, error) {
	return sdk.OperationRecord{}, nil
}

func (f *q05Graph) ReadGeneratedIssue(_ context.Context, _ int64, index int64, actor int64) (sdk.NativeSnapshot, int64, error) {
	for i, create := range f.creates {
		if int64(5+i) != index {
			continue
		}
		return sdk.NativeSnapshot{Issue: &sdk.SnapshotIssue{
			ID: strconv.FormatInt(int64(100+i), 10), Index: strconv.FormatInt(index, 10),
			Title: create.Title, Content: create.Body, ContentVersion: 1,
			TitleDigest: forgejo.ContentDigest(create.Title), ContentDigest: forgejo.ContentDigest(create.Body),
			Visible: true, Complete: true,
			Provenance: sdk.SnapshotCreationProvenance{PosterID: strconv.FormatInt(actor, 10)},
		}, Dependencies: &sdk.SnapshotDependencyPage{IssueID: strconv.FormatInt(int64(100+i), 10), Items: []sdk.SnapshotDependency{}, Complete: true}}, f.rev, nil
	}
	return sdk.NativeSnapshot{}, 0, errors.New("unknown generated issue")
}

// ReadAcceptanceEvidence brackets the same echoed native facts for the
// admission path: the submitted title/body digests at the owner revision
// with an empty edge set (the accepted tasks have no dependencies).
func (f *q05Graph) ReadAcceptanceEvidence(_ context.Context, _ string, issue string, _ []string) (control.AcceptanceEvidence, error) {
	for i, create := range f.creates {
		if strconv.FormatInt(int64(5+i), 10) != issue {
			continue
		}
		return control.AcceptanceEvidence{
			Revision: f.rev,
			Issue: control.AcceptanceIssueView{
				Index: issue, Visible: true,
				TitleDigest: forgejo.ContentDigest(create.Title), TitleOccurrence: factory.OriginalOccurrence,
				ContentDigest: forgejo.ContentDigest(create.Body), ContentVer: 1,
			},
		}, nil
	}
	return control.AcceptanceEvidence{}, errors.New("unknown generated issue")
}

const (
	generateIntentID = "d222222222222222222222222"
	generateDraftID  = "d111111111111111111111111"
	generateIdeaA    = "d555555555555555555555555"
	generateIdeaB    = "d666666666666666666666666"
	generateTaskA    = "d700000000000000000000001"
	generateTaskB    = "d700000000000000000000002"
	generateTaskC    = "d700000000000000000000003"
	generateAuthored = "d444444444444444444444444"
)

// generateOptionsDraft is the decomposition-options-suitable draft: the
// supported leave-workflow objective with attributed effective scope.
func generateOptionsDraft() string {
	return `{"id":"` + generateDraftID + `","intent":"` + generateIntentID + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":[{"id":"` + generateIdeaA + `","description":"Capture leave requests","consequence":"Requests can be recorded",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
		`{"id":"` + generateIdeaB + `","description":"Managers review requests","consequence":"Decisions are visible",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"}],` +
		`"effective_outcomes":["` + generateIdeaA + `","` + generateIdeaB + `"],` +
		`"state":"draft","max_issues":10,"planning_calls":12}`
}

// generateAcceptedBody carries the reviewed draft as predecessor and
// preserves its substantive scope exactly; only the authored task
// differs from any planner candidate.
func generateAcceptedBody(id string) string {
	return `{"id":"` + id + `","intent":"` + generateIntentID + `","predecessor":"` + generateDraftID + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":[{"id":"` + generateIdeaA + `","description":"Capture leave requests","consequence":"Requests can be recorded",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
		`{"id":"` + generateIdeaB + `","description":"Managers review requests","consequence":"Decisions are visible",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"}],` +
		`"effective_outcomes":["` + generateIdeaA + `","` + generateIdeaB + `"],` +
		`"tasks":[{"key":"` + generateAuthored + `","objective":"Build the leave tracker","outcomes":["Requests list"],"idea_ids":["` + generateIdeaA + `"]}],` +
		`"state":"accepted","max_issues":10,"planning_calls":12}`
}

// generateAnswer fences one decomposition_options answer for the saved
// draft head: the covering "keep" candidate plus "split" when requested.
func generateAnswer(t *testing.T, operation string, keys ...string) string {
	t.Helper()
	tasks := []factory.IntentTask{
		{Key: generateTaskA, Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{generateIdeaA}},
		{Key: generateTaskB, Objective: "Managers review requests", Outcomes: []string{"Decisions are visible"}, IdeaIDs: []string{generateIdeaB}},
	}
	out := factory.SemanticPlannerOutput{Intent: generateIntentID, Revision: generateDraftID, Operation: operation}
	for _, key := range keys {
		candidate := tasks
		if key == "split" {
			candidate = []factory.IntentTask{{
				Key: generateTaskC, Objective: "Decide vacation requests with manager review",
				Outcomes: []string{"Requests can be recorded", "Decisions are visible"}, IdeaIDs: []string{generateIdeaA, generateIdeaB},
			}}
		}
		out.Candidates = append(out.Candidates, factory.SemanticCandidate{Key: key, Rationale: "Cohesive feature issues with independent roots joined once.", Tasks: candidate})
	}
	data, err := json.Marshal(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: "completed",
		Summary: "recorded choice ready", Output: &factory.PhaseOutput{Decomposition: &out},
	})
	if err != nil {
		t.Fatal(err)
	}
	return "work log\n```result-json\n" + string(data) + "\n```\ntrailer\n"
}

// recordGeneratePlan saves the options draft, funds the repository,
// plans the requested operation through the real mux, settles the
// fenced answer and accepts the scope-preserving successor,
// returning the accepted revision ID.
func recordGeneratePlan(t *testing.T, product *API, s *store.Store, call func(method, path, body string) (int, []byte), operation, acceptID string, keys ...string) string {
	t.Helper()
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", generateOptionsDraft()); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	seedPlanFunding(t, ctx, s)
	product.Coordinator = &control.Coordinator{Store: s, Host: &planStubHost{}, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/plan", `{"revision":"`+generateDraftID+`","operation":"`+operation+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("plan status=%d body=%s", code, body)
	}
	var launched struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(body, &launched); err != nil {
		t.Fatal(err)
	}
	settlePlanRun(t, ctx, s, launched.RunID, generateAnswer(t, operation, keys...))
	if code, body := call(http.MethodPost, "/api/factory/intents", generateAcceptedBody(acceptID)); code != http.StatusCreated {
		t.Fatalf("accept status=%d body=%s", code, body)
	}
	return acceptID
}

// wireGenerateCoordinator swaps the planning consumer for the native
// serving tail: the declared graph executor with the given judgment,
// reusing the same ephemeral store.
func wireGenerateCoordinator(product *API, s *store.Store, t *testing.T, chooser control.ProductJudgment) *q05Graph {
	fake := &q05Graph{t: t, store: s, rev: 7}
	product.Coordinator = &control.Coordinator{
		Store: s, Graph: fake, GraphActorID: 21, AcceptanceReads: fake,
		Planning: &control.PlanningJudgments{Journal: s, Funding: s, Chooser: chooser},
	}
	if err := s.SavePlanningAllowance(context.Background(), factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: generateIntentID, Owner: 1, Amount: 32, Unit: factory.PlanningAllowanceUnit, Ceiling: 32, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	return fake
}

// TestAcceptGenerateCommitmentBridge executes the recorded-plan serving
// seam through the real mux: options-suitable draft -> funded plan ->
// settled single-candidate answer -> scope-preserving accept ->
// generate POST resolves the recorded choice with zero comparative
// calls and adopts exactly the planner graph, never the authored task.
// Repeat generate reuses the durable operation with no second graph.
// Declared external judgments: none consulted (single candidate needs
// no comparison). Declared native facts from the fake executor echoing
// exact submitted payloads. No live model call.
func TestAcceptGenerateCommitmentBridge(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	recordGeneratePlan(t, product, s, call, "decomposition_options", acceptID, "keep")
	fake := wireGenerateCoordinator(product, s, t, q05KeepChooser{})

	code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{}`)
	if code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", code, body)
	}
	var gen factory.IntentGeneration
	if err := json.Unmarshal(body, &gen); err != nil {
		t.Fatal(err)
	}
	if gen.State != "completed" || gen.RevisionID != acceptID || gen.IntentID != generateIntentID || gen.OperationID != "intent-"+acceptID {
		t.Fatalf("generation state=%s rev=%s op=%s", gen.State, gen.RevisionID, gen.OperationID)
	}
	if len(gen.Delivered) != 0 {
		t.Fatalf("delivered=%d", len(gen.Delivered))
	}
	var submitted []string
	for _, task := range gen.SubmittedTasks {
		submitted = append(submitted, task.Key)
	}
	if len(submitted) != 2 || submitted[0] != generateTaskA || submitted[1] != generateTaskB {
		t.Fatalf("submitted=%v want recorded planner tasks", submitted)
	}
	bindings, err := s.IntentTaskBindings(context.Background(), acceptID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 2 || bindings[0].TaskKey != generateTaskA || bindings[1].TaskKey != generateTaskB {
		t.Fatalf("bindings=%+v", bindings)
	}
	var receipt sdk.IssueGraphReceipt
	if err := json.Unmarshal(gen.Receipt, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.ActorID != 21 || receipt.RepositoryID != 7 || len(receipt.Issues) != 2 {
		t.Fatalf("receipt=%+v", receipt)
	}
	if fake.submits != 1 || len(fake.creates) != 2 {
		t.Fatalf("submits=%d creates=%+v", fake.submits, fake.creates)
	}
	if len(fake.checks) == 0 {
		t.Fatal("no owner checks observed")
	}
	for _, check := range fake.checks {
		if check[0] != 7 || check[1] != 1 || check[2] != 21 {
			t.Fatalf("owner check=%v", check)
		}
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{}`); code != http.StatusOK {
		t.Fatalf("repeat generate status=%d body=%s", code, body)
	}
	if fake.submits != 1 {
		t.Fatalf("repeat submitted a second graph: %d", fake.submits)
	}
}

// TestGenerateRecordedMultiChoiceAdoptsFundedPick proves the recorded
// multi-candidate answer resolves through exactly one funded comparative
// choice over HTTP: the declared keeper selects "keep" and only its
// tasks submit. No live model call.
func TestGenerateRecordedMultiChoiceAdoptsFundedPick(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	recordGeneratePlan(t, product, s, call, "decomposition_options", acceptID, "keep", "split")
	fake := wireGenerateCoordinator(product, s, t, q05KeepChooser{})

	code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{}`)
	if code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", code, body)
	}
	var gen factory.IntentGeneration
	if err := json.Unmarshal(body, &gen); err != nil {
		t.Fatal(err)
	}
	if gen.State != "completed" {
		t.Fatalf("generation state=%s", gen.State)
	}
	var submitted []string
	for _, task := range gen.SubmittedTasks {
		submitted = append(submitted, task.Key)
	}
	if len(submitted) != 2 || submitted[0] != generateTaskA || submitted[1] != generateTaskB {
		t.Fatalf("submitted=%v want funded keep tasks", submitted)
	}
	if fake.submits != 1 {
		t.Fatalf("submits=%d", fake.submits)
	}
}

// TestGenerateRecordedChoiceRefusals proves every non-choice Generate
// refuses through the real mux before any funded comparison or native
// effect: drafts report intent_not_accepted, accepted directions with
// no recorded plan or with a propose_intent recording report
// planning_choice_unavailable (even the refusing chooser never fires),
// and an accepted head in an invisible repository reports not_found.
func TestGenerateRecordedChoiceRefusals(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	ctx := context.Background()

	if code, body := call(http.MethodPost, "/api/factory/intents", generateOptionsDraft()); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	fake := wireGenerateCoordinator(product, s, t, q05RefusingChooser{})
	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{}`); code != http.StatusConflict || errorCode(body) != "intent_not_accepted" {
		t.Fatalf("draft generate status=%d body=%s", code, body)
	}

	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	if code, body := call(http.MethodPost, "/api/factory/intents", generateAcceptedBody(acceptID)); code != http.StatusCreated {
		t.Fatalf("accept status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{}`); code != http.StatusConflict || errorCode(body) != "planning_choice_unavailable" {
		t.Fatalf("plan-less generate status=%d body=%s", code, body)
	}
	if fake.submits != 0 {
		t.Fatalf("plan-less generate submitted a graph: %d", fake.submits)
	}

	const intent2 = "d888888888888888888888888"
	idea := func(id, desc, consequence string) factory.ProductIdea {
		return factory.ProductIdea{ID: id, Description: desc, Consequence: consequence, Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}
	}
	draft2 := factory.IntentRevision{
		ID: "d999999999999999999999999", Intent: intent2, Owner: 1, Repository: 99,
		Objective:         "Build a leave approval workflow so managers can review time-off requests",
		State:             "draft",
		MaxIssues:         10,
		PlanningCalls:     12,
		CreatedUnix:       time.Now().Unix(),
		EffectiveOutcomes: []string{generateIdeaA, generateIdeaB},
		Ideas:             []factory.ProductIdea{idea(generateIdeaA, "Capture leave requests", "Requests can be recorded"), idea(generateIdeaB, "Managers review requests", "Decisions are visible")},
	}
	if err := s.SaveIntentRevision(ctx, draft2); err != nil {
		t.Fatal(err)
	}
	accept2 := draft2
	accept2.ID, accept2.Predecessor, accept2.State = "daaaaaaaaaaaaaaaaaaaaaaaa", draft2.ID, "accepted"
	accept2.Tasks = []factory.IntentTask{{
		Key: generateAuthored, Objective: "Build the leave tracker",
		Outcomes: []string{"Requests list"}, IdeaIDs: []string{generateIdeaA},
	}}
	if err := s.SaveIntentRevision(ctx, accept2); err != nil {
		t.Fatal(err)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intent2+"/generate", `{}`); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("hidden-repo generate status=%d body=%s", code, body)
	}
	if fake.submits != 0 {
		t.Fatalf("hidden-repo generate submitted a graph: %d", fake.submits)
	}
}

// TestGenerateProposeRecordingRefusesWithoutComparison proves a recorded
// propose_intent run never feeds the decomposition connector: the
// operation gate refuses before the funded comparison, so even a
// multi-candidate propose answer with a refusing chooser reports
// planning_choice_unavailable with no graph submitted.
func TestGenerateProposeRecordingRefusesWithoutComparison(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	recordGeneratePlan(t, product, s, call, "propose_intent", acceptID, "keep", "split")
	wireGenerateCoordinator(product, s, t, q05RefusingChooser{})

	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{}`); code != http.StatusConflict || errorCode(body) != "planning_choice_unavailable" {
		t.Fatalf("propose generate status=%d body=%s", code, body)
	}
}
