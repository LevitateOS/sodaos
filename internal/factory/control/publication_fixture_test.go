package control

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// publicationTestDB exercises the production schema without fixture DDL.
func publicationTestDB(t *testing.T) (*store.Store, string) {
	t.Helper()
	db, dsn := postgresFixture(t, nil)
	return db, dsn
}

type fakePublishHost struct {
	export    func(project.FactoryExport) (project.FactoryExportState, error)
	exports   []project.FactoryExport
	bundle    []byte
	exportErr error
}

func (f *fakePublishHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	return project.FactoryState{}, errors.New("unexpected host launch")
}

func (f *fakePublishHost) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	return project.FactoryState{}, errors.New("unexpected host stop")
}

func (f *fakePublishHost) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	return project.FactoryState{}, host.ErrRunNotFound
}

func (f *fakePublishHost) FactoryTakeover(ctx context.Context, in project.FactoryTakeover) (project.TakeoverResult, error) {
	return project.TakeoverResult{}, errors.New("unexpected host takeover")
}

func (f *fakePublishHost) FactoryHarness(ctx context.Context) (project.FactoryHarnessPin, error) {
	return project.FactoryHarnessPin{}, errors.New("unexpected host harness")
}

func (f *fakePublishHost) FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	f.exports = append(f.exports, in)
	if f.export != nil {
		return f.export(in)
	}
	if f.exportErr != nil {
		return project.FactoryExportState{}, f.exportErr
	}
	return project.FactoryExportState{
		ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted,
		Container: strings.Repeat("c", 64), Candidate: in.Candidate,
		Bundle: base64.StdEncoding.EncodeToString(f.bundle),
	}, nil
}

type fakePublisher struct {
	ledger   map[string]factory.OperationOutcome
	observe  func(factory.PublicationWork) (factory.PublicationObservation, error)
	submit   func(factory.PublicationWork) (factory.OperationOutcome, error)
	push     func(factory.PublicationWork) (factory.OperationOutcome, error)
	submitPR func(factory.PublicationWork) (factory.OperationOutcome, error)
	lookup   func(string) (factory.OperationOutcome, error)
	cancel   func(string) (factory.OperationOutcome, error)
	adopt    func(factory.PublicationWork, factory.OperationOutcome) (factory.BranchOutcome, error)
	adoptPR  func(factory.PublicationWork, factory.OperationOutcome) (factory.PRCreationOutcome, error)

	observes []factory.PublicationWork
	submits  []factory.PublicationWork
	pushes   []factory.PublicationWork
	creates  []factory.PublicationWork
	lookups  []string
	cancels  []string
}

func (f *fakePublisher) ObservePublication(ctx context.Context, work factory.PublicationWork) (factory.PublicationObservation, error) {
	f.observes = append(f.observes, work)
	return f.observe(work)
}

func (f *fakePublisher) SubmitPublish(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error) {
	f.submits = append(f.submits, work)
	return f.record(work, factory.OpRefPublish, f.submit)
}

func (f *fakePublisher) PushBranch(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error) {
	f.pushes = append(f.pushes, work)
	return f.record(work, factory.OpRefPublish, f.push)
}

func (f *fakePublisher) SubmitPRCreate(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error) {
	f.creates = append(f.creates, work)
	return f.record(work, factory.OpPRCreate, f.submitPR)
}

func (f *fakePublisher) LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.lookups = append(f.lookups, operationID)
	if f.lookup != nil {
		return f.lookup(operationID)
	}
	outcome, ok := f.ledger[operationID]
	if !ok {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	return outcome, nil
}

func (f *fakePublisher) CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.cancels = append(f.cancels, operationID)
	if f.cancel != nil {
		return f.cancel(operationID)
	}
	outcome := f.ledger[operationID]
	if outcome.Effect == factory.OpEffectCommitted {
		outcome.Cancellation = factory.OpCancelTooLate
	} else {
		outcome.Effect, outcome.Cancellation, outcome.Reason = factory.OpEffectNotCommitted, factory.OpCancelCancelled, "cancelled_before_admission"
	}
	outcome.OperationID, outcome.InstallationID = operationID, "test-installation"
	f.ledger[operationID] = outcome
	return outcome, nil
}

func (f *fakePublisher) AdoptBranch(work factory.PublicationWork, outcome factory.OperationOutcome) (factory.BranchOutcome, error) {
	return f.adopt(work, outcome)
}

func (f *fakePublisher) AdoptPRCreation(work factory.PublicationWork, outcome factory.OperationOutcome) (factory.PRCreationOutcome, error) {
	return f.adoptPR(work, outcome)
}

func (f *fakePublisher) record(w factory.PublicationWork, kind string, call func(factory.PublicationWork) (factory.OperationOutcome, error)) (factory.OperationOutcome, error) {
	outcome, err := call(w)
	if err == nil && !outcome.NotObserved {
		outcome = publicationTestOutcome(w, kind, outcome)
		f.ledger[w.OperationID] = outcome
	}
	return outcome, err
}

func publicationTestOutcome(w factory.PublicationWork, kind string, outcome factory.OperationOutcome) factory.OperationOutcome {
	outcome.OperationID, outcome.InstallationID, outcome.Kind = w.OperationID, "test-installation", kind
	outcome.ActorID, outcome.RepositoryID = w.ActorID, w.Repository
	return outcome
}

type publishFixture struct {
	reads *fakeEvidenceSource
	db    *store.Store
	dsn   string
	seed  dispatchFixture
	host  *fakePublishHost
	exec  *fakePublisher
	coord *Coordinator
}

func publishSeed(t *testing.T) *publishFixture {
	t.Helper()
	db, dsn := publicationTestDB(t)
	fx := dispatchSeed(t, db)
	fh := &fakePublishHost{bundle: []byte("bundle-bytes")}
	coord := NewCoordinator(db, fh, &fakeDispatchBroker{})
	reads := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	coord.AcceptanceReads = reads
	return &publishFixture{db: db, dsn: dsn, seed: fx, host: fh, coord: coord, reads: reads}
}

func (fx *publishFixture) wire(exec *fakePublisher) {
	fx.exec = exec
	fx.coord.Publication = exec
}

func (fx *publishFixture) finishReported(t *testing.T, issue int64) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	decision := fx.seed.accept(t, issue, "d"+strings.Repeat("a", 23)+string(rune('0'+issue%10)))
	now := time.Now().Truncate(time.Second)
	prompt := []byte("prompt")
	a := factory.Assignment{
		ID: factory.NewID(), ProjectID: fx.seed.proj, Role: project.RoleCoder,
		Repository: fx.seed.repo, Issue: issue, Revision: 0, NativeRev: 41,
		Acceptance: decision.ID, Preparation: "f111111111111111111111111",
		Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "m",
		Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: dispatchDigest("prompt"),
		Run: factory.NewID(), RunHistory: []string{}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: now.Unix(),
	}
	authority, err := fx.coord.EffectiveAuthority(ctx, a.Repository)
	if err != nil || !authority.Effective {
		t.Fatalf("fixture authority: %+v %v", authority, err)
	}
	a.Authority = authority.Authority
	a.Authority.RequirementsID, err = fx.db.RequirementHead(ctx, a.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	a.Authority.ApprovalID, err = fx.db.ApprovalHead(ctx, a.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	fx.reads.evidence[strconv.FormatInt(a.Repository, 10)+"/"+strconv.FormatInt(issue, 10)] = AcceptanceEvidence{
		Revision: 9,
		Issue:    AcceptanceIssueView{Index: decision.IssueIndex, TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: decision.Sources[0].ID, Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	a.RunHistory = []string{a.Run}
	r := factory.Reservation{AssignmentID: a.ID, Repository: fx.seed.repo, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 30}
	run := factory.Run{
		ID: a.Run, ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: strings.Repeat("c", 40),
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
	}
	view := factory.RunView{RunID: a.Run, Repository: fx.seed.repo, Issue: issue, Attempt: a.ID}
	if err := fx.db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: a.ID, Repository: a.Repository, Authority: a.Authority}, dispatchControlForAssignment(t, fx.db, a), a, r, run, view); err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Reconciled = factory.Succeeded, true
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	result := factory.AssignmentResult{
		AssignmentID: a.ID, RunID: a.Run, Status: "completed",
		Summary: "done", Candidate: strings.Repeat("2", 40),
		Findings: []string{}, Reported: true, RecordedUnix: now.Unix(),
	}
	a.Stage, a.Outcome, a.Reason, a.Result, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, &result, now.Unix()
	if err := fx.db.FinishAssignment(ctx, a); err != nil {
		t.Fatal(err)
	}
	return a
}

func pendingOutcome() factory.OperationOutcome {
	return factory.OperationOutcome{Effect: factory.OpEffectPending, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionPending}
}

func committedOutcome(receipt string) factory.OperationOutcome {
	return factory.OperationOutcome{
		Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone,
		Completion: factory.OpCompletionComplete, Receipt: []byte(receipt),
	}
}

func refusedOutcome(reason string) factory.OperationOutcome {
	return factory.OperationOutcome{
		Effect: factory.OpEffectNotCommitted, Cancellation: factory.OpCancelNone, Reason: reason,
	}
}

func happyPublisher() *fakePublisher {
	comparison := strings.Repeat("1", 40)
	published := map[string]string{}
	exec := &fakePublisher{
		ledger: map[string]factory.OperationOutcome{},
		observe: func(w factory.PublicationWork) (factory.PublicationObservation, error) {
			return factory.PublicationObservation{
				TargetRef:  factory.PublishBranchName(w.AssignmentID),
				TargetTip:  published[w.AssignmentID],
				Comparison: comparison, NativeRev: 9, ObservedUnix: time.Now().Unix(),
			}, nil
		},
		submit: func(w factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil },
		push: func(w factory.PublicationWork) (factory.OperationOutcome, error) {
			published[w.AssignmentID] = w.Candidate
			return committedOutcome(`{"ref":"` + factory.PublishBranchName(w.AssignmentID) + `"}`), nil
		},
		submitPR: func(w factory.PublicationWork) (factory.OperationOutcome, error) {
			return committedOutcome(`{"pr_number":9}`), nil
		},
		adopt: func(w factory.PublicationWork, o factory.OperationOutcome) (factory.BranchOutcome, error) {
			return factory.BranchOutcome{Operation: o, Ref: factory.PublishBranchName(w.AssignmentID), NewOID: w.Candidate}, nil
		},
		adoptPR: func(w factory.PublicationWork, o factory.OperationOutcome) (factory.PRCreationOutcome, error) {
			return factory.PRCreationOutcome{
				Operation: o, HeadRef: factory.PublishBranchName(w.AssignmentID), BaseRef: w.TargetBranch,
				HeadOID: w.Candidate, BaseOID: w.ComparisonOID,
				PRNumber: 9, PRID: 8, IssueID: 10, AuthorID: 7,
			}, nil
		},
	}
	return exec
}

func (fx *publishFixture) publication(t *testing.T, a factory.Assignment) factory.Publication {
	t.Helper()
	p, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (fx *publishFixture) pass(t *testing.T) PublishReport {
	t.Helper()
	report := fx.coord.PublishPass(context.Background())
	if len(report.Errors) != 0 {
		t.Fatalf("publication pass: %+v", report)
	}
	return report
}
