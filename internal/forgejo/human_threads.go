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

// HumanThreads translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanThreads struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanThreadWriter = HumanThreads{}
var _ factoryservice.HumanThreadReader = HumanThreads{}

func (s HumanThreads) ReadHumanThreadSelection(ctx context.Context, q factory.HumanThreadSelectionQuery) (factory.HumanThreadSelection, error) {
	var empty factory.HumanThreadSelection
	if q.Validate() != nil {
		return empty, factoryservice.ErrHumanThreadQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	scope := factory.HumanThreadCommand{Host: q.Host, Actor: q.Actor, Repository: q.Repository, Number: q.Number}
	repository, _, err := s.scope(ctx, scope)
	if err != nil {
		return empty, humanThreadWriteError(err)
	}
	p, err := s.Authority.RepositoryThreadSnapshot(ctx, repository, extensions.RepositoryThreadSnapshotRequest{Index: strconv.FormatInt(q.Number, 10), ThreadID: q.ThreadID})
	if err != nil {
		return empty, humanThreadWriteError(err)
	}
	if _, _, err := s.scope(ctx, scope); err != nil {
		return empty, humanThreadWriteError(err)
	}
	selection := factory.HumanThreadSelection{
		Host: q.Host, Actor: q.Actor, Repository: q.Repository, Number: q.Number, ThreadID: q.ThreadID,
		OriginalAnchor: humanThreadOriginalAnchor(p.Thread.OriginalAnchor), Resolved: p.Thread.Resolved,
		Resolve:   collaboration.Capability{State: collaboration.Availability(p.ResolveAction.State), Reason: p.ResolveAction.Reason},
		Unresolve: collaboration.Capability{State: collaboration.Availability(p.UnresolveAction.State), Reason: p.UnresolveAction.Reason},
	}
	if selection.Validate(q) != nil {
		return empty, factoryservice.ErrHumanThreadUnavailable
	}
	return selection, nil
}

// Resolve existing canonical mappings and the live human. Thread preparation
// owns Code/PR reads, exact conversation/original context and resolve/unresolve policy.
// Cleanup resolves only the current principal so permission loss does not
// discard an own receipt.
func (s HumanThreads) scope(ctx context.Context, q factory.HumanThreadCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanThreadQuery
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
		return "", "", humanThreadWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanThreadAdmission
	}
	return repository, actor, nil
}

func (s HumanThreads) PrepareHumanThread(ctx context.Context, q factory.HumanThreadCommand) (factory.HumanThreadCapture, error) {
	var empty factory.HumanThreadCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanThreadQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanThreadWriteError(err)
	}
	operation, err := extensions.HumanThreadOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanThreadQuery
	}
	lookup, err := s.Authority.GetHumanThread(ctx, repository, operation)
	if err != nil {
		return empty, humanThreadWriteError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanThreadConflict
	}
	snapshot, err := s.Authority.RepositoryThreadSnapshot(ctx, repository, extensions.RepositoryThreadSnapshotRequest{Index: strconv.FormatInt(q.Number, 10), ThreadID: q.ThreadID})
	if err != nil {
		return empty, humanThreadWriteError(err)
	}
	if !humanThreadOriginalAnchor(snapshot.Thread.OriginalAnchor).Equal(q.Anchor) {
		return empty, factoryservice.ErrHumanThreadConflict
	}
	action := snapshot.UnresolveAction
	if q.Resolved {
		action = snapshot.ResolveAction
	}
	if action.State != "allowed" {
		return empty, factoryservice.ErrHumanThreadAdmission
	}
	intent := extensions.HumanThreadRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Resolved: q.Resolved}
	if !extensions.ValidHumanThreadRequest(intent) {
		return empty, factoryservice.ErrHumanThreadQuery
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanThreadCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/review permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanThreads) decode(ctx context.Context, c factory.HumanThreadCapture) (extensions.HumanThreadRequest, string, error) {
	var q extensions.HumanThreadRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanThreadQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanThreadInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanThreadQuery
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
		return q, "", humanThreadWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanThreadAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanThreadRequest(q) {
		return q, "", factoryservice.ErrHumanThreadQuery
	}
	operation, err := extensions.HumanThreadOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || q.Resolved != c.Request.Resolved || q.Expected.Thread.RootCommentID != c.Request.ThreadID || !humanThreadOriginalAnchor(q.Expected.Thread.OriginalAnchor).Equal(c.Request.Anchor) ||
		q.Expected.Review.RepositoryID != repository || q.Expected.Review.ActorID != actor || q.Expected.Review.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanThreadQuery
	}
	_, digest, err := extensions.CanonicalHumanThreadIntent(q)
	return q, digest, err
}

func (s HumanThreads) SubmitHumanThread(ctx context.Context, c factory.HumanThreadCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanThread(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanThreadWriteError(err)
	}
	return humanThreadRecord(c, q, digest, r)
}

func (s HumanThreads) LookupHumanThread(ctx context.Context, c factory.HumanThreadCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanThread(ctx, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanThreadWriteError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanThreadNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanThreadRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanThreadUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanThreads) CancelHumanThread(ctx context.Context, c factory.HumanThreadCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanThread(ctx, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanThreadWriteError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanThreadNamespace
	}
	if lookup.Record != nil {
		if _, err := humanThreadRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanThread(ctx, c.Authority, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanThreadWriteError(err)
	}
	return humanThreadRecord(c, q, digest, r)
}

func humanThreadRecord(c factory.HumanThreadCapture, q extensions.HumanThreadRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanThreadNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// Thread resolution has no post-primary notification work. Proved no-effect
	// needs no completion; an unresolved primary remains pending.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanThreadUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanThread || r.ActorID != q.Expected.Review.ActorID || r.RepositoryID != q.Expected.Review.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanThreadConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.ResolveReviewThread, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanThreadReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanThreadReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanThreadConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanThreadUnavailable
	}
	return record, record.Validate()
}

func humanThreadWriteError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanThreadQuery), errors.Is(err, extensions.ErrThreadSnapshotQuery), errors.Is(err, extensions.ErrThreadSnapshotTarget):
		return factoryservice.ErrHumanThreadQuery
	case errors.Is(err, extensions.ErrHumanThreadAdmission), errors.Is(err, extensions.ErrThreadSnapshotAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanThreadAdmission
	case errors.Is(err, extensions.ErrHumanThreadConflict), errors.Is(err, extensions.ErrThreadSnapshotChanged):
		return factoryservice.ErrHumanThreadConflict
	case errors.Is(err, extensions.ErrHumanThreadUnavailable), errors.Is(err, extensions.ErrThreadSnapshotUnavailable):
		return factoryservice.ErrHumanThreadUnavailable
	case errors.Is(err, extensions.ErrThreadSnapshotTooLarge):
		return factoryservice.ErrHumanThreadTooLarge
	default:
		return err
	}
}

// Original context is retained exactly; nil never becomes current blame.
func humanThreadOriginalAnchor(a *extensions.RepositoryInlineAnchor) *factory.ReviewThreadAnchor {
	if a == nil {
		return nil
	}
	anchor := factory.ReviewThreadAnchor(*a)
	return &anchor
}
