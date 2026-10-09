package control_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// setupProject records the project and submits the policy, grants and
// capacity the journey runs under through the production grant path.
// Identity and project rows are fixture seeding; every grant travels the
// same coordinator calls an operator uses.
func (fx *st15Fixture) setupProject() error {
	ctx := fx.ctx
	db := fx.db
	owner := fx.cfg.CreatorID
	if err := db.UpsertUser(ctx, store.User{ID: owner, Login: "soda-maintainer"}); err != nil {
		return err
	}
	if err := db.UpsertUser(ctx, store.User{ID: fx.cfg.ReviewerID, Login: "soda-reviewer"}); err != nil {
		return err
	}
	if err := db.UpsertUser(ctx, store.User{ID: fx.cfg.ActorID, Login: "soda-merger"}); err != nil {
		return err
	}
	if err := db.CreateProject(ctx, store.Project{ID: fx.projectID, Name: "st15", RepositoryID: fx.cfg.Repository, OwnerID: owner, Repository: fx.cfg.Owner + "/" + fx.cfg.Repo}); err != nil {
		return err
	}
	if err := db.MarkReady(ctx, fx.projectID, ""); err != nil {
		return err
	}
	actor := func(kind string, tokenID, actorID int64) factory.ActorBindingRef {
		return factory.ActorBindingRef{TokenID: tokenID, ActorID: actorID, Kind: kind}
	}
	policy := factory.RepositoryPolicy{
		Repository: fx.cfg.Repository, GrantedBy: owner, Enabled: true, TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: project.FactoryHarnessMuse, HarnessVers: fx.versions, Model: "muse-spark-1.3"},
			project.RoleReviewer: {Harness: project.FactoryHarnessMuse, HarnessVers: fx.versions, Model: "muse-spark-1.3"},
		},
		Checks: []string{"st15-build", "st15-test"}, MergeMethod: factory.MergeFastForward,
		Publish:       actor(factory.OpRefPublish, fx.cfg.TokenID, fx.cfg.ActorID),
		Create:        actor(factory.OpPRCreate, fx.cfg.TokenID, fx.cfg.ActorID),
		Review:        actor(factory.OpReviewSubmit, fx.cfg.ReviewerTokenID, fx.cfg.ReviewerID),
		Merge:         actor(factory.OpMerge, fx.cfg.TokenID, fx.cfg.ActorID),
		AttemptLimits: factory.DefaultAttemptLimits(),
		MaxConcurrent: 2,
	}
	if _, err := fx.coord.ApplyPolicy(ctx, factory.NewID(), "soda-maintainer", 0, policy); err != nil {
		return err
	}
	if _, err := fx.coord.ApplyCapacity(ctx, factory.NewID(), "soda-maintainer", 0,
		factory.Capacity{UpdatedBy: owner, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
		return err
	}
	if _, err := fx.coord.ApplyOperatorGrant(ctx, factory.NewID(), "soda-maintainer", 0,
		factory.OperatorGrant{Repository: fx.cfg.Repository, GrantedBy: owner, MaxConcurrent: 2, Active: true}); err != nil {
		return err
	}
	if _, err := fx.coord.ApplySponsorship(ctx, factory.NewID(), "soda-maintainer", 0, factory.Sponsorship{
		Repository: fx.cfg.Repository, GrantedBy: owner, Generation: 1, Connection: "st15-muse", GrantID: "st15-grant",
		Roles: []string{project.RoleCoder, project.RoleReviewer}, AllowanceMinutes: 60, MaxConcurrent: 2, Active: true,
	}); err != nil {
		return err
	}
	// Revoked until the provider gate: no intake auto-dispatch may
	// launch while the journey stages its inputs. The withdrawal also
	// closes the dispatch gate; activation reopens it.
	_, err := fx.coord.ApplyEnvironmentGrant(ctx, factory.NewID(), "soda-maintainer", 0,
		project.EnvironmentGrant{Repository: fx.cfg.Repository, Owner: owner, Profile: &project.Profile{
			ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
			Architecture: "amd64", Image: fx.image, Revision: strings.Repeat("c", 40),
		}, Active: false})
	return err
}

// prepareRoles admits and prepares coder and reviewer environments from
// the seeded base through the real ST01 executor.
func (fx *st15Fixture) prepareRoles() error {
	ctx := fx.ctx
	db := fx.db
	owner := fx.cfg.CreatorID
	digest := st15SHA256([]byte(fx.sourceHead + "st15-requirements"))
	if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{
		ID: "d" + st15RandHex(12), Project: fx.projectID, Approver: owner,
		SourceCommit: fx.sourceHead, SetupDigest: digest, InputsDigest: digest,
	}); err != nil {
		return err
	}
	reqHead, err := db.RequirementHead(ctx, fx.projectID)
	if err != nil {
		return err
	}
	if err := db.AdmitApprovalDecision(ctx, project.ApprovalDecision{
		ID: "d" + st15RandHex(12), Project: fx.projectID, Requirement: reqHead, Approver: owner,
		EffectsDigest: digest, ReadinessDigest: digest, Verified: true,
	}); err != nil {
		return err
	}
	parent := fx.t.TempDir()
	dir := filepath.Join(parent, "base")
	if _, err := st15Git(ctx, parent, "clone", "-q", "--branch", "main", fx.repoURL(), dir); err != nil {
		return err
	}
	bundlePath := filepath.Join(parent, "base.bundle")
	if _, err := st15Git(ctx, dir, "bundle", "create", bundlePath, "HEAD"); err != nil {
		return err
	}
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		return err
	}
	for _, role := range []string{project.RoleCoder, project.RoleReviewer} {
		reqHead, err := db.RequirementHead(ctx, fx.projectID)
		if err != nil {
			return err
		}
		apprHead, err := db.ApprovalHead(ctx, fx.projectID)
		if err != nil {
			return err
		}
		id := "f" + st15RandHex(12)
		files := map[string][]byte{
			"setup.sh": []byte("#!/bin/sh\n# ST15 fixture: the journey base needs no build.\nexit 0\n"),
			"check.sh": []byte("#!/bin/sh\n/usr/bin/git --version\n/usr/bin/python3 --version\n"),
		}
		prep := project.Preparation{
			ID: id, Project: fx.projectID, Role: role,
			Requirements: project.RequirementAcceptance{ID: reqHead, Revision: 1, Approver: owner, SourceCommit: fx.sourceHead, Digest: digest},
			Approval:     project.AdminApproval{ID: apprHead, Revision: 1, Approver: owner, EffectsDigest: digest},
			SourceCommit: fx.sourceHead, SetupDigest: project.SetupDigestOf(files), Tools: []string{"git", "python3"},
		}
		if _, admitted, err := db.AdmitPreparation(ctx, project.StoredPreparation{Preparation: prep}); err != nil || !admitted {
			return fmt.Errorf("admit %s: admitted=%v err=%v", id, admitted, err)
		}
		state, err := hostexec.NewClient(fx.cfg.HostSocket).Prepare(ctx, project.Prepare{Preparation: prep, Setup: project.ApprovedSetup{Files: files, Bundle: bundle}})
		if err != nil {
			return err
		}
		if state.ID != id {
			return fmt.Errorf("prepare returned foreign state: %+v", state)
		}
		deadline := time.Now().Add(15 * time.Minute)
		for {
			state, err = hostexec.NewClient(fx.cfg.HostSocket).InspectPreparation(ctx, project.PrepareInspect{Project: fx.projectID, ID: id})
			if err != nil {
				return err
			}
			if state.Phase == project.PrepareReady {
				stored, err := db.Preparation(ctx, id)
				if err != nil {
					return err
				}
				state.ID, state.Project, state.Role = id, fx.projectID, role
				stored.State = state
				if err := db.ObservePreparation(ctx, stored); err != nil {
					return err
				}
				if role == project.RoleCoder {
					fx.coderPrep = id
				} else {
					fx.reviewPrep = id
				}
				fx.t.Logf("ST15 preparation %s ready", id)
				break
			}
			if state.Phase == project.PrepareFailed || state.Phase == project.PrepareStopped {
				return fmt.Errorf("preparation %s ended %s", id, state.Phase)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("preparation %s not ready: %s", id, state.Phase)
			}
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}

// seedHumanWork creates retained member state in the project container:
// dirty work files, service data, an authorized key and a live member
// process. Every byte is snapshotted for the end-of-journey comparison.
func (fx *st15Fixture) seedHumanWork() error {
	ctx := fx.ctx
	const member = "soda-tester"
	name := fx.container
	if _, err := st15Pexec(ctx, name, "", nil, "/usr/bin/id", member); err != nil {
		if _, err := st15Pexec(ctx, name, "", nil, "/usr/sbin/useradd", "--create-home", "--shell", "/bin/bash", member); err != nil {
			return fmt.Errorf("member useradd: %w", err)
		}
	}
	for _, dir := range []string{"/home/" + member + "/work", "/home/" + member + "/service-data", "/home/" + member + "/.ssh"} {
		if _, err := st15Pexec(ctx, name, "", nil, "/usr/bin/mkdir", "-p", dir); err != nil {
			return err
		}
	}
	files := map[string]string{
		"/home/" + member + "/work/dirty.txt":        "human dirty work must survive every control\n",
		"/home/" + member + "/service-data/rows.txt": "service row 1\nservice row 2\n",
		"/home/" + member + "/.ssh/authorized_keys":  "ssh-ed25519 ST15-FIXTURE-KEY-NOT-A-KEY soda-tester\n",
	}
	for path, content := range files {
		if _, err := st15Pexec(ctx, name, "", []byte(content), "/usr/bin/tee", path); err != nil {
			return err
		}
	}
	if _, err := st15Pexec(ctx, name, "", nil, "/usr/bin/chown", "-R", member+":"+member, "/home/"+member); err != nil {
		return err
	}
	if _, err := st15Pexec(ctx, name, member, nil, "/usr/bin/true"); err != nil {
		return fmt.Errorf("member execution unavailable: %v", err)
	}
	// The retained "service": one member-owned sleeper whose PID must
	// still be alive after every factory intervention.
	pidOut, err := st15Pexec(ctx, name, member, nil, "/usr/bin/sh", "-c", "sleep 86400 & echo $!")
	if err != nil {
		return err
	}
	snapshot := map[string]string{}
	for path := range files {
		raw, err := st15Pexec(ctx, name, "", nil, "/usr/bin/sha256sum", path)
		if err != nil {
			return err
		}
		snapshot[path] = strings.Fields(strings.TrimSpace(string(raw)))[0]
	}
	snapshot["service_pid"] = strings.TrimSpace(string(pidOut))
	st15Receipt(fx.t, "human-snapshot", snapshot)
	return nil
}
