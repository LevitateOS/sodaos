package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Feed communication contract (R32.2/R32.14, R42.2): durable comment
// identity/revision/scope plus the intake/judgment/draft/native-effect
// pipeline behind one comment POST. The comment body is a human or
// worker submission; every derived fact (route, draft, issue link) is
// server-recorded with the exact source revision bound through each
// stage, so a retry recovers the original instead of duplicating it.
// Reads never spend or dispatch: funding applies to judgment calls
// only, and native effects apply through the issue/graph creator only.

// Comment routing outcomes from the funded JEV Choice assessment.
const (
	// CommentRouteNewIssue routes the comment to Product Lead LLM
	// issue authoring followed by validated native creation.
	CommentRouteNewIssue = "new_issue"
	// CommentRouteExistingWork links the comment to already tracked work.
	CommentRouteExistingWork = "existing_work"
	// CommentRouteDiscussion keeps the comment as discussion only: no
	// issue is ever created from it.
	CommentRouteDiscussion = "discussion"
	// CommentRouteClarify marks the comment as waiting on its author's
	// clarification before any further routing.
	CommentRouteClarify = "clarify"
)

// Comment pipeline states. Intake, judgment, draft and native effect
// advance separately so call admission, funding, comment save,
// judgment, draft and native commit stay distinguishable.
const (
	CommentIntakeRecorded  = "recorded"
	CommentJudgmentPending = "pending"
	CommentJudgmentApplied = "applied"
	CommentJudgmentFailed  = "failed"
	CommentDraftPending    = "pending"
	CommentDraftApplied    = "applied"
	CommentDraftInvalid    = "invalid"
	CommentEffectPending   = "pending"
	CommentEffectApplied   = "applied"
	CommentEffectVerified  = "verified"
	CommentEffectFailed    = "failed"
	CommentEffectBlocked   = "blocked"
)

// CommentAttemptLimit bounds funded judgment/draft/native retries behind
// one comment revision: a lost effect reply reconciles by re-reading
// native state, never by blind repeated spending.
const CommentAttemptLimit = 6

// CommentBodyLimit bounds one comment submission.
const CommentBodyLimit = 4096

var commentID = regexp.MustCompile(`^[a-f0-9]{32}$`)

var commentKey = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Comment source kinds: the feed entry a comment was submitted
// against. The server resolves the named entry inside the caller's
// visible repository at submission time and persists the resolved
// snapshot; caller-supplied source facts never enter judgment or
// authoring directly.
const (
	// CommentSourceRecord names one canonical workflow record entry
	// (a factory assignment rendered into the feed).
	CommentSourceRecord = "record"
	// CommentSourcePost names one stored communication post entry.
	CommentSourcePost = "post"
)

// CommentRequest is one comment POST body: the repository scope, the
// submission text, the optional bound issue and reply parent, the
// optional source record/post the comment answers, and the
// caller-chosen idempotency key. The author is the host-admitted
// caller, never a body field.
type CommentRequest struct {
	Repository     int64  `json:"repository"`
	Body           string `json:"body"`
	Issue          int64  `json:"issue,omitempty"`
	ReplyTo        string `json:"reply_to,omitempty"`
	SourceKind     string `json:"source_kind,omitempty"`
	SourceID       string `json:"source_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Validate admits one comment request with its scope, bounded body and
// idempotency key.
func (r CommentRequest) Validate() error {
	if r.Repository <= 0 {
		return errors.New("comment needs its repository scope")
	}
	body := strings.TrimSpace(r.Body)
	if body == "" || utf8.RuneCountInString(r.Body) > CommentBodyLimit {
		return errors.New("comment needs its bounded body")
	}
	if r.Issue < 0 {
		return errors.New("comment needs its non-negative issue scope")
	}
	if r.ReplyTo != "" && !commentID.MatchString(r.ReplyTo) {
		return errors.New("comment needs its known reply parent")
	}
	if (r.SourceKind == "") != (r.SourceID == "") {
		return errors.New("comment source needs its kind and identity together")
	}
	if r.SourceKind != "" && r.SourceKind != CommentSourceRecord && r.SourceKind != CommentSourcePost {
		return errors.New("comment needs its known source kind")
	}
	if r.SourceID != "" && !commentID.MatchString(r.SourceID) {
		return errors.New("comment needs its known source identity")
	}
	if !commentKey.MatchString(r.IdempotencyKey) {
		return errors.New("comment needs its idempotency key")
	}
	return nil
}

// CommentSourceSummaryLimit bounds one persisted source summary: the
// same preview bound the feed serving layer applies to records.
const CommentSourceSummaryLimit = 512

// CommentSource is the server-resolved snapshot of the feed entry one
// comment was submitted against: its kind and stable identity, the
// bound native issue and pull request where recorded, the produced
// candidate ref, its recorded status and a bounded summary preview.
// It is resolved inside the caller's visible repository at
// submission time and persisted with the comment, so judgment and
// authoring consume authorized recorded evidence instead of
// caller-asserted facts or a later re-read of moved work.
type CommentSource struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Issue   int64  `json:"issue,omitempty"`
	PR      int64  `json:"pr,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Status  string `json:"status,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// Validate admits one resolved comment source snapshot.
func (s CommentSource) Validate() error {
	if s.Kind != CommentSourceRecord && s.Kind != CommentSourcePost {
		return errors.New("comment source needs its known kind")
	}
	if !commentID.MatchString(s.ID) {
		return errors.New("comment source needs its stable identity")
	}
	if s.Issue < 0 || s.PR < 0 {
		return errors.New("comment source needs its non-negative issue and pull request")
	}
	if len(s.Ref) > 256 || len(s.Status) > 128 || len([]rune(s.Summary)) > CommentSourceSummaryLimit {
		return errors.New("comment source evidence outside its bound")
	}
	return nil
}

// Comment work candidate origins, strongest binding first.
const (
	CommentCandidateBound  = "bound"
	CommentCandidateSource = "source"
	CommentCandidateParent = "parent"
	CommentCandidateRecent = "recent"
)

// CommentWorkCandidate is one already-tracked issue the routing
// judgment may link a comment to: the native issue identity plus a
// bounded recorded summary and status, and how the candidate reached
// the judgment (bound issue scope, source snapshot, reply parent or
// recent repository work).
type CommentWorkCandidate struct {
	Title        string `json:"title,omitempty"`
	Objective    string `json:"objective,omitempty"`
	WorkerStatus string `json:"execution_status,omitempty"`
	Issue        int64  `json:"issue"`
	Summary      string `json:"summary,omitempty"`
	Status       string `json:"status,omitempty"`
	Origin       string `json:"origin"`
}

// Validate admits one existing-work candidate.
func (c CommentWorkCandidate) Validate() error {
	if c.Issue <= 0 {
		return errors.New("comment work candidate needs its issue")
	}
	switch c.Origin {
	case CommentCandidateBound, CommentCandidateSource, CommentCandidateParent, CommentCandidateRecent:
	default:
		return errors.New("comment work candidate needs its known origin")
	}
	if len([]rune(c.Summary)) > CommentSourceSummaryLimit || len(c.Status) > 128 || len([]rune(c.Objective)) > 4000 || len(c.WorkerStatus) > 128 {
		return errors.New("comment work candidate evidence outside its bound")
	}
	return nil
}

// CommentRecord is one durable communication record: stable identity,
// per-edit revision, scope and authorship. Edits add revisions; the
// identity survives. Deleted or revoked comments keep their tombstone
// so bound pipeline rows still resolve.
type CommentRecord struct {
	ID         string `json:"id"`
	Revision   int64  `json:"revision"`
	Repository int64  `json:"repository"`
	Author     string `json:"author"`
	Issue      int64  `json:"issue,omitempty"`
	ReplyTo    string `json:"reply_to,omitempty"`
	Edited     bool   `json:"edited"`
	Deleted    bool   `json:"deleted"`
	Revoked    bool   `json:"revoked"`
}

// Validate admits one stored comment record.
func (r CommentRecord) Validate() error {
	if !commentID.MatchString(r.ID) {
		return errors.New("comment record needs its stable identity")
	}
	if r.Revision < 0 {
		return errors.New("comment record needs its non-negative revision")
	}
	if r.Repository <= 0 {
		return errors.New("comment record needs its repository scope")
	}
	if r.Author == "" {
		return errors.New("comment record needs its author")
	}
	if r.Issue < 0 {
		return errors.New("comment record needs its non-negative issue scope")
	}
	if r.ReplyTo != "" && !commentID.MatchString(r.ReplyTo) {
		return errors.New("comment record needs its known reply parent")
	}
	return nil
}

// CommentPipeline is the durable per-comment request row: the bound
// source revision plus the separate intake/judgment/draft/effect
// states, the assessed route, the linked native issue and the bounded
// attempt count shared by judgment, draft and native retries.
type CommentPipeline struct {
	Comment  string `json:"comment"`
	Revision int64  `json:"revision"`
	Intake   string `json:"intake"`
	Judgment string `json:"judgment"`
	Route    string `json:"route,omitempty"`
	Draft    string `json:"draft"`
	Effect   string `json:"effect"`
	Issue    int64  `json:"issue"`
	Attempts int    `json:"attempts"`
	Resume   bool   `json:"resume,omitempty"`
}

// CommentDecisionID binds authoring records to the original comment revision.
func CommentDecisionID(kind, comment string, revision int64) string {
	sum := sha256.Sum256([]byte("soda-comment-intent-v1:" + kind + ":" + comment + ":" + strconv.FormatInt(revision, 10)))
	return "d" + hex.EncodeToString(sum[:])[:24]
}

// Validate admits one pipeline row with known states and a bounded
// attempt count.
func (p CommentPipeline) Validate() error {
	if !commentID.MatchString(p.Comment) {
		return errors.New("comment pipeline needs its comment identity")
	}
	if p.Revision < 0 {
		return errors.New("comment pipeline needs its non-negative source revision")
	}
	if p.Intake != CommentIntakeRecorded {
		return errors.New("comment pipeline needs its recorded intake")
	}
	switch p.Judgment {
	case CommentJudgmentPending, CommentJudgmentApplied, CommentJudgmentFailed:
	default:
		return errors.New("comment pipeline needs its known judgment state")
	}
	switch p.Route {
	case "", CommentRouteNewIssue, CommentRouteExistingWork, CommentRouteDiscussion, CommentRouteClarify:
	default:
		return errors.New("comment pipeline needs its known route")
	}
	switch p.Draft {
	case CommentDraftPending, CommentDraftApplied, CommentDraftInvalid:
	default:
		return errors.New("comment pipeline needs its known draft state")
	}
	switch p.Effect {
	case CommentEffectPending, CommentEffectApplied, CommentEffectVerified, CommentEffectFailed, CommentEffectBlocked:
	default:
		return errors.New("comment pipeline needs its known effect state")
	}
	if p.Issue < 0 || p.Attempts < 0 || p.Attempts > CommentAttemptLimit {
		return errors.New("comment pipeline needs its bounded issue and attempts")
	}
	if p.Resume && (p.Attempts != CommentAttemptLimit || p.Judgment != CommentJudgmentApplied || p.Route != CommentRouteNewIssue) {
		return errors.New("comment resumption needs its exhausted applied new-work route")
	}
	return nil
}

// AdmitCommentJudgment reports whether one funded judgment call may run
// for the pipeline: intake recorded, judgment still pending, attempts
// remaining. Reads and re-serves never pass this gate.
func (p CommentPipeline) AdmitCommentJudgment() bool {
	return p.Intake == CommentIntakeRecorded && p.Judgment == CommentJudgmentPending && p.Attempts < CommentAttemptLimit
}

// AdmitCommentDraft reports whether Product Lead LLM authoring may run:
// the new_issue route judged in, attempts remaining, no applied draft.
func (p CommentPipeline) AdmitCommentDraft() bool {
	return p.Judgment == CommentJudgmentApplied && p.Route == CommentRouteNewIssue &&
		p.Draft == CommentDraftPending && (p.Attempts < CommentAttemptLimit || p.Resume)
}

func (p CommentPipeline) RetryAllowed() bool {
	if p.Effect == CommentEffectApplied || p.Effect == CommentEffectVerified || p.Resume {
		return false
	}
	return p.Attempts >= CommentAttemptLimit || p.Judgment == CommentJudgmentFailed || p.Draft == CommentDraftInvalid || p.Effect == CommentEffectFailed || p.Route == CommentRouteClarify || p.Route == CommentRouteExistingWork
}

// AdmitCommentEffect reports whether the native issue/graph effect may
// run: draft applied, effect not yet settled, attempts remaining.
func (p CommentPipeline) AdmitCommentEffect() bool {
	if p.Judgment != CommentJudgmentApplied || p.Route != CommentRouteNewIssue || p.Draft != CommentDraftApplied {
		return false
	}
	return (p.Effect == CommentEffectPending || p.Effect == CommentEffectFailed) && (p.Attempts < CommentAttemptLimit || p.Resume)
}

// CommentEffectSettled reports whether the native outcome is final:
// verified, applied, or blocked (deleted/revoked source, lost access).
// Only an unsettled effect may retry or reconcile.
func (p CommentPipeline) CommentEffectSettled() bool {
	return p.Effect == CommentEffectVerified || p.Effect == CommentEffectApplied || p.Effect == CommentEffectBlocked
}

// IssueDraft is one Product Lead LLM-authored issue proposal bound to
// its exact comment source revision: title, body, objective and
// acceptance plus the Soda work metadata the native creation validates.
type IssueDraft struct {
	Comment    string `json:"comment"`
	Revision   int64  `json:"revision"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Objective  string `json:"objective"`
	Acceptance string `json:"acceptance"`
	Metadata   string `json:"metadata"`
}

// Validate admits one issue draft with its source binding and complete
// authoring: no field may be blank and malformed drafts refuse before
// any native effect.
func (d IssueDraft) Validate() error {
	if !commentID.MatchString(d.Comment) {
		return errors.New("issue draft needs its comment source")
	}
	if d.Revision < 0 {
		return errors.New("issue draft needs its non-negative source revision")
	}
	if strings.TrimSpace(d.Title) == "" || len(d.Title) > 256 {
		return errors.New("issue draft needs its bounded title")
	}
	if strings.TrimSpace(d.Body) == "" || len(d.Body) > 16384 {
		return errors.New("issue draft needs its bounded body")
	}
	if strings.TrimSpace(d.Objective) == "" || len(d.Objective) > 2048 {
		return errors.New("issue draft needs its bounded objective")
	}
	if strings.TrimSpace(d.Acceptance) == "" || len(d.Acceptance) > 4096 {
		return errors.New("issue draft needs its bounded acceptance")
	}
	if strings.TrimSpace(d.Metadata) == "" || len(d.Metadata) > 2048 {
		return errors.New("issue draft needs its work metadata")
	}
	return nil
}

// CommentResponse is one comment POST answer: the durable identity and
// revision plus the separate pipeline states and the issue link once
// the native effect verifies. A retry returns the same identity with
// its current states instead of a duplicate comment. POST answers as
// soon as the comment and its intake are durable; judgment, draft and
// effect continue service-side and are re-read through the comment
// GET. ExistingIssue names the already-tracked issue an
// existing_work judgment selected (the same identity Issue carries
// on that route); Source echoes the persisted resolved source
// snapshot when the submission named one.
type CommentResponse struct {
	Attempts      int            `json:"attempts"`
	AttemptLimit  int            `json:"attempt_limit"`
	Resume        bool           `json:"resume,omitempty"`
	Comment       string         `json:"comment_id"`
	Revision      int64          `json:"revision"`
	Intake        string         `json:"intake"`
	Judgment      string         `json:"judgment"`
	Route         string         `json:"route,omitempty"`
	Draft         string         `json:"draft"`
	Effect        string         `json:"effect"`
	Issue         int64          `json:"issue,omitempty"`
	ExistingIssue int64          `json:"existing_issue,omitempty"`
	Source        *CommentSource `json:"source,omitempty"`
}
