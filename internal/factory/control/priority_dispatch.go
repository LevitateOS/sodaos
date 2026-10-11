package control

import (
	"context"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// The complete queue is inspected before preference selection. The bound
// refuses an incomplete set rather than selecting from the first page.
const MaxPriorityControls = 4096

func completePriorityQueue(ctx context.Context, deps DispatchDeps) ([]factory.IssueControl, error) {
	var all []factory.IssueControl
	var seen, repository, issue int64
	hasCursor := false
	for {
		batch, err := deps.Store.DispatchControlsAfter(ctx, MaxDispatchVisits, seen, repository, issue, hasCursor)
		if err != nil {
			return nil, err
		}
		if len(all)+len(batch) > MaxPriorityControls {
			return nil, ErrPriorityUnavailable
		}
		all = append(all, batch...)
		if len(batch) < MaxDispatchVisits {
			return all, nil
		}
		last := batch[len(batch)-1]
		seen, repository, issue = last.FirstSeenUnix, last.Repository, last.Issue
		hasCursor = true
	}
}

func priorityCandidate(plan *attemptPlan) factory.PriorityCandidate {
	return factory.PriorityCandidate{
		ID:      fmt.Sprintf("%d/%d", plan.control.Repository, plan.control.Issue),
		Control: plan.control, Objective: plan.inputs.Issue.Title + "\n" + plan.inputs.Issue.Body,
		Acceptance:       plan.acceptance,
		AcceptedSections: append(promptSections(plan.acceptance.Sources, plan.inputs.Comments), promptSections(plan.acceptance.Resolutions, plan.inputs.Comments)...),
		Authority:        plan.effective.Authority, Preparation: plan.prep.Preparation.ID,
		PreparationRevision: plan.prep.Preparation.Revision, Source: plan.inputs.Tip, GateRevision: plan.gateRev, PlannedMinutes: plan.planned,
		PromptDigest: plan.promptSHA,
	}
}

func dispatchPriorityPass(ctx context.Context, deps DispatchDeps, report DispatchReport) DispatchReport {
	controls, err := completePriorityQueue(ctx, deps)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: "priority_incomplete", Detail: "complete candidate inventory unavailable"})
		return report
	}
	occupancy, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return report
	}
	var candidates []factory.PriorityCandidate
	plans := make(map[string]*attemptPlan)
	collect := deps
	collect.collectPlan = func(plan *attemptPlan) {
		candidate := priorityCandidate(plan)
		candidates = append(candidates, candidate)
		plans[candidate.ID] = plan
	}
	for _, control := range controls {
		if ctx.Err() != nil {
			report.Errors = append(report.Errors, DispatchError{Reason: "priority_incomplete", Detail: "candidate assessment interrupted"})
			return report
		}
		dispatchOne(ctx, collect, &occupancy, control, &report)
	}
	selected, err := selectPriority(ctx, deps.Priority, candidates, time.Now())
	if err != nil {
		for _, c := range candidates {
			report.Waits = append(report.Waits, DispatchWait{Repository: c.Control.Repository, Issue: c.Control.Issue, Reason: "priority_unavailable", Detail: "finite priority selection is unavailable"})
		}
		return report
	}
	if selected == "" {
		return report
	}
	chosen := plans[selected]
	// Re-assess the complete inventory after consultation, including new
	// candidates. A new alternative or changed relevant input makes the answer
	// stale. This does not execute or spend execution allowance.
	freshControls, err := completePriorityQueue(ctx, deps)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: "priority_incomplete"})
		return report
	}
	freshOccupancy, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore})
		return report
	}
	var freshCandidates []factory.PriorityCandidate
	fresh := deps
	fresh.collectPlan = func(plan *attemptPlan) { freshCandidates = append(freshCandidates, priorityCandidate(plan)) }
	silent := newDispatchReport()
	for _, control := range freshControls {
		dispatchOne(ctx, fresh, &freshOccupancy, control, &silent)
	}
	oldHash, _ := factory.PrioritySnapshot(candidates)
	newHash, hashErr := factory.PrioritySnapshot(freshCandidates)
	if hashErr != nil || oldHash != newHash {
		report.Waits = append(report.Waits, DispatchWait{Repository: chosen.control.Repository, Issue: chosen.control.Issue, Reason: WaitInputsStale, Detail: "complete priority candidate snapshot changed"})
		return report
	}
	deps.checkPlan = func(plan *attemptPlan) bool {
		before, _ := factory.PrioritySnapshot([]factory.PriorityCandidate{priorityCandidate(chosen)})
		after, _ := factory.PrioritySnapshot([]factory.PriorityCandidate{priorityCandidate(plan)})
		return before == after
	}
	dispatchOne(ctx, deps, &freshOccupancy, chosen.control, &report)
	return report
}
