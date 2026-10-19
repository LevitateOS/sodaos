package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
)

func testCreationProfile() project.Profile {
	return project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: runtime.GOARCH, Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
}

func profileTestResponse() *http.Response {
	raw, _ := json.Marshal(testCreationProfile())
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}
}

func TestProfileReadKeepsOwnerAndRequestGuards(t *testing.T) {
	s := apiTestServer(t)
	s.Config.OperatorID = 2
	calls := 0
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		raw, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || r.URL.Path != "/profile" || strings.TrimSpace(string(raw)) != "{}" {
			t.Error("unbounded native profile request")
		}
		return profileTestResponse(), nil
	})}
	for _, tc := range []struct {
		path, login string
		want        int
	}{
		{"/api/repositories/7/profiles", "alice", 200},
		{"/api/repositories/7/profiles", "bob", 403},
		{"/api/repositories/07/profiles", "alice", 400},
		{"/api/repositories/7/profiles?image=other", "alice", 400},
		{"/api/repositories/7/profiles?", "alice", 404},
	} {
		before := calls
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("GET", tc.path, "", tc.login))
		if w.Code != tc.want || (tc.want != 200 && calls != before) {
			t.Fatal(tc, w.Code, calls)
		}
	}
}

func TestProfilePreflightNeverReservesOnUnavailableOrAuthorityChange(t *testing.T) {
	for _, kind := range []string{"unavailable", "generation", "transfer", "wrong architecture response", "wrong profile"} {
		t.Run(kind, func(t *testing.T) {
			owner := "alice"
			s := apiTestServer(t)
			productRequest := apiTestRequest("POST", "/api/environments", `{"repository_id":"7"}`, "alice")
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				if in.Operation != extensions.OperationRepository || in.RepositoryID != "7" {
					t.Error("unexpected native callback", in.Operation, in.RepositoryID)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: owner, Name: "demo", Permission: "write"}}
			}
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/profile" {
					t.Error("unexpected mutation", r.URL.Path)
				}
				switch kind {
				case "unavailable":
					return nil, errors.New("not installed")
				case "generation":
					productRequest.Header.Set(extensions.ContextHeader, strings.Replace(productRequest.Header.Get(extensions.ContextHeader), nativeProductGeneration, "expired", 1))
				case "transfer":
					owner = "bob"
				case "wrong architecture response":
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"rocky-headless","architecture":"other"}`))}, nil
				}
				return profileTestResponse(), nil
			})}
			if kind == "wrong profile" {
				productRequest = apiTestRequest("POST", "/api/environments", `{"repository_id":"7","profile_id":"fedora-kde"}`, "alice")
			}
			w := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, w, productRequest, callback)
			if w.Code < 400 {
				t.Fatal(w.Code, w.Body.String())
			}
			if _, err := s.Store.ProjectByRepository(t.Context(), 7); err == nil {
				t.Fatal("preflight failure reserved a project")
			}
		})
	}
}
