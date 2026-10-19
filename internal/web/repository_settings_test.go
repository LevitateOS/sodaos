package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
)

func TestRepositorySettingsProtectedContext(t *testing.T) {
	denied := false
	s := apiTestServer(t)
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		t.Error("repository context read native project state or mutated it")
		return nil, fmt.Errorf("unexpected native operation")
	})}
	// The old Soda-rendered repository settings page is retired. The native
	// Forgejo contribution supplies the repository context to this API instead.
	route := config.SodaPath + "/repositories/7/settings/spaces"
	for _, suffix := range []string{"", "?repository_id=8", "?", "/../spaces"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", route+suffix, nil))
		if w.Code != 404 {
			t.Fatal("retired page route survived", suffix, w.Code)
		}
	}
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation != extensions.OperationRepository || in.RepositoryID != "7" {
			t.Error("unexpected callback", in.Operation, in.RepositoryID)
			return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
		}
		if denied {
			return extensions.CallbackResponse{ErrorCode: "not_found"}
		}
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "current", Name: "renamed?#", Permission: "write"}}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments?repository_id=7", "", "alice"), callback)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `"owner":"current"`) || !strings.Contains(body, `"name":"renamed?#"`) || strings.Contains(body, "stale/ignored") || strings.Contains(body, "evil.test") {
		t.Fatal(w.Code, body)
	}
	denied = true
	w = httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments?repository_id=7", "", "alice"), callback)
	if w.Code < 400 || strings.Contains(w.Body.String(), "data-actor") || strings.Contains(w.Body.String(), "current/") {
		t.Fatal("denied repository metadata leaked", w.Code)
	}
}
