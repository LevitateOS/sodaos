package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Feed comment backend (R32.2/R32.14): one comment POST persists the
// comment identity, revision and scope plus its durable intake request
// and answers promptly with the stored pipeline states. The funded JEV
// routing, Product Lead LLM draft and native effect advance
// service-side behind the stored request rows (DrivePendingComments,
// bound to the coordinator lifecycle), never inside the browser
// request: a request that times out or a service that restarts resumes
// the same stored request exactly once, and the comment GET re-reads
// the settled outcome. Unwired stages wait truthfully in pending:
// nothing spends and nothing dispatches without its collaborator.
// Reads never drive: GET serves stored state only. No per-comment
// approvals exist; a discussion route only ever records discussion.

// CommentJudgment is one routing outcome: the assessed route plus,
// for existing_work, the already-tracked issue the judgment selected.
type CommentJudgment struct {
	Route    string
	Issue    int64
	Question string
}

var commentClarificationQuestions = map[string]string{
	"clarify_target":   "Which app, page, or existing issue should this request apply to?",
	"clarify_result":   "What should change, and what should the result look like?",
	"clarify_conflict": "Which of the conflicting requirements should take precedence?",
}

// CommentParentEvidence carries the recorded reply-parent facts one
// judgment may weigh: the parent comment identity, a bounded preview
// of its stored text and the native issue its own pipeline linked.
type CommentParentEvidence struct {
	Comment string
	Body    string
	Issue   int64
}

// CommentJudgeInput carries everything one routing judgment may
// weigh: the comment with its persisted source snapshot, its reply
// parent's recorded evidence, and the existing-work candidates.
type CommentJudgeInput struct {
	Comment    store.FeedCommentDetail
	Parent     *CommentParentEvidence
	Candidates []factory.CommentWorkCandidate
}

// CommentJudge assesses one comment through the funded JEV Choice
// transport and returns the judgment with its funding receipt bound
// to the given snapshot.
type CommentJudge interface {
	JudgeComment(ctx context.Context, input CommentJudgeInput, snapshot string) (CommentJudgment, factory.PriorityReceipt, error)
}

// CommentDrafter authors one issue draft from a comment through the
// Product Lead LLM: title, body, objective and acceptance plus the Soda
// work metadata the native creation validates.
type CommentDrafter interface {
	DraftCommentIssue(ctx context.Context, comment store.FeedCommentDetail) (factory.IssueDraft, error)
}

// CommentIssueCreator validates and creates native issues from drafts
// through the existing native issue/graph ops, and looks up the issue
// already created from a comment so a lost effect reply reconciles
// instead of duplicating.
type CommentIssueCreator interface {
	CreateCommentIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error)
	LookupCommentIssue(ctx context.Context, repository int64, commentID string) (int64, error)
}

// CommentCollaborators carries the wired Lane B stages. Nil stages wait
// in pending; the driver never spends or dispatches through a nil.
type CommentCollaborators struct {
	Judge   CommentJudge
	Drafter CommentDrafter
	Creator CommentIssueCreator
}

// commentRouteRubric is the single JEV Choice rubric behind comment
// routing: code supplies the complete comment state — body, bound
// issue scope, resolved source snapshot, reply-parent evidence and
// the existing-work candidates — and the explicit outcomes.
const commentRouteRubric = "Route one feed comment to exactly one outcome: " +
	"new_issue when it requests a concrete change that needs new tracked work; " +
	"existing_<issue> for one listed candidate when the comment belongs to that already tracked work; " +
	"discussion when it is an acknowledgement or discussion that must never create an issue; " +
	"clarify_target when the target is ambiguous, clarify_result when the desired result is unclear, or clarify_conflict when requirements conflict. " +
	"Match the requested deliverable against actual issue objectives. Project membership, recent activity, generic worker status, or being a reply do not establish an objective match. " +
	"Use existing work for the same deliverable or a correction of that deliverable; a distinct requested change needs new work even when the comment replies to an older post."

// commentExistingChoice names the Choice key selecting one
// existing-work candidate issue.
func commentExistingChoice(issue int64) string {
	return "existing_" + strconv.FormatInt(issue, 10)
}

// commentChoiceKeys lists the admitted Choice keys for one judgment:
// the three base routes plus one key per existing-work candidate.
// With no candidates there is no existing-work key to select, so a
// comment can never be routed to work the server never listed.
func commentChoiceKeys(candidates []factory.CommentWorkCandidate) []string {
	keys := []string{factory.CommentRouteNewIssue, factory.CommentRouteDiscussion, "clarify_target", "clarify_result", "clarify_conflict"}
	for _, candidate := range candidates {
		keys = append(keys, commentExistingChoice(candidate.Issue))
	}
	return keys
}

// commentJudgmentFromSelection parses one recorded Choice selection
// into its route and selected issue. Only selections the choice
// actually offered admit: an existing-work key outside the candidate
// set refuses instead of linking unlisted work.
func commentJudgmentFromSelection(selected string, candidates []factory.CommentWorkCandidate) (CommentJudgment, error) {
	if question, ok := commentClarificationQuestions[selected]; ok {
		return CommentJudgment{Route: factory.CommentRouteClarify, Question: question}, nil
	}
	switch selected {
	case factory.CommentRouteNewIssue, factory.CommentRouteDiscussion, factory.CommentRouteClarify:
		return CommentJudgment{Route: selected}, nil
	case factory.CommentRouteExistingWork:
		// Receipts recorded before candidate-bound choices selected
		// the bare route with no issue identity; replay keeps that
		// recorded outcome instead of re-judging it.
		return CommentJudgment{Route: selected}, nil
	}
	for _, candidate := range candidates {
		if selected == commentExistingChoice(candidate.Issue) {
			return CommentJudgment{Route: factory.CommentRouteExistingWork, Issue: candidate.Issue}, nil
		}
	}
	return CommentJudgment{}, errors.New("comment judgment selected an outcome it was never offered")
}

// commentCandidateIssues extracts the candidate set a recorded
// selection may resolve against when replaying a stored receipt:
// the candidates are recovered from the receipt's own candidate
// identities, so replay never depends on today's candidate list.
func commentCandidatesFromReceipt(receipt factory.PriorityReceipt) []factory.CommentWorkCandidate {
	var candidates []factory.CommentWorkCandidate
	for _, candidate := range receipt.Candidates {
		issue, ok := strings.CutPrefix(candidate.ID, "existing_")
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(issue, 10, 64)
		if err == nil && n > 0 {
			candidates = append(candidates, factory.CommentWorkCandidate{Issue: n, Origin: factory.CommentCandidateRecent})
		}
	}
	return candidates
}

// commentJudgeState renders the complete supplied state for one
// routing Choice: the comment, its persisted source snapshot, its
// reply parent's recorded evidence and the existing-work candidates.
func commentJudgeState(input CommentJudgeInput) map[string]any {
	comment := input.Comment
	state := map[string]any{
		"repository": comment.Record.Repository, "issue": comment.Record.Issue,
		"reply_to": comment.Record.ReplyTo, "body": comment.Body,
	}
	if comment.Source != nil {
		state["source"] = map[string]any{
			"kind": comment.Source.Kind, "id": comment.Source.ID,
			"issue": comment.Source.Issue, "pr": comment.Source.PR,
			"ref": comment.Source.Ref, "status": comment.Source.Status,
			"summary": comment.Source.Summary,
		}
	}
	if input.Parent != nil {
		state["parent"] = map[string]any{
			"comment": input.Parent.Comment, "body": input.Parent.Body,
			"issue": input.Parent.Issue,
		}
	}
	candidates := make([]map[string]any, 0, len(input.Candidates))
	for _, candidate := range input.Candidates {
		candidates = append(candidates, map[string]any{
			"issue": candidate.Issue, "summary": candidate.Summary,
			"status": candidate.Status, "origin": candidate.Origin,
			"title": candidate.Title, "objective": candidate.Objective, "execution_status": candidate.WorkerStatus,
		})
	}
	state["candidates"] = candidates
	return state
}

// TypeSafeCommentJudge routes comments through the existing funded
// JEV Choice transport. Funding is reserved by the driver before this
// call; this method grants no authority and performs no dispatch.
type TypeSafeCommentJudge struct {
	Priority *control.TypeSafePriority
	Prepare  func(context.Context, CommentJudgeInput) (CommentJudgeInput, error)
}

func (j *TypeSafeCommentJudge) PrepareCommentInput(ctx context.Context, in CommentJudgeInput) (CommentJudgeInput, error) {
	if j.Prepare == nil {
		return in, nil
	}
	return j.Prepare(ctx, in)
}

// JudgeComment assesses one comment through one funded Choice call.
func (j *TypeSafeCommentJudge) JudgeComment(ctx context.Context, input CommentJudgeInput, snapshot string) (CommentJudgment, factory.PriorityReceipt, error) {
	var empty CommentJudgment
	if j == nil || j.Priority == nil {
		return empty, factory.PriorityReceipt{}, control.ErrPriorityUnavailable
	}
	keys := commentChoiceKeys(input.Candidates)
	choices := make(map[string]any, len(keys))
	for _, key := range keys {
		switch key {
		case factory.CommentRouteNewIssue:
			choices[key] = "A concrete new deliverable or change that no listed issue already covers."
		case factory.CommentRouteDiscussion:
			choices[key] = "Conversation or acknowledgement without a request to change the product."
		case factory.CommentRouteClarify:
			choices[key] = "The desired result cannot be determined from the comment and supplied context."
		default:
			if question, ok := commentClarificationQuestions[key]; ok {
				choices[key] = "The comment requires an answer to: " + question
			} else {
				choices[key] = "Continue the specific listed issue only if its objective covers this requested result."
			}
		}
	}
	answer, err := j.Priority.Select(ctx, commentJudgeState(input), commentRouteRubric, choices)
	if err != nil {
		return empty, factory.PriorityReceipt{}, err
	}
	request := factory.PriorityRequest{Snapshot: snapshot, Rubric: commentRouteRubric}
	for _, key := range keys {
		request.Candidates = append(request.Candidates, factory.PriorityCandidate{ID: key})
	}
	if err := answer.Validate(request); err != nil {
		return empty, factory.PriorityReceipt{}, err
	}
	judgment, err := commentJudgmentFromSelection(answer.Selected, input.Candidates)
	if err != nil {
		return empty, factory.PriorityReceipt{}, err
	}
	receipt := factory.PriorityReceipt{
		Snapshot: snapshot, Candidates: request.Candidates,
		Calls:    []factory.PriorityCall{{Request: request, Answer: answer}},
		Selected: answer.Selected, Outcome: "routed", CreatedUnix: time.Now().Unix(),
	}
	return judgment, receipt, nil
}

// parseFeedCommentBody admits one comment POST body: scope, bounded
// text and idempotency key. The author is the host-admitted caller.
func parseFeedCommentBody(data []byte) (factory.CommentRequest, error) {
	var req factory.CommentRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return factory.CommentRequest{}, errFeedQueryInvalid
	}
	if err := req.Validate(); err != nil {
		return factory.CommentRequest{}, err
	}
	return req, nil
}

// apiFactoryFeedComment saves one comment and answers promptly with
// its durable identity and stored pipeline states. It never drives
// the pipeline: judgment, authoring and the native effect continue
// service-side behind the stored request (DrivePendingComments), so
// a browser timeout loses nothing and the outcome is re-read through
// the comment GET. A named source entry is resolved inside the
// caller's visible repository before anything is saved; unknown or
// cross-repository sources refuse without disclosure. A retried
// submission recovers the original comment and its current states
// instead of duplicating them.
func (s *API) apiFactoryFeedComment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method != http.MethodPost {
		auth.JSONError(w, 405, "invalid_request", "Feed comments post only.")
		return
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		auth.JSONError(w, 400, "invalid_request", "Feed comment body is invalid.")
		return
	}
	req, err := parseFeedCommentBody(raw)
	if err != nil {
		auth.JSONError(w, 400, "invalid_request", "Feed comment body is invalid.")
		return
	}
	access, err := s.visibleRepository(r, v, req.Repository)
	if err != nil {
		auth.JSONError(w, 404, "repository_not_found", "Repository is not visible.")
		return
	}
	_ = access
	if req.ReplyTo != "" {
		parent, err := s.Store.FeedComment(r.Context(), req.ReplyTo)
		if err != nil || parent.Record.Repository != req.Repository || parent.Record.Deleted || parent.Record.Revoked {
			auth.JSONError(w, 404, "source_not_found", "The comment you are replying to is no longer visible.")
			return
		}
	}
	var source *factory.CommentSource
	if req.SourceKind != "" {
		resolved, err := s.Store.ResolveCommentSource(r.Context(), req.Repository, req.SourceKind, req.SourceID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				auth.JSONError(w, 404, "source_not_found", "Comment source is not visible.")
			} else {
				auth.JSONError(w, 503, "store_unavailable", "Could not resolve the comment source.")
			}
			return
		}
		source = resolved
	}
	subscriber := strconv.FormatInt(v.User.ID, 10)
	id, revision, err := s.Store.SaveFeedCommentWithSource(r.Context(), req, subscriber, v.User.Login, time.Now().Unix(), source)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not save the feed comment.")
		return
	}
	if err := s.Store.RecordCommentIntake(r.Context(), id, revision); err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not record the comment intake.")
		return
	}
	pipe, err := s.Store.CommentPipeline(r.Context(), id)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read the comment request.")
		return
	}
	response := commentResponse(pipe)
	if detail, err := s.Store.FeedComment(r.Context(), id); err == nil {
		response.Source = detail.Source
	}
	auth.JSONResponse(w, 200, response)
}

// apiFactoryFeedCommentOne serves one stored comment with its pipeline
// states. It reads stored state only: no funding, no judgment, no
// draft and no native effect ever run on this path.
func (s *API) apiFactoryFeedCommentOne(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method != http.MethodGet {
		auth.JSONError(w, 405, "invalid_request", "Feed comment reads only.")
		return
	}
	detail, err := s.Store.FeedComment(r.Context(), r.PathValue("id"))
	if err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Feed comment is not found.")
		return
	}
	if _, err := s.visibleRepository(r, v, detail.Record.Repository); err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Feed comment is not found.")
		return
	}
	if detail.Payload != nil && detail.Payload.Communication != nil {
		auth.JSONResponse(w, 200, feedWorkerCommentResponse(detail))
		return
	}
	pipe, err := s.Store.CommentPipeline(r.Context(), detail.Record.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read the comment request.")
		return
	}
	progress, _ := s.Store.CommentProgress(r.Context(), detail.Record.ID)
	auth.JSONResponse(w, 200, feedCommentResponse(detail, pipe, progress, strconv.FormatInt(v.User.ID, 10)))
}

func feedCommentResponse(detail store.FeedCommentDetail, pipe factory.CommentPipeline, progress, actor string) map[string]any {
	return map[string]any{
		"comment_id": detail.Record.ID, "revision": detail.Record.Revision,
		"repository": detail.Record.Repository, "author": detail.Record.Author,
		"author_name": detail.AuthorName, "at": detail.At, "issue_scope": detail.Record.Issue,
		"reply_to": detail.Record.ReplyTo, "edited": detail.Record.Edited,
		"deleted": detail.Record.Deleted, "revoked": detail.Record.Revoked,
		"body": commentBodyFor(detail), "status": store.PostStatus(&pipe),
		"intake": pipe.Intake, "judgment": pipe.Judgment, "route": pipe.Route,
		"draft": pipe.Draft, "effect": pipe.Effect, "issue": pipe.Issue,
		"existing_issue": commentExistingIssue(pipe), "source": detail.Source,
		"attempts": pipe.Attempts, "attempt_limit": factory.CommentAttemptLimit,
		"detail":    progress,
		"can_retry": detail.Record.Author == actor && !detail.Record.Deleted && !detail.Record.Revoked && pipe.RetryAllowed(),
	}
}

func (s *API) apiFactoryFeedReplies(w http.ResponseWriter, r *http.Request, v store.Session) {
	q := r.URL.Query()
	repository, err := strconv.ParseInt(q.Get("repository"), 10, 64)
	if err != nil || repository <= 0 {
		auth.JSONError(w, 400, "invalid_request", "Choose a project.")
		return
	}
	if _, err := s.visibleRepository(r, v, repository); err != nil {
		auth.JSONError(w, 404, "source_not_found", "Conversation is not visible.")
		return
	}
	source, err := s.Store.ResolveCommentSource(r.Context(), repository, q.Get("source_kind"), q.Get("source_id"))
	if err != nil {
		auth.JSONError(w, 404, "source_not_found", "Conversation is not visible.")
		return
	}
	var before int64
	if raw := q.Get("before"); raw != "" {
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before <= 0 {
			auth.JSONError(w, 400, "invalid_cursor", "Refresh the conversation.")
			return
		}
	}
	ids, next, err := s.Store.FeedReplyIDs(r.Context(), repository, source.Kind, source.ID, before)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read replies.")
		return
	}
	items := []map[string]any{}
	for _, id := range ids {
		detail, err := s.Store.FeedComment(r.Context(), id)
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Could not read replies.")
			return
		}
		if detail.Payload != nil && detail.Payload.Communication != nil {
			items = append(items, feedWorkerCommentResponse(detail))
			continue
		}
		pipe, err := s.Store.CommentPipeline(r.Context(), id)
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Could not read reply progress.")
			return
		}
		progress, _ := s.Store.CommentProgress(r.Context(), id)
		items = append(items, feedCommentResponse(detail, pipe, progress, strconv.FormatInt(v.User.ID, 10)))
	}
	auth.JSONResponse(w, 200, map[string]any{"items": items, "next_before": next})
}

func (s *API) apiFactoryFeedCommentRetry(w http.ResponseWriter, r *http.Request, v store.Session) {
	detail, err := s.Store.FeedComment(r.Context(), r.PathValue("id"))
	if err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Comment not found.")
		return
	}
	if _, err = s.visibleRepository(r, v, detail.Record.Repository); err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Comment not found.")
		return
	}
	if detail.Record.Author != strconv.FormatInt(v.User.ID, 10) {
		auth.JSONError(w, 403, "comment_author_required", "Only the author can revise this request.")
		return
	}
	var in struct {
		ExpectedRevision int64  `json:"expected_revision"`
		Body             string `json:"body"`
	}
	if !auth.DecodeAPIObject(w, r, &in) || !s.checkLifecycleSession(w, r, v) {
		return
	}
	if err := s.Store.RetryFeedComment(r.Context(), detail.Record.ID, detail.Record.Author, in.ExpectedRevision, in.Body, time.Now().Unix()); err != nil {
		auth.JSONError(w, 409, "comment_retry_conflict", err.Error())
		return
	}
	pipe, err := s.Store.CommentPipeline(r.Context(), detail.Record.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read the updated request.")
		return
	}
	auth.JSONResponse(w, 200, commentResponse(pipe))
}

// commentBodyFor hides the text of closed sources while keeping their
// tombstone resolvable.
func commentBodyFor(detail store.FeedCommentDetail) string {
	if detail.Record.Deleted || detail.Record.Revoked {
		return ""
	}
	return detail.Body
}

// commentCollaborators wires the funded JEV Choice judge from server
// configuration, the Product Lead authoring drafter behind the
// coordinator's intent-generation funding transport, and the native
// comment-issue creator behind the coordinator's graph executor. The
// drafter is bound whenever the coordinator and store exist; a
// missing enrolled model account then fails honestly at authoring
// time instead of leaving the stage silently unwired. Unwired
// stages wait in pending.
func (s *API) commentCollaborators() *CommentCollaborators {
	wired := &CommentCollaborators{}
	if s.Config.TypeSafeAPIKeyFile != "" {
		wired.Judge = &TypeSafeCommentJudge{Prepare: s.prepareCommentJudgeInput, Priority: &control.TypeSafePriority{
			APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel,
		}}
	}
	if s.Coordinator != nil && s.Store != nil {
		wired.Drafter = &IntentCommentDrafter{Store: s.Store, Transport: s.Coordinator}
	}
	if s.Coordinator != nil && s.Coordinator.Graph != nil && s.Coordinator.GraphActorID > 0 && s.Store != nil {
		wired.Creator = &GraphCommentIssueCreator{Store: s.Store, Graph: s.Coordinator.Graph, Actor: s.Coordinator.GraphActorID}
	}
	return wired
}

// driveCommentPipeline advances one durable comment request through
// judgment, draft and native effect, reloading after every step so a
// concurrent advance wins honestly through CAS instead of being
// dropped. Each stage checks its own admission; a refused stage waits
// in pending without spending.
func driveCommentPipeline(ctx context.Context, st *store.Store, wired *CommentCollaborators, id string) (factory.CommentResponse, error) {
	var empty factory.CommentResponse
	if st == nil {
		return empty, errors.New("comment store is unavailable")
	}
	// One drive flows a fresh submission through every wired stage
	// but consumes at most one attempt per stage: a failed call waits
	// for the next replay instead of burning the bounded retries in one
	// request. A concurrent advance wins honestly through CAS; the
	// response below always reloads stored truth.
	for range 8 {
		pipe, err := st.CommentPipeline(ctx, id)
		if err != nil {
			return empty, err
		}
		detail, err := st.FeedComment(ctx, id)
		if err != nil {
			return empty, err
		}
		if detail.Record.Deleted || detail.Record.Revoked {
			// The write path already settles the effect as blocked;
			// report the stored states without spending anything.
			return commentResponse(pipe), nil
		}
		var advanced bool
		switch {
		case pipe.AdmitCommentJudgment() && wired != nil && wired.Judge != nil:
			advanced, err = driveCommentJudgment(ctx, st, wired.Judge, pipe, detail)
		case pipe.AdmitCommentDraft() && wired != nil && wired.Drafter != nil:
			advanced, err = driveCommentDraft(ctx, st, wired.Drafter, pipe, detail)
		case pipe.AdmitCommentEffect() && wired != nil && wired.Creator != nil:
			advanced, err = driveCommentEffect(ctx, st, wired.Creator, pipe, detail)
		case pipe.Route == factory.CommentRouteExistingWork && pipe.Judgment == factory.CommentJudgmentApplied && pipe.Effect == factory.CommentEffectPending && pipe.Attempts < factory.CommentAttemptLimit && wired != nil:
			advanced, err = driveCommentFeedback(ctx, st, wired.Creator, pipe, detail)
		default:
			return commentResponse(pipe), nil
		}
		if err != nil {
			// A concurrent advance wins honestly: report its stored
			// outcome instead of failing the replay.
			if errors.Is(err, store.ErrStaleRevision) {
				fresh, reloadErr := st.CommentPipeline(ctx, id)
				if reloadErr != nil {
					return empty, reloadErr
				}
				return commentResponse(fresh), nil
			}
			return empty, err
		}
		if !advanced {
			fresh, err := st.CommentPipeline(ctx, id)
			if err != nil {
				return empty, err
			}
			return commentResponse(fresh), nil
		}
	}
	pipe, err := st.CommentPipeline(ctx, id)
	if err != nil {
		return empty, err
	}
	return commentResponse(pipe), nil
}

// commentCandidateLimit bounds the existing-work candidates one
// routing judgment may weigh.
const commentCandidateLimit = 8

// collectCommentJudgeInput assembles one judgment's complete input
// from recorded state: the comment with its persisted source
// snapshot, its reply parent's recorded evidence, and the
// existing-work candidates merged bound-first (bound issue scope,
// source snapshot, reply parent, recent repository work), deduped by
// issue with recorded summaries filled from whichever origin has
// them. A failed recent-work read fails the judgment instead of
// routing blind; an unreadable or closed parent simply contributes
// no evidence.
func collectCommentJudgeInput(ctx context.Context, st *store.Store, detail store.FeedCommentDetail) (CommentJudgeInput, error) {
	input := CommentJudgeInput{Comment: detail}
	recent, err := st.RecentCommentWorkCandidates(ctx, detail.Record.Repository, commentCandidateLimit)
	if err != nil {
		return CommentJudgeInput{}, err
	}
	byIssue := map[int64]*factory.CommentWorkCandidate{}
	var ordered []*factory.CommentWorkCandidate
	add := func(candidate factory.CommentWorkCandidate) {
		if candidate.Issue <= 0 {
			return
		}
		if existing, ok := byIssue[candidate.Issue]; ok {
			if existing.Summary == "" {
				existing.Summary = candidate.Summary
			}
			if existing.Status == "" {
				existing.Status = candidate.Status
			}
			return
		}
		merged := candidate
		byIssue[candidate.Issue] = &merged
		ordered = append(ordered, &merged)
	}
	if detail.Record.Issue > 0 {
		add(factory.CommentWorkCandidate{Issue: detail.Record.Issue, Origin: factory.CommentCandidateBound})
	}
	if detail.Source != nil && detail.Source.Issue > 0 {
		add(factory.CommentWorkCandidate{
			Issue: detail.Source.Issue, Summary: detail.Source.Summary,
			Status: detail.Source.Status, Origin: factory.CommentCandidateSource,
		})
	}
	if detail.Record.ReplyTo != "" {
		parent := &CommentParentEvidence{Comment: detail.Record.ReplyTo}
		if parentDetail, err := st.FeedComment(ctx, detail.Record.ReplyTo); err == nil &&
			parentDetail.Record.Repository == detail.Record.Repository &&
			!parentDetail.Record.Deleted && !parentDetail.Record.Revoked {
			parent.Body = store.TruncateFeedSummary(parentDetail.Body)
			parent.Issue = parentDetail.Record.Issue
		}
		if parentPipe, err := st.CommentPipeline(ctx, detail.Record.ReplyTo); err == nil && parent.Issue <= 0 {
			parent.Issue = parentPipe.Issue
		}
		input.Parent = parent
		if parent.Issue > 0 {
			add(factory.CommentWorkCandidate{Issue: parent.Issue, Origin: factory.CommentCandidateParent})
		}
	}
	for _, candidate := range recent {
		add(candidate)
	}
	for _, candidate := range ordered {
		if err := candidate.Validate(); err != nil {
			return CommentJudgeInput{}, err
		}
		input.Candidates = append(input.Candidates, *candidate)
		if len(input.Candidates) == commentCandidateLimit {
			break
		}
	}
	return input, nil
}

// applyCommentJudgment advances one pipeline with a settled routing
// outcome. An existing_work judgment persists which work it selected
// in the pipeline's issue link; every other route clears it.
func driveCommentFeedback(ctx context.Context, st *store.Store, creator CommentIssueCreator, pipe factory.CommentPipeline, detail store.FeedCommentDetail) (bool, error) {
	requested, err := st.CommentFeedbackRequested(ctx, pipe.Comment, pipe.Revision)
	if err != nil || !requested {
		return false, err
	}
	writer, ok := creator.(interface {
		DeliverCommentFeedback(context.Context, store.FeedCommentDetail, int64) error
	})
	if !ok {
		_ = st.NoteCommentProgress(ctx, pipe.Comment, "Feedback delivery is unavailable. Restore Factory’s connection before retrying.")
		return false, nil
	}
	if err = writer.DeliverCommentFeedback(ctx, detail, pipe.Issue); err != nil {
		_ = st.NoteCommentProgress(ctx, pipe.Comment, "Could not confirm delivery to the issue discussion. Check repository access and refresh before retrying.")
		return false, noteCommentFailure(ctx, st, pipe, false)
	}
	next := pipe
	next.Effect = factory.CommentEffectApplied
	if err = st.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		return false, err
	}
	_ = st.NoteCommentProgress(ctx, pipe.Comment, "Feedback added to the issue discussion. Accepted requirements have not been changed.")
	return true, nil
}

func applyCommentJudgment(ctx context.Context, st *store.Store, pipe factory.CommentPipeline, judgment CommentJudgment) error {
	next := pipe
	next.Judgment = factory.CommentJudgmentApplied
	next.Route = judgment.Route
	next.Issue = 0
	if judgment.Route == factory.CommentRouteExistingWork {
		next.Issue = judgment.Issue
	}
	if err := st.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		return err
	}
	if judgment.Route == factory.CommentRouteClarify {
		question := judgment.Question
		if question == "" {
			question = commentClarificationQuestions["clarify_result"]
		}
		return st.NoteCommentProgress(ctx, pipe.Comment, question)
	}
	return nil
}

// driveCommentJudgment runs one JEV Choice call behind the comment
// source revision. JEV calls are free; a reservation is taken before
// the first call as duplicate-effect protection and a provider-limit
// bound, not a monetary gate. A lost or failed call retries under the
// still-open reservation, bounded by the attempt count. A refused
// reservation with no open call waits on the provider limit. A
// receipt already recorded behind this revision — a crash between
// recording the answer and advancing the pipeline — replays its
// stored selection without a second call, so restart resumes the
// same judgment exactly once.
func driveCommentJudgment(ctx context.Context, st *store.Store, judge CommentJudge, pipe factory.CommentPipeline, detail store.FeedCommentDetail) (bool, error) {
	snapshot := store.CommentJudgmentSnapshot(pipe.Comment, pipe.Revision)
	if receipt, err := st.PriorityJudgment(ctx, snapshot); err == nil && receipt.Selected != "" {
		judgment, err := commentJudgmentFromSelection(receipt.Selected, commentCandidatesFromReceipt(receipt))
		if err != nil {
			return false, noteCommentFailure(ctx, st, pipe, true)
		}
		if err := applyCommentJudgment(ctx, st, pipe, judgment); err != nil {
			return false, err
		}
		return true, nil
	}
	funded, err := st.ReserveCommentJudgment(ctx, pipe.Comment, pipe.Revision)
	if err != nil {
		return false, err
	}
	if !funded {
		open, err := st.CommentJudgmentReservationOpen(ctx, pipe.Comment, pipe.Revision)
		if err != nil || !open {
			return false, err
		}
	}
	input, err := collectCommentJudgeInput(ctx, st, detail)
	if err != nil {
		_ = st.NoteCommentProgress(ctx, pipe.Comment, "The original request context could not be read. Refresh the project and retry assessment.")
		return false, noteCommentFailure(ctx, st, pipe, true)
	}
	if preparer, ok := judge.(interface {
		PrepareCommentInput(context.Context, CommentJudgeInput) (CommentJudgeInput, error)
	}); ok {
		input, err = preparer.PrepareCommentInput(ctx, input)
		if err != nil {
			_ = st.NoteCommentProgress(ctx, pipe.Comment, err.Error())
			return false, noteCommentFailure(ctx, st, pipe, true)
		}
	}
	judgment, receipt, err := judge.JudgeComment(ctx, input, snapshot)
	if err != nil {
		_ = st.NoteCommentProgress(ctx, pipe.Comment, "JEV could not assess this request. Check the System One connection and retry assessment.")
		return false, noteCommentFailure(ctx, st, pipe, true)
	}
	switch judgment.Route {
	case factory.CommentRouteNewIssue, factory.CommentRouteDiscussion, factory.CommentRouteClarify:
		if judgment.Issue != 0 {
			return false, noteCommentFailure(ctx, st, pipe, true)
		}
	case factory.CommentRouteExistingWork:
		if _, err := commentJudgmentFromSelection(commentExistingChoice(judgment.Issue), input.Candidates); err != nil {
			return false, noteCommentFailure(ctx, st, pipe, true)
		}
	default:
		return false, noteCommentFailure(ctx, st, pipe, true)
	}
	if err := st.RecordCommentJudgmentReceipt(ctx, pipe.Comment, pipe.Revision, receipt); err != nil {
		return false, noteCommentFailure(ctx, st, pipe, false)
	}
	if err := applyCommentJudgment(ctx, st, pipe, judgment); err != nil {
		return false, err
	}
	return true, nil
}

// driveCommentDraft runs one Product Lead LLM authoring call for a
// new_issue route. A malformed draft settles the draft as invalid
// before any native effect; a failed call keeps its attempt for retry.
func driveCommentDraft(ctx context.Context, st *store.Store, drafter CommentDrafter, pipe factory.CommentPipeline, detail store.FeedCommentDetail) (bool, error) {
	draft, err := drafter.DraftCommentIssue(ctx, detail)
	if err != nil {
		// A launched planning run that is still settling — or a drive
		// interrupted by service shutdown — is not an authoring
		// failure: the draft stays pending with its bounded attempts
		// intact and the next service pass resumes the same funded
		// attempt instead of spending again.
		if errors.Is(err, ErrCommentDraftPending) || ctx.Err() != nil {
			_ = st.NoteCommentProgress(ctx, pipe.Comment, err.Error())
			return false, nil
		}
		_ = st.NoteCommentProgress(ctx, pipe.Comment, err.Error())
		return false, noteCommentFailure(ctx, st, pipe, false)
	}
	draft.Comment = pipe.Comment
	draft.Revision = pipe.Revision
	if err := draft.Validate(); err != nil {
		next := pipe
		next.Draft = factory.CommentDraftInvalid
		next.Effect = factory.CommentEffectBlocked
		if advanceErr := st.AdvanceCommentPipeline(ctx, pipe, next); advanceErr != nil {
			return false, advanceErr
		}
		if wipeErr := st.DiscardCommentDraftData(ctx, pipe.Comment); wipeErr != nil {
			return false, wipeErr
		}
		return true, nil
	}
	if err := st.RecordCommentDraft(ctx, draft); err != nil {
		return false, err
	}
	next := pipe
	next.Draft = factory.CommentDraftApplied
	if err := st.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		return false, err
	}
	return true, nil
}

// driveCommentEffect applies and verifies the native issue behind an
// applied draft. It reconciles first: an issue already created from
// this comment (lost effect reply) verifies without duplicating. The
// draft must still bind the current source revision or the effect
// refuses instead of creating from stale text.
func driveCommentEffect(ctx context.Context, st *store.Store, creator CommentIssueCreator, pipe factory.CommentPipeline, detail store.FeedCommentDetail) (bool, error) {
	if detail.Record.Revision != pipe.Revision {
		return false, nil
	}
	draft, err := st.CommentDraft(ctx, pipe.Comment, pipe.Revision)
	if err != nil {
		return false, noteCommentFailure(ctx, st, pipe, false)
	}
	if existing, err := creator.LookupCommentIssue(ctx, detail.Record.Repository, pipe.Comment); err == nil && existing > 0 {
		next := pipe
		next.Effect = factory.CommentEffectVerified
		next.Issue = existing
		if err := st.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
			return false, err
		}
		return true, nil
	}
	issue, err := creator.CreateCommentIssue(ctx, draft, detail.Record.Repository)
	if err != nil || issue <= 0 {
		_ = st.NoteCommentProgress(ctx, pipe.Comment, "Could not confirm issue creation. Retry checks for an existing issue before creating one.")
		return false, noteCommentFailure(ctx, st, pipe, false)
	}
	next := pipe
	next.Effect = factory.CommentEffectApplied
	next.Issue = issue
	if err := st.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		return false, err
	}
	// Verify through native truth in the same drive: a confirmed link
	// settles verified, otherwise applied still reports the issue and a
	// later drive verifies without creating again.
	applied, err := st.CommentPipeline(ctx, pipe.Comment)
	if err != nil {
		return true, nil
	}
	if existing, err := creator.LookupCommentIssue(ctx, detail.Record.Repository, pipe.Comment); err == nil && existing > 0 {
		settled := applied
		settled.Effect = factory.CommentEffectVerified
		settled.Issue = existing
		if err := st.AdvanceCommentPipeline(ctx, applied, settled); err != nil {
			return true, nil
		}
	}
	return true, nil
}

// noteCommentFailure records one retry attempt; when the bounded
// attempts run out the open stage settles honestly (judgment failed,
// otherwise the draft/effect stays retryable only while attempts
// remain, and the caller sees pending until then).
func noteCommentFailure(ctx context.Context, st *store.Store, pipe factory.CommentPipeline, failJudgment bool) error {
	if err := st.NoteCommentAttempt(ctx, pipe.Comment); err != nil {
		if failJudgment {
			next := pipe
			next.Judgment = factory.CommentJudgmentFailed
			_ = st.AdvanceCommentPipeline(ctx, pipe, next)
		}
		return nil
	}
	fresh, err := st.CommentPipeline(ctx, pipe.Comment)
	if err != nil {
		return err
	}
	if fresh.Attempts < factory.CommentAttemptLimit {
		return nil
	}
	next := fresh
	if failJudgment && next.Judgment == factory.CommentJudgmentPending {
		next.Judgment = factory.CommentJudgmentFailed
	}
	if advanceErr := st.AdvanceCommentPipeline(ctx, fresh, next); advanceErr != nil {
		return advanceErr
	}
	return nil
}

// enrichFeedPosts fills the server-derived post enrichment on one
// served window from the canonical comment owners: author identity,
// display name, project, time and pipeline status. Records pass
// through for their own canonical producers (Lane A role/effect
// results). Closed sources never reach this point; the store already
// filters them.
func enrichFeedPosts(ctx context.Context, st *store.Store, projects map[int64]string, entries []factory.FeedEntry) {
	var ids []string
	for _, entry := range entries {
		if entry.Kind == factory.FeedKindPost {
			ids = append(ids, entry.ID)
		}
	}
	if len(ids) == 0 || st == nil {
		return
	}
	details, err := st.FeedPostDetails(ctx, ids)
	if err != nil {
		return
	}
	for i, entry := range entries {
		if entry.Kind != factory.FeedKindPost {
			continue
		}
		detail, ok := details[entry.ID]
		if !ok {
			continue
		}
		entries[i].Actor = detail.Record.Author
		if detail.Actor != "" {
			entries[i].Actor = detail.Actor
		}
		entries[i].Assignment = detail.Assignment
		entries[i].Issue = detail.Record.Issue
		entries[i].Payload = detail.Payload
		if detail.Payload != nil && detail.Payload.Communication != nil {
			entries[i].Activity = detail.Payload.Communication.Kind
		}
		entries[i].Author = detail.AuthorName
		entries[i].Project = projects[entry.Repository]
		entries[i].At = detail.At
		entries[i].Status = detail.Status
		entries[i].Summary = store.TruncateFeedSummary(detail.Body)
	}
}

// commentExistingIssue names the already-tracked issue one pipeline
// linked on the existing_work route, zero on every other route.
func commentExistingIssue(pipe factory.CommentPipeline) int64 {
	if pipe.Route == factory.CommentRouteExistingWork {
		return pipe.Issue
	}
	return 0
}

// commentResponse renders one stored pipeline row as the POST answer.
func commentResponse(pipe factory.CommentPipeline) factory.CommentResponse {
	return factory.CommentResponse{
		Attempts: pipe.Attempts, AttemptLimit: factory.CommentAttemptLimit,
		Comment: pipe.Comment, Revision: pipe.Revision, Intake: pipe.Intake,
		Judgment: pipe.Judgment, Route: pipe.Route, Draft: pipe.Draft,
		Effect: pipe.Effect, Issue: pipe.Issue, ExistingIssue: commentExistingIssue(pipe),
	}
}

// commentDriveLimit bounds the stored comment requests one service
// pass advances: authoring and native effects are expensive, and the
// coordinator pass must stay responsive to factory work.
const commentDriveLimit = 4

// drivePendingComments advances one bounded page of durable comment
// requests through their wired stages and reports how many drove
// without error. Per-request failures keep their stored state for a
// later pass; exactly-once advancement rests on the pipeline CAS,
// the judgment reservation and receipt, the single funded planning
// attempt and the deterministic native operation identity.
func drivePendingComments(ctx context.Context, st *store.Store, wired *CommentCollaborators, limit int) int {
	if st == nil {
		return 0
	}
	pending, err := st.PendingCommentPipelines(ctx, limit)
	if err != nil {
		return 0
	}
	driven := 0
	for _, pipe := range pending {
		if ctx.Err() != nil {
			break
		}
		_ = st.NoteCommentProgress(ctx, pipe.Comment, "")
		if _, err := driveCommentPipeline(ctx, st, wired, pipe.Comment); err == nil {
			driven++
		}
	}
	return driven
}

// DrivePendingComments is the service-owned durable continuation for
// feed comments, bound to the coordinator's Start and progress
// lifecycle by the server wiring. It advances stored comment
// requests independently of any browser request, so a submission
// whose POST already answered — or whose service restarted mid-
// flight — still reaches its recorded issue exactly once.
func (s *API) DrivePendingComments(ctx context.Context) {
	if s == nil || s.Store == nil {
		return
	}
	_ = drivePendingComments(ctx, s.Store, s.commentCollaborators(), commentDriveLimit)
}
