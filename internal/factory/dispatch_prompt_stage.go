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
		if in.Role != project.RoleCoder || in.BaseCommit != "" || in.Review != nil || in.CheckAssessment != nil {
			return errors.New("initial coding prompt carries child evidence")
		}
		return nil
	}
	if !ValidID(in.PublicationAssignment) || !ValidCommit(in.BaseCommit) {
		return errors.New("child prompt needs its publication and verified base")
	}
	if in.Role == project.RoleReviewer && in.Review != nil {
		return errors.New("independent reviewer cannot receive a prior review verdict")
	}
	if in.Review != nil && in.Review.Validate() != nil {
		return errors.New("invalid correction review evidence")
	}
	if a := in.CheckAssessment; a != nil {
		if a.Validate() != nil || a.Repository != in.Repository || a.HeadOID != in.SourceCommit || a.BaseOID != in.BaseCommit ||
			a.BaseRef != in.TargetBranch || a.ChecksDigest != ChecksDigest(in.RequiredChecks) {
			return errors.New("prompt check evidence does not match its exact candidate and checks")
		}
	}
	if in.Role == project.RoleCoder {
		if in.Review == nil || in.CheckAssessment == nil ||
			(in.CheckAssessment.Verdict != CheckPass && in.CheckAssessment.Verdict != CheckFailed) ||
			(in.Review.Verdict != "request-changes" && in.CheckAssessment.Verdict != CheckFailed) {
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
		b.WriteString("\n## Independent review findings\n\n")
		b.WriteString("Verdict: " + r.Verdict + "\n")
		b.WriteString(fenceCollision(r.Summary) + "\n\n")
		b.WriteString(fenceCollision(r.Body) + "\n")
		for _, finding := range r.Findings {
			b.WriteString("- " + fenceCollision(finding) + "\n")
		}
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
