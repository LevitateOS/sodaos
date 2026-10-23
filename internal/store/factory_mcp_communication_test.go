package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestMCPCommunicationPersistsFeedAndReplay(t *testing.T) {
	s, dsn := postgresFixture(t, nil)
	ctx := context.Background()
	cap := testMCPCapabilityRow()
	assignment := journeyAssignment(cap.AssignmentID, strings.Repeat("1", 32), "d"+strings.Repeat("a", 24), cap.Repository, 42, true)
	insertJourneyAssignment(t, journeySQL(t, dsn), assignment, "assigned", 1)
	cap = activateMCPCapability(t, ctx, s, cap)
	payload := []byte(`{"title":"Who can view a task?","summary":"The owner policy needs clarification.","question":"May team members view other people's tasks?","options":["Only owners","All team members"]}`)
	submission := testMCPSubmissionRow(cap, factory.MCPToolAsk, payload)
	receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
	if err != nil {
		t.Fatal(err)
	}
	if !factory.ValidID(receipt.FeedPostID) {
		t.Fatalf("missing feed identity: %+v", receipt)
	}
	replay, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
	if err != nil || replay != receipt {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	page, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != receipt.FeedPostID {
		t.Fatalf("feed order/replay: %+v", page)
	}
	details, err := s.FeedPostDetails(ctx, []string{receipt.FeedPostID})
	if err != nil {
		t.Fatal(err)
	}
	detail := details[receipt.FeedPostID]
	if detail.Actor != "builder" || detail.AuthorName != "Builder" || detail.Assignment != cap.AssignmentID || detail.Record.Issue != 42 || detail.Status != "ask" || detail.Payload.Communication.Question == "" {
		t.Fatalf("lost bound metadata/content: %+v", detail)
	}

	if detail.Payload.Communication.QuestionState != "open" {
		t.Fatal("new question is not open")
	}
	if _, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "The intended policy is owner-only.", ReplyTo: receipt.FeedPostID, IdempotencyKey: "question-reply"}, "7", "Owner", 1); err != nil {
		t.Fatal(err)
	}
	question, err := s.FeedComment(ctx, receipt.FeedPostID)
	if err != nil || question.Payload.Communication.QuestionState != "replied" {
		t.Fatalf("human reply not reflected: %+v %v", question, err)
	}
	// Answer uses the same authentic identity and is a durable thread reply.
	answer := testMCPSubmissionRow(cap, factory.MCPToolAnswer, []byte(`{"title":"Policy clarified","summary":"Only task owners may view their tasks.","reply_to":"`+receipt.FeedPostID+`"}`))
	answered, err := s.RecordMCPSubmission(ctx, answer, testMCPRecordedReceipt(answer))
	if err != nil {
		t.Fatal(err)
	}
	replies, _, err := s.FeedReplyIDs(ctx, 7, "post", receipt.FeedPostID, 0)
	if err != nil || len(replies) != 2 || replies[0] != answered.FeedPostID {
		t.Fatalf("replies=%v err=%v", replies, err)
	}
	question, err = s.FeedComment(ctx, receipt.FeedPostID)
	if err != nil || question.Payload.Communication.QuestionState != "answered" {
		t.Fatalf("explicit answer not reflected: %+v %v", question, err)
	}
	// No model/judgment request is invented for an authored worker report.
	if _, err = s.CommentPipeline(ctx, receipt.FeedPostID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("worker communication opened pipeline: %v", err)
	}
	// Unknown/spoofed identity fields refuse atomically without a receipt or post.
	bad := testMCPSubmissionRow(cap, factory.MCPToolAsk, []byte(`{"title":"Spoof","summary":"Try another identity","question":"Who?","actor":"inspector"}`))
	if _, err = s.RecordMCPSubmission(ctx, bad, testMCPRecordedReceipt(bad)); err == nil {
		t.Fatal("spoofed actor admitted")
	}
	if _, err = s.MCPReceipt(ctx, cap.AssignmentID, bad.SubmissionID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("refused report saved receipt: %v", err)
	}
	// Caller-selected Factory effects cannot become trusted cards.
	bad = testMCPSubmissionRow(cap, factory.MCPToolPartial, []byte(`{"title":"Merged","summary":"An untrusted report","completed":["claimed merge"],"component_id":"F04"}`))
	if _, err = s.RecordMCPSubmission(ctx, bad, testMCPRecordedReceipt(bad)); err == nil {
		t.Fatal("controller-owned component admitted")
	}
}

func TestMCPCommunicationBoundPreservesFinish(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	cap := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	payload := []byte(`{"title":"Progress","summary":"Working on the assigned task.","remaining":["Finish the task"]}`)
	for i := 0; i < factory.MCPMaxCommunications; i++ {
		submission := testMCPSubmissionRow(cap, factory.MCPToolPartial, payload)
		if _, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission)); err != nil {
			t.Fatal(err)
		}
	}
	over := testMCPSubmissionRow(cap, factory.MCPToolPartial, payload)
	if _, err := s.RecordMCPSubmission(ctx, over, testMCPRecordedReceipt(over)); !errors.Is(err, ErrMCPCommunicationCap) {
		t.Fatalf("communication bound=%v", err)
	}
	terminal := testMCPSubmissionRow(cap, factory.MCPToolFinishIncomplete, []byte(`{"outcome":"blocked","cause":"question","evidence_refs":[],"affected_task_ids":[],"requested_resolution":"Need owner input"}`))
	if _, err := s.RecordMCPSubmission(ctx, terminal, testMCPRecordedReceipt(terminal)); err != nil {
		t.Fatalf("communications consumed terminal capacity: %v", err)
	}
	if _, err := s.RecordMCPSubmission(ctx, testMCPSubmissionRow(cap, factory.MCPToolPartial, payload), factory.MCPReceipt{}); err == nil {
		t.Fatal("invalid receipt admitted")
	}
}

func TestMCPCommunicationEveryOutlet(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	cap := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	question := ""
	for _, tool := range factory.MCPCommunicationTools {
		payload := map[string]any{"title": "Worker update", "summary": "A concrete observation about the assigned work."}
		switch tool {
		case factory.MCPToolAsk:
			payload["question"] = "Which policy applies?"
		case factory.MCPToolAnswer, factory.MCPToolReply:
			payload["reply_to"] = question
		case factory.MCPToolFlag:
			payload["severity"] = "warning"
		case factory.MCPToolLearned:
			payload["scope"] = "Task access policy"
			payload["evidence_refs"] = []string{"src/access.go:12"}
		case factory.MCPToolSuggest:
			payload["rationale"] = "It reduces duplicated checks."
		case factory.MCPToolPartial:
			payload["completed"] = []string{"Reviewed the access rules"}
		case factory.MCPToolBlocked:
			payload["resolution"] = "Confirm the intended task access policy."
		case factory.MCPToolStuck:
			payload["attempts"] = []string{"Checked existing tests; they do not cover this case"}
		case factory.MCPToolRecordDecision:
			payload["decision"] = "Use the existing access helper"
			payload["rationale"] = "Keep the same policy across routes."
		}
		raw, _ := json.Marshal(payload)
		submission := testMCPSubmissionRow(cap, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		if tool == factory.MCPToolAsk {
			question = receipt.FeedPostID
		}
		detail, err := s.FeedComment(ctx, receipt.FeedPostID)
		if err != nil || detail.Payload == nil || detail.Payload.Communication.Kind != strings.TrimPrefix(string(tool), "factory.") {
			t.Fatalf("%s absent from stored feed: %+v %v", tool, detail, err)
		}
	}
}

func TestMCPFinishComponentUsesCanonicalRecord(t *testing.T) {
	s, dsn := postgresFixture(t, nil)
	ctx := context.Background()
	cap := testMCPCapabilityRow()
	a := journeyAssignment(cap.AssignmentID, strings.Repeat("1", 32), "d"+strings.Repeat("a", 24), 7, 42, true)
	a.Phase = "implement"
	insertJourneyAssignment(t, journeySQL(t, dsn), a, "assigned", 1)
	cap = activateMCPCapability(t, ctx, s, cap)
	submission := testMCPSubmissionRow(cap, factory.MCPToolFinishIncomplete, []byte(`{"component_id":"S12","outcome":"blocked","cause":"question","evidence_refs":[],"affected_task_ids":[],"requested_resolution":"Need owner input"}`))
	if _, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission)); err != nil {
		t.Fatal(err)
	}
	details, err := s.FeedRecordDetails(ctx, []string{a.ID})
	if err != nil {
		t.Fatal(err)
	}
	detail := details[a.ID]
	if detail.Payload == nil || detail.Payload.ComponentID != "S12" {
		t.Fatalf("finish presentation missing: %+v", detail)
	}
	if strings.Contains(detail.Status, "completed") || strings.Contains(detail.Status, "verified") {
		t.Fatalf("submission invented completion: %s", detail.Status)
	}
	page, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10})
	if err != nil || len(page) != 1 {
		t.Fatalf("finish duplicated canonical record: %+v %v", page, err)
	}
}

// TestPostResolutionDispositionPersistsAndExposes proves the R32.2 and
// R32.3 disposition route against real PostgreSQL: a recorded answer
// incorporates, moving its question past answered; a suggestion
// applies with its reason and refuses twice; refused resolutions must
// explain themselves; questions, status signals and missing posts
// refuse; dispositions surface on discussion reads beside the
// immutable posts; and recording partial, suggest or answer finishes
// nothing and grants no readiness.
func TestPostResolutionDispositionPersistsAndExposes(t *testing.T) {
	s, dsn := postgresFixture(t, nil)
	ctx := context.Background()
	cap := testMCPCapabilityRow()
	assignment := journeyAssignment(cap.AssignmentID, strings.Repeat("1", 32), "d"+strings.Repeat("a", 24), cap.Repository, 42, true)
	insertJourneyAssignment(t, journeySQL(t, dsn), assignment, "assigned", 1)
	cap = activateMCPCapability(t, ctx, s, cap)
	post := func(tool factory.MCPTool, payload string) factory.MCPReceipt {
		t.Helper()
		submission := testMCPSubmissionRow(cap, tool, []byte(payload))
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		if !factory.ValidID(receipt.FeedPostID) {
			t.Fatalf("%s recorded no post: %+v", tool, receipt)
		}
		return receipt
	}
	ask := post(factory.MCPToolAsk, `{"title":"Who can view a task?","summary":"The owner policy needs clarification.","question":"May team members view other people's tasks?"}`)
	answer := post(factory.MCPToolAnswer, `{"title":"Policy clarified","summary":"Only task owners may view their tasks.","reply_to":"`+ask.FeedPostID+`"}`)
	question, err := s.FeedComment(ctx, ask.FeedPostID)
	if err != nil || question.Payload.Communication.QuestionState != "answered" {
		t.Fatalf("answer not reflected: %+v %v", question, err)
	}
	resolve := func(payload string) error {
		t.Helper()
		submission := testMCPSubmissionRow(cap, factory.MCPToolResolvePost, []byte(payload))
		_, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		return err
	}
	if err := resolve(`{"post":"` + answer.FeedPostID + `","resolution":"incorporated"} {}`); err == nil {
		t.Fatal("trailing resolution object accepted")
	}
	if err := resolve(`{"post":"` + answer.FeedPostID + `","resolution":"incorporated"}`); err != nil {
		t.Fatalf("incorporate: %v", err)
	}
	adopted, err := s.FeedComment(ctx, answer.FeedPostID)
	if err != nil || adopted.Payload.Communication.Disposition != "incorporated" {
		t.Fatalf("incorporation not exposed: %+v %v", adopted, err)
	}
	question, err = s.FeedComment(ctx, ask.FeedPostID)
	if err != nil || question.Payload.Communication.QuestionState != "incorporated" {
		t.Fatalf("question hid its adoption: %+v %v", question, err)
	}
	suggest := post(factory.MCPToolSuggest, `{"title":"Cache the policy","summary":"Reading the policy file on every task is slow.","rationale":"Cache ownership at assignment start."}`)
	if err := resolve(`{"post":"` + suggest.FeedPostID + `","resolution":"applied","reason":"Cached at dispatch"}`); err != nil {
		t.Fatalf("apply: %v", err)
	}
	applied, err := s.FeedComment(ctx, suggest.FeedPostID)
	if err != nil || applied.Payload.Communication.Disposition != "applied" || applied.Payload.Communication.DispositionReason != "Cached at dispatch" {
		t.Fatalf("applied disposition lost: %+v %v", applied, err)
	}
	if err := resolve(`{"post":"` + suggest.FeedPostID + `","resolution":"refused","reason":"Second thoughts"}`); err == nil {
		t.Fatal("second disposition accepted")
	}
	other := post(factory.MCPToolSuggest, `{"title":"Skip the cache","summary":"Caching adds staleness risk.","rationale":"Read the file every time."}`)
	if err := resolve(`{"post":"` + other.FeedPostID + `","resolution":"refused"}`); err == nil {
		t.Fatal("unexplained refusal accepted")
	}
	if err := resolve(`{"post":"` + other.FeedPostID + `","resolution":"incorporated"}`); err == nil {
		t.Fatal("suggestion incorporated")
	}
	if err := resolve(`{"post":"` + answer.FeedPostID + `","resolution":"applied","reason":"Wrong kind"}`); err == nil {
		t.Fatal("answer applied")
	}
	if err := resolve(`{"post":"` + ask.FeedPostID + `","resolution":"applied","reason":"Answering is not resolving"}`); err == nil {
		t.Fatal("question resolved")
	}
	partial := post(factory.MCPToolPartial, `{"title":"Halfway","summary":"First half done.","completed":["Read the policy"],"remaining":["Apply it"]}`)
	if err := resolve(`{"post":"` + partial.FeedPostID + `","resolution":"applied","reason":"Status is not a proposal"}`); err == nil {
		t.Fatal("status signal resolved")
	}
	if err := resolve(`{"post":"` + factory.NewID() + `","resolution":"applied","reason":"Ghost post"}`); err == nil {
		t.Fatal("missing post resolved")
	}
	// Recording questions, answers, suggestions, partials and their
	// dispositions finishes nothing: the assignment carries no result
	// and no readiness path runs outside assignment finish.
	still, err := s.Assignment(ctx, assignment.ID)
	if err != nil || still.Result != nil {
		t.Fatalf("discussion traffic finished the role: %+v %v", still, err)
	}
	again, err := s.FeedComment(ctx, other.FeedPostID)
	if err != nil || again.Payload.Communication.Disposition != "" {
		t.Fatalf("refused dispositions leaked state: %+v %v", again, err)
	}
}
