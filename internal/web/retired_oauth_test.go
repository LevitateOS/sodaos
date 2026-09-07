package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuthCallbackIgnoresCallerDestination(t *testing.T) {
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
	if err := s.Store.BeginOAuth(t.Context(), "pending", "verifier"); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/-/soda/oauth/callback?state=pending&code=test-code&return_to=https://evil.example/", nil)
	r.AddCookie(&http.Cookie{Name: oauthCookie, Value: "pending"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != s.Config.ForgejoURL+"/" {
		t.Fatal(w.Code, w.Body.String())
	}
}
