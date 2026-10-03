package api

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// This file owns the Spaces collection inventory: the bounded project
// page, the unsettled-first factory runs, their display bindings and the
// exact read failures behind them. Correctness and display share these
// queries; failed reads stay visibly incomplete while successful empty
// reads stay empty.

// spacesInventory is one bounded collection read. runsErr and viewsErr
// name the failed factory reads, if any; the surviving rows stay usable.
// unsettledCapped reports the unsettled listing hit its bound, so
// per-project unsettled counts are lower bounds rather than exact.
type spacesInventory struct {
	projects        []store.Project
	runs            []factory.Run
	views           map[string]factory.RunView
	runsErr         error
	viewsErr        error
	unsettledCapped bool
}

// spacesAssociationPage bounds one project-page fetch: 128 scanned
// associations plus one row proving more remain.
const spacesAssociationPage = 129

// spacesRunPage bounds each factory-run listing behind one collection
// read. Unsettled rows union with newest-first history below.
const spacesRunPage = 1000

// loadSpacesInventory reads one project page after an exclusive cursor
// plus the collection's factory rows and bindings. An empty cursor
// starts from the beginning. A project-page failure aborts the read;
// factory failures attach to the inventory instead so surviving rows
// and the project page remain usable behind an incomplete flag.
func (s *API) loadSpacesInventory(ctx context.Context, after string) (spacesInventory, error) {
	inv := spacesInventory{views: map[string]factory.RunView{}}
	projects, err := s.Store.SpaceProjectsAfter(ctx, after, spacesAssociationPage)
	if err != nil {
		return spacesInventory{}, err
	}
	inv.projects = projects
	outstanding, outstandingErr := s.Store.FactoryUnsettledRuns(ctx, spacesRunPage)
	history, historyErr := s.Store.FactoryRuns(ctx, spacesRunPage)
	if outstandingErr == nil && len(outstanding) == spacesRunPage {
		inv.unsettledCapped = true
	}
	switch {
	case outstandingErr != nil && historyErr != nil:
		inv.runsErr = outstandingErr
	case outstandingErr != nil:
		inv.runsErr = outstandingErr
		inv.runs = history
	case historyErr != nil:
		inv.runsErr = historyErr
		inv.runs = outstanding
	default:
		inv.runs = mergeSpacesRuns(outstanding, history)
	}
	listed, err := s.Store.FactoryRunViews(ctx, spacesRunPage)
	if err != nil {
		inv.viewsErr = err
	} else {
		for _, view := range listed {
			inv.views[view.RunID] = view
		}
	}
	return inv, nil
}

// mergeSpacesRuns unions outstanding and history rows for display:
// history stays newest-first, and outstanding rows missing from the
// bounded history tail up the end. Those rows are older than every
// history row by construction, so newest-first order holds across the
// union. Either input may be nil when its read failed.
func mergeSpacesRuns(outstanding, history []factory.Run) []factory.Run {
	merged := make([]factory.Run, 0, len(outstanding)+len(history))
	seen := make(map[string]bool, len(outstanding)+len(history))
	for _, run := range history {
		if seen[run.ID] {
			continue
		}
		seen[run.ID] = true
		merged = append(merged, run)
	}
	var extra []factory.Run
	for _, run := range outstanding {
		if seen[run.ID] {
			continue
		}
		seen[run.ID] = true
		extra = append(extra, run)
	}
	for i := len(extra) - 1; i >= 0; i-- {
		merged = append(merged, extra[i])
	}
	return merged
}

// spacesRunUsage tracks the shared display budget across every row of
// one collection: unsettled rows admit first up to the collection cap,
// and settled history fills only what live work leaves unused.
type spacesRunUsage struct {
	unsettled     int
	history       int
	historyBudget int
}

// spacesRunCap bounds published factory rows per collection.
const spacesRunCap = 64

// newSpacesRunUsage reserves display budget for the inventory's live
// work before any row maps its runs.
func newSpacesRunUsage(runs []factory.Run) *spacesRunUsage {
	total := 0
	for _, run := range runs {
		if !run.Reconciled {
			total++
		}
	}
	if total > spacesRunCap {
		total = spacesRunCap
	}
	return &spacesRunUsage{historyBudget: spacesRunCap - total}
}

// factoryComplete reports whether every factory read behind the
// inventory succeeded without hitting its bound.
func (inv spacesInventory) factoryComplete() bool {
	return inv.runsErr == nil && inv.viewsErr == nil && !inv.unsettledCapped
}

// runsKnown reports whether per-project unsettled counts are exact.
// A capped unsettled listing is a known lower bound, not an unknown.
func (inv spacesInventory) runsKnown() bool {
	return inv.runsErr == nil
}
