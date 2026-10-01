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

// factoryAssignmentView renders one dispatch assignment with its exact
// bound inputs and recorded result. The prompt carries accepted native
// text plus IDs and digests only: the builder takes no credential
// input, so the recorded bytes stay inspectable without credentials.
type factoryAssignmentView struct {
	Result       *factoryAssignmentResult      `json:"result,omitempty"`
	Reservation  *factoryAssignmentReservation `json:"reservation,omitempty"`
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

func factoryAssignmentDTO(a factory.Assignment, reservation *factory.Reservation) factoryAssignmentView {
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
	return view
}

// currentIssueAssignment selects the latest recorded assignment for one
// native issue. Assignments list oldest first; the last one is current.
func currentIssueAssignment(assignments []factory.Assignment) (factory.Assignment, bool) {
	if len(assignments) == 0 {
		return factory.Assignment{}, false
	}
	return assignments[len(assignments)-1], true
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
	assignments, err := s.Store.IssueAssignments(r.Context(), repository, index)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue assignments.")
		return
	}
	current, ok := currentIssueAssignment(assignments)
	if !ok {
		auth.JSONError(w, 404, "not_found", "No dispatch assignment recorded this issue.")
		return
	}
	var reservation *factory.Reservation
	if held, err := s.Store.Reservation(r.Context(), current.ID); err == nil {
		reservation = &held
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read assignment reservation.")
		return
	}
	auth.JSONResponse(w, 200, factoryAssignmentDTO(current, reservation))
}
