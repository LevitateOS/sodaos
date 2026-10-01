package control_test

// These tests require an explicitly supplied, task-owned native fixture.
// They exercise the real Soda controller, native observation adapter, Git
// publisher, and Fountain operations. Stopped-run/preparation records and the
// host bundle response are seeded fixtures; they do not prove CLI execution.
// Absence of SODA_ST09_NATIVE skips native work and is never passing evidence.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

type nativeST09Config struct {
	TokenID          int64  `json:"token_id"`
	FountainURL      string `json:"fountain_url"`
	Socket           string `json:"socket"`
	TokenFile        string `json:"token_file"`
	Owner            string `json:"owner"`
	Repo             string `json:"repo"`
	Repository       int64  `json:"repository"`
	ActorID          int64  `json:"actor_id"`
	BaseBranch       string `json:"base_branch"`
	CreateBound      bool   `json:"create_bound"`
	CreatorID        int64  `json:"creator_id"`
	CreatorTokenFile string `json:"creator_token_file"`
}

func loadNativeST09(t *testing.T) nativeST09Config {
	t.Helper()
	raw := os.Getenv("SODA_ST09_NATIVE")
	if raw == "" {
		t.Skip("native proof NOT RUN: SODA_ST09_NATIVE is not configured")
	}
	var cfg nativeST09Config
	nativeMust(t, json.Unmarshal([]byte(raw), &cfg))
	if cfg.FountainURL == "" || cfg.Socket == "" || cfg.TokenFile == "" || cfg.Repository <= 0 || cfg.ActorID <= 0 || cfg.TokenID <= 0 {
		t.Fatal("native fixture requires explicit repository and publisher inputs")
	}
	if cfg.CreatorID <= 0 || cfg.CreatorID == cfg.ActorID || cfg.CreatorTokenFile == "" {
		t.Fatal("native acceptance requires a distinct authorized maintainer and restricted creator_token_file")
	}
	if !factory.ValidTargetBranch(cfg.BaseBranch) {
		t.Fatal("native fixture requires exact base_branch")
	}
	if !cfg.CreateBound {
		t.Fatal("native proof suite requires both publication and PR-create bindings")
	}
	return cfg
}
func nativeMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func nativeSecret(t *testing.T, path string) string {
	t.Helper()
	s, err := config.Secret(path)
	nativeMust(t, err)
	return s
}
func nativeRepoURL(c nativeST09Config) string {
	return strings.TrimRight(c.FountainURL, "/") + "/" + c.Owner + "/" + c.Repo + ".git"
}
func nativeAPI(t *testing.T, c nativeST09Config, credential, method, path string, body, target any) {
	t.Helper()
	var input io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		nativeMust(t, err)
		input = bytes.NewReader(raw)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.FountainURL, "/")+path, input)
	nativeMust(t, err)
	req.Header.Set("Authorization", "token "+nativeSecret(t, credential))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal("native API transport unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("native fixture API %s %s: status %d", method, path, response.StatusCode)
	}
	if target != nil {
		nativeMust(t, json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target))
	}
}

// Git credentials are read from the configured restricted file by askpass;
// admission headers are supplied only as the SDK's process environment.
func nativeGit(t *testing.T, c nativeST09Config, dir string, extra []string, args ...string) (string, error) {
	t.Helper()
	home := t.TempDir()
	askpass := filepath.Join(home, "askpass")
	nativeMust(t, os.WriteFile(askpass, []byte("#!/bin/sh\ncase \"$1\" in\n *Username*) printf '%s\\n' soda-publisher ;;\n *Password*) cat \"$SODA_PUBLISH_TOKEN_FILE\" ;;\n *) exit 1 ;;\nesac\n"), 0700))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prefix := []string{"-c", "http.followRedirects=false", "-c", "core.hooksPath=/dev/null"}
	cmd := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "TMPDIR=" + os.Getenv("TMPDIR"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=" + askpass, "SODA_PUBLISH_TOKEN_FILE=" + c.TokenFile, "LC_ALL=C"}
	cmd.Env = append(cmd.Env, extra...)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("native fixture Git command failed")
	}
	return strings.TrimSpace(string(out)), nil
}
func nativeGitOK(t *testing.T, c nativeST09Config, dir string, args ...string) string {
	t.Helper()
	out, err := nativeGit(t, c, dir, nil, args...)
	nativeMust(t, err)
	return out
}

type nativeCandidate struct {
	dir, base, head string
	bundle          []byte
}

func nativeNewCandidate(t *testing.T, c nativeST09Config) nativeCandidate {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "clone")
	nativeGitOK(t, c, "", "clone", "--branch", strings.TrimPrefix(c.BaseBranch, "refs/heads/"), nativeRepoURL(c), dir)
	base := nativeGitOK(t, c, dir, "rev-parse", "HEAD")
	return nativeAppendCandidate(t, c, nativeCandidate{dir: dir, base: base})
}
func nativeAppendCandidate(t *testing.T, c nativeST09Config, n nativeCandidate) nativeCandidate {
	t.Helper()
	nativeMust(t, os.WriteFile(filepath.Join(n.dir, "README.md"), []byte("ST09 candidate "+factory.NewID()+"\n"), 0600))
	nativeGitOK(t, c, n.dir, "add", "README.md")
	nativeGitOK(t, c, n.dir, "-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit", "-m", "ST09 candidate")
	n.head = nativeGitOK(t, c, n.dir, "rev-parse", "HEAD")
	bundle := filepath.Join(t.TempDir(), "candidate.bundle")
	nativeGitOK(t, c, n.dir, "bundle", "create", bundle, "HEAD")
	var err error
	n.bundle, err = os.ReadFile(bundle)
	nativeMust(t, err)
	return n
}
func nativeTip(t *testing.T, c nativeST09Config, ref string) string {
	t.Helper()
	out := nativeGitOK(t, c, "", "ls-remote", nativeRepoURL(c), ref)
	if out == "" {
		return ""
	}
	parts := strings.Fields(out)
	if len(parts) != 2 || parts[1] != ref || !factory.ValidCommit(parts[0]) {
		t.Fatal("native ref advertisement is not exact")
	}
	return parts[0]
}

type nativeHost struct{ bundle []byte }

func (*nativeHost) FactoryLaunch(context.Context, project.FactoryLaunch) (project.FactoryState, error) {
	return project.FactoryState{}, errors.New("unexpected launch")
}
func (*nativeHost) FactoryStop(context.Context, project.FactoryStop) (project.FactoryState, error) {
	return project.FactoryState{}, errors.New("unexpected stop")
}
func (*nativeHost) FactoryInspect(context.Context, project.FactoryInspect) (project.FactoryState, error) {
	return project.FactoryState{}, errors.New("unexpected inspect")
}
func (*nativeHost) FactoryHarness(context.Context) (project.FactoryHarnessPin, error) {
	return project.FactoryHarnessPin{}, errors.New("unexpected harness")
}
func (*nativeHost) FactoryTakeover(context.Context, project.FactoryTakeover) (project.TakeoverResult, error) {
	return project.TakeoverResult{}, errors.New("unexpected takeover")
}
func (h *nativeHost) FactoryExport(_ context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	return project.FactoryExportState{ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted, Container: strings.Repeat("c", 64), Candidate: in.Candidate, Bundle: base64.StdEncoding.EncodeToString(h.bundle)}, nil
}

type nativeBroker struct{}

func (nativeBroker) GetExecution(context.Context, string, string) (identity.Execution, error) {
	return identity.Execution{}, identity.ErrNotFound
}
func (nativeBroker) CloseExecution(context.Context, string, string) error {
	return errors.New("unexpected broker closure")
}

type nativeFixture struct {
	cfg                        nativeST09Config
	db                         *store.Store
	path, project, preparation string
	coord                      *control.Coordinator
	host                       *nativeHost
	pub                        *forgejo.Publisher
	background                 *forgejo.ServiceBackground
}

func nativeSeed(t *testing.T, c nativeST09Config, n nativeCandidate) *nativeFixture {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "publication.db")
	db, err := store.Open(path)
	nativeMust(t, err)
	fx := &nativeFixture{cfg: c, db: db, path: path, project: "p" + factory.NewID()[:24], preparation: "f" + factory.NewID()[:24], host: &nativeHost{bundle: n.bundle}}
	t.Cleanup(func() { _ = fx.db.Close() })
	nativeMust(t, db.UpsertUser(ctx, store.User{ID: c.CreatorID, Login: "soda-maintainer"}))
	nativeMust(t, db.CreateProject(ctx, store.Project{ID: fx.project, Name: "ST09 fixture", RepositoryID: c.Repository, OwnerID: c.CreatorID, Repository: c.Owner + "/" + c.Repo}))
	actor := func(kind string) factory.ActorBindingRef {
		return factory.ActorBindingRef{TokenID: c.TokenID, ActorID: c.ActorID, Kind: kind}
	}
	nativeMust(t, db.SaveRepositoryPolicy(ctx, factory.RepositoryPolicy{Repository: c.Repository, GrantedBy: c.CreatorID, Enabled: true, TargetBranch: c.BaseBranch, Roles: map[string]factory.RoleSelection{project.RoleCoder: {Harness: "1.2.3", Model: "fixture"}, project.RoleReviewer: {Harness: "1.2.3", Model: "fixture"}}, Checks: []string{"verify"}, MergeMethod: factory.MergeFastForward, Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate), Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge), MaxConcurrent: 2}))
	nativeMust(t, db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: c.CreatorID, MaxConcurrentRuns: 2, MaxQueued: 10}))
	nativeMust(t, db.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: c.Repository, GrantedBy: c.CreatorID, MaxConcurrent: 2, Active: true}))
	nativeMust(t, db.SaveSponsorship(ctx, factory.Sponsorship{Repository: c.Repository, GrantedBy: c.CreatorID, Generation: 1, Connection: "fixture-connection", GrantID: "fixture-grant", Roles: []string{project.RoleCoder}, AllowanceMinutes: 120, MaxConcurrent: 2, Active: true}))
	nativeMust(t, db.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{Repository: c.Repository, Owner: c.CreatorID, Profile: &project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: n.base}, Active: true}))
	requirement, approval := "d"+factory.NewID()[:24], "d"+factory.NewID()[:24]
	digest := strings.Repeat("d", 64)
	nativeMust(t, db.AdmitRequirementDecision(ctx, project.RequirementDecision{ID: requirement, Project: fx.project, Approver: c.CreatorID, SourceCommit: n.base, SetupDigest: digest, InputsDigest: digest}))
	nativeMust(t, db.AdmitApprovalDecision(ctx, project.ApprovalDecision{ID: approval, Project: fx.project, Requirement: requirement, Approver: c.CreatorID, EffectsDigest: digest, ReadinessDigest: digest, Verified: true}))
	for _, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := fx.preparation
		if role == project.RoleReviewer {
			id = "f" + factory.NewID()[:24]
		}
		prep := project.StoredPreparation{Preparation: project.Preparation{ID: id, Project: fx.project, Role: role, Requirements: project.RequirementAcceptance{ID: requirement, Revision: 1, Approver: c.CreatorID, SourceCommit: n.base, Digest: digest}, Approval: project.AdminApproval{ID: approval, Revision: 1, Approver: c.CreatorID, EffectsDigest: digest}, SourceCommit: n.base, SetupDigest: digest, Tools: []string{"git"}}, State: project.PrepareState{ID: id, Project: fx.project, Role: role, Phase: project.PrepareReady, Ready: true}}
		_, _, err := db.AdmitPreparation(ctx, prep)
		nativeMust(t, err)
	}
	fx.background = forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	rest := forgejo.New(c.FountainURL)
	root := t.TempDir()
	nativeMust(t, os.Chmod(root, 0700))
	fx.pub = forgejo.NewPublisher(fx.background, rest, c.FountainURL, root, c.TokenFile)
	fx.wire()
	return fx
}
func (fx *nativeFixture) wire() {
	observer := forgejo.NewServiceObserver(fx.cfg.Socket, uint32(os.Getuid()), fx.cfg.TokenFile, forgejo.New(fx.cfg.FountainURL))
	observer.ShareBackground(fx.background)
	fx.coord = control.NewCoordinator(fx.db, fx.host, nativeBroker{})
	fx.coord.Publication = fx.pub
	fx.coord.AcceptanceReads = api.NewServiceReadinessSource(observer)
}
func (fx *nativeFixture) assignment(t *testing.T, n nativeCandidate) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	c := fx.cfg
	rest := forgejo.New(c.FountainURL)
	creator, err := rest.Current(ctx, nativeSecret(t, c.CreatorTokenFile))
	nativeMust(t, err)
	repo, err := rest.RepositoryByID(ctx, nativeSecret(t, c.CreatorTokenFile), c.Repository)
	nativeMust(t, err)
	if creator.ID != c.CreatorID || repo.Permissions == nil || !repo.Permissions.Push {
		t.Fatal("fixture creator does not have verified native write authority")
	}
	var issue struct {
		Number int64 `json:"number"`
	}
	nativeAPI(t, c, c.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues", map[string]string{"title": "ST09 " + factory.NewID(), "body": "Publish this fixture candidate."}, &issue)
	if issue.Number <= 0 {
		t.Fatal("native issue lacks its number")
	}
	// REST fixture creation does not establish verified original-creation
	// provenance. The distinct, authenticated code-write maintainer explicitly
	// adopts the exact native inputs through the existing acceptance path.
	evidence, err := fx.coord.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(c.Repository, 10), strconv.FormatInt(issue.Number, 10), nil)
	nativeMust(t, err)
	if len(evidence.Dependencies) != 0 {
		t.Fatal("native fixture issue unexpectedly has prerequisites")
	}
	decision := factory.Acceptance{ID: "d" + factory.NewID()[:24], Repository: c.Repository, IssueIndex: strconv.FormatInt(issue.Number, 10), Approver: creator.ID, NativeRev: evidence.Revision, TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest, ContentVersion: evidence.Issue.ContentVer}
	receipt, err := fx.coord.AdmitAcceptance(ctx, factory.NewID(), "native:"+strconv.FormatInt(creator.ID, 10), decision)
	nativeMust(t, err)
	nativeReceipt(t, t.Name()+"-acceptance", struct {
		AuthenticatedActor int64
		CodeWrite          bool
		Evidence           control.AcceptanceEvidence
		Decision           factory.Acceptance
		Receipt            control.AcceptanceReceipt
	}{creator.ID, repo.Permissions.Push, evidence, decision, receipt})
	authority, err := fx.coord.EffectiveAuthority(ctx, c.Repository)
	nativeMust(t, err)
	if !authority.Effective {
		t.Fatalf("fixture authority ineffective: %+v", authority)
	}
	bound := authority.Authority
	bound.RequirementsID, err = fx.db.RequirementHead(ctx, fx.project)
	nativeMust(t, err)
	bound.ApprovalID, err = fx.db.ApprovalHead(ctx, fx.project)
	nativeMust(t, err)
	now := time.Now().Truncate(time.Second)
	prompt := []byte("ST09 recorded stopped-run fixture")
	hash := sha256.Sum256(prompt)
	a := factory.Assignment{Authority: bound, ID: factory.NewID(), ProjectID: fx.project, Role: project.RoleCoder, Repository: c.Repository, Issue: issue.Number, NativeRev: decision.NativeRev, Acceptance: decision.ID, Preparation: fx.preparation, Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "fixture", Connection: "fixture-connection", SourceCommit: n.base, Prompt: prompt, PromptSHA: hex.EncodeToString(hash[:]), Run: factory.NewID(), Stage: factory.AssignmentAssigned, Attempts: 1, CreatedUnix: now.Unix()}
	a.RunHistory = []string{a.Run}
	run := factory.Run{ID: a.Run, ProjectID: a.ProjectID, Role: a.Role, InputSHA: n.base, Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: a.Harness, Model: a.Model}
	nativeMust(t, fx.db.RecordDispatchPacket(ctx, a, factory.Reservation{AssignmentID: a.ID, Repository: c.Repository, Connection: a.Connection, State: factory.ReservationHeld, PlannedMinutes: 30}, run, factory.RunView{RunID: a.Run, Repository: c.Repository, Issue: a.Issue, Attempt: a.ID}))
	run.Reconciled, run.Outcome, run.Summary = true, factory.Succeeded, "completed fixture candidate"
	nativeMust(t, fx.db.SaveFactoryRun(ctx, run))
	nativeMust(t, fx.db.ConsumeReservation(ctx, a.ID))
	a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, now.Unix()
	a.Result = &factory.AssignmentResult{AssignmentID: a.ID, RunID: a.Run, Status: "completed", Summary: "fixture candidate", Candidate: n.head, Findings: []string{}, Reported: true, RecordedUnix: now.Unix()}
	nativeMust(t, fx.db.FinishAssignment(ctx, a))
	return a
}
func (fx *nativeFixture) drive(t *testing.T, a factory.Assignment) factory.Publication {
	t.Helper()
	var p factory.Publication
	for i := 0; i < 20; i++ {
		report := fx.coord.PublishPass(context.Background())
		var err error
		p, err = fx.db.PublicationByAssignment(context.Background(), a.ID)
		nativeMust(t, err)
		if len(report.Errors) != 0 {
			t.Fatalf("publication errors: %+v", report)
		}
		if p.Stage != factory.PublicationOpen && p.Stage != factory.PublicationFenced {
			return p
		}
		if i == 19 {
			t.Fatalf("publication did not settle: stage=%s report=%+v", p.Stage, report)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return p
}
func nativeReceipt(t *testing.T, label string, value any) {
	t.Helper()
	root := os.Getenv("ST09_RECEIPT_DIR")
	if root == "" {
		t.Fatal("ST09_RECEIPT_DIR is required to retain native proof")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("native receipt directory must be absolute")
	}
	nativeMust(t, os.MkdirAll(root, 0700))
	raw, err := json.MarshalIndent(value, "", "  ")
	nativeMust(t, err)
	nativeMust(t, os.WriteFile(filepath.Join(root, label+".json"), append(raw, '\n'), 0600))
}

func TestNativePublishCandidate(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	p := fx.drive(t, a)
	nativeReceipt(t, "candidate", p)
	if p.Stage != factory.PublicationPublished || p.Publish.Receipt == "" || p.PRCreate.Receipt == "" || p.PRNumber <= 0 {
		t.Fatalf("candidate not linked: %+v", p)
	}
	if nativeTip(t, c, factory.PublishBranchName(a.ID)) != n.head {
		t.Fatal("published branch is not exact candidate")
	}
	var pr struct {
		ID     int64  `json:"id"`
		Number int64  `json:"number"`
		State  string `json:"state"`
		Head   struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", c.Owner, c.Repo, p.PRNumber), nil, &pr)
	if pr.ID != p.PRID || pr.Number != p.PRNumber || pr.Head.Ref != factory.PublicationBranch(a.ID) || pr.Head.SHA != n.head || pr.Base.Ref != strings.TrimPrefix(c.BaseBranch, "refs/heads/") || pr.State != "open" {
		t.Fatalf("native PR differs from receipts: %+v", pr)
	}
	again := fx.coord.PublishPass(context.Background())
	if len(again.Published) != 0 {
		t.Fatal("finished publication advanced twice")
	}
}

func (fx *nativeFixture) work(t *testing.T, a factory.Assignment, n nativeCandidate, publicationID string) factory.PublicationWork {
	t.Helper()
	run, err := fx.db.FactoryRun(context.Background(), a.Run)
	nativeMust(t, err)
	return factory.PublicationWork{Bundle: n.bundle, AssignmentID: a.ID, Publication: publicationID, RunID: a.Run, Run: run, Candidate: n.head, BaseSHA: n.base, TargetBranch: fx.cfg.BaseBranch, OperationID: factory.PublicationOperationID(publicationID, factory.OpRefPublish, 1), AuthRevision: factory.AuthRevisionFor(a.ID, publicationID, 0), ExpectedOld: "absent", ComparisonRef: fx.cfg.BaseBranch, PRTitle: factory.PRTitleFor(a.Issue), PRBody: factory.PRBodyFor(a.ID, a.Acceptance, a.Run, n.head, n.base), Repository: a.Repository, Issue: a.Issue, ActorID: fx.cfg.ActorID}
}
func (fx *nativeFixture) observe(t *testing.T, w factory.PublicationWork) factory.PublicationWork {
	t.Helper()
	observation, err := fx.pub.ObservePublication(context.Background(), w)
	nativeMust(t, err)
	w.NativeRev, w.ComparisonOID, w.NotAfter = observation.NativeRev, observation.Comparison, time.Now().Add(10*time.Minute).Unix()
	return w
}
func (fx *nativeFixture) terminal(t *testing.T, id string) factory.OperationOutcome {
	t.Helper()
	for i := 0; i < 50; i++ {
		outcome, err := fx.pub.LookupOp(context.Background(), id)
		nativeMust(t, err)
		if !outcome.NotObserved && outcome.Effect != factory.OpEffectPending {
			return outcome
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("native operation did not reach a terminal effect")
	return factory.OperationOutcome{}
}

// Correction exercises the retained production publisher primitive. Its exact
// intent is retained before dispatch and its native receipt afterward. Automatic
// reviewer/coder correction selection belongs to ST10 and is not simulated here.
func TestNativePublishCorrection(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	first := fx.drive(t, a)
	if first.Stage != factory.PublicationPublished {
		t.Fatalf("initial publication: %+v", first)
	}
	corrected := nativeAppendCandidate(t, c, n)
	w := fx.work(t, a, corrected, factory.NewID())
	w.ExpectedOld = n.head
	w.CorrectionNumber, w.CorrectionAuthor = first.PRNumber, c.ActorID
	w = fx.observe(t, w)
	nativeReceipt(t, "correction-intent", w.Intent())
	registered, err := fx.pub.SubmitPublish(context.Background(), w)
	nativeMust(t, err)
	if registered.Effect != factory.OpEffectPending {
		t.Fatalf("correction not registered: %+v", registered)
	}
	pushed, err := fx.pub.PushBranch(context.Background(), w)
	nativeMust(t, err)
	if pushed.Effect == factory.OpEffectPending {
		pushed = fx.terminal(t, w.OperationID)
	}
	nativeReceipt(t, "correction", struct {
		Initial factory.Publication
		Intent  factory.PublicationIntent
		Outcome factory.OperationOutcome
	}{first, w.Intent(), pushed})
	_, err = fx.pub.AdoptBranch(w, pushed)
	nativeMust(t, err)
	if nativeTip(t, c, factory.PublishBranchName(a.ID)) != corrected.head {
		t.Fatal("correction did not update exact recorded branch")
	}
	var pr struct {
		ID   int64 `json:"id"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", c.Owner, c.Repo, first.PRNumber), nil, &pr)
	if pr.ID != first.PRID || pr.Head.SHA != corrected.head {
		t.Fatalf("correction did not retain exact PR: %+v", pr)
	}
}

func nativeUnexpectedRefs(t *testing.T, extra bool) {
	t.Helper()
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	w := fx.observe(t, fx.work(t, a, n, factory.NewID()))
	label := "wrong-ref"
	reason := "stale_base_or_result"
	if extra {
		label = "extra-ref"
		reason = "unexpected_ref_effects"
	}
	nativeReceipt(t, label+"-intent", w.Intent())
	registered, err := fx.pub.SubmitPublish(context.Background(), w)
	nativeMust(t, err)
	if registered.Effect != factory.OpEffectPending {
		t.Fatalf("operation not registered: %+v", registered)
	}
	env, err := fx.background.PublishPushEnv(context.Background(), w.OperationID, false)
	nativeMust(t, err)
	target := factory.PublishBranchName(a.ID)
	other := target + "-unexpected"
	args := []string{"push", "--no-follow-tags", nativeRepoURL(c), n.head + ":" + other}
	if extra {
		args = append(args, n.head+":"+target)
	}
	if _, err := nativeGit(t, c, n.dir, env, args...); err == nil {
		t.Fatal("unexpected ref command set succeeded")
	}
	outcome := fx.terminal(t, w.OperationID)
	nativeReceipt(t, label, struct {
		Intent  factory.PublicationIntent
		Outcome factory.OperationOutcome
	}{w.Intent(), outcome})
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Reason != reason || outcome.OperationID != w.OperationID || outcome.Kind != factory.OpRefPublish || outcome.ActorID != w.ActorID || outcome.RepositoryID != w.Repository || outcome.InstallationID != registered.InstallationID {
		t.Fatalf("unexpected refs lack native refusal: %+v", outcome)
	}
	if nativeTip(t, c, target) != "" || nativeTip(t, c, other) != "" {
		t.Fatal("refused operation changed a ref")
	}
}
func TestNativePublishWrongRefs(t *testing.T) { nativeUnexpectedRefs(t, false) }
func TestNativePublishExtraRefs(t *testing.T) { nativeUnexpectedRefs(t, true) }

type nativeLostReplies struct {
	*forgejo.Publisher
	submits, pushes, creates int
	submitOnly               bool
}

func (p *nativeLostReplies) SubmitPublish(ctx context.Context, w factory.PublicationWork) (factory.OperationOutcome, error) {
	outcome, err := p.Publisher.SubmitPublish(ctx, w)
	if err != nil {
		return outcome, err
	}
	p.submits++
	return factory.OperationOutcome{}, errors.New("injected lost branch-registration reply")
}
func (p *nativeLostReplies) PushBranch(ctx context.Context, w factory.PublicationWork) (factory.OperationOutcome, error) {
	outcome, err := p.Publisher.PushBranch(ctx, w)
	if err != nil {
		return outcome, err
	}
	p.pushes++
	if p.submitOnly {
		return outcome, nil
	}
	return factory.OperationOutcome{}, errors.New("injected lost committed-branch reply")
}
func (p *nativeLostReplies) SubmitPRCreate(ctx context.Context, w factory.PublicationWork) (factory.OperationOutcome, error) {
	outcome, err := p.Publisher.SubmitPRCreate(ctx, w)
	if err != nil {
		return outcome, err
	}
	p.creates++
	if p.submitOnly {
		return outcome, nil
	}
	return factory.OperationOutcome{}, errors.New("injected lost PR-creation reply")
}

// The transport wrapper discards actual native answers after each operation
// returns. Every subsequent pass uses a reopened Soda database and must recover
// the original native identity without replaying a primary mutation.
func TestNativePublishLostReplies(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	drops := &nativeLostReplies{Publisher: fx.pub}
	fx.coord.Publication = drops
	var p factory.Publication
	var branchIntent, createIntent *factory.PublicationIntent
	for i := 0; i < 20; i++ {
		_ = fx.coord.PublishPass(context.Background())
		var err error
		p, err = fx.db.PublicationByAssignment(context.Background(), a.ID)
		nativeMust(t, err)
		if branchIntent == nil && p.Publish.Work != nil {
			copy := *p.Publish.Work
			branchIntent = &copy
		}
		if createIntent == nil && p.PRCreate.Work != nil {
			copy := *p.PRCreate.Work
			createIntent = &copy
		}
		if branchIntent != nil && (p.Publish.Work == nil || *branchIntent != *p.Publish.Work) {
			t.Fatal("branch replay changed its persisted intent")
		}
		if createIntent != nil && (p.PRCreate.Work == nil || *createIntent != *p.PRCreate.Work) {
			t.Fatal("creation replay changed its persisted intent")
		}
		if p.Stage == factory.PublicationPublished {
			break
		}
		if p.Stage != factory.PublicationOpen {
			t.Fatalf("lost reply advanced incorrectly: %+v", p)
		}
		nativeMust(t, fx.db.Close())
		fx.db, err = store.Open(fx.path)
		nativeMust(t, err)
		fx.wire()
		fx.coord.Publication = drops
		time.Sleep(100 * time.Millisecond)
	}
	nativeReceipt(t, "lost-replies", struct {
		Publication              factory.Publication
		Submits, Pushes, Creates int
	}{p, drops.submits, drops.pushes, drops.creates})
	if p.Stage != factory.PublicationPublished || drops.submits != 1 || drops.pushes != 1 || drops.creates != 1 {
		t.Fatalf("lost-reply recovery did not preserve one operation per effect: stage=%s calls=%d/%d/%d", p.Stage, drops.submits, drops.pushes, drops.creates)
	}
	if p.Publish.Receipt == "" || p.PRCreate.Receipt == "" {
		t.Fatal("lost reply recovery lacks exact receipts")
	}
}

func TestNativePublishWithdrawal(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	fx.coord.Publication = &nativeLostReplies{Publisher: fx.pub, submitOnly: true}
	_ = fx.coord.PublishPass(context.Background())
	p, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
	nativeMust(t, err)
	if p.Publish.Work == nil || p.Publish.OperationID == "" {
		t.Fatal("withdrawal fixture did not persist native registration")
	}
	nativeReceipt(t, "withdrawal-registered", p)
	registered, err := fx.background.GetOperation(context.Background(), p.Publish.OperationID)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-registered-native", registered)
	_, err = fx.db.WithdrawDispatch(context.Background(), c.Repository, "ST09 withdrawal fixture", "native-maintainer")
	nativeMust(t, err)
	fx.coord.Publication = fx.pub
	report := fx.coord.PublishPass(context.Background())
	p, err = fx.db.PublicationByAssignment(context.Background(), a.ID)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-cancel-pass", struct {
		Publication factory.Publication
		Report      control.PublishReport
	}{p, report})
	cancelled, err := fx.background.GetOperation(context.Background(), p.Publish.OperationID)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-cancel-native", cancelled)
	if p.Publish.Cancellation == factory.OpCancelPending && p.Stage != factory.PublicationOpen {
		t.Fatal("pending native cancellation was presented as a finished withdrawal")
	}
	w := p.Publish.Work.Apply(fx.work(t, a, n, p.ID))
	outcome, err := fx.pub.PushBranch(context.Background(), w)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-delayed-push", outcome)
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Cancellation != factory.OpCancelCancelled || outcome.Reason != "cancelled_before_admission" || nativeTip(t, c, factory.PublishBranchName(a.ID)) != "" {
		t.Fatal("delayed push escaped confirmed cancellation")
	}
	p = fx.drive(t, a)
	nativeReceipt(t, "withdrawal", p)
	if p.Stage != factory.PublicationWithdrawn || p.Publish.Effect != factory.OpEffectNotCommitted || p.Publish.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("withdrawal not confirmed: %+v", p)
	}
}

type nativeLookalike struct {
	*forgejo.Publisher
	t      *testing.T
	cfg    nativeST09Config
	number int64
}

func (p *nativeLookalike) ObservePublication(ctx context.Context, w factory.PublicationWork) (factory.PublicationObservation, error) {
	if w.ExpectedOld != "absent" && p.number == 0 {
		var pr struct {
			Number int64 `json:"number"`
		}
		// This is an independent ordinary native writer in the test fixture,
		// not a factory fallback. It races before the fresh creation bracket.
		nativeAPI(p.t, p.cfg, p.cfg.TokenFile, http.MethodPost, "/api/v1/repos/"+p.cfg.Owner+"/"+p.cfg.Repo+"/pulls", map[string]any{"head": factory.PublicationBranch(w.AssignmentID), "base": strings.TrimPrefix(w.TargetBranch, "refs/heads/"), "title": w.PRTitle, "body": w.PRBody, "allow_maintainer_edit": false}, &pr)
		if pr.Number <= 0 {
			p.t.Fatal("lookalike fixture has no native PR")
		}
		p.number = pr.Number
	}
	return p.Publisher.ObservePublication(ctx, w)
}
func TestNativePublishBranchCommittedPRFailed(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	rival := &nativeLookalike{Publisher: fx.pub, t: t, cfg: c}
	fx.coord.Publication = rival
	p := fx.drive(t, a)
	nativeReceipt(t, "branch-committed-pr-failed", struct {
		Publication factory.Publication
		LookalikePR int64
	}{p, rival.number})
	if p.Stage != factory.PublicationFailed || p.Publish.Effect != factory.OpEffectCommitted || p.PRCreate.Effect != factory.OpEffectNotCommitted || p.PRCreate.Reason != "duplicate_pull_request" || p.PRNumber != 0 {
		t.Fatalf("partial outcome adopted lookalike or lost branch effect: %+v", p)
	}
	if p.Publish.Receipt == "" || p.PRCreate.OperationID == "" || rival.number <= 0 || nativeTip(t, c, factory.PublishBranchName(a.ID)) != n.head {
		t.Fatal("partial outcome lacks linked native evidence")
	}
}
