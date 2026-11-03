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

// HumanEdits translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanEdits struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanEditWriter = HumanEdits{}

func (s HumanEdits) PrepareHumanEdit(ctx context.Context, q factory.HumanEditCommand) (factory.HumanEditCapture, error) {
	var empty factory.HumanEditCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanEditQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := (HumanReads{Authority: s.Authority, Identities: s.Identities}).scope(ctx, q.Actor, q.Repository)
	if err != nil {
		return empty, humanEditError(err)
	}
	operation, err := extensions.HumanEditOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanEditQuery
	}
	lookup, err := s.Authority.GetHumanEdit(ctx, repository, operation)
	if err != nil {
		return empty, humanEditError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanEditConflict
	}
	snapshot, err := s.Authority.RepositoryEditSnapshot(ctx, repository, extensions.RepositoryEditSnapshotRequest{Subject: q.Subject, Index: strconv.FormatInt(q.Number, 10)})
	if err != nil {
		return empty, humanEditError(err)
	}
	if q.Title != nil && snapshot.TitleAction.State != "allowed" || q.Body != nil && snapshot.BodyAction.State != "allowed" {
		return empty, factoryservice.ErrHumanEditAdmission
	}
	intent := extensions.HumanEditRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Title: q.Title, Body: q.Body}
	if !extensions.ValidHumanEditRequest(intent) {
		return empty, factoryservice.ErrHumanEditUnavailable
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanEditCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	if q.Expected != nil {
		capture.Observed = humanEditBase(snapshot)
	}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after read/edit permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanEdits) decode(ctx context.Context, c factory.HumanEditCapture) (extensions.HumanEditRequest, string, error) {
	var q extensions.HumanEditRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanEditQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanEditInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanEditQuery
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
		return q, "", humanEditError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return q, "", factoryservice.ErrHumanEditAdmission
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanEditRequest(q) {
		return q, "", factoryservice.ErrHumanEditQuery
	}
	requested := c.Request
	requested.Title, requested.Body = q.Title, q.Body
	operation, err := extensions.HumanEditOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || !requested.SameRequest(c.Request) ||
		q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Subject != c.Request.Subject || q.Expected.Index != strconv.FormatInt(c.Request.Number, 10) {
		return q, "", factoryservice.ErrHumanEditQuery
	}
	if c.Request.Expected != nil && (c.Observed == nil || *c.Observed != *humanEditBase(q.Expected)) {
		return q, "", factoryservice.ErrHumanEditConflict
	}
	_, digest, err := extensions.CanonicalHumanEditIntent(q)
	return q, digest, err
}

func (s HumanEdits) SubmitHumanEdit(ctx context.Context, c factory.HumanEditCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	if !c.ExpectedCurrent() {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanEditBaseChanged
	}
	r, err := s.Authority.SubmitHumanEdit(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanEditError(err)
	}
	return humanEditRecord(c, q, digest, r)
}

func humanEditBase(snapshot extensions.RepositoryEditSnapshot) *factory.HumanEditBase {
	return &factory.HumanEditBase{ExternalID: snapshot.SubjectID, Title: snapshot.Title, BodyDigest: snapshot.BodyDigest}
}

func (s HumanEdits) LookupHumanEdit(ctx context.Context, c factory.HumanEditCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanEdit(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanEditError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanEditNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanEditRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanEditUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanEdits) CancelHumanEdit(ctx context.Context, c factory.HumanEditCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanEdit(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanEditError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanEditNamespace
	}
	if lookup.Record != nil {
		if _, err := humanEditRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanEdit(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanEditError(err)
	}
	return humanEditRecord(c, q, digest, r)
}

func humanEditRecord(c factory.HumanEditCapture, q extensions.HumanEditRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanEditNamespace
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
			return empty, factoryservice.ErrHumanEditUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanEdit || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanEditConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.EditDiscussion, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanEditReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanEditReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanEditConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanEditUnavailable
	}
	return record, record.Validate()
}

func humanEditError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanEditQuery):
		return factoryservice.ErrHumanEditQuery
	case errors.Is(err, extensions.ErrHumanEditAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanEditAdmission
	case errors.Is(err, extensions.ErrHumanEditConflict):
		return factoryservice.ErrHumanEditConflict
	case errors.Is(err, extensions.ErrHumanEditUnavailable):
		return factoryservice.ErrHumanEditUnavailable
	default:
		return err
	}
}
