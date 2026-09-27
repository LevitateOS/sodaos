package control

import (
	"net/http"
)

// FactoryGitProxy connects one host-owned relay to its exact broker lease.
// The public worker socket never receives a private broker endpoint.
func (c *Controller) FactoryGitProxy(id string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !factoryGitLeaseID(id) {
			http.Error(w, "Git unavailable.", http.StatusForbidden)
			return
		}
		copy := r.Clone(r.Context())
		copy.URL.Path = "/git/" + id + r.URL.Path
		c.serveGit(w, copy)
	})
}

func factoryGitLeaseID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, ch := range id {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}
