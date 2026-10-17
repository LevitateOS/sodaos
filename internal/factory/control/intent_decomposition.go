package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// scopedIntentTasksFromChosen adopts exactly the single planner candidate
// the Generate work command chose before serving: it binds the choice to
// the current accepted head and enforces it inside accepted scope before
// any native observation. No triplet judgment or template restructuring
// runs here: the choice arrives decided, so a second evaluation cannot
// rewrite it on the serving path.
func (c *Coordinator) scopedIntentTasksFromChosen(_ context.Context, r factory.IntentRevision, chosen *ChosenIntentPlan) ([]factory.IntentTask, error) {
	if chosen == nil {
		return nil, errors.New("chosen planner candidate is missing")
	}
	if chosen.Intent != r.Intent {
		return nil, errors.New("chosen planner candidate binds another intent")
	}
	if chosen.Operation != factory.IntentPlanningOptions {
		return nil, errors.New("chosen planner candidate is not a decomposition choice")
	}
	// The planner runs against the reviewed draft; the accepted successor
	// adopts its choice. Either the choice binds this exact revision or
	// the draft this acceptance reviewed — never a third revision.
	if chosen.Revision != r.ID && chosen.Revision != r.Predecessor {
		return nil, errors.New("chosen planner revision mismatch")
	}
	if len(chosen.Tasks) == 0 {
		return nil, errors.New("chosen planner candidate carries no tasks")
	}
	work := r
	work.Tasks = append([]factory.IntentTask(nil), chosen.Tasks...)
	if err := work.Validate(); err != nil {
		return nil, errors.New("chosen planner candidate exceeds accepted scope")
	}
	return work.Tasks, nil
}

// scopedIntentTasks selects the S07-advised task graph for native writes and
// enforces it inside accepted scope before any native observation. Review
// verdicts hold for a human challenge; funding failures surface for retry.
func (c *Coordinator) scopedIntentTasks(ctx context.Context, r factory.IntentRevision, previous *factory.IntentRevision) ([]factory.IntentTask, error) {
	decomp, err := c.RecomposeDecomposition(ctx, r, previous)
	if err != nil {
		return nil, err
	}
	if err = decomp.Validate(); err != nil {
		return nil, err
	}
	if decomp.Intent != r.Intent || decomp.Revision != r.ID {
		return nil, errors.New("scoped decomposition revision mismatch")
	}
	work := r
	work.Tasks = decomp.Tasks
	if err = work.Validate(); err != nil {
		return nil, errors.New("scoped decomposition exceeds accepted scope")
	}
	return decomp.Tasks, nil
}

// prepareIntentGraph expresses a reviewed decomposition change through the
// existing native issue owner. Reused open keys retain issue IDs and history;
// retired open tasks receive a visible supersession note without closing
// them. Delivered (closed) records are never edited: materially unchanged
// keys rebind the delivered issue, changed work creates a new issue under
// the stable key, and removed delivered keys are retained silently. It
// returns the submitted task graph and delivered bindings alongside the
// payload; callers persist them so adoption and later deltas bind submitted
// truth rather than re-deriving it.
// prepareIntentGraphWithChoice expresses a reviewed decomposition change
// through the existing native issue owner, adopting either the serving
// advised graph (nil choice) or exactly one pre-chosen planner candidate.
// The chosen path performs no triplet or template evaluation: submitted,
// adopted and delivered bindings all derive from the same chosen tasks.
func (c *Coordinator) prepareIntentGraphWithChoice(ctx context.Context, r factory.IntentRevision, revision int64, chosen *ChosenIntentPlan) (sdk.IssueGraphPayload, map[string]int64, []factory.IntentTask, map[string]factory.DeliveredIssue, error) {
	scoped := func(previous *factory.IntentRevision) ([]factory.IntentTask, error) {
		if chosen != nil {
			return c.scopedIntentTasksFromChosen(ctx, r, chosen)
		}
		return c.scopedIntentTasks(ctx, r, previous)
	}
	previous, err := c.Store.PreviousGeneratedIntent(ctx, r.Intent, r.ID)
	if errors.Is(err, store.ErrNotFound) {
		tasks, e := scoped(nil)
		if e != nil {
			return sdk.IssueGraphPayload{}, nil, nil, nil, e
		}
		work := r
		work.Tasks = tasks
		p, e := IntentGraphPayload(work)
		if e != nil {
			return sdk.IssueGraphPayload{}, nil, nil, nil, e
		}
		return p, nil, tasks, nil, nil
	}
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	generation, err := c.Store.IntentGeneration(ctx, previous.ID)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	if len(generation.SubmittedTasks) == 0 {
		return sdk.IssueGraphPayload{}, nil, nil, nil, errors.New("previous generation predates submitted-task records")
	}
	priorSubmitted := previous
	priorSubmitted.Tasks = generation.SubmittedTasks
	tasks, err := scoped(&priorSubmitted)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	var receipt sdk.IssueGraphReceipt
	if json.Unmarshal(generation.Receipt, &receipt) != nil || receipt.ActorID != generation.ActorID || receipt.RepositoryID != r.Repository {
		return sdk.IssueGraphPayload{}, nil, nil, nil, errors.New("previous graph receipt is unavailable")
	}
	oldIssues, err := intentReceiptIssues(generation.SubmittedTasks, generation, receipt)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	work := r
	work.Tasks = tasks
	prevWork := previous
	prevWork.Tasks = generation.SubmittedTasks
	movement, err := factory.DiffIntentTasks(prevWork, work)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	reused := map[string]bool{}
	for _, key := range movement.Reused {
		reused[key] = true
	}
	current := map[string]factory.IntentTask{}
	for _, task := range tasks {
		current[task.Key] = task
	}
	p := sdk.IssueGraphPayload{}
	ids := map[string]int64{}
	refs := map[string]sdk.IssueGraphRef{}
	delivered := map[string]factory.DeliveredIssue{}
	closed := map[string]factory.DeliveredIssue{}
	priorBindings, err := c.Store.IntentTaskBindings(ctx, previous.ID)
	if err != nil {
		return p, nil, nil, nil, err
	}
	type edge struct{ a, b int64 }
	oldEdges := map[edge]bool{}
	for _, task := range generation.SubmittedTasks {
		entry := oldIssues[task.Key]
		native, nativeRev, e := c.Graph.ReadGeneratedIssue(ctx, r.Repository, entry.IssueIndex, generation.ActorID)
		if e != nil {
			return p, nil, nil, nil, e
		}
		if nativeRev != revision {
			return p, nil, nil, nil, refuseAcceptance(RefusalStaleEvidence)
		}
		issue := native.Issue
		if issue == nil || !issue.Visible || !issue.Complete || issue.IsPull || issue.IsLocked || issue.ID != strconv.FormatInt(entry.IssueID, 10) {
			return p, nil, nil, nil, refuseAcceptance(RefusalObjectiveChanged)
		}
		// Delivered (closed) records are verified but never edited: the
		// native writer refuses closed edits atomically. A delivered
		// record still carries the body its writing revision gave it, so
		// stability is checked against the previous acceptance pins; the
		// rebuilt previous-revision body remains the fallback when the
		// previous adoption never pinned the key. Open records are
		// rewritten by every delta, so the rebuild is exact for them.
		if issue.IsClosed {
			stable, e := c.deliveredStable(ctx, priorBindings, task, entry, issue, prevWork)
			if e != nil {
				return p, nil, nil, nil, e
			}
			if !stable {
				return p, nil, nil, nil, refuseAcceptance(RefusalObjectiveChanged)
			}
			closed[task.Key] = factory.DeliveredIssue{IssueID: entry.IssueID, IssueIndex: entry.IssueIndex, ContentVersion: entry.ContentVersion}
		} else {
			if issue.Title != intentTaskTitle(task) || issue.Content != intentTaskBody(prevWork, task) || issue.ContentVersion != entry.ContentVersion {
				return p, nil, nil, nil, refuseAcceptance(RefusalObjectiveChanged)
			}
			title, body := issue.Title, issue.Content+"\nSuperseded by accepted product revision: "+r.ID+"\nThis task is retained as history and grants no current generated-work authority.\n"
			if next, ok := current[task.Key]; ok {
				title = intentTaskTitle(next)
				body = intentTaskBody(work, next)
			}
			p.Edits = append(p.Edits, sdk.IssueGraphEdit{IssueID: entry.IssueID, ExpectedTitle: issue.Title, ExpectedBody: issue.Content, ExpectedContentVersion: issue.ContentVersion, Title: title, Body: body})
		}
		if native.Dependencies == nil || !native.Dependencies.Complete || len(native.Dependencies.Items) != len(task.DependsOn) {
			return p, nil, nil, nil, refuseAcceptance(RefusalEdgeChanged)
		}
		expected := map[int64]bool{}
		for _, dep := range task.DependsOn {
			expected[oldIssues[dep].IssueID] = true
		}
		for _, d := range native.Dependencies.Items {
			dep, e := strconv.ParseInt(d.DependencyID, 10, 64)
			if e != nil || !d.Visible || !d.Complete || !expected[dep] {
				return p, nil, nil, nil, refuseAcceptance(RefusalEdgeChanged)
			}
			delete(expected, dep)
			oldEdges[edge{entry.IssueID, dep}] = true
		}
		if len(expected) > 0 {
			return p, nil, nil, nil, refuseAcceptance(RefusalEdgeChanged)
		}
	}
	for _, task := range tasks {
		old, ok := oldIssues[task.Key]
		if !ok {
			p.Creates = append(p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: intentTaskBody(work, task)})
			refs[task.Key] = sdk.IssueGraphRef{Key: task.Key}
			continue
		}
		if ref, closed := closed[task.Key]; closed {
			if reused[task.Key] {
				delivered[task.Key] = ref
				refs[task.Key] = sdk.IssueGraphRef{IssueID: ref.IssueID}
			} else {
				p.Creates = append(p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: intentTaskBody(work, task)})
				refs[task.Key] = sdk.IssueGraphRef{Key: task.Key}
			}
			continue
		}
		ids[task.Key] = old.IssueID
		refs[task.Key] = sdk.IssueGraphRef{IssueID: old.IssueID}
	}
	retained := map[edge]bool{}
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			a, b := refs[task.Key], refs[dep]
			key := edge{a.IssueID, b.IssueID}
			if key.a > 0 && key.b > 0 && oldEdges[key] {
				retained[key] = true
				continue
			}
			p.Dependencies = append(p.Dependencies, sdk.IssueGraphDependency{Issue: a, Prerequisite: b})
		}
	}
	// Stable order follows the previous task/dependency order; the immutable
	// packet is deterministic even though membership checks use maps.
	for _, task := range generation.SubmittedTasks {
		for _, dep := range task.DependsOn {
			key := edge{oldIssues[task.Key].IssueID, oldIssues[dep].IssueID}
			if !retained[key] {
				p.Dependencies = append(p.Dependencies, sdk.IssueGraphDependency{Issue: sdk.IssueGraphRef{IssueID: key.a}, Prerequisite: sdk.IssueGraphRef{IssueID: key.b}, Remove: true})
			}
		}
	}
	if err = sdk.ValidateIssueGraphPayload(p); err != nil {
		return p, nil, nil, nil, fmt.Errorf("accepted decomposition exceeds native packet bounds: %w", err)
	}
	if len(delivered) == 0 {
		delivered = nil
	}
	return p, ids, tasks, delivered, nil
}

// deliveredStable reports whether a closed native record still carries the
// exact content its writing revision gave it. The previous acceptance pins
// title and content digests at every adoption, so pins stay exact across
// any number of delivered hops with single-hop lookups. When the previous
// adoption never pinned the key, the rebuilt previous-revision body is the
// fallback; it is exact for records the previous submit wrote.
func (c *Coordinator) deliveredStable(ctx context.Context, prior []factory.IntentTaskBinding, task factory.IntentTask, entry sdk.IssueGraphIssueReceipt, issue *sdk.SnapshotIssue, prevWork factory.IntentRevision) (bool, error) {
	for _, b := range prior {
		if b.TaskKey != task.Key {
			continue
		}
		if b.IssueID != entry.IssueID || b.IssueIndex != entry.IssueIndex || b.ContentVersion != entry.ContentVersion {
			return false, errors.New("previous graph receipt does not bind submitted tasks")
		}
		decision, err := c.Store.AcceptanceDecision(ctx, b.AcceptanceID)
		if errors.Is(err, store.ErrNotFound) {
			break
		}
		if err != nil {
			return false, err
		}
		return issue.TitleDigest == decision.TitleDigest && issue.ContentDigest == decision.ContentDigest && issue.ContentVersion == decision.ContentVersion, nil
	}
	return issue.Title == intentTaskTitle(task) && issue.Content == intentTaskBody(prevWork, task) && issue.ContentVersion == entry.ContentVersion, nil
}

func intentReceiptIssues(tasks []factory.IntentTask, g factory.IntentGeneration, receipt sdk.IssueGraphReceipt) (map[string]sdk.IssueGraphIssueReceipt, error) {
	issues := map[string]sdk.IssueGraphIssueReceipt{}
	byID := map[int64]sdk.IssueGraphIssueReceipt{}
	for _, entry := range receipt.Issues {
		if entry.IssueID <= 0 || entry.IssueIndex <= 0 || byID[entry.IssueID].IssueID != 0 {
			return nil, errors.New("invalid native graph receipt")
		}
		byID[entry.IssueID] = entry
		if entry.Key != "" {
			if issues[entry.Key].IssueID != 0 {
				return nil, errors.New("duplicate native graph key")
			}
			issues[entry.Key] = entry
		}
	}
	for key, id := range g.TaskIssueIDs {
		entry, exists := byID[id]
		if !exists || entry.Key != "" {
			return nil, errors.New("native graph edit receipt missing reused task")
		}
		issues[key] = entry
	}
	for key, ref := range g.Delivered {
		if issues[key].IssueID != 0 || byID[ref.IssueID].IssueID != 0 {
			return nil, errors.New("delivered history unexpectedly mutated")
		}
		issues[key] = sdk.IssueGraphIssueReceipt{Key: key, IssueID: ref.IssueID, IssueIndex: ref.IssueIndex, ContentVersion: ref.ContentVersion}
	}
	if len(issues) != len(tasks) {
		return nil, errors.New("native graph receipt does not bind submitted tasks")
	}
	for _, task := range tasks {
		if issues[task.Key].IssueID == 0 {
			return nil, errors.New("native graph receipt missing submitted task")
		}
	}
	return issues, nil
}
