package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type preparationAcceptanceRequest struct {
	CommandID    string `json:"command_id"`
	DecisionID   string `json:"decision_id"`
	Predecessor  string `json:"predecessor,omitempty"`
	SourceCommit string `json:"source_commit"`
	SetupDigest  string `json:"setup_digest"`
	InputsDigest string `json:"inputs_digest"`
}

// apiPreparationAcceptances records the maintainer's exact-inputs acceptance
// of environment requirements. Current code-write authority admits it; the
// decision grants neither Project root nor host execution.
func (s *API) apiPreparationAcceptances(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if _, err := s.executionRepository(r, v, p.RepositoryID); err != nil {
		reportExecutionAuthorityError(w, err)
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in preparationAcceptanceRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	decision := project.RequirementDecision{
		ID: in.DecisionID, Predecessor: in.Predecessor, Project: p.ID,
		Approver: v.User.ID, SourceCommit: in.SourceCommit, SetupDigest: in.SetupDigest, InputsDigest: in.InputsDigest,
	}
	if decision.Validate() != nil {
		auth.JSONError(w, 400, "invalid_acceptance", "Requirement acceptance must name its exact approved inputs.")
		return
	}
	receipt, err := s.Coordinator.AdmitRequirement(r.Context(), in.CommandID, factoryPrincipal(v), decision)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, receipt)
}

type preparationActionRequest struct {
	CommandID       string `json:"command_id,omitempty"`
	Action          string `json:"action"`
	DecisionID      string `json:"decision_id,omitempty"`
	Predecessor     string `json:"predecessor,omitempty"`
	Requirement     string `json:"requirement,omitempty"`
	EffectsDigest   string `json:"effects_digest,omitempty"`
	ReadinessDigest string `json:"readiness_digest,omitempty"`
	Verified        bool   `json:"verified,omitempty"`
}

type preparationInspectView struct {
	Project      string `json:"project"`
	Hold         bool   `json:"hold"`
	HoldRevision int64  `json:"hold_revision"`
	Requirements string `json:"requirements,omitempty"`
	Approval     string `json:"approval,omitempty"`
}

// apiPreparationActions coordinates preparation holds, inspection and
// privileged-effect approval. Approval records reviewed effects without
// running installation commands through HTTP.
func (s *API) apiPreparationActions(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	var in preparationActionRequest
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	switch in.Action {
	case "hold":
		s.apiPreparationActionHold(w, r, v, p)
	case "inspect":
		s.apiPreparationActionInspect(w, r, v, p)
	case "approve":
		s.apiPreparationActionApprove(w, r, v, p, in)
	default:
		auth.JSONError(w, 400, "invalid_action", "Choose hold, inspect or approve.")
	}
}

func (s *API) apiPreparationActionHold(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) {
	if !s.authorizeLifecycleOperator(w, r, v, p.RepositoryID) || !s.checkLifecycleSession(w, r, v) {
		return
	}
	held, revision := s.preparationHoldState(w, r, p.ID)
	if revision < 0 {
		return
	}
	if held {
		if _, err := s.Host.HoldPreparation(r.Context(), project.PrepareHold{Project: p.ID, Hold: true, Revision: revision}); err != nil {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native marker was not confirmed; refresh and retry.")
			return
		}
	} else if !s.syncPreparationHold(w, r, p.ID, true, revision) {
		return
	} else {
		revision++
	}
	auth.JSONResponse(w, 200, struct {
		Hold         bool  `json:"hold"`
		HoldRevision int64 `json:"hold_revision"`
	}{true, revision})
}

func (s *API) apiPreparationActionInspect(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) {
	if _, allowed := s.authorizeEnvironmentRead(w, r, v, p); !allowed {
		return
	}
	held, revision := s.preparationHoldState(w, r, p.ID)
	if revision < 0 {
		return
	}
	view := preparationInspectView{Project: p.ID, Hold: held, HoldRevision: revision}
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

func (s *API) apiPreparationActionApprove(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, in preparationActionRequest) {
	if !settingsCommandID(w, in.CommandID) {
		return
	}
	decision := project.ApprovalDecision{
		ID: in.DecisionID, Predecessor: in.Predecessor, Project: p.ID,
		Requirement: in.Requirement, Approver: v.User.ID,
		EffectsDigest: in.EffectsDigest, ReadinessDigest: in.ReadinessDigest, Verified: in.Verified,
	}
	if decision.Validate() != nil {
		auth.JSONError(w, 400, "invalid_approval", "Approval must bind the current requirement and reviewed effects.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if v.User.ID != s.Config.OperatorID {
		login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 403, "project_membership_required", "Current project membership is required for privileged-effect approval.")
			return
		}
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Could not verify current project membership.")
			return
		}
		status, err := s.Host.ProjectAccess(r.Context(), project.ProjectAccessRequest{
			Project: p.ID, Login: login, Identity: v.User.ID,
		})
		if err != nil {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Current native project authority was not confirmed.")
			return
		}
		if status.Project != p.ID || status.Login != login || status.Identity != v.User.ID {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Current native project authority was not confirmed.")
			return
		}
		if status.Administrator == nil {
			auth.JSONError(w, 502, "native_outcome_unconfirmed", "Current native project authority was not confirmed.")
			return
		}
		if !*status.Administrator {
			auth.JSONError(w, 403, "project_administrator_required", "Current native project administrator authority is required.")
			return
		}
		// The native observation may take time. Recheck the admitting session
		// immediately before recording the approval receipt.
		if !s.checkLifecycleSession(w, r, v) {
			return
		}
	}
	receipt, err := s.Coordinator.AdmitApproval(r.Context(), in.CommandID, factoryPrincipal(v), decision)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, receipt)
}
