package control_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

func (fx *st15Fixture) st09() nativeST09Config {
	return nativeST09Config{
		TokenID: fx.cfg.TokenID, FountainURL: fx.cfg.FountainURL, Socket: fx.cfg.Socket,
		TokenFile: fx.cfg.TokenFile, Owner: fx.cfg.Owner, Repo: fx.cfg.Repo,
		Repository: fx.cfg.Repository, ActorID: fx.cfg.ActorID, BaseBranch: fx.cfg.BaseBranch,
		CreatorID: fx.cfg.CreatorID, CreatorTokenFile: fx.cfg.CreatorTokenFile,
	}
}

func (fx *st15Fixture) api(tokenFile, method, path string, body any, out any) {
	fx.t.Helper()
	nativeAPI(fx.t, fx.st09(), tokenFile, method, path, body, out)
}

func (fx *st15Fixture) git(dir string, args ...string) string {
	fx.t.Helper()
	out, err := st15Git(fx.ctx, dir, args...)
	if err != nil {
		fx.t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func (fx *st15Fixture) repoURL() string {
	token, err := os.ReadFile(fx.cfg.CreatorTokenFile)
	if err != nil {
		fx.t.Fatal(err)
	}
	return strings.Replace(fx.cfg.FountainURL, "://", "://soda-maintainer:"+strings.TrimSpace(string(token))+"@", 1) + "/" + fx.cfg.Owner + "/" + fx.cfg.Repo + ".git"
}

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
			project.RoleCoder:    {Harness: fx.versions, Model: "gpt-6-luna"},
			project.RoleReviewer: {Harness: fx.versions, Model: "gpt-6-luna"},
		},
		Checks: []string{"st15-build", "st15-test"}, MergeMethod: factory.MergeFastForward,
		Publish:       actor(factory.OpRefPublish, fx.cfg.TokenID, fx.cfg.ActorID),
		Create:        actor(factory.OpPRCreate, fx.cfg.TokenID, fx.cfg.ActorID),
		Review:        actor(factory.OpReviewSubmit, fx.cfg.ReviewerTokenID, fx.cfg.ReviewerID),
		Merge:         actor(factory.OpMerge, fx.cfg.TokenID, fx.cfg.ActorID),
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
		Repository: fx.cfg.Repository, GrantedBy: owner, Generation: 1, Connection: "st15-codex", GrantID: "st15-grant",
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

// seedRepository pushes the journey base: the buggy widget, its tests and
// the CI workflow the fixture statuses attest.
func (fx *st15Fixture) seedRepository() error {
	parent := fx.t.TempDir()
	dir := filepath.Join(parent, "seed")
	if _, err := st15Git(fx.ctx, parent, "clone", "-q", fx.repoURL(), dir); err != nil {
		return err
	}
	files := map[string]string{
		"widget.py":                  "def total(items):\n    return sum(items) + 1\n",
		"test_widget.py":             "from widget import total\n\n\ndef test_total_two():\n    assert total([1, 2]) == 4\n",
		".forgejo/workflows/ci.yaml": "name: st15 CI\non: [pull_request]\njobs:\n  build:\n    runs-on: st15\n    steps:\n      - run: python3 -m py_compile widget.py\n  test:\n    runs-on: st15\n    steps:\n      - run: python3 -m pytest -q\n",
	}
	// NOTE: test_widget.py pins the BUGGY total on purpose: the seeded
	// main is internally consistent (bug uncovered), and the journey fix
	// corrects both the widget and its test. The required regression path
	// tests/test_total_regression.py only enters through the correction.
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := st15Git(fx.ctx, dir, "add", "."); err != nil {
		return err
	}
	if _, err := st15Git(fx.ctx, dir, "commit", "-q", "-m", "ST15 journey base"); err != nil {
		return err
	}
	fx.sourceHead = fx.git(dir, "rev-parse", "HEAD")
	if _, err := st15Git(fx.ctx, dir, "push", "-q", "origin", "main"); err != nil {
		return err
	}
	head := fx.sourceHead
	st15Receipt(fx.t, "seed-base", map[string]any{"head": head, "files": []string{"widget.py", "test_widget.py", ".forgejo/workflows/ci.yaml"}})
	return nil
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

// wireCoordinator wires the journey coordinator over production readers
// and native executors, plus the in-process signed intake handler.
func (fx *st15Fixture) wireCoordinator() error {
	uid := uint32(os.Getuid())
	rest := forgejo.New(fx.cfg.FountainURL)
	fx.rest = rest
	background := forgejo.NewServiceBackground(fx.cfg.Socket, uid, "")
	observer := forgejo.NewServiceObserver(fx.cfg.Socket, uid, fx.cfg.CreatorTokenFile, rest)
	observer.ShareBackground(background)
	fx.observer, fx.bg = observer, background
	source := api.NewServiceReadinessSource(observer)
	coord := control.NewCoordinator(fx.db, hostexec.NewClient(fx.cfg.HostSocket), fx.client)
	coord.AcceptanceReads = source
	coord.Readiness = source
	coord.DispatchReads = source
	// The publisher requires an existing private root (ST09 precedent:
	// t.TempDir + 0700); a bare join path fails privateInputs and the
	// pass reports native_unavailable without attempting anything.
	publishRoot := filepath.Join(fx.scratch, "publish")
	if err := os.MkdirAll(publishRoot, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(publishRoot, 0o700); err != nil {
		return err
	}
	coord.Publication = forgejo.NewPublisher(background, rest, fx.cfg.FountainURL, publishRoot, fx.cfg.TokenFile)
	coord.Reviews = forgejo.NewReviewer(background, rest, fx.cfg.ReviewerTokenFile)
	coord.Checks = forgejo.NewCheckAssessor(background, rest, fx.cfg.TokenFile)
	coord.Merges = forgejo.NewMerger(background, rest, fx.cfg.TokenFile)
	fx.coord = coord
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	fx.intake = api.IntakeHandler{Coordinator: coord, Secret: secret}
	return nil
}

// postIntake delivers one signed native webhook to the real intake
// handler and returns the decoded outcome.
func (fx *st15Fixture) postIntake(event string, payload any) map[string]any {
	fx.t.Helper()
	body, err := json.Marshal(payload)
	nativeMust(fx.t, err)
	mac := hmac.New(sha256.New, fx.intake.Secret)
	mac.Write(body)
	request := httptest.NewRequest(http.MethodPost, "/api/factory/intake", bytes.NewReader(body))
	request.Header.Set("X-Forgejo-Delivery", "st15-"+st15RandHex(8))
	request.Header.Set("X-Forgejo-Signature", hex.EncodeToString(mac.Sum(nil)))
	request.Header.Set("X-Forgejo-Event", event)
	recorder := httptest.NewRecorder()
	fx.intake.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		fx.t.Fatalf("intake %s rejected: %d %s", event, recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
	var outcome map[string]any
	nativeMust(fx.t, json.Unmarshal(recorder.Body.Bytes(), &outcome))
	return outcome
}

type st15Issue struct {
	ID     int64 `json:"id"`
	Number int64 `json:"number"`
}

func (fx *st15Fixture) createIssue(title, body string) st15Issue {
	fx.t.Helper()
	var issue st15Issue
	fx.api(fx.cfg.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues",
		map[string]any{"title": title, "body": body}, &issue)
	if issue.ID <= 0 || issue.Number <= 0 {
		fx.t.Fatalf("issue not created: %+v", issue)
	}
	return issue
}

func (fx *st15Fixture) commentIssue(index int64, body string) int64 {
	fx.t.Helper()
	var comment struct {
		ID int64 `json:"id"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(index, 10)+"/comments",
		map[string]any{"body": body}, &comment)
	if comment.ID <= 0 {
		fx.t.Fatalf("comment not created on #%d", index)
	}
	return comment.ID
}

func (fx *st15Fixture) readIssue(index int64) (title, body string, id int64) {
	fx.t.Helper()
	var issue struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(index, 10), nil, &issue)
	return issue.Title, issue.Body, issue.ID
}

func (fx *st15Fixture) readComment(commentID int64) string {
	fx.t.Helper()
	var comment struct {
		Body string `json:"body"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/comments/"+strconv.FormatInt(commentID, 10), nil, &comment)
	return comment.Body
}

func (fx *st15Fixture) insertEdge(blockedID, blockerID int64) {
	fx.t.Helper()
	nativeMust(fx.t, store.SeedStagedDependencyEdge(fx.cfg.FountainDB, fx.cfg.CreatorID, blockedID, blockerID))
}

// issueOpenedHint delivers the creation webhook for one maintainer issue.
func (fx *st15Fixture) issueOpenedHint(issue st15Issue) {
	fx.t.Helper()
	outcome := fx.postIntake("issues", map[string]any{
		"action": "opened", "number": issue.Number,
		"repository": map[string]any{"id": fx.cfg.Repository, "permissions": map[string]any{"push": true, "admin": true}},
		"sender":     map[string]any{"id": fx.cfg.CreatorID},
		"issue":      map[string]any{"index": issue.Number},
	})
	fx.t.Logf("ST15 intake opened #%d: %v", issue.Number, outcome)
}

func (fx *st15Fixture) commentHint(index int64) {
	fx.t.Helper()
	outcome := fx.postIntake("issue_comment", map[string]any{
		"repository": map[string]any{"id": fx.cfg.Repository},
		"issue":      map[string]any{"index": index},
	})
	fx.t.Logf("ST15 intake comment #%d: %v", index, outcome)
}
