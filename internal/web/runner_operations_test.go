package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/runners"
	"github.com/levitateos/sodaos/internal/web/auth"
	"github.com/stretchr/testify/require"
)

// Operation-specific tests reuse the existing native extension fixtures.
var runnerAPIRequests = []struct{ name, method, path, body string }{
	{"list", "GET", "/api/settings/runners", ""},
	{"create", "POST", "/api/settings/runners", `{"id":"one","provider":"forgejo","registration_url":"https://untrusted.invalid","registration_id":"33834eef-e758-48c4-a676-1745426747aa","labels":"soda:host","registration_token":"synthetic-runner-secret"}`},
	{"start", "POST", "/api/settings/runners/one/start", `{}`},
	{"stop", "POST", "/api/settings/runners/one/stop", `{}`},
	{"restart", "POST", "/api/settings/runners/one/restart", `{}`},
	{"remove", "POST", "/api/settings/runners/one/remove", `{"confirm_id":"one"}`},
}

func TestRunnerMutationRequestBoundaries(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		// Create, start and restart have a separate unavailable gate and never
		// reach native execution. Stop and remove remain active mutations.
		if operation.name != "stop" && operation.name != "remove" {
			continue
		}
		for _, invalid := range []string{"method", "query", "empty query", "generation", "origin", "empty", "null", "array", "duplicate", "trailing", "unit", "account", "path", "command", "oversized"} {
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
					status = 404
				case "query":
					path += "?unit=sshd"
				case "empty query":
					path += "?"
					status = 404
				case "generation":
					status = 409
				case "origin":
					status = 403
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
					body = `{"unexpected":"` + strings.Repeat("x", auth.APIBodyLimit) + `"}`
					status = 413
				}
				request := apiTestRequest(method, path, body, "alice")
				switch invalid {
				case "origin":
					request.Header.Set("Origin", "https://untrusted.invalid")
				}
				w := httptest.NewRecorder()
				var headers func(http.Header)
				if invalid == "generation" {
					headers = func(h http.Header) { h.Set("X-Extension-Session-Generation", "changed") }
				}
				nativeAPIServeWithOptions(t, s, w, request, nil, headers)
				require.Equal(t, status, w.Code, w.Body.String())
				require.Zero(t, calls)
				require.NotContains(t, w.Body.String(), "synthetic-runner-secret")
			})
		}
	}
}

func TestRunnerOperationsDispatchOnceWithFixedTargetsAndSanitizedFailures(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		if operation.name == "create" || operation.name == "start" || operation.name == "restart" {
			continue
		}
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
						_, _ = fmt.Fprint(w, "synthetic-runner-secret")
						return
					}
					if operation.name == "list" {
						_, _ = fmt.Fprint(w, `{"runners":[],"unavailable":[]}`)
					} else {
						_, _ = fmt.Fprint(w, `{"ok":true}`)
					}
				})
				w := httptest.NewRecorder()
				nativeAPIServe(t, s, w, apiTestRequest(operation.method, operation.path, operation.body, "alice"))
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
