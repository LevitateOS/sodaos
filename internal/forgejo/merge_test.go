package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

func mergeTestWork() factory.MergeWork {
	return factory.MergeWork{
		MergeID: "abcdef0123456789abcdef0123456789", PublicationID: "1234567890abcdef1234567890abcdef",
		OperationID: "soda-1234567890abcdef1234567890abcdef-merge-1", AuthRevision: "attempt:merge:1",
		HeadRef: "refs/heads/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("a", 40), BaseOID: strings.Repeat("b", 40),
		Repository: 7, Issue: 3, PRNumber: 5, PRID: 17, IssueID: 21,
		PRAuthorID: 9, ReviewerID: 11, ActorID: 13,
		NativeRev: 12, NotAfter: time.Now().Add(5 * time.Minute).Unix(),
		AssessmentRevision: 2, ReviewID: 31,
	}
}

func mergeTestOutcome(t *testing.T, w factory.MergeWork) factory.OperationOutcome {
	t.Helper()
	raw, err := json.Marshal(mergeReceipt{
		HeadRef: w.HeadRef, BaseRef: w.BaseRef, OldOID: w.BaseOID, NewOID: w.HeadOID,
		Method: factory.MergeFastForward, ActorID: w.ActorID, RepositoryID: w.Repository,
		PRNumber: w.PRNumber, PRID: w.PRID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return factory.OperationOutcome{OperationID: w.OperationID, InstallationID: "installation", Kind: factory.OpMerge, ActorID: w.ActorID, RepositoryID: w.Repository, Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Receipt: raw}
}

func TestMergerReceipt(t *testing.T) {
	var m Merger
	w := mergeTestWork()
	cases := map[string]func(*factory.MergeWork, *factory.OperationOutcome){
		"new head":        func(w *factory.MergeWork, _ *factory.OperationOutcome) { w.HeadOID = strings.Repeat("c", 40) },
		"new base":        func(w *factory.MergeWork, _ *factory.OperationOutcome) { w.BaseOID = strings.Repeat("c", 40) },
		"new actor":       func(w *factory.MergeWork, _ *factory.OperationOutcome) { w.ActorID++ },
		"new pr":          func(w *factory.MergeWork, _ *factory.OperationOutcome) { w.PRID++ },
		"new operation":   func(w *factory.MergeWork, _ *factory.OperationOutcome) { w.OperationID += "-new" },
		"wrong kind":      func(_ *factory.MergeWork, o *factory.OperationOutcome) { o.Kind = factory.OpPRCreate },
		"missing receipt": func(_ *factory.MergeWork, o *factory.OperationOutcome) { o.Receipt = nil },
		"unknown field": func(_ *factory.MergeWork, o *factory.OperationOutcome) {
			o.Receipt = append([]byte(nil), o.Receipt[:len(o.Receipt)-1]...)
			o.Receipt = append(o.Receipt, []byte(`,"extra":1}`)...)
		},
		"pending": func(_ *factory.MergeWork, o *factory.OperationOutcome) { o.Effect = factory.OpEffectPending },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			work, outcome := mergeTestWork(), mergeTestOutcome(t, w)
			mutate(&work, &outcome)
			if _, err := m.AdoptMerge(work, outcome); err == nil {
				t.Fatalf("case %s adopted", name)
			}
		})
	}
	adopted, err := m.AdoptMerge(w, mergeTestOutcome(t, w))
	if err != nil {
		t.Fatalf("exact receipt refused: %v", err)
	}
	if adopted.MergedCommit != w.HeadOID || adopted.PRID != w.PRID || adopted.IssueID != w.IssueID || adopted.ActorID != w.ActorID {
		t.Fatalf("adopted outcome differs: %+v", adopted)
	}
}

func TestMergerIntentFastForwardOnly(t *testing.T) {
	w := mergeTestWork()
	intent, err := mergeIntent(w)
	if err != nil {
		t.Fatalf("exact intent refused: %v", err)
	}
	if intent.Kind != factory.OpMerge || intent.ExpectedNativeRevision != w.NativeRev {
		t.Fatalf("intent differs: %+v", intent)
	}
	var payload extensions.MergePayload
	if err := json.Unmarshal(intent.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if err := extensions.ValidateMergePayload(w.Repository, payload); err != nil {
		t.Fatalf("payload rejected by the SDK contract: %v", err)
	}
	if payload.Method != extensions.MergeMethodFastForwardOnly || payload.ExpectedHeadOID != w.HeadOID || payload.ExpectedBaseOID != w.BaseOID {
		t.Fatalf("payload differs: %+v", payload)
	}
	w.HeadOID = w.BaseOID
	if _, err := mergeIntent(w); err == nil {
		t.Fatal("no-op intent accepted")
	}
}

func mergeTestSnapshots(w factory.MergeWork) (NativeSnapshot, NativeSnapshot) {
	head, base := factory.RefHead(w.HeadRef), factory.RefHead(w.BaseRef)
	issue := NativeSnapshot{Revision: w.NativeRev, RepositoryID: strconv.FormatInt(w.Repository, 10), Issue: &IssueEvidence{
		ID: strconv.FormatInt(w.IssueID, 10), Index: strconv.FormatInt(w.PRNumber, 10), IsPull: true,
		Provenance: CreationProvenance{PosterID: strconv.FormatInt(w.PRAuthorID, 10)},
		Visible:    true, Complete: true,
	}}
	snapshot := NativeSnapshot{Revision: w.NativeRev, RepositoryID: strconv.FormatInt(w.Repository, 10),
		Pull: &PullEvidence{
			ID: strconv.FormatInt(w.PRID, 10), IssueID: strconv.FormatInt(w.IssueID, 10), Number: strconv.FormatInt(w.PRNumber, 10),
			HeadRepoID: strconv.FormatInt(w.Repository, 10), HeadBranch: head, HeadTip: w.HeadOID, BaseBranch: base,
			Visible: true, Complete: true,
		},
		Reviews: &ReviewPage{IssueID: strconv.FormatInt(w.IssueID, 10), Items: []ReviewEvidence{{
			ID: strconv.FormatInt(w.ReviewID, 10), IssueID: strconv.FormatInt(w.IssueID, 10), Type: "APPROVED",
			ReviewerID: strconv.FormatInt(w.ReviewerID, 10), CommitID: w.HeadOID, Official: true,
			Visible: true, Complete: true,
		}}, Total: 1, Complete: true},
		Checks: &CheckSet{SHA: w.HeadOID, Total: 0, Complete: true},
		Refs: []RefEvidence{
			{Ref: w.HeadRef, OID: w.HeadOID, Exists: true, Visible: true, Complete: true},
			{Ref: w.BaseRef, OID: w.BaseOID, Exists: true, Visible: true, Complete: true},
		},
	}
	return issue, snapshot
}

func TestMergerMatchTarget(t *testing.T) {
	w := mergeTestWork()
	issue, snapshot := mergeTestSnapshots(w)
	observed, err := matchMergeTarget(w, issue, snapshot)
	if err != nil {
		t.Fatalf("exact target refused: %v", err)
	}
	if observed.ReviewID != w.ReviewID || observed.NativeRev != w.NativeRev {
		t.Fatalf("observation differs: %+v", observed)
	}
	refusals := map[string]func(NativeSnapshot, NativeSnapshot){
		"closed":        func(i, _ NativeSnapshot) { i.Issue.IsClosed = true },
		"merged":        func(_, s NativeSnapshot) { s.Pull.HasMerged = true },
		"author":        func(i, _ NativeSnapshot) { i.Issue.Provenance.PosterID = "10" },
		"stale head":    func(_, s NativeSnapshot) { s.Pull.HeadTip = strings.Repeat("c", 40) },
		"stale base":    func(_, s NativeSnapshot) { s.Refs[1].OID = strings.Repeat("c", 40) },
		"author review": func(_, s NativeSnapshot) { s.Reviews.Items[0].ReviewerID = strconv.FormatInt(w.PRAuthorID, 10) },
	}
	for name, mutate := range refusals {
		issue, snapshot := mergeTestSnapshots(w)
		mutate(issue, snapshot)
		if _, err := matchMergeTarget(w, issue, snapshot); err == nil {
			t.Errorf("case %s accepted", name)
		} else {
			var refused *factory.PublicationRefusal
			if !errors.As(err, &refused) {
				t.Errorf("case %s did not refuse terminally: %v", name, err)
			}
		}
	}
	// A missing approval waits for evidence instead of failing: it may
	// still arrive on the exact head.
	waits := map[string]func(NativeSnapshot, NativeSnapshot){
		"no approval": func(_, s NativeSnapshot) { s.Reviews.Items = nil },
		"wrong head":  func(_, s NativeSnapshot) { s.Reviews.Items[0].CommitID = strings.Repeat("c", 40) },
		"wrong actor": func(_, s NativeSnapshot) { s.Reviews.Items[0].ReviewerID = "12" },
		"unofficial":  func(_, s NativeSnapshot) { s.Reviews.Items[0].Official = false },
		"dismissed":   func(_, s NativeSnapshot) { s.Reviews.Items[0].Dismissed = true },
	}
	for name, mutate := range waits {
		issue, snapshot := mergeTestSnapshots(w)
		mutate(issue, snapshot)
		if _, err := matchMergeTarget(w, issue, snapshot); err == nil {
			t.Errorf("case %s accepted", name)
		} else {
			var wait *factory.PublicationWait
			if !errors.As(err, &wait) || wait.Reason != "approval_missing" {
				t.Errorf("case %s did not wait for approval: %v", name, err)
			}
		}
	}
	// A live change request on the exact head refuses even with an approval.
	issue, snapshot = mergeTestSnapshots(w)
	snapshot.Reviews.Items = append(snapshot.Reviews.Items, ReviewEvidence{
		ID: "32", IssueID: strconv.FormatInt(w.IssueID, 10), Type: "REQUEST_CHANGES",
		ReviewerID: strconv.FormatInt(w.ReviewerID, 10), CommitID: w.HeadOID, Official: true,
		Visible: true, Complete: true,
	})
	if _, err := matchMergeTarget(w, issue, snapshot); err == nil {
		t.Error("exact-head change request accepted")
	}
	// A stale change request on an older head is historical, not blocking.
	issue, snapshot = mergeTestSnapshots(w)
	snapshot.Reviews.Items = append(snapshot.Reviews.Items, ReviewEvidence{
		ID: "32", IssueID: strconv.FormatInt(w.IssueID, 10), Type: "REQUEST_CHANGES",
		ReviewerID: strconv.FormatInt(w.ReviewerID, 10), CommitID: strings.Repeat("c", 40), Official: true,
		Stale: true, Visible: true, Complete: true,
	})
	if _, err := matchMergeTarget(w, issue, snapshot); err != nil {
		t.Errorf("stale change request refused: %v", err)
	}
	// Hidden evidence waits instead of guessing.
	issue, snapshot = mergeTestSnapshots(w)
	snapshot.Reviews.Items[0].Visible = false
	if _, err := matchMergeTarget(w, issue, snapshot); err == nil {
		t.Error("hidden review accepted")
	} else {
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) {
			t.Errorf("hidden review did not wait: %v", err)
		}
	}
}

func TestMergerMatchConfirmation(t *testing.T) {
	w := mergeTestWork()
	issue, snapshot := mergeTestSnapshots(w)
	issue.Issue.IsClosed, issue.Issue.ClosedUnix = true, 1200
	snapshot.Pull.HasMerged, snapshot.Pull.MergedCommit = true, w.HeadOID
	snapshot.Pull.MergerID, snapshot.Pull.MergedUnix = strconv.FormatInt(w.ActorID, 10), 1190
	snapshot.Refs[1].OID = w.HeadOID
	confirmation, err := matchMergeConfirmation(w, issue, snapshot)
	if err != nil {
		t.Fatalf("exact completion refused: %v", err)
	}
	if confirmation.MergedCommit != w.HeadOID || !confirmation.IssueClosed || confirmation.MergedUnix != 1190 || confirmation.ClosedUnix != 1200 {
		t.Fatalf("confirmation differs: %+v", confirmation)
	}
	refusals := map[string]func(NativeSnapshot, NativeSnapshot){
		"unmerged":     func(_, s NativeSnapshot) { s.Pull.HasMerged = false },
		"other commit": func(_, s NativeSnapshot) { s.Pull.MergedCommit = strings.Repeat("c", 40) },
		"other merger": func(_, s NativeSnapshot) { s.Pull.MergerID = "14" },
		"base moved":   func(_, s NativeSnapshot) { s.Refs[1].OID = strings.Repeat("c", 40) },
		"issue open":   func(i, _ NativeSnapshot) { i.Issue.IsClosed, i.Issue.ClosedUnix = false, 0 },
	}
	for name, mutate := range refusals {
		issue, snapshot := mergeTestSnapshots(w)
		issue.Issue.IsClosed, issue.Issue.ClosedUnix = true, 1200
		snapshot.Pull.HasMerged, snapshot.Pull.MergedCommit = true, w.HeadOID
		snapshot.Pull.MergerID, snapshot.Pull.MergedUnix = strconv.FormatInt(w.ActorID, 10), 1190
		snapshot.Refs[1].OID = w.HeadOID
		mutate(issue, snapshot)
		if _, err := matchMergeConfirmation(w, issue, snapshot); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func TestMergerLookupAfterCredentialLoss(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	bg := NewServiceBackground(serveScriptedBackground(t, fake), uint32(os.Getuid()), "")
	credential := observationCredential(t, "test-pat")
	m := NewMerger(bg, observationREST(t, 13, Repository{}, nil), credential)
	w := mergeTestWork()
	ctx := context.Background()
	pending, err := m.SubmitMerge(ctx, w)
	if err != nil || pending.Effect != factory.OpEffectPending {
		t.Fatalf("submit: %+v %v", pending, err)
	}
	if err = os.Remove(credential); err != nil {
		t.Fatal(err)
	}
	observed, err := m.LookupOp(ctx, w.OperationID)
	if err != nil || observed.OperationID != w.OperationID {
		t.Fatalf("credential-free lookup: %+v %v", observed, err)
	}
	cancelled, err := m.CancelOp(ctx, w.OperationID)
	if err != nil || cancelled.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("credential-free cancellation: %+v %v", cancelled, err)
	}
	absent, err := m.LookupOp(ctx, "st12-absent")
	if err != nil || !absent.NotObserved {
		t.Fatalf("absence: %+v %v", absent, err)
	}
}
