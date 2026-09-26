package control

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host/workspace"
	"github.com/levitateos/sodaos/internal/store"
)

type executorFunc func(context.Context, []byte, string, ...string) ([]byte, error)

func (f executorFunc) Run(ctx context.Context, b []byte, n string, a ...string) ([]byte, error) {
	return f(ctx, b, n, a...)
}

func testAttempt(t *testing.T, s *store.Store) factory.Attempt {
	t.Helper()
	a, err := factory.New(factory.WorkItem{RepositoryID: 1, Issue: 1, HumanID: 1, Objective: "change", BaseSHA: strings.Repeat("a", 40), PolicySHA: strings.Repeat("b", 64)}, "delivery", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	a, _, err = s.AdmitFactory(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestCancelWithdrawsAuthorityAndRemovesRecordedWorker(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(root, "execution.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	a := testAttempt(t, s)
	copy := a
	r, err := copy.BeginRun(factory.Implementation, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Image = "sha256:" + strings.Repeat("c", 64)
	r.Harness = "test"
	r.Model = "test"
	r.Resources = []factory.Resource{{Kind: "workspace", Name: factory.ResourceName(r.ID, "workspace"), ID: strings.Repeat("d", 64)}}
	if err = s.StartFactoryRun(t.Context(), &a, r); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.Method == "GET" && strings.HasSuffix(q.URL.Path, "/labels") {
			if err := json.NewEncoder(w).Encode([]forgejo.WorkLabel{{ID: 10, Name: "soda:cancelled"}}); err != nil {
				t.Error(err)
			}
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	var removed []string
	runtime := &workspace.Runtime{Exec: executorFunc(func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
		if args[0] == "inspect" {
			return []byte("false"), nil
		}
		switch args[1] {
		case "exists":
			return nil, nil
		case "inspect":
			return json.Marshal(map[string]string{"id": r.Resources[0].ID, "name": r.Resources[0].Name, "owner": r.ID})
		default:
			removed = append(removed, strings.Join(args, " "))
			return nil, nil
		}
	})}
	c := Controller{Config: Config{Root: root}, Store: s, Forgejo: forgejo.New(server.URL), Repository: forgejo.Repository{ID: 1, Name: "target", Owner: forgejo.User{ID: 1, Login: "soda-tester"}}, Workspace: runtime}
	if err = c.Cancel(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.FactoryAttempt(t.Context(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != factory.Cancelled || !got.CleanupComplete || got.Authority(time.Now()) == nil {
		t.Fatal("cancelled attempt retained authority or resources")
	}
	if len(removed) != 2 || removed[0] != "stop --time 3 "+r.Resources[0].ID {
		t.Fatal("worker process tree was not stopped by ID")
	}
}

func TestCIRequiresOneNativePREvaluation(t *testing.T) {
	run := forgejo.ActionRun{ID: 1, Commit: strings.Repeat("a", 40), Event: "pull_request", Status: "success"}
	evidence, done, err := ciResult([]forgejo.ActionRun{run})
	if err != nil || !done || !evidence.Passed || evidence.Commit != run.Commit {
		t.Fatal("native CI result rejected")
	}
	if _, _, err = ciResult([]forgejo.ActionRun{run, run}); err == nil {
		t.Fatal("duplicate evaluation accepted")
	}
	run.Event = "push"
	if _, _, err = ciResult([]forgejo.ActionRun{run}); err == nil {
		t.Fatal("unassigned event accepted")
	}
}

func TestChangedCandidateCannotRetainReviewTarget(t *testing.T) {
	a := factory.Attempt{ID: factory.NewID(), Candidate: strings.Repeat("a", 40), Work: factory.WorkItem{RepositoryID: 1, BaseSHA: strings.Repeat("b", 40)}}
	c := Controller{Implementer: forgejo.User{ID: 2}, Repository: forgejo.Repository{DefaultBranch: "main"}}
	p := forgejo.Pull{State: "open", User: forgejo.User{ID: 2}, Head: forgejo.PullBranch{Ref: "soda/factory/" + a.ID, SHA: strings.Repeat("c", 40), Repo: forgejo.Repository{ID: 1}}, Base: forgejo.PullBranch{Ref: "main", SHA: a.Work.BaseSHA, Repo: forgejo.Repository{ID: 1}}}
	if err := c.validatePull(p, a); err == nil {
		t.Fatal("stale candidate accepted")
	}
}
