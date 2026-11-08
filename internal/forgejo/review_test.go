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
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/store"
)

// Captured-receipt checks explicitly declare the retained equal-ID namespace;
// the private-transport journeys below deliberately use colliding provider IDs.
func reviewRetainedIdentities(t *testing.T) *FountainIdentities {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, _, cleanup, err := store.OpenEphemeral(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	host, err := db.BindConfiguredFountainLedger(t.Context(), "https://review-fixture.test")
	if err != nil {
		t.Fatal(err)
	}
	identities, err := NewFountainIdentities(t.Context(), db, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	for native := int64(1); native <= 7; native++ {
		if local, err := identities.LocalRepository(t.Context(), native); err != nil || int64(local) != native {
			t.Fatal("review repository fixture namespace differs", local, err)
		}
	}
	for native := int64(1); native <= 11; native++ {
		if local, err := identities.LocalActor(t.Context(), strconv.FormatInt(native, 10)); err != nil || int64(local) != native {
			t.Fatal("review actor fixture namespace differs", local, err)
		}
	}
	return identities
}

func canonicalReviewWork(t *testing.T, identities *FountainIdentities, w factory.ReviewWork) factory.ReviewWork {
	t.Helper()
	repository, err := identities.LocalRepository(t.Context(), w.Repository)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := identities.LocalActor(t.Context(), strconv.FormatInt(w.ActorID, 10))
	if err != nil {
		t.Fatal(err)
	}
	author, err := identities.LocalActor(t.Context(), strconv.FormatInt(w.PRAuthorID, 10))
	if err != nil {
		t.Fatal(err)
	}
	w.Repository, w.ActorID, w.PRAuthorID = int64(repository), int64(actor), int64(author)
	return w
}

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
	r := &Reviewer{Identities: reviewRetainedIdentities(t)}
	for _, event := range []string{"REQUEST_CHANGES", "APPROVED"} {
		w.Event = event
		o := reviewTestOutcome(t, w)
		adopted, err := r.AdoptReview(t.Context(), w, o)
		if err != nil || adopted.HeadOID != w.HeadOID || adopted.Event != event || adopted.ReviewID != 31 || adopted.MergeBaseOID != strings.Repeat("d", 40) || len(adopted.Comments) != 0 {
			t.Fatalf("adopt: %+v %v", adopted, err)
		}
	}
	commented := reviewTestCommentedWork()
	o := reviewTestOutcome(t, commented)
	adopted, err := r.AdoptReview(t.Context(), commented, o)
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
			if _, err := r.AdoptReview(t.Context(), w, o); err == nil {
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
	r := &Reviewer{Identities: reviewRetainedIdentities(t)}
	w := factory.ReviewWork{OperationID: "review-merge-base-ok", AuthRevision: "accepted-revision", Repository: 1, ActorID: 2, PRNumber: 3, PRID: 2, IssueID: 3, PRAuthorID: 1, HeadRef: "refs/heads/branch2", BaseRef: "refs/heads/master", HeadOID: "985f0301dba5e7b34be866819cd15ad3d8f508ee", BaseOID: "65f1bf27bc3bf70f64657658635e66094edbcb4d", NativeRev: 1, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Event: "APPROVED", Body: "looks good"}
	o := factory.OperationOutcome{OperationID: w.OperationID, InstallationID: "11111111-2222-4333-8444-555555555555", Kind: factory.OpReviewSubmit, ActorID: w.ActorID, RepositoryID: w.Repository, Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Receipt: []byte(capturedNativeReviewReceipt049)}
	adopted, err := r.AdoptReview(t.Context(), w, o)
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
		if _, err := r.AdoptReview(t.Context(), w, o); err == nil {
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
	identities := graphIdentityFixture(t, 7, 11, 9)
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	bg := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	credential := observationCredential(t, "test-pat")
	r := NewReviewer(bg, observationREST(t, 11), credential, identities)
	w := canonicalReviewWork(t, identities, reviewTestCommentedWork())
	ctx := context.Background()
	pending, err := r.SubmitReview(ctx, w)
	if err != nil || pending.Effect != factory.OpEffectPending || pending.ActorID != w.ActorID || pending.RepositoryID != w.Repository {
		t.Fatalf("submit: %+v %v", pending, err)
	}
	fake.mu.Lock()
	submitted := fake.intents[len(fake.intents)-1]
	beforeSubmits := fake.submits
	fake.mu.Unlock()
	var payload extensions.ReviewSubmitPayload
	if json.Unmarshal(submitted.Payload, &payload) != nil || submitted.ActorID != "11" || submitted.RepositoryID != "7" || payload.PRAuthorID != 9 || payload.HeadRepositoryID != 7 || len(payload.Comments) != 2 || payload.Comments[0].Key != w.Comments[0].Key || payload.Comments[0].Body != w.Comments[0].Body {
		t.Fatal("canonical scope reached the native review packet")
	}
	for _, scope := range [][3]int64{{1, w.ActorID, w.PRAuthorID}, {7, w.ActorID, w.PRAuthorID}, {w.Repository, 1, w.PRAuthorID}, {w.Repository, w.ActorID, 2}} {
		foreign := w
		foreign.Repository, foreign.ActorID, foreign.PRAuthorID = scope[0], scope[1], scope[2]
		if _, err := r.SubmitReview(ctx, foreign); err == nil {
			t.Fatal("foreign/unmapped scope reached review submit", scope)
		}
	}
	fake.mu.Lock()
	afterSubmits := fake.submits
	fake.mu.Unlock()
	if afterSubmits != beforeSubmits {
		t.Fatal("foreign review scope reached a native submit")
	}
	resolver := NewResolver(bg, r.rest, credential, identities)
	resolution := factory.ResolveWork{
		Request: factory.ResolveRequest{Repository: w.Repository, ActorID: w.ActorID, PRID: w.PRID, PRNumber: w.PRNumber, RootCommentID: 41, HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID, Digest: strings.Repeat("d", 64), Resolved: true},
		Binding: factory.ActorBindingRef{TokenID: 1, ActorID: w.ActorID, Kind: factory.OpThreadResolve}, PRAuthorID: w.PRAuthorID,
		OperationID: factory.NewID(), AuthRevision: "review-resolution", NativeRevision: 12, NotAfter: w.NotAfter,
	}
	resolutionPending, err := resolver.SubmitResolution(ctx, resolution)
	if err != nil || resolutionPending.ActorID != w.ActorID || resolutionPending.RepositoryID != w.Repository || resolutionPending.Effect != factory.OpEffectPending {
		t.Fatal("resolution lost canonical operation scope", resolutionPending, err)
	}
	fake.mu.Lock()
	resolveIntent := fake.intents[len(fake.intents)-1]
	committed := fake.ops[resolution.OperationID]
	committed.EffectState, committed.Outcome, committed.CompletionState = factory.OpEffectCommitted, factory.OpEffectCommitted, factory.OpCompletionComplete
	committed.Receipt, err = json.Marshal(extensions.ThreadResolveReceipt{RootCommentID: 41, PullRequestNumber: w.PRNumber, ConversationDigest: resolution.Request.Digest, Resolved: true})
	fake.ops[resolution.OperationID] = committed
	fake.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	var resolvePayload extensions.ThreadResolvePayload
	if json.Unmarshal(resolveIntent.Payload, &resolvePayload) != nil || resolveIntent.ActorID != "11" || resolveIntent.RepositoryID != "7" || resolvePayload.HeadRepositoryID != 7 || resolvePayload.RootCommentID != 41 || resolvePayload.ConversationDigest != resolution.Request.Digest {
		t.Fatal("canonical scope reached the native resolution packet")
	}
	resolutionOutcome, err := resolver.LookupResolution(ctx, resolution.OperationID)
	if err != nil || resolutionOutcome.ActorID != w.ActorID || resolutionOutcome.RepositoryID != w.Repository {
		t.Fatal("resolution readback lost canonical scope", resolutionOutcome, err)
	}
	var resolverID atomic.Int64
	resolverID.Store(11)
	var reads atomic.Int64
	resolvedDigest := strings.Repeat("f", 64)
	fake.sources = func(out http.ResponseWriter, in *http.Request) {
		reads.Add(1)
		var request struct {
			extensions.SnapshotRequest
			Token string `json:"token"`
		}
		if json.NewDecoder(in.Body).Decode(&request) != nil || request.RepositoryID != "7" || request.ActorID != "11" || request.Token != "test-pat" || request.PullNumber != "5" {
			t.Error("canonical scope reached native resolution evidence")
			http.Error(out, "invalid fixture scope", 400)
			return
		}
		_ = json.NewEncoder(out).Encode(extensions.NativeSnapshot{RepositoryID: "7", Threads: &extensions.SnapshotThreadPage{Total: 1, Complete: true, Items: []extensions.SnapshotThread{{RootID: "41", IssueID: "21", ReviewID: "31", Path: "file.go", Side: "new", Line: 3, OriginalCommit: w.HeadOID, HeadCommit: w.HeadOID, Resolved: true, ResolverID: strconv.FormatInt(resolverID.Load(), 10), Digest: resolvedDigest, MemberTotal: 1, Members: []extensions.SnapshotThreadMember{{ID: "41", IssueID: "21", Resolved: true, ContentDigest: ContentDigest("root finding"), ContentVersion: 1, Visible: true, Complete: true}}, Visible: true, Complete: true}}}})
	}
	operation := factory.DeliveryResolutionOperation{Work: resolution, Outcome: resolutionOutcome, ResolvedDigest: resolvedDigest}
	if confirmed, err := resolver.ConfirmResolution(ctx, operation); err != nil || !confirmed {
		t.Fatal("mapped native resolver could not confirm its thread", confirmed, err)
	}
	beforeReads := reads.Load()
	foreignOperation := operation
	foreignOperation.Outcome.ActorID = 1
	if _, err := resolver.ConfirmResolution(ctx, foreignOperation); err == nil || reads.Load() != beforeReads {
		t.Fatal("foreign canonical resolution outcome reached native evidence", err)
	}
	resolverID.Store(9)
	if confirmed, err := resolver.ConfirmResolution(ctx, operation); err != nil || confirmed {
		t.Fatal("another native resolver was attributed to this reviewer", confirmed, err)
	}
	if err = os.Remove(credential); err != nil {
		t.Fatal(err)
	}
	observed, err := r.LookupOp(ctx, w.OperationID)
	if err != nil || observed.OperationID != w.OperationID || observed.ActorID != w.ActorID || observed.RepositoryID != w.Repository {
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
	if observed, err := resolver.LookupResolution(ctx, resolution.OperationID); err != nil || observed.ActorID != w.ActorID || !bytes.Equal(observed.Receipt, resolutionOutcome.Receipt) {
		t.Fatal("credential loss hid the committed resolution", observed, err)
	}
	if cancelled, err := resolver.CancelResolution(ctx, resolution.OperationID); err != nil || cancelled.Effect != factory.OpEffectCommitted || cancelled.Cancellation != factory.OpCancelTooLate || cancelled.ActorID != w.ActorID {
		t.Fatal("cancellation erased the committed resolution", cancelled, err)
	}
}

func TestReviewerActorMismatch(t *testing.T) {
	identities := graphIdentityFixture(t, 7, 11, 9)
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	bg := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	r := NewReviewer(bg, observationREST(t, 12), observationCredential(t, "test-pat"), identities)
	_, err := r.SubmitReview(context.Background(), canonicalReviewWork(t, identities, reviewTestWork()))
	var wait *factory.PublicationWait
	if !errors.As(err, &wait) || wait.Reason != "authority_lost" || fake.submits != 0 {
		t.Fatalf("actor mismatch not stopped before submit: %v", err)
	}
}

func TestReviewerExactNativeTarget(t *testing.T) {
	identities := graphIdentityFixture(t, 7, 11, 9)
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
	var reads atomic.Int64
	var mode atomic.Int64
	peer := &fakeBackgroundServer{t: t, admission: strings.Repeat("a", 43)}
	peer.snapshot = func(request extensions.SnapshotRequest) extensions.NativeSnapshot {
		reads.Add(1)
		if request.RepositoryID != "7" || request.ActorID != "11" {
			t.Error("canonical review scope reached native evidence")
		}
		if request.IssueIndex != "" {
			i := *issue.Issue
			i.Title, i.Content, i.TitleDigest, i.ContentDigest, i.Complete = "PR", "exact body", ContentDigest("PR"), ContentDigest("exact body"), true
			if mode.Load() == 2 {
				i.Provenance.PosterID = "10"
			}
			return extensions.NativeSnapshot{RepositoryID: "7", Issue: &i}
		}
		pull := *snapshot.Pull
		pull.Complete = true
		refs := append([]RefEvidence(nil), snapshot.Refs...)
		for i := range refs {
			refs[i].Complete = true
		}
		reviews := &ReviewPage{IssueID: "21", Complete: true, Items: []ReviewEvidence{}}
		if mode.Load() == 1 {
			reviews.Total = 1
			reviews.Items = []ReviewEvidence{{ID: "31", IssueID: "21", ReviewerID: "11", Type: "PENDING", CommitID: w.HeadOID, ContentDigest: ContentDigest("draft"), Visible: true, Complete: true}}
		}
		return extensions.NativeSnapshot{RepositoryID: "7", Pull: &pull, Reviews: reviews, Refs: refs}
	}
	r := NewReviewer(NewServiceBackground(serveBackgroundSocket(t, peer), uint32(os.Getuid()), ""), observationREST(t, 11), observationCredential(t, "test-pat"), identities)
	canonical := canonicalReviewWork(t, identities, w)
	observed, err := r.ObserveReview(t.Context(), canonical)
	if err != nil || observed.NativeRev != 12 || observed.PRAuthorID != canonical.PRAuthorID || observed.PRID != w.PRID || observed.IssueID != w.IssueID || observed.HeadOID != w.HeadOID || observed.BaseOID != w.BaseOID {
		t.Fatal("mapped native review did not preserve canonical target evidence", observed, err)
	}
	for state, reason := range map[int64]string{1: "pending_review_exists", 2: "pr_mismatch"} {
		mode.Store(state)
		_, err := r.ObserveReview(t.Context(), canonical)
		var refusal *factory.PublicationRefusal
		if !errors.As(err, &refusal) || refusal.Reason != reason {
			t.Fatal("native pending reviewer/PR author was compared as canonical", state, err)
		}
	}
	before := reads.Load()
	for _, scope := range [][3]int64{{1, canonical.ActorID, canonical.PRAuthorID}, {7, canonical.ActorID, canonical.PRAuthorID}, {canonical.Repository, 1, canonical.PRAuthorID}, {canonical.Repository, canonical.ActorID, 2}} {
		foreign := canonical
		foreign.Repository, foreign.ActorID, foreign.PRAuthorID = scope[0], scope[1], scope[2]
		if _, err := r.ObserveReview(t.Context(), foreign); err == nil || reads.Load() != before {
			t.Fatal("foreign/unmapped scope reached native review reads", scope, err)
		}
	}
	t.Run("delivery_material", func(t *testing.T) {
		checkMappedDeliveryMaterial(t, identities, canonical, issue, snapshot)
	})
	t.Run("checks_merge_completion", func(t *testing.T) {
		actor, err := identities.LocalActor(t.Context(), "12")
		if err != nil {
			t.Fatal(err)
		}
		var merged atomic.Bool
		var advanced atomic.Bool
		var nativeMerger atomic.Int64
		var nativeReviewer atomic.Int64
		var nativeReads atomic.Int64
		nativeMerger.Store(12)
		nativeReviewer.Store(11)
		native := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
		native.sources = func(out http.ResponseWriter, in *http.Request) {
			nativeReads.Add(1)
			var request struct {
				extensions.SnapshotRequest
				Token string `json:"token"`
			}
			if json.NewDecoder(in.Body).Decode(&request) != nil || request.RepositoryID != "7" || request.ActorID != "12" || request.Token != "test-pat" {
				t.Error("canonical check/merge scope reached native evidence")
				http.Error(out, "invalid fixture scope", 400)
				return
			}
			if request.IssueIndex != "" {
				i := *issue.Issue
				i.Title, i.Content, i.Complete = "PR", "exact body", true
				if merged.Load() {
					i.IsClosed, i.ClosedUnix = true, 1700000030
				}
				if request.IssueIndex == "3" {
					i.ID, i.Index, i.IsPull = "33", "3", false
					i.Title, i.Content = "Original issue", "requested result"
					i.Lifecycle = []LifecycleEvent{{Kind: "closed", OccurrenceID: "77", AtUnix: 1700000030}}
				}
				i.TitleDigest, i.ContentDigest = ContentDigest(i.Title), ContentDigest(i.Content)
				_ = json.NewEncoder(out).Encode(extensions.NativeSnapshot{RepositoryID: "7", Issue: &i})
				return
			}
			refs := []RefEvidence{{Ref: w.HeadRef, OID: w.HeadOID, Exists: true, Visible: true, Complete: true}, {Ref: w.BaseRef, OID: w.BaseOID, Exists: true, Visible: true, Complete: true}}
			if merged.Load() {
				refs[1].OID = w.HeadOID
				if advanced.Load() {
					refs[1].OID = strings.Repeat("e", 40)
				}
			}
			if request.Ancestry != nil {
				_ = json.NewEncoder(out).Encode(extensions.NativeSnapshot{RepositoryID: "7", Refs: []RefEvidence{refs[1]}, Ancestry: &extensions.SnapshotAncestry{AncestorOID: request.Ancestry.AncestorOID, DescendantOID: request.Ancestry.DescendantOID, Reachable: true}})
				return
			}
			pull := *snapshot.Pull
			pull.Complete = true
			if merged.Load() {
				pull.HasMerged, pull.MergedCommit, pull.MergerID, pull.MergedUnix = true, w.HeadOID, strconv.FormatInt(nativeMerger.Load(), 10), 1700000020
			}
			reviews := &ReviewPage{IssueID: "21", Total: 1, Complete: true, Items: []ReviewEvidence{{ID: "31", IssueID: "21", ReviewerID: strconv.FormatInt(nativeReviewer.Load(), 10), Type: "APPROVED", CommitID: w.HeadOID, ContentDigest: ContentDigest("approved"), Official: true, Visible: true, Complete: true}}}
			checks := &CheckSet{SHA: w.HeadOID, Total: 2, Complete: true, Items: []CheckEvidence{{ID: "1", SHA: w.HeadOID, Context: "ci/test", State: "failure", UpdatedUnix: 1, Visible: true, Complete: true}, {ID: "2", SHA: w.HeadOID, Context: "ci/test", State: "success", UpdatedUnix: 2, Visible: true, Complete: true}}}
			_ = json.NewEncoder(out).Encode(extensions.NativeSnapshot{RepositoryID: "7", Pull: &pull, Reviews: reviews, Checks: checks, Refs: refs})
		}
		background := NewServiceBackground(serveScriptedBackground(t, native), uint32(os.Getuid()), "")
		credential := observationCredential(t, "test-pat")
		merger := NewMerger(background, observationREST(t, 12), credential, identities)
		assessor := NewCheckAssessor(background, merger.rest, credential, identities)
		target := factory.CheckTarget{Repository: canonical.Repository, PRID: w.PRID, PRNumber: w.PRNumber, IssueID: w.IssueID, HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID}
		checks, err := assessor.ObserveChecks(t.Context(), target, int64(actor))
		if err != nil || checks.NativeRev != 12 || checks.HeadTip != w.HeadOID || checks.BaseTip != w.BaseOID || len(checks.Checks) != 1 || checks.Checks[0].State != "success" {
			t.Fatal("mapped native checks lost exact-head latest-status evidence", checks, err)
		}
		work := factory.MergeWork{MergeID: factory.NewID(), PublicationID: factory.NewID(), OperationID: "soda-merge-scope-1", AuthRevision: "exact-merge-scope", Repository: canonical.Repository, ActorID: int64(actor), PRAuthorID: canonical.PRAuthorID, ReviewerID: canonical.ActorID, Issue: 3, IssueID: w.IssueID, PRID: w.PRID, PRNumber: w.PRNumber, HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID, NativeRev: 12, NotAfter: w.NotAfter, AssessmentRevision: 1, ReviewID: 31}
		observed, err := merger.ObserveMerge(t.Context(), work)
		if err != nil || observed.PRAuthorID != work.PRAuthorID || observed.ReviewerID != work.ReviewerID || observed.ReviewID != 31 || observed.Checks.Checks[0].State != "success" {
			t.Fatal("mapped merge did not preserve independent canonical approval", observed, err)
		}
		nativeReviewer.Store(9)
		if _, err := merger.ObserveMerge(t.Context(), work); err == nil {
			t.Fatal("native PR author's approval became independent")
		}
		nativeReviewer.Store(12)
		if _, err := merger.ObserveMerge(t.Context(), work); err == nil {
			t.Fatal("another native reviewer supplied the recorded approval")
		}
		nativeReviewer.Store(11)
		before := nativeReads.Load()
		for _, scope := range [][4]int64{{1, work.ActorID, work.PRAuthorID, work.ReviewerID}, {7, work.ActorID, work.PRAuthorID, work.ReviewerID}, {work.Repository, 1, work.PRAuthorID, work.ReviewerID}, {work.Repository, work.ActorID, 2, work.ReviewerID}, {work.Repository, work.ActorID, work.PRAuthorID, 1}} {
			foreign := work
			foreign.Repository, foreign.ActorID, foreign.PRAuthorID, foreign.ReviewerID = scope[0], scope[1], scope[2], scope[3]
			if _, err := merger.ObserveMerge(t.Context(), foreign); err == nil || nativeReads.Load() != before {
				t.Fatal("foreign/unmapped scope reached merge evidence", scope, err)
			}
			if _, err := merger.SubmitMerge(t.Context(), foreign); err == nil {
				t.Fatal("foreign/unmapped scope reached merge submission", scope)
			}
		}
		foreignTarget := target
		foreignTarget.Repository = 1
		if _, err := assessor.ObserveChecks(t.Context(), foreignTarget, int64(actor)); err == nil || nativeReads.Load() != before {
			t.Fatal("foreign check repository reached native evidence", err)
		}
		if _, err := assessor.ObserveChecks(t.Context(), target, 1); err == nil || nativeReads.Load() != before {
			t.Fatal("foreign assessor reached native evidence", err)
		}
		pending, err := merger.SubmitMerge(t.Context(), work)
		if err != nil || pending.ActorID != work.ActorID || pending.RepositoryID != work.Repository || pending.Effect != factory.OpEffectPending {
			t.Fatal("native merge submission lost canonical outcome scope", pending, err)
		}
		native.mu.Lock()
		intent := native.intents[len(native.intents)-1]
		committed := native.ops[work.OperationID]
		submits := native.submits
		native.mu.Unlock()
		var packet extensions.MergePayload
		if submits != 1 || json.Unmarshal(intent.Payload, &packet) != nil || intent.ActorID != "12" || intent.RepositoryID != "7" || packet.HeadRepositoryID != 7 || packet.ExpectedHeadOID != work.HeadOID || packet.ExpectedBaseOID != work.BaseOID {
			t.Fatal("canonical merge scope reached native operation payload")
		}
		committed.Outcome, committed.EffectState, committed.CompletionState = factory.OpEffectCommitted, factory.OpEffectCommitted, factory.OpCompletionComplete
		committed.Receipt, err = json.Marshal(mergeReceipt{ActorID: 12, RepositoryID: 7, PRNumber: w.PRNumber, PRID: w.PRID, HeadRef: w.HeadRef, BaseRef: w.BaseRef, OldOID: w.BaseOID, NewOID: w.HeadOID, Method: factory.MergeFastForward})
		if err != nil {
			t.Fatal(err)
		}
		native.mu.Lock()
		native.ops[work.OperationID] = committed
		native.mu.Unlock()
		outcome, err := merger.LookupOp(t.Context(), work.OperationID)
		if err != nil {
			t.Fatal(err)
		}
		adopted, err := merger.AdoptMerge(t.Context(), work, outcome)
		if err != nil || adopted.ActorID != work.ActorID || adopted.Operation.ActorID != work.ActorID || !bytes.Equal(adopted.Operation.Receipt, committed.Receipt) || adopted.MergedCommit != work.HeadOID {
			t.Fatal("native merge receipt did not retain canonical attribution", adopted, err)
		}
		merged.Store(true)
		confirmed, err := merger.ObserveCompletion(t.Context(), work)
		if err != nil || confirmed.MergerID != work.ActorID || !confirmed.OriginalClosed || confirmed.OriginalClosingOccurrence != "77" || confirmed.MergedCommit != work.HeadOID {
			t.Fatal("native merge completion lost canonical merger or original closure", confirmed, err)
		}
		advanced.Store(true)
		confirmed, err = merger.ObserveCompletion(t.Context(), work)
		if err != nil || confirmed.MergerID != work.ActorID || confirmed.BaseTip != strings.Repeat("e", 40) || !confirmed.BaseContainsMergedCommit {
			t.Fatal("advanced native base lost its exact ancestry proof", confirmed, err)
		}
		nativeMerger.Store(11)
		if _, err := merger.ObserveCompletion(t.Context(), work); err == nil {
			t.Fatal("another native merger was attributed to this operation")
		}
		if err := os.Remove(credential); err != nil {
			t.Fatal(err)
		}
		if recovered, err := merger.LookupOp(t.Context(), work.OperationID); err != nil || recovered.ActorID != work.ActorID || !bytes.Equal(recovered.Receipt, committed.Receipt) {
			t.Fatal("merge credential loss hid the committed receipt", recovered, err)
		}
		if cancelled, err := merger.CancelOp(t.Context(), work.OperationID); err != nil || cancelled.Effect != factory.OpEffectCommitted || cancelled.Cancellation != factory.OpCancelTooLate || cancelled.ActorID != work.ActorID {
			t.Fatal("merge cancellation erased a committed effect", cancelled, err)
		}
	})
}

// The real registry and private Unix/REST transports exercise the delivery
// discussion consumer with canonical keys different from native subjects.
func checkMappedDeliveryMaterial(t *testing.T, identities *FountainIdentities, canonical factory.ReviewWork, issue, snapshot NativeSnapshot) {
	t.Helper()
	ctx := t.Context()
	actor, err := identities.LocalActor(ctx, "12")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identities.registry.CollaborationActorByExternalID(ctx, identities.host, "13"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("new discussion participant was already allocated", err)
	}
	work := factory.MergeWork{MergeID: factory.NewID(), PublicationID: factory.NewID(), Repository: canonical.Repository, ActorID: int64(actor), PRAuthorID: canonical.PRAuthorID, ReviewerID: canonical.ActorID, Issue: 3, IssueID: canonical.IssueID, PRID: canonical.PRID, PRNumber: canonical.PRNumber, HeadRef: canonical.HeadRef, BaseRef: canonical.BaseRef, HeadOID: canonical.HeadOID, BaseOID: canonical.BaseOID}
	original, _ := json.Marshal(work)
	const rootBody = "Keep the approved boundary."
	const replyBody = "The correction preserves it."
	const summary = "Recheck the exact candidate."
	var contextCase atomic.Int64
	originalAnchor := func() *extensions.RepositoryInlineAnchor {
		which := contextCase.Load()
		if which == 0 {
			return nil
		}
		a := &extensions.RepositoryInlineAnchor{Base: work.BaseOID, Head: work.HeadOID, Path: "src/main.go", Side: "new", Scope: "lines", StartLine: 4, EndLine: 8}
		switch which {
		case 1, 2:
			a.Scope, a.StartLine, a.EndLine = "file", 0, 0
		case 5:
			a.StartLine = a.EndLine
		}
		if which == 1 || which == 3 {
			a.Side = "old"
		}
		return a
	}
	recorded := func() (string, int64) {
		if which := contextCase.Load(); which == 1 || which == 2 {
			return "", 0
		}
		return "new", 8
	}
	conversation := func(resolve bool) string {
		side, line := recorded()
		state := "resolved"
		if contextCase.Load() == 5 && !resolve {
			state = "open"
		}
		location := extensions.RenderThreadLocation("src/main.go", side, line)
		digest, err := extensions.ResolveConversationDigest([]string{
			extensions.ResolveConversationLine(extensions.ResolveConversationMember{ID: 41, Body: rootBody, ContentVer: 2, Location: location, ResolverState: state, OriginalAnchor: originalAnchor()}),
			extensions.ResolveConversationLine(extensions.ResolveConversationMember{ID: 42, Body: replyBody, ContentVer: 1, Location: location, ResolverState: "open"}),
		})
		if err != nil {
			t.Fatal(err)
		}
		return digest
	}
	comment := func(id, poster, body string, version int) CommentEvidence {
		return CommentEvidence{ID: id, IssueID: "21", PosterID: poster, ReviewID: "31", Type: "code", Content: body, ContentDigest: ContentDigest(body), ContentVersion: version, Visible: true, Complete: true}
	}
	var mode, nativeReads, restReads atomic.Int64
	var afterReview atomic.Bool
	peer := &fakeBackgroundServer{t: t, admission: strings.Repeat("a", 43)}
	peer.revision = func() int64 {
		if mode.Load() == 5 && afterReview.Load() {
			return 13
		}
		return 12
	}
	peer.snapshot = func(request extensions.SnapshotRequest) extensions.NativeSnapshot {
		nativeReads.Add(1)
		if request.RepositoryID != "7" || request.ActorID != "12" {
			t.Error("canonical delivery scope reached native snapshot")
		}
		if len(request.CommentIDs) != 0 {
			if strings.Join(request.CommentIDs, ",") != "41,42" {
				t.Error("native discussion member locators changed", request.CommentIDs)
			}
			items := []CommentEvidence{comment("41", "11", rootBody, 2), comment("42", "13", replyBody, 1)}
			if mode.Load() == 1 {
				items = items[:1]
			}
			return extensions.NativeSnapshot{RepositoryID: "7", Comments: &CommentPage{IssueID: "21", Items: items, Total: len(items), Complete: true}}
		}
		if request.IssueIndex != "" {
			i := *issue.Issue
			i.Title, i.Content, i.TitleDigest, i.ContentDigest, i.Complete = "Native delivery PR", "Exact requested change", ContentDigest("Native delivery PR"), ContentDigest("Exact requested change"), true
			items := []CommentEvidence{comment("61", "9", "Owner context", 1), comment("63", "", "Unattributed native note", 1)}
			for index := range items {
				items[index].Type, items[index].ReviewID = "comment", ""
			}
			return extensions.NativeSnapshot{RepositoryID: "7", Issue: &i, Comments: &CommentPage{IssueID: "21", Items: items, Total: len(items), Complete: true}}
		}
		pull := *snapshot.Pull
		pull.Complete = true
		if mode.Load() == 7 {
			pull.HeadRepoID = "8"
		}
		refs := append([]RefEvidence(nil), snapshot.Refs...)
		for i := range refs {
			refs[i].Complete = true
		}
		review := ReviewEvidence{ID: "31", IssueID: "21", ReviewerID: "11", Type: "REQUEST_CHANGES", CommitID: work.HeadOID, ContentDigest: ContentDigest(summary), Official: true, Visible: true, Complete: true}
		if mode.Load() == 6 {
			review = ReviewEvidence{ID: "31", IssueID: "21", Complete: true}
		}
		side, line := recorded()
		resolved, resolver := contextCase.Load() != 5, "11"
		if !resolved {
			resolver = ""
		}
		thread := ThreadEvidence{RootID: "41", IssueID: "21", ReviewID: "31", Path: "src/main.go", Side: side, Line: line, OriginalAnchor: originalAnchor(), OriginalCommit: work.HeadOID, HeadCommit: work.HeadOID, Resolved: resolved, ResolverID: resolver, Digest: conversation(false), MemberTotal: 2, Visible: true, Complete: true, Members: []ThreadMemberEvidence{
			{ID: "41", IssueID: "21", OriginalAnchor: originalAnchor(), Resolved: resolved, ContentDigest: ContentDigest(rootBody), ContentVersion: 2, Visible: true, Complete: true},
			{ID: "42", IssueID: "21", ContentDigest: ContentDigest(replyBody), ContentVersion: 1, Visible: true, Complete: true},
		}}
		switch mode.Load() {
		case 9:
			thread.Members[1].OriginalAnchor = originalAnchor()
		case 10:
			thread.Digest = strings.Repeat("f", 64)
		case 11:
			thread.Members[1].Resolved = true
		case 12:
			thread.OriginalAnchor, thread.Members[0].OriginalAnchor = nil, nil
		case 13:
			thread.Members[1].ID = thread.RootID
		}
		return extensions.NativeSnapshot{RepositoryID: "7", Pull: &pull, Refs: refs,
			Reviews:     &ReviewPage{IssueID: "21", Items: []ReviewEvidence{review}, Total: 1, Complete: true},
			Threads:     &ThreadPage{IssueID: "21", PullNumber: "5", Items: []ThreadEvidence{thread}, Total: 1, Complete: true},
			Eligibility: &MergeEligibility{PullNumber: "5", HeadTip: work.HeadOID, BaseTip: work.BaseOID, Blocker: extensions.MergeBlockerRequestedChanges, BlockingReviewIDs: []string{"31"}, Visible: true, Complete: true},
		}
	}
	rest := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, in *http.Request) {
		restReads.Add(1)
		if in.Method != http.MethodGet || in.Header.Get("Authorization") != "token test-pat" {
			t.Error("delivery REST read lost its private credential")
			http.Error(out, "invalid fixture authority", 403)
			return
		}
		out.Header().Set("Content-Type", "application/json")
		switch in.URL.Path {
		case "/api/v1/user":
			id := int64(12)
			if mode.Load() == 8 {
				id = 9
			}
			_ = json.NewEncoder(out).Encode(User{ID: id, Login: "native-merger"})
		case "/api/v1/repositories/7":
			id := int64(7)
			if mode.Load() == 2 {
				id = 8
			}
			_ = json.NewEncoder(out).Encode(Repository{ID: id, Owner: User{Login: "native-owner"}, Name: "native-repository", FullName: "native-owner/native-repository"})
		case "/api/v1/repos/native-owner/native-repository/pulls/5/reviews/31":
			user, body := int64(11), summary
			if mode.Load() == 3 {
				user = 9
			}
			if mode.Load() == 4 {
				body = "Edited native review"
			}
			afterReview.Store(true)
			_ = json.NewEncoder(out).Encode(map[string]any{"id": 31, "body": body, "user": map[string]any{"id": user}})
		default:
			t.Error("canonical or unexpected delivery REST locator", in.URL.Path)
			http.NotFound(out, in)
		}
	}))
	t.Cleanup(rest.Close)
	credential := observationCredential(t, "test-pat")
	merger := NewMerger(NewServiceBackground(serveBackgroundSocket(t, peer), uint32(os.Getuid()), ""), New(rest.URL), credential, identities)
	material, err := merger.ReadDeliveryMaterial(ctx, work)
	if err != nil {
		t.Fatal("mapped native discussion unavailable", err, "snapshots", nativeReads.Load(), "REST reads", restReads.Load())
	}
	participant, err := identities.registry.CollaborationActorByExternalID(ctx, identities.host, "13")
	if err != nil || participant.ID == 13 {
		t.Fatal("authenticated discussion did not observe its new canonical participant", participant, err)
	}
	if material.Repository != work.Repository || material.Revision != 12 || material.PRID != 17 || material.IssueID != 21 || material.Head != work.HeadOID || material.Base != work.BaseOID || material.Body != "Exact requested change" || len(material.Comments) != 4 || len(material.Reviews) != 1 || len(material.Threads) != 1 {
		t.Fatal("delivery material lost its canonical scope or exact native target", material)
	}
	byID := map[string]factory.DeliveryComment{}
	for _, item := range material.Comments {
		byID[item.ID] = item
	}
	if byID["61"].PosterID != strconv.FormatInt(work.PRAuthorID, 10) || byID["63"].PosterID != "" || byID["41"].PosterID != strconv.FormatInt(work.ReviewerID, 10) || byID["42"].PosterID != strconv.FormatInt(int64(participant.ID), 10) || byID["41"].Content != rootBody || byID["41"].Digest != ContentDigest(rootBody) || byID["41"].ContentVersion == nil || *byID["41"].ContentVersion != 2 || byID["42"].Content != replyBody {
		t.Fatal("discussion bodies/native IDs or canonical participants changed", byID)
	}
	review, thread := material.Reviews[0], material.Threads[0]
	if review.ActorID != strconv.FormatInt(work.ReviewerID, 10) || review.ID != "31" || review.Content != summary || review.Digest != ContentDigest(summary) || thread.RootID != "41" || thread.ResolverID != review.ActorID || !thread.Resolved || thread.Digest != conversation(false) || len(thread.Members) != 2 || thread.Members[0].ID != "41" || thread.Members[1].ID != "42" || material.Eligible || material.Blocker != extensions.MergeBlockerRequestedChanges || strings.Join(material.BlockingReviewIDs, ",") != "31" {
		t.Fatal("canonical approval/resolver attribution changed native discussion facts", material)
	}
	if review.Official == nil || !*review.Official || review.Stale == nil || *review.Stale || byID["41"].Invalidated == nil || *byID["41"].Invalidated || thread.Members[1].Resolved == nil || *thread.Members[1].Resolved {
		t.Fatal("observed native metadata became unknown or changed", material)
	}
	for which := int64(0); which <= 5; which++ {
		contextCase.Store(which)
		material, err := merger.ReadDeliveryMaterial(ctx, work)
		if err != nil || len(material.Threads) != 1 {
			t.Fatal("full-context delivery read failed", which, err)
		}
		thread := material.Threads[0]
		if !thread.OriginalAnchor.Equal(deliveryThreadAnchor(originalAnchor())) || !thread.Members[0].OriginalAnchor.Equal(thread.OriginalAnchor) || thread.Members[1].OriginalAnchor != nil {
			t.Fatal("immutable or unknown member context changed", which, thread)
		}
		actual, err := thread.ConversationDigest(material.Comments, false)
		if err != nil || actual != conversation(false) || actual != thread.Digest {
			t.Fatal("canonical/native complete context digests diverged", which, err)
		}
		predicted, err := thread.ConversationDigest(material.Comments, true)
		if err != nil || predicted != conversation(true) {
			t.Fatal("canonical/native root-resolution digests diverged", which, err)
		}
	}
	contextCase.Store(4)
	current, _ := json.Marshal(work)
	if !bytes.Equal(current, original) {
		t.Fatal("native projection changed durable merge work")
	}
	beforeNative, beforeREST := nativeReads.Load(), restReads.Load()
	for _, scope := range [][4]int64{{1, work.ActorID, work.PRAuthorID, work.ReviewerID}, {7, work.ActorID, work.PRAuthorID, work.ReviewerID}, {work.Repository, 1, work.PRAuthorID, work.ReviewerID}, {work.Repository, work.ActorID, 2, work.ReviewerID}, {work.Repository, work.ActorID, work.PRAuthorID, 1}} {
		foreign := work
		foreign.Repository, foreign.ActorID, foreign.PRAuthorID, foreign.ReviewerID = scope[0], scope[1], scope[2], scope[3]
		if _, err := merger.ReadDeliveryMaterial(ctx, foreign); err == nil || nativeReads.Load() != beforeNative || restReads.Load() != beforeREST {
			t.Fatal("foreign/unmapped delivery scope reached native access", scope, err)
		}
	}
	for _, unbound := range []*Merger{nil, NewMerger(merger.background, merger.rest, credential, nil)} {
		if _, err := unbound.ReadDeliveryMaterial(ctx, work); err == nil || nativeReads.Load() != beforeNative || restReads.Load() != beforeREST {
			t.Fatal("unbound delivery reader reached native access", err)
		}
	}
	for state, reason := range map[int64]string{1: "missing member", 2: "wrong repository echo", 3: "wrong summary author", 4: "edited summary", 5: "revision moved", 6: "hidden review", 7: "foreign head repository", 8: "credential owner changed", 9: "unknown context invented", 10: "conversation digest changed", 11: "reply resolver state changed", 12: "original root context removed", 13: "duplicate member"} {
		mode.Store(state)
		afterReview.Store(false)
		if _, err := merger.ReadDeliveryMaterial(ctx, work); err == nil {
			t.Fatal("changed delivery material accepted", reason)
		}
	}
	mode.Store(0)
	if err := os.Remove(credential); err != nil {
		t.Fatal(err)
	}
	beforeNative, beforeREST = nativeReads.Load(), restReads.Load()
	if _, err := merger.ReadDeliveryMaterial(ctx, work); err == nil || nativeReads.Load() != beforeNative || restReads.Load() != beforeREST {
		t.Fatal("credential withdrawal reached delivery material", err)
	}
}

func TestReviewerLostSubmitReply(t *testing.T) {
	identities := graphIdentityFixture(t, 7, 11, 9)
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
	r := NewReviewer(NewServiceBackground(socket, uint32(os.Getuid()), ""), observationREST(t, 11), observationCredential(t, "test-pat"), identities)
	w = canonicalReviewWork(t, identities, w)
	got, err := r.SubmitReview(context.Background(), w)
	if err != nil {
		t.Fatal(err)
	}
	adopted, err := r.AdoptReview(t.Context(), w, got)
	if err != nil || adopted.ReviewID != 31 || adopted.ReviewerID != w.ActorID || adopted.Operation.ActorID != w.ActorID || adopted.Operation.RepositoryID != w.Repository || !bytes.Equal(adopted.Operation.Receipt, fixture.Receipt) || fake.submits != 1 {
		t.Fatalf("lost reply did not adopt one existing operation: %+v %v calls=%d", adopted, err, fake.submits)
	}
}
