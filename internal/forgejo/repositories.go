package forgejo

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const RepositoryPageSize = 12
const RepositoryPageLimit = 100

// SearchOwnedRepositories uses Forgejo 15.0.7's actor-reduced native search.
// Pagination is upstream-owned; never search a locally truncated repository list.
func (c *Client) SearchOwnedRepositories(ctx context.Context, token string, owner int64, query string, page int) ([]Repository, error) {
	if owner <= 0 || page < 1 || page > RepositoryPageLimit || len(query) > 200 || !utf8.ValidString(query) || strings.IndexFunc(query, unicode.IsControl) >= 0 {
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
	if result.OK == nil || !*result.OK || result.Data == nil || len(*result.Data) > RepositoryPageSize {
		return nil, ErrInvalidResponse
	}
	seen := make(map[int64]bool)
	for _, repo := range *result.Data {
		if repo.ID <= 0 || seen[repo.ID] || repo.Owner.ID != owner || !repositoryPart(repo.Owner.Login) || !repositoryPart(repo.Name) || repo.FullName != repo.Owner.Login+"/"+repo.Name {
			return nil, ErrInvalidResponse
		}
		seen[repo.ID] = true
	}
	return *result.Data, nil
}

func repositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && utf8.ValidString(value) && !strings.ContainsAny(value, "/\\") && strings.IndexFunc(value, unicode.IsControl) < 0
}
