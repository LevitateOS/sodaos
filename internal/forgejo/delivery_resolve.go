package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

type Resolver struct{ reviewer *Reviewer }

func NewResolver(background *ServiceBackground, rest *Client, tokenFile string, identities *FountainIdentities) *Resolver {
	return &Resolver{reviewer: NewReviewer(background, rest, tokenFile, identities)}
}

func resolveOutcome(id string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	o, err := reviewOutcomeState(id, record)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	if record.Kind == "" && record.ActorID == "" && record.RepositoryID == "" && o.Effect == factory.OpEffectNotCommitted && o.Cancellation == factory.OpCancelCancelled && len(record.Receipt) == 0 {
		return o, nil
	}
	actor, actorErr := strconv.ParseInt(record.ActorID, 10, 64)
	repository, repoErr := strconv.ParseInt(record.RepositoryID, 10, 64)
	if record.Kind != factory.OpThreadResolve || actorErr != nil || repoErr != nil || actor <= 0 || repository <= 0 || len(record.Receipt) > 4096 {
		return factory.OperationOutcome{}, errors.New("native resolution scope differs from its intent")
	}
	o.ActorID, o.RepositoryID = actor, repository
	o.Receipt = append([]byte(nil), record.Receipt...)
	return o, nil
}

func (r *Resolver) LookupResolution(ctx context.Context, id string) (factory.OperationOutcome, error) {
	if r == nil || r.reviewer == nil || r.reviewer.background == nil || r.reviewer.Identities == nil {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "resolution_unavailable"}
	}
	lookup, err := r.reviewer.background.GetOperation(ctx, id)
	if err != nil {
		return factory.OperationOutcome{}, reviewCallError(err)
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	if lookup.Record == nil {
		return factory.OperationOutcome{}, errors.New("resolution lookup lacks native record")
	}
	return r.localOutcome(ctx, id, *lookup.Record)
}

func (r *Resolver) SubmitResolution(ctx context.Context, w factory.ResolveWork) (factory.OperationOutcome, error) {
	if w.Validate() != nil || w.NotAfter <= time.Now().Unix() {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "resolution_intent_expired"}
	}
	repository, actor, err := r.nativeScope(ctx, w)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	if err := r.reviewer.checkActor(ctx, w.Request.ActorID); err != nil {
		return factory.OperationOutcome{}, err
	}
	req := w.Request
	req.Repository, req.ActorID = repository, actor
	payload := extensions.ThreadResolvePayload{HeadRef: req.HeadRef, BaseRef: req.BaseRef, ExpectedHeadOID: req.HeadOID, ExpectedBaseOID: req.BaseOID, ConversationDigest: req.Digest, PullRequestNumber: req.PRNumber, HeadRepositoryID: req.Repository, RootCommentID: req.RootCommentID, Resolved: true}
	if extensions.ValidateThreadResolvePayload(req.Repository, payload) != nil {
		return factory.OperationOutcome{}, &factory.PublicationRefusal{Reason: "invalid_intent"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	intent := extensions.OperationIntent{OperationID: w.OperationID, ActorID: strconv.FormatInt(req.ActorID, 10), RepositoryID: strconv.FormatInt(req.Repository, 10), Kind: factory.OpThreadResolve, AuthorizationRevision: w.AuthRevision, ExpectedNativeRevision: w.NativeRevision, NotAfter: w.NotAfter, Payload: raw}
	record, err := r.reviewer.background.SubmitOperation(ctx, extensions.CredentialFile(r.reviewer.tokenFile), intent)
	if err != nil {
		observed, readErr := r.LookupResolution(ctx, w.OperationID)
		if readErr == nil && !observed.NotObserved {
			return observed, nil
		}
		return factory.OperationOutcome{}, reviewCallError(err)
	}
	return r.localOutcome(ctx, w.OperationID, record)
}

func (r *Resolver) CancelResolution(ctx context.Context, id string) (factory.OperationOutcome, error) {
	if r == nil || r.reviewer == nil || r.reviewer.background == nil || r.reviewer.Identities == nil {
		return factory.OperationOutcome{}, &factory.PublicationWait{Reason: "resolution_unavailable"}
	}
	record, err := r.reviewer.background.CancelOperation(ctx, id)
	if err != nil {
		return r.LookupResolution(ctx, id)
	}
	return r.localOutcome(ctx, id, record)
}

func (r *Resolver) ConfirmResolution(ctx context.Context, operation factory.DeliveryResolutionOperation) (bool, error) {
	w := operation.Work
	if w.Validate() != nil || !factory.ValidDigest(operation.ResolvedDigest) || operation.Outcome.Effect != factory.OpEffectCommitted || operation.Outcome.Completion != factory.OpCompletionComplete {
		return false, errors.New("resolution readback lacks its exact committed intent")
	}
	if operation.Outcome.OperationID != w.OperationID || operation.Outcome.Kind != factory.OpThreadResolve || operation.Outcome.ActorID != w.Request.ActorID || operation.Outcome.RepositoryID != w.Request.Repository {
		return false, errors.New("resolution readback differs from its canonical scope")
	}
	var receipt extensions.ThreadResolveReceipt
	if json.Unmarshal(operation.Outcome.Receipt, &receipt) != nil || receipt.RootCommentID != w.Request.RootCommentID || receipt.PullRequestNumber != w.Request.PRNumber || receipt.ConversationDigest != w.Request.Digest || !receipt.Resolved {
		return false, errors.New("resolution receipt differs from its accepted conversation")
	}
	repository, actor, err := r.nativeScope(ctx, w)
	if err != nil {
		return false, err
	}
	if err := r.reviewer.checkActor(ctx, w.Request.ActorID); err != nil {
		return false, err
	}
	reader := &BackgroundSnapshotReader{Client: r.reviewer.background, ActorID: strconv.FormatInt(actor, 10)}
	snapshot, err := BracketedRead(ctx, reader, extensions.CredentialFile(r.reviewer.tokenFile), SnapshotRequest{RepositoryID: strconv.FormatInt(repository, 10), PullNumber: strconv.FormatInt(w.Request.PRNumber, 10), Families: []SnapshotFamily{FamilyReviewThreads}, Limit: SnapshotPageLimit})
	if err != nil {
		return false, reviewReadError(err)
	}
	for _, thread := range snapshot.Threads.Items {
		if thread.RootID == strconv.FormatInt(w.Request.RootCommentID, 10) {
			return thread.Visible && thread.Complete && thread.Resolved && thread.ResolverID == strconv.FormatInt(actor, 10) && thread.Digest == operation.ResolvedDigest && thread.HeadCommit == w.Request.HeadOID, nil
		}
	}
	return false, nil
}

func (r *Resolver) nativeScope(ctx context.Context, w factory.ResolveWork) (int64, int64, error) {
	if r == nil || r.reviewer == nil {
		return 0, 0, &factory.PublicationWait{Reason: "resolution_unavailable"}
	}
	repository, actor, err := r.reviewer.nativeScope(ctx, w.Request.Repository, w.Request.ActorID)
	if err != nil {
		return 0, 0, err
	}
	if _, err := r.reviewer.Identities.NativeActor(ctx, collaboration.ActorID(w.PRAuthorID)); err != nil {
		return 0, 0, err
	}
	return repository, actor, nil
}

func (r *Resolver) localOutcome(ctx context.Context, id string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	o, err := resolveOutcome(id, record)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	return r.reviewer.Identities.LocalOperationOutcome(ctx, o)
}
