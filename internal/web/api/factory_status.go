package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

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
