package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func operatorServer(t *testing.T, code int, body string) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "operator.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operator/factory" || r.Method != http.MethodPost {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var envelope map[string]string
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			http.Error(w, "bad envelope", http.StatusBadRequest)
			return
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func TestOperatorDispatchSendsEnvelope(t *testing.T) {
	socket := operatorServer(t, http.StatusOK, `{"runs":[]}`)
	var out bytes.Buffer
	if err := run(context.Background(), []string{"--socket", socket, "status"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "{\"runs\":[]}\n" {
		t.Fatal("status response lost", out.String())
	}
}

func TestOperatorDispatchMapsResponses(t *testing.T) {
	socket := operatorServer(t, http.StatusConflict, "conflict")
	var out bytes.Buffer
	err := run(context.Background(), []string{"--socket", socket, "--command", strings.Repeat("a", 32), "stop", strings.Repeat("b", 32)}, &out)
	if err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatal("conflict unmapped", err)
	}
}

func TestOperatorDispatchRejectsLocalMisuse(t *testing.T) {
	for _, args := range [][]string{
		{"status"},
		{"--socket", "/tmp/x.sock"},
		{"--socket", "/tmp/x.sock", "run", "attempt"},
		{"--socket", "/tmp/x.sock", "--command", "id", "status"},
		{"--socket", "/tmp/x.sock", "stop", "run"},
		{"--socket", "/tmp/x.sock", "--command", "id", "reconcile", "run"},
	} {
		var out bytes.Buffer
		if err := run(context.Background(), args, &out); err == nil {
			t.Fatalf("misuse accepted: %v", args)
		}
	}
}
