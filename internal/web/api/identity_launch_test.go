package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

type identityRoundTrip func(*http.Request) (*http.Response, error)

func (f identityRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func (f *identityFake) EndLease(_ context.Context, actor int64, id string) error {
	f.owner = actor
	f.calls++
	return nil
}

func TestIdentityLaunchBindsCurrentActorAndCompensatesRetiredSession(t *testing.T) {
	for _, retire := range []bool{false, true} {
		t.Run(map[bool]string{false: "current", true: "retired"}[retire], func(t *testing.T) {
			s, mux, fake := identityFixture(t, true)
			project := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: project, RepositoryID: 7, OwnerID: 1}); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.MarkReady(t.Context(), project, "10.0.0.2"); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.Join(t.Context(), project, 1, "soda-owner"); err != nil {
				t.Fatal(err)
			}
			calls := 0
			s.Host = &host.Client{HTTP: &http.Client{Transport: identityRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				var in identity.TerminalStart
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					t.Fatal(err)
				}
				if in.ActorID != 1 || in.ProjectID != project || in.Login != "soda-owner" || len(in.Scope) != 64 {
					t.Fatalf("wrong trusted actor: %+v", in)
				}
				if retire {
					if err := s.Store.DeleteSession(t.Context(), "session"); err != nil {
						t.Fatal(err)
					}
				}
				w := httptest.NewRecorder()
				_ = json.NewEncoder(w).Encode(identity.Lease{ID: "lease", ExecutionID: strings.Repeat("a", 32)})
				return w.Result(), nil
			})}}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, identityRequest("POST", "/api/environments/"+project+"/identity/launch", `{"connection_id":"connection","cols":80,"rows":24}`))
			if calls != 1 {
				t.Fatalf("launch calls %d", calls)
			}
			if retire {
				if w.Code != 401 || fake.calls != 1 || fake.owner != 1 {
					t.Fatalf("retired session not ended: %d calls %d", w.Code, fake.calls)
				}
			} else if w.Code != 200 || !strings.Contains(w.Body.String(), `"terminal_id"`) {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestIdentityLaunchRejectsCallerAuthorityAndMissingWrite(t *testing.T) {
	_, mux, _ := identityFixture(t, false)
	for _, body := range []string{`{"connection_id":"connection","cols":80,"rows":24,"actor_id":"2"}`, `{"connection_id":"connection","cols":0,"rows":24}`} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, identityRequest("POST", "/api/environments/p0123456789abcdef01234567/identity/launch", body))
		if w.Code != 400 {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}
