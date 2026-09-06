package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestOAuthCallbackStoresActualConsentAndRotatesSession(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/login/oauth/access_token":
			fmt.Fprint(w, `{"access_token":"test-access","refresh_token":"test-refresh","token_type":"bearer","expires_in":3600}`)
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":1,"login":"alice"}`)
		case "/login/oauth/introspect":
			fmt.Fprint(w, `{"active":true,"scope":"read:user","sub":"1","aud":["client"]}`)
		default:
			t.Error("unexpected OAuth request")
			w.WriteHeader(404)
		}
	})
	login := httptest.NewRecorder()
	s.ServeHTTP(login, httptest.NewRequest("GET", "/login?return_to=%2Fapp%2F&administration=1", nil))
	location, err := url.Parse(login.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Query().Get("scope") != "write:user write:repository write:issue write:organization write:notification write:admin" {
		t.Fatal("incorrect requested consent")
	}
	state := location.Query().Get("state")
	callback := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/oauth/callback?"+url.Values{"state": {state}, "code": {"test-code"}, "return_to": {"https://untrusted.invalid"}}.Encode(), nil)
		r.AddCookie(&http.Cookie{Name: "soda_oauth", Value: state})
		r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-alice"})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	result := callback()
	if result.Code != 303 || result.Header().Get("Location") != "/app/" {
		t.Fatal("callback destination invalid", result.Code)
	}
	var session *http.Cookie
	for _, cookie := range result.Result().Cookies() {
		if cookie.Name == "soda_session" {
			session = cookie
		}
	}
	if session == nil || !session.HttpOnly || !session.Secure {
		t.Fatal("session cookie missing protections")
	}
	grant, err := s.Store.Grant(context.Background(), session.Value, 1)
	if err != nil || grant.Scopes != "read:user" {
		t.Fatal("requested scopes substituted for native consent", err)
	}
	if _, err = s.Store.Session(context.Background(), "session-alice"); err == nil {
		t.Fatal("old session not rotated")
	}
	if replay := callback(); replay.Code != 400 || calls != 3 {
		t.Fatal("replayed OAuth state reached provider")
	}
}
