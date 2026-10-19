//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

func TestOperatorExtensionManagerBridge(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "op-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	callback := filepath.Join(root, "fountain.sock")
	listener, err := net.Listen("unix", callback)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	fountain := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/extensions/manager/state" {
			t.Error("unexpected callback path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked)
		_, _ = io.WriteString(w, `{"error":{"code":"maintenance"}}`)
	})}
	go func() { _ = fountain.Serve(listener) }()
	t.Cleanup(func() { _ = fountain.Close() })
	product := &api.API{ExtensionManager: forgejo.NewServiceBackground(callback, uint32(os.Geteuid()), "")}
	for _, foreign := range []bool{false, true} {
		uid := uint32(os.Geteuid())
		if foreign {
			uid++
		}
		socket := filepath.Join(root, fmt.Sprintf("operator-%t.sock", foreign))
		server, ln, err := operatorHTTPServer(socket, uid, control.NewCoordinator(nil, nil, nil), product.OperatorHandler())
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = server.Serve(ln) }()
		t.Cleanup(func() { _ = server.Close() })
		transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}}
		t.Cleanup(transport.CloseIdleConnections)
		client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
		req, _ := http.NewRequest(http.MethodPost, "http://soda-operator/operator/extensions/state", bytes.NewBufferString(`{"package_id":"tailnet"}`))
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if foreign {
			if response != nil {
				_ = response.Body.Close()
			}
			if err == nil || calls.Load() != 1 {
				t.Fatal("foreign operator peer reached manager")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil || response.StatusCode != http.StatusLocked || string(data) != `{"error":{"code":"maintenance"}}` || calls.Load() != 1 {
			t.Fatalf("manager rejection changed across operator bridge: %d %s %v", response.StatusCode, data, err)
		}
	}
}

func operatorRoundTrip(t *testing.T, uid uint32) (*http.Response, error) {
	t.Helper()
	db := postgresFixture(t)
	coordinator := control.NewCoordinator(db, nil, nil)
	socket := filepath.Join(t.TempDir(), "operator.sock")
	server, listener, err := operatorHTTPServer(socket, uid, coordinator, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close(); _ = server.Close() })
	go func() { _ = server.Serve(listener) }()
	// A bounded client turns a server-side deadlock into a fast failure
	// instead of hanging the whole package until the go test timeout.
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	req, err := http.NewRequest(http.MethodPost, "http://soda-operator/operator/factory", bytes.NewReader([]byte(`{"type":"status"}`)))
	if err != nil {
		t.Fatal(err)
	}
	return client.Do(req)
}

func TestOperatorSocketAdmitsConfiguredPeer(t *testing.T) {
	res, err := operatorRoundTrip(t, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatal("configured peer refused", res.StatusCode)
	}
}

func TestOperatorSocketRebindsStalePath(t *testing.T) {
	db := postgresFixture(t)
	coordinator := control.NewCoordinator(db, nil, nil)
	socket := filepath.Join(t.TempDir(), "operator.sock")
	first, firstListener, err := operatorHTTPServer(socket, uint32(os.Geteuid()), coordinator, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = firstListener.Close()
	_ = first.Close()
	second, secondListener, err := operatorHTTPServer(socket, uint32(os.Geteuid()), coordinator, nil)
	if err != nil {
		t.Fatal("restart did not rebind stale socket path:", err)
	}
	t.Cleanup(func() { _ = secondListener.Close(); _ = second.Close() })
}

func TestOperatorSocketRefusesForeignPeer(t *testing.T) {
	foreign := uint32(os.Geteuid()) + 1
	if _, err := operatorRoundTrip(t, foreign); err == nil {
		t.Fatal("foreign peer admitted")
	}
}

func TestOperatorProjectInventory(t *testing.T) {
	db := postgresFixture(t)
	ctx := context.Background()
	if err := db.UpsertUser(ctx, store.User{ID: 42, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	add := func(number int) string {
		t.Helper()
		id := fmt.Sprintf("p%024x", number)
		if err := db.CreateProject(ctx, store.Project{ID: id, OwnerID: 42, RepositoryID: int64(number), Name: "fixture"}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	first, second := add(1), add(2)
	if err := db.MarkReady(ctx, first, "192.0.2.2"); err != nil {
		t.Fatal(err)
	}
	coordinator := control.NewCoordinator(db, nil, nil)
	productAPI := &api.API{Store: db}
	socket := filepath.Join(t.TempDir(), "operator.sock")
	server, listener, err := operatorHTTPServer(socket, uint32(os.Geteuid()), coordinator, productAPI.OperatorHandler())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close(); _ = server.Close() })
	go func() { _ = server.Serve(listener) }()
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	read := func(method, suffix string, status int) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, "http://soda-operator/operator/projects"+suffix, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = res.Body.Close() })
		if res.StatusCode != status {
			t.Fatalf("%s inventory status %d, want %d", method, res.StatusCode, status)
		}
		return res
	}
	res := read(http.MethodGet, "", http.StatusOK)
	var inventory struct {
		Projects []struct {
			ID    string `json:"id"`
			Ready bool   `json:"ready"`
		} `json:"projects"`
		FactoryWork     int `json:"factory_work"`
		PreparationWork int `json:"preparation_work"`
	}
	if err := json.NewDecoder(res.Body).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Projects) != 2 || inventory.Projects[0].ID != first || !inventory.Projects[0].Ready || inventory.Projects[1].ID != second || inventory.Projects[1].Ready {
		t.Fatalf("durable enrollment was filtered by nonexistent native state: %+v", inventory.Projects)
	}
	read(http.MethodPost, "", http.StatusMethodNotAllowed)
	read(http.MethodGet, "?after="+first, http.StatusBadRequest)
	for number := 3; number <= 129; number++ {
		add(number)
	}
	read(http.MethodGet, "", http.StatusServiceUnavailable)
}
