package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// factoryAssignmentResult renders the recorded dispatch result: the
// harness status, summary, candidate reference and findings, plus
// whether the harness reported it or the supervisor synthesized it from
// the observed run outcome.
type factoryAssignmentResult struct {
	Findings     []string `json:"findings"`
	Status       string   `json:"status"`
	Summary      string   `json:"summary"`
	Candidate    string   `json:"candidate"`
	RunID        string   `json:"run_id"`
	ReviewPassed bool     `json:"review_passed"`
	Reported     bool     `json:"reported"`
	RecordedUnix int64    `json:"recorded_unix"`
}

// factoryAssignmentReservation renders the assignment's held capacity:
// its state and the planned minutes its deadline implied. Only
// confirmed actual usage is ever charged; the plan is not a charge.
type factoryAssignmentReservation struct {
	Connection     string `json:"connection"`
	State          string `json:"state"`
	PlannedMinutes int    `json:"planned_minutes"`
}

// factoryAssignmentPublication renders one assignment's publication:
// its stage and outcome, the exact linked PR once creation commits,
// and each conditional operation's effect with its bounded native
// reason. Absent until publication starts.
type factoryAssignmentPublication struct {
	PublishCompletion    string `json:"publish_completion,omitempty"`
	PublishCancellation  string `json:"publish_cancellation,omitempty"`
	PRCreateCompletion   string `json:"pr_create_completion,omitempty"`
	PRCreateCancellation string `json:"pr_create_cancellation,omitempty"`
	WithdrawRequested    bool   `json:"withdraw_requested"`
	PublishEffect        string `json:"publish_effect"`
	PublishReason        string `json:"publish_reason,omitempty"`
	PRCreateEffect       string `json:"pr_create_effect"`
	PRCreateReason       string `json:"pr_create_reason,omitempty"`
	Stage                string `json:"stage"`
	Reason               string `json:"reason,omitempty"`
	PRNumber             string `json:"pr_number,omitempty"`
	PRID                 string `json:"pr_id,omitempty"`
	PublishOperation     string `json:"publish_operation,omitempty"`
	PRCreateOp           string `json:"pr_create_operation,omitempty"`
}

// factoryAssignmentView renders one dispatch assignment with its exact
// bound inputs and recorded result. The prompt carries accepted native
// text plus IDs and digests only: the builder takes no credential
// input, so the recorded bytes stay inspectable without credentials.
type factoryAssignmentView struct {
	Result       *factoryAssignmentResult      `json:"result,omitempty"`
	Reservation  *factoryAssignmentReservation `json:"reservation,omitempty"`
	Publication  *factoryAssignmentPublication `json:"publication,omitempty"`
	Authority    factory.AuthorityRef          `json:"authority"`
	RunHistory   []string                      `json:"run_history"`
	ID           string                        `json:"id"`
	ProjectID    string                        `json:"project_id"`
	Role         string                        `json:"role"`
	Acceptance   string                        `json:"acceptance"`
	Preparation  string                        `json:"preparation"`
	Harness      string                        `json:"harness"`
	HarnessVers  string                        `json:"harness_version"`
	Model        string                        `json:"model"`
	Connection   string                        `json:"connection"`
	SourceCommit string                        `json:"source_commit"`
	PromptSHA    string                        `json:"prompt_sha"`
	Prompt       string                        `json:"prompt"`
	Run          string                        `json:"run"`
	Stage        string                        `json:"stage"`
	Outcome      factory.Outcome               `json:"outcome,omitempty"`
	Reason       string                        `json:"reason,omitempty"`
	Repository   string                        `json:"repository"`
	Issue        string                        `json:"issue"`
	Attempts     int                           `json:"attempts"`
	Revision     int64                         `json:"revision"`
	NativeRev    int64                         `json:"native_revision"`
	CreatedUnix  int64                         `json:"created_unix"`
	FinishedUnix int64                         `json:"finished_unix,omitempty"`
}

func factoryAssignmentDTO(a factory.Assignment, reservation *factory.Reservation, publication *factory.Publication) factoryAssignmentView {
	view := factoryAssignmentView{
		Authority: a.Authority, RunHistory: a.RunHistory,
		ID: a.ID, ProjectID: a.ProjectID, Role: a.Role, Acceptance: a.Acceptance,
		Preparation: a.Preparation, Harness: a.Harness, HarnessVers: a.HarnessVers,
		Model: a.Model, Connection: a.Connection, SourceCommit: a.SourceCommit,
		PromptSHA: a.PromptSHA, Prompt: string(a.Prompt), Run: a.Run, Stage: a.Stage,
		Outcome: a.Outcome, Reason: a.Reason,
		Repository: strconv.FormatInt(a.Repository, 10), Issue: strconv.FormatInt(a.Issue, 10),
		Attempts: a.Attempts, Revision: a.Revision, NativeRev: a.NativeRev,
		CreatedUnix: a.CreatedUnix, FinishedUnix: a.FinishedUnix,
	}
	if a.Result != nil {
		view.Result = &factoryAssignmentResult{
			Findings: a.Result.Findings, Status: a.Result.Status, Summary: a.Result.Summary,
			Candidate: a.Result.Candidate, RunID: a.Result.RunID,
			ReviewPassed: a.Result.ReviewPassed, Reported: a.Result.Reported,
			RecordedUnix: a.Result.RecordedUnix,
		}
	}
	if reservation != nil {
		view.Reservation = &factoryAssignmentReservation{
			Connection: reservation.Connection, State: reservation.State,
			PlannedMinutes: reservation.PlannedMinutes,
		}
	}
	if publication != nil {
		rendered := &factoryAssignmentPublication{
			PublishCompletion: publication.Publish.Completion, PublishCancellation: publication.Publish.Cancellation,
			PRCreateCompletion: publication.PRCreate.Completion, PRCreateCancellation: publication.PRCreate.Cancellation,
			WithdrawRequested: publication.WithdrawRequested,
			PublishEffect:     publication.Publish.Effect, PublishReason: publication.Publish.Reason,
			PRCreateEffect: publication.PRCreate.Effect, PRCreateReason: publication.PRCreate.Reason,
			Stage: publication.Stage, Reason: publication.Reason,
			PublishOperation: publication.Publish.OperationID, PRCreateOp: publication.PRCreate.OperationID,
		}
		if publication.PRNumber != 0 {
			rendered.PRNumber = strconv.FormatInt(publication.PRNumber, 10)
			rendered.PRID = strconv.FormatInt(publication.PRID, 10)
		}
		view.Publication = rendered
	}
	return view
}

// apiFactoryAssignment shows the current dispatch assignment for one
// native issue: its exact bound inputs, staged prompt, recorded result
// and held capacity. Visibility of the repository authorizes the read;
// the prompt carries no credentials by construction.
func (s *API) apiFactoryAssignment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	repository, issue, ok := s.factoryIssue(w, r, v)
	if !ok {
		return
	}
	index, _ := strconv.ParseInt(issue, 10, 64)
	current, err := s.Store.LatestIssueAssignment(r.Context(), repository, index)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "No dispatch assignment recorded this issue.")
			return
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue assignments.")
		return
	}
	var reservation *factory.Reservation
	if held, err := s.Store.Reservation(r.Context(), current.ID); err == nil {
		reservation = &held
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read assignment reservation.")
		return
	}
	var publication *factory.Publication
	if recorded, err := s.Store.PublicationByAssignment(r.Context(), current.ID); err == nil {
		publication = &recorded
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read assignment publication.")
		return
	}
	auth.JSONResponse(w, 200, factoryAssignmentDTO(current, reservation, publication))
}
