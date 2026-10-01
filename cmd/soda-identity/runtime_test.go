package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
)

func testHostServer(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) *host.Client {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "host.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: http.HandlerFunc(handle)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return host.NewClient(socket)
}

func TestRuntimeDelegatesBothKindsToHost(t *testing.T) {
	var paths []string
	client := testHostServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var in identity.DeliveryWire
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		out := identity.DeliveryWire{Lease: in.Lease, Credential: []byte(`{"returned":true}`)}
		_ = json.NewEncoder(w).Encode(out)
	})
	runtime := nativeRuntime{Host: client}
	ctx := context.Background()
	for _, kind := range []string{identity.Terminal, identity.Factory} {
		lease := identity.Lease{Kind: kind, ExecutionID: "execution", Binding: &identity.Binding{Kind: kind, ID: "execution"}}
		if err := runtime.Validate(ctx, lease); err != nil {
			t.Fatal(kind, err)
		}
		if err := runtime.Stop(ctx, lease); err != nil {
			t.Fatal(kind, err)
		}
		data, err := runtime.Finish(ctx, lease)
		if err != nil || !bytes.Equal(data, []byte(`{"returned":true}`)) {
			t.Fatal(kind, data, err)
		}
	}
	want := []string{"/identity/validate", "/identity/stop", "/identity/finish", "/identity/validate", "/identity/stop", "/identity/finish"}
	if len(paths) != len(want) {
		t.Fatal("host callbacks missing", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatal("host callbacks out of order", paths)
		}
	}
}

func TestRuntimeRefusesUnboundLease(t *testing.T) {
	client := testHostServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unbound lease reached the host")
	})
	runtime := nativeRuntime{Host: client}
	ctx := context.Background()
	lease := identity.Lease{Kind: identity.Factory, ExecutionID: "execution"}
	if err := runtime.Validate(ctx, lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unbound lease validated", err)
	}
	if _, err := runtime.Finish(ctx, lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unbound lease finished", err)
	}
	if err := runtime.Stop(ctx, lease); err != nil {
		t.Fatal("unbound stop failed", err)
	}
}
