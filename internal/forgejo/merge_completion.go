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

func (m *Merger) AdoptMerge(w factory.MergeWork, o factory.OperationOutcome) (factory.MergeOutcome, error) {
	var empty factory.MergeOutcome
	if err := w.Validate(); err != nil {
		return empty, err
	}
	if o.NotObserved || o.OperationID != w.OperationID || o.InstallationID == "" || o.Kind != factory.OpMerge || o.ActorID != w.ActorID || o.RepositoryID != w.Repository || o.Effect != factory.OpEffectCommitted || len(o.Receipt) == 0 || len(o.Receipt) > factory.MaxPublicationReceipt {
		return empty, errors.New("merge outcome differs from its recorded intent")
	}
	var receipt mergeReceipt
	decoder := json.NewDecoder(bytes.NewReader(o.Receipt))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return empty, errors.New("native merge receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return empty, errors.New("native merge receipt is malformed")
	}
	if receipt.ActorID != w.ActorID || receipt.RepositoryID != w.Repository || receipt.PRNumber != w.PRNumber || receipt.PRID != w.PRID ||
		receipt.HeadRef != w.HeadRef || receipt.BaseRef != w.BaseRef || receipt.OldOID != w.BaseOID || receipt.NewOID != w.HeadOID || receipt.Method != factory.MergeFastForward {
		return empty, errors.New("native merge receipt differs from its recorded intent")
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
	reader := &BackgroundSnapshotReader{Client: &backgroundClientAdapter{background: m.background}, ActorID: strconv.FormatInt(w.ActorID, 10)}
	credential := extensions.CredentialFile(m.tokenFile)
	repo, number := strconv.FormatInt(w.Repository, 10), strconv.FormatInt(w.PRNumber, 10)
	issue, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, IssueIndex: number, Families: []SnapshotFamily{FamilyIssue}})
	if err != nil {
		return empty, mergeReadError(err)
	}
	snapshot, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, PullNumber: number, Families: []SnapshotFamily{FamilyPull, FamilyRefs}, Refs: []string{w.HeadRef, w.BaseRef}})
	if err != nil {
		return empty, mergeReadError(err)
	}
	if snapshot.Revision != issue.Revision {
		return empty, &factory.PublicationWait{Reason: "revision_moved"}
	}
	return matchMergeConfirmation(w, issue, snapshot)
}

func matchMergeConfirmation(w factory.MergeWork, issue, snapshot NativeSnapshot) (factory.MergeConfirmation, error) {
	var empty factory.MergeConfirmation
	i, p := issue.Issue, snapshot.Pull
	if i == nil || p == nil || !i.Visible || !p.Visible || !i.IsPull ||
		i.ID != strconv.FormatInt(w.IssueID, 10) || i.Provenance.PosterID != strconv.FormatInt(w.PRAuthorID, 10) ||
		p.ID != strconv.FormatInt(w.PRID, 10) || p.IssueID != i.ID || p.HeadRepoID != strconv.FormatInt(w.Repository, 10) ||
		p.HeadBranch != factory.RefHead(w.HeadRef) || p.BaseBranch != factory.RefHead(w.BaseRef) {
		return empty, &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	if !p.HasMerged || p.MergedCommit != w.HeadOID {
		return empty, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	merger, err := strconv.ParseInt(p.MergerID, 10, 64)
	if err != nil || merger != w.ActorID || p.MergedUnix <= 0 {
		return empty, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	var baseTip string
	for _, ref := range snapshot.Refs {
		if !ref.Visible || !ref.Exists {
			return empty, &factory.PublicationRefusal{Reason: "pr_mismatch"}
		}
		if ref.Ref == w.BaseRef {
			baseTip = ref.OID
		}
	}
	if baseTip != w.HeadOID {
		return empty, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	if !i.IsClosed || i.ClosedUnix <= 0 {
		return empty, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	return factory.MergeConfirmation{MergedCommit: w.HeadOID, BaseTip: baseTip, MergerID: merger, MergedUnix: p.MergedUnix, ClosedUnix: i.ClosedUnix, NativeRev: snapshot.Revision, ObservedUnix: time.Now().Unix(), IssueClosed: true}, nil
}
