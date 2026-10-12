package forgejo

import (
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

// TestLifecyclePayloadJoin pins the Soda-built native payloads and kinds
// byte-for-byte against the L07 join capture
// (.coord/l07-consumer/captured-work.json, sha256 bd4b0a98) that the
// native join test consumes (Fountain 17fbd56ab3, 3/3 through the real
// writer). Builder output here must equal what the writer admitted there.
func TestLifecyclePayloadJoin(t *testing.T) {
	if factory.LifecycleKindIssue != sdk.OperationKindLifecycleIssue ||
		factory.LifecycleKindReview != sdk.OperationKindLifecycleReview ||
		factory.LifecycleKindNote != sdk.OperationKindLifecycleNote {
		t.Fatal("factory kinds diverged from SDK canon")
	}
	if factory.LifecycleEventKind(factory.LifecycleEventIssueCreated) != `issue.lifecycle.apply` ||
		factory.LifecycleEventKind(factory.LifecycleEventPRCreated) != `pull_request.review.request` ||
		factory.LifecycleEventKind(factory.LifecycleEventVerifiedMerge) != `issue.comment.publish` {
		t.Fatal("event->kind map diverged from capture")
	}
	issue, err := LifecycleIssuePayload(6, 3, 4)
	if err != nil || string(issue) != `{"issue_id":6,"label_id":3,"assignee_id":4}` {
		t.Fatalf("issue payload: %s %v", issue, err)
	}
	review, err := LifecycleReviewPayload(2, []int64{2})
	if err != nil || string(review) != `{"issue_id":2,"reviewer_ids":[2]}` {
		t.Fatalf("review payload: %s %v", review, err)
	}
	note, err := LifecycleNotePayload(5, "Factory completed the accepted work.", "pr/2@0123456789abcdef0123456789abcdef01234567")
	if err != nil || string(note) != `{"issue_id":5,"content":"Factory completed the accepted work.","merge_ref":"pr/2@0123456789abcdef0123456789abcdef01234567"}` {
		t.Fatalf("note payload: %s %v", note, err)
	}
}
