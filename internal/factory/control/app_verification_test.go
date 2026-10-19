package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

type appVerificationReads struct{ tip string }

func (r *appVerificationReads) ReadDispatchInputs(_ context.Context, _, _ string, _ []string, target string) (DispatchInputs, error) {
	return DispatchInputs{Tip: r.tip, TipRef: target}, nil
}

func TestApplicationVerificationRequiresApprovedNativeCheck(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx := context.Background()
	fx := dispatchSeed(t, db)
	idea := factory.ProductIdea{ID: phasedDecisionID(31), Description: "A starter can save a task", Consequence: "Tasks persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}
	task := factory.IntentTask{Key: phasedDecisionID(32), Objective: "Implement task storage", Outcomes: []string{"Tasks persist"}, IdeaIDs: []string{idea.ID}}
	revision := factory.IntentRevision{ID: phasedDecisionID(21), Intent: phasedDecisionID(20), Owner: 7, Repository: 7, Objective: "Task app", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{task}, EffectiveOutcomes: []string{idea.ID}}
	if err := db.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatal(err)
	}
	binding := factory.IntentTaskBinding{RevisionID: revision.ID, TaskKey: task.Key, IssueID: 3, IssueIndex: 3, AcceptanceID: phasedDecisionID(33), CreatedUnix: 1}
	if err := db.BindIntentTaskAcceptance(ctx, binding); err != nil {
		t.Fatal(err)
	}
	manifest := factory.ApprovedAppCheck{IntentID: revision.Intent, AcceptedRevision: revision.ID, Outcomes: []string{idea.ID}}
	encoded, _ := json.Marshal(manifest)
	script := []byte("#!/bin/sh\n" + factory.ApprovedAppCheckPrefix + string(encoded) + "\nexit 0\n")
	a := phaseTestRepairAssignment(t, project.ExecutionVerify, project.RoleReviewer, phasedDeliverySubject())
	a.ProjectID = fx.proj
	a.SourceCommit = fx.tip
	seedPreparation, err := db.Preparation(ctx, "f222222222222222222222222")
	if err != nil {
		t.Fatal(err)
	}
	p := seedPreparation.Preparation
	p.ID = a.Preparation
	if _, _, err := db.AdmitPreparation(ctx, project.StoredPreparation{Preparation: p}); err != nil {
		t.Fatal(err)
	}
	a.Subject.Delivery = &factory.DeliverySubject{DeliveryID: strings.Repeat("7", 32), IntentID: revision.Intent, AcceptedRevision: revision.ID, ScopeDigest: revision.Digest(), SourceCommit: fx.tip, PreparationDigest: factory.DeliveryPreparationDigest(p), VerificationManifestDigest: manifest.Digest()}
	zero := 0
	state := project.PrepareState{ID: p.ID, Project: p.Project, Role: p.Role, SourceCommit: p.SourceCommit, SetupDigest: p.SetupDigest, Phase: project.PrepareReady, Ready: true, CheckExit: &zero, Output: "task storage passed"}
	report := factory.AppVerificationReport{DeliveryID: a.Subject.Delivery.DeliveryID, InputDigest: strings.Repeat("9", 64), Outcome: factory.AppVerificationVerified,
		CheckReceipts: []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: factory.MCPInputDigest([]byte(state.Output)), LogExcerpt: "task storage passed"}}, Coverage: []factory.ReassessmentCoverage{{IntentID: revision.Intent, OutcomeID: idea.ID, TaskKeys: []string{task.Key}, DeliveredAcceptanceIDs: []string{binding.AcceptanceID}}}}
	a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, time.Now().Unix()
	a.Result = &factory.AssignmentResult{AssignmentID: a.ID, RunID: a.Run, Phase: project.ExecutionVerify, Status: "completed", Summary: "Application checks passed", Reported: true, RecordedUnix: time.Now().Unix(), PhaseResult: &factory.PhaseResult{Version: 2, Phase: project.ExecutionVerify, Status: "completed", Summary: "Application checks passed", Output: &factory.PhaseOutput{Verification: &report}}}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	raw, _ := json.Marshal(a)
	if _, err := sqlDB.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,0,$2,'finished',0,$3)`, a.ID, a.Run, string(raw)); err != nil {
		t.Fatal(err)
	}
	event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: 3, Generation: 1, PublicationID: factory.NewID(), AcceptanceID: binding.AcceptanceID, MergedCommit: fx.tip, ClosingOccurrence: "1"}
	raw, _ = json.Marshal(event)
	if _, err := sqlDB.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,1,$1,$2,'1',$3)`, event.ID, event.PublicationID, string(raw)); err != nil {
		t.Fatal(err)
	}
	host := &fakeDispatchHost{inspectPreparation: func(project.PrepareInspect) (project.PrepareState, error) { return state, nil }, contextRead: func(request project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
		material := fixturePreparationContext(request, role)
		material.Check = script
		return material, nil
	}}
	reads := &appVerificationReads{tip: fx.tip}
	c := &Coordinator{Store: db, Host: host, DispatchReads: reads}
	// Worker prose and a normal successful setup check are insufficient.
	goodScript := script
	script = []byte("#!/bin/sh\nexit 0\n")
	if err := c.verifyApplicationReport(ctx, a); err == nil {
		t.Fatal("ordinary preparation was promoted to application completion")
	}
	script = goodScript
	bad := 1
	state.CheckExit = &bad
	if err := c.verifyApplicationReport(ctx, a); err == nil {
		t.Fatal("failing native check was accepted")
	}
	state.CheckExit = &zero
	reads.tip = strings.Repeat("e", 40)
	if err := c.verifyApplicationReport(ctx, a); err == nil {
		t.Fatal("stale integrated source was accepted")
	}
	reads.tip = fx.tip
	claimed := report.CheckReceipts[0].ArtifactDigest
	report.CheckReceipts[0].ArtifactDigest = strings.Repeat("f", 64)
	if err := c.verifyApplicationReport(ctx, a); err == nil {
		t.Fatal("forged receipt was accepted")
	}
	report.CheckReceipts[0].ArtifactDigest = claimed
	acceptedCoverage := report.Coverage[0]
	report.Coverage = append(report.Coverage, factory.ReassessmentCoverage{IntentID: revision.Intent, OutcomeID: phasedDecisionID(99), TaskKeys: []string{task.Key}, DeliveredAcceptanceIDs: []string{binding.AcceptanceID}})
	if err := c.verifyApplicationReport(ctx, a); err == nil {
		t.Fatal("unknown outcome was accepted as delivered scope")
	}
	report.Coverage = []factory.ReassessmentCoverage{acceptedCoverage}
	if err := c.verifyApplicationReport(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := c.verifyApplicationReport(ctx, a); err != nil {
		t.Fatal("completion replay", err)
	}
	var count int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM factory_feed_comments WHERE data->'communication'->>'component_id'='F10'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("trusted completion count %d: %v", count, err)
	}
	entries, _, _, err := db.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	post, err := db.FeedComment(ctx, entries[0].ID)
	if err != nil || post.Actor != "factory" || post.Status != "application:verified" || post.Payload.Communication.ComponentID != "F10" {
		t.Fatalf("completion projection: %+v %v", post, err)
	}
	pending, err := db.PendingAppVerifications(ctx, 16)
	if err != nil || len(pending) != 0 {
		t.Fatalf("completed delivery still pending: %+v %v", pending, err)
	}
}
