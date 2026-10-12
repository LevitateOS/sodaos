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
	suffix := "q05-" + strconv.Itoa(os.Getpid())
	callbackPath := "@soda-callback-" + suffix
	callbackListener, err := net.Listen("unix", callbackPath)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil || in.Operation != extensions.OperationCurrentActor {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}})
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
	product := &API{Config: &config.Config{ForgejoURL: "https://soda.test"}, Store: s}
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
		var reader *strings.Reader
		if body == "" {
			reader = strings.NewReader("")
		} else {
			reader = strings.NewReader(body)
		}
		r := httptest.NewRequest(method, path, reader)
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
	errorCode := func(t *testing.T, body []byte) string {
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

	const draftID = "d111111111111111111111111"
	const intentID = "d222222222222222222222222"
	draft := `{"id":"` + draftID + `","intent":"` + intentID + `","objective":"Time-off requests my manager can approve","state":"draft","max_issues":10,"planning_calls":8,` +
		`"ideas":[{"id":"d333333333333333333333333","description":"Track leave requests","consequence":"Managers review requests","decision":"accept"}]}`
	if code, body := call(http.MethodPost, "/api/factory/intents", draft); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/factory/intents/"+intentID+"/allowance", ""); code != http.StatusNotFound || errorCode(t, body) != "allowance_not_found" {
		t.Fatalf("fresh allowance status=%d body=%s", code, body)
	}
	// Current factory-work browser shape: positive amount, no ceiling.
	workShape := `{"id":"d444444444444444444444444","intent":"` + intentID + `","amount":8,"unit":"planning_calls"}`
	if code, body := call(http.MethodPost, "/api/factory/intent-allowances", workShape); code != http.StatusBadRequest || errorCode(t, body) != "invalid_allowance" {
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
	if code != http.StatusServiceUnavailable || errorCode(t, body) != "planning_unavailable" {
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
