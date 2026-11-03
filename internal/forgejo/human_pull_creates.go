package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

// HumanPullCreates translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanPullCreates struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanPullCreateWriter = HumanPullCreates{}

// Resolve only existing base/source mappings and the live human. The native
// snapshot admits both repositories; cleanup does not introduce another source
// read requirement after a committed operation.
func (s HumanPullCreates) scope(ctx context.Context, q factory.HumanPullCreateCommand) (string, string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", "", factoryservice.ErrHumanPullCreateQuery
	}
	repository, err := s.Identities.NativeRepository(ctx, q.Repository)
	if err != nil {
		return "", "", "", err
	}
	head, err := s.Identities.NativeRepository(ctx, q.HeadRepository)
	if err != nil {
		return "", "", "", err
	}
	actor, err := s.Identities.NativeActor(ctx, q.Actor)
	if err != nil {
		return "", "", "", err
	}
	current, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return "", "", "", humanPullCreateError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", "", factoryservice.ErrHumanPullCreateAdmission
	}
	return repository, head, actor, nil
}

func (s HumanPullCreates) PrepareHumanPullCreate(ctx context.Context, q factory.HumanPullCreateCommand) (factory.HumanPullCreateCapture, error) {
	var empty factory.HumanPullCreateCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanPullCreateQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, head, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanPullCreateError(err)
	}
	operation, err := extensions.HumanPullCreateOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanPullCreateQuery
	}
	lookup, err := s.Authority.GetHumanPullCreate(ctx, repository, operation)
	if err != nil {
		return empty, humanPullCreateError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanPullCreateConflict
	}
	snapshot, err := s.Authority.RepositoryPullCreateSnapshot(ctx, repository, extensions.RepositoryPullCreateSnapshotRequest{HeadRepositoryID: head, BaseBranch: q.BaseBranch, HeadBranch: q.HeadBranch})
	if err != nil {
		return empty, humanPullCreateError(err)
	}
	if snapshot.Action.State != "allowed" {
		return empty, factoryservice.ErrHumanPullCreateAdmission
	}
	intent := extensions.HumanPullCreateRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Title: q.Title, Body: q.Body, AllowMaintainerEdit: q.AllowMaintainerEdit}
	if !extensions.ValidHumanPullCreateRequest(intent) {
		return empty, factoryservice.ErrHumanPullCreateUnavailable
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanPullCreateCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Decode the exact saved preparation without refreshing tips, revision or deadline.
// Native admission remains responsible for current session and repository access.
func (s HumanPullCreates) decode(ctx context.Context, c factory.HumanPullCreateCapture) (extensions.HumanPullCreateRequest, string, error) {
	var q extensions.HumanPullCreateRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanPullCreateQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanPullCreateInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanPullCreateQuery
	}
	repository, head, actor, err := s.scope(ctx, c.Request)
	if err != nil {
		return q, "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanPullCreateRequest(q) {
		return q, "", factoryservice.ErrHumanPullCreateQuery
	}
	requested := c.Request
	requested.Title, requested.Body = q.Title, q.Body
	requested.BaseBranch, requested.HeadBranch, requested.AllowMaintainerEdit = q.Expected.BaseBranch, q.Expected.HeadBranch, q.AllowMaintainerEdit
	operation, err := extensions.HumanPullCreateOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || !requested.SameRequest(c.Request) ||
		q.Expected.RepositoryID != repository || q.Expected.HeadRepositoryID != head || q.Expected.ActorID != actor {
		return q, "", factoryservice.ErrHumanPullCreateQuery
	}
	_, digest, err := extensions.CanonicalHumanPullCreateIntent(q)
	return q, digest, err
}

func (s HumanPullCreates) SubmitHumanPullCreate(ctx context.Context, c factory.HumanPullCreateCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanPullCreate(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanPullCreateError(err)
	}
	return humanPullCreateRecord(c, q, digest, r)
}

func (s HumanPullCreates) LookupHumanPullCreate(ctx context.Context, c factory.HumanPullCreateCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanPullCreate(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanPullCreateError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanPullCreateNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanPullCreateRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanPullCreateUnavailable
	}
	return lookup, factory.ValidateHumanPullCreateObservation(c, lookup)
}

func (s HumanPullCreates) CancelHumanPullCreate(ctx context.Context, c factory.HumanPullCreateCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanPullCreate(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanPullCreateError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanPullCreateNamespace
	}
	if lookup.Record != nil {
		if _, err := humanPullCreateRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanPullCreate(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanPullCreateError(err)
	}
	return humanPullCreateRecord(c, q, digest, r)
}

func humanPullCreateRecord(c factory.HumanPullCreateCapture, q extensions.HumanPullCreateRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanPullCreateNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// Native completion describes post-primary notifications only. The neutral
	// contract describes the whole operation: proved no-effect has no completion
	// work; an unresolved primary remains pending, never complete.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled || r.CompletionState != "" {
			return empty, factoryservice.ErrHumanPullCreateUnavailable
		}
		return record, factory.ValidateHumanPullCreateObservation(c, collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupFound, Record: &record})
	}
	if r.Kind != extensions.OperationKindHumanPullCreate || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanPullCreateConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.OpenPull, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanPullCreateReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanPullCreateReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanPullCreateConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanPullCreateUnavailable
	}
	return record, factory.ValidateHumanPullCreateObservation(c, collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupFound, Record: &record})
}

func humanPullCreateError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanPullCreateQuery), errors.Is(err, extensions.ErrPullCreateSnapshotQuery):
		return factoryservice.ErrHumanPullCreateQuery
	case errors.Is(err, extensions.ErrHumanPullCreateAdmission), errors.Is(err, extensions.ErrPullCreateSnapshotAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanPullCreateAdmission
	case errors.Is(err, extensions.ErrHumanPullCreateConflict), errors.Is(err, extensions.ErrPullCreateSnapshotChanged):
		return factoryservice.ErrHumanPullCreateConflict
	case errors.Is(err, extensions.ErrHumanPullCreateUnavailable), errors.Is(err, extensions.ErrPullCreateSnapshotUnavailable):
		return factoryservice.ErrHumanPullCreateUnavailable
	default:
		return err
	}
}
