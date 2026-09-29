package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func TestExtensionResponseIsBounded(t *testing.T) {
	response := &http.Response{
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(strings.Repeat("x", auth.APIBodyLimit+1))),
		ContentLength: int64(auth.APIBodyLimit + 1),
	}
	if err := boundExtensionResponse(response); err != errExtensionResponseTooLarge {
		t.Fatalf("expected bounded response rejection, got %v", err)
	}
}

func TestExtensionRejectsUnknownRouteAndUnverifiedActor(t *testing.T) {
	handler, closeTransport := Extension("/unavailable-soda-socket")
	defer closeTransport()
	for _, test := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/session", http.StatusForbidden},
		{"PATCH", "/me/preferences", http.StatusForbidden},
		{"POST", "/terminal", http.StatusNotFound},
		{"GET", "/session?actor=42", http.StatusNotFound},
		{"GET", "//session", http.StatusNotFound},
	} {
		r := httptest.NewRequest(test.method, test.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Errorf("%s %s: got %d, want %d", test.method, test.path, w.Code, test.status)
		}
	}
}

func TestExtensionTransportDoesNotForwardBrowserCredentials(t *testing.T) {
	in := httptest.NewRequest("PATCH", "/me/preferences", nil)
	in.Header.Set("Cookie", "native=private")
	in.Header.Set("Authorization", "Bearer private")
	in.Header.Set("X-Forwarded-Host", "untrusted")
	in.Header.Set("X-Soda-Expected-User-ID", "99")
	in.Header.Set("Origin", "https://forgejo.test")
	in.Header.Set(extensions.AdmissionHeader, "private-admission")
	out := in.Clone(in.Context())
	rewriteExtensionRequest(&httputil.ProxyRequest{In: in, Out: out})
	if out.URL.Path != "/api/me/preferences" {
		t.Fatalf("private service path = %q", out.URL.Path)
	}
	for _, name := range []string{"Cookie", "Authorization", "X-Forwarded-Host", "X-Soda-Expected-User-ID"} {
		if out.Header.Get(name) != "" {
			t.Errorf("forwarded %s", name)
		}
	}
	if out.Header.Get("Origin") != "https://forgejo.test" || out.Header.Get(extensions.AdmissionHeader) != "private-admission" {
		t.Fatal("private bridge lost Origin or admission")
	}
}
