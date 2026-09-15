package api

import (
	"errors"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func (s *API) tailnetRoutes() {
	s.mux.HandleFunc("GET /settings/tailnet", s.tailnetPage)
	s.mux.HandleFunc("/api/settings/tailnet", s.Auth.Protected(s.apiTailnetSettings, http.MethodGet))
	s.mux.HandleFunc("/api/settings/tailnet/host", s.Auth.Protected(s.apiTailnetHost, http.MethodPost))
	s.mux.HandleFunc("/api/settings/tailnet/enrollment", s.Auth.Protected(s.apiTailnetEnrollment, http.MethodPost))
	s.mux.HandleFunc("/api/repositories/{repositoryID}/tailnet-options", s.Auth.Protected(s.apiTailnetOptions, http.MethodGet))
	s.mux.HandleFunc("/api/environments/{id}/tailnet", s.Auth.Protected(s.apiProjectTailnet, http.MethodGet, http.MethodPost))
}

func tailnetQuery(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_query", "Tailnet operations do not accept query parameters.")
		return false
	}
	return true
}

func (s *API) tailnetSession(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	cookie, err := auth.RequestCookie(r, auth.SessionCookie)
	if err != nil || r.Context().Err() != nil || s.Auth.RequireCurrentSession(r.Context(), cookie.Value, v) != nil {
		auth.JSONError(w, 401, "unauthorized", "Soda context changed. Already-dispatched work may have completed; reconnect and observe before retrying.")
		return false
	}
	return true
}

func tailnetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tailnet.ErrInvalid):
		auth.JSONError(w, 400, "invalid_tailnet", "Invalid Tailnet fields or confirmation.")
	case errors.Is(err, tailnet.ErrConflict):
		auth.JSONError(w, 409, "tailnet_changed", "Policy, host or original project identity changed. Review before acting.")
	case errors.Is(err, tailnet.ErrUnsupported):
		auth.JSONError(w, 422, "tailnet_unsupported", "This Tailnet operation or required runtime capability is unsupported. No requested operation was dispatched.")
	case errors.Is(err, tailnet.ErrUnavailable):
		auth.JSONError(w, 503, "tailnet_unavailable", "Tailnet management or its observation is unavailable. No empty or disconnected state was inferred.")
	default:
		auth.JSONError(w, 502, "tailnet_unconfirmed", "Operation unconfirmed. Policy, credentials or native state may have changed. Observe before retrying; no automatic rollback or replay occurred.")
	}
}

func (s *API) apiTailnetSettings(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.authorizeOperator(w, r, v) || !tailnetQuery(w, r) || !s.tailnetSession(w, r, v) {
		return
	}
	result, err := s.Host.TailnetSettings(r.Context())
	if !s.tailnetSession(w, r, v) {
		return
	}
	if err != nil {
		tailnetError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}

func (s *API) apiTailnetHost(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.authorizeOperator(w, r, v) || !tailnetQuery(w, r) {
		return
	}
	var in tailnet.HostRequest
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if in.Validate() != nil {
		tailnetError(w, tailnet.ErrInvalid)
		return
	}
	if !s.tailnetSession(w, r, v) {
		return
	}
	result, err := s.Host.TailnetHost(r.Context(), in)
	if !s.tailnetSession(w, r, v) {
		return
	}
	if err != nil {
		tailnetError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}

func (s *API) apiTailnetEnrollment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.authorizeOperator(w, r, v) || !tailnetQuery(w, r) {
		return
	}
	var in tailnet.EnrollmentRequest
	defer func() { in.ClientSecret = "" }()
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if in.Validate() != nil {
		tailnetError(w, tailnet.ErrInvalid)
		return
	}
	if !s.tailnetSession(w, r, v) {
		return
	}
	result, err := s.Host.TailnetEnrollment(r.Context(), in)
	in.ClientSecret = ""
	if !s.tailnetSession(w, r, v) {
		return
	}
	if err != nil {
		tailnetError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}

func (s *API) apiTailnetOptions(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !tailnetQuery(w, r) {
		return
	}
	id, ok := auth.PositiveID(r.PathValue("repositoryID"))
	if !ok {
		auth.JSONError(w, 400, "invalid_repository", "Provide one canonical repository ID.")
		return
	}
	access, err := s.visibleRepository(r, v, id)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	if access.repository.Owner.ID != v.User.ID {
		auth.JSONError(w, 403, "owner_required", "Only the current human repository owner can select creation options.")
		return
	}
	if !s.tailnetSession(w, r, v) {
		return
	}
	result, err := s.Host.TailnetOptions(r.Context())
	if !s.tailnetSession(w, r, v) {
		return
	}
	if err != nil {
		tailnetError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}

func (s *API) authorizeProjectTailnet(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) bool {
	if v.User.ID == s.Config.OperatorID {
		return s.authorizeOperator(w, r, v)
	}
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if r.Method == http.MethodGet {
		_, err = s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
		if err == nil {
			return true
		}
		if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Membership could not be checked.")
			return false
		}
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if !allowed {
		auth.JSONError(w, 403, "tailnet_access_denied", "Private network reads require current membership or administration; changes require current project administration.")
		return false
	}
	return true
}

func parseProjectTailnetMutation(w http.ResponseWriter, r *http.Request, in *tailnet.ProjectRequest) bool {
	if r.Method != http.MethodPost {
		return true
	}
	// Project ID is selected only by the canonical route/store association.
	var body struct {
		Action    string `json:"action"`
		Revision  string `json:"revision"`
		Binding   string `json:"binding,omitempty"`
		ConfirmID string `json:"confirm_id"`
	}
	if !auth.DecodeAPIObject(w, r, &body) {
		return false
	}
	in.Action, in.Revision, in.Binding, in.ConfirmID = body.Action, body.Revision, body.Binding, body.ConfirmID
	if in.Action == "inspect" || in.Validate() != nil {
		tailnetError(w, tailnet.ErrInvalid)
		return false
	}
	return true
}

func (s *API) admitProjectTailnetDispatch(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) bool {
	return s.authorizeProjectTailnet(w, r, v, p) && s.tailnetSession(w, r, v)
}

func (s *API) confirmProjectTailnetDispatch(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) bool {
	return s.tailnetSession(w, r, v) && s.authorizeProjectTailnet(w, r, v, p)
}

func (s *API) admitProjectTailnetRequest(w http.ResponseWriter, r *http.Request, v store.Session) (store.Project, bool) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return p, false
	}
	if !s.authorizeProjectTailnet(w, r, v, p) {
		return p, false
	}
	if !p.Ready {
		auth.JSONError(w, 409, "not_provisioned", "Project provisioning is incomplete.")
		return p, false
	}
	return p, true
}

func (s *API) apiProjectTailnet(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !tailnetQuery(w, r) {
		return
	}
	p, ok := s.admitProjectTailnetRequest(w, r, v)
	if !ok {
		return
	}
	in := tailnet.ProjectRequest{Project: p.ID, Action: "inspect"}
	if !parseProjectTailnetMutation(w, r, &in) {
		return
	}
	if !s.admitProjectTailnetDispatch(w, r, v, p) {
		return
	}
	result, err := s.Host.TailnetProject(r.Context(), in)
	if !s.confirmProjectTailnetDispatch(w, r, v, p) {
		return
	}
	if err != nil {
		tailnetError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}
