package auth

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

func (s *Service) discardGrantOr(ctx context.Context, session string, fallback error) error {
	if deleteErr := s.discardGrant(ctx, session); deleteErr != nil {
		return deleteErr
	}
	return fallback
}

func (s *Service) refreshUserGrant(ctx context.Context, session string, uid int64, grant store.Grant) (store.Grant, error) {
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		return store.Grant{}, forgejo.ErrUnavailable
	}
	renewed, err := s.Forgejo.RefreshGrant(ctx, s.Config.OAuthClientID, secret, grant.Refresh)
	if err != nil {
		// A timeout may already have consumed the rotating refresh credential. Do
		// not replay it or a pending provider write. Reauthenticate explicitly.
		return store.Grant{}, s.discardGrantOr(ctx, session, store.ErrGrantUnavailable)
	}
	grant.Access, grant.Refresh = renewed.Access, renewed.Refresh
	grant.Expires = renewed.ExpiresAt
	if err = s.Store.ReplaceGrant(ctx, session, uid, grant); err != nil {
		return store.Grant{}, s.discardGrantOr(ctx, session, err)
	}
	return grant, nil
}

func (s *Service) UserGrant(r *http.Request, v store.Session) (store.Grant, error) {
	cookie, err := RequestCookie(r, SessionCookie)
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
	return s.refreshUserGrant(r.Context(), cookie.Value, v.User.ID, grant)
}

func (s *Service) discardGrant(ctx context.Context, session string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.Store.DeleteGrant(cleanup, session)
}

func (s *Service) Provider(next func(http.ResponseWriter, *http.Request, store.Session, string), scope string, methods ...string) http.HandlerFunc {
	return s.Protected(func(w http.ResponseWriter, r *http.Request, v store.Session) {
		grant, err := s.UserGrant(r, v)
		if err != nil {
			ProviderError(w, err)
			return
		}
		required := scope
		if r.Method == http.MethodGet {
			required = strings.Replace(scope, "write:", "read:", 1)
		}
		if !forgejo.HasScope(grant.Scopes, required) {
			JSONError(w, http.StatusForbidden, "consent_required", "Forgejo consent does not include this operation. Revoke the old Soda grant in native Applications settings, then sign in with the required consent.")
			return
		}
		next(w, r, v, grant.Access)
	}, methods...)
}

func grantReauthentication(err error) bool {
	return errors.Is(err, store.ErrGrantUnavailable) || errors.Is(err, store.ErrGrantKey)
}

func providerOperationUnavailable(status int) bool {
	return status == 405 || status == 423
}

func providerValidationStatus(status int) bool {
	return status == 400 || status == 422
}

func providerHTTPStatus(native *forgejo.HTTPError) (int, string, string, bool) {
	switch native.Status {
	case 401:
		return 401, "provider_unauthenticated", "Forgejo authorization expired; sign in again.", true
	case 403:
		return 403, "provider_forbidden", "Forgejo denied this operation.", true
	case 404:
		return 404, "provider_not_found", "Forgejo object not found or not visible.", true
	case 409:
		return 409, "provider_conflict", "Forgejo reported a conflict; reload before editing.", true
	case 429:
		return 429, "provider_rate_limit", "Forgejo rate limit reached; try later.", true
	}
	if providerOperationUnavailable(native.Status) {
		return 409, "provider_operation_unavailable", "Forgejo does not permit this operation in its current state or settings.", true
	}
	if providerValidationStatus(native.Status) {
		return 422, "provider_validation", "Forgejo rejected the supplied fields.", true
	}
	return 0, "", "", false
}

func knownProviderError(err error) (int, string, string, bool) {
	switch {
	case errors.Is(err, ErrRepositoryConsent):
		return 403, "consent_required", "Forgejo user and repository consent is required.", true
	case errors.Is(err, ErrProviderIdentity):
		return 401, "provider_identity_mismatch", "Sign in again.", true
	case errors.Is(err, forgejo.ErrResponseTooLarge):
		return 413, "provider_response_too_large", "This object exceeds the dashboard's supported size. Use native Git or Forgejo for larger files.", true
	case grantReauthentication(err):
		return 401, "reauthentication_required", "Sign in again to authorize Forgejo access.", true
	}
	var native *forgejo.HTTPError
	if !errors.As(err, &native) {
		return 0, "", "", false
	}
	return providerHTTPStatus(native)
}

func ProviderError(w http.ResponseWriter, err error) {
	if code, kind, msg, ok := knownProviderError(err); ok {
		JSONError(w, code, kind, msg)
		return
	}
	JSONError(w, 503, "provider_unavailable", "Forgejo could not complete this request. A write may have completed; inspect native state before retrying.")
}
