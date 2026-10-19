package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestShutdownServersClosesConnectionsAndReturnsTimeout(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte("finished"))
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, requestErr := http.Post("http://"+listener.Addr().String(), "text/plain", strings.NewReader("hold"))
		if requestErr == nil {
			_ = response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("request did not reach held handler")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = shutdownServers(ctx, server)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown timeout was not returned: %v", err)
	}
	close(release)
	select {
	case <-requestDone:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not finish after handler release")
	}
	if err := <-serveDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}
}
