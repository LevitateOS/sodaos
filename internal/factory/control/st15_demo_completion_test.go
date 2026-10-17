package control_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// authorizeDemoMerge answers the demo's open unbound merge rows the way
// the prompt-entry serving journey authorizes its correction: the
// approving round's clean acceptance journals on the publication and
// the union-bound authorization answers the row through the decision
// gate production enforces. Anything not ready yet (no recorded cycle
// for the head, no reviewer result, an admit the gate still refuses)
// stays silent for a later poll and is noted; the note joins the
// terminal error when the merge never settles, so a permanently
// refusing gate fails loudly instead of timing out unexplained.
func (fx *st15Fixture) authorizeDemoMerge(ctx context.Context) {
	merges, err := fx.db.OpenMerges(ctx, 16)
	if err != nil {
		fx.authNote = fmt.Sprintf("open merges unreadable: %v", err)
		return
	}
	for _, m := range merges {
		if m.Stage != factory.MergeOpen || m.Authorization.Bound() {
			continue
		}
		if m.PublicationID != fx.pubA {
			continue
		}
		auth, err := fx.admitDemoAcceptance(ctx, m)
		if err != nil {
			fx.authNote = err.Error()
			continue
		}
		if err := fx.coord.AuthorizeMerge(ctx, m.PublicationID, auth); err != nil {
			fx.authNote = fmt.Sprintf("demo authorize refused: %v", err)
			continue
		}
		fx.authNote = ""
	}
}

// admitDemoAcceptance journals the approving round's clean acceptance
// on the demo publication and binds it to the open row, mirroring the
// serving journey's admitServingDeliveryAcceptance for the clean_review
// delivery. A row whose publication already journals the round's
// acceptance reuses it, so repeated polls never duplicate the journal.
func (fx *st15Fixture) admitDemoAcceptance(ctx context.Context, m factory.Merge) (factory.MergeAuthorization, error) {
	var empty factory.MergeAuthorization
	refuse := func(format string, args ...any) (factory.MergeAuthorization, error) {
		return empty, fmt.Errorf(format, args...)
	}
	p, err := fx.db.PublicationByAssignment(ctx, m.AssignmentID)
	if err != nil {
		return refuse("demo publication unreadable: %v", err)
	}
	round := 0
	var cycle factory.ReviewCycle
	for _, recorded := range p.ReviewCycles {
		if recorded.Head == m.HeadOID && recorded.Round >= round {
			round, cycle = recorded.Round, recorded
		}
	}
	if round == 0 {
		return refuse("no recorded review cycle answers merge head %.8s", m.HeadOID)
	}
	reviewer, err := fx.db.Assignment(ctx, cycle.ReviewerAssignment)
	if err != nil {
		return refuse("approving reviewer assignment unreadable: %v", err)
	}
	if reviewer.Result == nil {
		return refuse("approving reviewer recorded no result")
	}
	assessment, err := fx.db.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		return refuse("recorded check assessment unreadable: %v", err)
	}
	if !factory.ValidDigest(assessment.ChecksDigest) {
		return refuse("recorded checks lack their digest: %+v", assessment)
	}
	for _, recorded := range p.DeliveryAcceptances {
		if recorded.Round == round && recorded.Kind == factory.AcceptanceCleanReview {
			return fx.demoUnionBinding(ctx, m, recorded, assessment.Revision)
		}
	}
	emptyContent, err := factory.FindingContentDigest(nil)
	if err != nil {
		return refuse("%v", err)
	}
	coverage, err := factory.ClearCoverageDigest(cycle.Head, cycle.BaseOID, emptyContent)
	if err != nil {
		return refuse("%v", err)
	}
	headID, _, err := fx.db.PlanHead(ctx, cycle.AttemptRoot)
	if err != nil {
		return refuse("admitted plan head unreadable: %v", err)
	}
	plan, err := fx.db.InvestigationPlan(ctx, cycle.AttemptRoot, headID)
	if err != nil {
		return refuse("admitted plan unreadable: %v", err)
	}
	policy, err := fx.db.RepositoryPolicy(ctx, p.Repository)
	if err != nil {
		return refuse("repository policy unreadable: %v", err)
	}
	journaled, acceptance, err := control.AdmitClearReview(control.ClearAdmissionInput{
		Publication: p, Cycle: cycle,
		Report: factory.ClearReviewReport{
			CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
			Head: cycle.Head, BaseOID: cycle.BaseOID, CoverageDigest: coverage,
		},
		Carried:        nil,
		Result:         reviewer.Result.ToResult(),
		ScopeDigest:    plan.ScopeDigest,
		PolicyRevision: strconv.FormatInt(policy.Revision, 10),
		CheckDigest:    assessment.ChecksDigest,
		ReviewerRunID:  cycle.ReviewerAssignment,
		DiffBase:       p.PRCreate.BaseOID,
	})
	if err != nil {
		return refuse("explicit clear refused: %v", err)
	}
	journaled.Revision = p.Revision + 1
	if err := fx.db.UpdatePublication(ctx, journaled); err != nil {
		return refuse("delivery acceptance unjournaled: %v", err)
	}
	return fx.demoUnionBinding(ctx, m, acceptance, assessment.Revision)
}

// demoUnionBinding carries one journaled clean acceptance to the open
// demo merge row: the generation binds the acceptance identity, kind,
// attempt root, origin receipt and review round over the row's exact
// head and base and the acceptance tuple's real merge-base, with the
// observed native revision and attempt deadline production records.
// The union-aware gate validates the shape on every pass.
func (fx *st15Fixture) demoUnionBinding(ctx context.Context, m factory.Merge, acceptance factory.DeliveryAcceptance, assessmentRev int64) (factory.MergeAuthorization, error) {
	var empty factory.MergeAuthorization
	observed, err := fx.db.CheckAssessment(ctx, m.Repository, m.PRNumber)
	if err != nil {
		return empty, fmt.Errorf("merge observation unreadable: %v", err)
	}
	if observed.NativeRev <= 0 {
		return empty, fmt.Errorf("merge observation carries no native revision")
	}
	notAfter, err := fx.db.CapAttemptDeadline(ctx, m.AssignmentID, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		return empty, fmt.Errorf("merge attempt deadline unavailable: %v", err)
	}
	return factory.MergeAuthorization{
		Generation: factory.MergeGeneration{
			Authority: m.Authority, PublicationID: m.PublicationID,
			NativeOperationID: factory.MergeGenerationOperationID(m.PublicationID, 1),
			Fingerprint:       acceptance.CheckDigest,
			Head:              m.HeadOID, Base: m.BaseOID, DiffBase: acceptance.Tuple.DiffBase,
			ChecksDigest:   acceptance.CheckDigest,
			AcceptanceKind: acceptance.Kind, AcceptanceID: acceptance.ID,
			AttemptRoot: acceptance.AttemptRoot, OriginReceipt: acceptance.OriginReceipt,
			Generation:         1,
			ReviewRound:        acceptance.Round,
			AssessmentRevision: assessmentRev,
			ObservedNativeRev:  observed.NativeRev, NotAfter: notAfter,
		},
		ResolutionsConfirmed: true,
	}, nil
}

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
			// The enforced merge gate authorizes the admitted correction:
			// the approving round's clean acceptance journals on the
			// publication and the union-bound authorization answers the
			// row, so the merge submits instead of waiting judgment.
			fx.authorizeDemoMerge(ctx)
			m, err := fx.db.MergeByPublication(ctx, fx.pubA)
			if err == nil && m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
				done <- mergeOutcome{m, last}
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		if fx.authNote != "" {
			driven <- fmt.Errorf("merge did not settle: %s", fx.authNote)
			return
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
	// sawContinue records the D13 cycle that decided continue while the
	// fence held: the poll below watches the pending cycles from the
	// merge settle until the dependant pickup, so the funded continue
	// decision is observed, never inferred from the pickup alone.
	sawContinue := ""
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
		if cycles, err := fx.db.PendingReassessmentCycles(fx.ctx, fx.cfg.Repository); err == nil {
			for _, record := range cycles {
				if record.Cycle.Decision == factory.ReassessmentOptionContinue && sawContinue == "" {
					sawContinue = record.Cycle.CycleID
				}
			}
		}
		// The pickup waits for the settled merge: breaking only once
		// both are observed proves the fence held the dependant behind
		// the completion cascade instead of dispatching it early.
		if id := fx.runForIssue(fx.issueBIndex); id != "" && pending != nil {
			runB = id
			break pollB
		}
		time.Sleep(2 * time.Second)
	}
	if runB == "" {
		return errors.New("ST15 no run recorded for dependant issue")
	}
	if sawContinue == "" {
		return errors.New("ST15 drove no funded continue reassessment before the dependant pickup")
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
	// The merge rode the union path: the generation binds the journaled
	// clean acceptance behind the row's exact candidate, never a bare
	// funded-decision binding.
	if !pending.merge.Authorization.Generation.UnionBound() ||
		pending.merge.Authorization.Generation.AcceptanceKind != factory.AcceptanceCleanReview {
		return fmt.Errorf("merge merged outside its union-bound acceptance: %+v", pending.merge.Authorization)
	}
	st15Receipt(fx.t, "merge-A", map[string]any{
		"merge": pending.merge.ID, "commit": pending.merge.MergedCommit,
		"pr": pending.merge.PRNumber, "report": pending.report,
		"dependant_trigger": trigger,
	})
	st15Receipt(fx.t, "reassessment-continue", map[string]any{
		"cycle": sawContinue, "decision": factory.ReassessmentOptionContinue,
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
