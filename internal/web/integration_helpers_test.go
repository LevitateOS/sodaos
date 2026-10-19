// Helpers retained for the kept integration tests after unit-test removal.
package web

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func spacesAPI(t *testing.T, s *Server, path string, callback func(extensions.CallbackRequest) extensions.CallbackResponse) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest(http.MethodGet, path, "", "alice"), callback)
	return w
}

func spacesCallback(status int, calls *atomic.Int32) func(extensions.CallbackRequest) extensions.CallbackResponse {
	return func(in extensions.CallbackRequest) extensions.CallbackResponse {
		calls.Add(1)
		if in.Operation != extensions.OperationRepository {
			return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
		}
		switch status {
		case http.StatusForbidden:
			return extensions.CallbackResponse{ErrorCode: "forbidden"}
		case http.StatusNotFound:
			return extensions.CallbackResponse{ErrorCode: "not_found"}
		case http.StatusServiceUnavailable:
			return extensions.CallbackResponse{ErrorCode: "unavailable"}
		default:
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "write"}}
		}
	}
}
