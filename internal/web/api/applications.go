package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type applicationView struct {
	Application     application.Spec         `json:"application"`
	AdministratorID string                   `json:"administrator_id"`
	Observation     *application.Observation `json:"observation"`
	Operator        bool                     `json:"operator"`
	Project         *applicationProjectView  `json:"project,omitempty"`
}

type applicationProjectView struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Path       string `json:"path"`
}

func (s *API) applicationView(r *http.Request, v store.Session, record application.Record, operator bool) applicationView {
	view := appView(record, operator)
	if repository, err := s.Store.ApplicationProject(r.Context(), record.Application.ID); err == nil && repository > 0 {
		if access, err := s.visibleRepository(r, v, repository); err == nil {
			view.Project = &applicationProjectView{Repository: strconv.FormatInt(repository, 10), Name: access.repository.FullName, Path: "/" + access.repository.FullName}
		}
	}
	return view
}

func appView(r application.Record, operator bool) applicationView {
	return applicationView{Application: r.Application, AdministratorID: strconv.FormatInt(r.AdministratorID, 10), Observation: r.Observation, Operator: operator}
}

func (s *API) applicationOperator(v store.Session) bool {
	return s.Config.OperatorID > 0 && v.User.ID == s.Config.OperatorID
}

func applicationNoQuery(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return false
	}
	return true
}

func (s *API) applicationOriginAllowed(spec application.Spec) bool {
	app, err := url.Parse(spec.Origin)
	if err != nil {
		return false
	}
	forgejo, err := url.Parse(s.Config.ForgejoURL)
	if err != nil || forgejo.Hostname() == "" {
		return false
	}
	return !strings.EqualFold(strings.TrimSuffix(app.Hostname(), "."), strings.TrimSuffix(forgejo.Hostname(), "."))
}

func (s *API) loadApplication(w http.ResponseWriter, r *http.Request, v store.Session) (application.Record, bool) {
	var record application.Record
	var err error
	if s.applicationOperator(v) {
		record, err = s.Store.OperatorApplication(r.Context(), r.PathValue("id"))
	} else {
		record, err = s.Store.Application(r.Context(), r.PathValue("id"), v.User.ID)
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Application not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Application records are unavailable.")
		}
		return record, false
	}
	if !s.applicationOriginAllowed(record.Application) {
		auth.JSONError(w, 409, "application_origin_conflict", "Application and Forgejo require distinct hostnames.")
		return record, false
	}
	return record, true
}

// Reads only stored references and observations. Navigation performs no native
// install, start or refresh effect; Inspect is an explicit action.
func (s *API) apiApplications(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !applicationNoQuery(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		s.apiApplicationRegister(w, r, v)
		return
	}
	var records []application.Record
	var err error
	operator := s.applicationOperator(v)
	if operator {
		records, err = s.Store.OperatorApplications(r.Context())
	} else {
		records, err = s.Store.Applications(r.Context(), v.User.ID)
	}
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Application inventory is unavailable.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	complete := true
	views := make([]applicationView, 0, len(records))
	for _, record := range records {
		views = append(views, s.applicationView(r, v, record, operator))
	}
	auth.JSONResponse(w, 200, struct {
		Applications []applicationView `json:"applications"`
		Complete     bool              `json:"complete"`
	}{views, complete})
}

func (s *API) apiApplication(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !applicationNoQuery(w, r) {
		return
	}
	record, ok := s.loadApplication(w, r, v)
	if !ok || !s.checkLifecycleSession(w, r, v) {
		return
	}
	auth.JSONResponse(w, 200, s.applicationView(r, v, record, s.applicationOperator(v)))
}

type applicationRegistration struct {
	Application     application.Spec `json:"application"`
	AdministratorID int64            `json:"administrator_id"`
}

func (s *API) apiApplicationProject(w http.ResponseWriter, r *http.Request, v store.Session) {
	record, ok := s.loadApplication(w, r, v)
	if !ok {
		return
	}
	var input struct {
		Repository string `json:"repository"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	repository, ok := auth.PositiveID(input.Repository)
	if !ok {
		auth.JSONError(w, 400, "invalid_repository", "Choose a project.")
		return
	}
	if _, err := s.visibleRepository(r, v, repository); err != nil {
		auth.JSONError(w, 404, "repository_not_found", "Project is not visible.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if err := s.Store.SetApplicationProject(r.Context(), record.Application.ID, record.AdministratorID, repository); err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not link the application to this project.")
		return
	}
	auth.JSONResponse(w, 200, s.applicationView(r, v, record, s.applicationOperator(v)))
}

func (s *API) apiApplicationRegister(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.authorizeOperator(w, r, v) {
		return
	}
	var in applicationRegistration
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	record := application.Record{Application: in.Application, AdministratorID: in.AdministratorID}
	if record.Validate() != nil || in.Application.Revision != 1 {
		auth.JSONError(w, 400, "invalid_application", "Use admitted initial application references and an explicit administrator.")
		return
	}
	if !s.applicationOriginAllowed(in.Application) {
		auth.JSONError(w, 409, "application_origin_conflict", "Application and Forgejo require distinct hostnames.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if err := s.Store.CreateApplication(r.Context(), record); err != nil {
		auth.JSONError(w, 409, "application_not_registered", "References could not be reserved; inspect existing records before retrying.")
		return
	}
	auth.JSONResponse(w, 201, appView(record, true))
}
