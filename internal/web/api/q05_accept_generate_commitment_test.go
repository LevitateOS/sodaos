package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// q05KeepChooser is the declared test-local decomposition judgment: the
// accepted task decomposition stands as authored. No live model call.
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

// q05ReviewChooser is the declared test-local decomposition refusal: the
// reviewed decomposition needs reconsideration. No live model call.
type q05ReviewChooser struct{}

func (q05ReviewChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	probabilities := map[string]float64{}
	for id := range choices {
		probabilities[id] = 0
	}
	if _, ok := choices["review"]; !ok {
		return factory.PriorityAnswer{}, errors.New("review is not an option")
	}
	probabilities["review"] = 1
	return factory.PriorityAnswer{Selected: "review", Confidence: 0.9, Probabilities: probabilities, Model: "q05-declared"}, nil
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
// with an empty edge set (the accepted task has no dependencies).
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

// TestAcceptGenerateCommitmentBridge executes the changed native/API
// commitment seam through the real mux: exact U accepted nextRevision
// body (predecessor, repository "7", decided ideas, authored tasks) ->
// session/repository-owner guard -> accepted 201 -> generate POST ->
// real Coordinator.GenerateAcceptedIntent with declared executor facts
// -> durable completed generation read back over HTTP. Declared
// external judgments: q05KeepChooser (keep/0.9) for the decomposition
// triplet; declared native facts from the fake executor echoing exact
// submitted payloads. No live model call; the stub never touches the
// real writer (native goldens cover that half separately).
func TestAcceptGenerateCommitmentBridge(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	fake := &q05Graph{t: t, store: s, rev: 7}
	product.Coordinator = &control.Coordinator{
		Store: s, Graph: fake, GraphActorID: 21, AcceptanceReads: fake,
		Planning: &control.PlanningJudgments{Journal: s, Funding: s, Chooser: q05KeepChooser{}},
	}

	const draftID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	const ideaID = "d333333333333333333333333"
	const taskKey = "d444444444444444444444444"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","repository":"0","objective":"Time-off requests my manager can approve",` +
		`"ideas":[],"tasks":[],"state":"draft","max_issues":20,"planning_calls":8}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	uShape := `{"id":"d555555555555555555555555","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", uShape); code != http.StatusCreated {
		t.Fatalf("allowance status=%d body=%s", code, body)
	}
	accepted := func(id, predecessor, repository string) string {
		return `{"id":"` + id + `","intent":"` + intentID + `","predecessor":"` + predecessor + `","repository":"` + repository + `",` +
			`"objective":"Time-off requests my manager can approve",` +
			`"ideas":[{"id":"` + ideaID + `","description":"Track leave requests","consequence":"Managers review requests",` +
			`"decision":"accept","source":"requested","provenance":"human","actor":"1"}],` +
			`"tasks":[{"key":"` + taskKey + `","objective":"Build the leave tracker","outcomes":["Requests list"],"idea_ids":["` + ideaID + `"]}],` +
			`"state":"accepted","max_issues":20,"planning_calls":8}`
	}
	// Stale CAS refusal: valid shape, wrong predecessor. No head move.
	const staleID = "d666666666666666666666666"
	if code, body := call(http.MethodPost, "/api/factory/intents", accepted("d777777777777777777777777", staleID, "7")); code != http.StatusPreconditionFailed || errorCode(body) != "stale_revision" {
		t.Fatalf("stale accept status=%d body=%s", code, body)
	}
	// Invisible-repository guard refusal on a second intent. No generation.
	const intent2 = "d888888888888888888888888"
	const draft2ID = "d999999999999999999999999"
	const accept2ID = "daaaaaaaaaaaaaaaaaaaaaaaaa"
	draft2 := `{"id":"` + draft2ID + `","intent":"` + intent2 + `","repository":"0","objective":"Second request",` +
		`"ideas":[],"tasks":[],"state":"draft","max_issues":20,"planning_calls":8}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft2); code != http.StatusCreated {
		t.Fatalf("draft2 save status=%d body=%s", code, body)
	}
	accept2 := `{"id":"` + accept2ID + `","intent":"` + intent2 + `","predecessor":"` + draft2ID + `","repository":"99",` +
		`"objective":"Second request",` +
		`"ideas":[{"id":"` + ideaID + `","description":"Track leave requests","consequence":"Managers review requests",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"}],` +
		`"tasks":[{"key":"` + taskKey + `","objective":"Build the leave tracker","outcomes":["Requests list"],"idea_ids":["` + ideaID + `"]}],` +
		`"state":"accepted","max_issues":20,"planning_calls":8}`
	if code, body := call(http.MethodPost, "/api/factory/intents", accept2); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("invisible-repo accept status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intent2+"/generation", ""); code != http.StatusOK {
		t.Fatalf("intent2 generation status=%d", code)
	} else {
		var pending map[string]string
		if err := json.Unmarshal(body, &pending); err != nil || pending["state"] != "pending" {
			t.Fatalf("intent2 generation body=%s", body)
		}
	}
	// Successful accept on the first intent.
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	code, body := call(http.MethodPost, "/api/factory/intents", accepted(acceptID, draftID, "7"))
	if code != http.StatusCreated {
		t.Fatalf("accept status=%d body=%s", code, body)
	}
	var saved factory.IntentRevision
	if err := json.Unmarshal(body, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID != acceptID || saved.State != "accepted" || saved.Owner != 1 || saved.Repository != 7 || saved.Predecessor != draftID {
		t.Fatalf("accepted scope id=%s state=%s owner=%d repo=%d pred=%s", saved.ID, saved.State, saved.Owner, saved.Repository, saved.Predecessor)
	}
	// Generate through the handler into the real coordinator.
	code, body = call(http.MethodPost, "/api/factory/intents/"+intentID+"/generate", `{}`)
	if code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", code, body)
	}
	var gen factory.IntentGeneration
	if err := json.Unmarshal(body, &gen); err != nil {
		t.Fatal(err)
	}
	if gen.State != "completed" || gen.RevisionID != acceptID || gen.IntentID != intentID || gen.OperationID != "intent-"+acceptID {
		t.Fatalf("generation state=%s rev=%s op=%s", gen.State, gen.RevisionID, gen.OperationID)
	}
	if len(gen.Delivered) != 0 {
		t.Fatalf("delivered=%d", len(gen.Delivered))
	}
	bindings, err := s.IntentTaskBindings(context.Background(), acceptID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 || bindings[0].TaskKey != taskKey || bindings[0].IssueID != 100 || bindings[0].IssueIndex != 5 || bindings[0].ContentVersion != 1 {
		t.Fatalf("bindings=%+v", bindings)
	}
	var receipt sdk.IssueGraphReceipt
	if err := json.Unmarshal(gen.Receipt, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.ActorID != 21 || receipt.RepositoryID != 7 || len(receipt.Issues) != 1 || receipt.Issues[0].Key != taskKey {
		t.Fatalf("receipt=%+v", receipt)
	}
	if fake.submits != 1 || len(fake.creates) != 1 || fake.creates[0].Key != taskKey {
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
	// Read-back and repeat: same operation, no second graph.
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/generation", ""); code != http.StatusOK {
		t.Fatalf("generation read status=%d", code)
	} else {
		var reread factory.IntentGeneration
		if err := json.Unmarshal(body, &reread); err != nil || reread.OperationID != gen.OperationID || reread.State != "completed" {
			t.Fatalf("generation reread body=%s", body)
		}
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/generate", `{}`); code != http.StatusOK {
		t.Fatalf("repeat generate status=%d body=%s", code, body)
	}
	if fake.submits != 1 {
		t.Fatalf("repeat submitted a second graph: %d", fake.submits)
	}
}

// TestGenerateDecompositionRefusalNeedsReview proves a deliberate
// decomposition refusal reports 409 intent_review_required (never a
// storage 503): the accepted direction is saved, but the proposed work
// needs reconsideration before native work can be created. No graph is
// submitted and the accepted head is retained.
func TestGenerateDecompositionRefusalNeedsReview(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	fake := &q05Graph{t: t, store: s, rev: 7}
	product.Coordinator = &control.Coordinator{
		Store: s, Graph: fake, GraphActorID: 21, AcceptanceReads: fake,
		Planning: &control.PlanningJudgments{Journal: s, Funding: s, Chooser: q05ReviewChooser{}},
	}

	const draftID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	const ideaID = "d333333333333333333333333"
	const taskKey = "d444444444444444444444444"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","repository":"0","objective":"Time-off requests my manager can approve",` +
		`"ideas":[],"tasks":[],"state":"draft","max_issues":20,"planning_calls":8}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	uShape := `{"id":"d555555555555555555555555","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", uShape); code != http.StatusCreated {
		t.Fatalf("allowance status=%d body=%s", code, body)
	}
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	accepted := `{"id":"` + acceptID + `","intent":"` + intentID + `","predecessor":"` + draftID + `","repository":"7",` +
		`"objective":"Time-off requests my manager can approve",` +
		`"ideas":[{"id":"` + ideaID + `","description":"Track leave requests","consequence":"Managers review requests",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"}],` +
		`"tasks":[{"key":"` + taskKey + `","objective":"Build the leave tracker","outcomes":["Requests list"],"idea_ids":["` + ideaID + `"]}],` +
		`"state":"accepted","max_issues":20,"planning_calls":8}`
	if code, body := call(http.MethodPost, "/api/factory/intents", accepted); code != http.StatusCreated {
		t.Fatalf("accept status=%d body=%s", code, body)
	}
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/generate", `{}`)
	if code != http.StatusConflict || errorCode(body) != "intent_review_required" {
		t.Fatalf("refused generate status=%d body=%s", code, body)
	}
	if fake.submits != 0 {
		t.Fatalf("refused decomposition submitted a graph: %d", fake.submits)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID, ""); code != http.StatusOK {
		t.Fatalf("intent read status=%d", code)
	} else {
		var head factory.IntentRevision
		if err := json.Unmarshal(body, &head); err != nil || head.ID != acceptID || head.State != "accepted" {
			t.Fatalf("accepted head moved body=%s", body)
		}
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/generation", ""); code != http.StatusOK {
		t.Fatalf("generation read status=%d", code)
	} else {
		var pending map[string]string
		if err := json.Unmarshal(body, &pending); err != nil || pending["state"] != "pending" {
			t.Fatalf("generation body=%s", body)
		}
	}
}
