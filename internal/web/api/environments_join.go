package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func validJoinSSHSelection(selection string) bool {
	return selection == "" || selection == "saved" || selection == "none"
}

func (s *API) joinPublicKeys(ctx context.Context, userID int64, selection string) ([]string, error) {
	if selection == "none" {
		return []string{}, nil
	}
	keys, err := s.Store.Keys(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(keys) > 32 {
		return nil, errTooManyJoinKeys
	}
	public := make([]string, 0, len(keys))
	for _, key := range keys {
		public = append(public, key.Public)
	}
	return public, nil
}

var errTooManyJoinKeys = errors.New("too many development keys")

func (s *API) persistEnvironmentJoin(ctx context.Context, r *http.Request, v store.Session, p store.Project, login string, public []string) error {
	if !s.extensionSessionCurrent(ctx, r, v) {
		return errJoinUnauthorized
	}
	if err := s.Host.Join(ctx, project.Account{Project: p.ID, Login: login, Identity: v.User.ID, Keys: public}); err != nil {
		return errJoinAccountIncomplete
	}
	if err := s.Store.Join(ctx, p.ID, v.User.ID, login); err != nil {
		return errJoinMembershipNotSaved
	}
	return nil
}

var (
	errJoinUnauthorized       = errors.New("join unauthorized")
	errJoinAccountIncomplete  = errors.New("join account incomplete")
	errJoinMembershipNotSaved = errors.New("join membership not saved")
)

func (s *API) writeJoinLogin(w http.ResponseWriter, login string) {
	auth.JSONResponse(w, 200, struct {
		Login string `json:"login"`
	}{login})
}

func (s *API) reportJoinPersist(w http.ResponseWriter, login string, err error) {
	switch err {
	case errJoinUnauthorized:
		auth.JSONError(w, 401, "unauthorized", "Soda context changed. Reconnect before acting.")
	case errJoinAccountIncomplete:
		auth.JSONError(w, 502, "account_incomplete", "Native account provisioning was not confirmed. Membership was not recorded; ask the operator to inspect the account.")
	case errJoinMembershipNotSaved:
		auth.JSONError(w, 503, "membership_not_saved", "Native account provisioning returned but membership could not be saved. Ask the operator to inspect the retained account.")
	case nil:
		s.writeJoinLogin(w, login)
	}
}

func (s *API) admitNewJoin(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, access repositoryAccess, sshKeys string) (login string, public []string, ok bool) {
	if !p.Ready {
		auth.JSONError(w, 409, "not_provisioned", "Environment provisioning is incomplete.")
		return "", nil, false
	}
	login = access.actor.Login
	if !project.ValidLogin(login) || login == "root" {
		auth.JSONError(w, 422, "unsupported_linux_login", "Your Forgejo username is not supported as a project Linux account. No automatic rename is performed.")
		return "", nil, false
	}
	// Empty legacy requests retain saved-key behavior. New browser callers can
	// explicitly choose account-only provisioning even when saved SSH keys exist.
	public, err := s.joinPublicKeys(r.Context(), v.User.ID, sshKeys)
	if errors.Is(err, errTooManyJoinKeys) {
		auth.JSONError(w, 422, "too_many_keys", "Native onboarding supports at most 32 development keys.")
		return "", nil, false
	}
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read development keys.")
		return "", nil, false
	}
	return login, public, true
}

func (s *API) apiJoinEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	var input struct {
		SSHKeys string `json:"ssh_keys"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if !validJoinSSHSelection(input.SSHKeys) {
		auth.JSONError(w, 400, "invalid_ssh_selection", "Select saved or none for optional external SSH keys.")
		return
	}
	access, err := s.executionRepository(r, v, p.RepositoryID)
	if err != nil {
		reportExecutionAuthorityError(w, err)
		return
	}
	if !s.currentJoinSession(r, v) {
		auth.ProviderError(w, auth.ErrNativeSessionChanged)
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err == nil {
		s.writeJoinLogin(w, login)
		return
	}
	if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not inspect membership.")
		return
	}
	login, public, ok := s.admitNewJoin(w, r, v, p, access, input.SSHKeys)
	if !ok {
		return
	}
	// Fresh admission after provider/state I/O, not rollback after dispatch.
	s.reportJoinPersist(w, login, s.persistEnvironmentJoin(r.Context(), r, v, p, login, public))
}

func (s *API) currentJoinSession(r *http.Request, v store.Session) bool {
	return s.extensionSessionCurrent(r.Context(), r, v)
}
