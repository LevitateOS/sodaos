package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
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

func TestEnrichFeedRecordsFillsRecordContent(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	st, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	id := strings.Repeat("1", 32)
	assignment := factory.Assignment{
		ID: id, AttemptRoot: id, PublicationAssignment: id,
		ProjectID: "p" + strings.Repeat("1", 24), Role: "coder", Phase: "correct",
		Acceptance: "d" + strings.Repeat("a", 24), Preparation: "f" + strings.Repeat("b", 24),
		Run: "run-1", RunHistory: []string{"run-1"}, Stage: factory.AssignmentFinished,
		Outcome: factory.Succeeded, Repository: 7, Issue: 3,
		CreatedUnix: 1700000000, FinishedUnix: 1700000600,
		Result: &factory.AssignmentResult{
			AssignmentID: id, RunID: strings.Repeat("a", 32), Phase: "correct",
			Status: "completed", Summary: "addressed both findings", RecordedUnix: 1700000600,
		},
	}
	rawJSON, err := json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data)
		VALUES($1,7,3,'run-1','finished',1,$2)`, id, string(rawJSON)); err != nil {
		t.Fatal(err)
	}
	entries := []factory.FeedEntry{
		{Kind: factory.FeedKindRecord, ID: id, Repository: 7, Seq: 2},
		{Kind: factory.FeedKindPost, ID: strings.Repeat("2", 32), Repository: 7, Seq: 1},
	}
	enrichFeedRecords(ctx, st, map[int64]string{7: "owner/repo"}, entries)
	record := entries[0]
	if record.Actor != "fixer" || record.Author != "Fixer" || record.Project != "owner/repo" ||
		record.At != 1700000600 || record.Status != "correct:finished:succeeded" ||
		record.Issue != 3 || record.Summary != "addressed both findings" {
		t.Fatalf("record entry: %+v", record)
	}
	// Comments pass through untouched: their enrichment stays with the
	// comment owner.
	if entries[1].Actor != "" || entries[1].Project != "" || entries[1].Summary != "" {
		t.Fatalf("post entry touched: %+v", entries[1])
	}
	// A nil store still fills the project from the visibility recheck
	// and leaves every other field empty instead of guessing.
	bare := []factory.FeedEntry{{Kind: factory.FeedKindRecord, ID: id, Repository: 7, Seq: 2}}
	enrichFeedRecords(ctx, nil, map[int64]string{7: "owner/repo"}, bare)
	if bare[0].Project != "owner/repo" || bare[0].Actor != "" || bare[0].Status != "" {
		t.Fatalf("bare entry: %+v", bare[0])
	}
}
