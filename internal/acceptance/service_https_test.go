package acceptance

import (
	"bytes"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPSOrigin(t *testing.T) {
	canonical, err := HTTPSOrigin("https://example.test:443")
	if err != nil || canonical != "https://example.test:443/" {
		t.Fatalf("valid origin rejected: %q %v", canonical, err)
	}
	canonical, err = HTTPSOrigin("https://example.test")
	if err != nil || canonical != "https://example.test/" {
		t.Fatalf("bare origin rejected: %q %v", canonical, err)
	}
	for _, value := range []string{
		"http://example.test",
		"https://user:secret@example.test",
		"https://example.test/?code=x",
		"https://example.test/path",
		"https://example.test:0",
		"https://example.test:notaport",
	} {
		if _, err := HTTPSOrigin(value); err == nil {
			t.Errorf("origin accepted: %q", value)
		}
	}
	if _, err := HTTPSOrigin("https://example.test:notaport"); err == nil || err.Error() != "invalid HTTPS origin" {
		t.Errorf("bad port message = %v", err)
	}
	if _, err := HTTPSOrigin("http://example.test"); err == nil || err.Error() != "plain configured HTTPS origin required" {
		t.Errorf("downgrade message = %v", err)
	}
}

func writeCAFile(t *testing.T, dir string, contents []byte, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTrustedCAFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte("synthetic; never parsed here\n")
	if _, err := trustedCAFile(writeCAFile(t, dir, data, 0o600)); err != nil {
		t.Errorf("restricted CA rejected: %v", err)
	}
	if _, err := trustedCAFile("relative/ca.pem"); err == nil {
		t.Error("relative CA accepted")
	}
	if _, err := trustedCAFile(filepath.Join(dir, "missing.pem")); err == nil {
		t.Error("missing CA accepted")
	}
	writable := writeCAFile(t, dir, data, 0o600)
	if err := os.Chmod(writable, 0o664); err != nil {
		t.Fatal(err)
	}
	if _, err := trustedCAFile(writable); err == nil {
		t.Error("group-writable CA accepted")
	}
	target := writeCAFile(t, dir, data, 0o600)
	link := filepath.Join(dir, "link.pem")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := trustedCAFile(link); err == nil {
		t.Error("symlink CA accepted")
	}
}

func pemCA(t *testing.T, server *httptest.Server) []byte {
	t.Helper()
	if server.Certificate() == nil {
		t.Fatal("test server exposes no certificate")
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
}

func TestCheckServiceHTTPS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	ca := writeCAFile(t, t.TempDir(), pemCA(t, server), 0o600)
	var stdout bytes.Buffer
	if err := CheckServiceHTTPS(server.URL+"/", ca, &stdout); err != nil {
		t.Fatalf("pinned TLS check failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Configured-origin TLS verified; HTTP status 200") {
		t.Errorf("unexpected success line: %q", stdout.String())
	}
}

func TestCheckServiceHTTPSAcceptsRedirectStatusWithoutFollowing(t *testing.T) {
	var followed bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/target" {
			followed = true
			return
		}
		http.Redirect(w, r, "/target", http.StatusFound)
	}))
	defer server.Close()
	ca := writeCAFile(t, t.TempDir(), pemCA(t, server), 0o600)
	var stdout bytes.Buffer
	if err := CheckServiceHTTPS(server.URL+"/", ca, &stdout); err != nil {
		t.Fatalf("redirect status rejected: %v", err)
	}
	if followed {
		t.Error("redirect target was followed")
	}
	if !strings.Contains(stdout.String(), "HTTP status 302") {
		t.Errorf("unexpected redirect line: %q", stdout.String())
	}
}

func TestCheckServiceHTTPSRefusals(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer other.Close()
	ca := writeCAFile(t, t.TempDir(), pemCA(t, server), 0o600)
	var stdout bytes.Buffer
	if err := CheckServiceHTTPS(server.URL+"/", ca, &stdout); err == nil {
		t.Error("unexpected status accepted")
	}
	wrongCA := writeCAFile(t, t.TempDir(), pemCA(t, other), 0o600)
	if err := CheckServiceHTTPS(server.URL+"/", wrongCA, &stdout); err == nil {
		t.Error("wrong CA accepted")
	}
	if err := RunServiceHTTPS([]string{server.URL + "/", wrongCA}, &stdout); err == nil {
		t.Error("wrong CA accepted at entrypoint")
	} else if strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("failure leaks origin: %q", err.Error())
	}
}

func TestRunServiceHTTPS(t *testing.T) {
	if err := RunServiceHTTPS([]string{"only-one"}, io.Discard); err == nil {
		t.Error("short argv accepted")
	} else if _, ok := err.(*UsageError); !ok {
		t.Errorf("short argv error = %T, want *UsageError", err)
	}
	if err := RunServiceHTTPS([]string{"http://example.test", "/abs/ca"}, io.Discard); err == nil {
		t.Error("bad origin accepted")
	} else if _, ok := err.(*UsageError); !ok {
		t.Errorf("bad origin error = %T, want *UsageError", err)
	}
	err := RunServiceHTTPS([]string{"https://example.test/", "/abs/missing-ca"}, io.Discard)
	if err == nil {
		t.Fatal("missing CA accepted")
	}
	if !strings.HasPrefix(err.Error(), "HTTPS substrate check failed (") || !strings.HasSuffix(err.Error(), ").") {
		t.Errorf("failure shape = %q", err.Error())
	}
	if strings.Contains(err.Error(), "example.test") {
		t.Errorf("failure leaks origin: %q", err.Error())
	}
}
