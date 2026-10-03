package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
	"github.com/levitateos/sodaos/internal/web/auth"
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
	errProfileUnavailable = errors.New("profile unavailable")
	errSessionChanged     = errors.New("session changed")
)

func parseCreateEnvironmentInput(w http.ResponseWriter, r *http.Request) (*createEnvironmentInput, int64, bool) {
	var input createEnvironmentInput
	if !auth.DecodeAPIObject(w, r, &input) {
		return nil, 0, false
	}
	repoID, valid := auth.PositiveID(input.RepositoryID)
	if !valid || (input.ProfileID != "" && input.ProfileID != project.RockyHeadless) || (input.Tailnet != nil && input.Tailnet.Validate() != nil) {
		auth.JSONError(w, 400, "invalid_repository", "Provide a repository_id and a supported profile_id.")
		return nil, 0, false
	}
	return &input, repoID, true
}

func (s *API) verifyRepositoryOwner(w http.ResponseWriter, r *http.Request, v store.Session, repoID int64) bool {
	access, err := s.visibleRepository(r, v, repoID)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if access.repository.Owner.ID != access.actor.ID {
		auth.JSONError(w, 403, "owner_required", "Only the human repository owner can create its environment. Organization-owned environments are not supported.")
		return false
	}
	return true
}

// lookupReservation reads one repository's retained reservation. A ready
// reservation stays a conflict; an unready one is reconciled from host
// evidence by the caller instead of being provisioned anew.
func (s *API) lookupReservation(w http.ResponseWriter, ctx context.Context, repoID int64) (store.Project, bool, bool) {
	p, err := s.Store.ProjectByRepository(ctx, repoID)
	if errors.Is(err, store.ErrNotFound) {
		return store.Project{}, false, true
	}
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not inspect reservation.")
		return store.Project{}, false, false
	}
	return p, true, true
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

func (s *API) resolveNativeProfile(ctx context.Context) (project.Profile, error) {
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	profile, err := s.Host.ResolveProfile(tctx)
	if err != nil {
		return project.Profile{}, errProfileUnavailable
	}
	return profile, nil
}

func (s *API) precheckProfileAndTailnet(w http.ResponseWriter, ctx context.Context, sel *tailnet.ProjectSelection) (project.Profile, bool) {
	profile, err := s.resolveNativeProfile(ctx)
	if err != nil {
		auth.JSONError(w, 422, "profile_unavailable", "The native installed Project OS is unavailable or incompatible. No reservation was created.")
		return project.Profile{}, false
	}
	if err := s.checkTailnetPreflight(ctx, sel); err != nil {
		tailnetError(w, err)
		return project.Profile{}, false
	}
	return profile, true
}

func (s *API) verifyCurrentSessionMatch(r *http.Request, v store.Session) error {
	if !s.extensionSessionCurrent(r.Context(), r, v) {
		return errSessionChanged
	}
	return nil
}

func (s *API) reconfirmRepositoryAndSession(w http.ResponseWriter, r *http.Request, v store.Session, repoID int64) (repositoryAccess, bool) {
	access, err := s.visibleRepository(r, v, repoID)
	if err != nil {
		auth.ProviderError(w, err)
		return repositoryAccess{}, false
	}
	if err := s.verifyCurrentSessionMatch(r, v); err != nil {
		auth.JSONError(w, 401, "unauthorized", "Native Forgejo session changed.")
		return repositoryAccess{}, false
	}
	if access.repository.Owner.ID != v.User.ID {
		auth.JSONError(w, 403, "owner_required", "Repository ownership changed.")
		return repositoryAccess{}, false
	}
	return access, true
}

func (s *API) provisionAndSaveProject(w http.ResponseWriter, r *http.Request, p store.Project) (store.Project, bool) {
	opCtx, cancel := operationContext(r.Context(), createOperationTimeout)
	defer cancel()
	env, err := s.Host.Create(opCtx, project.Create{ID: p.ID, Owner: p.OwnerID, Profile: p.Profile})
	if err != nil {
		auth.JSONResponse(w, 502, struct {
			Error       apiError        `json:"error"`
			Environment EnvironmentView `json:"environment"`
		}{apiError{"provisioning_incomplete", "Reservation retained; native provisioning was not confirmed. Inspect this environment with the operator; do not recreate it."}, EnvironmentDTO(p)})
		return p, false
	}
	if err = s.Store.MarkReady(opCtx, p.ID, env.IP); err != nil {
		auth.JSONResponse(w, 503, struct {
			Error       apiError        `json:"error"`
			Environment EnvironmentView `json:"environment"`
		}{apiError{"result_not_saved", "Native provisioning returned, but its result was not saved. Inspect the retained environment; do not recreate it."}, EnvironmentDTO(p)})
		return p, false
	}
	p.Ready = true
	return p, true
}

// reconcileCreate finishes a retained unready reservation from host
// evidence, or reports its truthful unresolved state. Ownership and
// session freshness were already verified by the caller. It never
// deletes or recreates the native root.
func (s *API) reconcileCreate(w http.ResponseWriter, r *http.Request, p store.Project) {
	env, nativeErr := s.Host.Inspect(r.Context(), p.ID)
	observed, nativeErr := observedEnvironment(p, env, nativeErr)
	if observed == nil || !provisioningConfirmed(p, *observed) {
		auth.JSONResponse(w, 409, struct {
			Error             apiError             `json:"error"`
			Environment       EnvironmentView      `json:"environment"`
			Observed          *project.Environment `json:"observed"`
			NativeUnavailable bool                 `json:"native_unavailable"`
		}{apiError{"provisioning_incomplete", "Reservation retained; native provisioning was not confirmed. Inspect this environment with the operator; do not recreate it."}, EnvironmentDTO(p), observed, nativeErr != nil})
		return
	}
	if updated, finished := s.reconcileProvisioning(r.Context(), p, *observed); finished {
		auth.JSONResponse(w, 200, EnvironmentDTO(updated))
		return
	}
	auth.JSONResponse(w, 503, struct {
		Error       apiError        `json:"error"`
		Environment EnvironmentView `json:"environment"`
	}{apiError{"result_not_saved", "Native provisioning was confirmed, but its result was not saved. Inspect the retained environment; do not recreate it."}, EnvironmentDTO(p)})
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
	if !s.verifyRepositoryOwner(w, r, v, repositoryID) {
		return
	}
	existing, found, ok := s.lookupReservation(w, r.Context(), repositoryID)
	if !ok {
		return
	}
	if found && existing.Ready {
		auth.JSONError(w, 409, "reservation_failed", "Repository already has a reservation. Refresh; do not recreate it.")
		return
	}
	if found {
		// Finish the retained reservation from host evidence instead of
		// provisioning anew. Profile and Tailnet prechecks describe a new
		// reservation, not this one; only ownership and session apply.
		if _, ok := s.reconfirmRepositoryAndSession(w, r, v, repositoryID); !ok {
			return
		}
		s.reconcileCreate(w, r, existing)
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
	_, _ = rand.Read(bytes)
	p := store.Project{Profile: &profile, ID: "p" + hex.EncodeToString(bytes), Name: repo.Name, RepositoryID: repo.ID, OwnerID: v.User.ID, Repository: repo.FullName}
	if err := s.Store.CreateProject(r.Context(), p); err != nil {
		// Lost a concurrent creation race: the winner owns the
		// reservation. Reconcile it when it is still unready.
		if winner, rerr := s.Store.ProjectByRepository(r.Context(), repositoryID); rerr == nil && !winner.Ready {
			s.reconcileCreate(w, r, winner)
			return
		}
		auth.JSONError(w, 409, "reservation_failed", "Repository may already have an environment reservation. Refresh its environment before retrying.")
		return
	}
	p, ok = s.provisionAndSaveProject(w, r, p)
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
	auth.JSONResponse(w, 201, result)
}
