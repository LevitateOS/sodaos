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
	if !ValidPublicationOperationID(w.OperationID) || strings.TrimSpace(w.AuthRevision) == "" || len(w.AuthRevision) > 512 || w.NativeRev < 1 || w.NotAfter <= 0 {
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
	end := strings.Index(rest, "```")
	if end < 0 {
		return ReviewReport{}, false
	}
	body := strings.TrimSpace(rest[:end])
	if body == "" || len(body) > 65536+8192 {
		return ReviewReport{}, false
	}
	var report ReviewReport
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return ReviewReport{}, false
	}
	if report.Validate() != nil {
		return ReviewReport{}, false
	}
	return report, true
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
