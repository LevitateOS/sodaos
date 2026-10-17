// FW32 reassessment-coder adoption tests (T35–T39).
//
// Deterministic fixtures only: the Jev proposal judgment arrives as
// caller-supplied option/probability/confidence inputs. No live Jev
// consultation, no native effects, no PG, no source writes.
package control

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func adoptHex(seed byte, n int) string { return strings.Repeat(string([]byte{seed}), n) }

func adoptDecision(seed byte) string { return "d" + strings.Repeat(string([]byte{seed}), 24) }

func adoptPrior() (factory.IntentRevision, map[string]factory.IntentTaskBinding) {
	ideaA, ideaB := adoptDecision('a'), adoptDecision('b')
	rev := factory.IntentRevision{
		ID: adoptDecision('1'), Intent: adoptDecision('0'), Owner: 1, Repository: 7,
		Objective: "Serve requests", State: "accepted", MaxIssues: 4, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{
			{ID: ideaA, Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "supported", Evidence: "observed", Provenance: "autonomous"},
			{ID: ideaB, Description: "Idle work", Consequence: "Work completes", Decision: "accept", Source: "supported", Evidence: "observed", Provenance: "autonomous"},
		},
		EffectiveOutcomes: []string{ideaA, ideaB},
		Tasks: []factory.IntentTask{
			{Key: adoptDecision('c'), Objective: "Delivered history", Outcomes: []string{"History preserved"}, IdeaIDs: []string{ideaA}},
			{Key: adoptDecision('d'), Objective: "Idle work", Outcomes: []string{"Work done"}, IdeaIDs: []string{ideaB}},
			{Key: adoptDecision('e'), Objective: "Downstream", Outcomes: []string{"Downstream done"}, IdeaIDs: []string{ideaB}, DependsOn: []string{adoptDecision('d')}},
		},
	}
	bindings := map[string]factory.IntentTaskBinding{
		adoptDecision('c'): {RevisionID: rev.ID, TaskKey: adoptDecision('c'), IssueID: 101, IssueIndex: 1, AcceptanceID: adoptDecision('5')},
		adoptDecision('d'): {RevisionID: rev.ID, TaskKey: adoptDecision('d'), IssueID: 102, IssueIndex: 2, AcceptanceID: adoptDecision('6')},
		adoptDecision('e'): {RevisionID: rev.ID, TaskKey: adoptDecision('e'), IssueID: 103, IssueIndex: 3, AcceptanceID: adoptDecision('2')},
	}
	return rev, bindings
}

func adoptPriors(rev factory.IntentRevision, bindings map[string]factory.IntentTaskBinding) map[string]ReassessmentPrior {
	return map[string]ReassessmentPrior{
		rev.Intent: {
			Revision:  rev,
			Bindings:  bindings,
			Protected: map[string]bool{adoptDecision('c'): true},
			PrereqAcceptance: map[string]map[string]string{
				adoptDecision('c'): {},
				adoptDecision('d'): {},
				adoptDecision('e'): {adoptDecision('d'): adoptDecision('6')},
			},
		},
	}
}

func adoptAuthority(intent string) ReassessmentAuthority {
	ideaA, ideaB := adoptDecision('a'), adoptDecision('b')
	return ReassessmentAuthority{
		EffectiveIdeas:    map[string]map[string]bool{intent: {ideaA: true, ideaB: true}},
		DeliveredVerified: map[string]bool{adoptDecision('8'): true, adoptDecision('4'): true},
	}
}

func adoptReport(rev factory.IntentRevision) factory.ReassessmentReport {
	ideaA, ideaB := adoptDecision('a'), adoptDecision('b')
	changed := adoptDecision('d')
	tasks := []factory.IntentTask{
		{Key: adoptDecision('c'), Objective: "Delivered history", Outcomes: []string{"History preserved"}, IdeaIDs: []string{ideaA}},
		{Key: changed, Objective: "Idle work with validation", Outcomes: []string{"Work done"}, IdeaIDs: []string{ideaB}},
		{Key: adoptDecision('e'), Objective: "Downstream", Outcomes: []string{"Downstream done"}, IdeaIDs: []string{ideaB}, DependsOn: []string{changed}},
	}
	return factory.ReassessmentReport{
		CompletionID: adoptHex('1', 32), CycleID: adoptHex('2', 32),
		InventoryDigest: adoptHex('f', 64), SourceCommit: adoptHex('c', 40),
		PreparationDigest: adoptHex('0', 64), LocalInputDigest: adoptHex('9', 64),
		Inspected:            []factory.InspectedSource{{Path: "internal/store/factory_completions.go", Commit: adoptHex('c', 40), Evidence: "sealed row"}},
		Summary:              "Split idle work; no scope change",
		RetainedIssueIndexes: []int64{1, 3, 4},
		IntentChanges: []factory.ReassessmentIntentChange{{
			IntentID: rev.Intent, ExpectedRevision: adoptDecision('9'),
			ScopeDigest: rev.Digest(), Reason: "Clarify the idle task within its accepted scope",
			Tasks: tasks,
		}},
		Coverage: []factory.ReassessmentCoverage{
			{IntentID: rev.Intent, OutcomeID: ideaA, TaskKeys: []string{adoptDecision('c')}, DeliveredAcceptanceIDs: []string{adoptDecision('8')}},
			{IntentID: rev.Intent, OutcomeID: ideaB, TaskKeys: []string{changed}, DeliveredAcceptanceIDs: []string{adoptDecision('4')}},
		},
	}
}

func adoptFreshness() ReassessmentFreshness {
	return ReassessmentFreshness{
		SourceCommit: adoptHex('c', 40), PreparationDigest: adoptHex('0', 64),
		ScopeDigestValid: true, AuthorityValid: true,
	}
}

func adoptJudgment() ReassessmentProposalJudgment {
	return ReassessmentProposalJudgment{Option: factory.ReassessmentOptionAccept, Probability: 0.80, Confidence: 0.80}
}

func adoptAssignment(t *testing.T, completion, cycle string) factory.Assignment {
	t.Helper()
	prompt := []byte("reassessment prompt with the complete inventory inline")
	sum := sha256.Sum256(prompt)
	id := factory.NewID()
	run := factory.NewID()
	return factory.Assignment{
		ID: id, AttemptRoot: id,
		ProjectID: "p" + strings.Repeat("d", 24), Role: project.RoleCoder,
		Repository: 7, Preparation: "f" + strings.Repeat("b", 24),
		Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "test-model",
		Connection: "conn", SourceCommit: adoptHex('c', 40), ActorID: 7,
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: run, RunHistory: []string{run}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: 1700000000, Revision: 0, NativeRev: 11,
		Subject: factory.ExecutionSubject{
			Kind:       factory.SubjectCompletion,
			Completion: &factory.CompletionSubject{CompletionID: completion, CycleID: cycle, InventoryDigest: adoptHex('f', 64)},
		},
		Phase: project.ExecutionReassess,
	}
}

func adoptCycle(completion, cycle string) factory.ReassessmentCycle {
	ledger, err := factory.ReassessmentLedger{}.AdmitCoderRun(adoptHex('7', 32), factory.ReassessmentOptionReassess)
	if err != nil {
		panic(err)
	}
	return factory.ReassessmentCycle{
		Ledger: ledger, EventID: completion, CycleID: cycle,
		State: factory.ReassessmentReassessing, Repository: 7, Generation: 3,
		Decision: factory.ReassessmentOptionReassess,
	}
}

// T35: the cycle admits exactly its fresh completion-subject coder bound
// to the SAME recorded paid run. A continue decision launches no agent;
// a parented or publishing assignment never reopens the completed root;
// the wrong role, phase, run, repository, cycle or state refuses.
func TestAdmitReassessmentCoder(t *testing.T) {
	completion, cycle := adoptHex('1', 32), adoptHex('2', 32)
	a := adoptAssignment(t, completion, cycle)
	c := adoptCycle(completion, cycle)
	c.Ledger.CoderRuns = []string{a.Run}
	if err := AdmitReassessmentCoder(c, a, factory.ReassessmentOptionReassess); err != nil {
		t.Fatal("fresh reassessment root refused:", err)
	}
	if err := AdmitReassessmentCoder(c, adoptAssignment(t, completion, cycle), factory.ReassessmentOptionContinue); err == nil {
		t.Fatal("continue launched an agent")
	}
	child := adoptAssignment(t, completion, cycle)
	child.ParentAssignment = adoptHex('7', 32)
	child.AttemptRoot = adoptHex('7', 32)
	if err := AdmitReassessmentCoder(c, child, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("parented assignment reopened its completed root")
	}
	publishing := adoptAssignment(t, completion, cycle)
	publishing.PublicationAssignment = adoptHex('8', 32)
	if err := AdmitReassessmentCoder(c, publishing, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("publishing assignment bound its cycle")
	}
	reviewer := adoptAssignment(t, completion, cycle)
	reviewer.Role = project.RoleReviewer
	if err := AdmitReassessmentCoder(c, reviewer, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("reviewer admitted to the coder phase")
	}
	wrongPhase := a
	wrongPhase.Phase = project.ExecutionInvestigate
	if err := AdmitReassessmentCoder(c, wrongPhase, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("wrong phase admitted its coder")
	}
	other := adoptCycle(adoptHex('0', 32), cycle)
	if err := AdmitReassessmentCoder(other, adoptAssignment(t, completion, cycle), factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("another completion bound its cycle")
	}
	otherCycle := c
	otherCycle.CycleID = adoptHex('3', 32)
	if err := AdmitReassessmentCoder(otherCycle, a, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("another cycle bound its assignment")
	}
	bare := c
	bare.Ledger = factory.ReassessmentLedger{}
	if err := AdmitReassessmentCoder(bare, a, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("coder admitted without its ledger run")
	}
	wrongRun := a
	wrongRun.Run = factory.NewID()
	wrongRun.RunHistory = []string{wrongRun.Run}
	if err := wrongRun.Validate(); err != nil {
		t.Fatal("wrong-run fixture refused its own validation:", err)
	}
	if err := AdmitReassessmentCoder(c, wrongRun, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("unrecorded run bound its cycle")
	}
	wrongRepo := a
	wrongRepo.Repository++
	if err := wrongRepo.Validate(); err != nil {
		t.Fatal("wrong-repo fixture refused its own validation:", err)
	}
	if err := AdmitReassessmentCoder(c, wrongRepo, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("cross-repository reassessment admitted")
	}
	attention := c
	attention.State = factory.ReassessmentAttention
	if err := AdmitReassessmentCoder(attention, a, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("attention cycle admitted its coder")
	}
	stale := a
	second := factory.NewID()
	stale.RunHistory = []string{stale.Run, second}
	stale.Run = second
	stale.Attempts = 2
	if err := stale.Validate(); err != nil {
		t.Fatal("stale-attempt fixture refused its own validation:", err)
	}
	if err := AdmitReassessmentCoder(c, stale, factory.ReassessmentOptionReassess); err == nil {
		t.Fatal("stale second attempt reused its root")
	}
}

// T35: the 320 KiB prompt exception is pinned to genuine reassessment
// cycles while every other prompt keeps the 64 KiB bound.
func TestReassessmentPromptBoundPinned(t *testing.T) {
	completion, cycle := adoptHex('1', 32), adoptHex('2', 32)
	base := adoptAssignment(t, completion, cycle)
	big := make([]byte, project.MaxReassessPrompt)
	for i := range big {
		big[i] = 'p'
	}
	sum := sha256.Sum256(big)
	reassess := base
	reassess.Prompt, reassess.PromptSHA = big, hex.EncodeToString(sum[:])
	if err := reassess.Validate(); err != nil {
		t.Fatal("320 KiB reassess prompt refused:", err)
	}
	over := make([]byte, project.MaxReassessPrompt+1)
	for i := range over {
		over[i] = 'p'
	}
	overSum := sha256.Sum256(over)
	tooBig := base
	tooBig.Prompt, tooBig.PromptSHA = over, hex.EncodeToString(overSum[:])
	if err := tooBig.Validate(); err == nil {
		t.Fatal("oversized reassess prompt admitted")
	}
	issue := base
	issue.Subject = factory.ExecutionSubject{
		Kind:  factory.SubjectIssue,
		Issue: &factory.IssueSubject{Issue: 3, Acceptance: adoptDecision('3')},
	}
	issue.Phase = project.ExecutionInvestigate
	issue.Issue, issue.Acceptance = 3, adoptDecision('3')
	issue.PublicationAssignment = factory.NewID()
	issue.Prompt, issue.PromptSHA = big, hex.EncodeToString(sum[:])
	if err := issue.Validate(); err == nil {
		t.Fatal("320 KiB investigate prompt admitted")
	}
}

// T36/T37: one coherent revision validates, preflights deterministically,
// and stages serially. Protected bindings are copied frozen; the changed
// idle task and its transitively affected dependent rebind; the packet
// stays within its native 64-change/48 KiB bounds.
func TestValidateReassessmentAdoption(t *testing.T) {
	rev, bindings := adoptPrior()
	report := adoptReport(rev)
	packets, validation, journal, err := ValidateReassessmentAdoption(
		report, adoptPriors(rev, bindings), adoptAuthority(rev.Intent),
		adoptJudgment(), adoptFreshness(), adoptHex('f', 64), 4, []int64{2},
		adoptHex('1', 32), adoptHex('2', 32),
	)
	if err != nil {
		t.Fatal("coherent revision refused:", err)
	}
	if len(packets) != 1 {
		t.Fatalf("one intent needs one packet: %d", len(packets))
	}
	packet := packets[0]
	if packet.NoEffect || packet.SuccessorRevision != adoptDecision('9') {
		t.Fatalf("packet does not carry its successor: %+v", packet)
	}
	if len(packet.CopiedBindings) != 1 || packet.CopiedBindings[0].TaskKey != adoptDecision('c') {
		t.Fatalf("protected binding not copied frozen: %+v", packet.CopiedBindings)
	}
	copied := packet.CopiedBindings[0]
	if copied.IssueID != 101 || copied.AcceptanceID != adoptDecision('5') {
		t.Fatal("unchanged binding lost its native or acceptance identity")
	}
	found := map[string]bool{}
	for _, key := range packet.ReboundKeys {
		found[key] = true
	}
	if !found[adoptDecision('d')] || !found[adoptDecision('e')] {
		t.Fatalf("changed task and transitive dependent not rebound: %v", packet.ReboundKeys)
	}
	if len(packet.Payload.Edits) != 1 || packet.Payload.Edits[0].IssueID != 102 {
		t.Fatalf("packet does not edit exactly the changed idle task: %+v", packet.Payload)
	}
	if err := validation.Validate(); err != nil {
		t.Fatal("validation report refused:", err)
	}
	if len(journal.Steps) != 1 || journal.Steps[0].IntentID != rev.Intent {
		t.Fatalf("journal does not stage its single step: %+v", journal.Steps)
	}
	again, _, _, err := ValidateReassessmentAdoption(
		report, adoptPriors(rev, bindings), adoptAuthority(rev.Intent),
		adoptJudgment(), adoptFreshness(), adoptHex('f', 64), 4, []int64{2},
		adoptHex('1', 32), adoptHex('2', 32),
	)
	if err != nil {
		t.Fatal("deterministic preflight refused its replay:", err)
	}
	if again[0].PacketDigest != packet.PacketDigest {
		t.Fatal("packet digest is not deterministic")
	}
}

// T36: attention, weak judgments, scope inflation and protected edits
// refuse before any prefix commits.
func TestValidateReassessmentAdoptionRefusals(t *testing.T) {
	rev, bindings := adoptPrior()
	base := func() (factory.ReassessmentReport, map[string]ReassessmentPrior, ReassessmentAuthority, ReassessmentProposalJudgment, ReassessmentFreshness) {
		return adoptReport(rev), adoptPriors(rev, bindings), adoptAuthority(rev.Intent), adoptJudgment(), adoptFreshness()
	}
	validate := func(report factory.ReassessmentReport, priors map[string]ReassessmentPrior, authority ReassessmentAuthority, judgment ReassessmentProposalJudgment, freshness ReassessmentFreshness) error {
		_, _, _, err := ValidateReassessmentAdoption(report, priors, authority, judgment, freshness,
			adoptHex('f', 64), 4, []int64{2}, adoptHex('1', 32), adoptHex('2', 32))
		return err
	}
	withAttention := adoptReport(rev)
	withAttention.Attention = []factory.ReassessmentAttentionEntry{{IssueIndex: 4, Reason: "needs product review"}}
	if err := validate(withAttention, adoptPriors(rev, bindings), adoptAuthority(rev.Intent), adoptJudgment(), adoptFreshness()); err == nil {
		t.Fatal("attention did not hold the event")
	}
	report, priors, authority, _, freshness := base()
	weak := adoptJudgment()
	weak.Option = factory.ReassessmentOptionNeedsAttention
	if err := validate(report, priors, authority, weak, freshness); err == nil {
		t.Fatal("needs_attention proposal accepted")
	}
	report, priors, authority, _, freshness = base()
	low := adoptJudgment()
	low.Probability = 0.79
	if err := validate(report, priors, authority, low, freshness); err == nil {
		t.Fatal("low-probability proposal accepted")
	}
	report, priors, authority, _, freshness = base()
	report.IntentChanges[0].Tasks[1].IdeaIDs = []string{adoptDecision('7')}
	if err := validate(report, priors, authority, adoptJudgment(), freshness); err == nil {
		t.Fatal("scope expansion admitted")
	}
	report, priors, authority, _, freshness = base()
	report.IntentChanges[0].Tasks[0].Objective = "Rewritten history"
	if err := validate(report, priors, authority, adoptJudgment(), freshness); err == nil {
		t.Fatal("protected edit admitted")
	}
	report, priors, authority, _, freshness = base()
	stale := adoptFreshness()
	stale.AuthorityValid = false
	if err := validate(report, priors, authority, adoptJudgment(), stale); err == nil {
		t.Fatal("stale authority admitted before effects")
	}
	report, priors, _, _, freshness = base()
	unverified := adoptAuthority(rev.Intent)
	unverified.DeliveredVerified = map[string]bool{}
	if err := validate(report, priors, unverified, adoptJudgment(), freshness); err == nil {
		t.Fatal("unverified delivery admitted")
	}
}

// T37: a changed prerequisite behind a protected dependent refuses; the
// per-intent packet refuses past 64 native changes before any prefix.
func TestReassessmentProtectedRouteAndPacketBound(t *testing.T) {
	rev, bindings := adoptPrior()
	priors := adoptPriors(rev, bindings)
	priors[rev.Intent] = ReassessmentPrior{
		Revision: rev, Bindings: bindings,
		Protected: map[string]bool{
			adoptDecision('c'): true,
			adoptDecision('e'): true,
		},
		PrereqAcceptance: map[string]map[string]string{
			adoptDecision('c'): {},
			adoptDecision('d'): {},
			adoptDecision('e'): {adoptDecision('d'): adoptDecision('6')},
		},
	}
	report := adoptReport(rev)
	_, _, _, err := ValidateReassessmentAdoption(report, priors, adoptAuthority(rev.Intent),
		adoptJudgment(), adoptFreshness(), adoptHex('f', 64), 4, []int64{2},
		adoptHex('1', 32), adoptHex('2', 32))
	if err == nil {
		t.Fatal("changed protected prerequisite route admitted")
	}
	big := rev
	big.MaxIssues = 64
	bigPriors := adoptPriors(big, bindings)
	bigPriors[big.Intent] = ReassessmentPrior{
		Revision: big, Bindings: bindings,
		Protected: map[string]bool{adoptDecision('c'): true},
		PrereqAcceptance: map[string]map[string]string{
			adoptDecision('c'): {},
			adoptDecision('d'): {},
			adoptDecision('e'): {adoptDecision('d'): adoptDecision('6')},
		},
	}
	many := adoptReport(big)
	tasks := make([]factory.IntentTask, 0, 12)
	keys := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		key := "d" + adoptHex(byte('a'+i%6), 12) + adoptHex(byte('0'+i/6), 12)
		deps := append([]string(nil), keys...)
		tasks = append(tasks, factory.IntentTask{
			Key: key, Objective: "New idle work", Outcomes: []string{"Work done"},
			IdeaIDs:   []string{adoptDecision('b')},
			DependsOn: deps,
		})
		keys = append(keys, key)
	}
	many.IntentChanges[0].Tasks = append(many.IntentChanges[0].Tasks, tasks...)
	many.Coverage[1].TaskKeys = append(many.Coverage[1].TaskKeys, keys...)
	_, _, _, err = ValidateReassessmentAdoption(many, bigPriors, adoptAuthority(big.Intent),
		adoptJudgment(), adoptFreshness(), adoptHex('f', 64), 4, []int64{2},
		adoptHex('1', 32), adoptHex('2', 32))
	if err == nil {
		t.Fatal("oversized packet admitted")
	}
}

// T38: serial journal with prefix preservation. The second intent waits
// behind the first; a lost reply resumes its same operation; a wrong
// readback never advances the cursor; an activated head never releases
// until materialized.
func TestReassessmentJournalSerialRecovery(t *testing.T) {
	journal := ReassessmentJournal{
		CompletionID: adoptHex('1', 32), CycleID: adoptHex('2', 32), InventoryDigest: adoptHex('f', 64),
		Steps: []ReassessmentJournalStep{
			{IntentID: adoptDecision('0'), SuccessorRevision: adoptDecision('9'), PacketDigest: adoptHex('a', 64), OperationID: "op-first", State: reassessmentStepStaged},
			{IntentID: adoptDecision('1'), SuccessorRevision: adoptDecision('8'), PacketDigest: adoptHex('b', 64), OperationID: "op-second", State: reassessmentStepStaged},
		},
	}
	if err := journal.GuardRecoveryStep(adoptDecision('1')); err == nil {
		t.Fatal("generic recovery ran a later staged revision")
	}
	if _, err := journal.MarkStepActive(adoptDecision('1')); err == nil {
		t.Fatal("later step activated out of order")
	}
	first, err := journal.MarkStepActive(adoptDecision('0'))
	if err != nil || first.OperationID != "op-first" {
		t.Fatal("current step did not activate:", err)
	}
	op, err := journal.ResumeOperation(adoptDecision('0'))
	if err != nil || op != "op-first" {
		t.Fatal("lost reply started a second operation:", op, err)
	}
	if err := journal.MarkStepCommitted(adoptDecision('0'), adoptHex('0', 64)); err == nil {
		t.Fatal("wrong readback advanced the cursor")
	}
	if err := journal.Releasable(); err == nil {
		t.Fatal("activated head released before materialization")
	}
	if err := journal.MarkStepCommitted(adoptDecision('0'), adoptHex('a', 64)); err != nil {
		t.Fatal("exact readback did not commit its prefix:", err)
	}
	if err := journal.GuardRecoveryStep(adoptDecision('0')); err == nil {
		t.Fatal("recovery reran a committed prefix")
	}
	if err := journal.GuardRecoveryStep(adoptDecision('1')); err != nil {
		t.Fatal("recovery did not advance to the next step:", err)
	}
	if _, err := journal.MarkStepActive(adoptDecision('1')); err != nil {
		t.Fatal("second step did not activate after its prefix:", err)
	}
	if err := journal.MarkStepCommitted(adoptDecision('1'), adoptHex('b', 64)); err != nil {
		t.Fatal("second readback did not commit:", err)
	}
	if err := journal.Releasable(); err != nil {
		t.Fatal("confirmed full adoption did not release:", err)
	}
	empty := ReassessmentJournal{CompletionID: adoptHex('1', 32), CycleID: adoptHex('2', 32)}
	if err := empty.Releasable(); err != nil {
		t.Fatal("accepted no-change report did not release:", err)
	}
}

// T39: source, scope and authority freshness invalidates before effects.
// A no-change accepted report releases without graph mutations.
func TestReassessmentFreshnessAndNoChangeRelease(t *testing.T) {
	rev, bindings := adoptPrior()
	quiet := adoptReport(rev)
	quiet.IntentChanges = nil
	quiet.Coverage = nil
	quiet.RetainedIssueIndexes = []int64{1, 2, 3, 4}
	if err := quiet.Validate(); err != nil {
		t.Fatal("empty change set refused:", err)
	}
	packets, _, journal, err := ValidateReassessmentAdoption(
		quiet, adoptPriors(rev, bindings), adoptAuthority(rev.Intent),
		adoptJudgment(), adoptFreshness(), adoptHex('f', 64), 4, nil,
		adoptHex('1', 32), adoptHex('2', 32),
	)
	if err != nil {
		t.Fatal("no-change report refused:", err)
	}
	if len(packets) != 0 || len(journal.Steps) != 0 {
		t.Fatal("no-change report staged graph mutations")
	}
	if err := journal.Releasable(); err != nil {
		t.Fatal("confirmed no-change adoption did not release:", err)
	}
	report := adoptReport(rev)
	changedSource := adoptFreshness()
	changedSource.SourceCommit = adoptHex('d', 40)
	_, _, _, err = ValidateReassessmentAdoption(report, adoptPriors(rev, bindings), adoptAuthority(rev.Intent),
		adoptJudgment(), changedSource, adoptHex('f', 64), 4, []int64{2},
		adoptHex('1', 32), adoptHex('2', 32))
	if err == nil {
		t.Fatal("changed source admitted before effects")
	}
	movedScope := adoptFreshness()
	movedScope.ScopeDigestValid = false
	_, _, _, err = ValidateReassessmentAdoption(report, adoptPriors(rev, bindings), adoptAuthority(rev.Intent),
		adoptJudgment(), movedScope, adoptHex('f', 64), 4, []int64{2},
		adoptHex('1', 32), adoptHex('2', 32))
	if err == nil {
		t.Fatal("moved scope admitted before effects")
	}
}

// T38: the staged change materializes as its exact successor revision:
// frozen accepted scope, prior predecessor, fresh identity, proposal
// tasks. Wrong-intent, repeated-identity, scope-mismatched and
// unaccepted-prior changes refuse before any store effect.
func TestSuccessorIntentRevision(t *testing.T) {
	rev, _ := adoptPrior()
	report := adoptReport(rev)
	change := report.IntentChanges[0]
	successor, err := SuccessorIntentRevision(rev, change, 1700000001)
	if err != nil {
		t.Fatal("staged change did not materialize:", err)
	}
	if successor.ID != adoptDecision('9') || successor.Predecessor != rev.ID {
		t.Fatalf("successor lacks its fresh chained identity: %+v", successor)
	}
	if successor.Intent != rev.Intent || successor.State != "accepted" {
		t.Fatalf("successor left its accepted scope: %+v", successor)
	}
	if len(successor.Tasks) != len(change.Tasks) {
		t.Fatalf("successor lost proposal tasks: %d", len(successor.Tasks))
	}
	if err := successor.Validate(); err != nil {
		t.Fatal("successor refused its own validation:", err)
	}
	other := change
	other.IntentID = adoptDecision('f')
	if _, err := SuccessorIntentRevision(rev, other, 1700000001); err == nil {
		t.Fatal("another intent materialized its successor")
	}
	repeat := change
	repeat.ExpectedRevision = rev.ID
	if _, err := SuccessorIntentRevision(rev, repeat, 1700000001); err == nil {
		t.Fatal("successor repeated its prior revision")
	}
	drifted := change
	drifted.ScopeDigest = adoptHex('e', 64)
	if _, err := SuccessorIntentRevision(rev, drifted, 1700000001); err == nil {
		t.Fatal("drifted scope materialized its successor")
	}
	draft := rev
	draft.State = "draft"
	if _, err := SuccessorIntentRevision(draft, change, 1700000001); err == nil {
		t.Fatal("unaccepted prior materialized its successor")
	}
	if _, err := SuccessorIntentRevision(rev, change, 0); err == nil {
		t.Fatal("successor materialized without its creation time")
	}
}

// T35: the single coder prompt carries the COMPLETE sealed inventory
// inline within the reassess-only 320 KiB bound. Incomplete, unordered
// and oversized inventories refuse; nothing is truncated.
func TestRenderReassessmentPrompt(t *testing.T) {
	promptInput := func() ReassessmentPromptInventory {
		return ReassessmentPromptInventory{
			Issues: []NativeInventoryIssue{
				{Index: 1, Data: "issue one body"},
				{Index: 3, Data: "issue three body"},
				{Index: 4, Data: "issue four body"},
			},
			CompletionID: adoptHex('1', 32), CycleID: adoptHex('2', 32),
			InventoryDigest: adoptHex('f', 64), SourceCommit: adoptHex('c', 40),
			PreparationDigest: adoptHex('0', 64),
			Repository:        7, Generation: 3, Complete: true,
		}
	}
	rendered, err := RenderReassessmentPrompt(promptInput())
	if err != nil {
		t.Fatal("complete inventory refused its prompt:", err)
	}
	text := string(rendered)
	for _, want := range []string{adoptHex('1', 32), adoptHex('f', 64), "issue one body", "issue three body", "issue four body"} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt lost its inline inventory: %q", want)
		}
	}
	incomplete := promptInput()
	incomplete.Complete = false
	if _, err := RenderReassessmentPrompt(incomplete); err == nil {
		t.Fatal("incomplete inventory rendered its prompt")
	}
	unordered := promptInput()
	unordered.Issues[1], unordered.Issues[2] = unordered.Issues[2], unordered.Issues[1]
	if _, err := RenderReassessmentPrompt(unordered); err == nil {
		t.Fatal("unordered inventory rendered its prompt")
	}
	big := promptInput()
	big.Issues = []NativeInventoryIssue{{Index: 1, Data: strings.Repeat("b", project.MaxReassessPrompt)}}
	if _, err := RenderReassessmentPrompt(big); err == nil {
		t.Fatal("oversized prompt truncated its inventory")
	}
	badDigest := promptInput()
	badDigest.InventoryDigest = "not-a-digest"
	if _, err := RenderReassessmentPrompt(badDigest); err == nil {
		t.Fatal("unsealed inventory rendered its prompt")
	}
}

// T35/result: a completed reassess result carries exactly its typed
// report within the 64 KiB bound; held runs carry none; cross-phase
// payloads refuse.
func TestReassessmentPhaseResultPayload(t *testing.T) {
	rev, _ := adoptPrior()
	report := adoptReport(rev)
	completed := factory.PhaseResult{
		Version: 2, Phase: project.ExecutionReassess, Status: "completed",
		Summary: "reassessment proposal", Output: &factory.PhaseOutput{Reassessment: &report},
	}
	if err := completed.Validate(); err != nil {
		t.Fatal("completed reassessment report refused:", err)
	}
	held := factory.PhaseResult{
		Version: 2, Phase: project.ExecutionReassess, Status: "failed",
		Summary: "held for attention",
	}
	if err := held.Validate(); err != nil {
		t.Fatal("held reassessment refused:", err)
	}
	withReport := factory.PhaseResult{
		Version: 2, Phase: project.ExecutionReassess, Status: "failed",
		Summary: "held for attention", Output: &factory.PhaseOutput{Reassessment: &report},
	}
	if err := withReport.Validate(); err == nil {
		t.Fatal("held reassessment admitted its report")
	}
	investigate := factory.PhaseResult{
		Version: 2, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan", Output: &factory.PhaseOutput{Reassessment: &report},
	}
	if err := investigate.Validate(); err == nil {
		t.Fatal("investigation admitted its reassessment report")
	}
}
