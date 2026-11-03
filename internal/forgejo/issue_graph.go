package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/sodaos/internal/config"
)

// IssueGraphWriter uses the shared service admission and a separate enrolled
// native graph actor. Its secret stays outside Projects and agent processes.
type IssueGraphWriter struct {
	GraphCodec
	Identities *FountainIdentities
	background sdk.BackgroundClient
	rest       *Client
	tokenFile  string
}

func NewIssueGraphWriter(background sdk.BackgroundClient, rest *Client, tokenFile string, identities *FountainIdentities) *IssueGraphWriter {
	return &IssueGraphWriter{background: background, rest: rest, tokenFile: tokenFile, Identities: identities}
}

func (w *IssueGraphWriter) checkActor(ctx context.Context, actor int64) (int64, error) {
	if w == nil || w.background == nil || w.rest == nil || w.tokenFile == "" || actor <= 0 {
		return 0, ErrUnavailable
	}
	native, err := w.Identities.NativeActor(ctx, collaboration.ActorID(actor))
	if err != nil {
		return 0, err
	}
	nativeActor, err := originalCreationID(native)
	if err != nil {
		return 0, err
	}
	token, err := config.Secret(w.tokenFile)
	if err != nil {
		return 0, ErrUnavailable
	}
	current, err := loadActor(ctx, w.rest, token)
	if err != nil {
		return 0, err
	}
	if current.ID != nativeActor {
		return 0, ErrInvalidResponse
	}
	return nativeActor, nil
}

// CheckIntentOwner proves current native code-write authority at one idle
// revision. It grants no execution or spending permission.
func (w *IssueGraphWriter) CheckIntentOwner(ctx context.Context, repository, owner, actor int64) (int64, error) {
	if w == nil || repository <= 0 || owner <= 0 {
		return 0, ErrInvalidResponse
	}
	nativeRepository, err := w.Identities.NativeRepository(ctx, collaboration.RepositoryID(repository))
	if err != nil {
		return 0, err
	}
	id, err := w.Identities.NativeActor(ctx, collaboration.ActorID(owner))
	if err != nil {
		return 0, err
	}
	nativeActor, err := w.checkActor(ctx, actor)
	if err != nil {
		return 0, err
	}
	before, err := w.background.ReadNativeRevision(ctx)
	if err != nil {
		return 0, err
	}
	if !before.Idle {
		return 0, ErrNativeBusy
	}
	answer, err := w.background.ReadSnapshot(ctx, sdk.CredentialFile(w.tokenFile), sdk.SnapshotRequest{RepositoryID: nativeRepository, ActorID: strconv.FormatInt(nativeActor, 10), Families: []string{sdk.SnapshotFamilyRepositoryWriters}, WriterIDs: []string{id}})
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
	if answer.RepositoryID != nativeRepository || len(answer.Writers) != 1 || !exists {
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
func (w *IssueGraphWriter) SubmitGraph(ctx context.Context, g factory.IntentGeneration, repository int64) (factory.GraphOperationOutcome, error) {
	intent, err := graphOperationIntent(g, repository)
	if err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	nativeRepository, nativeActor, err := w.nativeScope(ctx, repository, g.ActorID)
	if err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	intent.RepositoryID, intent.ActorID = nativeRepository, nativeActor
	if _, err = w.checkActor(ctx, g.ActorID); err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	record, err := w.background.SubmitOperation(ctx, sdk.CredentialFile(w.tokenFile), intent)
	if err != nil {
		lookup, readErr := w.background.GetOperation(ctx, g.OperationID)
		if readErr != nil || lookup.Record == nil {
			return factory.GraphOperationOutcome{}, err
		}
		record = *lookup.Record
	}
	if err = validateGraphRecord(g, nativeRepository, nativeActor, record); err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	return GraphOutcomeFromNative(record), nil
}

func (w *IssueGraphWriter) LookupGraph(ctx context.Context, g factory.IntentGeneration, repository int64) (factory.GraphOperationLookup, error) {
	if w == nil || w.background == nil {
		return factory.GraphOperationLookup{}, ErrUnavailable
	}
	nativeRepository, nativeActor, err := w.nativeScope(ctx, repository, g.ActorID)
	if err != nil {
		return factory.GraphOperationLookup{}, err
	}
	lookup, err := w.background.GetOperation(ctx, g.OperationID)
	if err != nil {
		return factory.GraphOperationLookup{}, err
	}
	if lookup.OperationID != g.OperationID {
		return factory.GraphOperationLookup{}, ErrInvalidResponse
	}
	if lookup.Record != nil {
		if err := validateGraphRecord(g, nativeRepository, nativeActor, *lookup.Record); err != nil {
			return factory.GraphOperationLookup{}, err
		}
	}
	return GraphLookupFromNative(lookup), nil
}

func (w *IssueGraphWriter) CancelGraph(ctx context.Context, operationID string) (factory.GraphOperationOutcome, error) {
	if w == nil || w.background == nil || w.Identities == nil {
		return factory.GraphOperationOutcome{}, ErrUnavailable
	}
	record, err := w.background.CancelOperation(ctx, operationID)
	if err != nil {
		return factory.GraphOperationOutcome{}, err
	}
	if record.OperationID != operationID {
		return factory.GraphOperationOutcome{}, ErrInvalidResponse
	}
	return GraphOutcomeFromNative(record), nil
}

func validateGraphRecord(g factory.IntentGeneration, repository, actor string, r sdk.OperationRecord) error {
	if r.OperationID != g.OperationID {
		return ErrInvalidResponse
	}
	// A pre-submit tombstone has no invented actor or operation kind.
	if r.Kind == "" && r.EffectState == sdk.EffectNotCommitted && r.CancellationStatus == sdk.CancellationCancelled {
		return nil
	}
	if r.Kind != sdk.OperationKindIssueGraph || r.ActorID != actor || r.RepositoryID != repository {
		return ErrInvalidResponse
	}
	return nil
}

func (w *IssueGraphWriter) ReadGeneratedIssue(ctx context.Context, repository, index, actor int64) (factory.GeneratedIssueObservation, int64, error) {
	nativeRepository, nativeActor, err := w.nativeScope(ctx, repository, actor)
	if err != nil {
		return factory.GeneratedIssueObservation{}, 0, err
	}
	if _, err := w.checkActor(ctx, actor); err != nil {
		return factory.GeneratedIssueObservation{}, 0, err
	}
	reader := &BackgroundSnapshotReader{Client: w.background, ActorID: nativeActor}
	snapshot, err := BracketedRead(ctx, reader, sdk.CredentialFile(w.tokenFile), SnapshotRequest{RepositoryID: nativeRepository, IssueIndex: strconv.FormatInt(index, 10), Families: []SnapshotFamily{FamilyIssue, FamilyDependencies}})
	if err != nil {
		return factory.GeneratedIssueObservation{}, 0, err
	}
	observation, err := GeneratedIssueObservationFromNative(sdk.NativeSnapshot{RepositoryID: snapshot.RepositoryID, Issue: snapshot.Issue, Dependencies: snapshot.Dependencies})
	if err == nil && observation.Issue != nil && observation.Issue.Creator != nil {
		creator, mapErr := w.Identities.LocalActor(ctx, strconv.FormatInt(int64(*observation.Issue.Creator), 10))
		if mapErr != nil {
			return factory.GeneratedIssueObservation{}, 0, mapErr
		}
		observation.Issue.Creator = &creator
	}
	return observation, snapshot.Revision, err
}

func (w *IssueGraphWriter) nativeScope(ctx context.Context, repository, actor int64) (string, string, error) {
	if w == nil {
		return "", "", ErrUnavailable
	}
	nativeRepository, err := w.Identities.NativeRepository(ctx, collaboration.RepositoryID(repository))
	if err != nil {
		return "", "", err
	}
	nativeActor, err := w.Identities.NativeActor(ctx, collaboration.ActorID(actor))
	return nativeRepository, nativeActor, err
}

// DecodeGraphReceipt retains the provider evidence while projecting its actor
// and repository into existing local mappings. A receipt cannot allocate IDs.
func (w *IssueGraphWriter) DecodeGraphReceipt(ctx context.Context, raw json.RawMessage) (factory.GraphReceiptFacts, error) {
	if w == nil || w.Identities == nil || w.Identities.registry == nil {
		return factory.GraphReceiptFacts{}, ErrUnavailable
	}
	facts, err := w.GraphCodec.DecodeGraphReceipt(ctx, raw)
	if err != nil {
		return factory.GraphReceiptFacts{}, err
	}
	repository, err := w.Identities.registry.CollaborationRepositoryByExternalID(ctx, w.Identities.host, strconv.FormatInt(facts.Repository, 10))
	if err != nil {
		return factory.GraphReceiptFacts{}, err
	}
	actor, err := w.Identities.registry.CollaborationActorByExternalID(ctx, w.Identities.host, strconv.FormatInt(facts.Actor, 10))
	if err != nil {
		return factory.GraphReceiptFacts{}, err
	}
	facts.Repository, facts.Actor = int64(repository.ID), int64(actor.ID)
	return facts, nil
}
