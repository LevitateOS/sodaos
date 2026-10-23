package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
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
	State        any                              `json:"state"`
	Instructions any                              `json:"instructions"`
	Criteria     any                              `json:"criteria,omitempty"`
	Questions    map[string]systemOneQuestionArgs `json:"questions,omitempty"`
	CapabilityID string                           `json:"capability_id,omitempty"`
}

func isSystemOneTool(tool factory.MCPTool) bool {
	return tool == factory.MCPToolSystemOneNoul || tool == factory.MCPToolSystemOneChoice || tool == factory.MCPToolSystemOneScore
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
	description := "Ask a generic System One " + kind + " judgment using supplied state. Returns typed probabilities; does not change Factory or library state. Never include credentials. Pass questions to batch up to 8 independent same-primitive judgments over the shared state in one round trip."
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
		if s.deps.SystemOne == nil {
			return nil, nil, errors.New("System One provider is not configured")
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
		if prior, err := s.deps.PriorityJudgments.PriorityJudgment(ctx, snapshot); err == nil && prior.Outcome == systemOneAnswerOutcome && prior.Answer != nil {
			return mcpJSONResult(prior.Answer)
		}
		if err := s.requireCurrentWork(ctx, capability); err != nil {
			return nil, nil, err
		}
		reserved, err := s.deps.PriorityJudgments.ReservePriorityJudgment(ctx, snapshot, 1)
		if err != nil {
			return nil, nil, errors.New("priority consultation is unavailable")
		}
		if !reserved {
			replay, err := s.deps.PriorityJudgments.PriorityJudgment(ctx, snapshot)
			if err != nil || replay.Outcome != systemOneAnswerOutcome || replay.Answer == nil {
				return nil, nil, errors.New("priority consultation is unavailable")
			}
			return mcpJSONResult(replay.Answer)
		}
		result, err := s.deps.SystemOne.Evaluate(ctx, kind, args.State, args.Instructions, args.Criteria)
		if err != nil {
			return nil, nil, err
		}
		if err := s.deps.PriorityJudgments.RecordPriorityJudgment(ctx, factory.PriorityReceipt{
			Snapshot: snapshot, Outcome: systemOneAnswerOutcome, Answer: result, CreatedUnix: time.Now().Unix(),
		}); err != nil {
			return nil, nil, err
		}
		return mcpJSONResult(result)
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
	if prior, err := s.deps.PriorityJudgments.PriorityJudgment(ctx, snapshot); err == nil && prior.Outcome == systemOneAnswerOutcome && prior.Answer != nil {
		return mcpJSONResult(prior.Answer)
	}
	if err := s.requireCurrentWork(ctx, capability); err != nil {
		return nil, nil, err
	}
	reserved, err := s.deps.PriorityJudgments.ReservePriorityJudgment(ctx, snapshot, 1)
	if err != nil {
		return nil, nil, errors.New("priority consultation is unavailable")
	}
	if !reserved {
		replay, err := s.deps.PriorityJudgments.PriorityJudgment(ctx, snapshot)
		if err != nil || replay.Outcome != systemOneAnswerOutcome || replay.Answer == nil {
			return nil, nil, errors.New("priority consultation is unavailable")
		}
		return mcpJSONResult(replay.Answer)
	}
	result, err := s.deps.SystemOne.EvaluateQuestions(ctx, kind, args.State, questions)
	if err != nil {
		return nil, nil, err
	}
	if err := s.deps.PriorityJudgments.RecordPriorityJudgment(ctx, factory.PriorityReceipt{
		Snapshot: snapshot, Outcome: systemOneAnswerOutcome, Answer: result, CreatedUnix: time.Now().Unix(),
	}); err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(result)
}
