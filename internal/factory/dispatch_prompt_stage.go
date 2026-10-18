package factory

import (
	"errors"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

func promptStage(in PromptInputs) string {
	if in.Role == project.RoleReviewer {
		return "review"
	}
	if in.PublicationAssignment != "" {
		return "correction"
	}
	return "coding"
}

func validatePromptStage(in PromptInputs) error {
	if err := validatePromptRole(in); err != nil {
		return err
	}
	if in.PublicationAssignment == "" {
		return nil
	}
	if !ValidID(in.PublicationAssignment) || !ValidCommit(in.BaseCommit) {
		return errors.New("child prompt needs its publication and verified base")
	}
	if err := validatePromptReviewEvidence(in); err != nil {
		return err
	}
	if err := validatePromptCheckEvidence(in); err != nil {
		return err
	}
	return validateCorrectionPrompt(in)
}

// validatePromptRole admits the prompt role with its base, requiring
// initial coding prompts to carry no child evidence.
func validatePromptRole(in PromptInputs) error {
	if (in.Role != project.RoleCoder && in.Role != project.RoleReviewer) || !ValidCommit(in.ApprovedBase) {
		return errors.New("invalid prompt role or approved base")
	}
	if in.PublicationAssignment == "" {
		if in.Role != project.RoleCoder || in.BaseCommit != "" || in.Review != nil || in.CheckAssessment != nil ||
			in.TypedReview != nil || in.TypedClear != nil {
			return errors.New("initial coding prompt carries child evidence")
		}
	}
	return nil
}

// validatePromptReviewEvidence admits the carried review evidence:
// at most one source, with a single valid typed verdict.
// Fresh independent reviews carry no prior verdict (caller passes nil);
// same-head re-reviews carry the full outstanding review/check evidence
// as carried input to independently assess, never to rubber-stamp.
// Distinct reviewer/corrector sessions are retained by the dispatcher.
func validatePromptReviewEvidence(in PromptInputs) error {
	if in.Review != nil && in.Review.Validate() != nil {
		return errors.New("invalid correction review evidence")
	}
	if in.TypedReview != nil || in.TypedClear != nil {
		if in.Review != nil {
			return errors.New("dispatch prompt carries two review evidence sources")
		}
		if (in.TypedReview == nil) == (in.TypedClear == nil) {
			return errors.New("dispatch prompt carries no single typed verdict")
		}
		if in.TypedReview != nil {
			return validateReviewShape(in.TypedReview)
		}
		return validateClearShape(in.TypedClear)
	}
	return nil
}

// validatePromptCheckEvidence binds the check assessment to its exact
// candidate and required checks.
func validatePromptCheckEvidence(in PromptInputs) error {
	if a := in.CheckAssessment; a != nil {
		if a.Validate() != nil || a.Repository != in.Repository || a.HeadOID != in.SourceCommit || a.BaseOID != in.PRBase ||
			a.BaseRef != in.TargetBranch || a.ChecksDigest != ChecksDigest(in.RequiredChecks) {
			return errors.New("prompt check evidence does not match its exact candidate and checks")
		}
	}
	return nil
}

// validateCorrectionPrompt admits one correction prompt's consolidated
// review and failed-check evidence. Legacy corrections carry the string
// verdict; typed corrections carry the findings verdict (always
// blocking) or the explicit clear beside failing checks (fix the
// checks). Anything else is not a correction.
func validateCorrectionPrompt(in PromptInputs) error {
	if in.Role != project.RoleCoder {
		return nil
	}
	if in.CheckAssessment == nil ||
		(in.CheckAssessment.Verdict != CheckPass && in.CheckAssessment.Verdict != CheckFailed) {
		return errors.New("correction needs consolidated current review and failed-check evidence")
	}
	switch {
	case in.Review != nil:
		if in.Review.Verdict != "request-changes" && in.CheckAssessment.Verdict != CheckFailed {
			return errors.New("correction needs consolidated current review and failed-check evidence")
		}
	case in.TypedReview != nil:
	case in.TypedClear != nil:
		if in.CheckAssessment.Verdict != CheckFailed {
			return errors.New("correction needs consolidated current review and failed-check evidence")
		}
	default:
		return errors.New("correction needs consolidated current review and failed-check evidence")
	}
	return nil
}

func writePromptStage(b *strings.Builder, in PromptInputs) {
	if in.CheckAssessment != nil {
		a := in.CheckAssessment
		b.WriteString("\n## Recorded check evidence\n\n")
		b.WriteString("Head: " + a.HeadOID + " base: " + a.BaseOID + "; verdict: " + a.Verdict + "; reason: " + a.Reason + "\n")
		b.WriteString("Assessment revision: " + strconv.FormatInt(a.Revision, 10) + " native revision: " + strconv.FormatInt(a.NativeRev, 10) + "\n")
		for _, result := range a.Results {
			b.WriteString("- " + strconv.Quote(result.Context) + ": " + strconv.Quote(result.State) + "; passed: " + strconv.FormatBool(result.Passed) + "\n")
		}
	}
	if in.Review != nil {
		r := in.Review
		if in.Role == project.RoleReviewer {
			b.WriteString("\n## Carried review findings for independent re-review\n\n")
			b.WriteString("Assess every carried actionable finding explicitly as verified|still_open with evidence; report new defects separately. Do not repost duplicate native comments for unchanged findings: stable finding keys reuse their native comment IDs.\n")
		} else {
			b.WriteString("\n## Independent review findings\n\n")
		}
		b.WriteString("Verdict: " + r.Verdict + "\n")
		b.WriteString(fenceCollision(r.Summary) + "\n\n")
		b.WriteString(fenceCollision(r.Body) + "\n")
		for _, finding := range r.Findings {
			b.WriteString("- " + fenceCollision(finding) + "\n")
		}
	}
	writePromptTypedVerdict(b, in)
}

// writePromptTypedReviewContract directs a phased re-review: close with
// exactly one v2 phase result carrying exactly one typed verdict — the
// findings verdict with new keyed defects and verified|still_open
// dispositions, or the explicit clear when nothing blocks — answering the
// carried cycle identity. A findings verdict with no new defect and no
// still-open assessment refuses; report the explicit clear instead. No
// legacy review fence is accepted on this contract.
func writePromptTypedReviewContract(b *strings.Builder, in PromptInputs) {
	cycle := ""
	if in.TypedReview != nil {
		cycle = in.TypedReview.CycleID
	} else if in.TypedClear != nil {
		cycle = in.TypedClear.CycleID
	}
	b.WriteString("Close your work with exactly one fenced block carrying the strict v2 phase result for phase \"review\" with exactly one typed verdict:\n\n")
	b.WriteString("```" + ResultFence + "\n")
	b.WriteString(`{"version":2,"phase":"review","status":"completed|blocked|failed|cancelled","summary":"...","output":{"review":{"cycle_id":"` + cycle + `","new_findings":[{"key":"...","severity":"...","defect":"...","evidence":"...","required_outcome":"...","fix_kind":"...","actionable":true}],"assessments":[{"key":"...","status":"verified|still_open","evidence":"..."}]} OR "clear":{...}}}` + "\n")
	b.WriteString("```\n\n")
	b.WriteString("Answer cycle " + cycle + " on the exact candidate: assess every carried actionable finding explicitly as verified|still_open with evidence; report new defects with stable keys, severity, evidence and required outcome. " +
		"Report the explicit clear only when no blocking correctness, security, acceptance or required-check finding remains. Never change the candidate to approve your own correction.\n")
}

// writePromptTypedVerdict renders the phased current verdict a re-review
// or correction must answer: keyed findings with typed dispositions. The
// rendering is deterministic evidence, never a second authority: the
// consumer binds the same verdict by cycle, head and carried set.
func writePromptTypedVerdict(b *strings.Builder, in PromptInputs) {
	if in.TypedReview == nil && in.TypedClear == nil {
		return
	}
	if in.Role == project.RoleReviewer {
		b.WriteString("\n## Carried typed review verdict for independent re-review\n\n")
		b.WriteString("Assess every carried actionable finding explicitly as verified|still_open with evidence; report new defects separately with stable keys. Do not repost duplicate native comments for unchanged findings.\n")
	} else {
		b.WriteString("\n## Phased review evidence\n\n")
	}
	if in.TypedReview != nil {
		r := in.TypedReview
		b.WriteString("Findings verdict for cycle " + r.CycleID + " on candidate " + in.SourceCommit + " at base " + in.BaseCommit + ".\n")
		for _, finding := range r.NewFindings {
			b.WriteString("- [" + finding.Key + "] (" + finding.Severity + "/" + finding.FixKind + ") defect: " +
				fenceCollision(finding.Defect) + " evidence: " + fenceCollision(finding.Evidence) +
				" required outcome: " + fenceCollision(finding.RequiredOutcome) + "\n")
		}
		for _, assessment := range r.Assessments {
			b.WriteString("- [" + assessment.Key + "] " + assessment.Status + ": " + fenceCollision(assessment.Evidence) + "\n")
		}
		return
	}
	c := in.TypedClear
	b.WriteString("Explicit clear for cycle " + c.CycleID + " on candidate " + c.Head + " at base " + c.BaseOID + ".\n")
	for _, assessment := range c.Assessments {
		b.WriteString("- [" + assessment.Key + "] verified: " + fenceCollision(assessment.Evidence) + "\n")
	}
}

// Admit aggregate text before joining, quoting or copying it into the prompt.
// The final encoded prompt must also fit the existing transport cap.
func admitPromptText(in PromptInputs) error {
	budget := &promptTextBudget{remaining: project.MaxFactoryPrompt}
	if err := budget.admitFields(in); err != nil {
		return err
	}
	if err := budget.admitSources(in); err != nil {
		return err
	}
	if err := budget.admitChecks(in); err != nil {
		return err
	}
	if err := budget.admitReview(in); err != nil {
		return err
	}
	if err := budget.admitTypedReview(in); err != nil {
		return err
	}
	if err := budget.admitTypedClear(in); err != nil {
		return err
	}
	return budget.admitCheckAssessment(in)
}

// promptTextBudget spends one transport bound across prompt input texts.
type promptTextBudget struct {
	remaining int
}

// add charges one text with its quoting overhead, refusing when the
// bound is exceeded.
func (b *promptTextBudget) add(text string, overhead int) bool {
	if len(text) > b.remaining || overhead > b.remaining-len(text) {
		return false
	}
	b.remaining -= len(text) + overhead
	return true
}

// admit charges a batch of texts with one shared overhead.
func (b *promptTextBudget) admit(texts []string, overhead int) error {
	for _, text := range texts {
		if !b.add(text, overhead) {
			return errors.New("dispatch prompt input exceeds the transport bound")
		}
	}
	return nil
}

// admitFields charges the scalar prompt input fields.
func (b *promptTextBudget) admitFields(in PromptInputs) error {
	return b.admit([]string{
		in.Title, in.Body, in.AcceptanceID, in.TargetBranch, in.SourceCommit,
		in.ApprovedBase, in.BaseCommit, in.PublicationAssignment, in.Preparation, in.RequirementsID,
		in.ApprovalID, in.Harness, in.Model, in.Role, in.ProviderConnection,
	}, 0)
}

// admitSources charges prompt sources and resolutions.
func (b *promptTextBudget) admitSources(in PromptInputs) error {
	for _, sources := range [][]PromptSource{in.Sources, in.Resolutions} {
		for _, source := range sources {
			if !b.add(source.ID, 32) || !b.add(source.Content, 0) {
				return errors.New("dispatch prompt input exceeds the transport bound")
			}
		}
	}
	return nil
}

// admitChecks charges required checks and prerequisites.
func (b *promptTextBudget) admitChecks(in PromptInputs) error {
	for _, check := range in.RequiredChecks {
		if !b.add(check, 2) {
			return errors.New("dispatch prompt input exceeds the transport bound")
		}
	}
	for _, p := range in.Prerequisites {
		if err := b.admit([]string{p.Occurrence, p.DependsOn, p.Outcome, in.Control.EndpointHeads[p.Occurrence], in.Control.Fingerprint, in.Control.Satisfied[p.Occurrence].Acceptance}, 32); err != nil {
			return err
		}
	}
	return nil
}

// admitReview charges the carried legacy review evidence.
func (b *promptTextBudget) admitReview(in PromptInputs) error {
	if r := in.Review; r != nil {
		if err := b.admit([]string{r.Verdict, r.Summary, r.Body}, 32); err != nil {
			return err
		}
		return b.admit(r.Findings, 4)
	}
	return nil
}

// admitTypedReview charges the carried typed findings verdict.
func (b *promptTextBudget) admitTypedReview(in PromptInputs) error {
	if r := in.TypedReview; r != nil {
		if !b.add(r.CycleID, 32) {
			return errors.New("dispatch prompt input exceeds the transport bound")
		}
		for _, finding := range r.NewFindings {
			if err := b.admit([]string{finding.Key, finding.Severity, finding.FixKind, finding.Defect, finding.Evidence, finding.RequiredOutcome}, 4); err != nil {
				return err
			}
		}
		for _, assessment := range r.Assessments {
			if err := b.admit([]string{assessment.Key, assessment.Status, assessment.Evidence}, 4); err != nil {
				return err
			}
		}
	}
	return nil
}

// admitTypedClear charges the carried explicit clear verdict.
func (b *promptTextBudget) admitTypedClear(in PromptInputs) error {
	if c := in.TypedClear; c != nil {
		if err := b.admit([]string{c.CycleID, c.Conclusion, c.Head, c.BaseOID, c.CoverageDigest}, 32); err != nil {
			return err
		}
		for _, assessment := range c.Assessments {
			if err := b.admit([]string{assessment.Key, assessment.Status, assessment.Evidence}, 4); err != nil {
				return err
			}
		}
	}
	return nil
}

// admitCheckAssessment charges the recorded check evidence.
func (b *promptTextBudget) admitCheckAssessment(in PromptInputs) error {
	if a := in.CheckAssessment; a != nil {
		if err := b.admit([]string{a.HeadOID, a.BaseOID, a.Verdict, a.Reason}, 32); err != nil {
			return err
		}
		for _, result := range a.Results {
			if !b.add(result.Context, 32) || !b.add(result.State, 8) {
				return errors.New("dispatch prompt input exceeds the transport bound")
			}
		}
	}
	return nil
}
