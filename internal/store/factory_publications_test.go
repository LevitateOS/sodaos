package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func publicationStoreFixture(t *testing.T) *Store {
	t.Helper()
	db := grantStoreFixture(t)
	ctx := context.Background()
	projectID := "p" + strings.Repeat("d", 24)
	requirementID, approvalID := "d"+strings.Repeat("c", 24), "d"+strings.Repeat("e", 24)
	policy := grantTestPolicy()
	policy.Repository = 7
	for _, err := range []error{
		db.UpsertUser(ctx, User{ID: 7, Login: "soda-tester"}),
		db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 7, MaxConcurrentRuns: 2, MaxQueued: 10}),
		db.SaveRepositoryPolicy(ctx, policy),
		db.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: 7, GrantedBy: 7, Active: true, MaxConcurrent: 2}),
		db.SaveSponsorship(ctx, factory.Sponsorship{Repository: 7, GrantedBy: 7, Connection: "conn", GrantID: "grant", Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true}),
		db.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{Repository: 7, Owner: 7, Active: true, Profile: &project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40)}}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.CreateProject(ctx, Project{ID: projectID, Name: "factory", RepositoryID: 7, OwnerID: 7, Repository: "soda-tester/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{ID: requirementID, Project: projectID, Approver: 7, SourceCommit: strings.Repeat("1", 40), SetupDigest: strings.Repeat("a", 64), InputsDigest: strings.Repeat("b", 64)}); err != nil {
		t.Fatal(err)
	}
	if err := db.AdmitApprovalDecision(ctx, project.ApprovalDecision{ID: approvalID, Project: projectID, Requirement: requirementID, Approver: 7, EffectsDigest: strings.Repeat("a", 64), ReadinessDigest: strings.Repeat("b", 64), Verified: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.AdmitAcceptanceDecision(ctx, factory.Acceptance{ID: "d" + strings.Repeat("a", 24), Repository: 7, IssueIndex: "3", Approver: 7, NativeRev: 9, TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64)}); err != nil {
		t.Fatal(err)
	}
	return db
}

func publicationTestRecord() factory.Publication {
	return factory.Publication{
		Authority:    factory.AuthorityRef{Policy: 1, Operator: 1, Environment: 1, Sponsorship: 1, RequirementsID: "d" + strings.Repeat("c", 24), ApprovalID: "d" + strings.Repeat("e", 24)},
		Publish:      factory.PublicationOperation{Kind: factory.OpRefPublish},
		PRCreate:     factory.PublicationOperation{Kind: factory.OpPRCreate},
		ID:           factory.NewID(),
		AssignmentID: factory.NewID(),
		ProjectID:    "p" + strings.Repeat("d", 24),
		Role:         project.RoleCoder,
		Acceptance:   "d" + strings.Repeat("a", 24),
		Preparation:  "f" + strings.Repeat("b", 24),
		Run:          factory.NewID(),
		Candidate:    strings.Repeat("2", 40),
		BaseSHA:      strings.Repeat("1", 40),
		TargetBranch: "refs/heads/main",
		Stage:        factory.PublicationOpen,
		Repository:   7,
		Issue:        3,
		CreatedUnix:  1100,
	}
}

func TestRecordPublicationRoundTrip(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	p := publicationTestRecord()
	if err := db.RecordPublication(ctx, p); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, err := db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.ID != p.ID || got.Stage != factory.PublicationOpen {
		t.Fatalf("round trip: %+v", got)
	}
	if err := db.RecordPublication(ctx, p); err == nil {
		t.Fatal("duplicate assignment publication accepted")
	}
	if _, err := db.PublicationByAssignment(ctx, factory.NewID()); err == nil {
		t.Fatal("missing publication read accepted")
	}
}

func TestUpdatePublicationComparesAndSwaps(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	p := publicationTestRecord()
	seedPublicationAssignment(t, db, p)
	if err := db.RecordPublication(ctx, p); err != nil {
		t.Fatalf("record: %v", err)
	}
	p.Revision++
	p.NativeRev, p.ObservedUnix = 9, 1150
	p.Publish = factory.PublicationOperation{
		Work:        publicationStoreIntent("soda-test-publish-1"),
		OperationID: "soda-test-publish-1", Kind: factory.OpRefPublish,
		Effect: factory.OpEffectPending, Attempts: 1, UpdatedUnix: 1150,
	}
	if err := db.UpdatePublication(ctx, p); err != nil {
		t.Fatalf("update: %v", err)
	}
	stale := p
	stale.Revision = 1
	stale.Comparison = strings.Repeat("c", 40)
	if err := db.UpdatePublication(ctx, stale); err == nil {
		t.Fatal("stale publication update accepted")
	}
	got, err := db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Revision != 1 || got.Publish.OperationID != "soda-test-publish-1" {
		t.Fatalf("stored: %+v", got)
	}
}

func finishedPublishableAssignment(t *testing.T, now time.Time, issue int64, reported bool, status string) (factory.Assignment, factory.Reservation, factory.Run, factory.RunView) {
	t.Helper()
	a, r, run, view := dispatchTestPacket(t, now)
	a.Issue = issue
	view.Issue = issue
	result := factory.AssignmentResult{
		AssignmentID: a.ID, RunID: a.Run, Status: status,
		Summary: "done", Candidate: strings.Repeat("2", 40),
		Findings: []string{}, Reported: reported, RecordedUnix: now.Unix(),
	}
	a.Stage, a.Outcome, a.Reason, a.Result, a.FinishedUnix =
		factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, &result, now.Unix()
	return a, r, run, view
}

func recordFinishedAssignment(t *testing.T, db *Store, now time.Time, issue int64, reported bool, status string) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	a, r, run, view := finishedPublishableAssignment(t, now, issue, reported, status)
	assigned := a
	assigned.Stage, assigned.Outcome, assigned.Reason, assigned.Result, assigned.FinishedUnix =
		factory.AssignmentAssigned, "", "", nil, 0
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(assigned), assigned, r, run, view); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishAssignment(ctx, a); err != nil {
		t.Fatal(err)
	}
	settled := run
	settled.Outcome, settled.Reconciled = factory.Succeeded, true
	if err := db.SaveFactoryRun(ctx, settled); err != nil {
		t.Fatal(err)
	}
	if err := db.ReleaseReservation(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPublishableAssignmentsSelectsReportedCompleted(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	want := recordFinishedAssignment(t, db, now, 3, true, "completed")
	recordFinishedAssignment(t, db, now, 4, false, "completed")
	recordFinishedAssignment(t, db, now, 5, true, "blocked")
	got, err := db.PublishableAssignments(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ID != want.ID {
		t.Fatalf("publishable: %+v", got)
	}
	p := publicationTestRecord()
	p.AssignmentID = want.ID
	p.Repository, p.Issue = want.Repository, want.Issue
	if err := db.RecordPublication(ctx, p); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, err = db.PublishableAssignments(ctx, 10)
	if err != nil {
		t.Fatalf("relist: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("published assignment still publishable: %+v", got)
	}
}

func TestOutstandingPublicationsListsOpenAndFenced(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	open := publicationTestRecord()
	if err := db.RecordPublication(ctx, open); err != nil {
		t.Fatalf("record open: %v", err)
	}
	fenced := publicationTestRecord()
	fenced.AssignmentID = factory.NewID()
	if err := db.RecordPublication(ctx, fenced); err != nil {
		t.Fatalf("record fenced: %v", err)
	}
	fenced.Stage, fenced.Outcome, fenced.Reason, fenced.FinishedUnix =
		factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonFenced, 1200
	fenced.Revision++
	if err := db.UpdatePublication(ctx, fenced); err != nil {
		t.Fatalf("fence: %v", err)
	}
	done := publicationTestRecord()
	done.AssignmentID = factory.NewID()
	done.Issue = 9
	if err := db.RecordPublication(ctx, done); err != nil {
		t.Fatalf("record done: %v", err)
	}
	done.Stage, done.Outcome, done.Reason, done.FinishedUnix =
		factory.PublicationFailed, factory.Failed, factory.PublishReasonRefused, 1200
	done.Revision++
	if err := db.UpdatePublication(ctx, done); err != nil {
		t.Fatalf("fail: %v", err)
	}
	got, err := db.OutstandingPublications(ctx, 7, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("outstanding: %+v", got)
	}
	if got[0].AssignmentID != open.AssignmentID || got[1].AssignmentID != fenced.AssignmentID {
		t.Fatalf("order: %+v", got)
	}
}

func publicationStoreIntent(id string) *factory.PublicationIntent {
	return &factory.PublicationIntent{
		OperationID: id, AuthRevision: "accepted-revision", Candidate: strings.Repeat("2", 40),
		TargetBranch: "refs/heads/main", ExpectedOld: "absent", ComparisonRef: "refs/heads/main", ComparisonOID: strings.Repeat("1", 40),
		PRTitle: "Factory candidate for #3", Repository: 7, ActorID: 5, NativeRev: 9, NotAfter: 1300,
	}
}

func seedPublicationAssignment(t *testing.T, db *Store, p factory.Publication) {
	t.Helper()
	a, reservation, run, view := dispatchTestPacket(t, time.Now())
	a.ID, a.Run, a.RunHistory = p.AssignmentID, p.Run, []string{p.Run}
	a.Repository, a.Issue, a.ProjectID, a.Acceptance, a.Authority = p.Repository, p.Issue, p.ProjectID, p.Acceptance, p.Authority
	reservation.AssignmentID, reservation.Repository = a.ID, a.Repository
	run.ID, run.ProjectID = a.Run, a.ProjectID
	view.RunID, view.Attempt, view.Repository, view.Issue = a.Run, a.ID, a.Repository, a.Issue
	if err := db.RecordDispatchPacket(context.Background(), dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}
}
