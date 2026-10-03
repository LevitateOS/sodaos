package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
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

type fakeDispatchHost struct {
	launch   func(project.FactoryLaunch) (project.FactoryState, error)
	inspect  func(project.FactoryInspect) (project.FactoryState, error)
	harness  func() (project.FactoryHarnessPin, error)
	launches []project.FactoryLaunch
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

func (f *fakeDispatchHost) FactoryHarness(ctx context.Context) (project.FactoryHarnessPin, error) {
	return f.harness()
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
			project.RoleCoder:    {Harness: fx.harness, Model: "test-model"},
			project.RoleReviewer: {Harness: fx.harness, Model: "test-model"},
		},
		Checks: []string{"ci"}, MergeMethod: factory.MergeFastForward,
		Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate),
		Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge),
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
	if err := db.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{Repository: fx.repo, Owner: 7, Profile: &project.Profile{
		ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
		Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
	}, Active: true}); err != nil {
		t.Fatal(err)
	}
	reqID, apprID := "d111111111111111111111111", "d222222222222222222222222"
	commit, digest := strings.Repeat("e", 40), strings.Repeat("d", 64)
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
		harness: func() (project.FactoryHarnessPin, error) {
			return project.FactoryHarnessPin{Harness: project.FactoryHarnessCodex, Version: fx.harness, SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
		},
	}
	fx.broker = &fakeDispatchBroker{}
	fx.reads = &fakeDispatchReads{inputs: map[string]DispatchInputs{}}
	return fx
}

func (fx *dispatchFixture) accept(t *testing.T, issue int64, id string) factory.Acceptance {
	t.Helper()
	title, body, answer := fmt.Sprintf("objective %d", issue), fmt.Sprintf("body %d", issue), fmt.Sprintf("answer %d", issue)
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
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
	if _, _, err := fx.db.RecordIssueAssessment(context.Background(), control, seen); err != nil {
		t.Fatal(err)
	}
}

func (fx *dispatchFixture) deps() DispatchDeps {
	coord := &Coordinator{Store: fx.db}
	var reads DispatchReads
	if fx.reads != nil {
		reads = fx.reads
	}
	return DispatchDeps{Store: fx.db, Host: fx.host, Broker: fx.broker, Reads: reads, Authority: coord.EffectiveAuthority}
}

func TestDispatchPassLaunchesOldestWithinShortLimit(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
		t.Fatal(err)
	}
	oldHead := fx.accept(t, 3, "d333333333333333333333333")
	youngHead := fx.accept(t, 5, "d555555555555555555555555")
	now := time.Now()
	fx.queueAt(t, 3, oldHead.ID, now.Add(-2*time.Hour))
	fx.queueAt(t, 5, youngHead.ID, now.Add(-time.Hour))

	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 1 || report.Launched[0].Issue != 3 {
		t.Fatalf("launched = %+v", report.Launched)
	}
	if len(report.Waits) != 1 || report.Waits[0].Issue != 5 || report.Waits[0].Reason != WaitCapacity {
		t.Fatalf("waits = %+v", report.Waits)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	launched := report.Launched[0]
	a, err := db.Assignment(ctx, launched.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Stage != factory.AssignmentAssigned || a.Attempts != 1 || a.Run != launched.RunID ||
		a.SourceCommit != fx.tip || a.Acceptance != oldHead.ID || a.Connection != "conn" {
		t.Fatalf("assignment = %+v", a)
	}
	if a.Authority.Sponsorship != 1 || a.Authority.RequirementsID == "" || a.Authority.ApprovalID == "" {
		t.Fatalf("assignment authority = %+v", a.Authority)
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("host launches = %d", len(fx.host.launches))
	}
	sent := fx.host.launches[0]
	if string(sent.Prompt) != string(a.Prompt) || sent.Run.Assignment != a.PromptSHA ||
		sent.Run.SourceCommit != fx.tip || sent.Run.Actor != 7 || sent.Run.Connection != "conn" ||
		sent.HarnessSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("launch = %+v", sent.Run)
	}
	if sent.Run.Preparation != "f111111111111111111111111" {
		t.Fatalf("launch preparation = %+v", sent.Run)
	}
	if run, err := db.FactoryRun(ctx, launched.RunID); err != nil {
		t.Fatal(err)
	} else if span := run.Deadline.Sub(run.Started); span <= 119*time.Minute || span > 120*time.Minute {
		t.Fatalf("run deadline span = %v", span)
	}
	run, err := db.FactoryRun(ctx, launched.RunID)
	if err != nil || run.InputSHA != fx.tip || run.Reconciled {
		t.Fatalf("run = %+v %v", run, err)
	}
	view, err := db.FactoryRunView(ctx, launched.RunID)
	if err != nil || view.Attempt != launched.AssignmentID || view.Issue != 3 {
		t.Fatalf("view = %+v %v", view, err)
	}
	reservation, err := db.Reservation(ctx, launched.AssignmentID)
	if err != nil || reservation.State != factory.ReservationHeld || reservation.PlannedMinutes != 120 {
		t.Fatalf("reservation = %+v %v", reservation, err)
	}

	// Settle the first run through the accounting hook, then the younger
	// issue drains on the next pass.
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	candidate := strings.Repeat("d", 40)
	output := "work\n```result-json\n" +
		`{"status":"completed","summary":"fixed","candidate":"` + candidate + `","review_passed":false,"findings":[]}` +
		"\n```"
	finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
	if !ok || finished.Outcome != factory.Succeeded || finished.Result == nil || !finished.Result.Reported {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	if finished.Result.Candidate != candidate {
		t.Fatalf("candidate = %q", finished.Result.Candidate)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total < 1 {
		t.Fatalf("usage total = %d", total)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 1 || second.Launched[0].Issue != 5 {
		t.Fatalf("second pass launched = %+v waits = %+v", second.Launched, second.Waits)
	}
}

func waitReason(report DispatchReport, issue int64) string {
	for _, wait := range report.Waits {
		if wait.Issue == issue {
			return wait.Reason
		}
	}
	return ""
}

func TestDispatchAccountingNeverRefreshes(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	sp, err := db.Sponsorship(ctx, fx.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sp.AllowanceMinutes = 1
	if err := db.SaveSponsorship(ctx, sp); err != nil {
		t.Fatal(err)
	}
	headA := fx.accept(t, 3, "d333333333333333333333333")
	headB := fx.accept(t, 5, "d555555555555555555555555")
	fx.queue(t, 3, headA.ID)
	fx.queue(t, 5, headB.ID)

	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 {
		t.Fatalf("first pass = %+v %+v", first.Launched, first.Waits)
	}
	run, err := db.FactoryRun(ctx, first.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, ok := AccountSettledRun(ctx, db, run, "", time.Now()); !ok {
		t.Fatal("settled run unaccounted")
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 1 {
		t.Fatalf("usage total = %d", total)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || waitReason(second, 5) != WaitAllowance {
		t.Fatalf("second pass = %+v %+v", second.Launched, second.Waits)
	}

	// An acceptance edit authorizes new work but refreshes no budget: the
	// re-queued issue still waits on the exhausted allowance.
	edited := headB
	edited.ID = "d666666666666666666666666"
	edited.Predecessor = headB.ID
	if err := db.AdmitAcceptanceDecision(ctx, edited); err != nil {
		t.Fatal(err)
	}
	control, err := db.IssueControl(ctx, fx.repo, 5)
	if err != nil {
		t.Fatal(err)
	}
	control.Acceptance = edited.ID
	control.Fingerprint = strings.Repeat("3", 64)
	if _, changed, err := db.RecordIssueAssessment(ctx, control, time.Now()); err != nil || !changed {
		t.Fatalf("re-queue = %v %v", changed, err)
	}
	third := DispatchPass(ctx, fx.deps())
	if len(third.Launched) != 0 || waitReason(third, 5) != WaitAllowance {
		t.Fatalf("third pass = %+v %+v", third.Launched, third.Waits)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 1 {
		t.Fatalf("usage total after edit = %d", total)
	}

	// A withdrawal and authorized resume likewise change no usage.
	coord := &Coordinator{Store: db}
	if _, _, err := coord.withdrawAndStopRuns(ctx, fx.repo, "test", "tester"); err != nil {
		t.Fatal(err)
	}
	_, revision, _, err := db.DispatchState(ctx, fx.repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coord.ReopenDispatch(ctx, factory.NewID(), "tester", fx.repo, revision); err != nil {
		t.Fatal(err)
	}
	fourth := DispatchPass(ctx, fx.deps())
	if len(fourth.Launched) != 0 || waitReason(fourth, 5) != WaitAllowance {
		t.Fatalf("fourth pass = %+v %+v", fourth.Launched, fourth.Waits)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 1 {
		t.Fatalf("usage total after resume = %d", total)
	}
}

func TestDispatchInterruptionReleasesOnlyUnused(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("host down")
	}

	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 0 || len(first.Waits) != 1 || first.Waits[0].Reason != WaitLaunchRefused {
		t.Fatalf("first pass = %+v %+v %+v", first.Launched, first.Waits, first.Errors)
	}
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if list[0].Attempts != 1 || list[0].Stage != factory.AssignmentAssigned {
		t.Fatalf("assignment = %+v", list[0])
	}
	run, err := db.FactoryRun(ctx, list[0].Run)
	if err != nil || !run.Reconciled || run.Outcome != factory.Failed {
		t.Fatalf("run = %+v %v", run, err)
	}
	reservation, err := db.Reservation(ctx, list[0].ID)
	if err != nil || reservation.State != factory.ReservationReleased {
		t.Fatalf("reservation = %+v %v", reservation, err)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 0 {
		t.Fatalf("usage total = %d", total)
	}

	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || waitReason(second, 3) != WaitLaunchRefused {
		t.Fatalf("second pass = %+v %+v", second.Launched, second.Waits)
	}
	third := DispatchPass(ctx, fx.deps())
	if len(third.Launched) != 0 || waitReason(third, 3) != WaitLaunchExhausted {
		t.Fatalf("third pass = %+v %+v %+v", third.Launched, third.Waits, third.Errors)
	}
	done, err := db.Assignment(ctx, list[0].ID)
	if err != nil || done.Stage != factory.AssignmentFinished || done.Outcome != factory.Failed || done.Attempts != 3 {
		t.Fatalf("assignment = %+v %v", done, err)
	}
	if len(done.RunHistory) != 3 {
		t.Fatalf("run history = %v", done.RunHistory)
	}
	// A fourth pass finds the terminal attempt recorded, not new work.
	fourth := DispatchPass(ctx, fx.deps())
	if len(fourth.Waits) != 1 || fourth.Waits[0].Reason != WaitAttemptRecorded {
		t.Fatalf("fourth pass = %+v %+v", fourth.Launched, fourth.Waits)
	}
}

func TestDispatchApprovedReceiptReleasesWhenBrokerUnknown(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("acquire refused")
	}
	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryApproved}, nil
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || waitReason(report, 3) != WaitLaunchRefused {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if reservation, _ := db.Reservation(ctx, list[0].ID); reservation.State != factory.ReservationReleased {
		t.Fatalf("reservation = %+v", reservation)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 0 {
		t.Fatalf("phantom usage = %d", total)
	}
}

func TestDispatchApprovedReceiptFencesWhenBrokerHolds(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("launch outcome unconfirmed")
	}
	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryApproved}, nil
	}
	fx.broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionLive, Digest: "d"}, nil
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Errors) != 1 || report.Errors[0].Reason != DispatchErrFence {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if reservation, _ := db.Reservation(ctx, list[0].ID); reservation.State != factory.ReservationHeld {
		t.Fatalf("reservation = %+v", reservation)
	}
}

func TestDispatchFencedLaunchStaysHeld(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("launch outcome unconfirmed")
	}
	fx.broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionLive, Digest: "d"}, nil
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || len(report.Errors) != 1 || report.Errors[0].Reason != DispatchErrFence {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	run, err := db.FactoryRun(ctx, list[0].Run)
	if err != nil || run.Reconciled {
		t.Fatalf("fenced run settled: %+v %v", run, err)
	}
	reservation, err := db.Reservation(ctx, list[0].ID)
	if err != nil || reservation.State != factory.ReservationHeld {
		t.Fatalf("reservation = %+v %v", reservation, err)
	}
}

func TestDispatchWaitReasons(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T, mutate func(fx *dispatchFixture, db *store.Store)) (dispatchFixture, DispatchReport) {
		t.Helper()
		db, _ := dispatchTestDB(t)
		fx := dispatchSeed(t, db)
		head := fx.accept(t, 3, "d333333333333333333333333")
		fx.queue(t, 3, head.ID)
		mutate(&fx, db)
		return fx, DispatchPass(ctx, fx.deps())
	}
	cases := map[string]struct {
		mutate   func(fx *dispatchFixture, db *store.Store)
		reason   string
		launched int64
		waited   int64
	}{
		"reads unwired": {
			mutate: func(fx *dispatchFixture, db *store.Store) { fx.reads = nil },
			reason: WaitInputsUnavailable,
		},
		"native busy": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.reads.err = &AcceptanceRefusal{Reason: RefusalNativeBusy}
			},
			reason: WaitInputsBusy,
		},
		"stale read": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.reads.err = &AcceptanceRefusal{Reason: RefusalStaleEvidence}
			},
			reason: WaitInputsStale,
		},
		"objective changed": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.ContentDigest = strings.Repeat("9", 64)
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitInputsChanged,
		},
		"source changed": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Comments[0].ContentVer = 7
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitInputsChanged,
		},
		"hidden issue": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.Visible = false
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitInputsHidden,
		},
		"closed issue": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.Closed = true
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitIssueClosed,
		},
		"locked issue": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.Locked = true
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitIssueLocked,
		},
		"missing tip": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Tip = "short"
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitSource,
		},
		"oversized prompt": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				big := strings.Repeat("x", project.MaxFactoryPrompt)
				in.Issue.Body, in.Issue.ContentDigest = big, dispatchDigest(big)
				decision := fx.decision[3]
				decision.ContentDigest = dispatchDigest(big)
				decision.Predecessor, decision.ID = decision.ID, "d777777777777777777777777"
				if err := db.AdmitAcceptanceDecision(ctx, decision); err != nil {
					t.Fatal(err)
				}
				fx.decision[3] = decision
				control, err := db.IssueControl(ctx, fx.repo, 3)
				if err != nil {
					t.Fatal(err)
				}
				control.Acceptance, control.Fingerprint = decision.ID, strings.Repeat("4", 64)
				if _, _, err := db.RecordIssueAssessment(ctx, control, time.Now()); err != nil {
					t.Fatal(err)
				}
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitPrompt,
			waited: 3,
		},
		"harness mismatch": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.host.harness = func() (project.FactoryHarnessPin, error) {
					return project.FactoryHarnessPin{Harness: project.FactoryHarnessCodex, Version: "9.9.9", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
				}
			},
			reason: WaitHarness,
		},
		"harness unpinned": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.host.harness = func() (project.FactoryHarnessPin, error) {
					return project.FactoryHarnessPin{}, nil
				}
			},
			reason: WaitHarness,
		},
		"reviewer-only sponsorship": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				sp, err := db.Sponsorship(ctx, fx.repo, "conn")
				if err != nil {
					t.Fatal(err)
				}
				sp.Roles = []string{project.RoleReviewer}
				if err := db.SaveSponsorship(ctx, sp); err != nil {
					t.Fatal(err)
				}
			},
			reason: WaitSponsorshipRole,
		},
		"stale preparation": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				head, err := db.RequirementHead(ctx, fx.proj)
				if err != nil {
					t.Fatal(err)
				}
				commit, digest := strings.Repeat("e", 40), strings.Repeat("d", 64)
				if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{
					ID: "d999999999999999999999999", Project: fx.proj, Predecessor: head,
					Approver: 7, SourceCommit: commit, SetupDigest: digest, InputsDigest: digest,
				}); err != nil {
					t.Fatal(err)
				}
			},
			reason: WaitPreparation,
			waited: 3,
		},
		"repository full": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				policy, err := db.RepositoryPolicy(ctx, fx.repo)
				if err != nil {
					t.Fatal(err)
				}
				policy.MaxConcurrent = 1
				if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
					t.Fatal(err)
				}
				human := factory.Run{
					ID: factory.NewID(), ProjectID: fx.proj, Role: project.RoleCoder, InputSHA: strings.Repeat("c", 40),
					Started: time.Now(), Deadline: time.Now().Add(time.Hour),
					Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
				}
				if err := db.RecordFactoryRun(ctx, human); err != nil {
					t.Fatal(err)
				}
			},
			reason: WaitRepository,
		},
		"sponsorship full": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				sp, err := db.Sponsorship(ctx, fx.repo, "conn")
				if err != nil {
					t.Fatal(err)
				}
				sp.MaxConcurrent = 1
				if err := db.SaveSponsorship(ctx, sp); err != nil {
					t.Fatal(err)
				}
				head := fx.accept(t, 9, "d999999999999999999999999")
				fx.queue(t, 9, head.ID)
			},
			reason:   WaitSponsorship,
			launched: 3,
			waited:   9,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.waited == 0 {
				tc.waited = 3
			}
			fx, report := setup(t, tc.mutate)
			if tc.launched == 0 && len(report.Launched) != 0 {
				t.Fatalf("launched = %+v", report.Launched)
			}
			if tc.launched != 0 && (len(report.Launched) != 1 || report.Launched[0].Issue != tc.launched) {
				t.Fatalf("launched = %+v", report.Launched)
			}
			if got := waitReason(report, tc.waited); got != tc.reason {
				t.Fatalf("wait = %q, want %q (waits %+v errors %+v)", got, tc.reason, report.Waits, report.Errors)
			}
			if tc.launched == 0 && len(fx.host.launches) != 0 {
				t.Fatal("host launched while waiting")
			}
		})
	}
}

func TestDispatchNeverFallsBackToAnotherConnection(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	sp, err := db.Sponsorship(ctx, fx.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sp.AllowanceMinutes = 1
	if err := db.SaveSponsorship(ctx, sp); err != nil {
		t.Fatal(err)
	}
	rich := factory.Sponsorship{
		Repository: fx.repo, GrantedBy: 7, Generation: 1, Connection: "zzz-rich", GrantID: "grant2",
		Roles: []string{project.RoleCoder}, AllowanceMinutes: 10080, MaxConcurrent: 8, Active: true,
	}
	if err := db.SaveSponsorship(ctx, rich); err != nil {
		t.Fatal(err)
	}
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 {
		t.Fatalf("first pass = %+v %+v", first.Launched, first.Waits)
	}
	a, err := db.Assignment(ctx, first.Launched[0].AssignmentID)
	if err != nil || a.Connection != "conn" {
		t.Fatalf("assignment connection = %+v %v", a, err)
	}
	run, err := db.FactoryRun(ctx, first.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, ok := AccountSettledRun(ctx, db, run, "", time.Now()); !ok {
		t.Fatal("settled run unaccounted")
	}
	head2 := fx.accept(t, 5, "d555555555555555555555555")
	fx.queue(t, 5, head2.ID)
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || waitReason(second, 5) != WaitAllowance {
		t.Fatalf("second pass = %+v %+v", second.Launched, second.Waits)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "zzz-rich"); total != 0 {
		t.Fatalf("fallback connection consumed %d", total)
	}
}

func TestAccountSettledRunResults(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T) (*store.Store, factory.Assignment, factory.Run) {
		t.Helper()
		db, _ := dispatchTestDB(t)
		fx := dispatchSeed(t, db)
		head := fx.accept(t, 3, "d333333333333333333333333")
		fx.queue(t, 3, head.ID)
		report := DispatchPass(ctx, fx.deps())
		if len(report.Launched) != 1 {
			t.Fatalf("launched = %+v", report.Launched)
		}
		a, err := db.Assignment(ctx, report.Launched[0].AssignmentID)
		if err != nil {
			t.Fatal(err)
		}
		run, err := db.FactoryRun(ctx, report.Launched[0].RunID)
		if err != nil {
			t.Fatal(err)
		}
		return db, a, run
	}
	settle := func(t *testing.T, db *store.Store, run factory.Run, outcome factory.Outcome) factory.Run {
		t.Helper()
		run.Outcome, run.Summary, run.Reconciled = outcome, "host says so", true
		if err := db.SaveFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	t.Run("reported candidate", func(t *testing.T) {
		db, a, run := setup(t)
		run = settle(t, db, run, factory.Succeeded)
		candidate := strings.Repeat("d", 40)
		output := "```result-json\n" +
			`{"status":"completed","summary":"fixed it","candidate":"` + candidate + `","review_passed":false,"findings":[]}` + "\n```"
		finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
		if !ok || finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
		if finished.Result == nil || !finished.Result.Reported || finished.Result.Candidate != candidate {
			t.Fatalf("result = %+v", finished.Result)
		}
		if r, _ := db.Reservation(ctx, a.ID); r.State != factory.ReservationConsumed {
			t.Fatalf("reservation = %+v", r)
		}
		if total, _ := db.UsageTotal(ctx, 7, "conn"); total < 1 {
			t.Fatalf("usage = %d", total)
		}
		if again, ok := AccountSettledRun(ctx, db, run, output, time.Now()); ok || again.Stage != "" {
			t.Fatalf("second accounting = %+v %v", again, ok)
		}
	})
	t.Run("completed without report needs human", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Succeeded)
		finished, ok := AccountSettledRun(ctx, db, run, "chatty output, no fence", time.Now())
		if !ok || finished.Outcome != factory.NeedsHuman || finished.Result.Status != "blocked" {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
	})
	t.Run("failed run", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Failed)
		finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
		if !ok || finished.Outcome != factory.Failed || finished.Result.Status != "failed" {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
	})
	t.Run("cancelled run", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Cancelled)
		finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
		if !ok || finished.Outcome != factory.Cancelled || finished.Result.Status != "cancelled" {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
	})
	t.Run("unassigned run ignored", func(t *testing.T) {
		db, _, _ := setup(t)
		human := factory.Run{
			ID: factory.NewID(), ProjectID: "p765432109876543210987654", Role: project.RoleCoder,
			InputSHA: strings.Repeat("c", 40), Started: time.Now(), Deadline: time.Now().Add(time.Hour),
			Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
			Outcome: factory.Succeeded, Summary: "ok", Reconciled: true,
		}
		if _, ok := AccountSettledRun(ctx, db, human, "", time.Now()); ok {
			t.Fatal("unassigned run accounted")
		}
	})
}

func TestRecoveryConsumesSettledRunWithoutHook(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 {
		t.Fatalf("first pass = %+v", first.Launched)
	}
	// The run settles without the accounting hook, as if the process died
	// between the run save and the hook.
	run, err := db.FactoryRun(ctx, first.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Failed, "boom", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Recovered) != 1 || second.Recovered[0] != first.Launched[0].AssignmentID {
		t.Fatalf("recovered = %+v errors = %+v", second.Recovered, second.Errors)
	}
	a, err := db.Assignment(ctx, first.Launched[0].AssignmentID)
	if err != nil || a.Stage != factory.AssignmentFinished || a.Outcome != factory.Failed {
		t.Fatalf("assignment = %+v %v", a, err)
	}
	if r, _ := db.Reservation(ctx, a.ID); r.State != factory.ReservationConsumed {
		t.Fatalf("reservation = %+v", r)
	}
	if total, _ := db.UsageTotal(ctx, 7, "conn"); total < 1 {
		t.Fatalf("usage = %d", total)
	}
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

func TestCompletionTriggersDependantReassessment(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	endpointHead := fx.accept(t, 3, "d333333333333333333333333")
	endpoint := endpointHead
	// The dependant adopts the endpoint as a code prerequisite.
	dependent := factory.Acceptance{
		ID: "d555555555555555555555555", Repository: fx.repo, IssueIndex: "5", Approver: 5, NativeRev: 41,
		TitleDigest: dispatchDigest("dep title"), ContentDigest: dispatchDigest("dep body"), ContentVersion: 1,
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: "100", DependsOn: "200", PrereqAcceptance: endpointHead.ID,
			EndpointRepo: fx.repo, EndpointIssue: 3, Outcome: factory.PrereqCode,
		}},
	}
	if err := db.AdmitAcceptanceDecision(ctx, dependent); err != nil {
		t.Fatal(err)
	}
	source := &fakeAcceptanceSource{evidence: map[string]AcceptanceEvidence{
		"7/5": {
			Issue:        AcceptanceIssueView{Index: "5", TitleDigest: dependent.TitleDigest, ContentDigest: dependent.ContentDigest, ContentVer: 1, Visible: true},
			Dependencies: []AcceptanceEdge{{Occurrence: "100", DependsOn: "200", Visible: true}},
			Revision:     41,
		},
		"7/3": {
			Issue: AcceptanceIssueView{Index: "3", TitleDigest: endpoint.TitleDigest, ContentDigest: endpoint.ContentDigest,
				ContentVer: endpoint.ContentVersion, Visible: true},
			Comments: []AcceptanceComment{{ID: "11", Digest: endpoint.Sources[0].Digest, ContentVer: 0, Visible: true}},
			Revision: 41,
		},
	}}
	coord := &Coordinator{Store: db, AcceptanceReads: source}
	outcome, err := coord.assessCascade(ctx, fx.repo, 5, map[factory.DependenceRef]bool{})
	if err != nil || !outcome.changed || outcome.control.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant assessment = %+v %v", outcome, err)
	}
	revision := outcome.control.Revision
	reads := source.reads

	fx.queue(t, 3, endpointHead.ID)
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 1 {
		t.Fatalf("launched = %+v", report.Launched)
	}
	run, err := db.FactoryRun(ctx, report.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
	if !ok || finished.Outcome != factory.NeedsHuman {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	coord.assessDispatchDependants(ctx, fx.repo, 3)
	if source.reads <= reads {
		t.Fatal("dependant was not reassessed after completion")
	}
	after, err := db.IssueControl(ctx, fx.repo, 5)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != revision || after.Readiness != factory.ReadinessBlocked || after.Reason != factory.BlockerCodePending {
		t.Fatalf("dependant after completion = %+v", after)
	}
}

func TestHostNotFoundMatcherPinsHostSentinel(t *testing.T) {
	if !isHostNotFound(host.ErrRunNotFound) {
		t.Fatal("host sentinel not recognized")
	}
	if !isHostNotFound(fmt.Errorf("inspect: %w", host.ErrRunNotFound)) {
		t.Fatal("wrapped host sentinel not recognized")
	}
	if isHostNotFound(errors.New("boom")) || isHostNotFound(nil) {
		t.Fatal("unrelated error recognized as host miss")
	}
}

func TestDispatchPassWithoutDepsReports(t *testing.T) {
	report := DispatchPass(context.Background(), DispatchDeps{})
	if len(report.Errors) != 1 || len(report.Launched) != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestIntakeTriggersAutomaticDispatch(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	decision := fx.accept(t, 3, "d333333333333333333333333")
	source := &fakeAcceptanceSource{evidence: map[string]AcceptanceEvidence{
		"7/3": {
			Issue: AcceptanceIssueView{
				Index: "3", TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest,
				ContentVer: decision.ContentVersion, Visible: true,
			},
			Comments: []AcceptanceComment{{
				ID: "11", Digest: decision.Sources[0].Digest,
				ContentVer: decision.Sources[0].ContentVersion, Visible: true,
			}},
			Revision: 41,
		},
	}}
	coord := &Coordinator{Store: db, Host: fx.host, Broker: fx.broker, AcceptanceReads: source, DispatchReads: fx.reads}
	control, changed, err := coord.ObserveIssueEvent(ctx, IntakeHint{Delivery: "st08-intake-1", Repository: fx.repo, Issue: 3})
	if err != nil || !changed || control.Readiness != factory.ReadinessQueued {
		t.Fatalf("intake = %+v %v %v", control, changed, err)
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("automatic launches = %d", len(fx.host.launches))
	}
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 || list[0].Stage != factory.AssignmentAssigned {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	// A duplicate delivery replays without dispatching again.
	again, changed, err := coord.ObserveIssueEvent(ctx, IntakeHint{Delivery: "st08-intake-1", Repository: fx.repo, Issue: 3})
	if err != nil || changed || again.Revision != control.Revision {
		t.Fatalf("replay = %+v %v %v", again, changed, err)
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("launches after replay = %d", len(fx.host.launches))
	}
}

func TestDispatchWithdrawnGateRecordsNothing(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	if _, err := db.WithdrawDispatch(ctx, fx.repo, "test", "tester"); err != nil {
		t.Fatal(err)
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || len(report.Waits) != 1 || report.Waits[0].Reason != WaitDispatchClosed {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 0 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if len(fx.host.launches) != 0 {
		t.Fatal("host launched behind a closed gate")
	}
}

// rendezvousReads meets the first two accepted-input reads at a barrier
// and lets later reads through. Both passes check limits on their empty
// snapshot before either records, so the test deterministically exercises
// the concurrent-admission interleaving instead of racing for it.
type rendezvousReads struct {
	mu       sync.Mutex
	inputs   map[string]DispatchInputs
	arrivals int
	gate     chan struct{}
}

func (r *rendezvousReads) ReadDispatchInputs(_ context.Context, repository, issue string, _ []string, _ string) (DispatchInputs, error) {
	r.mu.Lock()
	r.arrivals++
	if r.arrivals == 2 {
		close(r.gate)
	}
	gate, gated := r.gate, r.arrivals <= 2
	r.mu.Unlock()
	if gated {
		<-gate
	}
	in, ok := r.inputs[repository+"/"+issue]
	if !ok {
		return DispatchInputs{}, &AcceptanceRefusal{Reason: RefusalIncompleteEvidence}
	}
	return in, nil
}

type mutexDispatchHost struct {
	mu       sync.Mutex
	launches int
	pin      project.FactoryHarnessPin
}

func (h *mutexDispatchHost) FactoryLaunch(_ context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	h.mu.Lock()
	h.launches++
	h.mu.Unlock()
	return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
}

func (h *mutexDispatchHost) FactoryInspect(context.Context, project.FactoryInspect) (project.FactoryState, error) {
	return project.FactoryState{}, host.ErrRunNotFound
}

func (h *mutexDispatchHost) FactoryHarness(context.Context) (project.FactoryHarnessPin, error) {
	return h.pin, nil
}

func TestConcurrentDispatchPassesPreserveCapacityAndBudget(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
		t.Fatal(err)
	}
	oldHead := fx.accept(t, 3, "d333333333333333333333333")
	youngHead := fx.accept(t, 5, "d555555555555555555555555")
	now := time.Now()
	fx.queueAt(t, 3, oldHead.ID, now.Add(-2*time.Hour))
	fx.queueAt(t, 5, youngHead.ID, now.Add(-time.Hour))
	inputs := make(map[string]DispatchInputs, len(fx.reads.inputs))
	for key, in := range fx.reads.inputs {
		inputs[key] = in
	}
	coord := &Coordinator{Store: db}
	deps := DispatchDeps{
		Store:  db,
		Broker: &fakeDispatchBroker{},
		Reads:  &rendezvousReads{inputs: inputs, gate: make(chan struct{})},
		Host: &mutexDispatchHost{pin: project.FactoryHarnessPin{
			Harness: project.FactoryHarnessCodex, Version: fx.harness,
			SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64),
		}},
		Authority: coord.EffectiveAuthority,
	}
	reports := make([]DispatchReport, 2)
	var wg sync.WaitGroup
	for i := range reports {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reports[i] = DispatchPass(ctx, deps)
		}()
	}
	wg.Wait()
	launched := len(reports[0].Launched) + len(reports[1].Launched)
	if launched != 1 {
		t.Fatalf("concurrent passes launched %d runs against capacity 1: %+v %+v", launched, reports[0], reports[1])
	}
	for i, report := range reports {
		if len(report.Errors) != 0 {
			t.Fatalf("pass %d errors: %+v", i, report.Errors)
		}
	}
	held, err := db.HeldReservations(ctx, store.MaxHeldReservations)
	if err != nil {
		t.Fatal(err)
	}
	planned := 0
	for _, r := range held {
		planned += r.PlannedMinutes
	}
	if len(held) != 1 || planned != 120 {
		t.Fatalf("held reservations = %d (%d planned minutes), want 1 (120)", len(held), planned)
	}
}
