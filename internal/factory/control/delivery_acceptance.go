package control

import (
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// ClearAdmissionInput is the controller's complete explicit-clear input:
// the publication and its recorded review cycle, the independent reviewer
// run, the reviewer's explicit clear report, the controller's canonical
// carried findings, the reviewer's finished process result, and the
// controller-observed scope/policy/check evidence the acceptance binds.
// No allowance, chooser or delivery-call reservation travels here: a clear
// review creates acceptance with zero delivery calls by construction.
type ClearAdmissionInput struct {
	Publication    factory.Publication
	Cycle          factory.ReviewCycle
	Report         factory.ClearReviewReport
	Carried        []factory.Finding
	Result         factory.Result
	ScopeDigest    string
	PolicyRevision string
	CheckDigest    string
	ReviewerRunID  string
	DiffBase       string
}

// AdmitClearReview admits one completed independent explicit clear review
// as clean_review delivery acceptance and journals it on the publication.
// Every D07/T41-T43 gate runs here:
//
//   - the cycle is recorded on this publication and validates;
//   - the report is the reviewer's explicit clear over the controller's
//     canonical carried set (silence, native approval, empty output,
//     incomplete coverage, omitted carried findings and stale tuples
//     refuse);
//   - the coverage digest recomputed from the canonical set must equal the
//     reported one, so same-key rewritten content invalidates;
//   - the report answers the cycle's exact head/base, which must still be
//     the publication's current candidate and PR creation base;
//   - the reviewer run is the cycle's independent reviewer assignment,
//     distinct from any corrector session;
//   - the reviewer process finished completed on the exact head.
//
// The funded jev_acceptance variant is constructed by FW36 alongside D08
// routing; the union type and its validation already admit both kinds.
func AdmitClearReview(in ClearAdmissionInput) (factory.Publication, factory.DeliveryAcceptance, error) {
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
		return published, empty, errors.New("clear review consumes an unrecorded cycle")
	}
	if !factory.ValidID(in.ReviewerRunID) || in.ReviewerRunID != in.Cycle.ReviewerAssignment {
		return published, empty, errors.New("clear review needs its independent reviewer run")
	}
	if in.Cycle.CorrectionAssignment != "" && in.Cycle.CorrectionAssignment == in.Cycle.ReviewerAssignment {
		return published, empty, errors.New("reviewer and corrector sessions must stay distinct")
	}
	if err := in.Report.Validate(in.Carried); err != nil {
		return published, empty, err
	}
	if in.Report.CycleID != in.Cycle.ID {
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
	if in.Report.CoverageDigest != coverage {
		return published, empty, errors.New("clear review coverage answers another input")
	}
	if in.Report.Head != in.Cycle.Head || in.Report.BaseOID != in.Cycle.BaseOID {
		return published, empty, errors.New("clear review answers a superseded tuple")
	}
	if in.Cycle.Head != published.Candidate || in.Cycle.BaseOID != published.PRCreate.BaseOID {
		return published, empty, errors.New("clear review answers a superseded tuple")
	}
	candidate, err := in.Result.CompletedCandidate()
	if err != nil {
		return published, empty, err
	}
	if candidate != in.Cycle.Head {
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
		PolicyRevision: in.PolicyRevision, OriginReceipt: in.ReviewerRunID,
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
func AdmitCorrectionReportCanonical(cycle factory.ReviewCycle, runID, newCandidate string, canonical []factory.Finding, report factory.CorrectionReport, evidence []CorrectionFixEvidence, checks *factory.CheckAssessment) (CorrectionAdmission, error) {
	var empty CorrectionAdmission
	if err := factory.CheckSubmittedFindings(canonical, report.Findings); err != nil {
		return empty, err
	}
	return AdmitCorrectionReport(cycle, runID, newCandidate, report, evidence, checks)
}
