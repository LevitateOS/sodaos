package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/access"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// factoryRepositoryAuthority translates current native authority into the shared
// admission contract. Sessions and native callbacks stay inside this adapter;
// a desktop adapter must supply its own account/connection authority instead.
func (s *API) factoryRepositoryAuthority(r *http.Request, v store.Session, actor collaboration.ActorID) factoryservice.RepositoryAuthority {
	port := factoryservice.RepositoryAuthority{Actor: actor, Source: access.FountainSession}
	if s == nil || s.FountainIdentities == nil {
		return port
	}
	port.Host = s.FountainIdentities.HostID()
	port.Current = func(ctx context.Context, repository collaboration.RepositoryID, permission access.Permission) (factoryservice.RepositoryAuthorization, error) {
		if !s.extensionSessionCurrent(ctx, r, v) {
			return factoryservice.RepositoryAuthorization{}, auth.ErrNativeSessionChanged
		}
		nativeActor, err := s.FountainIdentities.NativeActor(ctx, actor)
		if err != nil {
			return factoryservice.RepositoryAuthorization{}, err
		}
		if nativeActor != strconv.FormatInt(v.User.ID, 10) {
			return factoryservice.RepositoryAuthorization{}, auth.ErrNativeSessionChanged
		}
		request := r.WithContext(ctx)
		var name string
		switch permission {
		case access.Read:
			visible, err := s.factoryVisibleRepository(request, v, int64(repository))
			if errors.Is(err, extensions.ErrRepositoryNotVisible) {
				return factoryservice.RepositoryAuthorization{}, factoryservice.ErrRepositoryHidden
			}
			if err != nil {
				return factoryservice.RepositoryAuthorization{}, err
			}
			name = visible.repository.FullName
		case access.Configure:
			if err := s.requireFactoryOwner(request, v, int64(repository)); err != nil {
				return factoryservice.RepositoryAuthorization{}, err
			}
		case access.Collaborate:
			if _, err := s.factoryExecutionRepository(request, v, int64(repository)); err != nil {
				return factoryservice.RepositoryAuthorization{}, err
			}
		default:
			return factoryservice.RepositoryAuthorization{}, access.ErrDenied
		}
		// Generation can change while native permission reads are in flight.
		if !s.extensionSessionCurrent(ctx, r, v) {
			return factoryservice.RepositoryAuthorization{}, auth.ErrNativeSessionChanged
		}
		now := time.Now()
		// This bounds one fresh result, never a reusable session or permission
		// cache. Every shared Check invokes this current-authority callback.
		expires := now.Add(time.Minute)
		if deadline, ok := ctx.Deadline(); ok && deadline.Before(expires) {
			expires = deadline
		}
		admitted, err := access.Admit(access.ActorEvidence{
			Source: access.FountainSession, Host: port.Host, Repository: repository, Actor: actor,
			Authority: "fountain-session", Revision: collaboration.Revision(factory.MCPInputDigest([]byte(v.ContextID))),
			Permissions: []access.Permission{permission}, ObservedAt: now, ExpiresAt: expires,
		}, now)
		if err != nil {
			return factoryservice.RepositoryAuthorization{}, err
		}
		return factoryservice.RepositoryAuthorization{Actor: admitted, Name: name}, nil
	}
	return port
}

// factoryRepositoryVisibility is the canonical current-read port shared by
// catalogue and complete feed assembly. Only confirmed native denial is hidden;
// changed sessions and unknown provider access stay errors.
func (s *API) factoryRepositoryVisibility(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.RepositoryVisibility {
	return s.factoryRepositoryAuthority(r, v, admitted).Visibility
}
