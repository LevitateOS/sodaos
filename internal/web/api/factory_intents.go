package api

import (
	"context"
	"database/sql"
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

func (s *API) apiIntentAllowance(w http.ResponseWriter, r *http.Request, v store.Session) {
	result, err := s.Store.PlanningAllowance(r.Context(), r.PathValue("intentID"), v.User.ID)
	if errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 404, "allowance_not_found", "No planning allowance saved for this request.")
		return
	}
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}

func (s *API) apiIntentAllowanceSave(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factory.PlanningAllowance
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	in.Owner = v.User.ID
	in.CreatedUnix = time.Now().Unix()
	out, err := s.storeIntentAllowance(r.Context(), v.User.ID, in)
	if err != nil {
		var clientErr *allowanceWriteError
		if errors.As(err, &clientErr) {
			auth.JSONError(w, clientErr.code, clientErr.name, clientErr.message)
			return
		}
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, out)
}

type allowanceWriteError struct {
	code    int
	name    string
	message string
}

func (e *allowanceWriteError) Error() string { return e.name + ": " + e.message }

// storeIntentAllowance persists one allowance revision past the session
// wall: intent binding, server-grounded defaults, validation, CAS save.
// Browsers cannot source authoritative limits, so the server grounds them:
// a ceiling below the amount rises to the operator budget (covering the
// amount, capped at field bounds; explicit sufficient ceilings pass
// through), and counters come from the live head, or zero on first save
// where nothing could have been charged. Exact replays keep their bytes.
// Status mapping matches the endpoint exactly; session/auth failures stay
// in the handler.
func (s *API) storeIntentAllowance(ctx context.Context, userID int64, in factory.PlanningAllowance) (factory.PlanningAllowance, error) {
	if _, err := s.Store.IntentRevision(ctx, in.Intent, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return factory.PlanningAllowance{}, &allowanceWriteError{code: 404, name: "not_found", message: "Build request not found."}
		}
		return factory.PlanningAllowance{}, err
	}
	current, err := s.Store.PlanningAllowance(ctx, in.Intent, userID)
	switch {
	case err == nil && current.ID == in.ID:
		in.CreatedUnix = current.CreatedUnix
	case err == nil:
		in.Reserved, in.Consumed = current.Reserved, current.Consumed
	case errors.Is(err, sql.ErrNoRows):
		in.Reserved, in.Consumed = 0, 0
	}
	if in.Ceiling < in.Amount {
		if fill := s.Store.PriorityCallBudget; fill > in.Ceiling {
			in.Ceiling = fill
		}
		if in.Ceiling < in.Amount {
			in.Ceiling = in.Amount
		}
		if in.Ceiling > 100 {
			in.Ceiling = 100
		}
	}
	if err := in.Validate(); err != nil {
		return factory.PlanningAllowance{}, &allowanceWriteError{code: 400, name: "invalid_allowance", message: err.Error()}
	}
	if err := s.Store.SavePlanningAllowance(ctx, in); err != nil {
		return factory.PlanningAllowance{}, err
	}
	return in, nil
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
