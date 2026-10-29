package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Opening the exact comment proposal saves a draft only. The ordinary intent
// planner, review, acceptance and generation retain all later authority.
func (s *API) apiFactoryFeedProductReview(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in struct {
		Revision int64 `json:"revision"`
	}
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	detail, err := s.Store.FeedComment(r.Context(), r.PathValue("id"))
	if err != nil || detail.Source == nil || detail.Source.Product == nil {
		auth.JSONError(w, 404, "product_change_not_found", "Product change is not available.")
		return
	}
	if !s.factoryOwner(w, r, v, detail.Record.Repository) {
		return
	}
	if _, err = s.Store.IntentRevision(r.Context(), detail.Source.Product.Intent, v.User.ID); err != nil {
		auth.JSONError(w, 404, "product_change_not_found", "Product change is not available.")
		return
	}
	if _, err = s.Store.ReviewCommentProductChange(r.Context(), detail.Record.ID, in.Revision, v.User.ID, time.Now().Unix()); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			auth.JSONError(w, 412, "stale_revision", "The comment or accepted product changed. Refresh before reviewing.")
		} else if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "product_change_not_found", "Product change is not available.")
		} else {
			factoryCommandError(w, err)
		}
		return
	}
	detail, err = s.Store.FeedComment(r.Context(), detail.Record.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	pipe, err := s.Store.CommentPipeline(r.Context(), detail.Record.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	progress, _ := s.Store.CommentProgress(r.Context(), detail.Record.ID)
	auth.JSONResponse(w, 200, s.feedCommentWithProduct(r, v, detail, pipe, progress))
}

func (s *API) canReviewCommentProduct(r *http.Request, v store.Session, repository int64) bool {
	if !s.extensionSessionCurrent(r.Context(), r, v) {
		return false
	}
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		return false
	}
	if access.repository.Permissions != nil && access.repository.Permissions.Admin {
		return true
	}
	allowed, err := s.environmentAdministrator(r, access)
	return err == nil && allowed
}

func (s *API) feedCommentWithProduct(r *http.Request, v store.Session, detail store.FeedCommentDetail, pipe factory.CommentPipeline, progress string) map[string]any {
	response := feedCommentResponse(detail, pipe, progress, strconv.FormatInt(v.User.ID, 10))
	if routing, err := s.Store.CommentRouting(r.Context(), detail.Record.ID); err == nil && routing != nil {
		response["routing"] = routing
	}
	if !detail.Record.Deleted && !detail.Record.Revoked {
		if stalled := commentStalled(pipe, s.commentCollaborators()); stalled != "" {
			response["stalled"] = stalled
		}
	}
	if view, err := s.Store.CommentProductChange(r.Context(), detail, pipe, v.User.ID); err == nil && view != nil {
		view.CanReview = view.CanReview && s.canOpenCommentProduct(r, v, detail.Record.Repository, view)
		response["product_change"] = view
		response["can_retry"] = false
	} else if err != nil && isCommentProductLane(detail, pipe) {
		// The lane is same-product but the read failed: project the miss
		// explicitly so the comment never falls back to detached generic
		// progress that would claim unrelated native work.
		response["product_change"] = store.UnavailableCommentProductChange(detail.Source.Product.Intent, detail.Source.Product.AcceptedRevision)
		response["can_retry"] = false
	}
	return response
}

// isCommentProductLane reports whether the comment sits on the same-product
// review handoff: an accepted source behind an applied new-issue pipeline
// with no linked native issue. Only this lane may carry product_change, so
// only this lane maps a projection error to the explicit unavailable view.
func isCommentProductLane(detail store.FeedCommentDetail, pipe factory.CommentPipeline) bool {
	return detail.Source != nil && detail.Source.Product != nil &&
		pipe.Judgment == factory.CommentJudgmentApplied && pipe.Route == factory.CommentRouteNewIssue && pipe.Issue == 0
}

// canOpenCommentProduct separates owner-read capability from admin review
// authorization. The accepted feed action only opens the owner's private
// build request behind it — the same owner-private gate as GET
// /factory/intents/{id} — so an owner with repository Read keeps it without
// repository admin. A collaborator stays denied by that owner-private gate.
// Every other state still opens the admin-gated review write.
func (s *API) canOpenCommentProduct(r *http.Request, v store.Session, repository int64, view *store.CommentProductChange) bool {
	if view.State == "accepted" {
		_, err := s.Store.IntentRevision(r.Context(), view.Intent, v.User.ID)
		return err == nil
	}
	return s.canReviewCommentProduct(r, v, repository)
}
