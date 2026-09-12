package web

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func (s *Server) tailnetRoutes() {
	s.mux.HandleFunc("GET /settings/tailnet", s.tailnetPage)
	s.mux.HandleFunc("/api/settings/tailnet", s.apiProtected(s.apiTailnetSettings, http.MethodGet))
	s.mux.HandleFunc("/api/settings/tailnet/host", s.apiProtected(s.apiTailnetHost, http.MethodPost))
	s.mux.HandleFunc("/api/settings/tailnet/enrollment", s.apiProtected(s.apiTailnetEnrollment, http.MethodPost))
	s.mux.HandleFunc("/api/repositories/{repositoryID}/tailnet-options", s.apiProtected(s.apiTailnetOptions, http.MethodGet))
	s.mux.HandleFunc("/api/environments/{id}/tailnet", s.apiProtected(s.apiProjectTailnet, http.MethodGet, http.MethodPost))
}
func tailnetQuery(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_query", "Tailnet operations do not accept query parameters.")
		return false
	}
	return true
}
func (s *Server) tailnetSession(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || r.Context().Err() != nil || s.requireCurrentSession(r.Context(), cookie.Value, v) != nil {
		jsonError(w, 401, "unauthorized", "Soda context changed. Already-dispatched work may have completed; reconnect and observe before retrying.")
		return false
	}
	return true
}
func tailnetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tailnet.ErrInvalid):
		jsonError(w, 400, "invalid_tailnet", "Invalid Tailnet fields or confirmation.")
	case errors.Is(err, tailnet.ErrConflict):
		jsonError(w, 409, "tailnet_changed", "Policy, host or original project identity changed. Review before acting.")
	case errors.Is(err, tailnet.ErrUnsupported):
		jsonError(w, 422, "tailnet_unsupported", "This Tailnet operation or native version is unsupported. No requested operation was dispatched.")
	case errors.Is(err, tailnet.ErrUnavailable):
		jsonError(w, 503, "tailnet_unavailable", "Tailnet management or its observation is unavailable. No empty or disconnected state was inferred.")
	default:
		jsonError(w, 502, "tailnet_unconfirmed", "Operation unconfirmed. Policy, credentials or native state may have changed. Observe before retrying; no automatic rollback or replay occurred.")
	}
}
func (s *Server) apiTailnetSettings(w http.ResponseWriter, r *http.Request, v store.Session) {
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
	jsonResponse(w, 200, result)
}
func (s *Server) apiTailnetHost(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.authorizeOperator(w, r, v) || !tailnetQuery(w, r) {
		return
	}
	var in tailnet.HostRequest
	if !decodeAPIObject(w, r, &in) {
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
	jsonResponse(w, 200, result)
}
func (s *Server) apiTailnetEnrollment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.authorizeOperator(w, r, v) || !tailnetQuery(w, r) {
		return
	}
	var in tailnet.EnrollmentRequest
	defer func() { in.ClientSecret = "" }()
	if !decodeAPIObject(w, r, &in) {
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
	jsonResponse(w, 200, result)
}
func (s *Server) apiTailnetOptions(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !tailnetQuery(w, r) {
		return
	}
	id, ok := positiveID(r.PathValue("repositoryID"))
	if !ok {
		jsonError(w, 400, "invalid_repository", "Provide one canonical repository ID.")
		return
	}
	access, err := s.visibleRepository(r, v, id)
	if err != nil {
		providerError(w, err)
		return
	}
	if access.repository.Owner.ID != v.User.ID {
		jsonError(w, 403, "owner_required", "Only the current human repository owner can select creation options.")
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
	jsonResponse(w, 200, result)
}
func (s *Server) authorizeProjectTailnet(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) bool {
	if v.User.ID == s.Config.OperatorID {
		return s.authorizeOperator(w, r, v)
	}
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		providerError(w, err)
		return false
	}
	if r.Method == http.MethodGet {
		_, err = s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
		if err == nil {
			return true
		}
		if !errors.Is(err, sql.ErrNoRows) {
			jsonError(w, 503, "store_unavailable", "Membership could not be checked.")
			return false
		}
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		providerError(w, err)
		return false
	}
	if !allowed {
		jsonError(w, 403, "tailnet_access_denied", "Private network reads require current membership or administration; changes require current project administration.")
		return false
	}
	return true
}
func (s *Server) apiProjectTailnet(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !tailnetQuery(w, r) {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if !s.authorizeProjectTailnet(w, r, v, p) {
		return
	}
	if !p.Ready {
		jsonError(w, 409, "not_provisioned", "Project provisioning is incomplete.")
		return
	}
	in := tailnet.ProjectRequest{Project: p.ID, Action: "inspect"}
	if r.Method == http.MethodPost {
		// Project ID is selected only by the canonical route/store association.
		var body struct {
			Action    string `json:"action"`
			Revision  string `json:"revision"`
			Binding   string `json:"binding,omitempty"`
			ConfirmID string `json:"confirm_id"`
		}
		if !decodeAPIObject(w, r, &body) {
			return
		}
		in.Action, in.Revision, in.Binding, in.ConfirmID = body.Action, body.Revision, body.Binding, body.ConfirmID
		if in.Action == "inspect" || in.Validate() != nil {
			tailnetError(w, tailnet.ErrInvalid)
			return
		}
	}
	if !s.tailnetSession(w, r, v) {
		return
	}
	result, err := s.Host.TailnetProject(r.Context(), in)
	if !s.tailnetSession(w, r, v) {
		return
	}
	if err != nil {
		tailnetError(w, err)
		return
	}
	jsonResponse(w, 200, result)
}
