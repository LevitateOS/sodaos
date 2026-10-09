package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func dispatchDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func fixturePreparationContext(in project.FactoryPreparationContextRequest, role string) project.FactoryPreparationContext {
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
	}
}

type fakeDispatchHost struct {
	launch          func(project.FactoryLaunch) (project.FactoryState, error)
	inspect         func(project.FactoryInspect) (project.FactoryState, error)
	harness         func(string) (project.FactoryHarnessPin, error)
	families        []string
	launches        []project.FactoryLaunch
	contextRequests []project.FactoryPreparationContextRequest
	contextRoles    []string
}

func (f *fakeDispatchHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	f.launches = append(f.launches, in)
	return f.launch(in)
}

func (f *fakeDispatchHost) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	if f.inspect == nil {
		return project.FactoryState{}, host.ErrRunNotFound
	}
	return f.inspect(in)
}

func (f *fakeDispatchHost) FactoryHarness(ctx context.Context, family string) (project.FactoryHarnessPin, error) {
	f.families = append(f.families, family)
	return f.harness(family)
}

func (f *fakeDispatchHost) ReadPreparationContext(_ context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
	f.contextRequests = append(f.contextRequests, in)
	f.contextRoles = append(f.contextRoles, role)
	return fixturePreparationContext(in, role), nil
}

func (f *fakeDispatchHost) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	return project.FactoryState{}, errors.New("unexpected host stop")
}

func (f *fakeDispatchHost) FactoryTakeover(ctx context.Context, in project.FactoryTakeover) (project.TakeoverResult, error) {
	return project.TakeoverResult{}, errors.New("unexpected host takeover")
}

func (f *fakeDispatchHost) FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	return project.FactoryExportState{}, errors.New("unexpected host export")
}

func (*fakeDispatchHost) PrepareCandidate(context.Context, project.FactoryCandidate) (project.PrepareState, error) {
	return project.PrepareState{}, errors.New("candidate preparation unavailable in dispatch fixture")
}

func (*fakeDispatchHost) InspectPreparation(context.Context, project.PrepareInspect) (project.PrepareState, error) {
	return project.PrepareState{}, errors.New("preparation inspection unavailable in dispatch fixture")
}

func (*fakeDispatchHost) StopPreparation(context.Context, project.PrepareStop) (project.PrepareState, error) {
	return project.PrepareState{}, errors.New("preparation stop unavailable in dispatch fixture")
}

type fakeDispatchBroker struct {
	get func(kind, id string) (identity.Execution, error)
}

func (f *fakeDispatchBroker) GetExecution(ctx context.Context, kind, id string) (identity.Execution, error) {
	if f.get == nil {
		return identity.Execution{}, identity.ErrNotFound
	}
	return f.get(kind, id)
}

func (f *fakeDispatchBroker) CloseExecution(ctx context.Context, kind, id string) error {
	return errors.New("unexpected broker close")
}

type fakeDispatchReads struct {
	inputs map[string]DispatchInputs
	err    error
	calls  int
}

func (f *fakeDispatchReads) ReadDispatchInputs(ctx context.Context, repository, issue string, commentIDs []string, targetRef string) (DispatchInputs, error) {
	f.calls++
	if f.err != nil {
		return DispatchInputs{}, f.err
	}
	in, ok := f.inputs[repository+"/"+issue]
	if !ok {
		return DispatchInputs{}, &AcceptanceRefusal{Reason: RefusalIncompleteEvidence}
	}
	return in, nil
}

type dispatchFixture struct {
	db       *store.Store
	host     *fakeDispatchHost
	broker   *fakeDispatchBroker
	reads    *fakeDispatchReads
	policy   factory.RepositoryPolicy
	proj     string
	repo     int64
	tip      string
	harness  string
	decision map[int64]factory.Acceptance
}

func dispatchTestDB(t *testing.T) (*store.Store, string) {
	t.Helper()
	db, dsn := postgresFixture(t, nil)
	return db, dsn
}

func dispatchSeed(t *testing.T, db *store.Store) dispatchFixture {
	t.Helper()
	ctx := context.Background()
	fx := dispatchFixture{db: db, repo: 7, proj: "p765432109876543210987654", tip: strings.Repeat("c", 40), harness: "1.2.3", decision: map[int64]factory.Acceptance{}}
	if err := db.UpsertUser(ctx, store.User{ID: 7, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProject(ctx, store.Project{ID: fx.proj, Name: "factory", RepositoryID: fx.repo, OwnerID: 7, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	actor := func(kind string) factory.ActorBindingRef {
		return factory.ActorBindingRef{TokenID: 1, ActorID: 7, Kind: kind}
	}
	fx.policy = factory.RepositoryPolicy{
		Repository: fx.repo, GrantedBy: 7, Enabled: true, TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: project.FactoryHarnessCodex, HarnessVers: fx.harness, Model: "test-model"},
			project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: fx.harness, Model: "test-model"},
		},
		Checks: []string{"ci"}, MergeMethod: factory.MergeFastForward,
		Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate),
		Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge),
		AttemptLimits: factory.DefaultAttemptLimits(),
		MaxConcurrent: 2,
	}
	if err := db.SaveRepositoryPolicy(ctx, fx.policy); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 7, MaxConcurrentRuns: 2, MaxQueued: 10}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: fx.repo, GrantedBy: 7, MaxConcurrent: 2, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveSponsorship(ctx, factory.Sponsorship{
		Repository: fx.repo, GrantedBy: 7, Generation: 1, Connection: "conn", GrantID: "grant",
		Roles: []string{project.RoleCoder}, AllowanceMinutes: 120, MaxConcurrent: 2, Active: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveConnectionUsageBudget(ctx, factory.ConnectionUsageBudget{
		Connection: "conn", RollingMinutes: factory.DefaultConnectionUsageBudgetMinutes,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{Repository: fx.repo, Owner: 7, Profile: &project.Profile{
		ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
		Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
	}, Active: true}); err != nil {
		t.Fatal(err)
	}
	reqID, apprID := "d111111111111111111111111", "d222222222222222222222222"
	commit, digest := fx.tip, strings.Repeat("d", 64)
	if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{
		ID: reqID, Project: fx.proj, Approver: 7, SourceCommit: commit, SetupDigest: digest, InputsDigest: digest,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.AdmitApprovalDecision(ctx, project.ApprovalDecision{
		ID: apprID, Project: fx.proj, Requirement: reqID, Approver: 7,
		EffectsDigest: digest, ReadinessDigest: digest, Verified: true,
	}); err != nil {
		t.Fatal(err)
	}
	forrole := func(role, id string) {
		prep := project.StoredPreparation{Preparation: project.Preparation{
			ID: id, Project: fx.proj, Role: role,
			Requirements: project.RequirementAcceptance{ID: reqID, Revision: 1, Approver: 7, SourceCommit: commit, Digest: digest},
			Approval:     project.AdminApproval{ID: apprID, Revision: 1, Approver: 7, EffectsDigest: digest},
			SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
		}}
		prep.State = project.PrepareState{ID: id, Project: fx.proj, Role: role, Phase: project.PrepareReady, Ready: true}
		if _, _, err := db.AdmitPreparation(ctx, prep); err != nil {
			t.Fatal(err)
		}
	}
	forrole(project.RoleCoder, "f111111111111111111111111")
	forrole(project.RoleReviewer, "f222222222222222222222222")
	fx.host = &fakeDispatchHost{
		launch: func(in project.FactoryLaunch) (project.FactoryState, error) {
			return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
		},
		harness: func(family string) (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: family, Version: fx.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	fx.broker = &fakeDispatchBroker{}
	fx.reads = &fakeDispatchReads{inputs: map[string]DispatchInputs{}}
	return fx
}

func (fx *dispatchFixture) accept(t *testing.T, issue int64, id string, bodyOverride ...string) factory.Acceptance {
	t.Helper()
	title, body, answer := fmt.Sprintf("objective %d; inspect README.md", issue), fmt.Sprintf("body %d", issue), fmt.Sprintf("answer %d", issue)
	if len(bodyOverride) > 0 {
		body = bodyOverride[0]
	}
	decision := factory.Acceptance{
		ID: id, Repository: fx.repo, IssueIndex: fmt.Sprintf("%d", issue), Approver: 5, NativeRev: 41,
		TitleDigest: dispatchDigest(title), ContentDigest: dispatchDigest(body), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: dispatchDigest(answer)}},
	}
	if err := fx.db.AdmitAcceptanceDecision(context.Background(), decision); err != nil {
		t.Fatal(err)
	}
	fx.decision[issue] = decision
	fx.reads.inputs[fmt.Sprintf("%d/%d", fx.repo, issue)] = DispatchInputs{
		Issue: DispatchIssue{
			Title: title, Body: body, TitleDigest: decision.TitleDigest,
			ContentDigest: decision.ContentDigest, ContentVer: 2, Visible: true,
		},
		Comments: []DispatchComment{{ID: "11", Content: answer, Digest: dispatchDigest(answer), Visible: true}},
		Tip:      fx.tip, TipRef: "refs/heads/main", Revision: 41,
	}
	return decision
}

func (fx *dispatchFixture) queue(t *testing.T, issue int64, head string) {
	t.Helper()
	fx.queueAt(t, issue, head, time.Now())
}

func (fx *dispatchFixture) queueAt(t *testing.T, issue int64, head string, seen time.Time) {
	t.Helper()
	control := factory.IssueControl{
		Repository: fx.repo, Issue: issue, Acceptance: head,
		NativeRev: fx.reads.inputs[fmt.Sprintf("%d/%d", fx.repo, issue)].Revision,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
	if _, _, err := fx.db.RecordIssueAssessment(context.Background(), control, seen); err != nil {
		t.Fatal(err)
	}
}

func dispatchControlForAssignment(t *testing.T, db *store.Store, a factory.Assignment) factory.IssueControl {
	t.Helper()
	control := factory.IssueControl{
		Repository: a.Repository, Issue: a.Issue, Acceptance: a.Acceptance, NativeRev: a.NativeRev,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
	stored, _, err := db.RecordIssueAssessment(context.Background(), control, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func (fx *dispatchFixture) deps() DispatchDeps {
	coord := &Coordinator{Store: fx.db}
	var reads DispatchReads
	if fx.reads != nil {
		reads = fx.reads
	}
	return DispatchDeps{Store: fx.db, Host: fx.host, Broker: fx.broker, Reads: reads, Authority: coord.EffectiveAuthority}
}

func waitReason(report DispatchReport, issue int64) string {
	for _, wait := range report.Waits {
		if wait.Issue == issue {
			return wait.Reason
		}
	}
	return ""
}

type fakeAcceptanceSource struct {
	evidence map[string]AcceptanceEvidence
	reads    int
}

func (f *fakeAcceptanceSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (AcceptanceEvidence, error) {
	f.reads++
	evidence, ok := f.evidence[repository+"/"+issue]
	if !ok {
		return AcceptanceEvidence{}, &AcceptanceRefusal{Reason: RefusalIncompleteEvidence}
	}
	return evidence, nil
}
