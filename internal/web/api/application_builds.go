package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func authQueryDelivery(r *http.Request) (string, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", err
	}
	if len(query) == 0 && !r.URL.ForceQuery {
		return "", nil
	}
	values := query["delivery_id"]
	if len(query) != 1 || len(values) != 1 || !factory.ValidID(values[0]) {
		return "", errors.New("invalid delivery filter")
	}
	return values[0], nil
}

func (s *API) apiApplicationBuilds(w http.ResponseWriter, r *http.Request, v store.Session) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.authorizeOperator(w, r, v) {
		return
	}
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		query, err := authQueryDelivery(r)
		if err != nil {
			auth.JSONError(w, 400, "invalid_query", "Use only one delivery_id filter.")
			return
		}
		builds, err := s.Store.ApplicationBuilds(r.Context(), repository, query)
		if err != nil {
			auth.JSONError(w, 503, "builds_unavailable", "Application builds are unavailable.")
			return
		}
		name := v.User.Name
		if name == "" {
			name = v.User.Login
		}
		auth.JSONResponse(w, 200, struct {
			Builds        []store.ApplicationBuild `json:"builds"`
			Administrator struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"administrator"`
		}{Builds: builds, Administrator: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{strconv.FormatInt(v.User.ID, 10), name}})
		return
	}
	if !applicationNoQuery(w, r) {
		return
	}
	var input struct {
		DeliveryID  string `json:"delivery_id"`
		Environment string `json:"environment"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	if !factory.ValidID(input.DeliveryID) || (input.Environment != "company" && input.Environment != "preview") {
		auth.JSONError(w, 400, "invalid_delivery", "Choose a verified delivery from the project feed.")
		return
	}
	request, _, err := s.Store.VerifiedFactoryBuildInput(r.Context(), repository, input.DeliveryID, input.Environment)
	if err != nil {
		auth.JSONError(w, 409, "delivery_unavailable", "This delivery has no current verified application bundle. Follow its checks in Feed.")
		return
	}
	release, ok := s.acquireApplicationMutation("factory-build:" + request.Run + ":" + input.Environment)
	if !ok {
		auth.JSONError(w, 409, "build_pending", "Application packaging is in progress. Refresh its recorded result before retrying.")
		return
	}
	defer release()
	if existing, err := s.Store.ApplicationBuilds(r.Context(), repository, input.DeliveryID); err == nil {
		for _, build := range existing {
			if build.Artifact.Environment == input.Environment {
				auth.JSONResponse(w, 200, build)
				return
			}
		}
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "native_unavailable", "Application packaging host is unavailable.")
		return
	}
	ctx, cancel := operationContext(r.Context(), 4*time.Minute)
	defer cancel()
	artifact, err := s.Host.FactoryApplicationBuild(ctx, request)
	if err != nil {
		auth.JSONError(w, 502, "build_unconfirmed", "Application packaging was not confirmed. Refresh builds before retrying the same delivery.")
		return
	}
	build, err := s.Store.SaveApplicationBuild(ctx, repository, input.DeliveryID, artifact)
	if err != nil {
		auth.JSONError(w, 409, "build_changed", "The packaged bytes no longer match the current verified delivery. Refresh the project.")
		return
	}
	auth.JSONResponse(w, 201, build)
}

func (s *API) apiApplicationBuildSetup(w http.ResponseWriter, r *http.Request, v store.Session) {
	w.Header().Set("Cache-Control", "no-store")
	if !applicationNoQuery(w, r) || !s.authorizeOperator(w, r, v) {
		return
	}
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	var input struct{}
	if !auth.DecodeAPIObject(w, r, &input) || !s.checkLifecycleSession(w, r, v) {
		return
	}
	build, err := s.Store.ApplicationBuild(r.Context(), repository, r.PathValue("build"))
	if err != nil || !s.applicationOriginAllowed(application.Spec{Origin: build.Artifact.Origin}) {
		auth.JSONError(w, 409, "build_origin_conflict", "Use a separate application address from Soda Forge.")
		return
	}
	record, err := s.Store.CreateApplicationBuildSetup(r.Context(), repository, r.PathValue("build"), v.User.ID)
	if err != nil {
		auth.JSONError(w, 409, "build_setup_conflict", "This version cannot reserve a new app at its address. Refresh: an existing company app may need an update, or the verified delivery may have changed.")
		return
	}
	auth.JSONResponse(w, 201, record)
}
