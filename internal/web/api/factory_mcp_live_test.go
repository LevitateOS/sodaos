package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The paid-journal case needs an attributable work post without invoking the
// independently qualified native dispatch producer. Keep its SQL seed in this
// existing activation fixture rather than exposing a production Store writer.
func seedLiveMCPAssignment(t *testing.T, ctx context.Context, dsn string, assignment factory.Assignment) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	raw, err := json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,'assigned',1,$5)`, assignment.ID, assignment.Repository, assignment.Issue, assignment.Run, string(raw)); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryLiveMCPActivatesAndPublishesBoundFeed(t *testing.T) {
	exerciseFactoryLiveMCP(t, nil)
}

// This exercises the real PostgreSQL activation/stop guards, bound MCP HTTP
// and production host client. The native task owner is an explicit HTTP peer;
// actual Muse child execution is qualified by the native integration, not here.
func TestFactoryLiveMCPNativeTasksBoundToObservedRun(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	st, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	projectID := "p" + strings.Repeat("d", 24)
	if err := st.UpsertUser(ctx, store.User{ID: 7, Login: "native-task-worker"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, store.Project{ID: projectID, Name: "Native task integration", RepositoryID: 7, OwnerID: 7, Repository: "owner/repo"}); err != nil {
		t.Fatal(err)
	}
	write := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	write(`INSERT INTO factory_dispatch(repository,revision,open,data) VALUES(7,1,true,'{}')`)
	for scenarioIndex, scenario := range []struct {
		name, harness, phase string
		profile              factory.MCPProfile
		allowed, denied      string
	}{
		{"builder", "muse", project.ExecutionImplement, factory.MCPProfileBuilder, "start_task", "release_output"},
		{"fixer", "muse", project.ExecutionCorrect, factory.MCPProfileFixer, "release_output", "start_task"},
		{"inspector", "muse", project.ExecutionReview, factory.MCPProfileInspector, "", "release_output"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cap := fakeCatalogCapability()
			cap.ID, cap.AssignmentID, cap.RunID = factory.NewID(), factory.NewID(), factory.NewID()
			cap.Profile, cap.State = scenario.profile, factory.MCPCapabilityStaged
			cap.SourceDigest, cap.AcceptedRevision = strings.Repeat("a", 40), "3"
			prepID := "f" + factory.NewID()[:24]
			role := project.RoleCoder
			if scenario.phase == project.ExecutionReview {
				role = project.RoleReviewer
			}
			a := factory.Assignment{ID: cap.AssignmentID, AttemptRoot: cap.AssignmentID, Repository: 7, ProjectID: projectID, Issue: 42, ActorID: 7, Role: role, Phase: scenario.phase, Preparation: prepID, Run: cap.RunID, RunHistory: []string{cap.RunID}, Stage: factory.AssignmentAssigned, NativeRev: 3, SourceCommit: cap.SourceDigest, Harness: scenario.harness + "-fixture-v1", HarnessVers: "fixture-v1", Model: "fixture-model", Subject: factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 42, Acceptance: "d" + strings.Repeat("2", 24)}}}
			a.Issue, a.Acceptance = int64(42+scenarioIndex), "d"+factory.NewID()[:24]
			a.Subject.Issue = &factory.IssueSubject{Issue: a.Issue, Acceptance: a.Acceptance}
			a.AcceptedContext = &factory.AcceptedContext{AcceptanceID: a.Acceptance, Repository: a.Repository, Issue: a.Issue, NativeRevision: a.NativeRev, SourceCommit: a.SourceCommit, Preparation: prepID, Title: "Write one admitted output", Body: "Create proof.txt and run its approved check."}
			cap.Subject = a.Subject.SubjectKey(a.Repository)
			deadline := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)
			plan := factory.InvestigationPlan{ID: factory.NewID(), AttemptRoot: a.AttemptRoot, ScopeDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64), SourceCommit: a.SourceCommit, IntegrationCommit: a.SourceCommit, Preparation: prepID, Approach: "Write the admitted task output", AcceptanceCoverage: []string{"requested output"}, Tasks: []factory.PlanTask{{ID: "proof", Objective: "Create proof", Outputs: []string{"proof.txt"}, AcceptanceRefs: []string{"requested output"}, Lane: "main", OwnedPaths: []string{"proof.txt"}, CheckIDs: []string{"native_check"}}}, Lanes: []factory.PlanLane{{ID: "main", TaskIDs: []string{"proof"}}}, Checks: []factory.CheckSpec{{ID: "native_check", Argv: []string{"/bin/sh", "check.sh"}, Cwd: ".", TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true}}}
			plan.Digest, err = factory.PlanDigest(plan)
			if err != nil || plan.Validate() != nil {
				t.Fatal("invalid admitted plan fixture", err)
			}
			planBytes, _ := json.Marshal(plan)
			children := 1
			if scenario.phase == project.ExecutionCorrect {
				children = 0
			}
			manifest, err := project.MarshalExecutionManifest(project.ExecutionManifest{Plan: planBytes, PlanDigest: project.PlanDigest(planBytes), Invocation: cap.RunID, Phase: scenario.phase, Run: cap.RunID, Deadline: deadline, Effort: "low", Contributor: scenario.harness == "muse", MaxChildren: children})
			if err != nil {
				t.Fatal(err)
			}
			if scenario.phase == project.ExecutionReview {
				manifest = nil
			}
			invocation := factory.NewID()
			run := factory.Run{ID: cap.RunID, ProjectID: projectID, Role: role, InputSHA: a.SourceCommit, Started: deadline.Add(-10 * time.Minute), Deadline: deadline, Image: "sha256:" + strings.Repeat("e", 64), Harness: a.Harness, Model: a.Model, ExecutionManifest: manifest, IdentityLeaseID: factory.NewID(), IdentityGeneration: 3, IdentityBinding: &identity.Binding{ID: cap.RunID, Project: projectID, Kind: identity.Factory, Generation: 3, InvocationID: invocation, Scope: "factory-" + scenario.harness}}
			selection := factory.RoleSelection{Harness: scenario.harness, HarnessVers: a.HarnessVers, Model: a.Model, Profile: factory.ExecutionProfile{Effort: "low", Contributor: scenario.harness == "muse"}}
			actor := func(kind string) factory.ActorBindingRef {
				return factory.ActorBindingRef{ActorID: 7, TokenID: 1, Kind: kind}
			}
			policy := factory.RepositoryPolicy{Repository: 7, GrantedBy: 7, Enabled: true, TargetBranch: "refs/heads/main", Roles: map[string]factory.RoleSelection{project.RoleCoder: selection, project.RoleReviewer: selection}, Checks: []string{"native-check"}, MergeMethod: factory.MergeFastForward, Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate), Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge), AttemptLimits: factory.DefaultAttemptLimits(), ReassessmentMinutes: 10, IntentMinutes: 5, MaxConcurrent: 1}
			run.Admission = &factory.RunAdmission{Policy: policy, Profile: project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: run.Image, Revision: strings.Repeat("e", 40)}, Tool: project.FactoryHarnessPin{Harness: scenario.harness, Version: a.HarnessVers, Image: run.Image, SHA256: strings.Repeat("f", 64)}}
			if err := run.Validate(); err != nil {
				t.Fatal("invalid recorded run fixture", err)
			}
			saveAssignment := func(value factory.Assignment) {
				raw, _ := json.Marshal(value)
				write(`UPDATE factory_assignments SET data=$2 WHERE id=$1`, value.ID, string(raw))
			}
			saveRun := func(value factory.Run) {
				raw, _ := json.Marshal(value)
				write(`UPDATE factory_runs SET data=$2 WHERE id=$1`, value.ID, string(raw))
			}
			seedLiveMCPAssignment(t, ctx, dsn, a)
			seedLiveConsultationScope(t, ctx, db, a)
			write(`INSERT INTO factory_plan_heads(attempt_root,plan_id,generation) VALUES($1,$2,0)`, a.AttemptRoot, plan.ID)
			prep, _ := json.Marshal(project.StoredPreparation{Preparation: project.Preparation{ID: prepID, Project: projectID, Role: role, SetupDigest: cap.PreparationDigest}})
			write(`INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data) VALUES($1,$2,$3,1,'','',$4)`, prepID, projectID, role, string(prep))
			raw, _ := json.Marshal(run)
			write(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, cap.RunID, string(raw))
			if err := st.StageMCPCapability(ctx, cap); err != nil {
				t.Fatal(err)
			}
			request, err := control.ContextBundleRequest(cap, a, true, false)
			if err != nil {
				t.Fatal(err)
			}
			if reserved, err := st.ReserveContextBundleSelection(ctx, cap, request.Snapshot); err != nil || !reserved {
				t.Fatal("context selection fixture not admitted", err)
			}
			answer := factory.PriorityAnswer{Selected: factory.ContextBundleCurrent, Model: "declared-fixture", Confidence: 1, Probabilities: map[string]float64{factory.ContextBundleCurrent: 1, factory.ContextBundleSnapshotHold: 0, factory.ContextBundleAttempt: 0}}
			if err := st.RecordPriorityJudgment(ctx, factory.PriorityReceipt{Snapshot: request.Snapshot, Candidates: request.Candidates, Purpose: factory.JudgmentContextBundle, Outcome: "context-bundle", Selected: answer.Selected, Calls: []factory.PriorityCall{{Request: request, Answer: answer}}, CreatedUnix: time.Now().Unix()}); err != nil {
				t.Fatal(err)
			}
			state := project.FactoryState{ID: cap.RunID, Project: projectID, Role: role, Generation: 3, Phase: project.FactoryRunning, Live: true, Unit: "declared-native-task-fixture.service", Invocation: invocation, Container: strings.Repeat("b", 64)}
			var stateMu sync.Mutex
			var nativeCalls atomic.Int32
			nativeResult := map[string]any{"task_id": "proof", "owner": "declared-native-task-peer", "native_receipt": factory.NewID()}
			native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/factory-inspect" {
					stateMu.Lock()
					defer stateMu.Unlock()
					_ = json.NewEncoder(w).Encode(state)
					return
				}
				if r.URL.Path != "/factory-native-task" || r.Method != http.MethodPost {
					http.NotFound(w, r)
					return
				}
				var request project.FactoryNativeTask
				decoder := json.NewDecoder(r.Body)
				decoder.DisallowUnknownFields()
				if decoder.Decode(&request) != nil || request.Project != projectID || request.Run != cap.RunID || request.InvocationID != invocation || request.ManifestDigest != project.ExecutionManifestDigest(manifest) || request.Action != scenario.allowed || request.Arguments.TaskID != "proof" {
					t.Error("host request lost its exact native run, invocation, manifest, role or task binding")
					http.Error(w, "wrong native scope", http.StatusConflict)
					return
				}
				nativeCalls.Add(1)
				_ = json.NewEncoder(w).Encode(project.FactoryNativeTaskState{Result: nativeResult})
			}))
			defer native.Close()
			product := &API{Store: st, Host: &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}}
			key := []byte(strings.Repeat("k", 32))
			secret, _ := factory.DeriveMCPCapabilitySecret(key, cap)
			server := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: st, Submissions: st, Assignments: st, Runs: st, Prepare: product.prepareFactoryWorkerMCP, NativeTask: product.factoryNativeTask}, key))
			defer server.Close()
			client := mcp.NewClient(&mcp.Implementation{Name: "native-task-bound-integration"}, nil)
			session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{cap.ID, secret}}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			listed, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, tool := range listed.Tools {
				seen[tool.Name] = true
			}
			if seen["factory."+scenario.denied] || (scenario.allowed != "" && !seen["factory."+scenario.allowed]) || (scenario.allowed == "" && (seen["factory.start_task"] || seen["factory.release_output"])) {
				t.Fatal("bound discovery exposed a tool outside the actual provider/phase")
			}
			contextResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
			if catalogRefused(contextResult, err) {
				t.Fatal("bound context read failed", err)
			}
			var contextBody struct {
				Current factory.CurrentContext `json:"current"`
			}
			if err := json.Unmarshal([]byte(catalogText(t, contextResult)), &contextBody); err != nil {
				t.Fatal(err)
			}
			contextTools := map[string]bool{}
			for _, tool := range contextBody.Current.Role.Tools {
				contextTools[string(tool)] = true
			}
			if contextTools["factory."+scenario.denied] || (scenario.allowed != "" && !contextTools["factory."+scenario.allowed]) || (scenario.allowed == "" && (contextTools["factory.start_task"] || contextTools["factory.release_output"])) {
				t.Fatal("current role guidance disagreed with bound native task discovery")
			}
			refused := func(tool string, args map[string]any) {
				t.Helper()
				before := nativeCalls.Load()
				// A native liveness denial legitimately closes an SDK session.
				// Give every refusal its own real handshake so a prior closed
				// connection cannot make later scoping checks pass accidentally.
				probe := mcp.NewClient(&mcp.Implementation{Name: "native-task-refusal"}, nil)
				peer, err := probe.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{cap.ID, secret}}}, nil)
				var result *mcp.CallToolResult
				if err == nil {
					defer peer.Close()
					result, err = peer.CallTool(ctx, &mcp.CallToolParams{Name: "factory." + tool, Arguments: args})
				}
				if !catalogRefused(result, err) || nativeCalls.Load() != before {
					t.Fatal("refused scope reached the native task owner", tool, result, err)
				}
			}
			args := map[string]any{"payload": map[string]any{"task_id": "proof"}}
			refused(scenario.denied, args)
			if scenario.allowed == "" {
				return
			}
			if scenario.phase == project.ExecutionCorrect {
				// A valid native correction binding does not supply the missing
				// publication/review facts. MCP must keep that work held. Exercise
				// its production host callback separately without claiming a full
				// correction transport admission.
				refused(scenario.allowed, args)
				active, err := st.GetMCPCapability(ctx, cap.ID)
				if err != nil {
					t.Fatal(err)
				}
				for replay := 0; replay < 2; replay++ {
					value, err := product.factoryNativeTask(ctx, active, scenario.allowed, "proof")
					if err != nil || value["native_receipt"] != nativeResult["native_receipt"] {
						t.Fatal("bound correction native result unavailable", value, err)
					}
				}
				if nativeCalls.Load() != 2 {
					t.Fatal("correction replay did not reach its native owner")
				}
				return
			}
			var first string
			for repeat := 0; repeat < 2; repeat++ {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory." + scenario.allowed, Arguments: args})
				if catalogRefused(result, err) {
					t.Fatal("native tool refused valid bound work", catalogText(t, result), err)
				}
				text := catalogText(t, result)
				var value map[string]any
				if err := json.Unmarshal([]byte(text), &value); err != nil || value["native_receipt"] != nativeResult["native_receipt"] || value["owner"] != nativeResult["owner"] {
					t.Fatal("MCP replaced the actual native task result", text, err)
				}
				if repeat == 0 {
					first = text
				} else if text != first {
					t.Fatal("native exact replay was changed by transport")
				}
			}
			if nativeCalls.Load() != 2 {
				t.Fatal("replay did not reach the one native task owner")
			}
			if count, err := st.MCPSubmissionCount(ctx, a.ID); err != nil || count != 0 {
				t.Fatal("native task transport invented a second mutation journal", count, err)
			}
			for _, field := range []string{"project", "run", "invocation_id", "manifest_digest"} {
				refused(scenario.allowed, map[string]any{"payload": map[string]any{"task_id": "proof", field: "caller-invented"}})
			}
			refused(scenario.allowed, map[string]any{"payload": map[string]any{"task_id": ""}})
			refused(scenario.allowed, map[string]any{"payload": map[string]any{"task_id": "unlisted-task"}})
			write(`DELETE FROM factory_plan_heads WHERE attempt_root=$1`, a.AttemptRoot)
			refused(scenario.allowed, args)
			write(`INSERT INTO factory_plan_heads(attempt_root,plan_id,generation) VALUES($1,$2,0)`, a.AttemptRoot, plan.ID)
			for _, change := range []struct {
				name  string
				apply func(*factory.Assignment)
			}{
				{"source", func(a *factory.Assignment) { a.SourceCommit = strings.Repeat("f", 40) }},
				{"accepted revision", func(a *factory.Assignment) { a.NativeRev++ }},
				{"subject", func(a *factory.Assignment) {
					a.Subject.Issue = &factory.IssueSubject{Issue: 43, Acceptance: "d" + strings.Repeat("2", 24)}
				}},
			} {
				changed := a
				change.apply(&changed)
				saveAssignment(changed)
				refused(scenario.allowed, args)
				saveAssignment(a)
			}
			for _, change := range []func(*project.FactoryState){
				func(s *project.FactoryState) { s.ID = factory.NewID() },
				func(s *project.FactoryState) { s.Invocation = factory.NewID() },
				func(s *project.FactoryState) { s.Generation++ },
				func(s *project.FactoryState) { s.Project = "p" + strings.Repeat("e", 24) },
				func(s *project.FactoryState) { s.Role = project.RoleReviewer },
				func(s *project.FactoryState) { s.Live = false },
			} {
				stateMu.Lock()
				original := state
				change(&state)
				stateMu.Unlock()
				refused(scenario.allowed, args)
				stateMu.Lock()
				state = original
				stateMu.Unlock()
			}
			// The real host client owns exact inspect-address validation even
			// when this callback is exercised without the transport's prior
			// activation observation.
			active, err := st.GetMCPCapability(ctx, cap.ID)
			if err != nil {
				t.Fatal(err)
			}
			stateMu.Lock()
			state.Project = "p" + strings.Repeat("e", 24)
			stateMu.Unlock()
			before := nativeCalls.Load()
			if _, err := product.factoryNativeTask(ctx, active, scenario.allowed, "proof"); err == nil || nativeCalls.Load() != before {
				t.Fatal("foreign host observation reached native task effect")
			}
			stateMu.Lock()
			state.Project = projectID
			stateMu.Unlock()
			foreign := run
			foreign.IdentityBinding = nil
			saveRun(foreign)
			refused(scenario.allowed, args)
			saveRun(run)
			foreign = run
			foreign.ExecutionManifest = nil
			saveRun(foreign)
			refused(scenario.allowed, args)
			saveRun(run)
			write(`UPDATE factory_dispatch SET open=false WHERE repository=7`)
			refused(scenario.allowed, args)
			write(`UPDATE factory_dispatch SET open=true WHERE repository=7`)
			terminal, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.finish_incomplete", Arguments: map[string]any{"submission_id": factory.NewID(), "payload": map[string]any{"outcome": "blocked", "cause": "awaiting_input", "evidence_refs": []string{}, "affected_task_ids": []string{"proof"}, "requested_resolution": "Retain the actual task and request its missing input."}}})
			if catalogRefused(terminal, err) {
				t.Fatal("could not record actual terminal proposal", terminal, err)
			}
			refused(scenario.allowed, args)
		})
	}
}

// This opt-in case uses real TypeSafe HTTPS, MCP HTTP and PostgreSQL. Native
// worker liveness and recorded preparation/acceptance are explicit fixture
// peers; this is not a launched Factory coding-provider qualification.
func TestFactoryLiveTypeSafeMCPConsultationConversation(t *testing.T) {
	keyPath := os.Getenv("SODA_LIVE_TYPESAFE_KEY_FILE")
	if keyPath == "" {
		t.Skip("SODA_LIVE_TYPESAFE_KEY_FILE opt-in unavailable")
	}
	info, err := os.Lstat(keyPath)
	if err != nil || !strings.HasPrefix(filepath.Clean(keyPath), "/home/") || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() == 0 {
		t.Fatal("live TypeSafe integration requires a private regular 0600 keyfile under /home")
	}
	wire := &liveMCPTypeSafeTransport{}
	exerciseFactoryLiveMCP(t, wire)
}

func exerciseFactoryLiveMCP(t *testing.T, wire *liveMCPTypeSafeTransport) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	st, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
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
	assignment := factory.Assignment{ID: cap.AssignmentID, Repository: cap.Repository, ProjectID: cap.Project, Issue: 42, Role: project.RoleCoder, Phase: project.ExecutionImplement, Preparation: prepID, Run: cap.RunID, Stage: factory.AssignmentAssigned, SourceCommit: strings.Repeat("a", 40), RunHistory: []string{cap.RunID}, NativeRev: 3, Subject: factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 42, Acceptance: "d" + strings.Repeat("2", 24)}}}
	if wire != nil {
		cap.Profile = factory.MCPProfileResearcher
		assignment.AttemptRoot, assignment.Acceptance = assignment.ID, "d"+strings.Repeat("2", 24)
		assignment.Role, assignment.Phase = project.RoleReviewer, project.ExecutionResearch
		assignment.Subject = factory.ExecutionSubject{Kind: factory.SubjectResearch, Research: &factory.ResearchSubject{Issue: 42, Acceptance: assignment.Acceptance, Sources: []string{"accepted-objective"}}}
		assignment.AcceptedContext = &factory.AcceptedContext{AcceptanceID: assignment.Acceptance, Repository: 7, Issue: 42, NativeRevision: 3, SourceCommit: assignment.SourceCommit, Preparation: prepID, Title: "Research equipment approval", Body: "Assess whether a EUR 1200 equipment request needs Finance approval under the stated policy."}
	}
	cap.SourceDigest, cap.AcceptedRevision, cap.Subject = assignment.SourceCommit, "3", assignment.Subject.SubjectKey(assignment.Repository)
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
	if wire != nil {
		seedLiveConsultationScope(t, ctx, db, assignment)
	}
	prep := project.StoredPreparation{Preparation: project.Preparation{ID: prepID, Project: cap.Project, Role: assignment.Role, SetupDigest: cap.PreparationDigest}}
	raw, _ = json.Marshal(prep)
	write(`INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data) VALUES($1,$2,$3,1,'','',$4)`, prepID, cap.Project, assignment.Role, string(raw))
	write(`INSERT INTO factory_dispatch(repository,revision,open,data) VALUES(7,1,true,'{}')`)
	raw, _ = json.Marshal(factory.Run{ID: cap.RunID, ProjectID: cap.Project, Role: assignment.Role, InputSHA: assignment.SourceCommit})
	write(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, cap.RunID, string(raw))
	if err := st.StageMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}

	fingerprint, err := control.ContextBundleFingerprint(cap, assignment, true, false)
	if err != nil {
		t.Fatal(err)
	}
	var admitted atomic.Int32
	var reservations sync.WaitGroup
	for i := 0; i < 2; i++ {
		reservations.Add(1)
		go func() {
			defer reservations.Done()
			ok, err := st.ReserveContextBundleSelection(ctx, cap, fingerprint)
			if err != nil {
				t.Error(err)
			}
			if ok {
				admitted.Add(1)
			}
		}()
	}
	reservations.Wait()
	if admitted.Load() != 1 {
		t.Fatal("concurrent capability selection admitted more than one call")
	}
	if ok, err := st.ReserveContextBundleSelection(ctx, cap, strings.Repeat("f", 64)); err != nil || ok {
		t.Fatal("changed selection inputs refilled capability consultation", err)
	}
	var live atomic.Bool
	live.Store(true)
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/factory-inspect" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(project.FactoryState{ID: cap.RunID, Project: cap.Project, Role: assignment.Role, Generation: 3, Phase: project.FactoryRunning, Live: live.Load(), Unit: "simulated-worker-test.service", Invocation: "invocation-test", Container: strings.Repeat("b", 64)})
	}))
	defer native.Close()
	product := &API{Store: st, Host: &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, cap)
	deps := FactoryMCPDeps{Capabilities: st, Submissions: st, Assignments: st, Runs: st, PriorityJudgments: st, Prepare: product.prepareFactoryWorkerMCP}
	if wire != nil {
		deps.SystemOne = &control.TypeSafeSystemOne{APIKeyFile: os.Getenv("SODA_LIVE_TYPESAFE_KEY_FILE"), Client: &http.Client{Transport: wire, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	}
	server := httptest.NewServer(factoryBoundCatalogHandler(deps, key))
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

	assertHeldRead := func() {
		t.Helper()
		var countBefore, countAfter, spentBefore, spentAfter int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM factory_priority_judgments").Scan(&countBefore); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, "SELECT spent FROM factory_judgment_budget WHERE id=1").Scan(&spentBefore); err != nil {
			t.Fatal(err)
		}
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
		if catalogRefused(result, err) {
			t.Fatalf("held snapshot refused: %v %v", result, err)
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(catalogText(t, result)), &body); err != nil {
			t.Fatal(err)
		}
		selection, ok := body["context_selection"].(map[string]any)
		if !ok || selection["action"] != "hold" || body["attempt"] != nil || body["lineage"] != nil {
			t.Fatal("missing or uncertain receipt broadened history", body)
		}
		if guidance, ok := body["library_guidance"].(string); !ok || !strings.Contains(guidance, "current_only") || !strings.Contains(guidance, "frozen accepted inputs") {
			t.Fatal("admitted context lost current-library read and authority guidance")
		}
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM factory_priority_judgments").Scan(&countAfter); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, "SELECT spent FROM factory_judgment_budget WHERE id=1").Scan(&spentAfter); err != nil {
			t.Fatal(err)
		}
		if countAfter != countBefore || spentAfter != spentBefore {
			t.Fatal("context read wrote rows or spent a consultation")
		}
	}
	assertHeldRead()
	request := factory.PriorityRequest{Snapshot: fingerprint, Candidates: []factory.PriorityCandidate{{ID: factory.ContextBundleCurrent}, {ID: factory.ContextBundleSnapshotHold}, {ID: factory.ContextBundleAttempt}}}
	answer := factory.PriorityAnswer{Selected: factory.ContextBundleAttempt, Model: "fixture", Confidence: 0.6, Probabilities: map[string]float64{factory.ContextBundleCurrent: 0.2, factory.ContextBundleSnapshotHold: 0.2, factory.ContextBundleAttempt: 0.6}}
	receipt := factory.PriorityReceipt{Candidates: request.Candidates, Snapshot: fingerprint, Purpose: factory.JudgmentContextBundle, Outcome: "context-bundle", Selected: answer.Selected, Calls: []factory.PriorityCall{{Request: request, Answer: answer}}, CreatedUnix: time.Now().Unix()}
	if wire != nil {
		request, err = control.ContextBundleRequest(cap, assignment, true, false)
		if err != nil {
			t.Fatal(err)
		}
		answer.Selected, answer.Confidence = factory.ContextBundleCurrent, 1
		answer.Probabilities = map[string]float64{factory.ContextBundleCurrent: 1, factory.ContextBundleSnapshotHold: 0, factory.ContextBundleAttempt: 0}
		receipt.Candidates, receipt.Selected, receipt.Calls = request.Candidates, answer.Selected, []factory.PriorityCall{{Request: request, Answer: answer}}
	}
	if err := st.RecordPriorityJudgment(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	if wire != nil {
		verifyLiveMCPConsultations(t, ctx, st, db, session, active, wire)
		live.Store(false)
		if _, err := session.ListTools(ctx, nil); err == nil {
			t.Fatal("simulated retired native worker retained tool access")
		}
		return
	}
	assertHeldRead()
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
	libraryResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.search_library", Arguments: map[string]any{"payload": map[string]any{"kind": "knowledge", "current_only": true}}})
	if catalogRefused(libraryResult, err) {
		t.Fatalf("admitted current-library read failed: %v %v", libraryResult, err)
	}
	var library factory.LibraryPage
	if err := json.Unmarshal([]byte(catalogText(t, libraryResult)), &library); err != nil || len(library.Items) != 1 || library.Items[0].SourcePost != response.FeedPostID || library.Items[0].Revision != 1 {
		t.Fatalf("current-library read lost exact provenance: %+v %v", library, err)
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

func seedLiveConsultationScope(t *testing.T, ctx context.Context, db *sql.DB, a factory.Assignment) {
	t.Helper()
	accepted := factory.Acceptance{ID: a.Acceptance, Repository: a.Repository, IssueIndex: strconv.FormatInt(a.Issue, 10), Approver: 7, NativeRev: a.NativeRev, Initial: true, TitleOccurrence: factory.OriginalOccurrence, TitleDigest: factory.MCPInputDigest([]byte(a.AcceptedContext.Title)), ContentDigest: factory.MCPInputDigest([]byte(a.AcceptedContext.Body))}
	if accepted.Validate() != nil || a.AcceptedContext.ValidateFor(a) != nil || a.AcceptedContext.ValidateAcceptance(accepted) != nil {
		t.Fatal("invalid recorded acceptance fixture")
	}
	raw, _ := json.Marshal(accepted)
	if _, err := db.ExecContext(ctx, `INSERT INTO issue_acceptance_decisions(id,repository,issue,data) VALUES($1,$2,$3,$4)`, accepted.ID, a.Repository, a.Issue, string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO issue_acceptance_heads(repository,issue,decision) VALUES($1,$2,$3)`, a.Repository, a.Issue, accepted.ID); err != nil {
		t.Fatal(err)
	}
	allowance := factory.AttemptAllowance{Repository: a.Repository, Issue: a.Issue, RootAssignment: a.ID, Limits: factory.DefaultAttemptLimits(), CheckpointUnix: time.Now().Unix()}
	if err := allowance.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(allowance)
	if _, err := db.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,$2,$3,0,$4)`, a.Repository, a.Issue, a.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
}

type liveMCPTypeSafeExchange struct {
	request  map[string]any
	response map[string]any
}

// Capture only public JSON bodies, never HTTP headers or keyfile bytes. This
// delegates to the real HTTPS transport; it does not supply model answers.
type liveMCPTypeSafeTransport struct {
	calls     atomic.Int32
	mu        sync.Mutex
	exchanges []liveMCPTypeSafeExchange
	before    func(map[string]any) error
}

func (wire *liveMCPTypeSafeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodPost || request.URL.String() != "https://api.typesafe.ai/v1/systemone" || request.GetBody == nil {
		return nil, errors.New("unexpected live TypeSafe destination")
	}
	body, err := request.GetBody()
	if err != nil {
		return nil, err
	}
	var sent map[string]any
	err = json.NewDecoder(body).Decode(&sent)
	_ = body.Close()
	if err != nil {
		return nil, err
	}
	if wire.before != nil {
		if err := wire.before(sent); err != nil {
			return nil, err
		}
	}
	wire.calls.Add(1)
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	_ = response.Body.Close()
	if err != nil || len(raw) > 65536 {
		return nil, errors.New("live TypeSafe response unavailable or oversized")
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	var received map[string]any
	if response.StatusCode == http.StatusOK {
		if err := json.Unmarshal(raw, &received); err != nil {
			return nil, errors.New("invalid live TypeSafe response JSON")
		}
	}
	wire.mu.Lock()
	wire.exchanges = append(wire.exchanges, liveMCPTypeSafeExchange{sent, received})
	wire.mu.Unlock()
	return response, nil
}

func verifyLiveMCPConsultations(t *testing.T, ctx context.Context, st *store.Store, db *sql.DB, session *mcp.ClientSession, cap factory.MCPCapability, wire *liveMCPTypeSafeTransport) {
	t.Helper()
	current, err := st.FactoryCurrentContext(ctx, cap, time.Now())
	if err != nil || current.Evidence.Availability != "complete" {
		t.Fatalf("live consultation fixture lacks real admitted current facts: %v %v", current.Evidence.MissingReasons, err)
	}
	state := map[string]any{"department": "Operations", "request": map[string]any{"equipment": "Laptop", "amount_eur": 1200}, "policy": "Manager approval is required for every equipment request. Finance approval is additionally required when the amount is greater than EUR 1000. This research advice grants no approval or implementation authority."}
	criteria := map[string]any{"manager_and_finance": "Both manager and Finance approval are required.", "manager_only": "Manager approval is sufficient under the supplied threshold."}
	// Prepare every independent wording and all options before dispatch or any
	// answer inspection. The shared group affects display, never call identity.
	instructions := []string{
		"Which approval path does the stated policy require for this equipment request? Apply the supplied amount and threshold without granting approval.",
		"Assess this request against the given rule. Select whether manager approval alone suffices or Finance must approve as well; the selection is research advice only.",
		"Using only the recorded equipment amount and policy, choose the required approval route. Compare both available routes and do not treat your answer as authorization.",
	}
	group := factory.NewID()
	args := make([]map[string]any, len(instructions))
	posts := make(map[string]string, len(instructions))
	for i, instruction := range instructions {
		args[i] = map[string]any{"state": state, "instructions": instruction, "criteria": criteria, "consultation_group": group}
		snapshot, err := factory.SystemOneWorkerSnapshot(cap.ID, factory.MCPToolSystemOneChoice, state, instruction, criteria)
		if err != nil {
			t.Fatal(err)
		}
		posts[instruction] = factory.FeedConsultationPostID(snapshot)
	}
	if len(posts) != 3 {
		t.Fatal("independent questions lost distinct call identities")
	}
	wire.before = func(sent map[string]any) error {
		questions, _ := sent["questions"].(map[string]any)
		question, _ := questions["judgment"].(map[string]any)
		instruction, _ := question["instructions"].(string)
		post, err := st.FeedComment(ctx, posts[instruction])
		if err != nil || post.Consultation == nil || post.Consultation.Status != store.ConsultationStatusPending || post.Consultation.GroupID != group || post.Consultation.Instructions != instruction {
			return errors.New("exact pending consultation was not durable before real provider dispatch")
		}
		return nil
	}
	var spentBefore int
	if err := db.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1`).Scan(&spentBefore); err != nil {
		t.Fatal(err)
	}
	results := make([]*mcp.CallToolResult, len(args))
	errs := make([]error, len(args))
	var calls sync.WaitGroup
	for i := range args {
		calls.Add(1)
		go func(i int) {
			defer calls.Done()
			results[i], errs[i] = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolSystemOneChoice), Arguments: args[i]})
		}(i)
	}
	calls.Wait()
	for i, result := range results {
		if catalogRefused(result, errs[i]) {
			t.Fatalf("live TypeSafe MCP consultation %d refused: %v %v", i+1, result, errs[i])
		}
		var response struct {
			Consultation struct {
				PostID   string `json:"post_id"`
				WorkPost struct {
					Kind string `json:"kind"`
					ID   string `json:"id"`
				} `json:"work_post"`
			} `json:"consultation"`
		}
		if err := json.Unmarshal([]byte(catalogText(t, result)), &response); err != nil || response.Consultation.PostID != posts[instructions[i]] || response.Consultation.WorkPost.Kind != "record" || response.Consultation.WorkPost.ID != cap.AssignmentID {
			t.Fatal("MCP response returned a different consultation or work thread identity", err)
		}
		replay, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolSystemOneChoice), Arguments: args[i]})
		if catalogRefused(replay, err) || catalogText(t, result) != catalogText(t, replay) {
			t.Fatalf("live consultation %d replay differs: %v", i+1, err)
		}
	}
	if wire.calls.Load() != 3 || len(wire.exchanges) != 3 {
		t.Fatalf("expected three real calls and no replay dispatch, got %d", wire.calls.Load())
	}
	models := map[string]bool{}
	for _, exchange := range wire.exchanges {
		questions := exchange.request["questions"].(map[string]any)
		question := questions["judgment"].(map[string]any)
		instruction := question["instructions"].(string)
		snapshot, _ := factory.SystemOneWorkerSnapshot(cap.ID, factory.MCPToolSystemOneChoice, state, instruction, criteria)
		post, err := st.FeedComment(ctx, posts[instruction])
		if err != nil || post.Consultation == nil || post.Consultation.Status != store.ConsultationStatusReturned || post.Consultation.Assignment != cap.AssignmentID || post.Consultation.Snapshot != snapshot || post.Consultation.GroupID != group || post.Consultation.SourceRevision != cap.AcceptedRevision {
			t.Fatalf("live question/result lost its exact work identity: %v", err)
		}
		answers, _ := exchange.response["answers"].(map[string]any)
		actualAnswer := map[string]any{"answer": answers["judgment"], "model": exchange.response["model"], "usage": exchange.response["usage"]}
		assertLiveJSON := func(label string, a, b any) {
			t.Helper()
			left, _ := json.Marshal(a)
			right, _ := json.Marshal(b)
			if !bytes.Equal(left, right) {
				t.Fatal(label + " differs from the actual provider exchange")
			}
		}
		assertLiveJSON("recorded state", post.Consultation.State, exchange.request["state"])
		assertLiveJSON("recorded question", post.Consultation.Instructions, question["instructions"])
		assertLiveJSON("recorded alternatives", post.Consultation.Criteria, question["criteria"])
		assertLiveJSON("recorded answer", post.Consultation.Answer, actualAnswer)
		receipt, err := st.PriorityJudgment(ctx, snapshot)
		if err != nil || receipt.Outcome != systemOneAnswerOutcome {
			t.Fatal("real answer lacks its durable funding receipt", err)
		}
		assertLiveJSON("funding answer", receipt.Answer, actualAnswer)
		if post.Consultation.Model == "" || post.Consultation.Model != exchange.response["model"] {
			t.Fatal("actual provider identity was lost")
		}
		models[post.Consultation.Model] = true
	}
	thread, _, err := st.FeedReplyIDs(ctx, cap.Repository, "record", cap.AssignmentID, 0)
	if err != nil || len(thread) != 3 {
		t.Fatalf("live consultation thread has %d comments, expected 3: %v", len(thread), err)
	}
	followup := map[string]any{"submission_id": factory.NewID(), "payload": map[string]any{"title": "Approval research recorded", "summary": "I retained all three actual judgments with the supplied policy. My next action is to cite the policy in the approval design review; this research does not approve the request or implement a change.", "reply_to": posts[instructions[0]]}}
	first, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolReply), Arguments: followup})
	if catalogRefused(first, err) {
		t.Fatalf("actual MCP follow-up reply refused: %v %v", first, err)
	}
	replay, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolReply), Arguments: followup})
	if catalogRefused(replay, err) || catalogText(t, first) != catalogText(t, replay) {
		t.Fatal("follow-up replay did not retain its identity", err)
	}
	var receipt factory.MCPReceipt
	if err := json.Unmarshal([]byte(catalogText(t, first)), &receipt); err != nil || receipt.FeedPostID == "" {
		t.Fatal("follow-up receipt lacks its post", err)
	}
	reply, err := st.FeedComment(ctx, receipt.FeedPostID)
	if err != nil || reply.Consultation != nil || reply.Record.ReplyTo != posts[instructions[0]] || reply.Payload.Communication == nil || reply.Payload.Communication.Kind != "reply" {
		t.Fatal("agent follow-up was conflated with provider output", err)
	}
	thread, _, err = st.FeedReplyIDs(ctx, cap.Repository, "record", cap.AssignmentID, 0)
	if err != nil || len(thread) != 4 {
		t.Fatal("follow-up did not join the existing work thread exactly once", err)
	}
	var spentAfter int
	if err := db.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1`).Scan(&spentAfter); err != nil || spentAfter-spentBefore != 3 || wire.calls.Load() != 3 {
		t.Fatalf("consultation/reply replay changed funding: delta=%d calls=%d error=%v", spentAfter-spentBefore, wire.calls.Load(), err)
	}
	t.Logf("Real TypeSafe MCP boundary: models=%v, provider calls=3, replay calls=0, exact consultation comments=3, separate follow-up=1; native worker liveness is simulated.", models)
}
