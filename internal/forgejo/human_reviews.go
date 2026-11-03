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

// HumanReviews translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanReviews struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanReviewWriter = HumanReviews{}

// Resolve existing canonical mappings and the live human. Review preparation
// owns Code/PR reads and event-specific native policy. Cleanup resolves only
// the current principal so permission loss does not discard an own receipt.
func (s HumanReviews) scope(ctx context.Context, q factory.HumanReviewCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanReviewQuery
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
		return "", "", humanReviewWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanReviewAdmission
	}
	return repository, actor, nil
}

func (s HumanReviews) PrepareHumanReview(ctx context.Context, q factory.HumanReviewCommand) (factory.HumanReviewCapture, error) {
	var empty factory.HumanReviewCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanReviewQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanReviewWriteError(err)
	}
	operation, err := extensions.HumanReviewOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanReviewQuery
	}
	lookup, err := s.Authority.GetHumanReview(ctx, repository, operation)
	if err != nil {
		return empty, humanReviewWriteError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanReviewConflict
	}
	snapshot, err := s.Authority.RepositoryReviewSnapshot(ctx, repository, extensions.RepositoryReviewSnapshotRequest{Index: strconv.FormatInt(q.Number, 10)})
	if err != nil {
		return empty, humanReviewWriteError(err)
	}
	action := snapshot.CommentAction
	switch q.Event {
	case factory.HumanReviewDraft:
		action = snapshot.DraftAction
	case factory.HumanReviewApprove:
		action = snapshot.ApproveAction
	case factory.HumanReviewRequestChanges:
		action = snapshot.RequestChangesAction
	}
	if action.State != "allowed" {
		return empty, factoryservice.ErrHumanReviewAdmission
	}
	intent := extensions.HumanReviewRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Event: humanReviewEvent(q.Event), Body: q.Body}
	if !extensions.ValidHumanReviewRequest(intent) {
		return empty, factoryservice.ErrHumanReviewQuery
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanReviewCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/review permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanReviews) decode(ctx context.Context, c factory.HumanReviewCapture) (extensions.HumanReviewRequest, string, error) {
	var q extensions.HumanReviewRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanReviewQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanReviewInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanReviewQuery
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
		return q, "", humanReviewWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanReviewAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanReviewRequest(q) {
		return q, "", factoryservice.ErrHumanReviewQuery
	}
	operation, err := extensions.HumanReviewOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || q.Event != humanReviewEvent(c.Request.Event) || q.Body != c.Request.Body ||
		q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanReviewQuery
	}
	_, digest, err := extensions.CanonicalHumanReviewIntent(q)
	return q, digest, err
}

func (s HumanReviews) SubmitHumanReview(ctx context.Context, c factory.HumanReviewCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanReview(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanReviewWriteError(err)
	}
	return humanReviewRecord(c, q, digest, r)
}

func (s HumanReviews) LookupHumanReview(ctx context.Context, c factory.HumanReviewCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanReview(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanReviewWriteError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanReviewNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanReviewRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanReviewUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanReviews) CancelHumanReview(ctx context.Context, c factory.HumanReviewCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanReview(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanReviewWriteError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanReviewNamespace
	}
	if lookup.Record != nil {
		if _, err := humanReviewRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanReview(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanReviewWriteError(err)
	}
	return humanReviewRecord(c, q, digest, r)
}

func humanReviewRecord(c factory.HumanReviewCapture, q extensions.HumanReviewRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanReviewNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	// Native final-review completion covers mention/notification work after primary.
	// The neutral contract covers the whole operation: proved no-effect needs
	// no completion work; unresolved primary stays pending. Preserve explicit
	// needs-intervention instead of declaring the committed review fully complete.
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		// No submitted metadata exists on a pre-submit cancellation tombstone.
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanReviewUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanReview || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanReviewConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.SubmitReview, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanReviewReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanReviewReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanReviewConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanReviewUnavailable
	}
	return record, record.Validate()
}

func humanReviewWriteError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanReviewQuery), errors.Is(err, extensions.ErrReviewSnapshotQuery):
		return factoryservice.ErrHumanReviewQuery
	case errors.Is(err, extensions.ErrHumanReviewAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanReviewAdmission
	case errors.Is(err, extensions.ErrHumanReviewConflict), errors.Is(err, extensions.ErrReviewSnapshotChanged):
		return factoryservice.ErrHumanReviewConflict
	case errors.Is(err, extensions.ErrHumanReviewUnavailable):
		return factoryservice.ErrHumanReviewUnavailable
	default:
		return err
	}
}

// Human review events are product semantics, separate from native wire values.
func humanReviewEvent(event factory.HumanReviewEvent) string {
	switch event {
	case factory.HumanReviewDraft:
		return extensions.HumanReviewEventDraft
	case factory.HumanReviewComment:
		return extensions.HumanReviewEventComment
	case factory.HumanReviewApprove:
		return extensions.ReviewSubmitEventApproved
	case factory.HumanReviewRequestChanges:
		return extensions.ReviewSubmitEventRequestChanges
	default:
		return ""
	}
}
