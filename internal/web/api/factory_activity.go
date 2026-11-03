package api

import (
	"context"

	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) repositoryActivity(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.Activity {
	return factoryservice.Activity{Store: s.Store, WriteAccess: s.requestWriteAccess(r, v, admitted), Visibility: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (factoryservice.ActivityNavigation, error) {
		var empty factoryservice.ActivityNavigation
		if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
			return empty, auth.ErrNativeSessionChanged
		}
		access, err := s.factoryVisibleRepository(r.WithContext(ctx), v, int64(repository))
		if err != nil {
			return empty, err
		}
		owner, name := access.repository.Owner.Login, access.repository.Name
		if !validNativeLinkPart(owner) || !validNativeLinkPart(name) {
			return empty, factoryservice.ErrActivityNavigation
		}
		path := "/" + owner + "/" + name
		return factoryservice.ActivityNavigation{RepositoryPath: path, ItemURL: func(item factory.ActivityItem) string {
			if _, valid := auth.PositiveID(item.Issue); !valid {
				return ""
			}
			switch item.Kind {
			case "issue_created", "verified_merge":
				return path + "/issues/" + item.Issue
			case "pr_created":
				if _, valid := auth.PositiveID(item.PR); valid {
					return path + "/pulls/" + item.PR
				}
			}
			return ""
		}}, nil
	}}
}
