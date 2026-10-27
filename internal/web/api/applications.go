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
	"golang.org/x/net/publicsuffix"
)

type applicationView struct {
	Application     application.Spec                  `json:"application"`
	AdministratorID string                            `json:"administrator_id"`
	Observation     *application.Observation          `json:"observation"`
	Operator        bool                              `json:"operator"`
	Project         *applicationProjectView           `json:"project,omitempty"`
	Restore         *application.RestoreOperation     `json:"restore,omitempty"`
	Environment     string                            `json:"environment"`
	CanEmbed        bool                              `json:"can_embed"`
	Delivery        *store.ApplicationFactoryDelivery `json:"delivery,omitempty"`
}

type applicationProjectView struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Path       string `json:"path"`
}

// applicationLaunchView is the ordinary-user launch projection (FJ26). It
// carries the admitted application references with running/ready booleans so
// an intended employee can open and authenticate to the app. It deliberately
// omits the administrator grant, the native container generation and every
// lifecycle/administration control. Repository access is not app-data access:
// the container ID that admits native effects is never exposed here.
type applicationLaunchView struct {
	Application applicationLaunchReference `json:"application"`
	Running     bool                       `json:"running"`
	Ready       bool                       `json:"ready"`
	State       string                     `json:"state"`
	Project     *applicationProjectView    `json:"project,omitempty"`
	LaunchOnly  bool                       `json:"launch_only"`
	CanEmbed    bool                       `json:"can_embed"`
	Environment string                     `json:"environment"`
	DeliveryID  string                     `json:"delivery_id,omitempty"`
}

type applicationLaunchReference struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
	Name     string `json:"name"`
	Origin   string `json:"origin"`
}

func appLaunchView(record application.Record, project *applicationProjectView, canEmbed bool, environment string) applicationLaunchView {
	var running, ready bool
	state := "unknown"
	if observed := record.Observation; observed != nil {
		running, ready = observed.Running, observed.Ready
		switch {
		case observed.Container == "":
			state = "not-installed"
		case !running:
			state = "stopped"
		case ready:
			state = "ready"
		default:
			state = "running"
		}
	}
	a := record.Application
	return applicationLaunchView{Application: applicationLaunchReference{ID: a.ID, Revision: a.Revision, Name: a.Name, Origin: a.Origin}, Running: running, Ready: ready, State: state, Project: project, LaunchOnly: true, CanEmbed: canEmbed, Environment: environment}
}

func (s *API) applicationView(r *http.Request, v store.Session, record application.Record, operator bool) applicationView {
	view := appView(record, operator)
	view.CanEmbed = s.applicationEmbeddable(record.Application)
	view.Environment = s.applicationEnvironment(r, record.Application.ID)
	if delivery, err := s.Store.VerifiedApplicationDelivery(r.Context(), record.Application.ID); err == nil {
		view.Delivery = delivery
	}
	if pending, err := s.Store.PendingApplicationRestore(r.Context(), record.Application.ID); err == nil {
		view.Restore = pending
	}
	if repository, err := s.Store.ApplicationProject(r.Context(), record.Application.ID); err == nil && repository > 0 {
		if access, err := s.visibleRepository(r, v, repository); err == nil {
			view.Project = &applicationProjectView{Repository: strconv.FormatInt(repository, 10), Name: access.repository.FullName, Path: "/" + access.repository.FullName}
		}
	}
	return view
}

func appView(r application.Record, operator bool) applicationView {
	return applicationView{Environment: "company", Application: r.Application, AdministratorID: strconv.FormatInt(r.AdministratorID, 10), Observation: r.Observation, Operator: operator}
}

// launchableApplications returns the launch-only projection for applications
// the reader does not administer but may open through a visible linked
// project (FJ26). Operators and administrators already see their full views;
// this never duplicates them and never performs a native effect. Expired
// sessions and account changes return to the correct app: launch rows carry
// the admitted origin/identity with running/ready booleans and no stale
// personal data.
func (s *API) launchableApplications(r *http.Request, v store.Session, administered []application.Record) ([]applicationLaunchView, bool) {
	if s.applicationOperator(v) {
		return []applicationLaunchView{}, true
	}
	seen := make(map[string]bool, len(administered))
	for _, record := range administered {
		seen[record.Application.ID] = true
	}
	links, err := s.Store.ApplicationProjectLinks(r.Context())
	if err != nil {
		return []applicationLaunchView{}, false
	}
	launches := []applicationLaunchView{}
	complete := true
	for _, link := range links {
		if seen[link.ApplicationID] {
			continue
		}
		seen[link.ApplicationID] = true
		access, err := s.visibleRepository(r, v, link.Repository)
		if err != nil {
			continue
		}
		record, err := s.Store.LaunchApplicationRecord(r.Context(), link.ApplicationID)
		if err != nil {
			complete = false
			continue
		}
		if !s.applicationOriginAllowed(record.Application) {
			continue
		}
		project := &applicationProjectView{Repository: strconv.FormatInt(link.Repository, 10), Name: access.repository.FullName, Path: "/" + access.repository.FullName}
		launch := appLaunchView(record, project, s.applicationEmbeddable(record.Application), s.applicationEnvironment(r, record.Application.ID))
		// Repository access admits this minimal launch projection. Only the
		// current image's verified native byte join supplies its result identity.
		if delivery, err := s.Store.VerifiedApplicationDelivery(r.Context(), record.Application.ID); err == nil && delivery != nil && record.Observation != nil && delivery.BundleFileSHA256 == record.Observation.BundleFileSHA256 && delivery.Environment == launch.Environment {
			launch.DeliveryID = delivery.DeliveryID
		}
		launches = append(launches, launch)
	}
	return launches, complete
}

func (s *API) applicationEnvironment(r *http.Request, id string) string {
	value, err := s.Store.ApplicationEnvironment(r.Context(), id)
	if err != nil {
		return "unknown"
	}
	// A completed setup owns the environment association. A native image
	// observation may disprove it, including a manually admitted preview image.
	// Keep contradictory or failed readback projections out of either data lane.
	delivery, err := s.Store.VerifiedApplicationDelivery(r.Context(), id)
	if err != nil || (delivery != nil && delivery.Environment != value) {
		return "unknown"
	}
	return value
}

// Canlang's host-only Secure SameSite=Lax cookies work in a same-site HTTPS
// frame. Keep that identity boundary instead of proxying app data or relaxing
// cookies for arbitrary cross-site origins. First-owner setup remains top-level.
func (s *API) applicationEmbeddable(spec application.Spec) bool {
	app, err := url.Parse(spec.Origin)
	if err != nil || app.Scheme != "https" || !s.applicationOriginAllowed(spec) {
		return false
	}
	forge, err := url.Parse(s.Config.ForgejoURL)
	if err != nil || forge.Scheme != "https" {
		return false
	}
	appSite, appErr := publicsuffix.EffectiveTLDPlusOne(strings.ToLower(app.Hostname()))
	forgeSite, forgeErr := publicsuffix.EffectiveTLDPlusOne(strings.ToLower(forge.Hostname()))
	return appErr == nil && forgeErr == nil && appSite == forgeSite
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
	launches, launchComplete := s.launchableApplications(r, v, records)
	auth.JSONResponse(w, 200, struct {
		Applications []applicationView       `json:"applications"`
		Launchable   []applicationLaunchView `json:"launchable"`
		Complete     bool                    `json:"complete"`
		CanSetup     bool                    `json:"can_setup"`
	}{views, launches, complete && launchComplete, operator})
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
