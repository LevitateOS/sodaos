package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// AcceptanceSnapshotSource adapts the thin F-read transport to coordinator
// acceptance verification. It brackets the objective, every requested
// selected comment and the complete direct edge set through BracketedRead;
// transport failures arrive as acceptance refusals, never guessed evidence.
type AcceptanceSnapshotSource struct {
	Reader     forgejo.SnapshotReader
	Credential extensions.CredentialFile
}

// ReadAcceptanceEvidence brackets one native evidence read at a single idle
// revision. Reads are single-page: over-limit families refuse as incomplete
// instead of verifying decisions from a partial snapshot. The comments page
// covers the issue and is filtered to the requested IDs; unselected
// discussion is ignored and cannot grant scope.
func (s AcceptanceSnapshotSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (control.AcceptanceEvidence, error) {
	if s.Reader == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	req := forgejo.SnapshotRequest{
		RepositoryID: repository,
		IssueIndex:   issue,
		Families:     []forgejo.SnapshotFamily{forgejo.FamilyIssue, forgejo.FamilyDependencies},
		Limit:        forgejo.SnapshotPageLimit,
	}
	if len(commentIDs) != 0 {
		req.Families = append(req.Families, forgejo.FamilyComments)
	}
	snapshot, err := forgejo.BracketedRead(ctx, s.Reader, s.Credential, req)
	if err != nil {
		return control.AcceptanceEvidence{}, mapAcceptanceSnapshotError(err)
	}
	if snapshot.Issue == nil || snapshot.Dependencies == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	if len(commentIDs) != 0 && snapshot.Comments == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	evidence := control.AcceptanceEvidence{
		Revision: snapshot.Revision,
		Issue: control.AcceptanceIssueView{
			Index:         snapshot.Issue.Index,
			TitleDigest:   snapshot.Issue.TitleDigest,
			ContentDigest: snapshot.Issue.ContentDigest,
			PosterID:      snapshot.Issue.Provenance.PosterID,
			ContentVer:    snapshot.Issue.ContentVersion,
			Lifecycle:     len(snapshot.Issue.Lifecycle),
			ClosedUnix:    snapshot.Issue.ClosedUnix,
			Verified:      snapshot.Issue.Provenance.Verified,
			FirstCreated:  snapshot.Issue.Provenance.FirstCreated,
			Visible:       snapshot.Issue.Visible,
			Closed:        snapshot.Issue.IsClosed,
			IsPull:        snapshot.Issue.IsPull,
		},
	}
	if snapshot.Comments != nil {
		wanted := make(map[string]bool, len(commentIDs))
		for _, id := range commentIDs {
			wanted[id] = true
		}
		for _, item := range snapshot.Comments.Items {
			if !wanted[item.ID] {
				continue
			}
			evidence.Comments = append(evidence.Comments, control.AcceptanceComment{
				ID: item.ID, Digest: item.ContentDigest, ContentVer: item.ContentVersion, Visible: item.Visible,
			})
		}
	}
	for _, item := range snapshot.Dependencies.Items {
		evidence.Dependencies = append(evidence.Dependencies, control.AcceptanceEdge{
			Occurrence: item.OccurrenceID, DependsOn: item.DependencyID, Visible: item.Visible,
		})
	}
	return evidence, nil
}

// mapAcceptanceSnapshotError converts bracket failures to acceptance
// refusals. Stale, incomplete and busy brackets refuse without recording;
// malformed answers and transport failures report the source unavailable.
func mapAcceptanceSnapshotError(err error) error {
	switch {
	case errors.Is(err, forgejo.ErrStaleSnapshot):
		return &control.AcceptanceRefusal{Reason: control.RefusalStaleEvidence}
	case errors.Is(err, forgejo.ErrIncompleteSnapshot):
		return &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	case errors.Is(err, forgejo.ErrNativeBusy):
		return &control.AcceptanceRefusal{Reason: control.RefusalNativeBusy}
	case errors.Is(err, forgejo.ErrInvalidSnapshot):
		return &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	default:
		return err
	}
}

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
