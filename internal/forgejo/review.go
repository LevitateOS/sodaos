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

// Reviewer submits only exact body-only reviews through the shared native
// admission. Its distinct actor secret stays outside the Project and its agents.
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
	user, err := r.rest.Current(ctx, token)
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

func matchReviewTarget(w factory.ReviewWork, issue, snapshot NativeSnapshot) error {
	i, p := issue.Issue, snapshot.Pull
	if i == nil || p == nil || !i.Visible || !p.Visible || !i.IsPull || i.IsClosed || p.HasMerged ||
		i.ID != strconv.FormatInt(w.IssueID, 10) || i.Provenance.PosterID != strconv.FormatInt(w.PRAuthorID, 10) ||
		p.ID != strconv.FormatInt(w.PRID, 10) || p.IssueID != i.ID || p.HeadRepoID != strconv.FormatInt(w.Repository, 10) ||
		p.HeadBranch != factory.RefHead(w.HeadRef) || p.BaseBranch != factory.RefHead(w.BaseRef) {
		return &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	if p.HeadTip != w.HeadOID {
		return &factory.PublicationRefusal{Reason: "stale_head"}
	}
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
	if snapshot.Reviews == nil || snapshot.Reviews.IssueID != i.ID {
		return &factory.PublicationWait{Reason: "review_evidence_unavailable"}
	}
	for _, review := range snapshot.Reviews.Items {
		if !review.Visible || review.IssueID != i.ID {
			return &factory.PublicationWait{Reason: "review_evidence_unavailable"}
		}
		if review.ReviewerID == strconv.FormatInt(w.ActorID, 10) && review.Type == "PENDING" {
			return &factory.PublicationRefusal{Reason: "pending_review_exists"}
		}
	}
	return nil
}

func reviewIntent(w factory.ReviewWork) (extensions.OperationIntent, error) {
	if err := w.Validate(); err != nil {
		return extensions.OperationIntent{}, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	payload := extensions.ReviewSubmitPayload{PullRequestNumber: w.PRNumber, PRAuthorID: w.PRAuthorID, HeadRepositoryID: w.Repository, HeadRef: w.HeadRef, BaseRef: w.BaseRef, ExpectedHeadOID: w.HeadOID, ExpectedBaseOID: w.BaseOID, CommitID: w.HeadOID, Event: w.Event, Body: w.Body}
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

func reviewOperationOutcome(id string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
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
	if !factory.ValidOpEffect(o.Effect) || !factory.ValidOpCancellation(o.Cancellation) || (o.Completion != "" && !factory.ValidOpCompletion(o.Completion)) || len(o.Reason) > 64 || len(record.Receipt) > factory.MaxPublicationReceipt {
		return empty, errors.New("invalid native review operation state")
	}
	tombstone := record.Kind == "" && record.ActorID == "" && record.RepositoryID == "" && o.Effect == factory.OpEffectNotCommitted && o.Cancellation == factory.OpCancelCancelled
	if !tombstone {
		actor, ae := strconv.ParseInt(record.ActorID, 10, 64)
		repo, re := strconv.ParseInt(record.RepositoryID, 10, 64)
		if record.Kind != factory.OpReviewSubmit || ae != nil || re != nil || actor <= 0 || repo <= 0 {
			return empty, errors.New("native review operation scope differs")
		}
		o.ActorID, o.RepositoryID = actor, repo
	}
	o.Receipt = append([]byte(nil), record.Receipt...)
	return o, nil
}

// reviewReceipt is the wire shape observed from the staged native FT13 producer
// by TestNativeReviewWire. Ref/author/repository expectations are bound by the
// immutable submitted payload and envelope; they are not invented receipt fields.
type reviewReceipt struct {
	ReviewID   int64  `json:"review_id"`
	CommentID  int64  `json:"comment_id"`
	ReviewerID int64  `json:"reviewer_id"`
	IssueID    int64  `json:"issue_id"`
	PRID       int64  `json:"pr_id"`
	PRNumber   int64  `json:"pr_number"`
	HeadOID    string `json:"head_oid"`
	BaseOID    string `json:"base_oid"`
	CommitID   string `json:"commit_id"`
	Event      string `json:"event"`
}

func (r *Reviewer) AdoptReview(w factory.ReviewWork, o factory.OperationOutcome) (factory.ReviewOutcome, error) {
	var empty factory.ReviewOutcome
	if err := w.Validate(); err != nil {
		return empty, err
	}
	if o.NotObserved || o.OperationID != w.OperationID || o.InstallationID == "" || o.Kind != factory.OpReviewSubmit || o.ActorID != w.ActorID || o.RepositoryID != w.Repository || o.Effect != factory.OpEffectCommitted || len(o.Receipt) == 0 || len(o.Receipt) > factory.MaxPublicationReceipt {
		return empty, errors.New("review outcome differs from its recorded intent")
	}
	var receipt reviewReceipt
	decoder := json.NewDecoder(bytes.NewReader(o.Receipt))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return empty, errors.New("native review receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return empty, errors.New("native review receipt is malformed")
	}
	if receipt.ReviewID <= 0 || receipt.CommentID <= 0 || receipt.ReviewerID != w.ActorID || receipt.IssueID != w.IssueID || receipt.PRID != w.PRID || receipt.PRNumber != w.PRNumber || receipt.HeadOID != w.HeadOID || receipt.BaseOID != w.BaseOID || receipt.CommitID != w.HeadOID || receipt.Event != w.Event {
		return empty, errors.New("native review receipt differs from its recorded intent")
	}
	return factory.ReviewOutcome{Operation: o, ReviewID: receipt.ReviewID, CommentID: receipt.CommentID, ReviewerID: receipt.ReviewerID, PRID: receipt.PRID, PRNumber: receipt.PRNumber, IssueID: receipt.IssueID, HeadOID: receipt.HeadOID, BaseOID: receipt.BaseOID, CommitID: receipt.CommitID, Event: receipt.Event}, nil
}
