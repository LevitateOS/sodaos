package control_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

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
