package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/web/auth"

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

var errStoreReservation = errors.New("could not inspect repository reservation")

func isValidSearchTerm(q string) bool {
	return len(q) <= 200 && utf8.ValidString(q) && strings.IndexFunc(q, unicode.IsControl) < 0
}

func parseRepositoryPage(pageStr string) (int, bool) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || strconv.Itoa(page) != pageStr || page < 1 || page > forgejo.RepositoryPageLimit {
		return 0, false
	}
	return page, true
}

func parseRepositoryQuery(rawQuery string) (string, int, bool) {
	if len(rawQuery) > 2048 {
		return "", 0, false
	}
	q, err := url.ParseQuery(rawQuery)
	if err != nil || len(q) != 2 || len(q["page"]) != 1 || len(q["q"]) != 1 {
		return "", 0, false
	}
	page, ok := parseRepositoryPage(q.Get("page"))
	if !ok {
		return "", 0, false
	}
	query := q.Get("q")
	if !isValidSearchTerm(query) {
		return "", 0, false
	}
	return query, page, true
}

func parseNativeRepositoryQuery(rawQuery string) (string, string, bool) {
	if len(rawQuery) > 8192 {
		return "", "", false
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil || !nativeRepositoryQueryFields(query) {
		return "", "", false
	}
	search := query.Get("q")
	cursor := query.Get("cursor")
	if !isValidSearchTerm(search) || len(cursor) > 4096 {
		return "", "", false
	}
	return search, cursor, true
}

func nativeRepositoryQueryFields(query url.Values) bool {
	if len(query) < 1 || len(query) > 2 || len(query["q"]) != 1 || len(query["cursor"]) > 1 {
		return false
	}
	_, cursor := query["cursor"]
	return len(query) == 1 || cursor
}

func (s *API) authorizeRepositorySearch(ctx context.Context, r *http.Request, v store.Session) (store.Grant, forgejo.User, error) {
	grant, err := s.Auth.UserGrant(r, v)
	if err != nil {
		return store.Grant{}, forgejo.User{}, err
	}
	if !forgejo.HasScope(grant.Scopes, "read:user") || !forgejo.HasScope(grant.Scopes, "read:repository") {
		return store.Grant{}, forgejo.User{}, auth.ErrRepositoryConsent
	}
	actor, err := s.Forgejo.Current(ctx, grant.Access)
	if err != nil {
		return store.Grant{}, forgejo.User{}, err
	}
	if actor.ID != v.User.ID || actor.Login == "" {
		return store.Grant{}, forgejo.User{}, auth.ErrProviderIdentity
	}
	return grant, actor, nil
}

func isValidRepositoryIdentity(repo forgejo.Repository, actorID int64) (bool, error) {
	if repo.Owner.ID != actorID {
		return false, nil
	}
	if !auth.ValidRepositoryPart(repo.Owner.Login) || !auth.ValidRepositoryPart(repo.Name) {
		return false, forgejo.ErrInvalidResponse
	}
	return true, nil
}

func (s *API) resolveRepositoryChoice(ctx context.Context, access string, candidateID, actorID int64) (*repositoryChoice, error) {
	repo, err := s.Forgejo.RepositoryByID(ctx, access, candidateID)
	if err != nil {
		var denied *forgejo.HTTPError
		if errors.As(err, &denied) && (denied.Status == 403 || denied.Status == 404) {
			return nil, nil
		}
		return nil, err
	}
	valid, err := isValidRepositoryIdentity(repo, actorID)
	if err != nil || !valid {
		return nil, err
	}
	choice := &repositoryChoice{ID: strconv.FormatInt(repo.ID, 10), Owner: repo.Owner.Login, Name: repo.Name}
	p, err := s.Store.ProjectByRepository(ctx, repo.ID)
	if errors.Is(err, store.ErrNotFound) {
		choice.CanCreate = true
		return choice, nil
	}
	if err != nil {
		return nil, errStoreReservation
	}
	choice.Project = &repositoryProject{ID: p.ID, Provisioned: p.Ready}
	return choice, nil
}

func (s *API) collectRepositoryChoices(ctx context.Context, access string, repos []forgejo.Repository, actorID int64) ([]repositoryChoice, error) {
	items := make([]repositoryChoice, 0, len(repos))
	for _, candidate := range repos {
		choice, err := s.resolveRepositoryChoice(ctx, access, candidate.ID, actorID)
		if err != nil {
			return nil, err
		}
		if choice != nil {
			items = append(items, *choice)
		}
	}
	return items, nil
}

func (s *API) collectNativeRepositoryChoices(ctx context.Context, items []extensions.Repository, actor extensions.Actor) ([]repositoryChoice, error) {
	choices := make([]repositoryChoice, 0, len(items))
	for _, repo := range items {
		id, valid := auth.PositiveID(repo.ID)
		if !valid || repo.Owner != actor.Username || !auth.ValidRepositoryPart(repo.Owner) || !auth.ValidRepositoryPart(repo.Name) {
			return nil, forgejo.ErrInvalidResponse
		}
		choice := repositoryChoice{ID: repo.ID, Owner: repo.Owner, Name: repo.Name}
		project, err := s.Store.ProjectByRepository(ctx, id)
		if errors.Is(err, store.ErrNotFound) {
			choice.CanCreate = true
		} else if err != nil {
			return nil, errStoreReservation
		} else {
			choice.Project = &repositoryProject{ID: project.ID, Provisioned: project.Ready}
		}
		choices = append(choices, choice)
	}
	return choices, nil
}

func (s *API) verifyRepositorySession(ctx context.Context, r *http.Request, v store.Session) bool {
	return ctx.Err() == nil && s.extensionSessionCurrent(ctx, r, v)
}

func handleRepositoryChoicesError(w http.ResponseWriter, err error) {
	if errors.Is(err, errStoreReservation) {
		auth.JSONError(w, 503, "store_unavailable", "Could not inspect repository reservation.")
	} else {
		auth.ProviderError(w, err)
	}
}

func handleRepositorySessionFailure(w http.ResponseWriter, ctx context.Context) {
	if ctx.Err() != nil {
		auth.JSONError(w, 503, "repositories_unavailable", "Repository search exceeded its time budget.")
	} else {
		auth.JSONError(w, 401, "unauthenticated", "Soda context changed.")
	}
}

// Human-owner discovery only. Shared projects retain the separate Spaces read
// authority. This endpoint neither inspects nor mutates native project state.
func (s *API) apiRepositories(w http.ResponseWriter, r *http.Request, v store.Session) {
	if authority, native := requestExtensionAuthority(r); native {
		s.apiNativeRepositories(w, r, v, authority)
		return
	}
	s.apiProviderRepositories(w, r, v)
}

func (s *API) apiNativeRepositories(w http.ResponseWriter, r *http.Request, v store.Session, authority extensions.Authority) {
	query, cursor, ok := parseNativeRepositoryQuery(r.URL.RawQuery)
	if !ok {
		auth.JSONError(w, http.StatusBadRequest, "invalid_query", "Provide a search term and one bounded cursor.")
		return
	}
	select {
	case s.RepositorySlots <- struct{}{}:
		defer func() { <-s.RepositorySlots }()
	default:
		auth.JSONError(w, http.StatusServiceUnavailable, "repositories_unavailable", "Repository search is busy.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	page, err := authority.Native().SearchOwnedRepositories(ctx, query, cursor, forgejo.RepositoryPageSize)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "repositories_unavailable", "Native repository search is unavailable.")
		return
	}
	if len(page.Items) > forgejo.RepositoryPageSize || len(page.NextCursor) > 4096 {
		auth.JSONError(w, http.StatusServiceUnavailable, "repositories_unavailable", "Native repository search returned an invalid page.")
		return
	}
	items, err := s.collectNativeRepositoryChoices(ctx, page.Items, authority.Actor)
	if err != nil {
		handleRepositoryChoicesError(w, err)
		return
	}
	if ctx.Err() != nil || !s.extensionSessionCurrent(ctx, r, v) {
		handleRepositorySessionFailure(w, ctx)
		return
	}
	auth.JSONResponse(w, http.StatusOK, struct {
		Items      []repositoryChoice `json:"items"`
		NextCursor string             `json:"next_cursor,omitempty"`
	}{items, page.NextCursor})
}

func (s *API) apiProviderRepositories(w http.ResponseWriter, r *http.Request, v store.Session) {
	query, page, ok := parseRepositoryQuery(r.URL.RawQuery)
	if !ok {
		auth.JSONError(w, 400, "invalid_query", "Provide q and one bounded page.")
		return
	}
	select {
	case s.RepositorySlots <- struct{}{}:
		defer func() { <-s.RepositorySlots }()
	default:
		auth.JSONError(w, 503, "repositories_unavailable", "Repository search is busy.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	grant, actor, err := s.authorizeRepositorySearch(ctx, r, v)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	repos, err := s.Forgejo.SearchOwnedRepositories(ctx, grant.Access, actor.ID, query, page)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	items, err := s.collectRepositoryChoices(ctx, grant.Access, repos, actor.ID)
	if err != nil {
		handleRepositoryChoicesError(w, err)
		return
	}
	if !s.verifyRepositorySession(ctx, r, v) {
		handleRepositorySessionFailure(w, ctx)
		return
	}
	auth.JSONResponse(w, 200, struct {
		Items   []repositoryChoice `json:"items"`
		Page    int                `json:"page"`
		More    bool               `json:"more"`
		Limited bool               `json:"limited"`
	}{
		Items:   items,
		Page:    page,
		More:    len(repos) == forgejo.RepositoryPageSize && page < forgejo.RepositoryPageLimit,
		Limited: len(repos) == forgejo.RepositoryPageSize && page == forgejo.RepositoryPageLimit,
	})
}
