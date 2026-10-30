package api

// One role-filtered Factory MCP implementation serves the private listener
// through the Rust stdio relay. Host-staged credentials bind each interface to
// its worker; tool arguments carry content rather than identity. The SDK
// validates schemas, admission rechecks the current capability, and durable
// submissions share a transaction with their feed posts and receipts. Role
// results still pass through their existing workflow consumers.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

// FactoryMCPCapabilitySource loads recorded capabilities. *store.Store
// implements it; tests stub it.
type FactoryMCPCapabilitySource interface {
	GetMCPCapability(ctx context.Context, id string) (factory.MCPCapability, error)
}

// FactoryMCPSubmissionSink persists submissions and recovers
// receipts. *store.Store implements it; tests stub it.
type FactoryMCPSubmissionSink interface {
	control.MCPSubmissionJournal
	RecordMCPSubmission(ctx context.Context, submission factory.MCPSubmission, receipt factory.MCPReceipt) (factory.MCPReceipt, error)
	MCPSubmissionCount(ctx context.Context, assignmentID string) (int, error)
}

// FactoryMCPAssignmentSource loads the recorded assignment behind a
// capability for current-context projection. *store.Store implements
// it; tests stub it.
type FactoryMCPAssignmentSource interface {
	Assignment(ctx context.Context, id string) (factory.Assignment, error)
}

// FactoryMCPPriorityJudgmentSource reads the prepared context choice.
// *store.Store implements it; tests stub it.
type FactoryMCPPriorityJudgmentSource interface {
	PriorityJudgment(context.Context, string) (factory.PriorityReceipt, error)
}

// FactoryMCPRunSource loads recorded run rows behind an assignment
// for bounded evidence projection. *store.Store implements it;
// tests stub it.
type FactoryMCPRunSource interface {
	FactoryRun(ctx context.Context, id string) (factory.Run, error)
}

// FactoryMCPDeps wires the catalog server. Observed supplies the
// host-observed run state from outside SQL; nil keeps the recorded
// binding only (FW39 plugs host observation here). Catalog backs
// D16 proposal validation from the authenticated capability's approved
// preparation; nil keeps the catalog unavailable.
type FactoryMCPDeps struct {
	IssueMaker       func(context.Context, factory.MCPCapability, factory.MCPTool, map[string]any, string) (map[string]any, error)
	ContextCursorKey []byte
	// BoundCapability is supplied by the authenticated transport, never tool
	// arguments. Its immutable scope supplies identity and default revisions.
	BoundCapability      *factory.MCPCapability
	Limiter              *mcpRateLimiter
	Prepare              func(context.Context, factory.MCPCapability) (factory.MCPCapability, error)
	Capabilities         FactoryMCPCapabilitySource
	Submissions          FactoryMCPSubmissionSink
	Assignments          FactoryMCPAssignmentSource
	PriorityJudgments    FactoryMCPPriorityJudgmentSource
	Runs                 FactoryMCPRunSource
	Observed             func(capability factory.MCPCapability) (control.MCPRunObservation, error)
	Catalog              func(context.Context, factory.MCPCapability) (factory.ApprovedCatalog, error)
	Now                  func() time.Time
	SystemOne            FactorySystemOne
	VerificationEvidence func(context.Context, factory.MCPCapability) (map[string]any, error)
	RepairPreview        func(context.Context, factory.MCPCapability, json.RawMessage) (factory.RepairPreview, error)
	NativeTask           func(context.Context, factory.MCPCapability, string, string) (map[string]any, error)
}

// mcpCallArgs is the tool envelope. The bound server supplies capability
// identity and scope; the worker sends only its payload and, optionally, a
// submission_id to identify the submission. CapabilityID remains an internal
// admission field used by the unbound transport fixtures.
type mcpCallArgs struct {
	CapabilityID string         `json:"capability_id" jsonschema:"recorded capability identity"`
	SubmissionID string         `json:"submission_id,omitempty" jsonschema:"optional submission identity"`
	Payload      map[string]any `json:"payload,omitempty" jsonschema:"tool payload within 64 KiB"`
}

// mcpEnvelopeSchema advertises and enforces the envelope per tool.
// additionalProperties false refuses unknown top-level fields
// server-side through SDK schema validation, and required lists
// differ for reads and mutations.
func mcpEnvelopeSchema(tool factory.MCPTool) *mcp.Tool {
	required := []string{"capability_id"}
	if !factory.MCPIsRead(tool) {
		required = []string{"capability_id", "payload"}
	}
	definition := &mcp.Tool{
		Name:        string(tool),
		Description: factoryToolDescription(tool),
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"capability_id": {Type: "string"},
				"submission_id": {Type: "string"},
				"payload":       {Type: "object"},
			},
			Required:             required,
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		},
	}
	if payload := communicationPayloadSchema(tool); payload != nil {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = payload
	}
	if payload := inboxPayloadSchema(tool); payload != nil {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = payload
		if tool == factory.MCPToolDiscussion {
			definition.InputSchema.(*jsonschema.Schema).Required = []string{"capability_id", "payload"}
		}
	}
	if payload := rolePayloadSchema(tool); payload != nil {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = payload
	}
	if payload := libraryPayloadSchema(tool); payload != nil {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = payload
		if tool == factory.MCPToolReadLibraryItem {
			definition.InputSchema.(*jsonschema.Schema).Required = []string{"capability_id", "payload"}
		}
	}
	if payload := verificationJourneyPayloadSchema(tool); payload != nil {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = payload
	}
	if tool == factory.MCPToolPreviewRepair {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = repairPreviewPayloadSchema()
		definition.InputSchema.(*jsonschema.Schema).Required = []string{"capability_id", "payload"}
	}
	if tool == factory.MCPToolRecordDiagnosis {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = repairDiagnosisPayloadSchema()
	}
	if tool == factory.MCPToolDraftRepair {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = repairDraftPayloadSchema()
	}
	if tool == factory.MCPToolReassessmentScope {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = repairScopePayloadSchema()
	}
	if payload := resolvePostPayloadSchema(tool); payload != nil {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = payload
	}
	if tool == factory.MCPToolContext {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = contextPayloadSchema()
	}
	if tool == factory.MCPToolPublishArtifact {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = artifactPayloadSchema()
	}
	if isNativeTaskTool(tool) {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = &jsonschema.Schema{
			Type: "object", Properties: map[string]*jsonschema.Schema{
				"task_id": {Type: "string", Pattern: "^[a-z][a-z0-9_-]{0,31}$"},
			}, Required: []string{"task_id"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		}
		definition.InputSchema.(*jsonschema.Schema).Required = []string{"capability_id", "payload"}
	}
	if isIssueMakerTool(tool) {
		definition.InputSchema.(*jsonschema.Schema).Properties["payload"] = issueMakerPayloadSchema()
	}
	return definition
}

// mcpRateLimiter bounds traffic per run: 4 concurrent requests and
// 60 admitted calls per minute. Limits never extend deadlines;
// refusals are bounded protocol errors creating no receipt.
type mcpRateLimiter struct {
	mu     sync.Mutex
	active map[string]int
	window map[string]time.Time
	calls  map[string]int
	now    func() time.Time
}

func newMCPRateLimiter() *mcpRateLimiter {
	return &mcpRateLimiter{
		active: map[string]int{}, window: map[string]time.Time{},
		calls: map[string]int{}, now: time.Now,
	}
}

func (l *mcpRateLimiter) admit(runID string) (release func(), err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if start, ok := l.window[runID]; !ok || now.Sub(start) >= time.Minute {
		l.window[runID] = now
		l.calls[runID] = 0
	}
	if l.active[runID] >= factory.MCPMaxConcurrentRequests {
		return nil, errors.New("too many concurrent factory requests")
	}
	if l.calls[runID] >= factory.MCPMaxCallsPerMinute {
		return nil, errors.New("factory call rate exceeded")
	}
	l.calls[runID]++
	l.active[runID]++
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.active[runID]--
	}, nil
}

// factoryCatalogServer holds per-server admission dependencies.
type factoryCatalogServer struct {
	deps    FactoryMCPDeps
	limiter *mcpRateLimiter
}

func (s *factoryCatalogServer) now() time.Time {
	if s.deps.Now != nil {
		return s.deps.Now()
	}
	return time.Now()
}

func (s *factoryCatalogServer) observed(capability factory.MCPCapability) (control.MCPRunObservation, error) {
	if s.deps.Observed != nil {
		return s.deps.Observed(capability)
	}
	return control.MCPRunObservation{}, nil
}

// admitCall runs the shared gate: capability load, profile/tool
// binding, freshness/authority and the rate limit. It returns the
// admitted capability and the rate release the caller defers.
func (s *factoryCatalogServer) admitCall(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (factory.MCPCapability, func(), error) {
	if bound := s.deps.BoundCapability; bound != nil {
		if args.CapabilityID != "" && args.CapabilityID != bound.ID {
			return factory.MCPCapability{}, nil, errors.New("tool identity differs from this worker's context: omit capability_id, or pass the capability_id from factory.context")
		}
		args.CapabilityID = bound.ID
	}
	if !factory.ValidID(args.CapabilityID) {
		return factory.MCPCapability{}, nil, errors.New("factory capability refused: capability_id must be the 32-hex capability_id from factory.context")
	}
	capability, err := s.deps.Capabilities.GetMCPCapability(ctx, args.CapabilityID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return factory.MCPCapability{}, nil, errors.New("factory capability refused: no recorded capability has this capability_id; use the one from factory.context")
		}
		return factory.MCPCapability{}, nil, errors.New("factory capability unavailable: the capability store is unreachable; retry with the same capability_id")
	}
	if bound := s.deps.BoundCapability; bound != nil && capability.ScopeDigest() != bound.ScopeDigest() {
		return factory.MCPCapability{}, nil, errors.New("worker context changed; reconnect to the assigned server")
	}
	observed, err := s.observed(capability)
	if err != nil {
		return factory.MCPCapability{}, nil, errors.New("factory run observation unavailable")
	}
	if err := control.AdmitMCPCapability(capability, tool, s.now(), observed); err != nil {
		return factory.MCPCapability{}, nil, err
	}
	release, err := s.limiter.admit(capability.RunID)
	if err != nil {
		return factory.MCPCapability{}, nil, err
	}
	return capability, release, nil
}

// canonicalPayload renders deterministic canonical bytes for the
// payload map and enforces the 64 KiB result bound.
func canonicalPayload(payload map[string]any) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("missing tool payload: payload must be an object with this tool's fields")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("tool payload is not JSON: payload must be a JSON object")
	}
	if len(raw) > factory.MCPMaxSubmissionPayload {
		return nil, errors.New("tool payload exceeds 64 KiB")
	}
	return raw, nil
}

// mcpPayloadSpec is the one key definition per tool payload: required
// keys, documented optional keys and enum-constrained fields. The
// advertised schema is built from it and admission enforces it; the
// typed carriers inside are decoded by the settler (CheckMCPTerminalFinish
// runs that decode before anything is recorded). Unknown keys refuse.
type mcpPayloadSpec struct {
	required []string
	optional []string
	enums    map[string][]string
}

var mcpMutationPayloads = map[factory.MCPTool]mcpPayloadSpec{
	factory.MCPToolFinishIncomplete: {
		required: []string{"outcome", "cause"},
		optional: []string{"cause_detail", "evidence_refs"},
		enums:    map[string][]string{"outcome": {"blocked", "failed", "cancelled"}},
	},
	factory.MCPToolFinishProductPlan: {
		required: []string{"candidates", "evidence_refs", "coverage"},
		optional: []string{"questions", "verification_manifest_proposal", "walkthroughs"},
	},
	factory.MCPToolFinishIssuePlan: {
		required: []string{"evidence_refs", "approach", "tasks"},
	},
	factory.MCPToolFinishImplement: {
		optional: []string{"candidate"},
	},
	factory.MCPToolFinishReview: {
		required: []string{"conclusion", "evidence_refs"},
		optional: []string{"tuple", "coverage", "carried_assessments", "findings"},
		enums:    map[string][]string{"conclusion": {"clear", "findings", "incomplete"}},
	},
	factory.MCPToolFinishCorrection: {
		required: []string{"candidate", "dispositions", "evidence_refs"},
		optional: []string{"finding_digest", "fix_evidence"},
	},
	factory.MCPToolFinishReassessment: {
		required: []string{"subject", "report"},
		optional: []string{"input_digest"},
	},
	factory.MCPToolFinishAppVerify: {
		required: []string{"outcome", "coverage"},
		optional: []string{"delivery_id", "input_digest", "check_receipts", "artifact_refs", "failures", "evidence_refs"},
		enums:    map[string][]string{"outcome": {"verified", "failed", "incomplete"}},
	},
	factory.MCPToolSubmitTaskOutput: {
		required: []string{"task_id", "evidence_refs"},
		optional: []string{"manifest_digest", "output_digest", "native_child_ref"},
	},
}

// payloadKeyKnown reports whether the spec names the key. A retired
// presentation ID from an old worker is accepted and ignored.
func (spec mcpPayloadSpec) payloadKeyKnown(key string) bool {
	if key == "component_id" {
		return true
	}
	for _, list := range [][]string{spec.required, spec.optional} {
		for _, want := range list {
			if key == want {
				return true
			}
		}
	}
	return false
}

func checkPayloadKeys(tool factory.MCPTool, payload map[string]any, spec mcpPayloadSpec) error {
	for key := range payload {
		if !spec.payloadKeyKnown(key) {
			return fmt.Errorf("unknown payload field %q: %s takes %s", key, tool, strings.Join(append(append([]string(nil), spec.required...), spec.optional...), ", "))
		}
	}
	return nil
}

func checkPayloadRequired(payload map[string]any, spec mcpPayloadSpec) error {
	for _, want := range spec.required {
		value, ok := payload[want]
		if !ok {
			return fmt.Errorf("missing payload field %q: this tool requires %s", want, strings.Join(spec.required, ", "))
		}
		if text, ok := value.(string); ok && (text == "" || len(text) > factory.MCPMaxReasonChars) {
			return fmt.Errorf("payload field %q out of bound: it must be a non-empty string of at most %d characters", want, factory.MCPMaxReasonChars)
		}
	}
	return nil
}

func checkPayloadEnums(payload map[string]any, spec mcpPayloadSpec) error {
	for field, allowed := range spec.enums {
		text, _ := payload[field].(string)
		admitted := false
		for _, want := range allowed {
			if text == want {
				admitted = true
			}
		}
		if !admitted {
			return fmt.Errorf("payload field %q carries an unsupported value %.64q: use one of %s", field, text, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// checkPayload enforces the strict per-tool key shape.
func (s *factoryCatalogServer) checkPayload(tool factory.MCPTool, payload map[string]any) error {
	if isIssueMakerTool(tool) {
		return checkIssueMakerPayload(payload)
	}
	if tool == factory.MCPToolRecordJourneyResult {
		return checkVerificationJourneyPayload(payload)
	}
	if tool == factory.MCPToolRecordDiagnosis {
		return checkRepairDiagnosisPayload(payload)
	}
	if tool == factory.MCPToolDraftRepair {
		return checkRepairDraftPayload(payload)
	}
	if tool == factory.MCPToolResolvePost {
		return checkResolvePostPayload(tool, payload)
	}
	if tool == factory.MCPToolPublishArtifact {
		return checkArtifactPayload(payload)
	}
	if factory.MCPIsCommunication(tool) {
		return checkCommunicationPayload(tool, payload)
	}
	spec, ok := mcpMutationPayloads[tool]
	if !ok {
		return fmt.Errorf("unsupported factory tool %q: call one of the tools this server lists", tool)
	}
	if err := checkPayloadKeys(tool, payload, spec); err != nil {
		return err
	}
	if err := checkPayloadRequired(payload, spec); err != nil {
		return err
	}
	return checkPayloadEnums(payload, spec)
}

// checkManifestProposal validates a product plan's
// verification_manifest_proposal against the approved check catalog:
// catalog references, parameter bounds and artifact/fixture requirements.
// Outcome coverage against the planning scope is checked by settlement.
func (s *factoryCatalogServer) checkManifestProposal(ctx context.Context, cap factory.MCPCapability, payload map[string]any) error {
	raw, present := payload["verification_manifest_proposal"]
	if !present {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return errors.New("manifest proposal is not JSON")
	}
	if len(encoded) > factory.MCPMaxManifestBytes {
		return errors.New("manifest proposal exceeds 64 KiB")
	}
	proposal, err := factory.ParseVerificationManifestProposal(encoded)
	if err != nil {
		return errors.New("manifest proposal shape refused")
	}
	if s.deps.Catalog == nil {
		return errors.New("no approved check catalog")
	}
	catalog, err := s.deps.Catalog(ctx, cap)
	if err != nil {
		return err
	}
	return factory.ValidateManifestProposal(proposal, catalog)
}

// admitMutation runs the shared mutation gate after admitCall:
// strict payload shape, digest binding and control submission
// admission (terminal freeze, handoff cap).
func (s *factoryCatalogServer) admitMutation(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (factory.MCPCapability, factory.MCPSubmission, func(), error) {
	capability, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return factory.MCPCapability{}, factory.MCPSubmission{}, nil, err
	}
	s.bindPayload(ctx, capability, tool, args.Payload)
	raw, err := canonicalPayload(args.Payload)
	if err != nil {
		release()
		return factory.MCPCapability{}, factory.MCPSubmission{}, nil, err
	}
	if err := s.checkPayload(tool, args.Payload); err != nil {
		release()
		return factory.MCPCapability{}, factory.MCPSubmission{}, nil, err
	}
	submission := factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: args.SubmissionID,
		CapabilityID: capability.ID,
		Tool:         tool,
		PayloadHash:  factory.MCPInputDigest(raw), Payload: raw,
	}
	replay, err := control.AdmitMCPSubmission(ctx, s.deps.Submissions, capability, submission)
	if err == nil && !replay {
		err = s.checkNewFinish(ctx, capability, submission, args.Payload)
		if err == nil && tool != factory.MCPToolFinishIncomplete && !factory.MCPIsCommunication(tool) && tool != factory.MCPToolResolvePost {
			err = s.requireCurrentWork(ctx, capability)
		}
	}
	if err != nil {
		release()
		return factory.MCPCapability{}, factory.MCPSubmission{}, nil, err
	}
	return capability, submission, release, nil
}

func mcpTextResult(text string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
}

func mcpJSONResult(value any) (*mcp.CallToolResult, any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > factory.MCPMaxResultPayloadBytes {
		return nil, nil, errors.New("factory result exceeds 64 KiB")
	}
	return mcpTextResult(string(raw))
}

// newFactoryCatalogServer builds the shared, role-filtered catalog. Reads
// serve capability-bound state; mutations record immutable proposals
// with durable receipts; receipt recovers lost responses.
func newFactoryCatalogServer(deps FactoryMCPDeps) *mcp.Server {
	limiter := deps.Limiter
	if limiter == nil {
		limiter = newMCPRateLimiter()
	}
	catalog := &factoryCatalogServer{deps: deps, limiter: limiter}
	name := "soda-factory"
	if deps.BoundCapability != nil {
		name += "-" + string(deps.BoundCapability.Profile)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: name}, nil)
	for _, tool := range factory.MCPTools {
		tool := tool
		if deps.BoundCapability != nil && !factory.MCPToolAdmitted(deps.BoundCapability.Profile, tool) {
			continue
		}
		if deps.BoundCapability != nil && isNativeTaskTool(tool) && deps.NativeTask == nil {
			continue
		}
		if isSystemOneTool(tool) {
			catalog.addSystemOneTool(server, tool)
			continue
		}
		schema := mcpEnvelopeSchema(tool)
		if deps.BoundCapability != nil {
			bindMCPToolSchema(schema, tool, *deps.BoundCapability)
		}
		mcp.AddTool(server, schema, func(ctx context.Context, _ *mcp.CallToolRequest, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
			return catalog.serve(ctx, tool, args)
		})
	}
	return server
}

func (s *factoryCatalogServer) serve(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	args.SubmissionID = mcpSubmissionID(args.SubmissionID, mcpTakesSubmission(tool))
	if factory.MCPIsLibraryRead(tool) {
		return s.serveLibraryRead(ctx, tool, args)
	}
	switch tool {
	case factory.MCPToolCreateIssue, factory.MCPToolRefineIssue:
		return s.serveIssueMaker(ctx, tool, args)
	case factory.MCPToolStartTask, factory.MCPToolReleaseOutput:
		return s.serveNativeTask(ctx, tool, args)
	case factory.MCPToolPublishArtifact:
		return s.serveArtifact(ctx, args)
	case factory.MCPToolVerificationEvidence:
		cap, release, err := s.admitCall(ctx, tool, args)
		if err != nil {
			return nil, nil, err
		}
		defer release()
		if s.deps.VerificationEvidence == nil {
			return nil, nil, errors.New("native verification evidence unavailable")
		}
		evidence, err := s.deps.VerificationEvidence(ctx, cap)
		if err != nil {
			return nil, nil, err
		}
		return mcpJSONResult(evidence)
	case factory.MCPToolContext, factory.MCPToolStatus:
		return s.serveRead(ctx, tool, args)
	case factory.MCPToolVerificationScope:
		return s.serveVerificationScope(ctx, args)
	case factory.MCPToolPreviewRepair:
		return s.serveRepairPreview(ctx, args)
	case factory.MCPToolReassessmentScope:
		return s.serveReassessmentScope(ctx, args)
	case factory.MCPToolEvidence:
		return s.serveEvidence(ctx, args)
	case factory.MCPToolInbox, factory.MCPToolDiscussion:
		return s.serveInbox(ctx, tool, args)
	default:
		return s.serveMutation(ctx, tool, args)
	}
}

// serveRead serves capability-bound reads: the immutable assignment
// summary (context) and the live capability plus recorded receipt
// state (status). Reading never launches work and records no row.
func (s *factoryCatalogServer) serveRead(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	read, err := s.loadReadContext(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer read.release()
	attemptEligible, lineageEligible := factory.BundleEligibility(read.assignment, read.parentLoaded)
	foregrounded, selectionReason, selectionFingerprint := s.foregroundReadBundle(ctx, read, attemptEligible, lineageEligible)
	body := buildReadBody(read, attemptEligible, lineageEligible, foregrounded)
	if read.capability.Profile == factory.MCPProfileProductLead {
		catalogView := map[string]any{"available": false, "reason": "approved check catalog is unavailable"}
		if s.deps.Catalog != nil {
			if catalog, err := s.deps.Catalog(ctx, read.capability); err == nil {
				if check, ok := catalog.LookupCheck(factory.ApprovedAppCheckID); ok {
					catalogView = map[string]any{"available": true, "entries": []factory.ApprovedCheck{check}, "binding": catalog}
				}
			}
		}
		body["verification_catalog"] = catalogView
	}
	action := "serve"
	if selectionReason != "" {
		action = "hold"
	}
	if read.frozen {
		action = "stop"
	}
	body["context_selection"] = map[string]any{"bundle": foregrounded, "action": action, "reason_code": selectionReason, "fingerprint": selectionFingerprint}
	if err := s.addCheckpointContext(ctx, read, body); err != nil {
		return nil, nil, err
	}
	if err := s.addCurrentContext(ctx, read, body, args); err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(body)
}

// readContext binds one admitted read to its terminal, count and
// assignment lineage.
type readContext struct {
	capability   factory.MCPCapability
	release      func()
	terminal     factory.MCPReceipt
	frozen       bool
	count        int
	assignment   factory.Assignment
	parent       factory.Assignment
	parentLoaded bool
}

// loadReadContext admits one read and loads its terminal, submission
// count and assignment lineage. A missing assignment refuses instead
// of serving context without its assignment.
func (s *factoryCatalogServer) loadReadContext(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*readContext, error) {
	capability, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return nil, err
	}
	loaded := false
	defer func() {
		if !loaded {
			release()
		}
	}()
	read := &readContext{capability: capability, release: release}
	if read.terminal, read.frozen, err = s.deps.Submissions.TerminalMCPProposal(ctx, capability.AssignmentID); err != nil {
		return nil, err
	}
	if read.count, err = s.deps.Submissions.MCPSubmissionCount(ctx, capability.AssignmentID); err != nil {
		return nil, err
	}
	// Current-context projection reads the recorded assignment behind
	// the capability: prompt digest, exact source/preparation and, once
	// set, outcome/reason. Prompt bytes and actor identity never enter
	// the projection. Own-attempt run identities join the attempt
	// bundle and the linked parent outcome joins the lineage bundle;
	// both stay within the admitted local lineage.
	if read.assignment, err = s.deps.Assignments.Assignment(ctx, capability.AssignmentID); err != nil {
		return nil, err
	}
	if read.assignment.ParentAssignment != "" {
		if loaded, err := s.deps.Assignments.Assignment(ctx, read.assignment.ParentAssignment); err == nil && factory.ContextParentVisible(read.assignment, loaded) {
			read.parent, read.parentLoaded = loaded, true
		}
	}
	loaded = true
	return read, nil
}

// foregroundReadBundle is read-only. The classifier's recorded choice picks
// the foregrounded history; a missing, failed or unsure choice serves current
// context rather than holding the worker.
func (s *factoryCatalogServer) foregroundReadBundle(ctx context.Context, read *readContext, attemptEligible, lineageEligible bool) (string, string, string) {
	if resolution := read.capability.ContextCurrentResolution; resolution != nil {
		source, ok := s.deps.Submissions.(factoryMCPCurrentContextSource)
		if !ok {
			return factory.ContextBundleSnapshotHold, "required_current_unavailable", read.capability.ContextSelectionSnapshot
		}
		current, err := source.FactoryCurrentContext(ctx, read.capability, s.now())
		if err != nil || current.Evidence.Availability != "complete" {
			return factory.ContextBundleSnapshotHold, "required_current_unavailable", read.capability.ContextSelectionSnapshot
		}
		digest, err := factory.CurrentRequiredSemanticDigest(current.Role)
		if err != nil || resolution.ValidateFor(read.capability) != nil || digest != resolution.RequiredDigest {
			return factory.ContextBundleSnapshotHold, "context_selection_stale", read.capability.ContextSelectionSnapshot
		}
		return factory.ContextBundleCurrent, "", read.capability.ContextSelectionSnapshot
	}
	priority := s.deps.PriorityJudgments
	if priority == nil {
		priority, _ = s.deps.Submissions.(FactoryMCPPriorityJudgmentSource)
	}
	fingerprint := read.capability.ContextSelectionSnapshot
	if priority == nil || !factory.ValidDigest(fingerprint) {
		return factory.ContextBundleCurrent, "", ""
	}
	receipt, err := priority.PriorityJudgment(ctx, fingerprint)
	if err != nil {
		return factory.ContextBundleCurrent, "", ""
	}
	bundle, reason := control.ContextBundleFromReceipt(receipt, read.capability, read.assignment, attemptEligible, lineageEligible)
	if bundle == factory.ContextBundleSnapshotHold || reason != "" {
		return factory.ContextBundleCurrent, "", fingerprint
	}
	return bundle, "", fingerprint
}

// buildReadBody renders one read's projection with its eligible
// attempt and lineage sections.
func buildReadBody(read *readContext, attemptEligible, lineageEligible bool, foregrounded string) map[string]any {
	capability, assignment := read.capability, read.assignment
	serveAttempt := attemptEligible && (foregrounded == factory.ContextBundleAttempt || foregrounded == factory.ContextBundleLineage)
	serveLineage := lineageEligible && foregrounded == factory.ContextBundleLineage
	bundle := factory.ContextBundleCurrent
	body := map[string]any{
		"assignment":            capability.AssignmentID,
		"run":                   capability.RunID,
		"project":               capability.Project,
		"identity_source":       "authenticated_worker_context",
		"prompt_sha":            assignment.PromptSHA,
		"source_commit":         assignment.SourceCommit,
		"preparation":           assignment.Preparation,
		"profile":               string(capability.Profile),
		"phase":                 factory.MCPPhaseOf(capability.Profile),
		"subject":               capability.Subject,
		"repository":            capability.Repository,
		"generation":            capability.Generation,
		"capability":            capability.State,
		"deadline":              capability.DeadlineUnix,
		"submissions":           read.count,
		"terminal":              read.frozen,
		"route":                 FactoryMCPRoute,
		"limits":                map[string]any{"page_bytes": factory.MCPMaxPageBytes, "page_refs": factory.MCPMaxPageRefs, "result_bytes": factory.MCPMaxResultPayloadBytes},
		"source_digest":         capability.SourceDigest,
		"scope_digest":          capability.ScopeDigest(),
		"replies_tool":          string(factory.MCPToolInbox),
		"discussion_guidance":   "Read factory.inbox before finishing. A recorded role invitation grants only its exact discussion context and one useful reply through factory.reply with parent_id set to the invited post ID. It does not expand your assignment, execution authority or budget. Skip redundant contributions, explain concrete relevant evidence, and preserve the source identity. Do not invite other roles recursively.",
		"work_post":             map[string]any{"kind": "record", "id": assignment.ID},
		"library_tools":         []string{"factory.search_library", "factory.read_library_item"},
		"library_guidance":      factory.LibraryReadGuidance,
		"consultation_guidance": "System One questions and actual results are automatically recorded in your work post. Never include credentials or private key material. For substantive decisions prepare three independent, counterbalanced questions using the same facts and alternatives before reading any answer, then make three separate calls with the same consultation_group identity. A questions batch is one call and must not be described as three independent calls. Compare the results, then reply to the returned consultation.post_id with your interpretation, decision and next action, or use factory.record_decision with that parent_id for durable decisions. Your explanation must remain distinct from JEV's actual answer. Judgments do not replace native checks or authorization. Read your inbox and effective project decisions/knowledge through the library tools before adopting a change; preserve cited item IDs/revisions and source revision.",
	}
	if assignment.Issue > 0 {
		body["issue"] = assignment.Issue
	}
	if assignment.ActorID > 0 {
		body["actor_id"] = assignment.ActorID
	}
	if serveAttempt {
		body["attempt"] = map[string]any{"run_history": factory.AttemptContinuity(assignment)}
		bundle = factory.ContextBundleAttempt
	}
	if serveLineage {
		lineage := map[string]any{"parent_assignment": assignment.ParentAssignment}
		if read.parent.Outcome != "" {
			lineage["parent_outcome"] = string(read.parent.Outcome)
		}
		if read.parent.Reason != "" {
			lineage["parent_reason"] = read.parent.Reason
		}
		body["lineage"] = lineage
		bundle = factory.ContextBundleLineage
	} else if assignment.ParentAssignment != "" {
		body["lineage_incomplete"] = true
	}
	body["bundle"] = bundle
	if read.frozen {
		body["terminal_submission"] = read.terminal.SubmissionID
		body["terminal_state"] = read.terminal.State
	}
	if assignment.Outcome != "" {
		body["outcome"] = string(assignment.Outcome)
	}
	if assignment.Reason != "" {
		body["reason"] = assignment.Reason
	}
	return body
}

// serveEvidence reports incomplete evidence honestly: bounded pages
// over run/plan/review/inventory records arrive with FW40. A missing
// page is never an empty successful collection.
func (s *factoryCatalogServer) serveEvidence(ctx context.Context, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	capability, release, err := s.admitCall(ctx, factory.MCPToolEvidence, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	// Bounded run projection over the recorded assignment's own
	// attempt: opaque run identities with role and outcome only —
	// never summaries, admissions, identity bindings or prompt-derived
	// content. A missing assignment refuses instead of serving
	// unscoped rows, and an unresolvable recorded run refuses rather
	// than silently dropping it. Plan/review/inventory pages plus
	// pagination arrive with FW40, so evidence stays honestly
	// incomplete.
	assignment, err := s.deps.Assignments.Assignment(ctx, capability.AssignmentID)
	if err != nil {
		return nil, nil, err
	}
	history := assignment.RunHistory
	truncated := false
	if len(history) > factory.MCPMaxPageRefs {
		history = history[:factory.MCPMaxPageRefs]
		truncated = true
	}
	runs := make([]any, 0, len(history))
	for _, id := range history {
		if s.deps.Runs == nil {
			return nil, nil, errors.New("evidence run source unwired")
		}
		run, err := s.deps.Runs.FactoryRun(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		projected := map[string]any{"id": run.ID, "role": run.Role}
		if run.Outcome != "" {
			projected["outcome"] = string(run.Outcome)
		}
		runs = append(runs, projected)
	}
	body := map[string]any{
		"assignment": capability.AssignmentID,
		"runs":       runs,
		"complete":   false,
		"reason":     "plan/review/inventory pages plus pagination arrive with FW40",
	}
	if truncated {
		body["runs_truncated"] = true
	}
	return mcpJSONResult(body)
}

// serveMutation records one immutable proposal with its durable
// receipt. Downstream effects (graph writes, checkpoints, reviews,
// adoption) belong to FW40/FW41/FW42/FW44/FW45 consumers of the
// recorded proposal; this server never mutates the graph directly.
func (s *factoryCatalogServer) serveMutation(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	_, submission, release, err := s.admitMutation(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	receipt := factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	}
	recorded, err := s.deps.Submissions.RecordMCPSubmission(ctx, submission, receipt)
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(s.receiptBody(ctx, submission.AssignmentID, recorded))
}

// factoryCatalogMux serves the catalog on the fixed route only: POST
// reaches the SDK handler (stateless JSON, no SSE); every other
// method — including the SSE-establishing GET — returns 405.
// Requests carrying a browser Origin never belong on this private
// listener and refuse before the SDK sees them.
func factoryCatalogMux(server *mcp.Server) *http.ServeMux {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	mux := http.NewServeMux()
	mux.HandleFunc(FactoryMCPRoute, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Origin") != "" {
			http.Error(w, "origin refused", http.StatusForbidden)
			return
		}
		handler.ServeHTTP(w, r)
	})
	return mux
}

// FactoryMCPHandler exposes the catalog on the private listener.
// It refuses everything when the store is unavailable: transport
// without persistence must not admit calls.
func (s *API) FactoryMCPHandler() http.Handler {
	if s == nil || s.Store == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "factory MCP unavailable", http.StatusServiceUnavailable)
		})
	}
	if s.Config == nil {
		return unavailableFactoryMCP(errors.New("factory configuration unavailable"))
	}
	key, err := control.LoadFactoryMCPKey(s.Config.FactoryPublicationRoot)
	if err != nil {
		return unavailableFactoryMCP(err)
	}
	return factoryBoundCatalogHandler(FactoryMCPDeps{
		IssueMaker:           s.factoryIssueMakerTool,
		Capabilities:         s.Store,
		Submissions:          s.Store,
		Assignments:          s.Store,
		PriorityJudgments:    s.Store,
		Runs:                 s.Store,
		Prepare:              s.prepareFactoryWorkerMCP,
		SystemOne:            &control.TypeSafeSystemOne{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel},
		VerificationEvidence: s.factoryVerificationEvidence,
		RepairPreview:        s.factoryRepairPreview,
		Catalog:              s.factoryApprovedVerificationCatalog,
		NativeTask:           s.factoryNativeTask,
	}, key)
}

var (
	_ FactoryMCPCapabilitySource       = (*store.Store)(nil)
	_ FactoryMCPSubmissionSink         = (*store.Store)(nil)
	_ FactoryMCPAssignmentSource       = (*store.Store)(nil)
	_ FactoryMCPPriorityJudgmentSource = (*store.Store)(nil)
	_ FactoryMCPRunSource              = (*store.Store)(nil)
)

// mcpSubmissionID turns whatever the worker sent into the 32-hex submission
// identity: a valid identity passes, any other string is hashed so the same
// string recovers the same receipt, and a mutation without one gets a fresh
// identity. Workers never have to mint identifiers.
func mcpSubmissionID(id string, mutation bool) string {
	switch {
	case factory.ValidID(id):
		return id
	case id != "":
		sum := sha256.Sum256([]byte(id))
		return hex.EncodeToString(sum[:16])
	case mutation:
		return factory.NewID()
	}
	return ""
}
