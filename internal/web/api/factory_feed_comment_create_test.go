package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// stubIssueGraph is the in-memory control.IssueGraphExecutor: one
// recorded operation with its receipt, plus the owner gate.
type stubIssueGraph struct {
	revision  int64
	ownerErr  error
	submitErr error
	ops       map[string]sdk.OperationRecord
	lookups   int
	submitted []factory.IntentGeneration
}

func (s *stubIssueGraph) CheckIntentOwner(context.Context, int64, int64, int64) (int64, error) {
	if s.ownerErr != nil {
		return 0, s.ownerErr
	}
	return s.revision, nil
}

func (s *stubIssueGraph) SubmitGraph(_ context.Context, g factory.IntentGeneration, repository int64) (sdk.OperationRecord, error) {
	if s.submitErr != nil {
		return sdk.OperationRecord{}, s.submitErr
	}
	s.submitted = append(s.submitted, g)
	if record, ok := s.ops[g.OperationID]; ok {
		return record, nil
	}
	receipt, _ := json.Marshal(sdk.IssueGraphReceipt{
		ActorID: g.ActorID, RepositoryID: repository,
		Issues: []sdk.IssueGraphIssueReceipt{{Key: "c" + strings.Repeat("0", 31), IssueID: 9, IssueIndex: 42}},
	})
	record := sdk.OperationRecord{
		OperationID: g.OperationID, ActorID: "7", RepositoryID: "3",
		Kind: string(sdk.OperationKindIssueGraph), Receipt: receipt,
	}
	if s.ops == nil {
		s.ops = map[string]sdk.OperationRecord{}
	}
	s.ops[g.OperationID] = record
	return record, nil
}

func (s *stubIssueGraph) LookupGraph(_ context.Context, g factory.IntentGeneration, _ int64) (sdk.OperationLookup, error) {
	s.lookups++
	record, ok := s.ops[g.OperationID]
	if !ok {
		return sdk.OperationLookup{OperationID: g.OperationID}, nil
	}
	return sdk.OperationLookup{OperationID: g.OperationID, Record: &record}, nil
}

func (s *stubIssueGraph) CancelGraph(context.Context, string) (sdk.OperationRecord, error) {
	return sdk.OperationRecord{}, errors.New("unused")
}

func (s *stubIssueGraph) ReadGeneratedIssue(context.Context, int64, int64, int64) (sdk.NativeSnapshot, int64, error) {
	return sdk.NativeSnapshot{}, 0, errors.New("unused")
}

type stubCommentStore struct {
	author string
	err    error
}

func (s *stubCommentStore) FeedComment(context.Context, string) (store.FeedCommentDetail, error) {
	if s.err != nil {
		return store.FeedCommentDetail{}, s.err
	}
	return store.FeedCommentDetail{Record: factory.CommentRecord{Author: s.author}}, nil
}

func validCreateDraft(comment string) factory.IssueDraft {
	return factory.IssueDraft{
		Comment: comment, Revision: 1, Title: "Add unfinished filter",
		Body: "Filter the checklist to unfinished tasks.", Objective: "reviewable list filtering",
		Acceptance: "unfinished filter shows only open tasks", Metadata: `{"work_type":"code"}`,
	}
}

func TestGraphCommentIssueCreator(t *testing.T) {
	ctx := context.Background()
	comment := "c" + strings.Repeat("0", 31)

	t.Run("create validates owner gates and resolves the index", func(t *testing.T) {
		graph := &stubIssueGraph{revision: 12}
		creator := &GraphCommentIssueCreator{Store: &stubCommentStore{author: "5"}, Graph: graph, Actor: 7}
		draft := validCreateDraft(comment)
		index, err := creator.CreateCommentIssue(ctx, draft, 3)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if index != 42 {
			t.Fatalf("index = %d, want 42", index)
		}
		if len(graph.submitted) != 1 {
			t.Fatalf("submits = %d", len(graph.submitted))
		}
		got := graph.submitted[0]
		if got.OperationID != "commentissue-"+comment || got.ActorID != 7 || got.NativeRev != 12 {
			t.Fatalf("operation %+v", got)
		}
		var payload sdk.IssueGraphPayload
		if err := json.Unmarshal(got.Payload, &payload); err != nil || len(payload.Creates) != 1 {
			t.Fatalf("payload %s", got.Payload)
		}
		create := payload.Creates[0]
		if create.Key != comment || create.Title != draft.Title || !strings.Contains(create.Body, "Soda-Comment: "+comment) {
			t.Fatalf("create %+v", create)
		}
		// Lost effect reply reconciles through lookup, not a second submit.
		found, err := creator.LookupCommentIssue(ctx, 3, comment)
		if err != nil || found != 42 {
			t.Fatalf("lookup = %d, err = %v", found, err)
		}
		if len(graph.submitted) != 1 {
			t.Fatal("lookup submitted again")
		}
	})

	t.Run("unknown comment looks up zero", func(t *testing.T) {
		creator := &GraphCommentIssueCreator{Store: &stubCommentStore{author: "5"}, Graph: &stubIssueGraph{revision: 12}, Actor: 7}
		found, err := creator.LookupCommentIssue(ctx, 3, "d"+strings.Repeat("1", 31))
		if err != nil || found != 0 {
			t.Fatalf("lookup = %d, err = %v", found, err)
		}
	})

	t.Run("malformed draft refuses before any native effect", func(t *testing.T) {
		graph := &stubIssueGraph{revision: 12}
		creator := &GraphCommentIssueCreator{Store: &stubCommentStore{author: "5"}, Graph: graph, Actor: 7}
		broken := validCreateDraft(comment)
		broken.Title = ""
		if _, err := creator.CreateCommentIssue(ctx, broken, 3); err == nil {
			t.Fatal("malformed draft admitted")
		}
		if len(graph.submitted) != 0 {
			t.Fatal("malformed draft reached native")
		}
	})

	t.Run("owner gate refusal spends nothing native", func(t *testing.T) {
		graph := &stubIssueGraph{ownerErr: errors.New("no code-write")}
		creator := &GraphCommentIssueCreator{Store: &stubCommentStore{author: "5"}, Graph: graph, Actor: 7}
		if _, err := creator.CreateCommentIssue(ctx, validCreateDraft(comment), 3); err == nil {
			t.Fatal("owner refusal admitted")
		}
		if len(graph.submitted) != 0 {
			t.Fatal("refused owner submitted")
		}
	})

	t.Run("unwired creator stays unavailable", func(t *testing.T) {
		creator := &GraphCommentIssueCreator{}
		if _, err := creator.CreateCommentIssue(ctx, validCreateDraft(comment), 3); err == nil {
			t.Fatal("unwired create admitted")
		}
		if _, err := creator.LookupCommentIssue(ctx, 3, comment); err == nil {
			t.Fatal("unwired lookup admitted")
		}
	})
}
