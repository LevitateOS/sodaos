package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

// EncodeIssueIntakeGrant accepts the result of an admitted native authorization
// call. It does not authenticate browser-supplied proof. Retained provider JSON
// keeps its existing shape; only this adapter interprets its claims and bearer.
func (s *FountainIdentities) EncodeIssueIntakeGrant(ctx context.Context, grant sdk.IssueIntakeGrant) (factory.IssueIntakeScope, json.RawMessage, error) {
	raw, err := json.Marshal(grant)
	if err != nil {
		return factory.IssueIntakeScope{}, nil, err
	}
	scope, err := s.DecodeIssueIntakeGrant(ctx, raw)
	return scope, raw, err
}

// DecodeIssueIntakeGrant reads metadata from a retained provider envelope; it
// neither authenticates a bearer nor grants authority. The native issuer/write
// endpoint remains authoritative for admission, expiry and signature checks.
func (s *FountainIdentities) DecodeIssueIntakeGrant(ctx context.Context, raw json.RawMessage) (factory.IssueIntakeScope, error) {
	var grant sdk.IssueIntakeGrant
	var scope factory.IssueIntakeScope
	if s == nil || s.registry == nil || len(raw) == 0 || len(raw) > 8192 || json.Unmarshal(raw, &grant) != nil || grant.Token == "" || len(grant.Token) > 4096 ||
		!collaboration.ValidText(grant.Claims.Installation, 128) {
		return scope, ErrInvalidResponse
	}
	actor, actorErr := strconv.ParseInt(grant.Claims.Actor, 10, 64)
	repository, repoErr := strconv.ParseInt(grant.Claims.Repository, 10, 64)
	if actorErr != nil || repoErr != nil || strconv.FormatInt(actor, 10) != grant.Claims.Actor || strconv.FormatInt(repository, 10) != grant.Claims.Repository {
		return scope, ErrInvalidResponse
	}
	if actor <= 0 || repository <= 0 {
		return scope, ErrInvalidResponse
	}
	// A retained provider proof can only name already-observed subjects on
	// this adopted host. Decoding cannot allocate or adopt a namespace.
	localActor, err := s.registry.CollaborationActorByExternalID(ctx, s.host, grant.Claims.Actor)
	if err != nil {
		return scope, err
	}
	localRepository, err := s.registry.CollaborationRepositoryByExternalID(ctx, s.host, grant.Claims.Repository)
	if err != nil {
		return scope, err
	}
	scope = factory.IssueIntakeScope{Source: grant.Claims.Source, Revision: grant.Claims.Revision, Actor: localActor.ID, Repository: localRepository.ID, Digest: grant.Claims.Digest, Expires: grant.Claims.Expires}
	if scope.Validate() != nil {
		return factory.IssueIntakeScope{}, ErrInvalidResponse
	}
	return scope, nil
}

// FountainIssueIntake binds opaque native proofs and receipts to the adopted
// identity namespace while sharing the existing private service transport.
type FountainIssueIntake struct {
	Identities *FountainIdentities
	Background *ServiceBackground
}

// ApplyIssueIntake presents only the request-scoped native grant. It does not
// use, expand or enroll a Factory bot's repository permissions.
func (c *FountainIssueIntake) ApplyIssueIntake(ctx context.Context, proof json.RawMessage, mutation factory.IssueMakerMutation) (factory.IssueMakerReceipt, error) {
	var result factory.IssueMakerReceipt
	if c == nil || c.Background == nil {
		return result, errors.New("native issue intake is unavailable")
	}
	scope, err := c.Identities.DecodeIssueIntakeGrant(ctx, proof)
	if err != nil || mutation.Validate() != nil {
		return result, errors.New("invalid native issue intake")
	}
	var grant sdk.IssueIntakeGrant
	if err = json.Unmarshal(proof, &grant); err != nil {
		return result, ErrInvalidResponse
	}
	request := sdk.IssueIntakeMutation{Grant: grant.Token, Action: mutation.Action, Slot: mutation.Slot, Step: mutation.Step, Title: mutation.Title, Body: mutation.Body, ExpectedTitle: mutation.ExpectedTitle, ExpectedBodyDigest: mutation.ExpectedBodyDigest}
	if !sdk.ValidIssueIntakeMutation(request) {
		return result, errors.New("invalid native issue intake")
	}
	var receipt sdk.IssueIntakeReceipt
	if err := c.Background.call(ctx, sdk.BackgroundIssueIntakePath, request, &receipt); err != nil {
		return result, err
	}
	if receipt.Index <= 0 || receipt.IssueID <= 0 || receipt.Slot != request.Slot || receipt.Title != request.Title || receipt.BodyDigest != ContentDigest(request.Body) {
		return result, ErrInvalidResponse
	}
	return factory.IssueMakerReceipt{Issue: collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: scope.Repository, ExternalID: strconv.FormatInt(receipt.IssueID, 10), Number: receipt.Index}, Slot: receipt.Slot, Title: receipt.Title, BodyDigest: receipt.BodyDigest}, nil
}
