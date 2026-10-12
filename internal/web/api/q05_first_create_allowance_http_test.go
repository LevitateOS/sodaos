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

// TestFirstCreateAllowanceHTTPBoundary exercises the actual product API with
// browser-emitted allowance payloads: fresh draft -> 404 allowance, current
// factory-work shape (no ceiling) -> honest 400, U contract shape
// (ceiling=amount) -> 201, funded explore consumes one reservation (503
// without TypeSafe credentials, never 409), and an amount-only CAS leaves the
// intent head and generation state untouched. Declared external judgment:
// none consulted (TypeSafe key absent, so the funded gate is proven by the
// consumed reservation, not a live model answer).
func TestFirstCreateAllowanceHTTPBoundary(t *testing.T) {
	_, s, call, errorCode := q05Mux(t, "")

	const draftID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"accept"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", ""); code != http.StatusNotFound || errorCode(body) != "allowance_not_found" {
		t.Fatalf("fresh allowance status=%d body=%s", code, body)
	}
	// Current factory-work browser shape: positive amount, no ceiling.
	workShape := `{"id":"d444444444444444444444444","intent":"` + intentID + `","amount":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", workShape); code != http.StatusBadRequest || errorCode(body) != "invalid_allowance" {
		t.Fatalf("factory-work shape status=%d body=%s", code, body)
	}
	// U contract shape: ceiling=amount least privilege.
	uShape := `{"id":"d555555555555555555555555","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", uShape); code != http.StatusCreated {
		t.Fatalf("U allowance shape status=%d body=%s", code, body)
	}
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/explore", `{"revision":"`+draftID+`"}`)
	if code == http.StatusConflict {
		t.Fatalf("funded explore refused: %s", body)
	}
	if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" {
		t.Fatalf("explore without TypeSafe credentials status=%d body=%s", code, body)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", "")
	if code != http.StatusOK {
		t.Fatalf("allowance read status=%d body=%s", code, body)
	}
	var funded factory.PlanningAllowance
	if err := json.Unmarshal(body, &funded); err != nil {
		t.Fatal(err)
	}
	if funded.Consumed != 1 || funded.Reserved != 0 {
		t.Fatalf("funded explore consumed=%d reserved=%d", funded.Consumed, funded.Reserved)
	}
	// Amount-only CAS: new id + predecessor + live usage echo, head untouched.
	funded.ID = "d666666666666666666666666"
	funded.Predecessor = "d555555555555555555555555"
	raw, err := json.Marshal(funded)
	if err != nil {
		t.Fatal(err)
	}
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", string(raw)); code != http.StatusCreated {
		t.Fatalf("amount-only CAS status=%d body=%s", code, body)
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
		t.Fatalf("allowance CAS moved the intent head to %s", head.ID)
	}
	if _, err := s.IntentGeneration(context.Background(), draftID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("generation state err=%v", err)
	}
}

// q05Mux builds the real product mux on an ephemeral PG store with a fake
// native callback authority. keyFile, when non-empty, enables live TypeSafe
// judgments; empty keeps the deterministic no-credentials boundary.
func q05Mux(t *testing.T, keyFile string) (*API, *store.Store, func(method, path, body string) (int, []byte), func(body []byte) string) {
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
	s, _, cleanup, err := store.OpenEphemeral(context.Background(), admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	s.PriorityCallBudget = 32
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
	return product, s, call, errorCode
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
	uShape := `{"id":"d555555555555555555555555","intent":"` + intentID + `","amount":8,"ceiling":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", uShape); code != http.StatusCreated {
		t.Fatalf("%s U allowance shape status=%d body=%s", label, code, body)
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
	var funded factory.PlanningAllowance
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", ""); code != http.StatusOK {
		t.Fatalf("%s allowance read status=%d", label, code)
	} else if err := json.Unmarshal(body, &funded); err != nil {
		t.Fatal(err)
	}
	if funded.Consumed < 1 {
		t.Fatalf("%s live judgments consumed=%d", label, funded.Consumed)
	}
	head, err := s.IntentRevision(context.Background(), intentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if head.ID != explored.ID {
		t.Fatalf("%s stored head %s != explored %s", label, head.ID, explored.ID)
	}
}
