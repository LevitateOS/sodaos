package auth

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/identity"
)

const (
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

func (s *Service) saveBrokerBinding(binding brokerBinding) error {
	s.brokerMu.Lock()
	defer s.brokerMu.Unlock()
	s.pruneBrokerBindings()
	if len(s.brokerBindings) >= 128 {
		return identity.ErrBusy
	}
	if s.brokerBindings == nil {
		s.brokerBindings = make(map[string]brokerBinding)
	}
	if _, exists := s.brokerBindings[binding.state]; exists {
		return identity.ErrBusy
	}
	s.brokerBindings[binding.state] = binding
	return nil
}

func (s *Service) pruneBrokerBindings() {
	for key, binding := range s.brokerBindings {
		if !time.Now().Before(binding.expires) {
			delete(s.brokerBindings, key)
		}
	}
}

// BindNativeBrokerEnrollment binds provider OAuth state to the actor admitted
// by the current native extension request. The state is provider-generated and
// is consumed exactly once by brokerCallback.
func (s *Service) BindNativeBrokerEnrollment(owner int64, enrollment identity.Enrollment) error {
	if s.EnrollmentBroker == nil || enrollment.ID == "" || enrollment.ProviderID != identity.Forgejo {
		return identity.ErrDenied
	}
	if owner <= 0 {
		return identity.ErrDenied
	}
	state, err := s.brokerState(enrollment)
	if err != nil {
		return err
	}
	return s.saveBrokerBinding(brokerBinding{enrollment: enrollment.ID, state: state, owner: owner, expires: time.Now().Add(brokerLifetime)})
}

// ForgetNativeBrokerEnrollment removes only this actor's pending enrollment.
func (s *Service) ForgetNativeBrokerEnrollment(owner int64, id string) {
	s.brokerMu.Lock()
	defer s.brokerMu.Unlock()
	for state, binding := range s.brokerBindings {
		if binding.owner == owner && binding.enrollment == id {
			delete(s.brokerBindings, state)
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

func (s *Service) claimBrokerBinding(state string) (brokerBinding, error) {
	s.brokerMu.Lock()
	defer s.brokerMu.Unlock()
	s.pruneBrokerBindings()
	binding, ok := s.brokerBindings[state]
	if !ok || binding.state != state {
		return brokerBinding{}, identity.ErrDenied
	}
	delete(s.brokerBindings, state)
	return binding, nil
}

func (s *Service) finishBrokerCallback(r *http.Request, binding brokerBinding, query url.Values) error {
	ctx := r.Context()
	if query.Get("error") != "" {
		_ = s.EnrollmentBroker.CancelEnrollment(ctx, binding.owner, binding.enrollment)
		return identity.ErrDenied
	}
	if _, err := s.EnrollmentBroker.CompleteEnrollment(ctx, binding.owner, binding.enrollment, binding.state, query.Get("code")); err != nil {
		_ = s.EnrollmentBroker.CancelEnrollment(ctx, binding.owner, binding.enrollment)
		return identity.ErrDenied
	}
	retained, err := s.EnrollmentBroker.Enrollment(ctx, binding.owner, binding.enrollment)
	if err != nil || retained.State != "completed" || retained.Connection == nil {
		return identity.ErrDenied
	}
	return nil
}

func (s *Service) brokerCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	ctx, cancel := context.WithTimeout(r.Context(), brokerTimeout)
	defer cancel()
	r = r.WithContext(ctx)
	query, err := brokerCallbackQuery(r)
	if err != nil || s.EnrollmentBroker == nil {
		http.Error(w, "Enrollment could not be completed.", http.StatusForbidden)
		return
	}
	binding, err := s.claimBrokerBinding(query.Get("state"))
	if err == nil {
		err = s.finishBrokerCallback(r, binding, query)
	}
	if err != nil {
		http.Error(w, "Enrollment could not be completed.", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Forgejo connected. Return to the project settings and refresh connection status.\n"))
}
