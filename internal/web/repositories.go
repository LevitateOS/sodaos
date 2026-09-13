package web

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

type repositoryChoice struct {
	ID        string             `json:"id"`
	Owner     string             `json:"owner"`
	Name      string             `json:"name"`
	CanCreate bool               `json:"can_create"`
	Project   *repositoryProject `json:"project"`
}
type repositoryProject struct {
	ID          string `json:"id"`
	Provisioned bool   `json:"provisioned"`
}

// Human-owner discovery only. Shared projects retain the separate Spaces read
// authority. This endpoint neither inspects nor mutates native project state.
func (s *Server) apiRepositories(w http.ResponseWriter, r *http.Request, v store.Session) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	page, pageErr := strconv.Atoi(q.Get("page"))
	if err != nil || len(r.URL.RawQuery) > 2048 || len(q) != 2 || len(q["page"]) != 1 || len(q["q"]) != 1 || pageErr != nil || strconv.Itoa(page) != q.Get("page") || page < 1 || page > forgejo.RepositoryPageLimit || len(q.Get("q")) > 200 || !utf8.ValidString(q.Get("q")) || strings.IndexFunc(q.Get("q"), unicode.IsControl) >= 0 {
		jsonError(w, 400, "invalid_query", "Provide q and one bounded page.")
		return
	}
	select {
	case s.repositorySlots <- struct{}{}:
		defer func() { <-s.repositorySlots }()
	default:
		jsonError(w, 503, "repositories_unavailable", "Repository search is busy.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	grant, err := s.userGrant(r, v)
	if err != nil {
		providerError(w, err)
		return
	}
	if !forgejo.HasScope(grant.Scopes, "read:user") || !forgejo.HasScope(grant.Scopes, "read:repository") {
		providerError(w, errRepositoryConsent)
		return
	}
	actor, err := s.Forgejo.Current(ctx, grant.Access)
	if err != nil {
		providerError(w, err)
		return
	}
	if actor.ID != v.User.ID || actor.Login == "" {
		providerError(w, errProviderIdentity)
		return
	}
	repos, err := s.Forgejo.SearchOwnedRepositories(ctx, grant.Access, actor.ID, q.Get("q"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	result := struct {
		Items   []repositoryChoice `json:"items"`
		Page    int                `json:"page"`
		More    bool               `json:"more"`
		Limited bool               `json:"limited"`
	}{Items: []repositoryChoice{}, Page: page, More: len(repos) == forgejo.RepositoryPageSize && page < forgejo.RepositoryPageLimit, Limited: len(repos) == forgejo.RepositoryPageSize && page == forgejo.RepositoryPageLimit}
	for _, candidate := range repos {
		// Re-resolve stable identity after search, including transfer/departure races.
		repo, e := s.Forgejo.RepositoryByID(ctx, grant.Access, candidate.ID)
		if e != nil {
			var denied *forgejo.HTTPError
			if errors.As(e, &denied) && (denied.Status == 403 || denied.Status == 404) {
				continue
			}
			providerError(w, e)
			return
		}
		if repo.Owner.ID != actor.ID {
			continue
		}
		if !validRepositoryPart(repo.Owner.Login) || !validRepositoryPart(repo.Name) {
			providerError(w, forgejo.ErrInvalidResponse)
			return
		}
		item := repositoryChoice{ID: strconv.FormatInt(repo.ID, 10), Owner: repo.Owner.Login, Name: repo.Name}
		p, e := s.Store.ProjectByRepository(ctx, repo.ID)
		if errors.Is(e, sql.ErrNoRows) {
			item.CanCreate = true
		} else if e != nil {
			jsonError(w, 503, "store_unavailable", "Could not inspect repository reservation.")
			return
		} else {
			item.Project = &repositoryProject{ID: p.ID, Provisioned: p.Ready}
		}
		result.Items = append(result.Items, item)
	}
	if ctx.Err() != nil {
		jsonError(w, 503, "repositories_unavailable", "Repository search exceeded its time budget.")
		return
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || s.requireCurrentSession(ctx, cookie.Value, v) != nil {
		jsonError(w, 401, "unauthenticated", "Soda context changed.")
		return
	}
	jsonResponse(w, 200, result)
}
