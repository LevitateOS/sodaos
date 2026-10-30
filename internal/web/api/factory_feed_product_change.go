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
	detail, err := s.Store.FeedPost(r.Context(), r.PathValue("id"))
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
	if _, err = s.Store.ReviewPostProductChange(r.Context(), detail.Record.ID, in.Revision, v.User.ID, time.Now().Unix()); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			auth.JSONError(w, 412, "stale_revision", "The comment or accepted product changed. Refresh before reviewing.")
		} else if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "product_change_not_found", "Product change is not available.")
		} else {
			factoryCommandError(w, err)
		}
		return
	}
	detail, err = s.Store.FeedPost(r.Context(), detail.Record.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	pipe, err := s.Store.PostIntake(r.Context(), detail.Record.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	progress, _ := s.Store.PostProgress(r.Context(), detail.Record.ID)
	auth.JSONResponse(w, 200, s.feedPostWithProduct(r, v, detail, pipe, progress))
}

func (s *API) canReviewPostProduct(r *http.Request, v store.Session, repository int64) bool {
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

func (s *API) feedPostWithProduct(r *http.Request, v store.Session, detail store.FeedPostDetail, pipe factory.PostIntake, progress string) map[string]any {
	response := feedPostResponse(detail, pipe, progress, strconv.FormatInt(v.User.ID, 10))
	if maker, err := s.Store.PostIssueMaker(r.Context(), detail.Record.ID, pipe.Revision); err == nil {
		response["issue_maker"] = map[string]any{"post": maker.Scope.Post, "state": maker.State, "detail": maker.Detail, "checked": maker.Checked}
	} else if pipe.Judgment == factory.PostJudgmentApplied && pipe.Route == factory.PostRouteNewIssue && (detail.Source == nil || detail.Source.Product == nil) && s.Store.PostUsesIssueMaker(r.Context(), detail.Record.ID, pipe.Revision) {
		response["issue_maker"] = map[string]any{"post": "", "state": "queued"}
	}
	if pipe.Route == factory.PostRouteNewIssue && pipe.Draft == factory.PostDraftPending {
		if authoring, err := s.Store.PostAuthoring(r.Context(), detail.Record.ID); err == nil && authoring != nil {
			copy := *authoring
			if detail.Record.Author != strconv.FormatInt(v.User.ID, 10) {
				copy.Intent = ""
			}
			response["authoring"] = copy
			if copy.State == factory.PostAuthoringWaiting && copy.Run == "" && detail.Record.Author == strconv.FormatInt(v.User.ID, 10) {
				response["can_retry"] = true
			}
		}
	}
	if pipe.AdmitPostJudgment() && detail.Record.Author == strconv.FormatInt(v.User.ID, 10) && !detail.Record.Deleted && !detail.Record.Revoked {
		if _, err := s.Store.PostIssueContext(r.Context(), detail.Record.ID, detail.Record.Revision); err != nil {
			response["stalled"] = postStalledContext
			response["can_retry"] = true
		}
	}
	if routing, err := s.Store.PostRouting(r.Context(), detail.Record.ID); err == nil && routing != nil {
		response["routing"] = routing
	}
	if !detail.Record.Deleted && !detail.Record.Revoked {
		if stalled := postStalled(pipe, s.postCollaborators()); stalled != "" {
			response["stalled"] = stalled
		} else if stall, err := s.Store.PostStall(r.Context(), detail.Record.ID); err == nil && stall != "" && pipe.AdmitPostJudgment() && response["stalled"] == nil {
			response["stalled"] = stall
		}
	}
	if view, err := s.Store.PostProductChange(r.Context(), detail, pipe, v.User.ID); err == nil && view != nil {
		view.CanReview = view.CanReview && s.canOpenPostProduct(r, v, detail.Record.Repository, view)
		response["product_change"] = view
		response["can_retry"] = false
	} else if err != nil && isPostProductLane(detail, pipe) {
		// The lane is same-product but the read failed: project the miss
		// explicitly so the comment never falls back to detached generic
		// progress that would claim unrelated native work.
		response["product_change"] = store.UnavailablePostProductChange(detail.Source.Product.Intent, detail.Source.Product.AcceptedRevision)
		response["can_retry"] = false
	}
	return response
}

// isPostProductLane reports whether the comment sits on the same-product
// review handoff: an accepted source behind an applied new-issue pipeline
// with no linked native issue. Only this lane may carry product_change, so
// only this lane maps a projection error to the explicit unavailable view.
func isPostProductLane(detail store.FeedPostDetail, pipe factory.PostIntake) bool {
	return detail.Source != nil && detail.Source.Product != nil &&
		pipe.Judgment == factory.PostJudgmentApplied && pipe.Route == factory.PostRouteNewIssue && pipe.Issue == 0
}

// canOpenPostProduct separates owner-read capability from admin review
// authorization. The accepted feed action only opens the owner's private
// build request behind it — the same owner-private gate as GET
// /factory/intents/{id} — so an owner with repository Read keeps it without
// repository admin. A collaborator stays denied by that owner-private gate.
// Every other state still opens the admin-gated review write.
func (s *API) canOpenPostProduct(r *http.Request, v store.Session, repository int64, view *store.PostProductChange) bool {
	if view.State == "accepted" {
		_, err := s.Store.IntentRevision(r.Context(), view.Intent, v.User.ID)
		return err == nil
	}
	return s.canReviewPostProduct(r, v, repository)
}
