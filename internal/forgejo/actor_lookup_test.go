package forgejo

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type actorLookupRequest struct {
	proto, auth string
}

func TestActorCallersUsePrivateHTTP1LookupAndKeepSharedPool(t *testing.T) {
	const token = "actor-lookup-test-secret"
	const actorID = int64(17)
	ctx := context.Background()

	var mu sync.Mutex
	connections := 0
	var requests []actorLookupRequest
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		requests = append(requests, actorLookupRequest{proto: r.Proto, auth: r.Header.Get("Authorization")})
		mu.Unlock()
		if r.Header.Get("Authorization") != "token "+token {
			http.Error(w, "wrong credential", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{ID: actorID, Login: "soda-tester"})
	}))
	server.EnableHTTP2 = true
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			mu.Lock()
			connections++
			mu.Unlock()
		}
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	sharedTransport := server.Client().Transport.(*http.Transport)
	t.Cleanup(sharedTransport.CloseIdleConnections)
	rest := &Client{Base: server.URL, HTTP: &http.Client{Transport: sharedTransport}}
	seed, err := rest.Current(ctx, token)
	if err != nil || seed.ID != actorID {
		t.Fatalf("seed shared connection: user=%+v err=%v", seed, err)
	}
	mu.Lock()
	seedProtocol := requests[0].proto
	mu.Unlock()
	if seedProtocol != "HTTP/2.0" {
		t.Fatalf("shared connection protocol = %q, want HTTP/2.0", seedProtocol)
	}

	credential := filepath.Join(t.TempDir(), "actor-token")
	if err := os.WriteFile(credential, []byte(token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	background := &ServiceBackground{}
	publisher := NewPublisher(background, rest, "https://internal.invalid", t.TempDir(), credential)
	callers := []struct {
		name string
		call func() error
	}{
		{"checks", func() error { return NewCheckAssessor(background, rest, credential).checkActor(ctx, actorID) }},
		{"merge", func() error { return NewMerger(background, rest, credential).checkActor(ctx, actorID) }},
		{"review", func() error { return NewReviewer(background, rest, credential).checkActor(ctx, actorID) }},
		{"publish", func() error { return publisher.checkActor(ctx, actorID) }},
	}
	for i, caller := range callers {
		caller := caller
		t.Run(caller.name, func(t *testing.T) {
			beforeConnections := connectionCount(t, &mu, &connections)
			beforeRequests := requestCount(t, &mu, &requests)
			if err := caller.call(); err != nil {
				t.Fatalf("actor admission: %v", err)
			}
			if got := connectionCount(t, &mu, &connections); got != beforeConnections+1 {
				t.Fatalf("lookup opened %d connections, want one fresh connection", got-beforeConnections)
			}
			got := recordedRequests(t, &mu, &requests)
			if len(got) != beforeRequests+1 {
				t.Fatalf("lookup request count = %d, want %d: %+v", len(got)-beforeRequests, 1, got)
			}
			lookup := got[beforeRequests]
			if lookup.proto != "HTTP/1.1" || lookup.auth != "token "+token {
				t.Fatalf("lookup request did not use one fresh HTTP/1.1 connection with the exact credential: %+v", lookup)
			}
			if i == len(callers)-1 {
				publisher.mu.Lock()
				cachedID, cachedLogin := publisher.actorID, publisher.login
				publisher.mu.Unlock()
				if cachedID != actorID || cachedLogin != "soda-tester" {
					t.Fatalf("publisher actor binding was not cached: id=%d login=%q", cachedID, cachedLogin)
				}
			}
		})
	}

	beforeConnections := connectionCount(t, &mu, &connections)
	beforeRequests := requestCount(t, &mu, &requests)
	shared, err := rest.Current(ctx, token)
	if err != nil || shared.ID != actorID {
		t.Fatalf("shared pool stopped working after actor lookups: user=%+v err=%v", shared, err)
	}
	if got := connectionCount(t, &mu, &connections); got != beforeConnections {
		t.Fatalf("shared pool opened a replacement connection: before=%d after=%d", beforeConnections, got)
	}
	got := recordedRequests(t, &mu, &requests)
	if len(got) != beforeRequests+1 {
		t.Fatalf("shared pooled request count changed: %+v", got)
	}
	sharedRequest := got[beforeRequests]
	if sharedRequest.proto != "HTTP/2.0" || sharedRequest.auth != "token "+token {
		t.Fatalf("shared pooled request changed after actor lookups: %+v", sharedRequest)
	}
}

func TestLoadActorRefusesRedirectWithUnconfiguredSharedClient(t *testing.T) {
	redirected := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			http.Redirect(w, r, "/capture", http.StatusFound)
			return
		}
		redirected++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := New(server.URL)
	client.HTTP.CheckRedirect = nil
	if _, err := loadActor(context.Background(), client, "redirect-test-secret"); err == nil {
		t.Fatal("redirect response was accepted as an actor lookup")
	}
	if redirected != 0 {
		t.Fatalf("actor credential followed a redirect %d times", redirected)
	}
}

func connectionCount(t *testing.T, mu *sync.Mutex, count *int) int {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	return *count
}

func requestCount(t *testing.T, mu *sync.Mutex, requests *[]actorLookupRequest) int {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	return len(*requests)
}

func recordedRequests(t *testing.T, mu *sync.Mutex, requests *[]actorLookupRequest) []actorLookupRequest {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	return append([]actorLookupRequest(nil), (*requests)...)
}
