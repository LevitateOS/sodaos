package control

import (
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// ClearAdmissionInput is the controller's complete explicit-clear input:
// the publication and its recorded review cycle, the actual completed
// typed reviewer assignment behind the verdict, the controller's
// canonical carried findings, and the controller-observed scope/policy/
// check evidence the acceptance binds. The clear verdict is read from
// the reviewer's recorded typed result — never a parallel report, and
// never a faked candidate: review outputs stay candidate-free. No
// allowance, chooser or delivery-call reservation travels here: a clear
// review creates acceptance with zero delivery calls by construction.
type ClearAdmissionInput struct {
	Publication    factory.Publication
	Cycle          factory.ReviewCycle
	Reviewer       factory.Assignment
	Carried        []factory.Finding
	ScopeDigest    string
	PolicyRevision string
	CheckDigest    string
	DiffBase       string
}

// AdmitClearReview admits one completed independent explicit clear review
// as clean_review delivery acceptance and journals it on the publication.
// Every D07/T41-T43 gate runs here:
//
//   - the cycle is recorded on this publication and validates;
//   - the reviewer assignment is the cycle's independent reviewer,
//     distinct from any corrector session, finished succeeded with a
//     reported completed typed result;
//   - the recorded verdict is the reviewer's explicit clear over the
//     controller's canonical carried set (silence, native approval,
//     empty output, incomplete coverage, omitted carried findings and
//     stale tuples refuse);
//   - the coverage digest recomputed from the canonical set must equal the
//     reported one, so same-key rewritten content invalidates;
//   - the verdict answers the cycle's exact head/base, which must still be
//     the publication's current candidate and PR creation base, and must
//     equal the reviewer's admitted source commit: the reviewer finished
//     on the exact head through its assignment binding, not a result
//     candidate field.
//
// The funded jev_acceptance variant is constructed by FW36 alongside D08
// routing; the union type and its validation already admit both kinds.
func AdmitClearReview(in ClearAdmissionInput) (factory.Publication, factory.DeliveryAcceptance, error) {
	var empty factory.DeliveryAcceptance
	published := in.Publication
	if err := in.Cycle.Validate(); err != nil {
		return published, empty, err
	}
	var recorded factory.ReviewCycle
	found := false
	for _, cycle := range published.ReviewCycles {
		if cycle.ID == in.Cycle.ID {
			recorded, found = cycle, true
			break
		}
	}
	if !found {
		return published, empty, errors.New("clear review consumes an unrecorded cycle")
	}
	// The recorded row binds every immutable cycle field not just its
	// ID: a same-ID substituted cycle with another head, base, digest,
	// round, phase, trigger or assignment takes no admission.
	if recorded.PublicationID != published.ID ||
		recorded.AttemptRoot != in.Cycle.AttemptRoot ||
		recorded.Round != in.Cycle.Round ||
		recorded.Phase != in.Cycle.Phase ||
		recorded.Head != in.Cycle.Head ||
		recorded.BaseOID != in.Cycle.BaseOID ||
		recorded.FindingDigest != in.Cycle.FindingDigest ||
		recorded.ReviewerAssignment != in.Cycle.ReviewerAssignment ||
		recorded.CorrectionAssignment != in.Cycle.CorrectionAssignment ||
		recorded.TriggerDecision != in.Cycle.TriggerDecision ||
		recorded.TriggerReason != in.Cycle.TriggerReason {
		return published, empty, errors.New("clear review consumes a substituted cycle")
	}
	reviewer := in.Reviewer
	if err := reviewer.Validate(); err != nil {
		return published, empty, err
	}
	if reviewer.ID != in.Cycle.ReviewerAssignment {
		return published, empty, errors.New("clear review needs its independent reviewer assignment")
	}
	// The reviewer belongs to this publication's lineage: its owner
	// assignment and attempt root must match the recorded publication
	// and cycle, so a foreign attempt's reviewer takes no admission.
	if reviewer.PublicationAssignment != published.AssignmentID {
		return published, empty, errors.New("clear review needs its publication reviewer")
	}
	if reviewer.AttemptRoot != in.Cycle.AttemptRoot {
		return published, empty, errors.New("clear review needs its attempt reviewer")
	}
	if reviewer.Role != project.RoleReviewer || reviewer.Phase != project.ExecutionReview {
		return published, empty, errors.New("clear review needs its phased reviewer assignment")
	}
	if reviewer.Stage != factory.AssignmentFinished || reviewer.Outcome != factory.Succeeded {
		return published, empty, errors.New("clear review needs its finished reviewer")
	}
	if reviewer.Result == nil || !reviewer.Result.Reported || reviewer.Result.Status != "completed" {
		return published, empty, errors.New("clear review needs its reported reviewer result")
	}
	if reviewer.Result.PhaseResult == nil || reviewer.Result.PhaseResult.Output == nil ||
		reviewer.Result.PhaseResult.Output.Clear == nil {
		return published, empty, errors.New("clear review needs its recorded typed verdict")
	}
	report := *reviewer.Result.PhaseResult.Output.Clear
	if in.Cycle.CorrectionAssignment != "" && in.Cycle.CorrectionAssignment == reviewer.ID {
		return published, empty, errors.New("reviewer and corrector sessions must stay distinct")
	}
	if err := report.Validate(in.Carried); err != nil {
		return published, empty, err
	}
	if report.CycleID != in.Cycle.ID {
		return published, empty, errors.New("clear review answers another cycle")
	}
	content, err := factory.FindingContentDigest(in.Carried)
	if err != nil {
		return published, empty, err
	}
	coverage, err := factory.ClearCoverageDigest(in.Cycle.Head, in.Cycle.BaseOID, content)
	if err != nil {
		return published, empty, err
	}
	if report.CoverageDigest != coverage {
		return published, empty, errors.New("clear review coverage answers another input")
	}
	if report.Head != in.Cycle.Head || report.BaseOID != in.Cycle.BaseOID {
		return published, empty, errors.New("clear review answers a superseded tuple")
	}
	if in.Cycle.Head != published.Candidate || in.Cycle.BaseOID != published.PRCreate.BaseOID {
		return published, empty, errors.New("clear review answers a superseded tuple")
	}
	if reviewer.SourceCommit != in.Cycle.Head {
		return published, empty, errors.New("reviewer finished on another candidate")
	}
	if !factory.ValidDigest(in.ScopeDigest) || !factory.ValidDigest(in.CheckDigest) {
		return published, empty, errors.New("clear review lacks its bound evidence digests")
	}
	tuple := factory.CandidateTuple{Head: in.Cycle.Head, Base: in.Cycle.BaseOID, DiffBase: in.DiffBase}
	if published.Tuple != nil && published.Tuple.Head == in.Cycle.Head && published.Tuple.Base == in.Cycle.BaseOID {
		tuple = *published.Tuple
	} else if err := tuple.Validate(); err != nil {
		return published, empty, err
	}
	acceptance := factory.DeliveryAcceptance{
		Tuple: tuple, ID: factory.DeliveryAcceptanceID(published.ID, in.Cycle.AttemptRoot, in.Cycle.Round, factory.AcceptanceCleanReview),
		PublicationID: published.ID, AttemptRoot: in.Cycle.AttemptRoot,
		ScopeDigest: in.ScopeDigest, FindingDigest: content, CoverageDigest: coverage, CheckDigest: in.CheckDigest,
		PolicyRevision: in.PolicyRevision, OriginReceipt: reviewer.ID,
		Kind: factory.AcceptanceCleanReview, Round: in.Cycle.Round,
	}
	if err := acceptance.Validate(); err != nil {
		return published, empty, err
	}
	if err := appendDeliveryAcceptance(&published, acceptance); err != nil {
		return published, empty, err
	}
	return published, acceptance, nil
}

// JevAdmissionInput is the controller's complete funded-decision input:
// the publication and its recorded review cycle, the controller's
// canonical carried findings, the exact fingerprint evidence the funded
// decision judged with its fingerprint, the decided funded delivery
// decision, the funded call identity behind it, the controller-observed
// scope/policy/check evidence the acceptance binds, and the intended
// native resolution operations. No new funded call happens here: the
// decision is the already-paid evidence this acceptance records.
type JevAdmissionInput struct {
	Publication       factory.Publication
	Cycle             factory.ReviewCycle
	Carried           []factory.Finding
	FingerprintInput  factory.DeliveryFingerprintInput
	Fingerprint       string
	Decision          factory.DeliveryDecision
	DecisionID        string
	ScopeDigest       string
	PolicyRevision    string
	CheckDigest       string
	DiffBase          string
	ResolutionIntents []string
}

// AdmitJevAcceptance admits one funded ready_to_merge delivery decision
// as jev_acceptance delivery acceptance and journals it on the
// publication. Every D07/D08 gate runs here:
//
//   - the cycle is recorded on this publication and validates, over the
//     publication's exact current candidate and PR creation base;
//   - the fingerprint evidence recomputes exactly and matches both the
//     supplied fingerprint and the decision's judged fingerprint, over
//     this publication, cycle round, head and base, so stale semantic
//     evidence never rides a prior decision;
//   - the decision validates with the funded model behind it and its
//     effective action is exactly ready_to_merge: review_again,
//     judgment_attention and unavailable decisions never create
//     acceptance — only positive evidence does;
//   - the funded call identity is a valid charged request distinct from
//     the origin receipt, which names the cycle's independent reviewer
//     assignment behind the judged round;
//   - the finding digest is always the D06 complete content digest
//     computed from canonical findings at admission, and the coverage
//     digest recomputes from it exactly like the clear path.
//
// The union type and its validation already admit both kinds; merge-entry
// gating consumes the journal in FW37.
func AdmitJevAcceptance(in JevAdmissionInput) (factory.Publication, factory.DeliveryAcceptance, error) {
	var empty factory.DeliveryAcceptance
	published := in.Publication
	if err := in.Cycle.Validate(); err != nil {
		return published, empty, err
	}
	recorded := false
	for _, cycle := range published.ReviewCycles {
		if cycle.ID == in.Cycle.ID {
			recorded = true
			break
		}
	}
	if !recorded || in.Cycle.PublicationID != published.ID {
		return published, empty, errors.New("funded decision consumes an unrecorded cycle")
	}
	if in.Cycle.Head != published.Candidate || in.Cycle.BaseOID != published.PRCreate.BaseOID {
		return published, empty, errors.New("funded decision answers a superseded tuple")
	}
	if err := in.FingerprintInput.Validate(); err != nil {
		return published, empty, err
	}
	bound, err := factory.DeliveryFingerprint(in.FingerprintInput)
	if err != nil {
		return published, empty, err
	}
	if !factory.ValidDigest(in.Fingerprint) || bound != in.Fingerprint {
		return published, empty, errors.New("funded decision evidence differs from its fingerprint")
	}
	if in.FingerprintInput.PublicationID != published.ID || in.FingerprintInput.AttemptRoot != in.Cycle.AttemptRoot ||
		in.FingerprintInput.Round != in.Cycle.Round || in.FingerprintInput.Head != in.Cycle.Head ||
		in.FingerprintInput.Base != in.Cycle.BaseOID {
		return published, empty, errors.New("funded decision judged another round")
	}
	if err := in.Decision.Validate(); err != nil {
		return published, empty, err
	}
	if in.Decision.Answer.Fingerprint != in.Fingerprint || in.Decision.Answer.Model != in.FingerprintInput.Model {
		return published, empty, errors.New("funded decision judged different evidence")
	}
	if in.Decision.Effective != factory.DeliveryEffectiveReady {
		return published, empty, errors.New("only a ready funded decision creates acceptance")
	}
	if !factory.ValidID(in.DecisionID) {
		return published, empty, errors.New("funded acceptance lacks its funded call identity")
	}
	if !factory.ValidID(in.Cycle.ReviewerAssignment) {
		return published, empty, errors.New("funded acceptance lacks its independent reviewer run")
	}
	if in.Cycle.CorrectionAssignment != "" && in.Cycle.CorrectionAssignment == in.Cycle.ReviewerAssignment {
		return published, empty, errors.New("reviewer and corrector sessions must stay distinct")
	}
	content, err := factory.FindingContentDigest(in.Carried)
	if err != nil {
		return published, empty, err
	}
	coverage, err := factory.ClearCoverageDigest(in.Cycle.Head, in.Cycle.BaseOID, content)
	if err != nil {
		return published, empty, err
	}
	if !factory.ValidDigest(in.ScopeDigest) || !factory.ValidDigest(in.CheckDigest) {
		return published, empty, errors.New("funded acceptance lacks its bound evidence digests")
	}
	tuple := factory.CandidateTuple{Head: in.Cycle.Head, Base: in.Cycle.BaseOID, DiffBase: in.DiffBase}
	if published.Tuple != nil && published.Tuple.Head == in.Cycle.Head && published.Tuple.Base == in.Cycle.BaseOID {
		tuple = *published.Tuple
	} else if err := tuple.Validate(); err != nil {
		return published, empty, err
	}
	acceptance := factory.DeliveryAcceptance{
		Tuple: tuple, ID: factory.DeliveryAcceptanceID(published.ID, in.Cycle.AttemptRoot, in.Cycle.Round, factory.AcceptanceJevAcceptance),
		PublicationID: published.ID, AttemptRoot: in.Cycle.AttemptRoot,
		ScopeDigest: in.ScopeDigest, FindingDigest: content, CoverageDigest: coverage, CheckDigest: in.CheckDigest,
		PolicyRevision: in.PolicyRevision, DecisionID: in.DecisionID, OriginReceipt: in.Cycle.ReviewerAssignment,
		ResolutionIntents: in.ResolutionIntents,
		Kind:              factory.AcceptanceJevAcceptance, Round: in.Cycle.Round,
	}
	if err := acceptance.Validate(); err != nil {
		return published, empty, err
	}
	if err := appendDeliveryAcceptance(&published, acceptance); err != nil {
		return published, empty, err
	}
	return published, acceptance, nil
}

// appendDeliveryAcceptance records one admitted acceptance as the
// publication's next immutable journal entry: the acceptance validates,
// belongs to this publication and carries a fresh derived identity. A
// duplicate identity refuses; replay of an admission is identical, so a
// retried admit rebuilds the same entry instead of appending twice.
func appendDeliveryAcceptance(p *factory.Publication, acceptance factory.DeliveryAcceptance) error {
	if err := acceptance.Validate(); err != nil {
		return err
	}
	if acceptance.PublicationID != p.ID {
		return errors.New("delivery acceptance belongs to another publication")
	}
	for _, recorded := range p.DeliveryAcceptances {
		if recorded.ID == acceptance.ID {
			return errors.New("delivery acceptance already recorded")
		}
	}
	p.DeliveryAcceptances = append(p.DeliveryAcceptances, acceptance)
	return nil
}

// AdmitCorrectionReportCanonical admits one corrector run's report against
// its review cycle exactly like AdmitCorrectionReport, after binding the
// submitted findings to the controller's canonical set byte-for-byte. An
// agent submission that rewrites any canonical defect, provenance,
// severity, actionability, outcome, fix kind or location refuses before
// disposition rules run. Callers without a canonical set keep the legacy
// entrypoint; FW37 wires this canonical gate into every serving entry.
func AdmitCorrectionReportCanonical(cycle factory.ReviewCycle, runID, newCandidate string, canonical []factory.Finding, report factory.CorrectionReport, evidence []factory.CorrectionFixEvidence, checks *factory.CheckAssessment) (CorrectionAdmission, error) {
	var empty CorrectionAdmission
	if err := factory.CheckSubmittedFindings(canonical, report.Findings); err != nil {
		return empty, err
	}
	return AdmitCorrectionReport(cycle, runID, newCandidate, report, evidence, checks)
}
