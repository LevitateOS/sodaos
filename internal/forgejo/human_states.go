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

// HumanStates translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanStates struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanStateWriter = HumanStates{}

// Resolve existing canonical mappings and the live human. State preparation
// owns subject-unit and author-or-writer permission. No Code-unit read gates
// direct close/reopen or recovery after repository permission loss.
func (s HumanStates) scope(ctx context.Context, q factory.HumanStateCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanStateQuery
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
		return "", "", humanStateError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanStateAdmission
	}
	return repository, actor, nil
}

func (s HumanStates) PrepareHumanState(ctx context.Context, q factory.HumanStateCommand) (factory.HumanStateCapture, error) {
	var empty factory.HumanStateCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanStateQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanStateError(err)
	}
	operation, err := extensions.HumanStateOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanStateQuery
	}
	lookup, err := s.Authority.GetHumanState(ctx, repository, operation)
	if err != nil {
		return empty, humanStateError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanStateConflict
	}
	snapshot, err := s.Authority.RepositoryStateSnapshot(ctx, repository, extensions.RepositoryStateSnapshotRequest{Subject: q.Subject, Index: strconv.FormatInt(q.Number, 10)})
	if err != nil {
		return empty, humanStateError(err)
	}
	if q.Close && snapshot.CloseAction.State != "allowed" || !q.Close && snapshot.ReopenAction.State != "allowed" {
		return empty, factoryservice.ErrHumanStateAdmission
	}
	intent := extensions.HumanStateRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Close: q.Close}
	if !extensions.ValidHumanStateRequest(intent) {
		return empty, factoryservice.ErrHumanStateUnavailable
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanStateCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/state permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanStates) decode(ctx context.Context, c factory.HumanStateCapture) (extensions.HumanStateRequest, string, error) {
	var q extensions.HumanStateRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanStateQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanStateInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanStateQuery
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
		return q, "", humanStateError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanStateAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanStateRequest(q) {
		return q, "", factoryservice.ErrHumanStateQuery
	}
	requested := c.Request
	requested.Close = q.Close
	operation, err := extensions.HumanStateOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || !requested.SameRequest(c.Request) ||
		q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Subject != c.Request.Subject || q.Expected.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanStateQuery
	}
	_, digest, err := extensions.CanonicalHumanStateIntent(q)
	return q, digest, err
}

func (s HumanStates) SubmitHumanState(ctx context.Context, c factory.HumanStateCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanState(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanStateError(err)
	}
	return humanStateRecord(c, q, digest, r)
}

func (s HumanStates) LookupHumanState(ctx context.Context, c factory.HumanStateCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanState(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanStateError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanStateNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanStateRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanStateUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanStates) CancelHumanState(ctx context.Context, c factory.HumanStateCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanState(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanStateError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanStateNamespace
	}
	if lookup.Record != nil {
		if _, err := humanStateRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanState(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanStateError(err)
	}
	return humanStateRecord(c, q, digest, r)
}

func humanStateRecord(c factory.HumanStateCapture, q extensions.HumanStateRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanStateNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// Native completion covers stopwatch/notification work after atomic primary.
	// The neutral contract covers the whole operation: proved no-effect needs
	// no completion work; unresolved primary stays pending. Preserve explicit
	// needs-intervention instead of declaring committed state fully complete.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanStateUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanState || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanStateConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.ApplyDiscussionState, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanStateReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanStateReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanStateConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanStateUnavailable
	}
	return record, record.Validate()
}

func humanStateError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanStateQuery), errors.Is(err, extensions.ErrStateSnapshotQuery):
		return factoryservice.ErrHumanStateQuery
	case errors.Is(err, extensions.ErrHumanStateAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanStateAdmission
	case errors.Is(err, extensions.ErrHumanStateConflict), errors.Is(err, extensions.ErrStateSnapshotChanged):
		return factoryservice.ErrHumanStateConflict
	case errors.Is(err, extensions.ErrHumanStateUnavailable):
		return factoryservice.ErrHumanStateUnavailable
	default:
		return err
	}
}
