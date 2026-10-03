package control

import (
	"sync"

	"github.com/levitateos/sodaos/internal/factory"
)

// This file owns the fair-traversal cursors shared by readiness sweeping
// and dispatch visiting. Every bounded pass keeps its bound; the cursors
// only rotate the start position so repeated passes reach deep pages,
// repositories and queued work instead of revisiting the same prefix.
// Cursor state is process-local progress: durable sweep revisions and
// dispatch packets keep their existing store ownership.

// traversalState carries the readiness sweep cursors: the next start
// page per repository and the next start offset into enabled policies.
type traversalState struct {
	mu        sync.Mutex
	sweepPage map[int64]int
	sweepRepo int
}

// nextSweepPage returns the page the next sweep of one repository
// starts at. Unknown repositories start at the first page.
func (t *traversalState) nextSweepPage(repository int64) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if page := t.sweepPage[repository]; page >= 1 {
		return page
	}
	return 1
}

// advanceSweepPage continues the next sweep after the pages this one
// covered, or restarts at the first page once enumeration completed.
func (t *traversalState) advanceSweepPage(repository int64, next int, complete bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if complete {
		delete(t.sweepPage, repository)
		return
	}
	if t.sweepPage == nil {
		t.sweepPage = map[int64]int{}
	}
	t.sweepPage[repository] = next
}

// nextSweepRepo returns the enabled-policy offset the next
// all-repository sweep starts at.
func (t *traversalState) nextSweepRepo(total int) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if total <= 0 {
		return 0
	}
	return t.sweepRepo % total
}

// advanceSweepRepo continues the next all-repository sweep after the
// repositories this one visited, or restarts once every enabled
// repository was visited.
func (t *traversalState) advanceSweepRepo(visited, total int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if total <= 0 || total <= MaxSweepRepos {
		t.sweepRepo = 0
		return
	}
	t.sweepRepo = (t.sweepRepo + visited) % total
}

// DispatchQueueCursor is the fair-rotation position inside the
// deterministic oldest-first queued listing. A nil cursor disables
// rotation: the pass visits from the beginning and advances nothing.
type DispatchQueueCursor struct {
	mu        sync.Mutex
	set       bool
	firstSeen int64
	repo      int64
	issue     int64
}

// NewDispatchQueueCursor returns a cursor starting at the beginning of
// the queued listing.
func NewDispatchQueueCursor() *DispatchQueueCursor { return &DispatchQueueCursor{} }

// start reports the queued query position: the exclusive cursor when
// rotation already advanced, otherwise the beginning.
func (q *DispatchQueueCursor) start() (firstSeen, repo, issue int64, hasCursor bool) {
	if q == nil {
		return 0, 0, 0, false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.set {
		return 0, 0, 0, false
	}
	return q.firstSeen, q.repo, q.issue, true
}

// advance continues after the last visited control, or restarts at the
// beginning once a pass observed the end of the listing.
func (q *DispatchQueueCursor) advance(last factory.IssueControl, complete bool) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if complete {
		q.set = false
		return
	}
	q.set, q.firstSeen, q.repo, q.issue = true, last.FirstSeenUnix, last.Repository, last.Issue
}

// reset restarts rotation at the beginning of the listing.
func (q *DispatchQueueCursor) reset() {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.set = false
}
