package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreviouslyPendingReactLoginFinishesAtRetainedPage(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			fmt.Fprint(w, `{"access_token":"pending-access","refresh_token":"pending-refresh","token_type":"bearer","expires_in":3600}`)
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":1,"login":"alice"}`)
		case "/login/oauth/introspect":
			fmt.Fprint(w, `{"active":true,"scope":"read:user","sub":"1","aud":["client"]}`)
		default:
			t.Error("unexpected provider request")
			w.WriteHeader(500)
		}
	})
	// A state record issued by the old binary, not a new allowed return target.
	if err := s.Store.BeginOAuth(t.Context(), "pending-react", "verifier", "/app/"); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/oauth/callback?state=pending-react&code=test-code", nil)
	r.AddCookie(&http.Cookie{Name: "soda_oauth", Value: "pending-react"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != "/projects" {
		t.Fatal(w.Code, w.Body.String())
	}
}
