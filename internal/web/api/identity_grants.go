package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) identityProjectMember(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) bool {
	if !p.Ready {
		auth.JSONError(w, 409, "not_provisioned", "Project provisioning is incomplete.")
		return false
	}
	if !s.identityNamedMember(w, r, p, v.User.ID) {
		return false
	}
	if _, err := s.executionRepository(r, v, p.RepositoryID); err != nil {
		reportExecutionAuthorityError(w, err)
		return false
	}
	return true
}

func (s *API) identityNamedMember(w http.ResponseWriter, r *http.Request, p store.Project, userID int64) bool {
	login, err := s.Store.MemberLogin(r.Context(), p.ID, userID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not confirm the project member.")
		return false
	}
	if err != nil || login == "root" || !projectLogin.MatchString(login) {
		auth.JSONError(w, 403, "membership_required", "Subscription use requires a named provisioned member of this project.")
		return false
	}
	return true
}

func (s *API) apiIdentityCreateGrant(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	var input struct {
		ConnectionID              string `json:"connection_id"`
		UserID                    string `json:"user_id"`
		ConfirmSubscription       bool   `json:"confirm_subscription"`
		ConfirmCredentialExposure bool   `json:"confirm_credential_exposure"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	userID, valid := auth.PositiveID(input.UserID)
	in := identity.GrantRequest{
		ConnectionID: input.ConnectionID, UserID: userID,
		ProjectID: r.PathValue("id"), ConfirmSubscription: input.ConfirmSubscription,
		ConfirmCredentialExposure: input.ConfirmCredentialExposure,
	}
	if !valid || in.Validate() != nil {
		auth.JSONError(w, 400, "invalid_identity_grant", "Name a project member and confirm provider permission and credential exposure.")
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok || !s.identityProjectMember(w, r, v, p) || !s.identityNamedMember(w, r, p, userID) {
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	result, err := s.Identity.CreateGrant(r.Context(), v.User.ID, in)
	s.identityResult(w, r, v, result, err)
}

func (s *API) apiIdentityAvailable(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok || !s.identityProjectMember(w, r, v, p) {
		return
	}
	result, err := s.Identity.Available(r.Context(), v.User.ID, p.ID)
	if result == nil {
		result = []identity.Connection{}
	}
	s.identityResult(w, r, v, result, err)
}
