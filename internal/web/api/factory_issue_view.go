package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type factoryIssueView struct {
	Status     control.AcceptanceValidity `json:"status"`
	Repository string                     `json:"repository"`
	Issue      string                     `json:"issue"`
	Readiness  *readinessView             `json:"readiness,omitempty"`
	Checks     *checksView                `json:"checks,omitempty"`
	Merge      *mergeView                 `json:"merge,omitempty"`
}

// readinessView is the durable readiness verdict for one native issue:
// its classification, primary reason, structured blockers, assessed
// acceptance and record revision. It carries IDs and codes only.
type readinessView struct {
	Blockers   []factory.Blocker `json:"blockers,omitempty"`
	Acceptance string            `json:"acceptance,omitempty"`
	Readiness  string            `json:"readiness"`
	Reason     string            `json:"reason"`
	Revision   int64             `json:"revision"`
	Assessed   int64             `json:"assessed_unix"`
}

// checkResultView renders one required check's verdict: its latest
// observed native state on the exact head and whether it passes.
type checkResultView struct {
	Context string `json:"context"`
	State   string `json:"state,omitempty"`
	Passed  bool   `json:"passed"`
}

// checksView is the latest check assessment for one native PR: whether
// the exact head and verified base satisfy every configured required
// check, with the per-check results behind the verdict. It carries IDs,
// commits, codes and observed states only.
type checksView struct {
	Results    []checkResultView `json:"results"`
	Resolution string            `json:"resolution"`
	HeadOID    string            `json:"head_oid"`
	BaseOID    string            `json:"base_oid"`
	Verdict    string            `json:"verdict"`
	Reason     string            `json:"reason"`
	PRNumber   string            `json:"pr_number"`
	PRID       string            `json:"pr_id"`
	Policy     int64             `json:"policy_revision"`
	NativeRev  int64             `json:"native_revision"`
	Revision   int64             `json:"revision"`
	Assessed   int64             `json:"assessed_unix"`
}

// checksViewDTO renders one recorded assessment. Absence of a record
// leaves the issue without a verdict, never a guessed pass.
func checksViewDTO(a factory.CheckAssessment) *checksView {
	view := &checksView{
		Resolution: factory.CheckResolution(a.Reason),
		HeadOID:    a.HeadOID, BaseOID: a.BaseOID,
		Verdict: a.Verdict, Reason: a.Reason,
		PRNumber: strconv.FormatInt(a.PRNumber, 10), PRID: strconv.FormatInt(a.PRID, 10),
		Policy: a.PolicyRevision, NativeRev: a.NativeRev, Revision: a.Revision, Assessed: a.AssessedUnix,
	}
	for _, result := range a.Results {
		view.Results = append(view.Results, checkResultView{Context: result.Context, State: result.State, Passed: result.Passed})
	}
	return view
}

// mergeView is the latest merge for one repository issue: its stage,
// bounded reason, exact PR linkage and confirmed completion stamps. It
// carries IDs, commits and codes only.
type mergeView struct {
	Resolution string `json:"resolution"`
	HeadOID    string `json:"head_oid"`
	BaseOID    string `json:"base_oid"`
	Commit     string `json:"merged_commit,omitempty"`
	Stage      string `json:"stage"`
	Reason     string `json:"reason,omitempty"`
	PRNumber   string `json:"pr_number"`
	PRID       string `json:"pr_id"`
	Revision   int64  `json:"revision"`
	Finished   int64  `json:"finished_unix,omitempty"`
}

// mergeViewDTO renders one recorded merge. Absence of a record leaves
// the issue without a merge verdict, never a guessed completion.
func mergeViewDTO(m factory.Merge) *mergeView {
	return &mergeView{
		Resolution: factory.MergeResolution(m.Reason),
		HeadOID:    m.HeadOID, BaseOID: m.BaseOID, Commit: m.MergedCommit,
		Stage: m.Stage, Reason: m.Reason,
		PRNumber: strconv.FormatInt(m.PRNumber, 10), PRID: strconv.FormatInt(m.PRID, 10),
		Revision: m.Revision, Finished: m.FinishedUnix,
	}
}

// apiFactoryIssue shows the current acceptance for one native issue and
// whether it is still valid, plus the recorded readiness and check
// verdicts. Visibility of the repository authorizes the read; the
// validity assessment brackets fresh native evidence.
func (s *API) apiFactoryIssue(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	repository, issue, ok := s.factoryIssue(w, r, v)
	if !ok {
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	status, err := s.Coordinator.AcceptanceStatus(r.Context(), repository, issue)
	if err != nil {
		acceptanceDecisionError(w, err)
		return
	}
	view := factoryIssueView{Status: status, Repository: strconv.FormatInt(repository, 10), Issue: issue}
	index, _ := strconv.ParseInt(issue, 10, 64)
	if assessed, err := s.Store.IssueControl(r.Context(), repository, index); err == nil {
		view.Readiness = &readinessView{
			Blockers: assessed.Blockers, Acceptance: assessed.Acceptance,
			Readiness: assessed.Readiness, Reason: assessed.Reason,
			Revision: assessed.Revision, Assessed: assessed.AssessedUnix,
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue readiness.")
		return
	}
	if assessed, err := s.Store.CheckAssessment(r.Context(), repository, index); err == nil {
		view.Checks = checksViewDTO(assessed)
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read check assessment.")
		return
	}
	if merged, err := s.Store.MergeForIssue(r.Context(), repository, index); err == nil {
		view.Merge = mergeViewDTO(merged)
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue merge.")
		return
	}
	auth.JSONResponse(w, 200, view)
}
