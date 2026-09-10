package web

import (
	"encoding/json"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOSObservationUsesExistingReadAuthorityAndLogoutWins(t *testing.T) {
	s := apiTestServer(t)
	s.Config.OperatorID = 1
	id := "p0123456789abcdef01234567"
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1, Name: "legacy", Repository: "alice/legacy", Ready: false}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	logout := false
	invalid := ""
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/os" {
			t.Error("unexpected native operation", r.Method, r.URL.Path)
		}
		var in host.Create
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.ID != id {
			t.Error("wrong native target", err)
		}
		if logout {
			if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
				t.Error(err)
			}
		}
		out := host.OSObservation{Environment: host.Environment{ID: id, Running: true, Image: "sha256:" + strings.Repeat("a", 64)}, Release: &host.OSRelease{ID: "rocky", Version: "9.7", Name: "Rocky Linux 9.7"}}
		switch invalid {
		case "target":
			out.Environment.ID = "paaaaaaaaaaaaaaaaaaaaaaaa"
		case "shape":
			out.Release.Version = ""
		case "stopped":
			out.Environment.Running = false
		case "state":
			out.Unavailable = true
		case "image":
			out.Environment.Image = "untrusted-tag"
		case "bare image":
			out.Environment.Image = strings.Repeat("a", 64)
		case "profile":
			p := testCreationProfile()
			p.Revision = ""
			out.Environment.Profile = &p
		}
		json.NewEncoder(w).Encode(out)
	}))
	defer native.Close()
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = native.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(r)
	})}
	for _, tc := range []struct {
		login, query string
		allowed      bool
	}{{"bob", "", false}, {"alice", "?path=/etc/shadow", false}, {"alice", "", true}} {
		before := calls
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id+"/os"+tc.query, "", tc.login))
		if tc.allowed {
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"version":"9.7"`) {
				t.Fatal(w.Code, w.Body.String())
			}
		} else if w.Code < 400 || calls != before {
			t.Fatal("unauthorized/unbounded native read", w.Code, calls)
		}
	}
	for _, invalid = range []string{"target", "shape", "stopped", "state", "image", "bare image", "profile"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id+"/os", "", "alice"))
		if w.Code != 503 || strings.Contains(w.Body.String(), "Rocky") {
			t.Fatal("invalid native receipt escaped", invalid, w.Code, w.Body.String())
		}
	}
	invalid = ""
	logout = true
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id+"/os", "", "alice"))
	if w.Code != 401 || strings.Contains(w.Body.String(), "Rocky") {
		t.Fatal("logout lost publication race", w.Code, w.Body.String())
	}
	p, err := s.Store.Project(t.Context(), id)
	if err != nil || p.Profile != nil || p.Ready {
		t.Fatal("observation backfilled or provisioned legacy root", p, err)
	}
}
