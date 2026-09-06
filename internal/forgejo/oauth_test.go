package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGrantExchangeAndRefreshUseNativeForms(t *testing.T) {
	for _, refresh := range []bool{false, true} {
		t.Run(fmt.Sprint(refresh), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/login/oauth/access_token" || r.Method != "POST" {
					t.Error("wrong token endpoint")
				}
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				if r.Form.Get("client_id") != "client" || r.Form.Get("client_secret") != "secret" {
					t.Error("missing confidential client input")
				}
				if refresh {
					if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh" || r.Form.Get("code") != "" {
						t.Error("wrong refresh form")
					}
				} else if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code_verifier") != "verifier" || r.Form.Get("redirect_uri") != "https://soda.example/oauth/callback" {
					t.Error("PKCE/redirect binding lost")
				}
				fmt.Fprint(w, `{"access_token":"access","refresh_token":"rotated","token_type":"bearer","expires_in":3600}`)
			}))
			defer server.Close()
			client := New(server.URL)
			var result TokenResponse
			var err error
			if refresh {
				result, err = client.RefreshGrant(context.Background(), "client", "secret", "refresh")
			} else {
				result, err = client.ExchangeGrant(context.Background(), "client", "secret", "code", "https://soda.example/oauth/callback", "verifier")
			}
			if err != nil || result.Refresh != "rotated" || result.ExpiresIn != 3600 {
				t.Fatal("invalid result", err)
			}
		})
	}
}
func TestIntrospectionChecksActualScopeSubjectAndAudience(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		accept     bool
	}{
		{"actual older consent", `{"active":true,"scope":"read:user","sub":"12","aud":["client"]}`, true},
		{"wrong user", `{"active":true,"scope":"all","sub":"13","aud":["client"]}`, false},
		{"wrong client", `{"active":true,"scope":"all","sub":"12","aud":["other"]}`, false},
		{"inactive", `{"active":false}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, secret, ok := r.BasicAuth()
				if !ok || id != "client" || secret != "secret" || r.URL.Path != "/login/oauth/introspect" {
					t.Error("wrong introspection authentication")
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			scope, err := New(server.URL).GrantScopes(context.Background(), "client", "secret", "access", 12)
			if (err == nil) != tc.accept {
				t.Fatal("incorrect introspection acceptance", err)
			}
			if tc.accept && (scope != "read:user" || HasScope(scope, "write:repository")) {
				t.Fatal("requested scope substituted for consent")
			}
		})
	}
}
