package factory

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// PhaseResultVersion is the only admitted phase-result schema version.
// There is no v1 reader: legacy harness payloads are retired with the FW11
// result-parser migration, not translated.
const PhaseResultVersion = 2

// MaxPhaseResultBytes bounds one encoded phase result.
const MaxPhaseResultBytes = 64 * 1024

// MaxPhaseSummaryBytes bounds the result summary.
const MaxPhaseSummaryBytes = 16 * 1024

// ImplementationReceipt is the typed output of a completed implementation
// phase: the produced candidate and, for non-completed runs only,
// diagnostic references. Diagnostic references never authorize effects.
type ImplementationReceipt struct {
	Candidate      string   `json:"candidate"`
	DiagnosticRefs []string `json:"diagnostic_refs,omitempty"`
}

func (r ImplementationReceipt) validate(status string) error {
	if status == "completed" {
		if !ValidCommit(r.Candidate) {
			return errors.New("completed implementation lacks its candidate")
		}
		if len(r.DiagnosticRefs) != 0 {
			return errors.New("completed implementation retains diagnostics")
		}
		return nil
	}
	if r.Candidate != "" {
		return errors.New("non-completed implementation names a candidate")
	}
	for _, ref := range r.DiagnosticRefs {
		if ref == "" || len(ref) > MaxPlanTextBytes {
			return errors.New("implementation diagnostic outside its bound")
		}
	}
	return nil
}

// PhaseOutput is the single typed payload for its phase: an investigated
// plan for investigate, an implementation receipt for implement, and the
// D13 reassessment report for reassess (FW32). Decomposition, review and
// correction payloads are admitted by FW06B and FW06C through these same
// slots; until then only a nil output validates for those phases. The
// reassessment coder is a single coder with no children, source writes or
// publication: a completed reassess result carries exactly its typed
// report (including a valid empty change set), while a held reassess run
// carries no report. Every phase result stays within the 64 KiB bound,
// including reassess; the 320 KiB exception covers the prompt only.
type PhaseOutput struct {
	Plan           *InvestigationPlan     `json:"plan,omitempty"`
	Implementation *ImplementationReceipt `json:"implementation,omitempty"`
	Reassessment   *ReassessmentReport    `json:"reassessment,omitempty"`
}

func (o PhaseOutput) validate(phase, status string) error {
	switch phase {
	case project.ExecutionInvestigate:
		if o.Plan == nil || o.Implementation != nil || o.Reassessment != nil {
			return errors.New("investigation output must be exactly its plan")
		}
		if err := o.Plan.Validate(); err != nil {
			return err
		}
		if status == "completed" {
			if err := o.Plan.Executable(); err != nil {
				return err
			}
		}
		return nil
	case project.ExecutionImplement:
		if o.Implementation == nil || o.Plan != nil || o.Reassessment != nil {
			return errors.New("implementation output must be exactly its receipt")
		}
		return o.Implementation.validate(status)
	case project.ExecutionReassess:
		if o.Plan != nil || o.Implementation != nil {
			return errors.New("cross-phase result payload refused")
		}
		if status == "completed" {
			if o.Reassessment == nil {
				return errors.New("completed reassessment lacks its typed report")
			}
			return o.Reassessment.Validate()
		}
		if o.Reassessment != nil {
			return errors.New("held reassessment carries no report")
		}
		return nil
	case project.ExecutionDecompose, project.ExecutionReview, project.ExecutionCorrect:
		if o.Plan != nil || o.Implementation != nil || o.Reassessment != nil {
			return errors.New("cross-phase result payload refused")
		}
		return errors.New("phase payload admitted by its owning task")
	default:
		return errors.New("unknown result phase")
	}
}

// PhaseResult is the strict phase-bound process result. Completed
// decomposition, investigation and reassessment forbid a publishable
// candidate: planning and reassessment outputs carry no candidate at all,
// so such a result can never be interpreted as publishable code. Results
// stay within the 64 KiB bound in every phase, including reassess.
type PhaseResult struct {
	Output  *PhaseOutput `json:"output,omitempty"`
	Phase   string       `json:"phase"`
	Status  string       `json:"status"`
	Summary string       `json:"summary"`
	Version int          `json:"version"`
}

func (r PhaseResult) Validate() error {
	if r.Version != PhaseResultVersion {
		return errors.New("unsupported phase result version")
	}
	if !project.ValidExecutionPhase(r.Phase) || r.Phase == "" {
		return errors.New("phase result lacks its execution phase")
	}
	switch r.Status {
	case "completed", "blocked", "failed", "cancelled":
	default:
		return errors.New("invalid phase result status")
	}
	if r.Summary == "" || len(r.Summary) > MaxPhaseSummaryBytes {
		return errors.New("phase result summary outside its bound")
	}
	if r.Output == nil {
		if r.Status == "completed" {
			return errors.New("completed phase lacks its typed output")
		}
	} else if err := r.Output.validate(r.Phase, r.Status); err != nil {
		return err
	}
	encoded, err := json.Marshal(r)
	if err != nil || len(encoded) > MaxPhaseResultBytes {
		return errors.New("encoded phase result exceeds its bound")
	}
	return nil
}

// BindAssignment verifies the result against its recorded assignment: phase
// equality binds phase and subject lineage, since the assignment already
// binds role, subject and run. Pre-phase assignments cannot bind v2 results.
func (r PhaseResult) BindAssignment(a Assignment) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if a.Phase == "" {
		return errors.New("phase result cannot bind a pre-phase assignment")
	}
	if r.Phase != a.Phase {
		return errors.New("phase result does not match its assignment phase")
	}
	return nil
}

// ParsePhaseResult extracts the last fenced result block from harness output
// and validates it as a strict v2 phase result. Unknown fields and trailing
// JSON refuse; anything unparseable reports false instead of guessing.
func ParsePhaseResult(output string) (PhaseResult, bool) {
	start := strings.LastIndex(output, "```"+ResultFence)
	if start < 0 {
		return PhaseResult{}, false
	}
	rest := output[start+len("```"+ResultFence):]
	for offset := 0; offset < len(rest); {
		rel := strings.Index(rest[offset:], "```")
		if rel < 0 {
			return PhaseResult{}, false
		}
		body := strings.TrimSpace(rest[:offset+rel])
		offset += rel + len("```")
		if body == "" || len(body) > MaxPhaseResultBytes {
			continue
		}
		var result PhaseResult
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&result); err != nil {
			continue
		}
		if decoder.More() {
			continue
		}
		if result.Validate() != nil {
			continue
		}
		return result, true
	}
	return PhaseResult{}, false
}

// Result is the process contract. A completed harness reports an outcome; the
// platform still verifies its candidate, independent review and CI evidence.
type Result struct {
	Status       string   `json:"status"`
	Summary      string   `json:"summary"`
	Candidate    string   `json:"candidate"`
	ReviewPassed bool     `json:"review_passed"`
	Findings     []string `json:"findings"`
}

const ResultSchema = `{"type":"object","additionalProperties":false,"properties":{"status":{"type":"string","enum":["completed","blocked","failed","cancelled"]},"summary":{"type":"string"},"candidate":{"type":"string","pattern":"^([a-f0-9]{40})?$"},"review_passed":{"type":"boolean"},"findings":{"type":"array","items":{"type":"string"}}},"required":["status","summary","candidate","review_passed","findings"]}`

func (r Result) Validate() error {
	switch r.Status {
	case "completed", "blocked", "failed", "cancelled":
	default:
		return errors.New("invalid harness result status")
	}
	if len(r.Summary) > 16<<10 || len(r.Findings) > 64 {
		return errors.New("harness result is too large")
	}
	if r.Status == "completed" && !ValidCommit(r.Candidate) {
		return errors.New("completed result must identify its commit")
	}
	for _, finding := range r.Findings {
		if len(finding) > 4096 {
			return errors.New("review finding is too large")
		}
	}
	return nil
}
