package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// Exercise the actual durable submission/projection route for every worker
// presentation, not just catalog metadata. Terminal cards retain assignment
// lifecycle truth; controller and library mutation cards use their own owners.
func TestEveryWorkerFeedComponentHasDurableToolPath(t *testing.T) {
	s, dsn := postgresFixture(t, nil)
	ctx := context.Background()
	sqlDB := journeySQL(t, dsn)
	count := 0
	for index, component := range factory.FeedComponents() {
		if component.Authority != "worker" {
			continue
		}
		t.Run(component.ID, func(t *testing.T) {
			if len(component.Outlets) == 0 {
				t.Fatal("component has no tool producer")
			}
			tool := factory.MCPTool(component.Outlets[0])
			// These cards retain their ordinary communication outlet. Their
			// typed diagnosis/draft producers need the exact active repair
			// cause exercised by the existing control/HTTP integrations, not
			// this presentation fixture's generic communication payload.
			if tool == factory.MCPToolRecordDiagnosis {
				tool = factory.MCPToolPartial
			} else if tool == factory.MCPToolDraftRepair {
				tool = factory.MCPToolSuggest
			}
			profile := factory.MCPProfileBuilder
			if component.Roles[0] != "*" {
				profile = factory.MCPProfile(component.Roles[0])
			}
			if !factory.MCPToolAdmitted(profile, tool) {
				t.Fatalf("tool %s not admitted for %s", tool, profile)
			}
			cap := testMCPCapabilityRow()
			cap.Profile = profile
			a := journeyAssignment(cap.AssignmentID, factory.NewID(), "d"+strings.Repeat("a", 24), 7, int64(index+1), true)
			a.Phase = factory.MCPPhaseOf(profile)
			insertJourneyAssignment(t, sqlDB, a, "assigned", 0)
			cap = activateMCPCapability(t, ctx, s, cap)
			content := map[string]any{"component_id": component.ID, "title": component.Label, "summary": "A concrete worker observation.", "question": "Which outcome should this issue deliver?", "severity": "warning", "scope": "project", "evidence_refs": []string{"src/main.go"}, "resolution": "Supply missing evidence", "completed": []string{"Reviewed the source"}, "remaining": []string{"Apply the change"}, "attempts": []string{"Read the source"}, "decision": "Use the existing interface", "rationale": "It matches the accepted request", "alternatives": []string{"Add a new interface"}}
			if tool == factory.MCPToolReply {
				id, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "Context for the reply", IdempotencyKey: factory.NewID()}, "7", "Owner", 1)
				if err != nil {
					t.Fatal(err)
				}
				content["reply_to"] = id
			}
			if factory.MCPIsTerminal(tool) {
				content = map[string]any{"component_id": component.ID, "outcome": "incomplete", "cause": "More evidence required"}
			}
			raw, err := json.Marshal(content)
			if err != nil {
				t.Fatal(err)
			}
			submission := testMCPSubmissionRow(cap, tool, raw)
			receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
			if err != nil {
				t.Fatal(err)
			}
			if factory.MCPIsTerminal(tool) {
				details, err := s.FeedRecordDetails(ctx, []string{a.ID})
				if err != nil {
					t.Fatal(err)
				}
				if details[a.ID].Payload == nil || details[a.ID].Payload.ComponentID != component.ID {
					t.Fatalf("terminal presentation missing: %+v", details[a.ID])
				}
			} else {
				post, err := s.FeedComment(ctx, receipt.FeedPostID)
				if err != nil {
					t.Fatal(err)
				}
				if post.Payload == nil || post.Payload.Communication.ComponentID != component.ID || post.Actor != string(profile) {
					t.Fatalf("worker presentation missing: %+v", post)
				}
			}
			count++
		})
	}
	if count != 102 {
		t.Fatalf("expected all 102 worker variants, exercised %d", count)
	}
}
