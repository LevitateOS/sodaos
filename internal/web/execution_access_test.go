package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

func TestProjectExecutionRequiresCurrentWritePermission(t *testing.T) {
	for _, permissions := range []string{"", `,"permissions":{"pull":true,"push":false,"admin":true}`} {
		t.Run(permissions, func(t *testing.T) {
			s, _ := managementWebFixture(t)
			s.Config.OperatorID = 1
			s.Forgejo.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				if r.URL.Path == "/api/v1/user" {
					_, _ = fmt.Fprint(w, `{"id":1,"login":"alice"}`)
				} else {
					id := 7
					if strings.HasSuffix(r.URL.Path, "/8") {
						id = 8
					}
					_, _ = fmt.Fprintf(w, `{"id":%d,"name":"public","full_name":"alice/public","owner":{"id":1,"login":"alice"}%s}`, id, permissions)
				}
				return w.Result(), nil
			})
			fresh := "pabcdef0123456789abcdef01"
			require.NoError(t, s.Store.CreateProject(t.Context(), store.Project{ID: fresh, RepositoryID: 8, OwnerID: 1}))
			require.NoError(t, s.Store.MarkReady(t.Context(), fresh, "10.89.0.3"))
			calls := 0
			s.Host.HTTP.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				t.Error("execution denial reached native helper")
				return nil, fmt.Errorf("unexpected dispatch")
			})
			for _, action := range []struct{ method, path, body string }{
				{"POST", "/api/environments/" + fresh + "/join", `{}`},
				{"POST", "/api/environments/" + webTerminalProject + "/join", `{}`},
				{"POST", "/api/environments/" + webTerminalProject + "/terminal-sessions", `{"cols":80,"rows":24,"name":"Shell"}`},
				{"GET", "/api/environments/" + webTerminalProject + "/terminal-sessions/" + reservedTerminalID, ``},
				{"POST", "/api/environments/" + webTerminalProject + "/access-keys", `{"revision":"` + strings.Repeat("a", 64) + `","saved_fingerprints":[],"confirm_empty":true}`},
			} {
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest(action.method, action.path, action.body, "alice"))
				require.Equal(t, 403, w.Code, w.Body.String())
				require.Contains(t, w.Body.String(), "repository_write_required")
			}
			require.Zero(t, calls)
			_, err := s.Store.MemberLogin(t.Context(), fresh, 1)
			require.ErrorIs(t, err, store.ErrNotFound)
			// Read-only detail remains available, but never advertises execution authority.
			s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				_, _ = fmt.Fprintf(w, `{"id":%q,"running":true}`, webTerminalProject)
				return w.Result(), nil
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+webTerminalProject, "", "alice"))
			require.Equal(t, 200, w.Code)
			var detail struct {
				Execution     bool `json:"execution_allowed"`
				Administrator bool `json:"environment_administrator"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
			require.False(t, detail.Execution)
			require.True(t, detail.Administrator)
		})
	}
}

func TestBrowserTerminalDisconnectsAfterWritePermissionLoss(t *testing.T) {
	s, srv, calls, closed := terminalWebFixture(t, 0)
	var writer atomic.Bool
	writer.Store(true)
	s.Forgejo.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		if r.URL.Path == "/api/v1/user" {
			_, _ = fmt.Fprint(w, `{"id":1,"login":"alice"}`)
		} else {
			_, _ = fmt.Fprintf(w, `{"id":7,"name":"public","full_name":"alice/public","owner":{"id":1,"login":"alice"},"permissions":{"push":%t}}`, writer.Load())
		}
		return w.Result(), nil
	})
	c, _, err := terminalDial(t, srv, "")
	require.NoError(t, err)
	defer c.CloseNow()
	terminalReady(t, c)
	writer.Store(false)
	ctx, cancel := context.WithTimeout(t.Context(), 18*time.Second)
	defer cancel()
	_, _, err = c.Read(ctx)
	require.Error(t, err)
	require.NoError(t, ctx.Err(), "write loss did not disconnect before deadline")
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("native attachment survived write loss")
	}
	require.Equal(t, 1, calls.count("create"))
	// An existing member cannot reconnect through the WebSocket creation path.
	denied, _, err := terminalDial(t, srv, "")
	require.NoError(t, err)
	defer denied.CloseNow()
	require.NoError(t, denied.Write(ctx, websocket.MessageText, []byte(terminalAuth)))
	_, _, err = denied.Read(ctx)
	require.Error(t, err)
	require.Equal(t, 1, calls.count("create"))
}
