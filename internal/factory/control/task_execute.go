package control

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// FW14 ready-task execution against persisted plan task receipts (D05).
//
// This driver is the sole Go writer of plan task state. It folds verified
// typed task events — the executor's observations imported from the
// per-run journal — into receipt advances and release decisions. Run-output
// prose has no path here: ordinary model text never advances task state,
// and commit/tree evidence binds only from typed task_committed detail the
// executor observed. Checkpoint CAS and PR publication stay FW15 joints;
// this driver consumes their future events without performing them.
//
// The fold is deterministic over (plan, stored receipts, event page), so a
// restart observes receipts instead of redoing completed writes: an exact
// replay emits no writes, and SavePlanTaskReceipt replays are no-ops
// behind CAS. Polling and persistence stay with the future progress loop
// (FW24 joint); this file takes loaded receipts in and returns exact
// receipt writes out.

// Task event import errors. Binding and order violations fence the run:
// the caller refuses advancement instead of guessing past a gap or a
// foreign identity.
var (
	ErrTaskEventGap        = errors.New("task event sequence gap refuses advancement")
	ErrTaskEventBinding    = errors.New("task event does not bind its run")
	ErrTaskEventUnknown    = errors.New("task event names an unknown plan task")
	ErrTaskEventTransition = errors.New("task event violates its task lifecycle")
)

// TaskEventImport binds one verified journal page to the run that produced
// it. Events arrive in journal sequence order. AttemptRoot and PlanID name
// the immutable plan revision the receipts belong to; Invocation and
// PlanDigest rebind every event to the admitted run (mirroring the
// executor's record rule at import); MaxChildren caps aggregate releases.
type TaskEventImport struct {
	AttemptRoot string
	PlanID      string
	Invocation  string
	PlanDigest  string
	MaxChildren int
	Events      []project.TaskEvent
}

// TaskExecutionDecisions is what one fold releases or retires. NewlyReady
// names releasable tasks in plan order once their prerequisites complete —
// a completion event prompts immediate downstream release while unrelated
// lanes stay active. StopRun retires the run: no new task admission and
// RetireChildren names live native children the executor must stop. A
// failed prerequisite blocks its descendants; the failure itself stops the
// whole run per D05.
type TaskExecutionDecisions struct {
	NewlyReady     []string
	StopRun        bool
	RetireChildren []string
	LimitViolation bool
}

// TaskExecutionResult carries exact receipt writes plus decisions.
// NextSequence is the journal cursor after the fold: the highest applied
// or replayed sequence, for the caller's next page.
type TaskExecutionResult struct {
	Writes       []store.PlanTaskReceipt
	Decisions    TaskExecutionDecisions
	NextSequence int64
	Stopped      bool
	Contained    bool
}

// taskEventRef records one applied event on its receipt. The suffix keeps
// the import cursor restart-safe without a second table.
func taskEventRef(event project.TaskEvent) string {
	return event.Type + "@" + strconv.FormatInt(event.Sequence, 10)
}

// importCursor recovers the journal cursor from stored receipts: the
// highest applied event sequence. Refs the driver never wrote are ignored
// for the cursor; receipt content itself is still folded honestly.
func importCursor(receipts []store.PlanTaskReceipt) int64 {
	var cursor int64
	for _, receipt := range receipts {
		for _, ref := range receipt.Events {
			at := strings.LastIndex(ref, "@")
			if at < 0 {
				continue
			}
			seq, err := strconv.ParseInt(ref[at+1:], 10, 64)
			if err != nil || seq <= 0 {
				continue
			}
			if seq > cursor {
				cursor = seq
			}
		}
	}
	return cursor
}

// journalStopped reports whether a run_stopped event was ever applied.
func journalStopped(receipts []store.PlanTaskReceipt) bool {
	for _, receipt := range receipts {
		for _, ref := range receipt.Events {
			if strings.HasPrefix(ref, project.TaskEventRunStopped+"@") {
				return true
			}
		}
	}
	return false
}

// ImportTaskEvents folds one verified event page into receipt writes and
// release decisions. Missing receipts seed as pending; identical replays
// emit nothing; conflicting bytes, gaps, foreign identities and lifecycle
// violations refuse with the sentinel above.
func ImportTaskEvents(plan factory.InvestigationPlan, receipts []store.PlanTaskReceipt, imp TaskEventImport) (TaskExecutionResult, error) {
	var result TaskExecutionResult
	if err := plan.Validate(); err != nil {
		return result, err
	}
	if !factory.ValidID(imp.AttemptRoot) || !factory.ValidID(imp.PlanID) ||
		plan.AttemptRoot != imp.AttemptRoot || plan.ID != imp.PlanID {
		return result, errors.New("task event import names an unknown plan revision")
	}
	if imp.MaxChildren < project.MinTaskChildren || imp.MaxChildren > project.MaxTaskChildren {
		return result, errors.New("task event import child limit is outside its range")
	}
	if imp.Invocation == "" || len(imp.Invocation) > 128 {
		return result, errors.New("task event import invocation is invalid")
	}
	if !factory.ValidDigest(imp.PlanDigest) {
		return result, errors.New("task event import plan digest is invalid")
	}
	byTask := make(map[string]int, len(plan.Tasks))
	for i, task := range plan.Tasks {
		byTask[task.ID] = i
	}
	stored := make(map[string]store.PlanTaskReceipt, len(receipts))
	for _, receipt := range receipts {
		if receipt.AttemptRoot != imp.AttemptRoot || receipt.PlanID != imp.PlanID {
			return result, errors.New("task event import mixes plan revisions")
		}
		if _, ok := byTask[receipt.TaskID]; !ok {
			return result, ErrTaskEventUnknown
		}
		if _, dup := stored[receipt.TaskID]; dup {
			return result, errors.New("task event import carries duplicate receipts")
		}
		stored[receipt.TaskID] = receipt
	}
	next := importCursor(receipts)
	stopped := journalStopped(receipts)
	// Seed pending receipts for tasks the store never saw. Identical
	// replays of an existing receipt emit nothing below.
	pending := make(map[string]store.PlanTaskReceipt, len(plan.Tasks))
	for taskID, receipt := range stored {
		pending[taskID] = receipt
	}
	for _, task := range plan.Tasks {
		if _, ok := pending[task.ID]; ok {
			continue
		}
		seed := store.PlanTaskReceipt{
			AttemptRoot: imp.AttemptRoot, PlanID: imp.PlanID, TaskID: task.ID,
			State: store.PlanTaskPending,
		}
		if err := seed.Validate(); err != nil {
			return result, err
		}
		pending[task.ID] = seed
		result.Writes = append(result.Writes, seed)
	}
	// emit stages one receipt write when it changes stored state. The
	// write carries the stored CAS base; identical content emits nothing,
	// so replays never duplicate writes.
	emit := func(next store.PlanTaskReceipt) {
		if receiptsEqual(pending[next.TaskID], next) {
			return
		}
		pending[next.TaskID] = next
		result.Writes = append(result.Writes, next)
	}
	for _, event := range imp.Events {
		if err := event.Validate(); err != nil {
			return result, err
		}
		if event.Invocation != imp.Invocation || event.PlanDigest != imp.PlanDigest {
			return result, ErrTaskEventBinding
		}
		if event.Sequence <= next {
			if !eventReplayed(pending, event) {
				return result, ErrTaskEventGap
			}
			continue
		}
		if event.Sequence != next+1 {
			return result, ErrTaskEventGap
		}
		if stopped && event.Type != project.TaskEventRunStopped {
			return result, ErrTaskEventTransition
		}
		advanced, err := applyTaskEvent(plan, pending, byTask, event)
		if err != nil {
			return result, err
		}
		for _, write := range advanced {
			emit(write)
		}
		if event.Type == project.TaskEventRunStopped {
			stopped = true
		}
		next = event.Sequence
	}
	result.NextSequence = next
	result.Stopped = stopped
	// Failure containment and release compute over the folded state.
	failed := failedTasks(plan, pending)
	contained := len(failed) > 0
	if contained {
		cancelBlocked(plan, pending, failed, emit)
	}
	// A scheduler violation is recorded truthfully and flagged, and
	// releases freeze: the journal stays authoritative while the caller
	// fences the over-limit run instead of adding load to it.
	violated := overTaskLimits(plan, pending, imp.MaxChildren)
	decisions := TaskExecutionDecisions{}
	if contained {
		decisions.StopRun = true
		decisions.RetireChildren = activeChildren(plan, pending)
	} else if stopped {
		decisions.StopRun = true
	} else if !violated {
		decisions.NewlyReady = readyPlanTasks(plan, pending, imp.MaxChildren)
	}
	if violated {
		decisions.LimitViolation = true
	}
	result.Decisions = decisions
	result.Contained = contained
	return result, nil
}

// receiptsEqual compares two receipts ignoring no field: the caller sets
// the CAS base before calling, so equality means the store would no-op.
func receiptsEqual(a, b store.PlanTaskReceipt) bool {
	if a.Revision != b.Revision {
		return false
	}
	aa, bb := a, b
	aa.Revision, bb.Revision = 0, 0
	return aa.AttemptRoot == bb.AttemptRoot && aa.PlanID == bb.PlanID &&
		aa.TaskID == bb.TaskID && aa.State == bb.State && aa.Child == bb.Child &&
		aa.InputDigest == bb.InputDigest && aa.OutputDigest == bb.OutputDigest &&
		aa.Tree == bb.Tree && aa.Commit == bb.Commit && aa.Reason == bb.Reason &&
		strings.Join(aa.Events, "\x00") == strings.Join(bb.Events, "\x00") &&
		strings.Join(aa.Checks, "\x00") == strings.Join(bb.Checks, "\x00")
}

// eventReplayed reports whether an old sequence is already recorded on its
// task receipts. A lost reply replays the recorded bytes; anything else
// claiming an old sequence is a gap in our own evidence.
func eventReplayed(pending map[string]store.PlanTaskReceipt, event project.TaskEvent) bool {
	if event.Type == project.TaskEventRunStopped {
		for _, receipt := range pending {
			for _, ref := range receipt.Events {
				if ref == taskEventRef(event) {
					return true
				}
			}
		}
		return false
	}
	receipt, ok := pending[event.TaskKey]
	if !ok {
		return false
	}
	for _, ref := range receipt.Events {
		if ref == taskEventRef(event) {
			return true
		}
	}
	return false
}

// applyTaskEvent advances receipts for one fresh event. Terminal
// run_stopped cancels every non-terminal task; other types touch their
// named task only.
func applyTaskEvent(plan factory.InvestigationPlan, pending map[string]store.PlanTaskReceipt, byTask map[string]int, event project.TaskEvent) ([]store.PlanTaskReceipt, error) {
	if event.Type == project.TaskEventRunStopped {
		if _, ok := byTask[event.TaskKey]; !ok {
			return nil, ErrTaskEventUnknown
		}
		var writes []store.PlanTaskReceipt
		for _, task := range plan.Tasks {
			receipt := pending[task.ID]
			if terminalReceipt(receipt.State) {
				continue
			}
			next := receipt
			next.State = store.PlanTaskCancelled
			next.Reason = boundSummary(event.Detail, "run stopped")
			next.Events = appendRef(next.Events, taskEventRef(event))
			if err := next.Validate(); err != nil {
				return nil, err
			}
			writes = append(writes, next)
		}
		return writes, nil
	}
	if _, ok := byTask[event.TaskKey]; !ok {
		return nil, ErrTaskEventUnknown
	}
	receipt := pending[event.TaskKey]
	ref := taskEventRef(event)
	if terminalReceipt(receipt.State) {
		return nil, ErrTaskEventTransition
	}
	switch event.Type {
	case project.TaskEventChildStartIntent:
		next := receipt
		next.Events = appendRef(next.Events, ref)
		if err := next.Validate(); err != nil {
			return nil, err
		}
		return []store.PlanTaskReceipt{next}, nil
	case project.TaskEventChildStarted:
		if event.ChildID == "" {
			return nil, fmt.Errorf("%w: task child start lacks its native identity", ErrTaskEventTransition)
		}
		if receipt.State != store.PlanTaskPending {
			if receipt.Child != event.ChildID {
				return nil, fmt.Errorf("%w: task already has its child", ErrTaskEventTransition)
			}
			next := receipt
			next.Events = appendRef(next.Events, ref)
			if err := next.Validate(); err != nil {
				return nil, err
			}
			return []store.PlanTaskReceipt{next}, nil
		}
		next := receipt
		next.State = store.PlanTaskRunning
		next.Child = event.ChildID
		next.Events = appendRef(next.Events, ref)
		if err := next.Validate(); err != nil {
			return nil, err
		}
		return []store.PlanTaskReceipt{next}, nil
	case project.TaskEventChildReported:
		if !factory.ValidDigest(event.Detail) {
			return nil, fmt.Errorf("%w: task report lacks its output digest", ErrTaskEventTransition)
		}
		if receipt.State != store.PlanTaskRunning {
			return nil, fmt.Errorf("%w: task reported without its running child", ErrTaskEventTransition)
		}
		if event.ChildID != "" && event.ChildID != receipt.Child {
			return nil, fmt.Errorf("%w: task report names a foreign child", ErrTaskEventTransition)
		}
		if receipt.OutputDigest != "" && receipt.OutputDigest != event.Detail {
			return nil, fmt.Errorf("%w: task reported twice with different outputs", ErrTaskEventTransition)
		}
		next := receipt
		next.State = store.PlanTaskOutputReady
		next.OutputDigest = event.Detail
		next.Events = appendRef(next.Events, ref)
		if err := next.Validate(); err != nil {
			return nil, err
		}
		return []store.PlanTaskReceipt{next}, nil
	case project.TaskEventCheckpointReady:
		if receipt.State != store.PlanTaskRunning && receipt.State != store.PlanTaskOutputReady {
			return nil, fmt.Errorf("%w: checkpoint precedes its task execution", ErrTaskEventTransition)
		}
		if event.ChildID != "" && receipt.Child != "" && event.ChildID != receipt.Child {
			return nil, fmt.Errorf("%w: checkpoint names a foreign child", ErrTaskEventTransition)
		}
		checks, err := parseCheckEvidence(plan, event.Detail)
		if err != nil {
			return nil, err
		}
		next := receipt
		next.Checks = unionKeys(next.Checks, checks)
		next.Events = appendRef(next.Events, ref)
		if err := next.Validate(); err != nil {
			return nil, err
		}
		return []store.PlanTaskReceipt{next}, nil
	case project.TaskEventTaskCommitted:
		if receipt.State != store.PlanTaskOutputReady || receipt.OutputDigest == "" {
			return nil, fmt.Errorf("%w: task committed without its reported output", ErrTaskEventTransition)
		}
		if event.ChildID != "" && event.ChildID != receipt.Child {
			return nil, fmt.Errorf("%w: task commit names a foreign child", ErrTaskEventTransition)
		}
		tree, commit, err := parseCommitEvidence(event.Detail)
		if err != nil {
			return nil, err
		}
		next := receipt
		next.State = store.PlanTaskCompleted
		next.Tree, next.Commit = tree, commit
		next.Reason = boundSummary("task output accepted", "task output accepted")
		next.Events = appendRef(next.Events, ref)
		if err := next.Validate(); err != nil {
			return nil, err
		}
		return []store.PlanTaskReceipt{next}, nil
	case project.TaskEventTaskFailed:
		if receipt.State != store.PlanTaskRunning && receipt.State != store.PlanTaskOutputReady {
			return nil, fmt.Errorf("%w: task failed without its running child", ErrTaskEventTransition)
		}
		if event.ChildID != "" && event.ChildID != receipt.Child {
			return nil, fmt.Errorf("%w: task failure names a foreign child", ErrTaskEventTransition)
		}
		next := receipt
		next.State = store.PlanTaskFailed
		next.Reason = boundSummary(event.Detail, "task reported failure")
		next.Events = appendRef(next.Events, ref)
		if err := next.Validate(); err != nil {
			return nil, err
		}
		return []store.PlanTaskReceipt{next}, nil
	default:
		return nil, fmt.Errorf("%w: unknown task event type", ErrTaskEventTransition)
	}
}

// parseCheckEvidence reads checkpoint check evidence: space-separated plan
// check keys the executor verified. Empty detail carries none.
func parseCheckEvidence(plan factory.InvestigationPlan, detail string) ([]string, error) {
	if strings.TrimSpace(detail) == "" {
		return nil, nil
	}
	known := make(map[string]bool, len(plan.Checks))
	for _, check := range plan.Checks {
		known[check.ID] = true
	}
	var checks []string
	for _, id := range strings.Fields(detail) {
		if !factory.ValidPlanKey(id) || !known[id] {
			return nil, fmt.Errorf("%w: checkpoint names an unknown plan check", ErrTaskEventTransition)
		}
		checks = unionKeys(checks, []string{id})
	}
	return checks, nil
}

// parseCommitEvidence reads task_committed evidence: space-separated
// tree=<tree> and commit=<commit> pairs with 40-hex identities. Empty
// detail accepts a verification-only task with no commit; anything else
// must parse exactly, so model prose can never forge commit evidence.
func parseCommitEvidence(detail string) (tree, commit string, err error) {
	if strings.TrimSpace(detail) == "" {
		return "", "", nil
	}
	seen := make(map[string]bool)
	for _, field := range strings.Fields(detail) {
		key, value, found := strings.Cut(field, "=")
		if !found || seen[key] {
			return "", "", fmt.Errorf("%w: task commit evidence is malformed", ErrTaskEventTransition)
		}
		seen[key] = true
		if !factory.ValidCommit(value) {
			return "", "", fmt.Errorf("%w: task commit evidence lacks its identity", ErrTaskEventTransition)
		}
		switch key {
		case "tree":
			tree = value
		case "commit":
			commit = value
		default:
			return "", "", fmt.Errorf("%w: task commit evidence is malformed", ErrTaskEventTransition)
		}
	}
	return tree, commit, nil
}

// appendRef records one event ref once; replays stay byte-identical.
func appendRef(refs []string, ref string) []string {
	for _, existing := range refs {
		if existing == ref {
			return refs
		}
	}
	return append(refs, ref)
}

// unionKeys merges plan key sets without duplicates.
func unionKeys(into, from []string) []string {
	seen := make(map[string]bool, len(into)+len(from))
	var out []string
	for _, id := range into {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range from {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func terminalReceipt(state string) bool {
	return state == store.PlanTaskCompleted || state == store.PlanTaskFailed || state == store.PlanTaskCancelled
}

// failedTasks lists failed task keys in plan order.
func failedTasks(plan factory.InvestigationPlan, pending map[string]store.PlanTaskReceipt) []string {
	var failed []string
	for _, task := range plan.Tasks {
		if pending[task.ID].State == store.PlanTaskFailed {
			failed = append(failed, task.ID)
		}
	}
	return failed
}

// cancelBlocked cancels pending tasks whose prerequisites can never
// complete: anything transitively depending on a failed task. The reason
// names the nearest failed prerequisite in plan order, so a restart
// observes the block instead of releasing the descendant.
func cancelBlocked(plan factory.InvestigationPlan, pending map[string]store.PlanTaskReceipt, failed []string, emit func(store.PlanTaskReceipt)) {
	blocked := make(map[string]bool, len(failed))
	for _, id := range failed {
		blocked[id] = true
	}
	changed := true
	for changed {
		changed = false
		for _, task := range plan.Tasks {
			if blocked[task.ID] {
				continue
			}
			for _, dep := range task.DependsOn {
				if blocked[dep] {
					blocked[task.ID] = true
					changed = true
					break
				}
			}
		}
	}
	for _, task := range plan.Tasks {
		if !blocked[task.ID] {
			continue
		}
		receipt := pending[task.ID]
		if receipt.State != store.PlanTaskPending {
			continue
		}
		nearest := ""
		for _, dep := range task.DependsOn {
			if blocked[dep] {
				nearest = dep
				break
			}
		}
		if nearest == "" {
			for _, id := range failed {
				nearest = id
				break
			}
		}
		next := receipt
		next.State = store.PlanTaskCancelled
		next.Reason = "prerequisite " + nearest + " failed"
		if err := next.Validate(); err != nil {
			continue
		}
		emit(next)
	}
}

// readyPlanTasks lists releasable tasks in plan order: pending with every
// dependency completed, a free home lane and aggregate room. Plan order is
// already topological, so iteration order is the specified
// plan-order/key release order.
func readyPlanTasks(plan factory.InvestigationPlan, pending map[string]store.PlanTaskReceipt, maxChildren int) []string {
	laneHeld := make(map[string]bool)
	active := 0
	for _, task := range plan.Tasks {
		switch pending[task.ID].State {
		case store.PlanTaskRunning, store.PlanTaskOutputReady:
			laneHeld[task.Lane] = true
			active++
		}
	}
	var ready []string
	for _, task := range plan.Tasks {
		if pending[task.ID].State != store.PlanTaskPending {
			continue
		}
		blocked := false
		for _, dep := range task.DependsOn {
			if pending[dep].State != store.PlanTaskCompleted {
				blocked = true
				break
			}
		}
		if blocked || laneHeld[task.Lane] || active >= maxChildren {
			continue
		}
		laneHeld[task.Lane] = true
		active++
		ready = append(ready, task.ID)
	}
	return ready
}

// activeChildren names live native children for retirement ordering.
func activeChildren(plan factory.InvestigationPlan, pending map[string]store.PlanTaskReceipt) []string {
	var children []string
	for _, task := range plan.Tasks {
		receipt := pending[task.ID]
		if receipt.State == store.PlanTaskRunning || receipt.State == store.PlanTaskOutputReady {
			children = append(children, receipt.Child)
		}
	}
	return children
}

// overTaskLimits reports a scheduler violation the journal recorded
// anyway: more active tasks than admitted, or two actives in one lane.
func overTaskLimits(plan factory.InvestigationPlan, pending map[string]store.PlanTaskReceipt, maxChildren int) bool {
	active := 0
	lanes := make(map[string]int)
	for _, task := range plan.Tasks {
		switch pending[task.ID].State {
		case store.PlanTaskRunning, store.PlanTaskOutputReady:
			active++
			lanes[task.Lane]++
			if lanes[task.Lane] > 1 {
				return true
			}
		}
	}
	return active > maxChildren
}
