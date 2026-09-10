package web

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

// Opt-in local authentication proof. Uses the authorized development fixture
// account through native login/API, with a fresh retained Soda DB and OAuth app.
// No mocked provider responses or borrowed cookies. Runner observations below are
// synthetic; no native helper, provider registration or VM operation is performed.
func TestNativeConnectionFixture(t *testing.T) {
	dir := os.Getenv("SODA_NATIVE_CONNECTION_FIXTURE")
	if dir == "" {
		t.Skip("explicit local Forgejo fixture opt-in")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("absolute fresh fixture directory required")
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	credential, err := os.ReadFile(filepath.Join(root, ".local/screenshot-fixture/create-output.txt"))
	if err != nil {
		t.Fatal("fixture credential unavailable")
	}
	match := regexp.MustCompile(`generated random password is '([^']+)'`).FindSubmatch(credential)
	if len(match) != 2 {
		t.Fatal("fixture credential format unavailable")
	}
	upstream, _ := url.Parse("http://localhost:3300")
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.ErrorLog = log.New(io.Discard, "", 0)
	var soda *Server
	public := filepath.Join(dir, "public")
	preview := exec.Command("bun", "scripts/build-forgejo-preview.ts", "--out", filepath.Join(dir, "branding"))
	preview.Dir = root
	prepared, prepareErr := preview.CombinedOutput()
	if err := os.WriteFile(filepath.Join(dir, "preview.log"), prepared, 0600); err != nil {
		t.Fatal(err)
	}
	if prepareErr != nil {
		t.Fatal("candidate asset preparation failed; see preview.log")
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, config.SodaPath+"/") {
			soda.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			if _, err := os.Stat(filepath.Join(public, filepath.FromSlash(r.URL.Path))); err == nil {
				http.FileServer(http.Dir(public)).ServeHTTP(w, r)
				return
			}
		}
		proxy.ServeHTTP(w, r)
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	input, _ := json.Marshal(map[string]any{"name": "Soda connection fixture " + filepath.Base(dir), "redirect_uris": []string{server.URL + config.SodaPath + "/oauth/callback"}, "confidential_client": true})
	req, _ := http.NewRequest("POST", upstream.String()+"/api/v1/user/applications/oauth2", bytes.NewReader(input))
	req.SetBasicAuth("soda-screenshot", string(match[1]))
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatal("native fixture application request failed")
	}
	defer res.Body.Close()
	var app struct {
		ID       int64  `json:"id"`
		ClientID string `json:"client_id"`
		Secret   string `json:"client_secret"`
	}
	if res.StatusCode != 201 || json.NewDecoder(res.Body).Decode(&app) != nil || app.ClientID == "" || app.Secret == "" {
		t.Fatal("native fixture application creation failed", res.StatusCode)
	}
	secretFile := filepath.Join(dir, "oauth-secret")
	if err := os.WriteFile(secretFile, []byte(app.Secret), 0600); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "grant-key"), key, 0600); err != nil {
		t.Fatal(err)
	}
	db, err := store.OpenEncrypted(filepath.Join(dir, "soda.db"), key)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	userRequest, _ := http.NewRequest("GET", upstream.String()+"/api/v1/user", nil)
	userRequest.SetBasicAuth("soda-screenshot", string(match[1]))
	userResponse, err := (&http.Client{Timeout: 10 * time.Second}).Do(userRequest)
	if err != nil {
		t.Fatal("fixture identity request failed")
	}
	defer userResponse.Body.Close()
	var actor struct {
		ID int64 `json:"id"`
	}
	if userResponse.StatusCode != 200 || json.NewDecoder(userResponse.Body).Decode(&actor) != nil || actor.ID <= 0 {
		t.Fatal("fixture identity unavailable")
	}
	soda = New(config.Config{OperatorID: actor.ID, ForgejoURL: server.URL, ForgejoInternalURL: upstream.String(), OAuthClientID: app.ClientID, OAuthSecretFile: secretFile}, db)
	soda.Host = &host.Client{HTTP: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		if r.Method != "POST" || r.URL.Path != "/runners/list" {
			t.Error("unexpected fixture helper operation", r.Method, r.URL.Path)
			w.WriteHeader(503)
		} else {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		}
		return w.Result(), nil
	})}}
	receipt, _ := json.Marshal(map[string]any{"origin": server.URL, "native_origin": upstream.String(), "oauth_application_id": app.ID, "client_id": app.ClientID})
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bun", "test", "--timeout", "120000", "tests/forgejo/native-connection.test.ts")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SODA_CONNECTION_ORIGIN="+server.URL, "SODA_PAGE_STATE="+filepath.Join(dir, "browser-state.json"))
	logFile, err := os.OpenFile(filepath.Join(dir, "browser.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	err = cmd.Run()
	if closeErr := logFile.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal("native connection browser failed; see retained browser.log")
	}
	t.Log("real Forgejo connection browser passed; fixture and OAuth app retained at", dir)
}
