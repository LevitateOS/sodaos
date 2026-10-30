package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// Issue Maker captures first, then refines. Native creation is not held behind
// a model launch, source preparation or a product decomposition result.
func originalRequestIssue(comment store.FeedPostDetail) factory.IssueDraft {
	objective := comment.Body
	if len(objective) > 2048 {
		objective = "Implement the original request preserved in this issue."
	}
	title := postDraftTitle(comment.Body)
	for len(title) > 255 || !utf8.ValidString(title) {
		title = title[:len(title)-1]
	}
	return factory.IssueDraft{SourcePost: comment.Record.ID, Revision: comment.Record.Revision, Title: title,
		Body:      comment.Body,
		Objective: objective, Acceptance: "Pending refinement from the original request.", Metadata: `{"source":"feed_comment","role":"issue_maker","refinement":"pending"}`}
}

func (s *API) drivePostIssueMaker(ctx context.Context, pipe factory.PostIntake, comment store.FeedPostDetail) (factory.PostResponse, error) {
	progress, err := s.Store.StartPostIssueMaker(ctx, comment)
	if err != nil {
		return factory.PostResponse{}, err
	}
	save := func(state, detail string) {
		progress.State, progress.Detail = state, detail
		_ = s.Store.SaveIssueMakerProgress(ctx, progress)
	}
	if pipe.Effect != factory.PostEffectVerified {
		if s.issueMakerNative() == nil {
			save("waiting", "Issue Maker is waiting for repository issue-write access. No issue has been created yet.")
			return postResponse(pipe), nil
		}
		save("creating", "Recording your original request as an issue. AI refinement follows afterward.")
		if pipe.AdmitPostDraft() {
			draft := originalRequestIssue(comment)
			if err = draft.Validate(); err != nil {
				save("failed", err.Error())
				return postResponse(pipe), err
			}
			if err = s.Store.RecordPostDraft(ctx, draft); err != nil {
				return factory.PostResponse{}, err
			}
			next := pipe
			next.Draft = factory.PostDraftApplied
			if err = s.Store.AdvancePostIntake(ctx, pipe, next); err != nil {
				return factory.PostResponse{}, err
			}
			pipe = next
		}
		creator := &intakePostIssueCreator{API: s}
		if _, err = drivePostEffect(ctx, s.Store, creator, pipe, comment); err != nil {
			return factory.PostResponse{}, err
		}
		pipe, err = s.Store.PostIntake(ctx, pipe.SourcePost)
		if err != nil {
			return factory.PostResponse{}, err
		}
		if pipe.Effect != factory.PostEffectVerified {
			next := pipe
			next.Effect = factory.PostEffectFailed
			if err = s.Store.AdvancePostIntake(ctx, pipe, next); err == nil {
				pipe = next
			}
			state := "waiting"
			if pipe.Attempts >= factory.PostAttemptLimit {
				state = "failed"
			}
			save(state, "Issue Maker could not confirm native issue creation. Restore issue-write access and retry the original request; the recorded operation is reconciled before another creation.")
			return postResponse(pipe), nil
		}
	}
	if progress.Scope.Issue == 0 {
		draft, err := s.Store.PostDraft(ctx, pipe.SourcePost, pipe.Revision)
		if err != nil {
			return factory.PostResponse{}, err
		}
		if err = s.Store.RecordIssueMakerIssue(ctx, progress.Scope, "request", 0, pipe.Issue, draft.Title, postIssueBody(draft)); err != nil {
			return factory.PostResponse{}, err
		}
		progress, err = s.Store.PostIssueMaker(ctx, pipe.SourcePost, pipe.Revision)
		if err != nil {
			return factory.PostResponse{}, err
		}
		save("queued", fmt.Sprintf("Issue #%d is created. Issue Maker is queued to refine it; no AI run has started yet.", pipe.Issue))
		// Return after the first durable receipt so it is visible before any
		// preparation or model call is attempted on the next service pass.
		return postResponse(pipe), nil
	}
	if progress.State == "completed" || progress.State == "failed" {
		return postResponse(pipe), nil
	}
	if grant, err := s.Store.IssueIntakeGrant(ctx, pipe.SourcePost, pipe.Revision); err != nil || grant.Claims.Expires <= time.Now().Unix() {
		save("failed", "The issue exists. Its request-scoped write authorization expired before refinement; submit a follow-up request to continue.")
		return postResponse(pipe), nil
	}
	if s.Coordinator == nil {
		save("waiting", "The initial issue exists. AI refinement is waiting for an available model account.")
		return postResponse(pipe), nil
	}
	drafter := &IntentPostDrafter{Store: s.Store, Transport: s.Coordinator, IssueMaker: &progress.Scope}
	draft, err := drafter.DraftPostIssue(ctx, comment)
	if err != nil {
		var pending *postDraftPending
		if errors.As(err, &pending) {
			state := pending.progress.State
			progress.Run = pending.progress.Run
			detail := strings.ReplaceAll(pending.progress.Detail, "Product Lead", "Issue Maker")
			save(state, fmt.Sprintf("Issue #%d exists. %s", pipe.Issue, detail))
			_ = s.Store.NotePostAuthoring(ctx, pipe.SourcePost, pipe.Revision, pending.progress, time.Now().Unix())
			return postResponse(pipe), nil
		}
		if ctx.Err() != nil {
			return postResponse(pipe), ctx.Err()
		}
		save("failed", fmt.Sprintf("Issue #%d exists, but refinement failed: %s", pipe.Issue, store.TruncateFeedSummary(err.Error())))
		return postResponse(pipe), nil
	}
	if err = draft.Validate(); err != nil {
		save("failed", "The issue exists, but Issue Maker returned invalid refinement.")
		return postResponse(pipe), nil
	}
	if _, err = s.mutateIssueMakerIssue(ctx, progress.Scope, "refine", "request", draft.Title, postIssueBody(draft), "final-refinement"); err != nil {
		save("failed", "The issue exists, but its refinement could not be applied: "+store.TruncateFeedSummary(err.Error()))
		return postResponse(pipe), nil
	}
	save("completed", "Issue Maker finished. Each issue it created is listed in a comment below.")
	return postResponse(pipe), nil
}
