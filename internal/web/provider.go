package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// Forgejo grants belong to a user/application, even when Soda binds credential
// copies to individual sessions. Serialize local refreshes for that native user.
type providerLock struct {
	gate  chan struct{}
	users int
}
type providerLocks struct {
	mu      sync.Mutex
	entries map[int64]*providerLock
}

func (p *providerLocks) lock(ctx context.Context, uid int64) (func(), error) {
	p.mu.Lock()
	if p.entries == nil {
		p.entries = make(map[int64]*providerLock)
	}
	entry := p.entries[uid]
	if entry == nil {
		entry = &providerLock{gate: make(chan struct{}, 1)}
		p.entries[uid] = entry
	}
	entry.users++
	p.mu.Unlock()
	releaseRef := func() {
		p.mu.Lock()
		entry.users--
		if entry.users == 0 {
			delete(p.entries, uid)
		}
		p.mu.Unlock()
	}
	select {
	case entry.gate <- struct{}{}:
		return func() { <-entry.gate; releaseRef() }, nil
	case <-ctx.Done():
		releaseRef()
		return nil, ctx.Err()
	}
}

func (s *Server) userGrant(r *http.Request, v store.Session) (store.Grant, error) {
	cookie, err := r.Cookie("soda_session")
	if err != nil {
		return store.Grant{}, store.ErrGrantUnavailable
	}
	unlock, err := s.providerLocks.lock(r.Context(), v.User.ID)
	if err != nil {
		return store.Grant{}, err
	}
	defer unlock()
	grant, err := s.Store.Grant(r.Context(), cookie.Value, v.User.ID)
	if err != nil {
		return store.Grant{}, err
	}
	if grant.Expires > time.Now().Add(30*time.Second).Unix() {
		return grant, nil
	}
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		return store.Grant{}, forgejo.ErrUnavailable
	}
	renewed, err := s.Forgejo.RefreshGrant(r.Context(), s.Config.OAuthClientID, secret, grant.Refresh)
	if err != nil {
		// A timeout may already have consumed the rotating refresh credential. Do
		// not replay it or a pending provider write. Reauthenticate explicitly.
		if deleteErr := s.discardGrant(r.Context(), cookie.Value); deleteErr != nil {
			return store.Grant{}, deleteErr
		}
		return store.Grant{}, store.ErrGrantUnavailable
	}
	grant.Access, grant.Refresh = renewed.Access, renewed.Refresh
	grant.Expires = renewed.ExpiresAt
	if err = s.Store.ReplaceGrant(r.Context(), cookie.Value, v.User.ID, grant); err != nil {
		if deleteErr := s.discardGrant(r.Context(), cookie.Value); deleteErr != nil {
			return store.Grant{}, deleteErr
		}
		return store.Grant{}, err
	}
	return grant, nil
}

func (s *Server) discardGrant(ctx context.Context, session string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.Store.DeleteGrant(cleanup, session)
}

func (s *Server) apiProvider(next func(http.ResponseWriter, *http.Request, store.Session, string), scope string, methods ...string) http.HandlerFunc {
	return s.apiProtected(func(w http.ResponseWriter, r *http.Request, v store.Session) {
		grant, err := s.userGrant(r, v)
		if err != nil {
			providerError(w, err)
			return
		}
		required := scope
		if r.Method == http.MethodGet {
			required = strings.Replace(scope, "write:", "read:", 1)
		}
		if !forgejo.HasScope(grant.Scopes, required) {
			jsonError(w, http.StatusForbidden, "consent_required", "Forgejo consent does not include this operation. Revoke the old Soda grant in native Applications settings, then sign in with the required consent.")
			return
		}
		next(w, r, v, grant.Access)
	}, methods...)
}
func providerError(w http.ResponseWriter, err error) {
	if errors.Is(err, forgejo.ErrResponseTooLarge) {
		jsonError(w, 413, "provider_response_too_large", "This object exceeds the dashboard's supported size. Use native Git or Forgejo for larger files.")
		return
	}
	if errors.Is(err, store.ErrGrantUnavailable) || errors.Is(err, store.ErrGrantKey) {
		jsonError(w, 401, "reauthentication_required", "Sign in again to authorize Forgejo access.")
		return
	}
	var native *forgejo.HTTPError
	if errors.As(err, &native) {
		switch native.Status {
		case 401:
			jsonError(w, 401, "provider_unauthenticated", "Forgejo authorization expired; sign in again.")
			return
		case 403:
			jsonError(w, 403, "provider_forbidden", "Forgejo denied this operation.")
			return
		case 404:
			jsonError(w, 404, "provider_not_found", "Forgejo object not found or not visible.")
			return
		case 405, 423:
			jsonError(w, 409, "provider_operation_unavailable", "Forgejo does not permit this operation in its current state or settings.")
			return
		case 409:
			jsonError(w, 409, "provider_conflict", "Forgejo reported a conflict; reload before editing.")
			return
		case 400, 422:
			jsonError(w, 422, "provider_validation", "Forgejo rejected the supplied fields.")
			return
		case 429:
			jsonError(w, 429, "provider_rate_limit", "Forgejo rate limit reached; try later.")
			return
		}
	}
	jsonError(w, 503, "provider_unavailable", "Forgejo could not complete this request. A write may have completed; inspect native state before retrying.")
}
