package api

import (
	"context"
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
	// recorded proposal or decomposition-options run its exact reviewed draft
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
		case errors.Is(err, control.ErrPlanChoiceReview):
			auth.JSONError(w, 409, "intent_review_required", err.Error())
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

// intentPlanResponse names the admitted assignment and run under the current
// head. LaunchConfirmed is false when the native start observation was lost.
type intentPlanResponse struct {
	AssignmentID    string `json:"assignment_id"`
	RunID           string `json:"run_id"`
	RevisionID      string `json:"revision_id"`
	LaunchConfirmed *bool  `json:"launch_confirmed,omitempty"`
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
		if errors.Is(err, control.ErrIntentPlanningPromptBound) {
			auth.JSONError(w, 400, "planning_prompt_over_bound", "Approved planning context exceeds the staged prompt bound; narrow the approved source selection and plan again.")
			return
		}
		auth.JSONError(w, 503, "planning_unavailable", "Draft planning inputs are unavailable: "+err.Error())
		return
	}
	if wait != nil {
		auth.JSONResponse(w, 202, map[string]any{"launched": false, "waits": []control.DispatchWait{*wait}})
		return
	}
	report := s.Coordinator.DispatchIntentPlan(r.Context(), current.Intent, v.User.ID, in.Operation, plan, prep.SourceCommit, prep.ID)
	if len(report.Errors) != 0 {
		// A lost launch observation does not undo the funded packet. Return
		// that exact admitted identity so the caller reads its status instead
		// of treating an existing worker as a failed storage operation.
		if report.Errors[0].Reason == control.DispatchErrPromptBound {
			auth.JSONError(w, 400, "planning_prompt_over_bound", report.Errors[0].Detail)
			return
		}
		if report.Errors[0].Reason == control.DispatchErrHost || report.Errors[0].Reason == control.DispatchErrFence {
			a, readErr := s.Store.IntentPlanAttempt(r.Context(), current.Intent, current.ID)
			if readErr == nil && a.Validate() == nil && a.Subject.Intent != nil && a.Subject.Intent.OwnerID == v.User.ID && a.Subject.Intent.IntentID == current.Intent && a.Subject.Intent.RevisionID == current.ID && a.Subject.Intent.ScopeDigest == current.Digest() && a.SourceCommit == prep.SourceCommit && a.Preparation == prep.ID && a.PlanningContext != nil && a.PlanningContext.Operation == in.Operation {
				confirmed := false
				auth.JSONResponse(w, http.StatusAccepted, intentPlanResponse{AssignmentID: a.ID, RunID: a.Run, RevisionID: current.ID, LaunchConfirmed: &confirmed})
				return
			}
		}
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
	Revision     string                   `json:"revision"`
	State        string                   `json:"state"`
	AssignmentID string                   `json:"assignment_id,omitempty"`
	RunID        string                   `json:"run_id,omitempty"`
	Summary      string                   `json:"summary,omitempty"`
	Candidates   int                      `json:"candidates,omitempty"`
	Attention    string                   `json:"attention,omitempty"`
	Walkthroughs *factory.ProposalContent `json:"walkthroughs,omitempty"`
	Review       *exploreData             `json:"review,omitempty"`
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
			view.Walkthroughs = out.Decomposition.Walkthroughs
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
	if view.Walkthroughs != nil {
		if err := factory.ValidatePlanningWalkthroughs(*head, *view.Walkthroughs); err != nil {
			auth.JSONError(w, 409, "planning_walkthrough_invalid", "The saved walkthrough does not match this direction.")
			return
		}
		// A nil chooser replays existing advice only. Reading progress never
		// reserves credit or asks a model to judge.
		view.Review, _ = s.reviewPlanningWalkthrough(r.Context(), *head, *view.Walkthroughs, false)
	}
	auth.JSONResponse(w, 200, view)
}

func (s *API) reviewPlanningWalkthrough(ctx context.Context, head factory.IntentRevision, content factory.ProposalContent, consult bool) (*exploreData, error) {
	coord := &control.Coordinator{Store: s.Store}
	if consult {
		judge := s.IntentExploreChooser
		if judge == nil {
			judge = &control.TypeSafePriority{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel}
		}
		coord.Planning = &control.PlanningJudgments{Chooser: judge}
	}
	recommended, err := coord.JudgeConsolidation(ctx, head, content)
	if err != nil {
		return nil, err
	}
	checked, err := coord.JudgeConsistency(ctx, head, content, recommended)
	if err != nil {
		return nil, err
	}
	return &exploreData{Content: content, Challenges: []factory.AssumptionChallenge{}, Settled: []string{}, Recommendation: &recommended, Consistency: &checked}, nil
}

// apiIntentPlanReview checks an actual recorded planner walkthrough without
// changing the draft head, so acceptance still adopts that exact native plan.
func (s *API) apiIntentPlanReview(w http.ResponseWriter, r *http.Request, v store.Session) {
	head := s.admitIntentPlanHead(w, r, v)
	if head == nil || !s.factoryOwner(w, r, v, head.Repository) {
		return
	}
	var in struct {
		Revision string `json:"revision"`
	}
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if in.Revision != head.ID {
		auth.JSONError(w, 412, "stale_revision", "Review inputs changed.")
		return
	}
	attempt, err := s.Store.IntentPlanAttempt(r.Context(), head.Intent, head.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if attempt.Subject.Intent == nil || attempt.Subject.Intent.OwnerID != v.User.ID {
		auth.JSONError(w, 404, "not_found", "Build request not found.")
		return
	}
	view := renderIntentPlanOutcome(attempt)
	if view.State != "completed" || view.Walkthroughs == nil {
		auth.JSONError(w, 409, "walkthrough_unavailable", "This plan has no completed authored walkthrough. Plan again to produce one.")
		return
	}
	if err := factory.ValidatePlanningWalkthroughs(*head, *view.Walkthroughs); err != nil {
		factoryCommandError(w, err)
		return
	}
	review, err := s.reviewPlanningWalkthrough(r.Context(), *head, *view.Walkthroughs, true)
	if err != nil {
		reportJudgmentCallError(w, err, "Walkthrough review could not finish. Completed judgments are kept; try again or revise the direction.")
		return
	}
	current, err := s.Store.IntentRevision(r.Context(), head.Intent, v.User.ID)
	if err != nil || current.ID != head.ID {
		auth.JSONError(w, 412, "stale_revision", "The direction changed while its walkthrough was checked.")
		return
	}
	auth.JSONResponse(w, 200, review)
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
