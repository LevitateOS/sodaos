package control_test

// Native check-assessment proof (ST11, S-checks). Fixture candidates, PRs
// and commit statuses are prepared through ordinary native Git and REST
// writes; assessment reads only bracketed native snapshots plus disclosed
// accepted-policy records, and no agent narrative enters the verdict. The
// small native workflow is committed last and verified through ordinary
// native run/status reads, because a head carrying an Actions-posted
// status currently refuses checks-family snapshots with 503 (a Fountain
// defect routed to the coordinator, not a Soda verdict). The cancelled,
// skipped, arbitrary-state and genuine-workflow-pending snapshot cases
// are skipped until that conversion fix lands; the skips name the defect
// and un-skip mechanically. Absence of SODA_ST11_NATIVE skips native
// work and is never passing evidence.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func loadNativeST11(t *testing.T) nativeST09Config {
	t.Helper()
	raw := os.Getenv("SODA_ST11_NATIVE")
	if raw == "" {
		t.Skip("native proof NOT RUN: SODA_ST11_NATIVE is not configured")
	}
	var cfg nativeST09Config
	nativeMust(t, json.Unmarshal([]byte(raw), &cfg))
	if cfg.FountainURL == "" || cfg.Socket == "" || cfg.TokenFile == "" || cfg.Repository <= 0 || cfg.ActorID <= 0 || cfg.TokenID <= 0 {
		t.Fatal("native fixture requires explicit repository and assessor inputs")
	}
	if !factory.ValidTargetBranch(cfg.BaseBranch) {
		t.Fatal("native fixture requires exact base_branch")
	}
	return cfg
}

func nativeCheckReceipt(t *testing.T, label string, value any) {
	t.Helper()
	root := os.Getenv("ST11_RECEIPT_DIR")
	if root == "" {
		t.Fatal("ST11_RECEIPT_DIR is required to retain native proof")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("native receipt directory must be absolute")
	}
	nativeMust(t, os.MkdirAll(root, 0o700))
	raw, err := json.MarshalIndent(value, "", "  ")
	nativeMust(t, err)
	nativeMust(t, os.WriteFile(filepath.Join(root, label+".json"), append(raw, '\n'), 0o600))
}

func nativeCheckAssessor(t *testing.T, c nativeST09Config) *forgejo.CheckAssessor {
	t.Helper()
	return forgejo.NewCheckAssessor(forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""), forgejo.New(c.FountainURL), c.TokenFile)
}

func nativeCheckDB(t *testing.T) *store.Store {
	t.Helper()
	db, _ := postgresFixture(t, nil)
	return db
}

func nativeCheckPolicy(t *testing.T, db *store.Store, c nativeST09Config, checks []string) factory.RepositoryPolicy {
	t.Helper()
	ctx := context.Background()
	actor := func(kind string) factory.ActorBindingRef {
		return factory.ActorBindingRef{TokenID: c.TokenID, ActorID: c.ActorID, Kind: kind}
	}
	policy := factory.RepositoryPolicy{
		Repository: c.Repository, GrantedBy: c.ActorID, Enabled: true, TargetBranch: c.BaseBranch,
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: "1.2.3", Model: "fixture"},
			project.RoleReviewer: {Harness: "1.2.3", Model: "fixture"},
		},
		Checks: checks, MergeMethod: factory.MergeFastForward,
		Publish: actor(factory.OpRefPublish), Create: actor(factory.OpPRCreate),
		Review: actor(factory.OpReviewSubmit), Merge: actor(factory.OpMerge),
		MaxConcurrent: 2,
	}
	nativeMust(t, db.SaveRepositoryPolicy(ctx, policy))
	stored, err := db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	return stored
}

func nativeCheckAdopted(policy factory.RepositoryPolicy) factory.AdoptedChecks {
	return factory.AdoptedChecks{Checks: append([]string(nil), policy.Checks...), PolicyRevision: policy.Revision, Digest: factory.ChecksDigest(policy.Checks)}
}

type nativeCheckPR struct {
	branch   string
	head     string
	base     string
	prID     int64
	prNumber int64
	issueID  int64
}

func nativeCheckPublish(t *testing.T, c nativeST09Config, n nativeCandidate, label string) nativeCheckPR {
	t.Helper()
	branch := "st11-" + label + "-" + factory.NewID()[:12]
	nativeGitOK(t, c, n.dir, "push", nativeRepoURL(c), n.head+":refs/heads/"+branch)
	var pr struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/pulls",
		map[string]any{"head": branch, "base": strings.TrimPrefix(c.BaseBranch, "refs/heads/"), "title": "ST11 " + label, "body": "Exact check-assessment fixture", "allow_maintainer_edit": false}, &pr)
	if pr.ID <= 0 || pr.Number <= 0 {
		t.Fatal("native PR identity unconfirmed")
	}
	var issue struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(pr.Number, 10), nil, &issue)
	if issue.ID <= 0 || issue.Number != pr.Number {
		t.Fatal("native PR issue identity unconfirmed")
	}
	return nativeCheckPR{branch: branch, head: n.head, base: n.base, prID: pr.ID, prNumber: pr.Number, issueID: issue.ID}
}

func (p nativeCheckPR) target(c nativeST09Config) factory.CheckTarget {
	return factory.CheckTarget{
		Repository: c.Repository, PRNumber: p.prNumber, PRID: p.prID, IssueID: p.issueID,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
	}
}

func nativePostStatusOnce(t *testing.T, c nativeST09Config, sha string, body []byte) (int, int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(c.FountainURL, "/")+"/api/v1/repos/"+c.Owner+"/"+c.Repo+"/statuses/"+sha, bytes.NewReader(body))
	nativeMust(t, err)
	req.Header.Set("Authorization", "token "+nativeSecret(t, c.TokenFile))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal("native API transport unavailable")
	}
	defer response.Body.Close()
	var created struct {
		ID int64 `json:"id"`
	}
	if response.StatusCode == http.StatusCreated || response.StatusCode == http.StatusOK {
		nativeMust(t, json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&created))
	}
	return response.StatusCode, created.ID
}

func nativePostStatus(t *testing.T, c nativeST09Config, sha, checkContext, state string) {
	t.Helper()
	if !factory.ValidCommit(sha) || checkContext == "" {
		t.Fatal("status needs its exact head and context")
	}
	body, err := json.Marshal(map[string]any{
		"state": state, "context": checkContext,
		"description": "ST11 fixture evidence", "target_url": "https://st11.invalid/fixture",
	})
	nativeMust(t, err)
	// A busy host refuses the write with 503 while Actions run work
	// settles; only that wait retries, bounded, never another refusal.
	deadline := time.Now().Add(60 * time.Second)
	for {
		status, id := nativePostStatusOnce(t, c, sha, body)
		if status == http.StatusCreated || status == http.StatusOK {
			if id <= 0 {
				t.Fatal("native status identity unconfirmed")
			}
			return
		}
		if status != http.StatusServiceUnavailable || !time.Now().Before(deadline) {
			t.Fatalf("native status post refused: status %d", status)
		}
		time.Sleep(2 * time.Second)
	}
}

type nativeListedStatus struct {
	Context string `json:"context"`
	Status  string `json:"status"`
}

func nativeListStatuses(t *testing.T, c nativeST09Config, sha string) []nativeListedStatus {
	t.Helper()
	var listed []nativeListedStatus
	nativeAPI(t, c, c.TokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/commits/"+sha+"/statuses", nil, &listed)
	return listed
}

func nativeWaitStatusContext(t *testing.T, c nativeST09Config, sha, prefix string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		for _, entry := range nativeListStatuses(t, c, sha) {
			if strings.HasPrefix(entry.Context, prefix) {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("native context %q never reported on %s", prefix, sha[:12])
}

func nativeObserveChecks(t *testing.T, assessor *forgejo.CheckAssessor, target factory.CheckTarget, actorID int64) factory.ObservedChecks {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 60; i++ {
		observed, err := assessor.ObserveChecks(ctx, target, actorID)
		if err == nil {
			return observed
		}
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || (wait.Reason != "native_busy" && wait.Reason != "revision_moved") {
			t.Fatal(err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("native check observation did not settle")
	return factory.ObservedChecks{}
}

func nativeVerifyChecks(t *testing.T, db *store.Store, label string, target factory.CheckTarget, adopted factory.AdoptedChecks, current factory.RepositoryPolicy, observed factory.ObservedChecks) factory.CheckAssessment {
	t.Helper()
	assessment, err := factory.VerifyChecks(target, adopted, current, observed, time.Now().Unix())
	nativeMust(t, err)
	nativeMust(t, assessment.Validate())
	stored, err := db.RecordCheckAssessment(context.Background(), assessment)
	nativeMust(t, err)
	read, err := db.CheckAssessment(context.Background(), target.Repository, target.PRNumber)
	nativeMust(t, err)
	if read.Revision != stored.Revision || read.Verdict != assessment.Verdict || read.Reason != assessment.Reason ||
		read.HeadOID != target.HeadOID || read.BaseOID != target.BaseOID || read.ChecksDigest != adopted.Digest {
		t.Fatal("stored check assessment differs from its verdict")
	}
	nativeCheckReceipt(t, label, stored)
	return stored
}

// nativeEnsureWorkflow commits the small native workflow once: one
// push-triggered verify job. Native Actions runs it on separately
// managed capacity; with no runner the run waits and its pending status
// stays genuine evidence.
func nativeEnsureWorkflow(t *testing.T, c nativeST09Config) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "seed")
	nativeGitOK(t, c, "", "clone", "--branch", strings.TrimPrefix(c.BaseBranch, "refs/heads/"), nativeRepoURL(c), dir)
	workflow := filepath.Join(dir, ".gitea", "workflows", "st11-checks.yml")
	if _, err := os.Stat(workflow); err == nil {
		return nativeGitOK(t, c, dir, "rev-parse", "HEAD")
	}
	nativeMust(t, os.MkdirAll(filepath.Dir(workflow), 0o700))
	nativeMust(t, os.WriteFile(workflow, []byte("name: st11-checks\non: [push]\njobs:\n  verify:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo st11-checks\n"), 0o600))
	nativeGitOK(t, c, dir, "add", ".gitea/workflows/st11-checks.yml")
	nativeGitOK(t, c, dir, "-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit", "-m", "ST11 small native workflow")
	head := nativeGitOK(t, c, dir, "rev-parse", "HEAD")
	nativeGitOK(t, c, dir, "push", nativeRepoURL(c), "HEAD:"+c.BaseBranch)
	if nativeTip(t, c, c.BaseBranch) != head {
		t.Fatal("workflow seed did not land on the base branch")
	}
	return head
}

func TestNativeCheckPass(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-build", "st11-review-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "pass")
	nativePostStatus(t, c, p.head, "st11-build", "success")
	nativePostStatus(t, c, p.head, "st11-review-gate", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "pass", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPass || assessment.Reason != factory.CheckReasonPass {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Logf("PASS exact head/base satisfies every configured required check (native rev %d, policy rev %d)", assessment.NativeRev, assessment.PolicyRevision)
}

func TestNativeCheckPending(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "pending")
	nativePostStatus(t, c, p.head, "st11-gate", "pending")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "pending", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPending || assessment.Reason != factory.CheckReasonPending {
		t.Fatalf("verdict: %+v", assessment)
	}
	if len(assessment.Results) != 1 || assessment.Results[0].State != "pending" {
		t.Fatalf("pending state not observed: %+v", assessment.Results)
	}
	t.Log("PASS pending check waits without failing")
}

func TestNativeCheckMissing(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-present", "st11-absent"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "missing")
	nativePostStatus(t, c, p.head, "st11-present", "success")
	nativePostStatus(t, c, p.head, "st11-unrequired", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "missing", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPending || assessment.Reason != factory.CheckReasonPending {
		t.Fatalf("verdict: %+v", assessment)
	}
	if assessment.Results[1].State != "" || assessment.Results[1].Passed {
		t.Fatalf("missing check not reported: %+v", assessment.Results)
	}
	t.Log("PASS missing required check waits while unrequired success is ignored")
}

func TestNativeCheckFailed(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "failed")
	nativePostStatus(t, c, p.head, "st11-gate", "failure")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "failed", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonFailed {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS failed check fails the assessment")
}

func TestNativeCheckCancelled(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "cancelled")
	nativePostStatus(t, c, p.head, "st11-gate", "cancelled")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "cancelled", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonCancelled {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS cancelled check fails the assessment")
}

func TestNativeCheckSkipped(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "skipped")
	nativePostStatus(t, c, p.head, "st11-gate", "skipped")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "skipped", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonSkipped {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS skipped check fails the assessment")
}

func TestNativeCheckUnknownState(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "unknown")
	// The native API stores arbitrary state strings; only exact success
	// passes, so an unrecognized state fails closed.
	nativePostStatus(t, c, p.head, "st11-gate", "bogus")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "unknown-state", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonUnknownState {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS unrecognized native state fails closed")
}

func TestNativeCheckLatestWins(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "latest")
	nativePostStatus(t, c, p.head, "st11-gate", "failure")
	nativePostStatus(t, c, p.head, "st11-gate", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "latest-wins", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPass {
		t.Fatalf("verdict: %+v", assessment)
	}
	if len(assessment.Results) != 1 || assessment.Results[0].State != "success" {
		t.Fatalf("latest status did not win: %+v", assessment.Results)
	}
	t.Log("PASS latest status per context governs, matching combined-status semantics")
}

func TestNativeCheckStaleHead(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	n := nativeNewCandidate(t, c)
	p := nativeCheckPublish(t, c, n, "stalehead")
	nativePostStatus(t, c, p.head, "st11-gate", "success")
	moved := nativeAppendCandidate(t, c, n)
	nativeGitOK(t, c, moved.dir, "push", nativeRepoURL(c), moved.head+":refs/heads/"+p.branch)
	if nativeTip(t, c, "refs/heads/"+p.branch) != moved.head {
		t.Fatal("correction push did not move the head")
	}
	assessor := nativeCheckAssessor(t, c)
	stale := nativeVerifyChecks(t, db, "stale-head", p.target(c), nativeCheckAdopted(policy), policy,
		nativeObserveChecks(t, assessor, p.target(c), c.ActorID))
	if stale.Verdict != factory.CheckRefused || stale.Reason != factory.CheckReasonStaleHead {
		t.Fatalf("verdict: %+v", stale)
	}
	fresh := p
	fresh.head = moved.head
	current := nativeVerifyChecks(t, db, "stale-head-fresh", fresh.target(c), nativeCheckAdopted(policy), policy,
		nativeObserveChecks(t, assessor, fresh.target(c), c.ActorID))
	if current.Verdict != factory.CheckPending {
		t.Fatalf("old-head success carried to the new head: %+v", current)
	}
	t.Log("PASS moved head refuses the old verdict and owes fresh evidence")
}

func TestNativeCheckDefinitionsChanged(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-one", "st11-two"})
	adopted := nativeCheckAdopted(policy)
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "definitions")
	nativePostStatus(t, c, p.head, "st11-one", "success")
	nativePostStatus(t, c, p.head, "st11-two", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	before := nativeVerifyChecks(t, db, "definitions-adopted", p.target(c), adopted, policy, observed)
	if before.Verdict != factory.CheckPass {
		t.Fatalf("adopted verdict: %+v", before)
	}
	changed := policy
	changed.Checks = []string{"st11-one", "st11-two", "st11-three"}
	nativeMust(t, db.SaveRepositoryPolicy(context.Background(), changed))
	current, err := db.RepositoryPolicy(context.Background(), c.Repository)
	nativeMust(t, err)
	refused := nativeVerifyChecks(t, db, "definitions-changed", p.target(c), adopted, current, observed)
	if refused.Verdict != factory.CheckRefused || refused.Reason != factory.CheckReasonDefinitionsChanged {
		t.Fatalf("verdict: %+v", refused)
	}
	readopted := nativeVerifyChecks(t, db, "definitions-readopted", p.target(c), nativeCheckAdopted(current), current, observed)
	if readopted.Verdict != factory.CheckPending {
		t.Fatalf("new definition did not require fresh evidence: %+v", readopted)
	}
	t.Log("PASS changed approval definitions refuse until readopted against fresh evidence")
}

func TestNativeCheckPolicyEmpty(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "empty")
	nativePostStatus(t, c, p.head, "st11-gate", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	// A missing policy arrives as the zero record: zero configured checks
	// can never authorize a merge, however green the head looks.
	var missing factory.RepositoryPolicy
	assessment := nativeVerifyChecks(t, db, "policy-empty", p.target(c), nativeCheckAdopted(policy), missing, observed)
	if assessment.Verdict != factory.CheckRefused || assessment.Reason != factory.CheckReasonPolicyEmpty {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS zero configured checks refuse despite green evidence")
}

func TestNativeCheckStaleBase(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "stalebase")
	nativePostStatus(t, c, p.head, "st11-gate", "success")
	landed := nativeNewCandidate(t, c)
	nativeGitOK(t, c, landed.dir, "push", nativeRepoURL(c), landed.head+":"+c.BaseBranch)
	if nativeTip(t, c, c.BaseBranch) != landed.head {
		t.Fatal("base advance did not land")
	}
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "stale-base", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckRefused || assessment.Reason != factory.CheckReasonStaleBase {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS moved base refuses until the candidate is reverified")
}

// TestNativeCheckWorkflowPendingSnapshot assesses the genuine
// Actions-posted pending status through the authoritative snapshot read.
func TestNativeCheckWorkflowPendingSnapshot(t *testing.T) {
	c := loadNativeST11(t)
	nativeEnsureWorkflow(t, c)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-checks / verify (push)"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "wfpending")
	nativeWaitStatusContext(t, c, p.head, "st11-checks / ")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "workflow-pending", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPending || assessment.Reason != factory.CheckReasonPending {
		t.Fatalf("verdict: %+v", assessment)
	}
	if len(assessment.Results) != 1 || assessment.Results[0].State != "pending" {
		t.Fatalf("genuine workflow state not observed: %+v", assessment.Results)
	}
	t.Log("PASS genuine native workflow pending blocks without failing")
}

// TestNativeCheckWorkflowSeed runs last: it commits the small native
// workflow and proves Actions itself produces genuine pending evidence
// on separately managed capacity (no runner: the run waits). The seed
// runs after every snapshot case so the workflow file lands once the
// snapshot cases have published; the workflow run and its pending
// status are verified through ordinary native reads, and the seed
// head's entry shapes are retained in the receipt.
func TestNativeCheckWorkflowSeed(t *testing.T) {
	c := loadNativeST11(t)
	head := nativeEnsureWorkflow(t, c)
	nativeWaitStatusContext(t, c, head, "st11-checks / ")
	var runs struct {
		WorkflowRuns []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"workflow_runs"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/actions/runs?limit=5", nil, &runs)
	if len(runs.WorkflowRuns) == 0 || runs.WorkflowRuns[0].Status != "waiting" {
		t.Fatalf("native workflow run not waiting: %+v", runs.WorkflowRuns)
	}
	// One ordinary status beside the Actions-posted one, so the receipt
	// carries both entry shapes for the snapshot-fix comparison.
	nativePostStatus(t, c, head, "st11-seed-probe", "success")
	var entries []struct {
		Context     string `json:"context"`
		State       string `json:"state"`
		Status      string `json:"status"`
		TargetURL   string `json:"target_url"`
		Description string `json:"description"`
		Creator     struct {
			Login string `json:"login"`
		} `json:"creator"`
		Created string `json:"created_at"`
		Updated string `json:"updated_at"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/commits/"+head+"/statuses", nil, &entries)
	nativeCheckReceipt(t, "workflow-seed", map[string]any{
		"head": head, "base": c.BaseBranch, "workflow": ".gitea/workflows/st11-checks.yml",
		"run_id": runs.WorkflowRuns[0].ID, "run_status": runs.WorkflowRuns[0].Status,
		"entries": entries,
	})
	t.Log("PASS small native workflow runs on upstream capacity and reports genuine pending")
}
