package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func grantedTestServer(t *testing.T, upstream http.HandlerFunc) *Server {
	t.Helper()
	dir := t.TempDir()
	db, err := store.OpenEncrypted(filepath.Join(dir, "soda.db"), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	provider := httptest.NewServer(upstream)
	t.Cleanup(provider.Close)
	secretPath := filepath.Join(dir, "oauth-secret")
	if err = os.WriteFile(secretPath, []byte("test-client-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{PublicURL: "https://soda.example.test", ForgejoURL: "https://forgejo.example.test", ForgejoInternalURL: provider.URL, OAuthClientID: "client", OAuthSecretFile: secretPath, OperatorID: 1, AdminTokenFile: "/must-not-read-bootstrap-token"}, db)
	for i, login := range []string{"alice", "bob"} {
		uid := int64(i + 1)
		if err = db.UpsertUser(context.Background(), store.User{ID: uid, Login: login}); err != nil {
			t.Fatal(err)
		}
		grant := store.Grant{Access: "acting-" + login, Refresh: "refresh-" + login, Scopes: "write:user write:repository write:issue write:organization write:notification write:admin", Expires: time.Now().Add(time.Hour).Unix()}
		if err = db.CreateGrantedSession(context.Background(), "session-"+login, uid, "csrf-"+login, grant); err != nil {
			t.Fatal(err)
		}
	}
	return s
}
func TestActingUserDenialNeverUsesBootstrap(t *testing.T) {
	var calls atomic.Int32
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "token acting-bob" || r.URL.Path != "/api/v1/admin/users" {
			t.Error("wrong acting request")
		}
		w.WriteHeader(403)
		fmt.Fprint(w, `{"message":"private-native-details"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/people", "", "bob"))
	if w.Code != 403 || calls.Load() != 1 || bytes.Contains(w.Body.Bytes(), []byte("private-native")) {
		t.Fatalf("unexpected denial %d %s", w.Code, w.Body.String())
	}
}
func TestNonOperatorForgejoAdministratorCanCreatePerson(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong authority")
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input["username"] != "Native.Name" || input["must_change_password"] != true {
			t.Error("native onboarding altered")
		}
		fmt.Fprint(w, `{"id":9007199254740993,"login":"Native.Name"}`)
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/people", strings.NewReader(url.Values{"csrf": {"csrf-bob"}, "login": {"Native.Name"}, "email": {"person@example.test"}, "password": {"test-initial-password"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-bob"})
	s.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != "/people" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if _, err := s.Store.User(context.Background(), 9007199254740993); err == nil {
		t.Fatal("shadow user created before sign-in")
	}
}
func TestConcurrentRefreshAndLogout(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			t.Error("unexpected endpoint")
		}
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		fmt.Fprint(w, `{"access_token":"rotated-access","refresh_token":"rotated-refresh","token_type":"bearer","expires_in":3600}`)
	})
	ctx := context.Background()
	grant, err := s.Store.Grant(ctx, "session-alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	grant.Expires = time.Now().Add(-time.Minute).Unix()
	if err = s.Store.ReplaceGrant(ctx, "session-alice", 1, grant); err != nil {
		t.Fatal(err)
	}
	session, err := s.Store.Session(ctx, "session-alice")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.userGrant(apiTestRequest("GET", "/api/forgejo/me", "", "alice"), session)
			results <- err
		}()
	}
	<-entered
	if err = s.Store.DeleteSession(ctx, "session-alice"); err != nil {
		t.Fatal(err)
	}
	close(release)
	wg.Wait()
	close(results)
	for err := range results {
		if err == nil {
			t.Fatal("logged-out grant returned")
		}
	}
	if calls.Load() != 1 {
		t.Fatal("refresh replayed")
	}
	if _, err = s.Store.Session(ctx, "session-alice"); err == nil {
		t.Fatal("session resurrected")
	}
}
func TestLegacyAPIRequiresReauthentication(t *testing.T) {
	s := apiTestServer(t)
	s.Forgejo = forgejo.New("http://127.0.0.1:1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/me", "", "alice"))
	if w.Code != 401 {
		t.Fatalf("legacy session got provider authority: %d", w.Code)
	}
}
