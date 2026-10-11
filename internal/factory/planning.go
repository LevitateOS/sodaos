package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// PlanningInput retains the accepted stable scope and explicitly missing
// prerequisite outcomes. Provisional plans convey no permission to execute.
type PlanningInput struct {
	Candidate PriorityCandidate `json:"candidate"`
	Missing   []Blocker         `json:"missing,omitempty"`
}

type PlanningRecord struct {
	Snapshot    string         `json:"snapshot"`
	Input       PlanningInput  `json:"input"`
	Strategy    string         `json:"strategy"`
	Steps       []string       `json:"steps"`
	Provisional bool           `json:"provisional"`
	Outcome     string         `json:"outcome"`
	Judgment    PriorityAnswer `json:"judgment"`
	CreatedUnix int64          `json:"created_unix"`
}

func PlanningSnapshot(input PlanningInput) (string, error) {
	// Ordering observations are rechecked separately. Unrelated native writes
	// do not invalidate an otherwise identical implementation approach.
	input.Candidate.Control.NativeRev = 0
	input.Candidate.Control.AssessedUnix = 0
	data, err := json.Marshal(struct {
		Kind  string
		Input PlanningInput
	}{"bounded-planning-v1", input})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (r PlanningRecord) Validate() error {
	snapshot, err := PlanningSnapshot(r.Input)
	if err != nil || snapshot != r.Snapshot || r.CreatedUnix <= 0 || r.Input.Candidate.Control.Repository <= 0 || r.Input.Candidate.Control.Issue <= 0 {
		return errors.New("invalid planning record")
	}
	if r.Outcome != "planned" && r.Outcome != "waiting" && r.Outcome != "unavailable" {
		return errors.New("invalid planning outcome")
	}
	if r.Provisional != (len(r.Input.Missing) > 0) {
		return errors.New("invalid provisional planning state")
	}
	return nil
}

// ReadyToPlan permits only accepted, authorized scope with visible pending
// outcomes. Inaccessible evidence, cycles and stale acceptance are code refusals.
func ReadyToPlan(control IssueControl) bool {
	if control.Acceptance == "" {
		return false
	}
	if control.Readiness == ReadinessQueued {
		return true
	}
	if control.Readiness != ReadinessBlocked || len(control.Blockers) == 0 {
		return false
	}
	for _, b := range control.Blockers {
		if b.Code != BlockerCodePending && b.Code != BlockerResultPending {
			return false
		}
	}
	return true
}
