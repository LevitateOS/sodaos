package forgejo

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	RepositoryPageSize  = 12
	RepositoryPageLimit = 100
)

// SearchOwnedRepositories uses Forgejo 15.0.7's actor-reduced native search.
// Pagination is upstream-owned; never search a locally truncated repository list.
func validOwnedSearch(owner int64, query string, page int) bool {
	if owner <= 0 || page < 1 || page > RepositoryPageLimit || len(query) > 200 {
		return false
	}
	return utf8.ValidString(query) && strings.IndexFunc(query, unicode.IsControl) < 0
}

func validOwnedSearchPayload(result *bool, data *[]Repository) bool {
	return result != nil && *result && data != nil && len(*data) <= RepositoryPageSize
}

func validOwnedRepository(repo Repository, owner int64, seen map[int64]bool) bool {
	if repo.ID <= 0 || seen[repo.ID] || repo.Owner.ID != owner {
		return false
	}
	if !repositoryPart(repo.Owner.Login) || !repositoryPart(repo.Name) {
		return false
	}
	return repo.FullName == repo.Owner.Login+"/"+repo.Name
}

func (c *Client) SearchOwnedRepositories(ctx context.Context, token string, owner int64, query string, page int) ([]Repository, error) {
	if !validOwnedSearch(owner, query, page) {
		return nil, ErrInvalidResponse
	}
	q := url.Values{"uid": {strconv.FormatInt(owner, 10)}, "exclusive": {"true"}, "private": {"true"}, "q": {query}, "page": {strconv.Itoa(page)}, "limit": {strconv.Itoa(RepositoryPageSize)}, "sort": {"id"}, "order": {"asc"}}
	var result struct {
		OK   *bool         `json:"ok"`
		Data *[]Repository `json:"data"`
	}
	if err := c.request(ctx, "GET", "/repos/search?"+q.Encode(), token, nil, &result); err != nil {
		return nil, err
	}
	if !validOwnedSearchPayload(result.OK, result.Data) {
		return nil, ErrInvalidResponse
	}
	seen := make(map[int64]bool)
	for _, repo := range *result.Data {
		if !validOwnedRepository(repo, owner, seen) {
			return nil, ErrInvalidResponse
		}
		seen[repo.ID] = true
	}
	return *result.Data, nil
}

func repositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && utf8.ValidString(value) && !strings.ContainsAny(value, "/\\") && strings.IndexFunc(value, unicode.IsControl) < 0
}
