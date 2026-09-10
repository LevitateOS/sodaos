package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/runners"
	"github.com/stretchr/testify/require"
)

// Operation-specific tests reuse the existing handler/session fixtures. Page,
// OAuth entry and coordinated-logout fixture ports belong to Soda-pages.
var runnerAPIRequests = []struct{ name, method, path, body string }{
	{"list", "GET", "/api/settings/runners", ""},
	{"create", "POST", "/api/settings/runners", `{"id":"one","provider":"forgejo","registration_url":"https://untrusted.invalid","registration_id":"33834eef-e758-48c4-a676-1745426747aa","labels":"soda:host","registration_token":"synthetic-runner-secret"}`},
	{"start", "POST", "/api/settings/runners/one/start", `{"confirm_id":"one"}`},
	{"stop", "POST", "/api/settings/runners/one/stop", `{"confirm_id":"one"}`},
	{"restart", "POST", "/api/settings/runners/one/restart", `{"confirm_id":"one"}`},
	{"remove", "POST", "/api/settings/runners/one/remove", `{"confirm_id":"one"}`},
}

type runnerUnreadBody struct{ reads int }

func (b *runnerUnreadBody) Read([]byte) (int, error) {
	b.reads++
	return 0, fmt.Errorf("runner body must not be decoded before admission")
}
func (*runnerUnreadBody) Close() error { return nil }

func TestEveryRunnerAPIRejectsInvalidAuthorityBeforeDecodeOrNative(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		for _, denial := range []string{"nonoperator", "missing cookie", "duplicate cookie", "wrong actor", "missing actor", "missing scope", "missing grant", "expired grant", "missing session", "operator unset", "changed subject", "provider denied", "provider unavailable", "logout during authority"} {
			t.Run(operation.name+"/"+denial, func(t *testing.T) {
				nativeCalls := 0
				s := runnerWebFixture(t, func(http.ResponseWriter, *http.Request) { nativeCalls++ })
				login := "alice"
				if denial == "nonoperator" {
					login = "bob"
				}
				request := apiTestRequest(operation.method, operation.path, operation.body, login)
				unread := &runnerUnreadBody{}
				request.Body = unread
				status := 401
				switch denial {
				case "nonoperator":
					status = 403
				case "missing cookie":
					request.Header.Del("Cookie")
				case "duplicate cookie":
					request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-bob"})
				case "wrong actor":
					request.Header.Set(expectedUserHeader, "2")
					status = 403
				case "missing actor":
					request.Header.Del(expectedUserHeader)
					status = 400
				case "missing scope", "expired grant":
					grant, err := s.Store.Grant(t.Context(), "session-alice", 1)
					require.NoError(t, err)
					if denial == "missing scope" {
						grant.Scopes = "read:repository"
						status = 403
					} else {
						grant.Expires = time.Now().Add(-time.Minute).Unix()
					}
					require.NoError(t, s.Store.ReplaceGrant(t.Context(), "session-alice", 1, grant))
				case "missing grant":
					require.NoError(t, s.Store.DeleteSession(t.Context(), "session-alice"))
					require.NoError(t, s.Store.CreateSession(t.Context(), "session-alice", 1, "csrf-alice"))
				case "missing session":
					require.NoError(t, s.Store.DeleteSession(t.Context(), "session-alice"))
				case "operator unset":
					s.Config.OperatorID = 0
					status = 403
				}
				if denial == "changed subject" || denial == "provider denied" || denial == "provider unavailable" || denial == "logout during authority" || denial == "expired grant" {
					s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
						w := httptest.NewRecorder()
						switch denial {
						case "expired grant":
							require.Equal(t, "/login/oauth/access_token", r.URL.Path)
							w.WriteHeader(401)
						case "changed subject":
							fmt.Fprint(w, `{"id":2,"login":"bob","is_admin":true}`)
						case "provider denied":
							w.WriteHeader(403)
						case "provider unavailable":
							w.WriteHeader(503)
						case "logout during authority":
							v, err := s.Store.Session(t.Context(), "session-alice")
							require.NoError(t, err)
							require.NoError(t, s.Store.EndLoginContext(t.Context(), v.ContextID))
							fmt.Fprint(w, `{"id":1,"login":"alice","is_admin":false}`)
						}
						return w.Result(), nil
					})}
					if denial == "provider denied" {
						status = 403
					}
					if denial == "provider unavailable" {
						status = 503
					}
				}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, request)
				require.Equal(t, status, w.Code, w.Body.String())
				require.Zero(t, unread.reads)
				require.Zero(t, nativeCalls)
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
				require.NotContains(t, w.Body.String(), "synthetic-runner-secret")
			})
		}
	}
}

func TestRunnerMutationRequestBoundaries(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		if operation.method != "POST" {
			continue
		}
		for _, invalid := range []string{"method", "query", "empty query", "csrf", "origin", "content type", "empty", "null", "array", "duplicate", "trailing", "unit", "account", "path", "command", "oversized"} {
			t.Run(operation.name+"/"+invalid, func(t *testing.T) {
				calls := 0
				s := runnerWebFixture(t, func(http.ResponseWriter, *http.Request) { calls++ })
				body := operation.body
				path := operation.path
				method := operation.method
				status := 400
				switch invalid {
				case "method":
					method = "DELETE"
					status = 405
				case "query":
					path += "?unit=sshd"
				case "empty query":
					path += "?"
				case "csrf", "origin":
					status = 403
				case "content type":
					status = 415
				case "empty":
					body = ""
				case "null":
					body = "null"
				case "array":
					body = "[]"
				case "duplicate":
					if operation.name == "create" {
						body = strings.Replace(body, `"id":`, `"id":"two","id":`, 1)
					} else {
						body = `{"confirm_id":"one","confirm_id":"one"}`
					}
				case "trailing":
					body += " {}"
				case "unit", "account", "path", "command":
					body = strings.TrimSuffix(body, "}") + fmt.Sprintf(",%q:%q}", invalid, "synthetic-runner-secret")
				case "oversized":
					body = `{"unexpected":"` + strings.Repeat("x", apiBodyLimit) + `"}`
					status = 413
				}
				request := apiTestRequest(method, path, body, "alice")
				switch invalid {
				case "csrf":
					request.Header.Set("X-CSRF-Token", "wrong")
				case "origin":
					request.Header.Set("Origin", "https://untrusted.invalid")
				case "content type":
					request.Header.Set("Content-Type", "text/plain")
				}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, request)
				require.Equal(t, status, w.Code, w.Body.String())
				require.Zero(t, calls)
				require.NotContains(t, w.Body.String(), "synthetic-runner-secret")
			})
		}
	}
}

func TestRunnerOperationsDispatchOnceWithFixedTargetsAndSanitizedFailures(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/failure-%t", operation.name, fail), func(t *testing.T) {
				calls := 0
				var s *Server
				s = runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					require.Equal(t, "/runners/"+operation.name, r.URL.Path)
					require.Equal(t, "POST", r.Method)
					if operation.name == "create" {
						var input runners.CreateRequest
						require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
						require.Equal(t, s.Config.ForgejoInternalURL, input.RegistrationURL)
						require.Equal(t, "synthetic-runner-secret", input.RegistrationToken)
						require.Equal(t, "one", input.ID)
						require.Equal(t, runners.ProviderForgejo, input.Provider)
					} else {
						var input map[string]string
						require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
						expected := map[string]string{}
						if operation.name != "list" {
							expected["id"] = "one"
						}
						require.Equal(t, expected, input)
					}
					if fail {
						w.WriteHeader(500)
						fmt.Fprint(w, "synthetic-runner-secret")
						return
					}
					if operation.name == "list" {
						fmt.Fprint(w, "[]")
					} else {
						fmt.Fprint(w, `{"ok":true}`)
					}
				})
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest(operation.method, operation.path, operation.body, "alice"))
				status := 200
				if fail {
					status = 502
					if operation.name == "list" {
						status = 503
					}
				}
				require.Equal(t, status, w.Code, w.Body.String())
				require.Equal(t, 1, calls)
				require.NotContains(t, w.Body.String(), "synthetic-runner-secret")
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
				if fail && operation.name != "list" {
					require.Contains(t, w.Body.String(), "runner_unconfirmed")
				}
				if fail && operation.name == "list" {
					require.NotContains(t, w.Body.String(), `"runners":[]`)
				}
			})
		}
	}
}
