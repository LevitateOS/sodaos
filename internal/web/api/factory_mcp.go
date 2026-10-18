package api

// D14 fourteen-tool catalog server with capability admission,
// immutable submissions and receipt recovery (FW38).
//
// The pinned upstream Go MCP SDK (github.com/modelcontextprotocol/go-sdk
// v1.8.0) serves one implementation over stdio (through the Rust
// relay, FW39) and over the fixed Factory route on the dedicated
// private listener (internal/web + cmd/soda-dashboard). The SDK
// validates every call against the explicit per-tool input schema
// server-side: unknown tools and unknown fields refuse before
// admission. Every request validates capability plus
// profile/tool/input binding; mutations repeat freshness/authority
// checks at admission and persist through control admission into the
// store. Transport/schema/authentication failure creates no receipt.
//
// Reads serve capability-bound state from owned rows. Evidence pages
// over run/plan/review/inventory records, dispatch/settlement
// consumers and downstream mutation effects (graph writes,
// checkpoints, reviews) belong to FW40/FW41/FW42/FW44/FW45: this
// server records proposals with durable receipts and reports
// incomplete evidence honestly instead of an empty success.
// Proof-of-possession request MACs and host run/incarnation
// observation ride the FW39 relay/staging joint; until then the
// private Unix socket plus per-request capability validation is the
// request authentication, and empty host observations skip only the
// run-binding comparison.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	MCPReceipt(ctx context.Context, assignmentID, submissionID string) (factory.MCPReceipt, error)
	MCPSubmissionCount(ctx context.Context, assignmentID string) (int, error)
}

// FactoryMCPAssignmentSource loads the recorded assignment behind a
// capability for current-context projection. *store.Store implements
// it; tests stub it.
type FactoryMCPAssignmentSource interface {
	Assignment(ctx context.Context, id string) (factory.Assignment, error)
}

// FactoryMCPDeps wires the catalog server. Observed supplies the
// host-observed run state from outside SQL; nil keeps the recorded
// binding only (FW39 plugs host observation here). Catalog backs
// D16 proposal validation; nil names the unwired catalog honestly
// (FW40 wires the approved check catalog).
type FactoryMCPDeps struct {
	Capabilities FactoryMCPCapabilitySource
	Submissions  FactoryMCPSubmissionSink
	Assignments  FactoryMCPAssignmentSource
	Observed     func(capability factory.MCPCapability) (control.MCPRunObservation, error)
	Catalog      factory.ApprovedCatalog
	Now          func() time.Time
}

// mcpCallArgs is the uniform tool envelope. Mutations require
// submission_id, expected_input_digest and payload; reads require
// only capability_id; receipt requires capability_id plus
// submission_id. Payload canonical bytes are json.Marshal of the
// payload map (sorted keys, deterministic): expected_input_digest
// names the caller's capability scope snapshot and must equal the
// admitted capability's current scope digest, never the payload
// hash.
type mcpCallArgs struct {
	CapabilityID        string         `json:"capability_id" jsonschema:"recorded capability identity"`
	SubmissionID        string         `json:"submission_id,omitempty" jsonschema:"32-hex submission identity for mutations and receipt recovery"`
	ExpectedInputDigest string         `json:"expected_input_digest,omitempty" jsonschema:"SHA-256 hex of the canonical payload bytes"`
	Payload             map[string]any `json:"payload,omitempty" jsonschema:"tool payload within 64 KiB"`
}

// mcpEnvelopeSchema advertises and enforces the envelope per tool.
// additionalProperties false refuses unknown top-level fields
// server-side through SDK schema validation, and required lists
// differ for reads, receipt and mutations.
func mcpEnvelopeSchema(tool factory.MCPTool) *mcp.Tool {
	required := []string{"capability_id"}
	if tool == factory.MCPToolReceipt {
		required = []string{"capability_id", "submission_id"}
	} else if !factory.MCPIsRead(tool) {
		required = []string{"capability_id", "submission_id", "expected_input_digest", "payload"}
	}
	return &mcp.Tool{
		Name:        string(tool),
		Description: "Factory " + string(tool),
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"capability_id":         {Type: "string"},
				"submission_id":         {Type: "string"},
				"expected_input_digest": {Type: "string"},
				"payload":               {Type: "object"},
			},
			Required:             required,
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		},
	}
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
	if !factory.ValidID(args.CapabilityID) {
		return factory.MCPCapability{}, nil, errors.New("factory capability refused")
	}
	capability, err := s.deps.Capabilities.GetMCPCapability(ctx, args.CapabilityID)
	if err != nil {
		return factory.MCPCapability{}, nil, errors.New("factory capability refused")
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
		return nil, errors.New("missing tool payload")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("tool payload is not JSON")
	}
	if len(raw) > factory.MCPMaxSubmissionPayload {
		return nil, errors.New("tool payload exceeds 64 KiB")
	}
	return raw, nil
}

// mcpPayloadSpec is the strict per-tool payload shape: required
// keys, enum-constrained fields and hex-digest fields. Unknown
// payload keys refuse.
type mcpPayloadSpec struct {
	required []string
	enums    map[string][]string
	hex      []string
}

var mcpMutationPayloads = map[factory.MCPTool]mcpPayloadSpec{
	factory.MCPToolFinishIncomplete: {
		required: []string{"outcome", "cause", "evidence_refs", "affected_task_ids", "requested_resolution"},
		enums:    map[string][]string{"outcome": {"blocked", "failed", "cancelled"}},
	},
	factory.MCPToolFinishProductPlan: {
		required: []string{"candidates", "evidence_refs", "coverage"},
	},
	factory.MCPToolFinishIssuePlan: {
		required: []string{"evidence_refs", "approach", "tasks"},
	},
	factory.MCPToolFinishImplement: {
		required: []string{},
	},
	factory.MCPToolFinishReview: {
		required: []string{"conclusion", "coverage", "evidence_refs"},
		enums:    map[string][]string{"conclusion": {"clear", "findings", "incomplete"}},
	},
	factory.MCPToolFinishCorrection: {
		required: []string{"finding_digest", "dispositions", "evidence_refs"},
		hex:      []string{"finding_digest"},
	},
	factory.MCPToolFinishReassessment: {
		required: []string{"subject", "report"},
	},
	factory.MCPToolFinishAppVerify: {
		required: []string{"delivery_id", "outcome", "coverage"},
		enums:    map[string][]string{"outcome": {"verified", "failed", "incomplete"}},
	},
	factory.MCPToolSubmitTaskOutput: {
		required: []string{"task_id", "manifest_digest", "output_digest", "evidence_refs"},
		hex:      []string{"manifest_digest", "output_digest"},
	},
	factory.MCPToolRequestHandoff: {
		required: []string{"predecessor_task", "successor_task", "manifest_digest", "output_submission_id"},
		hex:      []string{"manifest_digest", "output_submission_id"},
	},
}

func isHex64(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range []byte(value) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func checkPayloadKeys(tool factory.MCPTool, payload map[string]any, spec mcpPayloadSpec) error {
	for key := range payload {
		known := false
		for _, want := range spec.required {
			if key == want {
				known = true
			}
		}
		if !known && !mcpOptionalPayloadKey(tool, key) {
			return fmt.Errorf("unknown payload field %q", key)
		}
	}
	return nil
}

func checkPayloadRequired(payload map[string]any, spec mcpPayloadSpec) error {
	for _, want := range spec.required {
		value, ok := payload[want]
		if !ok {
			return fmt.Errorf("missing payload field %q", want)
		}
		if text, ok := value.(string); ok && (text == "" || len(text) > factory.MCPMaxReasonChars) {
			return fmt.Errorf("payload field %q out of bound", want)
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
			return fmt.Errorf("payload field %q carries an unsupported value", field)
		}
	}
	return nil
}

func checkPayloadHex(payload map[string]any, spec mcpPayloadSpec) error {
	for _, field := range spec.hex {
		text, _ := payload[field].(string)
		if !isHex64(text) && !factory.ValidID(text) {
			return fmt.Errorf("payload field %q is not a digest", field)
		}
	}
	return nil
}

func (s *factoryCatalogServer) checkPayloadOutcome(tool factory.MCPTool, payload map[string]any) error {
	if tool == factory.MCPToolFinishImplement {
		if _, unchanged := payload["unchanged"]; unchanged {
			return nil
		}
		if _, candidate := payload["candidate"]; !candidate {
			return fmt.Errorf("implementation finish needs a candidate or an explicit unchanged outcome")
		}
	}
	if tool == factory.MCPToolFinishProductPlan {
		if err := s.checkManifestProposal(payload); err != nil {
			return err
		}
		if candidates, ok := payload["candidates"].([]any); !ok || len(candidates) < 1 || len(candidates) > 3 {
			return fmt.Errorf("product plan needs 1-3 graph candidates")
		}
	}
	return nil
}

// checkPayload enforces the strict per-tool shape plus the D16
// manifest-proposal contract on product plans.
func (s *factoryCatalogServer) checkPayload(tool factory.MCPTool, payload map[string]any) error {
	spec, ok := mcpMutationPayloads[tool]
	if !ok {
		return fmt.Errorf("unsupported factory tool")
	}
	if err := checkPayloadKeys(tool, payload, spec); err != nil {
		return err
	}
	if err := checkPayloadRequired(payload, spec); err != nil {
		return err
	}
	if err := checkPayloadEnums(payload, spec); err != nil {
		return err
	}
	if err := checkPayloadHex(payload, spec); err != nil {
		return err
	}
	return s.checkPayloadOutcome(tool, payload)
}

// mcpOptionalPayloadKey admits documented optional keys beyond the
// required set without opening arbitrary input.
func mcpOptionalPayloadKey(tool factory.MCPTool, key string) bool {
	switch tool {
	case factory.MCPToolFinishImplement:
		return key == "unchanged" || key == "candidate"
	case factory.MCPToolFinishProductPlan:
		return key == "questions" || key == "verification_manifest_proposal"
	case factory.MCPToolFinishIssuePlan:
		return key == "alternatives" || key == "unresolved_facts"
	case factory.MCPToolFinishReview:
		return key == "tuple" || key == "carried_assessments" || key == "findings"
	case factory.MCPToolFinishIncomplete:
		return key == "cause_detail"
	case factory.MCPToolSubmitTaskOutput:
		return key == "native_child_ref"
	case factory.MCPToolFinishAppVerify:
		return key == "check_receipts" || key == "artifact_refs" || key == "failures" || key == "input_digest"
	case factory.MCPToolFinishReassessment:
		return key == "input_digest"
	case factory.MCPToolFinishCorrection:
		return key == "candidate"
	default:
		return false
	}
}

// checkManifestProposal releases the D16 contract: when the product
// plan carries a verification_manifest_proposal, admission validates
// outcome coverage, approved catalog references, parameter bounds
// and artifact/fixture requirements. FW40 adopts valid proposals;
// FW45 verifies against adopted manifests.
func (s *factoryCatalogServer) checkManifestProposal(payload map[string]any) error {
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
	var proposal factory.VerificationManifestProposal
	if err := json.Unmarshal(encoded, &proposal); err != nil {
		return errors.New("manifest proposal shape refused")
	}
	if err := factory.ValidateManifestProposal(proposal, s.deps.Catalog); err != nil {
		return err
	}
	return nil
}

// admitMutation runs the shared mutation gate after admitCall:
// strict payload shape, digest binding and control submission
// admission (terminal freeze, handoff cap).
func (s *factoryCatalogServer) admitMutation(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (factory.MCPCapability, factory.MCPSubmission, func(), error) {
	capability, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return factory.MCPCapability{}, factory.MCPSubmission{}, nil, err
	}
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
		Tool:         tool, ExpectedInputDigest: args.ExpectedInputDigest,
		PayloadHash: factory.MCPInputDigest(raw), Payload: raw,
	}
	if err := submission.Validate(); err != nil {
		release()
		return factory.MCPCapability{}, factory.MCPSubmission{}, nil, err
	}
	if err := control.AdmitMCPSubmission(ctx, s.deps.Submissions, capability, submission); err != nil {
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

// newFactoryCatalogServer builds the fourteen-tool catalog. Reads
// serve capability-bound state; mutations record immutable proposals
// with durable receipts; receipt recovers lost responses.
func newFactoryCatalogServer(deps FactoryMCPDeps) *mcp.Server {
	catalog := &factoryCatalogServer{deps: deps, limiter: newMCPRateLimiter()}
	server := mcp.NewServer(&mcp.Implementation{Name: "soda-factory"}, nil)
	for _, tool := range factory.MCPTools {
		tool := tool
		mcp.AddTool(server, mcpEnvelopeSchema(tool), func(ctx context.Context, _ *mcp.CallToolRequest, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
			return catalog.serve(ctx, tool, args)
		})
	}
	return server
}

func (s *factoryCatalogServer) serve(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	switch tool {
	case factory.MCPToolContext, factory.MCPToolStatus:
		return s.serveRead(ctx, tool, args)
	case factory.MCPToolEvidence:
		return s.serveEvidence(ctx, args)
	case factory.MCPToolReceipt:
		return s.serveReceipt(ctx, args)
	default:
		return s.serveMutation(ctx, tool, args)
	}
}

// serveRead serves capability-bound reads: the immutable assignment
// summary (context) and the live capability plus recorded receipt
// state (status). Reading never launches work and records no row.
func (s *factoryCatalogServer) serveRead(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	capability, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	terminal, frozen, err := s.deps.Submissions.TerminalMCPProposal(ctx, capability.AssignmentID)
	if err != nil {
		return nil, nil, err
	}
	count, err := s.deps.Submissions.MCPSubmissionCount(ctx, capability.AssignmentID)
	if err != nil {
		return nil, nil, err
	}
	// Current-context projection reads the recorded assignment behind
	// the capability: prompt digest, exact source/preparation and, once
	// set, outcome/reason. Prompt bytes and actor identity never enter
	// the projection. Own-attempt run identities join the attempt
	// bundle and the linked parent outcome joins the lineage bundle;
	// both stay within the admitted local lineage. A missing
	// assignment refuses instead of serving context without its
	// assignment.
	assignment, err := s.deps.Assignments.Assignment(ctx, capability.AssignmentID)
	if err != nil {
		return nil, nil, err
	}
	var parent factory.Assignment
	parentLoaded := false
	if assignment.ParentAssignment != "" {
		if loaded, err := s.deps.Assignments.Assignment(ctx, assignment.ParentAssignment); err == nil {
			parent, parentLoaded = loaded, true
		}
	}
	attemptEligible, lineageEligible := factory.BundleEligibility(assignment, parentLoaded)
	bundle := factory.ContextBundleCurrent
	body := map[string]any{
		"assignment":    capability.AssignmentID,
		"prompt_sha":    assignment.PromptSHA,
		"source_commit": assignment.SourceCommit,
		"preparation":   assignment.Preparation,
		"profile":       string(capability.Profile),
		"phase":         factory.MCPPhaseOf(capability.Profile),
		"subject":       capability.Subject,
		"repository":    capability.Repository,
		"generation":    capability.Generation,
		"capability":    capability.State,
		"deadline":      capability.DeadlineUnix,
		"submissions":   count,
		"terminal":      frozen,
		"route":         FactoryMCPRoute,
		"limits":        map[string]any{"page_bytes": factory.MCPMaxPageBytes, "page_refs": factory.MCPMaxPageRefs, "result_bytes": factory.MCPMaxResultPayloadBytes},
		"source_digest": capability.SourceDigest,
		"scope_digest":  capability.ScopeDigest(),
	}
	if attemptEligible {
		body["attempt"] = map[string]any{"run_history": factory.AttemptContinuity(assignment)}
		bundle = factory.ContextBundleAttempt
	}
	if lineageEligible {
		lineage := map[string]any{"parent_assignment": assignment.ParentAssignment}
		if parent.Outcome != "" {
			lineage["parent_outcome"] = string(parent.Outcome)
		}
		if parent.Reason != "" {
			lineage["parent_reason"] = parent.Reason
		}
		body["lineage"] = lineage
		bundle = factory.ContextBundleLineage
	} else if assignment.ParentAssignment != "" {
		body["lineage_incomplete"] = true
	}
	body["bundle"] = bundle
	if frozen {
		body["terminal_submission"] = terminal.SubmissionID
		body["terminal_state"] = terminal.State
	}
	if assignment.Outcome != "" {
		body["outcome"] = string(assignment.Outcome)
	}
	if assignment.Reason != "" {
		body["reason"] = assignment.Reason
	}
	return mcpJSONResult(body)
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
	_ = capability
	return mcpJSONResult(map[string]any{
		"complete": false,
		"reason":   "evidence pages over run/plan/review/inventory records arrive with FW40",
	})
}

// serveReceipt recovers the durable receipt for one submission: the
// lost-response path. Unknown IDs refuse without disclosing other
// assignments.
func (s *factoryCatalogServer) serveReceipt(ctx context.Context, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	capability, release, err := s.admitCall(ctx, factory.MCPToolReceipt, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	if !factory.ValidID(args.SubmissionID) {
		return nil, nil, errors.New("receipt not found")
	}
	receipt, err := s.deps.Submissions.MCPReceipt(ctx, capability.AssignmentID, args.SubmissionID)
	if err != nil {
		return nil, nil, errors.New("receipt not found")
	}
	return mcpJSONResult(map[string]any{
		"submission_id": receipt.SubmissionID,
		"state":         receipt.State,
	})
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
	return mcpJSONResult(map[string]any{
		"submission_id": recorded.SubmissionID,
		"state":         recorded.State,
	})
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
	return factoryCatalogMux(newFactoryCatalogServer(FactoryMCPDeps{
		Capabilities: s.Store,
		Submissions:  s.Store,
		Assignments:  s.Store,
	}))
}

var _ FactoryMCPCapabilitySource = (*store.Store)(nil)
var _ FactoryMCPSubmissionSink = (*store.Store)(nil)
var _ FactoryMCPAssignmentSource = (*store.Store)(nil)
