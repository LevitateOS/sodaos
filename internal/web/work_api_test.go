package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPersonalWorkUsesActingGrantAndLosslessNativeIdentity(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/issues/search" || r.Header.Get("Authorization") != "token acting-bob" || r.URL.Query().Get("assigned") != "true" || r.URL.Query().Has("uid") {
			t.Error("personal work identity or fixed query boundary changed")
		}
		fmt.Fprint(w, `[{"id":9007199254740993,"number":12,"title":"Review","repository":{"id":10,"name":"demo","owner":"alice","full_name":"alice/demo"},"pull_request":{}}]`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/work?assigned=true&type=pulls&uid=1", "", "bob"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) || !strings.Contains(w.Body.String(), `"kind":"pulls"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestNotificationLinksAreReconstructedWithoutRemoteCredentials(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":9007199254740993,"unread":true,"repository":{"id":10,"name":"demo","owner":{"id":1,"login":"alice"}},"subject":{"title":"Subject","html_url":"https://untrusted.invalid/alice/demo/issues/12?credential=fixture-only"}}]`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/notifications", "", "bob"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"route":"/repositories/alice/demo/issues/12"`) || strings.Contains(w.Body.String(), "untrusted") || strings.Contains(w.Body.String(), "fixture-only") {
		t.Fatal("native URL forwarded instead of fixed subject link")
	}
}
func TestNotificationNativeDenialAndResetContentAck(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PATCH" || r.URL.Path != "/api/v1/notifications/threads/9" || r.URL.Query().Get("to-status") != "read" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong native notification operation")
		}
		if calls == 1 {
			w.WriteHeader(403)
		} else {
			w.WriteHeader(205)
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/notifications/9", `{"state":"read"}`, "bob"))
	if w.Code != 403 || calls != 1 {
		t.Fatal("native denial retried or hidden")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/notifications/9", `{"state":"read"}`, "bob"))
	if w.Code != 204 || calls != 2 {
		t.Fatal("native reset-content acknowledgement lost", w.Code)
	}
}
func TestProfileRepositorySearchDoesNotWidenMissingOwner(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/users/missing" {
			t.Error("missing owner became unfiltered repository search")
		}
		w.WriteHeader(404)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repository-search?owner=missing", "", "bob"))
	if w.Code != 404 || calls != 1 {
		t.Fatal("missing native profile search widened")
	}
}
