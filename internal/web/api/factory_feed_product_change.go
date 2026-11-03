package api

import (
	"context"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) posts(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.Posts {
	return factoryservice.Posts{
		Store: s.Store, Collaborators: s.FactoryApplication().Background().Collaborators(),
		Visibility: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
			request := r.WithContext(ctx)
			if actor != admitted || !s.extensionSessionCurrent(ctx, request, v) {
				return auth.ErrNativeSessionChanged
			}
			_, err := s.factoryVisibleRepository(request, v, repository)
			return err
		},
		WriteAccess: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
			request := r.WithContext(ctx)
			if actor != admitted || !s.extensionSessionCurrent(ctx, request, v) {
				return auth.ErrNativeSessionChanged
			}
			return s.requireFactoryOwner(request, v, repository)
		},
	}
}

func (s *API) postCommands(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.PostCommands {
	capture := func(ctx context.Context, actor collaboration.ActorID, id string, revision int64, authorization bool) error {
		request := r.WithContext(ctx)
		if actor != admitted || !s.extensionSessionCurrent(ctx, request, v) {
			return auth.ErrNativeSessionChanged
		}
		if authorization {
			return s.issueIntakeCapture(request, v, admitted).Capture(ctx, actor, id, revision)
		}
		return s.postContext(request, v, admitted).Capture(ctx, actor, id, revision)
	}
	return factoryservice.PostCommands{
		Posts: s.posts(r, v, admitted),
		Capture: factoryservice.PostCapture{
			Authorization: func(ctx context.Context, actor collaboration.ActorID, id string, revision int64) error {
				return capture(ctx, actor, id, revision, true)
			},
			Context: func(ctx context.Context, actor collaboration.ActorID, id string, revision int64) error {
				return capture(ctx, actor, id, revision, false)
			},
		},
	}
}
