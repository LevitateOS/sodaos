package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

// Real routed handlers and store; transport doubles never execute native commands.
func TestMutationAdmissionAfterProviderIO(t *testing.T) {
	for _, operation := range []string{"join", "start", "stop", "apply"} {
		for _, change := range []string{"unchanged", "logout", "user", "context", "csrf", "store", "cancel", "provider-denied", "provider-unavailable"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				s, _ := managementWebFixture(t)
				project := webTerminalProject
				path, body := "/lifecycle", `{"action":"`+operation+`"}`
				if operation == "stop" {
					body = `{"action":"stop","confirm_stop":true}`
				}
				if operation == "join" {
					// A fresh project has no membership to short-circuit provisioning.
					project = "pabcdef0123456789abcdef01"
					if err := s.Store.CreateProject(t.Context(), store.Project{ID: project, RepositoryID: 8, OwnerID: 1}); err != nil {
						t.Fatal(err)
					}
					if err := s.Store.MarkReady(t.Context(), project, "10.89.0.3"); err != nil {
						t.Fatal(err)
					}
					path, body = "/join", `{"ssh_keys":"none"}`
				} else if operation == "apply" {
					path, body = "/access-keys", `{"revision":"`+strings.Repeat("a", 64)+`","saved_fingerprints":[],"confirm_empty":true}`
				}
				r := apiTestRequest("POST", "/api/environments/"+project+path, body, "alice")
				ctx, cancel := context.WithCancel(r.Context())
				defer cancel()
				r = r.WithContext(ctx)

				// Another actor's terminals in the target project must survive a
				// denied Stop, even when real Alice logout correctly ends Alice access.
				bob, err := s.Store.Session(t.Context(), "session-bob")
				if err != nil {
					t.Fatal(err)
				}
				terminalCtx, endTerminal := context.WithCancel(t.Context())
				defer endTerminal()
				peerCtx, endPeer := context.WithCancel(t.Context())
				defer endPeer()
				s.terminals = map[string]*browserTerminal{"retained": {terminalBinding: terminalBinding{session: bob, project: store.Project{ID: project}}, cancel: endTerminal}}
				s.terminalPeers = map[*http.Request]*terminalPeer{r: {contextID: bob.ContextID, project: project, cancel: endPeer}}
				nativeCalls := 0
				s.Host.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
					nativeCalls++
					var in map[string]any
					if err := json.NewDecoder(req.Body).Decode(&in); err != nil || in["project"] != project {
						t.Error("untrusted native target", err)
					}
					w := httptest.NewRecorder()
					switch operation {
					case "join":
						if req.URL.Path != "/account" || in["identity"] != float64(1) || in["login"] != "current-login" {
							t.Error("Join lost original actor/provider login")
						}
						_, _ = w.WriteString(`{"ok":true}`)
					case "apply":
						if req.URL.Path != "/access-keys" || in["identity"] != float64(1) || in["login"] != "original-alice" || in["apply"] != true {
							t.Error("Apply remapped original membership")
						}
						_, _ = w.WriteString(`{"revision":"` + strings.Repeat("a", 64) + `","keys":[]}`)
					default:
						if req.URL.Path != "/lifecycle" || in["action"] != operation {
							t.Error("lifecycle action changed")
						}
						if operation == "stop" && (!s.terminalStopping[project] || terminalCtx.Err() == nil || peerCtx.Err() == nil) {
							t.Error("admitted Stop lost terminal coordination")
						}
						_ = json.NewEncoder(w).Encode(host.LifecycleState{Environment: host.Environment{ID: project, Running: operation == "start"}, BootEnabled: operation == "start"})
					}
					return w.Result(), nil
				})}
				providerCalls := 0
				s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
					providerCalls++
					w := httptest.NewRecorder()
					if req.URL.Path == "/api/v1/user" {
						_, _ = w.WriteString(`{"id":1,"login":"current-login"}`)
						return w.Result(), nil
					}
					if operation == "start" || operation == "stop" {
						if req.URL.Path == "/api/v1/repositories/7" {
							// Lifecycle must finish the later organization-owner call
							// too, not admit from a merely visible repository.
							_, _ = w.WriteString(`{"id":7,"name":"repo","full_name":"team/repo","owner":{"id":99,"login":"team"}}`)
							return w.Result(), nil
						}
						if req.URL.Path != "/api/v1/users/current-login/orgs/team/permissions" {
							t.Error("unexpected organization authority request", req.URL.Path)
						}
					} else if req.URL.Path != "/api/v1/repositories/7" && req.URL.Path != "/api/v1/repositories/8" {
						t.Error("unexpected provider request", req.URL.Path)
					}
					// Complete the state change while the actual provider call is
					// suspended, then deliberately return its stale successful result.
					switch change {
					case "logout":
						logout := httptest.NewRecorder()
						s.ServeHTTP(logout, apiTestRequest("POST", "/api/session/logout", `{}`, "alice"))
						if logout.Code != 204 {
							t.Fatal("logout failed", logout.Code)
						}
					case "user", "csrf", "context":
						if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
							t.Fatal(err)
						}
						uid, csrf := int64(1), "csrf-alice"
						if change == "user" {
							uid = 2
						} else if change == "csrf" {
							csrf = "changed-csrf"
						}
						if change == "context" {
							// Rebind the same token/actor/CSRF to a distinct real login
							// context so only the context comparison can detect drift.
							if err := s.Store.BeginOAuth(t.Context(), "new-context", store.OAuthLogin{}, "", ""); err != nil {
								t.Fatal(err)
							}
							a, err := s.Store.ConsumeOAuth(t.Context(), "new-context", "")
							if err != nil {
								t.Fatal(err)
							}
							err = s.Store.FinishOAuth(t.Context(), a, store.User{ID: 1, Login: "alice"}, "session-alice", csrf, store.Grant{Access: "synthetic", Refresh: "synthetic", Expires: time.Now().Add(time.Hour).Unix()})
							if err != nil {
								t.Fatal(err)
							}
						} else if err := s.Store.CreateSession(t.Context(), "session-alice", uid, csrf); err != nil {
							t.Fatal(err)
						}
					case "store":
						if err := s.Store.Close(); err != nil {
							t.Fatal(err)
						}
					case "cancel":
						cancel()
					case "provider-denied":
						w.WriteHeader(403)
						return w.Result(), nil
					case "provider-unavailable":
						w.WriteHeader(503)
						return w.Result(), nil
					}
					if operation == "start" || operation == "stop" {
						_, _ = w.WriteString(`{"is_owner":true}`)
						return w.Result(), nil
					}
					id := 7
					if operation == "join" {
						id = 8
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "name": "repo", "full_name": "alice/repo", "owner": map[string]any{"id": 1, "login": "alice"}})
					return w.Result(), nil
				})}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				want := 401
				switch change {
				case "unchanged":
					want = 200
				case "provider-denied":
					want = 403
				case "provider-unavailable":
					want = 503
				case "store", "cancel":
					if operation == "apply" { // Saved-key read precedes final admission.
						want = 503
					}
				}
				wantProviderCalls := 2
				if operation == "start" || operation == "stop" {
					wantProviderCalls = 3
				}
				if w.Code != want || providerCalls != wantProviderCalls || (nativeCalls == 1) != (change == "unchanged") || nativeCalls > 1 {
					t.Errorf("admission: status=%d want=%d provider=%d native=%d body=%s", w.Code, want, providerCalls, nativeCalls, w.Body.String())
				}
				shouldEnd := operation == "stop" && change == "unchanged"
				if (terminalCtx.Err() != nil) != shouldEnd || (peerCtx.Err() != nil) != shouldEnd || s.terminalStopping[project] {
					t.Error("denied mutation disturbed terminals or leaked Stop admission")
				}
				if operation == "join" && change != "store" {
					login, err := s.Store.MemberLogin(t.Context(), project, 1)
					if change == "unchanged" {
						if err != nil || login != "current-login" {
							t.Error("confirmed Join did not record original login", err)
						}
					} else if !errors.Is(err, sql.ErrNoRows) {
						t.Error("denied Join recorded membership", err)
					}
				}
			})
		}
	}
}

// Force a post-authentication body delay, including the operator lifecycle path
// that legitimately skips provider I/O and Apply's later decoding boundary.
type mutationAdmissionBody struct {
	io.Reader
	beforeRead func()
}

func (b *mutationAdmissionBody) Read(p []byte) (int, error) {
	if b.beforeRead != nil {
		f := b.beforeRead
		b.beforeRead = nil
		f()
	}
	return b.Reader.Read(p)
}

func TestMutationAdmissionAfterDecode(t *testing.T) {
	for _, operation := range []string{"start", "stop", "apply"} {
		for _, change := range []string{"logout", "store", "cancel"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				s, calls := managementWebFixture(t)
				s.Config.OperatorID = 1
				path, body := "/lifecycle", `{"action":"`+operation+`"}`
				if operation == "stop" {
					body = `{"action":"stop","confirm_stop":true}`
				} else if operation == "apply" {
					path, body = "/access-keys", `{"revision":"`+strings.Repeat("a", 64)+`","saved_fingerprints":[],"confirm_empty":true}`
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				r := apiTestRequest("POST", "/api/environments/"+webTerminalProject+path, body, "alice").WithContext(ctx)
				r.Body = io.NopCloser(&mutationAdmissionBody{Reader: strings.NewReader(body), beforeRead: func() {
					switch change {
					case "logout":
						w := httptest.NewRecorder()
						s.ServeHTTP(w, apiTestRequest("POST", "/api/session/logout", `{}`, "alice"))
						if w.Code != 204 {
							t.Fatal("logout failed", w.Code)
						}
					case "store":
						if err := s.Store.Close(); err != nil {
							t.Fatal(err)
						}
					case "cancel":
						cancel()
					}
				}})
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != 401 || len(*calls) != 0 {
					t.Fatal("post-decode stale session reached native dispatch", w.Code, *calls)
				}
			})
		}
	}
}
