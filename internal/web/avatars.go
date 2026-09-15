package web

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/avatar"
)

var (
	errInvalidAvatarOptions = errors.New("invalid avatar options")
	errInvalidAvatarSize    = errors.New("invalid avatar size")
)

const avatarPrefix = "/-/soda/avatars/"

// The injected renderer is the only dependency of this public image handler.
// It never reads sessions, Forgejo credentials, users, or the Soda database.
type avatarHandler struct {
	render func(string, int) (string, error)
}

func parseAvatarSize(value string) (int, error) {
	if value == "" || strings.IndexFunc(value, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return 0, errInvalidAvatarSize
	}
	size, err := strconv.Atoi(value)
	if err != nil || size < 1 || size > 1024 {
		return 0, errInvalidAvatarSize
	}
	return size, nil
}

func parseAvatarQuery(query url.Values) (int, error) {
	size := 128
	for key, values := range query {
		if len(values) != 1 {
			return 0, errInvalidAvatarOptions
		}
		switch key {
		case "s":
			parsed, err := parseAvatarSize(values[0])
			if err != nil {
				return 0, err
			}
			size = parsed
		case "d":
			// Forgejo always supplies identicon; never fetch a caller's fallback URL.
			if values[0] != "identicon" {
				return 0, errInvalidAvatarOptions
			}
		default:
			return 0, errInvalidAvatarOptions
		}
	}
	return size, nil
}

func admitAvatarRoute(r *http.Request) (string, int, string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return "", http.StatusMethodNotAllowed, "Method not allowed."
	}
	const prefix = avatarPrefix + avatar.Version + "/"
	hash, found := strings.CutPrefix(r.URL.Path, prefix)
	if !found || strings.Contains(hash, "/") || r.URL.EscapedPath() != r.URL.Path {
		return "", http.StatusNotFound, "Avatar route not found."
	}
	hash, err := avatar.NormalizeHash(hash)
	if err != nil {
		return "", http.StatusBadRequest, "Invalid avatar hash."
	}
	return hash, 0, ""
}

func admitAvatarQuery(rawQuery string) (int, int, string) {
	if len(rawQuery) > 128 {
		return 0, http.StatusBadRequest, "Invalid avatar options."
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return 0, http.StatusBadRequest, "Invalid avatar options."
	}
	size, err := parseAvatarQuery(query)
	if err == errInvalidAvatarSize {
		return 0, http.StatusBadRequest, "Invalid avatar size."
	}
	if err != nil {
		return 0, http.StatusBadRequest, "Invalid avatar options."
	}
	return size, 0, ""
}

func parseAvatarRequest(r *http.Request) (string, int, int, string) {
	hash, status, message := admitAvatarRoute(r)
	if status != 0 {
		return "", 0, status, message
	}
	size, status, message := admitAvatarQuery(r.URL.RawQuery)
	if status != 0 {
		return "", 0, status, message
	}
	return hash, size, 0, ""
}

func (h avatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	hash, size, status, message := parseAvatarRequest(r)
	if status == http.StatusMethodNotAllowed {
		w.Header().Set("Allow", "GET, HEAD")
	}
	if status != 0 {
		avatarError(w, r, status, message)
		return
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
