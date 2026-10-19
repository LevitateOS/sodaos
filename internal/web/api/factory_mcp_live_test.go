package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFactoryLiveMCPActivatesAndPublishesBoundFeed(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	st, dsn, cleanup, err := store.OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cap := fakeCatalogCapability()
	cap.ID = factory.NewID()
	cap.RunID = cap.ID
	cap.State = factory.MCPCapabilityStaged
	cap.DeadlineUnix = time.Now().Add(time.Hour).Unix()
	prepID := "f" + strings.Repeat("1", 24)
	assignment := factory.Assignment{ID: cap.AssignmentID, Repository: cap.Repository, ProjectID: cap.Project, Issue: 42, Role: project.RoleCoder, Phase: project.ExecutionImplement, Preparation: prepID, Run: cap.RunID, Stage: factory.AssignmentAssigned, SourceCommit: strings.Repeat("a", 40)}
	if err := st.UpsertUser(ctx, store.User{ID: 7, Login: "worker-test"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, store.Project{ID: cap.Project, Name: "Worker test", RepositoryID: 7, OwnerID: 7, Repository: "owner/repo"}); err != nil {
		t.Fatal(err)
	}
	write := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := json.Marshal(assignment)
	write(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,42,$2,'assigned',1,$3)`, assignment.ID, cap.RunID, string(raw))
	prep := project.StoredPreparation{Preparation: project.Preparation{ID: prepID, Project: cap.Project, Role: project.RoleCoder, SetupDigest: cap.PreparationDigest}}
	raw, _ = json.Marshal(prep)
	write(`INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data) VALUES($1,$2,$3,1,'','',$4)`, prepID, cap.Project, project.RoleCoder, string(raw))
	write(`INSERT INTO factory_dispatch(repository,revision,open,data) VALUES(7,1,true,'{}')`)
	raw, _ = json.Marshal(factory.Run{ID: cap.RunID, ProjectID: cap.Project, Role: project.RoleCoder})
	write(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, cap.RunID, string(raw))
	if err := st.StageMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	var live atomic.Bool
	live.Store(true)
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/factory-inspect" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(project.FactoryState{ID: cap.RunID, Project: cap.Project, Role: project.RoleCoder, Generation: 3, Phase: project.FactoryRunning, Live: live.Load(), Unit: "worker-test.service", Invocation: "invocation-test", Container: strings.Repeat("b", 64)})
	}))
	defer native.Close()
	product := &API{Store: st, Host: &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, cap)
	server := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: st, Submissions: st, Assignments: st, Runs: st, Prepare: product.prepareFactoryWorkerMCP}, key))
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "live-worker-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{cap.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	active, err := st.GetMCPCapability(ctx, cap.ID)
	if err != nil || active.State != factory.MCPCapabilityActive {
		t.Fatal("authenticated live worker did not activate")
	}
	submissionID := factory.NewID()
	args := map[string]any{"submission_id": submissionID, "payload": map[string]any{"title": "Access helper found", "summary": "The existing helper already checks task ownership.", "scope": "Task access routes", "evidence_refs": []string{"src/access.go:12"}}}
	args["payload"].(map[string]any)["source"] = map[string]any{"path": "src/access.go", "start_line": 12, "code": "return task.Owner == staff.ID", "highlight_lines": []int{12}}
	args["payload"].(map[string]any)["commits"] = []map[string]any{{"revision": strings.Repeat("a", 40), "summary": "Use the existing ownership helper"}}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.learned", Arguments: args})
	if catalogRefused(result, err) {
		t.Fatalf("worker learned: %v %v", result, err)
	}
	var response struct {
		FeedPostID string `json:"feed_post_id"`
	}
	if err := json.Unmarshal([]byte(catalogText(t, result)), &response); err != nil || response.FeedPostID == "" {
		t.Fatalf("missing post receipt: %v", err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.learned", Arguments: args})
	if catalogRefused(result, err) {
		t.Fatal("exact transport replay failed")
	}
	page, _, _, err := st.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Kinds: []string{factory.FeedKindRecord}, Limit: 10})
	if err != nil || len(page) != 2 {
		t.Fatalf("worker filter feed=%+v err=%v", page, err)
	}
	enrichFeedPosts(ctx, st, map[int64]string{7: "owner/repo"}, page)
	if page[0].ID != response.FeedPostID || page[0].Actor != "builder" || page[0].Issue != 42 || page[0].Payload.Communication.Kind != "learned" {
		t.Fatalf("bound feed projection=%+v", page[0])
	}
	communication := page[0].Payload.Communication
	if communication.Source == nil || communication.Source.Code != "return task.Owner == staff.ID" || communication.Source.HighlightLines[0] != 12 || len(communication.Commits) != 1 || communication.Commits[0].Revision != strings.Repeat("a", 40) {
		t.Fatalf("source and commits lost between MCP and stored feed: %+v", communication)
	}
	human, _, _, err := st.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Kinds: []string{factory.FeedKindPost}, Limit: 10})
	if err != nil || len(human) != 0 {
		t.Fatalf("worker appeared as human comment=%+v %v", human, err)
	}

	// A real stored human reply becomes readable without a second model/JEV call.
	humanReply, _, err := st.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "Please reuse that helper for all task routes.", ReplyTo: response.FeedPostID, IdempotencyKey: "human-reply"}, "7", "Owner", time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	unrelated, _, err := st.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "An unrelated project conversation", IdempotencyKey: "unrelated"}, "7", "Owner", time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	inboxResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.inbox", Arguments: map[string]any{}})
	if catalogRefused(inboxResult, err) {
		t.Fatalf("inbox failed: %v %v", inboxResult, err)
	}
	var inbox factory.MCPFeedPage
	if err := json.Unmarshal([]byte(catalogText(t, inboxResult)), &inbox); err != nil {
		t.Fatal(err)
	}
	if len(inbox.Items) != 2 || inbox.Items[0].ID != humanReply {
		t.Fatalf("reply missing or unrelated item leaked: %+v", inbox)
	}
	discussion, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.discussion", Arguments: map[string]any{"payload": map[string]any{"post_id": response.FeedPostID}}})
	if catalogRefused(discussion, err) {
		t.Fatalf("discussion failed: %v %v", discussion, err)
	}
	if !strings.Contains(catalogText(t, discussion), "Please reuse that helper") {
		t.Fatal("human reply body absent from discussion")
	}
	hidden, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.discussion", Arguments: map[string]any{"payload": map[string]any{"post_id": unrelated}}})
	if !catalogRefused(hidden, err) {
		t.Fatal("unrelated project discussion was readable")
	}
	live.Store(false)
	if _, err := session.ListTools(ctx, nil); err == nil {
		t.Fatal("retired native process retained tool access")
	}
}
