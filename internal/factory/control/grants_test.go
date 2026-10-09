package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func grantPolicy() factory.RepositoryPolicy {
	return factory.RepositoryPolicy{
		Repository: 42, GrantedBy: 7, Enabled: true,
		TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
			project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
		},
		Checks:        []string{"native-ci/build"},
		MergeMethod:   factory.MergeFastForward,
		Publish:       factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
		Create:        factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
		Review:        factory.ActorBindingRef{TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
		Merge:         factory.ActorBindingRef{TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		MaxConcurrent: 2,
	}
}

func grantProfile() *project.Profile {
	return &project.Profile{
		ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
		Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
	}
}

func grantProject(t *testing.T, c *Coordinator, id string) {
	t.Helper()
	ctx := context.Background()
	if err := c.Store.UpsertUser(ctx, store.User{ID: 7, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.CreateProject(ctx, store.Project{ID: id, Name: "repo", RepositoryID: 42, OwnerID: 7, Repository: "soda-tester/repo"}); err != nil {
		t.Fatal(err)
	}
}

func grantFullAuthority(t *testing.T, c *Coordinator) {
	t.Helper()
	ctx := context.Background()
	if _, err := c.ApplyPolicy(ctx, factory.NewID(), "native:7", 0, grantPolicy()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyOperatorGrant(ctx, factory.NewID(), "os-uid:0", 0,
		factory.OperatorGrant{Repository: 42, GrantedBy: 1, Active: true, MaxConcurrent: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyCapacity(ctx, factory.NewID(), "os-uid:0", 0,
		factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplySponsorship(ctx, factory.NewID(), "native:9", 0,
		factory.Sponsorship{
			Repository: 42, GrantedBy: 9, Connection: "conn-1", GrantID: "grant-1",
			Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
		}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyEnvironmentGrant(ctx, factory.NewID(), "native:7", 0,
		project.EnvironmentGrant{Repository: 42, Owner: 7, Profile: grantProfile(), Active: true}); err != nil {
		t.Fatal(err)
	}
}

func TestApplyPolicyReplaysAndClosesDispatch(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	grantFullAuthority(t, c)
	first, err := c.EffectiveAuthority(ctx, 42)
	if err != nil || !first.Effective {
		t.Fatalf("authority: %+v %v", first, err)
	}
	registration := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42, Authority: first.Authority}
	if err = c.RegisterDispatch(ctx, registration); err != nil {
		t.Fatal("registration refused under effective authority", err)
	}
	paused := grantPolicy()
	paused.Revision, paused.Paused = 1, true
	second, err := c.ApplyPolicy(ctx, factory.NewID(), "native:7", 1, paused)
	if err != nil || !second.Withdrawn || len(second.Captured) != 1 || second.Captured[0] != registration.ID {
		t.Fatalf("withdraw: %+v %v", second, err)
	}
	if second.Effective.Effective {
		t.Fatal("paused policy reports effective authority")
	}
	stale := grantPolicy()
	stale.Revision = 1
	if _, err = c.ApplyPolicy(ctx, factory.NewID(), "native:7", 1, stale); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("stale policy revision applied", err)
	}
	command := factory.NewID()
	applied, err := c.ApplyOperatorGrant(ctx, command, "os-uid:0", 1,
		factory.OperatorGrant{Repository: 42, Revision: 1, GrantedBy: 1, Active: true, MaxConcurrent: 3})
	if err != nil {
		t.Fatal(err)
	}
	again, err := c.ApplyOperatorGrant(ctx, command, "os-uid:0", 1,
		factory.OperatorGrant{Repository: 42, Revision: 1, GrantedBy: 1, Active: true, MaxConcurrent: 3})
	if err != nil || again.CommandID != applied.CommandID || again.Revision != applied.Revision {
		t.Fatalf("replay: %+v %v", again, err)
	}
	changed := factory.OperatorGrant{Repository: 42, Revision: 1, GrantedBy: 1, Active: true, MaxConcurrent: 4}
	if _, err = c.ApplyOperatorGrant(ctx, command, "os-uid:0", 1, changed); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatal("changed payload reused the command identity", err)
	}
}

func TestSponsorshipWithdrawalKeepsDispatchWithActiveSibling(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	active := factory.Sponsorship{
		Repository: 42, GrantedBy: 9, Connection: "conn-keep", GrantID: "grant-1",
		Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
	}
	if _, err := c.ApplySponsorship(ctx, factory.NewID(), "native:9", 0, active); err != nil {
		t.Fatal(err)
	}
	leaving := active
	leaving.Connection, leaving.GrantID = "conn-leave", "grant-2"
	if _, err := c.ApplySponsorship(ctx, factory.NewID(), "native:9", 0, leaving); err != nil {
		t.Fatal(err)
	}
	leaving.Revision, leaving.Active = 1, false
	receipt, err := c.ApplySponsorship(ctx, factory.NewID(), "native:9", 1, leaving)
	if err != nil || receipt.Withdrawn {
		t.Fatalf("sibling withdrawal closed dispatch: %+v %v", receipt, err)
	}
	active.Revision, active.Active = 1, false
	last, err := c.ApplySponsorship(ctx, factory.NewID(), "native:9", 1, active)
	if err != nil || !last.Withdrawn {
		t.Fatalf("last withdrawal kept dispatch open: %+v %v", last, err)
	}
}

func TestReopenRequiresEffectiveGrants(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	if _, err := c.ApplyPolicy(ctx, factory.NewID(), "native:7", 0, grantPolicy()); err != nil {
		t.Fatal(err)
	}
	paused := grantPolicy()
	paused.Revision, paused.Paused = 1, true
	withdrawn, err := c.ApplyPolicy(ctx, factory.NewID(), "native:7", 1, paused)
	if err != nil || !withdrawn.Withdrawn {
		t.Fatalf("withdraw: %+v %v", withdrawn, err)
	}
	if _, err = c.ReopenDispatch(ctx, factory.NewID(), "native:7", 42, 1); !errors.Is(err, ErrIneffectiveAuthority) {
		t.Fatal("reopen bypassed ineffective grants", err)
	}
}

func TestRequirementAndApprovalDecisions(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	projectID := "p123456789012345678901234"
	grantProject(t, c, projectID)
	digest := strings.Repeat("d", 64)
	requirement := project.RequirementDecision{
		ID: "d123456789012345678901234", Project: projectID,
		Approver: 7, SourceCommit: strings.Repeat("e", 40), SetupDigest: digest, InputsDigest: digest,
	}
	receipt, err := c.AdmitRequirement(ctx, factory.NewID(), "native:7", requirement)
	if err != nil || receipt.Head != requirement.ID || receipt.Depth != 1 {
		t.Fatalf("requirement: %+v %v", receipt, err)
	}
	approval := project.ApprovalDecision{
		ID: "d223456789012345678901234", Project: projectID,
		Requirement: "d999999999999999999999999", Approver: 7,
		EffectsDigest: digest, ReadinessDigest: digest, Verified: true,
	}
	if _, err = c.AdmitApproval(ctx, factory.NewID(), "native:7", approval); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatal("approval bound a superseded requirement", err)
	}
	approval.Requirement = requirement.ID
	approved, err := c.AdmitApproval(ctx, factory.NewID(), "native:7", approval)
	if err != nil || approved.Head != approval.ID || approved.HoldActive {
		t.Fatalf("approval: %+v %v", approved, err)
	}
}
