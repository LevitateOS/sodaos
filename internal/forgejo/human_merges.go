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

// HumanMerges uses the request's admitted human, never an agent grant. Native
// push and manual-mark operations have distinct identities and receipt shapes;
// canonical bindings remain owned by Factory's single merge command journal.
type HumanMerges struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.HumanMergeWriter = HumanMerges{}

// Cleanup resolves only existing mappings and the live principal. Repository
// permissions may be lost after an effect; the native owner still admits reads
// of that human's receipt without granting another mutation.
func (s HumanMerges) scope(ctx context.Context, q factory.HumanMergeCommand) (string, string, error) {
	if s.Identities == nil || s.Identities.HostID() != q.Host {
		return "", "", factoryservice.ErrHumanMergeQuery
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
		return "", "", humanMergeWriteError(err)
	}
	if current.ID != actor || current.ID != s.Authority.Actor.ID {
		return "", "", factoryservice.ErrHumanMergeAdmission
	}
	return repository, actor, nil
}

func humanMergeOperationID(actor, repository, command string, manual bool) (string, error) {
	if manual {
		return extensions.HumanManualMergeOperationID(actor, repository, command)
	}
	return extensions.HumanMergeOperationID(actor, repository, command)
}

func (s HumanMerges) get(ctx context.Context, repository, operation string, manual bool) (extensions.OperationLookup, error) {
	if manual {
		return s.Authority.GetHumanManualMerge(ctx, repository, operation)
	}
	return s.Authority.GetHumanMerge(ctx, repository, operation)
}

func (s HumanMerges) PrepareHumanMerge(ctx context.Context, q factory.HumanMergeCommand) (factory.HumanMergeCapture, error) {
	var empty factory.HumanMergeCapture
	if q.Validate() != nil {
		return empty, factoryservice.ErrHumanMergeQuery
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	repository, actor, err := s.scope(ctx, q)
	if err != nil {
		return empty, err
	}
	manual := q.Method == "manual"
	operation, err := humanMergeOperationID(actor, repository, q.ID, manual)
	if err != nil {
		return empty, factoryservice.ErrHumanMergeQuery
	}
	lookup, err := s.get(ctx, repository, operation, manual)
	if err != nil {
		return empty, humanMergeWriteError(err)
	}
	// Existing native writing without its original application capture cannot
	// be adopted by observing a new preparation, even for the same operator.
	if lookup.Status != extensions.BackgroundOutcomeNotObserved || lookup.Record != nil {
		return empty, factoryservice.ErrHumanMergeConflict
	}
	number := strconv.FormatInt(q.Number, 10)
	var intent any
	deadline := time.Now().Add(5 * time.Minute).Unix()
	if manual {
		p, err := s.Authority.RepositoryManualMergeSnapshot(ctx, repository, extensions.RepositoryManualMergeSnapshotRequest{Index: number, CommitOID: q.CommitOID})
		if err != nil {
			return empty, humanMergeWriteError(err)
		}
		if p.Action.State != "allowed" {
			return empty, factoryservice.ErrHumanMergeAdmission
		}
		r := extensions.HumanManualMergeRequest{OperationID: operation, Expected: p, NotAfter: deadline}
		if !extensions.ValidHumanManualMergeRequest(r) {
			return empty, factoryservice.ErrHumanMergeQuery
		}
		intent = r
	} else {
		p, err := s.Authority.RepositoryMergeSnapshot(ctx, repository, extensions.RepositoryMergeSnapshotRequest{Index: number, Method: q.Method})
		if err != nil {
			return empty, humanMergeWriteError(err)
		}
		if p.Action.State != "allowed" {
			return empty, factoryservice.ErrHumanMergeAdmission
		}
		r := extensions.HumanMergeRequest{OperationID: operation, Expected: p, Message: q.Message, NotAfter: deadline}
		if !extensions.ValidHumanMergeRequest(r) {
			return empty, factoryservice.ErrHumanMergeQuery
		}
		intent = r
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	c := factory.HumanMergeCapture{Request: q, Authority: lookup.InstallationID, OperationID: operation, ProviderIntent: raw}
	_, err = c.Command()
	return c, err
}

type humanMergeIntent struct {
	push       *extensions.HumanMergeRequest
	manual     *extensions.HumanManualMergeRequest
	repository string
	actor      string
	kind       string
	digest     string
}

// decode never refreshes preparation. The entire saved provider request must
// agree with canonical method/message/selected commit and native scope/identity.
func (s HumanMerges) decode(ctx context.Context, c factory.HumanMergeCapture) (humanMergeIntent, error) {
	var intent humanMergeIntent
	if _, err := c.Command(); err != nil || !extensions.ValidHumanMergeInstallationID(c.Authority) {
		return intent, factoryservice.ErrHumanMergeQuery
	}
	repository, actor, err := s.scope(ctx, c.Request)
	if err != nil {
		return intent, err
	}
	manual := c.Request.Method == "manual"
	operation, err := humanMergeOperationID(actor, repository, c.Request.ID, manual)
	if err != nil || c.OperationID != operation {
		return intent, factoryservice.ErrHumanMergeQuery
	}
	decoder := json.NewDecoder(bytes.NewReader(c.ProviderIntent))
	decoder.DisallowUnknownFields()
	number := strconv.FormatInt(c.Request.Number, 10)
	if manual {
		var q extensions.HumanManualMergeRequest
		if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanManualMergeRequest(q) || q.OperationID != operation || q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Index != number || q.Expected.CommitOID != c.Request.CommitOID {
			return intent, factoryservice.ErrHumanMergeQuery
		}
		_, intent.digest, err = extensions.CanonicalHumanManualMergeIntent(q)
		intent.manual, intent.kind = &q, extensions.OperationKindHumanManualMerge
	} else {
		var q extensions.HumanMergeRequest
		if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !extensions.ValidHumanMergeRequest(q) || q.OperationID != operation || q.Expected.RepositoryID != repository || q.Expected.ActorID != actor || q.Expected.Index != number || q.Expected.Method != c.Request.Method || q.Message != c.Request.Message {
			return intent, factoryservice.ErrHumanMergeQuery
		}
		_, intent.digest, err = extensions.CanonicalHumanMergeIntent(q)
		intent.push, intent.kind = &q, extensions.OperationKindHumanMerge
	}
	intent.repository, intent.actor = repository, actor
	return intent, humanMergeWriteError(err)
}

func (s HumanMerges) SubmitHumanMerge(ctx context.Context, c factory.HumanMergeCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	intent, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	var r extensions.OperationRecord
	if intent.manual != nil {
		r, err = s.Authority.SubmitHumanManualMerge(ctx, c.Authority, *intent.manual)
	} else {
		r, err = s.Authority.SubmitHumanMerge(ctx, c.Authority, *intent.push)
	}
	if err != nil {
		return collaboration.OperationRecord{}, humanMergeWriteError(err)
	}
	return humanMergeRecord(c, intent, r)
}

func (s HumanMerges) LookupHumanMerge(ctx context.Context, c factory.HumanMergeCapture) (collaboration.OperationLookup, error) {
	var empty collaboration.OperationLookup
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	intent, err := s.decode(ctx, c)
	if err != nil {
		return empty, err
	}
	r, err := s.get(ctx, intent.repository, c.OperationID, intent.manual != nil)
	if err != nil {
		return empty, humanMergeWriteError(err)
	}
	if r.InstallationID != c.Authority {
		return empty, factoryservice.ErrHumanMergeNamespace
	}
	o := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupNotObserved}
	if r.Record != nil {
		record, err := humanMergeRecord(c, intent, *r.Record)
		if err != nil {
			return empty, err
		}
		o.State, o.Record = collaboration.LookupFound, &record
	} else if r.Status != extensions.BackgroundOutcomeNotObserved {
		return empty, factoryservice.ErrHumanMergeUnavailable
	}
	return o, factory.ValidateHumanMergeObservation(c, o)
}

func (s HumanMerges) CancelHumanMerge(ctx context.Context, c factory.HumanMergeCapture) (collaboration.OperationRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	intent, err := s.decode(ctx, c)
	if err != nil {
		return collaboration.OperationRecord{}, err
	}
	lookup, err := s.get(ctx, intent.repository, c.OperationID, intent.manual != nil)
	if err != nil {
		return collaboration.OperationRecord{}, humanMergeWriteError(err)
	}
	if lookup.InstallationID != c.Authority {
		return collaboration.OperationRecord{}, factoryservice.ErrHumanMergeNamespace
	}
	if lookup.Record != nil {
		if _, err := humanMergeRecord(c, intent, *lookup.Record); err != nil {
			return collaboration.OperationRecord{}, err
		}
	}
	var r extensions.OperationRecord
	if intent.manual != nil {
		r, err = s.Authority.CancelHumanManualMerge(ctx, c.Authority, intent.repository, c.OperationID)
	} else {
		r, err = s.Authority.CancelHumanMerge(ctx, c.Authority, intent.repository, c.OperationID)
	}
	if err != nil {
		return collaboration.OperationRecord{}, humanMergeWriteError(err)
	}
	return humanMergeRecord(c, intent, r)
}

func humanMergeRecord(c factory.HumanMergeCapture, intent humanMergeIntent, r extensions.OperationRecord) (collaboration.OperationRecord, error) {
	var empty collaboration.OperationRecord
	if r.InstallationID != c.Authority || r.OperationID != c.OperationID {
		return empty, factoryservice.ErrHumanMergeNamespace
	}
	record := collaboration.OperationRecord{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID,
		Effect: collaboration.EffectState(r.EffectState), Cancellation: collaboration.CancellationState(r.CancellationStatus), Completion: collaboration.CompletionState(r.CompletionState), Reason: r.ReasonCode}
	if r.CompletionState == "" {
		record.Completion = collaboration.CompletionPending
		if r.EffectState == extensions.EffectNotCommitted {
			record.Completion = collaboration.CompletionComplete
		}
	}
	if r.Kind == "" {
		if r.ActorID != "" || r.RepositoryID != "" || r.IntentDigest != "" || len(r.Receipt) != 0 || r.CompletionState != "" {
			return empty, factoryservice.ErrHumanMergeUnavailable
		}
	} else {
		if r.Kind != intent.kind || r.ActorID != intent.actor || r.RepositoryID != intent.repository || r.IntentDigest != intent.digest {
			return empty, factoryservice.ErrHumanMergeConflict
		}
		cmd, err := c.Command()
		if err != nil {
			return empty, err
		}
		record.Binding = &collaboration.OperationBinding{Repository: c.Request.Repository, Actor: c.Request.Actor, Kind: collaboration.Merge, Atomicity: collaboration.Atomic, IntentDigest: cmd.Digest}
		if r.EffectState == extensions.EffectCommitted {
			if intent.manual != nil {
				var receipt extensions.HumanManualMergeReceipt
				if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanManualMergeReceipt(receipt, *intent.manual) {
					return empty, factoryservice.ErrHumanMergeConflict
				}
			} else {
				var receipt extensions.HumanMergeReceipt
				if json.Unmarshal(r.Receipt, &receipt) != nil || !extensions.ValidHumanMergeReceipt(receipt, *intent.push) {
					return empty, factoryservice.ErrHumanMergeConflict
				}
			}
			record.Receipt = &collaboration.ProviderReceipt{Authority: c.Authority, Reference: c.OperationID, IntentDigest: intent.digest, Payload: bytes.Clone(r.Receipt)}
		} else if len(r.Receipt) != 0 {
			return empty, factoryservice.ErrHumanMergeUnavailable
		}
	}
	o := collaboration.OperationLookup{Host: c.Request.Host, Authority: c.Authority, OperationID: c.OperationID, State: collaboration.LookupFound, Record: &record}
	return record, factory.ValidateHumanMergeObservation(c, o)
}

func humanMergeWriteError(err error) error {
	switch {
	case errors.Is(err, extensions.ErrHumanMergeQuery), errors.Is(err, extensions.ErrHumanManualMergeQuery), errors.Is(err, extensions.ErrMergeSnapshotQuery), errors.Is(err, extensions.ErrManualMergeSnapshotQuery), errors.Is(err, extensions.ErrManualMergeSnapshotCommit):
		return factoryservice.ErrHumanMergeQuery
	case errors.Is(err, extensions.ErrHumanMergeAdmission), errors.Is(err, extensions.ErrHumanManualMergeAdmission), errors.Is(err, extensions.ErrMergeSnapshotAdmission), errors.Is(err, extensions.ErrManualMergeSnapshotAdmission), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrHumanMergeAdmission
	case errors.Is(err, extensions.ErrHumanMergeConflict), errors.Is(err, extensions.ErrHumanManualMergeConflict), errors.Is(err, extensions.ErrMergeSnapshotChanged), errors.Is(err, extensions.ErrManualMergeSnapshotChanged):
		return factoryservice.ErrHumanMergeConflict
	case errors.Is(err, extensions.ErrHumanMergeUnavailable), errors.Is(err, extensions.ErrHumanManualMergeUnavailable), errors.Is(err, extensions.ErrMergeSnapshotUnavailable), errors.Is(err, extensions.ErrManualMergeSnapshotUnavailable):
		return factoryservice.ErrHumanMergeUnavailable
	default:
		return err
	}
}
