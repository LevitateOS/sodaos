package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

// Real JSON handlers/store, fake host only: not installed account/SSH evidence.
func TestJSONEnvironmentReservationAndExplicitJoins(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/alice/demo" {
			t.Error("unexpected provider path")
		}
		fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`)
	})
	accountCalls := 0
	rejectAccount := false
	id := ""
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/create":
			var input host.Create
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if input.Owner != 1 {
				t.Error("caller selected owner")
			}
			id = input.ID
			json.NewEncoder(w).Encode(host.Environment{ID: id, IP: "10.89.0.2", Running: true})
		case "/account":
			accountCalls++
			var input host.Account
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if input.Project != id || input.Identity < 1 || input.Identity > 2 || len(input.Keys) != 1 {
				t.Error("invalid fixed account request")
			}
			if rejectAccount {
				w.WriteHeader(500)
			} else {
				fmt.Fprint(w, `{"ok":true}`)
			}
		default:
			t.Error("unexpected native operation")
			w.WriteHeader(500)
		}
	}))
	defer native.Close()
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = native.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(r)
	})}
	perform := func(method, path, body, login string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest(method, path, body, login))
		return w
	}
	if w := perform("POST", "/api/environments", `{"owner":"alice","repository":"demo"}`, "bob"); w.Code != 403 || id != "" {
		t.Fatal("nonowner provisioned", w.Code)
	}
	if w := perform("POST", "/api/environments", `{"owner":"alice","repository":"demo","admin":true}`, "alice"); w.Code != 400 || id != "" {
		t.Fatal("privilege field accepted", w.Code)
	}
	if w := perform("POST", "/api/environments", `{"owner":"alice","repository":"demo"}`, "alice"); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, uid := range []int64{1, 2} {
		if _, err := s.Store.MemberLogin(context.Background(), id, uid); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal("implicit join", err)
		}
	}
	if w := perform("POST", "/api/environments/"+id+"/join", `{}`, "bob"); w.Code != 422 || accountCalls != 0 {
		t.Fatal("missing keys reached native")
	}
	for _, uid := range []int64{1, 2} {
		if err := s.Store.AddKey(context.Background(), uid, "public-key-double", "fingerprint"); err != nil {
			t.Fatal(err)
		}
	}
	rejectAccount = true
	if w := perform("POST", "/api/environments/"+id+"/join", `{}`, "bob"); w.Code != 502 {
		t.Fatal(w.Code)
	}
	if _, err := s.Store.MemberLogin(context.Background(), id, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("failed native join recorded", err)
	}
	rejectAccount = false
	for _, login := range []string{"alice", "bob"} {
		if w := perform("POST", "/api/environments/"+id+"/join", `{}`, login); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	before := accountCalls
	if w := perform("POST", "/api/environments/"+id+"/join", `{}`, "bob"); w.Code != 200 || accountCalls != before {
		t.Fatal("existing membership reprovisioned")
	}
}
func TestIncompleteEnvironmentStillInspected(t *testing.T) {
	s := apiTestServer(t)
	id := "p0123456789abcdef01234567"
	if err := s.Store.CreateProject(context.Background(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1, Name: "demo", Repository: "alice/demo"}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/inspect" {
			t.Error("mutated incomplete environment")
		}
		return nil, errors.New("missing native container")
	})}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id, "", "alice"))
	var result struct {
		Environment struct {
			Provisioned bool `json:"provisioned"`
		} `json:"environment"`
		NativeUnavailable bool              `json:"native_unavailable"`
		Observed          *host.Environment `json:"observed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || calls != 1 || result.Environment.Provisioned || !result.NativeUnavailable || result.Observed != nil {
		t.Fatal(w.Code, w.Body.String())
	}
}
