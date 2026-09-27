//go:build linux

package terminal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestGitRelayCapability(t *testing.T) {
	called := false
	handler := gitRelay("private-capability", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-Soda-Git-Invocation") != "" {
			t.Fatal("capability forwarded")
		}
	}))
	for _, capability := range []string{"", "wrong-capability"} {
		request := httptest.NewRequest("GET", "/owner/repo.git/info/refs", nil)
		request.Header.Set("X-Soda-Git-Invocation", capability)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if called || response.Code != http.StatusForbidden {
			t.Fatal("unbound request admitted")
		}
	}
	request := httptest.NewRequest("GET", "/owner/repo.git/info/refs", nil)
	request.Header.Set("X-Soda-Git-Invocation", "private-capability")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if !called {
		t.Fatal("bound request denied")
	}
}

func TestGitUnitBindingRejectsReplacementAndWrongAccount(t *testing.T) {
	b := identity.Binding{Kind: identity.Terminal, Scope: "git", ID: strings.Repeat("a", 32), Project: "project", ContainerID: strings.Repeat("c", 64), Generation: 1, Login: "soda-tester", UID: 1000, GID: 1000, InvocationID: strings.Repeat("b", 32), CredentialRoot: "/run/soda-git/" + strings.Repeat("a", 32)}
	body := "ActiveState=active\nInvocationID=" + b.InvocationID + "\nUser=soda-tester\nGroup=1000\nControlGroup=/system.slice/soda-git-" + b.ID + ".service\n"
	if !gitBinding(b) || !gitUnitMatches([]byte(body), b) {
		t.Fatal("valid binding denied")
	}
	for _, replacement := range []string{strings.Replace(body, b.InvocationID, strings.Repeat("d", 32), 1), strings.Replace(body, "User=soda-tester", "User=root", 1), strings.Replace(body, "Group=1000", "Group=0", 1), strings.Replace(body, "ActiveState=active", "ActiveState=inactive", 1), strings.Replace(body, "/system.slice/", "/other.slice/", 1)} {
		if gitUnitMatches([]byte(replacement), b) {
			t.Fatal("foreign execution admitted")
		}
	}
}

func TestGitCommandPinsNativeHelperAndTransport(t *testing.T) {
	command := gitCommand(context.Background(), museCaller{Container: strings.Repeat("a", 64), Login: "soda-tester", GID: 1000, Home: "/home/soda-tester"}, identity.GitLaunchRequest{CWD: "/work with spaces", Remote: "origin"}, "soda-git-test.service", "http://127.0.0.1:1234/owner/repo.git", "/run/soda-git/test/gitconfig")
	args := strings.Join(command.Args, "\n")
	for _, required := range []string{"--property=KillMode=control-group", "--working-directory=/work with spaces", "--setenv=GIT_CONFIG_GLOBAL=/dev/null", "--setenv=GIT_CONFIG_KEY_0=http.proxy", "--setenv=GIT_CONFIG_VALUE_1=false", "/usr/libexec/git-core/git-remote-http", "--setenv=GIT_CONFIG_KEY_3=include.path", "--setenv=GIT_CONFIG_VALUE_3=/run/soda-git/test/gitconfig"} {
		if !strings.Contains(args, required) {
			t.Fatal(required)
		}
	}
	if strings.Contains(args, "podman\nkill") || strings.Contains(args, "auth.json") {
		t.Fatal("incorrect lifetime or credential boundary")
	}
}

func TestGitSessionMustMatchReservedNativeBinding(t *testing.T) {
	lease := identity.Lease{ID: strings.Repeat("a", 32), ProviderID: identity.Forgejo, ConnectionID: "connection", Generation: 2, ActorID: 13, ProjectID: "project", ExecutionID: strings.Repeat("b", 32), RepositoryID: 42, Kind: identity.Terminal, Deadline: time.Now().Add(time.Hour)}
	binding := identity.Binding{Kind: identity.Terminal, Scope: "git", ID: lease.ExecutionID, Project: lease.ProjectID, ContainerID: strings.Repeat("c", 64), UID: 1000, GID: 1000, Login: "soda-tester", Generation: lease.Generation, InvocationID: strings.Repeat("d", 32), CredentialRoot: "/run/soda-git/" + lease.ExecutionID}
	session := identity.GitSession{Lease: lease, Name: "Soda Tester", Email: "soda-tester@example.test"}
	session.Lease.Binding = &binding
	encoded, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &session); err != nil {
		t.Fatal(err)
	}
	if !gitSessionMatches(session, lease, binding) {
		t.Fatal("matching broker session denied")
	}
	for _, change := range []func(*identity.GitSession){
		func(s *identity.GitSession) { s.Lease.ActorID++ },
		func(s *identity.GitSession) { s.Lease.RepositoryID++ },
		func(s *identity.GitSession) { s.Lease.ConnectionID = "foreign" },
		func(s *identity.GitSession) { s.Lease.ProviderID = identity.Muse },
		func(s *identity.GitSession) { s.Lease.Deadline = s.Lease.Deadline.Add(time.Second) },
		func(s *identity.GitSession) { s.Lease.Binding = nil },
		func(s *identity.GitSession) { altered := binding; altered.UID++; s.Lease.Binding = &altered },
	} {
		mismatch := session
		change(&mismatch)
		if gitSessionMatches(mismatch, lease, binding) {
			t.Fatal("mismatched broker session admitted")
		}
	}
}

func TestGitNestedCommandUsesChildWorkingDirectoryAndUnitBoundary(t *testing.T) {
	c := museCaller{Container: strings.Repeat("a", 64), Child: strings.Repeat("b", 64), Login: "soda-tester", Home: "/root", UID: 0, GID: 0, NestedPID: 1234}
	command := gitCommand(t.Context(), c, identity.GitLaunchRequest{CWD: "/work with spaces", Remote: "origin"}, "soda-git-test.service", "http://127.0.0.1:1234/owner/repo.git", "/run/soda-git/credentials/git-test/gitconfig")
	args := strings.Join(command.Args, "\n")
	for _, required := range []string{"--uid=root", "--gid=0", "/usr/bin/nsenter", "--target=1234", "--mount", "--net", "--root", "--setuid=0", "--setgid=0", "--wdns=/work with spaces", "--setenv=GIT_CONFIG_VALUE_3=/run/soda-git/credentials/git-test/gitconfig"} {
		if !strings.Contains(args, required) {
			t.Fatal(required)
		}
	}
	if strings.Contains(args, "--working-directory=/work with spaces") || strings.Contains(args, "--wd=/work with spaces") {
		t.Fatal("parent working directory used for child Git")
	}
}

func TestGitNestedBindingPinsRegistrationAndChild(t *testing.T) {
	id := strings.Repeat("a", 32)
	registration := strings.Repeat("b", 32)
	b := identity.Binding{Kind: identity.Terminal, Scope: "git", ID: id, Project: "project", ContainerID: strings.Repeat("c", 64), ChildID: strings.Repeat("d", 64), Generation: 1, Login: "soda-tester", UID: 0, GID: 0, InvocationID: strings.Repeat("e", 32), CredentialRoot: "/run/soda-muse/nested/" + registration + "/git-" + id}
	if !gitBinding(b) || !gitStopBinding(b) {
		t.Fatal("nested binding denied")
	}
	body := "ActiveState=active\nInvocationID=" + b.InvocationID + "\nUser=root\nGroup=0\nControlGroup=/system.slice/soda-git-" + id + ".service\n"
	if !gitUnitMatches([]byte(body), b) {
		t.Fatal("nested native unit denied")
	}
	for _, path := range []string{"/run/soda-muse/nested/" + registration + "/git-" + strings.Repeat("f", 32), "/run/soda-muse/nested/../git-" + id, "/run/soda-git/" + id} {
		changed := b
		changed.CredentialRoot = path
		if gitBinding(changed) || gitStopBinding(changed) {
			t.Fatalf("foreign nested config admitted: %s", path)
		}
	}
	if gitUnitMatches([]byte(strings.Replace(body, "User=root", "User=soda-tester", 1)), b) {
		t.Fatal("wrong unit user admitted")
	}
}
