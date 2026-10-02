package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

type fakeCheckObserver struct {
	observe func(factory.CheckTarget) (factory.ObservedChecks, error)

	targets []factory.CheckTarget
	actors  []int64
}

func (f *fakeCheckObserver) ObserveChecks(ctx context.Context, target factory.CheckTarget, actorID int64) (factory.ObservedChecks, error) {
	f.targets = append(f.targets, target)
	f.actors = append(f.actors, actorID)
	return f.observe(target)
}

func happyCheckObserver() *fakeCheckObserver {
	return &fakeCheckObserver{
		observe: func(target factory.CheckTarget) (factory.ObservedChecks, error) {
			return factory.ObservedChecks{
				Checks:           []factory.ObservedCheck{{Context: "ci", State: factory.CheckStateSuccess}},
				NativeRev:        9,
				ObservedContexts: 1,
				HeadTip:          target.HeadOID,
				BaseTip:          target.BaseOID,
				Complete:         true,
			}, nil
		},
	}
}

func checkSeed(t *testing.T, issue int64) (*publishFixture, factory.Publication) {
	t.Helper()
	fx := publishSeed(t)
	a := fx.finishReported(t, issue)
	fx.wire(happyPublisher())
	fx.pass(t)
	p, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stage != factory.PublicationPublished {
		t.Fatalf("check seed publication not published: %+v", p)
	}
	return fx, p
}

func TestAssessPublicationChecksRecordsPass(t *testing.T) {
	fx, p := checkSeed(t, 3)
	observer := happyCheckObserver()
	report := fx.coord.AssessPublicationChecks(context.Background(), observer, 7, p.AssignmentID)
	if report.Unavailable || len(report.Errors) != 0 || len(report.Waits) != 0 || len(report.Assessed) != 1 {
		t.Fatalf("assess report: %+v", report)
	}
	link := report.Assessed[0]
	if link.PublicationID != p.ID || link.HeadOID != p.Candidate || link.Verdict != factory.CheckPass || link.Revision <= 0 {
		t.Fatalf("assess link: %+v", link)
	}
	if len(observer.targets) != 1 || observer.actors[0] != 7 {
		t.Fatalf("assess observation: %+v %+v", observer.targets, observer.actors)
	}
	target := observer.targets[0]
	if target.Repository != p.Repository || target.PRNumber != p.PRNumber || target.PRID != p.PRID ||
		target.HeadOID != p.Candidate || target.BaseOID != p.PRCreate.BaseOID {
		t.Fatalf("assess target: %+v", target)
	}
	stored, err := fx.db.CheckAssessment(context.Background(), p.Repository, p.PRNumber)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Verdict != factory.CheckPass || stored.HeadOID != p.Candidate || stored.Revision != link.Revision {
		t.Fatalf("stored assessment: %+v", stored)
	}
}

func TestAssessPublicationChecksRecordsFailure(t *testing.T) {
	fx, p := checkSeed(t, 4)
	observer := happyCheckObserver()
	observer.observe = func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, _ := happyCheckObserver().observe(target)
		observed.Checks = []factory.ObservedCheck{{Context: "ci", State: "failure"}}
		return observed, nil
	}
	report := fx.coord.AssessPublicationChecks(context.Background(), observer, 7, p.AssignmentID)
	if len(report.Errors) != 0 || len(report.Assessed) != 1 || report.Assessed[0].Verdict != factory.CheckFailed {
		t.Fatalf("failed verdict unrecorded: %+v", report)
	}
}

func TestAssessPublicationChecksWaitsAndErrors(t *testing.T) {
	fx, p := checkSeed(t, 5)
	if report := fx.coord.AssessPublicationChecks(context.Background(), nil, 7, p.AssignmentID); !report.Unavailable || len(report.Assessed) != 0 {
		t.Fatalf("missing assessor must wait: %+v", report)
	}
	busy := happyCheckObserver()
	busy.observe = func(factory.CheckTarget) (factory.ObservedChecks, error) {
		return factory.ObservedChecks{}, &factory.PublicationWait{Reason: "native_busy"}
	}
	if report := fx.coord.AssessPublicationChecks(context.Background(), busy, 7, p.AssignmentID); len(report.Waits) != 1 || report.Waits[0].Reason != "native_busy" {
		t.Fatalf("busy observation must wait: %+v", report)
	}
	refused := happyCheckObserver()
	refused.observe = func(factory.CheckTarget) (factory.ObservedChecks, error) {
		return factory.ObservedChecks{}, &factory.PublicationRefusal{Reason: "pr_merged"}
	}
	if report := fx.coord.AssessPublicationChecks(context.Background(), refused, 7, p.AssignmentID); len(report.Errors) != 1 || report.Errors[0].Reason != "pr_merged" {
		t.Fatalf("refused observation must error: %+v", report)
	}
	broken := happyCheckObserver()
	broken.observe = func(factory.CheckTarget) (factory.ObservedChecks, error) {
		return factory.ObservedChecks{}, errors.New("transport down")
	}
	if report := fx.coord.AssessPublicationChecks(context.Background(), broken, 7, p.AssignmentID); len(report.Errors) != 1 || report.Errors[0].Reason != "checks_unavailable" {
		t.Fatalf("broken observation must error: %+v", report)
	}
	if report := fx.coord.AssessPublicationChecks(context.Background(), happyCheckObserver(), 7, factory.NewID()); len(report.Waits) != 1 || report.Waits[0].Reason != "publication_missing" {
		t.Fatalf("missing publication must wait: %+v", report)
	}
}
