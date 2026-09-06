package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaginationUsesNativeCapAndNeverFollowsURLs(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Link", `<https://untrusted.invalid/arbitrary?page=2&limit=1>; rel="next",<https://forgejo.example/api/v1/user/repos?page=9>; rel="last"`)
		w.Header().Set("X-Total-Count", "9007199254740993")
		fmt.Fprint(w, `[{"id":7,"name":"demo"}]`)
	}))
	defer server.Close()
	items, metadata, err := New(server.URL).MyRepositories(context.Background(), "acting-token", 1)
	if err != nil || len(items) != 1 || metadata.NextPage == nil || *metadata.NextPage != 2 || metadata.Total == nil || *metadata.Total != "9007199254740993" || requests != 1 {
		t.Fatal("invalid metadata", err)
	}
}
func TestPaginationRejectsMalformedNextAndTotal(t *testing.T) {
	for _, headers := range []http.Header{
		{"Link": {`<https://forgejo.example/?page=40>; rel="next"`}},
		{"Link": {`<https://forgejo.example/?page=2>; rel="next",<https://forgejo.example/?page=2>; rel="next"`}},
		{"X-Total-Count": {"not-a-count"}},
		{"Link": {"garbage"}},
	} {
		if _, err := pagination(headers, 1); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	metadata, err := pagination(http.Header{"X-Total-Count": {"1"}}, 1)
	if err != nil || metadata.NextPage != nil {
		t.Fatal("invented final-page continuation")
	}
}
