package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

const (
	brokerCookie   = "__Secure-soda-broker-enrollment"
	brokerLifetime = 10 * time.Minute
	brokerTimeout  = 15 * time.Second
)

// EnrollmentBroker completes native enrollment without delivering credentials.
type EnrollmentBroker interface {
	CompleteEnrollment(context.Context, int64, string, string, string) (identity.Enrollment, error)
	Enrollment(context.Context, int64, string) (identity.Enrollment, error)
	CancelEnrollment(context.Context, int64, string) error
}

type brokerBinding struct {
	enrollment, state string
	owner             int64
	contextID, csrf   string
	tokenHash         [32]byte
	expires           time.Time
}

func (s *Service) brokerState(enrollment identity.Enrollment) (string, error) {
	u, err := url.Parse(enrollment.VerificationURL)
	if err != nil || !s.brokerAuthorizeURL(u) {
		return "", identity.ErrDenied
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q["state"]) != 1 || len(q["redirect_uri"]) != 1 {
		return "", identity.ErrDenied
	}
	state := q.Get("state")
	if state == "" || len(state) > 128 || q.Get("redirect_uri") != s.Config.ForgejoURL+config.SodaPath+"/identity/callback" {
		return "", identity.ErrDenied
	}
	return state, nil
}

func (s *Service) brokerAuthorizeURL(u *url.URL) bool {
	return u.Scheme == "https" && u.User == nil && u.Fragment == "" && u.RawPath == "" && u.Scheme+"://"+u.Host == s.Config.ForgejoURL && u.Path == "/login/oauth/authorize"
}

func brokerBindingKey() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", errors.New("broker enrollment unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(random[:]), nil
}

func (s *Service) saveBrokerBinding(key string, binding brokerBinding) error {
	s.brokerMu.Lock()
	defer s.brokerMu.Unlock()
	s.pruneBrokerBindings()
	if len(s.brokerBindings) >= 128 {
		return identity.ErrBusy
	}
	if s.brokerBindings == nil {
		s.brokerBindings = make(map[string]brokerBinding)
	}
	s.brokerBindings[key] = binding
	return nil
}

func (s *Service) pruneBrokerBindings() {
	for key, binding := range s.brokerBindings {
		if !time.Now().Before(binding.expires) {
			delete(s.brokerBindings, key)
		}
	}
}

// BindBrokerEnrollment binds a native authorization URL to this initiating session.
func (s *Service) BindBrokerEnrollment(w http.ResponseWriter, r *http.Request, session store.Session, enrollment identity.Enrollment) error {
	if s.EnrollmentBroker == nil || enrollment.ID == "" || enrollment.ProviderID != identity.Forgejo {
		return identity.ErrDenied
	}
	state, err := s.brokerState(enrollment)
	if err != nil {
		return err
	}
	cookie, err := RequestCookie(r, SessionCookie)
	if err != nil {
		return identity.ErrDenied
	}
	key, err := brokerBindingKey()
	if err != nil {
		return err
	}
	s.withSessionEndGate(func() {
		if err = s.RequireCurrentSession(r.Context(), cookie.Value, session); err != nil {
			return
		}
		err = s.saveBrokerBinding(key, brokerBinding{enrollment: enrollment.ID, state: state, owner: session.User.ID, contextID: session.ContextID, csrf: session.CSRF, tokenHash: sha256.Sum256([]byte(cookie.Value)), expires: time.Now().Add(brokerLifetime)})
	})
	if err != nil {
		return identity.ErrDenied
	}
	s.cookie(w, brokerCookie, key, int(brokerLifetime.Seconds()))
	return nil
}

func (b brokerBinding) matches(session store.Session, token string) bool {
	return b.owner == session.User.ID && b.contextID == session.ContextID && b.csrf == session.CSRF && b.tokenHash == sha256.Sum256([]byte(token))
}

// ForgetBrokerEnrollment removes only bindings belonging to the initiating session.
func (s *Service) ForgetBrokerEnrollment(r *http.Request, session store.Session, id string) {
	cookie, err := RequestCookie(r, SessionCookie)
	if err != nil {
		return
	}
	s.brokerMu.Lock()
	defer s.brokerMu.Unlock()
	for key, binding := range s.brokerBindings {
		if binding.enrollment == id && binding.matches(session, cookie.Value) {
			delete(s.brokerBindings, key)
		}
	}
}

func brokerCallbackQuery(r *http.Request) (url.Values, error) {
	if len(r.URL.RawQuery) > 8192 {
		return nil, identity.ErrDenied
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || !validBrokerCallbackState(query) {
		return nil, identity.ErrDenied
	}
	if !validBrokerCallbackResult(query) {
		return nil, identity.ErrDenied
	}
	return query, nil
}

func validBrokerCallbackState(query url.Values) bool {
	return len(query["state"]) == 1 && query.Get("state") != "" && len(query.Get("state")) <= 128
}

func validBrokerCallbackResult(query url.Values) bool {
	return len(query["code"])+len(query["error"]) == 1 && len(query.Get("code")) <= 4096 && len(query.Get("error")) <= 256 && (query.Get("code") != "" || query.Get("error") != "")
}

func (s *Service) claimBrokerBinding(r *http.Request, session store.Session, state string) (brokerBinding, error) {
	cookie, err := RequestCookie(r, brokerCookie)
	if err != nil {
		return brokerBinding{}, identity.ErrDenied
	}
	token, err := RequestCookie(r, SessionCookie)
	if err != nil {
		return brokerBinding{}, identity.ErrDenied
	}
	s.brokerMu.Lock()
	defer s.brokerMu.Unlock()
	s.pruneBrokerBindings()
	binding, ok := s.brokerBindings[cookie.Value]
	if !ok || binding.state != state || !binding.matches(session, token.Value) {
		return brokerBinding{}, identity.ErrDenied
	}
	delete(s.brokerBindings, cookie.Value)
	return binding, nil
}

func (s *Service) finishBrokerCallback(r *http.Request, session store.Session, binding brokerBinding, query url.Values) error {
	ctx := r.Context()
	if query.Get("error") != "" {
		_ = s.EnrollmentBroker.CancelEnrollment(ctx, session.User.ID, binding.enrollment)
		return identity.ErrDenied
	}
	if _, err := s.EnrollmentBroker.CompleteEnrollment(ctx, session.User.ID, binding.enrollment, binding.state, query.Get("code")); err != nil {
		_ = s.EnrollmentBroker.CancelEnrollment(ctx, session.User.ID, binding.enrollment)
		return identity.ErrDenied
	}
	cookie, err := RequestCookie(r, SessionCookie)
	if err != nil || s.RequireCurrentSession(ctx, cookie.Value, session) != nil {
		_ = s.EnrollmentBroker.CancelEnrollment(ctx, session.User.ID, binding.enrollment)
		return identity.ErrDenied
	}
	retained, err := s.EnrollmentBroker.Enrollment(ctx, session.User.ID, binding.enrollment)
	if err != nil || retained.State != "completed" || retained.Connection == nil {
		return identity.ErrDenied
	}
	return s.RequireCurrentSession(ctx, cookie.Value, session)
}

func (s *Service) brokerCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	ctx, cancel := context.WithTimeout(r.Context(), brokerTimeout)
	defer cancel()
	r = r.WithContext(ctx)
	query, err := brokerCallbackQuery(r)
	session, sessionErr := s.BrowserSession(r)
	if err != nil || sessionErr != nil || s.EnrollmentBroker == nil {
		http.Error(w, "Enrollment could not be completed.", http.StatusForbidden)
		return
	}
	s.withSessionEndGate(func() {
		cookie, cookieErr := RequestCookie(r, SessionCookie)
		if cookieErr != nil || s.RequireCurrentSession(r.Context(), cookie.Value, session) != nil {
			err = identity.ErrDenied
			return
		}
		var binding brokerBinding
		binding, err = s.claimBrokerBinding(r, session, query.Get("state"))
		if err != nil {
			return
		}
		s.cookie(w, brokerCookie, "", -1)
		err = s.finishBrokerCallback(r, session, binding, query)
	})
	if err != nil {
		http.Error(w, "Enrollment could not be completed.", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Forgejo connected. Return to the project settings and refresh connection status.\n"))
}
