package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

// Real routed handlers/store; the transport doubles complete a session change
// during I/O, then deliberately return a late result (even after cancellation).
// This checks publication, not rollback or cancellation of an admitted read.
func TestEnvironmentReadPublicationRechecksSession(t *testing.T) {
	const permissions = "/api/v1/users/current-login/orgs/team/permissions"
	for _, read := range []struct {
		name, suffix, changeAt                           string
		operator, providerUnavailable, nativeUnavailable bool
	}{
		{name: "detail/provider", changeAt: permissions},
		{name: "detail/helper", changeAt: "/inspect"},
		{name: "detail/native-unavailable", changeAt: "/inspect", nativeUnavailable: true},
		{name: "detail/degraded-member", changeAt: "/inspect", providerUnavailable: true},
		{name: "detail/operator", changeAt: "/inspect", operator: true},
		{name: "members/provider", suffix: "/members", changeAt: permissions},
		{name: "members/degraded-member", suffix: "/members", changeAt: "/api/v1/repositories/7", providerUnavailable: true},
		{name: "connection/helper", suffix: "/connection", changeAt: "/connection"},
	} {
		for _, change := range []string{"unchanged", "logout", "user", "context", "csrf", "store", "cancel"} {
			t.Run(read.name+"/"+change, func(t *testing.T) {
				s, _ := managementWebFixture(t)
				if read.operator {
					s.Config.OperatorID = 1
				}
				project, err := s.Store.Project(t.Context(), webTerminalProject)
				require.NoError(t, err)
				members, err := s.Store.Members(t.Context(), webTerminalProject)
				require.NoError(t, err)
				original, err := s.Store.Session(t.Context(), "session-alice")
				require.NoError(t, err)
				grant, err := s.Store.Grant(t.Context(), "session-alice", original.User.ID)
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				changed := false
				changeSession := func(path string) {
					if path != read.changeAt {
						return
					}
					require.False(t, changed, "read boundary reached more than once")
					changed = true
					switch change {
					case "logout":
						logout := httptest.NewRecorder()
						s.ServeHTTP(logout, apiTestRequest(http.MethodPost, "/api/session/logout", `{}`, "alice"))
						require.Equal(t, http.StatusNoContent, logout.Code)
						_, err := s.Store.Session(t.Context(), "session-alice")
						require.ErrorIs(t, err, sql.ErrNoRows)
					case "user", "context", "csrf":
						require.NoError(t, s.Store.DeleteSession(t.Context(), "session-alice"))
						uid, csrf := original.User.ID, original.CSRF
						if change == "user" {
							uid = 2
						} else if change == "csrf" {
							csrf = "rotated-csrf"
						}
						if change == "context" {
							// Change only the login context, not the token/actor/CSRF/grant.
							require.NoError(t, s.Store.BeginOAuth(t.Context(), "read-new-context", store.OAuthLogin{}, "", ""))
							attempt, err := s.Store.ConsumeOAuth(t.Context(), "read-new-context", "")
							require.NoError(t, err)
							require.NoError(t, s.Store.FinishOAuth(t.Context(), attempt, original.User, "session-alice", csrf, grant))
						} else {
							require.NoError(t, s.Store.CreateGrantedSession(t.Context(), "session-alice", uid, csrf, grant))
						}
						replacement, err := s.Store.Session(t.Context(), "session-alice")
						require.NoError(t, err)
						require.Equal(t, uid, replacement.User.ID)
						require.Equal(t, csrf, replacement.CSRF)
						if change == "context" {
							require.NotEqual(t, original.ContextID, replacement.ContextID)
						} else {
							require.Equal(t, original.ContextID, replacement.ContextID)
						}
					case "store":
						require.NoError(t, s.Store.Close())
					case "cancel":
						cancel()
					}
				}

				providerCalls := 0
				s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
					providerCalls++
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "token acting-alice", r.Header.Get("Authorization"))
					changeSession(r.URL.Path)
					w := httptest.NewRecorder()
					switch r.URL.Path {
					case "/api/v1/user":
						_, _ = w.WriteString(`{"id":1,"login":"current-login"}`)
					case "/api/v1/repositories/7":
						if read.providerUnavailable {
							w.WriteHeader(http.StatusServiceUnavailable)
						} else {
							_, _ = w.WriteString(`{"id":7,"name":"repo","full_name":"team/repo","owner":{"id":99,"login":"team"}}`)
						}
					case permissions:
						_, _ = w.WriteString(`{"is_owner":true}`)
					default:
						t.Fatalf("unexpected provider operation %s", r.URL.Path)
					}
					return w.Result(), nil
				})}
				nativeCalls := 0
				s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
					nativeCalls++
					require.Equal(t, http.MethodPost, r.Method)
					wantPath := "/inspect"
					if read.suffix == "/connection" {
						wantPath = "/connection"
					}
					require.Equal(t, wantPath, r.URL.Path, "read must not mutate native state")
					var input host.Create
					require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
					require.Equal(t, host.Create{ID: webTerminalProject}, input)
					changeSession(r.URL.Path)
					if read.nativeUnavailable {
						return nil, errors.New("synthetic-private-helper-error")
					}
					w := httptest.NewRecorder()
					observed := host.Environment{ID: webTerminalProject, IP: "10.89.0.2", Running: true}
					if read.suffix == "/connection" {
						require.NoError(t, json.NewEncoder(w).Encode(host.Connection{Environment: observed, HostKey: "public-fixture", Fingerprint: "SHA256:fixture"}))
					} else {
						require.NoError(t, json.NewEncoder(w).Encode(observed))
					}
					return w.Result(), nil
				})}

				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest(http.MethodGet, "/api/environments/"+webTerminalProject+read.suffix, "", "alice").WithContext(ctx))
				require.True(t, changed, "request never reached the selected I/O boundary")
				wantProvider := 3
				if read.operator || read.suffix == "/connection" {
					wantProvider = 0
				} else if read.providerUnavailable {
					wantProvider = 2
				}
				require.Equal(t, wantProvider, providerCalls)
				wantNative := 1
				if read.suffix == "/members" {
					wantNative = 0
				}
				require.Equal(t, wantNative, nativeCalls)
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
				require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
				for _, secret := range []string{original.CSRF, grant.Access, "synthetic-private-helper-error"} {
					require.NotContains(t, w.Body.String(), secret)
				}
				wantStatus := http.StatusUnauthorized
				if change == "unchanged" {
					wantStatus = http.StatusOK
				} else if read.name == "members/provider" && (change == "store" || change == "cancel") {
					// The existing membership query fails before final publication.
					wantStatus = http.StatusServiceUnavailable
				}
				require.Equal(t, wantStatus, w.Code, w.Body.String())
				if change == "unchanged" {
					var result struct {
						Environment          environmentView   `json:"environment"`
						Observed             *host.Environment `json:"observed"`
						Administrator        bool              `json:"environment_administrator"`
						AuthorityUnavailable bool              `json:"authority_unavailable"`
						NativeUnavailable    bool              `json:"native_unavailable"`
						Login                string            `json:"login"`
						Items                []struct {
							UserID string `json:"user_id"`
							Login  string `json:"login"`
						} `json:"items"`
						Connection      host.Connection `json:"connection"`
						RoutingVerified bool            `json:"routing_verified"`
					}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
					require.Equal(t, read.providerUnavailable, result.AuthorityUnavailable)
					switch read.suffix {
					case "":
						require.Equal(t, environmentDTO(project), result.Environment)
						require.Equal(t, "original-alice", result.Login)
						require.Equal(t, !read.providerUnavailable, result.Administrator)
						require.Equal(t, read.nativeUnavailable, result.NativeUnavailable)
						if read.nativeUnavailable {
							require.Nil(t, result.Observed)
						} else {
							require.Equal(t, &host.Environment{ID: webTerminalProject, IP: "10.89.0.2", Running: true}, result.Observed)
						}
					case "/members":
						if read.providerUnavailable {
							require.Len(t, result.Items, 1)
							require.Equal(t, "1", result.Items[0].UserID)
							require.Equal(t, "original-alice", result.Items[0].Login)
						} else {
							require.Len(t, result.Items, 2)
							require.Contains(t, w.Body.String(), "original-bob")
						}
					case "/connection":
						require.Equal(t, "original-alice", result.Login)
						require.Equal(t, webTerminalProject, result.Connection.Environment.ID)
						require.Equal(t, "public-fixture", result.Connection.HostKey)
						require.Equal(t, "SHA256:fixture", result.Connection.Fingerprint)
						require.False(t, result.RoutingVerified)
					}
				} else {
					var result map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
					require.Len(t, result, 1, "refusal must not include protected data")
					require.Contains(t, result, "error")
					if wantStatus == http.StatusUnauthorized {
						require.Contains(t, w.Body.String(), `"code":"unauthenticated"`)
					}
				}
				if change != "store" {
					retained, err := s.Store.Project(t.Context(), webTerminalProject)
					require.NoError(t, err)
					require.Equal(t, project, retained)
					retainedMembers, err := s.Store.Members(t.Context(), webTerminalProject)
					require.NoError(t, err)
					require.Equal(t, members, retainedMembers)
					_, err = s.Store.Session(t.Context(), "session-bob")
					require.NoError(t, err, "read/logout must preserve the other actor")
				}
			})
		}
	}
}
