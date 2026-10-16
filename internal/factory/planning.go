package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// InspectedSource binds one examined repository path to the exact commit
// observed and the evidence taken from it.
type InspectedSource struct {
	Path     string `json:"path"`
	Commit   string `json:"commit"`
	Evidence string `json:"evidence"`
}

// RejectedAlternative records one considered approach and why it lost.
type RejectedAlternative struct {
	Approach string `json:"approach"`
	Reason   string `json:"reason"`
}

// InvestigationPlan is the immutable investigated plan revision consumed by
// one implementation run: accepted scope and input digests, source and
// prerequisite identities, inspected evidence, findings, approach,
// considered alternatives, acceptance coverage, and the validated
// task/lane/check graph with its canonical semantic digest. Listed
// unresolved facts are mandatory: Executable refuses while any remain.
type InvestigationPlan struct {
	ID                 string                `json:"id"`
	AttemptRoot        string                `json:"attempt_root"`
	ScopeDigest        string                `json:"scope_digest"`
	InputDigest        string                `json:"input_digest"`
	SourceCommit       string                `json:"source_commit"`
	IntegrationCommit  string                `json:"integration_commit"`
	Preparation        string                `json:"preparation"`
	Approach           string                `json:"approach"`
	Digest             string                `json:"digest"`
	PrerequisiteDigest []string              `json:"prerequisite_digests,omitempty"`
	Inspected          []InspectedSource     `json:"inspected"`
	Findings           []string              `json:"findings"`
	Unresolved         []string              `json:"unresolved_facts,omitempty"`
	Rejected           []RejectedAlternative `json:"rejected_alternatives,omitempty"`
	AcceptanceCoverage []string              `json:"acceptance_coverage"`
	Tasks              []PlanTask            `json:"tasks"`
	Lanes              []PlanLane            `json:"lanes"`
	Checks             []CheckSpec           `json:"checks"`
	Generation         int                   `json:"generation"`
}

// PlanDigest returns the canonical semantic digest over plan content. The
// stored Digest must equal this value; FW07 persists revisions keyed by
// (attempt_root, plan_id) with a separate current pointer, so the original
// consumed plan always survives replacement.
func PlanDigest(plan InvestigationPlan) (string, error) {
	plan.Digest = ""
	data, err := json.Marshal(struct {
		Version string
		Plan    InvestigationPlan
	}{"investigation-plan-v1", plan})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Validate checks the plan revision shape, bounds and graph. It does not
// admit the plan for coding: Executable additionally requires every
// mandatory fact resolved, and FW12 revalidates source, prerequisite and
// preparation bindings against current state before launch.
func (p InvestigationPlan) Validate() error {
	if !ValidID(p.ID) || !ValidID(p.AttemptRoot) {
		return errors.New("invalid plan identity")
	}
	if p.Generation < 0 {
		return errors.New("invalid plan generation")
	}
	if !ValidDigest(p.ScopeDigest) || !ValidDigest(p.InputDigest) {
		return errors.New("invalid plan scope or input digest")
	}
	if !ValidCommit(p.SourceCommit) || !ValidCommit(p.IntegrationCommit) {
		return errors.New("invalid plan source identity")
	}
	for _, digest := range p.PrerequisiteDigest {
		if !ValidDigest(digest) {
			return errors.New("invalid plan prerequisite digest")
		}
	}
	if !ValidPreparationRef(p.Preparation) {
		return errors.New("invalid plan preparation")
	}
	if len(p.Inspected) > MaxInspectedRefs {
		return errors.New("plan inspects too many sources")
	}
	for _, ref := range p.Inspected {
		if !validOwnershipEntry(ref.Path) || strings.HasSuffix(ref.Path, "/") {
			return errors.New("inspected source is not an exact file")
		}
		if !ValidCommit(ref.Commit) {
			return errors.New("inspected source lacks its commit")
		}
		if ref.Evidence == "" || len(ref.Evidence) > MaxPlanTextBytes {
			return errors.New("inspected evidence outside its bound")
		}
	}
	for _, text := range append(append([]string(nil), p.Findings...), p.Unresolved...) {
		if text == "" || len(text) > MaxPlanTextBytes {
			return errors.New("plan finding outside its bound")
		}
	}
	if p.Approach == "" || len(p.Approach) > MaxPlanTextBytes {
		return errors.New("plan approach outside its bound")
	}
	for _, alt := range p.Rejected {
		if alt.Approach == "" || len(alt.Approach) > MaxPlanTextBytes || alt.Reason == "" || len(alt.Reason) > MaxPlanTextBytes {
			return errors.New("rejected alternative outside its bound")
		}
	}
	if len(p.AcceptanceCoverage) == 0 {
		return errors.New("plan lacks acceptance coverage")
	}
	for _, text := range p.AcceptanceCoverage {
		if text == "" || len(text) > MaxPlanTextBytes {
			return errors.New("plan acceptance coverage outside its bound")
		}
	}
	if err := ValidatePlanGraph(p.Tasks, p.Lanes, p.Checks); err != nil {
		return err
	}
	digest, err := PlanDigest(p)
	if err != nil || digest != p.Digest {
		return errors.New("plan digest does not match its content")
	}
	encoded, err := json.Marshal(p)
	if err != nil || len(encoded) > MaxEncodedPlan {
		return errors.New("encoded plan exceeds its bound")
	}
	return nil
}

// Executable reports whether the validated plan may launch implementation.
// Stale source, prerequisite or preparation bindings are rechecked by FW12
// admission; this gate covers the plan's own completeness.
func (p InvestigationPlan) Executable() error {
	if err := p.Validate(); err != nil {
		return err
	}
	if len(p.Unresolved) != 0 {
		return errors.New("plan carries unresolved mandatory facts")
	}
	return nil
}

// PlanningInput retains the accepted stable scope and explicitly missing
// prerequisite outcomes. Provisional plans convey no permission to execute.
type PlanningInput struct {
	Candidate PriorityCandidate `json:"candidate"`
	Missing   []Blocker         `json:"missing,omitempty"`
}

type PlanningRecord struct {
	Snapshot    string         `json:"snapshot"`
	Input       PlanningInput  `json:"input"`
	Strategy    string         `json:"strategy"`
	Steps       []string       `json:"steps"`
	Provisional bool           `json:"provisional"`
	Outcome     string         `json:"outcome"`
	Judgment    PriorityAnswer `json:"judgment"`
	CreatedUnix int64          `json:"created_unix"`
}

func PlanningSnapshot(input PlanningInput) (string, error) {
	// Ordering observations are rechecked separately. Unrelated native writes
	// do not invalidate an otherwise identical implementation approach.
	input.Candidate.Control.NativeRev = 0
	input.Candidate.Control.AssessedUnix = 0
	data, err := json.Marshal(struct {
		Kind  string
		Input PlanningInput
	}{"bounded-planning-v1", input})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (r PlanningRecord) Validate() error {
	snapshot, err := PlanningSnapshot(r.Input)
	if err != nil || snapshot != r.Snapshot || r.CreatedUnix <= 0 || r.Input.Candidate.Control.Repository <= 0 || r.Input.Candidate.Control.Issue <= 0 {
		return errors.New("invalid planning record")
	}
	if r.Outcome != "planned" && r.Outcome != "waiting" && r.Outcome != "unavailable" {
		return errors.New("invalid planning outcome")
	}
	if r.Provisional != (len(r.Input.Missing) > 0) {
		return errors.New("invalid provisional planning state")
	}
	return nil
}

// ReadyToPlan permits only accepted, authorized scope with visible pending
// outcomes. Inaccessible evidence, cycles and stale acceptance are code refusals.
func ReadyToPlan(control IssueControl) bool {
	if control.Acceptance == "" {
		return false
	}
	if control.Readiness == ReadinessQueued {
		return true
	}
	if control.Readiness != ReadinessBlocked || len(control.Blockers) == 0 {
		return false
	}
	for _, b := range control.Blockers {
		if b.Code != BlockerCodePending && b.Code != BlockerResultPending {
			return false
		}
	}
	return true
}
