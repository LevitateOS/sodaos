package control_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/project"
)

// nativeMergeSetup seeds one assignment with the reviewer binding
// separated from the author, and wires the production merger. It waits
// for native quiescence first: back-to-back tests share one fixture,
// and setup writes must not race the previous test's merge activity.
func nativeMergeSetup(t *testing.T, c nativeST12Config) (*nativeFixture, nativeCandidate) {
	t.Helper()
	nativeMergeIdle(t, forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""))
	n := nativeNewCandidate(t, c.nativeST09Config)
	fx := nativeSeed(t, c.nativeST09Config, n)
	ctx := context.Background()
	policy, err := fx.db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	policy.Review = factory.ActorBindingRef{TokenID: c.ReviewerTokenID, ActorID: c.ReviewerID, Kind: factory.OpReviewSubmit}
	nativeMust(t, fx.db.SaveRepositoryPolicy(ctx, policy))
	fx.coord.Merges = forgejo.NewMerger(fx.background, forgejo.New(c.FountainURL), c.TokenFile)
	return fx, n
}

// nativeMergeAssignment mirrors the shared fixture assignment, except the
// native issue write retries the transient busy-host 503 through
// nativeMergeAPI instead of failing fast.
func nativeMergeAssignment(t *testing.T, c nativeST12Config, fx *nativeFixture, n nativeCandidate) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	cfg := fx.cfg
	rest := forgejo.New(cfg.FountainURL)
	creator, err := rest.Current(ctx, nativeSecret(t, cfg.CreatorTokenFile))
	nativeMust(t, err)
	repo, err := rest.RepositoryByID(ctx, nativeSecret(t, cfg.CreatorTokenFile), cfg.Repository)
	nativeMust(t, err)
	if creator.ID != cfg.CreatorID || repo.Permissions == nil || !repo.Permissions.Push {
		t.Fatal("fixture creator does not have verified native write authority")
	}
	var issue struct {
		Number int64 `json:"number"`
	}
	nativeMergeAPI(t, c, http.MethodPost, "/api/v1/repos/"+cfg.Owner+"/"+cfg.Repo+"/issues", map[string]string{"title": "ST12 " + factory.NewID(), "body": "Merge this fixture candidate."}, &issue)
	if issue.Number <= 0 {
		t.Fatal("native issue lacks its number")
	}
	evidence, err := fx.coord.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(cfg.Repository, 10), strconv.FormatInt(issue.Number, 10), nil)
	nativeMust(t, err)
	if len(evidence.Dependencies) != 0 {
		t.Fatal("native fixture issue unexpectedly has prerequisites")
	}
	decision := factory.Acceptance{ID: "d" + factory.NewID()[:24], Repository: cfg.Repository, IssueIndex: strconv.FormatInt(issue.Number, 10), Approver: creator.ID, NativeRev: evidence.Revision, TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest, ContentVersion: evidence.Issue.ContentVer}
	receipt, err := fx.coord.AdmitAcceptance(ctx, factory.NewID(), "native:"+strconv.FormatInt(creator.ID, 10), decision)
	nativeMust(t, err)
	nativeReceipt(t, t.Name()+"-acceptance", struct {
		AuthenticatedActor int64
		CodeWrite          bool
		Evidence           control.AcceptanceEvidence
		Decision           factory.Acceptance
		Receipt            control.AcceptanceReceipt
	}{creator.ID, repo.Permissions.Push, evidence, decision, receipt})
	authority, err := fx.coord.EffectiveAuthority(ctx, cfg.Repository)
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
	prompt := []byte("ST12 recorded stopped-run fixture")
	hash := sha256.Sum256(prompt)
	assignmentID := factory.NewID()
	a := factory.Assignment{Authority: bound, ID: assignmentID, AttemptRoot: assignmentID, PublicationAssignment: assignmentID, ProjectID: fx.project, ActorID: c.CreatorID, Role: project.RoleCoder, Repository: cfg.Repository, Issue: issue.Number, NativeRev: decision.NativeRev, Acceptance: decision.ID, Preparation: fx.preparation, Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "fixture", Connection: "fixture-connection", SourceCommit: n.base, Prompt: prompt, PromptSHA: hex.EncodeToString(hash[:]), Run: factory.NewID(), Stage: factory.AssignmentAssigned, Attempts: 1, CreatedUnix: now.Unix()}
	a.RunHistory = []string{a.Run}
	run := factory.Run{ID: a.Run, ProjectID: a.ProjectID, Role: a.Role, InputSHA: n.base, Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: a.Harness, Model: a.Model}
	nativeMust(t, fx.db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: a.ID, Repository: a.Repository, Authority: a.Authority}, nativeDispatchControl(t, fx.db, a), a, factory.Reservation{AssignmentID: a.ID, Repository: cfg.Repository, Connection: a.Connection, State: factory.ReservationHeld, PlannedMinutes: 30}, run, factory.RunView{RunID: a.Run, Repository: cfg.Repository, Issue: a.Issue, Attempt: a.ID}))
	run.Reconciled, run.Outcome, run.Summary = true, factory.Succeeded, "completed fixture candidate"
	nativeMust(t, fx.db.SaveFactoryRun(ctx, run))
	nativeMust(t, fx.db.ConsumeReservation(ctx, a.ID))
	a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, now.Unix()
	a.Result = &factory.AssignmentResult{AssignmentID: a.ID, RunID: a.Run, Status: "completed", Summary: "fixture candidate", Candidate: n.head, Findings: []string{}, Reported: true, RecordedUnix: now.Unix()}
	nativeMust(t, fx.db.FinishAssignment(ctx, a))
	return a
}

// nativeMergeApprove submits one conditional review through the
// production reviewer and requires its native commit.
func nativeMergeApprove(t *testing.T, c nativeST12Config, p factory.Publication, event, body string) {
	t.Helper()
	if p.PRCreate.Work == nil {
		t.Fatalf("published publication lacks its recorded creation intent: stage=%s reason=%s publish=%+v prcreate=%+v", p.Stage, p.Reason, p.Publish, p.PRCreate)
	}
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-proof",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p.PRNumber, PRID: p.PRID,
		IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.PRCreate.HeadOID, BaseOID: p.PRCreate.BaseOID,
		Event: event, Body: body,
	})
}

// nativeMergeAssess runs the production ST11 read path and records the
// verdict: the exact record MergePass consumes.
func nativeMergeAssess(t *testing.T, c nativeST12Config, fx *nativeFixture, p factory.Publication) factory.CheckAssessment {
	t.Helper()
	assessor := forgejo.NewCheckAssessor(forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""), forgejo.New(c.FountainURL), c.TokenFile)
	target := factory.CheckTarget{
		Repository: c.Repository, PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.PRCreate.HeadOID, BaseOID: p.PRCreate.BaseOID,
	}
	observed := nativeObserveChecks(t, assessor, target, c.ActorID)
	policy, err := fx.db.RepositoryPolicy(context.Background(), c.Repository)
	nativeMust(t, err)
	adopted := factory.AdoptedChecks{Checks: append([]string(nil), policy.Checks...), PolicyRevision: policy.Revision, Digest: factory.ChecksDigest(policy.Checks)}
	assessment, err := factory.VerifyChecks(target, adopted, policy, observed, time.Now().Unix())
	nativeMust(t, err)
	stored, err := fx.db.RecordCheckAssessment(context.Background(), assessment)
	nativeMust(t, err)
	return stored
}

// nativeMergeOpenRow records an open merge row exactly like mergeOne's
// construction, for reconcile-gating tests: MergePass only opens rows
// behind a current pass, so tests that regress the evidence afterwards
// seed their row directly instead of driving the opening policy.
func nativeMergeOpenRow(t *testing.T, fx *nativeFixture, p factory.Publication) {
	t.Helper()
	ctx := context.Background()
	policy, err := fx.db.RepositoryPolicy(ctx, p.Repository)
	nativeMust(t, err)
	m := factory.Merge{
		Operation: factory.MergeOperation{Kind: factory.OpMerge},
		Authority: p.Authority, ID: factory.NewID(), PublicationID: p.ID,
		AssignmentID: p.AssignmentID, ProjectID: p.ProjectID, Role: p.Role,
		Acceptance: p.Acceptance, HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID,
		Stage: factory.MergeOpen, Repository: p.Repository, Issue: p.Issue,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID,
		PRAuthorID: p.PRCreate.Work.ActorID, ReviewerID: policy.Review.ActorID,
		CreatedUnix: time.Now().Unix(),
	}
	nativeMust(t, fx.db.RecordMerge(ctx, m))
}

// nativeMergeDrive runs MergePass until the merge leaves open/fenced or
// the deadline passes. Waits retry; transient native_unavailable errors
// retry too, since the fenced host refuses with 503 while a native
// mutation reservation is held. Any other error fails fast.
func nativeMergeDrive(t *testing.T, c nativeST12Config, fx *nativeFixture, publicationID string) (factory.Merge, control.MergeReport) {
	t.Helper()
	nativeMergeIdle(t, forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""))
	ctx := context.Background()
	var last control.MergeReport
	for i := 0; i < 90; i++ {
		last = fx.coord.MergePass(ctx)
		for _, entry := range last.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", last)
			}
		}
		m, err := fx.db.MergeByPublication(ctx, publicationID)
		if err == nil && m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
			return m, last
		}
		time.Sleep(200 * time.Millisecond)
	}
	m, _ := fx.db.MergeByPublication(ctx, publicationID)
	t.Fatalf("merge did not settle: %+v last=%+v", m, last)
	return factory.Merge{}, last
}

// nativeMergePublishDrive runs PublishPass until one assignment's
// publication leaves open/fenced, tolerating only transient
// native_unavailable errors like nativeMergeDrive. Any terminal stage
// returns; the caller decides whether it is the scenario outcome.
func nativeMergePublishDrive(t *testing.T, c nativeST12Config, fx *nativeFixture, a factory.Assignment) factory.Publication {
	t.Helper()
	nativeMergeIdle(t, forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""))
	ctx := context.Background()
	var p factory.Publication
	for i := 0; i < 120; i++ {
		report := fx.coord.PublishPass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("publication errors: %+v", report)
			}
		}
		var err error
		p, err = fx.db.PublicationByAssignment(ctx, a.ID)
		nativeMust(t, err)
		if p.Stage != factory.PublicationOpen && p.Stage != factory.PublicationFenced {
			return p
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("publication did not settle: stage=%s", p.Stage)
	return factory.Publication{}
}

// nativeMergeSeedPublication publishes one candidate, reseeding with a
// fresh assignment when host churn (certain stale refusal, no effect)
// burns the seeding submit. Anything else fails fast: only churn
// retries, never a scenario outcome.
func nativeMergeSeedPublication(t *testing.T, c nativeST12Config, fx *nativeFixture, n nativeCandidate) (factory.Publication, factory.Assignment) {
	t.Helper()
	for attempt := 0; attempt < 5; attempt++ {
		a := nativeMergeAssignment(t, c, fx, n)
		p := nativeMergePublishDrive(t, c, fx, a)
		if p.Stage == factory.PublicationPublished {
			return p, a
		}
		stale := p.Stage == factory.PublicationFailed &&
			(p.Publish.Reason == "stale_native_revision" || p.PRCreate.Reason == "stale_native_revision")
		if !stale {
			t.Fatalf("publication settled unpublished: stage=%s reason=%s publish=%+v prcreate=%+v", p.Stage, p.Reason, p.Publish, p.PRCreate)
		}
		t.Logf("publication reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("publication never published")
	return factory.Publication{}, factory.Assignment{}
}

// nativeMergeInsertEdge records one blocked-by-blocker edge in the staged
// Forgejo database; Soda observes it back through production snapshots,
// so the proved observation path stays native.
func nativeMergeInsertEdge(t *testing.T, c nativeST12Config, blockedID, blockerID int64) {
	t.Helper()
	nativeMust(t, seedStagedDependencyEdge(c.FountainDB, c.CreatorID, blockedID, blockerID))
}

// nativeMergeGit runs one git command with retries: the shared fixture
// serves rapid back-to-back tests, and a single busy response must not
// fail the proof. Failures name the command.
func nativeMergeGit(t *testing.T, c nativeST12Config, dir string, args ...string) string {
	t.Helper()
	var err error
	var out string
	for i := 0; i < 5; i++ {
		if out, err = nativeGit(t, c.nativeST09Config, dir, nil, args...); err == nil {
			return out
		}
		time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
	}
	t.Fatalf("native fixture git %q failed: %v", args, err)
	return ""
}

// nativeMergeAdvanceBranch appends one commit to an existing native
// branch and returns its tip.
func nativeMergeAdvanceBranch(t *testing.T, c nativeST12Config, branch string) string {
	t.Helper()
	var dir string
	var err error
	for i := 0; i < 5; i++ {
		dir = filepath.Join(t.TempDir(), "advance")
		if _, err = nativeGit(t, c.nativeST09Config, "", nil, "clone", nativeRepoURL(c.nativeST09Config), dir); err == nil {
			break
		}
		time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("native fixture git clone failed: %v", err)
	}
	nativeMergeGit(t, c, dir, "checkout", branch)
	nativeMust(t, os.WriteFile(filepath.Join(dir, "advance.txt"), []byte("ST12 advance "+factory.NewID()+"\n"), 0o600))
	nativeMergeGit(t, c, dir, "add", "advance.txt")
	nativeMergeGit(t, c, dir, "-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit", "-m", "ST12 advance")
	nativeMergeGit(t, c, dir, "push", "origin", branch)
	return nativeMergeGit(t, c, dir, "rev-parse", "HEAD")
}
