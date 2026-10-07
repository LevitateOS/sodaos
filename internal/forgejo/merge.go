package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// Merger submits only exact fast-forward-only merges through the shared
// native admission. Its distinct actor secret stays outside the Project
// and its agents. Soda verifies exact head/base/review/check evidence
// before submitting; native protection still authorizes the effect.
type Merger struct {
	background *ServiceBackground
	rest       *Client
	tokenFile  string
}

func NewMerger(background *ServiceBackground, rest *Client, tokenFile string) *Merger {
	return &Merger{background: background, rest: rest, tokenFile: tokenFile}
}

func (m *Merger) checkActor(ctx context.Context, actor int64) error {
	if m.background == nil || m.rest == nil || m.tokenFile == "" {
		return &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	token, err := config.Secret(m.tokenFile)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	user, err := m.rest.Current(ctx, token)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	if user.ID != actor {
		return &factory.PublicationWait{Reason: "authority_lost"}
	}
	return nil
}

// ObserveMerge validates the recorded PR identity, exact branch tips,
// the independent approval on the exact head, and the latest native
// check evidence on the exact head. Issue and pull snapshot locators
// are mutually exclusive in the native SDK, so their two complete
// brackets must resolve to the same native revision. Tip mismatches
// stay in the returned observation for the verdict; anything else that
// misidentifies the PR refuses without recording.
func (m *Merger) ObserveMerge(ctx context.Context, w factory.MergeWork) (factory.MergeObservation, error) {
	var empty factory.MergeObservation
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
	snapshot, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, PullNumber: number, SHA: w.HeadOID, Families: []SnapshotFamily{FamilyPull, FamilyReviews, FamilyChecks, FamilyRefs}, Refs: []string{w.HeadRef, w.BaseRef}})
	if err != nil {
		return empty, mergeReadError(err)
	}
	if snapshot.Revision != issue.Revision {
		return empty, &factory.PublicationWait{Reason: "revision_moved"}
	}
	return matchMergeTarget(w, issue, snapshot)
}

func mergeReadError(err error) error {
	if errors.Is(err, ErrNativeBusy) {
		return &factory.PublicationWait{Reason: "native_busy"}
	}
	if errors.Is(err, ErrStaleSnapshot) {
		return &factory.PublicationWait{Reason: "revision_moved"}
	}
	return &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
}

func mergeIntent(w factory.MergeWork) (extensions.OperationIntent, error) {
	if err := w.Validate(); err != nil {
		return extensions.OperationIntent{}, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	payload := extensions.MergePayload{PullRequestNumber: w.PRNumber, HeadRepositoryID: w.Repository, HeadRef: w.HeadRef, BaseRef: w.BaseRef, ExpectedHeadOID: w.HeadOID, ExpectedBaseOID: w.BaseOID, Method: extensions.MergeMethodFastForwardOnly}
	if err := extensions.ValidateMergePayload(w.Repository, payload); err != nil {
		return extensions.OperationIntent{}, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return extensions.OperationIntent{}, err
	}
	return extensions.OperationIntent{OperationID: w.OperationID, ActorID: strconv.FormatInt(w.ActorID, 10), RepositoryID: strconv.FormatInt(w.Repository, 10), Kind: factory.OpMerge, AuthorizationRevision: w.AuthRevision, ExpectedNativeRevision: w.NativeRev, NotAfter: w.NotAfter, Payload: raw}, nil
}

// SubmitMerge sends the already persisted intent. A failed response is
// reconciled by that operation identity; no ordinary merge POST or
// guessed record is used.
func (m *Merger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	intent, err := mergeIntent(w)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	if w.NotAfter <= time.Now().Unix() {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "observation_expired"}
	}
	if err = m.checkActor(ctx, w.ActorID); err != nil {
		return factory.OperationOutcome{}, err
	}
	record, err := m.background.SubmitOperation(ctx, extensions.CredentialFile(m.tokenFile), intent)
	if err != nil {
		observed, lookupErr := m.LookupOp(ctx, w.OperationID)
		if lookupErr == nil && !observed.NotObserved {
			return observed, nil
		}
		return factory.OperationOutcome{}, mergeCallError(err)
	}
	outcome, err := mergeOperationOutcome(w.OperationID, record)
	if err == nil && (outcome.ActorID != w.ActorID || outcome.RepositoryID != w.Repository) {
		err = errors.New("merge operation differs from its recorded scope")
	}
	return outcome, err
}

// LookupOp uses installation admission even after the merge credential
// is withdrawn. Absence does not prove an earlier submission cannot
// still arrive.
func (m *Merger) LookupOp(ctx context.Context, id string) (factory.OperationOutcome, error) {
	if m.background == nil {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	lookup, err := m.background.GetOperation(ctx, id)
	if err != nil {
		return factory.OperationOutcome{}, mergeCallError(err)
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	if lookup.Record == nil {
		return factory.OperationOutcome{}, errors.New("merge lookup lacks its native record")
	}
	return mergeOperationOutcome(id, *lookup.Record)
}

// CancelOp records cancellation under owning-installation admission. A
// committed merge remains historical evidence; cancellation cannot
// erase it.
func (m *Merger) CancelOp(ctx context.Context, id string) (factory.OperationOutcome, error) {
	if m.background == nil {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	record, err := m.background.CancelOperation(ctx, id)
	if err != nil {
		observed, lookupErr := m.LookupOp(ctx, id)
		if lookupErr == nil && !observed.NotObserved {
			return observed, nil
		}
		return factory.OperationOutcome{}, mergeCallError(err)
	}
	return mergeOperationOutcome(id, record)
}

func mergeCallError(err error) error {
	var status *forgejopublish.StatusError
	if !errors.As(err, &status) {
		return err
	}
	switch status.Status {
	case 400:
		return &factory.PublicationRefusal{Reason: "invalid_intent"}
	case 401:
		return &factory.PublicationWait{Reason: "credential_invalid"}
	case 403:
		return &factory.PublicationWait{Reason: "authority_lost"}
	case 409:
		if status.Body == "intent_conflict" {
			return &factory.PublicationRefusal{Reason: "intent_conflict"}
		}
	}
	return errors.New("native merge operation response unconfirmed")
}

func mergeOperationOutcome(id string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if record.OperationID != id || record.InstallationID == "" {
		return empty, errors.New("merge operation identity differs")
	}
	if record.Outcome == extensions.BackgroundOutcomeOperationsUnavailable {
		return empty, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	o := factory.OperationOutcome{OperationID: id, InstallationID: record.InstallationID, Kind: record.Kind, Effect: record.EffectState, Cancellation: record.CancellationStatus, Completion: record.CompletionState, Reason: record.ReasonCode}
	if o.Cancellation == "" {
		o.Cancellation = factory.OpCancelNone
	}
	if !factory.ValidOpEffect(o.Effect) || !factory.ValidOpCancellation(o.Cancellation) || (o.Completion != "" && !factory.ValidOpCompletion(o.Completion)) || len(o.Reason) > 64 || len(record.Receipt) > factory.MaxPublicationReceipt {
		return empty, errors.New("invalid native merge operation state")
	}
	tombstone := record.Kind == "" && record.ActorID == "" && record.RepositoryID == "" && o.Effect == factory.OpEffectNotCommitted && o.Cancellation == factory.OpCancelCancelled
	if !tombstone {
		actor, ae := strconv.ParseInt(record.ActorID, 10, 64)
		repo, re := strconv.ParseInt(record.RepositoryID, 10, 64)
		if record.Kind != factory.OpMerge || ae != nil || re != nil || actor <= 0 || repo <= 0 {
			return empty, errors.New("native merge operation scope differs")
		}
		o.ActorID, o.RepositoryID = actor, repo
	}
	o.Receipt = append([]byte(nil), record.Receipt...)
	return o, nil
}
