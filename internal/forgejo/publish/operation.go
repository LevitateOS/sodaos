package publish

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

// StatusError reports a bounded native dispatch status with its bounded
// body. Callers map terminal statuses without retrying and reconcile
// after the rest.
type StatusError struct {
	Body   string
	Status int
}

func (e *StatusError) Error() string {
	return "background request returned status " + strconv.Itoa(e.Status)
}

// BackgroundOperations is the publisher's narrow native-operation
// transport: bracketed revision reads plus submit, lookup and cancel for
// the installation's own conditional operations, and the current
// installation admission bound into push headers. One installation holds
// one current service admission: the transport owns its bootstrap and
// rebinds after revocation, so readers and publishers never split it.
type BackgroundOperations interface {
	ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error)
	SubmitOperation(ctx context.Context, credential extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error)
	GetOperation(ctx context.Context, operationID string) (extensions.OperationLookup, error)
	CancelOperation(ctx context.Context, operationID string) (extensions.OperationRecord, error)
	// PublishPushEnv binds one registered operation to its push headers
	// under the current admission. Rebind refreshes a revoked admission
	// first; a push that failed before launching retries under the new
	// admission only after a lookup still shows it pending.
	PublishPushEnv(ctx context.Context, operationID string, rebind bool) ([]string, error)
}

// Refusal is a terminal verdict: the operation will not proceed under
// its recorded identity. The controller records it and fails the
// publication instead of retrying an identical call.
type Refusal struct{ Reason string }

func (e *Refusal) Error() string { return "publication refused: " + e.Reason }

// Wait reports that no verdict exists yet: the call reached no native
// decision and the publication stays open for a later pass. It never
// advances the stage.
type Wait struct{ Reason string }

func (e *Wait) Error() string { return "publication waits: " + e.Reason }

// Operation intent lifetimes. Submit deadlines stay well inside the
// native one-hour maximum; host time authorizes them at submit.
const operationLifetime = 10 * time.Minute

// PublishOpIntent is one branch-publication submit: the persisted
// operation identity, the policy-bound actor and authority, the bracketed
// observation and the exact ref tuple. A nonzero correction binds an
// update to its existing PR; zero means initial publication.
type PublishOpIntent struct {
	Credential            extensions.CredentialFile
	OperationID           string
	AuthRevision          string
	Ref                   string
	ExpectedOld           string
	NewOID                string
	ComparisonRef         string
	ExpectedComparisonOID string
	ActorID               int64
	RepositoryID          int64
	ExpectedRevision      int64
	NotAfter              int64
	CorrectionNumber      int64
	CorrectionAuthor      int64
}

// SubmitPublish submits one branch-publication intent under its persisted
// identity and reconciles the answer. A lost reply reconciles by lookup:
// the operation is adopted, never resubmitted blindly. A terminal record
// is adopted as well; only not_observed permits a later resubmit.
func (c Config) SubmitPublish(ctx context.Context, ops BackgroundOperations, in PublishOpIntent) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	payload, err := publishPayload(in)
	if err != nil {
		return empty, err
	}
	intent := extensions.OperationIntent{
		OperationID: in.OperationID, ActorID: strconv.FormatInt(in.ActorID, 10),
		RepositoryID: strconv.FormatInt(in.RepositoryID, 10), Kind: factory.OpRefPublish,
		AuthorizationRevision: in.AuthRevision, ExpectedNativeRevision: in.ExpectedRevision,
		NotAfter: in.NotAfter, Payload: payload,
	}
	return submitOperation(ctx, ops, in.Credential, intent)
}

func publishPayload(in PublishOpIntent) (json.RawMessage, error) {
	if !factory.ValidPublicationOperationID(in.OperationID) {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ActorID <= 0 || in.RepositoryID <= 0 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.AuthRevision == "" || len(in.AuthRevision) > 512 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ExpectedRevision < 1 || in.NotAfter <= time.Now().Unix() {
		return nil, &Wait{Reason: "observation_expired"}
	}
	payload := extensions.PublishPayload{
		Ref: in.Ref, ExpectedOld: in.ExpectedOld, NewOID: in.NewOID,
		ComparisonRef: in.ComparisonRef, ExpectedComparisonOID: in.ExpectedComparisonOID,
	}
	if in.CorrectionNumber != 0 || in.CorrectionAuthor != 0 {
		payload.Correction = &extensions.PublishCorrection{Number: in.CorrectionNumber, ExpectedAuthorID: in.CorrectionAuthor}
	}
	if err := extensions.ValidatePublishPayload(payload); err != nil {
		return nil, &Refusal{Reason: "invalid_refs"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// PRCreateOpIntent is one PR-creation submit: the persisted operation
// identity, the policy-bound actor and authority, the bracketed
// observation and the exact head/base snapshot with final title and body.
type PRCreateOpIntent struct {
	Credential       extensions.CredentialFile
	OperationID      string
	AuthRevision     string
	HeadRef          string
	BaseRef          string
	ExpectedHead     string
	ExpectedBase     string
	Title            string
	Body             string
	ActorID          int64
	RepositoryID     int64
	ExpectedRevision int64
	NotAfter         int64
}

// SubmitPRCreate submits one PR-creation intent under its persisted
// identity with the same lost-reply reconciliation as SubmitPublish.
func (c Config) SubmitPRCreate(ctx context.Context, ops BackgroundOperations, in PRCreateOpIntent) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	payload, err := prCreatePayload(in)
	if err != nil {
		return empty, err
	}
	intent := extensions.OperationIntent{
		OperationID: in.OperationID, ActorID: strconv.FormatInt(in.ActorID, 10),
		RepositoryID: strconv.FormatInt(in.RepositoryID, 10), Kind: factory.OpPRCreate,
		AuthorizationRevision: in.AuthRevision, ExpectedNativeRevision: in.ExpectedRevision,
		NotAfter: in.NotAfter, Payload: payload,
	}
	return submitOperation(ctx, ops, in.Credential, intent)
}

func prCreatePayload(in PRCreateOpIntent) (json.RawMessage, error) {
	if !factory.ValidPublicationOperationID(in.OperationID) {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ActorID <= 0 || in.RepositoryID <= 0 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.AuthRevision == "" || len(in.AuthRevision) > 512 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ExpectedRevision < 1 || in.NotAfter <= time.Now().Unix() {
		return nil, &Wait{Reason: "observation_expired"}
	}
	payload := extensions.PRCreatePayload{
		HeadRepositoryID: in.RepositoryID, HeadRef: in.HeadRef, BaseRef: in.BaseRef,
		ExpectedHeadOID: in.ExpectedHead, ExpectedBaseOID: in.ExpectedBase,
		Title: in.Title, Body: in.Body,
	}
	if err := extensions.ValidatePRCreatePayload(in.RepositoryID, payload); err != nil {
		return nil, &Refusal{Reason: "invalid_pr"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func submitOperation(ctx context.Context, ops BackgroundOperations, credential extensions.CredentialFile, intent extensions.OperationIntent) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if ops == nil {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	record, err := ops.SubmitOperation(ctx, credential, intent)
	if err != nil {
		return empty, mapSubmitError(ctx, ops, intent.OperationID, err)
	}
	outcome, err := mapRecord(intent.OperationID, record)
	if err != nil {
		return empty, err
	}
	if record.Kind != intent.Kind || record.ActorID != intent.ActorID || record.RepositoryID != intent.RepositoryID {
		return empty, errors.New("operation record differs from submitted intent")
	}
	return outcome, nil
}

// mapSubmitError reconciles a failed submit before reporting it. A lost
// reply may hide a recorded operation: the lookup adopts it, and only a
// not_observed lookup leaves the identity unsubmitted. Terminal dispatch
// verdicts refuse or wait without another submit.
func mapSubmitError(ctx context.Context, ops BackgroundOperations, operationID string, err error) error {
	var status *StatusError
	if !errors.As(err, &status) {
		return reconcileAfterSubmitError(ctx, ops, operationID, err)
	}
	switch {
	case status.Status == 400:
		return &Refusal{Reason: "invalid_intent"}
	case status.Status == 401:
		return &Wait{Reason: "credential_invalid"}
	case status.Status == 403:
		return &Wait{Reason: "authority_lost"}
	case status.Status == 409 && status.Body == "intent_conflict":
		return &Refusal{Reason: "intent_conflict"}
	case status.Status == 409 && status.Body == "cancelled_before_submit":
		return reconcileAfterSubmitError(ctx, ops, operationID, err)
	default:
		return reconcileAfterSubmitError(ctx, ops, operationID, err)
	}
}

// reconcileAfterSubmitError adopts a possibly recorded operation after a
// submit call failed. NotObserved leaves the identity free for a later
// submit; any recorded state is adopted; a second failure reports the
// original error so the pass reconciles later.
func reconcileAfterSubmitError(ctx context.Context, ops BackgroundOperations, operationID string, submitErr error) error {
	outcome, err := lookupOperation(ctx, ops, operationID)
	if err != nil {
		return submitErr
	}
	return &Adopted{Outcome: outcome}
}

// Adopted carries a reconciled operation outcome discovered while
// handling a failed call. The caller records it instead of the error.
type Adopted struct{ Outcome factory.OperationOutcome }

func (e *Adopted) Error() string { return "publication operation reconciled after a lost reply" }

// LookupOperation reconciles one persisted operation identity against
// native state. NotObserved means no record exists yet; it never proves
// that an earlier request cannot still arrive.
func (c Config) LookupOperation(ctx context.Context, ops BackgroundOperations, operationID string) (factory.OperationOutcome, error) {
	if !factory.ValidPublicationOperationID(operationID) {
		return factory.OperationOutcome{}, &Refusal{Reason: "invalid_operation"}
	}
	if ops == nil {
		return factory.OperationOutcome{}, &Wait{Reason: "operations_unavailable"}
	}
	return lookupOperation(ctx, ops, operationID)
}

func lookupOperation(ctx context.Context, ops BackgroundOperations, operationID string) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	lookup, err := ops.GetOperation(ctx, operationID)
	if err != nil {
		return empty, mapLookupError(err)
	}
	if lookup.OperationID != operationID || lookup.InstallationID == "" {
		return empty, errors.New("operation lookup identity differs")
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		if lookup.Record != nil {
			return empty, errors.New("operation lookup contradicts its status")
		}
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	if lookup.Record == nil {
		return empty, errors.New("operation lookup lacks its record")
	}
	if lookup.Record.InstallationID != lookup.InstallationID {
		return empty, errors.New("operation record installation differs from lookup")
	}
	return mapRecord(operationID, *lookup.Record)
}

func mapLookupError(err error) error {
	var status *StatusError
	if !errors.As(err, &status) {
		return err
	}
	switch status.Status {
	case 400:
		return &Refusal{Reason: "invalid_operation"}
	case 401, 403:
		return &Wait{Reason: "operations_unavailable"}
	default:
		return err
	}
}

// CancelOperation cancels one persisted operation identity and reconciles
// the answer. Cancellation contends on the same record as submission: a
// winning cancellation prevents the effect, a committed write makes it
// too late, and a lost reply reconciles by lookup.
func (c Config) CancelOperation(ctx context.Context, ops BackgroundOperations, operationID string) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if !factory.ValidPublicationOperationID(operationID) {
		return empty, &Refusal{Reason: "invalid_operation"}
	}
	if ops == nil {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	record, err := ops.CancelOperation(ctx, operationID)
	if err != nil {
		var status *StatusError
		if errors.As(err, &status) && (status.Status == 400 || status.Status == 401 || status.Status == 403) {
			return empty, &Wait{Reason: "operations_unavailable"}
		}
		outcome, lookupErr := lookupOperation(ctx, ops, operationID)
		if lookupErr != nil {
			return empty, err
		}
		return outcome, nil
	}
	return mapRecord(operationID, record)
}

// mapRecord adopts one native operation record. Identity, kind and state
// vocabularies are verified; anything else refuses instead of adopting a
// lookalike. An unimplemented-kind outcome waits without a record.
func mapRecord(operationID string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if record.OperationID != operationID || record.InstallationID == "" {
		return empty, errors.New("operation record identity differs")
	}
	if record.Outcome == extensions.BackgroundOutcomeOperationsUnavailable {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	outcome := factory.OperationOutcome{
		OperationID: record.OperationID, InstallationID: record.InstallationID, Kind: record.Kind,
		Effect: record.EffectState, Cancellation: record.CancellationStatus,
		Completion: record.CompletionState, Reason: record.ReasonCode,
	}
	if !factory.ValidOpEffect(outcome.Effect) {
		return empty, errors.New("operation effect is unknown")
	}
	if outcome.Cancellation == "" {
		outcome.Cancellation = factory.OpCancelNone
	}
	if !factory.ValidOpCancellation(outcome.Cancellation) {
		return empty, errors.New("operation cancellation is unknown")
	}
	// Cancellation may win before submit. Its tombstone has no actor,
	// repository or kind; every submitted record must retain all three.
	tombstone := record.Kind == "" && record.ActorID == "" && record.RepositoryID == "" &&
		outcome.Effect == factory.OpEffectNotCommitted && outcome.Cancellation == factory.OpCancelCancelled
	if !tombstone {
		if record.Kind != factory.OpRefPublish && record.Kind != factory.OpPRCreate {
			return empty, errors.New("operation kind differs from publication")
		}
		actor, actorErr := strconv.ParseInt(record.ActorID, 10, 64)
		repository, repositoryErr := strconv.ParseInt(record.RepositoryID, 10, 64)
		if actorErr != nil || repositoryErr != nil || actor <= 0 || repository <= 0 {
			return empty, errors.New("operation record lacks its native scope")
		}
		outcome.ActorID, outcome.RepositoryID = actor, repository
	}
	if outcome.Completion != "" && !factory.ValidOpCompletion(outcome.Completion) {
		return empty, errors.New("operation completion is unknown")
	}
	if len(outcome.Reason) > 64 {
		return empty, errors.New("operation reason exceeds bounds")
	}
	if len(record.Receipt) > factory.MaxPublicationReceipt {
		return empty, errors.New("operation receipt exceeds bounds")
	}
	outcome.Receipt = append([]byte(nil), record.Receipt...)
	return outcome, nil
}
