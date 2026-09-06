package web

import (
	"context"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestProjectCreateUsesProviderOwner(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = db.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err = db.CreateSession(ctx, "session", 1, "csrf"); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "token")
	if err = os.WriteFile(secret, []byte("operator"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{PublicURL: "https://soda.test", ForgejoInternalURL: "http://forgejo", AdminTokenFile: secret}, db)
	s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1}}`)), Header: make(http.Header)}, nil
	})}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/create" {
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ip":"10.89.0.2","running":true}`)), Header: make(http.Header)}, nil
	})}
	form := url.Values{"csrf": {"csrf"}, "repository": {"alice/demo"}}
	r := httptest.NewRequest("POST", "/projects", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	projects, err := db.Projects(ctx)
	if err != nil || len(projects) != 1 || !projects[0].Ready {
		t.Fatalf("%+v %v", projects, err)
	}
}
