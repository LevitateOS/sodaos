package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/store"
)

func TestRepositoryPickerAuthorityAndReservations(t *testing.T) {
	for _, kind := range []string{"new", "ready", "incomplete", "transfer", "hidden", "foreign-search", "wrong-actor", "unavailable"} {
		t.Run(kind, func(t *testing.T) {
			s := apiTestServer(t)
			if kind == "ready" || kind == "incomplete" {
				if err := s.Store.CreateProject(t.Context(), store.Project{ID: "p0123456789abcdef01234567", RepositoryID: 7, OwnerID: 1, Name: "repo", Repository: "alice/repo"}); err != nil {
					t.Fatal(err)
				}
				if kind == "ready" {
					if err := s.Store.MarkReady(t.Context(), "p0123456789abcdef01234567", "10.89.0.2"); err != nil {
						t.Fatal(err)
					}
				}
			}
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				if in.Operation != extensions.OperationOwnedRepositories {
					t.Error("unexpected native callback", in.Operation)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
				if kind == "unavailable" {
					return extensions.CallbackResponse{ErrorCode: "unavailable"}
				}
				items := []extensions.Repository{}
				if kind != "transfer" && kind != "hidden" {
					owner := "alice"
					if kind == "foreign-search" {
						owner = "other"
					}
					items = append(items, extensions.Repository{ID: "7", Owner: owner, Name: "renamed", Permission: "write"})
				}
				return extensions.CallbackResponse{Page: &extensions.RepositoryPage{Items: items}}
			}
			w := httptest.NewRecorder()
			r := apiTestRequest("GET", "/api/repositories?q=", "", "alice")
			var headers func(http.Header)
			if kind == "wrong-actor" {
				headers = func(h http.Header) {
					h.Set(extensions.ContextHeader, strings.Replace(h.Get(extensions.ContextHeader), `"id":"1"`, `"id":"2"`, 1))
				}
			}
			nativeAPIServeWithOptions(t, s, w, r, callback, headers)
			body := w.Body.String()
			switch kind {
			case "foreign-search", "wrong-actor", "unavailable":
				if w.Code == 200 || strings.Contains(body, "renamed") {
					t.Fatal(w.Code, body)
				}
			case "transfer", "hidden":
				if w.Code != 200 || !strings.Contains(body, `"items":[]`) {
					t.Fatal(w.Code, body)
				}
			default:
				if w.Code != 200 || !strings.Contains(body, `"name":"renamed"`) {
					t.Fatal(w.Code, body)
				}
				if kind == "new" && !strings.Contains(body, `"can_create":true,"project":null`) {
					t.Fatal(body)
				}
				if kind != "new" && !strings.Contains(body, `"can_create":false,"project":{"id":"p0123456789abcdef01234567","provisioned":`+fmt.Sprint(kind == "ready")) {
					t.Fatal(body)
				}
			}
		})
	}
}

func TestRepositoryPickerNativeGenerationChangeWinsPublication(t *testing.T) {
	s := apiTestServer(t)
	r := apiTestRequest("GET", "/api/repositories?q=", "", "alice")
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation != extensions.OperationOwnedRepositories {
			t.Error("unexpected callback", in.Operation)
		}
		r.Header.Set(extensions.ContextHeader, strings.Replace(r.Header.Get(extensions.ContextHeader), nativeProductGeneration, "expired", 1))
		return extensions.CallbackResponse{Page: &extensions.RepositoryPage{Items: []extensions.Repository{}}}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, r, callback)
	if w.Code != 401 || strings.Contains(w.Body.String(), `"items"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestRepositoryPickerNativePageBoundary(t *testing.T) {
	s := apiTestServer(t)
	oversized := false
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation != extensions.OperationOwnedRepositories {
			t.Error("unexpected callback", in.Operation)
		}
		count := in.Limit
		if oversized {
			count++
		}
		items := make([]extensions.Repository, count)
		for i := range items {
			items[i] = extensions.Repository{ID: fmt.Sprint(i + 1), Owner: "alice", Name: "repo", Permission: "write"}
		}
		return extensions.CallbackResponse{Page: &extensions.RepositoryPage{Items: items, NextCursor: "next"}}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories?q=&cursor=first", "", "alice"), callback)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"next_cursor":"next"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	oversized = true
	w = httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories?q=&cursor=first", "", "alice"), callback)
	if w.Code != 503 || strings.Contains(w.Body.String(), `"items"`) {
		t.Fatal("invalid native page was published", w.Code, w.Body.String())
	}
}

func TestRepositoryPickerQueryAndAdmission(t *testing.T) {
	s := apiTestServer(t)
	calls := 0
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		calls++
		return extensions.CallbackResponse{Page: &extensions.RepositoryPage{Items: []extensions.Repository{}}}
	}
	for _, query := range []string{"", "?page=1", "?q=&page=1", "?q=&cursor=a&cursor=b", "?q=&uid=2", "?q=%0a", "?q=" + strings.Repeat("x", 201), "?q=&cursor=" + strings.Repeat("x", 4097)} {
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories"+query, "", "alice"), callback)
		if w.Code != 400 || calls != 0 {
			t.Fatal(query, w.Code, w.Body.String(), calls)
		}
	}
	for range cap(s.API.RepositorySlots) {
		s.API.RepositorySlots <- struct{}{}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories?q=", "", "alice"), callback)
	if w.Code != 503 || calls != 0 {
		t.Fatal(w.Code, calls)
	}
	for range cap(s.API.RepositorySlots) {
		<-s.API.RepositorySlots
	}
	w = httptest.NewRecorder()
	nativeAPIServeWithOptions(t, s, w, apiTestRequest("GET", "/api/repositories?q=", "", "alice"), callback, func(h http.Header) {
		h.Set(extensions.ContextHeader, strings.Replace(h.Get(extensions.ContextHeader), `"id":"1"`, `"id":"2"`, 1))
	})
	if w.Code != 403 || calls != 0 {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
}
