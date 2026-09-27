package main

import "testing"

func TestRemoteRequestSelectsRepositoryWithoutCredentials(t *testing.T) {
	r, err := remoteRequest([]string{"origin", "soda://soda-tester/repository.git"}, "/home/soda-tester/work", "selected")
	if err != nil || r.Owner != "soda-tester" || r.Repository != "repository" || r.Remote != "origin" || r.ConnectionID != "selected" {
		t.Fatal("remote selection changed", err)
	}
}

func TestRemoteRequestAcceptsGitURLRewriteLabel(t *testing.T) {
	remote := "https://forgejo.example.test/soda-tester/repository.git"
	r, err := remoteRequest([]string{remote, "soda://soda-tester/repository.git"}, "/workspace/repo", "")
	if err != nil || r.Remote != remote || r.Owner != "soda-tester" || r.Repository != "repository" {
		t.Fatal("ordinary Git URL rewrite rejected", err)
	}
	if _, err := remoteRequest([]string{remote + "\nunsafe", "soda://soda-tester/repository.git"}, "/workspace/repo", ""); err == nil {
		t.Fatal("control-bearing remote label admitted")
	}
}

func TestRemoteRequestRejectsAuthorityAndPathOverrides(t *testing.T) {
	for _, remote := range []string{
		"https://example.test/soda-tester/repository.git",
		"soda://secret@soda-tester/repository.git",
		"soda://soda-tester:8080/repository.git",
		"soda://soda-tester/repository.git?token=secret",
		"soda://soda-tester/repository.git#fragment",
		"soda://soda-tester/other/repository.git",
		"soda://soda-tester/%2frepository.git",
		"soda://soda-tester/repository",
	} {
		if _, err := remoteRequest([]string{"origin", remote}, "/home/soda-tester/work", ""); err == nil {
			t.Fatal("invalid remote admitted", remote)
		}
	}
}
