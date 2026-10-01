// Publication records bind one finished assignment's validated candidate to
// its two native conditional operations: the branch publication and the PR
// creation. Each operation keeps its own persisted identity, attempt count,
// effect, cancellation and completion state; the publication advances only
// on attributable native effects, never on a lost reply, a guessed failure
// or a lookalike record. Partial outcomes stay visible: a committed branch
// with a failed PR creation keeps both facts linked to the assignment.
package factory

import (
	"errors"
	"fmt"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// Publication stages. Open work reconciles its recorded operations against
// native state and may still submit; every other stage is terminal for the
// automatic pass. Fenced work keeps an indeterminate native effect owned by
// offline recovery; later passes re-read it but submit nothing new.
const (
	PublicationOpen      = "open"
	PublicationPublished = "published"
	PublicationFailed    = "failed"
	PublicationWithdrawn = "withdrawn"
	PublicationFenced    = "fenced"
)

// ValidPublicationStage reports whether stage is a known publication stage.
func ValidPublicationStage(stage string) bool {
	switch stage {
	case PublicationOpen, PublicationPublished, PublicationFailed, PublicationWithdrawn, PublicationFenced:
		return true
	default:
		return false
	}
}

// MaxPublicationAttempts permits one immutable operation per kind. A
// terminal refusal never creates a replacement identity automatically.
const MaxPublicationAttempts = 1

// Publication finish reasons. Bounded codes: views expose why a
// publication finished, never the evidence bytes behind it.
const (
	PublishReasonLinked         = "pr_linked"
	PublishReasonRefused        = "native_refused"
	PublishReasonInvalid        = "candidate_invalid"
	PublishReasonWithdrawn      = "publication_withdrawn"
	PublishReasonFenced         = "effect_indeterminate"
	PublishReasonSuperseded     = "acceptance_superseded"
	PublishReasonExportFailed   = "export_unavailable"
	PublishReasonExhausted      = "attempts_exhausted"
	PublishReasonBadObservation = "observation_invalid"
	PublishReasonTargetOccupied = "target_occupied"
	PublishReasonUnattributed   = "effect_unattributed"
)

func validPublishReason(reason string) bool {
	switch reason {
	case PublishReasonLinked, PublishReasonRefused, PublishReasonInvalid,
		PublishReasonWithdrawn, PublishReasonFenced, PublishReasonSuperseded,
		PublishReasonExportFailed, PublishReasonExhausted, PublishReasonBadObservation,
		PublishReasonTargetOccupied, PublishReasonUnattributed:
		return true
	default:
		return false
	}
}

// Native operation effect states mirrored from the conditional-operation
// contract. Pending work may still resolve; committed and not_committed
// are terminal native verdicts; indeterminate keeps a retained native
// fence owned by offline recovery.
const (
	OpEffectPending       = "pending"
	OpEffectCommitted     = "committed"
	OpEffectNotCommitted  = "not_committed"
	OpEffectIndeterminate = "indeterminate"
)

// ValidOpEffect reports whether effect is a known native effect state.
func ValidOpEffect(effect string) bool {
	switch effect {
	case OpEffectPending, OpEffectCommitted, OpEffectNotCommitted, OpEffectIndeterminate:
		return true
	default:
		return false
	}
}

// Native cancellation states mirrored from the contract. None means no
// cancellation was ever requested; pending means the ordering against an
// in-flight write is still unknown.
const (
	OpCancelNone          = "none"
	OpCancelPending       = "pending"
	OpCancelCancelled     = "cancelled"
	OpCancelTooLate       = "too_late"
	OpCancelIndeterminate = "indeterminate"
)

// ValidOpCancellation reports whether status is a known cancellation state.
func ValidOpCancellation(status string) bool {
	switch status {
	case OpCancelNone, OpCancelPending, OpCancelCancelled, OpCancelTooLate, OpCancelIndeterminate:
		return true
	default:
		return false
	}
}

// Native completion states mirrored from the contract. Committed alone is
// not completed factory work: pending completion still owes derived native
// effects, and needs_intervention owes an operator decision.
const (
	OpCompletionPending           = "pending"
	OpCompletionComplete          = "complete"
	OpCompletionNeedsIntervention = "needs_intervention"
)

// ValidOpCompletion reports whether state is a known completion state.
func ValidOpCompletion(state string) bool {
	switch state {
	case OpCompletionPending, OpCompletionComplete, OpCompletionNeedsIntervention:
		return true
	default:
		return false
	}
}

// MaxPublicationReceipt bounds one recorded native receipt: the kind's
// decoded link fields are adopted separately, and the raw bytes stay
// inspectable without carrying unbounded native content.
const MaxPublicationReceipt = 4096

// PublicationOperation is one persisted conditional operation. Work is
// its immutable intent, recorded before submission. Effect, Cancellation and
// Completion mirror the latest native observations; Reason carries the
// bounded native reason code; Receipt carries the bounded raw native
// receipt. Decoded link fields are adopted only from this operation's own
// committed receipt and verified against the submitted intent.
type PublicationOperation struct {
	Work           *PublicationIntent `json:"work,omitempty"`
	OperationID    string             `json:"operation_id"`
	InstallationID string             `json:"installation_id,omitempty"`
	ActorID        int64              `json:"actor_id,omitempty"`
	RepositoryID   int64              `json:"repository_id,omitempty"`
	Kind           string             `json:"kind"`
	Effect         string             `json:"effect,omitempty"`
	Cancellation   string             `json:"cancellation,omitempty"`
	Completion     string             `json:"completion,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	Receipt        string             `json:"receipt,omitempty"`
	HeadRef        string             `json:"head_ref,omitempty"`
	BaseRef        string             `json:"base_ref,omitempty"`
	HeadOID        string             `json:"head_oid,omitempty"`
	BaseOID        string             `json:"base_oid,omitempty"`
	Attempts       int                `json:"attempts"`
	PRNumber       int64              `json:"pr_number,omitempty"`
	PRID           int64              `json:"pr_id,omitempty"`
	IssueID        int64              `json:"issue_id,omitempty"`
	UpdatedUnix    int64              `json:"updated_unix"`
}

// Validate rejects malformed operation records. An unsubmitted operation
// carries its kind and no observations; a submitted one carries its exact
// identity, attempt count and record time, with links only for its own
// kind. The effect stays empty while its submit is in flight.
func (o PublicationOperation) Validate() error {
	if o.Kind != OpRefPublish && o.Kind != OpPRCreate {
		return errors.New("invalid publication operation kind")
	}
	if o.Attempts < 0 || o.Attempts > MaxPublicationAttempts {
		return errors.New("invalid publication attempt count")
	}
	if o.Attempts == 0 {
		if o.Work != nil || o.OperationID != "" || o.InstallationID != "" || o.ActorID != 0 || o.RepositoryID != 0 || o.Effect != "" || o.Cancellation != "" ||
			o.Completion != "" || o.Reason != "" || o.Receipt != "" || o.UpdatedUnix != 0 {
			return errors.New("unsubmitted operation carries no observations")
		}
		return o.validateLinks()
	}
	if !ValidPublicationOperationID(o.OperationID) {
		return errors.New("invalid publication operation identity")
	}
	if o.Work == nil || o.Work.OperationID != o.OperationID {
		return errors.New("publication operation lacks its immutable intent")
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
		return errors.New("invalid publication cancellation state")
	}
	if o.Completion != "" && !ValidOpCompletion(o.Completion) {
		return errors.New("invalid publication completion state")
	}
	if len(o.Reason) > 64 || len(o.Receipt) > MaxPublicationReceipt {
		return errors.New("publication observation exceeds bounds")
	}
	return o.validateLinks()
}

func (o PublicationOperation) validateLinks() error {
	if o.PRNumber < 0 || o.PRID < 0 || o.IssueID < 0 {
		return errors.New("invalid publication native links")
	}
	if o.Kind == OpRefPublish {
		if o.PRNumber != 0 || o.PRID != 0 || o.IssueID != 0 {
			return errors.New("branch operation carries no PR links")
		}
		if o.HeadRef != "" || o.BaseRef != "" || o.HeadOID != "" || o.BaseOID != "" {
			return errors.New("branch operation carries no PR snapshot")
		}
		return nil
	}
	if (o.PRNumber == 0) != (o.PRID == 0) || (o.PRNumber == 0) != (o.IssueID == 0) {
		return errors.New("incomplete publication PR links")
	}
	// A committed creation may await adoption: links arrive only from
	// its own verified receipt. An uncommitted creation carries no
	// links, and partial links never stand.
	if o.Effect != OpEffectCommitted && o.PRNumber != 0 {
		return errors.New("uncommitted PR creation carries no PR links")
	}
	if (o.HeadRef != "" && !ValidTargetBranch(o.HeadRef)) || (o.BaseRef != "" && !ValidTargetBranch(o.BaseRef)) {
		return errors.New("invalid publication PR snapshot refs")
	}
	if (o.HeadOID != "" && !ValidCommit(o.HeadOID)) || (o.BaseOID != "" && !ValidCommit(o.BaseOID)) {
		return errors.New("invalid publication PR snapshot tips")
	}
	if o.PRNumber != 0 && (o.HeadRef == "" || o.BaseRef == "" || o.HeadOID == "" || o.BaseOID == "") {
		return errors.New("publication PR links lack their snapshot")
	}
	return nil
}

// ValidPublicationOperationID reports whether id is admissible as a native
// conditional operation identity: the contract's alphabet, bounded length.
func ValidPublicationOperationID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			c == '-' || c == '_' || c == '.' || c == ':' {
			continue
		}
		return false
	}
	return true
}

// PublicationOperationID derives the deterministic identity for one
// publication's kind attempt. The identity is persisted before its submit
// so a lost reply reconciles by lookup instead of resubmitting blindly;
// terminal identities are never reused.
func PublicationOperationID(publicationID, kind string, attempt int) string {
	suffix := "publish"
	if kind == OpPRCreate {
		suffix = "prcreate"
	}
	return fmt.Sprintf("soda-%s-%s-%d", publicationID, suffix, attempt)
}

// PublicationBranch derives the deterministic attributable branch for one
// assignment's publication. Corrections republish to this same branch with
// the prior tip as the expected lease; nothing else may ride along.
func PublicationBranch(assignmentID string) string {
	return "soda/factory/" + assignmentID
}

// Publication is one assignment's durable publication: the exact bound
// candidate and target, the two separately persisted conditional
// operations, the fresh observations the current attempt bound, and the
// exact linked PR once creation commits. Finished work is terminal and
// keeps its receipts for inspection.
type Publication struct {
	WithdrawRequested bool                 `json:"withdraw_requested,omitempty"`
	Publish           PublicationOperation `json:"publish"`
	PRCreate          PublicationOperation `json:"pr_create"`
	Authority         AuthorityRef         `json:"authority"`
	ID                string               `json:"id"`
	AssignmentID      string               `json:"assignment_id"`
	ProjectID         string               `json:"project_id"`
	Role              string               `json:"role"`
	Acceptance        string               `json:"acceptance"`
	Preparation       string               `json:"preparation"`
	Run               string               `json:"run"`
	Candidate         string               `json:"candidate"`
	BaseSHA           string               `json:"base_sha"`
	TargetBranch      string               `json:"target_branch"`
	Comparison        string               `json:"comparison_oid,omitempty"`
	TargetTip         string               `json:"target_tip,omitempty"`
	Stage             string               `json:"stage"`
	Outcome           Outcome              `json:"outcome,omitempty"`
	Reason            string               `json:"reason,omitempty"`
	Repository        int64                `json:"repository,string"`
	Issue             int64                `json:"issue,string"`
	Revision          int64                `json:"revision"`
	NativeRev         int64                `json:"native_revision"`
	PRNumber          int64                `json:"pr_number,omitempty"`
	PRID              int64                `json:"pr_id,omitempty"`
	CreatedUnix       int64                `json:"created_unix"`
	ObservedUnix      int64                `json:"observed_unix,omitempty"`
	FinishedUnix      int64                `json:"finished_unix,omitempty"`
}

// Validate rejects malformed publications. Open work carries no outcome;
// finished work carries its outcome, reason and finish time. A published
// publication links exactly the PR its own creation receipt proved.
func (p Publication) Validate() error {
	if !ValidID(p.ID) || !ValidID(p.AssignmentID) {
		return errors.New("invalid publication identity")
	}
	if !ValidProjectID(p.ProjectID) || !project.ValidFactoryRole(p.Role) {
		return errors.New("invalid publication address")
	}
	if p.Repository <= 0 || p.Issue <= 0 || p.Revision < 0 {
		return errors.New("invalid publication scope")
	}
	if !project.ValidDecisionID(p.Acceptance) {
		return errors.New("invalid publication acceptance")
	}
	if !ValidPreparationRef(p.Preparation) {
		return errors.New("invalid publication preparation")
	}
	if !ValidID(p.Run) {
		return errors.New("invalid publication run")
	}
	if !ValidCommit(p.Candidate) || !ValidCommit(p.BaseSHA) || p.Candidate == p.BaseSHA {
		return errors.New("publication candidate is not a fresh exact commit")
	}
	if !ValidTargetBranch(p.TargetBranch) {
		return errors.New("invalid publication target branch")
	}
	if p.Comparison != "" && !ValidCommit(p.Comparison) {
		return errors.New("invalid publication comparison observation")
	}
	if p.TargetTip != "" && !ValidCommit(p.TargetTip) {
		return errors.New("invalid publication target observation")
	}
	if p.NativeRev < 0 || (p.NativeRev == 0) != (p.ObservedUnix == 0) {
		return errors.New("invalid publication observation binding")
	}
	if p.ObservedUnix < 0 || p.CreatedUnix <= 0 {
		return errors.New("invalid publication time")
	}
	if err := p.Publish.Validate(); err != nil {
		return err
	}
	if err := p.PRCreate.Validate(); err != nil {
		return err
	}
	if p.Publish.Kind != OpRefPublish || p.PRCreate.Kind != OpPRCreate {
		return errors.New("publication operations have the wrong kinds")
	}
	for _, op := range []PublicationOperation{p.Publish, p.PRCreate} {
		if op.Work != nil && (op.Work.Candidate != p.Candidate || op.Work.Repository != p.Repository || op.Work.TargetBranch != p.TargetBranch) {
			return errors.New("publication operation intent differs from its publication")
		}
	}
	if (p.PRNumber == 0) != (p.PRID == 0) {
		return errors.New("incomplete publication PR links")
	}
	if p.PRNumber != 0 && (p.PRCreate.Effect != OpEffectCommitted || p.PRCreate.PRNumber != p.PRNumber || p.PRCreate.PRID != p.PRID) {
		return errors.New("publication PR is not the recorded creation effect")
	}
	switch p.Stage {
	case PublicationOpen:
		if p.Outcome != "" || p.Reason != "" || p.FinishedUnix != 0 {
			return errors.New("open publication carries no outcome")
		}
	case PublicationPublished:
		if p.Outcome != Succeeded || p.Reason != PublishReasonLinked || p.FinishedUnix <= 0 {
			return errors.New("published publication lacks its outcome")
		}
		if p.PRNumber <= 0 || p.PRID <= 0 {
			return errors.New("published publication lacks its PR links")
		}
		if p.PRCreate.Effect != OpEffectCommitted || p.PRCreate.PRNumber != p.PRNumber || p.PRCreate.PRID != p.PRID {
			return errors.New("published PR is not the recorded creation effect")
		}
		if p.Publish.Effect != OpEffectCommitted {
			return errors.New("published publication lacks its committed branch")
		}
		if p.Publish.Completion != OpCompletionComplete || p.PRCreate.Completion != OpCompletionComplete {
			return errors.New("published publication lacks native completion")
		}
	case PublicationFailed, PublicationWithdrawn, PublicationFenced:
		if !validPublishReason(p.Reason) || p.FinishedUnix <= 0 {
			return errors.New("finished publication lacks its outcome")
		}
		if p.Reason == PublishReasonLinked {
			return errors.New("unfinished publication cannot claim completed linkage")
		}
		switch p.Stage {
		case PublicationFailed:
			if p.Outcome != Failed {
				return errors.New("failed publication lacks its outcome")
			}
		case PublicationWithdrawn:
			if p.Outcome != Cancelled {
				return errors.New("withdrawn publication lacks its outcome")
			}
			for _, op := range []PublicationOperation{p.Publish, p.PRCreate} {
				if op.Attempts != 0 && op.Effect != OpEffectCommitted && op.Effect != OpEffectNotCommitted {
					return errors.New("withdrawn publication still has an unresolved native effect")
				}
			}
		case PublicationFenced:
			if p.Outcome != NeedsHuman {
				return errors.New("fenced publication lacks its outcome")
			}
		}
	default:
		return errors.New("invalid publication stage")
	}
	return nil
}

// PublicationWork is one executor call's exact bound inputs: the candidate
// bundle with its run/base identities, the policy-bound target and actor,
// the bracketed observation, the final PR title and body, and the
// persisted operation identity the call must use. No credential or
// credential-adjacent value enters the work record; secrets travel in
// restricted files outside it.
type PublicationWork struct {
	Bundle           []byte
	AssignmentID     string
	Publication      string
	RunID            string
	Run              Run
	Candidate        string
	BaseSHA          string
	TargetBranch     string
	OperationID      string
	AuthRevision     string
	ExpectedOld      string
	ComparisonRef    string
	ComparisonOID    string
	PRTitle          string
	PRBody           string
	Repository       int64
	Issue            int64
	ActorID          int64
	NativeRev        int64
	NotAfter         int64
	CorrectionNumber int64
	CorrectionAuthor int64
}

// MaxPublicationBody bounds the final PR body: attributable references
// only, never unbounded native text.
const MaxPublicationBody = 16 << 10

// PublicationIntent contains only the immutable native inputs. Bundles,
// supervisor credentials and changing run state stay outside this record.
type PublicationIntent struct {
	OperationID      string `json:"operation_id"`
	AuthRevision     string `json:"authorization_revision"`
	TargetBranch     string `json:"target_branch"`
	Candidate        string `json:"candidate"`
	ExpectedOld      string `json:"expected_old"`
	ComparisonRef    string `json:"comparison_ref"`
	ComparisonOID    string `json:"comparison_oid"`
	PRTitle          string `json:"pr_title"`
	PRBody           string `json:"pr_body"`
	Repository       int64  `json:"repository,string"`
	ActorID          int64  `json:"actor_id,string"`
	NativeRev        int64  `json:"native_revision"`
	NotAfter         int64  `json:"not_after"`
	CorrectionNumber int64  `json:"correction_number,omitempty"`
	CorrectionAuthor int64  `json:"correction_author,omitempty"`
}

// Intent takes the credential-free native inputs from an executor call.
func (w PublicationWork) Intent() PublicationIntent {
	return PublicationIntent{
		OperationID: w.OperationID, AuthRevision: w.AuthRevision,
		TargetBranch: w.TargetBranch, Candidate: w.Candidate, ExpectedOld: w.ExpectedOld,
		ComparisonRef: w.ComparisonRef, ComparisonOID: w.ComparisonOID,
		PRTitle: w.PRTitle, PRBody: w.PRBody, Repository: w.Repository, ActorID: w.ActorID,
		NativeRev: w.NativeRev, NotAfter: w.NotAfter,
		CorrectionNumber: w.CorrectionNumber, CorrectionAuthor: w.CorrectionAuthor,
	}
}

// Apply restores an exact native intent without changing local bundle/run inputs.
func (i PublicationIntent) Apply(w PublicationWork) PublicationWork {
	w.OperationID, w.AuthRevision = i.OperationID, i.AuthRevision
	w.TargetBranch, w.Candidate, w.ExpectedOld = i.TargetBranch, i.Candidate, i.ExpectedOld
	w.ComparisonRef, w.ComparisonOID = i.ComparisonRef, i.ComparisonOID
	w.PRTitle, w.PRBody, w.Repository, w.ActorID = i.PRTitle, i.PRBody, i.Repository, i.ActorID
	w.NativeRev, w.NotAfter = i.NativeRev, i.NotAfter
	w.CorrectionNumber, w.CorrectionAuthor = i.CorrectionNumber, i.CorrectionAuthor
	return w
}

// Validate rejects incomplete immutable native intents.
func (i PublicationIntent) Validate() error { return i.validate(true) }

func (i PublicationIntent) validate(observed bool) error {
	if !ValidTargetBranch(i.TargetBranch) || !ValidCommit(i.Candidate) {
		return errors.New("invalid publication target or candidate")
	}
	if !ValidPublicationOperationID(i.OperationID) {
		return errors.New("invalid publication operation identity")
	}
	if i.Repository <= 0 || i.ActorID <= 0 {
		return errors.New("invalid publication work scope")
	}
	if i.AuthRevision == "" || len(i.AuthRevision) > 512 {
		return errors.New("invalid publication authorization revision")
	}
	if i.NativeRev < 0 || i.NotAfter < 0 || (observed && (i.NativeRev == 0 || i.NotAfter == 0)) {
		return errors.New("invalid publication observation binding")
	}
	if i.ExpectedOld != "absent" && !ValidCommit(i.ExpectedOld) {
		return errors.New("invalid publication expected lease")
	}
	if !ValidTargetBranch(i.ComparisonRef) || ((observed || i.ComparisonOID != "") && !ValidCommit(i.ComparisonOID)) {
		return errors.New("invalid publication comparison binding")
	}
	if strings.TrimSpace(i.PRTitle) == "" || len(strings.TrimSpace(i.PRTitle)) > 255 {
		return errors.New("invalid publication PR title")
	}
	if len(i.PRBody) > MaxPublicationBody {
		return errors.New("publication PR body exceeds bounds")
	}
	if (i.CorrectionNumber == 0) != (i.CorrectionAuthor == 0) || i.CorrectionNumber < 0 || i.CorrectionAuthor < 0 {
		return errors.New("invalid publication correction binding")
	}
	if i.CorrectionNumber != 0 && i.ExpectedOld == "absent" {
		return errors.New("initial publication carries no correction")
	}
	return nil
}

// ValidateObservation checks candidate and identity inputs before native refs
// are observed; only the native revision and comparison tip may be absent.
func (w PublicationWork) ValidateObservation() error {
	if len(w.Bundle) == 0 || len(w.Bundle) > 4<<20 {
		return errors.New("publication bundle exceeds input limit")
	}
	if !ValidID(w.AssignmentID) || !ValidID(w.Publication) || !ValidID(w.RunID) || w.Issue <= 0 {
		return errors.New("invalid publication work identity")
	}
	if err := w.Run.Validate(); err != nil || w.Run.ID != w.RunID {
		return errors.New("invalid publication run")
	}
	if !ValidCommit(w.BaseSHA) || w.Candidate == w.Run.InputSHA || w.Candidate == w.BaseSHA {
		return errors.New("publication candidate is not a fresh exact commit")
	}
	if w.ComparisonRef == PublishBranchName(w.AssignmentID) {
		return errors.New("publication comparison is its own target")
	}
	return w.Intent().validate(false)
}

// Validate adds the exact native observation needed before submission.
func (w PublicationWork) Validate() error {
	if err := w.ValidateObservation(); err != nil {
		return err
	}
	return w.Intent().Validate()
}

// PRTitleFor derives the deterministic attributable PR title for one
// assignment's publication: the issue it answers, never untrusted text.
func PRTitleFor(issue int64) string {
	return fmt.Sprintf("Factory candidate for #%d", issue)
}

// PRBodyFor derives the deterministic attributable PR body: the exact
// bound references behind the candidate.
func PRBodyFor(assignmentID, acceptance, runID, candidate, source string) string {
	var b strings.Builder
	b.WriteString("Automated factory publication.\n\n")
	b.WriteString("Assignment: " + assignmentID + "\n")
	b.WriteString("Acceptance: " + acceptance + "\n")
	b.WriteString("Run: " + runID + "\n")
	b.WriteString("Candidate: " + candidate + "\n")
	b.WriteString("Source: " + source + "\n")
	return b.String()
}

// PublicationObservation is one executor observation: the bracketed idle
// native revision plus the exact target and comparison tips behind it. An
// empty TargetTip means the branch is absent; any other tip is exact.
type PublicationObservation struct {
	TargetRef    string
	TargetTip    string
	Comparison   string
	NativeRev    int64
	ObservedUnix int64
}

// PublicationWithdrawal reports ordered local withdrawal separately from
// outstanding native effects. Operation identities remain available for recovery.
type PublicationWithdrawal struct {
	Publications []string `json:"publications"`
	Operations   []string `json:"operations"`
	Pending      bool     `json:"pending"`
}

// OperationOutcome is one reconciled native operation answer: the effect,
// cancellation and completion states plus the bounded reason and receipt.
// NotObserved reports a lookup that found no record; absence never proves
// that an earlier request cannot still arrive.
type OperationOutcome struct {
	OperationID    string
	InstallationID string
	Kind           string
	ActorID        int64
	RepositoryID   int64
	Receipt        []byte
	Effect         string
	Cancellation   string
	Completion     string
	Reason         string
	NotObserved    bool
}

// PublicationRefusal is a terminal executor verdict: the publication
// will not proceed under its recorded inputs and the coordinator
// records its failure instead of retrying an identical call.
type PublicationRefusal struct{ Reason string }

func (e *PublicationRefusal) Error() string { return "publication refused: " + e.Reason }

// PublicationWait reports that no verdict exists yet: the call reached
// no native decision and the publication stays open for a later pass.
// It never advances the stage.
type PublicationWait struct{ Reason string }

func (e *PublicationWait) Error() string { return "publication waits: " + e.Reason }

// BranchOutcome is one reconciled branch publication: the operation state
// plus the realized ref tips adopted from its own committed receipt.
type BranchOutcome struct {
	Operation  OperationOutcome
	Ref        string
	OldOID     string
	NewOID     string
	Comparison string
}

// PRCreationOutcome is one reconciled PR creation: the operation state plus
// the exact PR identity adopted from its own committed receipt.
type PRCreationOutcome struct {
	Operation OperationOutcome
	HeadRef   string
	BaseRef   string
	HeadOID   string
	BaseOID   string
	PRNumber  int64
	PRID      int64
	IssueID   int64
	AuthorID  int64
}

// PublishBranchName returns the full native ref for one assignment's
// publication branch.
func PublishBranchName(assignmentID string) string {
	return "refs/heads/" + PublicationBranch(assignmentID)
}

// AuthRevisionFor binds one publication's Soda authority reference into
// the opaque native authorization revision: assignment, publication and
// Soda revision, never a credential.
func AuthRevisionFor(assignmentID, publicationID string, revision int64) string {
	return fmt.Sprintf("soda-assignment:%s:publication:%s:revision:%d", assignmentID, publicationID, revision)
}

// RefHead returns the branch name without its refs/heads/ prefix, or empty
// for a malformed ref.
func RefHead(ref string) string {
	name, ok := strings.CutPrefix(ref, "refs/heads/")
	if !ok || name == "" || strings.ContainsAny(ref, " ~^:?*\\") || strings.Contains(ref, "..") {
		return ""
	}
	return name
}
