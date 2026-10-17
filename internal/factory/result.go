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
// plan for investigate, an implementation receipt for implement, the D13
// reassessment report for reassess (FW32), the single chosen decomposition
// for decompose, the typed review verdict for review, the typed
// correction dispositions for correct, the D15 stall repair or D16
// delivery repair report for repair, and the D16 app verification report
// for verify. The reassessment coder is a single coder with no children,
// source writes or publication: a completed reassess result carries
// exactly its typed report (including a valid empty change set), while a
// held reassess run carries no report. Repair and verify follow the same
// shape: a completed repair carries exactly one repair report (stall or
// delivery repair, matching its subject), a completed verify carries its
// app verification report, and held runs carry no report. Every phase
// result stays within the 64 KiB bound, including reassess and repair;
// the 320 KiB exception covers the prompt only.
//
// Completed decompose, review, correct, repair and verify results carry no
// publishable candidate: the decomposition chooses task boundaries only,
// the review verdict assesses findings only, the correction accounts
// findings only, and repair/verification reports carry no candidate field
// at all, so such a result can never be interpreted as publishable code.
type PhaseOutput struct {
	Plan           *InvestigationPlan     `json:"plan,omitempty"`
	Implementation *ImplementationReceipt `json:"implementation,omitempty"`
	Reassessment   *ReassessmentReport    `json:"reassessment,omitempty"`
	Decomposition  *ScopedDecomposition   `json:"decomposition,omitempty"`
	Review         *ReviewCycleReport     `json:"review,omitempty"`
	Clear          *ClearReviewReport     `json:"clear,omitempty"`
	Correction     *CorrectionReport      `json:"correction,omitempty"`
	Repair         *StallRepairReport     `json:"repair,omitempty"`
	DeliveryRepair *DeliveryRepairReport  `json:"delivery_repair,omitempty"`
	Verification   *AppVerificationReport `json:"verification,omitempty"`
}

// validateReviewShape checks the structural review verdict: finding and
// assessment shapes with a bound cycle identity. A findings verdict must
// carry something blocking — a new defect or a still-open assessment — or
// it is not a findings verdict at all: a fully verified round reports an
// explicit clear instead. Coverage of the whole carried set belongs to
// admission, which sees the controller's canonical findings — never to
// this shape check.
func validateReviewShape(r *ReviewCycleReport) error {
	if !ValidID(r.CycleID) {
		return errors.New("review verdict lacks its cycle")
	}
	blocking := len(r.NewFindings) > 0
	for _, finding := range r.NewFindings {
		if err := finding.Validate(); err != nil {
			return err
		}
	}
	for _, assessment := range r.Assessments {
		if err := assessment.Validate(); err != nil {
			return err
		}
		if assessment.Status == FindingStillOpen {
			blocking = true
		}
	}
	if !blocking {
		return errors.New("review verdict carries no blocking finding; report an explicit clear")
	}
	return nil
}

// validateClearShape checks the structural explicit-clear verdict: the
// clear conclusion over an exact candidate with its coverage digest and
// bounded assessments. Carried-set binding belongs to admission.
func validateClearShape(r *ClearReviewReport) error {
	if r.Conclusion != ClearReviewConclusion {
		return errors.New("review report is not an explicit clear")
	}
	if !ValidID(r.CycleID) {
		return errors.New("invalid clear review cycle")
	}
	if !ValidCommit(r.Head) || !ValidCommit(r.BaseOID) || r.Head == r.BaseOID {
		return errors.New("clear review covers no exact candidate")
	}
	if !ValidDigest(r.CoverageDigest) {
		return errors.New("clear review lacks its complete coverage digest")
	}
	for _, assessment := range r.Assessments {
		if err := assessment.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (o PhaseOutput) validate(phase, status string) error {
	switch phase {
	case project.ExecutionInvestigate:
		if o.Plan == nil || !o.nilTypedOutputExceptPlan() {
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
		if o.Implementation == nil || !o.nilTypedOutputExceptImplementation() {
			return errors.New("implementation output must be exactly its receipt")
		}
		return o.Implementation.validate(status)
	case project.ExecutionReassess:
		if !o.nilTypedOutputExceptReassessment() {
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
	case project.ExecutionDecompose:
		if o.Decomposition == nil || !o.nilTypedOutputExceptDecomposition() {
			return errors.New("cross-phase result payload refused")
		}
		if status != "completed" {
			return errors.New("non-completed decomposition carries no payload")
		}
		return o.Decomposition.Validate()
	case project.ExecutionReview:
		if !o.nilTypedOutputExceptReview() {
			return errors.New("cross-phase result payload refused")
		}
		if status != "completed" {
			if o.Review != nil || o.Clear != nil {
				return errors.New("non-completed review carries no verdict")
			}
			return nil
		}
		if (o.Review == nil) == (o.Clear == nil) {
			return errors.New("completed review carries exactly one verdict")
		}
		if o.Review != nil {
			return validateReviewShape(o.Review)
		}
		return validateClearShape(o.Clear)
	case project.ExecutionCorrect:
		if o.Correction == nil || !o.nilTypedOutputExceptCorrection() {
			return errors.New("cross-phase result payload refused")
		}
		if status != "completed" {
			return errors.New("non-completed correction carries no payload")
		}
		return o.Correction.Validate()
	case project.ExecutionRepair:
		if !o.nilTypedOutputExceptRepair() {
			return errors.New("cross-phase result payload refused")
		}
		if status != "completed" {
			if o.Repair != nil || o.DeliveryRepair != nil {
				return errors.New("non-completed repair carries no report")
			}
			return nil
		}
		if (o.Repair == nil) == (o.DeliveryRepair == nil) {
			return errors.New("completed repair carries exactly one repair report")
		}
		if o.Repair != nil {
			return o.Repair.Validate()
		}
		return o.DeliveryRepair.Validate()
	case project.ExecutionVerify:
		if !o.nilTypedOutputExceptVerification() {
			return errors.New("cross-phase result payload refused")
		}
		if status != "completed" {
			if o.Verification != nil {
				return errors.New("non-completed verification carries no report")
			}
			return nil
		}
		if o.Verification == nil {
			return errors.New("completed verification lacks its typed report")
		}
		return o.Verification.Validate()
	default:
		return errors.New("unknown result phase")
	}
}

// nilTypedOutputExceptPlan reports whether every payload slot except the
// investigation plan is empty.
func (o PhaseOutput) nilTypedOutputExceptPlan() bool {
	return o.Implementation == nil && o.Reassessment == nil &&
		o.Decomposition == nil && o.Review == nil && o.Clear == nil && o.Correction == nil &&
		o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil
}

// nilTypedOutputExceptImplementation reports whether every payload slot
// except the implementation receipt is empty.
func (o PhaseOutput) nilTypedOutputExceptImplementation() bool {
	return o.Plan == nil && o.Reassessment == nil &&
		o.Decomposition == nil && o.Review == nil && o.Clear == nil && o.Correction == nil &&
		o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil
}

// nilTypedOutputExceptReassessment reports whether every payload slot
// except the reassessment report is empty.
func (o PhaseOutput) nilTypedOutputExceptReassessment() bool {
	return o.Plan == nil && o.Implementation == nil &&
		o.Decomposition == nil && o.Review == nil && o.Clear == nil && o.Correction == nil &&
		o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil
}

// nilTypedOutputExceptDecomposition reports whether every payload slot
// except the chosen decomposition is empty.
func (o PhaseOutput) nilTypedOutputExceptDecomposition() bool {
	return o.Plan == nil && o.Implementation == nil && o.Reassessment == nil &&
		o.Review == nil && o.Clear == nil && o.Correction == nil &&
		o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil
}

// nilTypedOutputExceptReview reports whether every payload slot except the
// two review verdicts is empty.
func (o PhaseOutput) nilTypedOutputExceptReview() bool {
	return o.Plan == nil && o.Implementation == nil && o.Reassessment == nil &&
		o.Decomposition == nil && o.Correction == nil &&
		o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil
}

// nilTypedOutputExceptCorrection reports whether every payload slot except
// the correction dispositions is empty.
func (o PhaseOutput) nilTypedOutputExceptCorrection() bool {
	return o.Plan == nil && o.Implementation == nil && o.Reassessment == nil &&
		o.Decomposition == nil && o.Review == nil && o.Clear == nil &&
		o.Repair == nil && o.DeliveryRepair == nil && o.Verification == nil
}

// nilTypedOutputExceptRepair reports whether every payload slot except the
// two repair reports is empty. Exactly one of them is set for a completed
// repair; the validate arm enforces which.
func (o PhaseOutput) nilTypedOutputExceptRepair() bool {
	return o.Plan == nil && o.Implementation == nil && o.Reassessment == nil &&
		o.Decomposition == nil && o.Review == nil && o.Clear == nil && o.Correction == nil &&
		o.Verification == nil
}

// nilTypedOutputExceptVerification reports whether every payload slot
// except the app verification report is empty.
func (o PhaseOutput) nilTypedOutputExceptVerification() bool {
	return o.Plan == nil && o.Implementation == nil && o.Reassessment == nil &&
		o.Decomposition == nil && o.Review == nil && o.Clear == nil && o.Correction == nil &&
		o.Repair == nil && o.DeliveryRepair == nil
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
// Repair and verify results additionally bind their subject kind and, for
// completed runs, the report's subject, cycle and inventory identities.
// A stall repair additionally runs the assignment's own head: a report
// for another commit never binds. The six older phases keep their
// historic phase-only binding.
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
	switch r.Phase {
	case project.ExecutionRepair:
		if a.Subject.Kind != SubjectStall && a.Subject.Kind != SubjectDeliveryRepair {
			return errors.New("repair result does not match its repair subject")
		}
		if err := a.Subject.Validate(); err != nil {
			return err
		}
		if r.Output != nil && r.Status == "completed" {
			if a.Subject.Kind == SubjectStall {
				subject, report := a.Subject.Stall, r.Output.Repair
				if report == nil || report.StallID != subject.StallID ||
					report.CycleID != subject.CycleID || report.InventoryDigest != subject.InventoryDigest {
					return errors.New("stall repair report does not match its assigned subject")
				}
				if report.SourceCommit != a.SourceCommit {
					return errors.New("stall repair report runs a head its assignment was not admitted against")
				}
			} else {
				subject, report := a.Subject.DeliveryRepair, r.Output.DeliveryRepair
				if report == nil || report.Subject.DeliveryID != subject.DeliveryID ||
					report.Subject.FailedVerificationRunID != subject.FailedVerificationRunID ||
					report.Subject.CycleID != subject.CycleID || report.Subject.InventoryDigest != subject.InventoryDigest {
					return errors.New("delivery repair report does not match its assigned subject")
				}
			}
		}
	case project.ExecutionVerify:
		if a.Subject.Kind != SubjectDelivery {
			return errors.New("verification result does not match its delivery subject")
		}
		if err := a.Subject.Validate(); err != nil {
			return err
		}
		if r.Status == "completed" && r.Output.Verification.DeliveryID != a.Subject.Delivery.DeliveryID {
			return errors.New("verification report does not match its assigned delivery")
		}
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

// CompletedCandidate returns the exact candidate a finished harness run
// reported. Any unfinished process (blocked, failed, cancelled) refuses:
// only a completed run on an exact commit can back a clear review's
// process-completion claim (D07/T42).
func (r Result) CompletedCandidate() (string, error) {
	if r.Status != "completed" {
		return "", errors.New("unfinished process backs no clear review")
	}
	if !ValidCommit(r.Candidate) {
		return "", errors.New("completed result lacks its exact candidate")
	}
	return r.Candidate, nil
}

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
