package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

// Authored journey over real handlers/database and an explicit native test double.
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
	server := New(config.Config{PublicURL: "https://soda.test", ForgejoURL: "https://forgejo.test", OperatorID: 99}, db)
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
		form := url.Values{"csrf": {"csrf"}}
		r := httptest.NewRequest("POST", "/projects/"+id+"/join", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "soda_session", Value: login})
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	if w := post("bob"); w.Code != 400 || nativeCalls != 0 {
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
		if w := post(login); w.Code != 303 {
			t.Fatal(login, w.Code, w.Body.String())
		}
	}
	for _, uid := range []int64{1, 2} {
		if _, err = db.MemberLogin(ctx, id, uid); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest("GET", "/projects/"+id, nil)
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "bob"})
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "ssh bob@10.89.0.2") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestPeopleRequiresOperatorOnServer(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = db.UpsertUser(ctx, store.User{ID: 2, Login: "bob"}); err != nil {
		t.Fatal(err)
	}
	if err = db.CreateSession(ctx, "session", 2, "csrf"); err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{PublicURL: "https://soda.test", OperatorID: 1}, db)
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "/people", strings.NewReader("csrf=csrf"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session"})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(method, w.Code)
		}
	}
}
