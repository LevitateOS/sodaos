// Immutable plan task, lane and check contracts. Plans are validated before
// coding starts: unique keys, explicit dependencies in topological order,
// exclusive ownership or explicit serialization, bounded sizes and required
// acceptance coverage. No unresolved mandatory fact enters an executable
// plan. FW07 persists revisions and receipts; FW12 admits them.
package factory

import (
	"errors"
	"path"
	"regexp"
	"strings"
)

// Frozen D04 plan bounds.
const (
	MaxPlanTasks       = 64
	MaxPlanLanes       = 4
	MaxTaskOwnership   = 32
	MaxPlanChecks      = 16
	MaxInspectedRefs   = 64
	MaxPlanTextBytes   = 4 * 1024
	MaxPlanPathBytes   = 1024
	MaxCheckArgv       = 64
	MaxCheckArgvBytes  = 16 * 1024
	MaxEncodedPlan     = 64 * 1024
	MinCheckTimeoutSec = 1
	MaxCheckTimeoutSec = 600
)

var planKey = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// ValidPlanKey reports whether key is an admissible plan-local
// task, lane or check key.
func ValidPlanKey(key string) bool { return planKey.MatchString(key) }

// Check scopes admitted for verification.
const (
	CheckScopeTask        = "task"
	CheckScopeIntegration = "integration"
)

// CheckSpec is one reusable verification: executed directly, never through
// an implicit shell. An explicitly declared script interpreter is allowed.
// Environment and secret custody stays with the prepared Project. Required
// integration checks run on the final exact candidate even when component
// checks already passed.
type CheckSpec struct {
	ID             string   `json:"id"`
	Argv           []string `json:"argv"`
	Cwd            string   `json:"cwd"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Scope          string   `json:"scope"`
	Required       bool     `json:"required"`
}

func (c CheckSpec) Validate() error {
	if !ValidPlanKey(c.ID) {
		return errors.New("invalid check key")
	}
	if c.Scope != CheckScopeTask && c.Scope != CheckScopeIntegration {
		return errors.New("invalid check scope")
	}
	if c.TimeoutSeconds < MinCheckTimeoutSec || c.TimeoutSeconds > MaxCheckTimeoutSec {
		return errors.New("check timeout outside its bound")
	}
	if len(c.Argv) == 0 || len(c.Argv) > MaxCheckArgv {
		return errors.New("check argv outside its bound")
	}
	total := 0
	for _, arg := range c.Argv {
		if arg == "" || strings.ContainsRune(arg, 0) {
			return errors.New("invalid check argument")
		}
		total += len(arg)
	}
	if total > MaxCheckArgvBytes {
		return errors.New("check argv exceeds its bound")
	}
	if c.Cwd == "" {
		return errors.New("check needs its working directory")
	}
	if c.Cwd != "." && !validOwnershipEntry(c.Cwd) {
		return errors.New("check working directory is not a clean relative path")
	}
	return nil
}

// validOwnershipEntry reports whether entry is an exact relative file or an
// explicit directory prefix. Globs, absolute paths, parent escapes and
// version-control or runtime state directories never count as ownership.
func validOwnershipEntry(entry string) bool {
	if entry == "" || len(entry) > MaxPlanPathBytes {
		return false
	}
	// An explicit directory prefix ends in exactly one slash; an exact
	// file never does.
	base := strings.TrimSuffix(entry, "/")
	if base == "" || base != path.Clean(base) || strings.HasSuffix(base, "/") {
		return false
	}
	entry = base
	if path.IsAbs(entry) || entry == "." || entry == ".." || strings.HasPrefix(entry, "../") || strings.Contains(entry, "/../") {
		return false
	}
	if strings.ContainsAny(entry, "*?[]{}\\") || strings.ContainsRune(entry, 0) {
		return false
	}
	for _, part := range strings.Split(entry, "/") {
		if part == ".git" || part == ".soda-home" {
			return false
		}
	}
	return true
}

// PlanTask is one unit of planned work: unique key, objective, expected
// outputs, acceptance coverage, dependency keys, home lane, exclusive paths
// or interfaces, and check references. Dependencies name task keys, never
// ordinal positions.
type PlanTask struct {
	ID             string   `json:"id"`
	Objective      string   `json:"objective"`
	Outputs        []string `json:"outputs"`
	AcceptanceRefs []string `json:"acceptance_refs"`
	DependsOn      []string `json:"depends_on"`
	Lane           string   `json:"lane"`
	OwnedPaths     []string `json:"owned_paths"`
	InterfaceWrite []string `json:"interface_writes"`
	CheckIDs       []string `json:"check_ids"`
}

// PlanLane groups ordered task keys that run one at a time.
type PlanLane struct {
	ID      string   `json:"id"`
	TaskIDs []string `json:"task_ids"`
}

// ValidatePlanGraph checks one task/lane/check set: key shape and
// uniqueness, lane membership, topological dependency order, ownership
// exclusivity across potentially concurrent tasks, check references and
// frozen bounds. Lanes run one task at a time, so same-lane tasks are
// never concurrent; cross-lane tasks without an ordering dependency must
// not share ownership or interface writes.
func ValidatePlanGraph(tasks []PlanTask, lanes []PlanLane, checks []CheckSpec) error {
	if len(tasks) == 0 || len(tasks) > MaxPlanTasks {
		return errors.New("plan task count outside its bound")
	}
	if len(lanes) == 0 || len(lanes) > MaxPlanLanes {
		return errors.New("plan lane count outside its bound")
	}
	if len(checks) > MaxPlanChecks {
		return errors.New("plan check count outside its bound")
	}
	knownChecks := make(map[string]bool, len(checks))
	for _, c := range checks {
		if err := c.Validate(); err != nil {
			return err
		}
		if knownChecks[c.ID] {
			return errors.New("duplicate check key")
		}
		knownChecks[c.ID] = true
	}
	knownLanes := make(map[string]int, len(lanes))
	for i, lane := range lanes {
		if !ValidPlanKey(lane.ID) {
			return errors.New("invalid lane key")
		}
		if _, dup := knownLanes[lane.ID]; dup {
			return errors.New("duplicate lane key")
		}
		knownLanes[lane.ID] = i
	}
	byKey := make(map[string]int, len(tasks))
	laneOf := make(map[string]string, len(tasks))
	seenLaneTasks := make(map[string]bool)
	for _, lane := range lanes {
		for _, id := range lane.TaskIDs {
			if !ValidPlanKey(id) {
				return errors.New("lane references an invalid task key")
			}
			if seenLaneTasks[id] {
				return errors.New("task enters two lanes")
			}
			seenLaneTasks[id] = true
			laneOf[id] = lane.ID
		}
	}
	for i, task := range tasks {
		if !ValidPlanKey(task.ID) {
			return errors.New("invalid task key")
		}
		if _, dup := byKey[task.ID]; dup {
			return errors.New("duplicate task key")
		}
		byKey[task.ID] = i
		if laneOf[task.ID] == "" || laneOf[task.ID] != task.Lane {
			return errors.New("task lane does not match its lane membership")
		}
		if _, ok := knownLanes[task.Lane]; !ok {
			return errors.New("task names an unknown lane")
		}
		if task.Objective == "" || len(task.Objective) > MaxPlanTextBytes {
			return errors.New("task objective outside its bound")
		}
		if len(task.Outputs) == 0 {
			return errors.New("task lacks expected outputs")
		}
		for _, text := range append(append([]string(nil), task.Outputs...), task.AcceptanceRefs...) {
			if text == "" || len(text) > MaxPlanTextBytes {
				return errors.New("task output or acceptance reference outside its bound")
			}
		}
		if len(task.AcceptanceRefs) == 0 {
			return errors.New("task lacks acceptance coverage")
		}
		if len(task.OwnedPaths) > MaxTaskOwnership {
			return errors.New("task ownership exceeds its bound")
		}
		for _, entry := range task.OwnedPaths {
			if !validOwnershipEntry(entry) {
				return errors.New("task ownership is not an exact file or directory prefix")
			}
		}
		for _, key := range task.InterfaceWrite {
			if key == "" || len(key) > 256 {
				return errors.New("task interface write outside its bound")
			}
		}
		for _, id := range task.CheckIDs {
			if !knownChecks[id] {
				return errors.New("task references an unknown check")
			}
		}
		// Dependencies name earlier task keys: the array is topologically
		// ordered and every declared task is required.
		for _, dep := range task.DependsOn {
			pos, ok := byKey[dep]
			if !ok || dep == task.ID {
				return errors.New("task depends on an unknown task")
			}
			if pos > i {
				return errors.New("task dependency is not earlier in plan order")
			}
		}
	}
	if len(seenLaneTasks) != len(tasks) {
		return errors.New("lane membership does not cover every task")
	}
	// Serialize conflicting ownership across potentially concurrent tasks:
	// cross-lane pairs without an ordering dependency in either direction
	// must neither overlap paths nor share an interface write.
	ordered := func(a, b int) bool {
		seen := make(map[int]bool)
		var visit func(int)
		visit = func(n int) {
			if seen[n] {
				return
			}
			seen[n] = true
			for _, dep := range tasks[n].DependsOn {
				visit(byKey[dep])
			}
		}
		visit(a)
		if seen[b] {
			return true
		}
		seen = make(map[int]bool)
		visit(b)
		return seen[a]
	}
	overlap := func(a, b string) bool {
		return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") ||
			strings.HasPrefix(a, strings.TrimSuffix(b, "/")+"/") ||
			strings.HasPrefix(b, strings.TrimSuffix(a, "/")+"/")
	}
	for i := range tasks {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[i].Lane == tasks[j].Lane || ordered(i, j) {
				continue
			}
			for _, a := range tasks[i].OwnedPaths {
				for _, b := range tasks[j].OwnedPaths {
					if overlap(a, b) {
						return errors.New("concurrent tasks share owned paths without a dependency")
					}
				}
			}
			for _, a := range tasks[i].InterfaceWrite {
				for _, b := range tasks[j].InterfaceWrite {
					if a == b {
						return errors.New("concurrent tasks share an interface write without a dependency")
					}
				}
			}
		}
	}
	return nil
}
