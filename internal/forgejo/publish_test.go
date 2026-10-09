package forgejo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

func publishWorkFixture(t *testing.T) factory.PublicationWork {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = source
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v", err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "--initial-branch=main")
	git("config", "user.name", "soda-tester")
	git("config", "user.email", "soda-tester@localhost")
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-m", "base")
	base := git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-m", "candidate")
	candidate := git("rev-parse", "HEAD")
	path := filepath.Join(root, "candidate.bundle")
	git("bundle", "create", path, "HEAD")
	bundle, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	work := factory.PublicationWork{
		Bundle: bundle, AssignmentID: factory.NewID(), Publication: factory.NewID(),
		RunID: factory.NewID(), Candidate: candidate, BaseSHA: base,
		TargetBranch: "refs/heads/main", OperationID: "soda-test-publish-1",
		AuthRevision: "soda-assignment:a:publication:p:revision:0",
		ExpectedOld:  "absent", ComparisonRef: "refs/heads/main", ComparisonOID: base,
		PRTitle: "Factory candidate for #3", PRBody: "body",
		Repository: 7, Issue: 3, ActorID: 11, NativeRev: 9, NotAfter: now.Add(time.Hour).Unix(),
	}
	work.Run = factory.Run{
		ID: work.RunID, ProjectID: "p" + strings.Repeat("d", 24), Role: "soda-coder",
		InputSHA: base, Started: now, Deadline: now.Add(time.Hour),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex", Model: "test",
	}
	work.AuthRevision = factory.AuthRevisionFor(work.AssignmentID, work.Publication, 0)
	return work
}

func publishTestPublisher(t *testing.T, background *ServiceBackground) (*Publisher, factory.PublicationWork) {
	t.Helper()
	rest := observationREST(t, 11)
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	work := publishWorkFixture(t)
	publisher := NewPublisher(background, rest, "http://127.0.0.1:9", root, observationCredential(t, "test-pat"))
	return publisher, work
}

func TestPublisherResolvesPerRepositoryRemote(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	publisher, work := publishTestPublisher(t, NewServiceBackground(socket, uint32(os.Getuid()), ""))
	cfg, err := publisher.publisherConfig(context.Background(), work)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.Remote != "http://127.0.0.1:9/o/n.git" || cfg.Username != "soda-tester" {
		t.Fatalf("remote: %+v", cfg)
	}
	work.ActorID = 12
	if _, err := publisher.publisherConfig(context.Background(), work); err == nil {
		t.Fatal("wrong actor configured")
	} else if wait, ok := err.(*factory.PublicationWait); !ok || wait.Reason != "authority_lost" {
		t.Fatalf("actor: %v", err)
	}
	unconfigured := NewPublisher(nil, nil, "", "", "")
	if _, err := unconfigured.publisherConfig(context.Background(), work); err == nil {
		t.Fatal("unconfigured publisher resolved")
	}
}

func TestPublisherSubmitsThroughSharedTransport(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	publisher, work := publishTestPublisher(t, NewServiceBackground(socket, uint32(os.Getuid()), ""))
	ctx := context.Background()
	outcome, err := publisher.SubmitPublish(ctx, work)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if outcome.Effect != factory.OpEffectPending {
		t.Fatalf("outcome: %+v", outcome)
	}
	lookup, err := publisher.LookupOp(ctx, work.OperationID)
	if err != nil || lookup.Effect != factory.OpEffectPending {
		t.Fatalf("lookup: %+v %v", lookup, err)
	}
	cancelled, err := publisher.CancelOp(ctx, work.OperationID)
	if err != nil || cancelled.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("cancel: %+v %v", cancelled, err)
	}
	work.OperationID = "soda-test-prcreate-1"
	created, err := publisher.SubmitPRCreate(ctx, work)
	if err != nil || created.Effect != factory.OpEffectPending {
		t.Fatalf("create: %+v %v", created, err)
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 1 {
		t.Fatalf("bootstraps: %d", bootstraps)
	}
}

func TestPublisherRefusesInvalidBundle(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	publisher, work := publishTestPublisher(t, NewServiceBackground(socket, uint32(os.Getuid()), ""))
	work.Bundle = []byte("not-a-bundle")
	if _, err := publisher.ObservePublication(context.Background(), work); err == nil {
		t.Fatal("invalid bundle observed")
	} else if refusal, ok := err.(*factory.PublicationRefusal); !ok || refusal.Reason != "candidate_invalid" {
		t.Fatalf("bundle: %v", err)
	}
}

func TestPublisherAdoptsExactReceipts(t *testing.T) {
	publisher, work := publishTestPublisher(t, nil)
	branch, err := publisher.AdoptBranch(work, factory.OperationOutcome{
		Effect: factory.OpEffectCommitted, OperationID: work.OperationID, InstallationID: "install-1",
		Kind: factory.OpRefPublish, ActorID: work.ActorID, RepositoryID: work.Repository,
		Receipt: []byte(`{"ref":"` + factory.PublishBranchName(work.AssignmentID) + `","old_oid":"` +
			strings.Repeat("0", 40) + `","new_oid":"` + work.Candidate + `","comparison_ref":"refs/heads/main",` +
			`"comparison_oid":"` + work.ComparisonOID + `","actor_id":11,"repository_id":7}`),
	})
	if err != nil {
		t.Fatalf("branch: %v", err)
	}
	if branch.NewOID != work.Candidate {
		t.Fatalf("branch: %+v", branch)
	}
	if _, err := publisher.AdoptBranch(work, factory.OperationOutcome{Effect: factory.OpEffectCommitted}); err == nil {
		t.Fatal("missing branch receipt adopted")
	}
	created, err := publisher.AdoptPRCreation(work, factory.OperationOutcome{
		Effect: factory.OpEffectCommitted, OperationID: work.OperationID, InstallationID: "install-1",
		Kind: factory.OpPRCreate, ActorID: work.ActorID, RepositoryID: work.Repository,
		Receipt: []byte(`{"head_ref":"` + factory.PublishBranchName(work.AssignmentID) + `","base_ref":"refs/heads/main",` +
			`"head_oid":"` + work.Candidate + `","base_oid":"` + work.ComparisonOID + `",` +
			`"author_id":11,"repository_id":7,"pr_id":8,"issue_id":10,"pr_number":9}`),
	})
	if err != nil {
		t.Fatalf("PR: %v", err)
	}
	if created.PRNumber != 9 || created.PRID != 8 || created.IssueID != 10 {
		t.Fatalf("PR: %+v", created)
	}
	if _, err := publisher.AdoptPRCreation(work, factory.OperationOutcome{
		Effect: factory.OpEffectCommitted, OperationID: work.OperationID, InstallationID: "install-1",
		Kind: factory.OpPRCreate, ActorID: work.ActorID, RepositoryID: work.Repository,
		Receipt: []byte(`{"head_ref":"refs/heads/other","base_ref":"refs/heads/main",` +
			`"head_oid":"` + work.Candidate + `","base_oid":"` + work.ComparisonOID + `",` +
			`"author_id":11,"repository_id":7,"pr_id":8,"issue_id":10,"pr_number":9}`),
	}); err == nil {
		t.Fatal("foreign PR adopted")
	}
}

func TestPublisherCanObserveWithoutPreviousObservation(t *testing.T) {
	publisher, work := publishTestPublisher(t, nil)
	publisher.background = &ServiceBackground{}
	work.NativeRev, work.ComparisonOID = 0, ""
	// Config resolution must not require the observation it is preparing
	// to obtain. It performs no native mutation.
	if _, err := publisher.publisherConfig(context.Background(), work); err != nil {
		t.Fatalf("unobserved work cannot configure publisher: %v", err)
	}
	if _, err := publisher.SubmitPublish(context.Background(), work); err == nil {
		t.Fatal("unobserved work submitted")
	}
}

func TestPublisherAdoptionRejectsForeignOperationMetadata(t *testing.T) {
	publisher, work := publishTestPublisher(t, nil)
	base := factory.OperationOutcome{
		Effect: factory.OpEffectCommitted, OperationID: work.OperationID, InstallationID: "install-1",
		Kind: factory.OpRefPublish, ActorID: work.ActorID, RepositoryID: work.Repository,
		Receipt: []byte(`{"ref":"` + factory.PublishBranchName(work.AssignmentID) + `","old_oid":"` +
			strings.Repeat("0", 40) + `","new_oid":"` + work.Candidate + `","comparison_ref":"refs/heads/main",` +
			`"comparison_oid":"` + work.ComparisonOID + `","actor_id":11,"repository_id":7}`),
	}
	for name, change := range map[string]func(*factory.OperationOutcome){
		"operation":    func(o *factory.OperationOutcome) { o.OperationID = "other" },
		"kind":         func(o *factory.OperationOutcome) { o.Kind = factory.OpPRCreate },
		"actor":        func(o *factory.OperationOutcome) { o.ActorID++ },
		"repository":   func(o *factory.OperationOutcome) { o.RepositoryID++ },
		"installation": func(o *factory.OperationOutcome) { o.InstallationID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			outcome := base
			change(&outcome)
			if _, err := publisher.AdoptBranch(work, outcome); err == nil {
				t.Fatal("foreign operation metadata adopted")
			}
		})
	}
}
