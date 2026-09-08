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

func (s *Server) apiLifecycle(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if !p.Ready {
		jsonError(w, 409, "not_provisioned", "Provisioning is incomplete; do not repair or recreate it.")
		return
	}
	action := "inspect"
	if r.Method == "POST" {
		var in struct {
			Action      string `json:"action"`
			ConfirmStop bool   `json:"confirm_stop"`
		}
		if !decodeAPIObject(w, r, &in) {
			return
		}
		if (in.Action != "start" && in.Action != "stop") || (in.Action == "stop" && !in.ConfirmStop) || (in.Action == "start" && in.ConfirmStop) {
			jsonError(w, 400, "invalid_action", "Choose Start or explicitly confirm Stop for everyone.")
			return
		}
		allowed := v.User.ID == s.Config.OperatorID
		if !allowed {
			access, err := s.visibleRepository(r, v, p.RepositoryID)
			if err != nil {
				providerError(w, err)
				return
			}
			allowed, err = s.environmentAdministrator(r, access)
			if err != nil {
				providerError(w, err)
				return
			}
		}
		if !allowed {
			jsonError(w, 403, "administrator_required", "Only the current project administrator or Soda operator can start/stop it.")
			return
		}
		action = in.Action
		if action == "stop" {
			s.terminalMu.Lock()
			if s.terminalStopping == nil {
				s.terminalStopping = make(map[string]bool)
			}
			if s.terminalStopping[p.ID] {
				s.terminalMu.Unlock()
				jsonError(w, 409, "stop_pending", "A Stop is already pending; inspect its outcome.")
				return
			}
			s.terminalStopping[p.ID] = true
			defer func() { s.terminalMu.Lock(); delete(s.terminalStopping, p.ID); s.terminalMu.Unlock() }()
			for key, entry := range s.terminals {
				if key.project == p.ID {
					entry.cancel()
				}
			}
			s.terminalMu.Unlock()
		}
	} else {
		if _, ok := s.authorizeEnvironmentRead(w, r, v, p); !ok {
			return
		}
	}
	result, err := s.Host.Lifecycle(r.Context(), host.Lifecycle{Project: p.ID, Action: action})
	if err != nil {
		jsonError(w, 502, "native_outcome_unconfirmed", "Native state was not confirmed. Refresh or ask the operator to inspect; do not repeat or repair blindly.")
		return
	}
	jsonResponse(w, 200, result)
}

var accessKeyRevision = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *Server) apiAccessKeys(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil || login == "root" || !projectLogin.MatchString(login) {
		jsonError(w, 403, "membership_required", "Only your own existing project access may be managed.")
		return
	}
	if !p.Ready {
		jsonError(w, 409, "not_provisioned", "Provisioning is incomplete.")
		return
	}
	if _, err = s.visibleRepository(r, v, p.RepositoryID); err != nil {
		providerError(w, err)
		return
	}
	in := host.AccessKeys{Project: p.ID, Login: login, Identity: v.User.ID}
	saved, err := s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		jsonError(w, 503, "store_unavailable", "Could not read saved keys.")
		return
	}
	fingerprints := []string{}
	keys := []string{}
	for _, k := range saved {
		fingerprints = append(fingerprints, k.Fingerprint)
		keys = append(keys, strings.TrimSpace(k.Public))
	}
	if r.Method == "POST" {
		var input struct {
			Revision          string   `json:"revision"`
			SavedFingerprints []string `json:"saved_fingerprints"`
			ConfirmEmpty      bool     `json:"confirm_empty"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !accessKeyRevision.MatchString(input.Revision) || input.SavedFingerprints == nil || (len(keys) == 0 && !input.ConfirmEmpty) || (len(keys) != 0 && input.ConfirmEmpty) {
			jsonError(w, 400, "invalid_key_confirmation", "Review this project's key changes and explicitly confirm removal of the last key.")
			return
		}
		if !slices.Equal(input.SavedFingerprints, fingerprints) {
			jsonError(w, 409, "saved_keys_changed", "Saved keys changed; review again before applying.")
			return
		}
		in.Apply = true
		in.Revision = input.Revision
		in.Keys = keys
	}
	native, err := s.Host.AccessKeys(r.Context(), in)
	if err != nil {
		jsonError(w, 502, "native_outcome_unconfirmed", "Key state/update was not confirmed. Refresh and inspect before any further action; saved-key changes alone do not revoke SSH.")
		return
	}
	installed := []string{}
	for _, public := range native.Keys {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(public))
		if err != nil {
			jsonError(w, 502, "native_unavailable", "Invalid native key observation.")
			return
		}
		installed = append(installed, ssh.FingerprintSHA256(key))
	}
	jsonResponse(w, 200, struct {
		Login     string   `json:"login"`
		Revision  string   `json:"revision"`
		Installed []string `json:"installed_fingerprints"`
		Saved     []string `json:"saved_fingerprints"`
		Applied   bool     `json:"applied"`
	}{login, native.Revision, installed, fingerprints, in.Apply})
}
