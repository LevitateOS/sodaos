package forgejo

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Pagination struct {
	NextPage *int    `json:"next_page"`
	Total    *string `json:"total"`
}

// Read only metadata from Forgejo's genAPILinks/SetLinkHeader contract. Never
// follow a Link URL: subsequent requests reconstruct the same explicit route.
func pagination(headers http.Header, page int) (Pagination, error) {
	var result Pagination
	if total := headers.Get("X-Total-Count"); total != "" {
		n, err := strconv.ParseInt(total, 10, 64)
		if err != nil || n < 0 {
			return result, ErrInvalidResponse
		}
		canonical := strconv.FormatInt(n, 10)
		result.Total = &canonical
	}
	for _, header := range headers.Values("Link") {
		for _, entry := range strings.Split(header, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			target, relation, ok := strings.Cut(entry, ">;")
			if !ok || !strings.HasPrefix(target, "<") {
				return Pagination{}, ErrInvalidResponse
			}
			if strings.TrimSpace(relation) != `rel="next"` {
				continue
			}
			if result.NextPage != nil {
				return Pagination{}, ErrInvalidResponse
			}
			u, err := url.Parse(strings.TrimPrefix(target, "<"))
			if err != nil {
				return Pagination{}, ErrInvalidResponse
			}
			next, err := strconv.Atoi(u.Query().Get("page"))
			if err != nil || next != page+1 || next > 1000000 {
				return Pagination{}, ErrInvalidResponse
			}
			result.NextPage = &next
		}
	}
	return result, nil
}
