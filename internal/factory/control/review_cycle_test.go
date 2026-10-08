package control

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestCheckPassUnavailableWithoutAssessor(t *testing.T) {
	fx := publishSeed(t)
	fx.coord.Checks = nil
	report := fx.coord.CheckPass(context.Background())
	if !report.Unavailable || len(report.Assessed) != 0 {
		t.Fatalf("unwired checks must report unavailable: %+v", report)
	}
}

func TestCheckPassAssessesPublishedOnce(t *testing.T) {
	fx, p := checkSeed(t, 11)
	observer := happyCheckObserver()
	fx.coord.Checks = observer
	first := fx.coord.CheckPass(context.Background())
	if first.Unavailable || len(first.Errors) != 0 || len(first.Assessed) != 1 {
		t.Fatalf("first check pass: %+v", first)
	}
	if first.Assessed[0].PublicationID != p.ID || first.Assessed[0].Verdict != factory.CheckPass {
		t.Fatalf("check link: %+v", first.Assessed[0])
	}
	policy, err := fx.db.RepositoryPolicy(context.Background(), p.Repository)
	if err != nil {
		t.Fatal(err)
	}
	if len(observer.actors) != 1 || observer.actors[0] != policy.Merge.ActorID {
		t.Fatalf("check actor: %+v", observer.actors)
	}
	second := fx.coord.CheckPass(context.Background())
	if second.Unavailable || len(second.Errors) != 0 || len(second.Assessed) != 0 {
		t.Fatalf("passing head reassessed: %+v", second)
	}
}

func TestCheckPassReassessesFailedHead(t *testing.T) {
	fx, p := checkSeed(t, 12)
	failing := happyCheckObserver()
	failing.observe = func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, _ := happyCheckObserver().observe(target)
		observed.Checks = []factory.ObservedCheck{{Context: "ci", State: "failure"}}
		return observed, nil
	}
	fx.coord.Checks = failing
	if report := fx.coord.CheckPass(context.Background()); len(report.Assessed) != 1 || report.Assessed[0].Verdict != factory.CheckFailed {
		t.Fatalf("failed verdict unrecorded: %+v", report)
	}
	passing := happyCheckObserver()
	fx.coord.Checks = passing
	if report := fx.coord.CheckPass(context.Background()); len(report.Assessed) != 1 || report.Assessed[0].Verdict != factory.CheckPass {
		t.Fatalf("failed head not reassessed: %+v", report)
	}
	stored, err := fx.db.CheckAssessment(context.Background(), p.Repository, p.PRNumber)
	if err != nil || stored.Verdict != factory.CheckPass || stored.HeadOID != p.Candidate {
		t.Fatalf("stored assessment: %+v %v", stored, err)
	}
}

func TestCheckPassOpensMergeOnFreshPass(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	// The merge reviewer must be independent of the PR author; override
	// before the assignment captures its authority.
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	a := fx.finishReported(t, 15)
	fx.wire(happyPublisher())
	fx.pass(t)
	p, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stage != factory.PublicationPublished {
		t.Fatalf("seed publication not published: %+v", p)
	}
	checks := happyCheckObserver()
	observe := checks.observe
	checks.observe = func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, err := observe(target)
		observed.NativeRev = 9
		return observed, err
	}
	fx.coord.Checks = checks
	fx.coord.Merges = happyMerger()
	report := fx.coord.CheckPass(ctx)
	if len(report.Errors) != 0 || len(report.Assessed) != 1 || report.Assessed[0].Verdict != factory.CheckPass {
		t.Fatalf("check pass: %+v", report)
	}
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatalf("fresh pass opened no merge: %v", err)
	}
	if m.Stage != factory.MergeMerged {
		t.Fatalf("fresh pass merge stage: %+v", m)
	}
}

type fakeReviewer struct {
	observed     factory.ReviewObservation
	outcome      factory.OperationOutcome
	adopted      factory.ReviewOutcome
	works        []factory.ReviewWork
	submitted    []factory.ReviewWork
	lookups      []string
	results      map[string]factory.OperationOutcome
	submitErr    error
	beforeSubmit func(factory.ReviewWork)
	cancelled    []string
}

func (f *fakeReviewer) ObserveReview(ctx context.Context, w factory.ReviewWork) (factory.ReviewObservation, error) {
	f.works = append(f.works, w)
	return f.observed, nil
}

func (f *fakeReviewer) SubmitReview(ctx context.Context, w factory.ReviewWork) (factory.OperationOutcome, error) {
	f.submitted = append(f.submitted, w)
	if f.beforeSubmit != nil {
		f.beforeSubmit(w)
	}
	outcome := f.outcome
	outcome.OperationID, outcome.InstallationID, outcome.Kind = w.OperationID, "test-installation", factory.OpReviewSubmit
	outcome.ActorID, outcome.RepositoryID = w.ActorID, w.Repository
	if f.results == nil {
		f.results = make(map[string]factory.OperationOutcome)
	}
	f.results[w.OperationID] = outcome
	return outcome, f.submitErr
}

func (f *fakeReviewer) LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.lookups = append(f.lookups, operationID)
	if outcome, ok := f.results[operationID]; ok {
		return outcome, nil
	}
	return factory.OperationOutcome{NotObserved: true}, nil
}

func (f *fakeReviewer) CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.cancelled = append(f.cancelled, operationID)
	outcome, ok := f.results[operationID]
	if !ok {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	outcome.Effect, outcome.Cancellation = factory.OpEffectNotCommitted, factory.OpCancelCancelled
	f.results[operationID] = outcome
	return outcome, nil
}

func (f *fakeReviewer) AdoptReview(w factory.ReviewWork, outcome factory.OperationOutcome) (factory.ReviewOutcome, error) {
	adopted := f.adopted
	adopted.Operation = outcome
	return adopted, nil
}

func reviewRunSeed(t *testing.T, head string) (*publishFixture, factory.Assignment, factory.Publication, factory.Run) {
	t.Helper()
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	a := fx.finishReported(t, 14)
	fx.wire(happyPublisher())
	fx.pass(t)
	p, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if head == "" {
		head = p.Candidate
	}
	now := time.Now().Truncate(time.Second)
	run := factory.Run{
		ID: factory.NewID(), ProjectID: a.ProjectID, Role: project.RoleReviewer, InputSHA: head,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
	}
	if err := fx.db.RecordFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "review finished", true
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{
		RunID: run.ID, Repository: a.Repository, Issue: a.Issue, Attempt: a.ID,
	}); err != nil {
		t.Fatal(err)
	}
	return fx, a, p, run
}

func TestSubmitReviewForRunUnavailable(t *testing.T) {
	fx, _, _, run := reviewRunSeed(t, "")
	fx.coord.Reviews = nil
	if _, err := fx.coord.SubmitReviewForRun(context.Background(), run.ID, "output"); err == nil {
		t.Fatal("unwired review submission succeeded")
	}
}

func TestSubmitReviewForRunSubmitsExactHead(t *testing.T) {
	fx, a, p, run := reviewRunSeed(t, "")
	exec := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":3}`),
		adopted:  factory.ReviewOutcome{ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: p.PRID, PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID, HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID, CommitID: p.Candidate, Event: "APPROVED"},
	}
	exec.adopted.Operation = exec.outcome
	exec.beforeSubmit = func(work factory.ReviewWork) {
		stored, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
		if err != nil || len(stored.ReviewOperations) != 1 || stored.ReviewOperations[0].Work != work {
			t.Fatalf("review work was not stored before native submission: %+v %v", stored.ReviewOperations, err)
		}
	}
	fx.coord.Reviews = exec
	output := "notes\n```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```"
	adopted, err := fx.coord.SubmitReviewForRun(context.Background(), run.ID, output)
	if err != nil {
		t.Fatal(err)
	}
	if adopted.ReviewID != 3 || adopted.ReviewerID != 6 || adopted.Event != "APPROVED" {
		t.Fatalf("adopted review: %+v", adopted)
	}
	if len(exec.works) != 1 {
		t.Fatalf("review observations: %d", len(exec.works))
	}
	w := exec.works[0]
	if w.HeadOID != p.Candidate || w.BaseOID != p.PRCreate.BaseOID || w.ActorID != 6 || w.Event != "APPROVED" || w.OperationID != "review-"+run.ID {
		t.Fatalf("review work: %+v", w)
	}
	if w.AuthRevision != factory.ReviewAuthRevision(a.ID, run.ID) {
		t.Fatalf("review auth revision: %q", w.AuthRevision)
	}
	stored, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
	if err != nil || len(stored.ReviewOperations) != 1 || !bytes.Equal(stored.ReviewOperations[0].Outcome.Receipt, exec.results[w.OperationID].Receipt) {
		t.Fatalf("native review receipt not retained byte-exactly: %+v %v", stored.ReviewOperations, err)
	}
}

func TestReviewRetryUsesSavedWorkAfterHeadAndPolicyChange(t *testing.T) {
	ctx := context.Background()
	fx, a, p, run := reviewRunSeed(t, "")
	exec := &fakeReviewer{
		observed:  factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:   committedOutcome(`{"review_id":3}`),
		adopted:   factory.ReviewOutcome{ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: p.PRID, PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID, HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID, CommitID: p.Candidate, Event: "APPROVED"},
		submitErr: errors.New("lost review response"),
	}
	fx.coord.Reviews = exec
	output := "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```"
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID, output); err == nil {
		t.Fatal("lost review response was reported as complete")
	}
	stored, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil || len(stored.ReviewOperations) != 1 {
		t.Fatalf("review intent not persisted: %+v %v", stored.ReviewOperations, err)
	}
	saved := stored.ReviewOperations[0].Work
	for _, mutate := range []struct {
		name string
		work func(*factory.ReviewWork)
	}{
		{"native revision", func(work *factory.ReviewWork) { work.NativeRev++ }},
		{"deadline", func(work *factory.ReviewWork) { work.NotAfter++ }},
		{"actor", func(work *factory.ReviewWork) {
			work.ActorID++
			if work.ActorID == work.PRAuthorID {
				work.ActorID++
			}
		}},
		{"body", func(work *factory.ReviewWork) { work.Body += " changed" }},
	} {
		changed := stored
		changed.ReviewOperations = append([]factory.ReviewOperation(nil), stored.ReviewOperations...)
		changed.ReviewOperations[0].Work = saved
		mutate.work(&changed.ReviewOperations[0].Work)
		changed.Revision++
		if err := fx.db.UpdatePublication(ctx, changed); !errors.Is(err, store.ErrPublicationConflict) {
			t.Fatalf("store accepted changed immutable review %s: %v", mutate.name, err)
		}
		unchanged, err := fx.db.PublicationByAssignment(ctx, a.ID)
		if err != nil || unchanged.ReviewOperations[0].Work != saved {
			t.Fatalf("rejected %s mutation changed saved review: %+v %v", mutate.name, unchanged.ReviewOperations, err)
		}
	}
	for name, list := range map[string]func() ([]factory.Publication, error){
		"repository recovery": func() ([]factory.Publication, error) { return fx.db.OutstandingPublications(ctx, a.Repository, 16) },
		"global recovery":     func() ([]factory.Publication, error) { return fx.db.OpenPublications(ctx, 16) },
	} {
		publications, err := list()
		found := false
		for _, publication := range publications {
			found = found || publication.ID == p.ID
		}
		if err != nil || !found {
			t.Fatalf("unresolved review omitted from %s: found=%t err=%v", name, found, err)
		}
	}

	correction := correctionRun(t, fx, a)
	candidate := strings.Repeat("d", 40)
	report := fx.coord.PublishCorrection(ctx, a.ID, correction.ID, correctionOutput(candidate))
	if len(report.Corrected) != 1 || report.Corrected[0].HeadOID != candidate {
		t.Fatalf("correction did not advance published head: %+v", report)
	}
	policy, err := fx.db.RepositoryPolicy(ctx, a.Repository)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review.ActorID = 7
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	exec.observed.NativeRev = 19
	exec.submitErr = nil
	adopted, err := fx.coord.SubmitReviewForRun(ctx, run.ID, "malformed output must not replace saved work")
	if err != nil {
		t.Fatal(err)
	}
	if adopted.ReviewerID != saved.ActorID || adopted.HeadOID != saved.HeadOID || len(exec.lookups) != 1 || len(exec.submitted) != 1 || len(exec.works) != 1 {
		t.Fatalf("retry rebuilt or resubmitted saved review: adopted=%+v lookups=%v submitted=%+v observed=%+v", adopted, exec.lookups, exec.submitted, exec.works)
	}
	expectedObservation := saved
	expectedObservation.NativeRev = 0
	expectedObservation.NotAfter = 0
	if exec.lookups[0] != saved.OperationID || exec.submitted[0] != saved || exec.works[0] != expectedObservation {
		t.Fatalf("retry did not reconcile the original tuple: saved=%+v lookup=%v submit=%+v observe=%+v", saved, exec.lookups, exec.submitted, exec.works)
	}
	for name, list := range map[string]func() ([]factory.Publication, error){
		"repository recovery": func() ([]factory.Publication, error) { return fx.db.OutstandingPublications(ctx, a.Repository, 16) },
		"global recovery":     func() ([]factory.Publication, error) { return fx.db.OpenPublications(ctx, 16) },
	} {
		publications, err := list()
		for _, publication := range publications {
			if publication.ID == p.ID {
				t.Fatalf("completed review remains in %s: %+v", name, publication.ReviewOperations)
			}
		}
		if err != nil {
			t.Fatalf("list %s after review completion: %v", name, err)
		}
	}
}

func TestWithdrawalCancelsRecordedReviewOperation(t *testing.T) {
	ctx := context.Background()
	fx, a, p, run := reviewRunSeed(t, "")
	exec := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  pendingOutcome(),
		adopted:  factory.ReviewOutcome{ReviewerID: 6, PRID: p.PRID, PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID, HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID, Event: "APPROVED"},
	}
	fx.coord.Reviews = exec
	output := "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```"
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID, output); err == nil {
		t.Fatal("pending review incorrectly reported complete")
	}
	if _, err := fx.db.WithdrawDispatch(ctx, a.Repository, "pause", "operator"); err != nil {
		t.Fatal(err)
	}
	withdrawal := fx.coord.cancelRepositoryPublications(ctx, a.Repository)
	found := false
	for _, operationID := range withdrawal.Operations {
		found = found || operationID == "review-"+run.ID
	}
	if !found {
		t.Fatalf("review operation omitted from withdrawal result: %+v", withdrawal)
	}
	if len(exec.cancelled) != 1 || exec.cancelled[0] != "review-"+run.ID {
		t.Fatalf("recorded review was not cancelled exactly once: %v", exec.cancelled)
	}
	stored, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil || len(stored.ReviewOperations) != 1 {
		t.Fatalf("review cancellation not retained: %+v %v", stored.ReviewOperations, err)
	}
	result := stored.ReviewOperations[0].Outcome
	if result.Effect != factory.OpEffectNotCommitted || result.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("review cancellation outcome lost: %+v", result)
	}
}

func TestSubmitReviewForRunRefusesStaleHead(t *testing.T) {
	fx, _, p, run := reviewRunSeed(t, strings.Repeat("9", 40))
	exec := &fakeReviewer{}
	fx.coord.Reviews = exec
	output := "```review-json\n{\"verdict\":\"approve\",\"summary\":\"s\",\"body\":\"b\",\"findings\":[]}\n```"
	if _, err := fx.coord.SubmitReviewForRun(context.Background(), run.ID, output); err == nil {
		t.Fatal("stale review submitted")
	}
	if len(exec.works) != 0 {
		t.Fatal("stale review observed native state")
	}
	_ = p
}

func TestReviewRegistrationRefusesClosedDispatchBeforeSubmit(t *testing.T) {
	ctx := context.Background()
	fx, a, _, run := reviewRunSeed(t, "")
	if err := store.CloseDispatchAfterReviewRegistration(ctx, fx.dsn); err != nil {
		t.Fatal("install isolated post-registration gate trigger")
	}
	exec := &fakeReviewer{observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()}}
	fx.coord.Reviews = exec
	output := "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```"
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID, output); err == nil {
		t.Fatal("review submitted after dispatch closed at the registration boundary")
	}
	if len(exec.submitted) != 0 {
		t.Fatalf("closed dispatch reached native review submit: %+v", exec.submitted)
	}
	stored, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.ReviewOperations) != 1 || stored.ReviewOperations[0].RunID != run.ID || stored.ReviewOperations[0].Outcome.OperationID != "" {
		t.Fatalf("post-registration refusal did not retain its unsent intent: %+v", stored.ReviewOperations)
	}
	open, _, withdrawal, err := fx.db.DispatchState(ctx, a.Repository)
	if err != nil || open || len(withdrawal.ActiveCauses) != 1 || withdrawal.ActiveCauses[0] != "test_review_close" {
		t.Fatalf("trigger did not close dispatch after registration: open=%t withdrawal=%+v err=%v", open, withdrawal, err)
	}
}

func TestStopSubmitsSettledReview(t *testing.T) {
	ctx := context.Background()
	fx, a, p, _ := reviewRunSeed(t, "")
	now := time.Now().Truncate(time.Second)
	run := factory.Run{
		ID: factory.NewID(), ProjectID: a.ProjectID, Role: project.RoleReviewer, InputSHA: p.Candidate,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
	}
	if err := fx.db.RecordFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{
		RunID: run.ID, Repository: a.Repository, Issue: a.Issue, Attempt: a.ID,
	}); err != nil {
		t.Fatal(err)
	}
	output := "notes\n```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"line 3 is wrong\",\"findings\":[\"line 3\"]}\n```"
	host := &stubHost{}
	broker := &stubBroker{}
	host.stop = func(in project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{
			ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted, Output: output,
			Retirement: "confirmed", LeaseID: "lease-" + in.ID, Generation: 3, CredentialReturned: true,
		}, nil
	}
	broker.close = func(string, string) error { return nil }
	broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{
			Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal,
			LeaseID: "lease-" + id, Binding: &identity.Binding{Kind: identity.Factory, ID: id, Generation: 3},
		}, nil
	}
	fx.coord.Host, fx.coord.Broker = host, broker
	exec := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":5}`),
		adopted:  factory.ReviewOutcome{ReviewID: 5, CommentID: 6, ReviewerID: 6, Event: "REQUEST_CHANGES"},
	}
	exec.adopted.Operation = exec.outcome
	fx.coord.Reviews = exec
	receipt, err := fx.coord.Stop(ctx, factory.Command{
		ID: factory.NewID(), Type: factory.CommandStop,
		Target: run.ID, Principal: "native:7", Digest: factory.CommandDigest(factory.CommandStop, run.ID),
	})
	if err != nil || !receipt.Confirmed {
		t.Fatalf("review stop: %+v %v", receipt, err)
	}
	if len(exec.works) != 1 || exec.works[0].Event != "REQUEST_CHANGES" || exec.works[0].HeadOID != p.Candidate {
		t.Fatalf("submitted review: %+v", exec.works)
	}
}

func TestRetryTruthfulReasons(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	grantFullAuthority(t, c)
	readyPreparations(t, c, lifecycleProject)
	settleStubs(host, broker)

	// A succeeded run relaunches nothing.
	succeeded := recordRun(t, c, func(r *factory.Run) { r.Role = project.RoleCoder })
	succeeded.Outcome, succeeded.Reconciled = factory.Succeeded, true
	if err := c.Store.SaveFactoryRun(context.Background(), succeeded); err != nil {
		t.Fatal(err)
	}
	decision, err := c.RetryRun(context.Background(), factory.NewID(), "native:7", succeeded.ID)
	if err != nil || !strings.Contains(decision.Reason, "nothing relaunches") {
		t.Fatalf("succeeded retry: %+v %v", decision, err)
	}

	// A review run resubmits through a new review, never dispatch.
	review := recordRun(t, c, func(r *factory.Run) { r.Role = project.RoleReviewer })
	if _, err := c.Stop(context.Background(), stopCommand(review.ID)); err != nil {
		t.Fatal(err)
	}
	decision, err = c.RetryRun(context.Background(), factory.NewID(), "native:7", review.ID)
	if err != nil || !strings.Contains(decision.Reason, "new review") {
		t.Fatalf("reviewer retry: %+v %v", decision, err)
	}

	// A failed coder run without a dispatchable issue records truthfully.
	failed := recordRun(t, c, func(r *factory.Run) { r.Role = project.RoleCoder })
	if _, err := c.Stop(context.Background(), stopCommand(failed.ID)); err != nil {
		t.Fatal(err)
	}
	decision, err = c.RetryRun(context.Background(), factory.NewID(), "native:7", failed.ID)
	if err != nil || !decision.Queued {
		t.Fatalf("failed retry refused: %+v %v", decision, err)
	}
	if !strings.Contains(decision.Reason, "no dispatchable issue") && !strings.Contains(decision.Reason, "reassessment unavailable") {
		t.Fatalf("failed retry reason: %q", decision.Reason)
	}
}
