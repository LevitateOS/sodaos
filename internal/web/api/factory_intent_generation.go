package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
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
	if _, err = s.Coordinator.GenerateAcceptedIntent(r.Context(), intent.Intent, v.User.ID); err != nil {
		factoryCommandError(w, err)
		return
	}
	generation, err := s.Store.IntentGeneration(r.Context(), intent.ID)
	if err != nil {
		factoryCommandError(w, err)
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
// the caller binds the current head revision, the requested operation
// and its inspected sources, and the enrolled host account answers the
// staged prompt. Missing live capabilities refuse truthfully.
func (s *API) apiIntentPlan(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in struct {
		Revision  string                    `json:"revision"`
		Operation string                    `json:"operation"`
		Inspected []factory.InspectedSource `json:"inspected"`
		Source    string                    `json:"source"`
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
	if !project.ValidCommit(in.Source) {
		auth.JSONError(w, 400, "invalid_source", "Planning source must name its commit.")
		return
	}
	content, err := factory.ComposeProposalContent(current)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	plan := factory.SemanticPlannerInput{
		Operation: in.Operation, Head: current, Content: content,
		Alternatives: factory.DescribeAlternatives(current.Objective), Inspected: in.Inspected,
	}
	if err := plan.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_plan_input", err.Error())
		return
	}
	if s.Coordinator == nil || s.Coordinator.Host == nil {
		auth.JSONError(w, 503, "planning_unavailable", "No enrolled model account can answer the staged prompt.")
		return
	}
	report := s.Coordinator.DispatchIntentPlan(r.Context(), current.Intent, v.User.ID, in.Operation, plan, in.Source)
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
