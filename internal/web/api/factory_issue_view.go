package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

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

// acceptancePreview exposes only the revision-bound public facts needed to
// construct an explicit human acceptance. Native issue text is never returned.
type acceptancePreview struct {
	Repository   string                        `json:"repository"`
	Issue        string                        `json:"issue"`
	TitleDigest  string                        `json:"title_digest"`
	TitleEvent   string                        `json:"title_occurrence"`
	BodyDigest   string                        `json:"body_digest"`
	ContentVer   int                           `json:"content_version"`
	NativeRev    int64                         `json:"native_revision"`
	Comments     []acceptancePreviewComment    `json:"comments"`
	Dependencies []acceptancePreviewDependency `json:"dependencies"`
}

type acceptancePreviewComment struct {
	ID         string `json:"id"`
	Digest     string `json:"digest"`
	ContentVer int    `json:"content_version"`
}

type acceptancePreviewDependency struct {
	Occurrence string `json:"occurrence"`
	DependsOn  string `json:"depends_on"`
}

// apiIssueAcceptancePreview is a bounded read for the native evidence that a
// code writer must review before submitting an explicit acceptance decision.
func (s *API) apiIssueAcceptancePreview(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, issue, commentIDs, ok := acceptancePreviewRequest(w, r, s, v)
	if !ok {
		return
	}
	if s.Coordinator == nil || s.Coordinator.AcceptanceReads == nil {
		auth.JSONError(w, 503, "native_evidence_unavailable", "Current native acceptance evidence is unavailable.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	evidence, err := s.Coordinator.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(repository, 10), issue, commentIDs)
	if err != nil {
		acceptancePreviewReadError(w, err)
		return
	}
	view, refusal := buildAcceptancePreview(repository, issue, commentIDs, evidence)
	if refusal != nil {
		acceptanceRefusalError(w, refusal)
		return
	}
	auth.JSONResponse(w, http.StatusOK, view)
}

func acceptancePreviewRequest(w http.ResponseWriter, r *http.Request, s *API, v store.Session) (int64, string, []string, bool) {
	if r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "Malformed acceptance preview query.")
		return 0, "", nil, false
	}
	commentIDs, valid := parseAcceptancePreviewCommentIDs(r.URL.RawQuery)
	if !valid {
		auth.JSONError(w, 400, "invalid_request", "Use at most 50 distinct positive comment_id values.")
		return 0, "", nil, false
	}
	repository, issue, ok := s.factoryIssue(w, r, v)
	if !ok {
		return 0, "", nil, false
	}
	if _, err := s.executionRepository(r, v, repository); err != nil {
		reportExecutionAuthorityError(w, err)
		return 0, "", nil, false
	}
	return repository, issue, commentIDs, true
}

func acceptancePreviewReadError(w http.ResponseWriter, err error) {
	var refusal *control.AcceptanceRefusal
	if errors.As(err, &refusal) {
		acceptanceRefusalError(w, refusal)
		return
	}
	auth.JSONError(w, 503, "native_evidence_unavailable", "Current native acceptance evidence is unavailable.")
}

func buildAcceptancePreview(repository int64, issue string, commentIDs []string, evidence control.AcceptanceEvidence) (acceptancePreview, *control.AcceptanceRefusal) {
	if refusal := validateAcceptancePreviewIssue(issue, evidence); refusal != nil {
		return acceptancePreview{}, refusal
	}
	view := acceptancePreview{
		Repository: strconv.FormatInt(repository, 10), Issue: issue,
		TitleDigest: evidence.Issue.TitleDigest, TitleEvent: evidence.Issue.TitleOccurrence,
		BodyDigest: evidence.Issue.ContentDigest, ContentVer: evidence.Issue.ContentVer,
		NativeRev:    evidence.Revision,
		Comments:     make([]acceptancePreviewComment, 0, len(commentIDs)),
		Dependencies: make([]acceptancePreviewDependency, 0, len(evidence.Dependencies)),
	}
	comments, refusal := acceptancePreviewComments(commentIDs, evidence.Comments)
	if refusal != nil {
		return acceptancePreview{}, refusal
	}
	dependencies, refusal := acceptancePreviewDependencies(evidence.Dependencies)
	if refusal != nil {
		return acceptancePreview{}, refusal
	}
	view.Comments, view.Dependencies = comments, dependencies
	return view, nil
}

func validateAcceptancePreviewIssue(issue string, evidence control.AcceptanceEvidence) *control.AcceptanceRefusal {
	if !evidence.Issue.Visible {
		return &control.AcceptanceRefusal{Reason: control.RefusalIssueHidden}
	}
	valid := evidence.Issue.Index == issue && evidence.Revision > 0 &&
		factory.ValidDigest(evidence.Issue.TitleDigest) && factory.ValidDigest(evidence.Issue.ContentDigest) &&
		factory.ValidTitleOccurrence(evidence.Issue.TitleOccurrence) && evidence.Issue.ContentVer >= 0
	if !valid {
		return &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	return nil
}

func acceptancePreviewComments(commentIDs []string, evidence []control.AcceptanceComment) ([]acceptancePreviewComment, *control.AcceptanceRefusal) {
	if len(evidence) != len(commentIDs) {
		return nil, &control.AcceptanceRefusal{Reason: control.RefusalSourceMissing}
	}
	wanted := make(map[string]bool, len(commentIDs))
	for _, id := range commentIDs {
		wanted[id] = true
	}
	comments := make([]acceptancePreviewComment, 0, len(commentIDs))
	for _, comment := range evidence {
		reason := acceptancePreviewCommentReason(comment, wanted)
		if reason != "" {
			return nil, &control.AcceptanceRefusal{Reason: reason}
		}
		delete(wanted, comment.ID)
		comments = append(comments, acceptancePreviewComment{ID: comment.ID, Digest: comment.Digest, ContentVer: comment.ContentVer})
	}
	if len(wanted) != 0 {
		return nil, &control.AcceptanceRefusal{Reason: control.RefusalSourceMissing}
	}
	return comments, nil
}

func acceptancePreviewCommentReason(comment control.AcceptanceComment, wanted map[string]bool) string {
	if !comment.Visible {
		return control.RefusalSourceHidden
	}
	id, valid := auth.PositiveID(comment.ID)
	valid = valid && strconv.FormatInt(id, 10) == comment.ID && wanted[comment.ID] && factory.ValidDigest(comment.Digest) && comment.ContentVer >= 0
	if !valid {
		return control.RefusalIncompleteEvidence
	}
	return ""
}

func acceptancePreviewDependencies(evidence []control.AcceptanceEdge) ([]acceptancePreviewDependency, *control.AcceptanceRefusal) {
	dependencies := make([]acceptancePreviewDependency, 0, len(evidence))
	for _, edge := range evidence {
		if !edge.Visible {
			return nil, &control.AcceptanceRefusal{Reason: control.RefusalEdgeHidden}
		}
		occurrence, valid := auth.PositiveID(edge.Occurrence)
		if !valid || strconv.FormatInt(occurrence, 10) != edge.Occurrence {
			return nil, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
		}
		dependsOn, valid := auth.PositiveID(edge.DependsOn)
		if !valid || strconv.FormatInt(dependsOn, 10) != edge.DependsOn {
			return nil, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
		}
		dependencies = append(dependencies, acceptancePreviewDependency{Occurrence: edge.Occurrence, DependsOn: edge.DependsOn})
	}
	return dependencies, nil
}

func parseAcceptancePreviewCommentIDs(raw string) ([]string, bool) {
	if len(raw) > 8192 {
		return nil, false
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, false
	}
	for key := range values {
		if key != "comment_id" {
			return nil, false
		}
	}
	ids := values["comment_id"]
	if len(ids) > factory.MaxAcceptedSources {
		return nil, false
	}
	seen := make(map[string]struct{}, len(ids))
	for i, value := range ids {
		id, ok := auth.PositiveID(value)
		if !ok {
			return nil, false
		}
		canonical := strconv.FormatInt(id, 10)
		if _, exists := seen[canonical]; exists {
			return nil, false
		}
		seen[canonical] = struct{}{}
		ids[i] = canonical
	}
	return ids, true
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
