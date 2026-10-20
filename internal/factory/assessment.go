package factory

import "errors"

// Non-code assessment (R34.3). An Inspector assesses an exact
// research, content or decision deliverable without code carriers:
// no candidate, tuple, coverage digest, Git commit or app
// verification. The deliverable binds by its immutable identity — a
// recorded report submission or a library item revision — and the
// verdict carries evidence against exactly that identity. A findings
// verdict must block; the explicit clear carries no findings. Owner
// decisions keep their recorded owner; assessment never rewrites the
// deliverable. Non-code findings reuse Finding with its existing
// validation: OriginRun names the assessing run, and FixKind keeps
// its meaning with the deliverable as the fix site (code = correct
// the deliverable content, verification = redo its supporting
// evidence or checks).
const (
	NonCodeDeliverableResearch = "research"
	NonCodeDeliverableLibrary  = "library"
)

// NonCodeDeliverable binds one assessment to its exact deliverable:
// a recorded finish submission for research, a library item revision
// for content and decisions. Revisions are immutable, so ID plus
// revision names the assessed bytes without a digest.
type NonCodeDeliverable struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Revision int64  `json:"revision,omitempty"`
}

// NonCodeAssessment is one Inspector verdict over a non-code
// deliverable: the bound deliverable plus either blocking findings
// or the explicit clear.
type NonCodeAssessment struct {
	Deliverable NonCodeDeliverable  `json:"deliverable"`
	Clear       bool                `json:"clear"`
	Findings    []Finding           `json:"findings,omitempty"`
	Assessments []FindingAssessment `json:"assessments,omitempty"`
}

func (d NonCodeDeliverable) validateShape() error {
	switch d.Kind {
	case NonCodeDeliverableResearch:
		if !ValidID(d.ID) {
			return errors.New("research deliverable lacks its report submission")
		}
		if d.Revision != 0 {
			return errors.New("research deliverable carries no revision")
		}
		return nil
	case NonCodeDeliverableLibrary:
		if !ValidID(d.ID) {
			return errors.New("library deliverable lacks its item")
		}
		if d.Revision < 1 {
			return errors.New("library deliverable lacks its revision")
		}
		return nil
	default:
		return errors.New("unknown assessment deliverable")
	}
}

// ValidateNonCodeSubject binds one deliverable to its assessment
// subject: research reports assess on research assignments, library
// items on collection assignments. The settle layer enforces this
// with the assigned subject in hand; output validation stays
// structural.
func ValidateNonCodeSubject(d NonCodeDeliverable, subjectKind string) error {
	switch d.Kind {
	case NonCodeDeliverableResearch:
		if subjectKind != SubjectResearch {
			return errors.New("research deliverable needs its research subject")
		}
		return nil
	case NonCodeDeliverableLibrary:
		if subjectKind != SubjectCollection {
			return errors.New("library deliverable needs its collection subject")
		}
		return nil
	default:
		return errors.New("unknown assessment deliverable")
	}
}

// validateNonCodeAssessment checks the structural assessment verdict:
// an exactly shaped deliverable with either blocking findings or the
// explicit clear carrying nothing.
func validateNonCodeAssessment(r *NonCodeAssessment) error {
	if r == nil {
		return errors.New("assessment verdict lacks its assessment")
	}
	if err := r.Deliverable.validateShape(); err != nil {
		return err
	}
	blocking := false
	for _, finding := range r.Findings {
		if err := finding.Validate(); err != nil {
			return err
		}
		blocking = true
	}
	for _, assessment := range r.Assessments {
		if err := assessment.Validate(); err != nil {
			return err
		}
		if assessment.Status == FindingStillOpen {
			blocking = true
		}
	}
	if r.Clear {
		if blocking {
			return errors.New("assessment clear carries blocking findings")
		}
		return nil
	}
	if !blocking {
		return errors.New("assessment verdict carries no blocking finding; report the explicit clear")
	}
	return nil
}
