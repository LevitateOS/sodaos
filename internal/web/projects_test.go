package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProjectCreateUsesProviderOwner(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token acting-alice" || r.URL.Path != "/api/v1/repos/alice/demo" {
			t.Error("wrong actor or route")
		}
		fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1}}`)
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/create" {
			t.Fatal(r.URL.Path)
		}
		var input struct {
			ID    string `json:"id"`
			Owner int64  `json:"owner"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.Owner != 1 {
			t.Fatal("wrong native owner")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":%q,"ip":"10.89.0.2","running":true}`, input.ID))), Header: make(http.Header)}, nil
	})}
	form := url.Values{"csrf": {"csrf-alice"}, "repository": {"alice/demo"}}
	r := httptest.NewRequest("POST", "/projects", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", s.Config.PublicURL)
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-alice"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	projects, err := s.Store.Projects(context.Background())
	if err != nil || len(projects) != 1 || !projects[0].Ready {
		t.Fatalf("%+v %v", projects, err)
	}
}
