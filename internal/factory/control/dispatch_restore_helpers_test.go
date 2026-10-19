package control

import (
	"context"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// Helpers for the restored dispatch/settlement integration tests. These
// seed through the real store on the disposable PostgreSQL fixture; the
// only doubles are external-system boundaries (Forgejo check observation).

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

// checkSeed finishes a reported assignment through publication on real PG
// and returns the published publication.
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

// correctionRun settles one coder child result carrying the given candidate.
func correctionRun(t *testing.T, fx *publishFixture, a factory.Assignment, candidate string) factory.Run {
	t.Helper()
	publication := fx.publication(t, a)
	_, run := fx.childRun(t, a, publication, project.RoleCoder, publication.Candidate, a.Preparation, true)
	accountChildResult(t, fx, run, correctionOutput(candidate))
	return run
}

func correctionOutput(candidate string) string {
	return "fixed the finding\n```" + factory.ResultFence + "\n{\"status\":\"completed\",\"summary\":\"fixed\",\"candidate\":\"" + candidate + "\",\"review_passed\":false,\"findings\":[]}\n```\n"
}

// phaseTestRun builds a run record bound to the given assignment.
func phaseTestRun(a factory.Assignment, outcome factory.Outcome) factory.Run {
	return factory.Run{
		ID: a.Run, ProjectID: a.ProjectID, Role: a.Role, InputSHA: a.SourceCommit,
		Started: time.Unix(1700000000, 0), Deadline: time.Unix(1700003600, 0),
		Image: "img", Harness: a.Harness, Model: a.Model, Outcome: outcome,
	}
}
