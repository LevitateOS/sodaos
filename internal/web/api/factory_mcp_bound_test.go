package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFactoryBoundMCPIdentityDiscoveryAndRevocation(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	assignment := sink.assignments[sink.capability.AssignmentID]
	assignment.Issue, assignment.ActorID = 42, 7
	sink.assignments[assignment.ID] = assignment
	key := []byte(strings.Repeat("k", 32))
	secret, err := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink, PriorityJudgments: sink}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "bound-worker-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{sink.capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, tool := range list.Tools {
		seen[tool.Name] = true
		if !factory.MCPToolAdmitted(sink.capability.Profile, factory.MCPTool(tool.Name)) {
			t.Fatalf("foreign tool advertised: %s", tool.Name)
		}
		raw, _ := json.Marshal(tool.InputSchema)
		if strings.Contains(string(raw), "capability_id") {
			t.Fatalf("identity leaked into model arguments: %s", tool.Name)
		}
	}
	if !seen["factory.partial"] || !seen["factory.record_decision"] || !seen[string(factory.MCPToolFinishImplement)] || !seen[string(factory.MCPToolResolvePost)] {
		t.Fatalf("incomplete worker catalog: %v", seen)
	}
	if seen[string(factory.MCPToolPreviewRepair)] {
		t.Fatal("builder discovered a Strategist-only preview")
	}
	if result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolPreviewRepair), Arguments: map[string]any{"payload": map[string]any{"report": map[string]any{}}}}); !catalogRefused(result, err) {
		t.Fatal("builder called a Strategist-only preview")
	}
	t.Run("Strategist read-only repair preview transport", func(t *testing.T) {
		strategist := newFakeCatalogSink()
		strategist.receipts = map[string]factory.PriorityReceipt{}
		strategist.capability.Profile, strategist.capability.Subject = factory.MCPProfileStrategist, "stall:7:preview"
		a := strategist.assignments[strategist.capability.AssignmentID]
		a.Phase = "repair"
		strategist.assignments[a.ID] = a
		calls := 0
		report := factory.StallRepairReport{StallID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32), InventoryDigest: strings.Repeat("3", 64), SourceCommit: strings.Repeat("4", 40), PreparationDigest: strings.Repeat("5", 64), Inspected: []factory.InspectedSource{}, Causes: []factory.CheckFailure{{CheckRef: "missing_producer", Detail: "The manual prerequisite needs an authorized producer."}}, RetainedIssueIndexes: []int64{1}, IntentChanges: []factory.ReassessmentIntentChange{}, Coverage: []factory.ReassessmentCoverage{}, Attention: []factory.ReassessmentAttentionEntry{}, Outcome: factory.RepairOutcomeNoChange}
		want, _ := json.Marshal(report)
		secret, err := factory.DeriveMCPCapabilitySecret(key, strategist.capability)
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: strategist, Submissions: strategist, Assignments: strategist, RepairPreview: func(_ context.Context, cap factory.MCPCapability, raw json.RawMessage) (factory.RepairPreview, error) {
			var proposal factory.StallRepairReport
			if err := json.Unmarshal(raw, &proposal); err != nil {
				t.Fatal(err)
			}
			got, _ := json.Marshal(proposal)
			if cap.ScopeDigest() != strategist.capability.ScopeDigest() || string(got) != string(want) {
				t.Fatal("transport lost its authenticated scope or typed report")
			}
			calls++
			return factory.RepairPreview{Subject: cap.Subject, InputDigest: cap.ScopeDigest(), Reason: "repair graph has no effect", Effects: []factory.RepairPreviewEffect{}, NextStep: "Submit diagnosis; the stall remains held."}, nil
		}}, key))
		defer server.Close()
		worker := mcp.NewClient(&mcp.Implementation{Name: "preview-worker"}, nil)
		bound, err := worker.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{strategist.capability.ID, secret}}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer bound.Close()
		tools, err := bound.ListTools(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, tool := range tools.Tools {
			found = found || tool.Name == string(factory.MCPToolPreviewRepair)
		}
		if !found {
			t.Fatal("Strategist preview absent from authenticated discovery")
		}
		for i := 0; i < 2; i++ {
			result, err := bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolPreviewRepair), Arguments: map[string]any{"payload": map[string]any{"report": report}}})
			if catalogRefused(result, err) || !strings.Contains(catalogText(t, result), strategist.capability.ScopeDigest()) {
				t.Fatal("bound preview read refused", err)
			}
		}
		for _, args := range []map[string]any{{}, {"payload": map[string]any{"report": report, "apply": true}}, {"expected_input_digest": strings.Repeat("9", 64), "payload": map[string]any{"report": report}}, {"capability_id": strategist.capability.ID, "payload": map[string]any{"report": report}}} {
			result, err := bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolPreviewRepair), Arguments: args})
			if !catalogRefused(result, err) {
				t.Fatal("invalid preview arguments reached a read", args)
			}
		}
		if calls != 2 || len(strategist.submissions) != 0 || len(strategist.reserved) != 0 {
			t.Fatal("preview wrote a submission or reserved a judgment", calls)
		}
		result, err := bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolReassessmentScope), Arguments: map[string]any{}})
		if catalogRefused(result, err) || !strings.Contains(catalogText(t, result), strategist.capability.ScopeDigest()) || !strings.Contains(catalogText(t, result), "current_role_page") {
			t.Fatal("named scope lost authenticated binding or canonical context paging", err)
		}
		var held map[string]any
		if err := json.Unmarshal([]byte(catalogText(t, result)), &held); err != nil || held["context_selection"].(map[string]any)["action"] != "hold" {
			t.Fatal("diagnosis fixture did not preserve an actual optional-context hold", err)
		}
		diagnosis := factory.RepairDiagnosis{Subject: strategist.capability.Subject, InputDigest: strategist.capability.ScopeDigest(), SourceCommit: strings.Repeat("4", 40), InventoryDigest: strings.Repeat("3", 64), Title: "Missing producer", Summary: "The sealed prerequisite has no authorized producer.", State: "supported", Causes: report.Causes, EvidenceRefs: []string{"sealed inventory"}, Alternatives: []string{"An ordinary capacity wait does not supply the missing result."}, Uncertainty: []string{"The owner must confirm the result after a producer is added."}}
		diagnosisArgs := map[string]any{"submission_id": strings.Repeat("8", 32), "payload": diagnosis}
		firstDiagnosis := ""
		for i := 0; i < 2; i++ {
			result, err := bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordDiagnosis), Arguments: diagnosisArgs})
			if catalogRefused(result, err) {
				t.Fatal("typed diagnosis transport refused", err)
			}
			if i > 0 && catalogText(t, result) != firstDiagnosis {
				t.Fatal("diagnosis exact replay changed the acknowledged receipt")
			}
			firstDiagnosis = catalogText(t, result)
		}
		for _, args := range []map[string]any{{"payload": map[string]any{"after": -1}}, {"payload": map[string]any{"after": 0, "repository": 8}}, {"payload": map[string]any{"cursor": "unbound cursor"}}} {
			result, err := bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolReassessmentScope), Arguments: args})
			if !catalogRefused(result, err) {
				t.Fatal("scope admitted invalid continuation", args)
			}
		}
		badDiagnosis := diagnosis
		badDiagnosis.State = "adopted"
		result, err = bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordDiagnosis), Arguments: map[string]any{"submission_id": factory.NewID(), "payload": badDiagnosis}})
		if !catalogRefused(result, err) || len(strategist.submissions) != 1 || len(strategist.reserved) != 0 {
			t.Fatal("diagnosis claimed adoption, duplicated a row or reserved judgment")
		}
		strategist.capability.State = factory.MCPCapabilityRevoked
		result, err = bound.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolPreviewRepair), Arguments: map[string]any{"payload": map[string]any{"report": report}}})
		if !catalogRefused(result, err) || calls != 2 {
			t.Fatal("revoked preview worker reached the reader")
		}
	})
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if catalogRefused(result, err) {
		t.Fatalf("bound context: %v %v", result, err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(catalogText(t, result)), &body); err != nil {
		t.Fatal(err)
	}
	if body["project"] != sink.capability.Project || body["run"] != sink.capability.RunID || body["issue"] != float64(42) || body["actor_id"] != float64(7) {
		t.Fatalf("wrong bound metadata: %v", body)
	}

	// A loaded parent and a confident selection still cannot grant a foreign
	// repository's disposition to this authenticated worker.
	assignment.ParentAssignment = factory.NewID()
	assignment.AttemptRoot, assignment.Repository, assignment.ProjectID = assignment.ID, sink.capability.Repository, sink.capability.Project
	sink.assignments[assignment.ID] = assignment
	sink.assignments[assignment.ParentAssignment] = factory.Assignment{ID: assignment.ParentAssignment, ActorID: assignment.ActorID, Repository: assignment.Repository + 1, ProjectID: assignment.ProjectID, AttemptRoot: assignment.AttemptRoot, Outcome: factory.Failed, Reason: "foreign parent private disposition"}
	storeSelectionReceipt(t, sink, assignment, false, true, factory.ContextBundleLineage)
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if catalogRefused(result, err) {
		t.Fatal("basic context was unavailable", err)
	}
	if strings.Contains(catalogText(t, result), "foreign parent private disposition") {
		t.Fatal("loaded parent bypassed field grant")
	}
	if err := json.Unmarshal([]byte(catalogText(t, result)), &body); err != nil {
		t.Fatal(err)
	}
	if body["lineage"] != nil || body["lineage_incomplete"] != true {
		t.Fatal("foreign parent projection was not withheld", body)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{"capability_id": sink.capability.ID}})
	if !catalogRefused(result, err) {
		t.Fatal("model could supply identity")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishIncomplete), Arguments: map[string]any{"submission_id": strings.Repeat("d", 32), "payload": map[string]any{"outcome": "blocked", "cause": "missing_required_context", "evidence_refs": []string{}, "affected_task_ids": []string{}, "requested_resolution": "Restore current context before new work."}}})
	if catalogRefused(result, err) {
		t.Fatalf("default bound revision refused: %v %v", result, err)
	}
	sink.mu.Lock()
	sink.capability.State = factory.MCPCapabilityRevoked
	sink.mu.Unlock()
	if _, err = session.ListTools(ctx, nil); err == nil {
		t.Fatal("revoked worker still discovers tools")
	}
}

// This HTTP fixture supplies an already-qualified read producer. The real
// Store producer, paging and immutable feed join run in the owning PG cases.
func (f *fakeCatalogSink) ReassessmentScope(_ context.Context, cap factory.MCPCapability, after int64) (factory.RepairScopeView, error) {
	return factory.RepairScopeView{Assignment: cap.AssignmentID, Run: cap.RunID, InputDigest: cap.ScopeDigest(), Subject: cap.Subject, SourceCommit: strings.Repeat("4", 40), InventoryDigest: strings.Repeat("3", 64), Records: []factory.RepairScopeRecord{}, Limit: factory.RepairObservationPageSize, MaximumPerTool: factory.MaxRepairObservations, Complete: true}, nil
}

// TestFactoryBoundMCPResearcherDiscoveryAndRefusal proves the R30.1
// role interface for the researcher: discovery advertises the shared
// reads plus finish_research, hides builder-only finishes, and a direct
// call to a foreign finish refuses instead of executing.
func TestFactoryBoundMCPResearcherDiscoveryAndRefusal(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	sink.capability.Profile = factory.MCPProfileResearcher
	sink.capability.Subject = "research:7:7"
	assignment := sink.assignments[sink.capability.AssignmentID]
	assignment.Phase = string(factory.MCPPhaseOf(factory.MCPProfileResearcher))
	sink.assignments[assignment.ID] = assignment
	key := []byte(strings.Repeat("k", 32))
	secret, err := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "bound-researcher-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{sink.capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, tool := range list.Tools {
		seen[tool.Name] = true
		if !factory.MCPToolAdmitted(sink.capability.Profile, factory.MCPTool(tool.Name)) {
			t.Fatalf("foreign tool advertised: %s", tool.Name)
		}
	}
	for _, want := range []string{"factory.context", "factory.search_library", "factory.finish_incomplete", string(factory.MCPToolFinishResearch), string(factory.MCPToolResearchScope), string(factory.MCPToolRecordResearch)} {
		if !seen[want] {
			t.Fatalf("researcher catalog missing %s: %v", want, seen)
		}
	}
	if seen[string(factory.MCPToolFinishImplement)] || seen[string(factory.MCPToolVerificationScope)] || seen[string(factory.MCPToolRecordJourneyResult)] {
		t.Fatalf("foreign work controls advertised to researcher: %v", seen)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishImplement), Arguments: map[string]any{"submission_id": strings.Repeat("d", 32), "payload": map[string]any{"source_commit": strings.Repeat("c", 40)}}})
	if !catalogRefused(result, err) {
		t.Fatal("researcher reached a foreign finish")
	}
}

func TestFactoryBoundMCPRejectsForeignSecret(t *testing.T) {
	sink := newFakeCatalogSink()
	handler := factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink}, []byte(strings.Repeat("k", 32)))
	request := httptest.NewRequest(http.MethodPost, FactoryMCPRoute, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	request.Header.Set("X-Soda-Factory-Capability", sink.capability.ID)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 64))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("foreign secret: %d", response.Code)
	}
	body, _ := io.ReadAll(response.Result().Body)
	if strings.Contains(string(body), sink.capability.AssignmentID) {
		t.Fatal("refusal disclosed worker context")
	}
}

// Required current facts use actual capability activation and authenticated MCP
// HTTP over a real PostgreSQL journal. Reads retain gaps and never spend or write.
func TestFactoryBoundCurrentContextUsesRecordedScopeAndState(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
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
	cap.ID, cap.RunID, cap.AssignmentID = factory.NewID(), factory.NewID(), factory.NewID()
	cap.Profile, cap.Subject, cap.State = factory.MCPProfileResearcher, "research:7:42", factory.MCPCapabilityStaged
	cap.SourceDigest, cap.AcceptedRevision = strings.Repeat("a", 40), "3"
	prepID := "f" + strings.Repeat("2", 24)
	a := factory.Assignment{ID: cap.AssignmentID, AttemptRoot: cap.AssignmentID, Repository: cap.Repository, ProjectID: cap.Project, ActorID: 7, Issue: 42, Role: project.RoleResearcher, Phase: project.ExecutionResearch, Preparation: prepID, Run: cap.RunID, Stage: factory.AssignmentAssigned, Revision: 1, NativeRev: 3, SourceCommit: cap.SourceDigest, Prompt: []byte("private prompt never projected"), Subject: factory.ExecutionSubject{Kind: factory.SubjectResearch, Research: &factory.ResearchSubject{Issue: 42, Acceptance: "d" + strings.Repeat("2", 24), Sources: []string{"accepted-source"}}}}
	if err := st.UpsertUser(ctx, store.User{ID: 7, Login: "current-worker"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, store.Project{ID: cap.Project, Name: "Current worker", RepositoryID: 7, OwnerID: 7, Repository: "owner/repo"}); err != nil {
		t.Fatal(err)
	}
	write := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := json.Marshal(a)
	write(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,42,$2,'assigned',1,$3)`, a.ID, cap.RunID, string(raw))
	prep := project.StoredPreparation{Preparation: project.Preparation{ID: prepID, Project: cap.Project, Role: project.RoleResearcher, SetupDigest: cap.PreparationDigest}}
	raw, _ = json.Marshal(prep)
	write(`INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data) VALUES($1,$2,$3,1,'','',$4)`, prepID, cap.Project, project.RoleResearcher, string(raw))
	write(`INSERT INTO factory_dispatch(repository,revision,open,data) VALUES(7,1,true,'{}')`)
	run := factory.Run{ID: cap.RunID, ProjectID: cap.Project, Role: project.RoleResearcher, InputSHA: cap.SourceDigest}
	raw, _ = json.Marshal(run)
	write(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, cap.RunID, string(raw))
	log := factory.ResearchLog{Questions: []string{"Does the retained source support this requirement?"}, Findings: []factory.ResearchRecordFinding{{Question: 0, Finding: "Recorded supported observation", Evidence: []string{"accepted-source"}}}, Records: 1}
	raw, _ = json.Marshal(log)
	write(`INSERT INTO factory_research_records(assignment,repository,data) VALUES($1,7,$2)`, a.ID, string(raw))
	if err := st.StageMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	if err := st.ActivateObservedMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	active, err := st.GetMCPCapability(ctx, cap.ID)
	if err != nil || active.State != factory.MCPCapabilityActive {
		t.Fatal("observed activation failed", err)
	}
	var storedState string
	if err := db.QueryRowContext(ctx, `SELECT data->>'state' FROM factory_mcp_capabilities WHERE id=$1`, cap.ID).Scan(&storedState); err != nil || storedState != factory.MCPCapabilityStaged {
		t.Fatal("fixture must retain staged JSON and canonical active column", err)
	}
	key := []byte(strings.Repeat("k", 32))
	secret, err := factory.DeriveMCPCapabilitySecret(key, active)
	if err != nil {
		t.Fatal(err)
	}
	var clock atomic.Int64
	clock.Store(time.Now().Unix())
	provider := &fakeSystemOne{}
	var verificationEvidence factory.CurrentVerificationEvidence
	server := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: st, Submissions: st, Assignments: st, Runs: st, PriorityJudgments: st, SystemOne: provider, VerificationEvidence: func(_ context.Context, bound factory.MCPCapability) (map[string]any, error) {
		if bound.Profile != factory.MCPProfileTestPilot {
			t.Fatal("verification evidence reached a foreign profile")
		}
		outcome := "native_check_passed"
		if verificationEvidence.ExecutionRequired {
			outcome = "native_preparation_check_passed"
			if verificationEvidence.ExecutionReceipt != nil {
				outcome = "native_manifest_passed"
			}
		}
		return map[string]any{"available": true, "evidence": verificationEvidence, "outcome": outcome}, nil
	}, Now: func() time.Time { return time.Unix(clock.Load(), 0) }}, key))
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "current-context-pg"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{active.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	counts := func() string {
		t.Helper()
		var value string
		if err := db.QueryRowContext(ctx, `SELECT jsonb_build_array((SELECT count(*) FROM factory_priority_judgments),(SELECT spent FROM factory_judgment_budget WHERE id=1),(SELECT count(*) FROM factory_mcp_submissions))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := counts()
	for i := 0; i < 2; i++ {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
		if catalogRefused(result, err) {
			t.Fatal("typed current context refused", err)
		}
		text := catalogText(t, result)
		if strings.Contains(text, "private prompt") {
			t.Fatal("current projection exposed excluded material")
		}
		var body struct {
			Current factory.CurrentContext `json:"current"`
		}
		if err := json.Unmarshal([]byte(text), &body); err != nil {
			t.Fatal(err)
		}
		current := body.Current
		projected, _ := json.Marshal(current)
		if strings.Contains(string(projected), "actor_id") {
			t.Fatal("typed current projection exposed actor identity")
		}
		if current.Version != "context-current-v1" || current.Binding.AcceptedRevision != "3" || current.Role.Research == nil || len(current.Role.Research.Questions) != 1 || !current.Role.Research.Questions[0].Answered {
			t.Fatal("required research facts missing from held context", text)
		}
		if current.Run == nil || !current.Run.Active || current.Run.Settled || current.Run.ObservedState != "not_wired" || current.Allowance != nil || current.Evidence.Availability != "incomplete" {
			t.Fatal("unknown allowance or unobserved run became confirmed", text)
		}
	}
	if counts() != before {
		t.Fatal("read context wrote or spent")
	}
	batch := factory.ResearchRecord{Experiments: []string{}}
	for i := 0; i < 32; i++ {
		batch.Experiments = append(batch.Experiments, strings.Repeat("😀", 400))
	}
	batchJSON, _ := json.Marshal(batch)
	decoded, err := factory.DecodeResearchRecord(factory.MCPToolRecordResearch, batchJSON)
	if err != nil {
		t.Fatal(err)
	}
	large, err := factory.AppendResearchRecord(log, decoded)
	if err != nil {
		t.Fatal(err)
	}
	largeJSON, _ := json.Marshal(large)
	write(`UPDATE factory_research_records SET data=$1 WHERE assignment=$2`, string(largeJSON), a.ID)
	readPage := func(cursor string) currentContextPage {
		t.Helper()
		args := map[string]any{}
		if cursor != "" {
			args["payload"] = map[string]any{"cursor": cursor}
		}
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: args})
		if catalogRefused(result, err) {
			t.Fatal("context continuation refused", err)
		}
		var response struct {
			Page currentContextPage `json:"current_role_page"`
		}
		if err := json.Unmarshal([]byte(catalogText(t, result)), &response); err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(response.Page)
		if len(encoded) > factory.MCPMaxPageBytes || len(response.Page.Records) > factory.MCPMaxPageRefs || !response.Page.PageComplete {
			t.Fatal("context page exceeds its bounds or is incomplete")
		}
		return response.Page
	}
	firstPage := readPage("")
	if firstPage.CollectionComplete || firstPage.NextCursor == "" {
		t.Fatal("oversize required input appeared as a complete first page")
	}
	page, fragments, nextPart := firstPage, "", 0
	for pages := 0; ; pages++ {
		if pages >= maxContextPages {
			t.Fatal("required context exceeded its finite page count")
		}
		for _, record := range page.Records {
			if record.Field != "research" {
				continue
			}
			if record.Part != nextPart || record.Encoding != "json-fragment" {
				t.Fatal("required context mixed or reordered fragments")
			}
			fragments += record.Content
			nextPart++
		}
		if page.CollectionComplete {
			break
		}
		if page.NextCursor == "" {
			t.Fatal("missing required context continuation")
		}
		page = readPage(page.NextCursor)
	}
	var complete factory.ResearchScopeView
	if json.Unmarshal([]byte(fragments), &complete) != nil || len(complete.Experiments) != 32 || complete.Experiments[31] != batch.Experiments[31] {
		t.Fatal("required current pages lost exact UTF-8 source content")
	}
	refuseCursor := func(cursor string) {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{"payload": map[string]any{"cursor": cursor}}})
		if !catalogRefused(result, err) {
			t.Fatal("modified, stale or expired context cursor was accepted")
		}
	}
	payload, signature, _ := strings.Cut(firstPage.NextCursor, ".")
	positionJSON, _ := base64.RawURLEncoding.DecodeString(payload)
	var position currentContextCursor
	_ = json.Unmarshal(positionJSON, &position)
	position.Offset++
	positionJSON, _ = json.Marshal(position)
	refuseCursor(base64.RawURLEncoding.EncodeToString(positionJSON) + "." + signature)
	large.Experiments[0] += " changed"
	raw, _ = json.Marshal(large)
	write(`UPDATE factory_research_records SET data=$1 WHERE assignment=$2`, string(raw), a.ID)
	refuseCursor(firstPage.NextCursor)
	write(`UPDATE factory_research_records SET data=$1 WHERE assignment=$2`, string(largeJSON), a.ID)
	clock.Add(6 * 60)
	refuseCursor(firstPage.NextCursor)
	clock.Add(-6 * 60)
	if counts() != before {
		t.Fatal("context paging or refusal spent or wrote a submission")
	}
	// Held required context refuses a new work identity while preserving
	// recovery of a previously admitted action and an honest blocked finish.
	workPayload := map[string]any{"experiments": []string{"Retained experiment before this context hold"}}
	workID := factory.NewID()
	workArgs := map[string]any{"submission_id": workID, "payload": workPayload}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordResearch), Arguments: workArgs})
	if !catalogRefused(result, err) || counts() != before {
		t.Fatal("held context admitted new work or wrote a refusal row")
	}
	workJSON, _ := json.Marshal(workPayload)
	raw, _ = json.Marshal(log)
	write(`UPDATE factory_research_records SET data=$1 WHERE assignment=$2`, string(raw), a.ID)
	retained := factory.MCPSubmission{AssignmentID: a.ID, SubmissionID: workID, CapabilityID: active.ID, Tool: factory.MCPToolRecordResearch, ExpectedInputDigest: active.ScopeDigest(), PayloadHash: factory.MCPInputDigest(workJSON), Payload: workJSON}
	if _, err = st.RecordMCPSubmission(ctx, retained, factory.MCPReceipt{SubmissionID: workID, Tool: retained.Tool, State: factory.MCPReceiptRecorded, InputDigest: retained.PayloadHash}); err != nil {
		t.Fatal(err)
	}
	afterRetained := counts()
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordResearch), Arguments: workArgs})
	if catalogRefused(result, err) || counts() != afterRetained {
		t.Fatal("context hold prevented exact immutable work replay or duplicated it")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolReceipt), Arguments: map[string]any{"submission_id": workID}})
	if catalogRefused(result, err) {
		t.Fatal("context hold prevented receipt recovery")
	}
	questionState := map[string]any{"fact": "retained"}
	questionArgs := map[string]any{"state": questionState, "instructions": "A new question during this hold"}
	beforeQuestion := counts()
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolSystemOneNoul), Arguments: questionArgs})
	if !catalogRefused(result, err) || provider.count() != 0 || counts() != beforeQuestion {
		t.Fatal("held context funded or dispatched a new worker consultation")
	}
	questionArgs["instructions"] = "Previously admitted question"
	questionSnapshot, err := factory.SystemOneWorkerSnapshot(active.ID, factory.MCPToolSystemOneNoul, questionState, questionArgs["instructions"], nil)
	if err != nil {
		t.Fatal(err)
	}
	if reserved, err := st.ReservePriorityJudgment(ctx, questionSnapshot, 1); err != nil || !reserved {
		t.Fatal("cannot seed retained paid action", err)
	}
	if err := st.RecordPriorityJudgment(ctx, factory.PriorityReceipt{Snapshot: questionSnapshot, Outcome: systemOneAnswerOutcome, Answer: map[string]any{"answer": map[string]any{"noul": 0.8}}, CreatedUnix: time.Now().Unix()}); err != nil {
		t.Fatal(err)
	}
	beforeQuestion = counts()
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolSystemOneNoul), Arguments: questionArgs})
	if catalogRefused(result, err) || provider.count() != 0 || counts() != beforeQuestion {
		t.Fatal("held context prevented retained paid-answer replay or funded it again")
	}
	blockedArgs := map[string]any{"submission_id": factory.NewID(), "payload": map[string]any{"outcome": "blocked", "cause": "missing_required_context", "evidence_refs": []string{"accepted-source"}, "affected_task_ids": []string{}, "requested_resolution": "Reconcile the approved accepted source before retry."}}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishIncomplete), Arguments: blockedArgs})
	if catalogRefused(result, err) {
		t.Fatal("context hold prevented an honest incomplete finish", err)
	}
	foreign := active
	foreign.Subject = "research:7:99"
	if _, err := st.FactoryCurrentContext(ctx, foreign, time.Now()); err == nil {
		t.Fatal("foreign subject read current context")
	}
	write(`UPDATE factory_mcp_capabilities SET state='revoked' WHERE id=$1`, active.ID)
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if !catalogRefused(result, err) {
		t.Fatal("revoked capability read required context")
	}
	// The actual approved assertion bytes must survive the authenticated
	// Test Pilot evidence and paged current-context transports. A receipt alone
	// supplies no semantic coverage, and reads must never settle the delivery.
	idea := factory.ProductIdea{ID: "d" + strings.Repeat("3", 24), Description: "Save a task", Consequence: "The saved title persists", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}
	task := factory.IntentTask{Key: "d" + strings.Repeat("4", 24), Objective: "Persist tasks", Outcomes: []string{idea.Consequence, "Read back persisted fields"}, IdeaIDs: []string{idea.ID}}
	accepted := factory.IntentRevision{ID: "d" + strings.Repeat("5", 24), Intent: "d" + strings.Repeat("6", 24), Owner: 7, Repository: 7, Objective: "Task app", State: "accepted", MaxIssues: 1, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{task}, EffectiveOutcomes: []string{idea.ID}}
	if err := st.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	binding := factory.IntentTaskBinding{RevisionID: accepted.ID, TaskKey: task.Key, IssueID: 43, IssueIndex: 43, AcceptanceID: "d" + strings.Repeat("7", 24), CreatedUnix: 1}
	if err := st.BindIntentTaskAcceptance(ctx, binding); err != nil {
		t.Fatal(err)
	}
	event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: 43, Generation: 1, PublicationID: factory.NewID(), AcceptanceID: binding.AcceptanceID, MergedCommit: a.SourceCommit, ClosingOccurrence: "1"}
	raw, _ = json.Marshal(event)
	write(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,1,$1,$2,'1',$3)`, event.ID, event.PublicationID, string(raw))
	manifest := factory.ApprovedAppCheck{IntentID: accepted.Intent, AcceptedRevision: accepted.ID, Outcomes: []string{idea.ID}}
	manifestJSON, _ := json.Marshal(manifest)
	approvedCheck := "#!/bin/sh\n" + factory.ApprovedAppCheckPrefix + string(manifestJSON) + "\ncan test --artifact checked.can-artifact.json\n"
	assertions := strings.Repeat("expect saved.title == requested.title // retained UTF-8: ✓\n", 100)
	verificationEvidence = factory.CurrentVerificationEvidence{Manifest: manifest, ManifestDigest: manifest.Digest(), PreparationDigest: strings.Repeat("b", 64), SourceCommit: a.SourceCommit,
		Receipt:         factory.VerificationCheckReceipt{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: factory.MCPInputDigest([]byte("private native output")), LogExcerpt: "Native approved application check passed"},
		ApprovedContext: factory.NewApprovedVerificationContext(project.FactoryPreparationContext{Project: cap.Project, ID: "f" + strings.Repeat("3", 24), Role: project.RoleReviewer, SourceCommit: a.SourceCommit, ApprovedBase: a.SourceCommit, DiffBase: a.SourceCommit, Candidate: a.SourceCommit, Setup: []byte("#!/bin/sh\ncan check app.can\n"), Check: []byte(approvedCheck), Files: []project.FactoryContextFile{{Path: "app.can", Content: []byte(assertions)}}}),
		State:           project.PrepareState{Output: "private native output"}}
	pilot := a
	pilot.ID, pilot.Run = factory.NewID(), factory.NewID()
	pilot.AttemptRoot, pilot.PublicationAssignment = pilot.ID, ""
	pilot.Issue, pilot.Role, pilot.Phase = 0, project.RoleReviewer, project.ExecutionVerify
	pilot.Preparation = verificationEvidence.ApprovedContext.ID
	pilot.Subject = factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &factory.DeliverySubject{DeliveryID: factory.NewID(), IntentID: accepted.Intent, AcceptedRevision: accepted.ID, ScopeDigest: accepted.Digest(), SourceCommit: a.SourceCommit, PreparationDigest: verificationEvidence.PreparationDigest, VerificationManifestDigest: manifest.Digest()}}
	_, deliveryPrerequisites, err := st.ApplicationVerificationEvidenceInputs(ctx, pilot)
	if err != nil {
		t.Fatal(err)
	}
	verificationEvidence.DeliveryPrerequisites = &deliveryPrerequisites
	raw, _ = json.Marshal(pilot)
	write(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,0,$2,'assigned',1,$3)`, pilot.ID, pilot.Run, string(raw))
	pilotRun := factory.Run{ID: pilot.Run, ProjectID: cap.Project, Role: project.RoleReviewer, InputSHA: a.SourceCommit}
	raw, _ = json.Marshal(pilotRun)
	write(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, pilot.Run, string(raw))
	pilotCap := cap
	pilotCap.ID, pilotCap.RunID, pilotCap.AssignmentID = factory.NewID(), pilot.Run, pilot.ID
	pilotCap.Profile, pilotCap.Subject = factory.MCPProfileTestPilot, pilot.Subject.SubjectKey(7)
	pilotPrep := project.StoredPreparation{Preparation: project.Preparation{ID: pilot.Preparation, Project: cap.Project, Role: project.RoleReviewer, SourceCommit: a.SourceCommit, SetupDigest: pilotCap.PreparationDigest}}
	raw, _ = json.Marshal(pilotPrep)
	write(`INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data) VALUES($1,$2,$3,1,'','',$4)`, pilot.Preparation, cap.Project, project.RoleReviewer, string(raw))
	if err := st.StageMCPCapability(ctx, pilotCap); err != nil {
		t.Fatal(err)
	}
	if err := st.ActivateObservedMCPCapability(ctx, pilotCap); err != nil {
		t.Fatal(err)
	}
	pilotCap, err = st.GetMCPCapability(ctx, pilotCap.ID)
	if err != nil {
		t.Fatal(err)
	}
	pilotSecret, err := factory.DeriveMCPCapabilitySecret(key, pilotCap)
	if err != nil {
		t.Fatal(err)
	}
	pilotSession, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{pilotCap.ID, pilotSecret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pilotSession.Close()
	beforeVerification := counts()
	verificationReadState := func() string {
		t.Helper()
		var digest string
		if err := db.QueryRowContext(ctx, `SELECT md5(jsonb_build_array((SELECT jsonb_agg(data ORDER BY event) FROM factory_completions),(SELECT jsonb_agg(data ORDER BY id) FROM factory_assignments),(SELECT jsonb_agg(data ORDER BY id) FROM factory_runs),(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c),(SELECT jsonb_agg(data ORDER BY root_assignment) FROM factory_attempt_allowances),(SELECT jsonb_agg(data ORDER BY assignment) FROM factory_reservations),(SELECT jsonb_agg(data ORDER BY run) FROM factory_usage))::text)`).Scan(&digest); err != nil {
			t.Fatal(err)
		}
		return digest
	}
	beforeVerificationState := verificationReadState()
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolVerificationEvidence), Arguments: map[string]any{}})
	if catalogRefused(result, err) {
		t.Fatal("approved verification evidence refused", err)
	}
	var observedEvidence struct {
		Evidence factory.CurrentVerificationEvidence `json:"evidence"`
		Outcome  string                              `json:"outcome"`
	}
	if json.Unmarshal([]byte(catalogText(t, result)), &observedEvidence) != nil || observedEvidence.Evidence.ValidateFor(*pilot.Subject.Delivery) != nil || observedEvidence.Outcome != "native_check_passed" || observedEvidence.Evidence.ApprovedContext.Check != approvedCheck || observedEvidence.Evidence.ApprovedContext.Files[0].Content != assertions || strings.Contains(catalogText(t, result), "private native output") {
		t.Fatal("authenticated evidence lost exact readable assertions or exposed logs")
	}
	prerequisites := observedEvidence.Evidence.DeliveryPrerequisites
	if prerequisites.IntentID != accepted.Intent || prerequisites.AcceptedRevision != accepted.ID || prerequisites.ScopeDigest != accepted.Digest() || prerequisites.SourceCommit != a.SourceCommit || len(prerequisites.Tasks) != 1 || prerequisites.Tasks[0].TaskKey != binding.TaskKey || prerequisites.Tasks[0].IssueID != binding.IssueID || prerequisites.Tasks[0].IssueIndex != binding.IssueIndex || prerequisites.Tasks[0].AcceptanceID != binding.AcceptanceID || prerequisites.Tasks[0].Completion != event {
		t.Fatal("authenticated evidence omitted canonical task/merge/closure prerequisites")
	}
	readCurrentVerification := func() factory.CurrentVerificationEvidence {
		t.Helper()
		verificationJSON, nextVerificationPart, cursor := "", 0, ""
		for pages := 0; ; pages++ {
			if pages >= maxContextPages {
				t.Fatal("approved assertion context exceeded its finite page bound")
			}
			args := map[string]any{}
			if cursor != "" {
				args["payload"] = map[string]any{"cursor": cursor}
			}
			result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: args})
			if catalogRefused(result, err) {
				t.Fatal("approved assertion current context refused", err)
			}
			var response struct {
				Page currentContextPage `json:"current_role_page"`
			}
			if json.Unmarshal([]byte(catalogText(t, result)), &response) != nil {
				t.Fatal("invalid approved assertion page")
			}
			for _, record := range response.Page.Records {
				if record.Field == "verification" {
					if record.Part != nextVerificationPart {
						t.Fatal("approved assertion context reordered fragments")
					}
					verificationJSON += record.Content
					nextVerificationPart++
				}
			}
			if response.Page.CollectionComplete {
				break
			}
			cursor = response.Page.NextCursor
		}
		var currentVerification factory.CurrentVerificationEvidence
		if json.Unmarshal([]byte(verificationJSON), &currentVerification) != nil {
			t.Fatal("invalid verification current context")
		}
		return currentVerification
	}
	currentVerification := readCurrentVerification()
	if currentVerification.ValidateFor(*pilot.Subject.Delivery) != nil || currentVerification.ApprovedContext.Files[0].Content != assertions || len(currentVerification.DeliveryPrerequisites.Tasks) != 1 || currentVerification.DeliveryPrerequisites.Tasks[0].Completion != event || counts() != beforeVerification {
		t.Fatal("required current context lost assertions or spent/settled during a read")
	}
	// A catalog-adopted definition is truthful before launch with only its
	// preparation receipt. The distinct per-run receipt becomes visible only
	// after the trusted executor has passed under this worker's invocation.
	verificationEvidence.AdoptedManifest = &factory.AdoptedVerificationManifest{Intent: accepted.Intent, AcceptedRevision: accepted.ID, ScopeDigest: accepted.Digest(), Catalog: factory.ApprovedProjectCheckCatalog{Project: cap.Project, Preparation: pilot.Preparation, Requirements: "d" + strings.Repeat("8", 24), Approval: "d" + strings.Repeat("9", 24), SourceCommit: a.SourceCommit, PreparationDigest: verificationEvidence.PreparationDigest, CheckDigest: factory.MCPInputDigest([]byte(approvedCheck))}, Proposal: factory.VerificationManifestProposal{Outcomes: accepted.EffectiveOutcomes, Checks: []factory.ManifestCheckRef{{Outcome: idea.ID, CatalogID: factory.ApprovedAppCheckID, Artifacts: []string{factory.ProjectCheckReceiptArtifact}, Fixtures: []string{factory.ProjectCheckPreparedFixture}}}}}
	verificationEvidence.ExecutionRequired = true
	currentVerification = readCurrentVerification()
	if currentVerification.ValidateFor(*pilot.Subject.Delivery) != nil || !currentVerification.ExecutionRequired || currentVerification.Execution != nil || currentVerification.ExecutionReceipt != nil {
		t.Fatal("prelaunch context invented a manifest execution receipt")
	}
	zero := 0
	verificationEvidence.Execution = &project.FactoryVerificationChecksState{Project: cap.Project, Run: pilot.Run, Incarnation: strings.Repeat("c", 32), ManifestDigest: strings.Repeat("d", 64), Phase: "passed", SourceCommit: a.SourceCommit, Preparation: pilot.Preparation, CheckDigest: verificationEvidence.AdoptedManifest.Catalog.CheckDigest, ExitCode: &zero, OutputDigest: factory.MCPInputDigest([]byte("native per-run output")), Retirement: "confirmed"}
	verificationEvidence.ExecutionReceipt = &factory.VerificationCheckReceipt{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: verificationEvidence.Execution.OutputDigest, LogExcerpt: "Native admitted checks passed"}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolVerificationEvidence), Arguments: map[string]any{}})
	if catalogRefused(result, err) || json.Unmarshal([]byte(catalogText(t, result)), &observedEvidence) != nil || observedEvidence.Outcome != "native_manifest_passed" || observedEvidence.Evidence.ExecutionReceipt == nil || observedEvidence.Evidence.ExecutionReceipt.ArtifactDigest != verificationEvidence.Execution.OutputDigest {
		t.Fatal("authenticated evidence lost per-run manifest receipt", err)
	}
	currentVerification = readCurrentVerification()
	if currentVerification.ValidateFor(*pilot.Subject.Delivery) != nil || currentVerification.ExecutionReceipt == nil || currentVerification.ExecutionReceipt.ArtifactDigest != verificationEvidence.Execution.OutputDigest || currentVerification.Receipt.ArtifactDigest == currentVerification.ExecutionReceipt.ArtifactDigest {
		t.Fatal("current context substituted preparation evidence for run receipt")
	}
	verificationEvidence.Execution.Run = factory.NewID()
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if !catalogRefused(result, err) || counts() != beforeVerification || verificationReadState() != beforeVerificationState {
		t.Fatal("foreign-run execution receipt was projected or spent during refusal")
	}
	verificationEvidence.Execution.Run = pilot.Run
	// The Test Pilot can retain intermediate observations without claiming
	// application completion. Identity, outcome coverage and uploaded artifacts
	// are checked again by the same transaction that records the feed post.
	tools, err := pilotSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	available := map[string]bool{}
	for _, tool := range tools.Tools {
		available[tool.Name] = true
	}
	if !available[string(factory.MCPToolVerificationScope)] || !available[string(factory.MCPToolRecordJourneyResult)] {
		t.Fatal("Test Pilot journey tools were not advertised")
	}
	readScope := func(after int64) factory.VerificationScopeView {
		t.Helper()
		before := counts()
		result, err := pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolVerificationScope), Arguments: map[string]any{"payload": map[string]any{"after": after}}})
		var view factory.VerificationScopeView
		if catalogRefused(result, err) || json.Unmarshal([]byte(catalogText(t, result)), &view) != nil || view.Assignment != pilot.ID || view.Run != pilot.Run || view.InputDigest != pilotCap.ScopeDigest() || view.Delivery != *pilot.Subject.Delivery || counts() != before || verificationReadState() != beforeVerificationState {
			t.Fatal("journey scope lost its exact binding or wrote during read", err)
		}
		return view
	}
	if scope := readScope(0); scope.Total != 0 || len(scope.Records) != 0 || !scope.Complete || scope.Maximum != factory.MaxVerificationJourneys {
		t.Fatal("empty journey scope was incomplete or invented observations")
	}
	artifactID, foreignArtifact := factory.NewID(), factory.NewID()
	write(`INSERT INTO factory_feed_artifacts(id,repository,assignment,name,media_type,content,complete,digest) VALUES($1,7,$2,'trace.txt','text/plain','fixture trace',true,$3),($4,7,$5,'foreign.txt','text/plain','foreign trace',true,$3)`, artifactID, pilot.ID, strings.Repeat("d", 64), foreignArtifact, a.ID)
	observation := factory.VerificationJourneyResult{DeliveryID: pilot.Subject.Delivery.DeliveryID, InputDigest: pilotCap.ScopeDigest(), SourceCommit: pilot.SourceCommit, Preparation: pilot.Preparation, ManifestDigest: pilot.Subject.Delivery.VerificationManifestDigest, OutcomeIDs: accepted.EffectiveOutcomes, Name: "Persist a saved task", Steps: []string{"Create a task", "Read back its saved title"}, Expected: "The saved title matches the request.", Observed: "The native assertions read back the saved title.", Result: "passed", EvidenceRefs: []string{verificationEvidence.Execution.OutputDigest}, ArtifactIDs: []string{artifactID}, Limitations: []string{"This observation does not attest operator deployment."}}
	journeyPayload := func(o factory.VerificationJourneyResult) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(o)
		var payload map[string]any
		if json.Unmarshal(raw, &payload) != nil {
			t.Fatal("invalid journey fixture")
		}
		return payload
	}
	journeyID := factory.NewID()
	journeyArgs := map[string]any{"submission_id": journeyID, "payload": journeyPayload(observation)}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordJourneyResult), Arguments: journeyArgs})
	if catalogRefused(result, err) {
		t.Fatal("scoped journey observation refused", err, catalogText(t, result))
	}
	firstReceipt := catalogText(t, result)
	afterJourney := counts()
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordJourneyResult), Arguments: journeyArgs})
	if catalogRefused(result, err) || catalogText(t, result) != firstReceipt || counts() != afterJourney {
		t.Fatal("lost journey reply recovery duplicated or changed its receipt", err)
	}
	for _, fault := range []string{"foreign-delivery", "foreign-outcome", "foreign-source", "foreign-preparation", "foreign-manifest", "foreign-input", "foreign-artifact", "verified", "changed-replay"} {
		bad := observation
		switch fault {
		case "foreign-delivery", "changed-replay":
			bad.DeliveryID = factory.NewID()
		case "foreign-outcome":
			bad.OutcomeIDs = []string{"d" + strings.Repeat("e", 24)}
		case "foreign-source":
			bad.SourceCommit = strings.Repeat("e", 40)
		case "foreign-preparation":
			bad.Preparation = "f" + strings.Repeat("e", 24)
		case "foreign-manifest":
			bad.ManifestDigest = strings.Repeat("e", 64)
		case "foreign-input":
			bad.InputDigest = strings.Repeat("e", 64)
		case "foreign-artifact":
			bad.ArtifactIDs = []string{foreignArtifact}
		case "verified":
			bad.Result = "verified"
		}
		id := factory.NewID()
		if fault == "changed-replay" {
			id = journeyID
		}
		result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordJourneyResult), Arguments: map[string]any{"submission_id": id, "payload": journeyPayload(bad)}})
		if !catalogRefused(result, err) || counts() != afterJourney || verificationReadState() != beforeVerificationState {
			t.Fatal("invalid journey changed canonical work or recorded evidence", fault)
		}
	}
	for i := 1; i < factory.MaxVerificationJourneys; i++ {
		observation.Name = "Retained observation " + strings.Repeat("x", i)
		raw, _ := json.Marshal(observation)
		sub := factory.MCPSubmission{AssignmentID: pilot.ID, CapabilityID: pilotCap.ID, SubmissionID: factory.NewID(), Tool: factory.MCPToolRecordJourneyResult, ExpectedInputDigest: pilotCap.ScopeDigest(), PayloadHash: factory.MCPInputDigest(raw), Payload: raw}
		if _, err := st.RecordMCPSubmission(ctx, sub, factory.MCPReceipt{SubmissionID: sub.SubmissionID, Tool: sub.Tool, State: factory.MCPReceiptRecorded, InputDigest: sub.PayloadHash}); err != nil {
			t.Fatal("append retained journey observation", err)
		}
	}
	var sequences int64
	seenJourneys := map[string]bool{}
	for {
		scope := readScope(sequences)
		if scope.Total != factory.MaxVerificationJourneys || len(scope.Records) > factory.VerificationJourneyPageSize {
			t.Fatal("journey scope omitted its total or exceeded its page bound")
		}
		for _, record := range scope.Records {
			if seenJourneys[record.ID] || record.Sequence <= sequences || record.Observation.DeliveryID != observation.DeliveryID || record.Observation.Observed != observation.Observed || record.FeedPostID == "" {
				t.Fatal("journey paging lost identity, ordering or exact observations")
			}
			sequences, seenJourneys[record.ID] = record.Sequence, true
		}
		if scope.Complete {
			break
		}
		if scope.NextAfter != sequences {
			t.Fatal("journey continuation skipped observations")
		}
	}
	if len(seenJourneys) != factory.MaxVerificationJourneys {
		t.Fatal("bounded journey paging lost observations")
	}
	afterJourney = counts()
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordJourneyResult), Arguments: map[string]any{"submission_id": factory.NewID(), "payload": journeyPayload(observation)}})
	if !catalogRefused(result, err) || counts() != afterJourney {
		t.Fatal("journey cap admitted another observation")
	}
	var recordedPost, verifiedPosts int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE data->'communication'->>'component_id'='TP03' AND data->>'assignment'=$1 AND data->'communication'->>'kind'='journey'),count(*) FILTER(WHERE data->'communication'->>'component_id'='F10') FROM factory_feed_comments`, pilot.ID).Scan(&recordedPost, &verifiedPosts); err != nil || recordedPost != factory.MaxVerificationJourneys || verifiedPosts != 0 || verificationReadState() != beforeVerificationState {
		t.Fatal("journey observations were missing or became verified work", recordedPost, verifiedPosts, err)
	}
	beforeVerification = counts()
	for _, fault := range []string{"missing-prerequisites", "stale-scope", "foreign-acceptance", "foreign-outcome", "duplicate-task"} {
		badPrerequisites := deliveryPrerequisites
		badPrerequisites.Tasks = append([]factory.VerificationDeliveredTask(nil), deliveryPrerequisites.Tasks...)
		verificationEvidence.DeliveryPrerequisites = &badPrerequisites
		switch fault {
		case "missing-prerequisites":
			verificationEvidence.DeliveryPrerequisites = nil
		case "stale-scope":
			badPrerequisites.ScopeDigest = strings.Repeat("e", 64)
		case "foreign-acceptance":
			badPrerequisites.Tasks[0].Completion.AcceptanceID = "d" + strings.Repeat("e", 24)
		case "foreign-outcome":
			badPrerequisites.Tasks[0].OutcomeIDs = []string{"d" + strings.Repeat("e", 24)}
		case "duplicate-task":
			badPrerequisites.Tasks = append(badPrerequisites.Tasks, badPrerequisites.Tasks[0])
		}
		result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
		if !catalogRefused(result, err) || counts() != beforeVerification {
			t.Fatal("invalid delivery prerequisites were projected or spent during refusal", fault)
		}
	}
	verificationEvidence.DeliveryPrerequisites = &deliveryPrerequisites
	verificationEvidence.ApprovedContext.Candidate = strings.Repeat("e", 40)
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if !catalogRefused(result, err) || counts() != beforeVerification || verificationReadState() != beforeVerificationState {
		t.Fatal("foreign verification source was projected or wrote during refusal")
	}
	// Observation capacity cannot consume the terminal slot. Terminal freeze
	// rejects a new journey identity while preserving the old receipt exactly.
	finish := factory.AppVerificationReport{DeliveryID: observation.DeliveryID, InputDigest: pilotCap.ScopeDigest(), Outcome: factory.AppVerificationFailed, Coverage: []factory.ReassessmentCoverage{}, CheckReceipts: []factory.VerificationCheckReceipt{*verificationEvidence.ExecutionReceipt}, Failures: []factory.CheckFailure{{CheckRef: idea.ID, Detail: "The approved readback does not exercise every accepted behavior."}}, EvidenceRefs: []string{"journey:" + journeyID}}
	finishJSON, _ := json.Marshal(finish)
	var finishPayload map[string]any
	if json.Unmarshal(finishJSON, &finishPayload) != nil {
		t.Fatal("invalid finish fixture")
	}
	finishJSON, _ = json.Marshal(finishPayload)
	finishID := factory.NewID()
	finishArgs := map[string]any{"submission_id": finishID, "payload": finishPayload}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishAppVerify), Arguments: finishArgs})
	if !catalogRefused(result, err) || !strings.Contains(catalogText(t, result), "context selection holds") || counts() != beforeVerification {
		t.Fatal("new evidence reference escaped the required-context hold", err)
	}
	// Store exercises new terminal reference admission; authenticated HTTP
	// then recovers that exact receipt while the changed context stays held.
	finishSub := factory.MCPSubmission{AssignmentID: pilot.ID, CapabilityID: pilotCap.ID, SubmissionID: finishID, Tool: factory.MCPToolFinishAppVerify, ExpectedInputDigest: pilotCap.ScopeDigest(), PayloadHash: factory.MCPInputDigest(finishJSON), Payload: finishJSON}
	if _, err := st.RecordMCPSubmission(ctx, finishSub, factory.MCPReceipt{SubmissionID: finishID, Tool: finishSub.Tool, State: factory.MCPReceiptRecorded, InputDigest: finishSub.PayloadHash}); err != nil {
		t.Fatal("journey observations consumed terminal capacity", err)
	}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishAppVerify), Arguments: finishArgs})
	if catalogRefused(result, err) {
		t.Fatal("HTTP finish dropped its canonical evidence refs", err, catalogText(t, result))
	}
	afterTerminal := counts()
	finishPayload["evidence_refs"] = []string{"journey:" + factory.NewID()}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishAppVerify), Arguments: finishArgs})
	if !catalogRefused(result, err) || counts() != afterTerminal {
		t.Fatal("finish replay replaced its original journey reference")
	}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordJourneyResult), Arguments: map[string]any{"submission_id": factory.NewID(), "payload": journeyPayload(observation)}})
	if !catalogRefused(result, err) || counts() != afterTerminal {
		t.Fatal("terminal Test Pilot admitted another journey observation")
	}
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolRecordJourneyResult), Arguments: journeyArgs})
	if catalogRefused(result, err) || catalogText(t, result) != firstReceipt || counts() != afterTerminal || verificationReadState() != beforeVerificationState {
		t.Fatal("terminal freeze lost or repeated a recorded journey", err)
	}
	write(`UPDATE factory_mcp_capabilities SET state='revoked' WHERE id=$1`, pilotCap.ID)
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolVerificationScope), Arguments: map[string]any{}})
	if !catalogRefused(result, err) || counts() != afterTerminal {
		t.Fatal("revoked Test Pilot read journey scope")
	}

	// A real Architect report must remain correctable before terminal freeze.
	// Exercise the observed invalid-ID/empty-integration report against this
	// same PostgreSQL journal, then settle only its corrected typed proposal.
	architect := a
	architect.ID, architect.Run = factory.NewID(), factory.NewID()
	architect.Issue = 43
	architect.AttemptRoot, architect.Role, architect.Phase = architect.ID, project.RoleCoder, project.ExecutionInvestigate
	architect.PublicationAssignment = architect.ID
	architect.Acceptance = "d" + strings.Repeat("2", 24)
	architect.Harness, architect.HarnessVers, architect.Model, architect.Connection = "codex-fixture", "fixture", "fixture", "fixture"
	architect.PromptSHA = project.FactoryPromptDigest(architect.Prompt)
	architect.Attempts, architect.RunHistory = 1, []string{architect.Run}
	architect.Subject = factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 43, Acceptance: "d" + strings.Repeat("2", 24)}}
	if err := architect.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(architect)
	write(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,43,$2,'assigned',1,$3)`, architect.ID, architect.Run, string(raw))
	architectCap := cap
	architectCap.ID, architectCap.AssignmentID, architectCap.RunID = factory.NewID(), architect.ID, architect.Run
	architectCap.Profile, architectCap.Subject = factory.MCPProfileArchitect, architect.Subject.SubjectKey(architect.Repository)
	if err := st.StageMCPCapability(ctx, architectCap); err != nil {
		t.Fatal(err)
	}
	if err := st.ActivateObservedMCPCapability(ctx, architectCap); err != nil {
		t.Fatal(err)
	}
	plan := factory.InvestigationPlan{
		ID: "native-proof-plan", AttemptRoot: architect.ID, ScopeDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64),
		SourceCommit: architect.SourceCommit, Preparation: architect.Preparation, Approach: "Write the single accepted output and run its declared check.",
		Inspected:          []factory.InspectedSource{{Path: "README.md", Commit: architect.SourceCommit, Evidence: "Read the admitted source."}},
		AcceptanceCoverage: []string{"The one accepted output is covered."},
		Tasks:              []factory.PlanTask{{ID: "proof", Objective: "Write proof.txt", Outputs: []string{"proof.txt"}, AcceptanceRefs: []string{"output"}, Lane: "serial", OwnedPaths: []string{"proof.txt"}, CheckIDs: []string{"verify"}}},
		Lanes:              []factory.PlanLane{{ID: "serial", TaskIDs: []string{"proof"}}},
		Checks:             []factory.CheckSpec{{ID: "verify", Argv: []string{"/bin/sh", "check.sh"}, Cwd: ".", TimeoutSeconds: 30, Scope: factory.CheckScopeTask, Required: true}},
	}
	plan.Digest, _ = factory.PlanDigest(plan)
	proposal := factory.MCPSubmission{AssignmentID: architect.ID, CapabilityID: architectCap.ID, SubmissionID: factory.NewID(), Tool: factory.MCPToolFinishIssuePlan, ExpectedInputDigest: architectCap.ScopeDigest()}
	encodePlan := func() {
		t.Helper()
		proposal.Payload, err = json.Marshal(map[string]any{"approach": plan.Approach, "tasks": plan, "evidence_refs": []string{"README.md"}})
		if err != nil {
			t.Fatal(err)
		}
		proposal.PayloadHash = factory.MCPInputDigest(proposal.Payload)
	}
	encodePlan()
	catalog := &factoryCatalogServer{deps: FactoryMCPDeps{Submissions: st, Assignments: st}}
	if err := catalog.validateIssuePlanSubmission(ctx, proposal); err == nil || !strings.Contains(err.Error(), "expected_plan_digest=") || !strings.Contains(err.Error(), "no terminal proposal recorded") {
		t.Fatal("invalid plan did not return a correctable refusal", err)
	}
	if count, err := st.MCPSubmissionCount(ctx, architect.ID); err != nil || count != 0 {
		t.Fatal("invalid plan consumed a terminal identity", count, err)
	}
	if _, frozen, err := st.TerminalMCPProposal(ctx, architect.ID); err != nil || frozen {
		t.Fatal("invalid plan froze its assignment", err)
	}
	plan.ID, plan.IntegrationCommit = factory.NewID(), architect.SourceCommit
	plan.Digest, _ = factory.PlanDigest(plan)
	encodePlan()
	if err := catalog.validateIssuePlanSubmission(ctx, proposal); err != nil {
		t.Fatal("corrected same submission identity was not recoverable", err)
	}
	if _, err := st.RecordMCPSubmission(ctx, proposal, factory.MCPReceipt{SubmissionID: proposal.SubmissionID, Tool: proposal.Tool, State: factory.MCPReceiptRecorded, InputDigest: proposal.PayloadHash}); err != nil {
		t.Fatal(err)
	}
	withdrawal, _ := json.Marshal(factory.Withdrawal{Repository: 7, Revision: 1, Cause: "fixture", ClosedBy: "fixture"})
	write(`UPDATE factory_dispatch SET data=$1 WHERE repository=7`, string(withdrawal))
	// A byte conflict under a retained identity still reaches the immutable
	// store guard rather than being recast as a new validation failure.
	originalPlanID := plan.ID
	plan.ID = "invalid-after-recording"
	encodePlan()
	if err := catalog.validateIssuePlanSubmission(ctx, proposal); err != nil {
		t.Fatal("new validation prevented immutable replay/conflict recovery", err)
	}
	conflictReceipt := factory.MCPReceipt{SubmissionID: proposal.SubmissionID, Tool: proposal.Tool, State: factory.MCPReceiptRecorded, InputDigest: proposal.PayloadHash}
	if _, err := st.RecordMCPSubmission(ctx, proposal, conflictReceipt); err != store.ErrMCPSubmissionConflict {
		t.Fatal("retained proposal bytes changed", err)
	}
	plan.ID = originalPlanID
	encodePlan()
	if settled, err := control.SettleMCPTerminalProposal(ctx, st, architect.ID, time.Now()); err != nil || !settled {
		t.Fatal("correct typed plan did not settle", err)
	}
	settledReceipt, err := st.MCPReceipt(ctx, architect.ID, proposal.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	settledBody := catalog.receiptBody(ctx, architect.ID, settledReceipt)
	settlement, ok := settledBody["settlement"].(map[string]any)
	if !ok || settlement["outcome"] != factory.Outcome(factory.Succeeded) || settlement["reported"] != true {
		t.Fatal("receipt hid the actual settled result", settledBody)
	}

	// Reports retained before typed plan admission remain immutable. Exercise
	// their exact replay through the authenticated SDK transport, where schema
	// validation must not prevent the handler from recovering the stored receipt.
	legacy := architect
	legacy.ID, legacy.Run = factory.NewID(), factory.NewID()
	legacy.AttemptRoot, legacy.PublicationAssignment = legacy.ID, legacy.ID
	legacy.RunHistory = []string{legacy.Run}
	raw, _ = json.Marshal(legacy)
	write(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,43,$2,'assigned',1,$3)`, legacy.ID, legacy.Run, string(raw))
	legacyCap := architectCap
	legacyCap.ID, legacyCap.AssignmentID, legacyCap.RunID = factory.NewID(), legacy.ID, legacy.Run
	if err := st.StageMCPCapability(ctx, legacyCap); err != nil {
		t.Fatal(err)
	}
	if err := st.ActivateObservedMCPCapability(ctx, legacyCap); err != nil {
		t.Fatal(err)
	}
	legacyCap, err = st.GetMCPCapability(ctx, legacyCap.ID)
	if err != nil {
		t.Fatal(err)
	}
	legacyPayload := map[string]any{"approach": "Retained legacy plan", "tasks": []any{"legacy task"}, "evidence_refs": []any{"README.md"}}
	legacyBytes, err := canonicalPayload(legacyPayload)
	if err != nil {
		t.Fatal(err)
	}
	legacyProposal := factory.MCPSubmission{AssignmentID: legacy.ID, CapabilityID: legacyCap.ID, SubmissionID: factory.NewID(), Tool: factory.MCPToolFinishIssuePlan, ExpectedInputDigest: legacyCap.ScopeDigest(), Payload: legacyBytes, PayloadHash: factory.MCPInputDigest(legacyBytes)}
	if _, err := st.RecordMCPSubmission(ctx, legacyProposal, factory.MCPReceipt{SubmissionID: legacyProposal.SubmissionID, Tool: legacyProposal.Tool, State: factory.MCPReceiptRecorded, InputDigest: legacyProposal.PayloadHash}); err != nil {
		t.Fatal(err)
	}
	legacySecret, err := factory.DeriveMCPCapabilitySecret(key, legacyCap)
	if err != nil {
		t.Fatal(err)
	}
	legacySession, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{legacyCap.ID, legacySecret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer legacySession.Close()
	beforeLegacyReplay := counts()
	recovered, err := legacySession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolReceipt), Arguments: map[string]any{"submission_id": legacyProposal.SubmissionID}})
	if catalogRefused(recovered, err) {
		t.Fatal("dedicated legacy receipt recovery refused", err)
	}
	for _, invalid := range []map[string]any{
		{"submission_id": legacyProposal.SubmissionID},
		{"payload": legacyPayload},
		{"submission_id": legacyProposal.SubmissionID, "payload": []any{}},
		{"submission_id": legacyProposal.SubmissionID, "payload": legacyPayload, "unknown": true},
		{"submission_id": legacyProposal.SubmissionID, "payload": legacyPayload, "capability_id": legacyCap.ID},
	} {
		refused, err := legacySession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishIssuePlan), Arguments: invalid})
		if !catalogRefused(refused, err) || !strings.Contains(catalogText(t, refused), "validating arguments:") || counts() != beforeLegacyReplay {
			t.Fatal("raw issue-plan registration weakened the bound envelope", err, catalogText(t, refused))
		}
	}
	replayed, err := legacySession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishIssuePlan), Arguments: map[string]any{"submission_id": legacyProposal.SubmissionID, "payload": legacyPayload}})
	if catalogRefused(replayed, err) {
		t.Fatal("SDK schema prevented retained legacy plan replay", err, catalogText(t, replayed))
	}
	if catalogText(t, replayed) != catalogText(t, recovered) || counts() != beforeLegacyReplay {
		t.Fatal("legacy replay changed its receipt, journal or budget")
	}
	legacyPayload["approach"] = "Changed legacy bytes"
	conflicted, err := legacySession.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishIssuePlan), Arguments: map[string]any{"submission_id": legacyProposal.SubmissionID, "payload": legacyPayload}})
	if !catalogRefused(conflicted, err) || !strings.Contains(catalogText(t, conflicted), store.ErrMCPSubmissionConflict.Error()) || counts() != beforeLegacyReplay {
		t.Fatal("legacy rewrite did not reach the immutable conflict guard", err, catalogText(t, conflicted))
	}

	// The normal owner overview reports only controller-adopted evidence for
	// the current accepted scope, independently of a successful worker finish.
	ownerRead := actorCall(t, &API{Config: &config.Config{ForgejoURL: "https://soda.test"}, Store: st}, "7", "current-worker")
	readOverview := func(want int) {
		t.Helper()
		before := verificationReadState()
		code, body := ownerRead(http.MethodGet, "/api/factory/intents/"+accepted.Intent+"/overview", "")
		var view factoryRequestOverviewView
		if code != http.StatusOK || json.Unmarshal(body, &view) != nil || view.Intent != accepted.Intent || view.WorkItems.OutcomesTotal != len(accepted.EffectiveOutcomes) || view.WorkItems.OutcomesEvidenced != want {
			t.Fatalf("owner overview evidence status=%d body=%s expected=%d", code, body, want)
		}
		if verificationReadState() != before {
			t.Fatal("overview read changed verification history or spending")
		}
	}
	readOverview(0)
	pilot.Stage, pilot.Outcome = factory.AssignmentFinished, factory.Succeeded
	pilot.Result = &factory.AssignmentResult{PhaseResult: &factory.PhaseResult{Output: &factory.PhaseOutput{Verification: &factory.AppVerificationReport{Outcome: factory.AppVerificationVerified}}}}
	raw, _ = json.Marshal(pilot)
	write(`UPDATE factory_assignments SET stage='finished',data=$2 WHERE id=$1`, pilot.ID, string(raw))
	readOverview(0)
	postID := factory.NewID()
	post := map[string]any{"actor": "factory", "assignment": pilot.ID, "event_identity": pilot.Subject.Delivery.DeliveryID, "communication": map[string]any{"component_id": "F10"}}
	raw, _ = json.Marshal(post)
	write(`INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data) VALUES($1,8,0,'factory','Application outcomes verified',$1,$2)`, postID, string(raw))
	readOverview(0)
	postID = factory.NewID()
	write(`INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data) VALUES($1,7,0,'factory','Application outcomes verified',$1,$2)`, postID, string(raw))
	readOverview(len(accepted.EffectiveOutcomes))
	pilot.Result.PhaseResult.Output.Verification.Outcome = factory.AppVerificationIncomplete
	raw, _ = json.Marshal(pilot)
	write(`UPDATE factory_assignments SET data=$2 WHERE id=$1`, pilot.ID, string(raw))
	readOverview(0)
	pilot.Result.PhaseResult.Output.Verification.Outcome = factory.AppVerificationVerified
	goodScope := *pilot.Subject.Delivery
	wrongScope := goodScope
	wrongScope.ScopeDigest = strings.Repeat("f", 64)
	pilot.Subject.Delivery = &wrongScope
	raw, _ = json.Marshal(pilot)
	write(`UPDATE factory_assignments SET data=$2 WHERE id=$1`, pilot.ID, string(raw))
	readOverview(0)
	pilot.Subject.Delivery = &goodScope
	raw, _ = json.Marshal(pilot)
	write(`UPDATE factory_assignments SET data=$2 WHERE id=$1`, pilot.ID, string(raw))
	previousAccepted := accepted
	nextAccepted := accepted
	nextAccepted.ID, nextAccepted.Predecessor = "d"+strings.Repeat("a", 24), accepted.ID
	if err := st.SaveIntentRevision(ctx, nextAccepted); err != nil {
		t.Fatal(err)
	}
	accepted = nextAccepted
	readOverview(0)
	if staleCount, err := st.VerifiedApplicationOutcomeCount(ctx, previousAccepted); err != nil || staleCount != 0 {
		t.Fatal("old accepted revision lent its F10 to the current overview", staleCount, err)
	}
}
