package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// TestFirstCreateAllowanceHTTPBoundary exercises the actual product API: a
// fresh draft explores without any allowance (503 without TypeSafe
// credentials, never 409) and leaves the intent head and generation state
// untouched.
func TestFirstCreateAllowanceHTTPBoundary(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")

	const draftID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"accept"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" {
		t.Fatalf("explore without TypeSafe credentials status=%d body=%s", code, body)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/"+intentID, "")
	if code != http.StatusOK {
		t.Fatalf("intent read status=%d body=%s", code, body)
	}
	var head factory.IntentRevision
	if err := json.Unmarshal(body, &head); err != nil {
		t.Fatal(err)
	}
	if head.ID != draftID {
		t.Fatalf("failed explore moved the intent head to %s", head.ID)
	}
	if _, err := s.IntentGeneration(context.Background(), draftID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("generation state err=%v", err)
	}
	t.Run("explicit repository repair funding through authenticated HTTP", func(t *testing.T) {
		path := "/api/repositories/7/factory/repair"
		if code, body := call(http.MethodGet, path, ""); code != 200 || !strings.Contains(string(body), `"revision":0`) {
			t.Fatalf("unfunded read: %d %s", code, body)
		}
		first := factoryRepairGrantRequest{CommandID: factory.NewID(), InvestigatorRuns: 1, ProposalJudgments: 1, ActiveMinutes: 30, Active: true}
		encoded, _ := json.Marshal(first)
		for i := 0; i < 2; i++ {
			if code, body := call(http.MethodPut, path, string(encoded)); code != 200 {
				t.Fatalf("grant/replay: %d %s", code, body)
			}
		}
		grant, err := s.RepairGrantCommand(context.Background(), 7)
		if err != nil || grant.Sponsor != 1 || grant.Revision != 0 || grant.ID != first.CommandID {
			t.Fatal("grant lost authenticated sponsor", err)
		}
		charge := factory.RepairCharge{ID: factory.NewID(), GrantID: grant.ID, Kind: factory.RepairChargeRun, EpisodeID: factory.NewID(), Cycle: 1, EvidenceDigest: strings.Repeat("a", 64), ActiveMinutes: 10, SpentUnix: time.Now().Unix()}
		if _, spent, err := s.AdmitStallInvestigation(context.Background(), 7, charge, time.Now()); err != nil || !spent {
			t.Fatal("canonical charge", err)
		}
		conflict := first
		conflict.ActiveMinutes = 29
		bad, _ := json.Marshal(conflict)
		if code, body := call(http.MethodPut, path, string(bad)); code != 409 || errorCode(body) != "command_conflict" {
			t.Fatalf("changed replay: %d %s", code, body)
		}
		stale := first
		stale.CommandID = factory.NewID()
		bad, _ = json.Marshal(stale)
		if code, body := call(http.MethodPut, path, string(bad)); code != 412 {
			t.Fatalf("stale head: %d %s", code, body)
		}
		raised := factoryRepairGrantRequest{CommandID: factory.NewID(), ExpectedRevision: 1, InvestigatorRuns: 2, ProposalJudgments: 2, ActiveMinutes: 60, Effect: "Owner explicitly funds one additional bounded investigation without refilling spent work.", Active: true}
		more, _ := json.Marshal(raised)
		if code, body := call(http.MethodPut, path, string(more)); code != 200 {
			t.Fatalf("explicit raise: %d %s", code, body)
		}
		if code, body := call(http.MethodPut, path, string(encoded)); code != 200 {
			t.Fatalf("old exact receipt replay: %d %s", code, body)
		}
		code, body := call(http.MethodGet, path, "")
		var view struct {
			Revision int64                  `json:"revision"`
			Grant    factory.RepairGrant    `json:"grant"`
			Charges  []factory.RepairCharge `json:"charges"`
		}
		if json.Unmarshal(body, &view) != nil || code != 200 || view.Revision != 2 || view.Grant.Sponsor != 1 || len(view.Charges) != 1 || view.Charges[0].ID != charge.ID {
			t.Fatalf("funding/readback refilled history: %d %s", code, body)
		}
		foreign := actorCall(t, product, "2", "bob")
		if code, _ := foreign(http.MethodPut, path, string(more)); code < 400 {
			t.Fatal("foreign actor funded repair")
		}
		charges, err := s.RepairCharges(context.Background(), 7)
		if err != nil || len(charges) != 1 {
			t.Fatal("read/refusal charged work", err)
		}
	})
	t.Run("owner verification allowance preserves immutable receipts", func(t *testing.T) {
		ctx := context.Background()
		idea := factory.ProductIdea{ID: "d121212121212121212121212", Description: "Store a task", Consequence: "Tasks remain readable", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}
		task := factory.IntentTask{Key: "d232323232323232323232323", Objective: "Implement task storage", Outcomes: []string{idea.Consequence}, IdeaIDs: []string{idea.ID}}
		head := factory.IntentRevision{ID: "d343434343434343434343434", Intent: "d454545454545454545454545", Owner: 1, Repository: 7, Objective: "Task app", State: "accepted", MaxIssues: 1, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{task}, EffectiveOutcomes: []string{idea.ID}}
		if err := s.SaveIntentRevision(ctx, head); err != nil {
			t.Fatal(err)
		}
		path := "/api/factory/intents/" + head.Intent + "/verification-allowance"
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			if admitted, handled := extensionFactoryIntentsRoute(strings.Split("factory/intents/"+head.Intent+"/verification-allowance", "/"), method); !admitted || !handled {
				t.Fatal("native bridge omitted verification allowance", method)
			}
		}
		if admitted, _ := extensionFactoryIntentsRoute(strings.Split("factory/intents/"+head.Intent+"/verification-allowance", "/"), http.MethodPut); admitted {
			t.Fatal("native bridge admitted another mutation method")
		}
		code, body := call(http.MethodGet, path, "")
		var initial factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(body, &initial) != nil || initial.Intent != head.Intent || initial.Revision != "" || initial.CommandID != "" || initial.Limits.Runs != 2 || initial.Limits.ActiveMinutes != 60 || initial.Consumption.Runs != 0 || initial.Remaining.Runs != 2 || initial.Remaining.ActiveSeconds != 3600 || !initial.CanIncrease {
			t.Fatalf("default verification read: %d %s", code, body)
		}
		first := factory.AppVerificationAllowanceChange{CommandID: factory.NewID(), Limits: factory.AppVerificationLimits{Runs: 3, ActiveMinutes: 60}}
		encoded, _ := json.Marshal(first)
		code, originalReceipt := call(http.MethodPost, path, string(encoded))
		var raised factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(originalReceipt, &raised) != nil || raised.Revision != first.CommandID || raised.CommandID != first.CommandID || raised.Intent != head.Intent || raised.Limits.Runs != 3 || raised.Limits.ActiveMinutes != 60 || raised.Consumption != initial.Consumption {
			t.Fatalf("owner increase: %d %s", code, originalReceipt)
		}
		second := first
		second.CommandID, second.ExpectedRevision, second.Limits.Runs = factory.NewID(), first.CommandID, 4
		more, _ := json.Marshal(second)
		if code, body := call(http.MethodPost, path, string(more)); code != 200 {
			t.Fatalf("second explicit increase: %d %s", code, body)
		}
		head.ID, head.Predecessor = "d565656565656565656565656", head.ID
		if err := s.SaveIntentRevision(ctx, head); err != nil {
			t.Fatal(err)
		}
		if code, body := call(http.MethodPost, path, string(encoded)); code != 200 || string(body) != string(originalReceipt) {
			t.Fatalf("old command did not replay its exact receipt before CAS: %d %s", code, body)
		}
		stale := first
		stale.CommandID, stale.Limits.Runs = factory.NewID(), 5
		raw, _ := json.Marshal(stale)
		if code, body := call(http.MethodPost, path, string(raw)); code != 409 || errorCode(body) != "stale_revision" {
			t.Fatalf("stale verification CAS: %d %s", code, body)
		}
		if _, err := s.FactoryCommand(ctx, stale.CommandID); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("stale increase retained a command", err)
		}
		decrease := first
		decrease.CommandID, decrease.ExpectedRevision = factory.NewID(), second.CommandID
		raw, _ = json.Marshal(decrease)
		if code, body := call(http.MethodPost, path, string(raw)); code != 409 || errorCode(body) != "increase_required" {
			t.Fatalf("decrease admitted: %d %s", code, body)
		}
		conflict := first
		conflict.Limits.Runs = 9
		raw, _ = json.Marshal(conflict)
		if code, body := call(http.MethodPost, path, string(raw)); code != 409 || errorCode(body) != "command_conflict" {
			t.Fatalf("changed command replay: %d %s", code, body)
		}
		if code, body := call(http.MethodPost, path, `{"command_id":"`+factory.NewID()+`","expected_revision":"`+second.CommandID+`","limits":{"runs":5,"active_minutes":60,"invented":true}}`); code != 400 {
			t.Fatalf("unknown limit field: %d %s", code, body)
		}
		if code, body := call(http.MethodPost, path, `{"command_id":"`+factory.NewID()+`","expected_revision":"`+second.CommandID+`","limits":{"runs":0,"active_minutes":60}}`); code != 400 {
			t.Fatalf("nonfinite limit: %d %s", code, body)
		}
		foreign := actorCall(t, product, "2", "bob")
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			if code, body := foreign(method, path, string(more)); code != 403 {
				t.Fatalf("foreign verification owner: %d %s", code, body)
			}
		}
		if code, body := call(http.MethodGet, "/api/factory/commands/"+first.CommandID, ""); code != 200 || !strings.Contains(string(body), first.CommandID) {
			t.Fatalf("immutable command readback: %d %s", code, body)
		}
		code, body = call(http.MethodGet, path, "")
		var current factory.AppVerificationAllowance
		if code != 200 || json.Unmarshal(body, &current) != nil || current.Revision != second.CommandID || current.CommandID != "" || current.Limits.Runs != 4 || current.Limits.ActiveMinutes != 60 || current.Consumption != initial.Consumption {
			t.Fatalf("current status refilled consumption: %d %s", code, body)
		}
		if runs, err := s.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
			t.Fatal("allowance reads or increases launched work", runs, err)
		}
	})
}

// q05Mux builds the real product mux on an ephemeral PG store with a fake
// native callback authority. keyFile, when non-empty, enables live TypeSafe
// judgments; empty keeps the deterministic no-credentials boundary.
func q05Mux(t *testing.T, keyFile string) (*API, *store.Store, func(method, path, body string) (int, []byte), func(body []byte) string) {
	product, s, _, call, errorCode := q05MuxFull(t, keyFile)
	return product, s, call, errorCode
}

// q05MuxFull builds the same mux as q05Mux and additionally returns the
// ephemeral database URL for seeds that have no production Store writer
// (canonical completions, controller-owned F10 facts).
func q05MuxFull(t *testing.T, keyFile string) (*API, *store.Store, string, func(method, path, body string) (int, []byte), func(body []byte) string) {
	t.Helper()
	suffix := "q05-" + strconv.Itoa(os.Getpid())
	callbackPath := "@soda-callback-" + suffix
	callbackListener, err := net.Listen("unix", callbackPath)
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
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}})
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
	callback.Listener = callbackListener
	callback.Start()
	t.Cleanup(callback.Close)
	t.Setenv(extensions.CallbackEnv, callbackPath)

	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	product := &API{Config: &config.Config{ForgejoURL: "https://soda.test", TypeSafeAPIKeyFile: keyFile}, Store: s}
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)

	call := func(method, path, body string) (int, []byte) {
		t.Helper()
		authority, err := json.Marshal(extensions.Authority{
			ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1",
			Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)},
			Actor:        extensions.Actor{ID: "1", Username: "alice"},
		})
		if err != nil {
			t.Fatal(err)
		}
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
	errorCode := func(body []byte) string {
		t.Helper()
		var decoded struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded.Error.Code
	}
	return product, s, dsn, call, errorCode
}

// TestFirstExploreLiveSuccess proves funded exploration success through the
// actual mux for a reviewed-direction draft (one accepted idea): U-contract
// allowance 201 -> explore 201 with saved proposal head and live judgments
// charged. Declared external judgment: LIVE TypeSafe API (model jev-latest
// default), keyed by TYPESAFE_API_KEY; skips when absent. Model-selected
// content is intentionally unasserted; only structural success facts are
// pinned (status, head advance, exploration presence, consumption). A 201
// here is funded-judged-saved success, not proof of coherent ready work.
func TestFirstExploreLiveSuccess(t *testing.T) {
	_, s, call, _ := q05Mux(t, q05KeyFile(t))

	const draftID = "d777777777777777777777777"
	const intentID = "d888888888888888888888888"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"accept"}]}`
	q05LiveExplore(t, s, call, draft, intentID, draftID, "reviewed-direction")
}

// TestFirstExploreFreshEntryLive is the byte-equivalent first-entry case:
// U nextRevision with no current revision emits repository "0", empty
// ideas/tasks, state draft and control defaults (max_issues 20,
// planning_calls 8). Same structural success assertions as the
// reviewed-direction case; no accept/repository/generate/native binding.
func TestFirstExploreFreshEntryLive(t *testing.T) {
	_, s, call, _ := q05Mux(t, q05KeyFile(t))

	const draftID = "d777777777777777777777777"
	const intentID = "d888888888888888888888888"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","repository":"0","objective":"Time-off requests my manager can approve",` +
		`"ideas":[],"tasks":[],"state":"draft","max_issues":20,"planning_calls":8}`
	q05LiveExplore(t, s, call, draft, intentID, draftID, "fresh-entry")
}

func q05KeyFile(t *testing.T) string {
	t.Helper()
	key := strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY"))
	if key == "" {
		t.Skip("TYPESAFE_API_KEY unavailable")
	}
	keyFile := filepath.Join(t.TempDir(), "typesafe.key")
	if err := os.WriteFile(keyFile, []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return keyFile
}

func q05LiveExplore(t *testing.T, s *store.Store, call func(method, path, body string) (int, []byte), draft, intentID, draftID, label string) {
	t.Helper()
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("%s draft save status=%d body=%s", label, code, body)
	}
	start := time.Now()
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	t.Logf("%s live explore status=%d elapsed=%s", label, code, time.Since(start).Round(time.Second))
	if code != http.StatusCreated {
		t.Fatalf("%s live explore status=%d body=%s", label, code, body)
	}
	var explored struct {
		factory.IntentRevision
		Exploration struct {
			Content factory.ProposalContent `json:"content"`
		} `json:"exploration"`
	}
	if err := json.Unmarshal(body, &explored); err != nil {
		t.Fatal(err)
	}
	if explored.ID == draftID || explored.Predecessor != draftID {
		t.Fatalf("%s explore head id=%s predecessor=%s", label, explored.ID, explored.Predecessor)
	}
	if !explored.Exploration.Content.Supported && len(explored.Exploration.Content.Unknowns) == 0 {
		t.Fatalf("%s explore returned no content", label)
	}
	head, err := s.IntentRevision(context.Background(), intentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if head.ID != explored.ID {
		t.Fatalf("%s stored head %s != explored %s", label, head.ID, explored.ID)
	}
}
