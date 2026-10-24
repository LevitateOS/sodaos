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
	if seen[string(factory.MCPToolFinishImplement)] {
		t.Fatalf("builder finish advertised to researcher: %v", seen)
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
		return map[string]any{"available": true, "evidence": verificationEvidence, "outcome": "native_check_passed"}, nil
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
	task := factory.IntentTask{Key: "d" + strings.Repeat("4", 24), Objective: "Persist tasks", Outcomes: []string{idea.Consequence}, IdeaIDs: []string{idea.ID}}
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
	if json.Unmarshal([]byte(verificationJSON), &currentVerification) != nil || currentVerification.ValidateFor(*pilot.Subject.Delivery) != nil || currentVerification.ApprovedContext.Files[0].Content != assertions || counts() != beforeVerification {
		t.Fatal("required current context lost assertions or spent/settled during a read")
	}
	verificationEvidence.ApprovedContext.Candidate = strings.Repeat("e", 40)
	result, err = pilotSession.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if !catalogRefused(result, err) || counts() != beforeVerification {
		t.Fatal("foreign verification source was projected or wrote during refusal")
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
}
