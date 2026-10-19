package factory

import (
	"strings"
	"testing"
)

func TestCommentRequestValidation(t *testing.T) {
	good := CommentRequest{Repository: 7, Body: "please fix the login redirect", IdempotencyKey: "key-1"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []CommentRequest{
		{Body: "x", IdempotencyKey: "k"},
		{Repository: 7, IdempotencyKey: "k"},
		{Repository: 7, Body: strings.Repeat("x", CommentBodyLimit+1), IdempotencyKey: "k"},
		{Repository: 7, Body: "x", Issue: -1, IdempotencyKey: "k"},
		{Repository: 7, Body: "x", ReplyTo: "nope", IdempotencyKey: "k"},
		{Repository: 7, Body: "x"},
	} {
		if err := bad.Validate(); err == nil {
			t.Fatalf("admitted %+v", bad)
		}
	}
}

func TestCommentPipelineAdmissionGates(t *testing.T) {
	fresh := CommentPipeline{Comment: strings.Repeat("a", 32), Intake: CommentIntakeRecorded, Judgment: CommentJudgmentPending, Draft: CommentDraftPending, Effect: CommentEffectPending}
	if err := fresh.Validate(); err != nil {
		t.Fatal(err)
	}
	if !fresh.AdmitCommentJudgment() {
		t.Fatal("fresh pipeline refuses judgment")
	}
	if fresh.AdmitCommentDraft() || fresh.AdmitCommentEffect() || fresh.CommentEffectSettled() {
		t.Fatal("fresh pipeline admits a later stage")
	}
	judged := fresh
	judged.Judgment = CommentJudgmentApplied
	judged.Route = CommentRouteNewIssue
	if !judged.AdmitCommentDraft() {
		t.Fatal("judged new_issue pipeline refuses draft")
	}
	discussed := fresh
	discussed.Judgment = CommentJudgmentApplied
	discussed.Route = CommentRouteDiscussion
	if discussed.AdmitCommentDraft() || discussed.AdmitCommentEffect() {
		t.Fatal("discussion route admits work stages")
	}
	drafted := judged
	drafted.Draft = CommentDraftApplied
	if !drafted.AdmitCommentEffect() {
		t.Fatal("drafted pipeline refuses effect")
	}
	if drafted.CommentEffectSettled() {
		t.Fatal("pending effect reports settled")
	}
	for _, effect := range []string{CommentEffectVerified, CommentEffectApplied, CommentEffectBlocked} {
		settled := drafted
		settled.Effect = effect
		if !settled.CommentEffectSettled() || settled.AdmitCommentEffect() {
			t.Fatalf("effect %q mishandles settling", effect)
		}
	}
	spent := fresh
	spent.Attempts = CommentAttemptLimit
	if spent.AdmitCommentJudgment() {
		t.Fatal("exhausted pipeline admits judgment")
	}
	if err := spent.Validate(); err != nil {
		t.Fatal(err)
	}
	over := fresh
	over.Attempts = CommentAttemptLimit + 1
	if err := over.Validate(); err == nil {
		t.Fatal("over-limit attempts admitted")
	}
}

func TestIssueDraftValidation(t *testing.T) {
	good := IssueDraft{Comment: strings.Repeat("b", 32), Title: "Fix login redirect", Body: "Change the callback.", Objective: "Logins land back in the app.", Acceptance: "Redirect returns 302 to /.", Metadata: `{"work":"soda"}`}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	broken := good
	broken.Acceptance = ""
	if err := broken.Validate(); err == nil {
		t.Fatal("malformed draft admitted")
	}
	broken = good
	broken.Metadata = ""
	if err := broken.Validate(); err == nil {
		t.Fatal("metadata-free draft admitted")
	}
}

func TestFeedEntryEnrichmentValidates(t *testing.T) {
	entry := FeedEntry{Kind: FeedKindPost, ID: strings.Repeat("c", 32), Repository: 7, Seq: 9,
		Actor: "42", Author: "ada", Project: "acme/app", At: 1729000000, Status: "open"}
	if err := entry.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCommentSourceValidation(t *testing.T) {
	good := CommentRequest{Repository: 7, Body: "more filter", IdempotencyKey: "key-1",
		SourceKind: CommentSourceRecord, SourceID: strings.Repeat("a", 32)}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	broken := good
	broken.SourceID = ""
	if err := broken.Validate(); err == nil {
		t.Fatal("kind without identity admitted")
	}
	broken = good
	broken.SourceKind = "unknown"
	if err := broken.Validate(); err == nil {
		t.Fatal("unknown source kind admitted")
	}
	broken = good
	broken.SourceID = "not-an-id"
	if err := broken.Validate(); err == nil {
		t.Fatal("malformed source identity admitted")
	}
	source := CommentSource{Kind: CommentSourcePost, ID: strings.Repeat("b", 32), Issue: 55, Summary: "filter"}
	if err := source.Validate(); err != nil {
		t.Fatal(err)
	}
	source.Summary = strings.Repeat("x", CommentSourceSummaryLimit+1)
	if err := source.Validate(); err == nil {
		t.Fatal("over-limit source summary admitted")
	}
	candidate := CommentWorkCandidate{Issue: 55, Origin: CommentCandidateBound}
	if err := candidate.Validate(); err != nil {
		t.Fatal(err)
	}
	candidate.Origin = "guessed"
	if err := candidate.Validate(); err == nil {
		t.Fatal("unknown candidate origin admitted")
	}
}
