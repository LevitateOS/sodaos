package control

// FW40 role consumers for recorded MCP terminal proposals.
//
// The fourteen-tool catalog records typed worker finishes as immutable
// proposals whose payload fields are loose: the transport bounds keys,
// enums and digests but the typed domain payload (a decomposition, an
// investigation plan, a review verdict, a reassessment or verification
// report) rides inside existing fields only when the worker supplies it
// in structured form. These consumers settle a proposal only when that
// structured carrier is present and strictly decodes into the existing
// typed PhaseResult output for the assignment's phase, binds to the
// assignment, and finishes through the same joints the run-output path
// uses (phase binding, decompose semantic settlement, repair chain).
//
// A proposal without a structured carrier (prose summaries such as
// candidates:["g1"] or report:"done") proves nothing typed: it stays
// recorded and parked, visible through its receipt, and touches
// nothing. A structured carrier that fails strict decode, validation
// or assignment binding fences the assignment as blocked for human
// attention instead of wedging it: the worker attempted a typed finish
// and that report stays inspectable through the accepted receipt.
// Nothing here coerces worker prose into settlement facts, and no
// downstream effect runs here: finished typed results advance through
// the existing dispatch, publication, review and adoption consumers
// exactly like harness-reported phase results.

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// mcpReviewTuple is the strict carrier shape for a review finish's
// tuple field: the cycle the verdict answers plus the exact head and
// base it covered. No other keys admit.
type mcpReviewTuple struct {
	CycleID string `json:"cycle_id"`
	Head    string `json:"head"`
	BaseOID string `json:"base_oid"`
}

// mapFW40Terminal maps one parked-role terminal proposal to its
// finished assignment. handled=false parks the proposal untouched.
// Carrier presence decides handling: absent carriers park, present
// carriers either settle typed or fence.
func mapFW40Terminal(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time) (finished factory.Assignment, note string, handled bool, err error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(submission.Payload, &fields); err != nil {
		return fenceMCPTerminal(assigned, submission, now, "terminal payload refused: "+err.Error()),
			"unparseable terminal fenced for human attention", true, nil
	}
	switch submission.Tool {
	case factory.MCPToolFinishProductPlan:
		return settleProductPlan(assigned, submission, fields, now)
	case factory.MCPToolFinishIssuePlan:
		return settleIssuePlan(assigned, submission, fields, now)
	case factory.MCPToolFinishReview:
		return settleReviewFinish(assigned, submission, fields, now)
	case factory.MCPToolFinishReassessment:
		return settleReassessmentFinish(assigned, submission, fields, now)
	case factory.MCPToolFinishAppVerify:
		return settleAppVerification(assigned, submission, fields, now)
	default:
		return factory.Assignment{}, "", false, nil
	}
}

// fenceFW40 finishes one assignment as blocked when its structured
// carrier cannot settle, through the shared terminal fence.
func fenceFW40(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time, detail string) (factory.Assignment, string, bool, error) {
	return fenceMCPTerminal(assigned, submission, now, detail),
		"structured terminal fenced for human attention", true, nil
}

// parkFW40 reports an unhandled proposal: no writes, receipt untouched.
func parkFW40() (factory.Assignment, string, bool, error) {
	return factory.Assignment{}, "", false, nil
}

func rawField(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, ok := fields[key]
	return raw, ok
}

func trimmedRaw(raw json.RawMessage) []byte {
	return bytes.TrimSpace(raw)
}

// isObjectRaw reports whether a payload field carries a JSON object.
func isObjectRaw(raw json.RawMessage, ok bool) bool {
	if !ok {
		return false
	}
	trimmed := trimmedRaw(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// isObjectArrayRaw reports whether a payload field carries a JSON
// array whose first element is an object: the structured-carrier shape
// for list fields. String lists and empty lists are not carriers.
func isObjectArrayRaw(raw json.RawMessage, ok bool) bool {
	if !ok {
		return false
	}
	trimmed := trimmedRaw(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return false
	}
	rest := bytes.TrimSpace(trimmed[1:])
	return len(rest) > 0 && rest[0] == '{'
}

// isArrayRaw reports whether a payload field carries any JSON array.
func isArrayRaw(raw json.RawMessage, ok bool) bool {
	if !ok {
		return false
	}
	trimmed := trimmedRaw(raw)
	return len(trimmed) > 0 && trimmed[0] == '['
}

// stringField decodes one payload field as a JSON string. Any other
// shape reports absent: the settler never coerces structured values
// into text facts.
func stringField(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := fields[key]
	if !ok {
		return "", false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", false
	}
	return text, true
}

// strictDecodeField decodes one carrier field with unknown fields
// refused and no trailing content, mirroring ParsePhaseResult.
func strictDecodeField(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errTrailingFieldContent
	}
	return nil
}

var errTrailingFieldContent = fieldError("carrier field carries trailing content")

type fieldError string

func (e fieldError) Error() string { return string(e) }

// phaseResultFor builds one completed strict v2 phase result naming
// its MCP tool and submission as its summary. The summary is a
// controller label, never evidence.
func phaseResultFor(submission factory.MCPSubmission, phase, fallback string, output *factory.PhaseOutput) factory.PhaseResult {
	return factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: phase, Status: "completed",
		Summary: mcpTerminalSummary(submission.Tool, submission.SubmissionID, "", fallback),
		Output:  output,
	}
}

// finishMCPBlocked finishes one assignment from a worker's own
// incomplete verdict as a synthesized blocked phased result, the
// same shape the run-output path records for non-completed reviewer
// runs: a reported reviewer result must be a completed verdict, so
// an incomplete verdict can only settle synthesized. The summary
// names the tool and submission behind the verdict.
func finishMCPBlocked(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time, fallback string) factory.Assignment {
	result := synthesizePhased(assigned, "blocked",
		mcpTerminalSummary(submission.Tool, submission.SubmissionID, "", fallback), now.Unix())
	finished := assigned
	finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, factory.NeedsHuman, factory.AssignReasonNoReport
	finished.FinishedUnix = now.Unix()
	finished.Result = &result
	return finished
}

// finishFromMCPPhaseResult finishes one assignment from a bound typed
// phase result through the same builders and semantic joint the
// run-output settlement path uses. Binding or shape failures fence
// instead of settling as fabricated success.
func finishFromMCPPhaseResult(assigned factory.Assignment, submission factory.MCPSubmission, parsed factory.PhaseResult, now time.Time) factory.Assignment {
	fence := func(detail string) factory.Assignment {
		return fenceMCPTerminal(assigned, submission, now, detail)
	}
	if !factory.ValidPhaseRole(assigned.Phase, assigned.Role) {
		return fence("assignment role does not match its execution phase")
	}
	if err := parsed.BindAssignment(assigned); err != nil {
		return fence("phase result does not bind its assignment: " + err.Error())
	}
	var result factory.AssignmentResult
	var outcome factory.Outcome
	var reason string
	if assigned.Role == project.RoleReviewer {
		built, builtOutcome, builtReason, ok := buildMCPReviewerResult(assigned, parsed, now.Unix())
		if !ok {
			return fence("phased result does not bind its assignment phase")
		}
		result, outcome, reason = built, builtOutcome, builtReason
	} else {
		result, outcome, reason = buildPhasedCoderResult(assigned, parsed, now.Unix())
	}
	finished := assigned
	finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, outcome, reason
	finished.FinishedUnix = now.Unix()
	finished.Result = &result
	if result.Reported && finished.Phase == project.ExecutionDecompose && finished.Subject.Intent != nil {
		if settled, valid := settleDecomposeSemantic(finished, now); !valid {
			finished.Outcome, finished.Reason, finished.Result = settled.Outcome, settled.Reason, settled.Result
		}
	}
	return finished
}

// buildMCPReviewerResult mirrors the phased reviewer builders in
// dispatch_result.go for a result that arrived by MCP instead of run
// output: reported, phase-bound, no candidate, no legacy findings.
func buildMCPReviewerResult(a factory.Assignment, parsed factory.PhaseResult, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string, bool) {
	result := factory.AssignmentResult{
		AssignmentID: a.ID, RunID: a.Run, Phase: a.Phase, PhaseResult: &parsed,
		Summary: parsed.Summary, Findings: []string{},
		Reported: true, RecordedUnix: recordedUnix,
	}
	switch parsed.Status {
	case "completed":
		result.Status = "completed"
	case "blocked":
		result.Status = "blocked"
	case "failed":
		result.Status = "failed"
		return result, factory.Failed, factory.AssignReasonRunFailed, true
	case "cancelled":
		result.Status = "cancelled"
		return result, factory.Cancelled, factory.AssignReasonCancelled, true
	default:
		return factory.AssignmentResult{}, "", "", false
	}
	if err := result.ValidatePhaseBinding(a.Role, a.Phase); err != nil {
		return factory.AssignmentResult{}, "", "", false
	}
	if err := result.ValidateForAssignment(a.Role, a.SourceCommit); err != nil {
		return factory.AssignmentResult{}, "", "", false
	}
	outcome := factory.Outcome(factory.Succeeded)
	reason := factory.AssignReasonReported
	if parsed.Status == "blocked" {
		outcome, reason = factory.NeedsHuman, factory.AssignReasonNoReport
	}
	return result, outcome, reason, true
}

// settleProductPlan settles one Product Lead finish whose candidates
// carrier holds structured semantic candidates. Intent, revision and
// operation bind from controller-recorded facts (the assignment
// subject and its staged planning input), never from worker payload:
// without a staged input there is no operation to bind, so the
// proposal parks instead of inventing one. A verification manifest
// proposal remains a typed proposal until exact accepted graph adoption.
func settleProductPlan(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time) (factory.Assignment, string, bool, error) {
	raw, ok := rawField(fields, "candidates")
	if !isObjectArrayRaw(raw, ok) {
		return parkFW40()
	}
	if assigned.Phase != project.ExecutionDecompose {
		return fenceFW40(assigned, submission, now, "product plan finish does not match its assignment phase")
	}
	in, staged, err := ParseStagedPlanningInput(assigned.Prompt)
	if err != nil || !staged {
		return parkFW40()
	}
	if assigned.Subject.Intent == nil {
		return fenceFW40(assigned, submission, now, "product plan finish lacks its intent subject")
	}
	var candidates []factory.SemanticCandidate
	if err := strictDecodeField(raw, &candidates); err != nil {
		return fenceFW40(assigned, submission, now, "product plan candidates refused: "+err.Error())
	}
	out := factory.SemanticPlannerOutput{
		Intent: assigned.Subject.Intent.IntentID, Revision: assigned.Subject.Intent.RevisionID,
		Operation: in.Operation, Candidates: candidates,
	}
	if raw, present := fields["verification_manifest_proposal"]; present {
		proposal, err := factory.ParseVerificationManifestProposal(raw)
		if err != nil || proposal.ValidateOutcomes(in.Head) != nil {
			return fenceFW40(assigned, submission, now, "product plan verification manifest differs from its planning scope")
		}
		out.VerificationManifestProposal = &proposal
	}
	parsed := phaseResultFor(submission, project.ExecutionDecompose, "worker proposes product plan",
		&factory.PhaseOutput{Decomposition: &out})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "product plan settled", true, nil
}

// settleIssuePlan settles one Architect finish whose tasks carrier
// holds the full typed investigation plan object. A tasks list or
// prose alone cannot prove an executable plan, so it parks. The plan
// must answer this assignment's attempt root and admitted source,
// and agree with the payload approach when one is carried.
func settleIssuePlan(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time) (factory.Assignment, string, bool, error) {
	raw, ok := rawField(fields, "tasks")
	if !isObjectRaw(raw, ok) {
		return parkFW40()
	}
	if assigned.Phase != project.ExecutionInvestigate {
		return fenceFW40(assigned, submission, now, "issue plan finish does not match its assignment phase")
	}
	var plan factory.InvestigationPlan
	if err := strictDecodeField(raw, &plan); err != nil {
		return fenceFW40(assigned, submission, now, "issue plan refused: "+err.Error())
	}
	if approach, present := stringField(fields, "approach"); present && approach != plan.Approach {
		return fenceFW40(assigned, submission, now, "issue plan approach contradicts its typed plan")
	}
	if plan.AttemptRoot != assigned.AttemptRoot || plan.SourceCommit != assigned.SourceCommit {
		return fenceFW40(assigned, submission, now, "issue plan answers another attempt or source")
	}
	parsed := phaseResultFor(submission, project.ExecutionInvestigate, "worker proposes issue plan",
		&factory.PhaseOutput{Plan: &plan})
	finished := finishFromMCPPhaseResult(assigned, submission, parsed, now)
	if finished.Result == nil || !finished.Result.Reported {
		return finished, "invalid issue plan fenced for human attention", true, nil
	}
	return finished, "issue plan settled", true, nil
}

// settleReviewFinish settles one Inspector finish. The conclusion
// enum is itself a typed worker verdict: incomplete settles as a
// reported blocked result. Clear and findings additionally need their
// structured carriers (tuple plus coverage digest, or tuple plus
// finding/assessment objects); without them the proposal parks —
// silence or prose coverage never clears work.
func settleReviewFinish(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time) (factory.Assignment, string, bool, error) {
	conclusion, ok := stringField(fields, "conclusion")
	if !ok {
		return parkFW40()
	}
	switch conclusion {
	case "clear", "findings", "incomplete":
	default:
		return parkFW40()
	}
	if conclusion == "incomplete" {
		if assigned.Phase != project.ExecutionReview {
			return fenceFW40(assigned, submission, now, "review finish does not match its assignment phase")
		}
		return finishMCPBlocked(assigned, submission, now, "worker reports incomplete review"), "incomplete review settled", true, nil
	}
	// Non-code subjects assess deliverables, never code cycles: a
	// deliverable binding replaces the tuple (and the clear coverage
	// digest). Research assesses its report, collections their content
	// and decisions; anything else fences. Code subjects keep the
	// tuple path below.
	if assigned.Subject.Kind == factory.SubjectResearch || assigned.Subject.Kind == factory.SubjectCollection {
		return settleNonCodeAssessment(assigned, submission, fields, now, conclusion)
	}
	// Carrier completeness parks before phase fencing: a verdict
	// without its tuple (and, for clear, its coverage digest) proves
	// nothing typed on any assignment.
	tupleRaw, tupleOK := rawField(fields, "tuple")
	if !isObjectRaw(tupleRaw, tupleOK) {
		return parkFW40()
	}
	if conclusion == "clear" {
		if coverage, coverageOK := stringField(fields, "coverage"); !coverageOK || !factory.ValidDigest(coverage) {
			return parkFW40()
		}
	}
	if assigned.Phase != project.ExecutionReview {
		return fenceFW40(assigned, submission, now, "review finish does not match its assignment phase")
	}
	var tuple mcpReviewTuple
	if err := strictDecodeField(tupleRaw, &tuple); err != nil {
		return fenceFW40(assigned, submission, now, "review tuple refused: "+err.Error())
	}
	return settleReviewVerdict(assigned, submission, fields, now, conclusion, tuple)
}

func settleReviewVerdict(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time, conclusion string, tuple mcpReviewTuple) (factory.Assignment, string, bool, error) {
	var assessments []factory.FindingAssessment
	if raw, present := rawField(fields, "carried_assessments"); present {
		if err := strictDecodeField(raw, &assessments); err != nil {
			return fenceFW40(assigned, submission, now, "review assessments refused: "+err.Error())
		}
	}
	if conclusion == "clear" {
		coverage, coverageOK := stringField(fields, "coverage")
		if !coverageOK || !factory.ValidDigest(coverage) {
			return parkFW40()
		}
		report := factory.ClearReviewReport{
			Conclusion: factory.ClearReviewConclusion, CycleID: tuple.CycleID,
			Head: tuple.Head, BaseOID: tuple.BaseOID, CoverageDigest: coverage,
			Assessments: assessments,
		}
		parsed := phaseResultFor(submission, project.ExecutionReview, "worker reports clear review",
			&factory.PhaseOutput{Clear: &report})
		return finishFromMCPPhaseResult(assigned, submission, parsed, now), "clear review settled", true, nil
	}
	findingsRaw, findingsOK := rawField(fields, "findings")
	_, assessmentsPresent := rawField(fields, "carried_assessments")
	if !isObjectArrayRaw(findingsRaw, findingsOK) && !assessmentsPresent {
		return parkFW40()
	}
	var findings []factory.Finding
	if findingsOK {
		if err := strictDecodeField(findingsRaw, &findings); err != nil {
			return fenceFW40(assigned, submission, now, "review findings refused: "+err.Error())
		}
	}
	report := factory.ReviewCycleReport{
		CycleID: tuple.CycleID, NewFindings: findings, Assessments: assessments,
	}
	parsed := phaseResultFor(submission, project.ExecutionReview, "worker reports review findings",
		&factory.PhaseOutput{Review: &report})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "review findings settled", true, nil
}

// settleNonCodeAssessment settles one Inspector finish over a non-code
// subject. The deliverable binds the verdict to its exact bytes — a
// recorded report submission or a library item revision — and the
// verdict carries either blocking findings with evidence or the
// explicit clear. No tuple, coverage digest, candidate, Git commit or
// app verification crosses this path; a code-shaped carrier on a
// non-code assignment fences.
func settleNonCodeAssessment(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time, conclusion string) (factory.Assignment, string, bool, error) {
	if assigned.Phase != project.ExecutionReview {
		return fenceFW40(assigned, submission, now, "assessment finish does not match its assignment phase")
	}
	raw, ok := rawField(fields, "deliverable")
	if !isObjectRaw(raw, ok) {
		return parkFW40()
	}
	var deliverable factory.NonCodeDeliverable
	if err := strictDecodeField(raw, &deliverable); err != nil {
		return fenceFW40(assigned, submission, now, "assessment deliverable refused: "+err.Error())
	}
	if err := factory.ValidateNonCodeSubject(deliverable, assigned.Subject.Kind); err != nil {
		return fenceFW40(assigned, submission, now, "assessment deliverable refused: "+err.Error())
	}
	if _, present := rawField(fields, "tuple"); present {
		return fenceFW40(assigned, submission, now, "assessment carries no code cycle")
	}
	if conclusion == "clear" {
		if _, present := rawField(fields, "findings"); present {
			return fenceFW40(assigned, submission, now, "assessment clear carries findings")
		}
		if _, present := rawField(fields, "carried_assessments"); present {
			return fenceFW40(assigned, submission, now, "assessment clear carries assessments")
		}
		assessment := &factory.NonCodeAssessment{Deliverable: deliverable, Clear: true}
		parsed := phaseResultFor(submission, project.ExecutionReview, "worker reports non-code assessment clear",
			&factory.PhaseOutput{Assessment: assessment})
		return finishFromMCPPhaseResult(assigned, submission, parsed, now), "non-code assessment clear settled", true, nil
	}
	findingsRaw, findingsOK := rawField(fields, "findings")
	_, assessmentsPresent := rawField(fields, "carried_assessments")
	if !isObjectArrayRaw(findingsRaw, findingsOK) && !assessmentsPresent {
		return parkFW40()
	}
	var findings []factory.Finding
	if findingsOK {
		if err := strictDecodeField(findingsRaw, &findings); err != nil {
			return fenceFW40(assigned, submission, now, "assessment findings refused: "+err.Error())
		}
	}
	var assessments []factory.FindingAssessment
	if assessmentsPresent {
		assessmentsRaw, _ := rawField(fields, "carried_assessments")
		if err := strictDecodeField(assessmentsRaw, &assessments); err != nil {
			return fenceFW40(assigned, submission, now, "assessment assessments refused: "+err.Error())
		}
	}
	assessment := &factory.NonCodeAssessment{Deliverable: deliverable, Findings: findings, Assessments: assessments}
	parsed := phaseResultFor(submission, project.ExecutionReview, "worker reports non-code assessment findings",
		&factory.PhaseOutput{Assessment: assessment})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "non-code assessment findings settled", true, nil
}

// settleReassessmentFinish settles one Strategist finish whose report
// carrier holds the typed report its admitted phase and subject
// select: a D13 reassessment report for reassess, a stall or delivery
// repair report for repair. The report is a proposal to existing
// adoption consumers, never a direct graph mutation here.
func settleReassessmentFinish(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time) (factory.Assignment, string, bool, error) {
	raw, ok := rawField(fields, "report")
	if !isObjectRaw(raw, ok) {
		return parkFW40()
	}
	var output *factory.PhaseOutput
	switch assigned.Phase {
	case project.ExecutionReassess:
		var report factory.ReassessmentReport
		if err := strictDecodeField(raw, &report); err != nil {
			return fenceFW40(assigned, submission, now, "reassessment report refused: "+err.Error())
		}
		output = &factory.PhaseOutput{Reassessment: &report}
	case project.ExecutionRepair:
		switch assigned.Subject.Kind {
		case factory.SubjectStall:
			var report factory.StallRepairReport
			if err := strictDecodeField(raw, &report); err != nil {
				return fenceFW40(assigned, submission, now, "stall repair report refused: "+err.Error())
			}
			output = &factory.PhaseOutput{Repair: &report}
		case factory.SubjectDeliveryRepair:
			var report factory.DeliveryRepairReport
			if err := strictDecodeField(raw, &report); err != nil {
				return fenceFW40(assigned, submission, now, "delivery repair report refused: "+err.Error())
			}
			output = &factory.PhaseOutput{DeliveryRepair: &report}
		default:
			return fenceFW40(assigned, submission, now, "repair finish does not match its repair subject")
		}
	default:
		return fenceFW40(assigned, submission, now, "reassessment finish does not match its assignment phase")
	}
	parsed := phaseResultFor(submission, assigned.Phase, "worker proposes reassessment", output)
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "reassessment settled", true, nil
}

// settleAppVerification settles one Test Pilot finish. Incomplete
// settles as a reported blocked result. Verified and failed need the
// exact delivery, admitted input digest, structured check receipts
// and coverage carriers; without them the proposal parks. The
// recorded report is a claim for the trusted-manifest consumers: it
// never deploys anything.
func settleAppVerification(assigned factory.Assignment, submission factory.MCPSubmission, fields map[string]json.RawMessage, now time.Time) (factory.Assignment, string, bool, error) {
	outcome, ok := stringField(fields, "outcome")
	if !ok {
		return parkFW40()
	}
	switch outcome {
	case factory.AppVerificationVerified, factory.AppVerificationFailed, factory.AppVerificationIncomplete:
	default:
		return parkFW40()
	}
	if outcome == factory.AppVerificationIncomplete {
		if assigned.Phase != project.ExecutionVerify {
			return fenceFW40(assigned, submission, now, "app verification finish does not match its assignment phase")
		}
		return finishMCPBlocked(assigned, submission, now, "worker reports incomplete verification"), "incomplete verification settled", true, nil
	}
	// Carrier completeness parks before phase fencing, as for review.
	deliveryID, deliveryOK := stringField(fields, "delivery_id")
	inputDigest, digestOK := stringField(fields, "input_digest")
	receiptsRaw, receiptsOK := rawField(fields, "check_receipts")
	coverageRaw, coverageOK := rawField(fields, "coverage")
	if !deliveryOK || !factory.ValidID(deliveryID) || !digestOK || !factory.ValidDigest(inputDigest) ||
		!isObjectArrayRaw(receiptsRaw, receiptsOK) || !isArrayRaw(coverageRaw, coverageOK) {
		return parkFW40()
	}
	if assigned.Phase != project.ExecutionVerify {
		return fenceFW40(assigned, submission, now, "app verification finish does not match its assignment phase")
	}
	var receipts []factory.VerificationCheckReceipt
	if err := strictDecodeField(receiptsRaw, &receipts); err != nil {
		return fenceFW40(assigned, submission, now, "verification check receipts refused: "+err.Error())
	}
	var coverage []factory.ReassessmentCoverage
	if err := strictDecodeField(coverageRaw, &coverage); err != nil {
		return fenceFW40(assigned, submission, now, "verification coverage refused: "+err.Error())
	}
	report := factory.AppVerificationReport{
		DeliveryID: deliveryID, InputDigest: inputDigest, Outcome: outcome,
		Coverage: coverage, CheckReceipts: receipts,
	}
	if raw, present := rawField(fields, "failures"); present {
		if err := strictDecodeField(raw, &report.Failures); err != nil {
			return fenceFW40(assigned, submission, now, "verification failures refused: "+err.Error())
		}
	}
	if raw, present := rawField(fields, "artifact_refs"); present {
		if err := strictDecodeField(raw, &report.ArtifactRefs); err != nil {
			return fenceFW40(assigned, submission, now, "verification artifact refs refused: "+err.Error())
		}
	}
	if raw, present := rawField(fields, "evidence_refs"); present {
		if err := strictDecodeField(raw, &report.EvidenceRefs); err != nil {
			return fenceFW40(assigned, submission, now, "verification evidence refs refused: "+err.Error())
		}
	}
	parsed := phaseResultFor(submission, project.ExecutionVerify, "worker reports app verification",
		&factory.PhaseOutput{Verification: &report})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "app verification settled", true, nil
}

// settleTypedImplementation settles one Builder finish on an implement
// assignment as a typed implementation receipt, so parents advance
// through the same consumers as harness-reported results. Assignments
// whose phase does not bind fence instead of settling.
func settleTypedImplementation(assigned factory.Assignment, submission factory.MCPSubmission, candidate string, now time.Time) factory.Assignment {
	parsed := phaseResultFor(submission, project.ExecutionImplement, "worker nominates "+candidate,
		&factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: candidate}})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now)
}
