package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

// inventoryFixture mirrors spacesFixture on an isolated ephemeral
// database and returns its connection URL so tests can corrupt it
// through a second connection. Tests skip when the disposable fixture
// environment is absent.
func inventoryFixture(t *testing.T) (*Server, string) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(context.Background(), admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	for i, login := range []string{"alice", "bob"} {
		if err = db.UpsertUser(context.Background(), store.User{ID: int64(i + 1), Login: login, Name: login}); err != nil {
			t.Fatal(err)
		}
	}
	s := New(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 999}, db)
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/terminal" {
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			_, _, _ = c.Read(r.Context())
			_ = c.Write(r.Context(), websocket.MessageText, []byte(`{"type":"metadata","terminals":[]}`))
			return
		}
		var input project.Create
		_ = json.NewDecoder(r.Body).Decode(&input)
		_ = json.NewEncoder(w).Encode(project.Environment{ID: input.ID, Running: true})
	}))
	t.Cleanup(helper.Close)
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, helper.Listener.Addr().String())
	}}}
	return s, dsn
}

func readSpacesPath(t *testing.T, s *Server, path string, callback func(extensions.CallbackRequest) extensions.CallbackResponse) api.SpacesView {
	t.Helper()
	w := spacesAPI(t, s, path, callback)
	var result api.SpacesView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal("collection", w.Code, w.Body.String())
	}
	return result
}

func inventoryRun(at time.Time, projectID string) factory.Run {
	return factory.Run{ID: factory.NewID(), ProjectID: projectID, Role: "coder",
		InputSHA: strings.Repeat("a", 40), Started: at, Deadline: at.Add(time.Hour),
		Image: "sha256:" + strings.Repeat("d", 64), Harness: "codex-0.157.1", Model: "test-model"}
}

func recordInventoryRun(t *testing.T, s *Server, r factory.Run, settled bool) {
	t.Helper()
	ctx := context.Background()
	if err := s.Store.RecordFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	if settled {
		r.Outcome, r.Summary, r.Reconciled = factory.Cancelled, "settled history", true
		if err := s.Store.SaveFactoryRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
}

func inventoryProject(t *testing.T, s *Server, id string, repository int64) {
	t.Helper()
	if err := s.Store.CreateProject(context.Background(), store.Project{
		ID: id, Name: "inventory", RepositoryID: repository, OwnerID: 1, Repository: "alice/inventory",
	}); err != nil {
		t.Fatal(err)
	}
}

// TestSpacesLiveWorkSurvivesHistoryCap proves the collection prefers live
// runs over settled history when the bounded display fills.
func TestSpacesLiveWorkSurvivesHistoryCap(t *testing.T) {
	s, _ := inventoryFixture(t)
	history, live := "p000000000000000000000001", "pfffffffffffffffffffffff0"
	inventoryProject(t, s, history, 101)
	inventoryProject(t, s, live, 102)
	now := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 64; i++ {
		recordInventoryRun(t, s, inventoryRun(now, history), true)
	}
	recordInventoryRun(t, s, inventoryRun(now, live), false)
	result := readSpacesPath(t, s, "/api/spaces", spacesCallback(0, new(atomic.Int32)))
	for _, item := range result.Items {
		if item.Environment.ID != live {
			continue
		}
		if len(item.FactoryRuns) != 1 {
			t.Fatalf("live project shows %d run rows behind cap-filling history", len(item.FactoryRuns))
		}
		if item.FactoryControl == nil || item.FactoryControl.UnsettledRuns != 1 {
			t.Fatalf("live project control wrong: %+v", item.FactoryControl)
		}
		return
	}
	t.Fatal("live project missing from its own collection")
}

// TestSpacesFailedRunReadIsIncomplete proves a failed run read stays
// visibly incomplete instead of becoming an authoritative empty result.
func TestSpacesFailedRunReadIsIncomplete(t *testing.T) {
	s, dsn := inventoryFixture(t)
	live := "pfffffffffffffffffffffff0"
	inventoryProject(t, s, live, 102)
	recordInventoryRun(t, s, inventoryRun(time.Now(), live), false)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,$2,$3,$4)`,
		"corrupt", true, false, `{"id":123}`); err != nil {
		t.Fatal(err)
	}
	if err = raw.Close(); err != nil {
		t.Fatal(err)
	}
	result := readSpacesPath(t, s, "/api/spaces", spacesCallback(0, new(atomic.Int32)))
	if result.Complete {
		t.Fatal("failed run read reported a complete collection")
	}
	for _, item := range result.Items {
		if item.FactoryControl != nil && item.FactoryControl.UnsettledRuns == 0 {
			t.Fatalf("failed run read reported authoritative zero unsettled: %+v", item.FactoryControl)
		}
	}
}

// TestSpacesFailedViewReadIsIncomplete proves a failed run-view read
// stays visibly incomplete while the surviving run rows remain.
func TestSpacesFailedViewReadIsIncomplete(t *testing.T) {
	s, dsn := inventoryFixture(t)
	live := "pfffffffffffffffffffffff0"
	inventoryProject(t, s, live, 102)
	recordInventoryRun(t, s, inventoryRun(time.Now(), live), false)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`DROP TABLE factory_run_views`); err != nil {
		t.Fatal(err)
	}
	if err = raw.Close(); err != nil {
		t.Fatal(err)
	}
	result := readSpacesPath(t, s, "/api/spaces", spacesCallback(0, new(atomic.Int32)))
	if result.Complete {
		t.Fatal("failed view read reported a complete collection")
	}
	found := false
	for _, item := range result.Items {
		if item.Environment.ID == live && len(item.FactoryRuns) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("surviving run rows disappeared with the failed view read")
	}
}

// TestSpacesEmptyReadsStayEmpty locks the companion contract: a
// successful empty factory read is a complete empty result, not an
// outage.
func TestSpacesEmptyReadsStayEmpty(t *testing.T) {
	s, _ := inventoryFixture(t)
	inventoryProject(t, s, "pfffffffffffffffffffffff0", 102)
	result := readSpacesPath(t, s, "/api/spaces", spacesCallback(0, new(atomic.Int32)))
	if !result.Complete || len(result.Items) != 1 {
		t.Fatalf("empty factory reads must stay complete: %+v", result)
	}
	row := result.Items[0]
	if len(row.FactoryRuns) != 0 || row.FactoryControl == nil || row.FactoryControl.UnsettledRuns != 0 {
		t.Fatalf("empty factory reads must stay empty: %+v %+v", row.FactoryRuns, row.FactoryControl)
	}
}

// TestSpacesProjectCursorReachesOmittedProject proves a bounded next
// step reaches authorized projects past both the 32-result and the
// 128-association windows, and that the walk terminates complete.
func TestSpacesProjectCursorReachesOmittedProject(t *testing.T) {
	s, _ := inventoryFixture(t)
	const total = 130
	for i := 1; i <= total; i++ {
		inventoryProject(t, s, fmt.Sprintf("p%024x", i), int64(100+i))
	}
	seen := map[string]bool{}
	after := ""
	pages := 0
	for {
		path := "/api/spaces"
		if after != "" {
			path += "?after=" + after
		}
		page := readSpacesPath(t, s, path, spacesCallback(0, new(atomic.Int32)))
		pages++
		if len(page.Items) > 32 {
			t.Fatalf("page %d published %d rows past the bound", pages, len(page.Items))
		}
		for _, item := range page.Items {
			if seen[item.Environment.ID] {
				t.Fatalf("cursor page repeated project %s", item.Environment.ID)
			}
			seen[item.Environment.ID] = true
		}
		if page.NextAfter == "" {
			if !page.Complete {
				t.Fatal("final cursor page stayed incomplete with nothing left to load")
			}
			break
		}
		if page.Complete {
			t.Fatal("cursor page claimed complete while more remained")
		}
		after = page.NextAfter
		if pages > 8 {
			t.Fatal("cursor walk did not terminate")
		}
	}
	if len(seen) != total {
		t.Fatalf("cursor walk reached %d of %d projects", len(seen), total)
	}
}
