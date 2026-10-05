// Port of test_avatar_integration.py: activation CLI, packaging, proxy routes.
package build

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func activateBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-activate", "soda-activate")
}

func runActivate(t *testing.T, argv ...string) ProcResult {
	t.Helper()
	return Run(t, RunOpt{}, activateBinary(t), argv...)
}

func TestAvatarHelpReportsPrivateActivationSurface(t *testing.T) {
	proc := runActivate(t, "--help")
	Check(t, proc.Code == 0, "exit = %d", proc.Code)
	for _, flag := range []string{"--bind-ip", "--certificate", "--private-key", "--local-tls"} {
		Check(t, strings.Contains(proc.Stdout, flag), "missing %s", flag)
	}
	Check(t, strings.Contains(proc.Stdout, "explicit private appliance address"), "stdout=%q", proc.Stdout)
}

func TestAvatarMissingBindIPRejected(t *testing.T) {
	proc := runActivate(t, "--local-tls")
	Check(t, proc.Code == 2, "exit = %d", proc.Code)
	Check(t, strings.Contains(proc.Stderr, "the following arguments are required: --bind-ip"), "stderr=%q", proc.Stderr)
}

func TestAvatarTLSModesRejectedBeforeEffects(t *testing.T) {
	for _, extra := range [][]string{
		{},
		{"--certificate", "/missing"},
		{"--local-tls", "--certificate", "/missing"},
		{"--local-tls", "--private-key", "/missing"},
	} {
		t.Run(strings.Join(extra, " "), func(t *testing.T) {
			proc := runActivate(t, append([]string{"--bind-ip", "192.168.1.5"}, extra...)...)
			Check(t, proc.Code == 2, "exit = %d", proc.Code)
		})
	}
}

func TestAvatarUnknownFlagsRejected(t *testing.T) {
	proc := runActivate(t, "--bind-ip", "192.168.1.5", "--bogus")
	Check(t, proc.Code == 2, "exit = %d", proc.Code)
	Check(t, strings.Contains(proc.Stderr, "unrecognized arguments: --bogus"), "stderr=%q", proc.Stderr)
}

func TestAvatarNonrootRefusedWithoutEffects(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root passes the operator check")
	}
	proc := runActivate(t, "--bind-ip", "192.168.1.5", "--local-tls")
	Check(t, proc.Code == 2, "exit = %d", proc.Code)
	lines := strings.Split(strings.TrimSpace(proc.Stderr), "\n")
	Check(t, lines[len(lines)-1] == "soda-activate: error: native host operator/root required", "stderr=%q", proc.Stderr)
}

func TestAvatarNoAvatarRuntimeCommand(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "cmd/soda-avatars"))
	Check(t, os.IsNotExist(err), "cmd/soda-avatars present")
	_, err = os.Stat(filepath.Join(RepoRoot, "scripts/build-native.sh"))
	Check(t, os.IsNotExist(err), "build-native.sh present")
	Check(t, !strings.Contains(ReadFile(t, "tools/soda-build/main.go"), "soda-avatars"), "soda-avatars in soda-build")
}

func caddyBinary(t *testing.T) string {
	t.Helper()
	path := os.Getenv("SODA_CADDY_BINARY")
	if path == "" {
		t.Skip("set SODA_CADDY_BINARY for real loopback routing checks")
	}
	abs, err := filepath.EvalSymlinks(path)
	Require(t, err == nil, "resolve caddy: %v", err)
	return abs
}

func adaptCaddy(t *testing.T, binary string, env []string, extra ...string) map[string]any {
	t.Helper()
	args := append([]string{"adapt"}, extra...)
	args = append(args, "--config", filepath.Join(RepoRoot, "appliance/config/proxy.Caddyfile"))
	result := Run(t, RunOpt{Env: env}, binary, args...)
	Require(t, result.Code == 0, "caddy adapt failed: %s", result.Stderr)
	var config map[string]any
	Require(t, json.Unmarshal([]byte(result.Stdout), &config) == nil, "parse adapted config")
	return config
}

func TestAvatarLocalTLSUsesNativeIssuerWithoutAutomaticClientTrust(t *testing.T) {
	binary := caddyBinary(t)
	for _, tc := range [][3]string{
		{"192.168.2.100", "https://192.168.2.100", "192.168.2.100:443"},
		{"fd00::5", "https://[fd00::5]", "[fd00::5]:443"},
	} {
		t.Run(tc[0], func(t *testing.T) {
			env := SetEnv(os.Environ(), "FORGEJO_ORIGIN", tc[1])
			env = SetEnv(env, "SODA_BIND", tc[0])
			env = SetEnv(env, "SODA_TLS", "internal")
			config := adaptCaddy(t, binary, env)
			Check(t, config["admin"].(map[string]any)["disabled"] == true, "admin enabled")
			ca := config["apps"].(map[string]any)["pki"].(map[string]any)["certificate_authorities"].(map[string]any)["local"].(map[string]any)
			Check(t, ca["install_trust"] == false, "install_trust set")
			policies := config["apps"].(map[string]any)["tls"].(map[string]any)["automation"].(map[string]any)["policies"].([]any)
			Require(t, len(policies) == 1, "policies = %v", policies)
			policy := policies[0].(map[string]any)
			subjects := policy["subjects"].([]any)
			Check(t, len(subjects) == 1 && subjects[0] == tc[0], "subjects = %v", subjects)
			issuers := policy["issuers"].([]any)
			Require(t, len(issuers) == 1, "issuers = %v", issuers)
			Check(t, issuers[0].(map[string]any)["module"] == "internal", "issuer = %v", issuers[0])
			servers := config["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
			Require(t, len(servers) == 1, "servers = %d", len(servers))
			for _, entry := range servers {
				server := entry.(map[string]any)
				listen := server["listen"].([]any)
				Check(t, len(listen) == 1 && listen[0] == tc[2], "listen = %v", listen)
				Check(t, server["automatic_https"].(map[string]any)["disable_redirects"] == true, "redirects enabled")
			}
		})
	}
}

func TestAvatarProductionRoutesWithTestOwnedUpstreams(t *testing.T) {
	binary := caddyBinary(t)
	temp := TempDir(t)
	env := SetEnv(os.Environ(), "FORGEJO_ORIGIN", "https://forge.example.test")
	env = SetEnv(env, "SODA_BIND", "127.0.0.1")
	env = SetEnv(env, "XDG_CONFIG_HOME", filepath.Join(temp, "config"))
	env = SetEnv(env, "XDG_DATA_HOME", filepath.Join(temp, "data"))
	config := adaptCaddy(t, binary, env, "--adapter", "caddyfile")

	upstream := func(t *testing.T, name string) int {
		t.Helper()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoded := map[string]any{"upstream": name, "path": r.URL.RequestURI()}
			// Python reads headers with .get: missing headers are null, not "".
			if cookie := r.Header.Get("Cookie"); cookie != "" {
				decoded["cookie"] = cookie
			} else {
				decoded["cookie"] = nil
			}
			if auth := r.Header.Get("Authorization"); auth != "" {
				decoded["authorization"] = auth
			} else {
				decoded["authorization"] = nil
			}
			body, _ := json.Marshal(decoded)
			w.Write(body)
		})
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		return server.Listener.Addr().(*net.TCPAddr).Port
	}
	ports := map[string]int{
		"127.0.0.1:3000": upstream(t, "forgejo"),
		"127.0.0.1:8080": upstream(t, "soda"),
	}
	var replaceDials func(node any)
	replaceDials = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			if dial, ok := value["dial"].(string); ok {
				if port, found := ports[dial]; found {
					value["dial"] = "127.0.0.1:" + strconv.Itoa(port)
				}
			}
			for _, child := range value {
				replaceDials(child)
			}
		case []any:
			for _, child := range value {
				replaceDials(child)
			}
		}
	}
	replaceDials(config)
	apps := config["apps"].(map[string]any)
	delete(apps, "tls")
	Check(t, config["admin"].(map[string]any)["disabled"] == true, "admin enabled")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	Require(t, err == nil, "pick port: %v", err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	servers := apps["http"].(map[string]any)["servers"].(map[string]any)
	Require(t, len(servers) == 1, "servers = %d", len(servers))
	var server map[string]any
	for _, entry := range servers {
		server = entry.(map[string]any)
	}
	server["listen"] = []any{"127.0.0.1:" + strconv.Itoa(port)}
	delete(server, "tls_connection_policies")
	server["automatic_https"] = map[string]any{"disable": true}
	conf := filepath.Join(temp, "caddy.json")
	encoded, err := json.Marshal(config)
	Require(t, err == nil, "encode config: %v", err)
	WriteFile(t, conf, encoded, 0o644)
	logPath := filepath.Join(temp, "caddy.log")
	logFile, err := os.Create(logPath)
	Require(t, err == nil, "open log: %v", err)
	t.Cleanup(func() { logFile.Close() })
	process := exec.Command(binary, "run", "--config", conf)
	process.Env = env
	process.Stdout = logFile
	process.Stderr = logFile
	Require(t, process.Start() == nil, "start caddy")
	t.Cleanup(func() {
		process.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() { process.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			process.Process.Kill()
			<-done
		}
	})

	client := &http.Client{Timeout: 3 * time.Second}
	probe := func(path string) (map[string]any, error) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:"+strconv.Itoa(port)+path, nil)
		if err != nil {
			return nil, err
		}
		req.Host = "forge.example.test"
		req.Header.Set("Cookie", "fixture=value")
		req.Header.Set("Authorization", "Bearer [REDACTED]")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, errors.New("status " + strconv.Itoa(resp.StatusCode))
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}
	request := func(t *testing.T, path string) map[string]any {
		t.Helper()
		decoded, err := probe(path)
		Require(t, err == nil, "request %s: %v", path, err)
		return decoded
	}
	started := false
	for range 100 {
		if _, err := probe("/"); err == nil {
			started = true
			break
		}
		if process.Process.Signal(syscall.Signal(0)) != nil {
			logText, _ := os.ReadFile(logPath)
			t.Fatalf("caddy exited: %s", logText)
		}
		time.Sleep(20 * time.Millisecond)
	}
	Require(t, started, "test proxy did not start")
	path := "/-/soda/avatars/v1/" + strings.Repeat("a", 32) + "?s=64&d=identicon"
	response := request(t, path)
	Check(t, response["upstream"] == "soda" && response["path"] == path && response["cookie"] == nil && response["authorization"] == nil,
		"avatar route = %v", response)
	for _, path := range []string{
		"/", "/api/v1/users/alice", "/user/login", "/assets/img/logo.png",
		"/alice/repo.git/info/refs?service=git-upload-pack",
		"/alice/repo.git/info/lfs/objects/batch",
		"/api/packages/alice", "/-/unrelated",
	} {
		response := request(t, path)
		Check(t, response["upstream"] == "forgejo", "%s upstream = %v", path, response["upstream"])
		Check(t, response["path"] == path, "%s path = %v", path, response["path"])
		Check(t, response["cookie"] == "fixture=value", "%s cookie = %v", path, response["cookie"])
	}
	for _, path := range []string{"/-/soda/avatars", "/-/soda/avatars-other/a", "/-/soda/avatarsx/a"} {
		response := request(t, path)
		Check(t, response["upstream"] == "forgejo", "%s upstream = %v", path, response["upstream"])
		Check(t, response["path"] == path, "%s path = %v", path, response["path"])
		Check(t, response["cookie"] == "fixture=value", "%s cookie = %v", path, response["cookie"])
		Check(t, response["authorization"] == "Bearer [REDACTED]", "%s auth = %v", path, response["authorization"])
	}
}
