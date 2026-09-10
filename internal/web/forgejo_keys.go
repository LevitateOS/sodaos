package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) apiForgejoKeys(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	query, err := url.ParseQuery(r.URL.RawQuery)
	page := int64(1)
	valid := true
	if query.Has("page") {
		page, valid = positiveID(query.Get("page"))
	}
	if err != nil || !valid || len(query) > 1 || len(query["page"]) > 1 || (len(query) == 1 && !query.Has("page")) || page > 8 {
		jsonError(w, 400, "invalid_page", "Select page 1 through 8 of your own public keys.")
		return
	}
	actor, err := s.Forgejo.Current(ctx, token)
	if err != nil {
		providerError(w, err)
		return
	}
	if actor.ID != v.User.ID {
		providerError(w, errProviderIdentity)
		return
	}
	keys, err := s.Forgejo.OwnPublicKeys(ctx, token, v.User.ID, int(page))
	if err != nil {
		providerError(w, err)
		return
	}
	type profileKey struct {
		developmentKeyView
		Title string `json:"title"`
	}
	items := make([]profileKey, 0, len(keys))
	for _, key := range keys {
		public, fingerprint, err := normalizeDevelopmentKey(key.Key)
		if err != nil {
			jsonError(w, 503, "invalid_profile_key", "Forgejo returned an unsupported development public key. No keys were imported.")
			return
		}
		items = append(items, profileKey{developmentKeyView{strconv.FormatInt(key.ID, 10), public, fingerprint}, key.Title})
	}
	// A pending read cannot disclose keys after logout/rotation won during IO.
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		providerError(w, store.ErrGrantUnavailable)
		return
	}
	current, err := s.Store.Session(ctx, cookie.Value)
	if err != nil || current.ContextID != v.ContextID || current.CSRF != v.CSRF {
		providerError(w, store.ErrGrantUnavailable)
		return
	}
	result := struct {
		Items []profileKey `json:"items"`
		Page  int64        `json:"page"`
		More  bool         `json:"more"`
	}{items, page, len(items) == 10}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > apiBodyLimit {
		jsonError(w, 413, "keys_too_large", "Public keys exceed the supported response size. Use native profile settings to select a key.")
		return
	}
	jsonResponse(w, 200, result)
}
