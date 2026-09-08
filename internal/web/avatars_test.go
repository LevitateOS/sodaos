package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
)

const avatarURL = "/-/soda/avatars/v1/0123456789abcdef0123456789abcdef"

func TestAvatarPublicRoute(t *testing.T) {
	// No database, provider URL, host connection or credentials are configured.
	s := New(config.Config{}, nil)
	var etag, body string
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		r := httptest.NewRequest(method, avatarURL+"?s=64&d=identicon", nil)
		r.Header.Set("Cookie", "soda_session=invalid; i_like_gitea=invalid")
		r.Header.Set("Authorization", "Bearer invalid")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", method, w.Code, w.Body.String())
		}
		if w.Header().Get("Content-Type") != "image/svg+xml" || w.Header().Get("Set-Cookie") != "" {
			t.Fatal(w.Header())
		}
		if w.Header().Get("Cache-Control") != "public, max-age=86400" || w.Header().Get("ETag") == "" {
			t.Fatal(w.Header())
		}
		if w.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "default-src 'none'") {
			t.Fatal("missing SVG security headers")
		}
		if method == http.MethodGet {
			etag, body = w.Header().Get("ETag"), w.Body.String()
			if !strings.Contains(body, `width="64"`) {
				t.Fatal("requested size missing")
			}
		} else if w.Body.Len() != 0 || w.Header().Get("ETag") != etag {
			t.Fatal("HEAD differs from GET")
		}
	}
	for _, validator := range []string{etag, "W/" + etag, `"another", ` + etag, "*"} {
		r := httptest.NewRequest(http.MethodGet, avatarURL+"?d=identicon&s=64", nil)
		r.Header.Set("If-None-Match", validator)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusNotModified || w.Body.Len() != 0 || w.Header().Get("ETag") != etag {
			t.Fatalf("conditional %q: %d", validator, w.Code)
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, avatarURL, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `width="128"`) || w.Header().Get("ETag") == etag {
		t.Fatal("default size or size-specific ETag failed")
	}
}

func TestAvatarRejectsInvalidInputBeforeRendering(t *testing.T) {
	cases := []struct {
		method, url string
		status      int
	}{
		{"POST", avatarURL, 405}, {"DELETE", avatarURL, 405}, {"OPTIONS", avatarURL, 405},
		{"GET", "/-/soda/avatars", 404}, {"GET", "/-/soda/avatars/", 404},
		{"GET", "/-/soda/avatars/v2/" + strings.Repeat("a", 32), 404},
		{"GET", avatarURL + "/extra", 404},
		{"GET", "/-/soda/avatars/v1/alice@example.test", 400},
		{"GET", "/-/soda/avatars/v1/" + strings.Repeat("z", 32), 400},
		{"GET", avatarURL + "?s=0", 400}, {"GET", avatarURL + "?s=1025", 400},
		{"GET", avatarURL + "?s=-1", 400}, {"GET", avatarURL + "?s=1.5", 400},
		{"GET", avatarURL + "?s=", 400}, {"GET", avatarURL + "?s=%2B32", 400},
		{"GET", avatarURL + "?s=32&s=32", 400}, {"GET", avatarURL + "?s=" + strings.Repeat("9", 100), 400},
		{"GET", avatarURL + "?seed=other", 400}, {"GET", avatarURL + "?accentColor=fff", 400},
		{"GET", avatarURL + "?d=https://example.test/a.svg", 400},
		{"GET", avatarURL + "?d=identicon&d=identicon", 400},
		{"GET", avatarURL + "?d=identicon;s=32", 400}, {"GET", avatarURL + "?s=%zz", 400},
		{"GET", avatarURL + "?" + strings.Repeat("x", 129), 400},
		{"HEAD", avatarURL + "?s=bad", 400},
	}
	h := avatarHandler{render: func(string, int) (string, error) { t.Error("invalid request reached renderer"); return "", nil }}
	for _, c := range cases {
		t.Run(c.method+c.url, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(c.method, c.url, nil))
			if w.Code != c.status || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("got %d %v", w.Code, w.Header())
			}
			if c.status == 405 && w.Header().Get("Allow") != "GET, HEAD" {
				t.Fatal("missing Allow")
			}
			if c.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD error has a body")
			}
		})
	}
}

func TestAvatarPathBoundaries(t *testing.T) {
	s := New(config.Config{}, nil)
	for _, p := range []string{
		"/-/soda/avatars/v1//" + strings.Repeat("a", 32),
		"/-/soda/avatars/v1/../v1/" + strings.Repeat("a", 32),
		"/-/soda/avatars/v1/%61" + strings.Repeat("a", 31),
		"/-/soda/avatars/v1/" + strings.Repeat("a", 32) + "%2fextra",
		"/-/soda/avatars-extra/v1/" + strings.Repeat("a", 32),
		"/-/soda/avatarsx/" + strings.Repeat("a", 32),
	} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" {
			t.Errorf("%s: %d %v", p, w.Code, w.Header())
		}
	}
	for _, size := range []string{"1", "1024"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", avatarURL+"?s="+size, nil))
		if w.Code != 200 {
			t.Fatalf("valid size %s: %d", size, w.Code)
		}
	}
}

func TestAvatarFailureIsBoundedAndUncached(t *testing.T) {
	h := avatarHandler{render: func(string, int) (string, error) { return "", errors.New("private implementation detail") }}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", avatarURL, nil))
	if w.Code != 503 || w.Header().Get("Cache-Control") != "no-store" || w.Body.String() != "Avatar unavailable.\n" || w.Header().Get("ETag") != "" {
		t.Fatalf("%d %v %s", w.Code, w.Header(), w.Body.String())
	}
}

func TestAvatarNormalizesProviderHash(t *testing.T) {
	h := avatarHandler{render: func(hash string, size int) (string, error) {
		if hash != "0123456789abcdef0123456789abcdef" {
			t.Errorf("unnormalized hash %q", hash)
		}
		return avatar.Render(hash, size)
	}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/-/soda/avatars/v1/0123456789ABCDEF0123456789ABCDEF", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
