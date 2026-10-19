package api

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Fake-transport authoring tests: the intent store and planning
// transport are stubbed, so no enrolled model credentials or paid
// calls are involved. The transport's DispatchIntentPlan records the
// settled attempt the way a finished host run would leave it.

type fakeDraftStore struct {
	head    *factory.IntentRevision
	attempt *factory.Assignment
	saves   int
}

func (f *fakeDraftStore) IntentRevision(_ context.Context, intent string, owner int64) (factory.IntentRevision, error) {
	if f.head == nil || f.head.Intent != intent || f.head.Owner != owner {
		return factory.IntentRevision{}, sql.ErrNoRows
	}
	return *f.head, nil
}

func (f *fakeDraftStore) SaveIntentRevision(_ context.Context, r factory.IntentRevision) error {
	if err := r.Validate(); err != nil {
		return err
	}
	f.saves++
	saved := r
	f.head = &saved
	return nil
}

func (f *fakeDraftStore) IntentPlanAttempt(_ context.Context, intentID, revisionID string) (factory.Assignment, error) {
	if f.attempt == nil || f.attempt.Subject.Intent == nil ||
		f.attempt.Subject.Intent.IntentID != intentID || f.attempt.Subject.Intent.RevisionID != revisionID {
		return factory.Assignment{}, store.ErrNotFound
	}
	return *f.attempt, nil
}

type fakeDraftTransport struct {
	store     *fakeDraftStore
	dispatch  int
	assemble  int
	finishRun bool
	output    factory.SemanticPlannerOutput
}

func (f *fakeDraftTransport) AssembleIntentPlanningInput(_ context.Context, head factory.IntentRevision, operation string) (factory.SemanticPlannerInput, project.Preparation, *control.DispatchWait, error) {
	f.assemble++
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		return factory.SemanticPlannerInput{}, project.Preparation{}, nil, err
	}
	in := factory.SemanticPlannerInput{
		Operation: operation, Head: head, Content: content,
		Alternatives: factory.DescribeAlternatives(head.Objective),
		Inspected: []factory.InspectedSource{{
			Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40),
			Evidence: "fake transport investigation",
		}},
	}
	return in, project.Preparation{ID: "prep", SourceCommit: strings.Repeat("c", 40)}, nil, nil
}

func (f *fakeDraftTransport) DispatchIntentPlan(_ context.Context, intentID string, owner int64, operation string, in factory.SemanticPlannerInput, source, preparationID string) control.DispatchReport {
	f.dispatch++
	head := *f.store.head
	prompt, err := control.StageIntentPlanningPrompt(head, in)
	if err != nil {
		return control.DispatchReport{Errors: []control.DispatchError{{Detail: err.Error()}}}
	}
	attempt := factory.Assignment{
		ID: factory.NewID(), Prompt: prompt, Stage: factory.AssignmentAssigned,
		Subject: factory.ExecutionSubject{
			Kind: factory.SubjectIntent,
			Intent: &factory.IntentSubject{
				IntentID: intentID, RevisionID: head.ID,
				ScopeDigest: head.Digest(), OwnerID: owner,
			},
		},
	}
	if f.finishRun {
		out := f.output
		attempt.Stage = factory.AssignmentFinished
		attempt.Outcome = factory.Succeeded
		attempt.Result = &factory.AssignmentResult{
			Reported: true,
			PhaseResult: &factory.PhaseResult{
				Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose,
				Status: "completed", Summary: "planned",
				Output: &factory.PhaseOutput{Decomposition: &out},
			},
		}
	}
	f.store.attempt = &attempt
	return control.DispatchReport{}
}

func draftTestComment() store.FeedCommentDetail {
	return store.FeedCommentDetail{
		Record: factory.CommentRecord{
			ID: strings.Repeat("a", 32), Revision: 0, Repository: 7, Author: "42",
		},
		Body: "Add a filter for unfinished tasks",
		At:   1729000000,
	}
}

func draftTestOutput(comment store.FeedCommentDetail) factory.SemanticPlannerOutput {
	ideaID := commentDecisionID("idea", comment.Record.ID, comment.Record.Revision)
	return factory.SemanticPlannerOutput{
		Intent:    commentDecisionID("intent", comment.Record.ID, comment.Record.Revision),
		Revision:  commentDecisionID("revision", comment.Record.ID, comment.Record.Revision),
		Operation: factory.IntentPlanningPropose,
		Candidates: []factory.SemanticCandidate{{
			Key: "only", Rationale: "One bounded change covers the request.",
			Tasks: []factory.IntentTask{{
				Key:       commentDecisionID("task", comment.Record.ID, comment.Record.Revision),
				Objective: "Add an unfinished-only filter to the task list",
				Outcomes:  []string{"Filter shows only unfinished tasks", "Clearing the filter restores the full list"},
				IdeaIDs:   []string{ideaID},
			}},
		}},
	}
}

func TestIntentCommentDrafterAuthorsAndReplays(t *testing.T) {
	comment := draftTestComment()
	st := &fakeDraftStore{}
	transport := &fakeDraftTransport{store: st, finishRun: true, output: draftTestOutput(comment)}
	drafter := &IntentCommentDrafter{Store: st, Transport: transport}

	draft, err := drafter.DraftCommentIssue(context.Background(), comment)
	if err != nil {
		t.Fatal(err)
	}
	if err := draft.Validate(); err != nil {
		t.Fatalf("authored draft invalid: %v", err)
	}
	if draft.Title != "Add an unfinished-only filter to the task list" ||
		draft.Objective != draft.Title ||
		!strings.Contains(draft.Acceptance, "Filter shows only unfinished tasks") ||
		!strings.Contains(draft.Body, "One bounded change covers the request.") ||
		!strings.Contains(draft.Metadata, `"source":"feed_comment"`) {
		t.Fatalf("draft mapping lost planner content: %+v", draft)
	}
	if st.saves != 1 || transport.dispatch != 1 {
		t.Fatalf("saves=%d dispatches=%d, want 1/1", st.saves, transport.dispatch)
	}
	if st.head == nil || st.head.MaxIssues != 1 || st.head.State != "draft" {
		t.Fatalf("comment intent not bounded: %+v", st.head)
	}
	// Replay reuses the same intent and recorded attempt: no new save,
	// no second funded dispatch, identical draft.
	again, err := drafter.DraftCommentIssue(context.Background(), comment)
	if err != nil {
		t.Fatal(err)
	}
	if again != draft {
		t.Fatalf("replay draft differs:\n%+v\n%+v", again, draft)
	}
	if st.saves != 1 || transport.dispatch != 1 || transport.assemble != 1 {
		t.Fatalf("replay spent again: saves=%d assemble=%d dispatch=%d", st.saves, transport.assemble, transport.dispatch)
	}
}

func TestIntentCommentDrafterAttentionRefuses(t *testing.T) {
	comment := draftTestComment()
	st := &fakeDraftStore{}
	out := draftTestOutput(comment)
	out.Candidates = nil
	out.Attention = "Which list should the filter apply to?"
	transport := &fakeDraftTransport{store: st, finishRun: true, output: out}
	drafter := &IntentCommentDrafter{Store: st, Transport: transport}
	if _, err := drafter.DraftCommentIssue(context.Background(), comment); err == nil ||
		!strings.Contains(err.Error(), "product decision") {
		t.Fatalf("attention err = %v", err)
	}
}

func TestIntentCommentDrafterMultiCandidateRefuses(t *testing.T) {
	comment := draftTestComment()
	st := &fakeDraftStore{}
	out := draftTestOutput(comment)
	second := out.Candidates[0]
	second.Key = "other"
	out.Candidates = append(out.Candidates, second)
	transport := &fakeDraftTransport{store: st, finishRun: true, output: out}
	drafter := &IntentCommentDrafter{Store: st, Transport: transport}
	if _, err := drafter.DraftCommentIssue(context.Background(), comment); err == nil ||
		!strings.Contains(err.Error(), "funded judgment") {
		t.Fatalf("multi-candidate err = %v", err)
	}
}

func TestIntentCommentDrafterPendingAttemptResumesWithoutRedispatch(t *testing.T) {
	comment := draftTestComment()
	st := &fakeDraftStore{}
	transport := &fakeDraftTransport{store: st, finishRun: false, output: draftTestOutput(comment)}
	drafter := &IntentCommentDrafter{Store: st, Transport: transport}
	// A frozen clock would make the old polling implementation wait forever.
	drafter.Now = func() time.Time { return time.Unix(comment.At, 0) }
	if _, err := drafter.DraftCommentIssue(context.Background(), comment); !errors.Is(err, ErrCommentDraftPending) {
		t.Fatalf("pending err = %v", err)
	}
	if transport.dispatch != 1 {
		t.Fatalf("dispatches=%d, want the one launched attempt", transport.dispatch)
	}
	st.attempt.Stage = factory.AssignmentFinished
	st.attempt.Outcome = factory.Succeeded
	out := draftTestOutput(comment)
	st.attempt.Result = &factory.AssignmentResult{
		Reported: true,
		PhaseResult: &factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose,
			Status: "completed", Summary: "planned",
			Output: &factory.PhaseOutput{Decomposition: &out},
		},
	}
	draft, err := drafter.DraftCommentIssue(context.Background(), comment)
	if err != nil {
		t.Fatal(err)
	}
	if err := draft.Validate(); err != nil {
		t.Fatal(err)
	}
	if st.saves != 1 || transport.dispatch != 1 || transport.assemble != 1 {
		t.Fatalf("resume spent again: saves=%d assemble=%d dispatch=%d", st.saves, transport.assemble, transport.dispatch)
	}
}

func TestCommentDraftIntentDeterministicAndBounded(t *testing.T) {
	comment := draftTestComment()
	first, err := commentDraftIntent(comment, 42, comment.At)
	if err != nil {
		t.Fatal(err)
	}
	second, err := commentDraftIntent(comment, 42, comment.At)
	if err != nil {
		t.Fatal(err)
	}
	if first.Intent != second.Intent || first.ID != second.ID || first.Ideas[0].ID != second.Ideas[0].ID {
		t.Fatal("comment intent identities are not deterministic")
	}
	if first.MaxIssues != 1 || len(first.EffectiveOutcomes) != 1 {
		t.Fatalf("comment intent not single-issue bounded: %+v", first)
	}
	oversize := comment
	oversize.Body = strings.Repeat("x", 4001)
	if _, err := commentDraftIntent(oversize, 42, comment.At); err == nil {
		t.Fatal("oversize comment body admitted into planning idea")
	}
}

// TestCommentDraftIntentCarriesSourceContext proves the planner
// receives the request inside its actual work context: the resolved
// source snapshot becomes one pending evidence idea beside the human
// request, never an effective outcome.
func TestCommentDraftIntentCarriesSourceContext(t *testing.T) {
	comment := draftTestComment()
	comment.Source = &factory.CommentSource{
		Kind: factory.CommentSourceRecord, ID: strings.Repeat("b", 32),
		Issue: 55, Status: "blocked", Summary: "Built the task filter",
	}
	head, err := commentDraftIntent(comment, 42, comment.At)
	if err != nil {
		t.Fatal(err)
	}
	if len(head.Ideas) != 2 {
		t.Fatalf("ideas %+v, want request plus source context", head.Ideas)
	}
	context := head.Ideas[1]
	if context.Decision != "pending" || context.Provenance != "autonomous" ||
		!strings.Contains(context.Description, "issue #55") ||
		!strings.Contains(context.Description, "Built the task filter") {
		t.Fatalf("source context idea: %+v", context)
	}
	if len(head.EffectiveOutcomes) != 1 || head.EffectiveOutcomes[0] != head.Ideas[0].ID {
		t.Fatalf("source context became an effective outcome: %+v", head.EffectiveOutcomes)
	}
	// Without a source, the intent is exactly the single request idea.
	bare, err := commentDraftIntent(draftTestComment(), 42, comment.At)
	if err != nil {
		t.Fatal(err)
	}
	if len(bare.Ideas) != 1 {
		t.Fatalf("source-less ideas %+v", bare.Ideas)
	}
}

func TestCommentDraftTitleBounds(t *testing.T) {
	long := strings.Repeat("word ", 80)
	title := commentDraftTitle(long)
	if len(title) > 256 || strings.HasSuffix(title, " ") {
		t.Fatalf("title out of bound: %d %q", len(title), title[len(title)-8:])
	}
	if got := commentDraftTitle("short objective"); got != "short objective" {
		t.Fatalf("short title = %q", got)
	}
}

func TestCommentCollaboratorsBindDrafter(t *testing.T) {
	s := &API{Config: &config.Config{}, Store: &store.Store{}, Coordinator: &control.Coordinator{}}
	wired := s.commentCollaborators()
	if wired.Drafter == nil {
		t.Fatal("drafter stayed nil behind a live coordinator")
	}
	if _, ok := wired.Drafter.(*IntentCommentDrafter); !ok {
		t.Fatalf("drafter type %T", wired.Drafter)
	}
	bare := (&API{Config: &config.Config{}}).commentCollaborators()
	if bare.Drafter != nil {
		t.Fatal("drafter bound without a coordinator or store")
	}
}
