package web

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"golang.org/x/crypto/ssh"
)

func (s *Server) apiRemoveDevelopmentKey(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	id, ok := positiveID(r.PathValue("key"))
	if !ok {
		jsonError(w, 400, "invalid_key", "Provide a saved key ID.")
		return
	}
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	removed, err := s.Store.RemoveKey(r.Context(), v.User.ID, id)
	if err != nil {
		jsonError(w, 503, "store_unavailable", "Could not remove saved key.")
		return
	}
	if !removed {
		jsonError(w, 404, "not_found", "Saved key not found.")
		return
	}
	jsonResponse(w, 200, map[string]any{"removed": true, "existing_project_access_changed": false})
}

func (s *Server) checkLifecycleEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return store.Project{}, false
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return store.Project{}, false
	}
	if !p.Ready {
		jsonError(w, 409, "not_provisioned", "Provisioning is incomplete; do not repair or recreate it.")
		return store.Project{}, false
	}
	return p, true
}

type lifecycleRequest struct {
	Action      string `json:"action"`
	ConfirmStop bool   `json:"confirm_stop"`
}

func decodeLifecycleRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	var in lifecycleRequest
	if !decodeAPIObject(w, r, &in) {
		return "", false
	}
	if (in.Action != "start" && in.Action != "stop") || (in.Action == "stop" && !in.ConfirmStop) || (in.Action == "start" && in.ConfirmStop) {
		jsonError(w, 400, "invalid_action", "Choose Start or explicitly confirm Stop for everyone.")
		return "", false
	}
	return in.Action, true
}

func (s *Server) authorizeLifecycleOperator(w http.ResponseWriter, r *http.Request, v store.Session, repositoryID int64) bool {
	if v.User.ID == s.Config.OperatorID {
		return true
	}
	access, err := s.visibleRepository(r, v, repositoryID)
	if err != nil {
		providerError(w, err)
		return false
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		providerError(w, err)
		return false
	}
	if !allowed {
		jsonError(w, 403, "administrator_required", "Only the current project administrator or Soda operator can start/stop it.")
		return false
	}
	return true
}

func (s *Server) checkLifecycleSession(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || s.requireCurrentSession(r.Context(), cookie.Value, v) != nil {
		jsonError(w, 401, "unauthorized", "Soda context changed. Reconnect before acting.")
		return false
	}
	return true
}

func (s *Server) acquireLifecycleStop(w http.ResponseWriter, projectID string) (func(), bool) {
	s.terminalMu.Lock()
	if s.terminalStopping == nil {
		s.terminalStopping = make(map[string]bool)
	}
	if s.terminalStopping[projectID] {
		s.terminalMu.Unlock()
		jsonError(w, 409, "stop_pending", "A Stop is already pending; inspect its outcome.")
		return nil, false
	}
	s.terminalStopping[projectID] = true
	for _, peer := range s.terminalPeers {
		if peer.project == projectID {
			peer.cancel()
		}
	}
	s.terminalMu.Unlock()
	cleanup := func() {
		s.terminalMu.Lock()
		delete(s.terminalStopping, projectID)
		s.terminalMu.Unlock()
	}
	return cleanup, true
}

func (s *Server) handleLifecycleMutation(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) (string, func(), bool) {
	action, ok := decodeLifecycleRequest(w, r)
	if !ok {
		return "", nil, false
	}
	if !s.authorizeLifecycleOperator(w, r, v, p.RepositoryID) {
		return "", nil, false
	}
	if !s.checkLifecycleSession(w, r, v) {
		return "", nil, false
	}
	if action == "stop" {
		cleanup, ok := s.acquireLifecycleStop(w, p.ID)
		if !ok {
			return "", nil, false
		}
		return action, cleanup, true
	}
	return action, nil, true
}

func (s *Server) apiLifecycle(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.checkLifecycleEnvironment(w, r)
	if !ok {
		return
	}
	action := "inspect"
	if r.Method == "POST" {
		var cleanup func()
		var ok bool
		action, cleanup, ok = s.handleLifecycleMutation(w, r, v, p)
		if !ok {
			return
		}
		if cleanup != nil {
			defer cleanup()
		}
	} else if _, ok := s.authorizeEnvironmentRead(w, r, v, p); !ok {
		return
	}
	result, err := s.Host.Lifecycle(r.Context(), host.Lifecycle{Project: p.ID, Action: action})
	if err != nil {
		jsonError(w, 502, "native_outcome_unconfirmed", "Native state was not confirmed. Refresh or ask the operator to inspect; do not repeat or repair blindly.")
		return
	}
	jsonResponse(w, 200, result)
}

var accessKeyRevision = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *Server) accessKeysEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return store.Project{}, false
	}
	return s.loadEnvironment(w, r)
}

func (s *Server) authorizeAccessKeysMember(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) (string, bool) {
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil || login == "root" || !projectLogin.MatchString(login) {
		jsonError(w, 403, "membership_required", "Only your own existing project access may be managed.")
		return "", false
	}
	if !p.Ready {
		jsonError(w, 409, "not_provisioned", "Provisioning is incomplete.")
		return "", false
	}
	if _, err = s.visibleRepository(r, v, p.RepositoryID); err != nil {
		providerError(w, err)
		return "", false
	}
	return login, true
}

func savedAccessKeyMaterial(saved []store.Key) (fingerprints, keys []string) {
	fingerprints = []string{}
	keys = []string{}
	for _, k := range saved {
		fingerprints = append(fingerprints, k.Fingerprint)
		keys = append(keys, strings.TrimSpace(k.Public))
	}
	return fingerprints, keys
}

func validAccessKeyConfirmation(revision string, savedFingerprints, keys []string, confirmEmpty bool) bool {
	return accessKeyRevision.MatchString(revision) && savedFingerprints != nil && (len(keys) == 0) == confirmEmpty
}

func (s *Server) applyAccessKeyMutation(w http.ResponseWriter, r *http.Request, v store.Session, fingerprints, keys []string, in *host.AccessKeys) bool {
	var input struct {
		Revision          string   `json:"revision"`
		SavedFingerprints []string `json:"saved_fingerprints"`
		ConfirmEmpty      bool     `json:"confirm_empty"`
	}
	if !decodeAPIObject(w, r, &input) {
		return false
	}
	if !validAccessKeyConfirmation(input.Revision, input.SavedFingerprints, keys, input.ConfirmEmpty) {
		jsonError(w, 400, "invalid_key_confirmation", "Review this project's key changes and explicitly confirm removal of the last key.")
		return false
	}
	if !slices.Equal(input.SavedFingerprints, fingerprints) {
		jsonError(w, 409, "saved_keys_changed", "Saved keys changed; review again before applying.")
		return false
	}
	in.Apply = true
	in.Revision = input.Revision
	in.Keys = keys
	return s.checkLifecycleSession(w, r, v)
}

func nativeAccessFingerprints(w http.ResponseWriter, publics []string) ([]string, bool) {
	installed := []string{}
	for _, public := range publics {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(public))
		if err != nil {
			jsonError(w, 502, "native_unavailable", "Invalid native key observation.")
			return nil, false
		}
		installed = append(installed, ssh.FingerprintSHA256(key))
	}
	return installed, true
}

func (s *Server) apiAccessKeys(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.accessKeysEnvironment(w, r)
	if !ok {
		return
	}
	login, ok := s.authorizeAccessKeysMember(w, r, v, p)
	if !ok {
		return
	}
	in := host.AccessKeys{Project: p.ID, Login: login, Identity: v.User.ID}
	saved, err := s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		jsonError(w, 503, "store_unavailable", "Could not read saved keys.")
		return
	}
	fingerprints, keys := savedAccessKeyMaterial(saved)
	if r.Method == "POST" && !s.applyAccessKeyMutation(w, r, v, fingerprints, keys, &in) {
		return
	}
	native, err := s.Host.AccessKeys(r.Context(), in)
	if err != nil {
		jsonError(w, 502, "native_outcome_unconfirmed", "Key state/update was not confirmed. Refresh and inspect before any further action; saved-key changes alone do not revoke SSH.")
		return
	}
	installed, ok := nativeAccessFingerprints(w, native.Keys)
	if !ok {
		return
	}
	jsonResponse(w, 200, struct {
		Login     string   `json:"login"`
		Revision  string   `json:"revision"`
		Installed []string `json:"installed_fingerprints"`
		Saved     []string `json:"saved_fingerprints"`
		Applied   bool     `json:"applied"`
	}{login, native.Revision, installed, fingerprints, in.Apply})
}
