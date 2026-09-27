package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	native "github.com/levitateos/sodaos/internal/identity/forgejo"
)

func TestFactoryGitRequiresAuthorizingRepository(t *testing.T) {
	c, _ := gitControllerFixture(t, http.NotFoundHandler())
	for _, repositoryID := range []int64{0, 8} {
		_, err := c.AcquireGit(t.Context(), identity.GitAcquireRequest{ExpectedRepositoryID: repositoryID, Owner: "soda-tester", Repository: "repo", Acquire: identity.AcquireRequest{ProviderID: identity.Forgejo, ConnectionID: "git-account", ActorID: 1, ProjectID: "project", Kind: identity.Factory, ExecutionID: "worker", Deadline: time.Now().Add(time.Hour)}})
		if !errors.Is(err, identity.ErrDenied) {
			t.Fatalf("factory repository %d admitted: %v", repositoryID, err)
		}
	}
}

func gitControllerFixture(t *testing.T, git http.Handler) (*Controller, identity.Connection) {
	t.Helper()
	c, db, _, _ := controllerFixture(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			_, _ = fmt.Fprint(w, `{"id":1,"login":"soda-tester","full_name":"Soda Tester"}`)
		case "/api/v1/user/emails":
			_, _ = fmt.Fprint(w, `[{"email":"soda-tester@example.test","primary":true,"verified":true}]`)
		case "/api/v1/repos/soda-tester/repo", "/api/v1/repositories/7":
			_, _ = fmt.Fprint(w, `{"id":7,"name":"repo","full_name":"soda-tester/repo","owner":{"id":1,"login":"soda-tester"},"permissions":{"pull":true,"push":true}}`)
		default:
			git.ServeHTTP(w, r)
		}
	}))
	t.Cleanup(s.Close)
	p, err := native.New(native.Config{Base: s.URL, ClientID: "broker", RedirectURL: s.URL + "/callback"}, "synthetic-client-secret")
	if err != nil {
		t.Fatal(err)
	}
	c.providers[identity.Forgejo] = p
	connection := identity.Connection{ProviderID: identity.Forgejo, ID: "git-account", OwnerID: 1, Generation: 1, State: identity.Ready, Email: "soda-tester@example.test"}
	data, err := json.Marshal(native.Credential{Access: "synthetic-private", Refresh: "synthetic-refresh", Expiry: time.Now().Add(time.Hour).Unix(), UserID: 1, Scopes: "read:user write:repository"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.IdentitySaveConnection(t.Context(), connection, data); err != nil {
		t.Fatal(err)
	}
	return c, connection
}

func gitLeaseFixture(t *testing.T, c *Controller, kind, execution string) identity.Lease {
	t.Helper()
	l, err := c.AcquireGit(t.Context(), identity.GitAcquireRequest{ExpectedRepositoryID: 7, Owner: "soda-tester", Repository: "repo", Acquire: identity.AcquireRequest{ProviderID: identity.Forgejo, ConnectionID: "git-account", ActorID: 1, ProjectID: "project", Kind: kind, ExecutionID: execution, Deadline: time.Now().Add(time.Hour)}})
	if err != nil {
		t.Fatal(err)
	}
	b := identity.Binding{Scope: "git", Kind: kind, ID: execution, Project: "project", Login: "soda-tester", Generation: 1}
	session, err := c.RegisterGit(t.Context(), l.ID, b)
	if err != nil || session.Email != "soda-tester@example.test" || session.Name != "Soda Tester" {
		t.Fatal("native attribution missing", err)
	}
	return session.Lease
}

func TestGitRelayEnforcesRepositoryFactoryAndPrivateInterface(t *testing.T) {
	var forwards atomic.Int32
	c, _ := gitControllerFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwards.Add(1)
		user, password, ok := r.BasicAuth()
		if !ok || user != "soda-tester" || password != "synthetic-private" {
			t.Error("upstream did not use connected native account")
		}
		_, _ = fmt.Fprint(w, "git-response")
	}))
	human := gitLeaseFixture(t, c, identity.Terminal, "human")
	worker := gitLeaseFixture(t, c, identity.Factory, "worker")
	for _, test := range []struct {
		lease   identity.Lease
		path    string
		runtime bool
		status  int
	}{
		{human, "/soda-tester/repo.git/info/refs?service=git-upload-pack", true, 200},
		{human, "/soda-tester/other.git/info/refs?service=git-upload-pack", true, 403},
		{worker, "/soda-tester/repo.git/info/refs?service=git-receive-pack", true, 403},
		{worker, "/soda-tester/repo.git/info/refs?service=git-upload-pack", true, 200},
		{human, "/soda-tester/repo.git/info/refs?service=git-upload-pack", false, 403},
	} {
		r := httptest.NewRequest("GET", "/git/"+test.lease.ID+test.path, nil)
		r.Header.Set("Authorization", "Bearer unrelated")
		w := httptest.NewRecorder()
		c.Handler(test.runtime).ServeHTTP(w, r)
		if w.Code != test.status || strings.Contains(w.Body.String(), "synthetic-private") {
			t.Fatal("relay boundary failed", w.Code)
		}
	}
	if forwards.Load() != 2 {
		t.Fatal("denied request reached upstream")
	}
}

func TestGitLeaseEndCancelsOnlyItsOwnStream(t *testing.T) {
	ready := make(chan struct{}, 2)
	c, connection := gitControllerFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready <- struct{}{}
		<-r.Context().Done()
	}))
	a := gitLeaseFixture(t, c, identity.Terminal, "a")
	b := gitLeaseFixture(t, c, identity.Terminal, "b")
	start := func(l identity.Lease) <-chan struct{} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			r := httptest.NewRequest("GET", "/git/"+l.ID+"/soda-tester/repo.git/info/refs?service=git-upload-pack", nil)
			c.Handler(true).ServeHTTP(httptest.NewRecorder(), r)
		}()
		return done
	}
	da, db := start(a), start(b)
	for range 2 {
		select {
		case <-ready:
		case <-time.After(3 * time.Second):
			t.Fatal("stream did not start")
		}
	}
	if err := c.EndLease(t.Context(), 1, a.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-da:
	case <-time.After(3 * time.Second):
		t.Fatal("revoked stream did not stop")
	}
	select {
	case <-db:
		t.Fatal("sibling stream stopped")
	default:
	}
	current, err := c.store.IdentityConnection(t.Context(), connection.ID)
	if err != nil || current.State != identity.Ready {
		t.Fatal("normal Git retirement changed custody", err)
	}
	if err := c.EndLease(t.Context(), 1, b.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-db:
	case <-time.After(3 * time.Second):
		t.Fatal("sibling cleanup did not complete")
	}
}

func TestControllerCloseCancelsGitStream(t *testing.T) {
	ready := make(chan struct{})
	c, _ := gitControllerFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(ready)
		<-r.Context().Done()
	}))
	l := gitLeaseFixture(t, c, identity.Terminal, "closing")
	done := make(chan struct{})
	go func() {
		defer close(done)
		r := httptest.NewRequest("GET", "/git/"+l.ID+"/soda-tester/repo.git/info/refs?service=git-upload-pack", nil)
		c.Handler(true).ServeHTTP(httptest.NewRecorder(), r)
	}()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not start")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown left Git stream running")
	}
}
