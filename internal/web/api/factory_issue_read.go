package api

import (
	"context"

	"net/http"
	"strconv"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) issueReads(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.Issues {
	controls := s.workControls(r, v, admitted)
	return factoryservice.Issues{Store: s.Store, Visibility: controls.Visibility, Authority: controls.Authority, Membership: controls.Membership,
		Status: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, issue int64) (control.AcceptanceValidity, error) {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return control.AcceptanceValidity{}, auth.ErrNativeSessionChanged
			}
			if s.Coordinator == nil {
				return control.AcceptanceValidity{}, factoryservice.ErrIssueUnavailable
			}
			return s.Coordinator.AcceptanceStatus(ctx, int64(repository), strconv.FormatInt(issue, 10))
		},
		Navigation: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, issue, pr int64) (*factoryservice.IssueLinks, error) {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return nil, auth.ErrNativeSessionChanged
			}
			access, err := s.factoryVisibleRepository(r.WithContext(ctx), v, int64(repository))
			if err != nil || !access.native {
				return nil, nil
			}
			owner, name := access.repository.Owner.Login, access.repository.Name
			if !validNativeLinkPart(owner) || !validNativeLinkPart(name) {
				return nil, nil
			}
			base := "/" + owner + "/" + name
			links := &factoryservice.IssueLinks{Issue: base + "/issues/" + strconv.FormatInt(issue, 10)}
			if pr > 0 {
				links.Pull = base + "/pulls/" + strconv.FormatInt(pr, 10)
			}
			return links, nil
		},
	}
}
