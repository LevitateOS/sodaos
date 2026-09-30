package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

func TestProjectExecutionRequiresCurrentNativeWritePermission(t *testing.T) {
	s, nativeCalls := managementWebFixture(t)
	s.Config.OperatorID = 1
	fresh := "pabcdef0123456789abcdef01"
	require.NoError(t, s.Store.CreateProject(t.Context(), store.Project{ID: fresh, RepositoryID: 8, OwnerID: 1}))
	require.NoError(t, s.Store.MarkReady(t.Context(), fresh, "10.89.0.3"))
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation != extensions.OperationRepository || in.RepositoryID != "7" && in.RepositoryID != "8" {
			t.Error("unexpected native authority lookup", in.Operation, in.RepositoryID)
			return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
		}
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "public", Permission: "read"}}
	}
	for _, action := range []struct{ method, path, body string }{
		{"POST", "/api/environments/" + fresh + "/join", `{}`},
		{"POST", "/api/environments/" + webTerminalProject + "/join", `{}`},
		{"POST", "/api/environments/" + webTerminalProject + "/access-keys", `{"revision":"` + strings.Repeat("a", 64) + `","saved_fingerprints":[],"confirm_empty":true}`},
	} {
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest(action.method, action.path, action.body, "alice"), callback)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		require.Contains(t, w.Body.String(), "repository_write_required")
	}
	require.Empty(t, *nativeCalls)
	_, err := s.Store.MemberLogin(t.Context(), fresh, 1)
	require.ErrorIs(t, err, store.ErrNotFound)

	// Existing membership still permits inspection, but it cannot advertise
	// execution when the native repository now grants read only.
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/inspect" {
			t.Error("unexpected native read", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"` + webTerminalProject + `","running":true}`))}, nil
	})}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments/"+webTerminalProject, "", "alice"), callback)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detail struct {
		Execution     bool `json:"execution_allowed"`
		Administrator bool `json:"environment_administrator"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	require.False(t, detail.Execution)
	require.True(t, detail.Administrator)
}
