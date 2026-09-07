package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/store"
)

func TestConcurrentNewJoinsAndResultPersistenceFailure(t *testing.T) {
	for _, failStore := range []bool{false, true} {
		t.Run(fmt.Sprint(failStore), func(t *testing.T) {
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":2,"login":"bob"}`)
				case "/api/v1/repositories/7":
					fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`)
				default:
					t.Error("unexpected provider call")
					w.WriteHeader(500)
				}
			})
			id := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1}); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.MarkReady(t.Context(), id, "10.89.0.2"); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.AddKey(t.Context(), 2, "public-key-double", "fingerprint"); err != nil {
				t.Fatal(err)
			}
			entered := make(chan struct{}, 2)
			release := make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/account" {
					t.Error("unexpected native call")
				}
				if failStore {
					s.Store.Close()
				} else {
					entered <- struct{}{}
					<-release
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
			})}
			count := 2
			if failStore {
				count = 1
			}
			results := make(chan *httptest.ResponseRecorder, count)
			for i := 0; i < count; i++ {
				go func() {
					w := httptest.NewRecorder()
					s.ServeHTTP(w, apiTestRequest("POST", "/api/environments/"+id+"/join", "{}", "bob"))
					results <- w
				}()
			}
			if !failStore {
				for i := 0; i < count; i++ {
					select {
					case <-entered:
					case <-time.After(10 * time.Second):
						t.Fatal("joins did not reach barrier")
					}
				}
				unblock()
			}
			successes, failures := 0, 0
			for i := 0; i < count; i++ {
				w := <-results
				switch {
				case w.Code == 200:
					successes++
				case w.Code == 503 && strings.Contains(w.Body.String(), `"membership_not_saved"`):
					failures++
				default:
					t.Fatal(w.Code, w.Body.String())
				}
			}
			if failures != 1 || (!failStore && successes != 1) || (failStore && successes != 0) {
				t.Fatal(successes, failures)
			}
			if !failStore {
				login, err := s.Store.MemberLogin(t.Context(), id, 2)
				if err != nil || login != "bob" {
					t.Fatal(login, err)
				}
			}
		})
	}
}

func TestConnectionRemainsOwnMembershipOnlyDuringProviderFailure(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) })
	id := "p0123456789abcdef01234567"
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/connection" {
			t.Error("unexpected native call")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"environment":{"id":%q,"ip":"10.89.0.2","running":true},"host_key":"public-fixture","fingerprint":"SHA256:fixture"}`, id)))}, nil
	})}
	for _, login := range []string{"alice", "bob"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id+"/connection", "", login))
		if w.Code != 403 || calls != 0 {
			t.Fatal("nonmember/operator obtained connection", w.Code)
		}
	}
	if err := s.Store.Join(t.Context(), id, 2, "original-bob"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id+"/connection", "", "bob"))
	if w.Code != 200 || calls != 1 || !strings.Contains(w.Body.String(), `"login":"original-bob"`) || !strings.Contains(w.Body.String(), `"routing_verified":false`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
