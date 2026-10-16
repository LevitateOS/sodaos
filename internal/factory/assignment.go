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
	"strconv"
	"strings"

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

// Subject kinds admitted for execution. Exactly one payload matches.
const (
	SubjectIssue  = "issue"
	SubjectIntent = "intent"
	// SubjectCompletion names one verified completion's reassessment cycle
	// (D13). It accepts only the reassess phase and the coder role.
	SubjectCompletion = "completion"
)

// IssueSubject binds one native issue attempt to the acceptance decision
// that authorized it.
type IssueSubject struct {
	Issue      int64  `json:"issue,string"`
	Acceptance string `json:"acceptance"`
}

// IntentSubject binds one intent planning attempt to its exact reviewed
// inputs. The owner is validated from the authoritative row at admission,
// never accepted from JSON.
type IntentSubject struct {
	IntentID    string `json:"intent_id"`
	RevisionID  string `json:"revision_id"`
	ScopeDigest string `json:"scope_digest"`
	OwnerID     int64  `json:"owner_id,string"`
}

// CompletionSubject binds one reassessment attempt to its verified
// completion and bounded cycle: the completion event, the immutable cycle
// and the sealed inventory digest the report must account for.
type CompletionSubject struct {
	CompletionID    string `json:"completion_id"`
	CycleID         string `json:"cycle_id"`
	InventoryDigest string `json:"inventory_digest"`
}

// ExecutionSubject is the canonical admitted work item: a native issue, a
// pre-issue intent draft or a D13 verified completion. Subject keys are
// `issue:<repository>:<issue>`, `intent:<intent_id>` and
// `completion:<completion_id>`.
type ExecutionSubject struct {
	Issue      *IssueSubject      `json:"issue,omitempty"`
	Intent     *IntentSubject     `json:"intent,omitempty"`
	Completion *CompletionSubject `json:"completion,omitempty"`
	Kind       string             `json:"kind"`
}

// SubjectKey returns the canonical admission key for subject.
func (s ExecutionSubject) SubjectKey(repository int64) string {
	if s.Kind == SubjectIntent && s.Intent != nil {
		return "intent:" + s.Intent.IntentID
	}
	if s.Kind == SubjectIssue && s.Issue != nil {
		return "issue:" + strconv.FormatInt(repository, 10) + ":" + strconv.FormatInt(s.Issue.Issue, 10)
	}
	if s.Kind == SubjectCompletion && s.Completion != nil {
		return "completion:" + s.Completion.CompletionID
	}
	return ""
}

// ParseSubjectKey splits a canonical subject key into kind, repository and
// issue/intent/completion identity. Intent and completion keys carry no
// repository segment.
func ParseSubjectKey(key string) (kind string, repository, issue int64, intent, completion string, err error) {
	if rest, ok := strings.CutPrefix(key, "intent:"); ok {
		if !project.ValidDecisionID(rest) {
			return "", 0, 0, "", "", errors.New("invalid intent subject key")
		}
		return SubjectIntent, 0, 0, rest, "", nil
	}
	if rest, ok := strings.CutPrefix(key, "completion:"); ok {
		if !ValidID(rest) {
			return "", 0, 0, "", "", errors.New("invalid completion subject key")
		}
		return SubjectCompletion, 0, 0, "", rest, nil
	}
	rest, ok := strings.CutPrefix(key, "issue:")
	if !ok {
		return "", 0, 0, "", "", errors.New("unknown subject key")
	}
	repo, num, found := strings.Cut(rest, ":")
	if !found {
		return "", 0, 0, "", "", errors.New("invalid issue subject key")
	}
	repository, err = strconv.ParseInt(repo, 10, 64)
	if err != nil || repository <= 0 {
		return "", 0, 0, "", "", errors.New("invalid issue subject key")
	}
	issue, err = strconv.ParseInt(num, 10, 64)
	if err != nil || issue <= 0 || strings.Trim(num, "0123456789") != "" {
		return "", 0, 0, "", "", errors.New("invalid issue subject key")
	}
	return SubjectIssue, repository, issue, "", "", nil
}

func (s ExecutionSubject) Validate() error {
	switch s.Kind {
	case SubjectIssue:
		if s.Issue == nil || s.Intent != nil || s.Completion != nil {
			return errors.New("issue subject needs exactly its issue payload")
		}
		if s.Issue.Issue <= 0 || !project.ValidDecisionID(s.Issue.Acceptance) {
			return errors.New("invalid issue subject")
		}
	case SubjectIntent:
		if s.Intent == nil || s.Issue != nil || s.Completion != nil {
			return errors.New("intent subject needs exactly its intent payload")
		}
		if !project.ValidDecisionID(s.Intent.IntentID) || !project.ValidDecisionID(s.Intent.RevisionID) {
			return errors.New("invalid intent subject identity")
		}
		if !ValidDigest(s.Intent.ScopeDigest) || s.Intent.OwnerID <= 0 {
			return errors.New("invalid intent subject scope or owner")
		}
	case SubjectCompletion:
		if s.Completion == nil || s.Issue != nil || s.Intent != nil {
			return errors.New("completion subject needs exactly its completion payload")
		}
		if !ValidID(s.Completion.CompletionID) || !ValidID(s.Completion.CycleID) {
			return errors.New("invalid completion subject identity")
		}
		if !ValidDigest(s.Completion.InventoryDigest) {
			return errors.New("invalid completion subject inventory digest")
		}
	default:
		return errors.New("unknown execution subject kind")
	}
	return nil
}

// Assignment is one durable automatic dispatch for an admitted subject. Run
// is the latest run identity; RunHistory lists every attempt's run identity
// in order. Result arrives at finish; the prompt bytes stay inspectable
// afterwards. ActorID binds the broker actor from the selected sponsorship
// grant to this assignment. Phase names the admitted execution phase and
// ParentAssignment the immediate parent; a root has no parent and keeps its
// own AttemptRoot. A D13 reassessment cycle is always a fresh root with no
// parent and no publication assignment. New admission always records
// Subject, Phase and ParentAssignment; pre-phase issue records without them
// validate for readback until FW11 migrates admission.
type Assignment struct {
	Authority             AuthorityRef      `json:"authority"`
	Result                *AssignmentResult `json:"result,omitempty"`
	Subject               ExecutionSubject  `json:"subject,omitempty"`
	Prompt                []byte            `json:"prompt"`
	RunHistory            []string          `json:"run_history"`
	ID                    string            `json:"id"`
	AttemptRoot           string            `json:"attempt_root"`
	ParentAssignment      string            `json:"parent_assignment,omitempty"`
	PublicationAssignment string            `json:"publication_assignment"`
	ProjectID             string            `json:"project_id"`
	Role                  string            `json:"role"`
	Phase                 string            `json:"phase,omitempty"`
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

// validateSubjectPhase enforces the D01 admission invariants for assignments
// carrying an explicit subject.
func (a Assignment) validateSubjectPhase() error {
	if err := a.Subject.Validate(); err != nil {
		return err
	}
	if !project.ValidExecutionPhase(a.Phase) || a.Phase == "" {
		return errors.New("subject assignment lacks its execution phase")
	}
	if !SubjectKindAllowed(a.Subject.Kind, a.Phase) {
		return errors.New("subject kind cannot enter its execution phase")
	}
	if !ValidPhaseRole(a.Phase, a.Role) {
		return errors.New("assignment role does not match its execution phase")
	}
	if a.ParentAssignment == "" {
		if a.AttemptRoot != a.ID {
			return errors.New("root assignment must carry its own attempt root")
		}
	} else {
		if !ValidID(a.ParentAssignment) || a.ParentAssignment == a.ID {
			return errors.New("invalid assignment parent")
		}
		if !ValidID(a.AttemptRoot) || a.AttemptRoot == a.ID {
			return errors.New("child assignment must keep its attempt root")
		}
	}
	switch a.Subject.Kind {
	case SubjectIssue:
		if a.Repository <= 0 || a.Issue <= 0 || a.Issue != a.Subject.Issue.Issue {
			return errors.New("issue assignment does not match its subject")
		}
		if a.Acceptance != a.Subject.Issue.Acceptance {
			return errors.New("issue assignment acceptance does not match its subject")
		}
	case SubjectIntent:
		// An intent draft is not a native issue: never record zero as a
		// fake issue number, and no acceptance decision exists before
		// planning. The repository slot is shared with issue attempts.
		if a.Issue != 0 || a.Repository <= 0 || a.Acceptance != "" {
			return errors.New("intent assignment carries issue authority")
		}
	case SubjectCompletion:
		// A reassessment cycle is not an issue attempt and never publishes:
		// a fresh attempt root with no parent or publication assignment, in
		// the completed repository slot, carrying no issue authority. It
		// reviews the sealed inventory but implements nothing.
		if a.ParentAssignment != "" || a.AttemptRoot != a.ID {
			return errors.New("reassessment cycle is a fresh attempt root")
		}
		if a.PublicationAssignment != "" {
			return errors.New("reassessment cycle carries no publication")
		}
		if a.Issue != 0 || a.Repository <= 0 || a.Acceptance != "" {
			return errors.New("completion assignment carries issue authority")
		}
	}
	return nil
}

// assignmentPromptBound returns the admitted prompt size for assignment:
// the D13 reassess-only exception when the recorded phase and completion
// subject match, the standard transport bound otherwise. Binding both pins
// the 320 KiB exception to genuine reassessment cycles.
func assignmentPromptBound(a Assignment) int {
	if a.Phase == project.ExecutionReassess && a.Subject.Kind == SubjectCompletion && a.Subject.Completion != nil {
		return project.MaxReassessPrompt
	}
	return project.MaxFactoryPrompt
}

// Validate rejects malformed assignments. Assigned work carries no outcome
// or result; finished work carries both plus its reason.
func (a Assignment) Validate() error {
	if !ValidID(a.ID) || !ValidID(a.AttemptRoot) {
		return errors.New("invalid assignment identity")
	}
	// A reassessment cycle never publishes, so it carries no publication
	// assignment; every other assignment names its publication owner.
	if a.Subject.Kind == SubjectCompletion {
		if a.PublicationAssignment != "" {
			return errors.New("reassessment cycle carries no publication")
		}
	} else if !ValidID(a.PublicationAssignment) {
		return errors.New("invalid assignment identity")
	}
	if !ValidProjectID(a.ProjectID) || !project.ValidFactoryRole(a.Role) {
		return errors.New("invalid assignment address")
	}
	if a.Subject.Kind == "" {
		// Pre-phase issue record: legacy scope rules apply, and no
		// half-migrated subject field may accompany it.
		if a.Phase != "" || a.ParentAssignment != "" || a.Subject.Issue != nil || a.Subject.Intent != nil || a.Subject.Completion != nil {
			return errors.New("subject assignment lacks its subject kind")
		}
		if a.Repository <= 0 || a.Issue <= 0 {
			return errors.New("invalid assignment scope")
		}
		if !project.ValidDecisionID(a.Acceptance) {
			return errors.New("invalid assignment acceptance")
		}
	} else if err := a.validateSubjectPhase(); err != nil {
		return err
	}
	if a.Revision < 0 || a.NativeRev < 1 {
		return errors.New("invalid assignment scope")
	}
	if a.ActorID <= 0 {
		return errors.New("invalid assignment execution actor")
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
	if len(a.Prompt) == 0 || len(a.Prompt) > assignmentPromptBound(a) {
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
		if a.Result.PhaseResult != nil {
			if err := a.Result.ValidatePhaseBinding(a.Role, a.Phase); err != nil {
				return errors.New("finished assignment lacks its bound phase result")
			}
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
