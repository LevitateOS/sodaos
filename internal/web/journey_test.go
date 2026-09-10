package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Journey over the retained JSON handlers/database and an explicit native test double.
// This does not establish Linux accounts, networking or SSH success.
func TestExplicitJoinsAndHonestNativeFailure(t *testing.T) {
	ctx := context.Background()
	server := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			login := strings.TrimPrefix(r.Header.Get("Authorization"), "token acting-")
			fmt.Fprintf(w, `{"id":%d,"login":%q}`, map[string]int{"alice": 1, "bob": 2}[login], login)
		case "/api/v1/repositories/7":
			fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`)
		case "/api/v1/users/bob/orgs/alice/permissions":
			fmt.Fprint(w, `{"is_owner":false}`)
		default:
			t.Error("unexpected provider path", r.URL.Path)
			w.WriteHeader(500)
		}
	})
	db := server.Store
	var err error
	id := "p0123456789abcdef01234567"
	if err = db.CreateProject(ctx, store.Project{ID: id, RepositoryID: 7, OwnerID: 1, Name: "demo", Repository: "alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if err = db.MarkReady(ctx, id, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	server.Config.OperatorID = 99
	nativeCalls := 0
	reject := true
	expectedKeys := 0
	server.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/inspect" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"p0123456789abcdef01234567","ip":"10.89.0.2","running":true}`)), Header: make(http.Header)}, nil
		}
		nativeCalls++
		var account host.Account
		if e := json.NewDecoder(r.Body).Decode(&account); e != nil {
			t.Fatal(e)
		}
		if account.Project != id || account.Identity <= 0 || len(account.Keys) != expectedKeys {
			t.Fatal(account)
		}
		code := 200
		if reject {
			code = 500
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	post := func(login string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/-/soda/api/environments/"+id+"/join", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", server.Config.ForgejoURL)
		r.Header.Set("X-CSRF-Token", "csrf-"+login)
		r.Header.Set(expectedUserHeader, map[string]string{"alice": "1", "bob": "2"}[login])
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-" + login})
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	if w := post("bob"); w.Code != 502 || nativeCalls != 1 {
		t.Fatal("account-only join must reach native provisioning and report its failure", w.Code)
	}
	if _, err = db.MemberLogin(ctx, id, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("failed account-only join recorded membership", err)
	}
	expectedKeys = 1
	for _, uid := range []int64{1, 2} {
		if err = db.AddKey(ctx, uid, "canonical-public-key-test-double", "fingerprint"); err != nil {
			t.Fatal(err)
		}
	}
	reject = true
	if w := post("bob"); w.Code != 502 {
		t.Fatal(w.Code)
	}
	if _, err = db.MemberLogin(ctx, id, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("failed native join recorded membership", err)
	}
	reject = false
	for _, login := range []string{"alice", "bob"} {
		if w := post(login); w.Code != 200 {
			t.Fatal(login, w.Code, w.Body.String())
		}
	}
	for _, uid := range []int64{1, 2} {
		if _, err = db.MemberLogin(ctx, id, uid); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest("GET", "/-/soda/api/environments/"+id, nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-bob"})
	r.Header.Set(expectedUserHeader, "2")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"login":"bob"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
