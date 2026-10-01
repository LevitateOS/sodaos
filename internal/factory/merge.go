// Merge records bind one published PR's exact candidate and verified base
// to its single native conditional merge. The merge advances only on
// attributable native effects behind current authority, exact
// head/base/review/check evidence and a confirmed issue outcome; a
// committed ref alone never finishes the bookkeeping. Confirmed
// completion is the attributable factory completion that releases
// eligible code dependants.
package factory

import (
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/project"
)

// Merge stages. Open work reconciles its recorded operation against
// native state and may still submit; every other stage is terminal for
// the automatic pass. Fenced work keeps an indeterminate native effect
// owned by offline recovery; later passes re-read it but submit nothing
// new.
const (
	MergeOpen      = "open"
	MergeMerged    = "merged"
	MergeFailed    = "failed"
	MergeWithdrawn = "withdrawn"
	MergeFenced    = "fenced"
)

// ValidMergeStage reports whether stage is a known merge stage.
func ValidMergeStage(stage string) bool {
	switch stage {
	case MergeOpen, MergeMerged, MergeFailed, MergeWithdrawn, MergeFenced:
		return true
	default:
		return false
	}
}

// MaxMergeAttempts permits one immutable merge operation. A terminal
// refusal never creates a replacement identity automatically.
const MaxMergeAttempts = 1

// Merge finish reasons. Bounded codes: views expose why a merge
// finished, never the evidence bytes behind it.
const (
	MergeReasonMerged       = "merge_completed"
	MergeReasonRefused      = "native_refused"
	MergeReasonInvalid      = "evidence_invalid"
	MergeReasonWithdrawn    = "merge_withdrawn"
	MergeReasonFenced       = "effect_indeterminate"
	MergeReasonSuperseded   = "acceptance_superseded"
	MergeReasonUnattributed = "effect_unattributed"
)

func validMergeReason(reason string) bool {
	switch reason {
	case MergeReasonMerged, MergeReasonRefused, MergeReasonInvalid,
		MergeReasonWithdrawn, MergeReasonFenced, MergeReasonSuperseded,
		MergeReasonUnattributed:
		return true
	default:
		return false
	}
}

// MergeResolution is the static maintainer-facing resolution for one
// merge reason: what would resolve it. No free text enters resolutions.
func MergeResolution(reason string) string {
	switch reason {
	case MergeReasonMerged:
		return "The exact candidate merged and its completion is confirmed."
	case MergeReasonRefused:
		return "Native refused the merge; inspect the recorded reason and correct the candidate."
	case MergeReasonInvalid:
		return "The merge evidence is stale or incomplete; reassess review and checks on the exact head."
	case MergeReasonWithdrawn:
		return "The merge was withdrawn before its native effect resolved."
	case MergeReasonFenced:
		return "The native merge effect is indeterminate; offline recovery owns it."
	case MergeReasonSuperseded:
		return "The acceptance changed; new work supersedes this merge."
	case MergeReasonUnattributed:
		return "The native merge effect cannot be attributed to its recorded intent."
	default:
		return ""
	}
}

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

// Merge is one publication's durable merge: the exact bound candidate
// and verified base, the separately persisted conditional operation,
// the fresh review/check observations the attempt bound, and the
// confirmed completion once native bookkeeping proves the outcome.
// Finished work is terminal and keeps its receipts for inspection.
type Merge struct {
	WithdrawRequested bool           `json:"withdraw_requested,omitempty"`
	Operation         MergeOperation `json:"operation"`
	Authority         AuthorityRef   `json:"authority"`
	ID                string         `json:"id"`
	PublicationID     string         `json:"publication_id"`
	AssignmentID      string         `json:"assignment_id"`
	ProjectID         string         `json:"project_id"`
	Role              string         `json:"role"`
	Acceptance        string         `json:"acceptance"`
	HeadRef           string         `json:"head_ref"`
	BaseRef           string         `json:"base_ref"`
	HeadOID           string         `json:"head_oid"`
	BaseOID           string         `json:"base_oid"`
	MergedCommit      string         `json:"merged_commit,omitempty"`
	Stage             string         `json:"stage"`
	Outcome           Outcome        `json:"outcome,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	Repository        int64          `json:"repository,string"`
	Issue             int64          `json:"issue,string"`
	PRNumber          int64          `json:"pr_number"`
	PRID              int64          `json:"pr_id"`
	IssueID           int64          `json:"issue_id"`
	PRAuthorID        int64          `json:"pr_author_id"`
	ReviewerID        int64          `json:"reviewer_id"`
	Revision          int64          `json:"revision"`
	NativeRev         int64          `json:"native_revision"`
	CreatedUnix       int64          `json:"created_unix"`
	ObservedUnix      int64          `json:"observed_unix,omitempty"`
	FinishedUnix      int64          `json:"finished_unix,omitempty"`
	MergedUnix        int64          `json:"merged_unix,omitempty"`
	ClosedUnix        int64          `json:"closed_unix,omitempty"`
}

// Validate rejects malformed merges. Open work carries no outcome;
// finished work carries its outcome, reason and finish time. A merged
// merge links exactly the PR its own merge receipt proved and carries
// the confirmed completion stamps.
func (m Merge) Validate() error {
	if !ValidID(m.ID) || !ValidID(m.PublicationID) || !ValidID(m.AssignmentID) {
		return errors.New("invalid merge identity")
	}
	if !ValidProjectID(m.ProjectID) || m.Role != project.RoleCoder {
		return errors.New("invalid merge address")
	}
	if m.Repository <= 0 || m.Issue <= 0 || m.PRNumber <= 0 || m.PRID <= 0 || m.IssueID <= 0 || m.Revision < 0 {
		return errors.New("invalid merge scope")
	}
	if m.PRAuthorID <= 0 || m.ReviewerID <= 0 || m.ReviewerID == m.PRAuthorID {
		return errors.New("merge review evidence needs its independent reviewer")
	}
	if !project.ValidDecisionID(m.Acceptance) {
		return errors.New("invalid merge acceptance")
	}
	if !ValidTargetBranch(m.HeadRef) || !ValidTargetBranch(m.BaseRef) || m.HeadRef == m.BaseRef {
		return errors.New("invalid merge refs")
	}
	if !ValidCommit(m.HeadOID) || !ValidCommit(m.BaseOID) || m.HeadOID == m.BaseOID {
		return errors.New("merge candidate is not a fresh exact commit")
	}
	if m.MergedCommit != "" && (!ValidCommit(m.MergedCommit) || m.MergedCommit != m.HeadOID) {
		return errors.New("merge realized commit differs from its exact head")
	}
	if m.NativeRev < 0 || (m.NativeRev == 0) != (m.ObservedUnix == 0) {
		return errors.New("invalid merge observation binding")
	}
	if m.ObservedUnix < 0 || m.CreatedUnix <= 0 || m.MergedUnix < 0 || m.ClosedUnix < 0 {
		return errors.New("invalid merge time")
	}
	if err := m.Operation.Validate(); err != nil {
		return err
	}
	if m.Operation.Kind != OpMerge {
		return errors.New("merge operation has the wrong kind")
	}
	if op := m.Operation; op.Work != nil && (op.Work.HeadOID != m.HeadOID || op.Work.BaseOID != m.BaseOID || op.Work.Repository != m.Repository || op.Work.PRNumber != m.PRNumber) {
		return errors.New("merge operation intent differs from its merge")
	}
	if op := m.Operation; op.PRNumber != 0 && (op.Effect != OpEffectCommitted || op.PRNumber != m.PRNumber || op.PRID != m.PRID || op.IssueID != m.IssueID) {
		return errors.New("merge PR is not the recorded merge effect")
	}
	switch m.Stage {
	case MergeOpen:
		if m.Outcome != "" || m.Reason != "" || m.FinishedUnix != 0 || m.MergedCommit != "" || m.MergedUnix != 0 || m.ClosedUnix != 0 {
			return errors.New("open merge carries no outcome")
		}
	case MergeMerged:
		if m.Outcome != Succeeded || m.Reason != MergeReasonMerged || m.FinishedUnix <= 0 {
			return errors.New("merged merge lacks its outcome")
		}
		if m.Operation.Effect != OpEffectCommitted || m.Operation.Completion != OpCompletionComplete {
			return errors.New("merged merge lacks native completion")
		}
		if m.Operation.MergedCommit != m.HeadOID || m.MergedCommit != m.HeadOID || m.MergedUnix <= 0 || m.ClosedUnix <= 0 {
			return errors.New("merged merge lacks its confirmed completion")
		}
		if m.Operation.PRNumber != m.PRNumber || m.Operation.PRID != m.PRID || m.Operation.IssueID != m.IssueID {
			return errors.New("merged PR is not the recorded merge effect")
		}
	case MergeFailed, MergeWithdrawn, MergeFenced:
		if !validMergeReason(m.Reason) || m.FinishedUnix <= 0 {
			return errors.New("finished merge lacks its outcome")
		}
		if m.Reason == MergeReasonMerged {
			return errors.New("unfinished merge cannot claim completed linkage")
		}
		switch m.Stage {
		case MergeFailed:
			if m.Outcome != Failed {
				return errors.New("failed merge lacks its outcome")
			}
		case MergeWithdrawn:
			if m.Outcome != Cancelled {
				return errors.New("withdrawn merge lacks its outcome")
			}
			if op := m.Operation; op.Attempts != 0 && op.Effect != OpEffectCommitted && op.Effect != OpEffectNotCommitted {
				return errors.New("withdrawn merge still has an unresolved native effect")
			}
		case MergeFenced:
			if m.Outcome != NeedsHuman {
				return errors.New("fenced merge lacks its outcome")
			}
		}
	default:
		return errors.New("invalid merge stage")
	}
	return nil
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

// VerifyMergeCheckEvidence enforces the ST11 read path: the latest
// persisted assessment must pass on this exact head and verified base
// under exactly the current adopted definitions. A pass never survives
// a changed head, base or policy without fresh assessment.
func VerifyMergeCheckEvidence(w MergeWork, a CheckAssessment, policy RepositoryPolicy) error {
	if err := w.ValidateTarget(); err != nil {
		return err
	}
	if err := a.Validate(); err != nil {
		return errors.New("merge check evidence is not a recorded assessment")
	}
	if policy.Repository != w.Repository {
		return errors.New("merge policy differs from its repository")
	}
	if a.Repository != w.Repository || a.PRNumber != w.PRNumber {
		return errors.New("merge check evidence differs from its PR")
	}
	if a.Verdict != CheckPass {
		return errors.New("merge check evidence does not pass")
	}
	if a.HeadOID != w.HeadOID || a.BaseOID != w.BaseOID {
		return errors.New("merge check evidence is stale")
	}
	if a.PolicyRevision != policy.Revision || a.ChecksDigest != ChecksDigest(policy.Checks) {
		return errors.New("merge check evidence predates its adopted definitions")
	}
	if w.AssessmentRevision != 0 && a.Revision != w.AssessmentRevision {
		return errors.New("merge check evidence differs from its bound assessment")
	}
	return nil
}

// MergeObservation is one executor observation: the bracketed idle
// native revision plus the exact PR linkage, tips, consumed approval
// and check evidence behind it.
type MergeObservation struct {
	Checks       ObservedChecks
	NativeRev    int64
	ObservedUnix int64
	PRID         int64
	PRNumber     int64
	IssueID      int64
	PRAuthorID   int64
	HeadRef      string
	BaseRef      string
	HeadOID      string
	BaseOID      string
	ReviewerID   int64
	ReviewID     int64
}

// MergeConfirmation is one observed native completion: the realized
// merge commit and stamps plus the exact base tip and issue closure
// behind them. A committed ref without this confirmation never finishes
// the bookkeeping.
type MergeConfirmation struct {
	MergedCommit string
	BaseTip      string
	MergerID     int64
	MergedUnix   int64
	ClosedUnix   int64
	NativeRev    int64
	ObservedUnix int64
	IssueClosed  bool
}

// MergeOutcome is one reconciled native merge: the operation state plus
// the exact PR identity and realized commit adopted from its own
// committed receipt.
type MergeOutcome struct {
	Operation    OperationOutcome
	HeadRef      string
	BaseRef      string
	HeadOID      string
	BaseOID      string
	MergedCommit string
	PRNumber     int64
	PRID         int64
	IssueID      int64
	ActorID      int64
}

// MergeWithdrawal reports ordered local withdrawal separately from
// outstanding native effects. Operation identities remain available for
// recovery.
type MergeWithdrawal struct {
	Merges     []string `json:"merges"`
	Operations []string `json:"operations"`
	Pending    bool     `json:"pending"`
}

// MergeAuthRevision binds one merge's Soda authority reference into the
// opaque native authorization revision: assignment, merge and Soda
// revision, never a credential.
func MergeAuthRevision(assignmentID, mergeID string, revision int64) string {
	return fmt.Sprintf("soda-assignment:%s:merge:%s:revision:%d", assignmentID, mergeID, revision)
}

// MergeTargetChanged reports whether the recorded merge target differs
// from the publication it was created from. Any difference refuses the
// merge instead of merging a moved candidate.
func MergeTargetChanged(m Merge, p Publication) bool {
	return m.Repository != p.Repository || m.Issue != p.Issue || m.PRNumber != p.PRNumber || m.PRID != p.PRID ||
		m.HeadRef == "" || m.BaseRef == "" || m.HeadOID == "" || m.BaseOID == "" ||
		p.PRCreate.HeadRef != m.HeadRef || p.PRCreate.BaseRef != m.BaseRef ||
		p.PRCreate.HeadOID != m.HeadOID || p.PRCreate.BaseOID != m.BaseOID
}
