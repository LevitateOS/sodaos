package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Journey over the retained JSON handlers/database and an explicit native test double.
// This does not establish Linux accounts, networking or SSH success.
func TestExplicitJoinsAndHonestNativeFailure(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, u := range []store.User{{ID: 1, Login: "alice"}, {ID: 2, Login: "bob"}} {
		if err = db.UpsertUser(ctx, u); err != nil {
			t.Fatal(err)
		}
		if err = db.CreateSession(ctx, u.Login, u.ID, "csrf"); err != nil {
			t.Fatal(err)
		}
	}
	id := "p0123456789abcdef01234567"
	if err = db.CreateProject(ctx, store.Project{ID: id, RepositoryID: 7, OwnerID: 1, Name: "demo", Repository: "alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if err = db.MarkReady(ctx, id, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	server := New(config.Config{ForgejoURL: "https://forgejo.test", OperatorID: 99}, db)
	nativeCalls := 0
	reject := false
	server.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/inspect" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"p0123456789abcdef01234567","ip":"10.89.0.2","running":true}`)), Header: make(http.Header)}, nil
		}
		nativeCalls++
		var account host.Account
		if e := json.NewDecoder(r.Body).Decode(&account); e != nil {
			t.Fatal(e)
		}
		if account.Project != id || account.Identity <= 0 || len(account.Keys) != 1 {
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
		r.Header.Set("X-CSRF-Token", "csrf")
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: login})
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	if w := post("bob"); w.Code != 422 || nativeCalls != 0 {
		t.Fatal("missing-key join reached native setup", w.Code)
	}
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
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "bob"})
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"login":"bob"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
