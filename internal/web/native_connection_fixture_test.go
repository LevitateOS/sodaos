package web

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/nativebuild"
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
	// Optional selected-delivery asset boundary. The older bundle must be verified
	// with its own verifier by the caller; bind these exact bytes to that receipt.
	// Asset-only mode does not execute the predecessor. The optional bound binary
	// adds a final genuine-document/backend phase after all normal consumers.
	predecessor := os.Getenv("SODA_CONNECTION_PREDECESSOR")
	phaseFile := filepath.Join(dir, "asset-phase")
	oldPublic := ""
	oldHashes := map[string]string{}
	predecessorBinary := os.Getenv("SODA_CONNECTION_PREDECESSOR_BINARY")
	backendPhase := filepath.Join(dir, "backend-phase")
	var oldProxy *httputil.ReverseProxy
	var transition func() (*Server, error)
	var transitionOnce sync.Once
	var migrated *Server
	var transitionErr error
	if predecessorBinary != "" && predecessor == "" {
		t.Fatal("predecessor backend requires the verified bundle binding")
	}
	if predecessor != "" {
		if !filepath.IsAbs(predecessor) {
			t.Fatal("absolute verified predecessor bundle required")
		}
		data, err := os.ReadFile(filepath.Join(predecessor, "build-info.json"))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != os.Getenv("SODA_CONNECTION_PREDECESSOR_MANIFEST_SHA256") {
			t.Fatal("predecessor manifest binding failed")
		}
		var inventory nativebuild.Inventory
		if json.Unmarshal(data, &inventory) != nil || inventory.Revision == "" || inventory.Revision != os.Getenv("SODA_CONNECTION_PREDECESSOR_REVISION") {
			t.Fatal("predecessor revision binding failed")
		}
		if predecessorBinary != "" {
			info, err := os.Lstat(predecessorBinary)
			if !filepath.IsAbs(predecessorBinary) || err != nil || !info.Mode().IsRegular() || info.Size() > 128<<20 {
				t.Fatal("predecessor executable must be a bounded regular file")
			}
			body, err := os.ReadFile(predecessorBinary)
			expected := inventory.Files["rootfs/usr/local/libexec/soda/soda-dashboard"].SHA256
			if err != nil || expected == "" || fmt.Sprintf("%x", sha256.Sum256(body)) != expected {
				t.Fatal("predecessor executable binding failed")
			}
		}
		oldPublic = filepath.Join(predecessor, "rootfs/var/lib/soda/forgejo/gitea/public")
		for _, name := range []string{"/assets/sodaspaces-page.js", "/assets/sodaspaces-api.js", "/assets/soda/forgejo/lit.js"} {
			path := filepath.Join(oldPublic, name)
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
				t.Fatal("predecessor module unavailable")
			}
			data, err := os.ReadFile(path)
			expected := inventory.Files["rootfs/var/lib/soda/forgejo/gitea/public"+name].SHA256
			if err != nil || expected == "" || fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
				t.Fatal("predecessor module binding failed")
			}
			oldHashes[name] = expected
		}
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend, _ := os.ReadFile(backendPhase)
		if strings.HasPrefix(r.URL.Path, config.SodaPath+"/") {
			if predecessorBinary != "" && string(backend) == "predecessor" {
				oldProxy.ServeHTTP(w, r)
				return
			}
			if predecessorBinary != "" && string(backend) == "candidate" {
				transitionOnce.Do(func() { migrated, transitionErr = transition() })
				if transitionErr != nil {
					http.Error(w, "fixture backend transition failed", 503)
					return
				}
				migrated.ServeHTTP(w, r)
				return
			}
			soda.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			assets := public
			if predecessorBinary != "" && string(backend) == "predecessor" {
				assets = oldPublic
			}
			phase, _ := os.ReadFile(phaseFile)
			if predecessor != "" && string(phase) == "predecessor" && oldHashes[r.URL.Path] != "" {
				assets = oldPublic
			}
			if _, err := os.Stat(filepath.Join(assets, filepath.FromSlash(r.URL.Path))); err == nil {
				// Default: stock preview cache. Selected transition: the inspected
				// installed revalidation policy, still a local file-server fixture.
				w.Header().Set("Cache-Control", "private, max-age=21600")
				if predecessor != "" {
					w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
				}
				http.FileServer(http.Dir(assets)).ServeHTTP(w, r)
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
		ID      int64 `json:"id"`
		IsAdmin bool  `json:"is_admin"`
	}
	if userResponse.StatusCode != 200 || json.NewDecoder(userResponse.Body).Decode(&actor) != nil || actor.ID <= 0 {
		t.Fatal("fixture identity unavailable")
	}
	if actor.IsAdmin {
		t.Fatal("native operator-without-Forgejo-admin fixture must remain non-admin")
	}
	soda = New(config.Config{OperatorID: actor.ID, ForgejoURL: server.URL, ForgejoInternalURL: upstream.String(), OAuthClientID: app.ClientID, OAuthSecretFile: secretFile}, db)
	soda.Host = &host.Client{HTTP: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/runners/list":
			_, _ = w.Write([]byte("[]"))
		case r.Method == "POST" && r.URL.Path == "/profile":
			// A fresh fixture repository may belong to this actor. Its owner
			// view reads creation metadata; this remains synthetic helper data,
			// not an installed profile or permission to create a project.
			_ = json.NewEncoder(w).Encode(testCreationProfile())
		default:
			t.Error("unexpected fixture helper operation", r.Method, r.URL.Path)
			w.WriteHeader(503)
		}
		return w.Result(), nil
	})}}
	if predecessorBinary != "" {
		// The normal consumers run first. The final browser phase then uses this
		// real old executable and its own fresh v6 DB, followed by current handlers
		// migrating that SAME DB. No retained appliance grants are copied here.
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		address := listener.Addr().String()
		_ = listener.Close()
		oldURL, _ := url.Parse("http://" + address)
		oldProxy = httputil.NewSingleHostReverseProxy(oldURL)
		oldProxy.ErrorLog = log.New(io.Discard, "", 0)
		keyPath := filepath.Join(dir, "predecessor-grant-key")
		if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0600); err != nil {
			t.Fatal(err)
		}
		cfg := config.Config{Listen: address, OperatorID: actor.ID, ForgejoURL: server.URL, ForgejoInternalURL: upstream.String(), OAuthClientID: app.ClientID, OAuthSecretFile: secretFile, GrantKeyFile: keyPath, Database: filepath.Join(dir, "predecessor.db"), HostSocket: filepath.Join(dir, "absent-native-helper.sock"), AdminTokenFile: filepath.Join(dir, "unused-admin-token")}
		body, _ := json.Marshal(cfg)
		configPath := filepath.Join(dir, "predecessor-config.json")
		if err := os.WriteFile(configPath, body, 0600); err != nil {
			t.Fatal(err)
		}
		output, err := os.OpenFile(filepath.Join(dir, "predecessor.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		child := exec.Command(predecessorBinary, "--config", configPath)
		child.Stdout, child.Stderr = output, output
		if err := child.Start(); err != nil {
			_ = output.Close()
			t.Fatal("predecessor process startup failed")
		}
		done := make(chan struct{})
		var childErr error
		go func() { childErr = child.Wait(); _ = output.Close(); close(done) }()
		t.Cleanup(func() {
			select {
			case <-done:
			default:
				_ = child.Process.Signal(syscall.SIGTERM)
				select {
				case <-done:
				case <-time.After(30 * time.Second):
					_ = child.Process.Kill()
					<-done
				}
			}
		})
		// b8af68c validates the obsolete admin-token PATH, but never reads it.
		// No token is created or borrowed. Detect early config/process refusal
		// before creating a browser context or exercising the normal consumers.
		select {
		case <-done:
			t.Fatal("predecessor exited before browser startup; see predecessor.log")
		case <-time.After(200 * time.Millisecond):
		}
		transition = func() (*Server, error) {
			if err := child.Process.Signal(syscall.SIGTERM); err != nil {
				return nil, fmt.Errorf("predecessor stop failed")
			}
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				return nil, fmt.Errorf("predecessor stop unconfirmed")
			}
			if childErr != nil {
				return nil, fmt.Errorf("predecessor exited unsuccessfully")
			}
			readVersion := func() (int, error) {
				u := url.URL{Scheme: "file", Path: cfg.Database, RawQuery: "mode=ro"}
				observed, err := sql.Open("sqlite", u.String())
				if err != nil {
					return 0, err
				}
				defer observed.Close()
				var version int
				err = observed.QueryRow("SELECT version FROM schema_version").Scan(&version)
				return version, err
			}
			before, err := readVersion()
			if err != nil || before != 6 {
				return nil, fmt.Errorf("predecessor did not leave schema v6")
			}
			upgraded, err := store.OpenEncrypted(cfg.Database, key)
			if err != nil {
				return nil, err
			}
			t.Cleanup(func() { _ = upgraded.Close() })
			after, err := readVersion()
			if err != nil || after != 9 {
				return nil, fmt.Errorf("current backend did not migrate to schema v9")
			}
			receipt, _ := json.Marshal(map[string]any{"before": before, "after": after, "same_database": true, "prior_exit_confirmed": true, "scope": "fresh synthetic fixture grants, not retained appliance data"})
			if err := os.WriteFile(filepath.Join(dir, "backend-transition.json"), receipt, 0600); err != nil {
				return nil, err
			}
			current := New(cfg, upgraded)
			current.Host = soda.Host
			return current, nil
		}
	}
	receipt, _ := json.Marshal(map[string]any{"origin": server.URL, "native_origin": upstream.String(), "oauth_application_id": app.ID, "client_id": app.ClientID})
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bun", "test", "--timeout", "120000", "tests/forgejo/native-connection.test.ts")
	cmd.Dir = root
	spki := sha256.Sum256(server.Certificate().RawSubjectPublicKeyInfo)
	cmd.Env = append(os.Environ(), "SODA_CONNECTION_ORIGIN="+server.URL, "SODA_PAGE_STATE="+filepath.Join(dir, "browser-state.json"), "SODA_CONNECTION_SPKI="+base64.StdEncoding.EncodeToString(spki[:]))
	logFile, err := os.OpenFile(filepath.Join(dir, "browser.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if predecessor != "" {
		hashes, err := json.Marshal(oldHashes)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "SODA_CONNECTION_ASSET_PHASE="+phaseFile, "SODA_CONNECTION_OLD_HASHES="+string(hashes))
	}
	if predecessorBinary != "" {
		cmd.Env = append(cmd.Env, "SODA_CONNECTION_BACKEND_PHASE="+backendPhase)
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
