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

// HumanInlines translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanInlines struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanInlineWriter = HumanInlines{}

// Resolve existing canonical mappings and the live human. Inline preparation
// owns Code/PR reads, exact immutable diff membership and mode-specific policy.
// Cleanup resolves only the current principal so permission loss does not
// discard an own receipt.
func (s HumanInlines) scope(ctx context.Context, q factory.HumanInlineCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanInlineQuery
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
		return "", "", humanInlineWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanInlineAdmission
	}
	return repository, actor, nil
}

func (s HumanInlines) PrepareHumanInline(ctx context.Context, q factory.HumanInlineCommand) (factory.HumanInlineCapture, error) {
	var empty factory.HumanInlineCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanInlineQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanInlineWriteError(err)
	}
	operation, err := extensions.HumanInlineOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanInlineQuery
	}
	lookup, err := s.Authority.GetHumanInline(ctx, repository, operation)
	if err != nil {
		return empty, humanInlineWriteError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanInlineConflict
	}
	snapshot, err := s.Authority.RepositoryInlineSnapshot(ctx, repository, extensions.RepositoryInlineSnapshotRequest{Index: strconv.FormatInt(q.Number, 10), Anchor: extensions.RepositoryInlineAnchor(q.Anchor)})
	if err != nil {
		return empty, humanInlineWriteError(err)
	}
	action := snapshot.Review.CommentAction
	if q.Draft {
		action = snapshot.Review.DraftAction
	}
	if action.State != "allowed" {
		return empty, factoryservice.ErrHumanInlineAdmission
	}
	intent := extensions.HumanInlineRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Draft: q.Draft, Body: q.Body}
	if !extensions.ValidHumanInlineRequest(intent) {
		return empty, factoryservice.ErrHumanInlineQuery
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanInlineCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/review permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanInlines) decode(ctx context.Context, c factory.HumanInlineCapture) (extensions.HumanInlineRequest, string, error) {
	var q extensions.HumanInlineRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanInlineQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanInlineInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanInlineQuery
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
		return q, "", humanInlineWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanInlineAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanInlineRequest(q) {
		return q, "", factoryservice.ErrHumanInlineQuery
	}
	operation, err := extensions.HumanInlineOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || q.Draft != c.Request.Draft || factory.ReviewThreadAnchor(q.Expected.Anchor) != c.Request.Anchor || q.Body != c.Request.Body ||
		q.Expected.Review.RepositoryID != repository || q.Expected.Review.ActorID != actor || q.Expected.Review.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanInlineQuery
	}
	_, digest, err := extensions.CanonicalHumanInlineIntent(q)
	return q, digest, err
}

func (s HumanInlines) SubmitHumanInline(ctx context.Context, c factory.HumanInlineCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanInline(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanInlineWriteError(err)
	}
	return humanInlineRecord(c, q, digest, r)
}

func (s HumanInlines) LookupHumanInline(ctx context.Context, c factory.HumanInlineCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanInline(ctx, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanInlineWriteError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanInlineNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanInlineRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanInlineUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanInlines) CancelHumanInline(ctx context.Context, c factory.HumanInlineCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanInline(ctx, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanInlineWriteError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanInlineNamespace
	}
	if lookup.Record != nil {
		if _, err := humanInlineRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanInline(ctx, c.Authority, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanInlineWriteError(err)
	}
	return humanInlineRecord(c, q, digest, r)
}

func humanInlineRecord(c factory.HumanInlineCapture, q extensions.HumanInlineRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanInlineNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// Native immediate-inline completion covers mention/notification work after primary.
	// The neutral contract covers the whole operation: proved no-effect needs
	// no completion work; unresolved primary stays pending. Preserve explicit
	// needs-intervention instead of declaring the committed inline fully complete.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanInlineUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanInline || r.ActorID != q.Expected.Review.ActorID || r.RepositoryID != q.Expected.Review.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanInlineConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.CreateReviewComment, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanInlineReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanInlineReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanInlineConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanInlineUnavailable
	}
	return record, record.Validate()
}

func humanInlineWriteError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanInlineQuery), errors.Is(err, extensions.ErrInlineSnapshotQuery), errors.Is(err, extensions.ErrInlineSnapshotAnchor):
		return factoryservice.ErrHumanInlineQuery
	case errors.Is(err, extensions.ErrHumanInlineAdmission), errors.Is(err, extensions.ErrInlineSnapshotAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanInlineAdmission
	case errors.Is(err, extensions.ErrHumanInlineConflict), errors.Is(err, extensions.ErrInlineSnapshotChanged):
		return factoryservice.ErrHumanInlineConflict
	case errors.Is(err, extensions.ErrHumanInlineUnavailable), errors.Is(err, extensions.ErrInlineSnapshotUnavailable):
		return factoryservice.ErrHumanInlineUnavailable
	default:
		return err
	}
}
