package control_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// ciPass seeds the passing verdict on the corrected head and records it.
func (fx *st15Fixture) ciPass() error {
	fx.seedCIStatus(fx.head2, "st15-build", "success", "widget compiles")
	fx.seedCIStatus(fx.head2, "st15-test", "success", "required regression test present")
	report := fx.coord.CheckPass(fx.ctx)
	link, ok := st15CheckLink(report, fx.assignA)
	if !ok || len(report.Errors) != 0 || link.Verdict != factory.CheckPass {
		return fmt.Errorf("CI pass unrecorded: %+v", report)
	}
	fx.ciPassRev = link.Revision
	st15Receipt(fx.t, "ci-pass", link)
	return nil
}

// mergeAndDependants drives the conditional merge while the dependant's
// automatic pickup runs under live browser control: the browser stops
// the dependant run, then latches and reopens the dispatch gate.
func (fx *st15Fixture) mergeAndDependants() error {
	nativeMergeIdle(fx.t, forgejo.NewServiceBackground(fx.cfg.Socket, uint32(os.Getuid()), ""))
	type mergeOutcome struct {
		merge  factory.Merge
		report control.MergeReport
	}
	done := make(chan mergeOutcome, 1)
	driven := make(chan error, 1)
	go func() {
		ctx := context.Background()
		var last control.MergeReport
		for i := 0; i < 240; i++ {
			last = fx.coord.MergePass(ctx)
			for _, entry := range last.Errors {
				if entry.Reason != "native_unavailable" {
					driven <- fmt.Errorf("merge errors: %+v", last)
					return
				}
			}
			m, err := fx.db.MergeByPublication(ctx, fx.pubA)
			if err == nil && m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
				done <- mergeOutcome{m, last}
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		driven <- errors.New("merge did not settle")
	}()
	// The dependant starts automatically inside the completing pass.
	// Poll for its run, but fail fast when the merge drive already
	// reported terminally: a failed merge never dispatches B, so
	// waiting out the run timeout would mask the real failure. No
	// fallback intake is synthesized: the dependant must arrive via the
	// completion cascade alone, or the journey fails truthfully.
	var runB string
	var pending *mergeOutcome
	trigger := "completion-cascade"
	deadlineB := time.Now().Add(45 * time.Minute)
pollB:
	for time.Now().Before(deadlineB) {
		select {
		case outcome := <-done:
			if outcome.merge.Stage != factory.MergeMerged {
				return fmt.Errorf("merge terminal without merging: %+v", outcome.merge)
			}
			pending = &outcome
		case err := <-driven:
			return err
		default:
		}
		if id := fx.runForIssue(fx.issueBIndex); id != "" {
			runB = id
			break pollB
		}
		time.Sleep(2 * time.Second)
	}
	if runB == "" {
		return errors.New("ST15 no run recorded for dependant issue")
	}
	fx.assignB = mustViewAttempt(fx, runB)
	// The control browser booted during review-2 and has been polling
	// for B's run since; it stops the run within seconds of record,
	// long before any CLI can finish. Await its verdict here.
	if fx.controlBrowser == nil {
		return errors.New("ST15 control browser never started")
	}
	if err := <-fx.controlBrowser; err != nil {
		return err
	}
	if pending == nil {
		select {
		case o := <-done:
			pending = &o
		case err := <-driven:
			return err
		case <-time.After(50 * time.Minute):
			return errors.New("merge drive timed out")
		}
	}
	if pending.merge.Stage != factory.MergeMerged {
		return fmt.Errorf("merge terminal without merging: %+v", pending.merge)
	}
	st15Receipt(fx.t, "merge-A", map[string]any{
		"merge": pending.merge.ID, "commit": pending.merge.MergedCommit,
		"pr": pending.merge.PRNumber, "report": pending.report,
		"dependant_trigger": trigger,
	})
	// The browser stop settled B through the dashboard coordinator: the
	// run is reconciled-cancelled and its assignment finished without a
	// result, so it never publishes.
	deadline := time.Now().Add(5 * time.Minute)
	for {
		assignment, err := fx.db.Assignment(fx.ctx, fx.assignB)
		if err != nil {
			return err
		}
		if assignment.Stage == factory.AssignmentFinished {
			if assignment.Outcome != factory.Cancelled {
				return fmt.Errorf("stopped dependant finished %s, want cancelled", assignment.Outcome)
			}
			st15Receipt(fx.t, "dependant-B", map[string]any{
				"assignment": assignment.ID, "run": runB, "outcome": assignment.Outcome,
			})
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("dependant assignment unsettled: %+v", assignment)
		}
		time.Sleep(2 * time.Second)
	}
}

// proveRetention verifies the human side survived: dirty files, service
// data and process, member execution, the same container, bounded usage,
// and the decoy's permanent silence.
func (fx *st15Fixture) proveRetention() error {
	ctx := fx.ctx
	var snapshot map[string]string
	raw, err := os.ReadFile(filepath.Join(st15ReceiptDir(fx.t), "human-snapshot.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return err
	}
	for _, path := range []string{"/home/soda-tester/work/dirty.txt", "/home/soda-tester/service-data/rows.txt", "/home/soda-tester/.ssh/authorized_keys"} {
		out, err := st15Pexec(ctx, fx.container, "", nil, "/usr/bin/sha256sum", path)
		if err != nil {
			return err
		}
		if got := strings.Fields(strings.TrimSpace(string(out)))[0]; got != snapshot[path] {
			return fmt.Errorf("retained file %s changed", path)
		}
	}
	if _, err := st15Pexec(ctx, fx.container, "soda-tester", nil, "/usr/bin/true"); err != nil {
		return fmt.Errorf("member execution lost: %v", err)
	}
	if _, err := st15Pexec(ctx, fx.container, "", nil, "/usr/bin/sh", "-c", "kill -0 "+snapshot["service_pid"]); err != nil {
		return fmt.Errorf("retained service process lost: %v", err)
	}
	id, err := st15Podman(ctx, nil, "inspect", "--format", "{{.ID}}", fx.container)
	if err != nil {
		return err
	}
	fx.t.Logf("ST15 container identity stable: %.12s", strings.TrimSpace(string(id)))
	runs, err := fx.db.FactoryRuns(ctx, 50)
	if err != nil {
		return err
	}
	total := 0
	rows := 0
	for _, run := range runs {
		view, err := fx.db.FactoryRunView(ctx, run.ID)
		if err != nil {
			continue
		}
		if view.Issue == fx.issueCIndex {
			return fmt.Errorf("withdrawn decoy ran: %s", run.ID)
		}
		usage, err := fx.db.RunUsage(ctx, run.ID)
		if err != nil {
			return fmt.Errorf("run %s lacks its usage row: %w", run.ID, err)
		}
		total += usage.Minutes
		rows++
	}
	if rows == 0 || total > 60 {
		return fmt.Errorf("usage out of bounds: %d minutes over %d runs", total, rows)
	}
	st15Receipt(fx.t, "retention", map[string]any{
		"usage_minutes": total, "usage_rows": rows, "runs": len(runs),
		"container": strings.TrimSpace(string(id)),
	})
	return nil
}
