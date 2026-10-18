package api

import (
	"net/http/httptest"
	"testing"
)

func TestParseFeedQueryParams(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/factory/feed?repositories=7,9,7&kinds=record&limit=10&cursor=v1:42", nil)
	query, err := parseFeedQueryParams(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(query.Repositories) != 2 || query.Repositories[0] != 7 || query.Repositories[1] != 9 {
		t.Fatalf("scope: %+v", query.Repositories)
	}
	if len(query.Kinds) != 1 || query.Kinds[0] != "record" || query.Limit != 10 || query.Cursor != "v1:42" {
		t.Fatalf("query: %+v", query)
	}
	r = httptest.NewRequest("GET", "/api/factory/feed?repositories=7", nil)
	query, err = parseFeedQueryParams(r)
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != 50 {
		t.Fatalf("default limit = %d, want 50", query.Limit)
	}
	for _, target := range []string{
		"/api/factory/feed",
		"/api/factory/feed?repositories=",
		"/api/factory/feed?repositories=0",
		"/api/factory/feed?repositories=abc",
		"/api/factory/feed?repositories=7&limit=0",
		"/api/factory/feed?repositories=7&limit=9999",
		"/api/factory/feed?repositories=7&limit=abc",
		"/api/factory/feed?repositories=7&cursor=v9:1",
		"/api/factory/feed?repositories=7&kinds=digest",
	} {
		r = httptest.NewRequest("GET", target, nil)
		if _, err := parseFeedQueryParams(r); err == nil {
			t.Fatalf("params admitted: %s", target)
		}
	}
}
