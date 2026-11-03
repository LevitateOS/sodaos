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

// HumanReviewRequests translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanReviewRequests struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanReviewRequestWriter = HumanReviewRequests{}

// Reviewer resolution is lookup-only, including cleanup. It proves identity,
// never current membership or permission. Native policy owns both add/remove.
func (s HumanReviewRequests) reviewer(ctx context.Context, q factory.HumanReviewRequestCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host || q.Validate() != nil {
		return "", "", factoryservice.ErrHumanReviewRequestQuery
	}
	if q.Reviewer > 0 {
		id, err := s.Identities.NativeActor(ctx, q.Reviewer)
		return "user", id, err
	}
	id, err := s.Identities.NativeTeam(ctx, q.Team)
	return "team", id, err
}

// Resolve existing canonical mappings and the live human. Reviewer preparation
// owns Code/PR reads, exact selected request evidence and separate add/remove policy.
// Cleanup resolves only the current principal so permission loss does not
// discard an own receipt.
func (s HumanReviewRequests) scope(ctx context.Context, q factory.HumanReviewRequestCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanReviewRequestQuery
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
		return "", "", humanReviewRequestWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanReviewRequestAdmission
	}
	return repository, actor, nil
}

func (s HumanReviewRequests) PrepareHumanReviewRequest(ctx context.Context, q factory.HumanReviewRequestCommand) (factory.HumanReviewRequestCapture, error) {
	var empty factory.HumanReviewRequestCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanReviewRequestQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanReviewRequestWriteError(err)
	}
	kind, reviewer, err := s.reviewer(ctx, q)
	if err != nil {
		return empty, err
	}
	operation, err := extensions.HumanReviewRequestOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanReviewRequestQuery
	}
	lookup, err := s.Authority.GetHumanReviewRequest(ctx, repository, operation)
	if err != nil {
		return empty, humanReviewRequestWriteError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanReviewRequestConflict
	}
	snapshot, err := s.Authority.RepositoryReviewRequestSnapshot(ctx, repository, extensions.RepositoryReviewRequestSnapshotRequest{Index: strconv.FormatInt(q.Number, 10), ReviewerKind: kind, ReviewerID: reviewer})
	if err != nil {
		return empty, humanReviewRequestWriteError(err)
	}
	action := snapshot.RemoveAction
	if q.Requested {
		action = snapshot.AddAction
	}
	if action.State != "allowed" {
		return empty, factoryservice.ErrHumanReviewRequestAdmission
	}
	intent := extensions.HumanReviewRequestChange{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Requested: q.Requested}
	if !extensions.ValidHumanReviewRequestChange(intent) {
		return empty, factoryservice.ErrHumanReviewRequestQuery
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanReviewRequestCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/review permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanReviewRequests) decode(ctx context.Context, c factory.HumanReviewRequestCapture) (extensions.HumanReviewRequestChange, string, error) {
	var q extensions.HumanReviewRequestChange
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanReviewRequestQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanReviewRequestInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanReviewRequestQuery
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
		return q, "", humanReviewRequestWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanReviewRequestAdmission
	}
	kind, reviewer, err := s.reviewer(ctx, c.Request)
	if err != nil {
		return q, "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanReviewRequestChange(q) {
		return q, "", factoryservice.ErrHumanReviewRequestQuery
	}
	operation, err := extensions.HumanReviewRequestOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || q.Requested != c.Request.Requested || q.Expected.ReviewerKind != kind || q.Expected.ReviewerID != reviewer ||
		q.Expected.Review.RepositoryID != repository || q.Expected.Review.ActorID != actor || q.Expected.Review.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanReviewRequestQuery
	}
	_, digest, err := extensions.CanonicalHumanReviewRequestIntent(q)
	return q, digest, err
}

func (s HumanReviewRequests) SubmitHumanReviewRequest(ctx context.Context, c factory.HumanReviewRequestCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanReviewRequest(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanReviewRequestWriteError(err)
	}
	return humanReviewRequestRecord(c, q, digest, r)
}

func (s HumanReviewRequests) LookupHumanReviewRequest(ctx context.Context, c factory.HumanReviewRequestCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanReviewRequest(ctx, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanReviewRequestWriteError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanReviewRequestNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanReviewRequestRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanReviewRequestUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanReviewRequests) CancelHumanReviewRequest(ctx context.Context, c factory.HumanReviewRequestCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanReviewRequest(ctx, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanReviewRequestWriteError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanReviewRequestNamespace
	}
	if lookup.Record != nil {
		if _, err := humanReviewRequestRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanReviewRequest(ctx, c.Authority, q.Expected.Review.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanReviewRequestWriteError(err)
	}
	return humanReviewRequestRecord(c, q, digest, r)
}

func humanReviewRequestRecord(c factory.HumanReviewRequestCapture, q extensions.HumanReviewRequestChange, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanReviewRequestNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// Committed reviewer changes retain native notification completion separately.
	// Proved no-effect needs no completion; an unresolved primary remains pending.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanReviewRequestUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanReviewRequest || r.ActorID != q.Expected.Review.ActorID || r.RepositoryID != q.Expected.Review.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanReviewRequestConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.RequestReview, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanReviewRequestReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanReviewRequestReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanReviewRequestConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanReviewRequestUnavailable
	}
	return record, record.Validate()
}

func humanReviewRequestWriteError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanReviewRequestQuery), errors.Is(err, extensions.ErrReviewRequestSnapshotQuery):
		return factoryservice.ErrHumanReviewRequestQuery
	case errors.Is(err, extensions.ErrHumanReviewRequestAdmission), errors.Is(err, extensions.ErrReviewRequestSnapshotAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanReviewRequestAdmission
	case errors.Is(err, extensions.ErrHumanReviewRequestConflict), errors.Is(err, extensions.ErrReviewRequestSnapshotChanged):
		return factoryservice.ErrHumanReviewRequestConflict
	case errors.Is(err, extensions.ErrHumanReviewRequestUnavailable), errors.Is(err, extensions.ErrReviewRequestSnapshotUnavailable):
		return factoryservice.ErrHumanReviewRequestUnavailable
	case errors.Is(err, extensions.ErrReviewRequestSnapshotTooLarge):
		return factoryservice.ErrHumanReviewRequestTooLarge
	default:
		return err
	}
}
