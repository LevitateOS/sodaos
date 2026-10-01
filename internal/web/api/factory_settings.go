package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// factoryPrincipal names the admitted browser actor. Body fields can never
// select the authorizing human; the host-derived session owns attribution.
func factoryPrincipal(v store.Session) string {
	return "native:" + strconv.FormatInt(v.User.ID, 10)
}

func factoryCommandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrCommandConflict):
		auth.JSONError(w, 409, "command_conflict", "Command identity reused for different content.")
	case errors.Is(err, store.ErrStaleRevision):
		auth.JSONError(w, 412, "stale_revision", "Grant changed while applying; refresh and retry.")
	case errors.Is(err, control.ErrCommandRunning):
		auth.JSONResponse(w, 202, map[string]any{"running": true})
	case errors.Is(err, control.ErrIneffectiveAuthority):
		auth.JSONError(w, 409, "authority_ineffective", "Current grants do not authorize this action.")
	default:
		auth.JSONError(w, 503, "store_unavailable", "Factory authority storage is unavailable.")
	}
}

// factoryRepository resolves the path repository under current native
// visibility. Absent or undisclosable repositories report 404.
func (s *API) factoryRepository(w http.ResponseWriter, r *http.Request, v store.Session) (int64, bool) {
	id, valid := auth.PositiveID(r.PathValue("repositoryID"))
	if !valid {
		auth.JSONError(w, 404, "not_found", "Repository not found.")
		return 0, false
	}
	if _, err := s.visibleRepository(r, v, id); err != nil {
		auth.ProviderError(w, err)
		return 0, false
	}
	return id, true
}

// factoryOwner admits the current native repository owner or administrator.
// Organization ownership counts; the appliance operator does not bypass.
func (s *API) factoryOwner(w http.ResponseWriter, r *http.Request, v store.Session, repository int64) bool {
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if access.repository.Permissions != nil && access.repository.Permissions.Admin {
		return true
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if !allowed {
		auth.JSONError(w, 403, "repository_admin_required", "Only the current repository owner or administrator can configure factory policy.")
		return false
	}
	return true
}

// factoryOperator admits only the configured Soda operator.
func (s *API) factoryOperator(w http.ResponseWriter, v store.Session) bool {
	if v.User.ID != s.Config.OperatorID {
		auth.JSONError(w, 403, "operator_required", "Only the configured Soda operator can change appliance capacity.")
		return false
	}
	return true
}

// settingsCommandID validates the client-generated command identity every
// settings mutation carries for idempotent replay.
func settingsCommandID(w http.ResponseWriter, commandID string) bool {
	if !factory.ValidID(commandID) {
		auth.JSONError(w, 400, "invalid_command", "Settings changes require a client-generated command identity.")
		return false
	}
	return true
}

type factoryQueueView struct {
	Queued int `json:"queued"`
	Active int `json:"active"`
}

type factoryStatusView struct {
	Policy           *factory.RepositoryPolicy  `json:"policy"`
	OperatorGrant    *factory.OperatorGrant     `json:"operator_grant"`
	Appliance        *factory.Capacity          `json:"capacity"`
	Sponsorships     []factory.Sponsorship      `json:"sponsorships"`
	EnvironmentGrant *project.EnvironmentGrant  `json:"environment_grant"`
	Preparation      *factoryPreparationView    `json:"preparation"`
	Queue            factoryQueueView           `json:"queue"`
	Effective        factory.EffectiveAuthority `json:"effective"`
	Repository       string                     `json:"repository"`
}

type factoryPreparationView struct {
	Project      string `json:"project"`
	Requirements string `json:"requirements,omitempty"`
	Approval     string `json:"approval,omitempty"`
	Ready        bool   `json:"ready"`
}

// sponsorshipView redacts the private allowance for viewers other than the
// sponsoring connection owner. References stay; spending details do not.
func sponsorshipView(sp factory.Sponsorship, owner bool) factory.Sponsorship {
	if owner {
		return sp
	}
	sp.AllowanceMinutes, sp.MaxConcurrent = 0, 0
	return sp
}

func (s *API) apiFactoryStatus(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	ctx := r.Context()
	view := factoryStatusView{Repository: strconv.FormatInt(repository, 10), Sponsorships: []factory.Sponsorship{}}
	if policy, err := s.Store.RepositoryPolicy(ctx, repository); err == nil {
		view.Policy = &policy
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read factory policy.")
		return
	}
	if grant, err := s.Store.OperatorGrant(ctx, repository); err == nil {
		view.OperatorGrant = &grant
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read operator grant.")
		return
	}
	if capacity, err := s.Store.Capacity(ctx); err == nil {
		view.Appliance = &capacity
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read appliance capacity.")
		return
	}
	sponsorships, err := s.Store.Sponsorships(ctx, repository)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read sponsorships.")
		return
	}
	for _, sp := range sponsorships {
		view.Sponsorships = append(view.Sponsorships, sponsorshipView(sp, sp.GrantedBy == v.User.ID))
	}
	if grant, err := s.Store.EnvironmentGrant(ctx, repository); err == nil {
		view.EnvironmentGrant = &grant
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read environment grant.")
		return
	}
	if preparation, ok := s.factoryPreparationStatus(w, r, repository); !ok {
		return
	} else {
		view.Preparation = preparation
	}
	effective, err := s.Coordinator.EffectiveAuthority(ctx, repository)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not evaluate factory authority.")
		return
	}
	view.Effective = effective
	controls, err := s.Store.IssueControls(ctx, repository, store.MaxIssueControls)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read factory queue.")
		return
	}
	for _, recorded := range controls {
		if recorded.Readiness == factory.ReadinessQueued {
			view.Queue.Queued++
		}
	}
	auth.JSONResponse(w, 200, view)
}

func (s *API) factoryPreparationStatus(w http.ResponseWriter, r *http.Request, repository int64) (*factoryPreparationView, bool) {
	p, err := s.Store.ProjectByRepository(r.Context(), repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, true
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read project preparation.")
		return nil, false
	}
	view := &factoryPreparationView{Project: p.ID}
	if head, err := s.Store.RequirementHead(r.Context(), p.ID); err == nil {
		view.Requirements = head
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read preparation acceptance.")
		return nil, false
	}
	if head, err := s.Store.ApprovalHead(r.Context(), p.ID); err == nil {
		view.Approval = head
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read preparation approval.")
		return nil, false
	}
	records, err := s.Store.ProjectPreparations(r.Context(), p.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read preparations.")
		return nil, false
	}
	ready := map[string]bool{}
	for _, record := range records {
		if record.State.Ready {
			ready[record.Preparation.Role] = true
		}
	}
	view.Ready = ready[project.RoleCoder] && ready[project.RoleReviewer]
	return view, true
}

type actorRefRequest struct {
	TokenID string `json:"token_id"`
	ActorID string `json:"actor_id"`
}

func (a actorRefRequest) resolve(kind string) (factory.ActorBindingRef, bool) {
	token, tokenOK := auth.PositiveID(a.TokenID)
	actor, actorOK := auth.PositiveID(a.ActorID)
	ref := factory.ActorBindingRef{TokenID: token, ActorID: actor, Kind: kind}
	return ref, tokenOK && actorOK && ref.Validate() == nil
}

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

type factoryEnvironmentGrantRequest struct {
	Profile          *project.Profile `json:"profile"`
	CommandID        string           `json:"command_id"`
	Project          string           `json:"project,omitempty"`
	ExpectedRevision int64            `json:"expected_revision"`
	Active           bool             `json:"active"`
}

func (s *API) apiFactoryEnvironmentGrant(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	owner, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	if !owner || !s.checkLifecycleSession(w, r, v) {
		if !owner {
			auth.JSONError(w, 403, "repository_owner_required", "Only the current repository owner can permit factory environment setup.")
		}
		return
	}
	var in factoryEnvironmentGrantRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	grant := project.EnvironmentGrant{
		Repository: repository, Revision: in.ExpectedRevision,
		Owner: v.User.ID, Profile: in.Profile, Project: in.Project, Active: in.Active,
	}
	if grant.Revision < 0 || grant.Validate() != nil {
		auth.JSONError(w, 400, "invalid_environment_grant", "Environment grant is incomplete or invalid.")
		return
	}
	receipt, err := s.Coordinator.ApplyEnvironmentGrant(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, grant)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

type factorySponsorshipRequest struct {
	CommandID        string   `json:"command_id"`
	GrantID          string   `json:"grant_id"`
	ExpectedRevision int64    `json:"expected_revision"`
	Generation       int64    `json:"generation"`
	Roles            []string `json:"roles"`
	AllowanceMinutes int      `json:"allowance_minutes"`
	MaxConcurrent    int      `json:"max_concurrent"`
	Active           bool     `json:"active"`
}

func (s *API) apiFactorySponsorship(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok || !s.checkLifecycleSession(w, r, v) {
		return
	}
	connection := r.PathValue("connection")
	var in factorySponsorshipRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	sponsorship := factory.Sponsorship{
		Repository: repository, Revision: in.ExpectedRevision, GrantedBy: v.User.ID,
		Connection: connection, GrantID: in.GrantID, Generation: in.Generation, Roles: in.Roles,
		AllowanceMinutes: in.AllowanceMinutes, MaxConcurrent: in.MaxConcurrent, Active: in.Active,
	}
	if sponsorship.Revision < 0 || sponsorship.Validate() != nil {
		auth.JSONError(w, 400, "invalid_sponsorship", "Sponsorship is incomplete or invalid.")
		return
	}
	if !s.checkSponsorshipBroker(w, r, v, sponsorship) {
		return
	}
	receipt, err := s.Coordinator.ApplySponsorship(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, sponsorship)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

// checkSponsorshipBroker verifies the sponsorship references the caller's
// own current connection and a live canonical broker grant. It reads grant
// metadata only; credential custody stays with the broker.
func (s *API) checkSponsorshipBroker(w http.ResponseWriter, r *http.Request, v store.Session, sponsorship factory.Sponsorship) bool {
	connection, err := s.Store.IdentityConnection(r.Context(), sponsorship.Connection)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Provider connection not found.")
			return false
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read provider connection.")
		return false
	}
	if connection.OwnerID != v.User.ID {
		auth.JSONError(w, 403, "connection_owner_required", "Only the connection owner can sponsor factory use.")
		return false
	}
	if connection.Generation != sponsorship.Generation {
		auth.JSONError(w, 409, "stale_credential_generation", "Connection credential changed; refresh and retry.")
		return false
	}
	grant, err := s.Store.IdentityGrant(r.Context(), sponsorship.GrantID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Broker grant not found.")
			return false
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read broker grant.")
		return false
	}
	if grant.ConnectionID != sponsorship.Connection || grant.Revoked {
		auth.JSONError(w, 409, "grant_unavailable", "Broker grant is revoked or belongs elsewhere.")
		return false
	}
	return true
}
