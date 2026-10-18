package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiIntentGenerate(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	intent, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if intent.State != "accepted" {
		auth.JSONError(w, 409, "intent_not_accepted", "Review the product direction before generating work.")
		return
	}
	if !s.factoryOwner(w, r, v, intent.Repository) {
		return
	}
	if s.Coordinator == nil || s.Coordinator.Graph == nil {
		auth.JSONError(w, 503, "native_graph_unavailable", "Native issue creation needs an enrolled graph actor.")
		return
	}
	// Generate consumes the original completed semantic result behind
	// this acceptance through the bound recorded-plan connector: the
	// recorded decomposition_options run its exact reviewed draft
	// settled, with source, preparation and head currency proven again
	// after the funded comparison and before native effects. A
	// still-running plan or a volatile preparation waits truthfully; a
	// missing, failed or stale plan and an attention answer refuse
	// before any funded choice or native observation. An accepted
	// direction with no recorded plan revises the direction and plans
	// it; no template graph is manufactured.
	generation, wait, err := s.Coordinator.GenerateAcceptedIntentFromRecordedPlan(r.Context(), intent.Intent, v.User.ID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPlanChoiceAttention):
			auth.JSONError(w, 409, "planning_attention", err.Error())
		case errors.Is(err, control.ErrPlanChoiceStale):
			auth.JSONError(w, 412, "stale_planning_choice", err.Error())
		case errors.Is(err, control.ErrPlanChoiceStore):
			factoryCommandError(w, err)
		default:
			auth.JSONError(w, 409, "planning_choice_unavailable", err.Error())
		}
		return
	}
	if wait != nil {
		auth.JSONResponse(w, 202, map[string]any{"launched": false, "waits": []control.DispatchWait{*wait}})
		return
	}
	auth.JSONResponse(w, 200, generation)
}

// intentPlanResponse answers a started draft planning run: the exact
// launched assignment and run under the current head revision.
type intentPlanResponse struct {
	AssignmentID string `json:"assignment_id"`
	RunID        string `json:"run_id"`
	RevisionID   string `json:"revision_id"`
}

// apiIntentPlan starts one draft planning run through the shared
// decompose dispatch path. Unlike accepted-intent Generate, which
// creates native work through the enrolled graph, this plans a draft:
// the caller binds only the saved revision and requested operation,
// while the server assembles source, preparation and inspected evidence
// from the authorized ready preparation and real preparation-context
// bytes. The enrolled host account answers the staged prompt. Missing
// live capabilities refuse truthfully, and missing preparation or
// context waits truthfully before any funding or launch.
func (s *API) apiIntentPlan(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in struct {
		Revision  string `json:"revision"`
		Operation string `json:"operation"`
	}
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	current, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if current.ID != in.Revision {
		auth.JSONError(w, 412, "stale_revision", "Review inputs changed.")
		return
	}
	if current.State != "draft" {
		auth.JSONError(w, 409, "intent_not_draft", "Only draft direction starts a planning run; accepted direction uses generate.")
		return
	}
	if !s.factoryOwner(w, r, v, current.Repository) {
		return
	}
	if !factory.ValidIntentPlanningOperation(in.Operation) {
		auth.JSONError(w, 400, "invalid_operation", "Unknown planning operation.")
		return
	}
	if s.Coordinator == nil || s.Coordinator.Host == nil {
		auth.JSONError(w, 503, "planning_unavailable", "No enrolled model account can answer the staged prompt.")
		return
	}
	plan, prep, wait, err := s.Coordinator.AssembleIntentPlanningInput(r.Context(), current, in.Operation)
	if err != nil {
		auth.JSONError(w, 503, "planning_unavailable", "Draft planning inputs are unavailable: "+err.Error())
		return
	}
	if wait != nil {
		auth.JSONResponse(w, 202, map[string]any{"launched": false, "waits": []control.DispatchWait{*wait}})
		return
	}
	report := s.Coordinator.DispatchIntentPlan(r.Context(), current.Intent, v.User.ID, in.Operation, plan, prep.SourceCommit, prep.ID)
	if len(report.Errors) != 0 {
		factoryCommandError(w, errors.New(report.Errors[0].Detail))
		return
	}
	if len(report.Launched) != 1 {
		auth.JSONResponse(w, 202, map[string]any{"launched": false, "waits": report.Waits})
		return
	}
	auth.JSONResponse(w, 201, intentPlanResponse{AssignmentID: report.Launched[0].AssignmentID, RunID: report.Launched[0].RunID, RevisionID: current.ID})
}

// intentPlanStatusView is the minimal scoped draft-planning status for
// one current revision: pending while the run is outstanding, the
// reported shape once it settles, and none before the first attempt.
// Copy stays about planning results, never source hashes.
type intentPlanStatusView struct {
	Revision     string `json:"revision"`
	State        string `json:"state"`
	AssignmentID string `json:"assignment_id,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Candidates   int    `json:"candidates,omitempty"`
	Attention    string `json:"attention,omitempty"`
}

// apiIntentPlanStatus shows the latest draft-planning attempt for the
// current revision. Unknown or foreign requests report 404 without
// disclosure. GET never judges, provisions or dispatches.
func (s *API) admitIntentPlanHead(w http.ResponseWriter, r *http.Request, v store.Session) *factory.IntentRevision {
	if !s.checkLifecycleSession(w, r, v) {
		return nil
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return nil
	}
	head, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Build request not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read build request.")
		}
		return nil
	}
	if head.State != "draft" {
		auth.JSONError(w, 409, "intent_not_draft", "Only draft direction carries a planning status; accepted direction uses generation.")
		return nil
	}
	return &head
}

func renderIntentPlanOutcome(a factory.Assignment) intentPlanStatusView {
	view := intentPlanStatusView{AssignmentID: a.ID, RunID: a.Run}
	switch {
	case a.Stage != factory.AssignmentFinished || a.Result == nil:
		view.State = "pending"
	case a.Outcome == factory.Failed:
		view.State, view.Summary = "failed", a.Result.Summary
	case a.Outcome == factory.Cancelled:
		view.State, view.Summary = "cancelled", a.Result.Summary
	case a.Result.Reported && a.Result.PhaseResult != nil && a.Result.PhaseResult.Status == "completed":
		view.State, view.Summary = "completed", a.Result.Summary
		if out := a.Result.PhaseResult.Output; out != nil && out.Decomposition != nil {
			view.Candidates = len(out.Decomposition.Candidates)
		}
	default:
		view.State, view.Summary = "attention", a.Result.Summary
		if out := a.Result.PhaseResult; out != nil && out.Output != nil && out.Output.Decomposition != nil {
			view.Attention = out.Output.Decomposition.Attention
		}
	}
	return view
}

func (s *API) apiIntentPlanStatus(w http.ResponseWriter, r *http.Request, v store.Session) {
	head := s.admitIntentPlanHead(w, r, v)
	if head == nil {
		return
	}
	a, err := s.Store.IntentPlanAttempt(r.Context(), head.Intent, head.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONResponse(w, 200, intentPlanStatusView{Revision: head.ID, State: "none"})
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read planning status.")
		}
		return
	}
	if a.Subject.Intent == nil || a.Subject.Intent.OwnerID != v.User.ID {
		auth.JSONError(w, 404, "not_found", "Build request not found.")
		return
	}
	view := renderIntentPlanOutcome(a)
	view.Revision = head.ID
	auth.JSONResponse(w, 200, view)
}

func (s *API) apiIntentGeneration(w http.ResponseWriter, r *http.Request, v store.Session) {
	intent, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	generation, err := s.Store.IntentGeneration(r.Context(), intent.ID)
	if err == store.ErrNotFound {
		auth.JSONResponse(w, 200, map[string]any{"state": "pending", "revision_id": intent.ID})
		return
	}
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, generation)
}
