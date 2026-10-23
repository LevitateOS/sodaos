package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type factoryRepairGrantRequest struct {
	CommandID         string `json:"command_id"`
	ExpectedRevision  int64  `json:"expected_revision"`
	InvestigatorRuns  int    `json:"investigator_runs"`
	ProposalJudgments int    `json:"proposal_judgments"`
	ActiveMinutes     int    `json:"active_minutes"`
	Effect            string `json:"effect"`
	Active            bool   `json:"active"`
}

// Repair funding is an explicit grant from the authenticated sponsor. It never
// manufactures broker membership, a subscription grant, or execution authority.
func (s *API) apiFactoryRepair(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	repository, ok := s.factoryRepository(w, r, v)
	if !ok || !s.factoryOwner(w, r, v, repository) {
		return
	}
	if r.Method == http.MethodGet {
		grant, err := s.Store.RepairGrantCommand(r.Context(), repository)
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONResponse(w, 200, map[string]any{"revision": 0, "grant": nil, "charges": []factory.RepairCharge{}, "episodes": []factory.RepairEpisodeView{}})
			return
		}
		if err != nil {
			factoryCommandError(w, err)
			return
		}
		charges, err := s.Store.RepairCharges(r.Context(), repository)
		if err != nil {
			factoryCommandError(w, err)
			return
		}
		episodes, err := s.Store.RepairEpisodeViews(r.Context(), repository)
		if err != nil {
			factoryCommandError(w, err)
			return
		}
		auth.JSONResponse(w, 200, map[string]any{"revision": grant.Revision + 1, "grant": grant, "charges": charges, "episodes": episodes})
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factoryRepairGrantRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	grant := factory.RepairGrant{ID: in.CommandID, Repository: repository, Sponsor: v.User.ID, InvestigatorRuns: in.InvestigatorRuns, ProposalJudgments: in.ProposalJudgments, ActiveMinutes: in.ActiveMinutes, Effect: in.Effect, Active: in.Active, CreatedUnix: time.Now().Unix()}
	if in.ExpectedRevision < 0 {
		auth.JSONError(w, 400, "invalid_repair_grant", "Repair revision must be nonnegative.")
		return
	}
	priorCommand, commandErr := s.Store.FactoryCommand(r.Context(), in.CommandID)
	if commandErr == nil {
		var prior factory.RepairGrant
		if priorCommand.Type != factory.CommandRepairGrant || priorCommand.Target != factory.RepairTarget(repository) || priorCommand.Principal != factoryPrincipal(v) || json.Unmarshal([]byte(priorCommand.Payload), &prior) != nil {
			factoryCommandError(w, store.ErrCommandConflict)
			return
		}
		grant.CreatedUnix, grant.Predecessor, grant.Revision = prior.CreatedUnix, prior.Predecessor, prior.Revision
		if in.ExpectedRevision != grant.Revision {
			factoryCommandError(w, store.ErrCommandConflict)
			return
		}
	} else if !errors.Is(commandErr, store.ErrNotFound) {
		factoryCommandError(w, commandErr)
		return
	} else {
		current, err := s.Store.RepairGrantCommand(r.Context(), repository)
		if err == nil {
			if in.ExpectedRevision != current.Revision+1 {
				factoryCommandError(w, store.ErrStaleRevision)
				return
			}
			if (grant.InvestigatorRuns > current.InvestigatorRuns || grant.ProposalJudgments > current.ProposalJudgments || grant.ActiveMinutes > current.ActiveMinutes) && grant.Effect == "" {
				auth.JSONError(w, 400, "invalid_repair_grant", "An explicit repair limit increase must state its effect and preserves prior consumption.")
				return
			}
			grant.Revision, grant.Predecessor = current.Revision+1, current.ID
		} else if !errors.Is(err, store.ErrNotFound) {
			factoryCommandError(w, err)
			return
		} else if in.ExpectedRevision != 0 {
			factoryCommandError(w, store.ErrStaleRevision)
			return
		}
	}
	if grant.Validate() != nil {
		auth.JSONError(w, 400, "invalid_repair_grant", "Repair limits exceed the finite repository allowance or are invalid.")
		return
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	command := factory.Command{ID: in.CommandID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(repository), Principal: factoryPrincipal(v), Payload: string(raw)}
	command.Digest = factory.SettingsDigest(command.Type, command.Target, command.Payload)
	recorded, _, err := s.Store.ApplyRepairGrantCommand(r.Context(), command, grant, time.Now())
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	var receipt factory.GrantReceipt
	if json.Unmarshal([]byte(recorded.Outcome), &receipt) != nil {
		factoryCommandError(w, errors.New("repair receipt unavailable"))
		return
	}
	auth.JSONResponse(w, 200, receipt)
}
