package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/factory"
)

// Frontend composes the Fountain adapter from independently configured product
// endpoints. Factory browser requests go directly to Factory; Project, account
// and host operations go to Forge. Each bridge verifies native authority and
// preserves the same bounded HTTP/WebSocket contract. Neither endpoint can
// execute the other's requests when its peer is unavailable.
func Frontend(forgeSocket, factorySocket string) (http.Handler, func()) {
	forge, closeForge := Extension(forgeSocket)
	factoryAPI, closeFactory := Extension(factorySocket)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if factory.OwnsAPIPath("/api" + r.URL.Path) {
				factoryAPI.ServeHTTP(w, r)
				return
			}
			forge.ServeHTTP(w, r)
		}), func() {
			closeForge()
			closeFactory()
		}
}
