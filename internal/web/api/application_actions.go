package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type applicationAction struct {
	Action            string            `json:"action"`
	Revision          int64             `json:"revision"`
	ExpectedContainer string            `json:"expected_container"`
	ConfirmStop       bool              `json:"confirm_stop"`
	Replacement       *application.Spec `json:"replacement"`
	ReplacementBuild  string            `json:"replacement_build,omitempty"`
}

func (s *API) acquireApplicationMutation(id string) (func(), bool) {
	s.applicationMu.Lock()
	defer s.applicationMu.Unlock()
	if s.applicationPending == nil {
		s.applicationPending = make(map[string]bool)
	}
	if s.applicationPending[id] {
		return nil, false
	}
	s.applicationPending[id] = true
	return func() { s.applicationMu.Lock(); delete(s.applicationPending, id); s.applicationMu.Unlock() }, true
}

func admitApplicationAction(record application.Record, in applicationAction) (application.Request, error) {
	request := application.Request{Application: record.Application, Action: in.Action, ExpectedContainer: in.ExpectedContainer}
	if in.Revision != record.Application.Revision {
		return request, errors.New("application action differs from current admission")
	}
	stopping := in.Action == "stop" || in.Action == "replace"
	if in.ConfirmStop != stopping || (in.Action == "replace") != (in.Replacement != nil) {
		return request, errors.New("application action differs from current admission")
	}
	if err := admitApplicationGeneration(record, in); err != nil {
		return request, err
	}
	return admitApplicationReplacement(record, in, request)
}

func admitApplicationGeneration(record application.Record, in applicationAction) error {
	if in.Action == "install" {
		if record.Application.Revision != 1 || (record.Observation != nil && record.Observation.Container != "") || in.ExpectedContainer != "" {
			return errors.New("application action differs from current admission")
		}
	} else {
		if record.Observation == nil {
			// An operator can recover an uncertain replacement using a freshly
			// observed native container ID. The host still checks that exact ID
			// and the app/data identity before stopping the writer.
			if in.Action == "replace" && in.ExpectedContainer != "" {
				return nil
			}
			if in.Action != "inspect" || in.ExpectedContainer != "" {
				return errors.New("application action differs from current admission")
			}
		} else if in.ExpectedContainer != record.Observation.Container {
			return errors.New("application action differs from current admission")
		}
	}
	return nil
}

func admitApplicationReplacement(record application.Record, in applicationAction, request application.Request) (application.Request, error) {
	if in.Replacement != nil {
		if err := record.Application.Replacement(*in.Replacement); err != nil {
			return request, err
		}
		// Native image admission checks storage compatibility before stopping the app.
		request.Application = *in.Replacement
	}
	return request, request.Validate()
}

func (s *API) apiApplicationAction(w http.ResponseWriter, r *http.Request, v store.Session) {
	record, request, ok := s.admitApplicationOperation(w, r, v)
	if !ok {
		return
	}
	release, ok := s.acquireApplicationMutation(record.Application.ID)
	if !ok {
		if request.Action == "inspect" {
			if current, visible := s.loadApplication(w, r, v); visible {
				auth.JSONResponse(w, 200, s.applicationView(r, v, current, s.applicationOperator(v)))
			}
		} else {
			auth.JSONError(w, 409, "application_pending", "An application operation is already in progress. Refresh its status before trying again.")
		}
		return
	}
	defer release()
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "native_unavailable", "Application host is unavailable.")
		return
	}
	s.performApplicationOperation(w, r, v, record, request)
}

func (s *API) admitApplicationOperation(w http.ResponseWriter, r *http.Request, v store.Session) (application.Record, application.Request, bool) {
	var empty application.Record
	var noRequest application.Request
	if !applicationNoQuery(w, r) {
		return empty, noRequest, false
	}
	record, ok := s.loadApplication(w, r, v)
	if !ok {
		return empty, noRequest, false
	}
	var in applicationAction
	if !auth.DecodeAPIObject(w, r, &in) {
		return empty, noRequest, false
	}
	if !s.authorizeApplicationAction(w, r, v, in.Action) {
		return empty, noRequest, false
	}
	if err := s.resolveApplicationBuildReplacement(r, record, &in); err != nil {
		auth.JSONError(w, 409, "build_replacement_conflict", "Choose a current verified build prepared for this application's project, name and address. Refresh before retrying.")
		return empty, noRequest, false
	}
	if !s.noApplicationRestore(w, r, record.Application.ID) {
		return empty, noRequest, false
	}
	request, err := admitApplicationAction(record, in)
	if err != nil {
		auth.JSONError(w, 409, "application_action_conflict", "Use the current revision/container, and explicitly confirm Stop or image replacement.")
		return empty, noRequest, false
	}
	if !s.applicationOriginAllowed(request.Application) {
		auth.JSONError(w, 409, "application_origin_conflict", "Application and Forgejo require distinct hostnames.")
		return empty, noRequest, false
	}
	return record, request, true
}

func (s *API) authorizeApplicationAction(w http.ResponseWriter, r *http.Request, v store.Session, action string) bool {
	if action == "install" || action == "replace" {
		return s.authorizeOperator(w, r, v)
	}
	return true
}

func (s *API) performApplicationOperation(w http.ResponseWriter, r *http.Request, v store.Session, record application.Record, request application.Request) {
	ctx, cancel := operationContext(r.Context(), 4*time.Minute)
	defer cancel()
	// Clear observed readiness durably before invoking native work. A lost
	// reply or process crash leaves unknown, including interrupted inspection.
	if err := s.Store.ObserveApplication(ctx, record.Application.ID, record.AdministratorID, record.Application.Revision, record.Observation, nil); err != nil {
		auth.JSONError(w, 409, "application_state_changed", "Application references or observation changed; inspect again.")
		return
	}
	result, err := s.Host.Application(ctx, request)
	if err != nil {
		auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native outcome is unknown. Inspect before acting again; desired references do not prove readiness.")
		return
	}
	if request.Action == "replace" {
		if err := s.Store.ReplaceApplication(ctx, record.AdministratorID, record.Application.Revision, request.Application); err != nil {
			auth.JSONError(w, 409, "application_state_changed", "The native app changed, but its references could not be saved. Refresh before acting again.")
			return
		}
	}
	// Readback identifies immutable image bytes. Ordinary lifecycle observations
	// of that same exact image retain it; replacement must establish new proof.
	if result.BundleFileSHA256 == "" && record.Observation != nil && record.Application.Image == result.Image && record.Observation.Image == result.Image && result.Container != "" {
		result.BundleFileSHA256 = record.Observation.BundleFileSHA256
	}
	if result.BundleFileSHA256 == "" && result.Container != "" && result.Image == request.Application.Image {
		result.BundleFileSHA256 = s.Store.ApplicationBuildImageProof(ctx, result.Image)
	}
	if err := s.Store.ObserveApplication(ctx, record.Application.ID, record.AdministratorID, request.Application.Revision, nil, &result); err != nil {
		auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native outcome could not be recorded. Inspect before acting again.")
		return
	}
	record.Application = request.Application
	record.Observation = &result
	auth.JSONResponse(w, 200, s.applicationView(r, v, record, s.applicationOperator(v)))
}
