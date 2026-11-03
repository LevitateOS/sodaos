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

// HumanComments translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanComments struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanCommentWriter = HumanComments{}

func (s HumanComments) PrepareHumanComment(ctx context.Context, q factory.HumanCommentCommand) (factory.HumanCommentCapture, error) {
	var empty factory.HumanCommentCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanCommentQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := (HumanReads{Authority: s.Authority, Identities: s.Identities}).scope(ctx, q.Actor, q.Repository)
	if err != nil {
		return empty, humanCommentError(err)
	}
	operation, err := extensions.HumanCommentOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanCommentQuery
	}
	lookup, err := s.Authority.GetHumanComment(ctx, repository, operation)
	if err != nil {
		return empty, humanCommentError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanCommentConflict
	}
	snapshot, err := s.Authority.RepositoryCommentSnapshot(ctx, repository, extensions.RepositoryCommentSnapshotRequest{Subject: q.Subject, Index: strconv.FormatInt(q.Number, 10)})
	if err != nil {
		return empty, humanCommentError(err)
	}
	if snapshot.Action.State != "allowed" {
		return empty, factoryservice.ErrHumanCommentAdmission
	}
	intent := extensions.HumanCommentRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Body: q.Body}
	if !extensions.ValidHumanCommentRequest(intent) {
		return empty, factoryservice.ErrHumanCommentUnavailable
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanCommentCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/comment permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanComments) decode(ctx context.Context, c factory.HumanCommentCapture) (extensions.HumanCommentRequest, string, error) {
	var q extensions.HumanCommentRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanCommentQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanCommentInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanCommentQuery
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
		return q, "", humanCommentError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanCommentAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanCommentRequest(q) {
		return q, "", factoryservice.ErrHumanCommentQuery
	}
	operation, err := extensions.HumanCommentOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || q.Body != c.Request.Body ||
		q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Subject != c.Request.Subject || q.Expected.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanCommentQuery
	}
	_, digest, err := extensions.CanonicalHumanCommentIntent(q)
	return q, digest, err
}

func (s HumanComments) SubmitHumanComment(ctx context.Context, c factory.HumanCommentCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanComment(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanCommentError(err)
	}
	return humanCommentRecord(c, q, digest, r)
}

func (s HumanComments) LookupHumanComment(ctx context.Context, c factory.HumanCommentCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanComment(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanCommentError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanCommentNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanCommentRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanCommentUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanComments) CancelHumanComment(ctx context.Context, c factory.HumanCommentCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanComment(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanCommentError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanCommentNamespace
	}
	if lookup.Record != nil {
		if _, err := humanCommentRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanComment(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanCommentError(err)
	}
	return humanCommentRecord(c, q, digest, r)
}

func humanCommentRecord(c factory.HumanCommentCapture, q extensions.HumanCommentRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanCommentNamespace
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
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.EffectState != extensions.EffectNotCommitted || r.CancellationStatus != extensions.CancellationCancelled {
			return empty, factoryservice.ErrHumanCommentUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanComment || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanCommentConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.CreateComment, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanCommentReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || strconv.FormatInt(receipt.RepositoryID, 10) != q.Expected.RepositoryID || strconv.FormatInt(receipt.ActorID, 10) != q.Expected.ActorID ||
			receipt.Subject != q.Expected.Subject || strconv.FormatInt(receipt.SubjectID, 10) != q.Expected.SubjectID || strconv.FormatInt(receipt.IssueID, 10) != q.Expected.IssueID || strconv.FormatInt(receipt.Index, 10) != q.Expected.Index || receipt.CommentID < 1 {
			return empty, factoryservice.ErrHumanCommentConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanCommentUnavailable
	}
	return record, record.Validate()
}

func humanCommentError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanCommentQuery):
		return factoryservice.ErrHumanCommentQuery
	case errors.Is(err, extensions.ErrHumanCommentAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanCommentAdmission
	case errors.Is(err, extensions.ErrHumanCommentConflict):
		return factoryservice.ErrHumanCommentConflict
	case errors.Is(err, extensions.ErrHumanCommentUnavailable):
		return factoryservice.ErrHumanCommentUnavailable
	default:
		return err
	}
}
