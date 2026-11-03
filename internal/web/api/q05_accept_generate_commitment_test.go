package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/forgejo"
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
	forgejo.GraphCodec
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

func (f *q05Graph) SubmitGraph(ctx context.Context, g factory.IntentGeneration, repo int64) (factory.GraphOperationOutcome, error) {
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
	return forgejo.GraphOutcomeFromNative(sdk.OperationRecord{OperationID: g.OperationID, EffectState: sdk.EffectCommitted, Receipt: raw}), nil
}

func (f *q05Graph) LookupGraph(_ context.Context, _ factory.IntentGeneration, _ int64) (factory.GraphOperationLookup, error) {
	return forgejo.GraphLookupFromNative(sdk.OperationLookup{Status: sdk.BackgroundOutcomeNotObserved}), nil
}

func (f *q05Graph) CancelGraph(_ context.Context, _ string) (factory.GraphOperationOutcome, error) {
	return factory.GraphOperationOutcome{}, nil
}

func (f *q05Graph) ReadGeneratedIssue(_ context.Context, _ int64, index int64, actor int64) (factory.GeneratedIssueObservation, int64, error) {
	for i, create := range f.creates {
		if int64(5+i) != index {
			continue
		}
		observation, err := forgejo.GeneratedIssueObservationFromNative(sdk.NativeSnapshot{Issue: &sdk.SnapshotIssue{
			ID: strconv.FormatInt(int64(100+i), 10), Index: strconv.FormatInt(index, 10),
			Title: create.Title, Content: create.Body, ContentVersion: 1,
			TitleDigest: forgejo.ContentDigest(create.Title), ContentDigest: forgejo.ContentDigest(create.Body),
			Visible: true, Complete: true,
			Provenance: sdk.SnapshotCreationProvenance{PosterID: strconv.FormatInt(actor, 10)},
		}, Dependencies: &sdk.SnapshotDependencyPage{IssueID: strconv.FormatInt(int64(100+i), 10), Items: []sdk.SnapshotDependency{}, Complete: true}})
		return observation, f.rev, err
	}
	return factory.GeneratedIssueObservation{}, 0, errors.New("unknown generated issue")
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

// q05WalkthroughContent authors the request-specific walkthroughs for the
// saved q05 draft head, mirroring planWalkthroughContent: deterministic
// planning-agent prose bound to the head's own objective and authorized
// outcomes, with no model call.
func q05WalkthroughContent(t *testing.T, head factory.IntentRevision) factory.ProposalContent {
	t.Helper()
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatal(err)
	}
	content.Producer, content.Supported = factory.PlanningProducer, true
	content.Alternatives = nil
	for _, key := range []string{"guided", "list"} {
		content.Alternatives = append(content.Alternatives, factory.ProductAlternative{
			Key: key, Name: key + " project planning", Audience: "The project owner", Summary: "Plan the recorded draft with visible progress.",
			Steps:      []factory.WalkthroughStep{{Actor: "The project owner", Action: "reviews the saved request", Result: "The planned work is visible"}},
			Visibility: "The owner sees the saved plan", Consequences: "The request stays editable before acceptance", Evidence: "The admitted request and inspected source", Outcomes: []string{head.Objective}, IdeaIDs: append([]string(nil), head.EffectiveOutcomes...),
		})
	}
	return content
}

// generateAnswer fences one decomposition_options answer for the saved
// draft head: the covering "keep" candidate plus "split" when requested,
// carrying the authored walkthroughs the walkthrough gate requires before
// native generate.
func generateAnswer(t *testing.T, head factory.IntentRevision, operation string, keys ...string) string {
	t.Helper()
	tasks := []factory.IntentTask{
		{Key: generateTaskA, Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{generateIdeaA}},
		{Key: generateTaskB, Objective: "Managers review requests", Outcomes: []string{"Decisions are visible"}, IdeaIDs: []string{generateIdeaB}},
	}
	walkthroughs := q05WalkthroughContent(t, head)
	out := factory.SemanticPlannerOutput{Intent: head.Intent, Revision: head.ID, Operation: operation, Walkthroughs: &walkthroughs}
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
	return string(data)
}

// recordGeneratePlan saves the options draft, funds the repository,
// plans the requested operation through the real mux, settles the
// fenced answer with authored walkthroughs, records the mocked
// consolidation/consistency review receipts and accepts the
// scope-preserving successor, returning the accepted revision ID.
func recordGeneratePlan(t *testing.T, product *API, s *store.Store, call func(method, path, body string) (int, []byte), operation, acceptID string, keys ...string) string {
	t.Helper()
	ctx := context.Background()
	draftBody, acceptedBody := generateOptionsDraft(), generateAcceptedBody(acceptID)
	app := strings.NewReplacer(
		"Build a leave approval workflow so managers can review time-off requests", "Build a private team task board with persisted priority",
		"Capture leave requests", "Store team tasks", "Requests can be recorded", "Team tasks persist",
		"Managers review requests", "Set task priority", "Decisions are visible", "Saved priority is visible",
		"Build the leave tracker", "Build the team task board", "Requests list", "Team tasks are visible",
		"Decide vacation requests with manager review", "Manage team tasks with saved priority")
	if operation == factory.IntentPlanningPropose {
		draftBody, acceptedBody = app.Replace(draftBody), app.Replace(acceptedBody)
		if factory.DescribeAlternatives("Build a private team task board with persisted priority").Supported {
			t.Fatal("authored app scope unexpectedly matched a canned workflow")
		}
	}
	if code, body := call(http.MethodPost, "/api/factory/intents", draftBody); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, generateIntentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	answer := generateAnswer(t, head, operation, keys...)
	if operation == factory.IntentPlanningPropose {
		answer = app.Replace(answer)
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
	settlePlanRun(t, ctx, s, launched.RunID, answer)
	// The walkthrough gate replays durable review receipts at serving
	// time, so the fixture records them explicitly here with the mocked chooser.
	judge := &exploreFixtureChooser{}
	product.IntentExploreChooser = judge
	code, body = call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/plan/review", `{"revision":"`+generateDraftID+`"}`)
	if code != http.StatusOK || judge.calls != 2 || !strings.Contains(string(body), `"recommended":"guided"`) {
		t.Fatalf("explicit walkthrough review failed: status=%d calls=%d body=%s", code, judge.calls, body)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents", acceptedBody); code != http.StatusCreated {
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
		Planning: &control.PlanningJudgments{Chooser: chooser},
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
	product, s, dsn, call, _ := q05MuxFull(t, "")
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	recordGeneratePlan(t, product, s, call, "decomposition_options", acceptID, "keep")
	fake := wireGenerateCoordinator(product, s, t, q05KeepChooser{})

	code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{"revision":"`+acceptID+`"}`)
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
	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{"revision":"`+acceptID+`"}`); code != http.StatusOK {
		t.Fatalf("repeat generate status=%d body=%s", code, body)
	}
	if fake.submits != 1 {
		t.Fatalf("repeat submitted a second graph: %d", fake.submits)
	}
	allowed, checks := false, 0
	captured, err := s.IntentGeneration(t.Context(), acceptID)
	if err != nil {
		t.Fatal(err)
	}
	denied := errors.New("declared current generation authority withdrawn")
	planning := factoryservice.RequestPlanning{Requests: factoryservice.Requests{Store: s, WriteAccess: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
		checks++
		if actor != 1 || repository != 7 || !allowed {
			return denied
		}
		return ctx.Err()
	}}, Generator: product.Coordinator}
	if _, _, err := planning.Generate(t.Context(), 1, generateIntentID, acceptID); !errors.Is(err, factoryservice.ErrRequestWriteAccess) || !errors.Is(err, denied) || fake.submits != 1 || checks != 1 {
		t.Fatal("shared generation replay bypassed current authority", fake.submits, checks, err)
	}
	allowed = true
	readback, wait, err := planning.Generate(t.Context(), 1, generateIntentID, acceptID)
	if err != nil || wait != nil || readback.OperationID != gen.OperationID || readback.State != gen.State || string(readback.Receipt) != string(captured.Receipt) || fake.submits != 1 || checks != 2 {
		t.Fatal("shared generation changed the committed operation", readback.OperationID, readback.State, fake.submits, checks, err)
	}
	t.Run("exact_generation_http", func(t *testing.T) {
		ctx := t.Context()
		requests := planning.Requests
		currentChecks, revokeAt := 0, 0
		transport := factoryhttpapi.Requests{Service: requests, Planning: planning, Host: product.FountainIdentities.HostID(), Actor: 1,
			Current: func(w http.ResponseWriter, r *http.Request) bool {
				currentChecks++
				if r.Header.Get("Authorization") != "Bearer declared-generation-client" || (revokeAt > 0 && currentChecks == revokeAt) {
					http.Error(w, "declared account admission unavailable", 401)
					return false
				}
				return true
			}}
		server := httptest.NewServer(transport.Handler())
		t.Cleanup(func() { server.Close() })
		base := "/api/factory/intents/" + generateIntentID + "/generate"
		send := func(method, path, body string, status int, discard bool) []byte {
			t.Helper()
			r, err := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Authorization", "Bearer declared-generation-client")
			response, err := server.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != status {
				data, _ := io.ReadAll(response.Body)
				t.Fatalf("generation HTTP %s: %d %s", path, response.StatusCode, data)
			}
			if discard {
				return nil
			}
			data, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			return data
		}
		nativeChecks, authorityChecks := len(fake.checks), checks
		for _, body := range []string{"", "null", "{}", `{"revision":1}`, `{"revision":"bad"}`, `{"revision":"` + acceptID + `","owner":"2"}`, `{"revision":"` + acceptID + `","revision":"` + generateDraftID + `"}`} {
			send(http.MethodPost, base, body, 400, false)
		}
		send(http.MethodPost, base, `{"revision":"`+strings.Repeat("x", 65536)+`"}`, 413, false)
		for _, query := range []string{"?", "?revision=" + acceptID, "?owner=1"} {
			send(http.MethodPost, base+query, `{"revision":"`+acceptID+`"}`, 400, false)
		}
		for _, revision := range []string{generateDraftID, "d" + strings.Repeat("9", 24)} {
			send(http.MethodPost, base, `{"revision":"`+revision+`"}`, 412, false)
		}
		currentChecks, revokeAt = 0, 3 // Current account is rechecked after decoding.
		send(http.MethodPost, base, `{"revision":"`+acceptID+`"}`, 401, false)
		revokeAt = 0
		if len(fake.checks) != nativeChecks || fake.submits != 1 || checks != authorityChecks {
			t.Fatal("refused generation reached authority/planning/native work", len(fake.checks), checks)
		}
		// The client loses this replay response; the next explicit request recovers
		// the same durable operation and original receipt without another graph.
		send(http.MethodPost, base, `{"revision":"`+acceptID+`"}`, 200, true)
		var recovered factory.IntentGeneration
		if err := json.Unmarshal(send(http.MethodPost, base, `{"revision":"`+acceptID+`"}`, 200, false), &recovered); err != nil {
			t.Fatal(err)
		}
		var originalReceipt, recoveredReceipt bytes.Buffer
		if err := json.Compact(&originalReceipt, captured.Receipt); err != nil {
			t.Fatal(err)
		}
		if err := json.Compact(&recoveredReceipt, recovered.Receipt); err != nil {
			t.Fatal(err)
		}
		if recovered.OperationID != captured.OperationID || recovered.RevisionID != acceptID || !bytes.Equal(recoveredReceipt.Bytes(), originalReceipt.Bytes()) || fake.submits != 1 {
			t.Fatal("lost reply changed the exact generation", recovered.OperationID, recovered.RevisionID, fake.submits)
		}
		// The Fountain frontend's private proxy reaches the same exact historical
		// generation read. Native/session facts remain the declared q05 peer.
		nativeMux := http.NewServeMux()
		product.registerExtensionProductRoutes(nativeMux)
		socket := "@soda-generation-recovery-" + strconv.Itoa(os.Getpid())
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
		authority, err := json.Marshal(extensions.Authority{ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1", Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "get"}, Actor: extensions.Actor{ID: "1", Username: "alice"}})
		if err != nil {
			t.Fatal(err)
		}
		proxyPath := "/factory/intents/" + generateIntentID + "/revisions/" + acceptID + "/generation"
		for _, probe := range []struct {
			method, path string
			status       int
		}{{http.MethodGet, proxyPath, 200}, {http.MethodPost, proxyPath, 404}, {http.MethodGet, proxyPath + "?", 404}, {http.MethodGet, proxyPath + "?owner=1", 404}, {http.MethodGet, proxyPath + "/extra", 404}, {http.MethodGet, "/factory/intents/" + generateIntentID + "/revisions/bad/generation", 404}} {
			r := httptest.NewRequest(probe.method, probe.path, nil)
			r.Header.Set(extensions.ContextHeader, string(authority))
			r.Header.Set(extensions.AdmissionHeader, "test-admission")
			r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
			w := httptest.NewRecorder()
			proxy.ServeHTTP(w, r)
			if w.Code != probe.status {
				t.Fatal("generation proxy refused wrong route", probe.path, w.Code, w.Body.String())
			}
			if probe.status == 200 {
				var served factory.IntentGeneration
				if err := json.Unmarshal(w.Body.Bytes(), &served); err != nil {
					t.Fatal(err)
				}
				got, _ := json.Marshal(served)
				want, _ := json.Marshal(captured)
				if string(got) != string(want) || w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("generation proxy changed or cached the original record")
				}
			}
		}
		native.Close()
		server.Close()
		transport.Actor = 9007199254740993
		server = httptest.NewServer(transport.Handler())
		nativeChecks = len(fake.checks)
		send(http.MethodPost, base, `{"revision":"`+acceptID+`"}`, 404, false)
		if len(fake.checks) != nativeChecks || fake.submits != 1 {
			t.Fatal("foreign owner reached native generation")
		}
	})
	t.Run("shared_scope_history_and_authority", func(t *testing.T) {
		ctx := t.Context()
		inspect, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer inspect.Close()
		state := func() string {
			t.Helper()
			var digest string
			if err := inspect.QueryRowContext(ctx, `SELECT md5(jsonb_build_array((SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM factory_intent_heads h),(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_intent_revisions r),(SELECT jsonb_agg(to_jsonb(g) ORDER BY revision) FROM factory_intent_generations g),(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM factory_intent_tasks t),(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c),(SELECT jsonb_agg(data ORDER BY id) FROM factory_assignments),(SELECT jsonb_agg(data ORDER BY id) FROM factory_runs),(SELECT jsonb_agg(data ORDER BY run) FROM factory_usage))::text)`).Scan(&digest); err != nil {
				t.Fatal(err)
			}
			return digest
		}
		requests := factoryservice.Requests{Store: s}
		read := func(limit int) factoryservice.RevisionHistory {
			t.Helper()
			before := state()
			code, body := call(http.MethodGet, "/api/factory/intents/"+generateIntentID+"/revisions?limit="+strconv.Itoa(limit), "")
			var overHTTP factoryservice.RevisionHistory
			if code != http.StatusOK || json.Unmarshal(body, &overHTTP) != nil {
				t.Fatalf("scope history status=%d body=%s", code, body)
			}
			direct, err := requests.ScopeHistory(ctx, 1, generateIntentID, limit)
			if err != nil {
				t.Fatal(err)
			}
			overHTTP.ObservedUnix = direct.ObservedUnix
			a, _ := json.Marshal(overHTTP)
			b, _ := json.Marshal(direct)
			if string(a) != string(b) || state() != before || fake.submits != 1 {
				t.Fatal("history hosts disagree or a read changed recorded work")
			}
			return direct
		}
		initial := read(5)
		if len(initial.Revisions) != 2 || initial.Truncated || !initial.Revisions[1].Comparison.Initial {
			t.Fatal("initial bounded history lost its root", initial)
		}
		accepted := initial.Revisions[0]
		if accepted.Revision != acceptID || accepted.Generation.State != "completed" || len(accepted.Generation.SubmittedKeys) != 2 || !accepted.Generation.SubmittedDiffers || len(accepted.Work) != 2 {
			t.Fatal("scope history lost the submitted graph and mappings", accepted)
		}
		for _, work := range accepted.Work {
			if work.Authority != "current" || work.State != "waiting" {
				t.Fatal("graph submission was mistaken for delivered work or lost authority", work)
			}
		}
		head, err := requests.Revision(ctx, 1, generateIntentID)
		if err != nil {
			t.Fatal(err)
		}
		oldTask := head.Tasks[0].Key
		raw, _ := json.Marshal(head)
		var next factory.IntentRevision
		if err := json.Unmarshal(raw, &next); err != nil {
			t.Fatal(err)
		}
		next.ID, next.Predecessor, next.State = "d"+strings.Repeat("c", 24), head.ID, "draft"
		next.Ideas[0].Decision, next.Ideas[0].Correction = "edit", "Capture leave requests with an explicit reason"
		next.Tasks[0].Objective = "Capture requests with a reason"
		next.Tasks[0].Outcomes = []string{"The request list includes its reason"}
		extra := factory.ProductIdea{ID: "d" + strings.Repeat("d", 24), Description: "Preserve request history", Consequence: "History remains inspectable", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}
		next.Ideas = append(next.Ideas, extra)
		next.EffectiveOutcomes = append(next.EffectiveOutcomes, extra.ID)
		next.Tasks = append(next.Tasks, factory.IntentTask{Key: "d" + strings.Repeat("e", 24), Objective: "Show request history", Outcomes: []string{"Prior requests remain visible"}, IdeaIDs: []string{extra.ID}, DependsOn: []string{oldTask}})
		raw, _ = json.Marshal(next)
		if code, body := call(http.MethodPost, "/api/factory/intents", string(raw)); code != http.StatusCreated {
			t.Fatalf("scope change status=%d body=%s", code, body)
		}
		changed := read(2)
		if len(changed.Revisions) != 2 || !changed.Truncated || changed.Revisions[0].GeneratedFrom != acceptID {
			t.Fatal("history lost generated lineage or bounded continuation", changed)
		}
		delta := changed.Revisions[0].Comparison
		if len(delta.Tasks.Added) != 1 || len(delta.Tasks.Changed) != 1 || delta.Tasks.Changed[0].Key != oldTask || !slices.Contains(delta.Tasks.Changed[0].ChangedFields, "objective") || !slices.Contains(delta.Tasks.Changed[0].ChangedFields, "outcomes") || len(delta.Ideas.Added) != 1 || len(delta.Ideas.Changed) != 1 || !delta.Ideas.Changed[0].CorrectionChanged || !delta.EdgesChanged || delta.State == nil || delta.State.To != "draft" {
			t.Fatal("scope history did not retain exact structural changes", delta)
		}
		for _, work := range changed.Revisions[1].Work {
			if work.Authority != "superseded" || work.State != "waiting" {
				t.Fatal("old mappings retained head authority or invented delivery", work)
			}
		}
		latest, err := requests.Revision(ctx, 1, generateIntentID)
		if err != nil {
			t.Fatal(err)
		}
		latest.ID, latest.Predecessor = "d"+strings.Repeat("f", 24), latest.ID
		latest.Tasks = latest.Tasks[1:]
		latest.Tasks[0].DependsOn = nil
		latest.Ideas = latest.Ideas[1:]
		latest.EffectiveOutcomes = latest.EffectiveOutcomes[1:]
		raw, _ = json.Marshal(latest)
		if code, body := call(http.MethodPost, "/api/factory/intents", string(raw)); code != http.StatusCreated {
			t.Fatalf("scope removal status=%d body=%s", code, body)
		}
		bounded := read(1)
		if len(bounded.Revisions) != 1 || !bounded.Truncated || len(bounded.Revisions[0].Comparison.Tasks.Removed) != 1 || bounded.Revisions[0].Comparison.Tasks.Removed[0].Key != oldTask || len(bounded.Revisions[0].Comparison.Ideas.Removed) != 1 || len(bounded.Revisions[0].Comparison.Effective.Removed) != 1 || !bounded.Revisions[0].Comparison.EdgesChanged {
			t.Fatal("page-boundary comparison lost removed scope", bounded)
		}
		full := read(5)
		if len(full.Revisions) != 4 || full.Truncated || !full.Revisions[3].Comparison.Initial {
			t.Fatal("complete scope chain was not retained", full)
		}
		before := state()
		for _, actor := range []collaboration.ActorID{0, 2, 9007199254740993} {
			if view, err := requests.ScopeHistory(ctx, actor, generateIntentID, 5); !errors.Is(err, factoryservice.ErrRequestNotFound) || view.Intent != "" {
				t.Fatal("scope history exposed another owner's request", actor, err)
			}
		}
		for _, suffix := range []string{"?limit=0", "?limit=6", "?extra=1", "?"} {
			if code, body := call(http.MethodGet, "/api/factory/intents/"+generateIntentID+"/revisions"+suffix, ""); code != http.StatusBadRequest {
				t.Fatalf("invalid history query status=%d body=%s", code, body)
			}
		}
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := requests.ScopeHistory(cancelled, 1, generateIntentID, 1); !errors.Is(err, context.Canceled) {
			t.Fatal("cancelled history served partial facts", err)
		}
		if state() != before {
			t.Fatal("refused history reads mutated retained work")
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		resumed, err := (factoryservice.Requests{Store: reopened}).ScopeHistory(ctx, 1, generateIntentID, 5)
		if err != nil {
			t.Fatal(err)
		}
		full.ObservedUnix = resumed.ObservedUnix
		a, _ := json.Marshal(full)
		b, _ := json.Marshal(resumed)
		if string(a) != string(b) || state() != before || fake.submits != 1 {
			t.Fatal("store reopen changed the scope history or started native work")
		}
		// The same private scope chain is usable by a direct HTTP host over
		// the reopened canonical store, without native session impersonation.
		transport := factoryhttpapi.Requests{Service: factoryservice.Requests{Store: reopened}, Host: product.FountainIdentities.HostID(), Actor: 1,
			Current: func(w http.ResponseWriter, r *http.Request) bool {
				if r.Header.Get("Authorization") != "Bearer declared-scope-client" {
					http.Error(w, "declared account admission unavailable", 401)
					return false
				}
				return true
			}}
		server := httptest.NewServer(transport.Handler())
		t.Cleanup(server.Close)
		direct := func(path string, status int) []byte {
			t.Helper()
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Authorization", "Bearer declared-scope-client")
			response, err := server.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != status {
				t.Fatalf("direct scope HTTP: %d %s %v", response.StatusCode, body, err)
			}
			return body
		}
		path := "/api/factory/intents/" + generateIntentID + "/revisions?limit=5"
		var served factoryservice.RevisionHistory
		if err := json.Unmarshal(direct(path, 200), &served); err != nil {
			t.Fatal(err)
		}
		served.ObservedUnix = full.ObservedUnix
		b, _ = json.Marshal(served)
		if string(a) != string(b) {
			t.Fatal("direct HTTP lost exact scope comparison", string(a), string(b))
		}
		direct("/api/factory/intents/"+generateIntentID+"/revisions?limit=6", 400)
		server.Close()
		transport.Actor = 9007199254740993
		server = httptest.NewServer(transport.Handler())
		t.Cleanup(server.Close)
		direct(path, 404)
		if state() != before || fake.submits != 1 {
			t.Fatal("direct HTTP scope reads changed journals or drove native effects")
		}
	})
	t.Run("generation_revision_survives_authority_gap_and_reopen", func(t *testing.T) {
		ctx := t.Context()
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		fake.store = reopened
		head, err := reopened.IntentRevision(ctx, generateIntentID, 1)
		if err != nil {
			t.Fatal(err)
		}
		head.Predecessor, head.ID, head.State = head.ID, "d"+strings.Repeat("8", 24), "accepted"
		if err := reopened.SaveIntentRevision(ctx, head); err != nil {
			t.Fatal(err)
		}
		expected := head.ID
		newer := head
		newer.Predecessor, newer.ID = head.ID, "d"+strings.Repeat("7", 24)
		coordinator := &control.Coordinator{Store: reopened, Graph: fake, GraphActorID: 21, AcceptanceReads: fake, Planning: &control.PlanningJudgments{Chooser: q05RefusingChooser{}}}
		admitted := false
		requests := factoryservice.Requests{Store: reopened, WriteAccess: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
			if actor != 1 || repository != 7 {
				return errors.New("wrong admission")
			}
			if !admitted {
				admitted = true
				return reopened.SaveIntentRevision(ctx, newer)
			}
			return nil
		}}
		revokedReads := 0
		transport := factoryhttpapi.Requests{Service: requests, Planning: factoryservice.RequestPlanning{Requests: requests, Generator: coordinator}, Host: product.FountainIdentities.HostID(), Actor: 1,
			Current: func(w http.ResponseWriter, r *http.Request) bool {
				if r.Header.Get("X-Read-Revoked") == "yes" {
					revokedReads++
					if revokedReads == 2 {
						http.Error(w, "declared account retired", 401)
						return false
					}
				}
				return true
			}}
		server := httptest.NewServer(transport.Handler())
		defer server.Close()
		nativeChecks := len(fake.checks)
		response, err := server.Client().Post(server.URL+"/api/factory/intents/"+generateIntentID+"/generate", "application/json", strings.NewReader(`{"revision":"`+expected+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != 412 || !strings.Contains(string(body), "stale_revision") || !admitted {
			t.Fatal("authority gap retargeted generation", response.StatusCode, string(body), err)
		}
		// Direct creation, chosen-plan adoption and background resume retain the
		// caller's original revision too; none may observe the replacement graph.
		if _, _, err := coordinator.GenerateAcceptedIntentFromRecordedPlan(ctx, generateIntentID, expected, 1); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal("creation lost expected revision", err)
		}
		if _, err := coordinator.GenerateAcceptedIntent(ctx, generateIntentID, expected, 1); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal("resume lost expected revision", err)
		}
		if _, err := coordinator.GenerateAcceptedIntent(ctx, generateIntentID, acceptID, 1); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal("completed historical generation retargeted resume", err)
		}
		late, err := server.Client().Post(server.URL+"/api/factory/intents/"+generateIntentID+"/generate", "application/json", strings.NewReader(`{"revision":"`+acceptID+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		late.Body.Close()
		if late.StatusCode != 412 {
			t.Fatal("historical generation POST selected a replacement acceptance", late.StatusCode)
		}
		chosen := control.ChosenIntentPlan{Intent: generateIntentID, Revision: generateDraftID, Operation: factory.IntentPlanningOptions, Attempt: strings.Repeat("a", 32)}
		if _, err := coordinator.GenerateAcceptedIntentFromChoice(ctx, generateIntentID, expected, 1, chosen); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal("choice adoption lost expected revision", err)
		}
		if len(fake.checks) != nativeChecks || fake.submits != 1 {
			t.Fatal("stale generation observed or submitted a graph", len(fake.checks), fake.submits)
		}
		if _, err := reopened.IntentGeneration(ctx, newer.ID); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("replacement head gained a generation", err)
		}
		original, err := reopened.IntentGeneration(ctx, acceptID)
		if err != nil || original.OperationID != captured.OperationID || string(original.Receipt) != string(captured.Receipt) {
			t.Fatal("reopen lost original generation receipt", err)
		}
		// Historical receipt reads remain owner-private after the head advances;
		// absence is explicit, and no read reaches Graph/chooser or changes journals.
		inspect, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer inspect.Close()
		snapshot := func() string {
			var digest string
			err := inspect.QueryRowContext(ctx, `SELECT md5(jsonb_build_array(
      (SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM factory_intent_heads h),
      (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_intent_revisions r),
      (SELECT jsonb_agg(to_jsonb(g) ORDER BY revision) FROM factory_intent_generations g),
      (SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM factory_intent_tasks t),
      (SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c),
      (SELECT jsonb_agg(data ORDER BY id) FROM factory_assignments),
      (SELECT jsonb_agg(data ORDER BY id) FROM factory_runs),
      (SELECT jsonb_agg(data ORDER BY run) FROM factory_usage),
      (SELECT jsonb_agg(to_jsonb(j) ORDER BY snapshot) FROM factory_priority_judgments j))::text)`).Scan(&digest)
			if err != nil {
				t.Fatal(err)
			}
			return digest
		}
		before := snapshot()
		generationPath := "/api/factory/intents/" + generateIntentID + "/revisions/" + acceptID + "/generation"
		read := func(method, path string, status int, revoked bool) []byte {
			t.Helper()
			r, err := http.NewRequestWithContext(ctx, method, server.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if revoked {
				r.Header.Set("X-Read-Revoked", "yes")
			}
			response, err := server.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != status {
				t.Fatal("saved generation read", path, response.StatusCode, string(body), err)
			}
			if status == 200 && response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("generation receipt was cacheable")
			}
			return body
		}
		var served factory.IntentGeneration
		if err := json.Unmarshal(read(http.MethodGet, generationPath, 200, false), &served); err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal(served)
		want, _ := json.Marshal(original)
		if string(got) != string(want) {
			t.Fatal("historical read changed the captured generation")
		}
		var absent struct {
			Intent   string `json:"intent_id"`
			Revision string `json:"revision_id"`
			State    string `json:"state"`
		}
		if err := json.Unmarshal(read(http.MethodGet, "/api/factory/intents/"+generateIntentID+"/revisions/"+newer.ID+"/generation", 200, false), &absent); err != nil || absent.Intent != generateIntentID || absent.Revision != newer.ID || absent.State != "not_requested" {
			t.Fatal("generation absence lost its exact scope", absent, err)
		}
		for _, suffix := range []string{"?", "?revision=" + newer.ID, "?owner=1"} {
			read(http.MethodGet, generationPath+suffix, 400, false)
		}
		read(http.MethodPost, generationPath, 405, false)
		read(http.MethodGet, "/api/factory/intents/"+generateIntentID+"/revisions/"+generateDraftID+"/generation", 404, false)
		read(http.MethodGet, "/api/factory/intents/"+generateIntentID+"/revisions/d999999999999999999999999/generation", 404, false)
		read(http.MethodGet, "/api/factory/intents/dbadbadbadbadbadbadbadbad/revisions/"+acceptID+"/generation", 404, false)
		if denied := read(http.MethodGet, generationPath, 401, true); strings.Contains(string(denied), captured.OperationID) {
			t.Fatal("retired read disclosed generation")
		}
		foreign := transport
		foreign.Actor = 9007199254740993
		foreignServer := httptest.NewServer(foreign.Handler())
		defer foreignServer.Close()
		denied, err := foreignServer.Client().Get(foreignServer.URL + generationPath)
		if err != nil {
			t.Fatal(err)
		}
		denied.Body.Close()
		if denied.StatusCode != 404 {
			t.Fatal("foreign actor read generation", denied.StatusCode)
		}
		if snapshot() != before || len(fake.checks) != nativeChecks || fake.submits != 1 {
			t.Fatal("saved generation read changed facts or drove native work")
		}
	})
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

	code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{"revision":"`+acceptID+`"}`)
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
// no recorded plan report planning_choice_unavailable (even the refusing chooser never fires),
// and an accepted head in an invisible repository reports not_found.
func TestGenerateRecordedChoiceRefusals(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	ctx := context.Background()

	if code, body := call(http.MethodPost, "/api/factory/intents", generateOptionsDraft()); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	fake := wireGenerateCoordinator(product, s, t, q05RefusingChooser{})
	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{"revision":"`+generateDraftID+`"}`); code != http.StatusConflict || errorCode(body) != "intent_not_accepted" {
		t.Fatalf("draft generate status=%d body=%s", code, body)
	}

	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	if code, body := call(http.MethodPost, "/api/factory/intents", generateAcceptedBody(acceptID)); code != http.StatusCreated {
		t.Fatalf("accept status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{"revision":"`+acceptID+`"}`); code != http.StatusConflict || errorCode(body) != "planning_choice_unavailable" {
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
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intent2+"/generate", `{"revision":"`+accept2.ID+`"}`); code != http.StatusNotFound || errorCode(body) != "not_found" {
		t.Fatalf("hidden-repo generate status=%d body=%s", code, body)
	}
	if fake.submits != 0 {
		t.Fatalf("hidden-repo generate submitted a graph: %d", fake.submits)
	}
}

// A completed proposal for an authored task-board scope reaches the same exact
// recorded-choice serving path, including cached finite comparative funding.
func TestGenerateRecordedProposalAdoptsAssessedGraph(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	const acceptID = "dbbbbbbbbbbbbbbbbbbbbbbbb"
	recordGeneratePlan(t, product, s, call, "propose_intent", acceptID, "keep", "split")
	fake := wireGenerateCoordinator(product, s, t, q05KeepChooser{})
	for pass := 0; pass < 2; pass++ {
		code, body := call(http.MethodPost, "/api/factory/intents/"+generateIntentID+"/generate", `{"revision":"`+acceptID+`"}`)
		if code != http.StatusOK {
			t.Fatalf("propose generate status=%d body=%s", code, body)
		}
		var result factory.IntentGeneration
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		if result.State != "completed" || len(result.SubmittedTasks) != 2 || result.SubmittedTasks[0].Key != generateTaskA || result.PlanAssignment == "" || fake.submits != 1 {
			t.Fatalf("proposal graph not exact or replayed twice: %+v submits=%d", result, fake.submits)
		}
	}
}

// Resolved-scope regression identities: a dedicated intent with decided
// accept/edit ideas, one pending idea and one declined idea. All IDs are
// valid decision IDs distinct from the generate-commitment fixtures.
const (
	resolvedScopeIntent   = "da10000000000000000000001"
	resolvedScopeDraft    = "da10000000000000000000002"
	resolvedScopeAccept   = "da10000000000000000000003"
	resolvedScopePend     = "da10000000000000000000004"
	resolvedScopeEdit     = "da10000000000000000000005"
	resolvedScopeDecl     = "da10000000000000000000006"
	resolvedScopeSucc     = "da10000000000000000000007"
	resolvedScopeAccepted = "da10000000000000000000008"
)

// resolvedScopeDraftBody saves a draft with nil effective outcomes: one
// accept, one pending (autonomous, so it stays undecided without forged
// human attribution) and one edit with a human correction.
func resolvedScopeDraftBody() string {
	return `{"id":"` + resolvedScopeDraft + `","intent":"` + resolvedScopeIntent + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":[{"id":"` + resolvedScopeAccept + `","description":"Capture leave requests","consequence":"Requests can be recorded",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
		`{"id":"` + resolvedScopePend + `","description":"Sync to the legacy tracker","consequence":"Legacy stays current",` +
		`"decision":"pending","source":"autonomous","provenance":"autonomous"},` +
		`{"id":"` + resolvedScopeEdit + `","description":"Delete requests automatically","consequence":"Old requests vanish","correction":"Retain requests until a manager decides",` +
		`"decision":"edit","source":"requested","provenance":"human","actor":"1"}],` +
		`"state":"draft","max_issues":10,"planning_calls":12}`
}

// q05ResolvedWalkthrough authors deterministic planning-agent prose bound
// to the given idea set, mirroring q05WalkthroughContent without reading
// the raw effective set.
func q05ResolvedWalkthrough(t *testing.T, head factory.IntentRevision, ids ...string) factory.ProposalContent {
	t.Helper()
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatal(err)
	}
	content.Producer, content.Supported = factory.PlanningProducer, true
	content.Alternatives = nil
	for _, key := range []string{"guided", "list"} {
		content.Alternatives = append(content.Alternatives, factory.ProductAlternative{
			Key: key, Name: key + " project planning", Audience: "The project owner", Summary: "Plan the recorded draft with visible progress.",
			Steps:      []factory.WalkthroughStep{{Actor: "The project owner", Action: "reviews the saved request", Result: "The planned work is visible"}},
			Visibility: "The owner sees the saved plan", Consequences: "The request stays editable before acceptance", Evidence: "The admitted request and inspected source", Outcomes: []string{head.Objective}, IdeaIDs: append([]string(nil), ids...),
		})
	}
	return content
}

// TestResolvedScopeDerivedBindingValidates proves the FUX repair through
// the real mux: a draft saved with nil effective outcomes resolves to its
// decided accept/edit ideas in revision order; a walkthrough binding
// exactly that set validates while pending inclusion refuses. Declared
// external judgments: none. No live model call.
func TestResolvedScopeDerivedBindingValidates(t *testing.T) {
	_, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", resolvedScopeDraftBody()); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, resolvedScopeIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	if resolved := factory.ResolvedScope(head); !slices.Equal(resolved, []string{resolvedScopeAccept, resolvedScopeEdit}) {
		t.Fatalf("resolved=%v want accept+edit in revision order", resolved)
	}
	if err := factory.ValidatePlanningWalkthroughs(head, q05ResolvedWalkthrough(t, head, resolvedScopeAccept, resolvedScopeEdit)); err != nil {
		t.Fatalf("exact derived binding refused: %v", err)
	}
	if err := factory.ValidatePlanningWalkthroughs(head, q05ResolvedWalkthrough(t, head, resolvedScopeAccept, resolvedScopePend, resolvedScopeEdit)); err == nil {
		t.Fatal("pending inclusion validated against the decided scope")
	}
}

// TestResolvedScopeExplicitOverrideBindsVerbatim proves an explicit
// nonempty subset still resolves verbatim: the walkthrough covering the
// override validates, extra decided IDs refuse, and exact head identity
// (intent/revision binding) still refuses on mismatch. No live model call.
func TestResolvedScopeExplicitOverrideBindsVerbatim(t *testing.T) {
	_, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	draft := `{"id":"` + resolvedScopeDraft + `","intent":"` + resolvedScopeIntent + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":[{"id":"` + resolvedScopeAccept + `","description":"Capture leave requests","consequence":"Requests can be recorded",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
		`{"id":"` + resolvedScopeEdit + `","description":"Managers review requests","consequence":"Decisions are visible","correction":"Managers review requests in one queue",` +
		`"decision":"edit","source":"requested","provenance":"human","actor":"1"}],` +
		`"effective_outcomes":["` + resolvedScopeAccept + `"],` +
		`"state":"draft","max_issues":10,"planning_calls":12}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, resolvedScopeIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	if resolved := factory.ResolvedScope(head); !slices.Equal(resolved, []string{resolvedScopeAccept}) {
		t.Fatalf("resolved=%v want explicit override verbatim", resolved)
	}
	if err := factory.ValidatePlanningWalkthroughs(head, q05ResolvedWalkthrough(t, head, resolvedScopeAccept)); err != nil {
		t.Fatalf("override binding refused: %v", err)
	}
	if err := factory.ValidatePlanningWalkthroughs(head, q05ResolvedWalkthrough(t, head, resolvedScopeAccept, resolvedScopeEdit)); err == nil {
		t.Fatal("extra decided ID validated against the explicit override")
	}
	foreign := q05ResolvedWalkthrough(t, head, resolvedScopeAccept)
	foreign.Revision = resolvedScopeSucc
	if err := factory.ValidatePlanningWalkthroughs(head, foreign); err == nil {
		t.Fatal("mismatched revision binding validated")
	}
	foreign = q05ResolvedWalkthrough(t, head, resolvedScopeAccept)
	foreign.Intent = generateIntentID
	if err := factory.ValidatePlanningWalkthroughs(head, foreign); err == nil {
		t.Fatal("mismatched intent binding validated")
	}
}

// TestResolvedScopeSuccessorPreservesCorrection proves correction coverage
// across revisions: a successor draft preserving the decided pattern
// resolves identically, its exact binding validates, and a
// scope-preserving accepted successor still saves (acceptance admission
// unchanged). No live model call.
func TestResolvedScopeSuccessorPreservesCorrection(t *testing.T) {
	_, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", resolvedScopeDraftBody()); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, resolvedScopeIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Ideas json.RawMessage `json:"ideas"`
	}
	if err := json.Unmarshal([]byte(resolvedScopeDraftBody()), &raw); err != nil {
		t.Fatal(err)
	}
	successor := `{"id":"` + resolvedScopeSucc + `","intent":"` + resolvedScopeIntent + `","predecessor":"` + resolvedScopeDraft + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":` + string(raw.Ideas) + `,` +
		`"state":"draft","max_issues":10,"planning_calls":12}`
	if code, body := call(http.MethodPost, "/api/factory/intents", successor); code != http.StatusCreated {
		t.Fatalf("successor save status=%d body=%s", code, body)
	}
	next, err := s.IntentRevision(ctx, resolvedScopeIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID != resolvedScopeSucc || next.Predecessor != resolvedScopeDraft {
		t.Fatalf("successor identity=%s predecessor=%s", next.ID, next.Predecessor)
	}
	if resolved := factory.ResolvedScope(next); !slices.Equal(resolved, factory.ResolvedScope(head)) || !slices.Equal(resolved, []string{resolvedScopeAccept, resolvedScopeEdit}) {
		t.Fatalf("successor resolved=%v want preserved accept+edit pattern", resolved)
	}
	if err := factory.ValidatePlanningWalkthroughs(next, q05ResolvedWalkthrough(t, next, resolvedScopeAccept, resolvedScopeEdit)); err != nil {
		t.Fatalf("successor exact binding refused: %v", err)
	}
	accepted := `{"id":"` + resolvedScopeAccepted + `","intent":"` + resolvedScopeIntent + `","predecessor":"` + resolvedScopeSucc + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":` + string(raw.Ideas) + `,` +
		`"effective_outcomes":["` + resolvedScopeAccept + `","` + resolvedScopeEdit + `"],` +
		`"tasks":[{"key":"` + generateAuthored + `","objective":"Build the leave tracker","outcomes":["Requests list"],"idea_ids":["` + resolvedScopeAccept + `"]}],` +
		`"state":"accepted","max_issues":10,"planning_calls":12}`
	if code, body := call(http.MethodPost, "/api/factory/intents", accepted); code != http.StatusCreated {
		t.Fatalf("scope-preserving accept status=%d body=%s", code, body)
	}
}

// TestResolvedScopeEmptyRefusesTruthfully proves the FJ-338 edge: with no
// decided accept/edit ideas (all pending/declined) the resolved scope is
// empty and every walkthrough refuses, including an empty binding. The fix
// never fabricates authority, never approves pending choices, and leaves
// acceptance behavior to the existing gates. No live model call.
func TestResolvedScopeEmptyRefusesTruthfully(t *testing.T) {
	_, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	draft := `{"id":"` + resolvedScopeDraft + `","intent":"` + resolvedScopeIntent + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":[{"id":"` + resolvedScopePend + `","description":"Sync to the legacy tracker","consequence":"Legacy stays current",` +
		`"decision":"pending","source":"autonomous","provenance":"autonomous"},` +
		`{"id":"` + resolvedScopeDecl + `","description":"Delete requests automatically","consequence":"Old requests vanish",` +
		`"decision":"decline","source":"requested","provenance":"human","actor":"1"}],` +
		`"state":"draft","max_issues":10,"planning_calls":12}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, resolvedScopeIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	if resolved := factory.ResolvedScope(head); len(resolved) != 0 {
		t.Fatalf("resolved=%v want empty with no decided ideas", resolved)
	}
	for name, ids := range map[string][]string{
		"empty binding":    {},
		"pending approval": {resolvedScopePend},
		"declined revival": {resolvedScopeDecl},
	} {
		if err := factory.ValidatePlanningWalkthroughs(head, q05ResolvedWalkthrough(t, head, ids...)); err == nil {
			t.Fatalf("%s validated against an empty resolved scope", name)
		}
	}
}

// Explicit-empty vs absent array identity: two drafts with the same
// product content and distinct intent/row identity, one posting
// `"effective_outcomes":[]` and the other omitting the key.
const (
	emptyArrayIntent  = "da20000000000000000000001"
	emptyArrayDraft   = "da20000000000000000000002"
	absentArrayIntent = "da20000000000000000000003"
	absentArrayDraft  = "da20000000000000000000004"
	emptyArrayAccept  = "da20000000000000000000005"
	emptyArrayEdit    = "da20000000000000000000006"
)

// emptyArrayDraftBody saves one accept and one human edit idea; withEmpty
// adds an explicit empty effective_outcomes array, without omits the key.
func emptyArrayDraftBody(intent, id string, withEmpty bool) string {
	body := `{"id":"` + id + `","intent":"` + intent + `","repository":"7",` +
		`"objective":"Build a leave approval workflow so managers can review time-off requests",` +
		`"ideas":[{"id":"` + emptyArrayAccept + `","description":"Capture leave requests","consequence":"Requests can be recorded",` +
		`"decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
		`{"id":"` + emptyArrayEdit + `","description":"Managers review requests","consequence":"Decisions are visible","correction":"Managers review requests in one queue",` +
		`"decision":"edit","source":"requested","provenance":"human","actor":"1"}],`
	if withEmpty {
		body += `"effective_outcomes":[],`
	}
	return body + `"state":"draft","max_issues":10,"planning_calls":12}`
}

// normalizeSavedBody drops the volatile identity/clock fields so two saves
// of the same product content compare byte-for-byte.
func normalizeSavedBody(t *testing.T, body []byte) string {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	delete(decoded, "id")
	delete(decoded, "intent")
	delete(decoded, "created_unix")
	normalized, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	return string(normalized)
}

// TestResolvedScopeExplicitEmptyArrayMatchesAbsent proves the explicit-empty
// array is behaviorally absent through the real mux: both saves 201, neither
// saved payload retains the key, both stored heads carry no effective
// subset, the resolved scope derives the same decided accept/edit pair, and
// the saved payloads are byte-identical once volatile identity/clock fields
// are normalized. No production change. No live model call.
func TestResolvedScopeExplicitEmptyArrayMatchesAbsent(t *testing.T) {
	_, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	code, explicitBody := call(http.MethodPost, "/api/factory/intents", emptyArrayDraftBody(emptyArrayIntent, emptyArrayDraft, true))
	if code != http.StatusCreated {
		t.Fatalf("explicit-empty save status=%d body=%s", code, explicitBody)
	}
	code, absentBody := call(http.MethodPost, "/api/factory/intents", emptyArrayDraftBody(absentArrayIntent, absentArrayDraft, false))
	if code != http.StatusCreated {
		t.Fatalf("absent save status=%d body=%s", code, absentBody)
	}
	if strings.Contains(string(explicitBody), "effective_outcomes") || strings.Contains(string(absentBody), "effective_outcomes") {
		t.Fatalf("saved payload retained an empty effective subset:\n%s\n%s", explicitBody, absentBody)
	}
	if normalizeSavedBody(t, explicitBody) != normalizeSavedBody(t, absentBody) {
		t.Fatalf("explicit-empty payload diverged from absent:\n%s\n%s", explicitBody, absentBody)
	}
	explicit, err := s.IntentRevision(ctx, emptyArrayIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := s.IntentRevision(ctx, absentArrayIntent, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(explicit.EffectiveOutcomes) != 0 || len(absent.EffectiveOutcomes) != 0 {
		t.Fatalf("stored heads carry an effective subset: %v %v", explicit.EffectiveOutcomes, absent.EffectiveOutcomes)
	}
	want := []string{emptyArrayAccept, emptyArrayEdit}
	if resolved := factory.ResolvedScope(explicit); !slices.Equal(resolved, want) {
		t.Fatalf("explicit-empty resolved=%v want decided accept+edit", resolved)
	}
	if resolved := factory.ResolvedScope(absent); !slices.Equal(resolved, want) {
		t.Fatalf("absent resolved=%v want decided accept+edit", resolved)
	}
	explicit.ID, explicit.Intent, explicit.CreatedUnix = "", "", 0
	absent.ID, absent.Intent, absent.CreatedUnix = "", "", 0
	explicitBytes, err := json.Marshal(explicit)
	if err != nil {
		t.Fatal(err)
	}
	absentBytes, err := json.Marshal(absent)
	if err != nil {
		t.Fatal(err)
	}
	if string(explicitBytes) != string(absentBytes) {
		t.Fatalf("stored heads diverged:\n%s\n%s", explicitBytes, absentBytes)
	}
}
