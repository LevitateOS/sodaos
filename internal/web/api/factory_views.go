package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
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
	Run   factoryRunRecord   `json:"run"`
	View  *factoryRunBinding `json:"view,omitempty"`
	State *factoryRunLive    `json:"state,omitempty"`
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
	auth.JSONResponse(w, 200, status)
}
