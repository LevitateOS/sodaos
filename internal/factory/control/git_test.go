package control

import (
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host/publish"
	"github.com/levitateos/sodaos/internal/host/workspace"
)

func TestFactoryGitRemoteRequiresAdmittedRepository(t *testing.T) {
	c := Controller{
		Config:     Config{Workspace: workspace.Config{GitSocket: "/run/soda-git-interface/launch.sock"}},
		Repository: forgejo.Repository{ID: 7},
		Publisher:  publish.Config{Remote: "https://forge.example.test/soda-tester/repo.git"},
		Workspace:  &workspace.Runtime{GitRemote: "https://forge.example.test/soda-tester/repo.git"},
	}
	a := factory.Attempt{Work: factory.WorkItem{RepositoryID: 8}}
	if err := c.prepareWorkspace(t.Context(), a, &factory.Run{}); err == nil {
		t.Fatal("foreign repository reached source preparation")
	}
	a.Work.RepositoryID = 7
	c.Workspace.GitRemote = "https://forge.example.test/soda-tester/other.git"
	if err := c.prepareWorkspace(t.Context(), a, &factory.Run{}); err == nil {
		t.Fatal("remote different from trusted repository reached source preparation")
	}
}
