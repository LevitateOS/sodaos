package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// requestDirectionView renders the effective direction of one build
// request: the owner's objective, the delivery approach, the explicit
// authorized outcome subset and the decision/source counts behind it.
// Idea bodies stay on the revision read; the overview carries direction
// and counts only.
type requestDirectionView struct {
	IdeasByDecision   map[string]int `json:"ideas_by_decision"`
	IdeasBySource     map[string]int `json:"ideas_by_source"`
	EffectiveOutcomes []string       `json:"effective_outcomes"`
	Objective         string         `json:"objective"`
	Approach          string         `json:"approach,omitempty"`
	IdeasTotal        int            `json:"ideas_total"`
	TasksTotal        int            `json:"tasks_total"`
	Unresolved        int            `json:"unresolved_assumptions"`
}

func requestDirectionDTO(r factory.IntentRevision) requestDirectionView {
	view := requestDirectionView{
		IdeasByDecision: map[string]int{}, IdeasBySource: map[string]int{},
		EffectiveOutcomes: append([]string(nil), factory.ResolvedScope(r)...),
		Objective:         r.Objective, Approach: r.Approach,
		IdeasTotal: len(r.Ideas), TasksTotal: len(r.Tasks),
	}
	if view.EffectiveOutcomes == nil {
		view.EffectiveOutcomes = []string{}
	}
	for _, idea := range r.Ideas {
		view.IdeasByDecision[idea.Decision]++
		source := idea.Source
		if source == "" {
			// H-I v2: empty source is unclassified unfinished draft
			// input and infers no authorship.
			source = "unclassified"
		}
		view.IdeasBySource[source]++
	}
	view.Unresolved = len(requestUnresolvedAssumptions(r.Ideas))
	return view
}

// requestUnresolvedAssumptions returns the IDs of consequential
// assumptions still awaiting the owner: material assumptions with no
// recorded decision. Independent authorized work need not wait for
// them; their dependents stay pending.
func requestUnresolvedAssumptions(ideas []factory.ProductIdea) []string {
	unresolved := []string{}
	for _, idea := range ideas {
		if idea.Assumption && idea.Material && idea.Decision == "pending" {
			unresolved = append(unresolved, idea.ID)
		}
	}
	return unresolved
}

// requestContextView renders the request's exact bound context: the
// native repository and the private department label. Repository "0"
// means unbound; the draft stays owner-private either way.
type requestContextView struct {
	Department string `json:"department,omitempty"`
	Repository string `json:"repository"`
}

// requestMappingView renders one task's exact native mapping: the bound
// native issue and its acceptance. IDs only; native content stays
// behind native authorization.
type requestMappingView struct {
	TaskKey        string `json:"task_key"`
	IssueID        string `json:"issue_id"`
	IssueIndex     string `json:"issue_index"`
	AcceptanceID   string `json:"acceptance_id"`
	CreatedUnix    int64  `json:"created_unix"`
	ContentVersion int    `json:"content_version"`
	Effective      bool   `json:"effective"`
}

func requestMappingDTO(b factory.IntentTaskBinding, tasks map[string]factory.IntentTask, effective map[string]bool) requestMappingView {
	view := requestMappingView{
		TaskKey: b.TaskKey,
		IssueID: strconv.FormatInt(b.IssueID, 10), IssueIndex: strconv.FormatInt(b.IssueIndex, 10),
		AcceptanceID: b.AcceptanceID, CreatedUnix: b.CreatedUnix, ContentVersion: b.ContentVersion,
		Effective: requestMappingEffective(tasks[b.TaskKey], effective),
	}
	return view
}

// requestMappingEffective reports whether one mapped task stays within
// the revision's explicit authorized outcome subset. An empty subset
// means legacy unscoped review input and leaves mappings effective.
func requestMappingEffective(task factory.IntentTask, effective map[string]bool) bool {
	if len(effective) == 0 {
		return true
	}
	for _, id := range task.IdeaIDs {
		if !effective[id] {
			return false
		}
	}
	return true
}

// requestWorkStates name one mapped work item's evidenced state.
// Delivered needs verified merge and confirmed original closure;
// attention names failed, fenced or cancelled work.
const (
	requestWorkDelivered = "delivered"
	requestWorkWorking   = "working"
	requestWorkWaiting   = "waiting"
	requestWorkAttention = "attention"
)

type requestWorkInput struct {
	AssignmentOutcome   factory.Outcome
	MergeStage          string
	HasAssignment       bool
	HasMerge            bool
	MergeClosureChecked bool
}

// requestWorkState derives one work item's state from its recorded
// attempt and merge facts. It reports evidence only; product-outcome
// completion is tracked separately and never inferred from closure.
func requestWorkState(in requestWorkInput) string {
	if in.HasMerge && in.MergeStage == factory.MergeMerged && in.MergeClosureChecked {
		return requestWorkDelivered
	}
	if in.HasMerge && (in.MergeStage == factory.MergeFailed || in.MergeStage == factory.MergeWithdrawn || in.MergeStage == factory.MergeFenced) {
		return requestWorkAttention
	}
	if in.HasAssignment && (in.AssignmentOutcome == factory.Failed || in.AssignmentOutcome == factory.Cancelled || in.AssignmentOutcome == factory.NeedsHuman) {
		return requestWorkAttention
	}
	if in.HasAssignment || in.HasMerge {
		return requestWorkWorking
	}
	return requestWorkWaiting
}

// requestCountsView renders evidenced work-item counts plus the
// separate product-outcome evidence tally. Counts come from recorded
// facts; several delivered work items may satisfy only part of one
// accepted product outcome, so no percentage is ever computed.
type requestCountsView struct {
	ByState           map[string]int `json:"by_state"`
	OutcomesTotal     int            `json:"outcomes_total"`
	OutcomesEvidenced int            `json:"outcomes_evidenced"`
	Total             int            `json:"total"`
}

// requestAttentionCodes name what needs the owner on one request.
const (
	requestAttentionGeneration = "generation_refused"
	requestAttentionWork       = "work_attention"
	requestAttentionAssumption = "unresolved_assumption"
)

// requestAttentionView names one item needing the owner: the code and
// the affected native issue or product idea where one exists.
type requestAttentionView struct {
	Code  string `json:"code"`
	Issue string `json:"issue,omitempty"`
	Idea  string `json:"idea,omitempty"`
}

// requestGenerationStates name the native-graph generation facts. The
// recorded state renders verbatim; not_requested means no generation
// record exists for the current revision.
const requestGenerationAbsent = "not_requested"

type requestGenerationView struct {
	State string `json:"state"`
}

// requestSetupStates name the environment setup facts for one request.
const (
	requestSetupUnassigned = "repository_unassigned"
	requestSetupMissing    = "project_missing"
	requestSetupPreparing  = "preparing"
	requestSetupReady      = "ready"
)

type requestSetupView struct {
	Project string `json:"project,omitempty"`
	State   string `json:"state"`
}

// factoryRequestOverviewView is the bounded request overview for one
// owner-private build request: direction, context, native mappings,
// evidenced work-item counts, attention, generation and setup state.
// Draft ownership and native repository visibility stay distinct:
// the read is owner-private, while mappings carry IDs that native
// authorization still governs.
type factoryRequestOverviewView struct {
	Direction    requestDirectionView   `json:"direction"`
	Context      requestContextView     `json:"context"`
	Generation   requestGenerationView  `json:"generation"`
	Setup        requestSetupView       `json:"setup"`
	WorkItems    requestCountsView      `json:"work_items"`
	Mappings     []requestMappingView   `json:"mappings"`
	Attention    []requestAttentionView `json:"attention"`
	Intent       string                 `json:"intent"`
	Revision     string                 `json:"revision"`
	State        string                 `json:"state"`
	ObservedUnix int64                  `json:"observed_unix"`
}

// apiFactoryRequestOverview shows the current overview for one
// owner-private build request. Unknown or foreign requests report 404
// without disclosure. GET never judges, provisions or dispatches.
func (s *API) apiFactoryRequestOverview(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	revision, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Build request not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read build request.")
		}
		return
	}
	view := factoryRequestOverviewView{
		Direction:  requestDirectionDTO(revision),
		Context:    requestContextView{Department: revision.Department, Repository: strconv.FormatInt(revision.Repository, 10)},
		Generation: requestGenerationView{State: requestGenerationAbsent},
		WorkItems:  requestCountsView{ByState: map[string]int{}},
		Mappings:   []requestMappingView{}, Attention: []requestAttentionView{},
		Intent: revision.Intent, Revision: revision.ID, State: revision.State,
		ObservedUnix: time.Now().Unix(),
	}
	view.WorkItems.OutcomesTotal = len(factory.ResolvedScope(revision))
	view.WorkItems.OutcomesEvidenced, err = s.Store.VerifiedApplicationOutcomeCount(r.Context(), revision)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read application verification evidence.")
		return
	}
	if generation, err := s.Store.IntentGeneration(r.Context(), revision.ID); err == nil {
		view.Generation.State = generation.State
		if generation.State == "refused" {
			view.Attention = append(view.Attention, requestAttentionView{Code: requestAttentionGeneration})
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read request generation.")
		return
	}
	bindings, err := s.Store.IntentTaskBindings(r.Context(), revision.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read request mappings.")
		return
	}
	tasks := make(map[string]factory.IntentTask, len(revision.Tasks))
	for _, task := range revision.Tasks {
		tasks[task.Key] = task
	}
	effective := make(map[string]bool, len(factory.ResolvedScope(revision)))
	for _, id := range factory.ResolvedScope(revision) {
		effective[id] = true
	}
	for _, id := range requestUnresolvedAssumptions(revision.Ideas) {
		view.Attention = append(view.Attention, requestAttentionView{Code: requestAttentionAssumption, Idea: id})
	}
	for _, binding := range bindings {
		view.Mappings = append(view.Mappings, requestMappingDTO(binding, tasks, effective))
		state, ok := s.requestWorkItemState(w, r, revision.Repository, binding)
		if !ok {
			return
		}
		view.WorkItems.ByState[state]++
		view.WorkItems.Total++
		if state == requestWorkAttention {
			view.Attention = append(view.Attention, requestAttentionView{Code: requestAttentionWork, Issue: strconv.FormatInt(binding.IssueIndex, 10)})
		}
	}
	view.Setup = s.requestSetupState(w, r, revision.Repository)
	if view.Setup.State == "" {
		return
	}
	auth.JSONResponse(w, 200, view)
}

// requestWorkItemState derives one mapped work item's evidenced state
// from its recorded attempt and merge facts.
func (s *API) requestWorkItemState(w http.ResponseWriter, r *http.Request, repository int64, binding factory.IntentTaskBinding) (string, bool) {
	var in requestWorkInput
	if assignment, err := s.Store.LatestIssueAssignment(r.Context(), repository, binding.IssueIndex); err == nil {
		in.HasAssignment, in.AssignmentOutcome = true, assignment.Outcome
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read work attempt.")
		return "", false
	}
	if merge, err := s.Store.MergeForIssue(r.Context(), repository, binding.IssueIndex); err == nil {
		in.HasMerge, in.MergeStage = true, merge.Stage
		in.MergeClosureChecked = merge.Stage == factory.MergeMerged && merge.OriginalClosedUnix > 0 && merge.OriginalClosingOccurrence != ""
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read work merge.")
		return "", false
	}
	return requestWorkState(in), true
}

// requestSetupState derives the environment setup state for one
// request's bound repository. An empty state reports that the error
// response is already written.
func (s *API) requestSetupState(w http.ResponseWriter, r *http.Request, repository int64) requestSetupView {
	if repository <= 0 {
		return requestSetupView{State: requestSetupUnassigned}
	}
	preparation, ok := s.factoryPreparationStatus(w, r, repository)
	if !ok {
		return requestSetupView{}
	}
	if preparation == nil {
		return requestSetupView{State: requestSetupMissing}
	}
	if preparation.Ready {
		return requestSetupView{Project: preparation.Project, State: requestSetupReady}
	}
	return requestSetupView{Project: preparation.Project, State: requestSetupPreparing}
}
