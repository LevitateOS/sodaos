package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

func reviewTestWork() factory.ReviewWork {
	return factory.ReviewWork{OperationID: "st10-review", AuthRevision: "attempt:review:1", Repository: 7, ActorID: 11, PRNumber: 5, PRID: 17, IssueID: 21, PRAuthorID: 9, HeadRef: "refs/heads/candidate", BaseRef: "refs/heads/main", HeadOID: strings.Repeat("a", 40), BaseOID: strings.Repeat("b", 40), NativeRev: 12, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Event: "REQUEST_CHANGES", Body: "file.go: correct the demonstrated failure."}
}

func reviewTestCommentedWork() factory.ReviewWork {
	w := reviewTestWork()
	w.Comments = []factory.FindingComment{
		{Key: strings.Repeat("0", 31) + "a", Path: "file.go", Side: "new", Line: 3, Body: "key line plus defect prose"},
		{Key: strings.Repeat("0", 31) + "b", Path: "file.go", Side: "new", Line: 9, Body: "second defect prose"},
	}
	return w
}

func reviewTestReceipt(t *testing.T, w factory.ReviewWork, commentIDs map[string]int64) []byte {
	t.Helper()
	receipt := extensions.ReviewSubmitReceipt{ReviewID: 31, CommentID: 41, ReviewerID: w.ActorID, PRID: w.PRID, IssueID: w.IssueID, PRNumber: w.PRNumber, CommitID: w.HeadOID, Event: w.Event, HeadOID: w.HeadOID, BaseOID: w.BaseOID, MergeBaseOID: strings.Repeat("d", 40)}
	for _, comment := range w.Comments {
		id, ok := commentIDs[comment.Key]
		if !ok {
			id = 100 + int64(len(receipt.Comments))
		}
		receipt.Comments = append(receipt.Comments, extensions.ReviewCommentReceipt{Key: comment.Key, CommentID: id})
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func reviewTestOutcome(t *testing.T, w factory.ReviewWork) factory.OperationOutcome {
	t.Helper()
	return factory.OperationOutcome{OperationID: w.OperationID, InstallationID: "installation", Kind: factory.OpReviewSubmit, ActorID: w.ActorID, RepositoryID: w.Repository, Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Receipt: reviewTestReceipt(t, w, nil)}
}

func TestReviewerReceipt(t *testing.T) {
	w := reviewTestWork()
	r := &Reviewer{}
	for _, event := range []string{"REQUEST_CHANGES", "APPROVED"} {
		w.Event = event
		o := reviewTestOutcome(t, w)
		adopted, err := r.AdoptReview(w, o)
		if err != nil || adopted.HeadOID != w.HeadOID || adopted.Event != event || adopted.ReviewID != 31 || adopted.MergeBaseOID != strings.Repeat("d", 40) || len(adopted.Comments) != 0 {
			t.Fatalf("adopt: %+v %v", adopted, err)
		}
	}
	commented := reviewTestCommentedWork()
	o := reviewTestOutcome(t, commented)
	adopted, err := r.AdoptReview(commented, o)
	if err != nil || len(adopted.Comments) != 2 || adopted.Comments[0].Key != commented.Comments[0].Key || adopted.Comments[1].Key != commented.Comments[1].Key || adopted.Comments[0].CommentID == adopted.Comments[1].CommentID {
		t.Fatalf("comment mapping not adopted: %+v %v", adopted, err)
	}
	cases := map[string]func(*factory.ReviewWork, *factory.OperationOutcome){
		"new actor":       func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.ActorID++ },
		"new operation":   func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.OperationID += "-new" },
		"wrong kind":      func(_ *factory.ReviewWork, o *factory.OperationOutcome) { o.Kind = factory.OpPRCreate },
		"missing receipt": func(_ *factory.ReviewWork, o *factory.OperationOutcome) { o.Receipt = nil },
		"unknown field": func(_ *factory.ReviewWork, o *factory.OperationOutcome) {
			o.Receipt = append(o.Receipt[:len(o.Receipt)-1], []byte(`,"unexpected":true}`)...)
		},
		"pending": func(_ *factory.ReviewWork, o *factory.OperationOutcome) { o.Effect = factory.OpEffectPending },
		"missing merge base": func(w *factory.ReviewWork, o *factory.OperationOutcome) {
			o.Receipt = reviewTestReceiptNoMergeBase(t, *w)
		},
		"dropped comment": func(w *factory.ReviewWork, o *factory.OperationOutcome) {
			o.Receipt = reviewTestReceipt(t, *w, map[string]int64{w.Comments[0].Key: 100})
			w.Comments = append(w.Comments, factory.FindingComment{Key: strings.Repeat("0", 31) + "c", Path: "file.go", Side: "new", Line: 12, Body: "third defect prose"})
		},
		"unknown key": func(w *factory.ReviewWork, o *factory.OperationOutcome) {
			receipt := extensions.ReviewSubmitReceipt{ReviewID: 31, CommentID: 41, ReviewerID: w.ActorID, PRID: w.PRID, IssueID: w.IssueID, PRNumber: w.PRNumber, CommitID: w.HeadOID, Event: w.Event, HeadOID: w.HeadOID, BaseOID: w.BaseOID, MergeBaseOID: strings.Repeat("d", 40)}
			receipt.Comments = append(receipt.Comments, extensions.ReviewCommentReceipt{Key: w.Comments[0].Key, CommentID: 100})
			receipt.Comments = append(receipt.Comments, extensions.ReviewCommentReceipt{Key: strings.Repeat("f", 32), CommentID: 101})
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			o.Receipt = raw
		},
		"duplicate comment record": func(w *factory.ReviewWork, o *factory.OperationOutcome) {
			o.Receipt = reviewTestReceipt(t, *w, map[string]int64{w.Comments[0].Key: 100, w.Comments[1].Key: 100})
		},
		"echoed reviewer differs": func(w *factory.ReviewWork, o *factory.OperationOutcome) {
			receipt := extensions.ReviewSubmitReceipt{ReviewID: 31, CommentID: 41, ReviewerID: w.ActorID + 1, PRID: w.PRID, IssueID: w.IssueID, PRNumber: w.PRNumber, CommitID: w.HeadOID, Event: w.Event, HeadOID: w.HeadOID, BaseOID: w.BaseOID, MergeBaseOID: strings.Repeat("d", 40)}
			for _, comment := range w.Comments {
				receipt.Comments = append(receipt.Comments, extensions.ReviewCommentReceipt{Key: comment.Key, CommentID: 100 + int64(len(receipt.Comments))})
			}
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			o.Receipt = raw
		},
		"echoed head differs": func(w *factory.ReviewWork, o *factory.OperationOutcome) {
			receipt := extensions.ReviewSubmitReceipt{ReviewID: 31, CommentID: 41, ReviewerID: w.ActorID, PRID: w.PRID, IssueID: w.IssueID, PRNumber: w.PRNumber, CommitID: w.HeadOID, Event: w.Event, HeadOID: strings.Repeat("e", 40), BaseOID: w.BaseOID, MergeBaseOID: strings.Repeat("d", 40)}
			for _, comment := range w.Comments {
				receipt.Comments = append(receipt.Comments, extensions.ReviewCommentReceipt{Key: comment.Key, CommentID: 100 + int64(len(receipt.Comments))})
			}
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			o.Receipt = raw
		},
		"receipt above review cap": func(_ *factory.ReviewWork, o *factory.OperationOutcome) {
			o.Receipt = bytes.Repeat([]byte{0x20}, factory.MaxReviewReceiptBytes+1)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			w := reviewTestCommentedWork()
			o := reviewTestOutcome(t, w)
			change(&w, &o)
			if _, err := r.AdoptReview(w, o); err == nil {
				t.Fatal("adopted mismatched review")
			}
		})
	}
}

// capturedNativeReviewReceipt049 is the exact public receipt bytes the
// native producer emitted for a genuinely zero-inline-comment approval
// (independent capture actual-native-review-receipt-049.json). It guards
// the cross-repository wire: the real decoder/adopter must accept these
// exact bytes, preserve every echoed identity, and still refuse any
// echoed-identity mismatch or unknown field.
const capturedNativeReviewReceipt049 = `{"base_oid":"65f1bf27bc3bf70f64657658635e66094edbcb4d","comment_id":2094,"commit_id":"985f0301dba5e7b34be866819cd15ad3d8f508ee","event":"APPROVED","head_oid":"985f0301dba5e7b34be866819cd15ad3d8f508ee","issue_id":3,"merge_base_oid":"65f1bf27bc3bf70f64657658635e66094edbcb4d","pr_id":2,"pr_number":3,"review_id":23,"reviewer_id":2}`

func TestReviewerAdoptsCapturedNativeReceipt(t *testing.T) {
	w := factory.ReviewWork{OperationID: "review-merge-base-ok", AuthRevision: "accepted-revision", Repository: 1, ActorID: 2, PRNumber: 3, PRID: 2, IssueID: 3, PRAuthorID: 1, HeadRef: "refs/heads/branch2", BaseRef: "refs/heads/master", HeadOID: "985f0301dba5e7b34be866819cd15ad3d8f508ee", BaseOID: "65f1bf27bc3bf70f64657658635e66094edbcb4d", NativeRev: 1, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Event: "APPROVED", Body: "looks good"}
	o := factory.OperationOutcome{OperationID: w.OperationID, InstallationID: "11111111-2222-4333-8444-555555555555", Kind: factory.OpReviewSubmit, ActorID: w.ActorID, RepositoryID: w.Repository, Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Receipt: []byte(capturedNativeReviewReceipt049)}
	adopted, err := (&Reviewer{}).AdoptReview(w, o)
	if err != nil {
		t.Fatalf("captured native receipt refused: %v", err)
	}
	if adopted.ReviewID != 23 || adopted.CommentID != 2094 || adopted.ReviewerID != 2 || adopted.PRID != 2 || adopted.IssueID != 3 || adopted.PRNumber != 3 || adopted.HeadOID != w.HeadOID || adopted.BaseOID != w.BaseOID || adopted.CommitID != w.HeadOID || adopted.Event != "APPROVED" || adopted.MergeBaseOID != w.BaseOID {
		t.Fatalf("captured native identities not preserved: %+v", adopted)
	}
	for _, tamper := range []func(map[string]any){func(m map[string]any) { m["head_oid"] = strings.Repeat("e", 40) }, func(m map[string]any) { m["reviewer_id"] = float64(99) }, func(m map[string]any) { m["unexpected"] = true }} {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(capturedNativeReviewReceipt049), &decoded); err != nil {
			t.Fatal(err)
		}
		tamper(decoded)
		raw, err := json.Marshal(decoded)
		if err != nil {
			t.Fatal(err)
		}
		o.Receipt = raw
		if _, err := (&Reviewer{}).AdoptReview(w, o); err == nil {
			t.Fatalf("tampered native receipt admitted: %s", raw)
		}
	}
}

func reviewTestReceiptNoMergeBase(t *testing.T, _ factory.ReviewWork) []byte {
	t.Helper()
	receipt := extensions.ReviewSubmitReceipt{ReviewID: 31, CommentID: 41}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestReviewerIntentBodyOnly(t *testing.T) {
	w := reviewTestWork()
	intent, err := reviewIntent(w)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err = json.Unmarshal(intent.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 10 || payload["body"] != w.Body || payload["commit_id"] != w.HeadOID || payload["event"] != w.Event || intent.ActorID != "11" || intent.RepositoryID != "7" || intent.Kind != factory.OpReviewSubmit {
		t.Fatalf("incorrect body-only intent: %+v", intent)
	}
	w.ActorID = w.PRAuthorID
	if _, err = reviewIntent(w); err == nil {
		t.Fatal("self review authorized")
	}
}

func TestReviewerIntentComments(t *testing.T) {
	w := reviewTestCommentedWork()
	intent, err := reviewIntent(w)
	if err != nil {
		t.Fatal(err)
	}
	var payload extensions.ReviewSubmitPayload
	if err = json.Unmarshal(intent.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Comments) != 2 || payload.Comments[0].Key != w.Comments[0].Key || payload.Comments[0].Path != "file.go" || payload.Comments[0].Side != "new" || payload.Comments[0].Line != 3 || payload.Comments[1].Key != w.Comments[1].Key {
		t.Fatalf("finding comments missing from intent: %+v", payload.Comments)
	}
	// A body that fits the work bound but pushes the complete encoded
	// payload past the native ceiling refuses at the Soda adapter.
	w.Body = strings.Repeat("b", factory.MaxReviewPayloadBytes)
	if _, err = reviewIntent(w); err == nil {
		t.Fatal("oversized review payload admitted")
	}
	w.Body = "file.go: correct the demonstrated failure."
	if _, err = reviewIntent(w); err != nil {
		t.Fatalf("bounded commented review refused: %v", err)
	}
	// Commented work that fails structural validation never reaches submit.
	w.Comments[1].Key = w.Comments[0].Key
	if _, err = reviewIntent(w); err == nil {
		t.Fatal("duplicate finding key admitted")
	}
}

func TestReviewOperationOutcomeReviewCap(t *testing.T) {
	record := func(receipt []byte) extensions.OperationRecord {
		return extensions.OperationRecord{OperationID: "st10-review", InstallationID: "installation", Kind: factory.OpReviewSubmit, ActorID: "11", RepositoryID: "7", EffectState: factory.OpEffectCommitted, CancellationStatus: factory.OpCancelNone, CompletionState: factory.OpCompletionComplete, Receipt: receipt}
	}
	sized, err := json.Marshal(extensions.ReviewSubmitReceipt{ReviewID: 1, CommentID: 2, MergeBaseOID: strings.Repeat("d", 40)})
	if err != nil {
		t.Fatal(err)
	}
	// Maximum review-specific receipts persist; one byte more refuses. The
	// outcome carries the raw receipt without decoding it, so trailing JSON
	// whitespace pads to the exact boundary. Small receipts from before the
	// review-specific cap stay valid.
	atCap := append(append([]byte(nil), sized...), bytes.Repeat([]byte{0x20}, factory.MaxReviewReceiptBytes-len(sized))...)
	if len(atCap) != factory.MaxReviewReceiptBytes {
		t.Fatalf("receipt boundary misbuilt: %d", len(atCap))
	}
	if _, err = reviewOperationOutcome("st10-review", record(atCap)); err != nil {
		t.Fatalf("maximum review receipt refused: %v", err)
	}
	if _, err = reviewOperationOutcome("st10-review", record(append(atCap, 0x20))); err == nil {
		t.Fatal("review receipt above its bound admitted")
	}
	if _, err = reviewOperationOutcome("st10-review", record(bytes.Repeat([]byte{0x20}, factory.MaxPublicationReceipt))); err != nil {
		t.Fatalf("generic-bound review receipt refused: %v", err)
	}
}

func TestReviewerLookupAfterCredentialLoss(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	bg := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	credential := observationCredential(t, "test-pat")
	r := NewReviewer(bg, observationREST(t, 11), credential)
	w := reviewTestWork()
	ctx := context.Background()
	pending, err := r.SubmitReview(ctx, w)
	if err != nil || pending.Effect != factory.OpEffectPending {
		t.Fatalf("submit: %+v %v", pending, err)
	}
	if err = os.Remove(credential); err != nil {
		t.Fatal(err)
	}
	observed, err := r.LookupOp(ctx, w.OperationID)
	if err != nil || observed.OperationID != w.OperationID {
		t.Fatalf("credential-free lookup: %+v %v", observed, err)
	}
	cancelled, err := r.CancelOp(ctx, w.OperationID)
	if err != nil || cancelled.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("credential-free cancellation: %+v %v", cancelled, err)
	}
	tombstone, err := r.CancelOp(ctx, "st10-before-submit")
	if err != nil || tombstone.Kind != "" || tombstone.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("tombstone: %+v %v", tombstone, err)
	}
	absent, err := r.LookupOp(ctx, "st10-absent")
	if err != nil || !absent.NotObserved {
		t.Fatalf("absence: %+v %v", absent, err)
	}
}

func TestReviewerActorMismatch(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	bg := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	r := NewReviewer(bg, observationREST(t, 12), observationCredential(t, "test-pat"))
	_, err := r.SubmitReview(context.Background(), reviewTestWork())
	var wait *factory.PublicationWait
	if !errors.As(err, &wait) || wait.Reason != "authority_lost" || fake.submits != 0 {
		t.Fatalf("actor mismatch not stopped before submit: %v", err)
	}
}

func TestReviewerExactNativeTarget(t *testing.T) {
	fixture := func() (factory.ReviewWork, NativeSnapshot, NativeSnapshot) {
		w := reviewTestWork()
		issue := NativeSnapshot{Issue: &IssueEvidence{ID: "21", Index: "5", IsPull: true, Visible: true, Provenance: CreationProvenance{PosterID: "9"}}}
		snapshot := NativeSnapshot{Pull: &PullEvidence{ID: "17", IssueID: "21", Number: "5", HeadRepoID: "7", HeadBranch: "candidate", BaseBranch: "main", HeadTip: w.HeadOID, MergeBase: strings.Repeat("d", 40), Visible: true}, Reviews: &ReviewPage{IssueID: "21", Complete: true}, Refs: []RefEvidence{{Ref: w.HeadRef, OID: w.HeadOID, Exists: true, Visible: true}, {Ref: w.BaseRef, OID: w.BaseOID, Exists: true, Visible: true}}}
		return w, issue, snapshot
	}
	w, issue, snapshot := fixture()
	if err := matchReviewTarget(w, issue, snapshot); err != nil {
		t.Fatal("merge base was confused with base tip:", err)
	}
	cases := map[string]func(*NativeSnapshot, *NativeSnapshot){
		"author": func(i, _ *NativeSnapshot) { i.Issue.Provenance.PosterID = "10" },
		"closed": func(i, _ *NativeSnapshot) { i.Issue.IsClosed = true },
		"head":   func(_, s *NativeSnapshot) { s.Pull.HeadTip = strings.Repeat("c", 40) },
		"base":   func(_, s *NativeSnapshot) { s.Refs[1].OID = strings.Repeat("c", 40) },
		"source": func(_, s *NativeSnapshot) { s.Pull.HeadBranch = "different" },
		"merged": func(_, s *NativeSnapshot) { s.Pull.HasMerged = true },
		"draft": func(_, s *NativeSnapshot) {
			s.Reviews.Items = []ReviewEvidence{{ID: "31", IssueID: "21", ReviewerID: "11", Type: "PENDING", Visible: true}}
		},
		"hidden review": func(_, s *NativeSnapshot) { s.Reviews.Items = []ReviewEvidence{{ID: "31", Visible: false}} },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			w, i, s := fixture()
			change(&i, &s)
			if err := matchReviewTarget(w, i, s); err == nil {
				t.Fatal("changed target accepted")
			}
		})
	}
}

func TestReviewerLostSubmitReply(t *testing.T) {
	w := reviewTestWork()
	fixture := reviewTestOutcome(t, w)
	record := extensions.OperationRecord{InstallationID: "install-1", OperationID: w.OperationID, Kind: factory.OpReviewSubmit, ActorID: "11", RepositoryID: "7", Outcome: factory.OpEffectCommitted, EffectState: factory.OpEffectCommitted, CancellationStatus: factory.OpCancelNone, CompletionState: factory.OpCompletionComplete, Receipt: fixture.Receipt}
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{w.OperationID: record}}
	socket := shortSocketPath(t, "r.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	base := fake.handler()
	server := &http.Server{Handler: http.HandlerFunc(func(out http.ResponseWriter, in *http.Request) {
		if in.URL.Path != extensions.BackgroundSubmitPath {
			base.ServeHTTP(out, in)
			return
		}
		discarded := httptest.NewRecorder()
		base.ServeHTTP(discarded, in)
		http.Error(out, "reply lost after native commit", http.StatusInternalServerError)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); _ = listener.Close() })
	r := NewReviewer(NewServiceBackground(socket, uint32(os.Getuid()), ""), observationREST(t, 11), observationCredential(t, "test-pat"))
	got, err := r.SubmitReview(context.Background(), w)
	if err != nil {
		t.Fatal(err)
	}
	adopted, err := r.AdoptReview(w, got)
	if err != nil || adopted.ReviewID != 31 || fake.submits != 1 {
		t.Fatalf("lost reply did not adopt one existing operation: %+v %v calls=%d", adopted, err, fake.submits)
	}
}
