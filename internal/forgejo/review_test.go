package forgejo

import (
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

func reviewTestOutcome(t *testing.T, w factory.ReviewWork) factory.OperationOutcome {
	t.Helper()
	raw, err := json.Marshal(reviewReceipt{ReviewID: 31, CommentID: 41, ReviewerID: w.ActorID, IssueID: w.IssueID, PRID: w.PRID, PRNumber: w.PRNumber, HeadOID: w.HeadOID, BaseOID: w.BaseOID, CommitID: w.HeadOID, Event: w.Event})
	if err != nil {
		t.Fatal(err)
	}
	return factory.OperationOutcome{OperationID: w.OperationID, InstallationID: "installation", Kind: factory.OpReviewSubmit, ActorID: w.ActorID, RepositoryID: w.Repository, Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Receipt: raw}
}

func TestReviewerReceipt(t *testing.T) {
	w := reviewTestWork()
	r := &Reviewer{}
	for _, event := range []string{"REQUEST_CHANGES", "APPROVED"} {
		w.Event = event
		o := reviewTestOutcome(t, w)
		adopted, err := r.AdoptReview(w, o)
		if err != nil || adopted.HeadOID != w.HeadOID || adopted.Event != event || adopted.ReviewID != 31 {
			t.Fatalf("adopt: %+v %v", adopted, err)
		}
	}
	cases := map[string]func(*factory.ReviewWork, *factory.OperationOutcome){
		"new head":        func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.HeadOID = strings.Repeat("c", 40) },
		"new base":        func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.BaseOID = strings.Repeat("c", 40) },
		"new actor":       func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.ActorID++ },
		"new pr":          func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.PRID++ },
		"new issue":       func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.IssueID++ },
		"new operation":   func(w *factory.ReviewWork, _ *factory.OperationOutcome) { w.OperationID += "-new" },
		"wrong kind":      func(_ *factory.ReviewWork, o *factory.OperationOutcome) { o.Kind = factory.OpPRCreate },
		"missing receipt": func(_ *factory.ReviewWork, o *factory.OperationOutcome) { o.Receipt = nil },
		"unknown field": func(_ *factory.ReviewWork, o *factory.OperationOutcome) {
			o.Receipt = append(o.Receipt[:len(o.Receipt)-1], []byte(`,"unexpected":true}`)...)
		},
		"pending": func(_ *factory.ReviewWork, o *factory.OperationOutcome) { o.Effect = factory.OpEffectPending },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			w := reviewTestWork()
			o := reviewTestOutcome(t, w)
			change(&w, &o)
			if _, err := r.AdoptReview(w, o); err == nil {
				t.Fatal("adopted mismatched review")
			}
		})
	}
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

func TestReviewerLookupAfterCredentialLoss(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	bg := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	credential := observationCredential(t, "test-pat")
	r := NewReviewer(bg, observationREST(t, 11, Repository{}, nil), credential)
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
	r := NewReviewer(bg, observationREST(t, 12, Repository{}, nil), observationCredential(t, "test-pat"))
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
	r := NewReviewer(NewServiceBackground(socket, uint32(os.Getuid()), ""), observationREST(t, 11, Repository{}, nil), observationCredential(t, "test-pat"))
	got, err := r.SubmitReview(context.Background(), w)
	if err != nil {
		t.Fatal(err)
	}
	adopted, err := r.AdoptReview(w, got)
	if err != nil || adopted.ReviewID != 31 || fake.submits != 1 {
		t.Fatalf("lost reply did not adopt one existing operation: %+v %v calls=%d", adopted, err, fake.submits)
	}
}
