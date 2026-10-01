package control_test

// Native review primitive proof. Fixture candidates and PRs are prepared through
// ordinary native writes. This exercises conditional reviews, not Project CLI runs.
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostpublish "github.com/levitateos/sodaos/internal/host/publish"
)

type nativeST10Config struct {
	nativeST09Config
	ReviewerID        int64  `json:"reviewer_id"`
	ReviewerTokenID   int64  `json:"reviewer_token_id"`
	ReviewerTokenFile string `json:"reviewer_token_file"`
}

func nativeReviewConfig(t *testing.T) nativeST10Config {
	t.Helper()
	if os.Getenv("SODA_ST10_NATIVE") == "" {
		t.Skip("native review proof NOT RUN: SODA_ST10_NATIVE not configured")
	}
	var c nativeST10Config
	nativeMust(t, json.Unmarshal([]byte(os.Getenv("SODA_ST10_NATIVE")), &c))
	if c.ActorID <= 0 || c.ReviewerID <= 0 || c.ReviewerID == c.ActorID || c.ReviewerTokenFile == "" {
		t.Fatal("native review needs separate enrolled actor")
	}
	return c
}

func TestNativeReviewWire(t *testing.T) {
	c := nativeReviewConfig(t)
	ctx := context.Background()
	n := nativeNewCandidate(t, c.nativeST09Config)
	branch := "st10-wire-" + factory.NewID()
	nativeGitOK(t, c.nativeST09Config, n.dir, "push", nativeRepoURL(c.nativeST09Config), n.head+":refs/heads/"+branch)
	var pr struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c.nativeST09Config, c.TokenFile, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/pulls", map[string]any{"head": branch, "base": "main", "title": "ST10 native primitive", "body": "Exact review fixture", "allow_maintainer_edit": false}, &pr)
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	var revision extensions.NativeRevisionObservation
	for i := 0; i < 50; i++ {
		var err error
		revision, err = bg.ReadNativeRevision(ctx)
		nativeMust(t, err)
		if revision.Idle {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !revision.Idle {
		t.Fatal("native revision remained busy")
	}
	payload := extensions.ReviewSubmitPayload{PullRequestNumber: pr.Number, PRAuthorID: c.ActorID, HeadRepositoryID: c.Repository, HeadRef: "refs/heads/" + branch, BaseRef: c.BaseBranch, ExpectedHeadOID: n.head, ExpectedBaseOID: n.base, CommitID: n.head, Event: extensions.ReviewSubmitEventRequestChanges, Body: "README.md: replace fixture text with the required corrected text."}
	raw, err := json.Marshal(payload)
	nativeMust(t, err)
	intent := extensions.OperationIntent{OperationID: "st10-wire-" + factory.NewID(), ActorID: strconv.FormatInt(c.ReviewerID, 10), RepositoryID: strconv.FormatInt(c.Repository, 10), Kind: factory.OpReviewSubmit, AuthorizationRevision: "st10-native-primitive", ExpectedNativeRevision: revision.Revision, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Payload: raw}
	record, err := bg.SubmitOperation(ctx, extensions.CredentialFile(c.ReviewerTokenFile), intent)
	nativeMust(t, err)
	nativeReceipt(t, "review-wire", record)
	if record.EffectState != factory.OpEffectCommitted {
		t.Fatalf("native review did not commit: effect=%s reason=%s", record.EffectState, record.ReasonCode)
	}
	var fields map[string]any
	nativeMust(t, json.Unmarshal(record.Receipt, &fields))
	t.Logf("receipt wire: %s", fmt.Sprint(fields))
}

func nativeReviewObserved(t *testing.T, r *forgejo.Reviewer, w factory.ReviewWork) factory.ReviewWork {
	t.Helper()
	for i := 0; i < 40; i++ {
		observed, err := r.ObserveReview(context.Background(), w)
		if err == nil {
			w.NativeRev = observed.NativeRev
			w.NotAfter = time.Now().Add(5 * time.Minute).Unix()
			return w
		}
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || (wait.Reason != "native_busy" && wait.Reason != "revision_moved") {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("native review observation did not settle")
	return factory.ReviewWork{}
}

func TestNativeReviewPrimitive(t *testing.T) {
	c := nativeReviewConfig(t)
	ctx := context.Background()
	n := nativeNewCandidate(t, c.nativeST09Config)
	branch := "st10-review-" + factory.NewID()
	nativeGitOK(t, c.nativeST09Config, n.dir, "push", nativeRepoURL(c.nativeST09Config), n.head+":refs/heads/"+branch)
	var pr struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c.nativeST09Config, c.TokenFile, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/pulls", map[string]any{"head": branch, "base": "main", "title": "ST10 review/fix primitive", "body": "Native review scope; fixture Git supplies candidate bytes."}, &pr)
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	snapshot, err := bg.ReadSnapshot(ctx, extensions.CredentialFile(c.ReviewerTokenFile), extensions.SnapshotRequest{RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ReviewerID, 10), IssueIndex: strconv.FormatInt(pr.Number, 10), Families: []string{"issue"}})
	nativeMust(t, err)
	if snapshot.Issue == nil {
		t.Fatal("native PR issue not observed")
	}
	issueID, err := strconv.ParseInt(snapshot.Issue.ID, 10, 64)
	nativeMust(t, err)
	for _, family := range []string{"pull", "reviews", "refs"} {
		req := extensions.SnapshotRequest{RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ReviewerID, 10), PullNumber: strconv.FormatInt(pr.Number, 10), Families: []string{family}}
		if family == "refs" {
			req.Refs = []string{"refs/heads/" + branch, c.BaseBranch}
		}
		observed, readErr := bg.ReadSnapshot(ctx, extensions.CredentialFile(c.ReviewerTokenFile), req)
		if readErr != nil {
			var status *hostpublish.StatusError
			if errors.As(readErr, &status) {
				t.Logf("native snapshot %s status=%d reason=%s", family, status.Status, status.Body)
			} else {
				t.Logf("native snapshot %s: %v", family, readErr)
			}
		} else {
			nativeReceipt(t, "primitive-snapshot-"+family, observed)
		}
	}
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	w := factory.ReviewWork{OperationID: "st10-review-" + factory.NewID(), AuthRevision: "st10-native-primitive", Repository: c.Repository, ActorID: c.ReviewerID, PRNumber: pr.Number, PRID: pr.ID, IssueID: issueID, PRAuthorID: c.ActorID, HeadRef: "refs/heads/" + branch, BaseRef: c.BaseBranch, HeadOID: n.head, BaseOID: n.base, Event: "REQUEST_CHANGES", Body: "README.md: fixture candidate must be corrected before approval."}
	w = nativeReviewObserved(t, r, w)
	nativeReceipt(t, "primitive-first-intent", w)
	first, err := r.SubmitReview(ctx, w)
	nativeMust(t, err)
	adopted, err := r.AdoptReview(w, first)
	nativeMust(t, err)
	nativeReceipt(t, "primitive-first-review", adopted)
	if first.Completion != factory.OpCompletionComplete || adopted.ReviewerID != c.ReviewerID {
		t.Fatal("independent first review incomplete")
	}

	corrected := nativeAppendCandidate(t, c.nativeST09Config, n)
	nativeGitOK(t, c.nativeST09Config, corrected.dir, "push", nativeRepoURL(c.nativeST09Config), corrected.head+":"+w.HeadRef)
	if _, err = r.ObserveReview(ctx, w); err == nil {
		t.Fatal("old head passed observation")
	}
	next := w
	next.OperationID = "st10-review-" + factory.NewID()
	next.HeadOID = corrected.head
	next.Event = "APPROVED"
	next.Body = "The revised fixture candidate resolves the README finding."
	next = nativeReviewObserved(t, r, next)
	stale := w
	stale.OperationID = "st10-stale-" + factory.NewID()
	stale.NativeRev = next.NativeRev
	stale.NotAfter = next.NotAfter
	nativeReceipt(t, "primitive-stale-intent", stale)
	staleOut, err := r.SubmitReview(ctx, stale)
	nativeMust(t, err)
	nativeReceipt(t, "primitive-stale-review", staleOut)
	if staleOut.Effect != factory.OpEffectNotCommitted || staleOut.Reason != "stale_head" {
		t.Fatalf("stale head was not rejected: %s/%s", staleOut.Effect, staleOut.Reason)
	}
	next = nativeReviewObserved(t, r, next)
	nativeReceipt(t, "primitive-fresh-intent", next)
	fresh, err := r.SubmitReview(ctx, next)
	nativeMust(t, err)
	approved, err := r.AdoptReview(next, fresh)
	nativeMust(t, err)
	nativeReceipt(t, "primitive-fresh-review", approved)
	if approved.ReviewID == adopted.ReviewID || approved.HeadOID == adopted.HeadOID || approved.Event != "APPROVED" || fresh.Completion != factory.OpCompletionComplete {
		t.Fatal("fresh review did not bind corrected candidate")
	}
	if _, err = r.AdoptReview(next, first); err == nil {
		t.Fatal("old review approved the changed head")
	}
	looked, err := r.LookupOp(ctx, w.OperationID)
	nativeMust(t, err)
	old, err := r.AdoptReview(w, looked)
	nativeMust(t, err)
	if old.ReviewID != adopted.ReviewID || old.HeadOID != n.head {
		t.Fatal("old native review lost historical identity")
	}

	cancelled, err := r.CancelOp(ctx, w.OperationID)
	nativeMust(t, err)
	nativeReceipt(t, "primitive-review-cancel-too-late", cancelled)
	if cancelled.Cancellation != factory.OpCancelTooLate || cancelled.Effect != factory.OpEffectCommitted {
		t.Fatal("committed review cancellation rewrote history")
	}
	t.Log("PASS native REQUEST_CHANGES -> fixture coder commit -> stale refusal -> fresh APPROVED; this is not Project CLI proof")
}

// TestNativeReviewStaleHeadDirect proves the native stale-head gate and the
// Soda receipt binding without pull snapshots: an intent naming the old head
// after a correction push does not commit and creates no review, while the
// exact new-head intent commits and adopts. Observation-driven early refusal
// stays with TestNativeReviewPrimitive once pull snapshots are repaired.
func TestNativeReviewStaleHeadDirect(t *testing.T) {
	c := nativeReviewConfig(t)
	ctx := context.Background()
	n := nativeNewCandidate(t, c.nativeST09Config)
	branch := "st10-stale-" + factory.NewID()
	nativeGitOK(t, c.nativeST09Config, n.dir, "push", nativeRepoURL(c.nativeST09Config), n.head+":refs/heads/"+branch)
	var pr struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c.nativeST09Config, c.TokenFile, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/pulls", map[string]any{"head": branch, "base": "main", "title": "ST10 stale direct", "body": "Old-head refusal without snapshots.", "allow_maintainer_edit": false}, &pr)
	var issue struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c.nativeST09Config, c.TokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(pr.Number, 10), nil, &issue)
	if issue.Number != pr.Number || issue.ID <= 0 {
		t.Fatal("native PR issue identity unconfirmed")
	}
	corrected := nativeAppendCandidate(t, c.nativeST09Config, n)
	nativeGitOK(t, c.nativeST09Config, corrected.dir, "push", nativeRepoURL(c.nativeST09Config), corrected.head+":refs/heads/"+branch)
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	var revision extensions.NativeRevisionObservation
	for i := 0; i < 50; i++ {
		var err error
		revision, err = bg.ReadNativeRevision(ctx)
		nativeMust(t, err)
		if revision.Idle {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !revision.Idle {
		t.Fatal("native revision remained busy")
	}
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	stale := factory.ReviewWork{OperationID: "st10-stale-" + factory.NewID(), AuthRevision: "st10-stale-direct", Repository: c.Repository, ActorID: c.ReviewerID, PRNumber: pr.Number, PRID: pr.ID, IssueID: issue.ID, PRAuthorID: c.ActorID, HeadRef: "refs/heads/" + branch, BaseRef: c.BaseBranch, HeadOID: n.head, BaseOID: n.base, NativeRev: revision.Revision, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Event: "REQUEST_CHANGES", Body: "README.md: finding against the superseded head."}
	staleOut, err := r.SubmitReview(ctx, stale)
	nativeMust(t, err)
	nativeReceipt(t, "stale-direct-refusal", staleOut)
	if staleOut.Effect != factory.OpEffectNotCommitted || staleOut.Reason != "stale_head" {
		t.Fatalf("old head was not refused as stale: %s/%s", staleOut.Effect, staleOut.Reason)
	}
	// The refused submit still advances the native revision; the exact
	// new-head intent must name the current idle revision, as production
	// observation does before every submit.
	for i := 0; i < 50; i++ {
		var err error
		revision, err = bg.ReadNativeRevision(ctx)
		nativeMust(t, err)
		if revision.Idle {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !revision.Idle {
		t.Fatal("native revision remained busy after stale refusal")
	}
	fresh := stale
	fresh.OperationID = "st10-stale-" + factory.NewID()
	fresh.HeadOID = corrected.head
	fresh.NativeRev = revision.Revision
	fresh.Event = "APPROVED"
	fresh.Body = "The corrected head resolves the finding."
	freshOut, err := r.SubmitReview(ctx, fresh)
	nativeMust(t, err)
	nativeReceipt(t, "stale-direct-fresh-outcome", freshOut)
	approved, err := r.AdoptReview(fresh, freshOut)
	nativeMust(t, err)
	nativeReceipt(t, "stale-direct-fresh", approved)
	if freshOut.Completion != factory.OpCompletionComplete || approved.HeadOID != corrected.head || approved.Event != "APPROVED" {
		t.Fatal("exact new-head review did not adopt")
	}
	if _, err = r.AdoptReview(stale, freshOut); err == nil {
		t.Fatal("new-head review approved the old head")
	}
	if _, err = r.AdoptReview(fresh, staleOut); err == nil {
		t.Fatal("refused old-head outcome approved the new head")
	}
	listed, err := bg.ReadSnapshot(ctx, extensions.CredentialFile(c.ReviewerTokenFile), extensions.SnapshotRequest{RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ReviewerID, 10), PullNumber: strconv.FormatInt(pr.Number, 10), Families: []string{"reviews"}})
	nativeMust(t, err)
	nativeReceipt(t, "stale-direct-reviews", listed)
	if listed.Reviews == nil || len(listed.Reviews.Items) != 1 || listed.Reviews.Items[0].ReviewerID != strconv.FormatInt(c.ReviewerID, 10) || listed.Reviews.Items[0].CommitID != corrected.head {
		t.Fatal("native review history does not show exactly the fresh review")
	}
	t.Log("PASS old-head direct submit refused and created no review; exact new-head review committed and adopted")
}

func TestNativeReviewRoleBinding(t *testing.T) {
	c := nativeReviewConfig(t)
	ctx := context.Background()
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	revision, err := bg.ReadNativeRevision(ctx)
	nativeMust(t, err)
	payload, err := json.Marshal(extensions.PublishPayload{Ref: "refs/heads/st10-reviewer-forbidden", ExpectedOld: "absent", NewOID: strings.Repeat("a", 40), ComparisonRef: c.BaseBranch, ExpectedComparisonOID: strings.Repeat("b", 40)})
	nativeMust(t, err)
	intent := extensions.OperationIntent{OperationID: "st10-role-" + factory.NewID(), ActorID: strconv.FormatInt(c.ReviewerID, 10), RepositoryID: strconv.FormatInt(c.Repository, 10), Kind: factory.OpRefPublish, AuthorizationRevision: "st10-native-role", ExpectedNativeRevision: revision.Revision, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Payload: payload}
	_, err = bg.SubmitOperation(ctx, extensions.CredentialFile(c.ReviewerTokenFile), intent)
	var denied *hostpublish.StatusError
	if !errors.As(err, &denied) || denied.Status != http.StatusForbidden {
		t.Fatal("review actor was not denied publication binding")
	}
	intent.ActorID = strconv.FormatInt(c.ActorID, 10)
	_, err = bg.SubmitOperation(ctx, extensions.CredentialFile(c.ReviewerTokenFile), intent)
	if !errors.As(err, &denied) || denied.Status != http.StatusForbidden {
		t.Fatal("review token impersonated publisher")
	}
	if nativeTip(t, c.nativeST09Config, "refs/heads/st10-reviewer-forbidden") != "" {
		t.Fatal("review actor published a branch")
	}
	nativeReceipt(t, "primitive-role-binding", map[string]any{"reviewer_id": c.ReviewerID, "publisher_id": c.ActorID, "reviewer_publication": "403 forbidden", "publisher_impersonation": "403 forbidden", "branch_absent": true})
}
