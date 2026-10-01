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

func TestExtensionProductRouteAllowlist(t *testing.T) {
	for _, test := range []struct {
		method, path string
		allowed      bool
	}{
		{http.MethodGet, "/repositories", true},
		{http.MethodGet, "/repositories/12/profiles", true},
		{http.MethodGet, "/repositories/12/factory", true},
		{http.MethodPut, "/repositories/12/factory/policy", true},
		{http.MethodPut, "/repositories/12/factory/operator-grant", true},
		{http.MethodPut, "/repositories/12/factory/environment-grant", true},
		{http.MethodPut, "/repositories/12/factory/sponsorships/conn-1", true},
		{http.MethodPost, "/repositories/12/factory/policy", false},
		{http.MethodGet, "/repositories/12/factory/policy", false},
		{http.MethodPut, "/factory/capacity", true},
		{http.MethodGet, "/factory/capacity", false},
		{http.MethodPost, "/repositories/12/factory/actions", true},
		{http.MethodGet, "/repositories/12/factory/actions", false},
		{http.MethodPost, "/factory/runs/0123456789abcdef0123456789abcdef/actions", true},
		{http.MethodGet, "/factory/runs/0123456789abcdef0123456789abcdef/actions", false},
		{http.MethodGet, "/factory/commands/0123456789abcdef0123456789abcdef", true},
		{http.MethodPost, "/factory/commands/0123456789abcdef0123456789abcdef", false},
		{http.MethodGet, "/factory/runs/0123456789abcdef0123456789abcdef", true},
		{http.MethodPost, "/factory/runs/0123456789abcdef0123456789abcdef", false},
		{http.MethodGet, "/factory/runs/0123456789abcdef0123456789abcdef/output", true},
		{http.MethodPost, "/factory/runs/0123456789abcdef0123456789abcdef/output", false},
		{http.MethodGet, "/repositories/12/factory/issues/3", true},
		{http.MethodPost, "/repositories/12/factory/issues/3", false},
		{http.MethodPost, "/repositories/12/factory/issues/3/acceptances", true},
		{http.MethodGet, "/repositories/12/factory/issues/3/acceptances", false},
		{http.MethodPost, "/repositories/12/factory/issues/3/withdrawal", true},
		{http.MethodGet, "/repositories/12/factory/issues/3/withdrawal", false},
		{http.MethodPost, "/environments/p0123456789abcdef01234567/preparation/acceptances", true},
		{http.MethodPost, "/environments/p0123456789abcdef01234567/preparation/actions", true},
		{http.MethodGet, "/environments/p0123456789abcdef01234567/preparation/acceptances", false},
		{http.MethodGet, "/spaces", true},
		{http.MethodPost, "/environments", true},
		{http.MethodPost, "/environments/p0123456789abcdef01234567/join", true},
		{http.MethodGet, "/identity/connections/abc-123/grants", true},
		{http.MethodPost, "/settings/runners/soda-runner-x/stop", false},
		{http.MethodPost, "/settings/tailnet/enrollment", true},
		{http.MethodGet, "/settings/runners", false},
		{http.MethodDelete, "/settings/runners", false},
		{http.MethodGet, "/settings/runners/secret/remove", false},
		{http.MethodPost, "/environments/12/connection", false},
		{http.MethodPost, "/admin/users", false},
		{http.MethodGet, "/environments/12/identity/launch", false},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			r := httptest.NewRequest(test.method, test.path, nil)
			if got := extensionRoute(r); got != test.allowed {
				t.Fatalf("extensionRoute(%s %s) = %t, want %t", test.method, test.path, got, test.allowed)
			}
		})
	}
}

func TestExtensionProductContributionScopes(t *testing.T) {
	for _, test := range []struct {
		path         string
		contribution extensions.Contribution
		allowed      bool
	}{
		{"/api/spaces", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, true},
		{"/api/environments/1/join", extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel"}, true},
		{"/api/settings/tailnet", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "admin"}, true},
		{"/api/settings/tailnet", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "global"}, false},
		{"/api/environments/1/tailnet", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, true},
	} {
		t.Run(test.path+"/"+test.contribution.ID, func(t *testing.T) {
			if got := extensionProductContribution(test.path, test.contribution); got != test.allowed {
				t.Fatalf("extensionProductContribution(%q, %+v) = %t, want %t", test.path, test.contribution, got, test.allowed)
			}
		})
	}
}

func TestNativeTerminalContributionScopes(t *testing.T) {
	for _, test := range []struct {
		name         string
		contribution extensions.Contribution
		allowed      bool
	}{
		{"Spaces page", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, true},
		{"workspace panel", extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel"}, true},
		{"admin runners page", extensions.Contribution{Kind: "page", ID: "runners", Scope: "admin"}, false},
		{"other global page", extensions.Contribution{Kind: "page", ID: "runners", Scope: "global"}, false},
		{"spaces admin scope", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "admin"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := nativeTerminalContribution(extensions.Authority{Contribution: test.contribution})
			if got != test.allowed {
				t.Fatalf("nativeTerminalContribution(%+v) = %t, want %t", test.contribution, got, test.allowed)
			}
		})
	}
}

func TestExtensionTransportDoesNotForwardBrowserCredentials(t *testing.T) {
	in := httptest.NewRequest("PATCH", "/me/preferences", nil)
	in.Header.Set("Cookie", "native=private")
	in.Header.Set("Authorization", "Bearer private")
	in.Header.Set("X-Forwarded-Host", "untrusted")
	in.Header.Set("Origin", "https://forgejo.test")
	in.Header.Set(extensions.AdmissionHeader, "private-admission")
	out := in.Clone(in.Context())
	rewriteExtensionRequest(&httputil.ProxyRequest{In: in, Out: out})
	if out.URL.Path != "/api/me/preferences" {
		t.Fatalf("private service path = %q", out.URL.Path)
	}
	for _, name := range []string{"Cookie", "Authorization", "X-Forwarded-Host"} {
		if out.Header.Get(name) != "" {
			t.Errorf("forwarded %s", name)
		}
	}
	if out.Header.Get("Origin") != "https://forgejo.test" || out.Header.Get(extensions.AdmissionHeader) != "private-admission" {
		t.Fatal("private bridge lost Origin or admission")
	}
}

func TestExtensionTerminalUpgradeForwardsOnlyProtocolAndAuthority(t *testing.T) {
	in := httptest.NewRequest(http.MethodGet, "/environments/p0123456789abcdef01234567/terminal", nil)
	in.Header.Set("Origin", "https://forgejo.test")
	in.Header.Set("Connection", "Upgrade")
	in.Header.Set("Upgrade", "websocket")
	in.Header.Set("Sec-WebSocket-Key", "synthetic-key")
	in.Header.Set("Sec-WebSocket-Version", "13")
	in.Header.Set("Cookie", "private=native")
	in.Header.Set("Authorization", "Bearer private")
	in.Header.Set(extensions.ContextHeader, "verified-context")
	in.Header.Set(extensions.AdmissionHeader, "verified-admission")
	out := in.Clone(in.Context())
	rewriteExtensionRequest(&httputil.ProxyRequest{In: in, Out: out})
	if out.URL.Path != "/api/environments/p0123456789abcdef01234567/terminal" || out.Header.Get("Upgrade") != "websocket" || out.Header.Get("Sec-WebSocket-Key") != "synthetic-key" {
		t.Fatal("terminal upgrade lost its private route or WebSocket protocol headers")
	}
	for _, name := range []string{"Cookie", "Authorization", "X-Forwarded-Host", "Sec-WebSocket-Protocol"} {
		if out.Header.Get(name) != "" {
			t.Errorf("terminal forwarded browser %s", name)
		}
	}
	if out.Header.Get(extensions.ContextHeader) != "verified-context" || out.Header.Get(extensions.AdmissionHeader) != "verified-admission" {
		t.Fatal("terminal upgrade lost native admission")
	}
}
