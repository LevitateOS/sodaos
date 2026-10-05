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

// factoryIssue resolves the path repository and issue index under current
// native visibility. Absent or undisclosable targets report 404.
func (s *API) factoryIssue(w http.ResponseWriter, r *http.Request, v store.Session) (repository int64, issue string, ok bool) {
	repository, ok = s.factoryRepository(w, r, v)
	if !ok {
		return 0, "", false
	}
	index, valid := auth.PositiveID(r.PathValue("issueID"))
	if !valid {
		auth.JSONError(w, 404, "not_found", "Factory issue not found.")
		return 0, "", false
	}
	return repository, strconv.FormatInt(index, 10), true
}

type acceptanceSourceRequest struct {
	ID             string `json:"id"`
	Digest         string `json:"digest"`
	ContentVersion int    `json:"content_version"`
}

type acceptancePrerequisiteRequest struct {
	Occurrence       string `json:"occurrence"`
	DependsOn        string `json:"depends_on"`
	PrereqAcceptance string `json:"prerequisite_acceptance,omitempty"`
	EndpointRepo     string `json:"endpoint_repository"`
	EndpointIssue    string `json:"endpoint_issue"`
	Outcome          string `json:"outcome"`
}

type issueAcceptanceRequest struct {
	CommandID      string                          `json:"command_id"`
	DecisionID     string                          `json:"decision_id"`
	Predecessor    string                          `json:"predecessor,omitempty"`
	TitleDigest    string                          `json:"title_digest"`
	ContentDigest  string                          `json:"content_digest"`
	Sources        []acceptanceSourceRequest       `json:"sources,omitempty"`
	Prerequisites  []acceptancePrerequisiteRequest `json:"prerequisites,omitempty"`
	Resolutions    []acceptanceSourceRequest       `json:"resolutions,omitempty"`
	NativeRevision int64                           `json:"native_revision"`
	ContentVersion int                             `json:"content_version"`
}

// apiIssueAcceptances records a maintainer's exact-inputs acceptance for
// one native issue. Current code-write authority admits it; the approver
// is the host-admitted caller, never a body field. Native text stays the
// source: the decision selects exact revisions but edits nothing.
func (s *API) apiIssueAcceptances(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, issue, ok := s.factoryIssue(w, r, v)
	if !ok {
		return
	}
	if _, err := s.executionRepository(r, v, repository); err != nil {
		reportExecutionAuthorityError(w, err)
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in issueAcceptanceRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	decision, ok := acceptanceDecision(w, repository, issue, v.User.ID, in)
	if !ok {
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	receipt, err := s.Coordinator.AdmitAcceptance(r.Context(), in.CommandID, factoryPrincipal(v), decision)
	if err != nil {
		acceptanceDecisionError(w, err)
		return
	}
	auth.JSONResponse(w, 201, receipt)
}

func acceptanceDecision(w http.ResponseWriter, repository int64, issue string, approver int64, in issueAcceptanceRequest) (factory.Acceptance, bool) {
	sources, ok := acceptanceSources(w, in.Sources)
	if !ok {
		return factory.Acceptance{}, false
	}
	resolutions, ok := acceptanceSources(w, in.Resolutions)
	if !ok {
		return factory.Acceptance{}, false
	}
	var prerequisites []factory.AcceptedPrerequisite
	for _, p := range in.Prerequisites {
		repo, repoOK := auth.PositiveID(p.EndpointRepo)
		index, indexOK := auth.PositiveID(p.EndpointIssue)
		if !repoOK || !indexOK {
			auth.JSONError(w, 400, "invalid_prerequisite", "Prerequisites must name their endpoint repository and issue.")
			return factory.Acceptance{}, false
		}
		prerequisites = append(prerequisites, factory.AcceptedPrerequisite{
			Occurrence: p.Occurrence, DependsOn: p.DependsOn, PrereqAcceptance: p.PrereqAcceptance,
			EndpointRepo: repo, EndpointIssue: index, Outcome: p.Outcome,
		})
	}
	decision := factory.Acceptance{
		ID: in.DecisionID, Predecessor: in.Predecessor, Repository: repository, IssueIndex: issue,
		Approver: approver, NativeRev: in.NativeRevision,
		TitleDigest: in.TitleDigest, ContentDigest: in.ContentDigest, ContentVersion: in.ContentVersion,
		Sources: sources, Prerequisites: prerequisites, Resolutions: resolutions,
	}
	if decision.Validate() != nil {
		auth.JSONError(w, 400, "invalid_acceptance", "Acceptance must name its decision and exact selected revisions.")
		return factory.Acceptance{}, false
	}
	return decision, true
}

func acceptanceSources(w http.ResponseWriter, in []acceptanceSourceRequest) ([]factory.SelectedSource, bool) {
	var sources []factory.SelectedSource
	for _, s := range in {
		source := factory.SelectedSource{ID: s.ID, ContentVersion: s.ContentVersion, Digest: s.Digest}
		if source.Validate() != nil {
			auth.JSONError(w, 400, "invalid_source", "Selected sources must name an exact native comment revision.")
			return nil, false
		}
		sources = append(sources, source)
	}
	return sources, true
}

type issueWithdrawalRequest struct {
	CommandID  string `json:"command_id"`
	DecisionID string `json:"decision_id"`
}

// apiIssueWithdrawal records a current maintainer's explicit withdrawal
// of the head acceptance for one native issue. Withdrawing a superseded
// decision is stale; a later acceptance advances past the latch.
func (s *API) apiIssueWithdrawal(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, issue, ok := s.factoryIssue(w, r, v)
	if !ok {
		return
	}
	if _, err := s.executionRepository(r, v, repository); err != nil {
		reportExecutionAuthorityError(w, err)
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in issueWithdrawalRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	index, _ := strconv.ParseInt(issue, 10, 64)
	receipt, err := s.Coordinator.WithdrawAcceptance(r.Context(), in.CommandID, factoryPrincipal(v), repository, index, in.DecisionID, v.User.ID)
	if err != nil {
		acceptanceDecisionError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

// acceptanceDecisionError maps acceptance failures to status codes.
// Refused or changed inputs conflict with 409; unavailable evidence or
// service reports 503, never a validity verdict.
func acceptanceDecisionError(w http.ResponseWriter, err error) {
	var refusal *control.AcceptanceRefusal
	switch {
	case errors.As(err, &refusal):
		acceptanceRefusalError(w, refusal)
	default:
		factoryCommandError(w, err)
	}
}

func acceptanceRefusalError(w http.ResponseWriter, refusal *control.AcceptanceRefusal) {
	body := map[string]any{"error": refusal.Reason, "message": acceptanceRefusalMessage(refusal.Reason)}
	switch refusal.Reason {
	case control.RefusalIncompleteEvidence, control.RefusalNativeBusy, control.RefusalSnapshotUnavailable:
		auth.JSONResponse(w, 503, body)
	default:
		auth.JSONResponse(w, 409, body)
	}
}

func acceptanceRefusalMessage(reason string) string {
	switch reason {
	case control.RefusalStaleEvidence:
		return "Native state changed during verification; refresh and retry."
	case control.RefusalIncompleteEvidence:
		return "Native evidence is incomplete; nothing was recorded."
	case control.RefusalNativeBusy:
		return "Native writers are busy; retry after they settle."
	case control.RefusalSnapshotUnavailable:
		return "Native evidence is unavailable; nothing was recorded."
	case control.RefusalIssueHidden:
		return "The bound actor cannot see the target issue."
	case control.RefusalObjectiveChanged:
		return "The issue title or body changed; accept the new revision."
	case control.RefusalSourceMissing:
		return "A selected comment no longer exists; accept the current set."
	case control.RefusalSourceHidden:
		return "A selected comment is no longer visible."
	case control.RefusalSourceChanged:
		return "A selected comment changed; accept the new revision."
	case control.RefusalEdgeHidden:
		return "A prerequisite edge is no longer visible."
	case control.RefusalEdgeChanged:
		return "The prerequisite edge set changed; adopt the revised set."
	case control.RefusalPrereqUnknown:
		return "The named prerequisite acceptance does not exist."
	case control.RefusalPrereqStale:
		return "The prerequisite acceptance moved; adopt the current head."
	case control.RefusalCreationUnverified:
		return "Creation provenance is unverified; adopt explicitly."
	case control.RefusalCreationEdited:
		return "The issue changed after creation; adopt explicitly."
	case control.RefusalCreationBlocked:
		return "The issue has prerequisites; adopt explicitly."
	case control.RefusalCreationFactory:
		return "Factory-created issues need explicit human adoption."
	default:
		return "Creation needs standing policy and a code-write creator."
	}
}
