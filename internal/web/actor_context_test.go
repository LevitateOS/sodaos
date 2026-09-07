package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
)

func TestExpectedActorGuardsReadsAndEveryMutationMethod(t *testing.T) {
	s := apiTestServer(t)
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		for _, tc := range []struct {
			name   string
			values []string
			status int
			code   string
		}{
			{"missing", nil, 400, "invalid_actor_context"},
			{"empty", []string{""}, 400, "invalid_actor_context"},
			{"zero", []string{"0"}, 400, "invalid_actor_context"},
			{"negative", []string{"-1"}, 400, "invalid_actor_context"},
			{"padded", []string{"01"}, 400, "invalid_actor_context"},
			{"signed", []string{"+1"}, 400, "invalid_actor_context"},
			{"spaces", []string{" 1 "}, 400, "invalid_actor_context"},
			{"overflow", []string{"9223372036854775808"}, 400, "invalid_actor_context"},
			{"duplicate", []string{"1", "1"}, 400, "invalid_actor_context"},
			{"combined", []string{"1,1"}, 400, "invalid_actor_context"},
			{"other user", []string{"2"}, 403, "identity_mismatch"},
			{"maximum other ID", []string{"9223372036854775807"}, 403, "identity_mismatch"},
			{"matching", []string{"1"}, 204, ""},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				called := false
				handler := s.apiProtected(func(w http.ResponseWriter, r *http.Request, _ store.Session) {
					called = true
					w.WriteHeader(204)
				}, method)
				r := apiTestRequest(method, "/api/actor-probe?expected_user_id=1", "{}", "alice")
				r.Header.Del(expectedUserHeader)
				for _, value := range tc.values {
					r.Header.Add(expectedUserHeader, value)
				}
				w := httptest.NewRecorder()
				handler(w, r)
				if w.Code != tc.status || called != (tc.status == 204) || (tc.code != "" && !strings.Contains(w.Body.String(), `"code":"`+tc.code+`"`)) {
					t.Fatal(w.Code, called, w.Body.String())
				}
			})
		}
	}
}

func TestBootstrapDoesNotAuthorizeAStalePage(t *testing.T) {
	s := apiTestServer(t)
	r := apiTestRequest("GET", "/api/session", "", "alice")
	r.Header.Del(expectedUserHeader)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"1"`) {
		t.Fatal(w.Code, w.Body.String())
	}

	// Another tab replaced the browser's Soda cookie. The old page's actor and
	// CSRF cannot silently mutate the new user's preferences.
	r = apiTestRequest("PATCH", "/api/me/preferences", `{"display_name":"wrong actor"}`, "alice")
	r.Header.Del("Cookie")
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-bob"})
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 || !strings.Contains(w.Body.String(), `"code":"identity_mismatch"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, id := range []int64{1, 2} {
		u, err := s.Store.User(t.Context(), id)
		if err != nil || u.Name == "wrong actor" {
			t.Fatal(u, err)
		}
	}
	// The optional bootstrap guard also rejects mismatches rather than exposing
	// the new actor/CSRF as if they matched the calling page.
	r = apiTestRequest("GET", "/api/session", "", "bob")
	r.Header.Set(expectedUserHeader, "1")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 || strings.Contains(w.Body.String(), "csrf-bob") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestActorHintIsNeitherAuthenticationNorProviderIdentity(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" || r.Header.Get("Authorization") != "token acting-alice" {
			t.Error("wrong acting request")
		}
		fmt.Fprint(w, `{"id":2,"login":"bob"}`)
	})
	r := apiTestRequest("GET", "/api/forgejo/me", "", "")
	r.Header.Set(expectedUserHeader, "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 401 || !strings.Contains(w.Body.String(), `"code":"unauthenticated"`) {
		t.Fatal(w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/me", "", "alice"))
	if w.Code != 401 || !strings.Contains(w.Body.String(), `"code":"provider_identity_mismatch"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
