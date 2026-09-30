package auth

import (
	"net/url"
)

func parseForgejoKeysPage(rawQuery string) (int64, bool) {
	query, err := url.ParseQuery(rawQuery)
	page := int64(1)
	valid := true
	if query.Has("page") {
		page, valid = PositiveID(query.Get("page"))
	}
	if err != nil || !valid || len(query) > 1 || len(query["page"]) > 1 || (len(query) == 1 && !query.Has("page")) || page > 8 {
		return 0, false
	}
	return page, true
}

type profileKeyView struct {
	developmentKeyView
	Title string `json:"title"`
}
