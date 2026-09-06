package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

func TestProjectRepositoryPicker(t *testing.T) {
	for _, test := range []struct {
		name        string
		body        string
		status      int
		missingKey  bool
		wantOptions bool
		wantError   bool
		wantCalls   int
	}{
		{
			name: "only unprovisioned repositories owned by the session user",
			body: `[
				{"id":10,"full_name":"alice/zebra","owner":{"id":1}},
				{"id":11,"full_name":"alice/Alpha","owner":{"id":1}},
				{"id":7,"full_name":"alice/existing","owner":{"id":1}},
				{"id":8,"full_name":"alice/incomplete","owner":{"id":1}},
				{"id":20,"full_name":"bob/private","owner":{"id":2}},
				{"id":21,"full_name":"team/private","owner":{"id":99}},
				{"id":0,"full_name":"alice/invalid","owner":{"id":1}}
			]`,
			status: 200, wantOptions: true, wantCalls: 2,
		},
		{name: "empty account", body: `[]`, status: 200, wantCalls: 1},
		{name: "all repositories already reserved", body: `[{"id":7,"full_name":"alice/existing","owner":{"id":1}},{"id":8,"full_name":"alice/incomplete","owner":{"id":1}}]`, status: 200, wantCalls: 2},
		{name: "provider unavailable", body: "provider-private-detail", status: 503, wantError: true, wantCalls: 1},
		{name: "invalid provider response", body: "{", status: 200, wantError: true, wantCalls: 1},
		{name: "missing credential", missingKey: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			db, err := store.Open(filepath.Join(dir, "soda.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
				t.Fatal(err)
			}
			if err := db.CreateSession(ctx, "session", 1, "csrf"); err != nil {
				t.Fatal(err)
			}
			for _, project := range []store.Project{
				{ID: "p-existing", RepositoryID: 7, OwnerID: 1, Name: "existing", Repository: "alice/existing"},
				{ID: "p-incomplete", RepositoryID: 8, OwnerID: 1, Name: "incomplete", Repository: "alice/incomplete"},
			} {
				if err := db.CreateProject(ctx, project); err != nil {
					t.Fatal(err)
				}
			}
			if err := db.MarkReady(ctx, "p-existing", "10.89.0.2"); err != nil {
				t.Fatal(err)
			}
			key := filepath.Join(dir, "token")
			if !test.missingKey {
				if err := os.WriteFile(key, []byte("operator-token"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			s := New(config.Config{PublicURL: "https://soda.test", ForgejoURL: "https://forgejo.test", ForgejoInternalURL: "http://forgejo", AdminTokenFile: key, OperatorID: 99}, db)
			calls := 0
			s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/api/v1/users/alice/repos" || r.Header.Get("Authorization") != "token operator-token" {
					t.Fatal("listing must use the session user, not the operator's repository inventory")
				}
				body := test.body
				if r.URL.Query().Get("page") != "1" {
					body = `[]`
				}
				return &http.Response{StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				t.Fatal("listing repositories must not provision or alter a project")
				return nil, nil
			})}
			r := httptest.NewRequest("GET", "/projects", nil)
			r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session"})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			body := w.Body.String()
			if w.Code != 200 || calls != test.wantCalls {
				t.Fatalf("HTTP %d, calls %d, body %s", w.Code, calls, body)
			}
			if !strings.Contains(body, "alice/existing") || !strings.Contains(body, "alice/incomplete") {
				t.Fatal("existing projects disappeared when loading the picker")
			}
			if strings.Contains(body, `<select id="project-repository"`) != test.wantOptions || strings.Contains(body, "Could not load your Forgejo repositories") != test.wantError {
				t.Fatal("wrong picker or error state", body)
			}
			if !test.wantOptions && !test.wantError && !strings.Contains(body, "No repositories available to create an environment") {
				t.Fatal("missing empty-state guidance")
			}
			if test.wantOptions {
				first := strings.Index(body, `<option value="alice/Alpha">`)
				last := strings.Index(body, `<option value="alice/zebra">`)
				if first < 0 || last <= first {
					t.Fatal("eligible options missing or not sorted", body)
				}
			}
			for _, forbidden := range []string{`<option value="alice/existing">`, `<option value="alice/incomplete">`, "bob/private", "team/private", "alice/invalid", "operator-token", "provider-private-detail", `placeholder="alice/repo-name"`} {
				if strings.Contains(body, forbidden) {
					t.Fatal("unexpected picker content", forbidden)
				}
			}
			if !strings.Contains(body, `href="https://forgejo.test/repo/create"`) || !strings.Contains(body, "Refresh repositories") {
				t.Fatal("missing native create/refresh navigation")
			}
		})
	}
}

func TestProjectCreateRejectsForgedPickerChoice(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertUser(t.Context(), store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateSession(t.Context(), "session", 1, "csrf"); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "token")
	if err := os.WriteFile(key, []byte("operator-token"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{PublicURL: "https://soda.test", ForgejoInternalURL: "http://forgejo", AdminTokenFile: key}, db)
	s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v1/repos/bob/private" {
			t.Fatal("submission did not recheck the selected repository")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":20,"full_name":"bob/private","owner":{"id":2}}`))}, nil
	})}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		t.Fatal("forged selection reached host provisioning")
		return nil, nil
	})}
	form := url.Values{"csrf": {"csrf"}, "repository": {"bob/private"}}
	r := httptest.NewRequest("POST", "/projects", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("forged selection accepted", w.Code)
	}
	projects, err := db.Projects(t.Context())
	if err != nil || len(projects) != 0 {
		t.Fatal("forged selection reserved a project", projects, err)
	}
}
