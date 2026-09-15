package webapp

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/webauth"
	"golang.org/x/crypto/ssh"
)

var accessKeyRevision = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *API) accessKeysEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		webauth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return store.Project{}, false
	}
	return s.loadEnvironment(w, r)
}

func (s *API) authorizeAccessKeysMember(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) (string, bool) {
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil || login == "root" || !projectLogin.MatchString(login) {
		webauth.JSONError(w, 403, "membership_required", "Only your own existing project access may be managed.")
		return "", false
	}
	if !p.Ready {
		webauth.JSONError(w, 409, "not_provisioned", "Provisioning is incomplete.")
		return "", false
	}
	if _, err = s.visibleRepository(r, v, p.RepositoryID); err != nil {
		webauth.ProviderError(w, err)
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

func (s *API) applyAccessKeyMutation(w http.ResponseWriter, r *http.Request, v store.Session, fingerprints, keys []string, in *host.AccessKeys) bool {
	var input struct {
		Revision          string   `json:"revision"`
		SavedFingerprints []string `json:"saved_fingerprints"`
		ConfirmEmpty      bool     `json:"confirm_empty"`
	}
	if !webauth.DecodeAPIObject(w, r, &input) {
		return false
	}
	if !validAccessKeyConfirmation(input.Revision, input.SavedFingerprints, keys, input.ConfirmEmpty) {
		webauth.JSONError(w, 400, "invalid_key_confirmation", "Review this project's key changes and explicitly confirm removal of the last key.")
		return false
	}
	if !slices.Equal(input.SavedFingerprints, fingerprints) {
		webauth.JSONError(w, 409, "saved_keys_changed", "Saved keys changed; review again before applying.")
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
			webauth.JSONError(w, 502, "native_unavailable", "Invalid native key observation.")
			return nil, false
		}
		installed = append(installed, ssh.FingerprintSHA256(key))
	}
	return installed, true
}

func (s *API) apiAccessKeys(w http.ResponseWriter, r *http.Request, v store.Session) {
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
		webauth.JSONError(w, 503, "store_unavailable", "Could not read saved keys.")
		return
	}
	fingerprints, keys := savedAccessKeyMaterial(saved)
	if r.Method == "POST" && !s.applyAccessKeyMutation(w, r, v, fingerprints, keys, &in) {
		return
	}
	native, err := s.Host.AccessKeys(r.Context(), in)
	if err != nil {
		webauth.JSONError(w, 502, "native_outcome_unconfirmed", "Key state/update was not confirmed. Refresh and inspect before any further action; saved-key changes alone do not revoke SSH.")
		return
	}
	installed, ok := nativeAccessFingerprints(w, native.Keys)
	if !ok {
		return
	}
	webauth.JSONResponse(w, 200, struct {
		Login     string   `json:"login"`
		Revision  string   `json:"revision"`
		Installed []string `json:"installed_fingerprints"`
		Saved     []string `json:"saved_fingerprints"`
		Applied   bool     `json:"applied"`
	}{login, native.Revision, installed, fingerprints, in.Apply})
}
