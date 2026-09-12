package web

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

// Delay body decoding after the real operator/provider gate. Session changes use
// the real store/logout route; the existing fixture never executes native commands.
func TestRunnerMutationAdmissionAfterDecode(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		if operation.method != http.MethodPost {
			continue
		}
		for _, change := range []string{"unchanged", "logout", "user", "context", "csrf", "store", "cancel"} {
			t.Run(operation.name+"/"+change, func(t *testing.T) {
				s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.URL.Path != "/runners/"+operation.name {
						t.Error("unexpected helper operation", r.Method, r.URL.Path)
					}
					_, _ = w.Write([]byte(`{"ok":true}`))
				})
				// Count attempts before the transport can reject a cancelled context;
				// transport cancellation alone is not web mutation admission.
				var calls atomic.Int32
				transport := s.Host.HTTP.Transport
				s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					return transport.RoundTrip(r)
				})
				original, err := s.Store.Session(t.Context(), "session-alice")
				require.NoError(t, err)
				grant, err := s.Store.Grant(t.Context(), "session-alice", original.User.ID)
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				r := apiTestRequest(operation.method, operation.path, operation.body, "alice").WithContext(ctx)
				read := false
				r.Body = io.NopCloser(&mutationAdmissionBody{Reader: strings.NewReader(operation.body), beforeRead: func() {
					read = true
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
							// Preserve the token, actor, CSRF and grant so only the
							// original-context comparison can detect this replacement.
							require.NoError(t, s.Store.BeginOAuth(t.Context(), "runner-new-context", store.OAuthLogin{}, "", ""))
							attempt, err := s.Store.ConsumeOAuth(t.Context(), "runner-new-context", "")
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
				}})
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				require.True(t, read, "request never reached the post-authorization body read")
				wantStatus, wantCalls := http.StatusUnauthorized, int32(0)
				if change == "unchanged" {
					wantStatus, wantCalls = http.StatusOK, 1
				}
				if w.Code != wantStatus || calls.Load() != wantCalls {
					t.Fatalf("status=%d want=%d helper_dispatches=%d want=%d", w.Code, wantStatus, calls.Load(), wantCalls)
				}
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
				require.NotContains(t, w.Body.String(), "synthetic-runner-secret")
				if change == "unchanged" {
					require.JSONEq(t, `{"ok":true}`, w.Body.String())
				} else {
					require.Contains(t, w.Body.String(), `"code":"reauthentication_required"`)
				}
			})
		}
	}
}
