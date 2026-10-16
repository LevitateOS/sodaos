package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// AssignmentResult extends the retained harness result contract with its
// dispatch binding: which assignment attempt produced it, whether the
// harness reported it or the supervisor synthesized it from the observed
// run outcome, and when it was recorded. A reported candidate is
// syntax-checked only; publication validates it against fresh native
// observations before use.
type AssignmentResult struct {
	Review       *ReviewReport `json:"review,omitempty"`
	PhaseResult  *PhaseResult  `json:"phase_result,omitempty"`
	AssignmentID string        `json:"assignment_id"`
	RunID        string        `json:"run_id"`
	Phase        string        `json:"phase,omitempty"`
	Status       string        `json:"status"`
	Summary      string        `json:"summary"`
	Candidate    string        `json:"candidate"`
	Findings     []string      `json:"findings"`
	ReviewPassed bool          `json:"review_passed"`
	Reported     bool          `json:"reported"`
	RecordedUnix int64         `json:"recorded_unix"`
}

// Validate extends Result validation with the dispatch binding.
func (r AssignmentResult) Validate() error {
	if !ValidID(r.AssignmentID) || !ValidID(r.RunID) {
		return errors.New("invalid result dispatch binding")
	}
	if r.RecordedUnix <= 0 {
		return errors.New("invalid result record time")
	}
	if r.Review != nil {
		if err := r.Review.Validate(); err != nil {
			return err
		}
	}
	if r.PhaseResult != nil {
		if err := r.PhaseResult.Validate(); err != nil {
			return err
		}
	}
	if err := (Result{
		Status: r.Status, Summary: r.Summary, Candidate: r.Candidate,
		ReviewPassed: r.ReviewPassed, Findings: r.Findings,
	}.Validate()); err != nil {
		// Completed planning exposes no candidate: the plan shape has no
		// candidate field, so such a result can never authorize effects.
		// The exemption covers exactly the candidate rule; every other
		// shape rule still refuses.
		if r.Candidate != "" || r.Status != "completed" || !isPlanningPhaseResult(r.PhaseResult) {
			return err
		}
		if len(r.Summary) > 16<<10 || len(r.Findings) > 64 {
			return err
		}
		for _, finding := range r.Findings {
			if len(finding) > 4096 {
				return err
			}
		}
	}
	return nil
}

// isPlanningPhaseResult reports whether phased output belongs to a planning
// phase whose completed result never names a publishable candidate.
// Decompose and reassess payloads arrive through their owning boundaries;
// until then their completed results cannot validate.
func isPlanningPhaseResult(phased *PhaseResult) bool {
	if phased == nil {
		return false
	}
	switch phased.Phase {
	case project.ExecutionInvestigate, project.ExecutionDecompose, project.ExecutionReassess:
		return true
	default:
		return false
	}
}

// ValidatePhaseBinding enforces the phase/role/result contract for v2
// results: the recorded phase matches the assignment phase and its admitted
// role, and the typed payload binds that same phase. Cross-phase payloads
// refuse, and a phased result never binds a pre-phase assignment. The
// legacy harness fields above stay untouched for the FW11 result-parser
// migration; new admission records PhaseResult.
func (r AssignmentResult) ValidatePhaseBinding(role, assignmentPhase string) error {
	if r.PhaseResult == nil {
		return errors.New("phased result lacks its typed output")
	}
	if r.Phase == "" || r.Phase != assignmentPhase {
		return errors.New("phased result does not match its assignment phase")
	}
	if !ValidPhaseRole(r.Phase, role) {
		return errors.New("phased result role does not match its phase")
	}
	if r.PhaseResult.Phase != r.Phase {
		return errors.New("phased result payload does not match its phase")
	}
	return r.PhaseResult.Validate()
}

// ValidateForAssignment enforces the role-specific result contract. A
// reviewer report is retained only for reviewer assignments and is bound to
// the exact candidate the assignment reviewed.
func (r AssignmentResult) ValidateForAssignment(role, sourceCommit string) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if role == project.RoleReviewer {
		if r.Reported {
			if r.Review == nil || r.Status != "completed" || r.Candidate != sourceCommit || r.ReviewPassed || len(r.Findings) != 0 || r.Summary != r.Review.Summary {
				return errors.New("reported reviewer result does not match its candidate")
			}
		} else if r.Review != nil || r.Status == "completed" || r.Candidate != "" || r.ReviewPassed || len(r.Findings) != 0 {
			return errors.New("synthesized reviewer result carries report evidence")
		}
		return nil
	}
	if r.Review != nil {
		return errors.New("non-reviewer result carries a review report")
	}
	return nil
}

// ToResult returns the retained harness result shape.
func (r AssignmentResult) ToResult() Result {
	return Result{
		Status: r.Status, Summary: r.Summary, Candidate: r.Candidate,
		ReviewPassed: r.ReviewPassed, Findings: r.Findings,
	}
}

// ResultFromHarness binds one validated harness-reported result to its
// attempt. Reported results arrive through the fenced report block only.
func ResultFromHarness(assignmentID, runID string, reported Result, recordedUnix int64) AssignmentResult {
	return AssignmentResult{
		AssignmentID: assignmentID, RunID: runID,
		Status: reported.Status, Summary: reported.Summary, Candidate: reported.Candidate,
		Findings: reported.Findings, ReviewPassed: reported.ReviewPassed,
		Reported: true, RecordedUnix: recordedUnix,
	}
}

// ResultDigest derives the deterministic identity input for one recorded
// attempt result: the hex SHA-256 over its canonical encoding. Automatic
// phase advancement keys on (parent_assignment, result digest, next phase),
// so the same completed parent replays to the same child instead of
// launching twice.
func ResultDigest(r AssignmentResult) string {
	encoded, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// ResultSynthesized binds one supervisor-derived result to its attempt.
// The supervisor maps the observed host outcome when the harness reports
// nothing parseable; the candidate stays empty until validated.
func ResultSynthesized(assignmentID, runID, status, summary string, recordedUnix int64) AssignmentResult {
	return AssignmentResult{
		AssignmentID: assignmentID, RunID: runID,
		Status: status, Summary: summary, Findings: []string{},
		Reported: false, RecordedUnix: recordedUnix,
	}
}

// ResultFence delimits the harness report block: the prompt instructs the
// CLI to close its work with exactly one fenced result-json block, and
// the supervisor parses the last such block strictly.
const ResultFence = "result-json"

// ParseHarnessResult extracts the last fenced result block from harness
// output and validates it strictly. Unknown fields refuse; anything
// unparseable reports false instead of guessing a result.
func ParseHarnessResult(output string) (Result, bool) {
	start := strings.LastIndex(output, "```"+ResultFence)
	if start < 0 {
		return Result{}, false
	}
	rest := output[start+len("```"+ResultFence):]
	// The JSON payload may itself contain nested fences (quoted diffs,
	// code samples), so the first fence after the opener is unreliable:
	// try every closing fence in order until one decodes. The first
	// valid payload wins; trailing chatter never mis-terminates it.
	for offset := 0; offset < len(rest); {
		rel := strings.Index(rest[offset:], "```")
		if rel < 0 {
			return Result{}, false
		}
		body := strings.TrimSpace(rest[:offset+rel])
		offset += rel + len("```")
		if body == "" || len(body) > project.MaxFactoryOutput {
			continue
		}
		var result Result
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&result); err != nil {
			continue
		}
		if result.Validate() != nil {
			continue
		}
		return result, true
	}
	return Result{}, false
}
