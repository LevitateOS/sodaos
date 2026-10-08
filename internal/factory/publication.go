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

// Review admission bounds the append-only native review history retained in
// one publication: at most 128 records and 1 MiB summed across each
// ReviewWork.Body plus the maximum raw receipt capacity reserved for that
// operation. Reserving receipt capacity keeps admitted pending work
// persistable when its native outcome arrives. These resource ceilings do
// not limit configured review or correction cycles; callers refuse a new
// review before native dispatch when either ceiling is reached, preserving
// history.
const (
	MaxPublicationReviewOperations = 128
	MaxPublicationReviewBytes      = 1 << 20
)

// Publication is one assignment's durable publication: the exact bound
// candidate and target, the separately persisted branch and PR operations,
// any immutable review work and its native outcome, the fresh observations
// the current attempt bound, and the exact linked PR once creation commits.
// Finished work keeps its receipts for inspection.
type Publication struct {
	WithdrawRequested bool                 `json:"withdraw_requested,omitempty"`
	Publish           PublicationOperation `json:"publish"`
	PRCreate          PublicationOperation `json:"pr_create"`
	Corrections       CorrectionOps        `json:"corrections,omitempty"`
	ReviewOperations  []ReviewOperation    `json:"review_operations,omitempty"`
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
	if !ValidProjectID(p.ProjectID) || p.Role != project.RoleCoder {
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
	initial := p.Candidate
	if len(p.Corrections) > 0 {
		if (p.Stage != PublicationPublished && p.Stage != PublicationFenced) || p.PRNumber <= 0 || p.PRCreate.Work == nil || p.Publish.Work == nil {
			return errors.New("corrections require a published linked PR")
		}
		initial = p.Publish.Work.Candidate
		head := initial
		for i := range p.Corrections {
			correction := p.Corrections[i]
			if err := correction.Validate(); err != nil {
				return err
			}
			if correction.Kind != OpRefPublish {
				return errors.New("correction is not a branch publication")
			}
			if correction.OperationID != PublicationOperationID(p.ID, OpRefPublish, i+2) {
				return errors.New("correction identity is not the next branch attempt")
			}
			work := correction.Work
			if work == nil {
				return errors.New("correction lacks its recorded intent")
			}
			if work.CorrectionNumber != p.PRNumber || work.CorrectionAuthor != p.PRCreate.Work.ActorID {
				return errors.New("correction does not bind the linked PR")
			}
			if work.Repository != p.Repository || work.TargetBranch != p.TargetBranch {
				return errors.New("correction intent differs from its publication")
			}
			if work.ExpectedOld != head || !ValidCommit(work.Candidate) || work.Candidate == head {
				return errors.New("correction does not chain its predecessor tip")
			}
			if correction.Effect == OpEffectCommitted {
				head = work.Candidate
			}
		}
		if head != p.Candidate {
			return errors.New("publication head differs from its committed corrections")
		}
	}
	if err := p.validateReviewOperations(nil); err != nil {
		return err
	}
	for _, op := range []PublicationOperation{p.Publish, p.PRCreate} {
		if op.Work != nil && (op.Work.Candidate != initial || op.Work.Repository != p.Repository || op.Work.TargetBranch != p.TargetBranch) {
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

// CanAppendReviewOperation performs the domain admission check before the
// coordinator retains a new work item or reaches the native submit call.
func (p Publication) CanAppendReviewOperation(op ReviewOperation) error {
	return p.validateReviewOperations(&op)
}

func (p Publication) validateReviewOperations(additional *ReviewOperation) error {
	count := len(p.ReviewOperations)
	if additional != nil {
		count++
	}
	if count == 0 {
		return nil
	}
	if (p.Stage != PublicationPublished && p.Stage != PublicationFenced) || p.PRCreate.Work == nil || p.PRCreate.Effect != OpEffectCommitted {
		return errors.New("review operations require a linked publication")
	}
	if count > MaxPublicationReviewOperations {
		return errors.New("publication review history exceeds its record bound")
	}
	seen := make(map[string]bool, count)
	retained := 0
	for i := 0; i < count; i++ {
		var op ReviewOperation
		if i < len(p.ReviewOperations) {
			op = p.ReviewOperations[i]
		} else {
			op = *additional
		}
		if err := op.ValidateForPublication(p); err != nil {
			return err
		}
		retained += len(op.Work.Body) + MaxPublicationReceipt
		if retained > MaxPublicationReviewBytes {
			return errors.New("publication review history exceeds its byte bound")
		}
		if seen[op.RunID] {
			return errors.New("duplicate review run operation")
		}
		seen[op.RunID] = true
	}
	return nil
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
