package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireCurrentSessionComparesIdentityContextAndCSRF(t *testing.T) {
	s := apiTestServer(t)
	original, err := s.Store.Session(t.Context(), "session-alice")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"unchanged", "user", "context", "csrf", "display"} {
		t.Run(field, func(t *testing.T) {
			v := original
			switch field {
			case "user":
				v.User.ID = 2
			case "context":
				v.ContextID = "other-context"
			case "csrf":
				v.CSRF = "other-csrf"
			case "display":
				v.User.Login, v.User.Name = "old-login", "old-name"
			}
			err := s.requireCurrentSession(t.Context(), "session-alice", v)
			wantOK := field == "unchanged" || field == "display"
			if (err == nil) != wantOK {
				t.Fatal("incorrect current-session decision", field, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.requireCurrentSession(ctx, "session-alice", original); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored", err)
	}
	if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
		t.Fatal(err)
	}
	if s.requireCurrentSession(t.Context(), "session-alice", original) == nil {
		t.Fatal("cached session survived logout")
	}
	if err := s.Store.Close(); err != nil {
		t.Fatal(err)
	}
	if s.requireCurrentSession(t.Context(), "session-alice", original) == nil {
		t.Fatal("unavailable store accepted")
	}
}

func TestPagesRecheckOriginalSessionAfterProviderIO(t *testing.T) {
	for _, page := range []struct {
		path   string
		status int
	}{
		{"/spaces", 503},
		{"/settings/runners", 401},
		{"/repositories/7/settings/spaces", 401},
	} {
		for _, change := range []string{"logout", "user", "csrf", "denied", "unavailable", "provider identity"} {
			t.Run(page.path+"/"+change, func(t *testing.T) {
				s := grantedTestServer(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected upstream connection") })
				calls := 0
				s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
					t.Error("page contacted native helper")
					return nil, errors.New("unexpected native call")
				})}
				s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
					calls++
					w := httptest.NewRecorder()
					switch r.URL.Path {
					case "/api/v1/user":
						switch change {
						case "denied":
							w.WriteHeader(403)
							return w.Result(), nil
						case "unavailable":
							w.WriteHeader(503)
							return w.Result(), nil
						case "provider identity":
							_, _ = w.Write([]byte(`{"id":2,"login":"bob"}`))
							return w.Result(), nil
						}
						if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
							t.Fatal(err)
						}
						if change != "logout" {
							uid, csrf := int64(1), "csrf-alice"
							if change == "user" {
								uid = 2
							} else {
								csrf = "rotated-csrf"
							}
							// Same token/context, independently changed actor or CSRF.
							if err := s.Store.CreateSession(t.Context(), "session-alice", uid, csrf); err != nil {
								t.Fatal(err)
							}
						}
						_, _ = w.Write([]byte(`{"id":1,"login":"alice"}`))
					case "/api/v1/repositories/7":
						_, _ = w.Write([]byte(`{"id":7,"name":"private-repository","full_name":"alice/private-repository","owner":{"id":1,"login":"alice"}}`))
					default:
						t.Error("unexpected provider request", r.URL.Path)
					}
					return w.Result(), nil
				})}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest("GET", page.path, "", "alice"))
				status := page.status
				if page.path != "/spaces" {
					if change == "denied" {
						status = 403
					}
					if change == "unavailable" {
						status = 503
					}
				}
				if w.Code != status || calls == 0 || strings.Contains(w.Body.String(), "private-repository") || strings.Contains(w.Body.String(), "<soda-") {
					t.Fatal("changed context published an authorized page", w.Code, w.Body.String())
				}
			})
		}
	}
}

func TestProfileReadRechecksAfterNativeIO(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			_, _ = w.Write([]byte(`{"id":1,"login":"alice"}`))
		} else {
			_, _ = w.Write([]byte(`{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`))
		}
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
			t.Fatal(err)
		}
		return profileTestResponse(), nil
	})}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories/7/profiles", "", "alice"))
	if w.Code != 401 || strings.Contains(w.Body.String(), "rocky-headless") {
		t.Fatal("logout published native profile", w.Code, w.Body.String())
	}
}
