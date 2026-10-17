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
	if (in.Role != project.RoleCoder && in.Role != project.RoleReviewer) || !ValidCommit(in.ApprovedBase) {
		return errors.New("invalid prompt role or approved base")
	}
	if in.PublicationAssignment == "" {
		if in.Role != project.RoleCoder || in.BaseCommit != "" || in.Review != nil || in.CheckAssessment != nil ||
			in.TypedReview != nil || in.TypedClear != nil {
			return errors.New("initial coding prompt carries child evidence")
		}
		return nil
	}
	if !ValidID(in.PublicationAssignment) || !ValidCommit(in.BaseCommit) {
		return errors.New("child prompt needs its publication and verified base")
	}
	// Fresh independent reviews carry no prior verdict (caller passes nil);
	// same-head re-reviews carry the full outstanding review/check evidence
	// as carried input to independently assess, never to rubber-stamp.
	// Distinct reviewer/corrector sessions are retained by the dispatcher.
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
			if err := validateReviewShape(in.TypedReview); err != nil {
				return err
			}
		} else if err := validateClearShape(in.TypedClear); err != nil {
			return err
		}
	}
	if a := in.CheckAssessment; a != nil {
		if a.Validate() != nil || a.Repository != in.Repository || a.HeadOID != in.SourceCommit || a.BaseOID != in.BaseCommit ||
			a.BaseRef != in.TargetBranch || a.ChecksDigest != ChecksDigest(in.RequiredChecks) {
			return errors.New("prompt check evidence does not match its exact candidate and checks")
		}
	}
	if in.Role == project.RoleCoder {
		if in.CheckAssessment == nil ||
			(in.CheckAssessment.Verdict != CheckPass && in.CheckAssessment.Verdict != CheckFailed) {
			return errors.New("correction needs consolidated current review and failed-check evidence")
		}
		// Legacy corrections carry the string verdict; typed corrections
		// carry the findings verdict (always blocking) or the explicit
		// clear beside failing checks (fix the checks). Anything else is
		// not a correction.
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
	remaining := project.MaxFactoryPrompt
	add := func(text string, overhead int) bool {
		if len(text) > remaining || overhead > remaining-len(text) {
			return false
		}
		remaining -= len(text) + overhead
		return true
	}
	tooLarge := errors.New("dispatch prompt input exceeds the transport bound")
	for _, text := range []string{
		in.Title, in.Body, in.AcceptanceID, in.TargetBranch, in.SourceCommit,
		in.ApprovedBase, in.BaseCommit, in.PublicationAssignment, in.Preparation, in.RequirementsID,
		in.ApprovalID, in.Harness, in.Model, in.Role, in.ProviderConnection,
	} {
		if !add(text, 0) {
			return tooLarge
		}
	}
	for _, sources := range [][]PromptSource{in.Sources, in.Resolutions} {
		for _, source := range sources {
			if !add(source.ID, 32) || !add(source.Content, 0) {
				return tooLarge
			}
		}
	}
	for _, check := range in.RequiredChecks {
		if !add(check, 2) {
			return tooLarge
		}
	}
	for _, p := range in.Prerequisites {
		for _, text := range []string{p.Occurrence, p.DependsOn, p.Outcome, in.Control.EndpointHeads[p.Occurrence], in.Control.Fingerprint, in.Control.Satisfied[p.Occurrence].Acceptance} {
			if !add(text, 32) {
				return tooLarge
			}
		}
	}
	if r := in.Review; r != nil {
		for _, text := range []string{r.Verdict, r.Summary, r.Body} {
			if !add(text, 32) {
				return tooLarge
			}
		}
		for _, text := range r.Findings {
			if !add(text, 4) {
				return tooLarge
			}
		}
	}
	if r := in.TypedReview; r != nil {
		if !add(r.CycleID, 32) {
			return tooLarge
		}
		for _, finding := range r.NewFindings {
			for _, text := range []string{finding.Key, finding.Severity, finding.FixKind, finding.Defect, finding.Evidence, finding.RequiredOutcome} {
				if !add(text, 4) {
					return tooLarge
				}
			}
		}
		for _, assessment := range r.Assessments {
			for _, text := range []string{assessment.Key, assessment.Status, assessment.Evidence} {
				if !add(text, 4) {
					return tooLarge
				}
			}
		}
	}
	if c := in.TypedClear; c != nil {
		for _, text := range []string{c.CycleID, c.Conclusion, c.Head, c.BaseOID, c.CoverageDigest} {
			if !add(text, 32) {
				return tooLarge
			}
		}
		for _, assessment := range c.Assessments {
			for _, text := range []string{assessment.Key, assessment.Status, assessment.Evidence} {
				if !add(text, 4) {
					return tooLarge
				}
			}
		}
	}
	if a := in.CheckAssessment; a != nil {
		for _, text := range []string{a.HeadOID, a.BaseOID, a.Verdict, a.Reason} {
			if !add(text, 32) {
				return tooLarge
			}
		}
		for _, result := range a.Results {
			if !add(result.Context, 32) || !add(result.State, 8) {
				return tooLarge
			}
		}
	}
	return nil
}
