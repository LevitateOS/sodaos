package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// planStubHost is the bounded host double for the draft-planning
// consumer proof: it records launches and answers nothing itself, so
// no provider call or native activation happens in the test.
type planStubHost struct {
	launches []project.FactoryLaunch
}

func (h *planStubHost) FactoryLaunch(_ context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	h.launches = append(h.launches, in)
	return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
}

func (h *planStubHost) FactoryInspect(_ context.Context, _ project.FactoryInspect) (project.FactoryState, error) {
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
// fixture shape with the session operator as actor.
func seedPlanFunding(t *testing.T, ctx context.Context, s *store.Store) {
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
	prep := project.StoredPreparation{Preparation: project.Preparation{
		ID: "f111111111111111111111111", Project: proj, Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d111111111111111111111111", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest},
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
	return `{"revision":"` + revision + `","operation":"propose_intent",` +
		`"inspected":[{"path":"internal/factory/intent.go","commit":"` + strings.Repeat("c", 40) + `","evidence":"Draft head carries attributed effective scope"}],` +
		`"source":"` + strings.Repeat("c", 40) + `"}`
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

	// Stale, malformed and non-draft callers refuse before any launch.
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", planRequestBody("d999999999999999999999999")); code != http.StatusPreconditionFailed || errorCode(body) != "stale_revision" {
		t.Fatalf("stale status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", `{"revision":"`+draftID+`","operation":"nope","inspected":[],"source":"`+strings.Repeat("c", 40)+`"}`); code != http.StatusBadRequest || errorCode(body) != "invalid_operation" {
		t.Fatalf("operation status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodPost, "/api/factory/intents/"+intentID+"/plan", `{"revision":"`+draftID+`","operation":"propose_intent","inspected":[],"source":"short"}`); code != http.StatusBadRequest || errorCode(body) != "invalid_source" {
		t.Fatalf("source status=%d body=%s", code, body)
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
	for _, want := range []string{"Sync only on explicit manager request", "Delete requests automatically", head.Digest()} {
		if !strings.Contains(string(launched.Prompt), want) {
			t.Fatalf("staged prompt drops %q", want)
		}
	}
}
