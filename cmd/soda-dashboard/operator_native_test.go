//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/web/api"
)

// Run only in a disposable native fixture. The ordinary local suite exercises
// its own UID; this gate proves the root -> UID2000 -> UID1000 peer chain.
func TestNativeOperatorUIDBridge(t *testing.T) {
	if os.Getenv("SODA_NATIVE_OPERATOR_GATE") != "1" || os.Geteuid() != 0 {
		t.Skip("requires the explicitly selected disposable root fixture")
	}
	root, err := os.MkdirTemp(os.TempDir(), "op-native-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	ipc := filepath.Join(root, "ipc")
	operator := filepath.Join(root, "operator")
	for _, dir := range []string{ipc, operator} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chown(ipc, 0, 2100); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ipc, 0o773|os.ModeSetgid|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(operator, 2000, 2000); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	start := func(role string, uid uint32) {
		t.Helper()
		cmd := exec.Command(executable, "-test.run=^TestNativeOperatorBridgeChild$", "-test.timeout=20s")
		cmd.Env = append(os.Environ(), "SODA_NATIVE_OPERATOR_ROLE="+role, "SODA_NATIVE_OPERATOR_ROOT="+root)
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: uid, Groups: []uint32{2100}}}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
		ready := make(chan bool, 1)
		go func() { scanner := bufio.NewScanner(stdout); ready <- scanner.Scan() && scanner.Text() == "READY" }()
		select {
		case ok := <-ready:
			if !ok {
				t.Fatalf("%s fixture did not start", role)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s fixture startup timed out", role)
		}
	}
	start("fountain", 1000)
	start("soda", 2000)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", filepath.Join(operator, "operator.sock"))
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodPost, "http://soda-operator/operator/extensions/state", bytes.NewBufferString(`{"package_id":"tailnet"}`))
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusConflict || string(data) != `{"error":{"code":"operation_in_progress"}}` {
		t.Fatalf("native manager response: %d %s %v", response.StatusCode, data, err)
	}
}

func TestNativeOperatorBridgeChild(t *testing.T) {
	role, root := os.Getenv("SODA_NATIVE_OPERATOR_ROLE"), os.Getenv("SODA_NATIVE_OPERATOR_ROOT")
	if role == "" {
		t.Skip("native fixture child only")
	}
	callback := filepath.Join(root, "ipc", "host.sock")
	if role == "soda" {
		product := &api.API{ExtensionManager: forgejo.NewServiceBackground(callback, 1000, "")}
		server, listener, err := operatorHTTPServer(filepath.Join(root, "operator", "operator.sock"), 0, control.NewCoordinator(nil, nil, nil), product.OperatorHandler())
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("READY")
		_ = server.Serve(listener)
		return
	}
	if role != "fountain" {
		t.Fatal("unknown native fixture role")
	}
	listener, err := net.Listen("unix", callback)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(callback, 0o660); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
		peer, err := extensions.PeerCredential(conn)
		return context.WithValue(ctx, nativePeerKey{}, err == nil && peer.UID == 2000)
	}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(nativePeerKey{}) != true || r.URL.Path != "/v1/extensions/manager/state" || r.Header.Get("Authorization") != "" {
			http.Error(w, "unexpected peer or authority", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":{"code":"operation_in_progress"}}`)
	})}
	fmt.Println("READY")
	_ = server.Serve(listener)
}

type nativePeerKey struct{}
