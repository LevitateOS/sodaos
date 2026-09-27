package forgejo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type gitNativeFixture struct {
	requests                            int
	userID, repoID                      int64
	verified, pull, push, noPermissions bool
	failure                             string
}

func gitProviderFixture(t *testing.T) (*Provider, *gitNativeFixture, []byte) {
	t.Helper()
	f := &gitNativeFixture{userID: 7, repoID: 42, verified: true, pull: true, push: true}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests++
		if r.Header.Get("Authorization") != "token fixture-access" {
			t.Error("native credential not centrally applied")
		}
		if r.URL.Path == f.failure {
			w.WriteHeader(502)
			_, _ = w.Write([]byte("sensitive native diagnostic"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/user":
			_, _ = fmt.Fprintf(w, `{"id":%d,"login":"soda-tester"}`, f.userID)
		case "/api/v1/user/emails":
			_, _ = fmt.Fprintf(w, `[{"email":"soda-tester@example.test","verified":%t,"primary":true}]`, f.verified)
		case "/api/v1/repos/soda-tester/repo", "/api/v1/repositories/42":
			permissions := "null"
			if !f.noPermissions {
				permissions = fmt.Sprintf(`{"pull":%t,"push":%t}`, f.pull, f.push)
			}
			_, _ = fmt.Fprintf(w, `{"id":%d,"owner":{"id":7,"login":"soda-tester"},"name":"repo","full_name":"soda-tester/repo","permissions":%s}`, f.repoID, permissions)
		default:
			t.Errorf("unexpected native route %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	p, err := New(Config{Base: server.URL, ClientID: "broker-client", RedirectURL: "https://forgejo.example.test/-/soda/identity/callback"}, "fixture-secret")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(Credential{Access: "fixture-access", Refresh: "fixture-refresh", Expiry: time.Now().Add(time.Hour).Unix(), UserID: 7, Scopes: requiredScopes})
	if err != nil {
		t.Fatal(err)
	}
	return p, f, data
}

func TestResolveGitRepositoryUsesNativeStableIdentity(t *testing.T) {
	p, f, data := gitProviderFixture(t)
	repo, err := p.ResolveGitRepository(t.Context(), 7, data, "soda-tester", "repo")
	if err != nil || repo.ID != 42 || repo.FullName != "soda-tester/repo" {
		t.Fatal("native repository resolution failed")
	}
	calls := f.requests
	if _, err := p.ResolveGitRepository(t.Context(), 8, data, "soda-tester", "repo"); !errors.Is(err, identity.ErrDenied) || f.requests != calls {
		t.Fatal("foreign credential accessed native repository")
	}
	if _, err := p.ResolveGitRepository(t.Context(), 7, data, "..", "repo"); !errors.Is(err, identity.ErrDenied) || f.requests != calls {
		t.Fatal("invalid repository path reached native service")
	}
	f.failure = "/api/v1/repos/soda-tester/repo"
	if _, err := p.ResolveGitRepository(t.Context(), 7, data, "soda-tester", "repo"); !errors.Is(err, identity.ErrDenied) || strings.Contains(err.Error(), "sensitive") {
		t.Fatal("native repository failure disclosed diagnostics")
	}
}

func TestGitAuthorityChecksCurrentNativePermissionAndAttribution(t *testing.T) {
	p, f, data := gitProviderFixture(t)
	repo, user, email, err := p.GitAuthority(t.Context(), 7, data, 42, true)
	if err != nil || repo.ID != 42 || user.ID != 7 || email != "soda-tester@example.test" {
		t.Fatal("verified native Git authority failed")
	}
	f.push = false
	if _, _, _, err := p.GitAuthority(t.Context(), 7, data, 42, true); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("cached push permission admitted write")
	}
	if _, _, _, err := p.GitAuthority(t.Context(), 7, data, 42, false); err != nil {
		t.Fatal("read authority rejected after write revoked")
	}
	f.pull = false
	if _, _, _, err := p.GitAuthority(t.Context(), 7, data, 42, false); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("revoked pull permission admitted read")
	}
}

func TestGitAuthorityRejectsUnverifiedNativeResponse(t *testing.T) {
	for _, reason := range []string{"foreign owner", "unverified email", "foreign repository", "missing permissions", "native failure", "foreign credential", "invalid credential", "invalid repository"} {
		t.Run(reason, func(t *testing.T) {
			p, f, data := gitProviderFixture(t)
			owner, repoID := int64(7), int64(42)
			switch reason {
			case "foreign owner":
				f.userID = 8
			case "unverified email":
				f.verified = false
			case "foreign repository":
				f.repoID = 43
			case "missing permissions":
				f.noPermissions = true
			case "native failure":
				f.failure = "/api/v1/user/emails"
			case "foreign credential":
				owner = 8
			case "invalid credential":
				data = []byte(`{"access":"fixture-access"}`)
			case "invalid repository":
				repoID = 0
			}
			repo, user, email, err := p.GitAuthority(t.Context(), owner, data, repoID, true)
			if !errors.Is(err, identity.ErrDenied) || repo.ID != 0 || user.ID != 0 || email != "" || strings.Contains(err.Error(), "sensitive") {
				t.Fatal("invalid native authority returned attribution or diagnostics")
			}
			if (reason == "foreign credential" || reason == "invalid credential" || reason == "invalid repository") && f.requests != 0 {
				t.Fatal("invalid request reached native service")
			}
		})
	}
}
