package api

import (
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
)

// rolePayloadSchema documents each existing result carrier at discovery. The
// structured schemas come from the same Go DTOs the settler decodes. Architect
// plans require their complete typed shape before freezing a terminal result.
func rolePayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	spec, ok := mcpMutationPayloads[tool]
	if !ok {
		return nil
	}
	props := map[string]*jsonschema.Schema{}
	for _, key := range spec.required {
		props[key] = &jsonschema.Schema{Description: "Required result field."}
	}
	for _, key := range []string{"component_id", "unchanged", "candidate", "questions", "conclusion", "evidence", "evidence_refs", "uncertainty", "verification_manifest_proposal", "walkthroughs", "alternatives", "unresolved_facts", "tuple", "carried_assessments", "findings", "cause_detail", "native_child_ref", "check_receipts", "artifact_refs", "failures", "input_digest", "fix_evidence"} {
		if mcpOptionalPayloadKey(tool, key) {
			props[key] = &jsonschema.Schema{}
		}
	}
	set := func(key, description string, shape *jsonschema.Schema) {
		if _, ok := props[key]; !ok {
			return
		}
		props[key] = &jsonschema.Schema{Description: description}
		if shape != nil {
			props[key].AnyOf = []*jsonschema.Schema{shape.CloneSchemas(), {}}
		}
	}
	text := &jsonschema.Schema{Type: "string"}
	strings := &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Type: "string"}}
	set("component_id", "Optional feed presentation ID allowed for this role and outlet; omit for its default presentation.", text)
	set("evidence_refs", "References to evidence actually inspected; never a claim that checks passed.", strings)
	set("candidate", "Exact candidate commit hash produced by this work. Publication and merge are independently verified.", text)
	set("input_digest", "The admitted source/input digest supplied in assignment context.", text)
	switch tool {
	case factory.MCPToolFinishIncomplete:
		set("cause", "Short cause code or explanation for ending without a complete result.", text)
		set("cause_detail", "Optional supporting explanation.", text)
		set("affected_task_ids", "IDs of tasks left unfinished.", strings)
		set("requested_resolution", "What needs to change before this work can proceed.", text)
	case factory.MCPToolFinishProductPlan:
		set("candidates", "One to three structured semantic graph candidates. Each contains key, rationale and full task metadata. String candidate names alone cannot settle a product plan. "+control.FinishProductPlanArgsGuidance, mcpCarrier[[]factory.SemanticCandidate]())
		set("coverage", "Explain how the candidates cover the accepted requested outcomes.", strings)
		set("questions", "Any unresolved product questions; required unresolved questions prevent executable adoption.", strings)
		set("verification_manifest_proposal", "Optional structured proposal using approved check catalog references; separately validated and adopted. "+factory.ApplicationBundleGuidance, mcpCarrier[factory.VerificationManifestProposal]())
		props["walkthroughs"] = mcpCarrier[factory.ProposalContent]()
		props["walkthroughs"].Description = "Required for newly staged intent plans: author two concrete product walkthroughs using producer soda-planning-agent-v1, the exact admitted Content facts/identity/exclusions, and every resolved-scope ID from Head in each alternative's idea_ids (the explicit nonempty EffectiveOutcomes verbatim, else the accept/edit idea IDs in revision order; pending/declined never resolve). JEV compares and checks this authored prose separately. Follow the staged product walkthrough instructions. " + control.FinishProductPlanArgsGuidance
	case factory.MCPToolFinishIssuePlan:
		set("approach", "Approach chosen after source investigation; must match tasks.approach.", text)
		props["tasks"] = investigationPlanSchema()
		set("alternatives", "Rejected approaches and why they were rejected.", mcpCarrier[[]factory.RejectedAlternative]())
		set("unresolved_facts", "Unresolved facts; required unresolved facts prevent executable adoption.", strings)
	case factory.MCPToolFinishImplement:
		set("unchanged", "Use true only when the assigned work is completed without producing a changed candidate; otherwise supply candidate.", &jsonschema.Schema{Type: "boolean"})
	case factory.MCPToolFinishReview:
		set("coverage", "For a clear review, the exact SHA-256 coverage digest; findings require the recorded inspected coverage.", text)
		set("tuple", "Exact review cycle and candidate/base binding. Required to settle clear/findings.", &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"cycle_id": {Type: "string"}, "head": {Type: "string"}, "base_oid": {Type: "string"}}, Required: []string{"cycle_id", "head", "base_oid"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}})
		set("findings", "Structured findings with actionable outcomes and evidence. A string summary cannot replace these records.", mcpCarrier[[]factory.Finding]())
		set("carried_assessments", "Disposition/evidence for each carried finding in this review cycle.", mcpCarrier[[]factory.FindingAssessment]())
	case factory.MCPToolFinishCorrection:
		set("finding_digest", "Digest of the complete canonical finding set being answered.", text)
		set("dispositions", "A structured disposition for each input finding; do not silently omit unresolved findings.", mcpCarrier[[]factory.CorrectionDisposition]())
		set("fix_evidence", "Relevant changed paths and exact commits for each fixed code finding. Verification-only findings use passing native check evidence instead. These are nominations, independently verified before publication.", mcpCarrier[[]factory.CorrectionFixEvidence]())
	case factory.MCPToolFinishReassessment:
		set("subject", "The admitted reassessment/repair subject from context.", text)
		outcome := &jsonschema.Schema{Type: "string", Enum: []any{factory.RepairOutcomeRepair, factory.RepairOutcomeVerifiedWait, factory.RepairOutcomeNoChange, factory.RepairOutcomeIncomplete}, Description: "repair proposes authorized graph changes; verified_wait records an evidenced wait; no_change completes diagnosis without graph changes; incomplete names missing facts or authority. Unresolved stalls remain held for owner attention."}
		stall := mcpCarrier[factory.StallRepairReport]()
		stall.Properties["outcome"] = outcome.CloneSchemas()
		report := &jsonschema.Schema{AnyOf: []*jsonschema.Schema{mcpCarrier[factory.ReassessmentReport](), stall, mcpCarrier[factory.DeliveryRepairReport]()}}
		set("report", "Full typed ReassessmentReport, StallRepairReport or DeliveryRepairReport, matching the assigned phase/subject. This proposes a change; it does not apply it.", report)
		// Enforce the stall outcome even through the existing carrier fallback:
		// an invalid value must be corrected before an immutable terminal is
		// recorded. Other reassessment/repair subjects keep their own contract.
		props["report"].If = &jsonschema.Schema{Type: "object", Required: []string{"stall_id"}}
		props["report"].Then = &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"outcome": outcome}, Required: []string{"outcome"}}
	case factory.MCPToolFinishAppVerify:
		set("delivery_id", "Exact assigned delivery identity.", text)
		set("evidence_refs", "Inspected evidence; journey:<submission ID> must resolve to this assignment's exact recorded observation. Verified claims may cite only passed journeys. These references never replace trusted check receipts.", strings)
		set("coverage", "Structured outcome coverage for the assigned delivery.", mcpCarrier[[]factory.ReassessmentCoverage]())
		set("check_receipts", "Observed typed receipts for approved checks; required to settle verified or failed.", mcpCarrier[[]factory.VerificationCheckReceipt]())
		set("artifact_refs", "Content digests from the publish response of this assignment's complete artifacts. Settlement resolves each digest back to the assignment and fences unknown, foreign or unfinished citations.", strings)
		set("failures", "Concrete failed checks with detail.", mcpCarrier[[]factory.CheckFailure]())
	case factory.MCPToolSubmitTaskOutput:
		set("task_id", "Task ID in the admitted execution manifest.", text)
		set("manifest_digest", "Digest of the exact admitted task manifest.", text)
		set("output_digest", "Digest of the observed task output.", text)
		set("native_child_ref", "Builder: observed child identity. Fixer: omit because correction runs in the bound parent.", text)
	case factory.MCPToolFinishResearch:
		set("questions", "The declared research questions this report answers.", strings)
		set("conclusion", "The useful outcome of the inquiry.", text)
		set("findings", "Answerable findings with their source refs.", strings)
		set("evidence", "Source refs actually inspected.", strings)
		set("uncertainty", "The open remainder the inquiry could not resolve.", text)
	case factory.MCPToolRequestHandoff:
		set("predecessor_task", "Approved predecessor task ID.", text)
		set("successor_task", "Approved dependent task ID; this must already be an edge in the admitted plan.", text)
		set("manifest_digest", "Digest of the exact admitted task manifest.", text)
		set("output_submission_id", "Recorded predecessor output submission; release still requires trusted checkpoint observation.", text)
	}
	for field, values := range spec.enums {
		enums := make([]any, len(values))
		for i, value := range values {
			enums[i] = value
		}
		props[field] = &jsonschema.Schema{Type: "string", Enum: enums, Description: "Recorded result outcome; downstream effects require their independent validation."}
	}
	return &jsonschema.Schema{Type: "object", Properties: props, Required: spec.required, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

func mcpCarrier[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic("Factory result schema cannot be described: " + err.Error())
	}
	return schema
}
