package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
)

func observationCredential(t *testing.T, secret string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "background-pat")
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func observationREST(t *testing.T, userID int64) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token test-pat" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/user":
			_ = json.NewEncoder(w).Encode(User{ID: userID, Login: "soda-tester"})
		case r.URL.Path == "/api/v1/repositories/7":
			_ = json.NewEncoder(w).Encode(Repository{ID: 7, Name: "n", FullName: "o/n", Owner: User{ID: 3, Login: "o"}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return New(server.URL)
}

// fakeBackgroundServer speaks the SDK bootstrap/revision/snapshot paths
// over a Unix socket so observer wiring is exercised without Fountain.
type fakeBackgroundServer struct {
	t          *testing.T
	admission  string
	bootstraps int
	token      string
}

func (f *fakeBackgroundServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case extensions.BackgroundBootstrapPath:
			f.bootstraps++
			_ = json.NewEncoder(w).Encode(map[string]string{
				"admission": f.admission, "installation_id": "install-1",
			})
		case extensions.BackgroundRevisionPath:
			if r.Header.Get(extensions.AdmissionHeader) != f.admission {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"revision": 12, "idle": true})
		case extensions.BackgroundSnapshotPath:
			if r.Header.Get(extensions.AdmissionHeader) != f.admission {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			var in struct {
				extensions.SnapshotRequest
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				http.Error(w, "bad", http.StatusBadRequest)
				return
			}
			f.token = in.Token
			_ = json.NewEncoder(w).Encode(map[string]any{"repository_id": in.RepositoryID})
		default:
			http.NotFound(w, r)
		}
	})
}

func serveBackgroundSocket(t *testing.T, fake *fakeBackgroundServer) string {
	t.Helper()
	socket := shortSocketPath(t, "background.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: fake.handler()}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func TestServiceObserverBootstrapsOnce(t *testing.T) {
	admission := strings.Repeat("a", 43)
	fake := &fakeBackgroundServer{t: t, admission: admission}
	socket := serveBackgroundSocket(t, fake)
	rest := observationREST(t, 11)
	observer := NewServiceObserver(socket, uint32(os.Getuid()), observationCredential(t, "test-pat"), rest)
	ctx := context.Background()
	observation, err := observer.SnapshotReader().ReadNativeRevision(ctx)
	if err != nil || observation.Revision != 12 || !observation.Idle {
		t.Fatal("revision observation wrong:", observation, err)
	}
	if _, err = observer.SnapshotReader().ReadNativeRevision(ctx); err != nil {
		t.Fatal(err)
	}
	if fake.bootstraps != 1 {
		t.Fatal("client bootstrapped again:", fake.bootstraps)
	}
	snapshot, err := observer.SnapshotReader().ReadSnapshot(ctx, extensions.CredentialFile("/nonexistent"),
		SnapshotRequest{RepositoryID: "7", Families: []SnapshotFamily{FamilyIssue}, IssueIndex: "3"})
	if err != nil || snapshot.RepositoryID != "7" {
		t.Fatal("snapshot read wrong:", snapshot, err)
	}
	if fake.token != "test-pat" {
		t.Fatal("snapshot did not present the service credential")
	}
}

func TestServiceObserverShortRevisionCallerDoesNotWaitForActorLookup(t *testing.T) {
	currentStarted := make(chan struct{})
	releaseCurrent := make(chan struct{})
	var releaseOnce sync.Once
	var currentStartedOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseCurrent) }) }
	defer release()
	var currentCalls atomic.Int32

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" {
			http.NotFound(w, r)
			return
		}
		currentCalls.Add(1)
		currentStartedOnce.Do(func() { close(currentStarted) })
		<-releaseCurrent
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{ID: 11, Login: "soda-tester"})
	}))
	t.Cleanup(rest.Close)
	fake := &fakeBackgroundServer{admission: strings.Repeat("a", 43)}
	observer := NewServiceObserver(serveBackgroundSocket(t, fake), uint32(os.Getuid()),
		observationCredential(t, "test-pat"), New(rest.URL))

	firstDone := make(chan error, 1)
	go func() {
		_, err := observer.SnapshotReader().ReadSnapshot(context.Background(), observer.Credential(), SnapshotRequest{
			RepositoryID: "7", IssueIndex: "3", Families: []SnapshotFamily{FamilyIssue}, Limit: SnapshotPageLimit,
		})
		firstDone <- err
	}()
	select {
	case <-currentStarted:
	case <-time.After(time.Second):
		t.Fatal("cold actor lookup did not reach Forgejo")
	}

	shortCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	shortSnapshotDone := make(chan error, 1)
	go func() {
		_, err := observer.SnapshotReader().ReadSnapshot(shortCtx, observer.Credential(), SnapshotRequest{
			RepositoryID: "7", IssueIndex: "3", Families: []SnapshotFamily{FamilyIssue}, Limit: SnapshotPageLimit,
		})
		shortSnapshotDone <- err
	}()
	revisionCtx, cancelRevision := context.WithTimeout(context.Background(), time.Second)
	defer cancelRevision()
	revisionDone := make(chan error, 1)
	go func() {
		_, err := observer.SnapshotReader().ReadNativeRevision(revisionCtx)
		revisionDone <- err
	}()
	select {
	case err := <-revisionDone:
		if err != nil {
			t.Fatalf("revision call could not proceed during actor lookup: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("revision caller remained blocked behind the actor lookup")
	}
	select {
	case err := <-shortSnapshotDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waiting snapshot did not stop at its own deadline: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("short snapshot waiter did not honor its deadline before Current was released")
	}
	if calls := currentCalls.Load(); calls != 1 {
		t.Fatalf("concurrent observer started %d actor lookups, want one", calls)
	}

	release()
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("cold snapshot read failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cold snapshot read did not finish after Current was released")
	}
}

func TestServiceObserverBootstrapFailure(t *testing.T) {
	rest := observationREST(t, 11)
	observer := NewServiceObserver(filepath.Join(t.TempDir(), "missing.sock"), uint32(os.Getuid()), observationCredential(t, "test-pat"), rest)
	if _, err := observer.SnapshotReader().ReadNativeRevision(context.Background()); err == nil {
		t.Fatal("missing socket observed")
	}
	unconfigured := NewServiceObserver("", 1001, "", rest)
	if _, err := unconfigured.SnapshotReader().ReadNativeRevision(context.Background()); err == nil {
		t.Fatal("unconfigured observer read")
	}
	if _, err := observer.SnapshotReader().ReadSnapshot(context.Background(), "", SnapshotRequest{}); err == nil {
		t.Fatal("invalid snapshot request read")
	}
}
