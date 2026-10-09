// Assignment records bind one automatic coding dispatch to its eligible
// issue: the queued readiness it answered, the acceptance and authority it
// bound, the exact source/harness/preparation it launched, the prompt bytes
// it staged, and the result it recorded. Assigned work holds a resource
// reservation and a live run identity; finished work is terminal and keeps
// its result for inspection. These records never carry credentials: prompts
// embed accepted native text plus IDs and digests only.
package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// Assignment stages. Assigned work holds its reservation and may still
// launch (bounded attempts); finished work is terminal and immutable.
const (
	AssignmentAssigned = "assigned"
	AssignmentFinished = "finished"
)

// MaxDispatchAttempts bounds launch tries per assignment. A launch that
// fails with confirmed-unused capacity releases its reservation and may
// try again; an exhausted assignment finishes failed instead of retrying
// forever.
const MaxDispatchAttempts = 3

// ValidAssignmentStage reports whether stage is a known assignment stage.
func ValidAssignmentStage(stage string) bool {
	return stage == AssignmentAssigned || stage == AssignmentFinished
}

// Assignment finish reasons. Bounded codes: views expose why an
// assignment finished, never the evidence bytes behind it.
const (
	AssignReasonReported   = "harness_reported"
	AssignReasonNoReport   = "harness_no_report"
	AssignReasonRunFailed  = "run_failed"
	AssignReasonCancelled  = "run_cancelled"
	AssignReasonExhausted  = "launch_exhausted"
	AssignReasonWithdrawn  = "dispatch_withdrawn"
	AssignReasonSuperseded = "acceptance_superseded"
)

func validAssignReason(reason string) bool {
	switch reason {
	case AssignReasonReported, AssignReasonNoReport, AssignReasonRunFailed,
		AssignReasonCancelled, AssignReasonExhausted, AssignReasonWithdrawn,
		AssignReasonSuperseded:
		return true
	default:
		return false
	}
}

// Assignment is one durable automatic dispatch for a native issue. Run is
// the latest run identity; RunHistory lists every attempt's run identity
// in order. Result arrives at finish; the prompt bytes stay inspectable
// afterwards. ActorID binds the broker actor from the selected sponsorship
// grant to this assignment.
type Assignment struct {
	Authority             AuthorityRef      `json:"authority"`
	Result                *AssignmentResult `json:"result,omitempty"`
	Prompt                []byte            `json:"prompt"`
	RunHistory            []string          `json:"run_history"`
	ID                    string            `json:"id"`
	AttemptRoot           string            `json:"attempt_root"`
	PublicationAssignment string            `json:"publication_assignment"`
	ProjectID             string            `json:"project_id"`
	Role                  string            `json:"role"`
	Acceptance            string            `json:"acceptance"`
	Preparation           string            `json:"preparation"`
	Harness               string            `json:"harness"`
	HarnessVers           string            `json:"harness_version"`
	Model                 string            `json:"model"`
	Connection            string            `json:"connection"`
	SourceCommit          string            `json:"source_commit"`
	PromptSHA             string            `json:"prompt_sha"`
	Run                   string            `json:"run"`
	Stage                 string            `json:"stage"`
	Outcome               Outcome           `json:"outcome,omitempty"`
	Reason                string            `json:"reason,omitempty"`
	Repository            int64             `json:"repository,string"`
	Issue                 int64             `json:"issue,string"`
	Revision              int64             `json:"revision"`
	NativeRev             int64             `json:"native_revision"`
	Attempts              int               `json:"attempts"`
	CreatedUnix           int64             `json:"created_unix"`
	FinishedUnix          int64             `json:"finished_unix,omitempty"`
	ActorID               int64             `json:"actor_id,string"`
}

// Validate rejects malformed assignments. Assigned work carries no outcome
// or result; finished work carries both plus its reason.
func (a Assignment) Validate() error {
	if !ValidID(a.ID) || !ValidID(a.AttemptRoot) || !ValidID(a.PublicationAssignment) {
		return errors.New("invalid assignment identity")
	}
	if !ValidProjectID(a.ProjectID) || !project.ValidFactoryRole(a.Role) {
		return errors.New("invalid assignment address")
	}
	if a.Repository <= 0 || a.Issue <= 0 || a.Revision < 0 || a.NativeRev < 1 {
		return errors.New("invalid assignment scope")
	}
	if a.ActorID <= 0 {
		return errors.New("invalid assignment execution actor")
	}
	if !project.ValidDecisionID(a.Acceptance) {
		return errors.New("invalid assignment acceptance")
	}
	if !ValidPreparationRef(a.Preparation) {
		return errors.New("invalid assignment preparation")
	}
	if a.Harness == "" || len(a.Harness) > 128 || !project.ValidHarnessVersion(a.HarnessVers) {
		return errors.New("invalid assignment harness")
	}
	if a.Model == "" || len(a.Model) > 128 {
		return errors.New("invalid assignment model")
	}
	if a.Connection == "" || len(a.Connection) > 128 {
		return errors.New("invalid assignment connection")
	}
	if !ValidCommit(a.SourceCommit) {
		return errors.New("invalid assignment source")
	}
	if len(a.Prompt) == 0 || len(a.Prompt) > project.MaxFactoryPrompt {
		return errors.New("invalid assignment prompt size")
	}
	if sum := sha256.Sum256(a.Prompt); hex.EncodeToString(sum[:]) != a.PromptSHA {
		return errors.New("assignment prompt does not match its digest")
	}
	if a.Attempts < 1 || a.Attempts > MaxDispatchAttempts {
		return errors.New("invalid assignment attempt count")
	}
	if !ValidID(a.Run) || len(a.RunHistory) == 0 || len(a.RunHistory) > MaxDispatchAttempts {
		return errors.New("invalid assignment run history")
	}
	seen := make(map[string]bool, len(a.RunHistory))
	for _, run := range a.RunHistory {
		if !ValidID(run) || seen[run] {
			return errors.New("invalid assignment run history")
		}
		seen[run] = true
	}
	if a.RunHistory[len(a.RunHistory)-1] != a.Run {
		return errors.New("assignment run is not its latest attempt")
	}
	if len(a.RunHistory) != a.Attempts {
		return errors.New("assignment attempts do not match its run history")
	}
	switch a.Stage {
	case AssignmentAssigned:
		if a.Outcome != "" || a.Reason != "" || a.Result != nil || a.FinishedUnix != 0 {
			return errors.New("assigned work carries no outcome")
		}
	case AssignmentFinished:
		if !validOutcome(a.Outcome) || a.Outcome == "" || !validAssignReason(a.Reason) || a.FinishedUnix <= 0 {
			return errors.New("finished assignment lacks its outcome")
		}
		if a.Result == nil || a.Result.ValidateForAssignment(a.Role, a.SourceCommit) != nil {
			return errors.New("finished assignment lacks its recorded result")
		}
		if a.Result.AssignmentID != a.ID || a.Result.RunID != a.Run {
			return errors.New("assignment result does not match its attempt")
		}
	default:
		return errors.New("invalid assignment stage")
	}
	return nil
}

// ValidPreparationRef reports whether id is an admissible preparation
// reference for dispatch records.
func ValidPreparationRef(id string) bool {
	return project.ValidPreparationID(id)
}
