package client

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/identity/control"
)

func TestGitProxyStreamsOverPrivateUnixTransport(t *testing.T) {
	const lease = "0123456789abcdef0123456789abcdef"
	const payload = "synthetic-git-pack\x00\x01"
	directory, err := os.MkdirTemp("../../../.artifacts", "client-git-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Error(err)
		}
	})
	socket := filepath.Join(directory, "relay.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	native := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/git/"+lease+"/soda-tester/repo.git/git-receive-pack" || r.URL.RawQuery != "service=git-receive-pack" || r.Header.Get("Git-Protocol") != "version=2" {
			t.Error("private Git path, query or protocol changed")
		}
		for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
			if r.Header.Get(name) != "" {
				t.Errorf("caller credential reached broker: %s", name)
			}
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != payload {
			t.Error("private Git request payload changed")
		}
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		_, _ = w.Write([]byte("part"))
		w.(http.Flusher).Flush()
		<-release
		_, _ = w.Write([]byte("end"))
	})}
	go func() {
		if err := native.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Error(err)
		}
	}()
	t.Cleanup(func() {
		if err := native.Close(); err != nil {
			t.Error(err)
		}
	})
	client := New(socket)
	t.Cleanup(client.HTTP.CloseIdleConnections)
	relay := httptest.NewServer(client.GitProxy(lease))
	defer func() { unblock(); relay.Close() }()
	caller := relay.Client()
	caller.Timeout = 3 * time.Second
	r, err := http.NewRequestWithContext(t.Context(), "POST", relay.URL+"/soda-tester/repo.git/git-receive-pack?service=git-receive-pack", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Git-Protocol", "version=2")
	r.Header.Set("Connection", "Git-Protocol")
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
		r.Header.Set(name, "caller-private-value")
	}
	response, err := caller.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	first := make([]byte, 4)
	if _, err := io.ReadFull(response.Body, first); err != nil || string(first) != "part" {
		t.Fatal("Git response buffered until upstream completion", err)
	}
	// The first bytes must arrive while the native response is still open.
	unblock()
	remaining, err := io.ReadAll(response.Body)
	if err != nil || string(remaining) != "end" || response.StatusCode != 200 {
		t.Fatal("private Git stream payload changed", err)
	}
}

func TestGitProxyRejectsInvalidLeaseLocator(t *testing.T) {
	client := New("unused-private-socket")
	for _, id := range []string{"", "../lease", "0123456789ABCDEF0123456789abcdef", "0123456789abcdef0123456789abcdef/extra", "0123456789abcdef0123456789abcdeg"} {
		w := httptest.NewRecorder()
		client.GitProxy(id).ServeHTTP(w, httptest.NewRequest("GET", "/soda-tester/repo.git/info/refs?service=git-upload-pack", nil))
		if w.Code != http.StatusForbidden {
			t.Fatal("invalid lease locator reached Unix transport")
		}
	}
}

func TestAdminBrokerDeniesPrivateGitOperations(t *testing.T) {
	server := httptest.NewServer((&control.Controller{}).Handler(false))
	defer server.Close()
	client := &Client{HTTP: server.Client()}
	client.HTTP.Transport = &redirectTransport{server.URL, server.Client().Transport}
	if _, err := client.AcquireGit(t.Context(), identity.GitAcquireRequest{}); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("admin listener admitted Git acquisition", err)
	}
	if _, err := client.RegisterGit(t.Context(), "0123456789abcdef0123456789abcdef", identity.Binding{}); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("admin listener admitted Git registration", err)
	}
	w := httptest.NewRecorder()
	client.GitProxy("0123456789abcdef0123456789abcdef").ServeHTTP(w, httptest.NewRequest("GET", "/soda-tester/repo.git/info/refs?service=git-upload-pack", nil))
	if w.Code != http.StatusForbidden {
		t.Fatal("admin listener admitted Git streaming")
	}
}
