package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/store"
)

// An operator is a configured stable ID, never a Forgejo site-admin flag. Recheck
// provider identity and the original Soda context after external authorization IO.
var errOperatorRequired = errors.New("configured Soda operator required")

func (s *API) operatorAuthorization(r *http.Request, v store.Session) error {
	if s.Config.OperatorID <= 0 || v.User.ID != s.Config.OperatorID {
		return errOperatorRequired
	}
	if _, ok := s.extensionActor(r, v); !ok {
		return auth.ErrNativeSessionChanged
	}
	if !s.extensionSessionCurrent(r.Context(), r, v) {
		return auth.ErrNativeSessionChanged
	}
	return nil
}

func (s *API) authorizeOperator(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	if err := s.operatorAuthorization(r, v); err != nil {
		if errors.Is(err, errOperatorRequired) {
			auth.JSONError(w, 403, "operator_required", "Only the configured Soda operator can manage this appliance.")
		} else {
			auth.ProviderError(w, err)
		}
		return false
	}
	return true
}
