package factory

// D14 canonical profile/tool DTOs (FW38).
//
// One Factory MCP implementation serves seven profiles over fourteen
// tools. This file freezes the canonical names, the execution phase
// each profile serves, and which profile may call which tool. Admission
// (control), persistence (store) and transport (web/api) enforce this
// matrix; nothing else invents a profile or tool name. The P6 fixture
// catalog (web/api factory_mcp_p6.go) proves transport only; this is
// the workflow authority FW40/FW41/FW42/FW44/FW45 consume.

import (
	"errors"
)

// MCPProfile is one admitted Factory agent profile. Profile names are
// workflow roles, never native account identities: native
// coder/reviewer accounts remain the two execution identities (D14).
type MCPProfile string

const (
	MCPProfileProductLead MCPProfile = "product_lead"
	MCPProfileArchitect   MCPProfile = "architect"
	MCPProfileBuilder     MCPProfile = "builder"
	MCPProfileInspector   MCPProfile = "inspector"
	MCPProfileFixer       MCPProfile = "fixer"
	MCPProfileStrategist  MCPProfile = "strategist"
	MCPProfileTestPilot   MCPProfile = "test_pilot"
)

// MCPProfiles lists the seven admitted profiles in contract order.
var MCPProfiles = []MCPProfile{
	MCPProfileProductLead,
	MCPProfileArchitect,
	MCPProfileBuilder,
	MCPProfileInspector,
	MCPProfileFixer,
	MCPProfileStrategist,
	MCPProfileTestPilot,
}

// MCPPhaseOf returns the execution phase a profile serves. Strategist
// serves reassess or repair depending on its admitted subject; both
// phases share the profile.
func MCPPhaseOf(profile MCPProfile) string {
	switch profile {
	case MCPProfileProductLead:
		return "decompose"
	case MCPProfileArchitect:
		return "investigate"
	case MCPProfileBuilder:
		return "implement"
	case MCPProfileInspector:
		return "review"
	case MCPProfileFixer:
		return "correct"
	case MCPProfileStrategist:
		return "reassess"
	case MCPProfileTestPilot:
		return "verify"
	default:
		return ""
	}
}

// MCPTool is one tool in the single fourteen-tool catalog.
type MCPTool string

const (
	MCPToolContext            MCPTool = "factory.context"
	MCPToolEvidence           MCPTool = "factory.evidence"
	MCPToolStatus             MCPTool = "factory.status"
	MCPToolReceipt            MCPTool = "factory.receipt"
	MCPToolFinishIncomplete   MCPTool = "factory.finish_incomplete"
	MCPToolFinishProductPlan  MCPTool = "factory.finish_product_plan"
	MCPToolFinishIssuePlan    MCPTool = "factory.finish_issue_plan"
	MCPToolFinishImplement    MCPTool = "factory.finish_implementation"
	MCPToolFinishReview       MCPTool = "factory.finish_review"
	MCPToolFinishCorrection   MCPTool = "factory.finish_correction"
	MCPToolFinishReassessment MCPTool = "factory.finish_reassessment"
	MCPToolFinishAppVerify    MCPTool = "factory.finish_app_verification"
	MCPToolSubmitTaskOutput   MCPTool = "factory.submit_task_output"
	MCPToolRequestHandoff     MCPTool = "factory.request_handoff"
)

// MCPTools lists the fourteen catalog tools in contract order.
var MCPTools = []MCPTool{
	MCPToolContext,
	MCPToolEvidence,
	MCPToolStatus,
	MCPToolReceipt,
	MCPToolFinishIncomplete,
	MCPToolFinishProductPlan,
	MCPToolFinishIssuePlan,
	MCPToolFinishImplement,
	MCPToolFinishReview,
	MCPToolFinishCorrection,
	MCPToolFinishReassessment,
	MCPToolFinishAppVerify,
	MCPToolSubmitTaskOutput,
	MCPToolRequestHandoff,
}

// D14 byte/count bounds. Tool schemas reject unknown fields and
// enforce these limits; transport never silently truncates.
const (
	MCPMaxRequestEnvelopeBytes = 96 * 1024
	MCPMaxResultPayloadBytes   = 64 * 1024
	MCPMaxPageBytes            = 32 * 1024
	MCPMaxPageRefs             = 50
	MCPMaxHandoffs             = 256
	MCPMaxConcurrentRequests   = 4
	MCPMaxCallsPerMinute       = 60
	MCPMaxInstructionBytes     = 16 * 1024
	MCPMaxConfigBytes          = 16 * 1024
	MCPMaxManifestBytes        = 64 * 1024
	MCPMaxLaunchEnvelopeBytes  = 1024 * 1024
	MCPMaxSubmissionPayload    = 64 * 1024
	MCPMaxReasonChars          = 1024
	MCPMaxRefChars             = 256
)

// mcpReadTools are bound reads: they never launch work and never
// freeze the assignment.
var mcpReadTools = map[MCPTool]bool{
	MCPToolContext:  true,
	MCPToolEvidence: true,
	MCPToolStatus:   true,
	MCPToolReceipt:  true,
}

// MCPIsRead reports whether the tool is a bound read.
func MCPIsRead(tool MCPTool) bool { return mcpReadTools[tool] }

// mcpTerminalTools end the assignment with one recorded proposal.
// Exactly one terminal identity wins per assignment (see
// mcp_submission.go); a second identity with conflicting terminal
// content refuses.
var mcpTerminalTools = map[MCPTool]bool{
	MCPToolFinishIncomplete:   true,
	MCPToolFinishProductPlan:  true,
	MCPToolFinishIssuePlan:    true,
	MCPToolFinishImplement:    true,
	MCPToolFinishReview:       true,
	MCPToolFinishCorrection:   true,
	MCPToolFinishReassessment: true,
	MCPToolFinishAppVerify:    true,
}

// MCPIsTerminal reports whether the tool records a terminal proposal.
func MCPIsTerminal(tool MCPTool) bool { return mcpTerminalTools[tool] }

// mcpToolProfiles is the admitted caller set per tool. All seven
// profiles read; finish_incomplete is the every-parent terminal;
// typed finishes, task output and handoff stay with their role.
var mcpToolProfiles = map[MCPTool]map[MCPProfile]bool{
	MCPToolContext:            allMCPProfiles(),
	MCPToolEvidence:           allMCPProfiles(),
	MCPToolStatus:             allMCPProfiles(),
	MCPToolReceipt:            allMCPProfiles(),
	MCPToolFinishIncomplete:   allMCPProfiles(),
	MCPToolFinishProductPlan:  singleMCPProfile(MCPProfileProductLead),
	MCPToolFinishIssuePlan:    singleMCPProfile(MCPProfileArchitect),
	MCPToolFinishImplement:    singleMCPProfile(MCPProfileBuilder),
	MCPToolFinishReview:       singleMCPProfile(MCPProfileInspector),
	MCPToolFinishCorrection:   singleMCPProfile(MCPProfileFixer),
	MCPToolFinishReassessment: singleMCPProfile(MCPProfileStrategist),
	MCPToolFinishAppVerify:    singleMCPProfile(MCPProfileTestPilot),
	MCPToolSubmitTaskOutput: {
		MCPProfileBuilder: true,
		MCPProfileFixer:   true,
	},
	MCPToolRequestHandoff: singleMCPProfile(MCPProfileBuilder),
}

func allMCPProfiles() map[MCPProfile]bool {
	admitted := make(map[MCPProfile]bool, len(MCPProfiles))
	for _, profile := range MCPProfiles {
		admitted[profile] = true
	}
	return admitted
}

func singleMCPProfile(profile MCPProfile) map[MCPProfile]bool {
	return map[MCPProfile]bool{profile: true}
}

// MCPToolKnown reports whether the tool belongs to the catalog.
// Unknown tools fail server-side; discovery never advertises them.
func MCPToolKnown(tool MCPTool) bool {
	_, ok := mcpToolProfiles[tool]
	return ok
}

// MCPToolAdmitted reports whether the profile may call the tool.
// A tool absent from discovery for this profile still refuses here:
// callers cannot reach it by direct call.
func MCPToolAdmitted(profile MCPProfile, tool MCPTool) bool {
	admitted, ok := mcpToolProfiles[tool]
	if !ok {
		return false
	}
	return admitted[profile]
}

// MCPDiscovery lists the tools advertised to the profile: the four
// reads plus its admitted mutations. Transport serves exactly this
// set per capability; anything else refuses before admission.
func MCPDiscovery(profile MCPProfile) []MCPTool {
	var tools []MCPTool
	for _, tool := range MCPTools {
		if MCPToolAdmitted(profile, tool) {
			tools = append(tools, tool)
		}
	}
	return tools
}

// ValidateMCPEnvelope bounds one request/response envelope before any
// admission work: oversize bytes refuse without creating a receipt.
func ValidateMCPEnvelope(payload []byte) error {
	if len(payload) == 0 {
		return errors.New("empty MCP envelope")
	}
	if len(payload) > MCPMaxRequestEnvelopeBytes {
		return errors.New("MCP envelope exceeds 96 KiB")
	}
	return nil
}
