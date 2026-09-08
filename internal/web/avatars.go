package web

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/avatar"
)

const avatarPrefix = "/-/soda/avatars/"

// The injected renderer is the only dependency of this public image handler.
// It never reads sessions, Forgejo credentials, users, or the Soda database.
type avatarHandler struct {
	render func(string, int) (string, error)
}

func (h avatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		avatarError(w, r, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}
	const prefix = avatarPrefix + avatar.Version + "/"
	hash, found := strings.CutPrefix(r.URL.Path, prefix)
	if !found || strings.Contains(hash, "/") || r.URL.EscapedPath() != r.URL.Path {
		avatarError(w, r, http.StatusNotFound, "Avatar route not found.")
		return
	}
	hash, err := avatar.NormalizeHash(hash)
	if err != nil {
		avatarError(w, r, http.StatusBadRequest, "Invalid avatar hash.")
		return
	}
	if len(r.URL.RawQuery) > 128 {
		avatarError(w, r, http.StatusBadRequest, "Invalid avatar options.")
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		avatarError(w, r, http.StatusBadRequest, "Invalid avatar options.")
		return
	}
	size := 128
	for key, values := range query {
		if len(values) != 1 {
			avatarError(w, r, http.StatusBadRequest, "Invalid avatar options.")
			return
		}
		switch key {
		case "s":
			value := values[0]
			if value == "" || strings.IndexFunc(value, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
				avatarError(w, r, http.StatusBadRequest, "Invalid avatar size.")
				return
			}
			size, err = strconv.Atoi(value)
			if err != nil || size < 1 || size > 1024 {
				avatarError(w, r, http.StatusBadRequest, "Invalid avatar size.")
				return
			}
		case "d":
			// Forgejo always supplies identicon; never fetch a caller's fallback URL.
			if values[0] != "identicon" {
				avatarError(w, r, http.StatusBadRequest, "Invalid avatar options.")
				return
			}
		default:
			avatarError(w, r, http.StatusBadRequest, "Invalid avatar options.")
			return
		}
	}
	svg, err := h.render(hash, size)
	if err != nil {
		avatarError(w, r, http.StatusServiceUnavailable, "Avatar unavailable.")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(svg))))
	// ServeContent implements GET/HEAD and conditional requests, including weak
	// If-None-Match validators. There is no per-user or unbounded in-memory cache.
	http.ServeContent(w, r, "avatar.svg", time.Time{}, strings.NewReader(svg))
}

func avatarError(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		fmt.Fprintln(w, message)
	}
}
