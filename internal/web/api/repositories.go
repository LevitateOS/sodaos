package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
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
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		auth.JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
		return
	}
	s.apiNativeRepositories(w, r, v, authority)
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
