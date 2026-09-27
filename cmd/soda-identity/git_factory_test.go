package main

import (
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestFactoryGitBaseMustMatchCurrentWorkerActorAndRepository(t *testing.T) {
	container := strings.Repeat("a", 64)
	base := identity.Lease{Kind: identity.Factory, ActorID: 1, ProjectID: "p" + strings.Repeat("b", 24), RepositoryID: 7, ExecutionID: strings.Repeat("c", 32), Deadline: time.Now().Add(time.Hour), Binding: &identity.Binding{Kind: identity.Factory, ID: container}}
	git := identity.Lease{Kind: identity.Factory, ActorID: 1, ProjectID: base.ProjectID, RepositoryID: 7, Binding: &identity.Binding{ContainerID: container}}
	if !factoryGitBaseMatches(base, git) {
		t.Fatal("exact worker denied")
	}
	for _, change := range []func(*identity.Lease){
		func(l *identity.Lease) { l.ActorID++ },
		func(l *identity.Lease) { l.ProjectID = "other" },
		func(l *identity.Lease) { l.RepositoryID++ },
		func(l *identity.Lease) { l.Binding.ContainerID = strings.Repeat("d", 64) },
	} {
		other := git
		binding := *git.Binding
		other.Binding = &binding
		change(&other)
		if factoryGitBaseMatches(base, other) {
			t.Fatal("foreign worker authority accepted")
		}
	}
	base.Deadline = time.Now().Add(-time.Second)
	if factoryGitBaseMatches(base, git) {
		t.Fatal("expired worker accepted")
	}
}

func TestFactoryGitSocketRequiresDedicatedInterface(t *testing.T) {
	c := settings{AdminSocket: "/run/soda/private/admin.sock", RuntimeSocket: "/run/soda/private/runtime.sock", MuseWorkerSocket: "/run/soda-muse-worker-interface/launch.sock"}
	for _, path := range []string{"relative/launch.sock", "/run/soda/private/launch.sock", c.MuseWorkerSocket, "/run/soda-git-worker-interface/other.sock"} {
		c.GitWorkerSocket = path
		if validateFactoryGitSettings(c) == nil {
			t.Fatal("invalid worker interface accepted", path)
		}
	}
	c.GitWorkerSocket = "/run/soda-git-worker-interface/launch.sock"
	if err := validateFactoryGitSettings(c); err != nil {
		t.Fatal(err)
	}
}
