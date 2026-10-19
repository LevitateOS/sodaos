package web

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

func TestFactoryActivityNativeVisibilityOwnerAndCAS(t *testing.T) {
	s := apiTestServer(t)
	body := `{"expected_revision":0,"settings":{"issue_created":false,"pr_created":true,"verified_merge":false}}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/repositories/7/factory/activity", "", "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out factory.RepositoryActivity
	decodeBody(t, w, &out)
	if out.Repository != "7" || out.RepositoryPath != "/alice/repo" || out.Revision != 0 || out.Settings != factory.DefaultActivitySettings() || out.Items == nil {
		t.Fatal(out)
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "bob"))
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	decodeBody(t, w, &out)
	if out.Revision != 1 || out.Settings.IssueCreated || !out.Settings.PRCreated {
		t.Fatal(out)
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "alice"))
	if w.Code != 412 {
		t.Fatal(w.Code, w.Body.String())
	}
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation == extensions.OperationRepository {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "bob", Name: "renamed", Permission: "read"}}
		}
		owner := false
		return extensions.CallbackResponse{Owner: &owner}
	}
	w = httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", body, "alice"), callback)
	if w.Code != 403 {
		t.Fatal("operator bypassed native ownership", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories/7/factory/activity", "", "alice"), func(extensions.CallbackRequest) extensions.CallbackResponse {
		return extensions.CallbackResponse{ErrorCode: "not_found"}
	})
	if w.Code != http.StatusNotFound {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("PUT", "/api/repositories/7/factory/activity", `{"expected_revision":1,"settings":{"issue_created":false}}`, "alice"))
	if w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestFactoryActivityRejectsChangedNativeSessionBeforeMutation(t *testing.T) {
	s := apiTestServer(t)
	var changed atomic.Bool
	proxy := nativeProductProxyForActorWithCallbacks(t, s, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, "alice", func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation == extensions.OperationRepository {
			changed.Store(true)
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "write"}}
		}
		owner := false
		return extensions.CallbackResponse{Owner: &owner}
	}, func(extensions.CallbackRequest) extensions.CallbackResponse {
		if changed.Load() {
			return extensions.CallbackResponse{Actor: &extensions.Actor{ID: "2", Username: "bob"}}
		}
		return extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}}
	})
	response := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "PUT", "/api/repositories/7/factory/activity", []byte(`{"expected_revision":0,"settings":{"issue_created":false,"pr_created":false,"verified_merge":false}}`))
	if response.StatusCode != 401 {
		t.Fatal(response.StatusCode)
	}
	settings, revision, err := s.Store.ActivitySettings(t.Context(), 7)
	if err != nil || revision != 0 || settings != factory.DefaultActivitySettings() {
		t.Fatal(settings, revision, err)
	}
}
