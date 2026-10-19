package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestOSObservationUsesExistingReadAuthorityAndGenerationWins(t *testing.T) {
	s := apiTestServer(t)
	s.Config.OperatorID = 1
	id := "p0123456789abcdef01234567"
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1, Name: "legacy", Repository: "alice/legacy", Ready: false}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	generationChanged := false
	var productRequest *http.Request
	invalid := ""
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/os" {
			t.Error("unexpected native operation", r.Method, r.URL.Path)
		}
		var in project.Create
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.ID != id {
			t.Error("wrong native target", err)
		}
		if generationChanged {
			productRequest.Header.Set(extensions.ContextHeader, strings.Replace(productRequest.Header.Get(extensions.ContextHeader), nativeProductGeneration, "expired", 1))
		}
		out := project.OSObservation{Environment: project.Environment{ID: id, Running: true, Image: "sha256:" + strings.Repeat("a", 64)}, Release: &project.OSRelease{ID: "rocky", Version: "9.7", Name: "Rocky Linux 9.7"}}
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
		_ = json.NewEncoder(w).Encode(out)
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
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments/"+id+"/os"+tc.query, "", tc.login), func(in extensions.CallbackRequest) extensions.CallbackResponse {
			if tc.login == "bob" {
				return extensions.CallbackResponse{ErrorCode: "not_found"}
			}
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "legacy", Permission: "write"}}
		})
		if tc.allowed {
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"version":"9.7"`) {
				t.Fatal(w.Code, w.Body.String())
			}
		} else if w.Code < 400 || calls != before {
			t.Fatal("unauthorized/unbounded native read", tc, w.Code, calls)
		}
	}
	for _, invalid = range []string{"target", "shape", "stopped", "state", "image", "bare image", "profile"} {
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/environments/"+id+"/os", "", "alice"))
		if w.Code != 503 || strings.Contains(w.Body.String(), "Rocky") {
			t.Fatal("invalid native receipt escaped", invalid, w.Code, w.Body.String())
		}
	}
	invalid = ""
	generationChanged = true
	w := httptest.NewRecorder()
	productRequest = apiTestRequest("GET", "/api/environments/"+id+"/os", "", "alice")
	nativeAPIServe(t, s, w, productRequest)
	if w.Code != 401 || strings.Contains(w.Body.String(), "Rocky") {
		t.Fatal("native generation change lost publication race", w.Code, w.Body.String())
	}
	p, err := s.Store.Project(t.Context(), id)
	if err != nil || p.Profile != nil || p.Ready {
		t.Fatal("observation backfilled or provisioned legacy root", p, err)
	}
}
