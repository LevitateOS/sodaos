package web

import (
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/store"
)

// Keep acting-identity inspection with the Soda session/access API. Forgejo's
// collaboration, account editing and administration screens remain native.
func (s *Server) forgejoRoutes() {
	s.mux.HandleFunc("/api/forgejo/me", s.apiProvider(s.apiForgejoMe, "read:user", "GET"))
}

func (s *Server) apiForgejoMe(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	user, err := s.Forgejo.Current(r.Context(), token)
	if err != nil {
		providerError(w, err)
		return
	}
	if user.ID != v.User.ID {
		jsonError(w, 401, "provider_identity_mismatch", "Sign in again.")
		return
	}
	jsonResponse(w, 200, struct {
		ID    string `json:"id"`
		Login string `json:"login"`
		Name  string `json:"full_name"`
		Admin bool   `json:"is_admin"`
	}{strconv.FormatInt(user.ID, 10), user.Login, user.Name, user.Admin})
}
