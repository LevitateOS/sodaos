package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNativeOAuthFailuresStayInFixedHostWithoutNewSession(t *testing.T) {
	for _, mode := range []string{"declined", "missing scopes", "actor changed", "provider unavailable"} {
		t.Run(mode, func(t *testing.T) {
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/login/oauth/access_token":
					if mode == "provider unavailable" {
						w.WriteHeader(503)
						return
					}
					fmt.Fprint(w, `{"access_token":"fixture-access","refresh_token":"fixture-refresh","token_type":"bearer","expires_in":3600}`)
				case "/api/v1/user":
					if mode == "actor changed" {
						fmt.Fprint(w, `{"id":2,"login":"bob"}`)
					} else {
						fmt.Fprint(w, `{"id":1,"login":"alice"}`)
					}
				case "/login/oauth/introspect":
					fmt.Fprint(w, `{"active":true,"scope":"read:user read:repository","sub":"1","aud":["client"]}`)
				default:
					t.Error("unexpected provider request")
					w.WriteHeader(500)
				}
			})
			start := httptest.NewRecorder()
			s.ServeHTTP(start, apiTestRequest("GET", "/login?destination=runners&expected_user_id=1", "", "alice"))
			location, err := url.Parse(start.Header().Get("Location"))
			if err != nil || start.Code != 302 {
				t.Fatal("start failed")
			}
			query := url.Values{"state": {location.Query().Get("state")}}
			if mode != "declined" {
				query.Set("code", "fixture-code")
			} else {
				query.Set("error", "access_denied")
			}
			request := apiTestRequest("GET", "/oauth/callback?"+query.Encode(), "", "alice")
			for _, cookie := range start.Result().Cookies() {
				request.AddCookie(cookie)
			}
			response := httptest.NewRecorder()
			s.ServeHTTP(response, request)
			if response.Code != 303 || response.Header().Get("Location") != s.Config.ForgejoURL+"/?soda-view=runners&soda-connect=failed" || len(response.Result().Cookies()) != 0 {
				t.Fatal("unsafe failure destination or session", response.Code)
			}
			if _, err := s.Store.Session(t.Context(), "session-alice"); err != nil {
				t.Fatal("existing session discarded", err)
			}
		})
	}
}

func TestInvalidBrowserCallbackUsesNativeErrorEntryWithoutClaimingAuthentication(t *testing.T) {
	s := apiTestServer(t)
	r := apiTestRequest("GET", "/oauth/callback?state=unknown&code=untrusted", "", "")
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != s.Config.ForgejoURL+"/user/login?redirect_to=%2F%3Fsoda-view%3Dspaces%26soda-connect%3Dfailed" || len(w.Result().Cookies()) != 0 {
		t.Fatal("invalid callback did not return safely", w.Code)
	}
}
