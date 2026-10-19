package forgejo

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestExtensionManagerUsesVerifiedPeerWithoutBootstrapOrReplay(t *testing.T) {
	socket := shortSocketPath(t, "manager.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/extensions/manager/state" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "" {
			t.Error("manager request used an unexpected path, method or credential")
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"package_id":"tailnet"}` {
			t.Error("manager request body changed")
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":{"code":"operation_in_progress"}}`)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	b := NewServiceBackground(socket, uint32(os.Geteuid()), "")
	status, data, err := b.ExtensionManager(context.Background(), "state", []byte(`{"package_id":"tailnet"}`))
	if err != nil || status != http.StatusConflict || !strings.Contains(string(data), "operation_in_progress") || calls.Load() != 1 {
		t.Fatalf("rejection was not preserved exactly once: %d %s %v (%d calls)", status, data, err, calls.Load())
	}
	b.hostUID++
	if _, _, err := b.ExtensionManager(context.Background(), "state", []byte(`{}`)); err == nil || calls.Load() != 1 {
		t.Fatal("request reached an unverified peer")
	}
	if _, _, err := b.ExtensionManager(context.Background(), "../bootstrap", []byte(`{}`)); err == nil || calls.Load() != 1 {
		t.Fatal("unlisted manager operation was accepted")
	}
}

func TestExtensionManagerRefusesRedirectAsyncAndOversizedAnswers(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusAccepted, http.StatusOK} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			socket := shortSocketPath(t, "manager.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", "http://extensionHost/other")
				w.WriteHeader(status)
				body := `{}`
				if status == http.StatusOK {
					body = `{"padding":"` + strings.Repeat("x", 128<<10) + `"}`
				}
				_, _ = io.WriteString(w, body)
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			b := NewServiceBackground(socket, uint32(os.Geteuid()), "")
			if _, _, err := b.ExtensionManager(context.Background(), "reconcile", []byte(`{}`)); err == nil || calls.Load() != 1 {
				t.Fatal("invalid answer accepted or request retransmitted")
			}
		})
	}
}
