package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
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
