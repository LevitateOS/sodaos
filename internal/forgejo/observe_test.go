package forgejo

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func observationREST(t *testing.T, userID int64, repo Repository, issues []ListedIssue) *Client {
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
		case strings.HasPrefix(r.URL.Path, "/api/v1/repositories/"):
			_ = json.NewEncoder(w).Encode(repo)
		case strings.HasSuffix(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode(issues)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return New(server.URL)
}

func TestServiceObserverListsIssuesOldestFirst(t *testing.T) {
	repo := Repository{ID: 7, Name: "n", FullName: "o/n", Owner: User{ID: 3, Login: "o"}}
	rest := observationREST(t, 11, repo, []ListedIssue{{Index: 3}, {Index: 9}})
	observer := NewServiceObserver("/run/soda/background.sock", 1001, observationCredential(t, "test-pat"), rest)
	indexes, hasMore, err := observer.ListIssuesPage(context.Background(), 7, 1)
	if err != nil || hasMore || len(indexes) != 2 || indexes[0] != 3 || indexes[1] != 9 {
		t.Fatal("issue listing wrong:", indexes, hasMore, err)
	}
}

func TestServiceObserverListBounds(t *testing.T) {
	repo := Repository{ID: 7, Name: "n", FullName: "o/n", Owner: User{ID: 3, Login: "o"}}
	full := make([]ListedIssue, 0, ObservationIssuePageSize)
	for i := int64(1); i <= ObservationIssuePageSize; i++ {
		full = append(full, ListedIssue{Index: i})
	}
	rest := observationREST(t, 11, repo, full)
	observer := NewServiceObserver("/run/soda/background.sock", 1001, observationCredential(t, "test-pat"), rest)
	indexes, hasMore, err := observer.ListIssuesPage(context.Background(), 7, 1)
	if err != nil || !hasMore || len(indexes) != ObservationIssuePageSize {
		t.Fatal("full page must report more:", len(indexes), hasMore, err)
	}
	over := append(append([]ListedIssue{}, full...), ListedIssue{Index: 999})
	restOver := observationREST(t, 11, repo, over)
	observerOver := NewServiceObserver("/run/soda/background.sock", 1001, observationCredential(t, "test-pat"), restOver)
	if _, _, err = observerOver.ListIssuesPage(context.Background(), 7, 1); err == nil {
		t.Fatal("oversized page accepted")
	}
	dupes := observationREST(t, 11, repo, []ListedIssue{{Index: 3}, {Index: 3}})
	observerDupes := NewServiceObserver("/run/soda/background.sock", 1001, observationCredential(t, "test-pat"), dupes)
	if _, _, err = observerDupes.ListIssuesPage(context.Background(), 7, 1); err == nil {
		t.Fatal("duplicate index accepted")
	}
	badRepo := Repository{ID: 8, Name: "n", FullName: "o/n", Owner: User{ID: 3, Login: "o"}}
	restBad := observationREST(t, 11, badRepo, nil)
	observerBad := NewServiceObserver("/run/soda/background.sock", 1001, observationCredential(t, "test-pat"), restBad)
	if _, _, err = observerBad.ListIssuesPage(context.Background(), 7, 1); err == nil {
		t.Fatal("mismatched repository accepted")
	}
	if _, _, err = observer.ListIssuesPage(context.Background(), 7, 0); err == nil {
		t.Fatal("page zero accepted")
	}
	unconfigured := NewServiceObserver("", 1001, "", nil)
	if _, _, err = unconfigured.ListIssuesPage(context.Background(), 7, 1); err == nil {
		t.Fatal("unconfigured observer listed")
	}
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
	socket := filepath.Join(t.TempDir(), "background.sock")
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
	repo := Repository{ID: 7, Name: "n", FullName: "o/n", Owner: User{ID: 3, Login: "o"}}
	rest := observationREST(t, 11, repo, nil)
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

func TestServiceObserverBootstrapFailure(t *testing.T) {
	repo := Repository{ID: 7, Name: "n", FullName: "o/n", Owner: User{ID: 3, Login: "o"}}
	rest := observationREST(t, 11, repo, nil)
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
