package factory

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/project"
)

// ErrPromptPrerequisiteEvidence marks missing or stale accepted dependency
// evidence; size and encoding refusals remain ordinary prompt errors.
var ErrPromptPrerequisiteEvidence = errors.New("dispatch prompt prerequisite evidence invalid")

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
	Sources               []PromptSource
	Resolutions           []PromptSource
	Prerequisites         []AcceptedPrerequisite
	Control               IssueControl
	Title                 string
	Body                  string
	AcceptanceID          string
	TargetBranch          string
	SourceCommit          string
	Preparation           string
	RequirementsID        string
	ApprovalID            string
	Harness               string
	Model                 string
	Role                  string
	ProviderConnection    string
	RequiredChecks        []string
	ApplianceConcurrent   int
	RepositoryConcurrent  int
	SponsorshipConcurrent int
	PlannedMinutes        int
	Repository            int64
	Issue                 int64
	NativeRev             int64
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
	if in.Title == "" || in.PlannedMinutes < 1 || in.ApplianceConcurrent < 1 || in.RepositoryConcurrent < 1 || in.SponsorshipConcurrent < 1 || len(in.RequiredChecks) == 0 || in.ProviderConnection == "" {
		return nil, errors.New("dispatch prompt needs selected controller inputs")
	}
	if len(in.Prerequisites) > 0 {
		if in.Control.Validate() != nil || in.Control.Readiness != ReadinessQueued || in.Control.Repository != in.Repository || in.Control.Issue != in.Issue || in.Control.Acceptance != in.AcceptanceID || in.Control.NativeRev != in.NativeRev || len(in.Control.EndpointHeads) != len(in.Prerequisites) {
			return nil, fmt.Errorf("%w: readiness identity mismatch", ErrPromptPrerequisiteEvidence)
		}
		for _, prerequisite := range in.Prerequisites {
			if prerequisite.Validate() != nil {
				return nil, fmt.Errorf("%w: invalid accepted prerequisite", ErrPromptPrerequisiteEvidence)
			}
			head, ok := in.Control.EndpointHeads[prerequisite.Occurrence]
			if !ok || (prerequisite.Outcome == PrereqCode && head != prerequisite.PrereqAcceptance) ||
				(prerequisite.Outcome == PrereqResult && !satisfiedResultOccurrence(in.Control, prerequisite.Occurrence)) {
				return nil, fmt.Errorf("%w: incomplete outcome evidence", ErrPromptPrerequisiteEvidence)
			}
		}
		for occurrence := range in.Control.Satisfied {
			found := false
			for _, prerequisite := range in.Prerequisites {
				found = found || prerequisite.Occurrence == occurrence && prerequisite.Outcome == PrereqResult
			}
			if !found {
				return nil, fmt.Errorf("%w: unaccepted result occurrence", ErrPromptPrerequisiteEvidence)
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Soda factory coding assignment\n\n")
	b.WriteString("Prompt template: soda-f07-f2-v3\n")
	b.WriteString("Repository: " + strconv.FormatInt(in.Repository, 10) + "\n")
	b.WriteString("Issue: " + strconv.FormatInt(in.Issue, 10) + "\n")
	b.WriteString("Acceptance: " + in.AcceptanceID + " @ native revision " + strconv.FormatInt(in.NativeRev, 10) + "\n")
	b.WriteString("Target: " + in.TargetBranch + " @ " + in.SourceCommit + "\n")
	b.WriteString("Preparation: " + in.Preparation + "\n")
	b.WriteString("Preparation requirements: " + in.RequirementsID + " approval: " + in.ApprovalID + "\n")
	b.WriteString("Role: " + in.Role + " Harness: " + in.Harness + " Model: " + in.Model + "\n")
	b.WriteString("Provider connection: " + strconv.Quote(in.ProviderConnection) + "\n")
	b.WriteString("\n## Controller policy\n\n")
	b.WriteString("Permitted actions: inspect and modify the prepared checkout to implement the accepted objective; run relevant local checks; report the resulting candidate and evidence. Do not change grants, select another provider or model, publish refs, create or merge pull requests, or expose credentials.\n")
	b.WriteString("Effective limits: this run may use at most " + strconv.Itoa(in.PlannedMinutes) + " minutes; appliance concurrency is " + strconv.Itoa(in.ApplianceConcurrent) + " and repository concurrency is " + strconv.Itoa(in.RepositoryConcurrent) + "; sponsorship concurrency for this connection in this repository is " + strconv.Itoa(in.SponsorshipConcurrent) + ". These are selected limits for this dispatch.\n")
	if len(in.RequiredChecks) > 0 {
		b.WriteString("Required evidence checks: " + strings.Join(in.RequiredChecks, "; ") + "\n")
	}
	b.WriteString("Required evidence: exact candidate commit, checks actually run and their outcomes, unresolved requirements, and remaining risks. Do not claim a check passed without its result.\n")
	b.WriteString("If accepted inputs conflict with controller policy, a required limit is unavailable, or the objective cannot be completed within the selected limits, stop further work and report blocked with the concrete reason. Never evade a limit by retrying, switching credentials or expanding scope.\n")
	b.WriteString("\nImplement the accepted requirements below in the prepared checkout. ")
	b.WriteString("Only the accepted objective, sources and resolutions authorize changes; ")
	b.WriteString("unselected discussion does not.\n")
	b.WriteString("\n## Objective\n\n")
	b.WriteString(fenceCollision(in.Title) + "\n\n")
	if in.Body != "" {
		b.WriteString(fenceCollision(in.Body) + "\n")
	}
	writePromptSection(&b, "Accepted sources", in.Sources)
	writePromptPrerequisites(&b, in.Prerequisites, in.Control)
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

func satisfiedResultOccurrence(control IssueControl, occurrence string) bool {
	satisfied, ok := control.Satisfied[occurrence]
	return ok && satisfied.Acceptance == control.Acceptance && satisfied.Validate() == nil
}

func writePromptPrerequisites(b *strings.Builder, prerequisites []AcceptedPrerequisite, control IssueControl) {
	if len(prerequisites) == 0 {
		return
	}
	b.WriteString("\n## Accepted prerequisites\n")
	for _, prerequisite := range prerequisites {
		head := control.EndpointHeads[prerequisite.Occurrence]
		b.WriteString("\n- Occurrence " + prerequisite.Occurrence + ": depends on issue " + prerequisite.DependsOn +
			" at repository " + strconv.FormatInt(prerequisite.EndpointRepo, 10) + ", issue " + strconv.FormatInt(prerequisite.EndpointIssue, 10) +
			"; outcome " + prerequisite.Outcome + "; endpoint acceptance " + head +
			"; satisfied as of queued readiness revision " + strconv.FormatInt(control.Revision, 10) + " (fingerprint " + control.Fingerprint + ")")
		if prerequisite.Outcome == PrereqResult {
			satisfied := control.Satisfied[prerequisite.Occurrence]
			b.WriteString("; lifecycle " + strconv.Itoa(satisfied.Lifecycle) + " closed at " + strconv.FormatInt(satisfied.ClosedUnix, 10) + " under acceptance " + satisfied.Acceptance)
		}
		b.WriteByte('\n')
	}
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
