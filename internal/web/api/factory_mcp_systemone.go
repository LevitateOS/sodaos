package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"strings"
)

type FactorySystemOne interface {
	Evaluate(context.Context, string, any, any, any) (map[string]any, error)
	EvaluateQuestions(context.Context, string, any, map[string]factory.SystemOneWorkerQuestion) (map[string]any, error)
}

// systemOneQuestionArgs is one batched same-primitive question over
// the shared call state.
type systemOneQuestionArgs struct {
	Instructions any `json:"instructions"`
	Criteria     any `json:"criteria,omitempty"`
}

type systemOneArgs struct {
	State             any                              `json:"state"`
	Instructions      any                              `json:"instructions"`
	Criteria          any                              `json:"criteria,omitempty"`
	Questions         map[string]systemOneQuestionArgs `json:"questions,omitempty"`
	CapabilityID      string                           `json:"capability_id,omitempty"`
	ConsultationGroup string                           `json:"consultation_group,omitempty"`
}

func isSystemOneTool(tool factory.MCPTool) bool {
	return tool == factory.MCPToolSystemOneNoul || tool == factory.MCPToolSystemOneChoice || tool == factory.MCPToolSystemOneScore
}

// The production journal also records the consultation in its authorized work
// discussion. An unavailable write prevents an unrecorded provider dispatch.
type consultationSink interface {
	RecordConsultationRequest(context.Context, store.SystemOneConsultation) error
	RecordConsultationResult(context.Context, string, string, string, map[string]any) error
}

func consultationModel(result map[string]any) string {
	if result == nil {
		return ""
	}
	if model, ok := result["model"].(string); ok {
		return model
	}
	return ""
}

func recordConsultationRequest(deps FactoryMCPDeps, ctx context.Context, cap factory.MCPCapability, snapshot, groupID string, tool factory.MCPTool, kind string, state, instructions, criteria any, questions map[string]any) error {
	sink, ok := deps.PriorityJudgments.(consultationSink)
	if !ok {
		return nil
	}
	sourceRevision := cap.AcceptedRevision
	if sourceRevision == "" {
		sourceRevision = cap.SourceDigest
	}
	return sink.RecordConsultationRequest(ctx, store.SystemOneConsultation{
		Repository: cap.Repository, Role: cap.Profile,
		FeedConsultation: factory.FeedConsultation{
			Snapshot: snapshot, GroupID: groupID, Assignment: cap.AssignmentID,
			SourceRevision: sourceRevision, Tool: string(tool), Kind: kind,
			State: state, Instructions: instructions, Criteria: criteria, Questions: questions,
			Status: store.ConsultationStatusPending,
		},
	})
}

func recordConsultationResult(deps FactoryMCPDeps, ctx context.Context, snapshot string, result map[string]any, callErr error) error {
	sink, ok := deps.PriorityJudgments.(consultationSink)
	if !ok {
		return nil
	}
	// Preserve the actual result even when the worker request was cancelled.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if callErr != nil {
		status := store.ConsultationStatusFailed
		if strings.Contains(callErr.Error(), "not configured") {
			status = store.ConsultationStatusUnavailable
		}
		if errors.Is(callErr, context.DeadlineExceeded) || errors.Is(callErr, context.Canceled) || strings.Contains(callErr.Error(), "request failed") {
			status = store.ConsultationStatusUncertain
		}
		return sink.RecordConsultationResult(writeCtx, snapshot, "", status, nil)
	}
	return sink.RecordConsultationResult(writeCtx, snapshot, consultationModel(result), store.ConsultationStatusReturned, result)
}

// systemOneAnswerOutcome marks a recorded worker consultation answer:
// the replayable counterpart to the tournament selected/fairness
// outcomes, read only through the funding snapshot.
const systemOneAnswerOutcome = "answered"

func (s *factoryCatalogServer) addSystemOneTool(server *mcp.Server, tool factory.MCPTool) {
	kind := strings.TrimPrefix(string(tool), "systemone.")
	// instructions/criteria stay optional in the schema so a batch can
	// supply questions instead; the handler enforces the required
	// fields per invocation shape before any funding.
	schema := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"state": {}, "instructions": {Types: []string{"string", "object", "array"}}, "criteria": {}}, Required: []string{"state"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	schema.Properties["questions"] = &jsonschema.Schema{Type: "object", Description: "Up to 8 independent same-primitive questions over the shared state, answered in one round trip. Mutually exclusive with instructions/criteria.", AdditionalProperties: &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"instructions": {}, "criteria": {}}, Required: []string{"instructions"}}}
	schema.Properties["consultation_group"] = &jsonschema.Schema{Type: "string", Description: "Optional shared 32-character lowercase hex identity for three separate consultations supporting the same decision. Grouping does not change funding or replay identity."}
	description := "Ask a generic System One " + kind + " judgment using supplied state. Records the question and result in your work post and returns their comment identity. Advice does not authorize or implement changes. Never include credentials. Pass questions to batch up to 8 independent same-primitive judgments over the shared state in one round trip. Follow with your interpretation and resulting decision as a reply to consultation.post_id."
	switch kind {
	case "noul":
		schema.Properties["criteria"] = &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"true": {}, "false": {}}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
		description += " Noul is probability of yes from 0 to 1."
	case "choice":
		schema.Properties["criteria"] = &jsonschema.Schema{Type: "object", Description: "2–255 named alternatives with their descriptions."}
	case "score":
		schema.Properties["criteria"] = &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Types: []string{"string", "object", "array"}}, MinItems: ptrInt(2), MaxItems: ptrInt(10), Description: "Ordered descriptions from lowest to highest; response score is between 0 and level count minus one."}
	}
	if s.deps.BoundCapability == nil {
		schema.Properties["capability_id"] = &jsonschema.Schema{Type: "string"}
		schema.Required = append(schema.Required, "capability_id")
	}
	mcp.AddTool(server, &mcp.Tool{Name: string(tool), Description: description, InputSchema: schema}, func(ctx context.Context, _ *mcp.CallToolRequest, args systemOneArgs) (*mcp.CallToolResult, any, error) {
		capability, release, err := s.admitCall(ctx, tool, mcpCallArgs{CapabilityID: args.CapabilityID})
		if err != nil {
			return nil, nil, err
		}
		defer release()
		if args.ConsultationGroup != "" && !factory.ValidID(args.ConsultationGroup) {
			return nil, nil, errors.New("consultation group must be a 32-character lowercase hex identity")
		}
		raw, err := json.Marshal(args)
		if err != nil || len(raw) > factory.MCPMaxSubmissionPayload {
			return nil, nil, errors.New("System One question exceeds 64 KiB")
		}
		if len(args.Questions) > 0 {
			return s.serveSystemOneBatch(ctx, capability, tool, kind, args)
		}
		if args.Instructions == nil {
			return nil, nil, errors.New("instructions are required")
		}
		if text, ok := args.Instructions.(string); ok && strings.TrimSpace(text) == "" {
			return nil, nil, errors.New("instructions are empty")
		}
		if kind == "choice" {
			criteria, ok := args.Criteria.(map[string]any)
			if !ok || len(criteria) < 2 || len(criteria) > 255 {
				return nil, nil, errors.New("choice requires 2–255 alternatives")
			}
		}
		if kind == "score" {
			levels, ok := args.Criteria.([]any)
			if !ok || len(levels) < 2 || len(levels) > 10 {
				return nil, nil, errors.New("score requires 2–10 levels")
			}
		}
		if s.deps.PriorityJudgments == nil {
			return nil, nil, errors.New("priority consultation is unavailable")
		}
		// Funded worker consultation (R30.3/R11.2): one reservation
		// per admitted caller, tool and question bytes. A recorded
		// answer replays without new spend or a second provider call;
		// a reservation without an answer means the prior consultation
		// failed, and the retained reservation holds instead of
		// spending again. Provider failures keep their spend.
		snapshot, err := factory.SystemOneWorkerSnapshot(capability.ID, tool, args.State, args.Instructions, args.Criteria)
		if err != nil {
			return nil, nil, errors.New("System One question is not hashable")
		}
		return s.serveFundedConsultation(ctx, capability, snapshot, tool, kind, args, nil, func() (map[string]any, error) {
			return s.deps.SystemOne.Evaluate(ctx, kind, args.State, args.Instructions, args.Criteria)
		})
	})
}

// serveSystemOneBatch judges one batch of independent same-primitive
// questions over shared state in a single funded round trip. The whole
// envelope funds, replays and records as one consultation; a question
// needing an earlier result or new evidence belongs in a later call.
func (s *factoryCatalogServer) serveSystemOneBatch(ctx context.Context, capability factory.MCPCapability, tool factory.MCPTool, kind string, args systemOneArgs) (*mcp.CallToolResult, any, error) {
	if args.Instructions != nil || args.Criteria != nil {
		return nil, nil, errors.New("single question and batch questions are mutually exclusive")
	}
	if len(args.Questions) > factory.SystemOneMaxBatchQuestions {
		return nil, nil, errors.New("System One batch needs 1 to 8 questions")
	}
	questions := make(map[string]factory.SystemOneWorkerQuestion, len(args.Questions))
	for id, question := range args.Questions {
		if question.Instructions == nil {
			return nil, nil, errors.New("instructions are required")
		}
		if text, ok := question.Instructions.(string); ok && strings.TrimSpace(text) == "" {
			return nil, nil, errors.New("instructions are empty")
		}
		questions[id] = factory.SystemOneWorkerQuestion{Instructions: question.Instructions, Criteria: question.Criteria}
	}
	if s.deps.PriorityJudgments == nil {
		return nil, nil, errors.New("priority consultation is unavailable")
	}
	snapshot, err := factory.SystemOneWorkerBatchSnapshot(capability.ID, tool, args.State, questions)
	if err != nil {
		return nil, nil, errors.New("System One question is not hashable")
	}
	batchQuestions := make(map[string]any, len(questions))
	for id, question := range questions {
		batchQuestions[id] = map[string]any{"instructions": question.Instructions, "criteria": question.Criteria}
	}
	return s.serveFundedConsultation(ctx, capability, snapshot, tool, kind, args, batchQuestions, func() (map[string]any, error) {
		return s.deps.SystemOne.EvaluateQuestions(ctx, kind, args.State, questions)
	})
}

func consultationResponse(cap factory.MCPCapability, snapshot string, answer map[string]any) (*mcp.CallToolResult, any, error) {
	result := make(map[string]any, len(answer)+1)
	for key, value := range answer {
		result[key] = value
	}
	result["consultation"] = map[string]any{"post_id": factory.FeedConsultationPostID(snapshot), "work_post": map[string]any{"kind": "record", "id": cap.AssignmentID}, "follow_up": "Reply to this consultation with your interpretation and chosen next action. Use factory.record_decision for a durable decision. JEV output above is advice, not an implemented or authorized change."}
	return mcpJSONResult(result)
}

func (s *factoryCatalogServer) serveFundedConsultation(ctx context.Context, cap factory.MCPCapability, snapshot string, tool factory.MCPTool, kind string, args systemOneArgs, questions map[string]any, call func() (map[string]any, error)) (*mcp.CallToolResult, any, error) {
	recordRequest := func() error {
		group := args.ConsultationGroup
		if group == "" {
			group = snapshot
		}
		return recordConsultationRequest(s.deps, ctx, cap, snapshot, group, tool, kind, args.State, args.Instructions, args.Criteria, questions)
	}
	replay := func(answer map[string]any) (*mcp.CallToolResult, any, error) {
		if err := recordRequest(); err != nil {
			return nil, nil, err
		}
		if err := recordConsultationResult(s.deps, ctx, snapshot, answer, nil); err != nil {
			return nil, nil, err
		}
		return consultationResponse(cap, snapshot, answer)
	}
	if prior, err := s.deps.PriorityJudgments.PriorityJudgment(ctx, snapshot); err == nil && prior.Outcome == systemOneAnswerOutcome && prior.Answer != nil {
		return replay(prior.Answer)
	}
	if err := s.requireCurrentWork(ctx, cap); err != nil {
		return nil, nil, err
	}
	if err := recordRequest(); err != nil {
		return nil, nil, errors.New("consultation could not be recorded; provider was not called")
	}
	unavailable := func(message string) (*mcp.CallToolResult, any, error) {
		var writeErr error
		if sink, ok := s.deps.PriorityJudgments.(consultationSink); ok {
			writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			writeErr = sink.RecordConsultationResult(writeCtx, snapshot, "", store.ConsultationStatusUnavailable, nil)
		}
		return nil, nil, errors.Join(errors.New(message), writeErr)
	}
	if s.deps.SystemOne == nil {
		return unavailable("System One provider is not configured")
	}
	reserved, err := s.deps.PriorityJudgments.ReservePriorityJudgment(ctx, snapshot, 1)
	if err != nil {
		return unavailable("priority consultation is unavailable")
	}
	if !reserved {
		prior, err := s.deps.PriorityJudgments.PriorityJudgment(ctx, snapshot)
		if err != nil || prior.Outcome != systemOneAnswerOutcome || prior.Answer == nil {
			callErr := errors.New("System One recorded an answer for this request that cannot be replayed")
			return nil, nil, errors.Join(callErr, recordConsultationResult(s.deps, ctx, snapshot, nil, callErr))
		}
		return replay(prior.Answer)
	}
	result, callErr := call()
	if callErr != nil {
		return nil, nil, errors.Join(callErr, recordConsultationResult(s.deps, ctx, snapshot, nil, callErr))
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	journalErr := s.deps.PriorityJudgments.RecordPriorityJudgment(writeCtx, factory.PriorityReceipt{Snapshot: snapshot, Outcome: systemOneAnswerOutcome, Answer: result, CreatedUnix: time.Now().Unix()})
	commentErr := recordConsultationResult(s.deps, ctx, snapshot, result, nil)
	if err := errors.Join(journalErr, commentErr); err != nil {
		return nil, nil, err
	}
	return consultationResponse(cap, snapshot, result)
}
