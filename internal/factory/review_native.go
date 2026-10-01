package factory

import (
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
