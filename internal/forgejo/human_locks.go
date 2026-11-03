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
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

// HumanLocks translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanLocks struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanLockWriter = HumanLocks{}

// Resolve existing canonical mappings and the live human. The lock snapshot
// owns subject-unit and writer permission; an unrelated code-unit read must not
// gate locking or recovery of a saved command after repository permission loss.
func (s HumanLocks) scope(ctx context.Context, q factory.HumanLockCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanLockQuery
	}
	repository, err := s.Identities.NativeRepository(ctx, q.Repository)
	if err != nil {
		return "", "", err
	}
	actor, err := s.Identities.NativeActor(ctx, q.Actor)
	if err != nil {
		return "", "", err
	}
	current, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return "", "", humanLockError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanLockAdmission
	}
	return repository, actor, nil
}

func (s HumanLocks) PrepareHumanLock(ctx context.Context, q factory.HumanLockCommand) (factory.HumanLockCapture, error) {
	var empty factory.HumanLockCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanLockQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanLockError(err)
	}
	operation, err := extensions.HumanLockOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanLockQuery
	}
	lookup, err := s.Authority.GetHumanLock(ctx, repository, operation)
	if err != nil {
		return empty, humanLockError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanLockConflict
	}
	snapshot, err := s.Authority.RepositoryLockSnapshot(ctx, repository, extensions.RepositoryLockSnapshotRequest{Subject: q.Subject, Index: strconv.FormatInt(q.Number, 10)})
	if err != nil {
		return empty, humanLockError(err)
	}
	if q.Lock && snapshot.LockAction.State != "allowed" || !q.Lock && snapshot.UnlockAction.State != "allowed" {
		return empty, factoryservice.ErrHumanLockAdmission
	}
	intent := extensions.HumanLockRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Lock: q.Lock, Reason: q.Reason}
	if !extensions.ValidHumanLockRequest(intent) {
		return empty, factoryservice.ErrHumanLockUnavailable
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanLockCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/lock permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanLocks) decode(ctx context.Context, c factory.HumanLockCapture) (extensions.HumanLockRequest, string, error) {
	var q extensions.HumanLockRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanLockQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanLockInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanLockQuery
	}
	repository, err := s.Identities.NativeRepository(ctx, c.Request.Repository)
	if err != nil {
		return q, "", err
	}
	actor, err := s.Identities.NativeActor(ctx, c.Request.Actor)
	if err != nil {
		return q, "", err
	}
	current, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return q, "", humanLockError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanLockAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanLockRequest(q) {
		return q, "", factoryservice.ErrHumanLockQuery
	}
	requested := c.Request
	requested.Lock, requested.Reason = q.Lock, q.Reason
	operation, err := extensions.HumanLockOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || !requested.SameRequest(c.Request) ||
		q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Subject != c.Request.Subject || q.Expected.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanLockQuery
	}
	_, digest, err := extensions.CanonicalHumanLockIntent(q)
	return q, digest, err
}

func (s HumanLocks) SubmitHumanLock(ctx context.Context, c factory.HumanLockCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanLock(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanLockError(err)
	}
	return humanLockRecord(c, q, digest, r)
}

func (s HumanLocks) LookupHumanLock(ctx context.Context, c factory.HumanLockCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanLock(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanLockError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanLockNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanLockRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanLockUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanLocks) CancelHumanLock(ctx context.Context, c factory.HumanLockCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanLock(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanLockError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanLockNamespace
	}
	if lookup.Record != nil {
		if _, err := humanLockRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanLock(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanLockError(err)
	}
	return humanLockRecord(c, q, digest, r)
}

func humanLockRecord(c factory.HumanLockCapture, q extensions.HumanLockRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanLockNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// The neutral contract describes the whole operation: proved no-effect has
	// no completion work; an unresolved primary stays pending. Native lock has
	// no separate notification step after its atomic state/history/receipt.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanLockUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanLock || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanLockConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.ApplyDiscussionLock, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanLockReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanLockReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanLockConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanLockUnavailable
	}
	return record, record.Validate()
}

func humanLockError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanLockQuery), errors.Is(err, extensions.ErrLockSnapshotQuery):
		return factoryservice.ErrHumanLockQuery
	case errors.Is(err, extensions.ErrHumanLockAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanLockAdmission
	case errors.Is(err, extensions.ErrHumanLockConflict), errors.Is(err, extensions.ErrLockSnapshotChanged):
		return factoryservice.ErrHumanLockConflict
	case errors.Is(err, extensions.ErrHumanLockUnavailable):
		return factoryservice.ErrHumanLockUnavailable
	default:
		return err
	}
}
