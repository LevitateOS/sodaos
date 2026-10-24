package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// planSourcePath is the approved source file the seeded preparation
// selects; the bounded host double returns synthetic bytes for exactly
// the requested paths, so the test approves their digest.
const planSourcePath = "internal/factory/intent.go"

func planSourceDigest() string {
	sum := sha256.Sum256([]byte("synthetic selected source: " + planSourcePath))
	return hex.EncodeToString(sum[:])
}

// planStubHost is the bounded host double for the draft-planning
// consumer proof: it records launches and answers nothing itself, so
// no provider call or native activation happens in the test.
type planStubHost struct {
	launches          []project.FactoryLaunch
	preparation       *project.PrepareState
	launchUnconfirmed bool
}

func (h *planStubHost) FactoryLaunch(_ context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	h.launches = append(h.launches, in)
	if h.launchUnconfirmed {
		return project.FactoryState{}, context.DeadlineExceeded
	}
	return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
}

func (h *planStubHost) FactoryInspect(_ context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	if h.launchUnconfirmed {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryRunning, Live: true}, nil
	}
	return project.FactoryState{}, host.ErrRunNotFound
}

func (h *planStubHost) FactoryHarness(_ context.Context, family string) (project.FactoryHarnessPin, error) {
	return project.FactoryHarnessPin{Harness: family, Version: "1.2.3", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
}

func (h *planStubHost) ReadPreparationContext(_ context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
	files := make([]project.FactoryContextFile, 0, len(in.Paths))
	for _, path := range in.Paths {
		files = append(files, project.FactoryContextFile{Path: path, Content: []byte("synthetic selected source: " + path)})
	}
	return project.FactoryPreparationContext{
		Project: in.Project, ID: in.ID, Role: role,
		SourceCommit: in.SourceCommit, ApprovedBase: in.ApprovedBase,
		DiffBase: in.DiffBase, Candidate: in.Candidate,
		Setup: []byte("synthetic approved setup instructions"),
		Check: []byte("synthetic required checks"),
		Files: files,
		Diff:  []byte("synthetic exact candidate diff"),
	}, nil
}

func (h *planStubHost) FactoryStop(_ context.Context, in project.FactoryStop) (project.FactoryState, error) {
	return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped, Retirement: "confirmed"}, nil
}

func (h *planStubHost) FactoryTakeover(_ context.Context, _ project.FactoryTakeover) (project.TakeoverResult, error) {
	return project.TakeoverResult{}, errors.New("unexpected host takeover")
}

func (h *planStubHost) FactoryExport(_ context.Context, _ project.FactoryExport) (project.FactoryExportState, error) {
	return project.FactoryExportState{}, errors.New("unexpected host export")
}

func (h *planStubHost) PrepareCandidate(_ context.Context, in project.FactoryCandidate) (project.PrepareState, error) {
	return project.PrepareState{ID: in.Preparation.ID, Project: in.Preparation.Project, Role: in.Preparation.Role, SourceCommit: in.Preparation.SourceCommit, SetupDigest: in.Preparation.SetupDigest, Container: strings.Repeat("c", 64), Phase: project.PrepareReady, Ready: true}, nil
}

func (h *planStubHost) InspectPreparation(_ context.Context, _ project.PrepareInspect) (project.PrepareState, error) {
	if h.preparation != nil {
		return *h.preparation, nil
	}
	return project.PrepareState{}, project.ErrPreparationNotFound
}

func (h *planStubHost) StopPreparation(_ context.Context, in project.PrepareStop) (project.PrepareState, error) {
	return project.PrepareState{ID: in.ID, Project: in.Project, Stopped: true, Retirement: "confirmed"}, nil
}

type planStubBroker struct{}

func (planStubBroker) GetExecution(context.Context, string, string) (identity.Execution, error) {
	return identity.Execution{}, identity.ErrNotFound
}

func (planStubBroker) CloseExecution(context.Context, string, string) error { return nil }

type planStubReads struct{}

func (planStubReads) ReadDispatchInputs(context.Context, string, string, []string, string) (control.DispatchInputs, error) {
	return control.DispatchInputs{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
}

// seedPlanFunding records the execution funding and authority the
// shared packet requires for repository 7, mirroring the dispatch
// fixture shape with the session operator as actor. With preparations,
// it admits the ready coder preparation selecting the approved source
// file the host double serves.
func seedPlanFunding(t *testing.T, ctx context.Context, s *store.Store) {
	seedPlanFundingWithPreparations(t, ctx, s, true)
}

func seedPlanFundingWithPreparations(t *testing.T, ctx context.Context, s *store.Store, preparations bool) {
	t.Helper()
	proj := "p765432109876543210987654"
	if err := s.CreateProject(ctx, store.Project{ID: proj, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	actor := func(kind string) factory.ActorBindingRef {
		return factory.ActorBindingRef{TokenID: 1, ActorID: 1, Kind: kind}
	}
	policy := factory.RepositoryPolicy{
		Repository: 7, GrantedBy: 1, Enabled: true, TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: project.FactoryHarnessCodex, HarnessVers: "1.2.3", Model: "test-model"},
			project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "1.2.3", Model: "test-model"},
		},
		Checks: []string{"ci"}, MergeMethod: factory.MergeFastForward,
		Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate),
		Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge),
		AttemptLimits:       factory.DefaultAttemptLimits(),
		ReassessmentMinutes: 30,
		MaxConcurrent:       2,
		IntentMinutes:       45,
	}
	if err := s.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 10}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: 7, GrantedBy: 1, MaxConcurrent: 2, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSponsorship(ctx, factory.Sponsorship{
		Repository: 7, GrantedBy: 1, ActorID: 1, ProjectID: proj, Generation: 1, Connection: "conn", GrantID: "grant",
		Roles: []string{project.RoleCoder}, AllowanceMinutes: 120, MaxConcurrent: 2, Active: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveConnectionUsageBudget(ctx, factory.ConnectionUsageBudget{Connection: "conn", RollingMinutes: factory.DefaultConnectionUsageBudgetMinutes}); err != nil {
		t.Fatal(err)
	}
	profile := &project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40)}
	if err := s.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{Repository: 7, Owner: 1, Profile: profile, Active: true}); err != nil {
		t.Fatal(err)
	}
	commit, digest := strings.Repeat("c", 40), strings.Repeat("d", 64)
	if err := s.AdmitRequirementDecision(ctx, project.RequirementDecision{
		ID: "d111111111111111111111111", Project: proj, Approver: 1, SourceCommit: commit, SetupDigest: digest, InputsDigest: digest,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.AdmitApprovalDecision(ctx, project.ApprovalDecision{
		ID: "d222222222222222222222222", Project: proj, Requirement: "d111111111111111111111111", Approver: 1,
		EffectsDigest: digest, ReadinessDigest: digest, Verified: true,
	}); err != nil {
		t.Fatal(err)
	}
	if !preparations {
		return
	}
	prep := project.StoredPreparation{Preparation: project.Preparation{
		ID: "f111111111111111111111111", Project: proj, Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest, SourceInputs: map[string]string{planSourcePath: planSourceDigest()}},
		Approval:     project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 1, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	prep.State = project.PrepareState{ID: "f111111111111111111111111", Project: proj, Role: project.RoleCoder, Phase: project.PrepareReady, Ready: true}
	if _, _, err := s.AdmitPreparation(ctx, prep); err != nil {
		t.Fatal(err)
	}
	review := project.StoredPreparation{Preparation: project.Preparation{
		ID: "f222222222222222222222222", Project: proj, Role: project.RoleReviewer,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest},
		Approval:     project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 1, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	review.State = project.PrepareState{ID: "f222222222222222222222222", Project: proj, Role: project.RoleReviewer, Phase: project.PrepareReady, Ready: true}
	if _, _, err := s.AdmitPreparation(ctx, review); err != nil {
		t.Fatal(err)
	}
}

const planDraftBody = `{"id":"d888888888888888888888888","intent":"d777777777777777777777777","objective":"Plan the recorded draft","state":"draft","repository":"7","max_issues":10,"planning_calls":3,` +
	`"ideas":[{"id":"d333333333333333333333333","description":"Capture leave requests","consequence":"Requests can be recorded","decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
	`{"id":"d444444444444444444444444","description":"Sync to the legacy tracker","consequence":"Legacy stays current","decision":"edit","correction":"Sync only on explicit manager request","source":"requested","provenance":"human","actor":"1"},` +
	`{"id":"d555555555555555555555555","description":"Delete requests automatically","consequence":"Old requests vanish","decision":"decline","source":"requested","provenance":"human","actor":"1"}]}`

func planRequestBody(revision string) string {
	return `{"revision":"` + revision + `","operation":"propose_intent"}`
}

// TestIntentPlanConsumerLaunchesBoundDecompose proves the authenticated
// Generate-adjacent draft-planning caller end to end: the typed product
// route binds the session owner to the current head, stages the
// complete input and launches through the shared dispatch path with a
// bounded host double, while stale, non-draft and malformed callers
// refuse and missing live capabilities fail truthfully.
func TestIntentPlanConsumerLaunchesBoundDecompose(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	ctx := context.Background()

	if code, body := call(http.MethodPost, "/api/factory/intents", planDraftBody); code != http.StatusCreated {
		t.Fatalf("draft save status=%d body=%s", code, body)
	}
	const intentID, draftID = "d777777777777777777777777", "d888888888888888888888888"

	// Stale and malformed callers refuse before any launch. The caller
	// binds only revision and operation; source and evidence are
	// server-owned, so no caller commit is accepted.
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", planRequestBody("d999999999999999999999999")); code != http.StatusPreconditionFailed || errorCode(body) != "stale_revision" {
		t.Fatalf("stale status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", `{"revision":"`+draftID+`","operation":"nope"}`); code != http.StatusBadRequest || errorCode(body) != "invalid_operation" {
		t.Fatalf("operation status=%d body=%s", code, body)
	}

	// Missing live capabilities refuse truthfully: no enrolled host.
	product.Coordinator = &control.Coordinator{Store: s}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", planRequestBody(draftID)); code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" {
		t.Fatalf("unavailable status=%d body=%s", code, body)
	}

	// The wired consumer launches through the shared path.
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", planRequestBody(draftID))
	if code != http.StatusCreated {
		t.Fatalf("plan status=%d body=%s", code, body)
	}
	var responded intentPlanResponse
	if err := json.Unmarshal(body, &responded); err != nil {
		t.Fatal(err)
	}
	if responded.RevisionID != draftID || responded.AssignmentID == "" || responded.RunID == "" {
		t.Fatalf("plan response not exact: %+v", responded)
	}
	if len(stub.launches) != 1 {
		t.Fatalf("host launches = %d", len(stub.launches))
	}
	launched, err := s.Assignment(ctx, responded.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	head, err := s.IntentRevision(ctx, intentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if launched.Subject.Intent == nil || launched.Subject.Intent.ScopeDigest != head.Digest() {
		t.Fatalf("launched subject not bound to its head: %+v", launched.Subject)
	}
	for _, want := range []string{"Sync only on explicit manager request", "Delete requests automatically", head.Digest(), "synthetic selected source: " + planSourcePath} {
		if !strings.Contains(string(launched.Prompt), want) {
			t.Fatalf("staged prompt drops %q", want)
		}
	}
	// The production catalog is reached through an authenticated live worker,
	// not a test catalog or worker-selected executable. It binds the protected
	// preparation, exact draft and native check receipt before any new finish.
	preparation, err := s.Preparation(ctx, launched.Preparation)
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.FactoryRun(ctx, launched.Run)
	if err != nil {
		t.Fatal(err)
	}
	cap, err := control.BuildMCPCapability(launched, run, preparation)
	if err != nil {
		t.Fatal(err)
	}
	selection := newFakeCatalogSink()
	selection.capability = cap
	attemptEligible, lineageEligible := factory.BundleEligibility(launched, false)
	storeSelectionReceipt(t, selection, launched, attemptEligible, lineageEligible, factory.ContextBundleCurrent)
	cap = selection.capability
	if err := s.StageMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	staged, err := s.GetMCPCapability(ctx, cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if staged.ContextSelectionSnapshot == "" {
		if reserved, err := s.ReserveContextBundleSelection(ctx, staged, cap.ContextSelectionSnapshot); err != nil || !reserved {
			t.Fatalf("fixture context selection was not admitted once: %v/%v", reserved, err)
		}
	}
	if err := s.ActivateObservedMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	cap, err = s.GetMCPCapability(ctx, cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cap.ContextSelectionSnapshot != selection.capability.ContextSelectionSnapshot {
		t.Fatal("activated fixture lost its exact context point")
	}
	zero := 0
	stub.preparation = &project.PrepareState{ID: preparation.Preparation.ID, Project: preparation.Preparation.Project, Role: preparation.Preparation.Role, SourceCommit: preparation.Preparation.SourceCommit, SetupDigest: preparation.Preparation.SetupDigest, Ready: true, Phase: project.PrepareReady, SetupExit: &zero, CheckExit: &zero}
	catalog, err := product.factoryApprovedVerificationCatalog(ctx, cap)
	if err != nil {
		t.Fatalf("production capability catalog: %v", err)
	}
	if _, ok := catalog.LookupCheck(factory.ApprovedAppCheckID); !ok {
		t.Fatal("approved project.check absent")
	}
	for _, fault := range []string{"source", "setup", "subject", "revision", "project"} {
		wrong := cap
		switch fault {
		case "source":
			wrong.SourceDigest = strings.Repeat("f", 40)
		case "setup":
			wrong.PreparationDigest = strings.Repeat("f", 64)
		case "subject":
			wrong.Subject += "-foreign"
		case "revision":
			wrong.AcceptedRevision = "999"
		case "project":
			wrong.Project = "p" + strings.Repeat("f", 24)
		}
		if _, err := product.factoryApprovedVerificationCatalog(ctx, wrong); err == nil {
			t.Fatalf("production catalog accepted wrong %s", fault)
		}
	}
	key := []byte(strings.Repeat("k", 32))
	secret, err := factory.DeriveMCPCapabilitySecret(key, cap)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: s, Submissions: s, Assignments: s, Runs: s, PriorityJudgments: selection, Catalog: product.factoryApprovedVerificationCatalog}, key))
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "approved-pl-catalog-pg"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{cap.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	contextResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if catalogRefused(contextResult, err) {
		t.Fatalf("catalog context read refused: %v", err)
	}
	var contextView struct {
		Catalog struct {
			Available bool                                `json:"available"`
			Entries   []factory.ApprovedCheck             `json:"entries"`
			Binding   factory.ApprovedProjectCheckCatalog `json:"binding"`
		} `json:"verification_catalog"`
	}
	if err := json.Unmarshal([]byte(catalogText(t, contextResult)), &contextView); err != nil || !contextView.Catalog.Available || len(contextView.Catalog.Entries) != 1 || contextView.Catalog.Entries[0].ID != factory.ApprovedAppCheckID || contextView.Catalog.Binding.Preparation != launched.Preparation || contextView.Catalog.Binding.CheckDigest != factory.MCPInputDigest([]byte("synthetic required checks")) {
		t.Fatal("worker context omitted the actual approved check catalog", err)
	}
	proposal := factory.VerificationManifestProposal{Outcomes: []string{head.Ideas[0].ID, head.Ideas[1].ID}}
	for _, outcome := range proposal.Outcomes {
		proposal.Checks = append(proposal.Checks, factory.ManifestCheckRef{Outcome: outcome, CatalogID: factory.ApprovedAppCheckID, Artifacts: []string{factory.ProjectCheckReceiptArtifact}, Fixtures: []string{factory.ProjectCheckPreparedFixture}})
	}
	candidates := []factory.SemanticCandidate{{Key: "one", Rationale: "Preserve the reviewed product choices.", Tasks: []factory.IntentTask{{Key: "d" + strings.Repeat("9", 24), Objective: "Record requests with explicit legacy synchronization", Outcomes: []string{"Requests can be recorded", "Legacy synchronization is explicit"}, IdeaIDs: proposal.Outcomes}}}}
	finish := func(id string, p any) (*mcp.CallToolResult, error) {
		return session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": id, "payload": map[string]any{"candidates": candidates, "coverage": "Every effective outcome uses the approved Project check.", "evidence_refs": []string{"approved preparation"}, "verification_manifest_proposal": p}}})
	}
	for _, fault := range []string{"catalog", "outcome", "artifact", "fixture", "missing-fixture", "parameter", "native-check"} {
		badJSON, _ := json.Marshal(proposal)
		var bad factory.VerificationManifestProposal
		_ = json.Unmarshal(badJSON, &bad)
		switch fault {
		case "catalog":
			bad.Checks[0].CatalogID = "worker.shell"
		case "outcome":
			bad.Outcomes[0] = "foreign"
			bad.Checks[0].Outcome = "foreign"
		case "artifact":
			bad.Checks[0].Artifacts = []string{"/private/file"}
		case "fixture":
			bad.Checks[0].Fixtures = []string{"foreign-team"}
		case "missing-fixture":
			bad.Checks[0].Fixtures = nil
		case "parameter":
			bad.Checks[0].Params = map[string]string{"command": "arbitrary"}
		case "native-check":
			failed := 1
			stub.preparation.CheckExit = &failed
		}
		id := factory.NewID()
		result, err := finish(id, bad)
		if !catalogRefused(result, err) {
			t.Fatalf("new manifest admitted %s", fault)
		}
		if _, err := s.MCPReceipt(ctx, launched.ID, id); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("refused %s wrote receipt: %v", fault, err)
		}
		stub.preparation.CheckExit = &zero
	}
	id := factory.NewID()
	result, err := finish(id, proposal)
	if catalogRefused(result, err) {
		t.Fatalf("approved manifest finish refused: %v %v", catalogText(t, result), err)
	}
	if err := s.AdmitApprovalDecision(ctx, project.ApprovalDecision{ID: "d" + strings.Repeat("a", 24), Predecessor: preparation.Preparation.Approval.ID, Project: cap.Project, Requirement: preparation.Preparation.Requirements.ID, Approver: 1, EffectsDigest: strings.Repeat("d", 64), ReadinessDigest: strings.Repeat("d", 64), Verified: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := product.factoryApprovedVerificationCatalog(ctx, cap); err == nil {
		t.Fatal("moved approval retained current catalog")
	}
	result, err = finish(id, proposal)
	if catalogRefused(result, err) {
		t.Fatalf("changed approval blocked exact immutable finish replay: %v", err)
	}
	changed := proposal
	changed.Checks = append([]factory.ManifestCheckRef(nil), proposal.Checks...)
	changed.Checks[0].CatalogID = "foreign"
	result, err = finish(id, changed)
	if !catalogRefused(result, err) {
		t.Fatal("same submission identity accepted changed manifest bytes")
	}
	next := head
	next.ID, next.Predecessor = "d"+strings.Repeat("f", 24), head.ID
	if err := s.SaveIntentRevision(ctx, next); err != nil {
		t.Fatal(err)
	}
	if _, err := product.factoryApprovedVerificationCatalog(ctx, cap); err == nil || !strings.Contains(err.Error(), "draft changed") {
		t.Fatal("moved draft retained planning catalog", err)
	}
	result, err = finish(id, proposal)
	if catalogRefused(result, err) {
		t.Fatalf("moved draft blocked immutable proposal replay: %v", err)
	}
	if settled, err := control.SettleMCPTerminalProposal(ctx, s, launched.ID, time.Now()); err != nil || !settled {
		t.Fatalf("approved MCP plan did not settle: %v/%v", settled, err)
	}
	finished, err := s.Assignment(ctx, launched.ID)
	if err != nil || finished.Result == nil || finished.Result.PhaseResult == nil || finished.Result.PhaseResult.Output == nil || finished.Result.PhaseResult.Output.Decomposition == nil || finished.Result.PhaseResult.Output.Decomposition.VerificationManifestProposal == nil || finished.Result.PhaseResult.Output.Decomposition.VerificationManifestProposal.Digest() != proposal.Digest() {
		t.Fatal("actual MCP settlement dropped proposal", err)
	}
}

// TestIntentPlanWaitsWithoutReadyPreparation proves missing preparation
// waits truthfully: with funding but no ready coder preparation, the
// route answers 202 with waits, leaving no run, assignment or allowance
// behind.
func TestIntentPlanWaitsWithoutReadyPreparation(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", planDraftBody); code != http.StatusCreated {
		t.Fatalf("save: %d %s", code, body)
	}
	seedPlanFundingWithPreparations(t, ctx, s, false)
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusAccepted {
		t.Fatalf("missing preparation status=%d body=%s", code, body)
	}
	var waited struct {
		Launched bool `json:"launched"`
		Waits    []struct {
			Reason string `json:"reason"`
		} `json:"waits"`
	}
	if err := json.Unmarshal(body, &waited); err != nil {
		t.Fatal(err)
	}
	if waited.Launched || len(waited.Waits) == 0 || waited.Waits[0].Reason == "" {
		t.Fatalf("missing preparation did not wait: %+v", waited)
	}
	if len(stub.launches) != 0 {
		t.Fatalf("missing preparation launched: %d", len(stub.launches))
	}
	if _, err := s.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: "d777777777777777777777777", OwnerID: 1}, ""); err == nil {
		t.Fatal("missing preparation funded an allowance")
	}
}

// TestIntentPlanRefusesMismatchedPreparationContext proves server-assembled
// evidence is verified against its approval: host bytes that do not match
// the approved source selection refuse before funding or launch.
func TestIntentPlanRefusesMismatchedPreparationContext(t *testing.T) {
	product, s, call, errorCode := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", planDraftBody); code != http.StatusCreated {
		t.Fatalf("save: %d %s", code, body)
	}
	seedPlanFundingWithPreparations(t, ctx, s, false)
	commit, digest := strings.Repeat("c", 40), strings.Repeat("d", 64)
	bad := project.StoredPreparation{Preparation: project.Preparation{
		ID: "f111111111111111111111111", Project: "p765432109876543210987654", Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest, SourceInputs: map[string]string{planSourcePath: digest}},
		Approval:     project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 1, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	bad.State = project.PrepareState{ID: "f111111111111111111111111", Project: "p765432109876543210987654", Role: project.RoleCoder, Phase: project.PrepareReady, Ready: true}
	if _, _, err := s.AdmitPreparation(ctx, bad); err != nil {
		t.Fatal(err)
	}
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusServiceUnavailable || errorCode(body) != "planning_unavailable" {
		t.Fatalf("mismatched context status=%d body=%s", code, body)
	}
	if len(stub.launches) != 0 {
		t.Fatalf("mismatched context launched: %d", len(stub.launches))
	}
	if _, err := s.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: "d777777777777777777777777", OwnerID: 1}, ""); err == nil {
		t.Fatal("mismatched context funded an allowance")
	}
}

// planSettledAnswer fences one decompose answer for the saved draft head:
// a covering single candidate, or attention-only when attention is set.
func planSettledAnswer(t *testing.T, status, attention string) string {
	t.Helper()
	answer := &factory.SemanticPlannerOutput{
		Intent: "d777777777777777777777777", Revision: "d888888888888888888888888", Operation: factory.IntentPlanningPropose,
	}
	if attention != "" {
		answer.Attention = attention
	} else {
		answer.Candidates = []factory.SemanticCandidate{{
			Key: "keep", Rationale: "Keep the recorded draft as the single review baseline.",
			Tasks: []factory.IntentTask{{
				Key: "d999999999999999999999999", Objective: "Plan the recorded draft",
				Outcomes: []string{"Draft is planned"}, IdeaIDs: []string{"d333333333333333333333333"},
			}},
		}}
	}
	data, err := json.Marshal(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: status,
		Summary: "planning update", Output: &factory.PhaseOutput{Decomposition: answer},
	})
	if err != nil {
		t.Fatal(err)
	}
	return "work log\n```result-json\n" + string(data) + "\n```\ntrailer\n"
}

func settlePlanRun(t *testing.T, ctx context.Context, s *store.Store, runID, answer string) {
	t.Helper()
	run, err := s.FactoryRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "planning done", true
	if err := s.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, ok := control.AccountSettledRun(ctx, s, run, answer, time.Now()); !ok {
		t.Fatal("plan run unsettled")
	}
}

// TestIntentPlanStatusReadsAttemptLifecycle proves the minimal scoped
// read: none before the first attempt, pending while the launched run
// is outstanding, and the reported completed shape with its candidate
// count once the run settles.
func TestIntentPlanStatusReadsAttemptLifecycle(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", planDraftBody); code != http.StatusCreated {
		t.Fatalf("save: %d %s", code, body)
	}
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{launchUnconfirmed: true}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	const statusURL = "/api/factory/intents/d777777777777777777777777/plan"
	if code, body := call(http.MethodGet, statusURL, ""); code != http.StatusOK {
		t.Fatalf("empty status=%d body=%s", code, body)
	} else {
		var viewed struct {
			Revision string `json:"revision"`
			State    string `json:"state"`
		}
		if err := json.Unmarshal(body, &viewed); err != nil {
			t.Fatal(err)
		}
		if viewed.Revision != "d888888888888888888888888" || viewed.State != "none" {
			t.Fatalf("empty status not exact: %+v", viewed)
		}
	}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusAccepted {
		t.Fatalf("plan status=%d body=%s", code, body)
	}
	var launched struct {
		RunID           string `json:"run_id"`
		LaunchConfirmed *bool  `json:"launch_confirmed"`
	}
	if err := json.Unmarshal(body, &launched); err != nil {
		t.Fatal(err)
	}
	if launched.LaunchConfirmed == nil || *launched.LaunchConfirmed || len(stub.launches) != 1 {
		t.Fatalf("uncertain launch did not retain exactly one admitted run: %s", body)
	}
	if replayCode, replayBody := call(http.MethodPost, statusURL, planRequestBody("d888888888888888888888888")); replayCode != http.StatusAccepted || len(stub.launches) != 1 {
		t.Fatalf("uncertain launch replay duplicated work or failed recovery: %d %s launches=%d", replayCode, replayBody, len(stub.launches))
	}
	if code, body := call(http.MethodGet, statusURL, ""); code != http.StatusOK {
		t.Fatalf("pending status=%d body=%s", code, body)
	} else {
		var viewed struct {
			State        string `json:"state"`
			AssignmentID string `json:"assignment_id"`
			RunID        string `json:"run_id"`
		}
		if err := json.Unmarshal(body, &viewed); err != nil {
			t.Fatal(err)
		}
		if viewed.State != "pending" || viewed.AssignmentID == "" || viewed.RunID != launched.RunID {
			t.Fatalf("pending status not exact: %+v", viewed)
		}
	}
	settlePlanRun(t, ctx, s, launched.RunID, planSettledAnswer(t, "completed", ""))
	code, body = call(http.MethodGet, statusURL, "")
	if code != http.StatusOK {
		t.Fatalf("settled status=%d body=%s", code, body)
	}
	var viewed struct {
		State      string `json:"state"`
		Summary    string `json:"summary"`
		Candidates int    `json:"candidates"`
	}
	if err := json.Unmarshal(body, &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.State != "completed" || viewed.Candidates != 1 || viewed.Summary == "" {
		t.Fatalf("settled status not exact: %+v", viewed)
	}
}

// TestIntentPlanStatusReadsAttention proves a blocked plan surfaces as
// attention with its reported text preserved and no candidates.
func TestIntentPlanStatusReadsAttention(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", planDraftBody); code != http.StatusCreated {
		t.Fatalf("save: %d %s", code, body)
	}
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusCreated {
		t.Fatalf("plan status=%d body=%s", code, body)
	}
	var launched struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(body, &launched); err != nil {
		t.Fatal(err)
	}
	settlePlanRun(t, ctx, s, launched.RunID, planSettledAnswer(t, "blocked", "Decide whether managers review in the same queue before issues can split."))
	code, body = call(http.MethodGet, "/api/factory/intents/d777777777777777777777777/plan", "")
	if code != http.StatusOK {
		t.Fatalf("attention status=%d body=%s", code, body)
	}
	var viewed struct {
		State      string `json:"state"`
		Attention  string `json:"attention"`
		Candidates int    `json:"candidates"`
	}
	if err := json.Unmarshal(body, &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.State != "attention" || viewed.Candidates != 0 || viewed.Attention != "Decide whether managers review in the same queue before issues can split." {
		t.Fatalf("attention status not exact: %+v", viewed)
	}
}
