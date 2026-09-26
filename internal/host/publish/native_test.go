package publish

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// Native proof uses a dedicated private Forgejo fixture and restricted bot token.
func TestNativePublication(t *testing.T) {
	path := os.Getenv("SODA_FACTORY_PUBLICATION_CONFIG")
	if path == "" {
		t.Skip("private publication fixture required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config Config
	if err = json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(config.Root, "publication-proof-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	git := &repository{dir: dir, token: config.TokenFile, user: config.Username}
	// Initialize an owned bare source with a nonempty placeholder bundle; replace
	// it with a real bundle once the exact fixture base has been fetched.
	if err = git.initialize(t.Context(), []byte("placeholder")); err != nil {
		t.Fatal(err)
	}
	if _, err = git.run(t.Context(), "fetch", config.Remote, "refs/heads/main:refs/heads/base"); err != nil {
		t.Fatal(err)
	}
	out, err := git.run(t.Context(), "rev-parse", "refs/heads/base")
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSpace(string(out))
	out, err = git.run(t.Context(), "rev-parse", "refs/heads/base^{tree}")
	if err != nil {
		t.Fatal(err)
	}
	tree := strings.TrimSpace(string(out))
	out, err = git.run(t.Context(), "-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit-tree", tree, "-p", base, "-m", "isolated publication proof")
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(out))
	if _, err = git.run(t.Context(), "update-ref", "refs/heads/candidate", commit); err != nil {
		t.Fatal(err)
	}
	if _, err = git.run(t.Context(), "symbolic-ref", "HEAD", "refs/heads/candidate"); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(dir, "candidate.bundle")
	if _, err = git.run(t.Context(), "bundle", "create", bundlePath, "HEAD"); err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := factory.New(factory.WorkItem{RepositoryID: 1, Issue: 1, HumanID: 1, Objective: "publication proof", BaseSHA: base, PolicySHA: strings.Repeat("a", 64)}, factory.NewID(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	run, err := attempt.BeginRun(factory.Implementation, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	run.Image = "sha256:" + strings.Repeat("b", 64)
	run.Harness = "proof"
	run.Model = "proof"
	request := Request{Attempt: attempt, Run: run, Commit: commit, Bundle: bundle}
	target := "refs/heads/" + Branch(attempt.ID)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := git.run(ctx, "push", "--force-with-lease="+target+":"+commit, config.Remote, ":"+target); err != nil {
			t.Errorf("owned branch cleanup: %v", err)
		}
	})
	if err = config.Candidate(t.Context(), request, func() error { return nil }); err != nil {
		t.Fatalf("publication: %v", err)
	}
	out, err = git.run(t.Context(), "ls-remote", config.Remote, target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != commit+"\t"+target {
		t.Fatal("published target differs from candidate")
	}
	if err = config.Candidate(t.Context(), request, func() error { return nil }); err == nil {
		t.Fatal("stale absence lease allowed repeat publication")
	}
}
