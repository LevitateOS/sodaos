package webapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
	"github.com/levitateos/sodaos/internal/webauth"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type createEnvironmentInput struct {
	RepositoryID string                    `json:"repository_id"`
	ProfileID    string                    `json:"profile_id"`
	Tailnet      *tailnet.ProjectSelection `json:"tailnet,omitempty"`
}

var (
	errStoreUnavailable     = errors.New("could not inspect reservation")
	errReservationFailed    = errors.New("repository already has a reservation")
	errProfileUnavailable   = errors.New("profile unavailable")
	errSessionCookieMissing = errors.New("session cookie missing")
	errSessionChanged       = errors.New("session changed")
)

func parseCreateEnvironmentInput(w http.ResponseWriter, r *http.Request) (*createEnvironmentInput, int64, bool) {
	var input createEnvironmentInput
	if !webauth.DecodeAPIObject(w, r, &input) {
		return nil, 0, false
	}
	repoID, valid := webauth.PositiveID(input.RepositoryID)
	if !valid || (input.ProfileID != "" && input.ProfileID != projectos.RockyHeadless) || (input.Tailnet != nil && input.Tailnet.Validate() != nil) {
		webauth.JSONError(w, 400, "invalid_repository", "Provide a repository_id and a supported profile_id.")
		return nil, 0, false
	}
	return &input, repoID, true
}

func (s *API) checkRepositoryReservation(ctx context.Context, repoID int64) error {
	_, err := s.Store.ProjectByRepository(ctx, repoID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return errStoreUnavailable
	}
	return errReservationFailed
}

func (s *API) precheckRepositoryAndReservation(w http.ResponseWriter, r *http.Request, v store.Session, repoID int64) bool {
	access, err := s.visibleRepository(r, v, repoID)
	if err != nil {
		webauth.ProviderError(w, err)
		return false
	}
	if access.repository.Owner.ID != access.actor.ID {
		webauth.JSONError(w, 403, "owner_required", "Only the human repository owner can create its environment. Organization-owned environments are not supported.")
		return false
	}
	if err := s.checkRepositoryReservation(r.Context(), repoID); err != nil {
		if errors.Is(err, errReservationFailed) {
			webauth.JSONError(w, 409, "reservation_failed", "Repository already has a reservation. Refresh; do not recreate it.")
		} else {
			webauth.JSONError(w, 503, "store_unavailable", "Could not inspect reservation.")
		}
		return false
	}
	return true
}

func (s *API) checkTailnetPreflight(ctx context.Context, sel *tailnet.ProjectSelection) error {
	if sel == nil || !sel.Enabled {
		return nil
	}
	options, err := s.Host.TailnetOptions(ctx)
	if err != nil {
		return err
	}
	if !options.Available {
		return tailnet.ErrUnsupported
	}
	if options.Revision != sel.Revision || options.Binding != sel.Binding {
		return tailnet.ErrConflict
	}
	return nil
}

func (s *API) resolveNativeProfile(ctx context.Context) (projectos.Profile, error) {
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	profile, err := s.Host.ResolveProfile(tctx)
	if err != nil {
		return projectos.Profile{}, errProfileUnavailable
	}
	return profile, nil
}

func (s *API) precheckProfileAndTailnet(w http.ResponseWriter, ctx context.Context, sel *tailnet.ProjectSelection) (projectos.Profile, bool) {
	profile, err := s.resolveNativeProfile(ctx)
	if err != nil {
		webauth.JSONError(w, 422, "profile_unavailable", "The native installed Project OS is unavailable or incompatible. No reservation was created.")
		return projectos.Profile{}, false
	}
	if err := s.checkTailnetPreflight(ctx, sel); err != nil {
		tailnetError(w, err)
		return projectos.Profile{}, false
	}
	return profile, true
}

func (s *API) verifyCurrentSessionMatch(r *http.Request, v store.Session) error {
	cookie, err := webauth.RequestCookie(r, webauth.SessionCookie)
	if err != nil {
		return errSessionCookieMissing
	}
	current, err := s.Store.Session(r.Context(), cookie.Value)
	if err != nil || current.ContextID != v.ContextID || current.CSRF != v.CSRF || current.User.ID != v.User.ID {
		return errSessionChanged
	}
	return nil
}

func (s *API) reconfirmRepositoryAndSession(w http.ResponseWriter, r *http.Request, v store.Session, repoID int64) (repositoryAccess, bool) {
	access, err := s.visibleRepository(r, v, repoID)
	if err != nil {
		webauth.ProviderError(w, err)
		return repositoryAccess{}, false
	}
	if err := s.verifyCurrentSessionMatch(r, v); err != nil {
		if errors.Is(err, errSessionCookieMissing) {
			webauth.JSONError(w, 401, "unauthorized", "Reconnect to Soda.")
		} else {
			webauth.JSONError(w, 401, "unauthorized", "Soda context changed.")
		}
		return repositoryAccess{}, false
	}
	if access.repository.Owner.ID != v.User.ID {
		webauth.JSONError(w, 403, "owner_required", "Repository ownership changed.")
		return repositoryAccess{}, false
	}
	return access, true
}

func (s *API) provisionAndSaveProject(w http.ResponseWriter, ctx context.Context, p store.Project) (store.Project, bool) {
	w.Header().Set("Location", config.SodaPath+"/api/environments/"+p.ID)
	env, err := s.Host.Create(ctx, host.Create{ID: p.ID, Owner: p.OwnerID, Profile: p.Profile})
	if err != nil {
		webauth.JSONResponse(w, 502, struct {
			Error       apiError        `json:"error"`
			Environment EnvironmentView `json:"environment"`
		}{apiError{"provisioning_incomplete", "Reservation retained; native provisioning was not confirmed. Inspect this environment with the operator; do not recreate it."}, EnvironmentDTO(p)})
		return p, false
	}
	if err = s.Store.MarkReady(ctx, p.ID, env.IP); err != nil {
		webauth.JSONResponse(w, 503, struct {
			Error       apiError        `json:"error"`
			Environment EnvironmentView `json:"environment"`
		}{apiError{"result_not_saved", "Native provisioning returned, but its result was not saved. Inspect the retained environment; do not recreate it."}, EnvironmentDTO(p)})
		return p, false
	}
	p.Ready = true
	return p, true
}

func (s *API) applyCreatedEnvironmentTailnet(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, sel *tailnet.ProjectSelection) (string, bool) {
	if sel == nil || !sel.Enabled {
		return "", true
	}
	if !s.authorizeProjectTailnet(w, r, v, p) || !s.tailnetSession(w, r, v) {
		return "", false
	}
	network, err := s.Host.TailnetProject(r.Context(), tailnet.ProjectRequest{
		Project: p.ID, Action: "enable", Revision: "0", Binding: sel.Binding, ConfirmID: p.ID,
	})
	if !s.tailnetSession(w, r, v) || !s.authorizeProjectTailnet(w, r, v, p) {
		return "", false
	}
	if err == nil && network.Outcome == "queued" {
		return "queued", true
	}
	return "unconfirmed", true
}

func (s *API) apiCreateEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	input, repositoryID, ok := parseCreateEnvironmentInput(w, r)
	if !ok {
		return
	}
	if !s.precheckRepositoryAndReservation(w, r, v, repositoryID) {
		return
	}
	profile, ok := s.precheckProfileAndTailnet(w, r.Context(), input.Tailnet)
	if !ok {
		return
	}
	access, ok := s.reconfirmRepositoryAndSession(w, r, v, repositoryID)
	if !ok {
		return
	}
	repo := access.repository
	bytes := make([]byte, 12)
	rand.Read(bytes)
	p := store.Project{Profile: &profile, ID: "p" + hex.EncodeToString(bytes), Name: repo.Name, RepositoryID: repo.ID, OwnerID: v.User.ID, Repository: repo.FullName}
	if err := s.Store.CreateProject(r.Context(), p); err != nil {
		webauth.JSONError(w, 409, "reservation_failed", "Repository may already have an environment reservation. Refresh its environment before retrying.")
		return
	}
	p, ok = s.provisionAndSaveProject(w, r.Context(), p)
	if !ok {
		return
	}
	tailnetOutcome, ok := s.applyCreatedEnvironmentTailnet(w, r, v, p, input.Tailnet)
	if !ok {
		return
	}
	result := struct {
		EnvironmentView
		TailnetOutcome string `json:"tailnet_outcome,omitempty"`
	}{EnvironmentView: EnvironmentDTO(p), TailnetOutcome: tailnetOutcome}
	webauth.JSONResponse(w, 201, result)
}
