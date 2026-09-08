package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/host"
)

func TestStableCreateRejectsLegacyAndInvalidIDsBeforeProvider(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid input reached provider")
		w.WriteHeader(500)
	})
	for _, body := range []string{`{}`, `{"owner":"alice","repository":"demo"}`, `{"repository_id":7}`, `{"repository_id":"0"}`, `{"repository_id":"07"}`, `{"repository_id":"9223372036854775808"}`, `{"repository_id":"7","owner":"alice"}`, `{"repository_id":"7","repository_id":"8"}`, "{\"repository_id\":\"7\xff\"}"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/environments", body, "alice"))
		if w.Code != 400 {
			t.Fatal(body, w.Code)
		}
	}
	for _, header := range []string{"Origin", "X-CSRF-Token", expectedUserHeader} {
		r := apiTestRequest("POST", "/api/environments", `{"repository_id":"7"}`, "alice")
		value := "wrong"
		if header == expectedUserHeader {
			value = "2"
		}
		r.Header.Set(header, value)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(header, w.Code)
		}
	}
}

func TestCreateRechecksCurrentOwnerAfterAdvisoryRead(t *testing.T) {
	var owner atomic.Int64
	owner.Store(1)
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":1,"login":"alice-renamed","is_admin":true}`)
		case "/api/v1/repositories/9223372036854775807":
			fmt.Fprintf(w, `{"id":9223372036854775807,"name":"renamed","full_name":"current/renamed","owner":{"id":%d,"login":"current"}}`, owner.Load())
		default:
			t.Error("not stable-ID lookup", r.URL.Path)
			w.WriteHeader(500)
		}
	})
	s.Config.OperatorID = 1 // Neither this nor the native admin flag permits impersonation.
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		t.Error("transferred repository reached helper")
		return nil, context.Canceled
	})}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/environments?repository_id=9223372036854775807", "", "alice"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"can_create":true`) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, next := range []int64{2, 99} { // Another human or an organization: neither is this acting owner.
		owner.Store(next)
		w = httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/environments", `{"repository_id":"9223372036854775807"}`, "alice"))
		if w.Code != 403 || !strings.Contains(w.Body.String(), `"owner_required"`) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestCreateReservationSurvivesConcurrentAndUncertainResults(t *testing.T) {
	for _, outcome := range []string{"success", "native failure", "invalid native result", "persistence failure"} {
		t.Run(outcome, func(t *testing.T) {
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":1,"login":"alice-renamed"}`)
				case "/api/v1/repositories/7":
					fmt.Fprint(w, `{"id":7,"name":"renamed","full_name":"alice-renamed/renamed","owner":{"id":1,"login":"alice-renamed"}}`)
				default:
					t.Error("unexpected provider path", r.URL.Path)
					w.WriteHeader(500)
				}
			})
			entered, release := make(chan string, 1), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			var calls atomic.Int32
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != "/create" {
					t.Error("implicit join or unexpected operation", r.URL.Path)
				}
				var input host.Create
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					t.Error(err)
				}
				if input.Owner != 1 {
					t.Error("wrong native owner")
				}
				entered <- input.ID
				<-release
				if outcome == "native failure" {
					return nil, context.DeadlineExceeded
				}
				if outcome == "persistence failure" {
					s.Store.Close()
				}
				ip := "10.89.0.2"
				if outcome == "invalid native result" {
					ip = "127.0.0.1"
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":%q,"running":true,"ip":%q}`, input.ID, ip)))}, nil
			})}
			perform := func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest("POST", "/api/environments", `{"repository_id":"7"}`, "alice"))
				return w
			}
			done := make(chan *httptest.ResponseRecorder, 1)
			go func() { done <- perform() }()
			var id string
			select {
			case id = <-entered:
			case <-time.After(10 * time.Second):
				t.Fatal("create did not reach helper")
			}
			p, err := s.Store.ProjectByRepository(t.Context(), 7)
			if err != nil || p.ID != id || p.Ready || p.Repository != "alice-renamed/renamed" {
				t.Fatal("reservation not retained before native call", p, err)
			}
			if second := perform(); second.Code != 409 {
				t.Fatal("duplicate reservation accepted", second.Code)
			}
			unblock()
			w := <-done
			want := 201
			if outcome == "native failure" || outcome == "invalid native result" {
				want = 502
			}
			if outcome == "persistence failure" {
				want = 503
			}
			if w.Code != want || calls.Load() != 1 {
				t.Fatal(w.Code, w.Body.String(), calls.Load())
			}
			if outcome != "persistence failure" {
				p, err = s.Store.ProjectByRepository(t.Context(), 7)
				if err != nil || p.ID != id || p.Ready != (want == 201) {
					t.Fatal("lost reservation", p, err)
				}
				members, err := s.Store.Members(t.Context(), id)
				if err != nil || len(members) != 0 {
					t.Fatal("implicit membership", members, err)
				}
			} else if !strings.Contains(w.Body.String(), `"result_not_saved"`) {
				t.Fatal(w.Body.String())
			}
		})
	}
}
