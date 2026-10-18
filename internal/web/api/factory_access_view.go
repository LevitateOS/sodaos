package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// accessEntryView reports whether the viewer may save one settings
// section, with the bounded refusal reason when not. The owning PUT
// APIs revalidate at save time.
type accessEntryView struct {
	Reason   string `json:"reason,omitempty"`
	Editable bool   `json:"editable"`
}

// accessEditableReasons are the bounded codes behind one editability
// entry. They mirror the owning PUT admission rules.
const (
	accessReasonSession    = "session_changed"
	accessReasonOwnerAdmin = "owner_or_admin_required"
	accessReasonOperator   = "operator_required"
	accessReasonRepoOwner  = "repository_owner_required"
	accessReasonConnOwner  = "connection_owner_required"
	accessReasonIdentity   = "identity_unavailable"
)

// accessViewerView renders the viewer's current standing: native
// repository powers, appliance operator identity, project membership
// and session currency. Each boundary stays separate; no power implies
// another.
type accessViewerView struct {
	CodeWrite        bool `json:"code_write"`
	OwnerOrAdmin     bool `json:"owner_or_admin"`
	EnvironmentOwner bool `json:"environment_owner"`
	Operator         bool `json:"operator"`
	Member           bool `json:"member"`
	SessionCurrent   bool `json:"session_current"`
}

// accessConnectionEntryView reports editability for one connection-owned
// section: sponsorships and usage budgets admit the connection owner.
type accessConnectionEntryView struct {
	Connection string `json:"connection"`
	Reason     string `json:"reason,omitempty"`
	Editable   bool   `json:"editable"`
}

// accessEditableView maps every Factory settings section to the viewer's
// current save power. Values live on the status route; this view carries
// powers only, never a second copy of the settings.
type accessEditableView struct {
	Sponsorships     []accessConnectionEntryView `json:"sponsorships"`
	UsageBudgets     []accessConnectionEntryView `json:"usage_budgets"`
	Policy           accessEntryView             `json:"policy"`
	ActivitySettings accessEntryView             `json:"activity_settings"`
	OperatorGrant    accessEntryView             `json:"operator_grant"`
	Capacity         accessEntryView             `json:"capacity"`
	EnvironmentGrant accessEntryView             `json:"environment_grant"`
}

type accessEditableInput struct {
	Viewer      accessViewerView
	Connections []string
	Owned       map[string]bool
	IdentityOK  bool
}

func accessEditableDTO(in accessEditableInput) accessEditableView {
	view := accessEditableView{Sponsorships: []accessConnectionEntryView{}, UsageBudgets: []accessConnectionEntryView{}}
	session := ""
	if !in.Viewer.SessionCurrent {
		session = accessReasonSession
	}
	gated := func(allowed bool, reason string) accessEntryView {
		if session != "" {
			return accessEntryView{Reason: session}
		}
		if !allowed {
			return accessEntryView{Reason: reason}
		}
		return accessEntryView{Editable: true}
	}
	view.Policy = gated(in.Viewer.OwnerOrAdmin, accessReasonOwnerAdmin)
	view.ActivitySettings = gated(in.Viewer.OwnerOrAdmin, accessReasonOwnerAdmin)
	view.OperatorGrant = gated(in.Viewer.Operator, accessReasonOperator)
	view.Capacity = gated(in.Viewer.Operator, accessReasonOperator)
	view.EnvironmentGrant = gated(in.Viewer.EnvironmentOwner, accessReasonRepoOwner)
	for _, connection := range in.Connections {
		entry := accessConnectionEntryView{Connection: connection}
		switch {
		case session != "":
			entry.Reason = session
		case !in.IdentityOK:
			entry.Reason = accessReasonIdentity
		case !in.Owned[connection]:
			entry.Reason = accessReasonConnOwner
		default:
			entry.Editable = true
		}
		view.Sponsorships = append(view.Sponsorships, entry)
		view.UsageBudgets = append(view.UsageBudgets, entry)
	}
	return view
}

// accessResolvers name who can resolve one missing authorization.
const (
	accessResolveOwnerAdmin = "owner_or_admin"
	accessResolveOperator   = "operator"
	accessResolveConnOwner  = "connection_owner"
	accessResolveMaintainer = "maintainer"
)

// accessActionView maps one missing authorization to the role that can
// resolve it and the settings section that owns the fix.
type accessActionView struct {
	Code      string `json:"code"`
	ResolveBy string `json:"resolve_by"`
	Setting   string `json:"setting"`
}

// accessRequiredActions renders the required owner actions behind an
// effective-authority verdict. Unknown codes stay visible with an empty
// resolver rather than vanishing or guessing.
func accessRequiredActions(missing []string) []accessActionView {
	actions := []accessActionView{}
	for _, code := range missing {
		action := accessActionView{Code: code}
		switch code {
		case factory.MissingPolicy, factory.MissingPolicyDisabled, factory.MissingPolicyPaused:
			action.ResolveBy, action.Setting = accessResolveOwnerAdmin, "policy"
		case factory.MissingOperatorGrant, factory.MissingOperatorWithdrawn, factory.MissingCapacity:
			action.ResolveBy, action.Setting = accessResolveOperator, "operator_grant"
			if code == factory.MissingCapacity {
				action.Setting = "capacity"
			}
		case factory.MissingSponsorship, factory.MissingSponsorshipGone:
			action.ResolveBy, action.Setting = accessResolveConnOwner, "sponsorships"
		case factory.MissingConnectionUsageBudget:
			action.ResolveBy, action.Setting = accessResolveConnOwner, "usage_budgets"
		case factory.MissingEnvironment, factory.MissingEnvironmentGone:
			action.ResolveBy, action.Setting = accessResolveOwnerAdmin, "environment_grant"
		case factory.MissingPreparation:
			action.ResolveBy, action.Setting = accessResolveOwnerAdmin, "project"
		case factory.MissingDispatch:
			action.ResolveBy, action.Setting = accessResolveMaintainer, "lifecycle"
		}
		actions = append(actions, action)
	}
	return actions
}

// accessUsageStatuses name why one connection's usage numbers are
// withheld instead of rendered.
const (
	accessUsageOwnerOnly    = "connection_owner_only"
	accessUsageUnavailable  = "usage_unavailable"
	accessUsageIdentityDown = "identity_unavailable"
)

// accessUsageView renders actual execution usage for one sponsored
// connection in real minutes, or the explicit status withholding it.
// Budget and spend belong to the connection owner; other viewers see
// the withholding status, never zeros that could read as no spend.
type accessUsageView struct {
	UsedMinutes   *int   `json:"used_minutes,omitempty"`
	Connection    string `json:"connection"`
	Unit          string `json:"unit"`
	Status        string `json:"status,omitempty"`
	BudgetMinutes int64  `json:"budget_minutes,omitempty"`
}

type accessUsageInput struct {
	Connection string
	Owned      bool
	BudgetOK   bool
	UsedOK     bool
	Budget     int64
	Used       int
}

// accessUsageDTO renders one connection's usage. Values render only for
// the verified connection owner with both reads succeeding; every other
// case names its withholding status.
func accessUsageDTO(in accessUsageInput) accessUsageView {
	view := accessUsageView{Connection: in.Connection, Unit: "minutes"}
	if !in.Owned {
		view.Status = accessUsageOwnerOnly
		return view
	}
	if !in.BudgetOK || !in.UsedOK {
		view.Status = accessUsageUnavailable
		return view
	}
	used := in.Used
	view.BudgetMinutes, view.UsedMinutes = in.Budget, &used
	return view
}

// factoryAccessView is the permission, readiness and usage read for one
// repository's Factory settings: the viewer's standing and save powers,
// the effective-authority verdict with required actions, preparation
// readiness and actual per-connection execution usage.
// accessAllowanceStatuses name one planning allowance's spend state.
const (
	accessAllowanceOK        = "ok"
	accessAllowanceExhausted = "exhausted"
)

// accessAllowanceView renders one request's planning allowance in real
// units: the independent revision's amount, reservation, consumption,
// remainder and ceiling, plus whether the viewer may adjust it. The
// listing is owner-private, so adjustment power is the viewer's own
// gated on session currency; the allowance route revalidates at save.
type accessAllowanceView struct {
	Editable  accessEntryView `json:"editable"`
	Intent    string          `json:"intent"`
	ID        string          `json:"id"`
	Unit      string          `json:"unit"`
	Effect    string          `json:"effect,omitempty"`
	Status    string          `json:"status"`
	Amount    int             `json:"amount"`
	Reserved  int             `json:"reserved"`
	Consumed  int             `json:"consumed"`
	Available int             `json:"available"`
	Ceiling   int             `json:"ceiling"`
}

func accessAllowanceDTO(a factory.PlanningAllowance, sessionCurrent bool) accessAllowanceView {
	view := accessAllowanceView{
		Intent: a.Intent, ID: a.ID, Unit: a.Unit, Effect: a.Effect, Status: accessAllowanceOK,
		Amount: a.Amount, Reserved: a.Reserved, Consumed: a.Consumed,
		Available: a.Available(), Ceiling: a.Ceiling,
	}
	if view.Available <= 0 {
		view.Status = accessAllowanceExhausted
	}
	view.Editable = accessEntryView{Editable: true}
	if !sessionCurrent {
		view.Editable = accessEntryView{Reason: accessReasonSession}
	}
	return view
}

// factoryAccessView is the permission, readiness and usage read for one
// repository's Factory settings: the viewer's standing and save powers,
// the effective-authority verdict with required actions, preparation
// readiness, actual per-connection execution usage and the viewer's
// planning allowances bound to the repository.
type factoryAccessView struct {
	Viewer            accessViewerView           `json:"viewer"`
	Editable          accessEditableView         `json:"editable"`
	Effective         factory.EffectiveAuthority `json:"effective"`
	Preparation       *factoryPreparationView    `json:"preparation"`
	RequiredActs      []accessActionView         `json:"required_actions"`
	Usage             []accessUsageView          `json:"usage"`
	Planning          []accessAllowanceView      `json:"planning"`
	Repository        string                     `json:"repository"`
	ObservedUnix      int64                      `json:"observed_unix"`
	PlanningTruncated bool                       `json:"planning_truncated,omitempty"`
}

// apiFactoryAccess shows the current permission, readiness and usage for
// one repository's Factory settings: execution usage plus the viewer's
// planning allowances. Values on the status route stay authoritative;
// this read adds powers, usage and required actions. It never judges,
// provisions or dispatches.
type admittedFactoryAccess struct {
	repository int64
	viewer     accessViewerView
	access     repositoryAccess
}

func (s *API) admitFactoryAccessViewer(ctx context.Context, w http.ResponseWriter, r *http.Request, v store.Session) *admittedFactoryAccess {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return nil
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return nil
	}
	viewer := accessViewerView{Operator: v.User.ID == s.Config.OperatorID, SessionCurrent: s.extensionSessionCurrent(ctx, r, v)}
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return nil
	}
	viewer.CodeWrite = repositoryExecutionAllowed(access)
	viewer.OwnerOrAdmin = access.repository.Permissions != nil && access.repository.Permissions.Admin
	if administrator, err := s.environmentAdministrator(r, access); err != nil {
		auth.ProviderError(w, err)
		return nil
	} else {
		viewer.EnvironmentOwner = administrator
		viewer.OwnerOrAdmin = viewer.OwnerOrAdmin || administrator
	}
	if p, err := s.Store.ProjectByRepository(ctx, repository); err == nil {
		if _, err := s.Store.MemberLogin(ctx, p.ID, v.User.ID); err == nil {
			viewer.Member = true
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read membership.")
			return nil
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read project.")
		return nil
	}
	return &admittedFactoryAccess{repository: repository, viewer: viewer, access: access}
}

func (s *API) collectFactoryAccessUsage(ctx context.Context, repository int64, sponsorships []factory.Sponsorship, editable *accessEditableInput, view *factoryAccessView) {
	for _, sponsorship := range sponsorships {
		if !editable.IdentityOK {
			view.Usage = append(view.Usage, accessUsageView{Connection: sponsorship.Connection, Unit: "minutes", Status: accessUsageIdentityDown})
			continue
		}
		in := accessUsageInput{Connection: sponsorship.Connection, Owned: editable.Owned[sponsorship.Connection]}
		if in.Owned {
			if budget, err := s.Store.ConnectionUsageBudget(ctx, sponsorship.Connection); err == nil {
				in.Budget, in.BudgetOK = budget.RollingMinutes, true
			} else if !errors.Is(err, store.ErrNotFound) {
				view.Usage = append(view.Usage, accessUsageView{Connection: sponsorship.Connection, Unit: "minutes", Status: accessUsageUnavailable})
				continue
			}
			if total, err := s.Store.UsageTotal(ctx, repository, sponsorship.Connection); err == nil {
				in.Used, in.UsedOK = total, true
			} else {
				view.Usage = append(view.Usage, accessUsageView{Connection: sponsorship.Connection, Unit: "minutes", Status: accessUsageUnavailable})
				continue
			}
			if !in.BudgetOK {
				// A missing budget is itself a required action
				// (connection_usage_budget_missing), not a zero budget.
				view.Usage = append(view.Usage, accessUsageView{Connection: sponsorship.Connection, Unit: "minutes", Status: accessUsageUnavailable})
				continue
			}
		}
		view.Usage = append(view.Usage, accessUsageDTO(in))
	}
}

func (s *API) apiFactoryAccess(w http.ResponseWriter, r *http.Request, v store.Session) {
	ctx := r.Context()
	admitted := s.admitFactoryAccessViewer(ctx, w, r, v)
	if admitted == nil {
		return
	}
	repository, viewer := admitted.repository, admitted.viewer
	sponsorships, err := s.Store.Sponsorships(ctx, repository)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read sponsorships.")
		return
	}
	editable := accessEditableInput{Viewer: viewer, Owned: map[string]bool{}}
	for _, sponsorship := range sponsorships {
		editable.Connections = append(editable.Connections, sponsorship.Connection)
	}
	if s.Identity != nil {
		if connections, err := s.Identity.Connections(ctx, v.User.ID); err == nil {
			editable.IdentityOK = true
			for _, connection := range connections {
				if connection.OwnerID == v.User.ID {
					editable.Owned[connection.ID] = true
				}
			}
		}
	}
	view := factoryAccessView{
		Viewer: viewer, Editable: accessEditableDTO(editable),
		RequiredActs: []accessActionView{}, Usage: []accessUsageView{}, Planning: []accessAllowanceView{},
		Repository: strconv.FormatInt(repository, 10), ObservedUnix: time.Now().Unix(),
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	effective, err := s.Coordinator.EffectiveAuthority(ctx, repository)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not evaluate factory authority.")
		return
	}
	view.Effective, view.RequiredActs = effective, accessRequiredActions(effective.Missing)
	if preparation, ok := s.factoryPreparationStatus(w, r, repository); !ok {
		return
	} else {
		view.Preparation = preparation
	}
	s.collectFactoryAccessUsage(ctx, repository, sponsorships, &editable, &view)
	allowances, truncated, err := s.Store.RepositoryIntentAllowances(ctx, repository, v.User.ID, store.MaxRepositoryAllowances)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read planning allowances.")
		return
	}
	for _, allowance := range allowances {
		view.Planning = append(view.Planning, accessAllowanceDTO(allowance, viewer.SessionCurrent))
	}
	view.PlanningTruncated = truncated
	auth.JSONResponse(w, 200, view)
}
