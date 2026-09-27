package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// IdentityClient exposes metadata and owner operations, never credential delivery.
type IdentityClient interface {
	Connections(context.Context, int64) ([]identity.Connection, error)
	Available(context.Context, int64, string) ([]identity.Connection, error)
	StartEnrollment(context.Context, int64, string, string) (identity.Enrollment, error)
	Enrollment(context.Context, int64, string) (identity.Enrollment, error)
	CancelEnrollment(context.Context, int64, string) error
	Grants(context.Context, int64, string) ([]identity.Grant, error)
	CreateGrant(context.Context, int64, identity.GrantRequest) (identity.Grant, error)
	RevokeGrant(context.Context, int64, string) error
	Leases(context.Context, int64, string) ([]identity.Lease, error)
	EndLease(context.Context, int64, string) error
	Revoke(context.Context, int64, string) error
}

func (s *API) identityRoutes() {
	s.mux.HandleFunc("/api/environments/{id}/identity/launch", s.Auth.Protected(s.apiIdentityLaunch, "POST"))
	s.mux.HandleFunc("/api/identity/connections", s.Auth.Protected(s.apiIdentityConnections, "GET"))
	s.mux.HandleFunc("/api/identity/enrollments", s.Auth.Protected(s.apiIdentityStartEnrollment, "POST"))
	s.mux.HandleFunc("/api/identity/enrollments/{identityID}", s.Auth.Protected(s.apiIdentityEnrollment, "GET"))
	s.mux.HandleFunc("/api/identity/enrollments/{identityID}/cancel", s.Auth.Protected(s.apiIdentityCancelEnrollment, "POST"))
	s.mux.HandleFunc("/api/identity/connections/{identityID}/grants", s.Auth.Protected(s.apiIdentityGrants, "GET"))
	s.mux.HandleFunc("/api/identity/connections/{identityID}/leases", s.Auth.Protected(s.apiIdentityLeases, "GET"))
	s.mux.HandleFunc("/api/identity/connections/{identityID}/revoke", s.Auth.Protected(s.apiIdentityRevoke, "POST"))
	s.mux.HandleFunc("/api/identity/grants/{identityID}/revoke", s.Auth.Protected(s.apiIdentityRevokeGrant, "POST"))
	s.mux.HandleFunc("/api/identity/leases/{identityID}/end", s.Auth.Protected(s.apiIdentityEndLease, "POST"))
	s.mux.HandleFunc("/api/environments/{id}/identity/grants", s.Auth.Protected(s.apiIdentityCreateGrant, "POST"))
	s.mux.HandleFunc("/api/environments/{id}/identity/connections", s.Auth.Protected(s.apiIdentityAvailable, "GET"))
}

func identityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrDenied):
		auth.JSONError(w, 403, "identity_denied", "This connection operation is not authorized.")
	case errors.Is(err, identity.ErrBusy), errors.Is(err, identity.ErrStale), errors.Is(err, identity.ErrUncertain):
		auth.JSONError(w, 409, "identity_unavailable", "Connection use is unavailable. Refresh status or reconnect before retrying.")
	default:
		auth.JSONError(w, 503, "identity_unavailable", "Identity service did not confirm the operation. Refresh before retrying.")
	}
}

func (s *API) identityAdmission(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return false
	}
	if s.Identity == nil {
		auth.JSONError(w, 503, "identity_unavailable", "Identity service is unavailable.")
		return false
	}
	return s.checkLifecycleSession(w, r, v)
}

func (s *API) identityResult(w http.ResponseWriter, r *http.Request, v store.Session, result any, err error) {
	if err != nil {
		identityError(w, err)
		return
	}
	if s.checkLifecycleSession(w, r, v) {
		auth.JSONResponse(w, 200, result)
	}
}

func (s *API) apiIdentityConnections(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	result, err := s.Identity.Connections(r.Context(), v.User.ID)
	if result == nil {
		result = []identity.Connection{}
	}
	s.identityResult(w, r, v, result, err)
}

func (s *API) apiIdentityStartEnrollment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	var input struct {
		ProviderID                string `json:"provider_id"`
		Label                     string `json:"label"`
		ConfirmCredentialExposure bool   `json:"confirm_credential_exposure"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	input.Label = strings.TrimSpace(input.Label)
	if !identity.ProviderValid(input.ProviderID) || input.Label == "" || len(input.Label) > 80 || !input.ConfirmCredentialExposure {
		auth.JSONError(w, 400, "invalid_enrollment", "Provide a label and confirm appliance trust and credential exposure.")
		return
	}
	result, err := s.Identity.StartEnrollment(r.Context(), v.User.ID, input.ProviderID, input.Label)
	if err == nil {
		err = s.bindIdentityEnrollment(w, r, v, input.ProviderID, result)
	}
	s.identityResult(w, r, v, result, err)
}

func (s *API) bindIdentityEnrollment(w http.ResponseWriter, r *http.Request, v store.Session, providerID string, result identity.Enrollment) error {
	if providerID != identity.Forgejo {
		return nil
	}
	err := s.Auth.BindBrokerEnrollment(w, r, v, result)
	if err != nil {
		_ = s.Identity.CancelEnrollment(r.Context(), v.User.ID, result.ID)
	}
	return err
}

func (s *API) apiIdentityEnrollment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	result, err := s.Identity.Enrollment(r.Context(), v.User.ID, r.PathValue("identityID"))
	s.identityResult(w, r, v, result, err)
}

func (s *API) apiIdentityGrants(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	result, err := s.Identity.Grants(r.Context(), v.User.ID, r.PathValue("identityID"))
	if result == nil {
		result = []identity.Grant{}
	}
	s.identityResult(w, r, v, result, err)
}

func (s *API) apiIdentityLeases(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	result, err := s.Identity.Leases(r.Context(), v.User.ID, r.PathValue("identityID"))
	if result == nil {
		result = []identity.Lease{}
	}
	s.identityResult(w, r, v, result, err)
}

func (s *API) identityMutation(w http.ResponseWriter, r *http.Request, v store.Session, operation func(context.Context, int64, string) error) {
	var input struct{}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	err := operation(r.Context(), v.User.ID, r.PathValue("identityID"))
	s.identityResult(w, r, v, struct {
		Confirmed bool `json:"confirmed"`
	}{err == nil}, err)
}

func (s *API) apiIdentityCancelEnrollment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	s.identityMutation(w, r, v, func(ctx context.Context, owner int64, id string) error {
		err := s.Identity.CancelEnrollment(ctx, owner, id)
		if err == nil {
			s.Auth.ForgetBrokerEnrollment(r, v, id)
		}
		return err
	})
}

func (s *API) apiIdentityRevoke(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	s.identityMutation(w, r, v, s.Identity.Revoke)
}

func (s *API) apiIdentityRevokeGrant(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	s.identityMutation(w, r, v, s.Identity.RevokeGrant)
}

func (s *API) apiIdentityEndLease(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	s.identityMutation(w, r, v, s.Identity.EndLease)
}
