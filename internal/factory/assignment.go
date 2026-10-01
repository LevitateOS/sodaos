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
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

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
// afterwards.
type Assignment struct {
	Authority    AuthorityRef      `json:"authority"`
	Result       *AssignmentResult `json:"result,omitempty"`
	Prompt       []byte            `json:"prompt"`
	RunHistory   []string          `json:"run_history"`
	ID           string            `json:"id"`
	ProjectID    string            `json:"project_id"`
	Role         string            `json:"role"`
	Acceptance   string            `json:"acceptance"`
	Preparation  string            `json:"preparation"`
	Harness      string            `json:"harness"`
	HarnessVers  string            `json:"harness_version"`
	Model        string            `json:"model"`
	Connection   string            `json:"connection"`
	SourceCommit string            `json:"source_commit"`
	PromptSHA    string            `json:"prompt_sha"`
	Run          string            `json:"run"`
	Stage        string            `json:"stage"`
	Outcome      Outcome           `json:"outcome,omitempty"`
	Reason       string            `json:"reason,omitempty"`
	Repository   int64             `json:"repository,string"`
	Issue        int64             `json:"issue,string"`
	Revision     int64             `json:"revision"`
	NativeRev    int64             `json:"native_revision"`
	Attempts     int               `json:"attempts"`
	CreatedUnix  int64             `json:"created_unix"`
	FinishedUnix int64             `json:"finished_unix,omitempty"`
}

// Validate rejects malformed assignments. Assigned work carries no outcome
// or result; finished work carries both plus its reason.
func (a Assignment) Validate() error {
	if !ValidID(a.ID) {
		return errors.New("invalid assignment identity")
	}
	if !ValidProjectID(a.ProjectID) || !project.ValidFactoryRole(a.Role) {
		return errors.New("invalid assignment address")
	}
	if a.Repository <= 0 || a.Issue <= 0 || a.Revision < 0 || a.NativeRev < 1 {
		return errors.New("invalid assignment scope")
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
		if a.Result == nil || a.Result.Validate() != nil {
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
	end := strings.Index(rest, "```")
	if end < 0 {
		return Result{}, false
	}
	body := strings.TrimSpace(rest[:end])
	if body == "" || len(body) > project.MaxFactoryOutput {
		return Result{}, false
	}
	var result Result
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return Result{}, false
	}
	if result.Validate() != nil {
		return Result{}, false
	}
	return result, true
}

// Reservation states. Held capacity counts against every applicable
// limit; consumed capacity served a recorded run; released capacity was
// confirmed unused and counts nowhere.
const (
	ReservationHeld     = "held"
	ReservationConsumed = "consumed"
	ReservationReleased = "released"
)

// ValidReservationState reports whether state is a known reservation state.
func ValidReservationState(state string) bool {
	switch state {
	case ReservationHeld, ReservationConsumed, ReservationReleased:
		return true
	default:
		return false
	}
}

// Reservation is one dispatch's held capacity: an appliance slot, a
// repository slot and a provider allowance slice, all keyed by the
// assignment. PlannedMinutes is the ceiling the run's deadline implies;
// only confirmed actual usage is ever charged.
type Reservation struct {
	AssignmentID   string `json:"assignment_id"`
	Repository     int64  `json:"repository,string"`
	Connection     string `json:"connection"`
	State          string `json:"state"`
	PlannedMinutes int    `json:"planned_minutes"`
	Revision       int64  `json:"revision"`
}

// Validate rejects malformed reservations.
func (r Reservation) Validate() error {
	if !ValidID(r.AssignmentID) || r.Repository <= 0 {
		return errors.New("invalid reservation scope")
	}
	if r.Connection == "" || len(r.Connection) > 128 {
		return errors.New("invalid reservation connection")
	}
	if !ValidReservationState(r.State) || r.Revision < 0 {
		return errors.New("invalid reservation state")
	}
	if r.PlannedMinutes < 1 || r.PlannedMinutes > 180 {
		return errors.New("invalid reservation plan")
	}
	return nil
}

// Usage is one settled run's confirmed provider consumption in whole
// minutes. Rows are append-only: edits and resume never rewrite them, so
// allowances only shrink until the sponsor grants a new revision.
type Usage struct {
	RunID        string `json:"run_id"`
	Repository   int64  `json:"repository,string"`
	Connection   string `json:"connection"`
	Minutes      int    `json:"minutes"`
	RecordedUnix int64  `json:"recorded_unix"`
}

// Validate rejects malformed usage rows.
func (u Usage) Validate() error {
	if !ValidID(u.RunID) || u.Repository <= 0 {
		return errors.New("invalid usage scope")
	}
	if u.Connection == "" || len(u.Connection) > 128 {
		return errors.New("invalid usage connection")
	}
	if u.Minutes < 0 || u.Minutes > 10080 || u.RecordedUnix <= 0 {
		return errors.New("invalid usage amount")
	}
	return nil
}

// PromptSource is one verified accepted text section: a native comment ID
// plus the content the bracketed read verified against its digest.
type PromptSource struct {
	ID      string
	Content string
}

// PromptInputs gathers the verified accepted inputs behind one dispatch
// prompt. Every text section was digest-verified against the acceptance
// head at dispatch; IDs, digests and commits are exact references. No
// credential or credential-adjacent value enters the prompt.
type PromptInputs struct {
	Sources      []PromptSource
	Resolutions  []PromptSource
	Title        string
	Body         string
	AcceptanceID string
	TargetBranch string
	SourceCommit string
	Preparation  string
	Harness      string
	Model        string
	Role         string
	Repository   int64
	Issue        int64
	NativeRev    int64
}

// fenceCollision marks the fenced report opener inside native text so a
// pasted fence cannot forge the harness report block. The visible text is
// preserved with a separating space.
func fenceCollision(text string) string {
	return strings.ReplaceAll(text, "```"+ResultFence, "``` "+ResultFence)
}

// BuildDispatchPrompt renders the deterministic assignment prompt from
// verified accepted inputs. The shape is fixed: locator and exact
// references, the accepted objective, accepted sources and resolutions,
// then the result contract. Prompts that exceed the transport bound
// refuse; truncation would silently change the requirements.
func BuildDispatchPrompt(in PromptInputs) ([]byte, error) {
	if in.Repository <= 0 || in.Issue <= 0 || in.NativeRev < 1 {
		return nil, errors.New("invalid dispatch prompt scope")
	}
	if !project.ValidDecisionID(in.AcceptanceID) || !ValidCommit(in.SourceCommit) || in.TargetBranch == "" {
		return nil, errors.New("invalid dispatch prompt references")
	}
	if in.Title == "" {
		return nil, errors.New("dispatch prompt needs its objective")
	}
	var b strings.Builder
	b.WriteString("# Soda factory coding assignment\n\n")
	b.WriteString("Repository: " + strconv.FormatInt(in.Repository, 10) + "\n")
	b.WriteString("Issue: " + strconv.FormatInt(in.Issue, 10) + "\n")
	b.WriteString("Acceptance: " + in.AcceptanceID + " @ native revision " + strconv.FormatInt(in.NativeRev, 10) + "\n")
	b.WriteString("Target: " + in.TargetBranch + " @ " + in.SourceCommit + "\n")
	b.WriteString("Preparation: " + in.Preparation + "\n")
	b.WriteString("Role: " + in.Role + " Harness: " + in.Harness + " Model: " + in.Model + "\n")
	b.WriteString("\nImplement the accepted requirements below in the prepared checkout. ")
	b.WriteString("Only the accepted objective, sources and resolutions authorize changes; ")
	b.WriteString("unselected discussion does not.\n")
	b.WriteString("\n## Objective\n\n")
	b.WriteString(fenceCollision(in.Title) + "\n\n")
	if in.Body != "" {
		b.WriteString(fenceCollision(in.Body) + "\n")
	}
	writePromptSection(&b, "Accepted sources", in.Sources)
	writePromptSection(&b, "Accepted resolutions", in.Resolutions)
	b.WriteString("\n## Report\n\n")
	b.WriteString("Close your work with exactly one fenced block:\n\n")
	b.WriteString("```" + ResultFence + "\n")
	b.WriteString(`{"status":"completed|blocked|failed|cancelled","summary":"...","candidate":"<40-hex commit or empty>","review_passed":false,"findings":[]}` + "\n")
	b.WriteString("```\n\n")
	b.WriteString("Report \"completed\" only with the exact commit your change produced. ")
	b.WriteString("Never print credentials, tokens or secret files; the report carries IDs and digests only.\n")
	prompt := []byte(b.String())
	if len(prompt) > project.MaxFactoryPrompt {
		return nil, errors.New("dispatch prompt exceeds the transport bound")
	}
	if !utf8.Valid(prompt) {
		return nil, errors.New("dispatch prompt is not valid text")
	}
	return prompt, nil
}

func writePromptSection(b *strings.Builder, title string, sources []PromptSource) {
	if len(sources) == 0 {
		return
	}
	b.WriteString("\n## " + title + "\n")
	for _, source := range sources {
		b.WriteString("\n### comment " + source.ID + "\n\n")
		b.WriteString(fenceCollision(source.Content) + "\n")
	}
}
