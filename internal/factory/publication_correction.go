package factory

import (
	"errors"
	"strings"
)

// CorrectionOps is one publication's ordered correction chain: branch
// republications after the linked PR exists. Each entry chains its
// predecessor tip and binds the same PR; the publication head follows
// the latest committed correction. See Publication.Validate.
type CorrectionOps []PublicationOperation

// Carried-finding assessment states. A report assesses every carried
// actionable finding and cannot silently omit one.
const (
	FindingVerified  = "verified"
	FindingStillOpen = "still_open"
)

// Correction dispositions. Exactly one disposition covers each supplied
// finding key. Optional-only findings are recorded declined_optional
// unless implemented and never force a coder run; actionable items
// cannot be declined by an agent.
const (
	DispositionFixed            = "fixed"
	DispositionUnresolved       = "unresolved"
	DispositionDisputed         = "disputed"
	DispositionDeclinedOptional = "declined_optional"
)

// FindingAssessment is one carried finding's explicit round assessment:
// its key, verified|still_open state and the evidence behind it.
type FindingAssessment struct {
	Key      string `json:"key"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

// Validate checks the assessment shape. Coverage of the whole carried set
// belongs to the cycle report, not to one assessment.
func (a FindingAssessment) Validate() error {
	if !ValidFindingKey(a.Key) {
		return errors.New("invalid finding assessment key")
	}
	switch a.Status {
	case FindingVerified, FindingStillOpen:
	default:
		return errors.New("invalid finding assessment state")
	}
	if strings.TrimSpace(a.Evidence) == "" || len(a.Evidence) > MaxFindingProseBytes {
		return errors.New("finding assessment lacks bounded evidence")
	}
	return nil
}

// CorrectionDisposition is one supplied finding key's correction outcome.
type CorrectionDisposition struct {
	Key         string `json:"key"`
	Disposition string `json:"disposition"`
}

// Validate checks the disposition value. Key coverage and the
// declined-optional/actionable rule belong to the correction report,
// which sees the whole finding set.
func (d CorrectionDisposition) Validate() error {
	if !ValidFindingKey(d.Key) {
		return errors.New("invalid correction disposition key")
	}
	switch d.Disposition {
	case DispositionFixed, DispositionUnresolved, DispositionDisputed, DispositionDeclinedOptional:
	default:
		return errors.New("invalid correction disposition")
	}
	return nil
}

// CorrectionReport is one correction run's output over the whole current
// finding set: the exact finding-set digest it answered plus exactly one
// disposition per supplied key. Fixed code findings require relevant
// changed paths/commits and exact checked-tree evidence behind them;
// verification-only findings may retain the same head when new passing
// checks prove their required outcome. Missing, duplicate, unknown or
// stale dispositions refuse; disputes use existing intervention. Every
// correction run consumes one existing correction cycle.
type CorrectionReport struct {
	SetDigest    string                  `json:"set_digest"`
	Findings     []Finding               `json:"findings"`
	Dispositions []CorrectionDisposition `json:"dispositions"`
}

// Validate binds the dispositions to the exact supplied finding set.
func (r CorrectionReport) Validate() error {
	if len(r.Findings) == 0 || len(r.Findings) > MaxOutstandingFindings {
		return errors.New("correction report needs its bounded finding set")
	}
	keys := make([]string, 0, len(r.Findings))
	byKey := make(map[string]Finding, len(r.Findings))
	for _, finding := range r.Findings {
		if err := finding.Validate(); err != nil {
			return err
		}
		if _, dup := byKey[finding.Key]; dup {
			return errors.New("correction report carries a duplicate finding")
		}
		byKey[finding.Key] = finding
		keys = append(keys, finding.Key)
	}
	digest, err := FindingSetDigest(keys)
	if err != nil {
		return err
	}
	if !ValidDigest(r.SetDigest) || r.SetDigest != digest {
		return errors.New("correction dispositions answer a changed finding set")
	}
	if len(r.Dispositions) != len(r.Findings) {
		return errors.New("correction dispositions omit a supplied finding")
	}
	seen := make(map[string]bool, len(r.Dispositions))
	for _, disposition := range r.Dispositions {
		if err := disposition.Validate(); err != nil {
			return err
		}
		if seen[disposition.Key] {
			return errors.New("correction carries a duplicate disposition")
		}
		seen[disposition.Key] = true
		finding, ok := byKey[disposition.Key]
		if !ok {
			return errors.New("correction dispositions answer an unknown finding")
		}
		if disposition.Disposition == DispositionDeclinedOptional && finding.Actionable {
			return errors.New("actionable finding cannot be declined")
		}
	}
	return nil
}

// ReviewCycle is one persisted review cycle (D07): publication, monotonic
// round number, triggering decision/reason, exact head/base, input
// finding digest and fresh reviewer/correction assignment IDs. Durable
// storage and fresh-session dispatch belong to FW18/FW20; this is the
// frozen contract they persist.
type ReviewCycle struct {
	PublicationID        string `json:"publication_id"`
	AttemptRoot          string `json:"attempt_root"`
	ID                   string `json:"id"`
	Phase                string `json:"phase"`
	TriggerDecision      string `json:"trigger_decision"`
	TriggerReason        string `json:"trigger_reason"`
	Head                 string `json:"head"`
	BaseOID              string `json:"base_oid"`
	FindingDigest        string `json:"finding_digest"`
	ReviewerAssignment   string `json:"reviewer_assignment"`
	CorrectionAssignment string `json:"correction_assignment,omitempty"`
	Round                int    `json:"round"`
}

// Validate checks the cycle identity derivation and bindings. The
// correction assignment stays empty while no correction run is required;
// reviewer and corrector sessions stay distinct.
func (c ReviewCycle) Validate() error {
	if !ValidID(c.PublicationID) || !ValidID(c.AttemptRoot) {
		return errors.New("invalid review cycle scope")
	}
	if c.Round < 1 {
		return errors.New("invalid review cycle round")
	}
	if strings.TrimSpace(c.Phase) == "" || len(c.Phase) > 64 {
		return errors.New("invalid review cycle phase")
	}
	if c.ID != ReviewCycleID(c.PublicationID, c.AttemptRoot, c.Round, c.Phase) {
		return errors.New("review cycle identity differs from its derivation")
	}
	if strings.TrimSpace(c.TriggerDecision) == "" || len(c.TriggerDecision) > 128 {
		return errors.New("review cycle lacks its triggering decision")
	}
	if strings.TrimSpace(c.TriggerReason) == "" || len(c.TriggerReason) > MaxFindingProseBytes {
		return errors.New("review cycle lacks its bounded triggering reason")
	}
	if !ValidCommit(c.Head) || !ValidCommit(c.BaseOID) || c.Head == c.BaseOID {
		return errors.New("review cycle candidate is not a fresh exact commit")
	}
	if !ValidDigest(c.FindingDigest) {
		return errors.New("invalid review cycle finding digest")
	}
	if !ValidID(c.ReviewerAssignment) {
		return errors.New("invalid review cycle reviewer assignment")
	}
	if c.CorrectionAssignment != "" && !ValidID(c.CorrectionAssignment) {
		return errors.New("invalid review cycle correction assignment")
	}
	return nil
}

// ReviewCycleReport is one reviewer round's verdict over the carried set:
// new findings plus an explicit verified|still_open assessment for every
// carried actionable finding, with evidence. Mechanically complete
// proposed fixes can reach Jev while their native threads are open; a
// reviewer may verify them in a new round, or Jev may accept the full
// set. Optional-only reviews go straight to Jev and never force a coder
// run. A clean first review also requires Jev's decision; it never
// self-approves to merge.
type ReviewCycleReport struct {
	CycleID     string              `json:"cycle_id"`
	NewFindings []Finding           `json:"new_findings,omitempty"`
	Assessments []FindingAssessment `json:"assessments,omitempty"`
}

// Validate binds the verdict to the carried finding set.
func (r ReviewCycleReport) Validate(carried []Finding) error {
	if !ValidID(r.CycleID) {
		return errors.New("invalid review cycle report identity")
	}
	if len(carried) > MaxOutstandingFindings {
		return errors.New("carried finding set exceeds its bound")
	}
	carriedByKey := make(map[string]Finding, len(carried))
	for _, finding := range carried {
		if err := finding.Validate(); err != nil {
			return err
		}
		if _, dup := carriedByKey[finding.Key]; dup {
			return errors.New("carried finding set carries a duplicate key")
		}
		carriedByKey[finding.Key] = finding
	}
	if len(r.NewFindings) > MaxFindingsPerReport {
		return errors.New("review cycle report carries too many findings")
	}
	for _, finding := range r.NewFindings {
		if err := finding.Validate(); err != nil {
			return err
		}
		if _, dup := carriedByKey[finding.Key]; dup {
			return errors.New("new finding reuses a carried key")
		}
		carriedByKey[finding.Key] = finding
	}
	assessed := make(map[string]bool, len(r.Assessments))
	for _, assessment := range r.Assessments {
		if _, ok := carriedByKey[assessment.Key]; !ok {
			return errors.New("finding assessment answers an unknown finding")
		}
		if assessed[assessment.Key] {
			return errors.New("finding assessment repeats its key")
		}
		assessed[assessment.Key] = true
		if err := assessment.Validate(); err != nil {
			return err
		}
	}
	for _, finding := range carried {
		if finding.Actionable && !assessed[finding.Key] {
			return errors.New("carried actionable finding lacks its assessment")
		}
	}
	return nil
}

// RequiresCorrectionRun reports whether the round verdict needs a coder
// correction run: any actionable new finding or any still-open carried
// assessment. Optional-only verdicts go straight to Jev.
func (r ReviewCycleReport) RequiresCorrectionRun() bool {
	for _, finding := range r.NewFindings {
		if finding.Actionable {
			return true
		}
	}
	for _, assessment := range r.Assessments {
		if assessment.Status == FindingStillOpen {
			return true
		}
	}
	return false
}
