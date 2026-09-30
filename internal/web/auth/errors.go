package auth

import (
	"errors"
	"net/http"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/forgejo"
)

var (
	// ErrRepositoryConsent means the acting grant lacks user/repository scopes.
	ErrRepositoryConsent = errors.New("repository and user consent required")
	// ErrProviderIdentity means Forgejo's acting user differs from the Soda session.
	ErrProviderIdentity = errors.New("provider identity differs from Soda session")
	// ErrNativeSessionChanged means the Forgejo session no longer matches the
	// authority that admitted this operation.
	ErrNativeSessionChanged = errors.New("native Forgejo session changed")
)

// ProviderError maps native Forgejo authority failures to bounded responses.
// It never includes provider response bodies, tokens, or callback data.
func ProviderError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNativeSessionChanged) {
		JSONError(w, http.StatusConflict, "extension_session_changed", "Native Forgejo session changed; reload before retrying.")
		return
	}
	if errors.Is(err, extensions.ErrRepositoryNotVisible) {
		JSONError(w, http.StatusNotFound, "not_found", "Forgejo object not found or not visible.")
		return
	}
	var native *forgejo.HTTPError
	if errors.As(err, &native) {
		switch native.Status {
		case http.StatusUnauthorized:
			JSONError(w, http.StatusUnauthorized, "native_authority_unavailable", "Current Forgejo authority is unavailable.")
		case http.StatusForbidden:
			JSONError(w, http.StatusForbidden, "native_access_denied", "Forgejo denied this operation.")
		case http.StatusNotFound:
			JSONError(w, http.StatusNotFound, "not_found", "Forgejo object not found or not visible.")
		case http.StatusConflict:
			JSONError(w, http.StatusConflict, "native_conflict", "Forgejo reported a conflict; reload before editing.")
		case http.StatusTooManyRequests:
			JSONError(w, http.StatusTooManyRequests, "native_rate_limit", "Forgejo rate limit reached; try later.")
		default:
			JSONError(w, http.StatusServiceUnavailable, "native_authority_unavailable", "Forgejo could not complete this request. Inspect native state before retrying.")
		}
		return
	}
	JSONError(w, http.StatusServiceUnavailable, "native_authority_unavailable", "Forgejo could not complete this request. Inspect native state before retrying.")
}
