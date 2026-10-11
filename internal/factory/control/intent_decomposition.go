package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

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
// existing native issue owner. Reused keys retain issue IDs and history;
// retired tasks receive a visible supersession note without closing them.
// It returns the submitted task graph alongside the payload; callers persist
// it on the generation record so adoption and later deltas bind submitted
// truth rather than re-deriving it.
func (c *Coordinator) prepareIntentGraph(ctx context.Context, r factory.IntentRevision, revision int64) (sdk.IssueGraphPayload, map[string]int64, []factory.IntentTask, error) {
	previous, err := c.Store.PreviousGeneratedIntent(ctx, r.Intent, r.ID)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		tasks, e := c.scopedIntentTasks(ctx, r, nil)
		if e != nil {
			return sdk.IssueGraphPayload{}, nil, nil, e
		}
		work := r
		work.Tasks = tasks
		p, e := IntentGraphPayload(work)
		if e != nil {
			return sdk.IssueGraphPayload{}, nil, nil, e
		}
		return p, nil, tasks, nil
	}
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, err
	}
	generation, err := c.Store.IntentGeneration(ctx, previous.ID)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, err
	}
	if len(generation.SubmittedTasks) == 0 {
		return sdk.IssueGraphPayload{}, nil, nil, errors.New("previous generation predates submitted-task records")
	}
	priorSubmitted := previous
	priorSubmitted.Tasks = generation.SubmittedTasks
	tasks, err := c.scopedIntentTasks(ctx, r, &priorSubmitted)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, err
	}
	var receipt sdk.IssueGraphReceipt
	if json.Unmarshal(generation.Receipt, &receipt) != nil || receipt.ActorID != generation.ActorID || receipt.RepositoryID != r.Repository {
		return sdk.IssueGraphPayload{}, nil, nil, errors.New("previous graph receipt is unavailable")
	}
	oldIssues, err := intentReceiptIssues(generation.SubmittedTasks, generation, receipt)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, err
	}
	work := r
	work.Tasks = tasks
	prevWork := previous
	prevWork.Tasks = generation.SubmittedTasks
	current := map[string]factory.IntentTask{}
	for _, task := range tasks {
		current[task.Key] = task
	}
	p := sdk.IssueGraphPayload{}
	ids := map[string]int64{}
	refs := map[string]sdk.IssueGraphRef{}
	for _, task := range tasks {
		if old, ok := oldIssues[task.Key]; ok {
			ids[task.Key] = old.IssueID
			refs[task.Key] = sdk.IssueGraphRef{IssueID: old.IssueID}
		} else {
			p.Creates = append(p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: intentTaskBody(work, task)})
			refs[task.Key] = sdk.IssueGraphRef{Key: task.Key}
		}
	}
	type edge struct{ a, b int64 }
	oldEdges := map[edge]bool{}
	for _, task := range generation.SubmittedTasks {
		entry := oldIssues[task.Key]
		native, nativeRev, e := c.Graph.ReadGeneratedIssue(ctx, r.Repository, entry.IssueIndex, generation.ActorID)
		if e != nil {
			return p, nil, nil, e
		}
		if nativeRev != revision {
			return p, nil, nil, refuseAcceptance(RefusalStaleEvidence)
		}
		issue := native.Issue
		// Delivered (closed) records stay in the delta: edits never change
		// closed state, delivery and prior acceptances are retained, and new
		// bindings chain forward. Locked or incomplete records still refuse.
		if issue == nil || !issue.Visible || !issue.Complete || issue.IsPull || issue.IsLocked || issue.ID != strconv.FormatInt(entry.IssueID, 10) || issue.Content != intentTaskBody(prevWork, task) || issue.Title != intentTaskTitle(task) || issue.ContentVersion != entry.ContentVersion {
			return p, nil, nil, refuseAcceptance(RefusalObjectiveChanged)
		}
		title, body := issue.Title, issue.Content+"\nSuperseded by accepted product revision: "+r.ID+"\nThis task is retained as history and grants no current generated-work authority.\n"
		if next, ok := current[task.Key]; ok {
			title = intentTaskTitle(next)
			body = intentTaskBody(work, next)
		}
		p.Edits = append(p.Edits, sdk.IssueGraphEdit{IssueID: entry.IssueID, ExpectedTitle: issue.Title, ExpectedBody: issue.Content, ExpectedContentVersion: issue.ContentVersion, Title: title, Body: body})
		if native.Dependencies == nil || !native.Dependencies.Complete || len(native.Dependencies.Items) != len(task.DependsOn) {
			return p, nil, nil, refuseAcceptance(RefusalEdgeChanged)
		}
		expected := map[int64]bool{}
		for _, dep := range task.DependsOn {
			expected[oldIssues[dep].IssueID] = true
		}
		for _, d := range native.Dependencies.Items {
			dep, e := strconv.ParseInt(d.DependencyID, 10, 64)
			if e != nil || !d.Visible || !d.Complete || !expected[dep] {
				return p, nil, nil, refuseAcceptance(RefusalEdgeChanged)
			}
			delete(expected, dep)
			oldEdges[edge{entry.IssueID, dep}] = true
		}
		if len(expected) > 0 {
			return p, nil, nil, refuseAcceptance(RefusalEdgeChanged)
		}
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
		return p, nil, nil, fmt.Errorf("accepted decomposition exceeds native packet bounds: %w", err)
	}
	return p, ids, tasks, nil
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
