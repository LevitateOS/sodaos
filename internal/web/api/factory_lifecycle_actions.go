package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// lifecycleConfigView renders one event's optional-effect configuration:
// exact actor/targets, revision, enablement and that event's own
// capability proof state. Unproved events render enabled=false
// regardless of any stored wish; the store refuses enabling them.
type lifecycleConfigView struct {
	Event        string   `json:"event"`
	Revision     string   `json:"revision"`
	Enabled      bool     `json:"enabled"`
	Capable      bool     `json:"capable"`
	ActorID      string   `json:"actor_id"`
	RepositoryID string   `json:"repository_id"`
	LabelID      string   `json:"label_id,omitempty"`
	AssigneeID   string   `json:"assignee_id,omitempty"`
	ReviewerIDs  []string `json:"reviewer_ids,omitempty"`
	Note         string   `json:"note,omitempty"`
}

func lifecycleConfigDTO(c factory.LifecycleConfig, capable bool) lifecycleConfigView {
	view := lifecycleConfigView{
		Event: c.Event, Revision: strconv.FormatInt(c.Revision, 10),
		Enabled: c.Enabled && capable, Capable: capable,
		ActorID:      strconv.FormatInt(c.ActorID, 10),
		RepositoryID: strconv.FormatInt(c.RepositoryID, 10),
		Note:         c.Note,
	}
	if c.LabelID > 0 {
		view.LabelID = strconv.FormatInt(c.LabelID, 10)
	}
	if c.AssigneeID > 0 {
		view.AssigneeID = strconv.FormatInt(c.AssigneeID, 10)
	}
	for _, id := range c.ReviewerIDs {
		view.ReviewerIDs = append(view.ReviewerIDs, strconv.FormatInt(id, 10))
	}
	return view
}

// lifecycleActionView renders one durable action with per-target results
// and attempt budget. Operation identities are exposed so clients can
// correlate native receipts; they grant nothing by themselves.
type lifecycleActionView struct {
	ID          string            `json:"id"`
	Event       string            `json:"event"`
	Action      string            `json:"action"`
	ActorID     string            `json:"actor_id"`
	IssueID     string            `json:"issue_id"`
	Attempts    int               `json:"attempts"`
	MaxAttempts int               `json:"max_attempts"`
	Revoked     bool              `json:"revoked"`
	Settled     bool              `json:"settled"`
	Results     map[string]string `json:"results,omitempty"`
}

func lifecycleActionDTO(a factory.LifecycleAction) lifecycleActionView {
	return lifecycleActionView{
		ID: a.ID, Event: a.Event, Action: a.Action,
		ActorID:  strconv.FormatInt(a.ActorID, 10),
		IssueID:  strconv.FormatInt(a.IssueID, 10),
		Attempts: a.Attempts, MaxAttempts: factory.MaxLifecycleAttempts,
		Revoked: a.Revoked, Settled: a.Settled(), Results: a.Results,
	}
}

// lifecycleConfigSave is the bounded owner-supplied configuration body.
// Actor and repository bind to the path repository and current native
// identity; unknown fields are rejected.
type lifecycleConfigSave struct {
	Revision    int64   `json:"revision"`
	Enabled     bool    `json:"enabled"`
	LabelID     int64   `json:"label_id,omitempty"`
	AssigneeID  int64   `json:"assignee_id,omitempty"`
	ReviewerIDs []int64 `json:"reviewer_ids,omitempty"`
	Note        string  `json:"note,omitempty"`
}

func decodeLifecycleConfigSave(body []byte) (lifecycleConfigSave, error) {
	var save lifecycleConfigSave
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&save); err != nil {
		return lifecycleConfigSave{}, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return lifecycleConfigSave{}, errors.New("trailing data")
	}
	return save, nil
}

// apiFactoryLifecycleConfig reads one event's configuration. Current
// repository visibility admits the read.
func (s *API) apiFactoryLifecycleConfig(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	event := r.PathValue("event")
	if !factory.ValidLifecycleEvent(event) {
		auth.JSONError(w, 404, "not_found", "Unknown lifecycle event.")
		return
	}
	capable, err := s.Store.CapabilityProved(r.Context(), event)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read capability proof.")
		return
	}
	c, err := s.Store.LifecycleConfig(r.Context(), event)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONResponse(w, 200, absentLifecycleConfigView(event, repository, capable))
			return
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read lifecycle configuration.")
		return
	}
	if c.RepositoryID != repository {
		auth.JSONError(w, 404, "not_found", "No lifecycle configuration for this repository.")
		return
	}
	auth.JSONResponse(w, 200, lifecycleConfigDTO(c, capable))
}

// absentLifecycleConfigView renders the default for an event with no
// saved configuration: disabled settings with truthful capability state.
func absentLifecycleConfigView(event string, repository int64, capable bool) lifecycleConfigView {
	return lifecycleConfigView{Event: event, Revision: "0", Capable: capable, RepositoryID: strconv.FormatInt(repository, 10)}
}

// apiFactoryLifecycleConfigSave stores one event's configuration. Only
// the current repository owner or administrator may configure effects;
// enabling additionally requires that event's own native capability
// proof, enforced by the store.
func (s *API) apiFactoryLifecycleConfigSave(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	if !s.factoryOwner(w, r, v, repository) {
		return
	}
	event := r.PathValue("event")
	if !factory.ValidLifecycleEvent(event) {
		auth.JSONError(w, 404, "not_found", "Unknown lifecycle event.")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(auth.APIBodyLimit)+1))
	if err != nil || len(body) > auth.APIBodyLimit {
		auth.JSONError(w, 413, "body_too_large", "Configuration body exceeds the limit.")
		return
	}
	save, err := decodeLifecycleConfigSave(body)
	if err != nil {
		auth.JSONError(w, 400, "invalid_request", "Configuration body is malformed.")
		return
	}
	c := factory.LifecycleConfig{
		Event: event, Revision: save.Revision, Enabled: save.Enabled,
		ActorID: v.User.ID, RepositoryID: repository,
		LabelID: save.LabelID, AssigneeID: save.AssigneeID,
		ReviewerIDs: save.ReviewerIDs, Note: save.Note,
	}
	stored, err := s.Store.SaveConfig(r.Context(), c)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrCapabilityUnproved):
			auth.JSONError(w, 409, "capability_unproved", "This event has no verified native capability yet.")
		case errors.Is(err, store.ErrLifecycleActionStale):
			auth.JSONError(w, 412, "stale_revision", "Configuration changed; refresh and retry.")
		case errors.Is(err, factory.ErrLifecycleAction):
			auth.JSONError(w, 400, "invalid_request", "Configuration violates event bounds.")
		default:
			auth.JSONError(w, 503, "store_unavailable", "Could not save lifecycle configuration.")
		}
		return
	}
	capable, err := s.Store.CapabilityProved(r.Context(), event)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read capability proof.")
		return
	}
	auth.JSONResponse(w, 200, lifecycleConfigDTO(stored, capable))
}

// apiFactoryLifecycleActions lists bounded durable action results for one
// issue/event. Current repository visibility admits the read; the issue
// query parameter is required.
func (s *API) apiFactoryLifecycleActions(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	event := r.PathValue("event")
	if event != "" && !factory.ValidLifecycleEvent(event) {
		auth.JSONError(w, 404, "not_found", "Unknown lifecycle event.")
		return
	}
	issue, valid := auth.PositiveID(r.URL.Query().Get("issue"))
	if !valid {
		auth.JSONError(w, 400, "invalid_request", "A positive issue parameter is required.")
		return
	}
	actions, err := s.Store.ListActions(r.Context(), repository, issue, event, 64)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read lifecycle actions.")
		return
	}
	views := make([]lifecycleActionView, 0, len(actions))
	for _, a := range actions {
		views = append(views, lifecycleActionDTO(a))
	}
	auth.JSONResponse(w, 200, map[string]any{"actions": views})
}
