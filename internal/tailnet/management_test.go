package tailnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type managementRoundTrip func(*http.Request) (*http.Response, error)

func (f managementRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func managementResponse(status int, s string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(s)), Header: make(http.Header)}
}
func nativeStatusFixture() string {
	return `{"Version":"1.102.4","CurrentTailnet":{"Name":"soda.example.test","MagicDNSEnabled":true},"BackendState":"Running","HaveNodeKey":true,"Self":{"ID":"self","DNSName":"host.example.ts.net.","TailscaleIPs":["100.64.0.1"]},"Peer":{"peer":{"ID":"peer","DNSName":"exit.example.ts.net.","TailscaleIPs":["100.64.0.2"],"Online":true,"ExitNodeOption":true}},"AuthURL":"https://login.tailscale.com/a/synthetic","Health":["sensitive native diagnostic"],"PrivateKey":"must-not-project"}`
}
func nativePrefsFixture() string {
	return `{"WantRunning":true,"ExitNodeID":"","ExitNodeIP":"","ExitNodeAllowLANAccess":false,"AdvertiseRoutes":["10.8.0.0/16"],"Persist":{"PrivateNodeKey":"must-not-project"}}`
}
func managementFixture(t *testing.T) *Management {
	t.Helper()
	m := NewManagement()
	p, _ := policyFixture(t)
	m.policy = *p
	m.command = func(context.Context, string, ...string) ([]byte, error) {
		t.Error("unexpected native mutation")
		return nil, errors.New("no command")
	}
	m.local = &http.Client{Transport: managementRoundTrip(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/localapi/v0/status":
			return managementResponse(200, nativeStatusFixture()), nil
		case "/localapi/v0/prefs":
			return managementResponse(200, nativePrefsFixture()), nil
		}
		t.Error("unexpected LocalAPI request")
		return managementResponse(500, ""), nil
	})}
	return m
}
func TestTailnetHostProjectionAndPassiveReads(t *testing.T) {
	m := managementFixture(t)
	for range 2 {
		result, e := m.Settings(t.Context())
		if e != nil || result.Validate() != nil || result.HostUnavailable {
			t.Fatal(result, e)
		}
		b, _ := json.Marshal(result)
		for _, secret := range []string{"sensitive native diagnostic", "PrivateNodeKey", "must-not-project", "login.tailscale.com"} {
			if strings.Contains(string(b), secret) {
				t.Fatal("private native state projected")
			}
		}
		if result.Host.HealthIssues != 1 || len(result.Host.Peers) != 1 || result.Host.Preferences.AdvertiseExitNode {
			t.Fatal(result.Host)
		}
	}
}
func TestTailnetHostUnavailableIsNotDisconnected(t *testing.T) {
	for _, kind := range []string{"version", "prefs", "oversize", "null", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			m := managementFixture(t)
			original := m.local.Transport
			m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/localapi/v0/status" {
					switch kind {
					case "version":
						return managementResponse(200, strings.Replace(nativeStatusFixture(), "1.102.4", "1.1.0", 1)), nil
					case "oversize":
						return managementResponse(200, nativeStatusFixture()+strings.Repeat(" ", 65536)), nil
					case "null":
						return managementResponse(200, "null"), nil
					case "duplicate":
						return managementResponse(200, strings.Replace(nativeStatusFixture(), `"HaveNodeKey":true`, `"HaveNodeKey":true,"HaveNodeKey":false`, 1)), nil
					}
				}
				if kind == "prefs" && r.URL.Path == "/localapi/v0/prefs" {
					return managementResponse(200, `{}`), nil
				}
				return original.RoundTrip(r)
			})
			v, e := m.Settings(t.Context())
			if e != nil || !v.HostUnavailable || v.Host != nil {
				t.Fatal(v, e)
			}
		})
	}
}
func TestTailnetHostReauthPreservesPrefsAndValidatesURL(t *testing.T) {
	m := managementFixture(t)
	patches, logins := 0, 0
	m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Path == "/localapi/v0/status":
			return managementResponse(200, strings.Replace(nativeStatusFixture(), `"Running"`, `"NeedsLogin"`, 1)), nil
		case r.URL.Path == "/localapi/v0/prefs" && r.Method == "GET":
			return managementResponse(200, nativePrefsFixture()), nil
		case r.URL.Path == "/localapi/v0/prefs" && r.Method == "PATCH":
			patches++
			b, _ := io.ReadAll(r.Body)
			var fields map[string]bool
			if json.Unmarshal(b, &fields) != nil || len(fields) != 2 || !fields["WantRunning"] || !fields["WantRunningSet"] {
				t.Error("unrelated prefs rewritten")
			}
			return managementResponse(200, `{}`), nil
		case r.URL.Path == "/localapi/v0/login-interactive":
			logins++
			return managementResponse(204, ""), nil
		}
		return managementResponse(500, ""), nil
	})
	before, _, e := m.observe(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	result, e := m.HostAction(t.Context(), HostRequest{Action: "signin", Revision: before.Revision})
	if e != nil || result.Validate() != nil || result.Outcome != "pending" || patches != 1 || logins != 1 {
		t.Fatal(result, e, patches, logins)
	}
	for _, bad := range []string{"https://evil.test/a/secret", "http://login.tailscale.com/a/secret", "https://login.tailscale.com.evil.test/a/secret", "https://login.tailscale.com/a/secret?token=secret", "https://user@login.tailscale.com/a/secret"} {
		if authenticationURL(bad) != "" {
			t.Fatal("unsafe auth URL")
		}
	}
}
func TestTailnetHostActionsConfirmAndSeparateFailedReadback(t *testing.T) {
	for _, action := range []string{"advertise-exit-node", "refresh-forgejo"} {
		t.Run(action, func(t *testing.T) {
			m := managementFixture(t)
			before, _, e := m.observe(t.Context())
			if e != nil {
				t.Fatal(e)
			}
			flag := true
			r := HostRequest{Action: action, Revision: before.Revision, Confirm: action}
			if action == "advertise-exit-node" {
				r.Advertise = &flag
			}
			commands := 0
			m.command = func(ctx context.Context, path string, args ...string) ([]byte, error) {
				if path == DefaultCLI && strings.Join(args, " ") == "version --json" {
					return []byte(`{"short":"1.102.4"}`), nil
				}
				commands++
				if action == "advertise-exit-node" && (path != DefaultCLI || strings.Join(args, " ") != "--socket="+hostSocket+" set --advertise-exit-node=true") {
					t.Error("unexpected command")
				}
				if action == "refresh-forgejo" && (path != "/usr/local/libexec/soda/soda-forgejo-tailnet" || len(args) != 0) {
					t.Error("unexpected refresh")
				}
				m.local.Transport = managementRoundTrip(func(*http.Request) (*http.Response, error) { return nil, errors.New("synthetic secret error") })
				return nil, nil
			}
			bad := r
			bad.Confirm = "other"
			if _, e = m.HostAction(t.Context(), bad); !errors.Is(e, ErrInvalid) {
				t.Fatal(e)
			}
			bad = r
			bad.Revision = strings.Repeat("f", 64)
			if _, e = m.HostAction(t.Context(), bad); !errors.Is(e, ErrConflict) {
				t.Fatal(e)
			}
			result, e := m.HostAction(t.Context(), r)
			if e != nil || result.Outcome != "confirmed" || !result.ReadbackUnavailable || result.Host != nil || commands != 1 {
				t.Fatal(result, e, commands)
			}
		})
	}
}
func TestTailnetInitialLoginUsesBoundedUpNotReset(t *testing.T) {
	m := managementFixture(t)
	base := m.local.Transport
	m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			return managementResponse(200, strings.Replace(strings.Replace(nativeStatusFixture(), `"HaveNodeKey":true`, `"HaveNodeKey":false`, 1), `"Running"`, `"NeedsLogin"`, 1)), nil
		}
		return base.RoundTrip(r)
	})
	m.command = func(ctx context.Context, path string, args ...string) ([]byte, error) {
		if path == DefaultCLI && strings.Join(args, " ") == "version --json" {
			return []byte(`{"short":"1.102.4"}`), nil
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Error("unbounded CLI")
		}
		if path != DefaultCLI || strings.Join(args, " ") != "--socket="+hostSocket+" up --json --timeout=5s" {
			t.Error("unexpected login command")
		}
		return nil, errors.New("synthetic native wait expired")
	}
	before, _, _ := m.observe(t.Context())
	result, e := m.HostAction(t.Context(), HostRequest{Action: "signin", Revision: before.Revision})
	if e != nil || result.Outcome != "pending" || result.AuthURL == "" {
		t.Fatal(result, e)
	}
}
func TestTailnetCredentialCheckIsScopedBoundedAndNoRegistration(t *testing.T) {
	for _, kind := range []string{"ok", "error", "oversized", "redirect", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			m := managementFixture(t)
			calls := 0
			m.provider.Transport = boundedProviderTransport{managementRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.String() != "https://api.tailscale.com/api/v2/oauth/token" || r.Method != "POST" {
					t.Error("device or arbitrary endpoint requested")
				}
				id, secret, ok := r.BasicAuth()
				if !ok || id != "synthetic-client" || secret != enrollmentInput().ClientSecret {
					t.Error("incorrect auth transport")
				}
				b, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(b), "scope=auth_keys") || !strings.Contains(string(b), "tags=tag%3Asoda-project") {
					t.Error("scope/tag constraint missing")
				}
				payload := `{"access_token":"synthetic-access","token_type":"Bearer","expires_in":3600}`
				switch kind {
				case "error":
					return managementResponse(403, `{"error":"synthetic secret"}`), nil
				case "redirect":
					res := managementResponse(307, payload)
					res.Header.Set("Location", "https://evil.test")
					return res, nil
				case "oversized":
					return managementResponse(200, payload+strings.Repeat(" ", responseLimit)), nil
				case "cancelled":
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return managementResponse(200, payload), nil
			})}
			ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
			defer cancel()
			e := m.checkCredential(ctx, enrollmentInput())
			if (e == nil) != (kind == "ok") || calls != 1 {
				t.Fatal(kind, e, calls)
			}
			if e != nil && strings.Contains(e.Error(), "synthetic") {
				t.Fatal("provider diagnostic exposed")
			}
		})
	}
}
func TestTailnetHostRevisionIncludesNativeIdentityAndCLIIsPinned(t *testing.T) {
	m := managementFixture(t)
	before, _, e := m.observe(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	base := m.local.Transport
	m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			return managementResponse(200, strings.Replace(nativeStatusFixture(), `"ID":"self"`, `"ID":"replacement"`, 1)), nil
		}
		return base.RoundTrip(r)
	})
	after, _, e := m.observe(t.Context())
	if e != nil || after.Revision == before.Revision {
		t.Fatal("replacement did not retire revision", e)
	}
	m.command = func(ctx context.Context, path string, args ...string) ([]byte, error) {
		if path != DefaultCLI || strings.Join(args, " ") != "version --json" {
			t.Error("unsupported CLI dispatched mutation")
		}
		return []byte(`{"short":"1.1.0"}`), nil
	}
	b := true
	if _, e = m.HostAction(t.Context(), HostRequest{Action: "advertise-exit-node", Confirm: "advertise-exit-node", Advertise: &b, Revision: after.Revision}); !errors.Is(e, ErrUnsupported) {
		t.Fatal(e)
	}
}
func TestTailnetExitSelectionPreservesRoutesAndLogoutHidesAuthURL(t *testing.T) {
	for _, action := range []string{"exit-node", "logout"} {
		t.Run(action, func(t *testing.T) {
			m := managementFixture(t)
			mutations := 0
			changed := false
			m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/localapi/v0/status":
					status := nativeStatusFixture()
					if changed && action == "logout" {
						status = strings.Replace(strings.Replace(status, `"Running"`, `"NeedsLogin"`, 1), `"HaveNodeKey":true`, `"HaveNodeKey":false`, 1)
					}
					return managementResponse(200, status), nil
				case "/localapi/v0/prefs":
					prefs := nativePrefsFixture()
					if changed && action == "exit-node" {
						prefs = strings.Replace(strings.Replace(prefs, `"ExitNodeID":""`, `"ExitNodeID":"peer"`, 1), `"ExitNodeAllowLANAccess":false`, `"ExitNodeAllowLANAccess":true`, 1)
					}
					return managementResponse(200, prefs), nil
				case "/localapi/v0/logout":
					if r.Method != "POST" || action != "logout" {
						t.Error("unexpected logout")
					}
					mutations++
					changed = true
					return managementResponse(204, ""), nil
				}
				t.Error("unexpected LocalAPI path")
				return managementResponse(500, ""), nil
			})
			m.command = func(ctx context.Context, path string, args ...string) ([]byte, error) {
				if path == DefaultCLI && strings.Join(args, " ") == "version --json" {
					return []byte(`{"short":"1.102.4"}`), nil
				}
				if path != DefaultCLI || strings.Join(args, " ") != "--socket="+hostSocket+" set --exit-node=100.64.0.2 --exit-node-allow-lan-access=true" {
					t.Error("unrelated route/preferences overwritten")
				}
				mutations++
				changed = true
				return nil, nil
			}
			before, _, e := m.observe(t.Context())
			if e != nil {
				t.Fatal(e)
			}
			r := HostRequest{Action: action, Confirm: action, Revision: before.Revision}
			if action == "exit-node" {
				ip := "100.64.0.2"
				lan := true
				r.ExitNode = &ip
				r.AllowLAN = &lan
			}
			v, e := m.HostAction(t.Context(), r)
			if e != nil || v.Validate() != nil || v.Outcome != "confirmed" || mutations != 1 || v.AuthURL != "" {
				t.Fatal(v, e, mutations)
			}
		})
	}
}
func TestTailnetOfflineExitNodeAndUnconfirmedClear(t *testing.T) {
	for _, mode := range []string{"offline", "clear-retained-id"} {
		t.Run(mode, func(t *testing.T) {
			m := managementFixture(t)
			calls := 0
			m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/status") {
					return managementResponse(200, strings.Replace(nativeStatusFixture(), `"Online":true`, `"Online":false`, 1)), nil
				}
				return managementResponse(200, strings.Replace(nativePrefsFixture(), `"ExitNodeID":""`, `"ExitNodeID":"peer"`, 1)), nil
			})
			m.command = func(ctx context.Context, path string, args ...string) ([]byte, error) {
				if path == DefaultCLI && strings.Join(args, " ") == "version --json" {
					return []byte(`{"short":"1.102.4"}`), nil
				}
				calls++
				return nil, nil
			}
			before, _, e := m.observe(t.Context())
			if e != nil {
				t.Fatal(e)
			}
			ip := "100.64.0.2"
			lan := false
			if mode == "clear-retained-id" {
				ip = ""
			}
			result, e := m.HostAction(t.Context(), HostRequest{Action: "exit-node", Confirm: "exit-node", Revision: before.Revision, ExitNode: &ip, AllowLAN: &lan})
			if mode == "offline" {
				if !errors.Is(e, ErrConflict) || calls != 0 {
					t.Fatal(e, calls)
				}
			} else if e != nil || result.Outcome != "unconfirmed" || calls != 1 {
				t.Fatal(result, e, calls)
			}
		})
	}
}

// Browser parity uses the existing bounded Go notification decoder, not a
// second browser framing engine. These are synthetic command/LocalAPI results.
func TestTailnetInitialLoginNotificationsAndTimeoutKeepNativeAuthentication(t *testing.T) {
	for _, mode := range []string{"multiple notifications", "observer timeout", "malformed notification", "private diagnostic"} {
		t.Run(mode, func(t *testing.T) {
			m := managementFixture(t)
			base := m.local.Transport
			m.local.Transport = managementRoundTrip(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/status") {
					status := strings.Replace(strings.Replace(nativeStatusFixture(), `"HaveNodeKey":true`, `"HaveNodeKey":false`, 1), `"Running"`, `"NeedsLogin"`, 1)
					return managementResponse(200, status), nil
				}
				return base.RoundTrip(r)
			})
			calls := 0
			m.command = func(ctx context.Context, path string, args ...string) ([]byte, error) {
				if strings.Join(args, " ") == "version --json" {
					return []byte(`{"short":"1.102.4"}`), nil
				}
				calls++
				if path != DefaultCLI || strings.Join(args, " ") != "--socket="+hostSocket+" up --json --timeout=5s" {
					t.Error("unexpected command")
				}
				switch mode {
				case "observer timeout":
					return nil, context.DeadlineExceeded
				case "malformed notification":
					return []byte(`not json`), nil
				case "private diagnostic":
					return []byte(`{"Error":"must-not-escape-private-diagnostic"}`), nil
				default:
					return []byte("{\n\"AuthURL\":\"https://login.tailscale.com/a/synthetic\"\n}\n{\"BackendState\":\"NeedsLogin\"}\n"), nil
				}
			}
			before, _, e := m.observe(t.Context())
			if e != nil {
				t.Fatal(e)
			}
			out, e := m.HostAction(t.Context(), HostRequest{Action: "signin", Revision: before.Revision})
			if e != nil || out.Outcome != "pending" || out.AuthURL == "" || calls != 1 {
				t.Fatal(out, e, calls)
			}
			data, _ := json.Marshal(out)
			if strings.Contains(string(data), "must-not-escape") {
				t.Fatal("raw diagnostic escaped")
			}
		})
	}
}

func TestTailnetBoundedOutputChild(t *testing.T) {
	if os.Args[len(os.Args)-1] != "soda-tailnet-stdout-probe" {
		return
	}
	fmt.Print(strings.Repeat("x", responseLimit+1))
	os.Exit(0)
}
func TestTailnetCommandOutputBounds(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	_, e := managementCommand(ctx, os.Args[0], "-test.run=^TestTailnetBoundedOutputChild$", "--", "soda-tailnet-stdout-probe")
	if !errors.Is(e, ErrUnconfirmed) {
		t.Fatal("oversized command output accepted", e)
	}
}
