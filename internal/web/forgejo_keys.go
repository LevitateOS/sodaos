package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func parseForgejoKeysPage(rawQuery string) (int64, bool) {
	query, err := url.ParseQuery(rawQuery)
	page := int64(1)
	valid := true
	if query.Has("page") {
		page, valid = positiveID(query.Get("page"))
	}
	if err != nil || !valid || len(query) > 1 || len(query["page"]) > 1 || (len(query) == 1 && !query.Has("page")) || page > 8 {
		return 0, false
	}
	return page, true
}

func collectProfileKeys(keys []forgejo.OwnPublicKey) ([]profileKeyView, error) {
	items := make([]profileKeyView, 0, len(keys))
	for _, key := range keys {
		public, fingerprint, err := normalizeDevelopmentKey(key.Key)
		if err != nil {
			return nil, err
		}
		items = append(items, profileKeyView{developmentKeyView{strconv.FormatInt(key.ID, 10), public, fingerprint}, key.Title})
	}
	return items, nil
}

type profileKeyView struct {
	developmentKeyView
	Title string `json:"title"`
}

func (s *Server) confirmKeysSession(ctx context.Context, r *http.Request, v store.Session) bool {
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		return false
	}
	current, err := s.Store.Session(ctx, cookie.Value)
	return err == nil && current.ContextID == v.ContextID && current.CSRF == v.CSRF
}

func (s *Server) apiForgejoKeys(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	page, ok := parseForgejoKeysPage(r.URL.RawQuery)
	if !ok {
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
	items, err := collectProfileKeys(keys)
	if err != nil {
		jsonError(w, 503, "invalid_profile_key", "Forgejo returned an unsupported development public key. No keys were imported.")
		return
	}
	// A pending read cannot disclose keys after logout/rotation won during IO.
	if !s.confirmKeysSession(ctx, r, v) {
		providerError(w, store.ErrGrantUnavailable)
		return
	}
	result := struct {
		Items []profileKeyView `json:"items"`
		Page  int64            `json:"page"`
		More  bool             `json:"more"`
	}{items, page, len(items) == 10}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > apiBodyLimit {
		jsonError(w, 413, "keys_too_large", "Public keys exceed the supported response size. Use native profile settings to select a key.")
		return
	}
	jsonResponse(w, 200, result)
}
