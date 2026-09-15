package webauth

import (
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/store"
)

// Keep acting-identity inspection with the Soda session/access API. Forgejo's
// collaboration, account editing and administration screens remain native.
func (s *Service) forgejoRoutes() {
	s.mux.HandleFunc("/api/forgejo/me", s.Provider(s.apiForgejoMe, "read:user", "GET"))
}

func (s *Service) apiForgejoMe(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	user, err := s.Forgejo.Current(r.Context(), token)
	if err != nil {
		ProviderError(w, err)
		return
	}
	if user.ID != v.User.ID {
		JSONError(w, 401, "provider_identity_mismatch", "Sign in again.")
		return
	}
	JSONResponse(w, 200, struct {
		ID    string `json:"id"`
		Login string `json:"login"`
		Name  string `json:"full_name"`
		Admin bool   `json:"is_admin"`
	}{strconv.FormatInt(user.ID, 10), user.Login, user.Name, user.Admin})
}
