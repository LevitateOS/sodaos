package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

const (
	prepTestProject = "p0123456789abcdef01234567"
	prepTestID      = "f0123456789abcdef01234567"
	prepTestDigest  = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	prepTestCommit  = "0123456789abcdef0123456789abcdef01234567"
)

func prepTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "prep.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	if err = s.UpsertUser(ctx, User{ID: 3, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	profile := &project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
	if err = s.CreateProject(ctx, Project{ID: prepTestProject, Name: "n", RepositoryID: 7, OwnerID: 3, Repository: "o/n", Profile: profile}); err != nil {
		t.Fatal(err)
	}
	return s, ctx
}

func prepTestPreparation() project.StoredPreparation {
	return project.StoredPreparation{Preparation: project.Preparation{
		ID: prepTestID, Project: prepTestProject, Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Revision: 1, Approver: 7, SourceCommit: prepTestCommit, Digest: prepTestDigest},
		Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Revision: 1, Approver: 9, EffectsDigest: prepTestDigest},
		SourceCommit: prepTestCommit, SetupDigest: prepTestDigest, Tools: []string{"python3"},
	}}
}

func TestLifecycleGrantRevisionRejectsStaleWriters(t *testing.T) {
	s, ctx := prepTestStore(t)
	grant := project.LifecycleGrant{Project: prepTestProject, Owner: 3, Active: true}
	profile := &project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
	grant.Profile = profile
	if err := s.SaveLifecycleGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	stored, err := s.LifecycleGrant(ctx, prepTestProject)
	if err != nil || !stored.Active || stored.Revision != 1 {
		t.Fatalf("grant: %+v %v", stored, err)
	}
	stored.Active = false
	if err = s.SaveLifecycleGrant(ctx, stored); err != nil {
		t.Fatal(err)
	}
	grant.Active = true
	if err = s.SaveLifecycleGrant(ctx, grant); err == nil {
		t.Fatal("stale grant revision accepted")
	}
}

func TestMaintenanceHoldPersistsWithCAS(t *testing.T) {
	s, ctx := prepTestStore(t)
	if _, err := s.MaintenanceHold(ctx, prepTestProject); err != sql.ErrNoRows {
		t.Fatalf("missing hold: %v", err)
	}
	if err := s.SaveMaintenanceHold(ctx, project.MaintenanceHold{Project: prepTestProject, Hold: true}); err != nil {
		t.Fatal(err)
	}
	held, err := s.MaintenanceHold(ctx, prepTestProject)
	if err != nil || !held.Hold || held.Revision != 1 {
		t.Fatalf("hold: %+v %v", held, err)
	}
	if err = s.SaveMaintenanceHold(ctx, project.MaintenanceHold{Project: prepTestProject, Revision: 1}); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveMaintenanceHold(ctx, project.MaintenanceHold{Project: prepTestProject, Revision: 1, Hold: true}); err == nil {
		t.Fatal("stale hold revision accepted")
	}
}

func TestAdmitPreparationIsIdempotentForExactInputs(t *testing.T) {
	s, ctx := prepTestStore(t)
	first, admitted, err := s.AdmitPreparation(ctx, prepTestPreparation())
	if err != nil || !admitted {
		t.Fatalf("admit: %v %v", admitted, err)
	}
	second, admitted, err := s.AdmitPreparation(ctx, prepTestPreparation())
	if err != nil || admitted || second.Preparation.ID != first.Preparation.ID {
		t.Fatalf("repeat: %v %v %+v", admitted, err, second)
	}
	changed := prepTestPreparation()
	changed.Preparation.SetupDigest = strings.Repeat("c", 64)
	if _, _, err = s.AdmitPreparation(ctx, changed); err == nil {
		t.Fatal("changed inputs reused the preparation identity")
	}
}

func TestPreparationRefsAreImmutable(t *testing.T) {
	s, ctx := prepTestStore(t)
	if _, _, err := s.AdmitPreparation(ctx, prepTestPreparation()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE project_preparations SET requirements='d99999999999999999999999' WHERE id=?`, prepTestID); err == nil {
		t.Fatal("requirement reference mutation accepted")
	}
	current, err := s.Preparation(ctx, prepTestID)
	if err != nil {
		t.Fatal(err)
	}
	current.State = project.PrepareState{ID: prepTestID, Project: prepTestProject, Role: project.RoleCoder, Phase: project.PrepareReady, Ready: true}
	if err = s.ObservePreparation(ctx, current); err != nil {
		t.Fatal(err)
	}
	if err = s.ObservePreparation(ctx, current); err == nil {
		t.Fatal("stale preparation observation accepted")
	}
	observed, err := s.Preparation(ctx, prepTestID)
	if err != nil || !observed.State.Ready || observed.Preparation.Revision != 1 {
		t.Fatalf("observed: %+v %v", observed, err)
	}
	listed, err := s.ProjectPreparations(ctx, prepTestProject)
	if err != nil || len(listed) != 1 || listed[0].Preparation.Role != project.RoleCoder {
		t.Fatalf("listed: %+v %v", listed, err)
	}
}
