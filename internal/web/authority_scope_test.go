package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyAdministrationConsentDoesNotGrantRoleOrReplay(t *testing.T) {
	s := grantedTestServer(t, func(http.ResponseWriter, *http.Request) { t.Error("missing admin consent reached provider") })
	grant, err := s.Store.Grant(t.Context(), "session-alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	grant.Scopes = "read:user read:repository"
	if err := s.Store.ReplaceGrant(t.Context(), "session-alice", 1, grant); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "/people", strings.NewReader("csrf=csrf-alice&login=test&email=test%40example.test&password=short"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-alice"})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 || !strings.Contains(w.Body.String(), "/login?administration=1") || !strings.Contains(w.Body.String(), "https://forgejo.example.test/user/settings/applications") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
