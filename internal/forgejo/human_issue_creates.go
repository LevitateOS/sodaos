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

// HumanIssueCreates translates the request's live human admission, never an agent
// grant. Provider identities/receipts remain separate from canonical bindings.
type HumanIssueCreates struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanIssueCreateWriter = HumanIssueCreates{}

// Resolve existing canonical mappings and the live human. The creation snapshot
// owns issue-unit permission; an unrelated code-unit read must not gate metadata
// creation, or recovery of a saved command after repository permission loss.
func (s HumanIssueCreates) scope(ctx context.Context, q factory.HumanIssueCreateCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanIssueCreateQuery
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
		return "", "", humanIssueCreateError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanIssueCreateAdmission
	}
	return repository, actor, nil
}

func (s HumanIssueCreates) PrepareHumanIssueCreate(ctx context.Context, q factory.HumanIssueCreateCommand) (factory.HumanIssueCreateCapture, error) {
	var empty factory.HumanIssueCreateCapture
	if q.Validate() != nil || s.Identities == nil || s.Identities.HostID() != q.Host {
		return empty, factoryservice.ErrHumanIssueCreateQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, humanIssueCreateError(err)
	}
	operation, err := extensions.HumanIssueCreateOperationID(actor, repository, q.ID)
	if err != nil {
		return empty, factoryservice.ErrHumanIssueCreateQuery
	}
	lookup, err := s.Authority.GetHumanIssueCreate(ctx, repository, operation)
	if err != nil {
		return empty, humanIssueCreateError(err)
	}
	// A previously observed operation without a saved application capture cannot
	// be attributed to a new preparation, even when its principal happens to match.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanIssueCreateConflict
	}
	snapshot, err := s.Authority.RepositoryIssueCreateSnapshot(ctx, repository)
	if err != nil {
		return empty, humanIssueCreateError(err)
	}
	if snapshot.Action.State != "allowed" {
		return empty, factoryservice.ErrHumanIssueCreateAdmission
	}
	intent := extensions.HumanIssueCreateRequest{OperationID: operation, Expected: snapshot, NotAfter: time.Now().Add(5 * time.Minute).Unix(), Title: q.Title, Body: q.Body}
	if !extensions.ValidHumanIssueCreateRequest(intent) {
		return empty, factoryservice.ErrHumanIssueCreateUnavailable
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	capture := factory.HumanIssueCreateCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = capture.Command()
	return capture, err
}

// Resolve only existing mappings and the current human. Receipt lookup/cancel
// remains possible after repository permission is lost; native admission still
// owns current session, installation, lifetime and contribution checks.
func (s HumanIssueCreates) decode(ctx context.Context, c factory.HumanIssueCreateCapture) (extensions.HumanIssueCreateRequest, string, error) {
	var q extensions.HumanIssueCreateRequest
	if _, err := c.Command(); err != nil {
		return q, "", factoryservice.ErrHumanIssueCreateQuery
	}
	if s.Identities == nil || s.Identities.HostID() != c.Request.Host || !extensions.ValidHumanIssueCreateInstallationID(c.Authority) {
		return q, "", factoryservice.ErrHumanIssueCreateQuery
	}
	repository, actor, err := s.scope(ctx, c.Request)
	if err != nil {
		return q, "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanIssueCreateRequest(q) {
		return q, "", factoryservice.ErrHumanIssueCreateQuery
	}
	requested := c.Request
	requested.Title, requested.Body = q.Title, q.Body
	operation, err := extensions.HumanIssueCreateOperationID(actor, repository, c.Request.ID)
	if err != nil || q.OperationID != operation || c.OperationID != operation || !requested.SameRequest(c.Request) ||
		q.Expected.RepositoryID != repository || q.Expected.ActorID != actor {
		return q, "", factoryservice.ErrHumanIssueCreateQuery
	}
	_, digest, err := extensions.CanonicalHumanIssueCreateIntent(q)
	return q, digest, err
}

func (s HumanIssueCreates) SubmitHumanIssueCreate(ctx context.Context, c factory.HumanIssueCreateCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	r, err := s.Authority.SubmitHumanIssueCreate(ctx, c.Authority, q)
	if err != nil {
		return collaboration.OperationRecord{}, humanIssueCreateError(err)
	}
	return humanIssueCreateRecord(c, q, digest, r)
}

func (s HumanIssueCreates) LookupHumanIssueCreate(ctx context.Context, c factory.HumanIssueCreateCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.Authority.GetHumanIssueCreate(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return empty, humanIssueCreateError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanIssueCreateNamespace
	}
	lookup := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanIssueCreateRecord(c, q, digest, *r.Record)
		if err != nil {
			return empty, err
		}
		lookup.State, lookup.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanIssueCreateUnavailable
	}
	return lookup, lookup.Validate()
}

func (s HumanIssueCreates) CancelHumanIssueCreate(ctx context.Context, c factory.HumanIssueCreateCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.Authority.GetHumanIssueCreate(ctx, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanIssueCreateError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanIssueCreateNamespace
	}
	if lookup.Record != nil {
		if _, err := humanIssueCreateRecord(c, q, digest, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	r, err := s.Authority.CancelHumanIssueCreate(ctx, c.Authority, q.Expected.RepositoryID, c.OperationID)
	if err != nil {
		return collaboration.OperationRecord{}, humanIssueCreateError(err)
	}
	return humanIssueCreateRecord(c, q, digest, r)
}

func (s HumanIssueCreates) ReadCreatedHumanIssue(ctx context.Context, c factory.HumanIssueCreateCapture, outcome collaboration.OperationLookup) (collaboration.IssueRef, error) {
	var empty collaboration.IssueRef
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	q, digest, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	if factory.ValidateHumanIssueCreateObservation(c, outcome) != nil || outcome.Record == nil || outcome.Record.Effect != collaboration.EffectCommitted || outcome.Record.Receipt == nil || outcome.Record.Receipt.IntentDigest != digest {
		return empty, factoryservice.ErrHumanIssueCreateConflict
	}
	var receipt extensions.HumanIssueCreateReceipt
	if json.Unmarshal(outcome.Record.Receipt.Payload, &receipt) != nil || !extensions.ValidHumanIssueCreateReceipt(receipt, q) {
		return empty, factoryservice.ErrHumanIssueCreateConflict
	}
	reads := factoryservice.CollaborationIssues{Source: HumanReads{Authority: s.Authority, Identities: s.Identities}}
	page, err := reads.Read(ctx, c.Request.Actor, c.Request.Repository, factoryservice.CollaborationIssueQuery{Number: receipt.Index, Limit: 1})
	if err != nil {
		return empty, err
	}
	ref := page.Items[0].Ref
	if ref.ExternalID != strconv.FormatInt(receipt.IssueID, 10) {
		return empty, factoryservice.ErrHumanIssueCreateConflict
	}
	return ref, nil
}

func humanIssueCreateRecord(c factory.HumanIssueCreateCapture, q extensions.HumanIssueCreateRequest, digest string, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanIssueCreateNamespace
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
			return empty, factoryservice.ErrHumanIssueCreateUnavailable
		}
		return record, record.Validate()
	}
	if r.Kind != extensions.OperationKindHumanIssueCreate || r.ActorID != q.Expected.ActorID || r.RepositoryID != q.Expected.RepositoryID || r.IntentDigest != digest {
		return empty, factoryservice.ErrHumanIssueCreateConflict
	}
	command, err := c.Command()
	if err != nil {
		return empty, err
	}
	record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.OpenIssue, Atomicity: collaboration.Atomic, IntentDigest: command.Digest}
	if r.EffectState == extensions.EffectCommitted {
		var receipt extensions.HumanIssueCreateReceipt
		if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanIssueCreateReceipt(receipt, q) {
			return empty, factoryservice.ErrHumanIssueCreateConflict
		}
		record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: digest, Payload: bytes.Clone(r.Receipt)}
	} else if len(r.Receipt) != 0 {
		return empty, factoryservice.ErrHumanIssueCreateUnavailable
	}
	return record, record.Validate()
}

func humanIssueCreateError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanIssueCreateQuery):
		return factoryservice.ErrHumanIssueCreateQuery
	case errors.Is(err, extensions.ErrHumanIssueCreateAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanIssueCreateAdmission
	case errors.Is(err, extensions.ErrHumanIssueCreateConflict):
		return factoryservice.ErrHumanIssueCreateConflict
	case errors.Is(err, extensions.ErrHumanIssueCreateUnavailable):
		return factoryservice.ErrHumanIssueCreateUnavailable
	default:
		return err
	}
}
