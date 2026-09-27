//go:build linux

package terminal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	b := identity.Binding{Kind: identity.Terminal, Scope: "git", ID: strings.Repeat("a", 32), Project: "project", ContainerID: strings.Repeat("c", 64), Generation: 1, Login: "soda-tester", UID: 1000, GID: 1000, InvocationID: strings.Repeat("b", 32)}
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
