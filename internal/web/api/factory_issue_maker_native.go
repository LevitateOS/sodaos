package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/forgejo"
)

func (s *API) issueMakerNative() factoryservice.IssueIntakeWriter {
	if s.IssueMakerNative != nil {
		return s.IssueMakerNative
	}
	if s.ExtensionManager != nil {
		return &forgejo.FountainIssueIntake{Identities: s.FountainIdentities, Background: s.ExtensionManager}
	}
	return nil
}

// issueIntakeCapture supplies only Fountain issuance/encoding. Canonical source
// checks, retained-proof reuse and durable capture belong to the shared service.
func (s *API) issueIntakeCapture(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.IssueIntakeCapture {
	return factoryservice.IssueIntakeCapture{Posts: s.posts(r, v, admitted), Effects: s.capturedIssueEffects(),
		Authorize: func(ctx context.Context, request factoryservice.IssueIntakeRequest) (factory.IssueIntakeScope, json.RawMessage, error) {
			var empty factory.IssueIntakeScope
			repository, err := s.FountainIdentities.NativeRepository(ctx, request.Repository)
			if err != nil {
				return empty, nil, err
			}
			nativeActor, err := s.FountainIdentities.NativeActor(ctx, request.Actor)
			if err != nil {
				return empty, nil, err
			}
			authority, ok := requestExtensionAuthority(r.WithContext(ctx))
			if !ok || authority.Actor.ID != nativeActor {
				return empty, nil, errors.New("native issue-write admission is unavailable for this author")
			}
			grant, err := authority.AuthorizeIssueIntake(ctx, repository, sdk.IssueIntakeAuthorizationRequest{Source: request.Source, Revision: request.Revision, Digest: request.Digest})
			if err != nil {
				return empty, nil, err
			}
			return s.FountainIdentities.EncodeIssueIntakeGrant(ctx, grant)
		}}
}

// capturedIssueEffects composes neutral source/journal behavior with the native
// proof decoder and conditional writer. No session enters the shared service.
func (s *API) capturedIssueEffects() *factoryservice.CapturedIssueEffects {
	effects := &factoryservice.CapturedIssueEffects{Store: s.Store, Writer: s.issueMakerNative()}
	if s.FountainIdentities != nil {
		effects.Decode = s.FountainIdentities.DecodeIssueIntakeGrant
	}
	return effects
}
