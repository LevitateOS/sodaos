package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiDepartments(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method == http.MethodGet {
		result, err := s.Store.Departments(r.Context(), v.User.ID)
		if err != nil {
			factoryCommandError(w, err)
			return
		}
		auth.JSONResponse(w, 200, result)
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var d factory.Department
	if !auth.DecodeAPIObject(w, r, &d) {
		return
	}
	d.Owner = v.User.ID
	if err := d.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_department", err.Error())
		return
	}
	if err := s.Store.CreateDepartment(r.Context(), d); err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, d)
}

func (s *API) apiIntent(w http.ResponseWriter, r *http.Request, v store.Session) {
	result, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 404, "not_found", "Build request not found.")
		return
	}
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}

func (s *API) apiIntents(w http.ResponseWriter, r *http.Request, v store.Session) {
	q := r.URL.Query()
	limit := 20
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			auth.JSONError(w, 400, "invalid_list", "Invalid list limit.")
			return
		}
		limit = n
	}
	var repository int64
	if raw := q.Get("repository"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			auth.JSONError(w, 400, "invalid_list", "Invalid repository filter.")
			return
		}
		repository = n
	}
	f := factory.IntentListFilter{Cursor: q.Get("cursor"), State: q.Get("state"), Repository: repository, Department: q.Get("department"), Limit: limit}
	if err := f.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_list", err.Error())
		return
	}
	result, next, err := s.Store.ListIntentRevisions(r.Context(), v.User.ID, f)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	out := map[string]any{"intents": result}
	if next != "" {
		out["next_cursor"] = next
	}
	auth.JSONResponse(w, 200, out)
}

func (s *API) apiIntentHistory(w http.ResponseWriter, r *http.Request, v store.Session) {
	intent := r.PathValue("intentID")
	if !project.ValidDecisionID(intent) {
		auth.JSONError(w, 404, "not_found", "Build request not found.")
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			auth.JSONError(w, 400, "invalid_history", "Invalid history limit.")
			return
		}
		limit = n
	}
	result, err := s.Store.IntentHistory(r.Context(), intent, v.User.ID, limit)
	if errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 404, "not_found", "Build request not found.")
		return
	}
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, map[string]any{"intent": intent, "revisions": result})
}

func (s *API) apiIntentRevision(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factory.IntentRevision
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	in.Owner = v.User.ID
	in.CreatedUnix = time.Now().Unix()
	if current, err := s.Store.IntentRevision(r.Context(), in.Intent, v.User.ID); err == nil && current.ID == in.ID {
		in.CreatedUnix = current.CreatedUnix
	}
	if in.Repository > 0 && !s.factoryOwner(w, r, v, in.Repository) {
		return
	}
	if err := in.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_intent", err.Error())
		return
	}
	if in.Predecessor != "" {
		current, err := s.Store.IntentRevision(r.Context(), in.Intent, v.User.ID)
		if err != nil {
			factoryCommandError(w, err)
			return
		}
		if current.ID != in.Predecessor && current.ID != in.ID {
			auth.JSONError(w, 412, "stale_revision", "The reviewed build request changed.")
			return
		}
		if current.ID == in.ID {
			in.CreatedUnix = current.CreatedUnix
		}
	}
	if err := s.Store.SaveIntentRevision(r.Context(), in); err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, in)
}
