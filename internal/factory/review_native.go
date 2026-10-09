package factory

import (
	"encoding/json"
	"errors"
	"strings"
)

// ReviewWork is one immutable body-only review authorization. Native PR and
// issue identities come from the recorded creation receipt, not a PR search.
// The reviewer owns findings; this work grants no ref publication.
type ReviewWork struct {
	OperationID  string `json:"operation_id"`
	AuthRevision string `json:"authorization_revision"`
	Repository   int64  `json:"repository"`
	ActorID      int64  `json:"actor_id"`
	PRNumber     int64  `json:"pr_number"`
	PRID         int64  `json:"pr_id"`
	IssueID      int64  `json:"issue_id"`
	PRAuthorID   int64  `json:"pr_author_id"`
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	HeadOID      string `json:"head_oid"`
	BaseOID      string `json:"base_oid"`
	NativeRev    int64  `json:"native_revision"`
	NotAfter     int64  `json:"not_after"`
	Event        string `json:"event"`
	Body         string `json:"body"`
}

// ReviewOperation retains one review run's exact immutable native work and
// the latest attributable operation outcome inside its publication record.
// The work itself is the native tuple; this wrapper owns only its run binding
// and reconciliation state.
type ReviewOperation struct {
	RunID   string           `json:"run_id"`
	Work    ReviewWork       `json:"work"`
	Outcome OperationOutcome `json:"outcome"`
}

// ValidateForPublication checks the durable operation against the exact PR
// identity and candidate history that admitted it. Historical actor, head and
// deadline values remain immutable when current policy or publication state
// later changes.
func (op ReviewOperation) ValidateForPublication(p Publication) error {
	if !ValidID(op.RunID) || op.Work.OperationID != "review-"+op.RunID {
		return errors.New("invalid review run operation identity")
	}
	if err := op.Work.Validate(); err != nil {
		return err
	}
	if op.Work.Repository != p.Repository || op.Work.PRNumber != p.PRNumber || op.Work.PRID != p.PRID ||
		op.Work.IssueID != p.PRCreate.IssueID || op.Work.PRAuthorID != p.PRCreate.Work.ActorID ||
		op.Work.HeadRef != p.PRCreate.HeadRef || op.Work.BaseRef != p.PRCreate.BaseRef || op.Work.BaseOID != p.PRCreate.BaseOID ||
		op.Work.AuthRevision != ReviewAuthRevision(p.AssignmentID, op.RunID) {
		return errors.New("review work differs from its linked publication")
	}
	knownHead := p.Publish.Work != nil && op.Work.HeadOID == p.Publish.Work.Candidate
	for _, correction := range p.Corrections {
		if correction.Effect == OpEffectCommitted && correction.Work != nil && op.Work.HeadOID == correction.Work.Candidate {
			knownHead = true
		}
	}
	if !knownHead {
		return errors.New("review head is outside the publication candidate history")
	}
	if op.Outcome.NotObserved {
		return errors.New("review outcome cannot persist an absent lookup")
	}
	if op.Outcome.OperationID == "" {
		if op.Outcome.InstallationID != "" || op.Outcome.Kind != "" || op.Outcome.ActorID != 0 || op.Outcome.RepositoryID != 0 ||
			op.Outcome.Effect != "" || op.Outcome.Cancellation != "" || op.Outcome.Completion != "" || op.Outcome.Reason != "" || len(op.Outcome.Receipt) != 0 {
			return errors.New("review outcome is incomplete")
		}
	} else if op.Outcome.OperationID != op.Work.OperationID {
		return errors.New("review outcome identity differs")
	}
	if op.Outcome.InstallationID != "" && len(op.Outcome.InstallationID) > 256 {
		return errors.New("review installation identity exceeds bounds")
	}
	if op.Outcome.Effect != "" && !ValidOpEffect(op.Outcome.Effect) {
		return errors.New("invalid review operation effect")
	}
	if op.Outcome.Cancellation != "" && !ValidOpCancellation(op.Outcome.Cancellation) {
		return errors.New("invalid review operation cancellation")
	}
	if op.Outcome.Completion != "" && !ValidOpCompletion(op.Outcome.Completion) {
		return errors.New("invalid review operation completion")
	}
	if len(op.Outcome.Reason) > 64 || len(op.Outcome.Receipt) > MaxPublicationReceipt {
		return errors.New("review operation outcome exceeds bounds")
	}
	if op.Outcome.Effect == OpEffectCommitted && (op.Outcome.InstallationID == "" || len(op.Outcome.Receipt) == 0 || op.Outcome.Completion == "") {
		return errors.New("committed review lacks its receipt or completion")
	}
	tombstone := op.Outcome.OperationID != "" && op.Outcome.Kind == "" && op.Outcome.ActorID == 0 && op.Outcome.RepositoryID == 0 &&
		op.Outcome.Effect == OpEffectNotCommitted && op.Outcome.Cancellation == OpCancelCancelled
	if op.Outcome.OperationID != "" && !tombstone && (op.Outcome.InstallationID == "" || op.Outcome.Kind != OpReviewSubmit || op.Outcome.ActorID != op.Work.ActorID || op.Outcome.RepositoryID != op.Work.Repository) {
		return errors.New("review operation outcome scope differs")
	}
	return nil
}

// WithOutcome adopts one native lookup/cancel/submit response without
// permitting the stable review identity or a terminal result to move.
func (op ReviewOperation) WithOutcome(outcome OperationOutcome) (ReviewOperation, error) {
	if outcome.NotObserved || outcome.OperationID != op.Work.OperationID {
		return ReviewOperation{}, errors.New("review outcome identity differs")
	}
	tombstone := outcome.Kind == "" && outcome.ActorID == 0 && outcome.RepositoryID == 0 &&
		outcome.Effect == OpEffectNotCommitted && outcome.Cancellation == OpCancelCancelled
	if !tombstone && (outcome.InstallationID == "" || outcome.Kind != OpReviewSubmit ||
		outcome.ActorID != op.Work.ActorID || outcome.RepositoryID != op.Work.Repository) {
		return ReviewOperation{}, errors.New("review outcome scope differs")
	}
	prior := op.Outcome
	if (prior.Effect == OpEffectCommitted || prior.Effect == OpEffectNotCommitted) && prior.Effect != outcome.Effect {
		return ReviewOperation{}, errors.New("review terminal effect changed")
	}
	if prior.InstallationID != "" && prior.InstallationID != outcome.InstallationID {
		return ReviewOperation{}, errors.New("review installation identity changed")
	}
	if len(prior.Receipt) > 0 && string(prior.Receipt) != string(outcome.Receipt) {
		return ReviewOperation{}, errors.New("review receipt changed")
	}
	op.Outcome = outcome
	return op, nil
}

// ValidateTarget checks the exact target before findings or operation identity
// exist. The native actor must be separate from the recorded PR author.
func (w ReviewWork) ValidateTarget() error {
	if w.Repository <= 0 || w.ActorID <= 0 || w.PRNumber <= 0 || w.PRID <= 0 || w.IssueID <= 0 || w.PRAuthorID <= 0 || w.ActorID == w.PRAuthorID {
		return errors.New("invalid independent review target")
	}
	if !ValidTargetBranch(w.HeadRef) || !ValidTargetBranch(w.BaseRef) || w.HeadRef == w.BaseRef || !ValidCommit(w.HeadOID) || !ValidCommit(w.BaseOID) {
		return errors.New("invalid exact review candidate")
	}
	return nil
}

// Validate checks persisted semantic inputs. Deadline freshness belongs to the
// caller at dispatch; expired historical work remains valid for reconciliation.
func (w ReviewWork) Validate() error {
	if err := w.ValidateTarget(); err != nil {
		return err
	}
	if !ValidPublicationOperationID(w.OperationID) || strings.TrimSpace(w.AuthRevision) == "" || len(w.AuthRevision) > 512 || w.NativeRev < 1 || w.NotAfter <= 0 || len(w.Body) > 65536 {
		return errors.New("invalid review authorization")
	}
	switch w.Event {
	case "APPROVED":
	case "REQUEST_CHANGES":
		if strings.TrimSpace(w.Body) == "" {
			return errors.New("requested changes need findings")
		}
	default:
		return errors.New("invalid review disposition")
	}
	return nil
}

// ReviewObservation reports the exact current native target after equal idle
// revision observations. Native review history is checked in that same bracket.
type ReviewObservation struct {
	NativeRev    int64  `json:"native_revision"`
	ObservedUnix int64  `json:"observed_unix"`
	PRID         int64  `json:"pr_id"`
	PRNumber     int64  `json:"pr_number"`
	IssueID      int64  `json:"issue_id"`
	PRAuthorID   int64  `json:"pr_author_id"`
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	HeadOID      string `json:"head_oid"`
	BaseOID      string `json:"base_oid"`
}

// ReviewAuthRevision binds one submitted review to its assignment and run:
// the opaque native authorization revision, never a credential.
func ReviewAuthRevision(assignmentID, runID string) string {
	return "soda-assignment:" + assignmentID + ":review-run:" + runID
}

// ReviewReportFence delimits the reviewer report block: the review prompt
// instructs the CLI to close its work with exactly one fenced review-json
// block, and the supervisor parses the last such block strictly.
const ReviewReportFence = "review-json"

// ReviewReport is one reviewer agent's genuine verdict on an exact
// candidate: approve it or request changes with concrete findings.
type ReviewReport struct {
	Verdict  string   `json:"verdict"`
	Summary  string   `json:"summary"`
	Body     string   `json:"body"`
	Findings []string `json:"findings"`
}

// Validate checks the verdict shape. Approvals carry a summary; requested
// changes carry findings in the body, never an empty report.
func (r ReviewReport) Validate() error {
	switch r.Verdict {
	case "approve":
	case "request-changes":
		if strings.TrimSpace(r.Body) == "" {
			return errors.New("requested changes need findings")
		}
	default:
		return errors.New("invalid review verdict")
	}
	if len(r.Body) > 65536 || len(r.Summary) > 4096 {
		return errors.New("review report exceeds its bound")
	}
	if len(r.Findings) > 64 {
		return errors.New("review report has too many findings")
	}
	for _, finding := range r.Findings {
		if len(finding) > 4096 {
			return errors.New("review finding is too large")
		}
	}
	return nil
}

// ParseReviewReport extracts the last fenced review block from reviewer
// output and validates it strictly. Unknown fields refuse; anything
// unparseable reports false instead of guessing a verdict.
func ParseReviewReport(output string) (ReviewReport, bool) {
	start := strings.LastIndex(output, "```"+ReviewReportFence)
	if start < 0 {
		return ReviewReport{}, false
	}
	rest := output[start+len("```"+ReviewReportFence):]
	// The JSON payload may itself contain nested fences (quoted diffs,
	// code samples), so the first fence after the opener is unreliable:
	// try every closing fence in order until one decodes. The first
	// valid payload wins; trailing chatter never mis-terminates it.
	for offset := 0; offset < len(rest); {
		rel := strings.Index(rest[offset:], "```")
		if rel < 0 {
			return ReviewReport{}, false
		}
		body := strings.TrimSpace(rest[:offset+rel])
		offset += rel + len("```")
		if body == "" || len(body) > 65536+8192 {
			continue
		}
		var report ReviewReport
		decoder := json.NewDecoder(strings.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&report); err != nil {
			continue
		}
		if report.Validate() != nil {
			continue
		}
		return report, true
	}
	return ReviewReport{}, false
}

// ReviewOutcome is the attributable native review identity adopted from its
// operation's committed receipt. It is historical evidence for this exact head
// and base; callers must re-observe before using it for a later decision.
type ReviewOutcome struct {
	Operation  OperationOutcome
	ReviewID   int64
	CommentID  int64
	ReviewerID int64
	PRID       int64
	PRNumber   int64
	IssueID    int64
	HeadOID    string
	BaseOID    string
	CommitID   string
	Event      string
}
