package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// nativeExpandingContextRead mimics the native reader's instruction
// expansion: every response carries the requested files plus root and
// ancestor AGENTS.md at the requested commit.
func nativeExpandingContextRead(contents map[string]string) func(project.FactoryPreparationContextRequest, string) (project.FactoryPreparationContext, error) {
	return func(in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
		seen := map[string]bool{}
		var files []project.FactoryContextFile
		add := func(path string) {
			if seen[path] {
				return
			}
			seen[path] = true
			content := "native instruction context: " + path
			if explicit, ok := contents[path]; ok {
				content = explicit
			}
			files = append(files, project.FactoryContextFile{Path: path, Content: []byte(content)})
		}
		add("AGENTS.md")
		for _, path := range in.Paths {
			add(path)
			parts := strings.Split(path, "/")
			prefix := ""
			for _, part := range parts[:len(parts)-1] {
				if prefix == "" {
					prefix = part
				} else {
					prefix += "/" + part
				}
				add(prefix + "/AGENTS.md")
			}
		}
		return project.FactoryPreparationContext{
			Project: in.Project, ID: in.ID, Role: role,
			SourceCommit: in.SourceCommit, ApprovedBase: in.ApprovedBase,
			DiffBase: in.DiffBase, Candidate: in.Candidate,
			Setup: []byte("native setup instructions"), Check: []byte("native required checks"),
			Files: files,
		}, nil
	}
}

func assemblySourceInputs(t *testing.T, paths []string) map[string]string {
	t.Helper()
	selected := make(map[string]string, len(paths))
	for _, path := range paths {
		sum := sha256.Sum256([]byte("native bytes: " + path))
		selected[path] = hex.EncodeToString(sum[:])
	}
	return selected
}

func assemblyContents(paths []string) map[string]string {
	contents := make(map[string]string, len(paths))
	for _, path := range paths {
		contents[path] = "native bytes: " + path
	}
	return contents
}

func assemblyHead(repo int64) factory.IntentRevision {
	return factory.IntentRevision{
		ID: "d888888888888888888888888", Intent: "d777777777777777777777777", Owner: 7, Repository: repo,
		Objective: "Decide vacation requests with manager review", State: "draft",
		MaxIssues: 10, CreatedUnix: time.Now().UTC().Truncate(time.Second).Unix(),
	}
}

// TestAssembleIntentPlanningInputReadsExpandedContext proves server-owned
// input assembly across chunked native reads: twelve approved files plus
// one nested path need two reads whose shared root AGENTS.md repeats
// identically, and the assembled evidence carries exactly the approved
// paths with digest-verified bytes — never native-added instruction
// files, never a repeated-path refusal.
func TestAssembleIntentPlanningInputReadsExpandedContext(t *testing.T) {
	ctx := context.Background()
	paths := []string{
		"internal/factory/a.go", "internal/factory/b.go", "internal/factory/c.go",
		"internal/factory/d.go", "internal/factory/e.go", "internal/factory/f.go",
		"internal/factory/g.go", "internal/factory/h.go", "internal/factory/i.go",
		"internal/factory/j.go", "internal/factory/k.go", "internal/factory/l.go",
		"internal/store/nested/deep/plan.go",
	}
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db, assemblySourceInputs(t, paths))
	fx.host.contextRead = nativeExpandingContextRead(assemblyContents(paths))
	coord := &Coordinator{Store: db, Host: fx.host}
	head := assemblyHead(fx.repo)
	in, prep, wait, err := coord.AssembleIntentPlanningInput(ctx, head, factory.IntentPlanningPropose)
	if err != nil {
		t.Fatal(err)
	}
	if wait != nil {
		t.Fatalf("expanded selection waited: %+v", wait)
	}
	if prep.ID != "f111111111111111111111111" || prep.SourceCommit != strings.Repeat("c", 40) {
		t.Fatalf("wrong preparation resolved: %+v", prep.ID)
	}
	if len(fx.host.contextRequests) < 2 {
		t.Fatalf("multi-chunk selection read once: %d", len(fx.host.contextRequests))
	}
	for _, request := range fx.host.contextRequests {
		if len(expandContextSelection(request.Paths)) > project.MaxFactoryContextPaths {
			t.Fatalf("read exceeds its native closure bound: %v", request.Paths)
		}
	}
	if len(in.Inspected) != len(paths) {
		t.Fatalf("inspected carries %d entries, want %d", len(in.Inspected), len(paths))
	}
	for i, ref := range in.Inspected {
		if ref.Path != paths[i] || ref.Commit != strings.Repeat("c", 40) || ref.Evidence != "native bytes: "+paths[i] {
			t.Fatalf("inspected entry not exact: %+v", ref)
		}
	}
	if in.Operation != factory.IntentPlanningPropose || in.Head.ID != head.ID {
		t.Fatalf("assembled input not bound: %+v", in.Operation)
	}
}

// TestAssembleIntentPlanningInputRefusesConflictingRepeat proves a
// repeated approved file with contradictory bytes refuses instead of
// assembling evidence from an inconsistent read.
func TestAssembleIntentPlanningInputRefusesConflictingRepeat(t *testing.T) {
	ctx := context.Background()
	paths := []string{"internal/factory/intent.go"}
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db, assemblySourceInputs(t, paths))
	contents := assemblyContents(paths)
	fx.host.contextRead = func(in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
		result, err := nativeExpandingContextRead(contents)(in, role)
		if err != nil {
			return result, err
		}
		result.Files = append(result.Files, project.FactoryContextFile{Path: paths[0], Content: []byte("contradictory bytes")})
		return result, nil
	}
	coord := &Coordinator{Store: db, Host: fx.host}
	if _, _, wait, err := coord.AssembleIntentPlanningInput(ctx, assemblyHead(fx.repo), factory.IntentPlanningPropose); err == nil || wait != nil {
		t.Fatalf("conflicting repeat assembled: wait=%+v err=%v", wait, err)
	}
}
