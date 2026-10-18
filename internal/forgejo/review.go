package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// Reviewer submits only exact reviews through the shared native admission:
// the summary body plus the recorded inline finding comments. Its distinct
// actor secret stays outside the Project and its agents.
type Reviewer struct {
	background *ServiceBackground
	rest       *Client
	tokenFile  string
}

func NewReviewer(background *ServiceBackground, rest *Client, tokenFile string) *Reviewer {
	return &Reviewer{background: background, rest: rest, tokenFile: tokenFile}
}

func (r *Reviewer) checkActor(ctx context.Context, actor int64) error {
	if r.background == nil || r.rest == nil || r.tokenFile == "" {
		return &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	token, err := config.Secret(r.tokenFile)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	user, err := loadActor(ctx, r.rest, token)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	if user.ID != actor {
		return &factory.PublicationWait{Reason: "authority_lost"}
	}
	return nil
}

// ObserveReview validates the recorded creation identity and exact branch tips.
// Issue and review snapshot locators are mutually exclusive in the native SDK,
// so their two complete brackets must resolve to the same native revision.
func (r *Reviewer) ObserveReview(ctx context.Context, w factory.ReviewWork) (factory.ReviewObservation, error) {
	var empty factory.ReviewObservation
	if err := w.ValidateTarget(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	if err := r.checkActor(ctx, w.ActorID); err != nil {
		return empty, err
	}
	reader := &BackgroundSnapshotReader{Client: r.background, ActorID: strconv.FormatInt(w.ActorID, 10)}
	credential := extensions.CredentialFile(r.tokenFile)
	repo, number := strconv.FormatInt(w.Repository, 10), strconv.FormatInt(w.PRNumber, 10)
	issue, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, IssueIndex: number, Families: []SnapshotFamily{FamilyIssue}})
	if err != nil {
		return empty, reviewReadError(err)
	}
	snapshot, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, PullNumber: number, Families: []SnapshotFamily{FamilyPull, FamilyReviews, FamilyRefs}, Refs: []string{w.HeadRef, w.BaseRef}})
	if err != nil {
		return empty, reviewReadError(err)
	}
	if snapshot.Revision != issue.Revision {
		return empty, &factory.PublicationWait{Reason: "revision_moved"}
	}
	if err = matchReviewTarget(w, issue, snapshot); err != nil {
		return empty, err
	}
	return factory.ReviewObservation{NativeRev: snapshot.Revision, ObservedUnix: time.Now().Unix(), PRID: w.PRID, PRNumber: w.PRNumber, IssueID: w.IssueID, PRAuthorID: w.PRAuthorID, HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID}, nil
}

func reviewReadError(err error) error {
	if errors.Is(err, ErrNativeBusy) {
		return &factory.PublicationWait{Reason: "native_busy"}
	}
	if errors.Is(err, ErrStaleSnapshot) {
		return &factory.PublicationWait{Reason: "revision_moved"}
	}
	return &factory.PublicationWait{Reason: "review_evidence_unavailable"}
}

func checkReviewIdentity(w factory.ReviewWork, i *IssueEvidence, p *PullEvidence) error {
	if i == nil || p == nil || !i.Visible || !p.Visible || !i.IsPull || i.IsClosed || p.HasMerged ||
		i.ID != strconv.FormatInt(w.IssueID, 10) || i.Provenance.PosterID != strconv.FormatInt(w.PRAuthorID, 10) ||
		p.ID != strconv.FormatInt(w.PRID, 10) || p.IssueID != i.ID || p.HeadRepoID != strconv.FormatInt(w.Repository, 10) ||
		p.HeadBranch != factory.RefHead(w.HeadRef) || p.BaseBranch != factory.RefHead(w.BaseRef) {
		return &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	return nil
}

func checkReviewRefs(snapshot NativeSnapshot, w factory.ReviewWork) error {
	for _, ref := range snapshot.Refs {
		if !ref.Visible || !ref.Exists {
			return &factory.PublicationRefusal{Reason: "pr_mismatch"}
		}
		if ref.Ref == w.HeadRef && ref.OID != w.HeadOID {
			return &factory.PublicationRefusal{Reason: "stale_head"}
		}
		if ref.Ref == w.BaseRef && ref.OID != w.BaseOID {
			return &factory.PublicationRefusal{Reason: "stale_base_or_result"}
		}
	}
	return nil
}

func checkPendingReview(snapshot NativeSnapshot, issueID string, actorID int64) error {
	if snapshot.Reviews == nil || snapshot.Reviews.IssueID != issueID {
		return &factory.PublicationWait{Reason: "review_evidence_unavailable"}
	}
	for _, review := range snapshot.Reviews.Items {
		if !review.Visible || review.IssueID != issueID {
			return &factory.PublicationWait{Reason: "review_evidence_unavailable"}
		}
		if review.ReviewerID == strconv.FormatInt(actorID, 10) && review.Type == "PENDING" {
			return &factory.PublicationRefusal{Reason: "pending_review_exists"}
		}
	}
	return nil
}

func matchReviewTarget(w factory.ReviewWork, issue, snapshot NativeSnapshot) error {
	i, p := issue.Issue, snapshot.Pull
	if err := checkReviewIdentity(w, i, p); err != nil {
		return err
	}
	if p.HeadTip != w.HeadOID {
		return &factory.PublicationRefusal{Reason: "stale_head"}
	}
	if err := checkReviewRefs(snapshot, w); err != nil {
		return err
	}
	return checkPendingReview(snapshot, i.ID, w.ActorID)
}

func reviewIntent(w factory.ReviewWork) (extensions.OperationIntent, error) {
	if err := w.Validate(); err != nil {
		return extensions.OperationIntent{}, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	payload := extensions.ReviewSubmitPayload{PullRequestNumber: w.PRNumber, PRAuthorID: w.PRAuthorID, HeadRepositoryID: w.Repository, HeadRef: w.HeadRef, BaseRef: w.BaseRef, ExpectedHeadOID: w.HeadOID, ExpectedBaseOID: w.BaseOID, CommitID: w.HeadOID, Event: w.Event, Body: w.Body}
	for _, comment := range w.Comments {
		payload.Comments = append(payload.Comments, extensions.ReviewComment{Key: comment.Key, Path: comment.Path, Side: comment.Side, Line: comment.Line, Body: comment.Body})
	}
	// The SDK validation enforces the complete encoded payload ceiling
	// (MaxReviewPayloadBytes), the per-report comment bound and the exact
	// diff-anchor rules against the same limits the native operation
	// enforces at admission.
	if err := extensions.ValidateReviewSubmitPayload(w.Repository, payload); err != nil {
		return extensions.OperationIntent{}, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return extensions.OperationIntent{}, err
	}
	return extensions.OperationIntent{OperationID: w.OperationID, ActorID: strconv.FormatInt(w.ActorID, 10), RepositoryID: strconv.FormatInt(w.Repository, 10), Kind: factory.OpReviewSubmit, AuthorizationRevision: w.AuthRevision, ExpectedNativeRevision: w.NativeRev, NotAfter: w.NotAfter, Payload: raw}, nil
}

// SubmitReview sends the already persisted intent. A failed response is reconciled
// by that operation identity; no ordinary review POST or guessed record is used.
func (r *Reviewer) SubmitReview(ctx context.Context, w factory.ReviewWork) (factory.OperationOutcome, error) {
	intent, err := reviewIntent(w)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	if w.NotAfter <= time.Now().Unix() {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "observation_expired"}
	}
	if err = r.checkActor(ctx, w.ActorID); err != nil {
		return factory.OperationOutcome{}, err
	}
	record, err := r.background.SubmitOperation(ctx, extensions.CredentialFile(r.tokenFile), intent)
	if err != nil {
		observed, lookupErr := r.LookupOp(ctx, w.OperationID)
		if lookupErr == nil && !observed.NotObserved {
			return observed, nil
		}
		return factory.OperationOutcome{}, reviewCallError(err)
	}
	outcome, err := reviewOperationOutcome(w.OperationID, record)
	if err == nil && (outcome.ActorID != w.ActorID || outcome.RepositoryID != w.Repository) {
		err = errors.New("review operation differs from its recorded scope")
	}
	return outcome, err
}

// LookupOp uses installation admission even after the reviewer credential is
// withdrawn. Absence does not prove an earlier submission cannot still arrive.
func (r *Reviewer) LookupOp(ctx context.Context, id string) (factory.OperationOutcome, error) {
	if r.background == nil {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	lookup, err := r.background.GetOperation(ctx, id)
	if err != nil {
		return factory.OperationOutcome{}, reviewCallError(err)
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	if lookup.Record == nil {
		return factory.OperationOutcome{}, errors.New("review lookup lacks its native record")
	}
	return reviewOperationOutcome(id, *lookup.Record)
}

// CancelOp records cancellation under owning-installation admission. A committed
// review remains historical evidence; cancellation cannot erase it.
func (r *Reviewer) CancelOp(ctx context.Context, id string) (factory.OperationOutcome, error) {
	if r.background == nil {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	record, err := r.background.CancelOperation(ctx, id)
	if err != nil {
		observed, lookupErr := r.LookupOp(ctx, id)
		if lookupErr == nil && !observed.NotObserved {
			return observed, nil
		}
		return factory.OperationOutcome{}, reviewCallError(err)
	}
	return reviewOperationOutcome(id, record)
}

func reviewCallError(err error) error {
	var status *forgejopublish.StatusError
	if !errors.As(err, &status) {
		return err
	}
	switch status.Status {
	case 400:
		return &factory.PublicationRefusal{Reason: "invalid_intent"}
	case 401:
		return &factory.PublicationWait{Reason: "credential_invalid"}
	case 403:
		return &factory.PublicationWait{Reason: "authority_lost"}
	case 409:
		if status.Body == "intent_conflict" {
			return &factory.PublicationRefusal{Reason: "intent_conflict"}
		}
	}
	return errors.New("native review operation response unconfirmed")
}

func reviewOutcomeState(id string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if record.OperationID != id || record.InstallationID == "" {
		return empty, errors.New("review operation identity differs")
	}
	if record.Outcome == extensions.BackgroundOutcomeOperationsUnavailable {
		return empty, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	o := factory.OperationOutcome{OperationID: id, InstallationID: record.InstallationID, Kind: record.Kind, Effect: record.EffectState, Cancellation: record.CancellationStatus, Completion: record.CompletionState, Reason: record.ReasonCode}
	if o.Cancellation == "" {
		o.Cancellation = factory.OpCancelNone
	}
	// Review operations admit the review-specific receipt cap: inline
	// comment IDs and the merge base travel in this receipt. Every generic
	// native receipt stays at the 4 KiB bound.
	if !factory.ValidOpEffect(o.Effect) || !factory.ValidOpCancellation(o.Cancellation) || (o.Completion != "" && !factory.ValidOpCompletion(o.Completion)) || len(o.Reason) > 64 || len(record.Receipt) > factory.MaxReviewReceiptBytes {
		return empty, errors.New("invalid native review operation state")
	}
	return o, nil
}

func reviewOutcomeScope(o *factory.OperationOutcome, record extensions.OperationRecord) error {
	tombstone := record.Kind == "" && record.ActorID == "" && record.RepositoryID == "" && o.Effect == factory.OpEffectNotCommitted && o.Cancellation == factory.OpCancelCancelled
	if !tombstone {
		actor, ae := strconv.ParseInt(record.ActorID, 10, 64)
		repo, re := strconv.ParseInt(record.RepositoryID, 10, 64)
		if record.Kind != factory.OpReviewSubmit || ae != nil || re != nil || actor <= 0 || repo <= 0 {
			return errors.New("native review operation scope differs")
		}
		o.ActorID, o.RepositoryID = actor, repo
	}
	return nil
}

func reviewOperationOutcome(id string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	o, err := reviewOutcomeState(id, record)
	if err != nil {
		return empty, err
	}
	if err := reviewOutcomeScope(&o, record); err != nil {
		return empty, err
	}
	o.Receipt = append([]byte(nil), record.Receipt...)
	return o, nil
}

// AdoptReview adopts one committed native review receipt against its
// submitted work. The receipt carries the summary review/comment IDs, the
// per-key inline comment IDs, the host-derived merge base and the exact
// admitted identity envelope (reviewer, PR, issue, number, commit, event,
// head/base OIDs), each validated against the immutable submitted work
// and operation envelope. The adopted
// per-key mapping is the stable native/factory identity: every submitted
// finding key arrives exactly once with its native comment record.
func checkReviewEnvelope(w factory.ReviewWork, o factory.OperationOutcome) error {
	if err := w.Validate(); err != nil {
		return err
	}
	if o.NotObserved || o.OperationID != w.OperationID || o.InstallationID == "" || o.Kind != factory.OpReviewSubmit || o.ActorID != w.ActorID || o.RepositoryID != w.Repository || o.Effect != factory.OpEffectCommitted || len(o.Receipt) == 0 || len(o.Receipt) > factory.MaxReviewReceiptBytes {
		return errors.New("review outcome differs from its recorded intent")
	}
	return nil
}

func decodeReviewReceipt(o factory.OperationOutcome) (extensions.ReviewSubmitReceipt, error) {
	var receipt extensions.ReviewSubmitReceipt
	decoder := json.NewDecoder(bytes.NewReader(o.Receipt))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return receipt, errors.New("native review receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return receipt, errors.New("native review receipt is malformed")
	}
	// The receipt echoes the immutable admitted identity envelope; every
	// echoed field must match the recorded work and operation envelope.
	// The receipt is historical evidence for its admitted tuple only.
	if receipt.ReviewID <= 0 || receipt.CommentID <= 0 {
		return receipt, errors.New("native review receipt differs from its recorded intent")
	}
	return receipt, nil
}

func checkReceiptIdentity(receipt extensions.ReviewSubmitReceipt, w factory.ReviewWork, o factory.OperationOutcome) error {
	if receipt.ReviewerID != w.ActorID || receipt.ReviewerID != o.ActorID || receipt.PRID != w.PRID || receipt.IssueID != w.IssueID || receipt.PRNumber != w.PRNumber || receipt.CommitID != w.HeadOID || receipt.Event != w.Event || receipt.HeadOID != w.HeadOID || receipt.BaseOID != w.BaseOID || o.RepositoryID != w.Repository {
		return errors.New("native review receipt differs from its recorded intent")
	}
	if !factory.ValidCommit(receipt.MergeBaseOID) {
		return errors.New("native review receipt lacks its merge base")
	}
	if len(receipt.Comments) != len(w.Comments) {
		return errors.New("native review receipt differs from its recorded comments")
	}
	return nil
}

func mapReceiptComments(receipt extensions.ReviewSubmitReceipt, w factory.ReviewWork) ([]factory.ReviewCommentMapping, error) {
	submitted := make(map[string]factory.FindingComment, len(w.Comments))
	for _, comment := range w.Comments {
		submitted[comment.Key] = comment
	}
	mappings := make([]factory.ReviewCommentMapping, 0, len(receipt.Comments))
	seenIDs := make(map[int64]bool, len(receipt.Comments))
	for _, comment := range receipt.Comments {
		if _, ok := submitted[comment.Key]; !ok {
			return nil, errors.New("native review receipt maps an unknown finding key")
		}
		delete(submitted, comment.Key)
		if comment.CommentID <= 0 || seenIDs[comment.CommentID] {
			return nil, errors.New("native review receipt carries an invalid comment record")
		}
		seenIDs[comment.CommentID] = true
		mapping := factory.ReviewCommentMapping{Key: comment.Key, CommentID: comment.CommentID}
		if err := mapping.Validate(); err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func (r *Reviewer) AdoptReview(w factory.ReviewWork, o factory.OperationOutcome) (factory.ReviewOutcome, error) {
	var empty factory.ReviewOutcome
	if err := checkReviewEnvelope(w, o); err != nil {
		return empty, err
	}
	receipt, err := decodeReviewReceipt(o)
	if err != nil {
		return empty, err
	}
	if err := checkReceiptIdentity(receipt, w, o); err != nil {
		return empty, err
	}
	mappings, err := mapReceiptComments(receipt, w)
	if err != nil {
		return empty, err
	}
	return factory.ReviewOutcome{Operation: o, Comments: mappings, ReviewID: receipt.ReviewID, CommentID: receipt.CommentID, ReviewerID: w.ActorID, PRID: w.PRID, PRNumber: w.PRNumber, IssueID: w.IssueID, HeadOID: w.HeadOID, BaseOID: w.BaseOID, CommitID: w.HeadOID, Event: w.Event, MergeBaseOID: receipt.MergeBaseOID}, nil
}
