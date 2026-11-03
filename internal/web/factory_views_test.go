package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
)

func factoryViewsFixture(t *testing.T, host roundTrip) *Server {
	t.Helper()
	s := apiTestServer(t)
	// Baseline fixtures explicitly retain equal numeric keys. The mapped
	// journey below uses a separate registry with differing provider subjects.
	for native := int64(1); native <= 7; native++ {
		if _, err := s.API.FountainIdentities.LocalRepository(t.Context(), native); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	s.Host.HTTP = &http.Client{Transport: host}
	return s
}

func factoryViewsRun(t *testing.T, s *Server) factory.Run {
	t.Helper()
	now := time.Now()
	r := factory.Run{ID: factory.NewID(), ProjectID: webTerminalProject, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-0.157.1", Model: "test"}
	if err := s.Store.RecordFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

func factoryViewsHostResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func readFactoryRun(t *testing.T, s *Server, runID string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/factory/runs/"+runID, "", "alice"))
	var decoded map[string]any
	if w.Code == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	return w.Code, decoded
}

func TestFactoryRunStatusRejectsForeignObservation(t *testing.T) {
	s := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		return factoryViewsHostResponse(200, `{"id":"`+strings.Repeat("f", 32)+`","project":"p999999999999999999999999","phase":"running","live":true,"invocation":"`+strings.Repeat("d", 32)+`","output":"foreign run output"}`), nil
	}))
	run := factoryViewsRun(t, s)
	if code, data := readFactoryRun(t, s, run.ID); code != 503 {
		t.Fatal("run read accepted another observation identity", code, data)
	}
}

func TestFactoryRunStatusPendingLiveAndExcerpt(t *testing.T) {
	inspect := &atomic.Value{}
	inspect.Store(factoryViewsHostResponse(404, "not found"))
	s := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/factory-inspect" {
			return factoryViewsHostResponse(500, "unexpected helper call"), nil
		}
		return inspect.Load().(*http.Response), nil
	}))
	r := factoryViewsRun(t, s)
	code, decoded := readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("pending status: %d", code)
	}
	if _, present := decoded["state"]; present {
		t.Fatalf("pending run carries host state: %v", decoded["state"])
	}
	if _, present := decoded["view"]; present {
		t.Fatalf("unbound run carries a binding: %v", decoded["view"])
	}
	run, _ := decoded["run"].(map[string]any)
	if run["id"] != r.ID || run["role"] != "coder" || run["harness"] != "codex-0.157.1" {
		t.Fatalf("record: %v", decoded["run"])
	}
	if _, present := run["identity_binding"]; present {
		t.Fatal("broker binding leaked into the display record")
	}
	live := `{"id":"` + r.ID + `","project":"` + webTerminalProject + `","phase":"running","live":true,` +
		`"container":"` + strings.Repeat("c", 64) + `","unit":"soda-factory-` + r.ID + `.service","invocation":"` + strings.Repeat("d", 32) + `"}`
	inspect.Store(factoryViewsHostResponse(200, live))
	code, decoded = readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("live status: %d", code)
	}
	state, _ := decoded["state"].(map[string]any)
	if state["phase"] != "running" || state["live"] != true || state["terminal"] != false {
		t.Fatalf("live state: %v", state)
	}
	if state["container"] != strings.Repeat("c", 64) || state["unit"] != "soda-factory-"+r.ID+".service" || state["invocation"] != strings.Repeat("d", 32) {
		t.Fatalf("process binding: %v", state)
	}
	big := `{"id":"` + r.ID + `","project":"` + webTerminalProject + `","phase":"completed","exit_code":0,"output":"` + strings.Repeat("x", 20000) + `"}`
	inspect.Store(factoryViewsHostResponse(200, big))
	code, decoded = readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("terminal status: %d", code)
	}
	state, _ = decoded["state"].(map[string]any)
	output, _ := state["output"].(string)
	if state["terminal"] != true || state["output_truncated"] != true || len(output) != 16*1024 {
		t.Fatalf("excerpt: truncated=%v len=%d", state["output_truncated"], len(output))
	}
}

func TestFactoryRunStatusBindingAndRefusals(t *testing.T) {
	t.Run("canonical_project_binding", checkMappedRunStatus)
	s := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		return factoryViewsHostResponse(404, "not found"), nil
	}))
	r := factoryViewsRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(t.Context(), factory.RunView{RunID: r.ID, Repository: 7, Issue: 42, Attempt: "attempt-1"}); err != nil {
		t.Fatal(err)
	}
	code, decoded := readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("bound status: %d", code)
	}
	view, _ := decoded["view"].(map[string]any)
	if view["repository"] != "7" || view["issue"] != "42" || view["attempt"] != "attempt-1" {
		t.Fatalf("binding: %v", decoded["view"])
	}
	other := factoryViewsRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(t.Context(), factory.RunView{RunID: other.ID, Repository: 8, Issue: 9}); err != nil {
		t.Fatal(err)
	}
	if code, _ := readFactoryRun(t, s, other.ID); code != 409 {
		t.Fatalf("mismatched binding: %d", code)
	}
	if code, _ := readFactoryRun(t, s, factory.NewID()); code != 404 {
		t.Fatalf("unknown run: %d", code)
	}
	if code, _ := readFactoryRun(t, s, "short"); code != 404 {
		t.Fatalf("bad run identity: %d", code)
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/factory/runs/"+r.ID+"?cursor=1", "", "alice"))
	if w.Code != 400 {
		t.Fatalf("query accepted: %d", w.Code)
	}
	readOnly := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "read"}}
	}
	denied := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, denied, apiTestRequest(http.MethodGet, "/api/factory/runs/"+r.ID, "", "bob"), readOnly)
	if denied.Code != 403 {
		t.Fatalf("read-only viewer: %d", denied.Code)
	}
	broken := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		return factoryViewsHostResponse(500, "helper failed"), nil
	}))
	id := factoryViewsRun(t, broken).ID
	if code, _ := readFactoryRun(t, broken, id); code != 503 {
		t.Fatalf("broken host: %d", code)
	}
}

// Real PostgreSQL owns both namespaces; private native callbacks and host
// inspection are declared peers. Reading status never dispatches execution.
func checkMappedRunStatus(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	s, err := NewFactory(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1}, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('collaboration_repositories','id'),9007199254740991,true)`); err != nil {
		t.Fatal(err)
	}
	foreignHost := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := db.PutCollaborationHost(ctx, foreignHost); err != nil {
		t.Fatal(err)
	}
	foreign, err := db.ObserveCollaborationRepository(ctx, foreignHost.ID, "7", "alice/factory")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ObserveCollaborationActor(ctx, foreignHost.ID, "1", "alice"); err != nil {
		t.Fatal(err)
	}
	repository, err := s.API.FountainIdentities.LocalRepository(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := s.API.FountainIdentities.LocalActor(ctx, "1")
	if err != nil || actor == 1 || repository <= 9007199254740991 || repository == foreign.ID {
		t.Fatal("run fixture did not separate native/canonical/provider identities", repository, actor, err)
	}
	if err := db.UpsertUser(ctx, store.User{ID: 1, Login: "alice", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProject(ctx, store.Project{ID: webTerminalProject, Name: "Factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkReady(ctx, webTerminalProject, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	var inspections, permissions atomic.Int64
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		inspections.Add(1)
		var request project.FactoryInspect
		if r.URL.Path != "/factory-inspect" || json.NewDecoder(r.Body).Decode(&request) != nil || request.Project != webTerminalProject || !factory.ValidID(request.ID) {
			t.Error("status lost the native Project/run inspection binding")
		}
		return factoryViewsHostResponse(404, "not found"), nil
	})}
	permission, nativeRepository := "write", "7"
	call := func(run string, want int) map[string]any {
		t.Helper()
		out := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, out, apiTestRequest(http.MethodGet, "/api/factory/runs/"+run, "", "alice"), func(in extensions.CallbackRequest) extensions.CallbackResponse {
			if in.Operation == extensions.OperationRepository {
				permissions.Add(1)
				if in.RepositoryID != nativeRepository || in.Authority.Actor.ID != "1" {
					t.Error("canonical run scope reached native permission/custody")
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: nativeRepository, Owner: "alice", Name: "factory", Permission: permission}}
			}
			return extensions.CallbackResponse{}
		})
		if out.Code != want {
			t.Fatalf("run status: %d want %d: %s", out.Code, want, out.Body.String())
		}
		var view map[string]any
		if json.Unmarshal(out.Body.Bytes(), &view) != nil {
			t.Fatal("run status did not return JSON")
		}
		return view
	}
	run := factoryViewsRun(t, s)
	if _, _, err := db.RecordFactoryRunView(ctx, factory.RunView{RunID: run.ID, Repository: int64(repository), Issue: 42, Attempt: "attempt-1"}); err != nil {
		t.Fatal(err)
	}
	status := call(run.ID, 200)
	view, _ := status["view"].(map[string]any)
	if view["repository"] != strconv.FormatInt(int64(repository), 10) || view["issue"] != "42" || view["attempt"] != "attempt-1" || inspections.Load() != 1 {
		t.Fatal("status lost exact canonical view or native host inspection", status)
	}
	for _, wrong := range []int64{7, int64(foreign.ID), 9007199254740994} {
		other := factoryViewsRun(t, s)
		if _, _, err := db.RecordFactoryRunView(ctx, factory.RunView{RunID: other.ID, Repository: wrong, Issue: 42}); err != nil {
			t.Fatal(err)
		}
		call(other.ID, 409)
		if inspections.Load() != 1 {
			t.Fatal("foreign/native/unmapped Factory binding reached host output")
		}
	}
	permission = "read"
	call(run.ID, 403)
	if inspections.Load() != 1 {
		t.Fatal("revoked write permission reached host output")
	}
	permission = "write"
	unmappedProject := "p" + strings.Repeat("e", 24)
	if err := db.CreateProject(ctx, store.Project{ID: unmappedProject, Name: "Unmapped", RepositoryID: 9, OwnerID: 1, Repository: "alice/unmapped"}); err != nil {
		t.Fatal(err)
	}
	unmappedRun := run
	unmappedRun.ID, unmappedRun.ProjectID = factory.NewID(), unmappedProject
	if err := db.RecordFactoryRun(ctx, unmappedRun); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.RecordFactoryRunView(ctx, factory.RunView{RunID: unmappedRun.ID, Repository: 9, Issue: 3}); err != nil {
		t.Fatal(err)
	}
	nativeRepository = "9"
	call(unmappedRun.ID, 409)
	nativeRepository = "7"
	if inspections.Load() != 1 {
		t.Fatal("unmapped Project reached host output by equal-number fallback")
	}
	if _, err := db.CollaborationRepositoryByExternalID(ctx, s.FountainHost.ID, "9"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("status read allocated an unobserved native repository", err)
	}
	// An issue-zero verification stays recognizable in canonical scope. Its
	// unfinished record does not acquire Retry/repair authority from a read.
	intent := factory.IntentRevision{ID: "d" + strings.Repeat("3", 24), Intent: "d" + strings.Repeat("3", 24), Owner: int64(actor), Repository: int64(repository), Objective: "Verify the current delivery", State: "draft", MaxIssues: 1, CreatedUnix: time.Now().Unix()}
	if err := db.SaveIntentRevision(ctx, intent); err != nil {
		t.Fatal(err)
	}
	delivery := factory.DeliverySubject{IntentID: intent.Intent, AcceptedRevision: intent.ID, ScopeDigest: strings.Repeat("a", 64), SourceCommit: strings.Repeat("b", 40), PreparationDigest: strings.Repeat("c", 64), VerificationManifestDigest: strings.Repeat("d", 64)}
	delivery.DeliveryID = factory.DeliveryID(delivery)
	verification := run
	verification.ID, verification.Role, verification.InputSHA = factory.NewID(), project.RoleReviewer, delivery.SourceCommit
	if err := db.RecordFactoryRun(ctx, verification); err != nil {
		t.Fatal(err)
	}
	assignment := factory.Assignment{ID: factory.NewID(), Run: verification.ID, ProjectID: webTerminalProject, Repository: int64(repository), Role: project.RoleReviewer, Phase: project.ExecutionVerify, Stage: factory.AssignmentAssigned, SourceCommit: delivery.SourceCommit, Subject: factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &delivery}}
	assignment.AttemptRoot = assignment.ID
	data, _ := json.Marshal(assignment)
	if _, err := raw.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,0,$3,'assigned',0,$4)`, assignment.ID, int64(repository), assignment.Run, string(data)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.RecordFactoryRunView(ctx, factory.RunView{RunID: verification.ID, Repository: int64(repository), Attempt: assignment.ID}); err != nil {
		t.Fatal(err)
	}
	// A malformed journal under the native number is deliberately unrelated.
	// A canonical status read must not consult it or advertise its contents.
	if _, err := raw.ExecContext(ctx, `INSERT INTO factory_commands(id,type,target,principal,digest,outcome,created) VALUES($1,$2,$3,'native:1',$4,'invalid unrelated native-key journal',$5)`, factory.DeliveryRepairCauseID(verification.ID), factory.CommandDeliveryRepair, "repository/7/delivery-repair/"+verification.ID, strings.Repeat("a", 64), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	status = call(verification.ID, 200)
	retry, _ := status["verification_retry"].(map[string]any)
	repair, _ := status["delivery_repair"].(map[string]any)
	if retry["available"] != false || repair["available"] != false || repair["progress"] != nil {
		t.Fatal("unfinished canonical verification gained availability or foreign repair progress", status)
	}
	// The new control argument is derived from the admitted session, never
	// supplied through the browser action body, even by a native maintainer.
	var beforeCommands string
	if err := raw.QueryRowContext(ctx, `SELECT md5(COALESCE(jsonb_agg(to_jsonb(c) ORDER BY id),'[]'::jsonb)::text) FROM factory_commands c`).Scan(&beforeCommands); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"actor":"7"`, `"owner":"` + strconv.FormatInt(int64(actor), 10) + `"`, `"principal":"native:7"`} {
		body := `{"action":"` + factory.ActionDeliveryRepair + `",` + field + `}`
		out := httptest.NewRecorder()
		nativeAPIServe(t, s, out, apiTestRequest(http.MethodPost, "/api/factory/runs/"+verification.ID+"/actions", body, "alice"))
		if out.Code != http.StatusBadRequest {
			t.Fatalf("repair body selected an admitted actor/principal: %d %s", out.Code, out.Body.String())
		}
	}
	var afterCommands string
	if err := raw.QueryRowContext(ctx, `SELECT md5(COALESCE(jsonb_agg(to_jsonb(c) ORDER BY id),'[]'::jsonb)::text) FROM factory_commands c`).Scan(&afterCommands); err != nil || afterCommands != beforeCommands {
		t.Fatal("refused repair actor override changed commands", err)
	}
	if native, err := db.Project(ctx, webTerminalProject); err != nil || native.RepositoryID != 7 || native.OwnerID != 1 {
		t.Fatal("run read rewrote native Project ownership", native, err)
	}
	if _, err := db.User(ctx, int64(actor)); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("canonical actor became a native user", err)
	}
	if permissions.Load() < 6 {
		t.Fatal("run reads did not recheck native permissions")
	}
}

func TestSpacesShowsFactoryRuns(t *testing.T) {
	t.Run("canonical_repository_scope", checkMappedSpacesRuns)
	s, _, _ := spacesFixture(t, true)
	first := lifecycleWebRun(t, s)
	second := lifecycleWebRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(t.Context(), factory.RunView{RunID: first.ID, Repository: 7, Issue: 42, Attempt: "attempt-1"}); err != nil {
		t.Fatal(err)
	}
	settled := second
	settled.Outcome, settled.Summary, settled.Reconciled = factory.Succeeded, "done", true
	if err := s.Store.SaveFactoryRun(t.Context(), settled); err != nil {
		t.Fatal(err)
	}
	view := readSpaces(t, s, spacesCallback(200, new(atomic.Int32)))
	if len(view.Items) != 1 {
		t.Fatalf("items: %d", len(view.Items))
	}
	rows := view.Items[0].FactoryRuns
	if len(rows) != 2 {
		t.Fatalf("runs: %+v", rows)
	}
	// Newest first; the unbound run carries no binding, never a guess.
	if rows[0].ID != second.ID || rows[0].Outcome != factory.Succeeded || !rows[0].Reconciled || rows[0].Issue != "" {
		t.Fatalf("settled row: %+v", rows[0])
	}
	if rows[1].ID != first.ID || rows[1].Role != "coder" || rows[1].Issue != "42" || rows[1].Attempt != "attempt-1" || rows[1].Reconciled {
		t.Fatalf("bound row: %+v", rows[1])
	}
	var raw strings.Builder
	_ = json.NewEncoder(&raw).Encode(view)
	assertNoSecrets(t, raw.String())
}

// Exercise collection projection through real PostgreSQL and the private
// native permission transport, with colliding external and distinct local IDs.
func checkMappedSpacesRuns(t *testing.T) {
	ctx := t.Context()
	s, dsn := inventoryFixture(t)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if _, err := raw.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('collaboration_repositories','id'),9007199254740991,true)`); err != nil {
		t.Fatal(err)
	}
	foreignHost := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := s.Store.PutCollaborationHost(ctx, foreignHost); err != nil {
		t.Fatal(err)
	}
	foreign, err := s.Store.ObserveCollaborationRepository(ctx, foreignHost.ID, "7", "alice/inventory")
	if err != nil {
		t.Fatal(err)
	}
	inventoryProject(t, s, webTerminalProject, 7)
	repository, err := s.Store.CollaborationRepositoryForNativeProject(ctx, webTerminalProject)
	if err != nil || repository.ID <= 9007199254740991 || repository.ID == foreign.ID {
		t.Fatal("fixture did not separate repository scopes", repository, err)
	}
	lifecycleWebGrants(t, s, int64(repository.ID), true)
	policy, err := s.Store.RepositoryPolicy(ctx, int64(repository.ID))
	if err != nil {
		t.Fatal(err)
	}
	// Neither the native number nor a colliding provider is the policy owner.
	for _, decoy := range []int64{7, int64(foreign.ID)} {
		policy.Repository, policy.Paused = decoy, false
		if err := s.Store.SaveRepositoryPolicy(ctx, policy); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Store.WithdrawDispatch(ctx, int64(repository.ID), factory.CauseControlPaused, "native:1"); err != nil {
		t.Fatal(err)
	}
	bound, unbound := lifecycleWebRun(t, s), lifecycleWebRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(ctx, factory.RunView{RunID: bound.ID, Repository: int64(repository.ID), Issue: 42, Attempt: "mapped-attempt"}); err != nil {
		t.Fatal(err)
	}
	unbound.Outcome, unbound.Reconciled = factory.Succeeded, true
	if err := s.Store.SaveFactoryRun(ctx, unbound); err != nil {
		t.Fatal(err)
	}
	transport := s.Host.HTTP.Transport
	nativeCalls := 0
	s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		nativeCalls++
		if strings.HasPrefix(r.URL.Path, "/factory-") {
			t.Error("collection performed a Factory native inspection", r.URL.Path)
		}
		return transport.RoundTrip(r)
	})
	permissions := new(atomic.Int32)
	callback := spacesCallback(http.StatusOK, permissions)
	view := readSpaces(t, s, callback)
	if !view.Complete || view.FactoryIncomplete || len(view.Items) != 1 {
		t.Fatalf("mapped collection incomplete: %+v", view)
	}
	row := view.Items[0]
	control := row.FactoryControl
	if control == nil || !control.Paused || control.DispatchOpen || control.WithdrawalCause != factory.CauseControlPaused || control.UnsettledRuns != 1 {
		t.Fatalf("control read outside canonical scope: %+v", control)
	}
	authority := row.FactoryAuthority
	paused := false
	if authority != nil {
		for _, missing := range authority.Missing {
			paused = paused || missing == factory.MissingPolicyPaused
		}
	}
	if authority == nil || authority.Effective || authority.DispatchOpen || !paused {
		t.Fatalf("authority read outside canonical scope: %+v", authority)
	}
	if len(row.FactoryRuns) != 2 || row.FactoryRuns[0].ID != unbound.ID || row.FactoryRuns[0].Issue != "" || row.FactoryRuns[0].Attempt != "" || !row.FactoryRuns[0].Reconciled || row.FactoryRuns[0].Outcome != factory.Succeeded || row.FactoryRuns[1].ID != bound.ID || row.FactoryRuns[1].Issue != "42" || row.FactoryRuns[1].Attempt != "mapped-attempt" {
		t.Fatalf("mapped/unbound rows lost: %+v", row.FactoryRuns)
	}
	// A denied Project's conflicting metadata must not taint the visible page.
	const unmapped = "pfffffffffffffffffffffff0"
	if err := s.Store.CreateProject(ctx, store.Project{ID: unmapped, Name: "unmapped", RepositoryID: 71, OwnerID: 1, Repository: "alice/unmapped"}); err != nil {
		t.Fatal(err)
	}
	missingRun := inventoryRun(time.Now(), unmapped)
	recordInventoryRun(t, s, missingRun, false)
	if _, _, err := s.Store.RecordFactoryRunView(ctx, factory.RunView{RunID: missingRun.ID, Repository: 71, Issue: 91, Attempt: "unmapped-binding"}); err != nil {
		t.Fatal(err)
	}
	nativeCalls = 0
	view = readSpaces(t, s, func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.RepositoryID == "71" {
			return extensions.CallbackResponse{ErrorCode: "not_found"}
		}
		if in.RepositoryID != "7" {
			t.Error("native permission check received a canonical number", in.RepositoryID)
		}
		return callback(in)
	})
	if !view.Complete || view.FactoryIncomplete || len(view.Items) != 1 || nativeCalls != 2 {
		t.Fatalf("hidden Project changed visible truth or native calls: %+v calls=%d", view, nativeCalls)
	}
	// Equal native numbers, another provider and unknown canonical IDs cannot
	// supply issue/attempt facts. Recorded outcomes still survive every refusal.
	rejected := map[string]bool{missingRun.ID: true}
	for _, wrong := range []int64{7, int64(foreign.ID), int64(repository.ID) + 99} {
		run := lifecycleWebRun(t, s)
		rejected[run.ID] = true
		if _, _, err := s.Store.RecordFactoryRunView(ctx, factory.RunView{RunID: run.ID, Repository: wrong, Issue: 91, Attempt: "foreign-binding"}); err != nil {
			t.Fatal(err)
		}
	}
	nativeCalls = 0
	view = readSpaces(t, s, callback)
	if view.Complete || !view.FactoryIncomplete || len(view.Items) != 2 || nativeCalls != 4 {
		t.Fatalf("conflicting scopes disguised or extra native calls: %+v calls=%d", view, nativeCalls)
	}
	for _, item := range view.Items {
		if item.Environment.ID == unmapped && (item.FactoryAuthority != nil || item.FactoryControl != nil) {
			t.Fatal("unmapped native number became Factory authority")
		}
		for _, run := range item.FactoryRuns {
			if rejected[run.ID] {
				if run.Issue != "" || run.Attempt != "" {
					t.Fatalf("conflicting display binding escaped: %+v", run)
				}
				delete(rejected, run.ID)
			}
		}
	}
	if len(rejected) != 0 {
		t.Fatal("conflicting bindings hid recorded runs", rejected)
	}
	var count, lastID int64
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM collaboration_repositories`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRowContext(ctx, `SELECT last_value FROM collaboration_repositories_id_seq`).Scan(&lastID); err != nil {
		t.Fatal(err)
	}
	if count != 2 || lastID != int64(repository.ID) {
		t.Fatal("collection allocated missing identities", count, lastID)
	}
}
