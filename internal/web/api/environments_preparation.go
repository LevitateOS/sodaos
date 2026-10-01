package api

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// preparationItemView reports one durable preparation without its setup and
// check logs; operators inspect logs through the host preparation identity.
type preparationItemView struct {
	ID           string                 `json:"id"`
	Role         string                 `json:"role"`
	Phase        string                 `json:"phase"`
	Ready        bool                   `json:"ready"`
	Stopped      bool                   `json:"stopped"`
	SourceCommit string                 `json:"source_commit"`
	SetupDigest  string                 `json:"setup_digest"`
	Missing      string                 `json:"missing,omitempty"`
	SetupExit    *int                   `json:"setup_exit,omitempty"`
	CheckExit    *int                   `json:"check_exit,omitempty"`
	Tools        []project.ResolvedTool `json:"tools,omitempty"`
}

type environmentPreparationView struct {
	Project      string                `json:"project"`
	Hold         bool                  `json:"hold"`
	HoldRevision int64                 `json:"hold_revision"`
	Requirements string                `json:"requirements,omitempty"`
	Approval     string                `json:"approval,omitempty"`
	Preparations []preparationItemView `json:"preparations"`
}

func preparationItemDTO(p project.StoredPreparation) preparationItemView {
	return preparationItemView{
		ID: p.Preparation.ID, Role: p.Preparation.Role,
		Phase: p.State.Phase, Ready: p.State.Ready, Stopped: p.State.Stopped,
		SourceCommit: p.Preparation.SourceCommit, SetupDigest: p.Preparation.SetupDigest,
		Missing: p.State.Missing, SetupExit: p.State.SetupExit, CheckExit: p.State.CheckExit,
		Tools: p.State.Tools,
	}
}

func (s *API) apiPreparation(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	if r.Method == "POST" {
		s.apiPreparationHold(w, r, v, p)
		return
	}
	if _, allowed := s.authorizeEnvironmentRead(w, r, v, p); !allowed {
		return
	}
	held, revision := s.preparationHoldState(w, r, p.ID)
	if revision < 0 {
		return
	}
	items, ok := s.preparationItems(w, r, p.ID)
	if !ok {
		return
	}
	view := environmentPreparationView{
		Project: p.ID, Hold: held, HoldRevision: revision, Preparations: items,
	}
	if head, err := s.Store.RequirementHead(r.Context(), p.ID); err == nil {
		view.Requirements = head
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read requirement acceptance.")
		return
	}
	if head, err := s.Store.ApprovalHead(r.Context(), p.ID); err == nil {
		view.Approval = head
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read privileged-effect approval.")
		return
	}
	auth.JSONResponse(w, 200, view)
}

func (s *API) preparationHoldState(w http.ResponseWriter, r *http.Request, projectID string) (bool, int64) {
	held, err := s.Store.MaintenanceHold(r.Context(), projectID)
	if errors.Is(err, store.ErrNotFound) {
		return false, 0
	}
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read maintenance hold.")
		return false, -1
	}
	return held.Hold, held.Revision
}

func (s *API) preparationItems(w http.ResponseWriter, r *http.Request, projectID string) ([]preparationItemView, bool) {
	records, err := s.Store.ProjectPreparations(r.Context(), projectID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read preparations.")
		return nil, false
	}
	items := []preparationItemView{}
	for _, record := range records {
		items = append(items, preparationItemDTO(record))
	}
	return items, true
}

type preparationHoldRequest struct {
	Action string `json:"action"`
}

func (s *API) apiPreparationHold(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) {
	var in preparationHoldRequest
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if in.Action != "hold" && in.Action != "release" {
		auth.JSONError(w, 400, "invalid_action", "Choose hold or release.")
		return
	}
	if !s.authorizeLifecycleOperator(w, r, v, p.RepositoryID) {
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	held, revision := s.preparationHoldState(w, r, p.ID)
	if revision < 0 {
		return
	}
	want := in.Action == "hold"
	if held == want {
		// Re-sync the marker at the recorded revision without advancing it.
		if _, err := s.Host.HoldPreparation(r.Context(), project.PrepareHold{Project: p.ID, Hold: want, Revision: revision}); err != nil {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native marker was not confirmed; refresh and retry.")
			return
		}
		auth.JSONResponse(w, 200, struct {
			Hold         bool  `json:"hold"`
			HoldRevision int64 `json:"hold_revision"`
			MarkerSynced bool  `json:"marker_synced"`
		}{held, revision, true})
		return
	}
	if !s.syncPreparationHold(w, r, p.ID, want, revision) {
		return
	}
	auth.JSONResponse(w, 200, struct {
		Hold         bool  `json:"hold"`
		HoldRevision int64 `json:"hold_revision"`
		MarkerSynced bool  `json:"marker_synced"`
	}{want, revision + 1, true})
}

// syncPreparationHold applies a hold change with the native marker first so a
// partial failure never records enforcement that does not exist. Holding sets
// the next revision; releasing addresses the recorded hold revision and
// refuses while preparations run. Both directions are retry-safe; a 409 means
// another administrator changed the hold first.
func (s *API) syncPreparationHold(w http.ResponseWriter, r *http.Request, projectID string, want bool, revision int64) bool {
	marker := revision
	if want {
		marker = revision + 1
	}
	if _, err := s.Host.HoldPreparation(r.Context(), project.PrepareHold{Project: projectID, Hold: want, Revision: marker}); err != nil {
		if want {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native hold was not confirmed; preparation stays admitted. Retry before maintaining.")
		} else {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native hold release was not confirmed; preparation stays denied. Stop running preparations and retry.")
		}
		return false
	}
	if err := s.Store.SaveMaintenanceHold(r.Context(), project.MaintenanceHold{Project: projectID, Revision: revision, Hold: want}); err != nil {
		auth.JSONError(w, 409, "stale_hold", "Hold changed while applying; refresh and retry.")
		return false
	}
	return true
}

// spacePreparationRole summarizes one role's preparations without ordering
// claims; records carry no timestamps.
type spacePreparationRole struct {
	Role         string   `json:"role"`
	Preparations int      `json:"preparations"`
	Ready        bool     `json:"ready"`
	Phases       []string `json:"phases"`
}

// spacePreparationView summarizes maintenance and role readiness for Spaces.
type spacePreparationView struct {
	Hold  bool                   `json:"hold"`
	Roles []spacePreparationRole `json:"roles"`
}

func summarizeSpacePreparation(records []project.StoredPreparation, hold bool) *spacePreparationView {
	view := &spacePreparationView{Hold: hold, Roles: []spacePreparationRole{}}
	byRole := map[string]*spacePreparationRole{}
	for _, record := range records {
		role := byRole[record.Preparation.Role]
		if role == nil {
			role = &spacePreparationRole{Role: record.Preparation.Role, Phases: []string{}}
			byRole[record.Preparation.Role] = role
		}
		role.Preparations++
		role.Ready = role.Ready || record.State.Ready
		seen := false
		for _, phase := range role.Phases {
			seen = seen || phase == record.State.Phase
		}
		if !seen {
			role.Phases = append(role.Phases, record.State.Phase)
		}
	}
	for _, role := range byRole {
		sort.Strings(role.Phases)
		view.Roles = append(view.Roles, *role)
	}
	sort.Slice(view.Roles, func(i, j int) bool { return view.Roles[i].Role < view.Roles[j].Role })
	return view
}

func (s *API) inspectSpacePreparation(ctx context.Context, projectID string) *spacePreparationView {
	records, err := s.Store.ProjectPreparations(ctx, projectID)
	if err != nil {
		return nil
	}
	held, err := s.Store.MaintenanceHold(ctx, projectID)
	hold := err == nil && held.Hold
	return summarizeSpacePreparation(records, hold)
}
