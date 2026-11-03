package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/levitateos/factory/pkg/factory/execution"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factorymcp "github.com/levitateos/factory/pkg/factory/mcpserver"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/identity"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// planSourcePath is the approved source file the seeded preparation
// selects; the bounded host double returns synthetic bytes for exactly
// the requested paths, so the test approves their digest.
const planSourcePath = "internal/factory/intent.go"

var planSourceContent = "synthetic selected source: " + planSourcePath + strings.Repeat("x", project.MaxFactoryContextFileBytes-len("synthetic selected source: "+planSourcePath))

func planSourceDigest() string {
	sum := sha256.Sum256([]byte(planSourceContent))
	return hex.EncodeToString(sum[:])
}

// planStubHost is the bounded host double for the draft-planning
// consumer proof: it records launches and answers nothing itself, so
// no provider call or native activation happens in the test.
type planStubHost struct {
	launches          []project.FactoryLaunch
	preparation       *execution.PreparationState
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

func (h *planStubHost) InspectTool(ctx context.Context, family string) (execution.ToolObservation, error) {
	return host.NewFactoryObserver(h, nil).InspectTool(ctx, family)
}

func (h *planStubHost) ReadWorkspaceContext(_ context.Context, scoped execution.PreparationContextRequest) (execution.PreparationContext, error) {
	in, role := scoped.Input, scoped.Role
	files := make([]project.FactoryContextFile, 0, len(in.Paths))
	for _, path := range in.Paths {
		content := "synthetic selected source: " + path
		if path == planSourcePath {
			content = planSourceContent
		}
		files = append(files, project.FactoryContextFile{Path: path, Content: []byte(content)})
	}
	return execution.PreparationContext{Material: project.FactoryPreparationContext{
		Project: in.Project, ID: in.ID, Role: role,
		SourceCommit: in.SourceCommit, ApprovedBase: in.ApprovedBase,
		DiffBase: in.DiffBase, Candidate: in.Candidate,
		Setup: []byte("synthetic approved setup instructions"),
		Check: []byte("synthetic required checks"),
		Files: files,
		Diff:  []byte("synthetic exact candidate diff"),
	}}, nil
}

func (h *planStubHost) FactoryStop(_ context.Context, in project.FactoryStop) (project.FactoryState, error) {
	return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped, Retirement: "confirmed"}, nil
}

func (h *planStubHost) FactoryTakeover(_ context.Context, _ project.FactoryTakeover) (project.TakeoverResult, error) {
	return project.TakeoverResult{}, errors.New("unexpected host takeover")
}

func (h *planStubHost) RecoverWork(ctx context.Context, r execution.WorkRecoveryRequest) (execution.WorkRecoveryReceipt, error) {
	return host.NewFactoryObserver(h, nil).RecoverWork(ctx, r)
}

func (h *planStubHost) FactoryExport(_ context.Context, _ project.FactoryExport) (project.FactoryExportState, error) {
	return project.FactoryExportState{}, errors.New("unexpected host export")
}

func (*planStubHost) PrepareWorkspace(context.Context, execution.StandingPreparationRequest) (execution.PreparationState, error) {
	return execution.PreparationState{}, errors.New("unexpected standing preparation")
}

func (h *planStubHost) PrepareWorkspaceCandidate(_ context.Context, request execution.CandidatePreparationRequest) (execution.PreparationState, error) {
	in := request.Input
	return execution.PreparationState{ID: in.Preparation.ID, Project: in.Preparation.Project, Role: in.Preparation.Role, SourceCommit: in.Preparation.SourceCommit, SetupDigest: in.Preparation.SetupDigest, Container: strings.Repeat("c", 64), Phase: project.PrepareReady, Ready: true}, nil
}

func (h *planStubHost) InspectWorkspacePreparation(_ context.Context, _ execution.PreparationAddress) (execution.PreparationState, error) {
	if h.preparation != nil {
		return *h.preparation, nil
	}
	return execution.PreparationState{}, execution.ErrPreparationNotObserved
}

func (h *planStubHost) RetireWorkspacePreparation(_ context.Context, address execution.PreparationAddress) (execution.PreparationState, error) {
	in := project.PrepareStop{Project: address.Workspace, ID: address.ID}
	return execution.PreparationState{ID: in.ID, Project: in.Project, Stopped: true, Retirement: "confirmed"}, nil
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
			project.RoleCoder:    {Harness: project.FactoryHarnessMuse, HarnessVers: "1.2.3", Model: "test-model"},
			project.RoleReviewer: {Harness: project.FactoryHarnessMuse, HarnessVers: "1.2.3", Model: "test-model"},
		},
		Checks: []string{"ci"}, MergeMethod: factory.MergeFastForward,
		Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate),
		Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge),
		AttemptLimits:       factory.DefaultAttemptLimits(),
		ReassessmentMinutes: 30,
		IntentMinutes:       45,
	}
	if err := s.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: 7, GrantedBy: 1, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSponsorship(ctx, factory.Sponsorship{
		Repository: 7, GrantedBy: 1, ActorID: 1, ProjectID: proj, Generation: 1, Connection: "conn", GrantID: "grant",
		Roles: []string{project.RoleCoder}, AllowanceMinutes: 120, Active: true,
	}); err != nil {
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
	prep := execution.PreparationRecord{Preparation: project.Preparation{
		ID: "f111111111111111111111111", Project: proj, Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest, SourceInputs: map[string]string{planSourcePath: planSourceDigest()}},
		Approval:     project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 1, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	prep.State = execution.PreparationState{Container: strings.Repeat("c", 64), SourceCommit: prep.Preparation.SourceCommit, SetupDigest: prep.Preparation.SetupDigest, ID: "f111111111111111111111111", Project: proj, Role: project.RoleCoder, Phase: project.PrepareReady, Ready: true}
	if _, _, err := s.AdmitPreparation(ctx, prep); err != nil {
		t.Fatal(err)
	}
	review := execution.PreparationRecord{Preparation: project.Preparation{
		ID: "f222222222222222222222222", Project: proj, Role: project.RoleReviewer,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest},
		Approval:     project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 1, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	review.State = execution.PreparationState{Container: strings.Repeat("c", 64), SourceCommit: review.Preparation.SourceCommit, SetupDigest: review.Preparation.SetupDigest, ID: "f222222222222222222222222", Project: proj, Role: project.RoleReviewer, Phase: project.PrepareReady, Ready: true}
	if _, _, err := s.AdmitPreparation(ctx, review); err != nil {
		t.Fatal(err)
	}
}

const planDraftBody = `{"id":"d888888888888888888888888","intent":"d777777777777777777777777","objective":"Plan the recorded draft","state":"draft","repository":"7","max_issues":10,"planning_calls":3,` +
	`"ideas":[{"id":"d333333333333333333333333","description":"Capture leave requests","consequence":"Requests can be recorded","decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
	`{"id":"d444444444444444444444444","description":"Sync to the legacy tracker","consequence":"Legacy stays current","decision":"edit","correction":"Sync only on explicit manager request","source":"requested","provenance":"human","actor":"1"},` +
	`{"id":"d555555555555555555555555","description":"Delete requests automatically","consequence":"Old requests vanish","decision":"decline","source":"requested","provenance":"human","actor":"1"}]}`

func TestIssueMakerBoundMCPPublishesIssuesDuringItsOwnRun(t *testing.T) {
	product, s, dsn, _, _ := q05MuxFull(t, "")
	ctx := context.Background()
	if err := s.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{}
	graph := &intakeNativePeer{}
	product.IssueMakerNative = graph
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}, Graph: graph, GraphActorID: 9}
	id, revision, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "Add login and notification preferences", IdempotencyKey: "maker-mcp"}, "1", "alice", time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordIssueMakerIntake(ctx, id, revision); err != nil {
		t.Fatal(err)
	}
	graph.identities = seedIntakeGrant(t, s, ctx, id)
	wired := &factoryservice.PostCollaborators{Judge: &fakePostJudge{judgment: factoryservice.PostJudgment{Route: factory.PostRouteNewIssue}}, IssueMaker: product.FactoryApplication().IssueMaker().Drive}
	if _, err = factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	if len(stub.launches) != 0 || len(graph.issues) != 1 {
		t.Fatal("initial capture waited for a model run")
	}
	if _, err = factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	if len(stub.launches) != 1 {
		progress, _ := s.PostIssueMaker(ctx, id, revision)
		t.Fatal("Issue Maker did not launch through the funded account", progress)
	}
	attempt, err := s.IntentPlanAttempt(ctx, factory.PostDecisionID("intent", id, revision), factory.PostDecisionID("revision", id, revision))
	if err != nil {
		t.Fatal(err)
	}
	preparation, err := s.Preparation(ctx, attempt.Preparation)
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.FactoryRun(ctx, attempt.Run)
	if err != nil {
		t.Fatal(err)
	}
	cap, err := control.BuildMCPCapability(attempt, run, preparation)
	if err != nil {
		t.Fatal(err)
	}
	if cap.Profile != factory.MCPProfileIssueMaker || !strings.Contains(string(attempt.Prompt), "Issue Maker assignment") || strings.Contains(string(attempt.Prompt), "Product walkthrough authored") {
		t.Fatal("intake launched as Product Lead instead of Issue Maker", cap.Profile)
	}
	selection := newFakeCatalogSink()
	selection.capability = cap
	attemptEligible, lineageEligible := factory.BundleEligibility(attempt, false)
	storeSelectionReceipt(t, selection, attempt, attemptEligible, lineageEligible, factory.ContextBundleCurrent)
	cap = selection.capability
	if err = s.StageMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	if reserved, err := s.BindContextBundleSelection(ctx, cap, cap.ContextSelectionSnapshot); err != nil || !reserved {
		t.Fatal("context selection", reserved, err)
	}
	if err = s.ActivateObservedMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	cap, err = s.GetMCPCapability(ctx, cap.ID)
	if err != nil {
		t.Fatal(err)
	}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, cap)
	server := httptest.NewServer(factorymcp.NewHandler(factorymcp.Dependencies{Capabilities: s, Submissions: s, Assignments: s, Runs: s, PriorityJudgments: selection, IssueMaker: product.factoryIssueMakerTool}, key))
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "issue-maker-bound-pg"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + factoryhttpapi.WorkerRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{cap.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, tool := range tools.Tools {
		listed[tool.Name] = true
	}
	if !listed["factory.create_issue"] || !listed["factory.refine_issue"] || listed["factory.start_task"] {
		t.Fatal("Issue Maker discovery lost its scoped tools", listed)
	}
	if factory.MCPToolAdmitted(factory.MCPProfileProductLead, factory.MCPToolCreateIssue) || factory.MCPToolAdmitted(factory.MCPProfileBuilder, factory.MCPToolRefineIssue) {
		t.Fatal("other roles gained intake writes")
	}
	call := func(tool, id, key, title, body string) *mcp.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: map[string]any{"submission_id": id, "payload": map[string]any{"key": key, "title": title, "body": body}}})
		if catalogRefused(result, err) {
			t.Fatalf("%s: %v %s", tool, err, catalogText(t, result))
		}
		var reply map[string]any
		if err := json.Unmarshal([]byte(catalogText(t, result)), &reply); err != nil {
			t.Fatal(err)
		}
		if reply["issue"] == nil || reply["post_id"] == nil || reply["receipt"] != nil {
			t.Fatal("issue action must return its native issue and post", reply)
		}
		return result
	}
	call("factory.create_issue", "notifications", "notifications", "Notification preferences", "Allow the user to select notification preferences.")
	call("factory.create_issue", "notifications", "notifications", "Notification preferences", "Allow the user to select notification preferences.")
	call("factory.refine_issue", "refine-login", "request", "Login screen", "Provide a login screen. Acceptance: valid credentials open the app.")
	message, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolPartial), Arguments: map[string]any{"submission_id": "intake-progress", "payload": map[string]any{"title": "Issues captured", "summary": "Login and notification issues are ready for their own work.", "completed": []string{"Captured login and notification issues"}}}})
	if catalogRefused(message, err) {
		t.Fatal("worker message did not write its post", err, catalogText(t, message))
	}
	var posted map[string]any
	if err := json.Unmarshal([]byte(catalogText(t, message)), &posted); err != nil || posted["post_id"] == nil || posted["receipt"] != nil || posted["submission_id"] != nil {
		t.Fatal("worker message did not return the post directly", posted, err)
	}
	post, err := s.PostIssueMaker(ctx, id, revision)
	if err != nil {
		t.Fatal(err)
	}
	replies, _, err := s.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil || len(replies) != 3 || len(graph.issues) != 2 || graph.issues[1].Title != "Login screen" {
		t.Fatal("live MCP writes did not publish one comment per issue", replies, len(graph.issues), err)
	}
	if attempt, err = s.Assignment(ctx, attempt.ID); err != nil || attempt.Stage != factory.AssignmentAssigned {
		t.Fatal("issues waited for the complete run", attempt.Stage, err)
	}
	if _, err = s.MCPDiscussion(ctx, cap, post.Scope.Post, ""); err != nil {
		t.Fatal("Issue Maker cannot read its own progress thread", err)
	}
	entries, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == attempt.ID {
			t.Fatal("internal intake run became a second feed post")
		}
	}
	artifactBody := "Captured issue context."
	artifactResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolPublishArtifact), Arguments: map[string]any{"payload": factory.ArtifactUpload{UploadID: factory.NewID(), Name: "intake.txt", MediaType: "text/plain", ContentBase64: base64.StdEncoding.EncodeToString([]byte(artifactBody)), Final: true}}})
	if catalogRefused(artifactResult, err) {
		t.Fatal("artifact upload required a submission receipt", err, catalogText(t, artifactResult))
	}
	var uploaded struct {
		Artifact   factory.FeedArtifact `json:"artifact"`
		PostID     string               `json:"post_id"`
		NextOffset int                  `json:"next_offset"`
		Complete   bool                 `json:"complete"`
	}
	if err := json.Unmarshal([]byte(catalogText(t, artifactResult)), &uploaded); err != nil || uploaded.PostID == "" || uploaded.Artifact.Digest == "" || !uploaded.Complete || uploaded.NextOffset != len(artifactBody) {
		t.Fatal("artifact upload lost its file or post", uploaded, err)
	}
	if _, content, err := s.FeedArtifact(ctx, cap.Repository, uploaded.Artifact.ID); err != nil || string(content) != artifactBody {
		t.Fatal("uploaded artifact bytes differ", err)
	}
	if count, err := s.MCPSubmissionCount(ctx, attempt.ID); err != nil || count != 0 {
		t.Fatal("worker actions acquired a duplicate submission journal", count, err)
	}
	// Finish through the admitted MCP transport, then lose the final native
	// refinement reply. The shared workflow must recover the same operation
	// after recomposition, without another run or another issue/reply.
	head, err := s.IntentRevision(ctx, factory.PostDecisionID("intent", id, revision), 1)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []factory.SemanticCandidate{{Key: "one", Rationale: "Preserve the requested login and notification preferences.", Tasks: []factory.IntentTask{{Key: factory.PostDecisionID("task", id, revision), Objective: "Login and notification preferences", Outcomes: []string{"Valid credentials open the app and notification preferences are retained"}, IdeaIDs: head.EffectiveOutcomes}}}}
	finish, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": factory.NewID(), "payload": map[string]any{"candidates": candidates, "coverage": "The recorded request is covered.", "evidence_refs": []string{"recorded request and captured issues"}}}})
	if catalogRefused(finish, err) {
		t.Fatal("Issue Maker finish refused", err, catalogText(t, finish))
	}
	// The shared command must retain the terminal fence even when a host calls
	// it directly, without the MCP handler's earlier admission checks.
	beforeTerminalWrite := len(graph.operations)
	if _, err := product.capturedIssueEffects().Tool(ctx, cap, factory.MCPToolCreateIssue, map[string]any{"key": "late", "title": "Late issue", "body": "Must refuse after finish"}, "after-terminal"); !errors.Is(err, store.ErrMCPSubmissionFrozen) || len(graph.operations) != beforeTerminalWrite {
		t.Fatal("shared issue command bypassed terminal proposal", err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "planning done", true
	if err := s.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, ok := control.AccountSettledRun(ctx, s, run, time.Now()); !ok {
		t.Fatal("Issue Maker run did not settle")
	}
	replies, _, err = s.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil {
		t.Fatal(err)
	}
	graph.lostReply = true
	if _, err := factoryservice.DrivePostIntake(ctx, s, wired, id); err != nil {
		t.Fatal(err)
	}
	post, err = s.PostIssueMaker(ctx, id, revision)
	if err != nil || post.State != "unknown" || graph.issues[1].Title != candidates[0].Tasks[0].Objective {
		t.Fatal("lost refinement reply was not retained as unknown", post, err)
	}
	committed, version := len(graph.operations), graph.issues[1].ContentVersion
	ledger, err := s.FountainLedgerHost(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	identities, err := forgejo.NewFountainIdentities(ctx, reopened, ledger.ID)
	if err != nil {
		t.Fatal(err)
	}
	graph.identities = identities
	recomposed := &API{Store: reopened, FountainIdentities: identities, IssueMakerNative: graph, Coordinator: &control.Coordinator{Store: reopened, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}}
	resumed := &factoryservice.PostCollaborators{IssueMaker: recomposed.FactoryApplication().IssueMaker().Drive}
	if count := factoryservice.DrivePendingPosts(ctx, reopened, resumed, 16); count != 1 {
		t.Fatal("unknown refinement was lost from durable continuation", count)
	}
	post, err = reopened.PostIssueMaker(ctx, id, revision)
	if err != nil || post.State != "completed" || len(graph.operations) != committed || graph.issues[1].ContentVersion != version || len(stub.launches) != 1 || len(graph.issues) != 2 {
		t.Fatal("recovery did not settle exactly the original refinement", post, err)
	}
	after, _, err := reopened.FeedReplyIDs(ctx, 7, "post", post.Scope.Post, 0)
	if err != nil || len(after) != len(replies) {
		t.Fatal("refinement recovery duplicated a role-thread reply", after, err)
	}
	if count := factoryservice.DrivePendingPosts(ctx, reopened, resumed, 16); count != 0 {
		t.Fatal("completed Issue Maker stayed in the work queue", count)
	}
}

// Exercise the public preparation read with real PostgreSQL decisions and the
// production native HTTP client. The explicit native peer enforces its actual
// aggregate contract; selected files must be read without dropping bytes.
func TestPreparationDecisionPreviewReadsBoundedNativeContextHTTP(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if err := s.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	seedPlanFundingWithPreparations(t, ctx, s, false)
	const projectID = "p765432109876543210987654"
	content := strings.Repeat("x", project.MaxFactoryContextFileBytes)
	sum := sha256.Sum256([]byte(content))
	digest := hex.EncodeToString(sum[:])
	requirement := project.RequirementDecision{ID: "d333333333333333333333333", Predecessor: "d111111111111111111111111", Project: projectID, Approver: 1, SourceCommit: strings.Repeat("c", 40), SetupDigest: strings.Repeat("d", 64), InputsDigest: strings.Repeat("e", 64), SourceInputs: map[string]string{"one.mjs": digest, "two.mjs": digest}}
	if err := s.AdmitRequirementDecision(ctx, requirement); err != nil {
		t.Fatal(err)
	}
	approval := project.ApprovalDecision{ID: "d444444444444444444444444", Predecessor: "d222222222222222222222222", Project: projectID, Requirement: requirement.ID, Approver: 1, EffectsDigest: strings.Repeat("d", 64), ReadinessDigest: strings.Repeat("e", 64), Verified: true}
	if err := s.AdmitApprovalDecision(ctx, approval); err != nil {
		t.Fatal(err)
	}
	states := map[string]execution.PreparationState{}
	zero := 0
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := "f" + strings.Repeat(string(rune('1'+i)), 24)
		prep := project.Preparation{ID: id, Project: projectID, Role: role, SourceCommit: requirement.SourceCommit, SetupDigest: requirement.SetupDigest, Requirements: requirement.Ref(2), Approval: project.AdminApproval{ID: approval.ID, Revision: 2, Approver: 1, EffectsDigest: approval.EffectsDigest}, Tools: []string{"node"}}
		state := execution.PreparationState{ID: id, Project: projectID, Role: role, SourceCommit: prep.SourceCommit, SetupDigest: prep.SetupDigest, Phase: project.PrepareReady, Ready: true, SetupExit: &zero, CheckExit: &zero, Container: strings.Repeat("a", 64)}
		if _, _, err := s.AdmitPreparation(ctx, execution.PreparationRecord{Preparation: prep, State: state}); err != nil {
			t.Fatal(err)
		}
		states[id] = state
	}
	var reads atomic.Int32
	var fault atomic.Bool
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prepare-inspect":
			var in project.PrepareInspect
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(states[in.ID])
		case "/prepare-context":
			var in project.FactoryPreparationContextRequest
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			reads.Add(1)
			material := project.FactoryPreparationContext{Project: in.Project, ID: in.ID, Role: states[in.ID].Role, SourceCommit: in.SourceCommit, ApprovedBase: in.ApprovedBase, DiffBase: in.DiffBase, Candidate: in.Candidate, Setup: []byte(content), Check: []byte(content), Files: []project.FactoryContextFile{{Path: "AGENTS.md", Content: []byte(content)}}}
			for _, name := range in.Paths {
				material.Files = append(material.Files, project.FactoryContextFile{Path: name, Content: []byte(content)})
			}
			if err := material.ValidateFor(in, states[in.ID].Role); err != nil {
				// The real route refuses an over-bounds selection with
				// the typed HTTP 413 signal before sending bytes; the
				// preview halves only on that signal, never on a
				// fixture-chosen status or error text.
				http.Error(w, "preparation context exceeds bounds", 413)
				return
			}
			if fault.Load() {
				material.Files[len(material.Files)-1].Content[0] = 'y'
			}
			_ = json.NewEncoder(w).Encode(material)
		default:
			http.NotFound(w, r)
		}
	}))
	defer native.Close()
	product.Host = &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}
	read := func() preparationDecisionPreview {
		code, body := call(http.MethodGet, "/api/environments/"+projectID+"/preparation", "")
		var view environmentPreparationView
		if code != 200 || json.Unmarshal(body, &view) != nil {
			t.Fatalf("preparation read %d %s", code, body)
		}
		return view.DecisionPreview
	}
	preview := read()
	// Each 32 KiB file fills the native aggregate bound together with setup,
	// check and AGENTS.md, so the two-file batch is truthfully refused and
	// each role halves to single-path reads: 1 refused batch + 2 singles per
	// role = 6 reads, still byte-exact.
	if preview.Requirements == nil || preview.Approval == nil || len(preview.Requirements.SourceInputs) != 2 || len(preview.Approval.Roles) != 2 || reads.Load() != 6 {
		t.Fatalf("complete bounded preparation preview missing: %+v, reads=%d", preview, reads.Load())
	}
	fault.Store(true)
	if preview := read(); preview.Requirements != nil || preview.Approval != nil || preview.Reason == "" {
		t.Fatalf("changed native source bytes retained decision preview: %+v", preview)
	}
}

// seedBoundedPreviewSelection admits a two-input requirement, its approval
// and both ready native preparations, returning the project ID. content is
// the exact selected bytes for every input path.
func seedBoundedPreviewSelection(t *testing.T, ctx context.Context, s *store.Store, inputs map[string]string) string {
	t.Helper()
	if err := s.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	seedPlanFundingWithPreparations(t, ctx, s, false)
	const projectID = "p765432109876543210987654"
	requirement := project.RequirementDecision{ID: "d333333333333333333333333", Predecessor: "d111111111111111111111111", Project: projectID, Approver: 1, SourceCommit: strings.Repeat("c", 40), SetupDigest: strings.Repeat("d", 64), InputsDigest: strings.Repeat("e", 64), SourceInputs: inputs}
	if err := s.AdmitRequirementDecision(ctx, requirement); err != nil {
		t.Fatal(err)
	}
	approval := project.ApprovalDecision{ID: "d444444444444444444444444", Predecessor: "d222222222222222222222222", Project: projectID, Requirement: requirement.ID, Approver: 1, EffectsDigest: strings.Repeat("d", 64), ReadinessDigest: strings.Repeat("e", 64), Verified: true}
	if err := s.AdmitApprovalDecision(ctx, approval); err != nil {
		t.Fatal(err)
	}
	zero := 0
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := "f" + strings.Repeat(string(rune('1'+i)), 24)
		prep := project.Preparation{ID: id, Project: projectID, Role: role, SourceCommit: requirement.SourceCommit, SetupDigest: requirement.SetupDigest, Requirements: requirement.Ref(2), Approval: project.AdminApproval{ID: approval.ID, Revision: 2, Approver: 1, EffectsDigest: approval.EffectsDigest}, Tools: []string{"node"}}
		state := execution.PreparationState{ID: id, Project: projectID, Role: role, SourceCommit: prep.SourceCommit, SetupDigest: prep.SetupDigest, Phase: project.PrepareReady, Ready: true, SetupExit: &zero, CheckExit: &zero, Container: strings.Repeat("a", 64)}
		if _, _, err := s.AdmitPreparation(ctx, execution.PreparationRecord{Preparation: prep, State: state}); err != nil {
			t.Fatal(err)
		}
	}
	return projectID
}

// TestPreparationDecisionPreviewBatchTransportFailureFailsClosed is the
// FJ-v3-711/712 bad-then-good masking regression: the native peer fails the
// two-path batch (HTTP 500) but would answer both single-path reads
// perfectly. The preview must refuse with the fixed reason after exactly
// one native call — no halve retry may let the passing leaves mask the
// failed batch.
func TestPreparationDecisionPreviewBatchTransportFailureFailsClosed(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	content := []byte("selected input bytes\n")
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	projectID := seedBoundedPreviewSelection(t, ctx, s, map[string]string{"one.mjs": digest, "two.mjs": digest})
	states := map[string]execution.PreparationState{}
	zero := 0
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := "f" + strings.Repeat(string(rune('1'+i)), 24)
		states[id] = execution.PreparationState{ID: id, Project: projectID, Role: role, SourceCommit: strings.Repeat("c", 40), SetupDigest: strings.Repeat("d", 64), Phase: project.PrepareReady, Ready: true, SetupExit: &zero, CheckExit: &zero, Container: strings.Repeat("a", 64)}
	}
	var reads atomic.Int32
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prepare-inspect":
			var in project.PrepareInspect
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(states[in.ID])
		case "/prepare-context":
			var in project.FactoryPreparationContextRequest
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			reads.Add(1)
			if len(in.Paths) > 1 {
				http.Error(w, "batch transport failure", 500)
				return
			}
			material := project.FactoryPreparationContext{Project: in.Project, ID: in.ID, Role: states[in.ID].Role, SourceCommit: in.SourceCommit, ApprovedBase: in.ApprovedBase, DiffBase: in.DiffBase, Candidate: in.Candidate, Setup: []byte("setup\n"), Check: []byte("check\n")}
			for _, name := range in.Paths {
				material.Files = append(material.Files, project.FactoryContextFile{Path: name, Content: content})
			}
			_ = json.NewEncoder(w).Encode(material)
		default:
			http.NotFound(w, r)
		}
	}))
	defer native.Close()
	product.Host = &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}
	code, body := call(http.MethodGet, "/api/environments/"+projectID+"/preparation", "")
	var view environmentPreparationView
	if code != 200 || json.Unmarshal(body, &view) != nil {
		t.Fatalf("preparation read %d %s", code, body)
	}
	if view.DecisionPreview.Requirements != nil || view.DecisionPreview.Approval != nil || view.DecisionPreview.Reason == "" {
		t.Fatalf("transport-failed batch masked by passing singles: %+v", view.DecisionPreview)
	}
	if reads.Load() != 1 {
		t.Fatalf("transport failure must fail closed without subset retry: reads=%d reason=%q", reads.Load(), view.DecisionPreview.Reason)
	}
}

// TestPreparationDecisionPreviewShortParentCancelsFast proves the preview
// honors a parent context shorter than its 30s budget: with a hanging native
// peer and a 3s parent deadline, the preview refuses with the fixed reason
// in seconds (not after the full budget) after exactly one native call —
// no halve retry and no reviewer attempt may follow cancellation.
func TestPreparationDecisionPreviewShortParentCancelsFast(t *testing.T) {
	product, s, _, _ := q05Mux(t, "")
	ctx := context.Background()
	content := []byte("selected input bytes\n")
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	projectID := seedBoundedPreviewSelection(t, ctx, s, map[string]string{"one.mjs": digest, "two.mjs": digest})
	states := map[string]execution.PreparationState{}
	zero := 0
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := "f" + strings.Repeat(string(rune('1'+i)), 24)
		states[id] = execution.PreparationState{ID: id, Project: projectID, Role: role, SourceCommit: strings.Repeat("c", 40), SetupDigest: strings.Repeat("d", 64), Phase: project.PrepareReady, Ready: true, SetupExit: &zero, CheckExit: &zero, Container: strings.Repeat("a", 64)}
	}
	var reads atomic.Int32
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prepare-inspect":
			var in project.PrepareInspect
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(states[in.ID])
		case "/prepare-context":
			var ctxIn project.FactoryPreparationContextRequest
			if json.NewDecoder(r.Body).Decode(&ctxIn) != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			reads.Add(1)
			// Bounded stand-in for a hung native peer: outlives the 3s
			// parent deadline so the parent (not the 30s budget) governs,
			// but always returns so the test server can close. A handler
			// parked on Done alone never notices the client disconnect
			// (no read/write), which would deadlock native.Close.
			select {
			case <-r.Context().Done():
			case <-time.After(6 * time.Second):
				http.Error(w, "parent gone", 500)
			}
			return
		default:
			http.NotFound(w, r)
		}
	}))
	defer native.Close()
	product.Host = &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)
	callCtx := func(parent context.Context, method, path, body string) (int, []byte) {
		t.Helper()
		authority, err := json.Marshal(extensions.Authority{
			ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1",
			Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)},
			Actor:        extensions.Actor{ID: "1", Username: "alice"},
		})
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(parent)
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code, w.Body.Bytes()
	}
	parent, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	code, body := callCtx(parent, http.MethodGet, "/api/environments/"+projectID+"/preparation", "")
	elapsed := time.Since(start)
	var view environmentPreparationView
	if code != 200 || json.Unmarshal(body, &view) != nil {
		t.Fatalf("preparation read %d %s", code, body)
	}
	if view.DecisionPreview.Requirements != nil || view.DecisionPreview.Approval != nil || view.DecisionPreview.Reason == "" {
		t.Fatalf("cancelled preview retained decision: %+v", view.DecisionPreview)
	}
	if elapsed >= 15*time.Second {
		t.Fatalf("shorter parent must govern, not the 30s budget: elapsed=%v", elapsed)
	}
	if reads.Load() != 1 {
		t.Fatalf("cancellation must fail closed without subset retry: reads=%d reason=%q", reads.Load(), view.DecisionPreview.Reason)
	}
}

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
	product, s, dsn, call, errorCode := q05MuxFull(t, "")
	ctx := context.Background()
	rawDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()

	// FJ-v3-580 long-objective leg: the recorded objective itself exceeds
	// 4000 UTF-8 bytes while staying within its 16000-byte authored
	// bound. ASCII keeps bytes == runes so this leg isolates the
	// preserved-objective path from the multibyte edge covered below.
	longObjective := "Record leave requests my manager can approve with a visible decision and explicit review. " +
		strings.Repeat("The request stays visible before acceptance and the reviewer decision remains explicit. ", 60)
	if got := utf8.RuneCountInString(longObjective); got <= 4000 || got > 16000 {
		t.Fatalf("long objective out of its band: %d runes", got)
	}
	if len(longObjective) != utf8.RuneCountInString(longObjective) {
		t.Fatal("long leg must stay ASCII to isolate the rune boundary")
	}
	longDraftBody := strings.Replace(planDraftBody, `"objective":"Plan the recorded draft"`, `"objective":"`+longObjective+`"`, 1)
	if longDraftBody == planDraftBody {
		t.Fatal("long draft body did not carry its objective")
	}
	if code, body := call(http.MethodPost, "/api/factory/intents", longDraftBody); code != http.StatusCreated {
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
	var responded factoryservice.PlanningLaunch
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
	// FJ-v3-580: the launched head carries the actual long objective.
	if head.Objective != longObjective {
		t.Fatal("launched head lost its long objective verbatim")
	}
	if got := utf8.RuneCountInString(head.Objective); got <= 4000 || got > 16000 {
		t.Fatalf("launched objective out of its long band: %d runes", got)
	}
	if launched.Subject.Intent == nil || launched.Subject.Intent.ScopeDigest != head.Digest() {
		t.Fatalf("launched subject not bound to its head: %+v", launched.Subject)
	}
	stagedInput, planningStaged, err := control.ParseStagedPlanningInput(launched.Prompt)
	if err != nil || !planningStaged || !stagedInput.RequireWalkthroughs || len(stagedInput.Inspected) != 1 || stagedInput.Inspected[0].Evidence != planSourceContent {
		t.Fatalf("authenticated planning lost complete native-bound source evidence: %v", err)
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
		if reserved, err := s.BindContextBundleSelection(ctx, staged, cap.ContextSelectionSnapshot); err != nil || !reserved {
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
	stub.preparation = &execution.PreparationState{Container: strings.Repeat("c", 64), ID: preparation.Preparation.ID, Project: preparation.Preparation.Project, Role: preparation.Preparation.Role, SourceCommit: preparation.Preparation.SourceCommit, SetupDigest: preparation.Preparation.SetupDigest, Ready: true, Phase: project.PrepareReady, SetupExit: &zero, CheckExit: &zero}
	catalog, err := product.factoryApprovedVerificationCatalog(ctx, cap)
	if err != nil {
		t.Fatalf("production capability catalog: %v", err)
	}
	if _, ok := catalog.LookupCheck(factory.ApprovedAppCheckID); !ok {
		t.Fatal("approved project.check absent")
	}
	// Exercise the production protected-preparation reader through the shared
	// service, with mutations committed after its runtime I/O has succeeded.
	views := factoryservice.WorkerVerification{Store: s, Catalog: product.Coordinator.ApprovedProjectCheckCatalog}
	checkReadState := func() string {
		var digest string
		if err := rawDB.QueryRowContext(ctx, `SELECT md5(jsonb_build_array((SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_mcp_capabilities c),(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM factory_assignments a),(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM project_preparations p),(SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_feed_posts))::text)`).Scan(&digest); err != nil {
			t.Fatal(err)
		}
		return digest
	}
	beforeCheckReads := checkReadState()
	if _, err := (factoryservice.WorkerVerification{Store: s}).ApprovedCatalog(ctx, cap); err == nil {
		t.Fatal("planning catalog invented a missing protected source")
	}
	for _, fault := range []string{"project", "preparation", "source", "digest", "dispatch", "cancelled", "unknown"} {
		readCtx, stop := context.WithCancel(ctx)
		reader := views
		reader.Catalog = func(ctx context.Context, assignment factory.Assignment) (factory.ApprovedProjectCheckCatalog, error) {
			observed, err := product.Coordinator.ApprovedProjectCheckCatalog(ctx, assignment)
			if err != nil {
				return observed, err
			}
			switch fault {
			case "project":
				observed.Project = "p" + strings.Repeat("f", 24)
			case "preparation":
				observed.Preparation = "f" + strings.Repeat("f", 24)
			case "source":
				observed.SourceCommit = strings.Repeat("f", 40)
			case "digest":
				observed.PreparationDigest = strings.Repeat("f", 64)
			case "dispatch":
				if _, err := rawDB.ExecContext(ctx, `UPDATE factory_dispatch SET open=false WHERE repository=$1`, cap.Repository); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				stop()
			case "unknown":
				return observed, errors.New("protected check read is unknown")
			}
			return observed, nil
		}
		view, err := reader.ApprovedCatalog(readCtx, cap)
		stop()
		if err == nil || view != nil {
			t.Fatalf("planning catalog accepted %s after protected read", fault)
		}
		if _, err := rawDB.ExecContext(ctx, `UPDATE factory_dispatch SET open=true WHERE repository=$1`, cap.Repository); err != nil {
			t.Fatal(err)
		}
	}
	if checkReadState() != beforeCheckReads {
		t.Fatal("planning catalog reads changed work, capabilities or effects")
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
	server := httptest.NewServer(factorymcp.NewHandler(factorymcp.Dependencies{Capabilities: s, Submissions: s, Assignments: s, Runs: s, PriorityJudgments: selection, Catalog: product.factoryApprovedVerificationCatalog}, key))
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "approved-pl-catalog-pg"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + factoryhttpapi.WorkerRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{cap.ID, secret}}}, nil)
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
	// FJ-v3-580: the live walkthroughs preserve the actual long objective:
	// facts[0] stays byte-verbatim while Outcomes[0] carries the review
	// label the post-fix comparator expects for objectives over 4000 bytes.
	walkthroughs := planLongWalkthroughContent(t, head)
	if walkthroughs.Facts[0] != head.Objective || len(walkthroughs.Facts[0]) != len(head.Objective) {
		t.Fatal("live walkthroughs lost their byte-verbatim long facts")
	}
	cloneWalkthroughs := func() factory.ProposalContent {
		raw, err := json.Marshal(walkthroughs)
		if err != nil {
			t.Fatal(err)
		}
		var cloned factory.ProposalContent
		if err := json.Unmarshal(raw, &cloned); err != nil {
			t.Fatal(err)
		}
		return cloned
	}
	finish := func(id string, p any) (*mcp.CallToolResult, error) {
		return session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": id, "payload": map[string]any{"candidates": candidates, "coverage": "Every effective outcome uses the approved Project check.", "evidence_refs": []string{"approved preparation"}, "verification_manifest_proposal": p, "walkthroughs": walkthroughs}}})
	}
	missingID := factory.NewID()
	missing, missingErr := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": missingID, "payload": map[string]any{"candidates": candidates, "coverage": "Every effective outcome is covered.", "evidence_refs": []string{"approved preparation"}}}})
	if !catalogRefused(missing, missingErr) {
		t.Fatal("new product planning accepted completion without authored walkthroughs")
	}
	if _, err := s.MCPWorkStatus(ctx, launched.ID, missingID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("missing walkthrough froze a terminal receipt", err)
	}
	// The FJ-v3-538 finish-path rule: a walkthrough carrier whose nested
	// facts break the byte-equal objective binding, and any carrier sent
	// as a string, refuse before persistence — no terminal receipt freezes.
	finishWalkthroughs := func(id string, w any) (*mcp.CallToolResult, error) {
		return session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": id, "payload": map[string]any{"candidates": candidates, "coverage": "Every effective outcome uses the approved Project check.", "evidence_refs": []string{"approved preparation"}, "verification_manifest_proposal": proposal, "walkthroughs": w}}})
	}
	mismatched := walkthroughs
	mismatched.Facts = append([]string{"A shortened paraphrase of the recorded objective"}, mismatched.Facts[1:]...)
	// FJ-v3-580 extensions at the same real boundary: a truncated long
	// fact (byte-exactness, not a paraphrase), a verbatim Outcomes[0]
	// where the long objective requires its review label, a nested
	// facts-as-string envelope, and stale pre-fix-style multibyte content.
	// None duplicates the existing paraphrase/string/outer-string cases.
	truncated := cloneWalkthroughs()
	truncated.Facts = append([]string{string([]rune(head.Objective)[:4000])}, truncated.Facts[1:]...)
	verbatimOutcome := cloneWalkthroughs()
	for i := range verbatimOutcome.Alternatives {
		verbatimOutcome.Alternatives[i].Outcomes[0] = head.Objective
	}
	factsStringRaw, err := json.Marshal(walkthroughs)
	if err != nil {
		t.Fatal(err)
	}
	var factsString map[string]any
	if err := json.Unmarshal(factsStringRaw, &factsString); err != nil {
		t.Fatal(err)
	}
	factsString["facts"] = "authored walkthrough prose"
	multiSig := " leave request approval decision"
	multiObjective := strings.Repeat("é", 4000-utf8.RuneCountInString(multiSig)) + multiSig
	staleMultibyte := cloneWalkthroughs()
	staleMultibyte.Objective = multiObjective
	staleMultibyte.Facts = append([]string{multiObjective}, staleMultibyte.Facts[1:]...)
	refusalText := func(result *mcp.CallToolResult, err error) string {
		if err != nil {
			return err.Error()
		}
		if result != nil && result.IsError && len(result.Content) == 1 {
			if text, ok := result.Content[0].(*mcp.TextContent); ok {
				return text.Text
			}
		}
		return ""
	}
	for _, tc := range []struct {
		name    string
		wantErr string
		call    func(string) (*mcp.CallToolResult, error)
	}{
		{"mismatched-facts", "", func(id string) (*mcp.CallToolResult, error) { return finishWalkthroughs(id, mismatched) }},
		{"walkthroughs-string", "", func(id string) (*mcp.CallToolResult, error) {
			return finishWalkthroughs(id, "authored walkthrough prose")
		}},
		{"candidates-string", "", func(id string) (*mcp.CallToolResult, error) {
			return session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": id, "payload": map[string]any{"candidates": "keep the draft", "coverage": "Every effective outcome is covered.", "evidence_refs": []string{"approved preparation"}}}})
		}},
		{"truncated-long-fact", "proposal content must lead with the full objective", func(id string) (*mcp.CallToolResult, error) {
			return finishWalkthroughs(id, truncated)
		}},
		{"verbatim-outcome-for-long", "invalid alternative outcome", func(id string) (*mcp.CallToolResult, error) {
			return finishWalkthroughs(id, verbatimOutcome)
		}},
		{"facts-string-envelope", "", func(id string) (*mcp.CallToolResult, error) {
			return finishWalkthroughs(id, factsString)
		}},
		{"stale-multibyte-substitute", "planning walkthrough changes its source facts or revision", func(id string) (*mcp.CallToolResult, error) {
			return finishWalkthroughs(id, staleMultibyte)
		}},
	} {
		id := factory.NewID()
		result, err := tc.call(id)
		if !catalogRefused(result, err) {
			t.Fatalf("refused finish admitted %s", tc.name)
		}
		if tc.wantErr != "" && !strings.Contains(refusalText(result, err), tc.wantErr) {
			t.Fatalf("refused %s carried another message: %q", tc.name, refusalText(result, err))
		}
		if _, err := s.MCPWorkStatus(ctx, launched.ID, id); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("refused %s wrote receipt: %v", tc.name, err)
		}
	}
	// The FJ-v3-553 transport rule: an outer payload sent as a string —
	// not only a nested carrier — refuses at the real MCP boundary
	// before persistence, so no terminal receipt freezes.
	outerID := factory.NewID()
	outerResult, outerErr := session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": outerID, "payload": "keep the recorded draft"}})
	if !catalogRefused(outerResult, outerErr) {
		t.Fatal("outer string payload admitted as finish arguments")
	}
	if _, err := s.MCPWorkStatus(ctx, launched.ID, outerID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("refused outer string payload wrote receipt: %v", err)
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
		if _, err := s.MCPWorkStatus(ctx, launched.ID, id); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("refused %s wrote receipt: %v", fault, err)
		}
		stub.preparation.CheckExit = &zero
	}
	// The FJ-v3-553 long-content rule, extended by FJ-v3-580: the nested
	// objective itself is long (>4000 bytes, byte-verbatim in facts[0])
	// plus worker-scale padded unknowns (~10KB, under the 64 KiB cap),
	// succeeding through live transport, not only through the direct
	// settler. Padded unknowns stay within their own <=4000-character
	// authored bounds, and the optional questions carrier rides the same
	// live finish the guidance preserves.
	longWalkthroughs := cloneWalkthroughs()
	longWalkthroughs.Unknowns = append(append([]string(nil), longWalkthroughs.Unknowns...),
		strings.Repeat("The request stays visible before acceptance. ", 80),
		strings.Repeat("Reviewer decisions remain explicit and revisable. ", 70),
		strings.Repeat("No new integrations ship with the first version. ", 70),
	)
	if got := utf8.RuneCountInString(longWalkthroughs.Objective); got <= 4000 || got > 16000 {
		t.Fatalf("long walkthrough objective out of its band: %d runes", got)
	}
	if longWalkthroughs.Facts[0] != head.Objective {
		t.Fatal("long walkthroughs lost their byte-verbatim facts")
	}
	questions := []string{"Is legacy synchronization in scope for the first version?", "Who approves the recorded leave policy?"}
	finishLong := func(id string, p any) (*mcp.CallToolResult, error) {
		return session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishProductPlan), Arguments: map[string]any{"submission_id": id, "payload": map[string]any{"candidates": candidates, "coverage": "Every effective outcome uses the approved Project check.", "evidence_refs": []string{"approved preparation"}, "verification_manifest_proposal": p, "walkthroughs": longWalkthroughs, "questions": questions}}})
	}
	if raw, err := json.Marshal(map[string]any{"walkthroughs": longWalkthroughs, "questions": questions}); err != nil {
		t.Fatal(err)
	} else if len(raw) <= 10<<10 || len(raw) >= 64<<10 {
		t.Fatalf("long live payload out of its worker-scale band: %d bytes", len(raw))
	}
	id := factory.NewID()
	result, err := finishLong(id, proposal)
	if catalogRefused(result, err) {
		t.Fatalf("approved manifest finish refused: %v %v", catalogText(t, result), err)
	}
	posted, found, err := s.TerminalMCPSubmission(ctx, launched.ID)
	if err != nil || !found {
		t.Fatal("approved finish was not posted", found, err)
	}
	lateApproval := views
	lateApproval.Catalog = func(ctx context.Context, assignment factory.Assignment) (factory.ApprovedProjectCheckCatalog, error) {
		observed, err := product.Coordinator.ApprovedProjectCheckCatalog(ctx, assignment)
		if err != nil {
			return observed, err
		}
		err = s.AdmitApprovalDecision(ctx, project.ApprovalDecision{ID: "d" + strings.Repeat("a", 24), Predecessor: preparation.Preparation.Approval.ID, Project: cap.Project, Requirement: preparation.Preparation.Requirements.ID, Approver: 1, EffectsDigest: strings.Repeat("d", 64), ReadinessDigest: strings.Repeat("d", 64), Verified: true})
		return observed, err
	}
	if view, err := lateApproval.ApprovedCatalog(ctx, cap); err == nil || view != nil {
		t.Fatal("planning catalog survived approval changed during protected read", err)
	}
	if _, err := product.factoryApprovedVerificationCatalog(ctx, cap); err == nil {
		t.Fatal("moved approval retained current catalog")
	}
	result, err = finishLong(id, proposal)
	if !catalogRefused(result, err) {
		t.Fatal("recorded finish bypassed a changed approval")
	}
	changed := proposal
	changed.Checks = append([]factory.ManifestCheckRef(nil), proposal.Checks...)
	changed.Checks[0].CatalogID = "foreign"
	result, err = finishLong(id, changed)
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
	result, err = finishLong(id, proposal)
	if !catalogRefused(result, err) {
		t.Fatal("recorded finish bypassed a changed draft")
	}
	retained, found, err := s.TerminalMCPSubmission(ctx, launched.ID)
	if err != nil || !found || string(retained.Payload) != string(posted.Payload) {
		t.Fatal("refused calls changed the original posted finish", found, err)
	}
	if settled, err := control.SettleMCPTerminalProposal(ctx, s, launched.ID, time.Now()); err != nil || !settled {
		t.Fatalf("approved MCP plan did not settle: %v/%v", settled, err)
	}
	finished, err := s.Assignment(ctx, launched.ID)
	if err != nil || finished.Result == nil || finished.Result.PhaseResult == nil || finished.Result.PhaseResult.Output == nil || finished.Result.PhaseResult.Output.Decomposition == nil || finished.Result.PhaseResult.Output.Decomposition.VerificationManifestProposal == nil || finished.Result.PhaseResult.Output.Decomposition.VerificationManifestProposal.Digest() != proposal.Digest() {
		t.Fatal("actual MCP settlement dropped proposal", err)
	}
	if finished.Result.PhaseResult.Output.Decomposition.Walkthroughs == nil {
		t.Fatal("actual MCP settlement dropped authored walkthroughs")
	}
	// FJ-v3-580: the live settlement preserved the actual long objective.
	settledWalkthroughs := finished.Result.PhaseResult.Output.Decomposition.Walkthroughs
	if settledWalkthroughs.Facts[0] != head.Objective {
		t.Fatal("actual MCP settlement dropped the preserved long objective")
	}
	if got := utf8.RuneCountInString(settledWalkthroughs.Objective); got <= 4000 || got > 16000 {
		t.Fatalf("settled objective out of its long band: %d runes", got)
	}

	// FJ-v3-586 corrected multibyte edge, proven at the comparator
	// level inside this fixture run: objectives with bytes>4000 but
	// runes<=4000 compose with a substitute Outcomes[0] under the
	// canonical byte rule, while facts[0] stays byte-verbatim. That
	// persisted old-valid shape passes acceptance and review with
	// exact identity/digest unchanged; a genuine unrelated mismatch
	// still refuses with the exact outcome_gap message, and a
	// verbatim long Outcomes[0] still refuses at the planning
	// comparator. No server binding check is touched; this only reads
	// the validators.
	edgeSig := " leave request approval decision"
	edgeObjective := strings.Repeat("é", 4000-utf8.RuneCountInString(edgeSig)) + edgeSig
	if got := utf8.RuneCountInString(edgeObjective); got != 4000 {
		t.Fatalf("edge objective must sit exactly on the rune boundary: %d runes", got)
	}
	if len(edgeObjective) <= 4000 {
		t.Fatalf("edge objective must exceed the byte bound: %d bytes", len(edgeObjective))
	}
	edgeHead := head
	edgeHead.Objective = edgeObjective
	edgeDigest := edgeHead.Digest()
	edgeBasis, err := factory.ComposeProposalContent(edgeHead)
	if err != nil {
		t.Fatalf("edge basis refused: %v", err)
	}
	if edgeBasis.Facts[0] != edgeObjective || len(edgeBasis.Facts[0]) != len(edgeObjective) {
		t.Fatal("edge basis did not preserve the byte-verbatim objective facts")
	}
	if edgeBasis.Alternatives[0].Outcomes[0] != "Implement the recorded product objective" {
		t.Fatalf("edge basis did not substitute the byte-rule label: %q", edgeBasis.Alternatives[0].Outcomes[0])
	}
	edgeLong := edgeSig + strings.Repeat("é", 4001)
	describedLong := factory.DescribeAlternatives(edgeLong)
	if !describedLong.Supported || len(describedLong.Alternatives) == 0 || len(describedLong.Alternatives[0].Outcomes) == 0 {
		t.Fatal("long derivation left supported coverage")
	}
	edgeSubstitute := describedLong.Alternatives[0].Outcomes[0]
	if edgeSubstitute != "Implement the recorded product objective" {
		t.Fatalf("substitute derivation did not trigger: %q", edgeSubstitute)
	}
	toPlanning := func(basis factory.ProposalContent, firstOutcome string) factory.ProposalContent {
		converted := basis
		converted.Producer, converted.Supported = factory.PlanningProducer, true
		converted.Alternatives = nil
		for _, key := range []string{"guided", "list"} {
			converted.Alternatives = append(converted.Alternatives, factory.ProductAlternative{
				Key: key, Name: key + " product planning", Audience: "The project owner", Summary: "Plan the recorded draft with visible progress.",
				Steps:      []factory.WalkthroughStep{{Actor: "The project owner", Action: "reviews the saved request", Result: "The planned work is visible"}},
				Visibility: "The owner sees the saved plan", Consequences: "The request stays editable before acceptance", Evidence: "The admitted request and inspected source", Outcomes: []string{firstOutcome}, IdeaIDs: append([]string(nil), factory.ResolvedScope(edgeHead)...),
			})
		}
		return converted
	}
	oldContent := toPlanning(edgeBasis, edgeSubstitute)
	// The persisted old-valid shape passes acceptance ...
	if err := factory.ValidatePlanningWalkthroughs(edgeHead, oldContent); err != nil {
		t.Fatalf("persisted old-valid walkthrough refused acceptance: %v", err)
	}
	// ... and review: the recorded label matches, so no outcome_gap.
	gapProposal := factory.ConsolidatedProposal{Objective: edgeObjective, Outcomes: []string{edgeSubstitute}, Alternative: oldContent.Alternatives[0]}
	if gaps := factory.CheckProposalEffects(oldContent, gapProposal); len(gaps) != 0 {
		t.Fatalf("persisted old-valid walkthrough emitted mismatches: %+v", gaps)
	}
	if edgeHead.Digest() != edgeDigest || oldContent.Objective != edgeHead.Objective || oldContent.Facts[0] != edgeHead.Objective {
		t.Fatal("acceptance or review mutated identity or digest")
	}
	// A genuine unrelated mismatch still refuses with the exact message.
	badProposal := factory.ConsolidatedProposal{Objective: edgeObjective, Outcomes: []string{"An unrelated invented outcome"}, Alternative: oldContent.Alternatives[0]}
	foundGap := false
	for _, gap := range factory.CheckProposalEffects(oldContent, badProposal) {
		if gap.Class == factory.MismatchOutcomeGap && gap.Detail == "The first outcome does not preserve the requested objective." {
			foundGap = true
		}
	}
	if !foundGap {
		t.Fatal("genuine mismatch carried no exact outcome gap")
	}
	// A verbatim long Outcomes[0] still refuses at the planning comparator.
	verbatimOld := toPlanning(edgeBasis, edgeObjective)
	if err := factory.ValidatePlanningWalkthroughs(edgeHead, verbatimOld); err == nil ||
		err.Error() != "planning walkthrough needs distinct alternatives preserving the objective" {
		t.Fatalf("verbatim long Outcomes[0] misreported: %v", err)
	}
}

// TestIntentPlanWaitsWithoutReadyPreparation proves missing preparation
// waits truthfully: with funding but no ready coder preparation, the
// route answers 202 with waits, leaving no run, assignment or allowance
// behind.
func TestIntentPlanWaitsWithoutReadyPreparation(t *testing.T) {
	product, s, dsn, call, _ := q05MuxFull(t, "")
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
	// The direct host uses the shared HTTP path with declared current authority
	// and the actual coordinator/store, without an API or native session object.
	requests := factoryservice.Requests{Store: s, WriteAccess: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
		if actor != 1 || repository != 7 {
			return errors.New("declared planning authority unavailable")
		}
		return ctx.Err()
	}}
	chooser := &exploreFixtureChooser{fail: true}
	transport := factoryhttpapi.Requests{Service: requests, Host: product.FountainIdentities.HostID(), Actor: 1,
		Planning:    factoryservice.RequestPlanning{Requests: requests, Planner: product.Coordinator},
		Exploration: factoryservice.RequestExploration{Requests: requests, Chooser: chooser},
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			if r.Header.Get("Authorization") != "Bearer declared-planning-client" {
				http.Error(w, "declared account admission unavailable", 401)
				return false
			}
			return true
		}}
	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	direct := func(method, path, body string, status int) []byte {
		t.Helper()
		r, err := http.NewRequestWithContext(t.Context(), method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-planning-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != status {
			t.Fatalf("shared planning HTTP %s %s: %d %s %v", method, path, response.StatusCode, data, err)
		}
		return data
	}
	base := "/api/factory/intents/d777777777777777777777777"
	if err := json.Unmarshal(direct(http.MethodPost, base+"/plan", planRequestBody("d888888888888888888888888"), 202), &waited); err != nil || waited.Launched || len(waited.Waits) == 0 {
		t.Fatal("direct planning transport lost actual preparation wait", waited, err)
	}
	direct(http.MethodGet, base+"/plan", "", 200)
	direct(http.MethodGet, base+"/plan?unexpected=1", "", 400)
	direct(http.MethodGet, base+"/generation", "", 200)
	direct(http.MethodPost, base+"/generate", `{"revision":"d888888888888888888888888"}`, 409)
	direct(http.MethodPost, base+"/plan", planRequestBody("d999999999999999999999999"), 412)
	server.Close()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	transport.Service.Store, transport.Planning.Requests.Store, transport.Exploration.Requests.Store = s, s, s
	transport.Planning.Planner = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	server = httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	direct(http.MethodPost, base+"/plan", planRequestBody("d888888888888888888888888"), 202)
	direct(http.MethodGet, base+"/plan", "", 200)
	if len(stub.launches) != 0 || chooser.calls != 0 {
		t.Fatal("HTTP wait/status/recovery launched work or consulted", len(stub.launches), chooser.calls)
	}
	if _, err := s.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: "d777777777777777777777777", OwnerID: 1}, ""); err == nil {
		t.Fatal("HTTP planning wait/recovery funded an allowance")
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
	bad := execution.PreparationRecord{Preparation: project.Preparation{
		ID: "f111111111111111111111111", Project: "p765432109876543210987654", Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest, SourceInputs: map[string]string{planSourcePath: digest}},
		Approval:     project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 1, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	bad.State = execution.PreparationState{Container: strings.Repeat("c", 64), SourceCommit: bad.Preparation.SourceCommit, SetupDigest: bad.Preparation.SetupDigest, ID: "f111111111111111111111111", Project: "p765432109876543210987654", Role: project.RoleCoder, Phase: project.PrepareReady, Ready: true}
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

func planWalkthroughContent(t *testing.T, head factory.IntentRevision) factory.ProposalContent {
	t.Helper()
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatal(err)
	}
	content.Producer, content.Supported = factory.PlanningProducer, true
	content.Alternatives = nil
	for _, key := range []string{"guided", "list"} {
		content.Alternatives = append(content.Alternatives, factory.ProductAlternative{
			Key: key, Name: key + " project planning", Audience: "The project owner", Summary: "Plan the recorded draft with visible progress.",
			Steps:      []factory.WalkthroughStep{{Actor: "The project owner", Action: "reviews the saved request", Result: "The planned work is visible"}},
			Visibility: "The owner sees the saved plan", Consequences: "The request stays editable before acceptance", Evidence: "The admitted request and inspected source", Outcomes: []string{head.Objective}, IdeaIDs: append([]string(nil), factory.ResolvedScope(head)...),
		})
	}
	return content
}

// planLongWalkthroughContent adapts planWalkthroughContent for objectives
// exceeding 4000 UTF-8 bytes: facts stay byte-verbatim while each
// alternative's Outcomes[0] carries the DescribeAlternatives review label
// (the substitute), matching what ValidatePlanningWalkthroughs and
// CheckProposalEffects expect. It refuses honestly when the head is not
// actually long or not supported, so a misbuilt fixture cannot pass.
func planLongWalkthroughContent(t *testing.T, head factory.IntentRevision) factory.ProposalContent {
	t.Helper()
	if got := len(head.Objective); got <= 4000 || got > 16000 {
		t.Fatalf("long walkthrough head out of its band: %d bytes", got)
	}
	described := factory.DescribeAlternatives(head.Objective)
	if !described.Supported || len(described.Alternatives) == 0 || len(described.Alternatives[0].Outcomes) == 0 {
		t.Fatal("long walkthrough head left supported coverage")
	}
	substitute := described.Alternatives[0].Outcomes[0]
	if substitute == head.Objective {
		t.Fatal("long walkthrough head was not substituted")
	}
	content := planWalkthroughContent(t, head)
	for i := range content.Alternatives {
		if len(content.Alternatives[i].Outcomes) == 0 {
			t.Fatal("planning alternative carries no outcomes")
		}
		content.Alternatives[i].Outcomes[0] = substitute
	}
	if content.Facts[0] != head.Objective {
		t.Fatal("long walkthroughs lost their byte-verbatim facts")
	}
	return content
}

// planSettledAnswer encodes one decompose answer for the saved draft head:
// a covering single candidate, or attention-only when attention is set.
func planSettledAnswer(t *testing.T, status, attention string, walkthroughs ...factory.ProposalContent) string {
	t.Helper()
	answer := &factory.SemanticPlannerOutput{
		Intent: "d777777777777777777777777", Revision: "d888888888888888888888888", Operation: factory.IntentPlanningPropose,
	}
	if len(walkthroughs) == 1 {
		answer.Walkthroughs = &walkthroughs[0]
	}
	if attention != "" {
		answer.Attention = attention
	} else {
		answer.Candidates = []factory.SemanticCandidate{{
			Key: "keep", Rationale: "Keep the recorded draft as the single review baseline.",
			Tasks: []factory.IntentTask{{
				Key: "d999999999999999999999999", Objective: "Plan the recorded draft",
				Outcomes: []string{"Draft is planned"}, IdeaIDs: []string{"d333333333333333333333333", "d444444444444444444444444"},
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
	return string(data)
}

// settlePlanRun records the planning worker's MCP finish for one decompose
// answer, then retires and accounts its run: completed answers finish
// through finish_product_plan, anything else through finish_incomplete.
func settlePlanRun(t *testing.T, ctx context.Context, s *store.Store, runID, answer string) {
	t.Helper()
	run, err := s.FactoryRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	launched, err := s.AssignmentByRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	preparation, err := s.Preparation(ctx, launched.Preparation)
	if err != nil {
		t.Fatal(err)
	}
	cap, err := control.BuildMCPCapability(launched, run, preparation)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.StageMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	if err := s.ActivateObservedMCPCapability(ctx, cap); err != nil {
		t.Fatal(err)
	}
	var parsed factory.PhaseResult
	if err := json.Unmarshal([]byte(answer), &parsed); err != nil {
		t.Fatal(err)
	}
	decomposition := parsed.Output.Decomposition
	tool := factory.MCPToolFinishIncomplete
	payload := map[string]any{"outcome": parsed.Status, "cause": parsed.Summary + ": " + decomposition.Attention, "evidence_refs": []string{"planning input"}}
	if parsed.Status == "completed" {
		tool = factory.MCPToolFinishProductPlan
		payload = map[string]any{"candidates": decomposition.Candidates, "coverage": []string{"Every authorized outcome is planned."}, "evidence_refs": []string{"planning input"}}
		if decomposition.Walkthroughs != nil {
			payload["walkthroughs"] = decomposition.Walkthroughs
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	submission := factory.MCPSubmission{AssignmentID: launched.ID, CapabilityID: cap.ID, SubmissionID: factory.NewID(), Tool: tool, Payload: raw}
	if _, err := s.RecordMCPSubmission(ctx, submission); err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "planning done", true
	if err := s.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, ok := control.AccountSettledRun(ctx, s, run, time.Now()); !ok {
		status, _, readErr := s.TerminalMCPProposal(ctx, launched.ID)
		t.Fatal("plan run unsettled", status, readErr, control.CheckMCPTerminalFinish(launched, submission, time.Now()))
	}
}

// TestIntentPlanStatusReadsAttemptLifecycle proves the minimal scoped
// read: none before the first attempt, pending while the launched run
// is outstanding, and the reported completed shape with its candidate
// count once the run settles.
func TestIntentPlanStatusReadsAttemptLifecycle(t *testing.T) {
	product, s, dsn, call, _ := q05MuxFull(t, "")
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
	const receiptURL = "/api/factory/intents/d777777777777777777777777/revisions/d888888888888888888888888/plan"
	readReceipt := func() factoryservice.PlanningReceipt {
		t.Helper()
		code, body := call(http.MethodGet, receiptURL, "")
		if code != 200 {
			t.Fatalf("planning receipt: %d %s", code, body)
		}
		var result factoryservice.PlanningReceipt
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	absent := readReceipt()
	if absent.Intent != "d777777777777777777777777" || absent.Revision != "d888888888888888888888888" || absent.State != "none" || absent.Operation != "" || absent.AssignmentID != "" || absent.RunID != "" {
		t.Fatal("empty planning receipt fabricated an attempt", absent)
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
	pending := readReceipt()
	if pending.Intent != absent.Intent || pending.Revision != absent.Revision || pending.State != "pending" || pending.Operation != factory.IntentPlanningPropose || pending.RunID != launched.RunID || pending.AssignmentID == "" {
		t.Fatal("pending planning receipt lost the recorded attempt", pending)
	}
	if _, body := call(http.MethodGet, receiptURL, ""); strings.Contains(string(body), "launch_confirmed") {
		t.Fatal("recorded attempt claimed native start", string(body))
	}
	t.Run("shared_planning_recomposition_and_current_authority", func(t *testing.T) {
		fresh, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer fresh.Close()
		const intent, revision = "d777777777777777777777777", "d888888888888888888888888"
		attempt, err := fresh.IntentPlanAttempt(ctx, intent, revision)
		if err != nil {
			t.Fatal(err)
		}
		allowance, err := fresh.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: intent, OwnerID: 1}, attempt.ID)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(allowance)
		allowed, checks := true, 0
		denied := errors.New("declared repository authority withdrawn")
		requests := factoryservice.Requests{Store: fresh, WriteAccess: func(ctx context.Context, actor collaboration.ActorID, repository int64) error {
			checks++
			if actor != 1 || repository != 7 || !allowed {
				return denied
			}
			return ctx.Err()
		}}
		planning := factoryservice.RequestPlanning{Requests: requests, Planner: &control.Coordinator{Store: fresh, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}}
		resumed, err := planning.Plan(ctx, 1, intent, revision, factory.IntentPlanningPropose)
		if err != nil || resumed.Launch == nil || resumed.Launch.AssignmentID != attempt.ID || resumed.Launch.RunID != launched.RunID || resumed.Launch.LaunchConfirmed == nil || *resumed.Launch.LaunchConfirmed || len(stub.launches) != 1 || checks != 1 {
			t.Fatal("fresh shared planning duplicated or lost the uncertain launch", resumed, len(stub.launches), checks, err)
		}
		if _, err := planning.Plan(ctx, 1, intent, revision, factory.IntentPlanningOptions); !errors.Is(err, store.ErrCommandConflict) || checks != 2 {
			t.Fatal("recorded attempt was adopted for a different operation", checks, err)
		}
		allowed = false
		if _, err := planning.Plan(ctx, 1, intent, revision, factory.IntentPlanningPropose); !errors.Is(err, factoryservice.ErrRequestWriteAccess) || !errors.Is(err, denied) || checks != 3 {
			t.Fatal("planning replay bypassed current authority", checks, err)
		}
		if _, err := planning.Plan(ctx, 7, intent, revision, factory.IntentPlanningPropose); !errors.Is(err, factoryservice.ErrRequestNotFound) || checks != 3 {
			t.Fatal("foreign request reached planning admission", checks, err)
		}
		cancelled, stop := context.WithCancel(ctx)
		stop()
		if _, err := planning.Plan(cancelled, 1, intent, revision, factory.IntentPlanningPropose); !errors.Is(err, context.Canceled) || checks != 3 {
			t.Fatal("shared service detached caller cancellation", checks, err)
		}
		allowance, err = fresh.IntentAttemptAllowance(ctx, 7, factory.IntentAttemptScope{IntentID: intent, OwnerID: 1}, attempt.ID)
		after, _ := json.Marshal(allowance)
		if err != nil || string(before) != string(after) || len(stub.launches) != 1 {
			t.Fatal("planning recovery or refusal spent another attempt", err)
		}
	})
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
	head, err := s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil {
		t.Fatal(err)
	}
	content := planWalkthroughContent(t, head)
	judge := &exploreFixtureChooser{}
	product.IntentExploreChooser = judge
	settlePlanRun(t, ctx, s, launched.RunID, planSettledAnswer(t, "completed", "", content))
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
	if judge.calls != 0 || !strings.Contains(string(body), `"walkthroughs"`) || strings.Contains(string(body), `"review"`) {
		t.Fatalf("GET must expose authored content without consulting: calls=%d body=%s", judge.calls, body)
	}
	// A command-capable shared instance must also keep reads free of model
	// access. Fail the configured chooser so an accidental consultation is
	// observable even if its error would otherwise leave advice absent.
	shared := factoryservice.RequestExploration{Requests: factoryservice.Requests{Store: s}, Chooser: judge}
	judge.fail = true
	status, err := shared.Status(ctx, 1, head.Intent)
	if err != nil || status.State != "completed" || status.Walkthroughs == nil || status.Review != nil || judge.calls != 0 {
		t.Fatal("shared status consulted or lost recorded planning facts", status.State, judge.calls, err)
	}
	judge.fail = false
	code, body = call(http.MethodPost, statusURL+"/review", `{"revision":"`+head.ID+`"}`)
	if code != http.StatusOK || judge.calls != 2 || !strings.Contains(string(body), `"recommended":"guided"`) {
		t.Fatalf("explicit walkthrough review failed: status=%d calls=%d body=%s", code, judge.calls, body)
	}
	for range 2 {
		code, body = call(http.MethodGet, statusURL, "")
		if code != http.StatusOK || judge.calls != 2 || !strings.Contains(string(body), `"review"`) {
			t.Fatalf("GET must restore review without consulting: status=%d calls=%d body=%s", code, judge.calls, body)
		}
		judge.fail = true
		status, err := shared.Status(ctx, 1, head.Intent)
		if err != nil || status.Review == nil || judge.calls != 2 {
			t.Fatal("shared status could not replay recorded advice without a chooser", judge.calls, err)
		}
		judge.fail = false
	}
	code, body = call(http.MethodPost, statusURL+"/review", `{"revision":"d999999999999999999999999"}`)
	if code != http.StatusPreconditionFailed || judge.calls != 2 {
		t.Fatalf("stale review consulted: status=%d calls=%d body=%s", code, judge.calls, body)
	}
	t.Run("historical_planning_receipt_read", func(t *testing.T) {
		ctx := t.Context()
		historical := readReceipt()
		if historical.State != "completed" || historical.AssignmentID != pending.AssignmentID || historical.RunID != pending.RunID || historical.Operation != factory.IntentPlanningPropose || historical.Walkthroughs == nil || historical.Review != nil {
			t.Fatal("recorded outcome was rejudged or lost its original identity", historical)
		}
		next := head
		next.ID, next.Predecessor, next.State = "d"+strings.Repeat("6", 24), head.ID, "accepted"
		recorded, err := s.IntentPlanAttempt(ctx, head.Intent, head.ID)
		if err != nil || recorded.Result == nil || recorded.Result.PhaseResult == nil || recorded.Result.PhaseResult.Output == nil || recorded.Result.PhaseResult.Output.Decomposition == nil || len(recorded.Result.PhaseResult.Output.Decomposition.Candidates) != 1 {
			t.Fatal("missing recorded graph for acceptance", err)
		}
		next.Tasks = recorded.Result.PhaseResult.Output.Decomposition.Candidates[0].Tasks
		if err := s.SaveIntentRevision(ctx, next); err != nil {
			t.Fatal(err)
		}
		// The original draft and its recorded attempt remain readable after acceptance.
		same := readReceipt()
		a, _ := json.Marshal(historical)
		b, _ := json.Marshal(same)
		if string(a) != string(b) || judge.calls != 2 || len(stub.launches) != 1 {
			t.Fatal("advanced head changed planning history or relaunched")
		}
		// The private proxy uses the native product route and original SDK admission.
		nativeMux := http.NewServeMux()
		product.registerExtensionProductRoutes(nativeMux)
		socket := "@soda-planning-receipt-" + strconv.Itoa(os.Getpid())
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		native := httptest.NewUnstartedServer(nativeMux)
		native.Listener.Close()
		native.Listener = listener
		native.Start()
		t.Cleanup(native.Close)
		proxy, closeProxy := Extension(socket)
		t.Cleanup(closeProxy)
		authority, err := json.Marshal(extensions.Authority{ExtensionID: "soda", InstanceID: "q05-test", SessionGeneration: "generation-1", Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "get"}, Actor: extensions.Actor{ID: "1", Username: "alice"}})
		if err != nil {
			t.Fatal(err)
		}
		proxyPath := "/factory/intents/" + head.Intent + "/revisions/" + head.ID + "/plan"
		for _, probe := range []struct {
			method, path string
			status       int
		}{{http.MethodGet, proxyPath, 200}, {http.MethodPost, proxyPath, 404}, {http.MethodGet, proxyPath + "?", 404}, {http.MethodGet, proxyPath + "?operation=propose_intent", 404}, {http.MethodGet, proxyPath + "/extra", 404}, {http.MethodGet, "/factory/intents/" + head.Intent + "/revisions/bad/plan", 404}} {
			r := httptest.NewRequest(probe.method, probe.path, nil)
			r.Header.Set(extensions.ContextHeader, string(authority))
			r.Header.Set(extensions.AdmissionHeader, "test-admission")
			r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
			w := httptest.NewRecorder()
			proxy.ServeHTTP(w, r)
			if w.Code != probe.status {
				t.Fatal("planning receipt proxy", probe.path, w.Code, w.Body.String())
			}
			if probe.status == 200 && (w.Body.String() != string(a)+"\n" || w.Header().Get("Cache-Control") != "no-store") {
				t.Fatal("proxy changed or cached recorded planning result", w.Body.String())
			}
		}
		native.Close()
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		inspect, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer inspect.Close()
		snapshot := func() string {
			var digest string
			err := inspect.QueryRowContext(ctx, `SELECT md5(jsonb_build_array(
      (SELECT jsonb_agg(to_jsonb(h) ORDER BY id) FROM factory_intent_heads h),
      (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_intent_revisions r),
      (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM factory_assignments a),
      (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),
      (SELECT jsonb_agg(to_jsonb(u) ORDER BY run) FROM factory_usage u),
      (SELECT jsonb_agg(to_jsonb(j) ORDER BY snapshot) FROM factory_priority_judgments j),
      (SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c))::text)`).Scan(&digest)
			if err != nil {
				t.Fatal(err)
			}
			return digest
		}
		before := snapshot()
		checks, revokeAt := 0, 0
		requests := factoryservice.Requests{Store: reopened}
		transport := factoryhttpapi.Requests{Service: requests, Actor: 1, Host: product.FountainIdentities.HostID(),
			Exploration: factoryservice.RequestExploration{Requests: requests, Chooser: judge},
			Current: func(w http.ResponseWriter, r *http.Request) bool {
				checks++
				if r.Header.Get("Authorization") != "Bearer declared-planning-reader" || (revokeAt > 0 && checks == revokeAt) {
					http.Error(w, "declared account withdrawn", 401)
					return false
				}
				return true
			}}
		server := httptest.NewServer(transport.Handler())
		defer func() { server.Close() }()
		read := func(method, path string, status int) []byte {
			t.Helper()
			r, err := http.NewRequestWithContext(ctx, method, server.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Authorization", "Bearer declared-planning-reader")
			response, err := server.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != status {
				t.Fatal("planning receipt HTTP", path, response.StatusCode, string(data), err)
			}
			if status == 200 && response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("planning receipt was cached")
			}
			return data
		}
		if data := read(http.MethodGet, receiptURL, 200); string(data) != string(a)+"\n" {
			t.Fatal("reopen changed recorded planning result", string(data))
		}
		for _, query := range []string{"?", "?owner=1", "?operation=decomposition_options"} {
			read(http.MethodGet, receiptURL+query, 400)
		}
		read(http.MethodPost, receiptURL, 405)
		read(http.MethodGet, "/api/factory/intents/"+head.Intent+"/revisions/"+next.ID+"/plan", 404)
		read(http.MethodGet, "/api/factory/intents/"+head.Intent+"/revisions/d999999999999999999999999/plan", 404)
		read(http.MethodGet, "/api/factory/intents/d999999999999999999999999/revisions/"+head.ID+"/plan", 404)
		checks, revokeAt = 0, 2
		if data := read(http.MethodGet, receiptURL, 401); strings.Contains(string(data), historical.AssignmentID) {
			t.Fatal("retired read disclosed original planning identity")
		}
		revokeAt = 0
		server.Close()
		transport.Actor = 9007199254740993
		server = httptest.NewServer(transport.Handler())
		read(http.MethodGet, receiptURL, 404)
		if snapshot() != before || judge.calls != 2 || len(stub.launches) != 1 {
			t.Fatal("planning receipt reads changed records, judged or launched work")
		}
	})

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
	if viewed.State != "attention" || viewed.Candidates != 0 {
		t.Fatalf("attention status not exact: %+v", viewed)
	}
}

// planSubsetDraftBody narrows the recorded draft to one explicit
// authorized outcome: capture stays in scope while the decided legacy-sync
// idea sits outside the subset and the declined idea stays refused.
const planSubsetDraftBody = `{"id":"d888888888888888888888888","intent":"d777777777777777777777777","objective":"Plan the recorded draft","state":"draft","repository":"7","max_issues":10,"planning_calls":3,` +
	`"effective_outcomes":["d333333333333333333333333"],` +
	`"ideas":[{"id":"d333333333333333333333333","description":"Capture leave requests","consequence":"Requests can be recorded","decision":"accept","source":"requested","provenance":"human","actor":"1"},` +
	`{"id":"d444444444444444444444444","description":"Sync to the legacy tracker","consequence":"Legacy stays current","decision":"edit","correction":"Sync only on explicit manager request","source":"requested","provenance":"human","actor":"1"},` +
	`{"id":"d555555555555555555555555","description":"Delete requests automatically","consequence":"Old requests vanish","decision":"decline","source":"requested","provenance":"human","actor":"1"}]}`

// TestIntentPlanRefusesCandidateOutsideExplicitSubset proves exact-subset
// enforcement through the real plan lifecycle with no native graph: a
// candidate binding the decided idea outside the explicit subset refuses
// with its exact-scope error, a declined reference keeps its own refusal,
// and the exact-subset candidate assesses clean. Settling the wide answer
// leaves attention on the plan status, never a completed candidate.
func TestIntentPlanRefusesCandidateOutsideExplicitSubset(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	if code, body := call(http.MethodPost, "/api/factory/intents", planSubsetDraftBody); code != http.StatusCreated {
		t.Fatalf("save: %d %s", code, body)
	} else if !strings.Contains(string(body), `"effective_outcomes":["d333333333333333333333333"]`) {
		t.Fatalf("saved payload lost its explicit subset verbatim: %s", body)
	}
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusCreated {
		t.Fatalf("plan status=%d body=%s", code, body)
	}
	var responded factoryservice.PlanningLaunch
	if err := json.Unmarshal(body, &responded); err != nil {
		t.Fatal(err)
	}
	head, err := s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil {
		t.Fatal(err)
	}
	launched, err := s.Assignment(ctx, responded.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	staged, planningStaged, err := control.ParseStagedPlanningInput(launched.Prompt)
	if err != nil || !planningStaged {
		t.Fatalf("launched plan carries no staged assessment input: %v", err)
	}
	content := planWalkthroughContent(t, head)
	answerFor := func(tasks ...factory.IntentTask) factory.SemanticPlannerOutput {
		return factory.SemanticPlannerOutput{
			Intent: head.Intent, Revision: head.ID, Operation: factory.IntentPlanningPropose,
			Walkthroughs: &content,
			Candidates:   []factory.SemanticCandidate{{Key: "subset", Rationale: "Exercise the explicit authorized subset.", Tasks: tasks}},
		}
	}
	exact := factory.IntentTask{Key: "d999999999999999999999999", Objective: "Plan the recorded draft", Outcomes: []string{"Draft is planned"}, IdeaIDs: []string{"d333333333333333333333333"}}
	wide := factory.IntentTask{Key: "d999999999999999999999999", Objective: "Plan the recorded draft", Outcomes: []string{"Draft is planned"}, IdeaIDs: []string{"d333333333333333333333333", "d444444444444444444444444"}}
	declined := factory.IntentTask{Key: "d999999999999999999999999", Objective: "Plan the recorded draft", Outcomes: []string{"Draft is planned"}, IdeaIDs: []string{"d555555555555555555555555"}}
	if err := factory.AssessSemanticOutput(staged, answerFor(exact)); err != nil {
		t.Fatalf("exact-subset candidate refused: %v", err)
	}
	if err := factory.AssessSemanticOutput(staged, answerFor(wide)); err == nil || !strings.Contains(err.Error(), "exceeds effective authorized outcomes") {
		t.Fatalf("outside-subset candidate must refuse with its exact-scope error: %v", err)
	}
	if err := factory.AssessSemanticOutput(staged, answerFor(declined)); err == nil || !strings.Contains(err.Error(), "undecided") {
		t.Fatalf("declined reference must keep its own refusal: %v", err)
	}
	settlePlanRun(t, ctx, s, responded.RunID, planSettledAnswer(t, "completed", "", content))
	finished, err := s.Assignment(ctx, responded.AssignmentID)
	if err != nil || finished.Outcome != factory.NeedsHuman || finished.Reason != factory.AssignReasonNoReport || finished.Result == nil || finished.Result.Reported {
		t.Fatalf("wide settlement reported: %+v %v", finished, err)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/d777777777777777777777777/plan", "")
	if code != http.StatusOK {
		t.Fatalf("refused status=%d body=%s", code, body)
	}
	var viewed struct {
		State      string `json:"state"`
		Summary    string `json:"summary"`
		Candidates int    `json:"candidates"`
	}
	if err := json.Unmarshal(body, &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.State != "attention" || viewed.Candidates != 0 || !strings.Contains(viewed.Summary, "exceeds effective authorized outcomes") {
		t.Fatalf("refused status not exact: %+v", viewed)
	}
}

// TestIntentPlanReviewAdmitsPersistedCompatByteWindow persists the
// FJ-v3-586 old-valid multibyte walkthrough through the real HTTP
// plan/review path against an isolated PG fixture, asserting unchanged
// identity/digest end to end. Facts[0] stays byte-verbatim, Outcomes[0]
// stays substituted under the canonical byte rule; an unrelated
// substituted-outcome mismatch still refuses with the exact messages and
// writes nothing durable. No server binding check is touched; this only
// reads the validators and the HTTP review receipts.
func TestIntentPlanReviewAdmitsPersistedCompatByteWindow(t *testing.T) {
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	compatSignal := "Approve leave requests with a visible reviewer decision. "
	compatObjective := compatSignal + strings.Repeat("é", 2000)
	if len(compatObjective) <= 4000 || utf8.RuneCountInString(compatObjective) >= 4000 {
		t.Fatalf("compat objective outside its byte window: %d bytes, %d runes", len(compatObjective), utf8.RuneCountInString(compatObjective))
	}
	compatDraftBody := strings.Replace(planDraftBody, `"objective":"Plan the recorded draft"`, `"objective":"`+compatObjective+`"`, 1)
	if compatDraftBody == planDraftBody {
		t.Fatal("compat draft body did not carry its objective")
	}
	if code, body := call(http.MethodPost, "/api/factory/intents", compatDraftBody); code != http.StatusCreated {
		t.Fatalf("compat draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil {
		t.Fatal(err)
	}
	compatDigest := head.Digest()
	if head.Objective != compatObjective || len(head.Objective) != len(compatObjective) {
		t.Fatal("HTTP save lost the byte-verbatim compat objective")
	}
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusCreated {
		t.Fatalf("compat plan status=%d body=%s", code, body)
	}
	var launched factoryservice.PlanningLaunch
	if err := json.Unmarshal(body, &launched); err != nil {
		t.Fatal(err)
	}
	head, err = s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil {
		t.Fatal(err)
	}
	if head.Digest() != compatDigest || head.Objective != compatObjective {
		t.Fatal("planning launch mutated persisted identity or digest")
	}
	walkthroughs := planLongWalkthroughContent(t, head)
	if walkthroughs.Facts[0] != head.Objective || len(walkthroughs.Facts[0]) != len(head.Objective) {
		t.Fatal("HTTP walkthroughs lost byte-verbatim facts")
	}
	settlePlanRun(t, ctx, s, launched.RunID, planSettledAnswer(t, "completed", "", walkthroughs))
	judge := &exploreFixtureChooser{}
	product.IntentExploreChooser = judge
	code, body = call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan/review", `{"revision":"d888888888888888888888888"}`)
	if code != http.StatusOK || judge.calls != 2 || !strings.Contains(string(body), `"recommended":"guided"`) {
		t.Fatalf("persisted compat HTTP review failed: status=%d calls=%d body=%s", code, judge.calls, body)
	}
	reloaded, err := s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil || reloaded.Digest() != compatDigest || reloaded.Objective != compatObjective {
		t.Fatalf("HTTP review mutated persisted identity or digest: %+v %v", reloaded, err)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/d777777777777777777777777/plan", "")
	if code != http.StatusOK || !strings.Contains(string(body), `"walkthroughs"`) || !strings.Contains(string(body), `"review"`) {
		t.Fatalf("HTTP status lost persisted walkthroughs or review: %d %s", code, body)
	}
	var viewed struct {
		Revision     string                   `json:"revision"`
		State        string                   `json:"state"`
		Walkthroughs *factory.ProposalContent `json:"walkthroughs"`
	}
	if err := json.Unmarshal(body, &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.Revision != reloaded.ID || viewed.State != "completed" || viewed.Walkthroughs == nil ||
		viewed.Walkthroughs.Facts[0] != compatObjective || viewed.Walkthroughs.Alternatives[0].Outcomes[0] != "Implement the recorded product objective" ||
		viewed.Walkthroughs.Intent != reloaded.Intent || viewed.Walkthroughs.Revision != reloaded.ID || viewed.Walkthroughs.Objective != compatObjective {
		t.Fatalf("HTTP persisted walkthrough changed identity or digest: %+v", viewed.Walkthroughs)
	}
	if err := factory.ValidatePlanningWalkthroughs(reloaded, *viewed.Walkthroughs); err != nil {
		t.Fatalf("HTTP persisted walkthrough refused acceptance: %v", err)
	}
	unrelated := *viewed.Walkthroughs
	unrelated.Alternatives = append([]factory.ProductAlternative(nil), viewed.Walkthroughs.Alternatives...)
	unrelated.Alternatives[0].Outcomes = append([]string{"An unrelated invented outcome"}, unrelated.Alternatives[0].Outcomes[1:]...)
	if err := factory.ValidatePlanningWalkthroughs(reloaded, unrelated); err == nil ||
		err.Error() != "planning walkthrough needs distinct alternatives preserving the objective" {
		t.Fatalf("HTTP unrelated mismatch misreported: %v", err)
	}
	verbatim := *viewed.Walkthroughs
	verbatim.Alternatives = nil
	for _, alternative := range viewed.Walkthroughs.Alternatives {
		alternative.Outcomes = append([]string{compatObjective}, alternative.Outcomes[1:]...)
		verbatim.Alternatives = append(verbatim.Alternatives, alternative)
	}
	if err := factory.ValidatePlanningWalkthroughs(reloaded, verbatim); err == nil ||
		err.Error() != "planning walkthrough needs distinct alternatives preserving the objective" {
		t.Fatalf("HTTP verbatim mismatch misreported: %v", err)
	}
	unrelatedProposal, err := factory.ConsolidateContent(factory.ProposalContent{
		Producer: unrelated.Producer, Supported: unrelated.Supported,
		Intent: unrelated.Intent, Revision: unrelated.Revision, Objective: unrelated.Objective,
		Facts: unrelated.Facts, Excluded: unrelated.Excluded, Exclusions: unrelated.Exclusions,
		Unknowns:     unrelated.Unknowns,
		Alternatives: append([]factory.ProductAlternative{unrelated.Alternatives[0]}, viewed.Walkthroughs.Alternatives[1:]...),
	}, unrelated.Alternatives[0].Key, strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("HTTP unrelated consolidation refused: %v", err)
	}
	mismatches := factory.CheckProposalEffects(*viewed.Walkthroughs, unrelatedProposal)
	if len(mismatches) != 1 || mismatches[0].Class != factory.MismatchOutcomeGap ||
		mismatches[0].Detail != "The first outcome does not preserve the requested objective." {
		t.Fatalf("HTTP unrelated mismatch misreported: %+v", mismatches)
	}
	if viewed.Walkthroughs.Alternatives[0].Outcomes[0] != "Implement the recorded product objective" {
		t.Fatal("HTTP refusal probe mutated the recorded walkthrough")
	}
	if _, err := s.IntentGeneration(ctx, reloaded.ID); err == nil {
		t.Fatal("HTTP review recorded a generation")
	}
}

// TestIntentPlanReviewPersistsMalformedAttention closes the FJ-v3-599 gap
// through the real HTTP plan path: a genuinely malformed compat completion
// (unrelated substituted Outcomes[0] bound to the real saved head) is refused
// before a terminal identity freezes. The worker can report that blocker
// through finish_incomplete, which persists needs-human with the validator
// message rather than a completed decomposition. GET surfaces that attention
// with that exact summary and zero candidates, and no generation is
// written. No server binding check is touched.
func TestIntentPlanReviewPersistsMalformedAttention(t *testing.T) {
	const mismatchDetail = "planning walkthrough needs distinct alternatives preserving the objective"
	product, s, call, _ := q05Mux(t, "")
	ctx := context.Background()
	compatSignal := "Approve leave requests with a visible reviewer decision. "
	compatObjective := compatSignal + strings.Repeat("é", 2000)
	if len(compatObjective) <= 4000 || utf8.RuneCountInString(compatObjective) >= 4000 {
		t.Fatalf("compat objective outside its byte window: %d bytes, %d runes", len(compatObjective), utf8.RuneCountInString(compatObjective))
	}
	compatDraftBody := strings.Replace(planDraftBody, `"objective":"Plan the recorded draft"`, `"objective":"`+compatObjective+`"`, 1)
	if compatDraftBody == planDraftBody {
		t.Fatal("compat draft body did not carry its objective")
	}
	if code, body := call(http.MethodPost, "/api/factory/intents", compatDraftBody); code != http.StatusCreated {
		t.Fatalf("compat draft save status=%d body=%s", code, body)
	}
	head, err := s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil {
		t.Fatal(err)
	}
	compatDigest := head.Digest()
	seedPlanFunding(t, ctx, s)
	stub := &planStubHost{}
	product.Coordinator = &control.Coordinator{Store: s, Host: stub, Broker: planStubBroker{}, DispatchReads: planStubReads{}}
	code, body := call(http.MethodPost, "/api/factory/intents/d777777777777777777777777/plan", planRequestBody("d888888888888888888888888"))
	if code != http.StatusCreated {
		t.Fatalf("compat plan status=%d body=%s", code, body)
	}
	var launched factoryservice.PlanningLaunch
	if err := json.Unmarshal(body, &launched); err != nil {
		t.Fatal(err)
	}
	head, err = s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil {
		t.Fatal(err)
	}
	if head.Digest() != compatDigest {
		t.Fatal("planning launch mutated persisted identity or digest")
	}
	malformed := planLongWalkthroughContent(t, head)
	malformed.Alternatives = append([]factory.ProductAlternative(nil), malformed.Alternatives...)
	malformed.Alternatives[0].Outcomes = append([]string{"An unrelated invented outcome"}, malformed.Alternatives[0].Outcomes[1:]...)
	attempt, err := s.Assignment(ctx, launched.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.FactoryRun(ctx, launched.RunID)
	if err != nil {
		t.Fatal(err)
	}
	preparation, err := s.Preparation(ctx, attempt.Preparation)
	if err != nil {
		t.Fatal(err)
	}
	capability, err := control.BuildMCPCapability(attempt, run, preparation)
	if err != nil {
		t.Fatal(err)
	}
	var malformedResult factory.PhaseResult
	if err := json.Unmarshal([]byte(planSettledAnswer(t, "completed", "", malformed)), &malformedResult); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"candidates": malformedResult.Output.Decomposition.Candidates, "walkthroughs": malformed, "coverage": "Every authorized outcome is planned.", "evidence_refs": []string{"planning input"}})
	if err != nil {
		t.Fatal(err)
	}
	submission := factory.MCPSubmission{AssignmentID: attempt.ID, CapabilityID: capability.ID, SubmissionID: factory.NewID(), Tool: factory.MCPToolFinishProductPlan, Payload: payload}
	if err := control.CheckMCPTerminalFinish(attempt, submission, time.Now()); err == nil || !strings.Contains(err.Error(), mismatchDetail) {
		t.Fatal("malformed completed walkthrough was not refused before persistence", err)
	}
	if count, err := s.MCPSubmissionCount(ctx, attempt.ID); err != nil || count != 0 {
		t.Fatal("refused completion froze a terminal submission", count, err)
	}
	settlePlanRun(t, ctx, s, launched.RunID, planSettledAnswer(t, "blocked", mismatchDetail))
	finished, err := s.Assignment(ctx, launched.AssignmentID)
	if err != nil || finished.Outcome != factory.NeedsHuman || finished.Reason != factory.AssignReasonNoReport || finished.Result == nil || finished.Result.Reported {
		t.Fatalf("malformed HTTP settlement reported: %+v %v", finished, err)
	}
	code, body = call(http.MethodGet, "/api/factory/intents/d777777777777777777777777/plan", "")
	if code != http.StatusOK {
		t.Fatalf("malformed status=%d body=%s", code, body)
	}
	var viewed struct {
		State      string `json:"state"`
		Summary    string `json:"summary"`
		Candidates int    `json:"candidates"`
	}
	if err := json.Unmarshal(body, &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.State != "attention" || viewed.Candidates != 0 || !strings.Contains(viewed.Summary, mismatchDetail) {
		t.Fatalf("malformed status not exact: %+v", viewed)
	}
	reloaded, err := s.IntentRevision(ctx, "d777777777777777777777777", 1)
	if err != nil || reloaded.Digest() != compatDigest || reloaded.Objective != compatObjective {
		t.Fatalf("malformed settle mutated persisted identity or digest: %+v %v", reloaded, err)
	}
	if _, err := s.IntentGeneration(ctx, reloaded.ID); err == nil {
		t.Fatal("malformed HTTP settlement recorded a generation")
	}
}

func (f *planStubHost) InspectRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	return host.NewFactoryObserver(f, nil).InspectRun(ctx, address)
}

func (f *planStubHost) RetireRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	return host.NewFactoryObserver(f, nil).RetireRun(ctx, address)
}

func (f *planStubHost) LaunchRun(ctx context.Context, launch execution.LaunchRequest) (execution.RunObservation, error) {
	return host.NewFactoryObserver(f, nil).LaunchRun(ctx, launch)
}

func (f planStubBroker) ReadCustody(ctx context.Context, address execution.RunAddress) (execution.CustodyObservation, error) {
	return host.NewFactoryObserver(nil, f).ReadCustody(ctx, address)
}

func (f planStubBroker) SealCustody(ctx context.Context, address execution.RunAddress) error {
	return host.NewFactoryObserver(nil, f).SealCustody(ctx, address)
}

func (f *planStubHost) ExportCandidate(ctx context.Context, request execution.CandidateRequest) (execution.CandidateBundle, error) {
	return host.NewFactoryObserver(f, nil).ExportCandidate(ctx, request)
}
