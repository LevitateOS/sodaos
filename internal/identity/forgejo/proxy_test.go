package forgejo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	upstream "github.com/levitateos/sodaos/internal/forgejo"
)

func proxyProvider(t *testing.T, handler http.Handler) (*Provider, []byte, upstream.Repository, upstream.User) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p, err := New(Config{Base: server.URL, ClientID: "broker", RedirectURL: "https://forgejo.example.test/-/soda/identity/callback"}, "fixture-secret")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(Credential{Access: "fixture-access", Refresh: "fixture-refresh", UserID: 7, Expiry: time.Now().Add(time.Hour).Unix(), Scopes: requiredScopes})
	if err != nil {
		t.Fatal(err)
	}
	return p, data, upstream.Repository{ID: 42, Name: "repo", Owner: upstream.User{ID: 7, Login: "soda-tester"}}, upstream.User{ID: 7, Login: "soda-tester"}
}

func TestProxyGitPreservesNativeTransportWithCentralCredentials(t *testing.T) {
	const payload = "000fsynthetic-pack\x00\x01"
	p, data, repo, user := proxyProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		login, access, ok := r.BasicAuth()
		if !ok || login != "soda-tester" || access != "fixture-access" {
			t.Error("native Basic OAuth credential missing")
		}
		for _, name := range []string{"Cookie", "Proxy-Authorization", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-Extra"} {
			if r.Header.Get(name) != "" {
				t.Errorf("inbound header forwarded: %s", name)
			}
		}
		if r.URL.Path != "/soda-tester/repo.git/git-receive-pack" || r.URL.RawQuery != "service=git-receive-pack" || r.Header.Get("Git-Protocol") != "version=2" || strings.Contains(r.Host, "attacker") {
			t.Error("Git request routing or protocol changed")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != payload {
			t.Error("native request body changed")
		}
		for _, name := range []string{"Set-Cookie", "Authorization", "Proxy-Authenticate", "WWW-Authenticate", "Location"} {
			w.Header().Set(name, "sensitive-native-value")
		}
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		w.Header().Set("Trailer", "Authorization, X-Git-Result")
		_, _ = w.Write([]byte("0008NAK\n"))
		w.(http.Flusher).Flush()
		_, _ = w.Write([]byte("0000"))
		w.Header().Set("Authorization", "sensitive-native-trailer")
		w.Header().Set("X-Git-Result", "finished")
	}))
	r := httptest.NewRequest("POST", "https://attacker.example.test/soda-tester/repo.git/git-receive-pack?service=git-receive-pack", strings.NewReader(payload))
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-Extra"} {
		r.Header.Set(name, "caller-private-value")
	}
	r.Header.Set("Git-Protocol", "version=2")
	r.Header.Set("Connection", "Git-Protocol")
	w := httptest.NewRecorder()
	if p.ProxyGit(w, r, 7, data, repo, user) || w.Code != 200 || w.Body.String() != "0008NAK\n0000" {
		t.Fatal("native Git stream changed")
	}
	for _, name := range []string{"Set-Cookie", "Authorization", "Proxy-Authenticate", "WWW-Authenticate", "Location"} {
		if w.Header().Get(name) != "" {
			t.Errorf("native sensitive response header forwarded: %s", name)
		}
	}
	if w.Header().Get("Content-Type") != "application/x-git-receive-pack-result" {
		t.Fatal("native Git content type changed")
	}
	if w.Result().Trailer.Get("Authorization") != "" || w.Result().Trailer.Get("X-Git-Result") != "finished" {
		t.Fatal("sensitive trailer forwarded or native trailer changed")
	}
}

func TestProxyGitBlocksRedirectsAndSignalsAuthenticationFailure(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusUnauthorized, http.StatusForbidden, http.StatusBadGateway} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			p, data, repo, user := proxyProvider(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Location", "https://attacker.example.test/private")
				w.Header().Set("WWW-Authenticate", "sensitive-native-value")
				w.Header().Set("Set-Cookie", "native-private-session")
				w.WriteHeader(status)
				_, _ = w.Write([]byte("sensitive native diagnostic"))
			}))
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/soda-tester/repo.git/info/refs?service=git-upload-pack", nil)
			rejected := p.ProxyGit(w, r, 7, data, repo, user)
			expected := status
			if status == http.StatusFound {
				expected = http.StatusBadGateway
			}
			if rejected != (status == http.StatusUnauthorized) || w.Code != expected || strings.Contains(w.Body.String(), "sensitive") || w.Header().Get("Location") != "" || w.Header().Get("WWW-Authenticate") != "" || w.Header().Get("Set-Cookie") != "" {
				t.Fatal("unsafe native error or incorrect auth rejection signal")
			}
		})
	}
}

func TestProxyGitRejectsInvalidCustodyAndPathIdentity(t *testing.T) {
	calls := 0
	p, data, repo, user := proxyProvider(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(200) }))
	for _, reason := range []string{"owner", "user", "repository", "path", "login"} {
		testRepo, testUser := repo, user
		owner := int64(7)
		switch reason {
		case "owner":
			owner = 8
		case "user":
			testUser.ID = 8
		case "repository":
			testRepo.ID = 0
		case "path":
			testRepo.Name = "../repo"
		case "login":
			testUser.Login = "invalid:login"
		}
		w := httptest.NewRecorder()
		if p.ProxyGit(w, httptest.NewRequest("GET", "/soda-tester/repo.git/info/refs", nil), owner, data, testRepo, testUser) || w.Code != 403 || calls != 0 {
			t.Fatal("invalid custody reached native transport")
		}
	}
}

func TestProxyGitSanitizesTransportFailure(t *testing.T) {
	p, data, repo, user := proxyProvider(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	p.config.Base = "http://127.0.0.1:0"
	w := httptest.NewRecorder()
	if p.ProxyGit(w, httptest.NewRequest("GET", "/soda-tester/repo.git/info/refs", nil), 7, data, repo, user) || w.Code != 502 || strings.Contains(w.Body.String(), "127.0.0.1") {
		t.Fatal("native transport error disclosed diagnostics")
	}
}
