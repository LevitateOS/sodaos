package factory

import (
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
	AssignmentID string   `json:"assignment_id"`
	RunID        string   `json:"run_id"`
	Status       string   `json:"status"`
	Summary      string   `json:"summary"`
	Candidate    string   `json:"candidate"`
	Findings     []string `json:"findings"`
	ReviewPassed bool     `json:"review_passed"`
	Reported     bool     `json:"reported"`
	RecordedUnix int64    `json:"recorded_unix"`
}

// Validate extends Result validation with the dispatch binding.
func (r AssignmentResult) Validate() error {
	if !ValidID(r.AssignmentID) || !ValidID(r.RunID) {
		return errors.New("invalid result dispatch binding")
	}
	if r.RecordedUnix <= 0 {
		return errors.New("invalid result record time")
	}
	return Result{
		Status: r.Status, Summary: r.Summary, Candidate: r.Candidate,
		ReviewPassed: r.ReviewPassed, Findings: r.Findings,
	}.Validate()
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
