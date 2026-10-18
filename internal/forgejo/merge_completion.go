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

	"github.com/levitateos/sodaos/internal/factory"
)

// mergeReceipt is the wire shape observed from the staged native FT14
// producer by TestNativeMergeWire. The realized commit is bound by the
// immutable submitted payload; it is not an invented receipt field.
type mergeReceipt struct {
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	OldOID       string `json:"old_oid"`
	NewOID       string `json:"new_oid"`
	Method       string `json:"method"`
	ActorID      int64  `json:"actor_id"`
	RepositoryID int64  `json:"repository_id"`
	PRNumber     int64  `json:"pr_number"`
	PRID         int64  `json:"pr_id"`
}

func checkMergeEnvelope(w factory.MergeWork, o factory.OperationOutcome) error {
	if err := w.Validate(); err != nil {
		return err
	}
	if o.NotObserved || o.OperationID != w.OperationID || o.InstallationID == "" || o.Kind != factory.OpMerge || o.ActorID != w.ActorID || o.RepositoryID != w.Repository || o.Effect != factory.OpEffectCommitted || len(o.Receipt) == 0 || len(o.Receipt) > factory.MaxPublicationReceipt {
		return errors.New("merge outcome differs from its recorded intent")
	}
	return nil
}

func decodeMergeReceipt(o factory.OperationOutcome) (mergeReceipt, error) {
	var receipt mergeReceipt
	decoder := json.NewDecoder(bytes.NewReader(o.Receipt))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return receipt, errors.New("native merge receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return receipt, errors.New("native merge receipt is malformed")
	}
	return receipt, nil
}

func checkMergeReceiptIdentity(receipt mergeReceipt, w factory.MergeWork) error {
	if receipt.ActorID != w.ActorID || receipt.RepositoryID != w.Repository || receipt.PRNumber != w.PRNumber || receipt.PRID != w.PRID ||
		receipt.HeadRef != w.HeadRef || receipt.BaseRef != w.BaseRef || receipt.OldOID != w.BaseOID || receipt.NewOID != w.HeadOID || receipt.Method != factory.MergeFastForward {
		return errors.New("native merge receipt differs from its recorded intent")
	}
	return nil
}

func (m *Merger) AdoptMerge(w factory.MergeWork, o factory.OperationOutcome) (factory.MergeOutcome, error) {
	var empty factory.MergeOutcome
	if err := checkMergeEnvelope(w, o); err != nil {
		return empty, err
	}
	receipt, err := decodeMergeReceipt(o)
	if err != nil {
		return empty, err
	}
	if err := checkMergeReceiptIdentity(receipt, w); err != nil {
		return empty, err
	}
	return factory.MergeOutcome{Operation: o, HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID, MergedCommit: w.HeadOID, PRNumber: w.PRNumber, PRID: w.PRID, IssueID: w.IssueID, ActorID: w.ActorID}, nil
}

// ObserveCompletion confirms the native bookkeeping behind a committed,
// completed merge: the PR merged to the exact head by the recorded
// actor, the base tip carries it, and the issue closed. Anything else
// refuses: a committed ref without this confirmation never finishes.
func (m *Merger) ObserveCompletion(ctx context.Context, w factory.MergeWork) (factory.MergeConfirmation, error) {
	var empty factory.MergeConfirmation
	if err := w.ValidateTarget(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	if err := m.checkActor(ctx, w.ActorID); err != nil {
		return empty, err
	}
	reader := &BackgroundSnapshotReader{Client: m.background, ActorID: strconv.FormatInt(w.ActorID, 10)}
	credential := extensions.CredentialFile(m.tokenFile)
	repo, number := strconv.FormatInt(w.Repository, 10), strconv.FormatInt(w.PRNumber, 10)
	issue, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, IssueIndex: number, Families: []SnapshotFamily{FamilyIssue}})
	if err != nil {
		return empty, mergeReadError(err)
	}
	originalNumber := strconv.FormatInt(w.Issue, 10)
	original, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, IssueIndex: originalNumber, Families: []SnapshotFamily{FamilyIssue}})
	if err != nil {
		return empty, mergeReadError(err)
	}
	snapshot, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, PullNumber: number, Families: []SnapshotFamily{FamilyPull, FamilyRefs}, Refs: []string{w.HeadRef, w.BaseRef}})
	if err != nil {
		return empty, mergeReadError(err)
	}
	if snapshot.Revision != issue.Revision || original.Revision != issue.Revision {
		return empty, &factory.PublicationWait{Reason: "revision_moved"}
	}
	baseTip, err := mergeCompletionBaseTip(w, snapshot)
	if err != nil {
		return empty, err
	}
	if baseTip != w.HeadOID {
		proof, err := BracketedRead(ctx, reader, credential, SnapshotRequest{
			RepositoryID: repo,
			Families:     []SnapshotFamily{FamilyRefs, FamilyAncestry},
			Refs:         []string{w.BaseRef},
			Ancestry:     &extensions.SnapshotAncestryRequest{AncestorOID: w.HeadOID, DescendantOID: baseTip},
		})
		if err != nil {
			return empty, mergeReadError(err)
		}
		if proof.Revision != issue.Revision || proof.Revision != snapshot.Revision {
			return empty, &factory.PublicationWait{Reason: "revision_moved"}
		}
		proofBaseTip, err := mergeCompletionBaseTip(w, proof)
		if err != nil || proofBaseTip != baseTip {
			return empty, &factory.PublicationWait{Reason: "revision_moved"}
		}
		snapshot.Ancestry = proof.Ancestry
	}
	return matchMergeConfirmation(w, issue, original, snapshot)
}

func mergeCompletionBaseTip(w factory.MergeWork, snapshot NativeSnapshot) (string, error) {
	var baseTip string
	for _, ref := range snapshot.Refs {
		if !ref.Visible || !ref.Exists {
			return "", &factory.PublicationRefusal{Reason: "pr_mismatch"}
		}
		if ref.Ref == w.BaseRef {
			baseTip = ref.OID
		}
	}
	if baseTip == "" {
		return "", &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	return baseTip, nil
}

func checkConfirmationIdentity(w factory.MergeWork, i *IssueEvidence, p *PullEvidence) error {
	if i == nil || p == nil || !i.Visible || !p.Visible || !i.IsPull ||
		i.ID != strconv.FormatInt(w.IssueID, 10) || i.Provenance.PosterID != strconv.FormatInt(w.PRAuthorID, 10) ||
		p.ID != strconv.FormatInt(w.PRID, 10) || p.IssueID != i.ID || p.HeadRepoID != strconv.FormatInt(w.Repository, 10) ||
		p.HeadBranch != factory.RefHead(w.HeadRef) || p.BaseBranch != factory.RefHead(w.BaseRef) {
		return &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	return nil
}

func checkConfirmationMerged(w factory.MergeWork, p *PullEvidence) (int64, error) {
	if !p.HasMerged || p.MergedCommit != w.HeadOID {
		return 0, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	merger, err := strconv.ParseInt(p.MergerID, 10, 64)
	if err != nil || merger != w.ActorID || p.MergedUnix <= 0 {
		return 0, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	return merger, nil
}

func confirmBaseContainsCommit(w factory.MergeWork, snapshot NativeSnapshot) (string, error) {
	baseTip, err := mergeCompletionBaseTip(w, snapshot)
	if err != nil {
		return "", err
	}
	if baseTip == w.HeadOID {
		return baseTip, nil
	}
	proof := snapshot.Ancestry
	if proof == nil || proof.AncestorOID != w.HeadOID || proof.DescendantOID != baseTip || !proof.Reachable {
		return "", &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	return baseTip, nil
}

func matchMergeConfirmation(w factory.MergeWork, issue, original, snapshot NativeSnapshot) (factory.MergeConfirmation, error) {
	var empty factory.MergeConfirmation
	i, p := issue.Issue, snapshot.Pull
	if err := checkConfirmationIdentity(w, i, p); err != nil {
		return empty, err
	}
	if original.Issue == nil || !original.Issue.Visible || original.Issue.IsPull ||
		original.Issue.Index != strconv.FormatInt(w.Issue, 10) {
		return empty, &factory.PublicationRefusal{Reason: "issue_mismatch"}
	}
	merger, err := checkConfirmationMerged(w, p)
	if err != nil {
		return empty, err
	}
	baseTip, err := confirmBaseContainsCommit(w, snapshot)
	if err != nil {
		return empty, err
	}
	if !i.IsClosed || i.ClosedUnix <= 0 {
		return empty, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	closingOccurrence, originalClosedUnix, closed := originalIssueClosure(original.Issue)
	if !closed {
		return empty, &factory.PublicationWait{Reason: "original_issue_closure_pending"}
	}
	return factory.MergeConfirmation{
		MergedCommit: w.HeadOID, BaseTip: baseTip, BaseContainsMergedCommit: true,
		MergerID: merger, MergedUnix: p.MergedUnix, ClosedUnix: i.ClosedUnix,
		OriginalClosed: true, OriginalClosingOccurrence: closingOccurrence, OriginalClosedUnix: originalClosedUnix,
		NativeRev: snapshot.Revision, ObservedUnix: time.Now().Unix(), IssueClosed: true,
	}, nil
}

func originalIssueClosure(issue *IssueEvidence) (string, int64, bool) {
	if issue == nil || !issue.Visible || issue.IsPull || !issue.IsClosed || issue.ClosedUnix <= 0 || !issue.Complete {
		return "", 0, false
	}
	lastLifecycle := ""
	closingOccurrence := ""
	for _, event := range issue.Lifecycle {
		switch event.Kind {
		case "closed", "reopened":
			if !factory.ValidNativeOccurrence(event.OccurrenceID) || event.AtUnix <= 0 {
				return "", 0, false
			}
			lastLifecycle = event.Kind
			if event.Kind == "closed" {
				closingOccurrence = event.OccurrenceID
			} else {
				closingOccurrence = ""
			}
		default:
			continue
		}
	}
	if lastLifecycle != "closed" || !factory.ValidNativeOccurrence(closingOccurrence) {
		return "", 0, false
	}
	return closingOccurrence, issue.ClosedUnix, true
}
