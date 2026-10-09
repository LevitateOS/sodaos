package store

import (
	"context"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestST10PublishableAssignmentsExcludeReviewer(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	coder := recordFinishedAssignment(t, db, now, 3, true, "completed")
	reviewer, reservation, run, view := finishedPublishableAssignment(t, now, 4, true, "completed")
	reviewer.Role, run.Role = project.RoleReviewer, project.RoleReviewer
	assigned := reviewer
	assigned.Stage, assigned.Outcome, assigned.Reason, assigned.Result, assigned.FinishedUnix = factory.AssignmentAssigned, "", "", nil, 0
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(assigned), assigned, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishAssignment(ctx, reviewer); err != nil {
		t.Fatal(err)
	}
	got, err := db.PublishableAssignments(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != coder.ID {
		t.Fatal("publication queue did not select only the coder assignment")
	}
}
