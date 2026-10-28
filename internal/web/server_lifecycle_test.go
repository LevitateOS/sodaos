package web

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

func serveCapabilityBackground(t *testing.T, installation string) string {
	t.Helper()
	const admission = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case extensions.BackgroundBootstrapPath:
			_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "install-1"})
		case extensions.BackgroundOperationsPath:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"installation": installation, "generation": "ops-9",
				"kinds": []string{"issue.lifecycle.apply", "pull_request.review.request", "issue.comment.publish"},
			})
		default:
			http.NotFound(w, r)
		}
	})
	dir, err := os.MkdirTemp("", "soda-web-cap-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "background.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func TestRefreshLifecycleCapabilitiesRegistersThenClears(t *testing.T) {
	db := postgresFixture(t)
	ctx := context.Background()
	socket := serveCapabilityBackground(t, "install-1")
	background := forgejo.NewServiceBackground(socket, uint32(os.Getuid()), "")
	s := &Server{Store: db, capabilities: forgejo.NewCapabilityHandshakeSource(background)}

	s.refreshLifecycleCapabilities(ctx)
	for _, event := range []string{factory.LifecycleEventIssueCreated, factory.LifecycleEventPRCreated, factory.LifecycleEventVerifiedMerge} {
		proved, err := db.CapabilityProved(ctx, event)
		if err != nil || !proved {
			t.Fatalf("event %s proved=%v err=%v after verified refresh", event, proved, err)
		}
	}

	dead := forgejo.NewServiceBackground(filepath.Join(socket, "missing.sock"), uint32(os.Getuid()), "")
	s.capabilities = forgejo.NewCapabilityHandshakeSource(dead)
	s.refreshLifecycleCapabilities(ctx)
	for _, event := range []string{factory.LifecycleEventIssueCreated, factory.LifecycleEventPRCreated} {
		proved, err := db.CapabilityProved(ctx, event)
		if err != nil || proved {
			t.Fatalf("event %s proved=%v err=%v after failed refresh", event, proved, err)
		}
	}
}
