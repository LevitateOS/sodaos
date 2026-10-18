package control

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// expandContextSelection mirrors the native reader's instruction
// expansion: root AGENTS.md plus every ancestor AGENTS.md for each
// approved path. The native side adds the same set to every request,
// so chunked reads legitimately repeat shared instruction files.
func expandContextSelection(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	add("AGENTS.md")
	for _, path := range paths {
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
	sort.Strings(out)
	return out
}

// chunkApprovedSelections groups approved paths so every read's native
// closure — the chunk plus its instruction expansion — stays within the
// context path bound. A single path whose own closure exceeds the bound
// waits instead of reading partially.
func chunkApprovedSelections(paths []string) ([][]string, error) {
	var chunks [][]string
	var current []string
	for _, path := range paths {
		grown := append(append([]string(nil), current...), path)
		if len(expandContextSelection(grown)) > project.MaxFactoryContextPaths {
			if len(current) > 0 {
				chunks = append(chunks, current)
				current = nil
			}
			if len(expandContextSelection([]string{path})) > project.MaxFactoryContextPaths {
				return nil, errors.New("approved source selection exceeds its context bound")
			}
		}
		current = append(current, path)
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks, nil
}

// AssembleIntentPlanningInput resolves the server-owned planning input
// for one draft revision under the requested operation. The caller binds
// only the saved revision and operation; source, preparation and
// inspected evidence come from the serving side: the authorized ready
// coder preparation supplies the source commit, and every inspected
// entry is built from actual preparation-context bytes the host returns
// at that commit, digest-verified against the approved source selection.
// Anything missing — preparation, admissible files, context, host or
// authority — waits truthfully before any funding or launch, and
// anything contradicting the approval refuses. No caller-supplied
// commit or evidence is accepted, and nothing is inferred from a hash.
func (c *Coordinator) AssembleIntentPlanningInput(ctx context.Context, head factory.IntentRevision, operation string) (factory.SemanticPlannerInput, project.Preparation, *DispatchWait, error) {
	var empty factory.SemanticPlannerInput
	if !factory.ValidIntentPlanningOperation(operation) {
		return empty, project.Preparation{}, nil, errors.New("unknown intent planning operation")
	}
	if c.Store == nil {
		return empty, project.Preparation{}, nil, errors.New("intent planning store unavailable")
	}
	proj, prep, paths, planWait, err := c.resolvePlanningPreparation(ctx, head)
	if err != nil || planWait != nil {
		return empty, project.Preparation{}, planWait, err
	}
	if c.Host == nil {
		return empty, project.Preparation{}, nil, errors.New("intent planning host unavailable")
	}
	deadline := time.Now().Add(2 * time.Minute)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	merged, planWait, err := c.readPlanningContext(bounded, proj, prep, paths, deadline, head.Repository)
	if err != nil || planWait != nil {
		return empty, project.Preparation{}, planWait, err
	}
	in, planWait, err := assemblePlanningInput(head, operation, prep, paths, merged)
	if err != nil || planWait != nil {
		return empty, project.Preparation{}, planWait, err
	}
	return in, prep, nil, nil
}

func (c *Coordinator) resolvePlanningPreparation(ctx context.Context, head factory.IntentRevision) (store.Project, project.Preparation, []string, *DispatchWait, error) {
	proj, err := c.Store.ProjectByRepository(ctx, head.Repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.Project{}, project.Preparation{}, nil, &DispatchWait{Repository: head.Repository, Reason: WaitAuthority, Detail: "project changed during draft planning"}, nil
		}
		return store.Project{}, project.Preparation{}, nil, nil, err
	}
	plan := &attemptPlan{projectID: proj.ID}
	if first, second := selectPreparationFor(ctx, c.dispatchDeps(), plan, project.RoleCoder, ""); second != nil {
		return store.Project{}, project.Preparation{}, nil, nil, errors.New(second.Detail)
	} else if first != nil {
		wait := timeOnlyWait(head.Repository, first)
		return store.Project{}, project.Preparation{}, nil, &wait, nil
	}
	prep := plan.prep.Preparation
	paths := make([]string, 0, len(prep.Requirements.SourceInputs))
	for path := range prep.Requirements.SourceInputs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return store.Project{}, project.Preparation{}, nil, &DispatchWait{Repository: head.Repository, Reason: WaitPreparation, Detail: "preparation selects no approved source files"}, nil
	}
	return proj, prep, paths, nil, nil
}

func (c *Coordinator) readPlanningContext(ctx context.Context, proj store.Project, prep project.Preparation, paths []string, deadline time.Time, repository int64) (map[string][]byte, *DispatchWait, error) {
	chunks, err := chunkApprovedSelections(paths)
	if err != nil {
		return nil, &DispatchWait{Repository: repository, Reason: WaitPreparation, Detail: err.Error()}, nil
	}
	merged := make(map[string][]byte, len(paths))
	base := project.FactoryPreparationContextRequest{
		Project: proj.ID, ID: prep.ID, SourceCommit: prep.SourceCommit,
		ApprovedBase: prep.SourceCommit, DiffBase: prep.SourceCommit, Candidate: prep.SourceCommit,
		NotAfter: deadline,
	}
	for _, chunk := range chunks {
		base.Paths = chunk
		if err := base.Validate(); err != nil {
			return nil, nil, err
		}
		result, err := c.Host.ReadPreparationContext(ctx, base, project.RoleCoder)
		if err != nil {
			return nil, &DispatchWait{Repository: repository, Reason: WaitPreparation, Detail: "preparation context unreadable"}, nil
		}
		if err := result.ValidateFor(base, project.RoleCoder); err != nil {
			return nil, nil, err
		}
		if err := mergePlanningContextFiles(prep, result.Files, merged); err != nil {
			return nil, nil, err
		}
	}
	return merged, nil, nil
}

func mergePlanningContextFiles(prep project.Preparation, files []project.FactoryContextFile, merged map[string][]byte) error {
	for _, file := range files {
		want, approved := prep.Requirements.SourceInputs[file.Path]
		if !approved {
			continue
		}
		if prev, seen := merged[file.Path]; seen {
			if !bytes.Equal(prev, file.Content) {
				return errors.New("preparation context contradicts its repeated source file")
			}
			continue
		}
		sum := sha256.Sum256(file.Content)
		if hex.EncodeToString(sum[:]) != want {
			return errors.New("preparation context does not match its approval")
		}
		if len(file.Content) > factory.MaxPlanTextBytes {
			return errors.New("approved source file exceeds its evidence bound")
		}
		merged[file.Path] = file.Content
	}
	return nil
}

func assemblePlanningInput(head factory.IntentRevision, operation string, prep project.Preparation, paths []string, merged map[string][]byte) (factory.SemanticPlannerInput, *DispatchWait, error) {
	var empty factory.SemanticPlannerInput
	inspected := make([]factory.InspectedSource, 0, len(paths))
	for _, path := range paths {
		content, ok := merged[path]
		if !ok {
			return empty, nil, errors.New("preparation context lacks its approved source file")
		}
		inspected = append(inspected, factory.InspectedSource{Path: path, Commit: prep.SourceCommit, Evidence: string(content)})
	}
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		return empty, nil, err
	}
	in := factory.SemanticPlannerInput{
		Operation: operation, Head: head, Content: content,
		Alternatives: factory.DescribeAlternatives(head.Objective), Inspected: inspected,
	}
	if err := in.Validate(); err != nil {
		return empty, nil, err
	}
	return in, nil, nil
}
