package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestRepositorySettingsUsesFreshStableIdentityAndSharedControls(t *testing.T) {
	denied := false
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":1,"login":"alice"}`)
		case "/api/v1/repositories/7":
			if denied {
				w.WriteHeader(404)
				return
			}
			fmt.Fprint(w, `{"id":7,"name":"renamed?#","full_name":"stale/ignored","html_url":"https://evil.test/","owner":{"id":2,"login":"current"}}`)
		default:
			t.Error("unexpected provider path", r.URL.Path)
			w.WriteHeader(500)
		}
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		t.Error("HTML read native state or mutated it")
		return nil, fmt.Errorf("unexpected native operation")
	})}
	route := config.SodaPath + "/repositories/7/settings/spaces"
	anonymous := httptest.NewRecorder()
	s.ServeHTTP(anonymous, httptest.NewRequest("GET", route, nil))
	if anonymous.Code != 303 || !strings.Contains(anonymous.Header().Get("Location"), "redirect_to=%2F%3Fsoda-view%3Drepository-spaces%26repository_id%3D7") || strings.Contains(anonymous.Body.String(), "data-actor") {
		t.Fatal(anonymous.Code, anonymous.Body.String())
	}
	for _, suffix := range []string{"?repository_id=8", "?", "/../spaces"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", route+suffix, nil))
		if w.Code < 400 {
			t.Fatal("route alias accepted", suffix, w.Code)
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/repositories/7/settings/spaces", "", "alice"))
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `data-actor="1" data-repository-id="7"`) || !strings.Contains(body, "/current/renamed%3F%23/settings") || strings.Contains(body, "stale/ignored") || strings.Contains(body, "evil.test") || !strings.Contains(body, "soda-repository-spaces.js") || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(w.Code, body)
	}
	if output := os.Getenv("SODA_REPOSITORY_SETTINGS_HTML"); output != "" {
		f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		err = json.NewEncoder(f).Encode(map[string]string{"html": body, "csp": w.Header().Get("Content-Security-Policy")})
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	denied = true
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/repositories/7/settings/spaces", "", "alice"))
	if w.Code < 400 || strings.Contains(w.Body.String(), "data-actor") || strings.Contains(w.Body.String(), "current/") {
		t.Fatal("denied repository metadata leaked", w.Code)
	}
}
