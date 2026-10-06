package publish

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

// PublishReceipt decodes one committed branch-publication receipt: the
// realized ref tuple plus the comparison tip behind it. Only the
// operation's own receipt proves its effect.
type PublishReceipt struct {
	Ref           string `json:"ref"`
	OldOID        string `json:"old_oid"`
	NewOID        string `json:"new_oid"`
	ComparisonRef string `json:"comparison_ref"`
	ComparisonOID string `json:"comparison_oid"`
	ActorID       int64  `json:"actor_id"`
	RepositoryID  int64  `json:"repository_id"`
}

// DecodePublishReceipt verifies one committed publish outcome against its
// submitted tuple and adopts the realized effect. A missing, oversized,
// malformed or mismatched receipt refuses: the branch cannot advance on
// an unattributed effect.
func DecodePublishReceipt(outcome factory.OperationOutcome, ref, expectedOld, newOID, comparisonRef, comparisonOID string, actorID, repositoryID int64) (factory.BranchOutcome, error) {
	branch := factory.BranchOutcome{Operation: outcome}
	if outcome.Effect != factory.OpEffectCommitted {
		return factory.BranchOutcome{}, errors.New("branch outcome is not committed")
	}
	if len(outcome.Receipt) == 0 || len(outcome.Receipt) > factory.MaxPublicationReceipt {
		return factory.BranchOutcome{}, errors.New("branch receipt is missing")
	}
	var receipt PublishReceipt
	decoder := json.NewDecoder(strings.NewReader(string(outcome.Receipt)))
	if err := decoder.Decode(&receipt); err != nil {
		return factory.BranchOutcome{}, errors.New("branch receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return factory.BranchOutcome{}, errors.New("branch receipt is malformed")
	}
	if receipt.Ref != ref || receipt.NewOID != newOID || receipt.ComparisonRef != comparisonRef ||
		receipt.RepositoryID != repositoryID || receipt.ActorID != actorID || actorID <= 0 ||
		!factory.ValidCommit(receipt.ComparisonOID) || !strings.EqualFold(receipt.ComparisonOID, comparisonOID) {
		return factory.BranchOutcome{}, errors.New("branch receipt differs from its intent")
	}
	if expectedOld == extensions.PublishExpectedOldAbsent {
		if receipt.OldOID != strings.Repeat("0", 40) && receipt.OldOID != strings.Repeat("0", 64) {
			return factory.BranchOutcome{}, errors.New("branch receipt differs from its intent")
		}
	} else if !strings.EqualFold(receipt.OldOID, expectedOld) {
		return factory.BranchOutcome{}, errors.New("branch receipt differs from its intent")
	}
	branch.Ref, branch.OldOID, branch.NewOID, branch.Comparison = receipt.Ref, receipt.OldOID, receipt.NewOID, receipt.ComparisonOID
	return branch, nil
}

// PRCreateReceipt decodes one committed PR-creation receipt: the exact
// native PR identity plus the head/base snapshot behind it.
type PRCreateReceipt struct {
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	HeadOID      string `json:"head_oid"`
	BaseOID      string `json:"base_oid"`
	AuthorID     int64  `json:"author_id"`
	RepositoryID int64  `json:"repository_id"`
	PRID         int64  `json:"pr_id"`
	IssueID      int64  `json:"issue_id"`
	PRNumber     int64  `json:"pr_number"`
}

// DecodePRCreateReceipt verifies one committed creation outcome against
// its submitted snapshot and adopts the exact PR. Only this operation's
// receipt proves creation: an existing or similar PR is never adopted.
func DecodePRCreateReceipt(outcome factory.OperationOutcome, headRef, baseRef, headOID, baseOID string, actorID, repositoryID int64) (factory.PRCreationOutcome, error) {
	created := factory.PRCreationOutcome{Operation: outcome}
	if outcome.Effect != factory.OpEffectCommitted {
		return factory.PRCreationOutcome{}, errors.New("PR creation outcome is not committed")
	}
	if len(outcome.Receipt) == 0 || len(outcome.Receipt) > factory.MaxPublicationReceipt {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt is missing")
	}
	var receipt PRCreateReceipt
	decoder := json.NewDecoder(strings.NewReader(string(outcome.Receipt)))
	if err := decoder.Decode(&receipt); err != nil {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt is malformed")
	}
	if receipt.HeadRef != headRef || receipt.BaseRef != baseRef ||
		!strings.EqualFold(receipt.HeadOID, headOID) || !strings.EqualFold(receipt.BaseOID, baseOID) ||
		receipt.RepositoryID != repositoryID || receipt.AuthorID != actorID || actorID <= 0 {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt differs from its intent")
	}
	if receipt.PRID <= 0 || receipt.IssueID <= 0 || receipt.PRNumber <= 0 {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt lacks its identity")
	}
	created.HeadRef, created.BaseRef, created.HeadOID, created.BaseOID = receipt.HeadRef, receipt.BaseRef, receipt.HeadOID, receipt.BaseOID
	created.PRNumber, created.PRID, created.IssueID, created.AuthorID = receipt.PRNumber, receipt.PRID, receipt.IssueID, receipt.AuthorID
	return created, nil
}

// OperationNotAfter returns the submit deadline for one operation: ten
// minutes of host time from now, inside the native maximum.
func OperationNotAfter(now time.Time) int64 {
	return now.Add(operationLifetime).Unix()
}
