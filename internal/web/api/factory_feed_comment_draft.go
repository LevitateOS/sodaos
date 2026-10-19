package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Product Lead comment authoring (R32.14 step 3, SUP-004 binding).
//
// There is no synchronous free-text LLM client in this codebase: the
// TypeSafe transport answers Choice judgments only (free, used for
// routing), and the one real paid LLM authoring transport is the
// intent-generation funding path — a funded decompose assignment
// answered by the enrolled host model account through
// AssembleIntentPlanningInput + DispatchIntentPlan, settled as a
// typed SemanticPlannerOutput. This drafter reuses exactly that
// path. Paid authoring accounting stays where the intent path puts
// it (the dispatch packet's intent attempt allowance and run usage),
// separate from free JEV judgment reservations. No new transport and
// no deterministic draft fabrication: when the planner offers no
// usable answer, drafting fails honestly and the pipeline waits.
//
// One comment source revision binds one durable draft intent through
// deterministic identities derived from the comment, so a retry, a
// lost response or a restart resumes the same intent and its single
// funded planning attempt instead of spending twice: the packet
// refuses a second funded attempt for the same intent/revision/owner
// triple, and this drafter reuses the recorded attempt whenever one
// exists. A multi-candidate answer refuses as ambiguous rather than
// silently taking the first candidate — choosing among semantic
// alternatives needs a funded judgment, and the accepted-direction
// chooser does not serve a draft comment intent (its own contract).
// An attention answer likewise refuses: the planner asked for a
// product decision a comment draft cannot invent.

// CommentDraftStore is the narrow store joint behind comment
// authoring. *store.Store implements it; tests stub it.
type CommentDraftStore interface {
	IntentRevision(ctx context.Context, intent string, owner int64) (factory.IntentRevision, error)
	SaveIntentRevision(ctx context.Context, r factory.IntentRevision) error
	IntentPlanAttempt(ctx context.Context, intentID, revisionID string) (factory.Assignment, error)
}

// CommentPlanTransport is the existing intent-generation funding and
// launch transport. *control.Coordinator implements it; tests stub
// it with a fake transport so authoring logic runs without enrolled
// model credentials.
type CommentPlanTransport interface {
	AssembleIntentPlanningInput(ctx context.Context, head factory.IntentRevision, operation string) (factory.SemanticPlannerInput, project.Preparation, *control.DispatchWait, error)
	DispatchIntentPlan(ctx context.Context, intentID string, owner int64, operation string, in factory.SemanticPlannerInput, source, preparationID string) control.DispatchReport
}

var _ CommentDraftStore = (*store.Store)(nil)
var _ CommentPlanTransport = (*control.Coordinator)(nil)
var _ CommentDrafter = (*IntentCommentDrafter)(nil)

// ErrCommentDraftPending reports a launched planning run that has not
// settled yet. The attempt keeps running
// under its own funding; the next drive resumes it without spending
// again.
var ErrCommentDraftPending = errors.New("comment planning run still settling")

// IntentCommentDrafter authors issue drafts from feed comments
// through the funded Product Lead planning path.
type IntentCommentDrafter struct {
	Store     CommentDraftStore
	Transport CommentPlanTransport
	// Now overrides the clock for tests.
	Now func() time.Time
}

// commentDecisionID derives one deterministic decision identity for
// a comment-bound record, so replays converge on the same intent,
// revision and idea instead of minting new ones.
func commentDecisionID(kind, comment string, revision int64) string {
	sum := sha256.Sum256([]byte("soda-comment-intent-v1:" + kind + ":" + comment + ":" + strconv.FormatInt(revision, 10)))
	return "d" + hex.EncodeToString(sum[:])[:24]
}

// commentSourceContext renders the recorded work context one
// comment was submitted in — its resolved source snapshot and/or its
// bound issue scope — as planner-facing evidence text. Only
// server-recorded facts render; an unbound top-level comment has no
// context to add.
func commentSourceContext(comment store.FeedCommentDetail) string {
	var facts []string
	if source := comment.Source; source != nil {
		fact := "Submitted against " + source.Kind + " " + source.ID
		if source.Issue > 0 {
			fact += " tracking issue #" + strconv.FormatInt(source.Issue, 10)
		}
		if source.PR > 0 {
			fact += " via pull request #" + strconv.FormatInt(source.PR, 10)
		}
		if source.Status != "" {
			fact += " (status " + source.Status + ")"
		}
		if source.Summary != "" {
			fact += ": " + source.Summary
		}
		facts = append(facts, fact)
	}
	if comment.Record.Issue > 0 && (comment.Source == nil || comment.Source.Issue != comment.Record.Issue) {
		facts = append(facts, "Bound to native issue #"+strconv.FormatInt(comment.Record.Issue, 10)+".")
	}
	return strings.Join(facts, " ")
}

// commentDraftIntent builds the durable draft intent one comment
// source revision authors against: the human request as one accepted
// requested idea, bounded to a single derived issue. The framing
// consequence states only what the pipeline itself does with the
// request; every semantic word of the draft comes from the planner.
// When the submission carried recorded work context, one additional
// pending evidence-supported idea states that context so the planner
// authors inside the actual project work; it is never an effective
// outcome and candidate tasks never have to cover it.
func commentDraftIntent(comment store.FeedCommentDetail, author int64, createdUnix int64) (factory.IntentRevision, error) {
	body := comment.Body
	if len(body) > 4000 {
		return factory.IntentRevision{}, errors.New("comment body exceeds the planning idea bound")
	}
	commentID, revision := comment.Record.ID, comment.Record.Revision
	ideaID := commentDecisionID("idea", commentID, revision)
	ideas := []factory.ProductIdea{{
		ID: ideaID, Description: body,
		Consequence: "The requested change is tracked as work and completed.",
		Decision:    "accept", Source: "requested", Provenance: "human", Actor: author,
	}}
	if context := commentSourceContext(comment); context != "" {
		ideas = append(ideas, factory.ProductIdea{
			ID: commentDecisionID("source", commentID, revision), Description: context,
			Consequence: "Authored work fits alongside this existing tracked work.",
			Decision:    "pending", Source: "supported", Provenance: "autonomous", Evidence: context,
		})
	}
	head := factory.IntentRevision{
		ID:         commentDecisionID("revision", commentID, revision),
		Intent:     commentDecisionID("intent", commentID, revision),
		Owner:      author,
		Repository: comment.Record.Repository,
		Objective:  body,
		Ideas:      ideas,
		State:      "draft", MaxIssues: 1, CreatedUnix: createdUnix,
		EffectiveOutcomes: []string{ideaID},
	}
	if err := head.Validate(); err != nil {
		return factory.IntentRevision{}, err
	}
	return head, nil
}

// ensureCommentDraftIntent loads the comment's durable draft intent,
// saving it first when this drive is its first authoring attempt. A
// concurrent save of the identical deterministic revision replays
// through the read-back instead of failing the drive.
func (d *IntentCommentDrafter) ensureCommentDraftIntent(ctx context.Context, comment store.FeedCommentDetail, author int64) (factory.IntentRevision, error) {
	intentID := commentDecisionID("intent", comment.Record.ID, comment.Record.Revision)
	if head, err := d.Store.IntentRevision(ctx, intentID, author); err == nil {
		return head, nil
	} else if !errors.Is(err, store.ErrNotFound) && !isNoRowsIntent(err) {
		return factory.IntentRevision{}, err
	}
	created := comment.At
	if created <= 0 {
		created = d.now().Unix()
	}
	head, err := commentDraftIntent(comment, author, created)
	if err != nil {
		return factory.IntentRevision{}, err
	}
	if err := d.Store.SaveIntentRevision(ctx, head); err != nil {
		if saved, readErr := d.Store.IntentRevision(ctx, intentID, author); readErr == nil {
			return saved, nil
		}
		return factory.IntentRevision{}, err
	}
	return head, nil
}

// isNoRowsIntent reports the store's raw no-head read: IntentRevision
// surfaces sql.ErrNoRows unwrapped when no head exists yet.
func isNoRowsIntent(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (d *IntentCommentDrafter) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// DraftCommentIssue authors one issue draft from a routed comment:
// ensure the comment-bound draft intent, reuse or launch its one
// funded planning attempt, return pending until settlement, then map
// the single assessed candidate task to the issue draft. The caller
// (driveCommentDraft) validates the draft before any native effect
// and settles malformed drafts invalid with their bytes wiped.
func (d *IntentCommentDrafter) DraftCommentIssue(ctx context.Context, comment store.FeedCommentDetail) (factory.IssueDraft, error) {
	var empty factory.IssueDraft
	if d == nil || d.Store == nil || d.Transport == nil {
		return empty, errors.New("comment authoring transport is unavailable")
	}
	author, err := strconv.ParseInt(comment.Record.Author, 10, 64)
	if err != nil || author <= 0 {
		return empty, errors.New("comment author is not a native owner")
	}
	if comment.Record.Repository <= 0 || strings.TrimSpace(comment.Body) == "" {
		return empty, errors.New("comment draft needs its repository and body")
	}
	head, err := d.ensureCommentDraftIntent(ctx, comment, author)
	if err != nil {
		return empty, err
	}
	attempt, err := d.commentPlanAttempt(ctx, head, author)
	if err != nil {
		return empty, err
	}
	// The coordinator also settles this run. Returning pending lets its
	// next progress pass do that work instead of waiting inside its lock.
	if attempt.Stage != factory.AssignmentFinished || attempt.Result == nil {
		return empty, ErrCommentDraftPending
	}
	switch attempt.Outcome {
	case factory.Failed:
		return empty, fmt.Errorf("comment planning run failed: %s", attempt.Result.Summary)
	case factory.Cancelled:
		return empty, fmt.Errorf("comment planning run was cancelled: %s", attempt.Result.Summary)
	}
	return commentDraftFromAttempt(comment, head, attempt)
}

// commentPlanAttempt returns the recorded planning attempt behind
// the comment intent, launching it through the funded transport when
// none exists yet. A launch that lands as a wait or a duplicate
// refusal reconciles through the recorded attempt instead of
// launching again.
func (d *IntentCommentDrafter) commentPlanAttempt(ctx context.Context, head factory.IntentRevision, author int64) (factory.Assignment, error) {
	var empty factory.Assignment
	if attempt, err := d.Store.IntentPlanAttempt(ctx, head.Intent, head.ID); err == nil {
		return attempt, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return empty, err
	}
	in, prep, wait, err := d.Transport.AssembleIntentPlanningInput(ctx, head, factory.IntentPlanningPropose)
	if err != nil {
		return empty, err
	}
	if wait != nil {
		return empty, fmt.Errorf("%w: %s", ErrCommentDraftPending, wait.Detail)
	}
	report := d.Transport.DispatchIntentPlan(ctx, head.Intent, author, factory.IntentPlanningPropose, in, prep.SourceCommit, prep.ID)
	if len(report.Errors) != 0 {
		return empty, errors.New(report.Errors[0].Detail)
	}
	attempt, err := d.Store.IntentPlanAttempt(ctx, head.Intent, head.ID)
	if err != nil {
		if len(report.Waits) != 0 {
			return empty, fmt.Errorf("%w: %s", ErrCommentDraftPending, report.Waits[0].Detail)
		}
		return empty, err
	}
	return attempt, nil
}

// commentDraftFromAttempt maps one settled planning attempt to the
// issue draft its single assessed candidate authors. The answer is
// re-assessed against its exact staged input here, so replaying the
// same recorded attempt reaches the same draft and spends nothing.
// Attention, multi-candidate and multi-task answers refuse: none can
// become one issue draft without an authority this stage lacks. A
// mapped draft that exceeds the issue-draft bounds returns as-is so
// the caller's validation settles it invalid before any effect.
func commentDraftFromAttempt(comment store.FeedCommentDetail, head factory.IntentRevision, attempt factory.Assignment) (factory.IssueDraft, error) {
	var empty factory.IssueDraft
	if attempt.Subject.Intent == nil || attempt.Subject.Intent.OwnerID != head.Owner {
		return empty, errors.New("recorded comment plan answers another owner")
	}
	in, staged, err := control.ParseStagedPlanningInput(attempt.Prompt)
	if err != nil || !staged {
		return empty, errors.New("recorded comment planning input is missing or corrupt")
	}
	if in.Head.Intent != head.Intent || in.Head.ID != head.ID {
		return empty, errors.New("recorded comment plan answers another intent revision")
	}
	result := attempt.Result
	if result == nil || !result.Reported || result.PhaseResult == nil ||
		result.PhaseResult.Status != "completed" || result.PhaseResult.Output == nil ||
		result.PhaseResult.Output.Decomposition == nil {
		return empty, errors.New("recorded comment plan carries no completed decomposition")
	}
	out := *result.PhaseResult.Output.Decomposition
	if out.Attention != "" {
		return empty, fmt.Errorf("comment planning needs a product decision first: %s", out.Attention)
	}
	if err := factory.AssessSemanticOutput(in, out); err != nil {
		return empty, err
	}
	if len(out.Candidates) != 1 {
		return empty, fmt.Errorf("comment planning offered %d candidates; choosing among them needs a funded judgment", len(out.Candidates))
	}
	candidate := out.Candidates[0]
	if len(candidate.Tasks) != 1 {
		return empty, fmt.Errorf("comment planning offered %d tasks for one issue", len(candidate.Tasks))
	}
	task := candidate.Tasks[0]
	fields := map[string]string{
		"source": "feed_comment", "intent": head.Intent, "revision": head.ID,
		"candidate": candidate.Key, "task": task.Key,
	}
	if comment.Source != nil {
		fields["source_kind"], fields["source_id"] = comment.Source.Kind, comment.Source.ID
		if comment.Source.Issue > 0 {
			fields["source_issue"] = strconv.FormatInt(comment.Source.Issue, 10)
		}
	}
	metadata, err := json.Marshal(fields)
	if err != nil {
		return empty, err
	}
	return factory.IssueDraft{
		Comment:    comment.Record.ID,
		Revision:   comment.Record.Revision,
		Title:      commentDraftTitle(task.Objective),
		Body:       commentDraftBody(task, candidate.Rationale),
		Objective:  task.Objective,
		Acceptance: strings.Join(task.Outcomes, "\n"),
		Metadata:   string(metadata),
	}, nil
}

// commentDraftTitle labels one draft from the authored task
// objective: the objective itself within the title bound, otherwise
// its longest whole-word prefix that fits. The full objective stays
// in the draft objective and body verbatim.
func commentDraftTitle(objective string) string {
	title := strings.Join(strings.Fields(objective), " ")
	if len(title) <= 256 {
		return title
	}
	trimmed := title[:256]
	if cut := strings.LastIndex(trimmed, " "); cut > 0 {
		trimmed = trimmed[:cut]
	}
	return strings.TrimSpace(trimmed)
}

// commentDraftBody renders the authored issue body from the planner
// output alone: the task objective, the candidate rationale and the
// outcome list. The creator appends the source-comment footer.
func commentDraftBody(task factory.IntentTask, rationale string) string {
	var b strings.Builder
	b.WriteString(task.Objective)
	if strings.TrimSpace(rationale) != "" {
		b.WriteString("\n\n" + rationale)
	}
	if len(task.Outcomes) != 0 {
		b.WriteString("\n\nOutcomes:")
		for _, outcome := range task.Outcomes {
			b.WriteString("\n- " + outcome)
		}
	}
	return b.String()
}
