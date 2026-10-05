package api

import (
	"net/http"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type factoryPolicyRequest struct {
	Roles            map[string]factory.RoleSelection `json:"roles"`
	PublishActor     actorRefRequest                  `json:"publish_actor"`
	CreateActor      actorRefRequest                  `json:"create_actor"`
	ReviewActor      actorRefRequest                  `json:"review_actor"`
	MergeActor       actorRefRequest                  `json:"merge_actor"`
	CommandID        string                           `json:"command_id"`
	TargetBranch     string                           `json:"target_branch"`
	MergeMethod      string                           `json:"merge_method"`
	RequiredChecks   []string                         `json:"required_checks"`
	ExpectedRevision int64                            `json:"expected_revision"`
	MaxConcurrent    int                              `json:"max_concurrent"`
	Enabled          bool                             `json:"enabled"`
	Paused           bool                             `json:"paused"`
}

func (s *API) apiFactoryPolicy(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok || !s.factoryOwner(w, r, v, repository) || !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factoryPolicyRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	publish, ok := in.PublishActor.resolve(factory.OpRefPublish)
	if !ok {
		auth.JSONError(w, 400, "invalid_actor_binding", "Publisher must reference an enrolled native publication binding.")
		return
	}
	create, ok := in.CreateActor.resolve(factory.OpPRCreate)
	if !ok {
		auth.JSONError(w, 400, "invalid_actor_binding", "Creator must reference an enrolled native PR creation binding.")
		return
	}
	review, ok := in.ReviewActor.resolve(factory.OpReviewSubmit)
	if !ok {
		auth.JSONError(w, 400, "invalid_actor_binding", "Reviewer must reference an enrolled native review binding.")
		return
	}
	merge, ok := in.MergeActor.resolve(factory.OpMerge)
	if !ok {
		auth.JSONError(w, 400, "invalid_actor_binding", "Merger must reference an enrolled native merge binding.")
		return
	}
	policy := factory.RepositoryPolicy{
		Repository: repository, Revision: in.ExpectedRevision, GrantedBy: v.User.ID,
		Enabled: in.Enabled, Paused: in.Paused, TargetBranch: in.TargetBranch, Roles: in.Roles,
		Checks: in.RequiredChecks, MergeMethod: in.MergeMethod,
		Publish: publish, Create: create, Review: review, Merge: merge, MaxConcurrent: in.MaxConcurrent,
	}
	if policy.Revision < 0 || policy.Validate() != nil {
		auth.JSONError(w, 400, "invalid_policy", "Repository policy is incomplete or invalid.")
		return
	}
	receipt, err := s.Coordinator.ApplyPolicy(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, policy)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

type factoryCapacityRequest struct {
	CommandID         string `json:"command_id"`
	ExpectedRevision  int64  `json:"expected_revision"`
	MaxConcurrentRuns int    `json:"max_concurrent_runs"`
	MaxQueued         int    `json:"max_queued"`
}

func (s *API) apiFactoryCapacity(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	if !s.factoryOperator(w, v) || !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factoryCapacityRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	capacity := factory.Capacity{
		Revision: in.ExpectedRevision, UpdatedBy: v.User.ID,
		MaxConcurrentRuns: in.MaxConcurrentRuns, MaxQueued: in.MaxQueued,
	}
	if capacity.Revision < 0 || capacity.Validate() != nil {
		auth.JSONError(w, 400, "invalid_capacity", "Appliance capacity is incomplete or invalid.")
		return
	}
	receipt, err := s.Coordinator.ApplyCapacity(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, capacity)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

type factoryOperatorGrantRequest struct {
	CommandID        string `json:"command_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	MaxConcurrent    int    `json:"max_concurrent"`
	Active           bool   `json:"active"`
}

func (s *API) apiFactoryOperatorGrant(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok || !s.factoryOperator(w, v) || !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factoryOperatorGrantRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	grant := factory.OperatorGrant{
		Repository: repository, Revision: in.ExpectedRevision,
		GrantedBy: v.User.ID, Active: in.Active, MaxConcurrent: in.MaxConcurrent,
	}
	if grant.Revision < 0 || grant.Validate() != nil {
		auth.JSONError(w, 400, "invalid_operator_grant", "Operator grant is incomplete or invalid.")
		return
	}
	receipt, err := s.Coordinator.ApplyOperatorGrant(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, grant)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}
