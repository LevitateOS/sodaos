package api

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// factoryRunOutputExcerpt bounds the final-output excerpt carried by the run
// status read. Live bytes stream through the output route; an ended run
// exposes its outcome, never a replayable transcript.
const factoryRunOutputExcerpt = 16 * 1024

// factoryRunRecord carries the recorded run facts a viewer may see. Broker
// lease internals stay out; live process identity comes from host state.
type factoryRunRecord struct {
	Started    time.Time       `json:"started"`
	Deadline   time.Time       `json:"deadline"`
	Outcome    factory.Outcome `json:"outcome,omitempty"`
	ID         string          `json:"id"`
	ProjectID  string          `json:"project_id"`
	Role       string          `json:"role"`
	Harness    string          `json:"harness"`
	Model      string          `json:"model"`
	InputSHA   string          `json:"input_sha"`
	Summary    string          `json:"summary,omitempty"`
	Reconciled bool            `json:"reconciled"`
}

func factoryRunRecordDTO(r factory.Run) factoryRunRecord {
	return factoryRunRecord{
		Started: r.Started, Deadline: r.Deadline, Outcome: r.Outcome,
		ID: r.ID, ProjectID: r.ProjectID, Role: r.Role, Harness: r.Harness,
		Model: r.Model, InputSHA: r.InputSHA, Summary: r.Summary, Reconciled: r.Reconciled,
	}
}

// factoryRunBindingDTO renders the display binding with native IDs in
// decimal-string form. An unbound run renders no binding, never a guess.
type factoryRunBinding struct {
	Repository string `json:"repository"`
	Issue      string `json:"issue,omitempty"`
	Attempt    string `json:"attempt,omitempty"`
}

func factoryRunBindingDTO(v factory.RunView) *factoryRunBinding {
	binding := &factoryRunBinding{Repository: strconv.FormatInt(v.Repository, 10)}
	if v.Issue > 0 {
		binding.Issue = strconv.FormatInt(v.Issue, 10)
	}
	if v.Attempt != "" {
		binding.Attempt = v.Attempt
	}
	return binding
}

// factoryRunLive carries the observed host state for one run: its phase and
// exact process binding plus a bounded final-output excerpt.
type factoryRunLive struct {
	ExitCode        *int   `json:"exit_code,omitempty"`
	Live            bool   `json:"live"`
	Terminal        bool   `json:"terminal"`
	OutputTruncated bool   `json:"output_truncated"`
	Phase           string `json:"phase"`
	Container       string `json:"container,omitempty"`
	Unit            string `json:"unit,omitempty"`
	Invocation      string `json:"invocation,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Retirement      string `json:"retirement,omitempty"`
	Output          string `json:"output,omitempty"`
}

// factoryHostTerminal reports whether a host phase is settled. Uncertain
// runs stay open: retirement or credential return is still unconfirmed.
func factoryHostTerminal(phase string) bool {
	switch phase {
	case project.FactoryCompleted, project.FactoryFailed, project.FactoryStopped:
		return true
	default:
		return false
	}
}

func factoryRunLiveDTO(out project.FactoryState) *factoryRunLive {
	live := &factoryRunLive{
		ExitCode: out.ExitCode, Live: out.Live, Terminal: factoryHostTerminal(out.Phase),
		Phase: out.Phase, Container: out.Container, Unit: out.Unit, Invocation: out.Invocation,
		Reason: out.Reason, Retirement: out.Retirement,
	}
	if len(out.Output) > factoryRunOutputExcerpt {
		live.Output, live.OutputTruncated = out.Output[:factoryRunOutputExcerpt], true
	} else {
		live.Output = out.Output
	}
	return live
}

type factoryRunStatus struct {
	Run               factoryRunRecord              `json:"run"`
	View              *factoryRunBinding            `json:"view,omitempty"`
	State             *factoryRunLive               `json:"state,omitempty"`
	ContextSelection  *control.ContextSelectionView `json:"context_selection,omitempty"`
	VerificationRetry *workCapabilityView           `json:"verification_retry,omitempty"`
	DeliveryRepair    *factoryDeliveryRepairView    `json:"delivery_repair,omitempty"`
}

type factoryDeliveryRepairView struct {
	Available bool                           `json:"available"`
	Progress  *factoryDeliveryRepairProgress `json:"progress,omitempty"`
}

// Only bounded lifecycle/cause facts leave the private repair journal. Its
// collector inputs, native source material and adoption packets stay private.
type factoryDeliveryRepairProgress struct {
	CauseID      string                         `json:"cause_id"`
	CycleID      string                         `json:"cycle_id"`
	FailedRun    string                         `json:"failed_run"`
	SourceCommit string                         `json:"source_commit"`
	State        string                         `json:"state"`
	Reason       string                         `json:"reason,omitempty"`
	RunID        string                         `json:"run_id,omitempty"`
	Failures     []factory.CheckFailure         `json:"failures"`
	FailureCount int                            `json:"failure_count"`
	Judgment     *factoryDeliveryRepairJudgment `json:"judgment,omitempty"`
}

type factoryDeliveryRepairJudgment struct {
	Option      string  `json:"option"`
	Probability float64 `json:"accept_probability"`
	Confidence  float64 `json:"confidence"`
}

func deliveryRepairProgressDTO(record store.DeliveryRepairRecord, assignment factory.Assignment) (*factoryDeliveryRepairProgress, error) {
	cause, cycle := record.Cause, record.Cycle
	if assignment.Subject.Delivery == nil || cause.FailedAssignmentID != assignment.ID || cause.FailedVerificationRunID != assignment.Run || cause.Repository != assignment.Repository || cause.ProjectID != assignment.ProjectID || cause.Delivery != *assignment.Subject.Delivery {
		return nil, errors.New("repair progress differs from its failed verification")
	}
	view := &factoryDeliveryRepairProgress{CauseID: factory.DeliveryRepairCauseID(assignment.Run), CycleID: cycle.ID, FailedRun: assignment.Run, SourceCommit: cause.Delivery.SourceCommit, State: cycle.State, Reason: cycle.Reason, RunID: cycle.RunID, Failures: cause.Report.Failures[:min(8, len(cause.Report.Failures))], FailureCount: len(cause.Report.Failures)}
	if cycle.ProposalOption != "" {
		if cycle.ProposalOption != factory.ReassessmentOptionAccept && cycle.ProposalOption != factory.ReassessmentOptionNeedsAttention || math.IsNaN(cycle.ProposalProbability) || math.IsInf(cycle.ProposalProbability, 0) || cycle.ProposalProbability < 0 || cycle.ProposalProbability > 1 || math.IsNaN(cycle.ProposalConfidence) || math.IsInf(cycle.ProposalConfidence, 0) || cycle.ProposalConfidence < 0 || cycle.ProposalConfidence > 1 {
			return nil, errors.New("repair progress has an invalid judgment")
		}
		view.Judgment = &factoryDeliveryRepairJudgment{Option: cycle.ProposalOption, Probability: cycle.ProposalProbability, Confidence: cycle.ProposalConfidence}
	}
	return view, nil
}

// apiFactoryRun maps one recorded run to its display status: recorded facts,
// display binding and observed host state. A run the host never recorded
// reports no state: admission is recorded before any host call, so absence
// means pending, not retired. Current code-write authority admits the read.
func (s *API) apiFactoryRun(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	run, p, ok := s.factoryActionRun(w, r, v)
	if !ok {
		return
	}
	status := factoryRunStatus{Run: factoryRunRecordDTO(run)}
	if run.ContextPending && !run.Reconciled && run.Outcome == "" {
		selection, selectionErr := control.ContextSelectionStatus(r.Context(), s.Store, run)
		if selectionErr != nil {
			auth.JSONError(w, 503, "context_unavailable", "Could not read the saved context hold.")
			return
		}
		status.ContextSelection = selection
	}
	if view, err := s.Store.FactoryRunView(r.Context(), run.ID); err == nil {
		if view.Repository != p.RepositoryID {
			auth.JSONError(w, 409, "view_conflict", "Run binding names another repository; refresh the inventory.")
			return
		}
		status.View = factoryRunBindingDTO(view)
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read run binding.")
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "host_unavailable", "Run observation is unavailable.")
		return
	}
	out, err := s.Host.FactoryInspect(r.Context(), project.FactoryInspect{Project: p.ID, ID: run.ID})
	if err == nil {
		status.State = factoryRunLiveDTO(out)
	} else if !errors.Is(err, host.ErrRunNotFound) {
		auth.JSONError(w, 503, "host_unavailable", "Run observation is unavailable.")
		return
	}
	if status.View != nil && status.View.Issue == "" {
		assignment, assignmentErr := s.Store.AssignmentByRun(r.Context(), run.ID)
		if assignmentErr != nil && !errors.Is(assignmentErr, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read verification assignment.")
			return
		}
		if assignmentErr == nil && assignment.ID == status.View.Attempt && assignment.Run == run.ID && assignment.ProjectID == p.ID && assignment.Repository == p.RepositoryID && assignment.Issue == 0 && assignment.Phase == project.ExecutionVerify && assignment.Role == project.RoleReviewer && assignment.Subject.Kind == factory.SubjectDelivery && assignment.Subject.Delivery != nil {
			available := false
			var retryErr error
			if s.Coordinator != nil {
				available, retryErr = s.Coordinator.DeliveryRetryAvailable(r.Context(), run.ID)
			}
			if retryErr != nil {
				auth.JSONError(w, 503, "store_unavailable", "Could not read verification Retry availability.")
				return
			}
			if available && s.Coordinator != nil {
				effective, authorityErr := s.Coordinator.EffectiveAuthority(r.Context(), p.RepositoryID)
				if authorityErr != nil {
					auth.JSONError(w, 503, "authority_unavailable", "Could not read current verification authority.")
					return
				}
				available = effective.Effective || len(effective.Missing) == 1 && effective.Missing[0] == factory.MissingDispatch
			} else if s.Coordinator == nil {
				available = false
			}
			status.VerificationRetry = &workCapabilityView{Available: available && run.Reconciled && (status.State == nil || !status.State.Live && status.State.Terminal)}
			status.DeliveryRepair = &factoryDeliveryRepairView{}
			{
				_, ownerErr := s.Store.IntentRevision(r.Context(), assignment.Subject.Delivery.IntentID, v.User.ID)
				if ownerErr != nil && !errors.Is(ownerErr, store.ErrNotFound) {
					auth.JSONError(w, 503, "store_unavailable", "Could not read application repair availability.")
					return
				}
				if ownerErr == nil {
					record, err := s.Store.DeliveryRepairRecord(r.Context(), p.RepositoryID, run.ID)
					if err == nil {
						status.DeliveryRepair.Progress, err = deliveryRepairProgressDTO(record, assignment)
					} else if errors.Is(err, store.ErrNotFound) {
						err = nil
						if s.Coordinator != nil {
							status.DeliveryRepair.Available, err = s.Coordinator.DeliveryRepairAvailable(r.Context(), run.ID)
						}
						status.DeliveryRepair.Available = status.DeliveryRepair.Available && run.Reconciled && (status.State == nil || !status.State.Live && status.State.Terminal)
					}
					if err != nil {
						auth.JSONError(w, 503, "store_unavailable", "Could not read application repair availability.")
						return
					}
				}
			}
		}
	}
	auth.JSONResponse(w, 200, status)
}
