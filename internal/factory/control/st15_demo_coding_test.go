package control_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/project"
)

// providerGate fails closed without a real provider credential: every
// input leg above runs credential-free, and no synthetic run ever
// substitutes for the composed execution below.
func (fx *st15Fixture) providerGate() error {
	if fx.synthetic {
		st15Receipt(fx.t, "provider-gate", map[string]any{"credential": "absent", "legs": fx.checks})
		return errors.New("ST15 requires SODA_ST15_PROVIDER_CREDENTIAL: a 0600 file of Muse CLI auth.json bytes; refusing synthetic execution")
	}
	return nil
}

// runForIssue returns one recorded run ID for an issue index, or "".
func (fx *st15Fixture) runForIssue(index int64) string {
	fx.t.Helper()
	runs, err := fx.db.FactoryRuns(fx.ctx, 50)
	if err != nil {
		fx.t.Fatal(err)
	}
	for _, run := range runs {
		view, err := fx.db.FactoryRunView(fx.ctx, run.ID)
		if err != nil {
			continue
		}
		if view.Issue == index {
			return run.ID
		}
	}
	return ""
}

// pollRunForIssue waits for a recorded run on one issue and returns it.
func (fx *st15Fixture) pollRunForIssue(index int64, timeout time.Duration) string {
	fx.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if id := fx.runForIssue(index); id != "" {
			return id
		}
		time.Sleep(2 * time.Second)
	}
	fx.t.Fatalf("ST15 no run recorded for issue #%d", index)
	return ""
}

// runBrowser executes the Playwright check in one mode. Secrets cross by
// file path only; the receipt carries IDs and counts.
func (fx *st15Fixture) runBrowser(mode, runID, role string, issue int64) error {
	fx.t.Helper()
	out := filepath.Join(st15ReceiptDir(fx.t), "browser-"+mode+".json")
	cmd := exec.CommandContext(fx.ctx, "/home/vince/.bun/bin/bun", "/home/vince/Projects/sodaos/.artifacts/st15-demo/spaces-check.ts")
	cmd.Env = append(os.Environ(),
		"ST15_FOUNTAIN="+fx.cfg.BrowserURL,
		"ST15_USER=soda-maintainer",
		"ST15_PASS_FILE="+fx.cfg.MaintainerPass,
		"ST15_MODE="+mode,
		"ST15_OUT="+out,
		"ST15_RUN_ID="+runID,
		"ST15_ROLE="+role,
		"ST15_ISSUE="+strconv.FormatInt(issue, 10),
		"ST15_REPO_ID="+strconv.FormatInt(fx.cfg.Repository, 10),
	)
	cmd.Dir = "/home/vince/Projects/sodaos"
	combined, err := cmd.CombinedOutput()
	fx.t.Logf("ST15 browser %s: %s", mode, strings.TrimSpace(string(combined)))
	if err != nil {
		return fmt.Errorf("ST15 browser %s: %w", mode, err)
	}
	return nil
}

// dispatchA launches the coding run through production intake
// auto-dispatch while the live browser watches: the re-observation records
// A queued now that authority is effective, the intake pass launches
// exactly one run, the watch attaches, observes output bytes, then
// detaches, and the run continues to completion without it.
func (fx *st15Fixture) dispatchA() error {
	// Dispatch scans recorded queued controls: re-observe through the
	// real intake handler so A flips from not_authorized to queued and
	// the intake pass auto-launches. Closed P stays skipped by plan
	// (WaitIssueClosed); B/C stay unqueued/withdrawn on record.
	fx.commentHint(fx.issueAIndex)
	ctrl := fx.observeControl(fx.issueAIndex)
	if ctrl.Readiness != factory.ReadinessQueued && ctrl.Readiness != factory.ReadinessActive {
		return fmt.Errorf("A not queued after authority activation: %+v", ctrl)
	}
	runID := fx.pollRunForIssue(fx.issueAIndex, 10*time.Minute)
	fx.t.Logf("ST15 auto-dispatch launched run %s for A", runID)
	if err := fx.runBrowser("watch", runID, project.RoleCoder, fx.issueAIndex); err != nil {
		fx.t.Fatal(err)
	}
	fx.t.Log("ST15 browser detached; the coding run continues without it")
	// An explicit pass must not duplicate the launch.
	report := fx.coord.Dispatch(fx.ctx)
	if len(report.Launched) != 0 {
		return fmt.Errorf("explicit dispatch duplicated the launch: %+v", report)
	}
	if len(report.Errors) != 0 {
		return fmt.Errorf("explicit dispatch errors: %+v", report)
	}
	assignment, output := fx.settleDispatched(runID)
	if assignment.Stage != factory.AssignmentFinished || assignment.Outcome != factory.Succeeded ||
		assignment.Result == nil || !assignment.Result.Reported || assignment.Result.Status != "completed" ||
		!factory.ValidCommit(assignment.Result.Candidate) {
		st15Receipt(fx.t, "coder-output", map[string]any{"output_tail": tailLines(output, 40)})
		return fmt.Errorf("coding run reported no completed candidate: %+v", assignment.Result)
	}
	fx.assignA, fx.runA = assignment.ID, runID
	st15Receipt(fx.t, "dispatch-A", map[string]any{
		"assignment": assignment.ID, "run": runID, "candidate": assignment.Result.Candidate,
	})
	st15Receipt(fx.t, "coder-output", map[string]any{"output_tail": tailLines(output, 40)})
	return nil
}

// stallForBrowserDiag is a diagnostic-only hook (default off): when
// ST15_DIAG_STALL_DIR names a directory, it records one synthetic run
// row plus view (never a provider run, never proof), writes the fixture
// connection info for a manual spaces-check.ts run, and sleeps so the
// fixture stays alive for browser/API debugging.
func (fx *st15Fixture) stallForBrowserDiag() {
	fx.t.Helper()
	dir := os.Getenv("ST15_DIAG_STALL_DIR")
	if dir == "" {
		return
	}
	now := time.Now().Truncate(time.Second)
	runID := factory.NewID()
	run := factory.Run{
		ID: runID, ProjectID: fx.projectID, Role: project.RoleCoder, InputSHA: strings.Repeat("d", 40),
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: fx.image, Harness: project.FactoryHarnessMuse, Model: "muse-spark-1.3",
	}
	nativeMust(fx.t, fx.db.RecordFactoryRun(fx.ctx, run))
	_, _, err := fx.db.RecordFactoryRunView(fx.ctx, factory.RunView{
		RunID: runID, Repository: fx.cfg.Repository, Issue: 2, Attempt: factory.NewID(),
	})
	nativeMust(fx.t, err)
	info := map[string]string{
		"fountain_url": fx.cfg.BrowserURL, "pass_file": fx.cfg.MaintainerPass,
		"repository": strconv.FormatInt(fx.cfg.Repository, 10), "run_id": runID,
		"role": project.RoleCoder, "issue": "2",
	}
	raw, err := json.MarshalIndent(info, "", "  ")
	nativeMust(fx.t, err)
	nativeMust(fx.t, os.MkdirAll(dir, 0o700))
	nativeMust(fx.t, os.WriteFile(filepath.Join(dir, "stall.json"), raw, 0o600))
	minutes := 15
	if raw := os.Getenv("ST15_DIAG_STALL_MINUTES"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 30 {
			minutes = parsed
		}
	}
	fx.t.Logf("ST15 DIAG stall: %s for %d minutes (synthetic run %s, never proof)", dir, minutes, runID)
	deadline := time.Now().Add(time.Duration(minutes) * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "release")); err == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func tailLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// publishA drives the production publication pass to the linked PR.
func (fx *st15Fixture) publishA() error {
	nativeMergeIdle(fx.t, forgejo.NewServiceBackground(fx.cfg.Socket, uint32(os.Getuid()), ""))
	last := ""
	for i := 0; i < 120; i++ {
		report := fx.coord.PublishPass(fx.ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				return fmt.Errorf("publication errors: %+v", report)
			}
		}
		p, err := fx.db.PublicationByAssignment(fx.ctx, fx.assignA)
		if err != nil {
			return err
		}
		state := fmt.Sprintf("stage=%s publish=%d/%s/%s prcreate=%d/%s/%s waits=%v errors=%v",
			p.Stage, p.Publish.Attempts, p.Publish.Effect, p.Publish.Completion,
			p.PRCreate.Attempts, p.PRCreate.Effect, p.PRCreate.Completion,
			report.Waits, report.Errors)
		if i%20 == 0 || state != last {
			fx.t.Logf("ST15 publish-A poll %d: %s", i, state)
			last = state
		}
		if p.Stage == factory.PublicationPublished {
			fx.pubA, fx.head1, fx.prNumber = p.ID, p.Candidate, p.PRNumber
			st15Receipt(fx.t, "publish-A", map[string]any{
				"publication": p.ID, "head": p.Candidate, "pr": p.PRNumber, "revision": p.Revision,
			})
			return nil
		}
		if p.Stage != factory.PublicationOpen && p.Stage != factory.PublicationFenced {
			return fmt.Errorf("publication terminal: %+v", p)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("publication did not settle")
}
func mustViewAttempt(fx *st15Fixture, runID string) string {
	fx.t.Helper()
	view, err := fx.db.FactoryRunView(fx.ctx, runID)
	if err != nil {
		fx.t.Fatal(err)
	}
	return view.Attempt
}
