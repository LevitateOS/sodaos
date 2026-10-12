package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

type scriptedOperationsClient struct {
	extensions.BackgroundClient
	answer extensions.OperationsHandshake
	err    error
}

func (f *scriptedOperationsClient) ReadOperations(context.Context) (extensions.OperationsHandshake, error) {
	return f.answer, f.err
}

func TestCapabilityHandshakeMapsNativeAnswer(t *testing.T) {
	source := NewCapabilityHandshakeSource(&scriptedOperationsClient{
		answer: extensions.OperationsHandshake{Installation: "install-1", Generation: "ops-abc", Kinds: []string{"issue.lifecycle.apply"}},
	})
	handshake, err := source.Handshake(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if handshake.Installation != "install-1" || handshake.Generation != "ops-abc" ||
		len(handshake.Kinds) != 1 || handshake.Kinds[0] != "issue.lifecycle.apply" {
		t.Fatalf("mapped handshake wrong: %+v", handshake)
	}
	installation, generation, lastErr, at := source.LastHandshake()
	if installation != "install-1" || generation != "ops-abc" || lastErr != nil || at.IsZero() {
		t.Fatalf("recorded outcome wrong: %q %q %v %v", installation, generation, lastErr, at)
	}
}

func TestCapabilityHandshakeRecordsFailure(t *testing.T) {
	failure := errors.New("native unreachable")
	source := NewCapabilityHandshakeSource(&scriptedOperationsClient{err: failure})
	if _, err := source.Handshake(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("handshake error = %v", err)
	}
	installation, generation, lastErr, at := source.LastHandshake()
	if installation != "" || generation != "" || !errors.Is(lastErr, failure) || at.IsZero() {
		t.Fatalf("recorded failure wrong: %q %q %v %v", installation, generation, lastErr, at)
	}
}

func serveOperationsBackground(t *testing.T, installation string) (string, *int) {
	t.Helper()
	const admission = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case extensions.BackgroundBootstrapPath:
			_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "install-1"})
		case extensions.BackgroundOperationsPath:
			calls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"installation": installation, "generation": "ops-abc",
				"kinds": []string{"issue.lifecycle.apply", "pull_request.review.request", "issue.comment.publish"},
			})
		default:
			http.NotFound(w, r)
		}
	})
	socket := shortSocketPath(t, "operations.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket, &calls
}

func TestServiceBackgroundReadOperationsRoundTrip(t *testing.T) {
	socket, calls := serveOperationsBackground(t, "install-1")
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	hooked := 0
	background.OnBootstrap = func() { hooked++ }
	handshake, err := background.ReadOperations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if handshake.Installation != "install-1" || handshake.Generation != "ops-abc" || len(handshake.Kinds) != 3 {
		t.Fatalf("handshake wrong: %+v", handshake)
	}
	if *calls != 1 {
		t.Fatalf("operations path called %d times", *calls)
	}
	if hooked != 1 {
		t.Fatalf("bootstrap hook fired %d times", hooked)
	}
}

func TestServiceBackgroundReadOperationsRefusesForeignInstallation(t *testing.T) {
	socket, _ := serveOperationsBackground(t, "install-2")
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	if _, err := background.ReadOperations(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid background response") {
		t.Fatalf("foreign installation error = %v", err)
	}
}
