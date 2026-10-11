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

// prepareIntentGraph expresses a reviewed decomposition change through the
// existing native issue owner. Reused keys retain issue IDs and history;
// retired tasks receive a visible supersession note without closing them.
func (c *Coordinator) prepareIntentGraph(ctx context.Context, r factory.IntentRevision, revision int64) (sdk.IssueGraphPayload, map[string]int64, error) {
	previous, err := c.Store.PreviousGeneratedIntent(ctx, r.Intent, r.ID)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		if e := c.AdviseIntentDecomposition(ctx, r, nil); e != nil {
			return sdk.IssueGraphPayload{}, nil, e
		}
		p, e := IntentGraphPayload(r)
		return p, nil, e
	}
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, err
	}
	if err = c.AdviseIntentDecomposition(ctx, r, &previous); err != nil {
		return sdk.IssueGraphPayload{}, nil, err
	}
	generation, err := c.Store.IntentGeneration(ctx, previous.ID)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, err
	}
	var receipt sdk.IssueGraphReceipt
	if json.Unmarshal(generation.Receipt, &receipt) != nil || receipt.ActorID != generation.ActorID || receipt.RepositoryID != r.Repository {
		return sdk.IssueGraphPayload{}, nil, errors.New("previous graph receipt is unavailable")
	}
	oldIssues, err := intentReceiptIssues(previous, generation, receipt)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, err
	}
	current := map[string]factory.IntentTask{}
	for _, task := range r.Tasks {
		current[task.Key] = task
	}
	p := sdk.IssueGraphPayload{}
	ids := map[string]int64{}
	refs := map[string]sdk.IssueGraphRef{}
	for _, task := range r.Tasks {
		if old, ok := oldIssues[task.Key]; ok {
			ids[task.Key] = old.IssueID
			refs[task.Key] = sdk.IssueGraphRef{IssueID: old.IssueID}
		} else {
			p.Creates = append(p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: intentTaskBody(r, task)})
			refs[task.Key] = sdk.IssueGraphRef{Key: task.Key}
		}
	}
	type edge struct{ a, b int64 }
	oldEdges := map[edge]bool{}
	for _, task := range previous.Tasks {
		entry := oldIssues[task.Key]
		native, nativeRev, e := c.Graph.ReadGeneratedIssue(ctx, r.Repository, entry.IssueIndex, generation.ActorID)
		if e != nil {
			return p, nil, e
		}
		if nativeRev != revision {
			return p, nil, refuseAcceptance(RefusalStaleEvidence)
		}
		issue := native.Issue
		if issue == nil || !issue.Visible || !issue.Complete || issue.IsPull || issue.IsClosed || issue.IsLocked || issue.ID != strconv.FormatInt(entry.IssueID, 10) || issue.Content != intentTaskBody(previous, task) || issue.Title != intentTaskTitle(task) || issue.ContentVersion != entry.ContentVersion {
			return p, nil, refuseAcceptance(RefusalObjectiveChanged)
		}
		title, body := issue.Title, issue.Content+"\nSuperseded by accepted product revision: "+r.ID+"\nThis task is retained as history and grants no current generated-work authority.\n"
		if next, ok := current[task.Key]; ok {
			title = intentTaskTitle(next)
			body = intentTaskBody(r, next)
		}
		p.Edits = append(p.Edits, sdk.IssueGraphEdit{IssueID: entry.IssueID, ExpectedTitle: issue.Title, ExpectedBody: issue.Content, ExpectedContentVersion: issue.ContentVersion, Title: title, Body: body})
		if native.Dependencies == nil || !native.Dependencies.Complete || len(native.Dependencies.Items) != len(task.DependsOn) {
			return p, nil, refuseAcceptance(RefusalEdgeChanged)
		}
		expected := map[int64]bool{}
		for _, dep := range task.DependsOn {
			expected[oldIssues[dep].IssueID] = true
		}
		for _, d := range native.Dependencies.Items {
			dep, e := strconv.ParseInt(d.DependencyID, 10, 64)
			if e != nil || !d.Visible || !d.Complete || !expected[dep] {
				return p, nil, refuseAcceptance(RefusalEdgeChanged)
			}
			delete(expected, dep)
			oldEdges[edge{entry.IssueID, dep}] = true
		}
		if len(expected) > 0 {
			return p, nil, refuseAcceptance(RefusalEdgeChanged)
		}
	}
	retained := map[edge]bool{}
	for _, task := range r.Tasks {
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
	for _, task := range previous.Tasks {
		for _, dep := range task.DependsOn {
			key := edge{oldIssues[task.Key].IssueID, oldIssues[dep].IssueID}
			if !retained[key] {
				p.Dependencies = append(p.Dependencies, sdk.IssueGraphDependency{Issue: sdk.IssueGraphRef{IssueID: key.a}, Prerequisite: sdk.IssueGraphRef{IssueID: key.b}, Remove: true})
			}
		}
	}
	if err = sdk.ValidateIssueGraphPayload(p); err != nil {
		return p, nil, fmt.Errorf("accepted decomposition exceeds native packet bounds: %w", err)
	}
	return p, ids, nil
}

func intentReceiptIssues(r factory.IntentRevision, g factory.IntentGeneration, receipt sdk.IssueGraphReceipt) (map[string]sdk.IssueGraphIssueReceipt, error) {
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
	if len(issues) != len(r.Tasks) {
		return nil, errors.New("native graph receipt does not bind accepted tasks")
	}
	for _, task := range r.Tasks {
		if issues[task.Key].IssueID == 0 {
			return nil, errors.New("native graph receipt missing accepted task")
		}
	}
	return issues, nil
}
