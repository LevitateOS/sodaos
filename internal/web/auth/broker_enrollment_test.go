package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

type enrollmentBrokerFake struct {
	unretained                             bool
	completions, retentions, cancellations int
	failure                                bool
	afterComplete                          func()
}

func (f *enrollmentBrokerFake) CompleteEnrollment(_ context.Context, owner int64, id, state, code string) (identity.Enrollment, error) {
	f.completions++
	if owner != 1 || id != "enrollment" || state != "native-state" || code != "native-code" {
		return identity.Enrollment{}, identity.ErrDenied
	}
	if f.afterComplete != nil {
		f.afterComplete()
	}
	if f.failure {
		return identity.Enrollment{}, errors.New("sensitive-native-error")
	}
	return identity.Enrollment{State: "completed", Connection: &identity.Connection{OwnerID: 1, ProviderID: identity.Forgejo}}, nil
}

func (f *enrollmentBrokerFake) Enrollment(context.Context, int64, string) (identity.Enrollment, error) {
	f.retentions++
	if f.unretained {
		return identity.Enrollment{State: "pending"}, nil
	}
	return identity.Enrollment{State: "completed", Connection: &identity.Connection{OwnerID: 1, ProviderID: identity.Forgejo}}, nil
}

func (f *enrollmentBrokerFake) CancelEnrollment(context.Context, int64, string) error {
	f.cancellations++
	return nil
}

func brokerFixture(t *testing.T) (*Service, *enrollmentBrokerFake, store.Session) {
	t.Helper()
	db, err := store.OpenEncrypted(filepath.Join(t.TempDir(), "broker.db"), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.UpsertUser(t.Context(), store.User{ID: 1, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"session", "other-session"} {
		if err := db.CreateGrantedSession(t.Context(), token, 1, "csrf", store.Grant{Access: "fixture-access", Refresh: "fixture-refresh", Expires: time.Now().Add(time.Hour).Unix(), Scopes: "read:user"}); err != nil {
			t.Fatal(err)
		}
	}
	s := New(&config.Config{ForgejoURL: "https://forgejo.example.test"}, db, nil)
	fake := &enrollmentBrokerFake{}
	s.EnrollmentBroker = fake
	s.SessionEndGate = &sync.Mutex{}
	session, err := db.Session(t.Context(), "session")
	if err != nil {
		t.Fatal(err)
	}
	return s, fake, session
}

func brokerEnrollment() identity.Enrollment {
	q := url.Values{"state": {"native-state"}, "redirect_uri": {"https://forgejo.example.test/-/soda/identity/callback"}}
	return identity.Enrollment{ID: "enrollment", ProviderID: identity.Forgejo, VerificationURL: "https://forgejo.example.test/login/oauth/authorize?" + q.Encode()}
}

func bindBroker(t *testing.T, s *Service, session store.Session) *http.Cookie {
	t.Helper()
	r := httptest.NewRequest("POST", "/api/identity/enrollments", nil)
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "session"})
	w := httptest.NewRecorder()
	if err := s.BindBrokerEnrollment(w, r, session, brokerEnrollment()); err != nil {
		t.Fatal(err)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 600 {
		t.Fatal("unsafe binding cookie")
	}
	return cookie
}

func callbackBroker(s *Service, cookie *http.Cookie, token, query string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "/identity/callback?"+query, nil)
	r.AddCookie(cookie)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: SessionCookie, Value: token})
	}
	w := httptest.NewRecorder()
	s.brokerCallback(w, r)
	return w
}

func TestBrokerCallbackBindsSessionAndRejectsReplay(t *testing.T) {
	s, fake, session := brokerFixture(t)
	cookie := bindBroker(t, s, session)
	for _, tc := range []struct{ token, query string }{
		{"", "state=native-state&code=native-code"},
		{"other-session", "state=native-state&code=native-code"},
		{"session", "state=foreign-state&code=native-code"},
		{"session", "state=native-state&state=native-state&code=native-code"},
		{"session", "state=native-state&code=native-code&error=denied"},
	} {
		if w := callbackBroker(s, cookie, tc.token, tc.query); w.Code != 403 {
			t.Fatal("invalid callback accepted")
		}
	}
	if fake.completions != 0 || len(s.brokerBindings) != 1 {
		t.Fatal("invalid callback consumed binding")
	}
	w := callbackBroker(s, cookie, "session", "state=native-state&code=native-code")
	if w.Code != 200 || fake.completions != 1 || fake.retentions != 1 {
		t.Fatal("bound native completion failed")
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" || strings.Contains(w.Body.String(), "native-code") {
		t.Fatal("callback leaked authorization response")
	}
	if w := callbackBroker(s, cookie, "session", "state=native-state&code=native-code"); w.Code != 403 || fake.completions != 1 {
		t.Fatal("callback replay accepted")
	}
}

func TestBrokerCallbackFailureLogoutAndExpiry(t *testing.T) {
	for _, reason := range []string{"native failure", "logout before callback", "logout during completion", "expired binding", "restart", "native denial"} {
		t.Run(reason, func(t *testing.T) {
			s, fake, session := brokerFixture(t)
			cookie := bindBroker(t, s, session)
			query := "state=native-state&code=native-code"
			switch reason {
			case "native failure":
				fake.failure = true
			case "logout before callback":
				if err := s.Store.DeleteSession(t.Context(), "session"); err != nil {
					t.Fatal(err)
				}
			case "logout during completion":
				fake.afterComplete = func() {
					if err := s.Store.DeleteSession(t.Context(), "session"); err != nil {
						t.Fatal(err)
					}
				}
			case "expired binding":
				b := s.brokerBindings[cookie.Value]
				b.expires = time.Now().Add(-time.Second)
				s.brokerBindings[cookie.Value] = b
			case "restart":
				s.brokerBindings = nil
			case "native denial":
				query = "state=native-state&error=sensitive-native-error"
			}
			w := callbackBroker(s, cookie, "session", query)
			if w.Code != 403 || fake.retentions != 0 || strings.Contains(w.Body.String(), "sensitive-native-error") {
				t.Fatal("unsafe failed callback")
			}
			if reason == "native failure" && fake.cancellations != 1 {
				t.Fatal("failed native enrollment not canceled")
			}
		})
	}
}

func TestBrokerBindingRejectsForeignURLAndIsBounded(t *testing.T) {
	s, _, session := brokerFixture(t)
	for _, raw := range []string{
		"https://foreign.example.test/login/oauth/authorize?state=native-state&redirect_uri=https://forgejo.example.test/-/soda/identity/callback",
		"https://forgejo.example.test/login/oauth/authorize?state=native-state&redirect_uri=https://foreign.example.test/callback",
		"https://forgejo.example.test/login/oauth/authorize?state=native-state&state=other&redirect_uri=https://forgejo.example.test/-/soda/identity/callback",
	} {
		enrollment := brokerEnrollment()
		enrollment.VerificationURL = raw
		r := httptest.NewRequest("POST", "/", nil)
		r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "session"})
		if err := s.BindBrokerEnrollment(httptest.NewRecorder(), r, session, enrollment); err == nil {
			t.Fatal("foreign authorization URL accepted")
		}
	}
	cookie := bindBroker(t, s, session)
	r := httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "other-session"})
	other, err := s.Store.Session(t.Context(), "other-session")
	if err != nil {
		t.Fatal(err)
	}
	s.ForgetBrokerEnrollment(r, other, "enrollment")
	if len(s.brokerBindings) != 1 {
		t.Fatal("different session removed binding")
	}
	r = httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "session"})
	s.ForgetBrokerEnrollment(r, session, "enrollment")
	if _, ok := s.brokerBindings[cookie.Value]; ok {
		t.Fatal("cancellation retained binding")
	}
	for i := 0; i < 128; i++ {
		s.brokerBindings[string(rune(i))] = brokerBinding{expires: time.Now().Add(time.Minute)}
	}
	if err := s.BindBrokerEnrollment(httptest.NewRecorder(), r, session, brokerEnrollment()); err == nil {
		t.Fatal("binding capacity exceeded")
	}
}

func TestBrokerCallbackRequiresRetainedConnection(t *testing.T) {
	s, fake, session := brokerFixture(t)
	cookie := bindBroker(t, s, session)
	fake.unretained = true
	w := callbackBroker(s, cookie, "session", "state=native-state&code=native-code")
	if w.Code != 403 || strings.Contains(w.Body.String(), "Forgejo connected") {
		t.Fatal("pending retention reported connection success")
	}
}
