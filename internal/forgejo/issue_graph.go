package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
)

// IssueGraphWriter uses the shared service admission and a separate enrolled
// native graph actor. Its secret stays outside Projects and agent processes.
type IssueGraphWriter struct {
	background sdk.BackgroundClient
	rest       *Client
	tokenFile  string
}

func NewIssueGraphWriter(background sdk.BackgroundClient, rest *Client, tokenFile string) *IssueGraphWriter {
	return &IssueGraphWriter{background: background, rest: rest, tokenFile: tokenFile}
}
func (w *IssueGraphWriter) checkActor(ctx context.Context, actor int64) error {
	if w == nil || w.background == nil || w.rest == nil || w.tokenFile == "" || actor <= 0 {
		return ErrUnavailable
	}
	token, err := config.Secret(w.tokenFile)
	if err != nil {
		return ErrUnavailable
	}
	current, err := loadActor(ctx, w.rest, token)
	if err != nil {
		return err
	}
	if current.ID != actor {
		return ErrInvalidResponse
	}
	return nil
}

// CheckIntentOwner proves current native code-write authority at one idle
// revision. It grants no execution or spending permission.
func (w *IssueGraphWriter) CheckIntentOwner(ctx context.Context, repository, owner, actor int64) (int64, error) {
	if repository <= 0 || owner <= 0 {
		return 0, ErrInvalidResponse
	}
	if err := w.checkActor(ctx, actor); err != nil {
		return 0, err
	}
	before, err := w.background.ReadNativeRevision(ctx)
	if err != nil {
		return 0, err
	}
	if !before.Idle {
		return 0, ErrNativeBusy
	}
	id := strconv.FormatInt(owner, 10)
	answer, err := w.background.ReadSnapshot(ctx, sdk.CredentialFile(w.tokenFile), sdk.SnapshotRequest{RepositoryID: strconv.FormatInt(repository, 10), ActorID: strconv.FormatInt(actor, 10), Families: []string{sdk.SnapshotFamilyRepositoryWriters}, WriterIDs: []string{id}})
	if err != nil {
		return 0, err
	}
	after, err := w.background.ReadNativeRevision(ctx)
	if err != nil {
		return 0, err
	}
	if !after.Idle {
		return 0, ErrNativeBusy
	}
	if after.Revision != before.Revision {
		return 0, ErrStaleSnapshot
	}
	allowed, exists := answer.Writers[id]
	if answer.RepositoryID != strconv.FormatInt(repository, 10) || len(answer.Writers) != 1 || !exists {
		return 0, ErrInvalidResponse
	}
	if !allowed {
		return 0, errors.New("accepted intent owner no longer has native code-write permission")
	}
	return after.Revision, nil
}

func graphOperationIntent(g factory.IntentGeneration, repository int64) (sdk.OperationIntent, error) {
	var p sdk.IssueGraphPayload
	d := json.NewDecoder(bytes.NewReader(g.Payload))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || sdk.ValidateIssueGraphPayload(p) != nil {
		return sdk.OperationIntent{}, ErrInvalidResponse
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return sdk.OperationIntent{}, ErrInvalidResponse
	}
	if g.ActorID <= 0 || repository <= 0 || g.OperationID == "" || g.NativeRev < 1 || g.NotAfter <= 0 || g.AuthRevision == "" {
		return sdk.OperationIntent{}, ErrInvalidResponse
	}
	return sdk.OperationIntent{OperationID: g.OperationID, ActorID: strconv.FormatInt(g.ActorID, 10), RepositoryID: strconv.FormatInt(repository, 10), Kind: sdk.OperationKindIssueGraph, AuthorizationRevision: g.AuthRevision, ExpectedNativeRevision: g.NativeRev, NotAfter: g.NotAfter, Payload: g.Payload}, nil
}

// SubmitGraph sends only the exact durable operation; an uncertain transport
// outcome is looked up under that identity, never recreated under a new one.
func (w *IssueGraphWriter) SubmitGraph(ctx context.Context, g factory.IntentGeneration, repository int64) (sdk.OperationRecord, error) {
	intent, err := graphOperationIntent(g, repository)
	if err != nil {
		return sdk.OperationRecord{}, err
	}
	if err = w.checkActor(ctx, g.ActorID); err != nil {
		return sdk.OperationRecord{}, err
	}
	record, err := w.background.SubmitOperation(ctx, sdk.CredentialFile(w.tokenFile), intent)
	if err != nil {
		lookup, readErr := w.background.GetOperation(ctx, g.OperationID)
		if readErr != nil || lookup.Record == nil {
			return sdk.OperationRecord{}, err
		}
		record = *lookup.Record
	}
	if err = validateGraphRecord(g, repository, record); err != nil {
		return sdk.OperationRecord{}, err
	}
	return record, nil
}
func (w *IssueGraphWriter) LookupGraph(ctx context.Context, g factory.IntentGeneration, repository int64) (sdk.OperationLookup, error) {
	if w == nil || w.background == nil {
		return sdk.OperationLookup{}, ErrUnavailable
	}
	lookup, err := w.background.GetOperation(ctx, g.OperationID)
	if err != nil {
		return sdk.OperationLookup{}, err
	}
	if lookup.OperationID != g.OperationID {
		return sdk.OperationLookup{}, ErrInvalidResponse
	}
	if lookup.Record != nil {
		if err := validateGraphRecord(g, repository, *lookup.Record); err != nil {
			return sdk.OperationLookup{}, err
		}
	}
	return lookup, nil
}
func (w *IssueGraphWriter) CancelGraph(ctx context.Context, operationID string) (sdk.OperationRecord, error) {
	if w == nil || w.background == nil {
		return sdk.OperationRecord{}, ErrUnavailable
	}
	return w.background.CancelOperation(ctx, operationID)
}
func validateGraphRecord(g factory.IntentGeneration, repository int64, r sdk.OperationRecord) error {
	if r.OperationID != g.OperationID {
		return ErrInvalidResponse
	}
	// A pre-submit tombstone has no invented actor or operation kind.
	if r.Kind == "" && r.EffectState == sdk.EffectNotCommitted && r.CancellationStatus == sdk.CancellationCancelled {
		return nil
	}
	if r.Kind != sdk.OperationKindIssueGraph || r.ActorID != strconv.FormatInt(g.ActorID, 10) || r.RepositoryID != strconv.FormatInt(repository, 10) {
		return ErrInvalidResponse
	}
	return nil
}

func (w *IssueGraphWriter) ReadGeneratedIssue(ctx context.Context, repository, index, actor int64) (sdk.NativeSnapshot, int64, error) {
	if err := w.checkActor(ctx, actor); err != nil {
		return sdk.NativeSnapshot{}, 0, err
	}
	reader := &BackgroundSnapshotReader{Client: w.background, ActorID: strconv.FormatInt(actor, 10)}
	snapshot, err := BracketedRead(ctx, reader, sdk.CredentialFile(w.tokenFile), SnapshotRequest{RepositoryID: strconv.FormatInt(repository, 10), IssueIndex: strconv.FormatInt(index, 10), Families: []SnapshotFamily{FamilyIssue, FamilyDependencies}})
	if err != nil {
		return sdk.NativeSnapshot{}, 0, err
	}
	return sdk.NativeSnapshot{RepositoryID: snapshot.RepositoryID, Issue: snapshot.Issue, Dependencies: snapshot.Dependencies}, snapshot.Revision, nil
}
