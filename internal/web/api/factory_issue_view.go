package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// validNativeLinkPart reports whether a native owner or repository name
// can appear in a local link path. It rejects traversal, separators,
// query/fragment markers and whitespace/control characters.
func validNativeLinkPart(part string) bool {
	return part != "" && part != "." && part != ".." && !strings.ContainsAny(part, "/\\%?#") && !strings.ContainsFunc(part, func(c rune) bool { return unicode.IsSpace(c) || unicode.IsControl(c) })
}

type factoryIssueView struct {
	Status       control.AcceptanceValidity `json:"status"`
	Attempt      *workAttemptView           `json:"attempt,omitempty"`
	Publication  *workPublicationView       `json:"publication,omitempty"`
	Closure      workClosureView            `json:"closure"`
	Capabilities workCapabilitiesView       `json:"capabilities"`
	Requested    *workRequestedView         `json:"requested,omitempty"`
	Links        *workLinksView             `json:"links,omitempty"`
	Repository   string                     `json:"repository"`
	Issue        string                     `json:"issue"`
	Journey      *factoryIssueJourneyView   `json:"journey"`
	Readiness    *readinessView             `json:"readiness,omitempty"`
	Checks       *checksView                `json:"checks,omitempty"`
	Merge        *mergeView                 `json:"merge,omitempty"`
	ObservedUnix int64                      `json:"observed_unix"`
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
// commits, codes and observed states only. Stale marks an assessment
// whose head is outside the publication's known candidate history.
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
	Stale      bool              `json:"stale,omitempty"`
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
// carries IDs, commits and codes only. Stale marks an open merge whose
// head is outside the publication's known candidate history; terminal
// merges stand on the head they finished with.
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
	Stale      bool   `json:"stale,omitempty"`
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

// workAttemptView is the latest publication-line dispatch attempt for one
// native issue: its role, stage and run binding. Prompt bytes and held
// capacity stay on the assignment route; this view carries identity and
// stage only. Absence of a record means no attempt was dispatched yet.
type workAttemptView struct {
	Outcome      factory.Outcome `json:"outcome,omitempty"`
	ID           string          `json:"id"`
	Role         string          `json:"role"`
	Stage        string          `json:"stage"`
	Reason       string          `json:"reason,omitempty"`
	Run          string          `json:"run"`
	Attempts     int             `json:"attempts"`
	CreatedUnix  int64           `json:"created_unix"`
	FinishedUnix int64           `json:"finished_unix,omitempty"`
}

func workAttemptDTO(a factory.Assignment) *workAttemptView {
	return &workAttemptView{
		Outcome: a.Outcome,
		ID:      a.ID, Role: a.Role, Stage: a.Stage, Reason: a.Reason, Run: a.Run,
		Attempts: a.Attempts, CreatedUnix: a.CreatedUnix, FinishedUnix: a.FinishedUnix,
	}
}

// workCorrectionView renders one native branch correction: the candidate
// it published and its operation effect. Receipt bytes stay out; the
// candidate commit identifies the corrected content.
type workCorrectionView struct {
	Candidate   string `json:"candidate,omitempty"`
	Effect      string `json:"effect,omitempty"`
	Completion  string `json:"completion,omitempty"`
	Reason      string `json:"reason,omitempty"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// workReviewView renders one independent review run: the exact head it
// judged, its disposition and its native outcome. Review prose stays on
// the native PR; this view carries the disposition and effect only.
type workReviewView struct {
	Run        string `json:"run"`
	HeadOID    string `json:"head_oid"`
	Event      string `json:"event"`
	Effect     string `json:"effect,omitempty"`
	Completion string `json:"completion,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// maxWorkHistoryEntries bounds the rendered correction and review tails.
// Totals carry the full recorded counts; only the most recent entries
// render inline.
const maxWorkHistoryEntries = 20

// workPublicationView renders one attempt's publication: the exact linked
// PR once creation commits, the candidate and base it binds, and the
// bounded correction and review history behind it.
type workPublicationView struct {
	factoryAssignmentPublication
	Corrections      []workCorrectionView `json:"corrections,omitempty"`
	Reviews          []workReviewView     `json:"reviews,omitempty"`
	Candidate        string               `json:"candidate"`
	BaseSHA          string               `json:"base_sha"`
	TargetBranch     string               `json:"target_branch"`
	ID               string               `json:"id"`
	Run              string               `json:"run"`
	CorrectionsTotal int                  `json:"corrections_total"`
	ReviewsTotal     int                  `json:"reviews_total"`
	CreatedUnix      int64                `json:"created_unix"`
	FinishedUnix     int64                `json:"finished_unix,omitempty"`
}

func workPublicationDTO(a factory.Assignment, p factory.Publication) *workPublicationView {
	rendered := factoryAssignmentDTO(a, nil, &p)
	view := &workPublicationView{
		factoryAssignmentPublication: *rendered.Publication,
		Candidate:                    p.Candidate, BaseSHA: p.BaseSHA, TargetBranch: p.TargetBranch,
		ID: p.ID, Run: p.Run,
		CorrectionsTotal: len(p.Corrections), ReviewsTotal: len(p.ReviewOperations),
		CreatedUnix: p.CreatedUnix, FinishedUnix: p.FinishedUnix,
	}
	corrections := p.Corrections
	if len(corrections) > maxWorkHistoryEntries {
		corrections = corrections[len(corrections)-maxWorkHistoryEntries:]
	}
	for _, correction := range corrections {
		entry := workCorrectionView{
			Effect: correction.Effect, Completion: correction.Completion,
			Reason: correction.Reason, UpdatedUnix: correction.UpdatedUnix,
		}
		if correction.Work != nil {
			entry.Candidate = correction.Work.Candidate
		}
		view.Corrections = append(view.Corrections, entry)
	}
	reviews := p.ReviewOperations
	if len(reviews) > maxWorkHistoryEntries {
		reviews = reviews[len(reviews)-maxWorkHistoryEntries:]
	}
	for _, review := range reviews {
		view.Reviews = append(view.Reviews, workReviewView{
			Run: review.RunID, HeadOID: review.Work.HeadOID, Event: review.Work.Event,
			Effect: review.Outcome.Effect, Completion: review.Outcome.Completion,
			Reason: review.Outcome.Reason,
		})
	}
	return view
}

// workCandidateHistory collects the publication's known candidate commits:
// its initial candidate plus every committed correction. Review, check
// and merge evidence names the head it judged; heads outside this set
// are superseded.
func workCandidateHistory(p factory.Publication) map[string]bool {
	history := map[string]bool{}
	if p.Publish.Work != nil {
		history[p.Publish.Work.Candidate] = true
	} else if p.Candidate != "" {
		history[p.Candidate] = true
	}
	for _, correction := range p.Corrections {
		if correction.Effect == factory.OpEffectCommitted && correction.Work != nil && correction.Work.Candidate != "" {
			history[correction.Work.Candidate] = true
		}
	}
	delete(history, "")
	return history
}

// workClosureStates name the original-issue closure facts. Pending means
// a PR exists but original closure is unconfirmed; confirmed means the
// merge recorded the closing occurrence and its timestamp.
const (
	workClosureNone      = "none"
	workClosurePending   = "pending"
	workClosureConfirmed = "confirmed"
)

type workClosureView struct {
	Occurrence string `json:"occurrence,omitempty"`
	State      string `json:"state"`
	ClosedUnix int64  `json:"closed_unix,omitempty"`
}

func workClosureDTO(publication *factory.Publication, merged *factory.Merge) workClosureView {
	if merged != nil && merged.Stage == factory.MergeMerged && merged.OriginalClosedUnix > 0 && merged.OriginalClosingOccurrence != "" {
		return workClosureView{Occurrence: merged.OriginalClosingOccurrence, State: workClosureConfirmed, ClosedUnix: merged.OriginalClosedUnix}
	}
	if publication != nil && publication.PRNumber > 0 {
		return workClosureView{State: workClosurePending}
	}
	if merged != nil && merged.PRNumber > 0 {
		return workClosureView{State: workClosurePending}
	}
	return workClosureView{State: workClosureNone}
}

// workCapabilityReasons are the bounded codes behind one action
// capability. Authority codes describe the viewer; state codes describe
// the recorded work. Admission revalidates at act time.
const (
	workCapWriteRequired = "repository_write_required"
	workCapNotVisible    = "repository_not_visible"
	workCapNoAuthority   = "native_authority_unavailable"
	workCapSession       = "session_changed"
	workCapNoRun         = "no_run"
	workCapUnsettled     = "run_unsettled"
	workCapSettled       = "run_settled"
	workCapRetryQueued   = "retry_queued"
	workCapMembership    = "membership_required"
	workCapDispatchOpen  = "dispatch_open"
	workCapDispatchShut  = "dispatch_closed"
)

type workCapabilityView struct {
	Reason    string `json:"reason,omitempty"`
	Available bool   `json:"available"`
}

// workCapabilitiesView reports the viewer's current intervention powers
// for one issue: read-only watch, run stop/retry/takeover and repository
// pause/resume. Each entry carries its refusal reason when unavailable;
// the owning lifecycle APIs remain authoritative at act time.
type workCapabilitiesView struct {
	Watch    workCapabilityView `json:"watch"`
	Stop     workCapabilityView `json:"stop"`
	Retry    workCapabilityView `json:"retry"`
	Takeover workCapabilityView `json:"takeover"`
	Pause    workCapabilityView `json:"pause"`
	Resume   workCapabilityView `json:"resume"`
}

type workCapabilityInput struct {
	AuthorityReason string
	HasRun          bool
	RunSettled      bool
	RetryQueued     bool
	Member          bool
	DispatchOpen    bool
	SessionCurrent  bool
}

func workCapabilityGates(in workCapabilityInput) (mutating, attempt string) {
	mutating = in.AuthorityReason
	if mutating == "" && !in.SessionCurrent {
		mutating = workCapSession
	}
	attempt = mutating
	if attempt == "" {
		switch {
		case !in.HasRun:
			attempt = workCapNoRun
		case !in.RunSettled:
			attempt = workCapUnsettled
		}
	}
	return mutating, attempt
}

func workStopCapability(in workCapabilityInput, mutating string) workCapabilityView {
	view := workCapabilityView{Available: mutating == "" && in.HasRun && !in.RunSettled}
	switch {
	case mutating != "":
		view.Reason = mutating
	case !in.HasRun:
		view.Reason = workCapNoRun
	case in.RunSettled:
		view.Reason = workCapSettled
	}
	return view
}

func workRetryCapability(in workCapabilityInput, attempt string) workCapabilityView {
	view := workCapabilityView{Available: attempt == "" && !in.RetryQueued}
	switch {
	case attempt != "":
		view.Reason = attempt
	case in.RetryQueued:
		view.Reason = workCapRetryQueued
	}
	return view
}

func workTakeoverCapability(in workCapabilityInput, attempt string) workCapabilityView {
	view := workCapabilityView{Available: attempt == "" && in.Member}
	switch {
	case attempt != "":
		view.Reason = attempt
	case !in.Member:
		view.Reason = workCapMembership
	}
	return view
}

func workPauseCapability(in workCapabilityInput, mutating string) workCapabilityView {
	view := workCapabilityView{Available: mutating == "" && in.DispatchOpen}
	switch {
	case mutating != "":
		view.Reason = mutating
	case !in.DispatchOpen:
		view.Reason = workCapDispatchShut
	}
	return view
}

func workResumeCapability(in workCapabilityInput, mutating string) workCapabilityView {
	view := workCapabilityView{Available: mutating == "" && !in.DispatchOpen}
	switch {
	case mutating != "":
		view.Reason = mutating
	case in.DispatchOpen:
		view.Reason = workCapDispatchOpen
	}
	return view
}

func workCapabilitiesDTO(in workCapabilityInput) workCapabilitiesView {
	view := workCapabilitiesView{}
	view.Watch = workCapabilityView{Available: in.AuthorityReason == ""}
	if in.AuthorityReason != "" {
		view.Watch.Reason = in.AuthorityReason
	}
	mutating, attempt := workCapabilityGates(in)
	view.Stop = workStopCapability(in, mutating)
	view.Retry = workRetryCapability(in, attempt)
	view.Takeover = workTakeoverCapability(in, attempt)
	view.Pause = workPauseCapability(in, mutating)
	view.Resume = workResumeCapability(in, mutating)
	return view
}

// workRetryRequestView renders a queued explicit retry: the requested
// control, distinct from the confirmed attempt outcome it follows.
type workRetryRequestView struct {
	CommandID string `json:"command_id"`
	Reason    string `json:"reason"`
	Prior     string `json:"prior_run"`
	Queued    bool   `json:"queued"`
}

// workRequestedView carries requested controls that are not confirmed
// effects. Empty means no control request is outstanding.
type workRequestedView struct {
	Retry *workRetryRequestView `json:"retry,omitempty"`
}

// workLinksView carries safe native links for one issue and its exact
// linked PR. Links render only when the viewer's native repository
// metadata supplies a safe path; otherwise they are omitted.
type workLinksView struct {
	Issue string `json:"issue_url"`
	Pull  string `json:"pull_url,omitempty"`
}

// apiFactoryIssue shows the current acceptance for one native issue and
// whether it is still valid, plus the recorded readiness, attempt,
// publication, review, check, merge and closure facts and the viewer's
// current action capabilities. The validity assessment brackets fresh
// native evidence. ObservedUnix marks when the view was assembled;
// absent optional sections mean no record exists yet, never a guess.
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
	acceptance := ""
	if assessed, err := s.Store.IssueControl(r.Context(), repository, index); err == nil {
		acceptance = assessed.Acceptance
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
	var merged *factory.Merge
	if record, err := s.Store.MergeForIssue(r.Context(), repository, index); err == nil {
		merged = &record
		view.Merge = mergeViewDTO(record)
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue merge.")
		return
	}
	var attempt *factory.Assignment
	if current, err := s.Store.LatestIssueAssignment(r.Context(), repository, index); err == nil {
		attempt = &current
		view.Attempt = workAttemptDTO(current)
	} else if !errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue attempt.")
		return
	}
	var publication *factory.Publication
	if attempt != nil {
		if record, err := s.Store.PublicationByAssignment(r.Context(), attempt.ID); err == nil {
			publication = &record
			view.Publication = workPublicationDTO(*attempt, record)
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read attempt publication.")
			return
		}
	}
	applyWorkStaleness(&view, publication, merged)
	journey, err := s.factoryIssueJourney(r.Context(), repository, index, issue, attempt, publication, time.Now().Unix())
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read issue journey.")
		return
	}
	view.Journey = journey
	view.Closure = workClosureDTO(publication, merged)
	capabilities, requested, ok := s.factoryIssueCapabilities(w, r, v, repository, attempt, acceptance)
	if !ok {
		return
	}
	view.Capabilities, view.Requested = capabilities, requested
	view.Links = s.factoryIssueLinks(r, v, repository, issue, publication, merged)
	view.ObservedUnix = time.Now().Unix()
	auth.JSONResponse(w, 200, view)
}

// applyWorkStaleness marks check and open-merge evidence whose head is
// outside the publication's known candidate history. Terminal merges
// stand on the head they finished with and are never marked stale.
func applyWorkStaleness(view *factoryIssueView, publication *factory.Publication, merged *factory.Merge) {
	if publication == nil {
		return
	}
	history := workCandidateHistory(*publication)
	if len(history) == 0 {
		return
	}
	if view.Checks != nil && !history[view.Checks.HeadOID] {
		view.Checks.Stale = true
	}
	if view.Merge != nil && merged != nil && merged.Stage == factory.MergeOpen && !history[view.Merge.HeadOID] {
		view.Merge.Stale = true
	}
}

// factoryIssueCapabilities derives the viewer's current intervention
// powers from native authority, session currency and durable records.
// It reads only; the owning lifecycle APIs revalidate at act time.
func (s *API) factoryIssueCapabilities(w http.ResponseWriter, r *http.Request, v store.Session, repository int64, attempt *factory.Assignment, acceptance string) (workCapabilitiesView, *workRequestedView, bool) {
	var capabilities workCapabilitiesView
	in := workCapabilityInput{SessionCurrent: s.extensionSessionCurrent(r.Context(), r, v)}
	if _, err := s.executionRepository(r, v, repository); err != nil {
		switch {
		case errors.Is(err, errRepositoryWriteRequired):
			in.AuthorityReason = workCapWriteRequired
		case errors.Is(err, errRepositoryDenied):
			in.AuthorityReason = workCapNotVisible
		default:
			in.AuthorityReason = workCapNoAuthority
		}
	}
	var requested *workRequestedView
	if attempt != nil {
		run, err := s.Store.FactoryRun(r.Context(), attempt.Run)
		if err == nil {
			in.HasRun, in.RunSettled = true, run.Reconciled
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read attempt run.")
			return capabilities, nil, false
		}
		if acceptance != "" {
			if decision, err := s.Store.PendingExplicitRetry(r.Context(), repository, attempt.Issue, acceptance); err == nil {
				in.RetryQueued = true
				requested = &workRequestedView{Retry: &workRetryRequestView{
					CommandID: decision.CommandID, Reason: decision.Reason,
					Prior: decision.Prior, Queued: decision.Queued,
				}}
			} else if !errors.Is(err, store.ErrNotFound) && !errors.Is(err, store.ErrExplicitRetryIneligible) {
				auth.JSONError(w, 503, "store_unavailable", "Could not read pending retry.")
				return capabilities, nil, false
			}
		}
		if _, err := s.Store.MemberLogin(r.Context(), attempt.ProjectID, v.User.ID); err == nil {
			in.Member = true
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read membership.")
			return capabilities, nil, false
		}
	}
	open, _, _, err := s.Store.DispatchState(r.Context(), repository)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read dispatch state.")
		return capabilities, nil, false
	}
	in.DispatchOpen = open
	return workCapabilitiesDTO(in), requested, true
}

// factoryIssueLinks builds safe native links from the viewer's current
// repository metadata. Links are omitted when that metadata is
// unavailable; the read itself never fails for missing links.
func (s *API) factoryIssueLinks(r *http.Request, v store.Session, repository int64, issue string, publication *factory.Publication, merged *factory.Merge) *workLinksView {
	access, err := s.visibleRepository(r, v, repository)
	if err != nil || !access.native {
		return nil
	}
	owner, name := access.repository.Owner.Login, access.repository.Name
	if !validNativeLinkPart(owner) || !validNativeLinkPart(name) {
		return nil
	}
	base := "/" + owner + "/" + name
	links := &workLinksView{Issue: base + "/issues/" + issue}
	pr := int64(0)
	if publication != nil {
		pr = publication.PRNumber
	} else if merged != nil {
		pr = merged.PRNumber
	}
	if pr > 0 {
		links.Pull = base + "/pulls/" + strconv.FormatInt(pr, 10)
	}
	return links
}
