package control

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// MaxCorrectionReportBytes bounds the complete fenced correction report block
// the parser attempts to decode. A correction report carries the same bounded
// finding set as a review report plus one small disposition per key, so it
// shares the review-report ceiling: worst valid encodings stay below 2 MiB.
const MaxCorrectionReportBytes = 2 << 20

// ParseCorrectionReport extracts the last fenced correction block from
// corrector output and validates it strictly. Unknown fields refuse; anything
// unparseable reports false instead of guessing dispositions.
func ParseCorrectionReport(output string) (factory.CorrectionReport, bool) {
	start := strings.LastIndex(output, "```"+factory.CorrectionReportFence)
	if start < 0 {
		return factory.CorrectionReport{}, false
	}
	rest := output[start+len("```"+factory.CorrectionReportFence):]
	for offset := 0; offset < len(rest); {
		rel := strings.Index(rest[offset:], "```")
		if rel < 0 {
			return factory.CorrectionReport{}, false
		}
		body := strings.TrimSpace(rest[:offset+rel])
		offset += rel + len("```")
		if body == "" || len(body) > MaxCorrectionReportBytes {
			continue
		}
		var report factory.CorrectionReport
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&report); err != nil {
			continue
		}
		if report.Validate() != nil {
			continue
		}
		return report, true
	}
	return factory.CorrectionReport{}, false
}

// CorrectionAdmission is one corrector run's admitted report: the cycle and
// run it answered, the finding-set digest covered, and the disposition keys
// by outcome. Admitted reports proceed to judgment; judgment (FW21) decides
// review_again or ready_to_merge, and native resolution (FW22) follows only
// an acceptance. Admission never resolves threads itself.
type CorrectionAdmission struct {
	CycleID          string   `json:"cycle_id"`
	RunID            string   `json:"run_id"`
	SetDigest        string   `json:"set_digest"`
	NewCandidate     string   `json:"new_candidate"`
	Fixed            []string `json:"fixed,omitempty"`
	Unresolved       []string `json:"unresolved,omitempty"`
	DeclinedOptional []string `json:"declined_optional,omitempty"`
}

// CorrectionBlock reports one correction that needs maintainer intervention
// instead of a retry: disputed findings use existing intervention, and only
// a human decision unblocks them. Refusals (plain errors) carry a corrected
// report; blocks do not.
type CorrectionBlock struct {
	Reason string
}

func (b *CorrectionBlock) Error() string { return "correction needs intervention: " + b.Reason }

// AdmitCorrectionReport validates one corrector run's report against the
// review cycle it answered. Every finding is accounted for exactly once
// (FW18 report validation plus the cycle's exact finding-set digest);
// missing, duplicate, unknown or stale dispositions refuse. Disputed findings
// block with intervention and never reach judgment. Actionable findings
// declined as optional refuse; optional-only reports admit without forcing
// failure. Fixed code findings require a changed candidate plus relevant
// changed paths/commits on the exact new tree; verification-only findings
// may retain the same head when new passing checks on that exact head prove
// the required outcome. Unchanged code cannot claim a code fix.
func AdmitCorrectionReport(cycle factory.ReviewCycle, runID, newCandidate string, report factory.CorrectionReport, evidence []factory.CorrectionFixEvidence, checks *factory.CheckAssessment) (CorrectionAdmission, error) {
	var empty CorrectionAdmission
	if err := cycle.Validate(); err != nil {
		return empty, err
	}
	if !factory.ValidID(runID) {
		return empty, errors.New("invalid correction run")
	}
	if !factory.ValidCommit(newCandidate) {
		return empty, errors.New("correction lacks its exact candidate")
	}
	if err := report.Validate(); err != nil {
		return empty, err
	}
	// The retained candidate binds the report's own nomination: the
	// settlement-mapped candidate and the corrector's nominated tree must
	// be the same commit, so a mismatched wiring refuses instead of
	// advancing a tree the report never accounted.
	if report.NewCandidate != newCandidate {
		return empty, errors.New("correction nomination differs from its recorded candidate")
	}
	if report.SetDigest != cycle.FindingDigest {
		return empty, errors.New("correction dispositions answer a changed finding set")
	}
	byKey := make(map[string]factory.Finding, len(report.Findings))
	for _, finding := range report.Findings {
		byKey[finding.Key] = finding
	}
	evidenceByKey := make(map[string]factory.CorrectionFixEvidence, len(evidence))
	for _, entry := range evidence {
		if err := entry.Validate(); err != nil {
			return empty, err
		}
		if _, dup := evidenceByKey[entry.Key]; dup {
			return empty, errors.New("correction fix evidence repeats its key")
		}
		evidenceByKey[entry.Key] = entry
	}
	admission := CorrectionAdmission{
		CycleID: cycle.ID, RunID: runID,
		SetDigest: report.SetDigest, NewCandidate: newCandidate,
	}
	for _, disposition := range report.Dispositions {
		finding := byKey[disposition.Key]
		entry, hasEvidence := evidenceByKey[disposition.Key]
		switch disposition.Disposition {
		case factory.DispositionDisputed:
			return empty, &CorrectionBlock{Reason: "disputed finding " + disposition.Key}
		case factory.DispositionFixed:
			if finding.FixKind == factory.FindingFixCode && newCandidate == cycle.Head {
				return empty, errors.New("unchanged code cannot fix a code defect")
			}
			if !hasEvidence {
				if finding.FixKind == factory.FindingFixCode || newCandidate != cycle.Head {
					return empty, errors.New("fixed finding lacks its changed paths and commits")
				}
			} else if len(entry.ChangedPaths) == 0 && len(entry.Commits) == 0 {
				return empty, errors.New("fixed finding lacks its changed paths and commits")
			}
			if newCandidate == cycle.Head {
				if checks == nil || checks.Verdict != factory.CheckPass || checks.HeadOID != newCandidate {
					return empty, errors.New("same-head fix lacks new passing checks on its exact head")
				}
			}
			admission.Fixed = append(admission.Fixed, disposition.Key)
		case factory.DispositionUnresolved:
			if hasEvidence {
				return empty, errors.New("unresolved finding carries fix evidence")
			}
			admission.Unresolved = append(admission.Unresolved, disposition.Key)
		case factory.DispositionDeclinedOptional:
			if hasEvidence {
				return empty, errors.New("declined optional finding carries fix evidence")
			}
			admission.DeclinedOptional = append(admission.DeclinedOptional, disposition.Key)
		default:
			return empty, errors.New("invalid correction disposition")
		}
	}
	for key := range evidenceByKey {
		if _, ok := byKey[key]; !ok {
			return empty, errors.New("correction fix evidence answers an unknown finding")
		}
		fixed := false
		for _, disposition := range report.Dispositions {
			if disposition.Key == key && disposition.Disposition == factory.DispositionFixed {
				fixed = true
				break
			}
		}
		if !fixed {
			return empty, errors.New("correction fix evidence answers an unfixed finding")
		}
	}
	if checks != nil {
		if err := checks.Validate(); err != nil {
			return empty, errors.New("correction check evidence is malformed")
		}
		if checks.HeadOID != newCandidate && checks.HeadOID != cycle.Head {
			return empty, errors.New("correction check evidence binds another head")
		}
	}
	return admission, nil
}

// admitFirstCorrectionReport binds a first correction report to the
// cycle's canonical carried findings before any journal/ref effect. The
// submitted findings must exactly equal the owning history's
// reconstruction (same keys, same content, same provenance), so a
// same-key rewritten defect, an omission or a changed origin refuses
// instead of journaling. The round key stays the finding-set digest;
// this content binding is the separate domain that keeps key equality
// from standing in for content freshness. Later reports for the same
// cycle answer the first journal's canonical set through
// AdmitCorrectionReportCanonical. PublishCorrectionWithReport's
// first-admission branch enters here.
func (c *Coordinator) admitFirstCorrectionReport(ctx context.Context, p factory.Publication, cycle factory.ReviewCycle, runID, newCandidate string, report factory.CorrectionReport, evidence []factory.CorrectionFixEvidence, checks *factory.CheckAssessment) (CorrectionAdmission, error) {
	canonical, err := c.reviewCarriedFindings(ctx, p, cycle)
	if err != nil {
		return CorrectionAdmission{}, err
	}
	return AdmitCorrectionReportCanonical(cycle, runID, newCandidate, canonical, report, evidence, checks)
}

// appendCorrectionJournal records one admitted report as the publication's
// next immutable correction journal entry: the cycle must belong to this
// publication, the run reports once, and the report answers the cycle's
// exact finding-set digest. The store write re-validates the whole journal
// (FW18 rules); this keeps refused reports from ever reaching it.
func appendCorrectionJournal(p *factory.Publication, cycle factory.ReviewCycle, runID string, report factory.CorrectionReport) error {
	if cycle.PublicationID != p.ID {
		return errors.New("correction consumes another publication's cycle")
	}
	for _, journal := range p.CorrectionJournals {
		if journal.RunID == runID {
			return errors.New("correction run reported twice")
		}
	}
	journal := factory.CorrectionJournal{CycleID: cycle.ID, RunID: runID, Report: report}
	if err := journal.Validate(); err != nil {
		return err
	}
	if journal.Report.SetDigest != cycle.FindingDigest {
		return errors.New("correction dispositions answer a changed finding set")
	}
	known := false
	for _, recorded := range p.ReviewCycles {
		if recorded.ID == cycle.ID {
			known = true
			break
		}
	}
	if !known {
		return errors.New("correction consumes an unrecorded cycle")
	}
	p.CorrectionJournals = append(p.CorrectionJournals, journal)
	return nil
}

// AdmitAndJournalCorrection admits one corrector run's report against its
// recorded cycle and appends the immutable journal entry. Refused or blocked
// reports change nothing; admitted reports persist exactly one journal entry
// per run over the cycle's exact finding-set digest. Journaling records the
// report for judgment; it never resolves native threads.
func (c *Coordinator) AdmitAndJournalCorrection(ctx context.Context, ownerAssignmentID, cycleID, runID, newCandidate string, report factory.CorrectionReport, evidence []factory.CorrectionFixEvidence, checks *factory.CheckAssessment) (CorrectionAdmission, error) {
	var empty CorrectionAdmission
	if c.Store == nil {
		return empty, errors.New("correction journal unavailable")
	}
	p, err := c.Store.PublicationByAssignment(ctx, ownerAssignmentID)
	if err != nil {
		return empty, errors.New("correction publication missing")
	}
	var cycle *factory.ReviewCycle
	for i := range p.ReviewCycles {
		if p.ReviewCycles[i].ID == cycleID {
			cycle = &p.ReviewCycles[i]
			break
		}
	}
	if cycle == nil {
		return empty, errors.New("correction consumes an unrecorded cycle")
	}
	admission, err := AdmitCorrectionReport(*cycle, runID, newCandidate, report, evidence, checks)
	if err != nil {
		return empty, err
	}
	if err := appendCorrectionJournal(&p, *cycle, runID, report); err != nil {
		return empty, err
	}
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, p); err != nil {
		return empty, errors.New("correction journal unrecorded")
	}
	return admission, nil
}
