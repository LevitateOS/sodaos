package web

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
	"golang.org/x/crypto/ssh"
)

func apiTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for i, login := range []string{"alice", "bob"} {
		id := int64(i + 1)
		if err = db.UpsertUser(context.Background(), store.User{ID: id, Login: login, Name: login}); err != nil {
			t.Fatal(err)
		}
		if err = db.CreateSession(context.Background(), "session-"+login, id, "csrf-"+login); err != nil {
			t.Fatal(err)
		}
	}
	return New(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1}, db)
}

func apiTestRequest(method, path, body, login string) *http.Request {
	r := httptest.NewRequest(method, config.SodaPath+path, strings.NewReader(body))
	if login != "" {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-" + login})
	}
	if id := map[string]string{"alice": "1", "bob": "2"}[login]; id != "" {
		r.Header.Set(expectedUserHeader, id)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://forgejo.example.test")
	r.Header.Set("X-CSRF-Token", "csrf-"+login)
	return r
}

func TestAPISessionAndErrorsRemainJSON(t *testing.T) {
	s := apiTestServer(t)
	for _, tc := range []struct {
		method, path, login string
		status              int
	}{
		{"GET", "/api/session", "alice", 200},
		{"GET", "/api/session", "", 401},
		{"GET", "/api/session", "unknown", 401},
		{"POST", "/api/session", "alice", 405},
		{"GET", "/api", "alice", 404},
		{"GET", "/api/missing", "alice", 404},
	} {
		t.Run(tc.method+tc.path+tc.login, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := apiTestRequest(tc.method, tc.path, "{}", tc.login)
			r.Header.Set("HX-Request", "true")
			s.ServeHTTP(w, r)
			if w.Code != tc.status || !json.Valid(w.Body.Bytes()) || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Location") != "" || w.Header().Get("HX-Redirect") != "" {
				t.Fatal("API cached or redirected")
			}
		})
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/session", "", "alice"))
	var v struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		CSRF     string `json:"csrf_token"`
		Operator bool   `json:"soda_operator"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.User.ID != "1" || v.CSRF != "csrf-alice" || !v.Operator {
		t.Fatalf("bad session view: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "session-alice") || strings.Contains(w.Body.String(), "admin_token") {
		t.Fatal("session credential exposed")
	}
}

func TestAPIProtectsEveryMutationMethod(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, invalid := range []string{"", "csrf", "origin", "missing-origin", "duplicate-origin", "duplicate-csrf", "fetch-site", "content-type"} {
			t.Run(method+invalid, func(t *testing.T) {
				s := apiTestServer(t)
				called := false
				s.mux.HandleFunc("/api/test-write", s.apiProtected(func(w http.ResponseWriter, r *http.Request, _ store.Session) {
					called = true
					w.WriteHeader(204)
				}, method))
				r := apiTestRequest(method, "/api/test-write", "{}", "alice")
				status := 403
				switch invalid {
				case "":
					status = 204
				case "csrf":
					r.Header.Set("X-CSRF-Token", "csrf-bob")
				case "origin":
					r.Header.Set("Origin", "https://evil.example.test")
				case "missing-origin":
					r.Header.Del("Origin")
				case "duplicate-origin":
					r.Header.Add("Origin", "https://evil.example.test")
				case "duplicate-csrf":
					r.Header.Add("X-CSRF-Token", "csrf-alice")
				case "fetch-site":
					r.Header.Set("Sec-Fetch-Site", "cross-site")
				case "content-type":
					r.Header.Set("Content-Type", "text/plain")
					status = 415
				}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != status || called != (invalid == "") {
					t.Fatalf("status=%d called=%v", w.Code, called)
				}
			})
		}
	}
}

func TestAPIPreferencesAreLocalAndUserScoped(t *testing.T) {
	s := apiTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/me/preferences", `{"display_name":" Alice Soda "}`, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	alice, _ := s.Store.User(context.Background(), 1)
	bob, _ := s.Store.User(context.Background(), 2)
	if alice.Name != "Alice Soda" || alice.Login != "alice" || bob.Name != "bob" {
		t.Fatal(alice, bob)
	}
	for _, body := range []string{`null`, `[]`, `{}`, `{"display_name":null}`, `{"display_name":"x","user_id":2}`, `{"display_name":"x"} {}`, `{"display_name":123}`} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("PATCH", "/api/me/preferences", body, "alice"))
		if w.Code != 400 {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/me/preferences", `{"display_name":"`+strings.Repeat("x", apiBodyLimit)+`"}`, "alice"))
	if w.Code != 413 {
		t.Fatal(w.Code)
	}
}

func TestAPIKeysCanonicalAndSeparate(t *testing.T) {
	s := apiTestServer(t)
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	canonical := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
	body, _ := json.Marshal(map[string]string{"public_key": canonical + " fixture-comment"})
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/me/development-keys", string(body), "alice"))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	keys, err := s.Store.Keys(context.Background(), 1)
	if err != nil || len(keys) != 1 || keys[0].Public != canonical+"\n" || keys[0].Fingerprint != ssh.FingerprintSHA256(key) {
		t.Fatal(keys, err)
	}
	other, err := s.Store.Keys(context.Background(), 2)
	if err != nil || len(other) != 0 {
		t.Fatal(other, err)
	}
	for _, input := range []string{"", "-----BEGIN OPENSSH PRIVATE KEY-----", `command="false" ` + canonical, canonical + "\n" + canonical} {
		body, _ = json.Marshal(map[string]string{"public_key": input})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/me/development-keys", string(body), "alice"))
		if w.Code != 400 {
			t.Fatalf("invalid key accepted: %d", w.Code)
		}
	}
}

func TestAPILogoutDeletesOnlyActingSession(t *testing.T) {
	s := apiTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/session/logout", "{}", "alice"))
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	if _, err := s.Store.Session(context.Background(), "session-alice"); err == nil {
		t.Fatal("session survived")
	}
	if _, err := s.Store.Session(context.Background(), "session-bob"); err != nil {
		t.Fatal("other session removed", err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("unsafe logout cookie")
	}
}

func TestAPIStorageFailureIsNotAuthenticationFailure(t *testing.T) {
	s := apiTestServer(t)
	s.Store.Close()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/session", "", "alice"))
	if w.Code != 503 || !json.Valid(w.Body.Bytes()) {
		t.Fatal(w.Code, w.Body.String())
	}
}
