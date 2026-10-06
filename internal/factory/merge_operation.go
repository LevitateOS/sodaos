package factory

import (
	"errors"
	"fmt"
)

// MergeOperation is one persisted conditional merge. Work is its
// immutable intent, recorded before submission. Effect, Cancellation
// and Completion mirror the latest native observations; Reason carries
// the bounded native reason code; Receipt carries the bounded raw
// native receipt. Link fields are adopted only from this operation's
// own committed receipt and verified against the submitted intent.
type MergeOperation struct {
	Work           *MergeIntent `json:"work,omitempty"`
	OperationID    string       `json:"operation_id"`
	InstallationID string       `json:"installation_id,omitempty"`
	ActorID        int64        `json:"actor_id,omitempty"`
	RepositoryID   int64        `json:"repository_id,omitempty"`
	Kind           string       `json:"kind"`
	Effect         string       `json:"effect,omitempty"`
	Cancellation   string       `json:"cancellation,omitempty"`
	Completion     string       `json:"completion,omitempty"`
	Reason         string       `json:"reason,omitempty"`
	Receipt        string       `json:"receipt,omitempty"`
	HeadRef        string       `json:"head_ref,omitempty"`
	BaseRef        string       `json:"base_ref,omitempty"`
	HeadOID        string       `json:"head_oid,omitempty"`
	BaseOID        string       `json:"base_oid,omitempty"`
	MergedCommit   string       `json:"merged_commit,omitempty"`
	Attempts       int          `json:"attempts"`
	PRNumber       int64        `json:"pr_number,omitempty"`
	PRID           int64        `json:"pr_id,omitempty"`
	IssueID        int64        `json:"issue_id,omitempty"`
	UpdatedUnix    int64        `json:"updated_unix"`
}

// Validate rejects malformed operation records. An unsubmitted operation
// carries its kind and no observations; a submitted one carries its
// exact identity, attempt count and record time. The effect stays empty
// while its submit is in flight.
func (o MergeOperation) Validate() error {
	if o.Kind != OpMerge {
		return errors.New("invalid merge operation kind")
	}
	if o.Attempts < 0 || o.Attempts > MaxMergeAttempts {
		return errors.New("invalid merge attempt count")
	}
	if o.Attempts == 0 {
		if o.Work != nil || o.OperationID != "" || o.InstallationID != "" || o.ActorID != 0 || o.RepositoryID != 0 || o.Effect != "" || o.Cancellation != "" ||
			o.Completion != "" || o.Reason != "" || o.Receipt != "" || o.UpdatedUnix != 0 {
			return errors.New("unsubmitted operation carries no observations")
		}
		return o.validateLinks()
	}
	if !ValidPublicationOperationID(o.OperationID) {
		return errors.New("invalid merge operation identity")
	}
	if o.Work == nil || o.Work.OperationID != o.OperationID {
		return errors.New("merge operation lacks its immutable intent")
	}
	if err := o.Work.Validate(); err != nil {
		return err
	}
	// An empty effect records a submit in flight: the identity is
	// persisted before its submit so a crash reconciles by lookup and a
	// withdrawal cancels by the same identity. The next pass reconciles
	// it before anything else.
	if o.Effect != "" && !ValidOpEffect(o.Effect) {
		return errors.New("submitted operation effect is unknown")
	}
	if o.UpdatedUnix <= 0 {
		return errors.New("submitted operation lacks its record time")
	}
	if o.Cancellation != "" && !ValidOpCancellation(o.Cancellation) {
		return errors.New("invalid merge cancellation state")
	}
	if o.Completion != "" && !ValidOpCompletion(o.Completion) {
		return errors.New("invalid merge completion state")
	}
	if len(o.Reason) > 64 || len(o.Receipt) > MaxPublicationReceipt {
		return errors.New("merge observation exceeds bounds")
	}
	return o.validateLinks()
}

func (o MergeOperation) validateLinks() error {
	if o.PRNumber < 0 || o.PRID < 0 || o.IssueID < 0 {
		return errors.New("invalid merge native links")
	}
	if (o.PRNumber == 0) != (o.PRID == 0) || (o.PRNumber == 0) != (o.IssueID == 0) {
		return errors.New("incomplete merge PR links")
	}
	// A committed merge may await adoption: links arrive only from its
	// own verified receipt. An uncommitted merge carries no links, and
	// partial links never stand.
	if o.Effect != OpEffectCommitted && o.PRNumber != 0 {
		return errors.New("uncommitted merge carries no PR links")
	}
	if (o.HeadRef != "" && !ValidTargetBranch(o.HeadRef)) || (o.BaseRef != "" && !ValidTargetBranch(o.BaseRef)) {
		return errors.New("invalid merge snapshot refs")
	}
	if (o.HeadOID != "" && !ValidCommit(o.HeadOID)) || (o.BaseOID != "" && !ValidCommit(o.BaseOID)) {
		return errors.New("invalid merge snapshot tips")
	}
	if o.MergedCommit != "" && !ValidCommit(o.MergedCommit) {
		return errors.New("invalid merge realized commit")
	}
	if o.PRNumber != 0 && (o.HeadRef == "" || o.BaseRef == "" || o.HeadOID == "" || o.BaseOID == "" || o.MergedCommit == "") {
		return errors.New("merge PR links lack their snapshot")
	}
	if o.MergedCommit != "" && o.HeadOID != "" && o.MergedCommit != o.HeadOID {
		return errors.New("merge realized commit differs from its exact head")
	}
	return nil
}

// MergeOperationID derives the deterministic identity for one merge.
// The identity is persisted before its submit so a lost reply
// reconciles by lookup instead of resubmitting blindly; terminal
// identities are never reused.
func MergeOperationID(publicationID string, attempt int) string {
	return fmt.Sprintf("soda-%s-merge-%d", publicationID, attempt)
}

// MergeWork is one executor call's exact bound inputs: the native PR
// identity, the exact candidate and verified base, the independent
// reviewer behind the approval, the policy-bound merge actor, the
// bracketed observation, and the persisted operation identity the call
// must use. No credential or credential-adjacent value enters the work
// record; secrets travel in restricted files outside it.
type MergeWork struct {
	MergeID            string
	PublicationID      string
	OperationID        string
	AuthRevision       string
	HeadRef            string
	BaseRef            string
	HeadOID            string
	BaseOID            string
	Repository         int64
	Issue              int64
	PRNumber           int64
	PRID               int64
	IssueID            int64
	PRAuthorID         int64
	ReviewerID         int64
	ActorID            int64
	NativeRev          int64
	NotAfter           int64
	AssessmentRevision int64
	ReviewID           int64
}

// MergeIntent contains only the immutable native inputs.
type MergeIntent struct {
	OperationID        string `json:"operation_id"`
	AuthRevision       string `json:"authorization_revision"`
	HeadRef            string `json:"head_ref"`
	BaseRef            string `json:"base_ref"`
	HeadOID            string `json:"head_oid"`
	BaseOID            string `json:"base_oid"`
	Repository         int64  `json:"repository,string"`
	ActorID            int64  `json:"actor_id,string"`
	PRNumber           int64  `json:"pr_number,string"`
	PRID               int64  `json:"pr_id,string"`
	IssueID            int64  `json:"issue_id,string"`
	PRAuthorID         int64  `json:"pr_author_id,string"`
	ReviewerID         int64  `json:"reviewer_id,string"`
	NativeRev          int64  `json:"native_revision"`
	NotAfter           int64  `json:"not_after"`
	AssessmentRevision int64  `json:"assessment_revision"`
	ReviewID           int64  `json:"review_id"`
}

// Intent takes the credential-free native inputs from an executor call.
func (w MergeWork) Intent() MergeIntent {
	return MergeIntent{
		OperationID: w.OperationID, AuthRevision: w.AuthRevision,
		HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID,
		Repository: w.Repository, ActorID: w.ActorID, PRNumber: w.PRNumber, PRID: w.PRID,
		IssueID: w.IssueID, PRAuthorID: w.PRAuthorID, ReviewerID: w.ReviewerID,
		NativeRev: w.NativeRev, NotAfter: w.NotAfter,
		AssessmentRevision: w.AssessmentRevision, ReviewID: w.ReviewID,
	}
}

// Apply restores an exact native intent without changing local inputs.
func (i MergeIntent) Apply(w MergeWork) MergeWork {
	w.OperationID, w.AuthRevision = i.OperationID, i.AuthRevision
	w.HeadRef, w.BaseRef, w.HeadOID, w.BaseOID = i.HeadRef, i.BaseRef, i.HeadOID, i.BaseOID
	w.Repository, w.ActorID, w.PRNumber, w.PRID = i.Repository, i.ActorID, i.PRNumber, i.PRID
	w.IssueID, w.PRAuthorID, w.ReviewerID = i.IssueID, i.PRAuthorID, i.ReviewerID
	w.NativeRev, w.NotAfter = i.NativeRev, i.NotAfter
	w.AssessmentRevision, w.ReviewID = i.AssessmentRevision, i.ReviewID
	return w
}

// ValidateTarget checks the exact merge target before evidence or
// operation identity exist. The reviewer must be independent of the
// recorded PR author.
func (w MergeWork) ValidateTarget() error {
	if w.Repository <= 0 || w.Issue <= 0 || w.PRNumber <= 0 || w.PRID <= 0 || w.IssueID <= 0 {
		return errors.New("invalid merge target scope")
	}
	if w.ActorID <= 0 || w.PRAuthorID <= 0 || w.ReviewerID <= 0 || w.ReviewerID == w.PRAuthorID {
		return errors.New("merge review evidence needs its independent reviewer")
	}
	if !ValidTargetBranch(w.HeadRef) || !ValidTargetBranch(w.BaseRef) || w.HeadRef == w.BaseRef {
		return errors.New("invalid merge refs")
	}
	if !ValidCommit(w.HeadOID) || !ValidCommit(w.BaseOID) || w.HeadOID == w.BaseOID {
		return errors.New("merge candidate is not a fresh exact commit")
	}
	return nil
}

// Validate rejects incomplete immutable native intents.
func (i MergeIntent) Validate() error { return i.validate(true) }

func (i MergeIntent) validate(observed bool) error {
	if !ValidTargetBranch(i.HeadRef) || !ValidTargetBranch(i.BaseRef) || i.HeadRef == i.BaseRef {
		return errors.New("invalid merge refs")
	}
	if !ValidCommit(i.HeadOID) || !ValidCommit(i.BaseOID) || i.HeadOID == i.BaseOID {
		return errors.New("merge candidate is not a fresh exact commit")
	}
	if !ValidPublicationOperationID(i.OperationID) {
		return errors.New("invalid merge operation identity")
	}
	if i.Repository <= 0 || i.ActorID <= 0 || i.PRNumber <= 0 || i.PRID <= 0 || i.IssueID <= 0 {
		return errors.New("invalid merge work scope")
	}
	if i.PRAuthorID <= 0 || i.ReviewerID <= 0 || i.ReviewerID == i.PRAuthorID {
		return errors.New("merge review evidence needs its independent reviewer")
	}
	if i.AuthRevision == "" || len(i.AuthRevision) > 512 {
		return errors.New("invalid merge authorization revision")
	}
	if i.NativeRev < 0 || i.NotAfter < 0 || (observed && (i.NativeRev == 0 || i.NotAfter == 0)) {
		return errors.New("invalid merge observation binding")
	}
	if observed && (i.AssessmentRevision <= 0 || i.ReviewID <= 0) {
		return errors.New("merge intent lacks its bound check and review evidence")
	}
	if i.AssessmentRevision < 0 || i.ReviewID < 0 {
		return errors.New("invalid merge evidence binding")
	}
	return nil
}

// ValidateObservation checks target and identity inputs before native
// evidence is observed; only the native revision, deadline and evidence
// bindings may be absent.
func (w MergeWork) ValidateObservation() error {
	if !ValidID(w.MergeID) || !ValidID(w.PublicationID) || w.Issue <= 0 {
		return errors.New("invalid merge work identity")
	}
	return w.Intent().validate(false)
}

// Validate adds the exact native observation needed before submission.
func (w MergeWork) Validate() error {
	if err := w.ValidateObservation(); err != nil {
		return err
	}
	return w.Intent().Validate()
}
