package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
// service-side behind the stored request rows (DrivePendingPosts,
// bound to the coordinator lifecycle), never inside the browser
// request: a request that times out or a service that restarts resumes
// the same stored request exactly once, and the comment GET re-reads
// the settled outcome. Unwired stages wait truthfully in pending:
// nothing spends and nothing dispatches without its collaborator.
// Reads never drive: GET serves stored state only. No per-comment
// approvals exist; a discussion route only ever records discussion.

// PostJudgment is one routing outcome: the assessed route plus,
// for existing_work, the already-tracked issue the judgment selected.
type PostJudgment struct {
	Route    string
	Issue    int64
	Question string
	// Routing is the Choice behind this outcome, kept for the feed.
	Routing *factory.PostRouting
}

// postRoutingLabels names the base routing options for people.
var postRoutingLabels = map[string]string{
	factory.PostRouteNewIssue:   "New work",
	factory.PostRouteDiscussion: "Just discussion",
	"clarify_target":            "Ask which app or issue",
	"clarify_result":            "Ask what the result should be",
	"clarify_conflict":          "Ask which requirement wins",
}

// postRouting records one routing answer with every offered option.
func postRouting(answer factory.PriorityAnswer, keys []string, candidates []factory.PostWorkCandidate) *factory.PostRouting {
	labels := make(map[string]string, len(keys))
	for key, label := range postRoutingLabels {
		labels[key] = label
	}
	for _, candidate := range candidates {
		label := fmt.Sprintf("Issue #%d", candidate.Issue)
		if name := strings.TrimSpace(candidate.Title); name != "" {
			label += " · " + name
		} else if objective := strings.TrimSpace(candidate.Objective); objective != "" {
			label += " · " + objective
		}
		labels[postExistingChoice(candidate.Issue)] = store.TruncateFeedSummary(label)
	}
	routing := &factory.PostRouting{Selected: answer.Selected, Confidence: answer.Confidence}
	for _, key := range keys {
		routing.Options = append(routing.Options, factory.PostRoutingOption{
			Key: key, Label: labels[key], Probability: answer.Probabilities[key],
		})
	}
	return routing
}

var postClarificationQuestions = map[string]string{
	"clarify_target":   "Which app, page, or existing issue should this request apply to?",
	"clarify_result":   "What should change, and what should the result look like?",
	"clarify_conflict": "Which of the conflicting requirements should take precedence?",
}

// PostParentEvidence carries the recorded reply-parent facts one
// judgment may weigh: the parent comment identity, a bounded preview
// of its stored text and the native issue its own pipeline linked.
type PostParentEvidence struct {
	SourcePost string
	Body       string
	Issue      int64
}

// PostJudgeInput carries everything one routing judgment may
// weigh: the comment with its persisted source snapshot, its reply
// parent's recorded evidence, and the existing-work candidates.
type PostJudgeInput struct {
	SourcePost store.FeedPostDetail
	Parent     *PostParentEvidence
	Candidates []factory.PostWorkCandidate
}

// PostJudge routes one comment through one JEV Choice.
type PostJudge interface {
	JudgePost(ctx context.Context, input PostJudgeInput) (PostJudgment, error)
}

// PostDrafter authors one issue draft from a comment through the
// Product Lead LLM: title, body, objective and acceptance plus the Soda
// work metadata the native creation validates.
type PostDrafter interface {
	DraftPostIssue(ctx context.Context, comment store.FeedPostDetail) (factory.IssueDraft, error)
}

// PostIssueCreator validates and creates native issues from drafts
// through the existing native issue/graph ops, and looks up the issue
// already created from a comment so a lost effect reply reconciles
// instead of duplicating.
type PostIssueCreator interface {
	CreatePostIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error)
	LookupPostIssue(ctx context.Context, repository int64, postID string) (int64, error)
}

// PostCollaborators carries the wired Lane B stages. Nil stages wait
// in pending; the driver never spends or dispatches through a nil.
type PostCollaborators struct {
	IssueMaker func(context.Context, factory.PostIntake, store.FeedPostDetail) (factory.PostResponse, error)
	Judge      PostJudge
	// Prepare adds the native issue context the judge weighs.
	Prepare func(context.Context, PostJudgeInput) (PostJudgeInput, error)
	Drafter PostDrafter
	Creator PostIssueCreator
}

// postRouteRubric is the single JEV Choice rubric behind comment
// routing: code supplies the complete comment state — body, bound
// issue scope, resolved source snapshot, reply-parent evidence and
// the existing-work candidates — and the explicit outcomes.
const postRouteRubric = "Route one feed comment to exactly one outcome: " +
	"new_issue when it requests a concrete change that needs new tracked work; " +
	"existing_<issue> for one listed candidate when the comment belongs to that already tracked work; " +
	"discussion when it is an acknowledgement or discussion that must never create an issue; " +
	"clarify_target when the target is ambiguous, clarify_result when the desired result is unclear, or clarify_conflict when requirements conflict. " +
	"Match the requested deliverable against actual issue objectives. Project membership, recent activity, generic worker status, or being a reply do not establish an objective match. " +
	"Use existing work for the same deliverable or a correction of that deliverable; a distinct requested change needs new work even when the comment replies to an older post."

// postExistingChoice names the Choice key selecting one
// existing-work candidate issue.
func postExistingChoice(issue int64) string {
	return "existing_" + strconv.FormatInt(issue, 10)
}

// postChoiceKeys lists the admitted Choice keys for one judgment:
// the three base routes plus one key per existing-work candidate.
// With no candidates there is no existing-work key to select, so a
// comment can never be routed to work the server never listed.
func postChoiceKeys(candidates []factory.PostWorkCandidate) []string {
	keys := []string{factory.PostRouteNewIssue, factory.PostRouteDiscussion, "clarify_target", "clarify_result", "clarify_conflict"}
	for _, candidate := range candidates {
		keys = append(keys, postExistingChoice(candidate.Issue))
	}
	return keys
}

// postJudgmentFromSelection parses one recorded Choice selection
// into its route and selected issue. Only selections the choice
// actually offered admit: an existing-work key outside the candidate
// set refuses instead of linking unlisted work.
func postJudgmentFromSelection(selected string, candidates []factory.PostWorkCandidate) (PostJudgment, error) {
	if question, ok := postClarificationQuestions[selected]; ok {
		return PostJudgment{Route: factory.PostRouteClarify, Question: question}, nil
	}
	if selected == factory.PostRouteNewIssue || selected == factory.PostRouteDiscussion {
		return PostJudgment{Route: selected}, nil
	}
	for _, candidate := range candidates {
		if selected == postExistingChoice(candidate.Issue) {
			return PostJudgment{Route: factory.PostRouteExistingWork, Issue: candidate.Issue}, nil
		}
	}
	return PostJudgment{}, errors.New("comment judgment selected an outcome it was never offered")
}

// postJudgeState renders the complete supplied state for one
// routing Choice: the comment, its persisted source snapshot, its
// reply parent's recorded evidence and the existing-work candidates.
func postJudgeState(input PostJudgeInput) map[string]any {
	comment := input.SourcePost
	state := map[string]any{
		"repository": comment.Record.Repository, "issue": comment.Record.Issue,
		"parent_id": comment.Record.ParentID, "body": comment.Body,
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
			"comment": input.Parent.SourcePost, "body": input.Parent.Body,
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

// TypeSafePostJudge routes comments through the JEV Choice transport.
// It grants no authority and performs no dispatch.
type TypeSafePostJudge struct {
	Priority *control.TypeSafePriority
}

// JudgePost assesses one comment through one Choice call.
func (j *TypeSafePostJudge) JudgePost(ctx context.Context, input PostJudgeInput) (PostJudgment, error) {
	var empty PostJudgment
	if j == nil || j.Priority == nil {
		return empty, control.ErrPriorityUnavailable
	}
	keys := postChoiceKeys(input.Candidates)
	choices := make(map[string]any, len(keys))
	for _, key := range keys {
		switch key {
		case factory.PostRouteNewIssue:
			choices[key] = "A concrete new deliverable or change that no listed issue already covers."
		case factory.PostRouteDiscussion:
			choices[key] = "Conversation or acknowledgement without a request to change the product."
		default:
			if question, ok := postClarificationQuestions[key]; ok {
				choices[key] = "The comment requires an answer to: " + question
			} else {
				choices[key] = "Continue the specific listed issue only if its objective covers this requested result."
			}
		}
	}
	answer, err := j.Priority.Select(ctx, postJudgeState(input), postRouteRubric, choices)
	if err != nil {
		return empty, err
	}
	request := factory.PriorityRequest{Rubric: postRouteRubric}
	for _, key := range keys {
		request.Candidates = append(request.Candidates, factory.PriorityCandidate{ID: key})
	}
	if err := answer.Validate(request); err != nil {
		return empty, err
	}
	judgment, err := postJudgmentFromSelection(answer.Selected, input.Candidates)
	if err != nil {
		return empty, err
	}
	judgment.Routing = postRouting(answer, keys, input.Candidates)
	return judgment, nil
}

// parseFeedPostBody admits one comment POST body: scope, bounded
// text and idempotency key. The author is the host-admitted caller.
func parseFeedPostBody(data []byte) (factory.PostRequest, error) {
	var req factory.PostRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return factory.PostRequest{}, errFeedQueryInvalid
	}
	if err := req.Validate(); err != nil {
		return factory.PostRequest{}, err
	}
	return req, nil
}

// apiFactoryFeedPost saves one comment and answers promptly with
// its durable identity and stored pipeline states. It never drives
// the pipeline: judgment, authoring and the native effect continue
// service-side behind the stored request (DrivePendingPosts), so
// a browser timeout loses nothing and the outcome is re-read through
// the comment GET. A named source entry is resolved inside the
// caller's visible repository before anything is saved; unknown or
// cross-repository sources refuse without disclosure. A retried
// submission recovers the original comment and its current states
// instead of duplicating them.
func (s *API) apiFactoryFeedPost(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method != http.MethodPost {
		auth.JSONError(w, 405, "invalid_request", "Feed posts use POST only.")
		return
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		auth.JSONError(w, 400, "invalid_request", "Feed post body is invalid.")
		return
	}
	req, err := parseFeedPostBody(raw)
	if err != nil {
		auth.JSONError(w, 400, "invalid_request", "Feed post body is invalid.")
		return
	}
	access, err := s.visibleRepository(r, v, req.Repository)
	if err != nil {
		auth.JSONError(w, 404, "repository_not_found", "Repository is not visible.")
		return
	}
	_ = access
	var source *factory.PostSource
	if req.ParentID != "" {
		resolved, err := s.Store.ResolvePostParent(r.Context(), req.Repository, req.ParentID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				auth.JSONError(w, 404, "source_not_found", "The parent post is not visible.")
			} else {
				auth.JSONError(w, 503, "store_unavailable", "Could not resolve the parent post.")
			}
			return
		}
		source = resolved
		req.SourceKind, req.SourceID = resolved.Kind, resolved.ID
	}
	subscriber := strconv.FormatInt(v.User.ID, 10)
	id, revision, err := s.Store.SaveFeedPostWithSource(r.Context(), req, subscriber, v.User.Login, time.Now().Unix(), source)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not save the feed post.")
		return
	}
	if err := s.Store.RecordIssueMakerIntake(r.Context(), id, revision); err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not record the post intake.")
		return
	}
	if s.Store.PostUsesIssueMaker(r.Context(), id, revision) {
		_ = s.captureIssueIntakeGrant(r, id, revision)
	}
	if err := s.capturePostIssueContext(r, id, revision); err != nil {
		_ = s.Store.NotePostStall(r.Context(), id, postStalledContext)
		_ = s.Store.NotePostProgress(r.Context(), id, "Could not read this repository's issues using your current access. Retry the request.")
	}
	pipe, err := s.Store.PostIntake(r.Context(), id)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read the post request.")
		return
	}
	response := postResponse(pipe)
	if detail, err := s.Store.FeedPost(r.Context(), id); err == nil {
		response.Source = detail.Source
	}
	auth.JSONResponse(w, 200, response)
}

// apiFactoryFeedPostOne serves one stored comment with its pipeline
// states. It reads stored state only: no funding, no judgment, no
// draft and no native effect ever run on this path.
func (s *API) apiFactoryFeedPostOne(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method != http.MethodGet {
		auth.JSONError(w, 405, "invalid_request", "Feed post reads only.")
		return
	}
	entry, err := s.Store.FeedEntryByID(r.Context(), r.PathValue("id"))
	if err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Feed post is not found.")
		return
	}
	if _, err := s.visibleRepository(r, v, entry.Repository); err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Feed post is not found.")
		return
	}
	if entry.Kind == factory.FeedKindRecord {
		entries := []factory.FeedEntry{entry}
		enrichFeedRecords(r.Context(), s.Store, nil, entries)
		enrichFeedParents(r.Context(), s.Store, entries)
		post := entries[0]
		auth.JSONResponse(w, 200, map[string]any{"post_id": post.ID, "kind": post.Kind, "repository": post.Repository, "parent_id": post.ParentID, "parent_author": post.ParentAuthor, "author_name": post.Author, "actor": post.Actor, "at": post.At, "body": post.Summary, "status": post.Status, "issue_scope": post.Issue, "payload": post.Payload, "assignment": post.Assignment})
		return
	}
	detail, err := s.Store.FeedPost(r.Context(), entry.ID)
	if err != nil {
		auth.JSONError(w, 404, "post_not_found", "Post is not found.")
		return
	}
	view, err := s.feedPost(r, v, detail)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read the post request.")
		return
	}
	auth.JSONResponse(w, 200, view)
}

// feedPostView reads one stored comment as every feed surface serves it.
func (s *API) feedPostView(r *http.Request, v store.Session, id string) (map[string]any, error) {
	detail, err := s.Store.FeedPost(r.Context(), id)
	if err != nil {
		entry, err := s.Store.FeedEntryByID(r.Context(), id)
		if err != nil {
			return nil, err
		}
		if _, err := s.visibleRepository(r, v, entry.Repository); err != nil {
			return nil, err
		}
		entries := []factory.FeedEntry{entry}
		enrichFeedRecords(r.Context(), s.Store, nil, entries)
		enrichFeedParents(r.Context(), s.Store, entries)
		post := entries[0]
		return map[string]any{"post_id": post.ID, "kind": post.Kind, "repository": post.Repository, "parent_id": post.ParentID, "parent_author": post.ParentAuthor, "author_name": post.Author, "actor": post.Actor, "at": post.At, "body": post.Summary, "status": post.Status, "issue_scope": post.Issue, "payload": post.Payload, "assignment": post.Assignment}, nil
	}
	return s.feedPost(r, v, detail)
}

// feedPost renders one stored comment: a worker post as recorded, a
// human comment with its pipeline outcome. It reads stored state only.
func (s *API) feedPost(r *http.Request, v store.Session, detail store.FeedPostDetail) (map[string]any, error) {
	if detail.Contribution != nil || detail.Consultation != nil || (detail.Payload != nil && detail.Payload.Communication != nil) {
		return feedWorkerPostResponse(detail), nil
	}
	pipe, err := s.Store.PostIntake(r.Context(), detail.Record.ID)
	if err != nil {
		return nil, err
	}
	progress, _ := s.Store.PostProgress(r.Context(), detail.Record.ID)
	return s.feedPostWithProduct(r, v, detail, pipe, progress), nil
}

func feedPostResponse(detail store.FeedPostDetail, pipe factory.PostIntake, progress, actor string) map[string]any {
	return map[string]any{
		"post_id": detail.Record.ID, "revision": detail.Record.Revision,
		"repository": detail.Record.Repository, "author": detail.Record.Author,
		"author_name": detail.AuthorName, "at": detail.At, "issue_scope": detail.Record.Issue,
		"parent_id": detail.Record.ParentID, "edited": detail.Record.Edited,
		"deleted": detail.Record.Deleted, "revoked": detail.Record.Revoked,
		"body": postBodyFor(detail), "status": store.PostStatus(&pipe),
		"intake": pipe.Intake, "judgment": pipe.Judgment, "route": pipe.Route,
		"draft": pipe.Draft, "effect": pipe.Effect, "issue": pipe.Issue,
		"existing_issue": postExistingIssue(pipe), "source": detail.Source,
		"attempts": pipe.Attempts, "attempt_limit": factory.PostAttemptLimit,
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
	source, err := s.Store.ResolvePostParent(r.Context(), repository, r.PathValue("id"))
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
		item, err := s.feedPostView(r, v, id)
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Could not read replies.")
			return
		}
		items = append(items, item)
	}
	auth.JSONResponse(w, 200, map[string]any{"items": items, "next_before": next})
}

func (s *API) apiFactoryFeedPostRetry(w http.ResponseWriter, r *http.Request, v store.Session) {
	detail, err := s.Store.FeedPost(r.Context(), r.PathValue("id"))
	if err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Post not found.")
		return
	}
	if _, err = s.visibleRepository(r, v, detail.Record.Repository); err != nil {
		auth.JSONError(w, 404, "comment_not_found", "Post not found.")
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
	if err := s.Store.RetryIssueMakerFeedPost(r.Context(), detail.Record.ID, detail.Record.Author, in.ExpectedRevision, in.Body, time.Now().Unix()); err != nil {
		auth.JSONError(w, 409, "comment_retry_conflict", err.Error())
		return
	}
	updated, err := s.Store.FeedPost(r.Context(), detail.Record.ID)
	if err == nil && s.Store.PostUsesIssueMaker(r.Context(), detail.Record.ID, updated.Record.Revision) {
		_ = s.captureIssueIntakeGrant(r, detail.Record.ID, updated.Record.Revision)
	}
	if err != nil || s.capturePostIssueContext(r, detail.Record.ID, updated.Record.Revision) != nil {
		_ = s.Store.NotePostStall(r.Context(), detail.Record.ID, postStalledContext)
		_ = s.Store.NotePostProgress(r.Context(), detail.Record.ID, "Could not refresh this request's issues using your current repository access. Try again.")
	}
	pipe, err := s.Store.PostIntake(r.Context(), detail.Record.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read the updated request.")
		return
	}
	auth.JSONResponse(w, 200, postResponse(pipe))
}

// postBodyFor hides the text of closed sources while keeping their
// tombstone resolvable.
func postBodyFor(detail store.FeedPostDetail) string {
	if detail.Record.Deleted || detail.Record.Revoked {
		return ""
	}
	return detail.Body
}

// postCollaborators wires the funded JEV Choice judge from server
// configuration, the Product Lead authoring drafter behind the
// coordinator's intent-generation funding transport, and the native
// comment-issue creator behind the coordinator's graph executor. The
// drafter is bound whenever the coordinator and store exist; a
// missing enrolled model account then fails honestly at authoring
// time instead of leaving the stage silently unwired. Unwired
// stages wait in pending.
func (s *API) postCollaborators() *PostCollaborators {
	wired := &PostCollaborators{}
	wired.IssueMaker = s.drivePostIssueMaker
	if s.jevConnected() {
		wired.Judge = &TypeSafePostJudge{Priority: &control.TypeSafePriority{
			APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel,
		}}
		wired.Prepare = s.preparePostJudgeInput
	}
	if s.Coordinator != nil && s.Store != nil {
		wired.Drafter = &IntentPostDrafter{Store: s.Store, Transport: s.Coordinator}
	}
	if s.Coordinator != nil && s.Coordinator.Graph != nil && s.Coordinator.GraphActorID > 0 && s.Store != nil {
		wired.Creator = &GraphPostIssueCreator{Store: s.Store, Graph: s.Coordinator.Graph, Actor: s.Coordinator.GraphActorID}
	}
	return wired
}

// Stalled stages: the request's next stage is admitted but this server has
// no collaborator to run it, so it cannot move until one is connected.
const (
	postStalledJEV         = "jev"
	postStalledProductLead = "product_lead"
	postStalledForge       = "forge"
	postStalledContext     = "context"
)

// postStalled names the collaborator the request is waiting for, or ""
// when its next stage can run (or nothing is left to run).
func postStalled(pipe factory.PostIntake, wired *PostCollaborators) string {
	if wired == nil {
		wired = &PostCollaborators{}
	}
	switch {
	case pipe.AdmitPostJudgment() && wired.Judge == nil:
		return postStalledJEV
	case pipe.AdmitPostDraft() && wired.Drafter == nil:
		return postStalledProductLead
	case pipe.AdmitPostEffect() && wired.Creator == nil:
		return postStalledForge
	}
	return ""
}

// drivePostIntake advances one durable comment request through
// judgment, draft and native effect, reloading after every step so a
// concurrent advance wins honestly through CAS instead of being
// dropped. Each stage checks its own admission; a refused stage waits
// in pending without spending.
func drivePostIntake(ctx context.Context, st *store.Store, wired *PostCollaborators, id string) (factory.PostResponse, error) {
	var empty factory.PostResponse
	if st == nil {
		return empty, errors.New("comment store is unavailable")
	}
	// One drive flows a fresh submission through every wired stage
	// but consumes at most one attempt per stage: a failed call waits
	// for the next replay instead of burning the bounded retries in one
	// request. A concurrent advance wins honestly through CAS; the
	// response below always reloads stored truth.
	for range 8 {
		pipe, err := st.PostIntake(ctx, id)
		if err != nil {
			return empty, err
		}
		detail, err := st.FeedPost(ctx, id)
		if err != nil {
			return empty, err
		}
		if detail.Record.Deleted || detail.Record.Revoked {
			// The write path already settles the effect as blocked;
			// report the stored states without spending anything.
			return postResponse(pipe), nil
		}
		if detail.Source != nil && detail.Source.Product != nil && pipe.Judgment == factory.PostJudgmentApplied && pipe.Route == factory.PostRouteNewIssue {
			// Product revisions enter the existing explicit review/adoption
			// owner. Creating a detached issue here would duplicate its graph.
			return postResponse(pipe), nil
		}
		if pipe.Judgment == factory.PostJudgmentApplied && pipe.Route == factory.PostRouteNewIssue && wired != nil && wired.IssueMaker != nil && st.PostUsesIssueMaker(ctx, id, pipe.Revision) {
			return wired.IssueMaker(ctx, pipe, detail)
		}
		var advanced bool
		switch {
		case pipe.AdmitPostJudgment() && wired != nil && wired.Judge != nil:
			advanced, err = drivePostJudgment(ctx, st, wired, pipe, detail)
		case pipe.AdmitPostDraft() && wired != nil && wired.Drafter != nil:
			advanced, err = drivePostDraft(ctx, st, wired.Drafter, pipe, detail)
		case pipe.AdmitPostEffect() && wired != nil && wired.Creator != nil:
			advanced, err = drivePostEffect(ctx, st, wired.Creator, pipe, detail)
		case pipe.Route == factory.PostRouteExistingWork && pipe.Judgment == factory.PostJudgmentApplied && pipe.Effect == factory.PostEffectPending && pipe.Attempts < factory.PostAttemptLimit && wired != nil:
			advanced, err = drivePostFeedback(ctx, st, wired.Creator, pipe, detail)
		default:
			return postResponse(pipe), nil
		}
		if err != nil {
			// A concurrent advance wins honestly: report its stored
			// outcome instead of failing the replay.
			if errors.Is(err, store.ErrStaleRevision) {
				fresh, reloadErr := st.PostIntake(ctx, id)
				if reloadErr != nil {
					return empty, reloadErr
				}
				return postResponse(fresh), nil
			}
			return empty, err
		}
		if !advanced {
			fresh, err := st.PostIntake(ctx, id)
			if err != nil {
				return empty, err
			}
			return postResponse(fresh), nil
		}
	}
	pipe, err := st.PostIntake(ctx, id)
	if err != nil {
		return empty, err
	}
	return postResponse(pipe), nil
}

// postCandidateLimit bounds the existing-work candidates one
// routing judgment may weigh.
const postCandidateLimit = 8

// collectPostJudgeInput assembles one judgment's complete input
// from recorded state: the comment with its persisted source
// snapshot, its reply parent's recorded evidence, and the
// existing-work candidates merged bound-first (bound issue scope,
// source snapshot, reply parent, recent repository work), deduped by
// issue with recorded summaries filled from whichever origin has
// them. A failed recent-work read fails the judgment instead of
// routing blind; an unreadable or closed parent simply contributes
// no evidence.
func collectPostJudgeInput(ctx context.Context, st *store.Store, detail store.FeedPostDetail) (PostJudgeInput, error) {
	input := PostJudgeInput{SourcePost: detail}
	recent, err := st.RecentPostWorkCandidates(ctx, detail.Record.Repository, postCandidateLimit)
	if err != nil {
		return PostJudgeInput{}, err
	}
	byIssue := map[int64]*factory.PostWorkCandidate{}
	var ordered []*factory.PostWorkCandidate
	add := func(candidate factory.PostWorkCandidate) {
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
		add(factory.PostWorkCandidate{Issue: detail.Record.Issue, Origin: factory.PostCandidateBound})
	}
	if detail.Source != nil && detail.Source.Issue > 0 {
		add(factory.PostWorkCandidate{
			Issue: detail.Source.Issue, Summary: detail.Source.Summary,
			Status: detail.Source.Status, Origin: factory.PostCandidateSource,
		})
	}
	if detail.Record.ParentID != "" {
		parent := &PostParentEvidence{SourcePost: detail.Record.ParentID}
		if parentDetail, err := st.FeedPost(ctx, detail.Record.ParentID); err == nil &&
			parentDetail.Record.Repository == detail.Record.Repository &&
			!parentDetail.Record.Deleted && !parentDetail.Record.Revoked {
			parent.Body = store.TruncateFeedSummary(parentDetail.Body)
			parent.Issue = parentDetail.Record.Issue
		}
		if parentPipe, err := st.PostIntake(ctx, detail.Record.ParentID); err == nil && parent.Issue <= 0 {
			parent.Issue = parentPipe.Issue
		}
		input.Parent = parent
		if parent.Issue > 0 {
			add(factory.PostWorkCandidate{Issue: parent.Issue, Origin: factory.PostCandidateParent})
		}
	}
	for _, candidate := range recent {
		add(candidate)
	}
	for _, candidate := range ordered {
		if err := candidate.Validate(); err != nil {
			return PostJudgeInput{}, err
		}
		input.Candidates = append(input.Candidates, *candidate)
		if len(input.Candidates) == postCandidateLimit {
			break
		}
	}
	return input, nil
}

// applyPostJudgment advances one pipeline with a settled routing
// outcome. An existing_work judgment persists which work it selected
// in the pipeline's issue link; every other route clears it.
func drivePostFeedback(ctx context.Context, st *store.Store, creator PostIssueCreator, pipe factory.PostIntake, detail store.FeedPostDetail) (bool, error) {
	requested, err := st.PostFeedbackRequested(ctx, pipe.SourcePost, pipe.Revision)
	if err != nil || !requested {
		return false, err
	}
	writer, ok := creator.(interface {
		DeliverPostFeedback(context.Context, store.FeedPostDetail, int64) error
	})
	if !ok {
		_ = st.NotePostProgress(ctx, pipe.SourcePost, "Feedback delivery is unavailable. Restore Factory’s connection before retrying.")
		return false, nil
	}
	if err = writer.DeliverPostFeedback(ctx, detail, pipe.Issue); err != nil {
		_ = st.NotePostProgress(ctx, pipe.SourcePost, "Could not confirm delivery to the issue discussion. Check repository access and refresh before retrying.")
		return false, notePostFailure(ctx, st, pipe, false)
	}
	next := pipe
	next.Effect = factory.PostEffectApplied
	if err = st.AdvancePostIntake(ctx, pipe, next); err != nil {
		return false, err
	}
	_ = st.NotePostProgress(ctx, pipe.SourcePost, "Feedback added to the issue discussion. Accepted requirements have not been changed.")
	return true, nil
}

func applyPostJudgment(ctx context.Context, st *store.Store, pipe factory.PostIntake, judgment PostJudgment) error {
	next := pipe
	next.Judgment = factory.PostJudgmentApplied
	next.Route = judgment.Route
	next.Issue = 0
	if judgment.Route == factory.PostRouteExistingWork {
		next.Issue = judgment.Issue
	}
	if err := st.AdvancePostIntake(ctx, pipe, next); err != nil {
		return err
	}
	if judgment.Route == factory.PostRouteNewIssue {
		_ = st.NotePostAuthoring(ctx, pipe.SourcePost, pipe.Revision, factory.PostAuthoring{State: factory.PostAuthoringQueued, Detail: "JEV identified new work. Waiting for the Product Lead to prepare the issue."}, time.Now().Unix())
	}
	if judgment.Routing != nil {
		if err := st.NotePostRouting(ctx, pipe.SourcePost, *judgment.Routing); err != nil {
			return err
		}
	}
	if judgment.Route == factory.PostRouteClarify {
		question := judgment.Question
		if question == "" {
			question = postClarificationQuestions["clarify_result"]
		}
		return st.NotePostProgress(ctx, pipe.SourcePost, question)
	}
	return nil
}

// drivePostJudgment routes one comment with one JEV Choice and stores
// the answer on the request row. A failed call retries on a later pass,
// bounded by the attempt count.
func drivePostJudgment(ctx context.Context, st *store.Store, wired *PostCollaborators, pipe factory.PostIntake, detail store.FeedPostDetail) (bool, error) {
	// Gathering context reads this server and Forge only; a failure there
	// (Forge restarting, say) spends no JEV attempt. The request waits
	// with the reason visible and the next pass reads again.
	input, err := collectPostJudgeInput(ctx, st, detail)
	if err != nil {
		_ = st.NotePostProgress(ctx, pipe.SourcePost, "The original request context could not be read yet. It is read again shortly.")
		return false, nil
	}
	if wired.Prepare != nil {
		if input, err = wired.Prepare(ctx, input); err != nil {
			_ = st.NotePostProgress(ctx, pipe.SourcePost, err.Error())
			stall := ""
			if errors.Is(err, errPostIssueContext) {
				stall = postStalledContext
			}
			_ = st.NotePostStall(ctx, pipe.SourcePost, stall)
			return false, nil
		}
	}
	_ = st.NotePostStall(ctx, pipe.SourcePost, "")
	judgment, err := wired.Judge.JudgePost(ctx, input)
	if err != nil {
		_ = st.NotePostProgress(ctx, pipe.SourcePost, "JEV could not assess this request. Check the System One connection and retry assessment.")
		return false, notePostFailure(ctx, st, pipe, true)
	}
	switch judgment.Route {
	case factory.PostRouteNewIssue, factory.PostRouteDiscussion, factory.PostRouteClarify:
		if judgment.Issue != 0 {
			return false, notePostFailure(ctx, st, pipe, true)
		}
	case factory.PostRouteExistingWork:
		if _, err := postJudgmentFromSelection(postExistingChoice(judgment.Issue), input.Candidates); err != nil {
			return false, notePostFailure(ctx, st, pipe, true)
		}
	default:
		return false, notePostFailure(ctx, st, pipe, true)
	}
	if err := applyPostJudgment(ctx, st, pipe, judgment); err != nil {
		return false, err
	}
	return true, nil
}

// drivePostDraft runs one Product Lead LLM authoring call for a
// new_issue route. A malformed draft settles the draft as invalid
// before any native effect; a failed call keeps its attempt for retry.
func drivePostDraft(ctx context.Context, st *store.Store, drafter PostDrafter, pipe factory.PostIntake, detail store.FeedPostDetail) (bool, error) {
	draft, err := drafter.DraftPostIssue(ctx, detail)
	if err != nil {
		var pending *postDraftPending
		if errors.As(err, &pending) {
			_ = st.NotePostAuthoring(ctx, pipe.SourcePost, pipe.Revision, pending.progress, time.Now().Unix())
		} else if ctx.Err() == nil {
			state := factory.PostAuthoringFailed
			if errors.Is(err, ErrPostDraftPending) {
				state = factory.PostAuthoringUnknown
			}
			_ = st.NotePostAuthoring(ctx, pipe.SourcePost, pipe.Revision, factory.PostAuthoring{State: state, Detail: err.Error()}, time.Now().Unix())
		}
		// A launched planning run that is still settling — or a drive
		// interrupted by service shutdown — is not an authoring
		// failure: the draft stays pending with its bounded attempts
		// intact and the next service pass resumes the same funded
		// attempt instead of spending again.
		if errors.Is(err, ErrPostDraftPending) || ctx.Err() != nil {
			_ = st.NotePostProgress(ctx, pipe.SourcePost, err.Error())
			return false, nil
		}
		_ = st.NotePostProgress(ctx, pipe.SourcePost, err.Error())
		return false, notePostFailure(ctx, st, pipe, false)
	}
	draft.SourcePost = pipe.SourcePost
	draft.Revision = pipe.Revision
	if err := draft.Validate(); err != nil {
		next := pipe
		next.Draft = factory.PostDraftInvalid
		next.Effect = factory.PostEffectBlocked
		if advanceErr := st.AdvancePostIntake(ctx, pipe, next); advanceErr != nil {
			return false, advanceErr
		}
		if wipeErr := st.DiscardPostDraftData(ctx, pipe.SourcePost); wipeErr != nil {
			return false, wipeErr
		}
		return true, nil
	}
	if err := st.RecordPostDraft(ctx, draft); err != nil {
		return false, err
	}
	next := pipe
	next.Draft = factory.PostDraftApplied
	if err := st.AdvancePostIntake(ctx, pipe, next); err != nil {
		return false, err
	}
	return true, nil
}

// drivePostEffect creates the native issue behind an applied draft.
// It reconciles first: an issue already created from this comment (lost
// effect reply) settles without duplicating, and the graph operation is
// idempotent. The draft must still bind the current source revision.
func drivePostEffect(ctx context.Context, st *store.Store, creator PostIssueCreator, pipe factory.PostIntake, detail store.FeedPostDetail) (bool, error) {
	if detail.Record.Revision != pipe.Revision {
		return false, nil
	}
	draft, err := st.PostDraft(ctx, pipe.SourcePost, pipe.Revision)
	if err != nil {
		return false, notePostFailure(ctx, st, pipe, false)
	}
	issue, err := creator.LookupPostIssue(ctx, detail.Record.Repository, pipe.SourcePost)
	if err != nil || issue <= 0 {
		issue, err = creator.CreatePostIssue(ctx, draft, detail.Record.Repository)
	}
	if err != nil || issue <= 0 {
		_ = st.NotePostProgress(ctx, pipe.SourcePost, "Could not confirm issue creation. Retry checks for an existing issue before creating one.")
		return false, notePostFailure(ctx, st, pipe, false)
	}
	next := pipe
	next.Effect = factory.PostEffectVerified
	next.Issue = issue
	if err := st.AdvancePostIntake(ctx, pipe, next); err != nil {
		return false, err
	}
	return true, nil
}

// notePostFailure records one retry attempt; when the bounded
// attempts run out the open stage settles honestly (judgment failed,
// otherwise the draft/effect stays retryable only while attempts
// remain, and the caller sees pending until then).
func notePostFailure(ctx context.Context, st *store.Store, pipe factory.PostIntake, failJudgment bool) error {
	if err := st.NotePostAttempt(ctx, pipe.SourcePost); err != nil {
		if failJudgment {
			next := pipe
			next.Judgment = factory.PostJudgmentFailed
			_ = st.AdvancePostIntake(ctx, pipe, next)
		}
		return nil
	}
	fresh, err := st.PostIntake(ctx, pipe.SourcePost)
	if err != nil {
		return err
	}
	if fresh.Attempts < factory.PostAttemptLimit {
		return nil
	}
	next := fresh
	if failJudgment && next.Judgment == factory.PostJudgmentPending {
		next.Judgment = factory.PostJudgmentFailed
	}
	if advanceErr := st.AdvancePostIntake(ctx, fresh, next); advanceErr != nil {
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
		entries[i].ParentID = detail.Record.ParentID
		if detail.Actor != "" {
			entries[i].Actor = detail.Actor
		}
		entries[i].Assignment = detail.Assignment
		entries[i].EventIdentity = detail.EventIdentity
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

// postExistingIssue names the already-tracked issue one pipeline
// linked on the existing_work route, zero on every other route.
func postExistingIssue(pipe factory.PostIntake) int64 {
	if pipe.Route == factory.PostRouteExistingWork {
		return pipe.Issue
	}
	return 0
}

// postResponse renders one stored pipeline row as the POST answer.
func postResponse(pipe factory.PostIntake) factory.PostResponse {
	return factory.PostResponse{
		Attempts: pipe.Attempts, AttemptLimit: factory.PostAttemptLimit,
		SourcePost: pipe.SourcePost, Revision: pipe.Revision, Intake: pipe.Intake,
		Judgment: pipe.Judgment, Route: pipe.Route, Draft: pipe.Draft,
		Effect: pipe.Effect, Issue: pipe.Issue, ExistingIssue: postExistingIssue(pipe),
	}
}

// postDriveLimit bounds the stored comment requests one service
// pass advances: authoring and native effects are expensive, and the
// coordinator pass must stay responsive to factory work.
const postDriveLimit = 4

// drivePendingPosts advances one bounded page of durable comment
// requests through their wired stages and reports how many drove
// without error. Per-request failures keep their stored state for a
// later pass; exactly-once advancement rests on the pipeline CAS, the
// single funded planning attempt and the deterministic native operation
// identity.
func drivePendingPosts(ctx context.Context, st *store.Store, wired *PostCollaborators, limit int) int {
	if st == nil {
		return 0
	}
	pending, err := st.PendingPostIntakes(ctx, limit)
	if err != nil {
		return 0
	}
	driven := 0
	for _, pipe := range pending {
		if ctx.Err() != nil {
			break
		}
		_ = st.NotePostProgress(ctx, pipe.SourcePost, "")
		if _, err := drivePostIntake(ctx, st, wired, pipe.SourcePost); err == nil {
			driven++
		}
	}
	return driven
}

// DrivePendingPosts is the service-owned durable continuation for
// feed comments, bound to the coordinator's Start and progress
// lifecycle by the server wiring. It advances stored comment
// requests independently of any browser request, so a submission
// whose POST already answered — or whose service restarted mid-
// flight — still reaches its recorded issue exactly once.
func (s *API) DrivePendingPosts(ctx context.Context) {
	if s == nil || s.Store == nil {
		return
	}
	_ = drivePendingPosts(ctx, s.Store, s.postCollaborators(), postDriveLimit)
	s.driveContributions(ctx)
}
