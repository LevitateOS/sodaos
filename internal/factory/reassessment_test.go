package factory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func reassessID(seed byte) string { return strings.Repeat(string([]byte{seed}), 32) }

func reassessDigest(seed byte) string { return strings.Repeat(string([]byte{seed}), 64) }

func reassessDecision(seed byte) string { return "d" + strings.Repeat(string([]byte{seed}), 24) }

func reassessCommit(seed byte) string { return strings.Repeat(string([]byte{seed}), 40) }

func canonicalCompletionEvent() CompletionEvent {
	return CompletionEvent{
		ID: reassessID('a'), PublicationID: reassessID('b'),
		MergedCommit: reassessCommit('c'), ClosingOccurrence: "901",
		AcceptanceID: reassessDecision('1'), Repository: 7, Issue: 12, Generation: 3,
	}
}

func canonicalReassessmentCycle() ReassessmentCycle {
	return ReassessmentCycle{
		EventID: reassessID('a'), CycleID: reassessID('d'),
		State: ReassessmentDeciding, Repository: 7, Generation: 3,
	}
}

func canonicalReassessmentTask(key, idea string) IntentTask {
	return IntentTask{
		Key: key, Objective: "Revise the idle task within its accepted scope",
		Outcomes: []string{"The accepted product behavior stays covered"},
		IdeaIDs:  []string{idea},
	}
}

func canonicalReassessmentReport() ReassessmentReport {
	idea := reassessDecision('2')
	intent := reassessDecision('3')
	change := ReassessmentIntentChange{
		IntentID: intent, ExpectedRevision: reassessDecision('4'),
		ScopeDigest: reassessDigest('e'), Reason: "Split the idle task for clarity",
		Tasks: []IntentTask{canonicalReassessmentTask(reassessDecision('5'), idea)},
	}
	return ReassessmentReport{
		CompletionID: reassessID('a'), CycleID: reassessID('d'),
		InventoryDigest: reassessDigest('f'), SourceCommit: reassessCommit('c'),
		PreparationDigest: reassessDigest('0'), LocalInputDigest: reassessDigest('9'),
		Inspected:            []InspectedSource{{Path: "internal/store/factory_completions.go", Commit: reassessCommit('c'), Evidence: "sealed row"}},
		Summary:              "One idle task split; no scope change",
		RetainedIssueIndexes: []int64{1, 3},
		IntentChanges:        []ReassessmentIntentChange{change},
		Coverage: []ReassessmentCoverage{{
			IntentID: intent, OutcomeID: idea,
			TaskKeys:               []string{reassessDecision('5')},
			DeliveredAcceptanceIDs: []string{reassessDecision('6')},
		}},
	}
}

// T33: the completion unique key is publication plus original closing
// occurrence; another recovery path observing the same effect coalesces.
func TestCompletionEventUniqueKey(t *testing.T) {
	first, second := canonicalCompletionEvent(), canonicalCompletionEvent()
	if first.UniqueKey() != second.UniqueKey() {
		t.Fatal("same completion effect needs one event key")
	}
	second.ClosingOccurrence = "902"
	if first.UniqueKey() == second.UniqueKey() {
		t.Fatal("distinct closing occurrences share an event key")
	}
	if err := first.Validate(); err != nil {
		t.Fatal("valid completion event refused:", err)
	}
	bad := first
	bad.Generation = 0
	if bad.Validate() == nil {
		t.Fatal("generation-zero completion admitted")
	}
}

// T33: generations order events per repository; the first completion
// carries generation 1.
func TestNextCompletionGeneration(t *testing.T) {
	next, err := NextCompletionGeneration(0)
	if err != nil || next != 1 {
		t.Fatal("first completion needs generation 1:", next, err)
	}
	next, err = NextCompletionGeneration(3)
	if err != nil || next != 4 {
		t.Fatal("generation did not advance by one:", next, err)
	}
	if _, err := NextCompletionGeneration(-1); err == nil {
		t.Fatal("negative generation admitted")
	}
}

// T35: two Jev requests and one coder run per cycle; the proposal
// assessment needs its reassess decision and failures stay charged.
func TestReassessmentLedgerBounds(t *testing.T) {
	ledger := ReassessmentLedger{}
	var err error
	ledger, err = ledger.ReserveDecisionCall(reassessID('1'), ReassessmentDecisionPurpose, "")
	if err != nil {
		t.Fatal("initial decision call refused:", err)
	}
	if _, err := ledger.ReserveDecisionCall(reassessID('2'), ReassessmentProposalPurpose, ReassessmentOptionContinue); err == nil {
		t.Fatal("proposal assessment without reassess admitted")
	}
	ledger, err = ledger.ReserveDecisionCall(reassessID('2'), ReassessmentProposalPurpose, ReassessmentOptionReassess)
	if err != nil {
		t.Fatal("proposal assessment refused:", err)
	}
	if _, err := ledger.ReserveDecisionCall(reassessID('3'), ReassessmentDecisionPurpose, ""); err == nil {
		t.Fatal("third Jev request admitted")
	}
	ledger, err = ledger.AdmitCoderRun(reassessID('4'), ReassessmentOptionReassess)
	if err != nil {
		t.Fatal("sole coder run refused:", err)
	}
	if _, err := ledger.AdmitCoderRun(reassessID('5'), ReassessmentOptionReassess); err == nil {
		t.Fatal("second coder run admitted")
	}
	if _, err := (ReassessmentLedger{}).AdmitCoderRun(reassessID('4'), ReassessmentOptionContinue); err == nil {
		t.Fatal("continue launched an agent")
	}
}

// T35: decision and proposal thresholds; low confidence holds the event
// with attention instead of inferring continue.
func TestReassessmentThresholds(t *testing.T) {
	if !ValidReassessmentDecision(ReassessmentOptionContinue, 0.50) {
		t.Fatal("threshold decision refused")
	}
	if ValidReassessmentDecision(ReassessmentOptionContinue, 0.49) {
		t.Fatal("low-confidence decision actionable")
	}
	if ValidReassessmentDecision("reschedule", 0.90) {
		t.Fatal("unknown decision option actionable")
	}
	if !ValidReassessmentProposal(ReassessmentOptionAccept, 0.80, 0.80) {
		t.Fatal("threshold proposal refused")
	}
	if ValidReassessmentProposal(ReassessmentOptionAccept, 0.79, 0.95) {
		t.Fatal("low-probability proposal acceptable")
	}
	if ValidReassessmentProposal(ReassessmentOptionNeedsAttention, 0.95, 0.95) {
		t.Fatal("needs-attention proposal acceptable")
	}
	cycle := canonicalReassessmentCycle()
	if _, err := cycle.RecordDecision(ReassessmentOptionContinue, 0.30); err == nil {
		t.Fatal("low-confidence decision recorded")
	}
	cycle, err := cycle.RecordDecision(ReassessmentOptionReassess, 0.75)
	if err != nil || cycle.Decision != ReassessmentOptionReassess {
		t.Fatal("valid reassess decision refused:", err)
	}
}

// T33/T35: one decision per completion across the pure cycle joint,
// including an empty inventory; only explicit Retry supersedes.
func TestReassessmentCycleJoints(t *testing.T) {
	cycle := canonicalReassessmentCycle()
	if err := cycle.Validate(); err != nil {
		t.Fatal("valid cycle refused:", err)
	}
	if err := cycle.AdvanceTo(ReassessmentReassessing); err == nil {
		t.Fatal("run admitted before its decision")
	}
	decided, err := cycle.RecordDecision(ReassessmentOptionContinue, 0.60)
	if err != nil {
		t.Fatal("valid continue refused:", err)
	}
	if err := decided.AdvanceTo(ReassessmentReleased); err != nil {
		t.Fatal("continue cannot release:", err)
	}
	retry := ReassessmentRetry{
		EventID: reassessID('a'), ExpectedPriorCycle: reassessID('d'),
		SuccessorCycleID: reassessID('e'), CommandID: reassessID('f'),
	}
	if _, err := decided.Supersede(retry); err == nil {
		t.Fatal("released cycle superseded")
	}
	held := canonicalReassessmentCycle()
	held.State = ReassessmentAttention
	wrong := retry
	wrong.ExpectedPriorCycle = reassessID('0')
	if _, err := held.Supersede(wrong); err == nil {
		t.Fatal("retry matched another cycle")
	}
	moved, err := held.Supersede(retry)
	if err != nil || moved.State != ReassessmentSuperseded || moved.Successor != reassessID('e') {
		t.Fatal("explicit retry did not supersede its attention cycle:", err)
	}
	empty := canonicalReassessmentReport()
	empty.RetainedIssueIndexes = nil
	empty.IntentChanges = nil
	empty.Coverage = nil
	if err := empty.Validate(); err != nil {
		t.Fatal("empty change set refused:", err)
	}
	if err := empty.AccountingHolds(0, nil); err != nil {
		t.Fatal("empty inventory accounting refused:", err)
	}
}

// T33: the completed issue root cannot reopen. A parented or
// publishing assignment never binds a cycle.
func TestReassessmentBindsFreshRootOnly(t *testing.T) {
	cycle := canonicalReassessmentCycle()
	fresh := func(t *testing.T) Assignment {
		t.Helper()
		prompt := testPrompt(t)
		a := testAssignment(prompt)
		a.RunHistory = []string{a.Run}
		a.Subject = ExecutionSubject{
			Kind: SubjectCompletion,
			Completion: &CompletionSubject{
				CompletionID: reassessID('a'), CycleID: reassessID('d'),
				InventoryDigest: reassessDigest('f'),
			},
		}
		a.Phase, a.Role = project.ExecutionReassess, project.RoleCoder
		a.ParentAssignment, a.PublicationAssignment = "", ""
		a.Issue, a.Acceptance = 0, ""
		return a
	}
	if err := cycle.BindsAssignment(fresh(t)); err != nil {
		t.Fatal("fresh reassessment root refused:", err)
	}
	other := canonicalReassessmentCycle()
	other.EventID = reassessID('0')
	if err := other.BindsAssignment(fresh(t)); err == nil {
		t.Fatal("another completion bound its cycle")
	}
	child := fresh(t)
	child.ParentAssignment = reassessID('7')
	child.AttemptRoot = reassessID('7')
	if err := cycle.BindsAssignment(child); err == nil {
		t.Fatal("parented assignment reopened its completed root")
	}
	publishing := fresh(t)
	publishing.PublicationAssignment = reassessID('8')
	if err := cycle.BindsAssignment(publishing); err == nil {
		t.Fatal("publishing assignment bound its cycle")
	}
}

// T35: typed scope/coverage/limits. Missing coverage, uncovered tasks,
// unknown issues and oversized reports refuse.
func TestReassessmentReportContracts(t *testing.T) {
	report := canonicalReassessmentReport()
	if err := report.Validate(); err != nil {
		t.Fatal("valid report refused:", err)
	}
	if err := report.AccountingHolds(3, []int64{2}); err != nil {
		t.Fatal("exact-once accounting refused:", err)
	}
	missing := canonicalReassessmentReport()
	missing.Coverage = nil
	if missing.Validate() == nil {
		t.Fatal("missing coverage admitted")
	}
	orphan := canonicalReassessmentReport()
	orphan.Coverage[0].TaskKeys = []string{reassessDecision('7')}
	if orphan.Validate() == nil {
		t.Fatal("coverage of an unproposed task admitted")
	}
	unscoped := canonicalReassessmentReport()
	unscoped.IntentChanges = append(unscoped.IntentChanges, ReassessmentIntentChange{
		IntentID: reassessDecision('8'), ExpectedRevision: reassessDecision('9'),
		ScopeDigest: reassessDigest('e'), Reason: "second intent",
		Tasks: []IntentTask{canonicalReassessmentTask(reassessDecision('0'), reassessDecision('a'))},
	})
	if unscoped.Validate() == nil {
		t.Fatal("unsorted intent changes admitted")
	}
	unknown := canonicalReassessmentReport()
	unknown.RetainedIssueIndexes = []int64{1, 9}
	if unknown.AccountingHolds(3, []int64{2}) == nil {
		t.Fatal("accounting admitted an unsealed issue")
	}
	partial := canonicalReassessmentReport()
	if partial.AccountingHolds(4, []int64{2}) == nil {
		t.Fatal("accounting admitted a skipped issue")
	}
	big := canonicalReassessmentReport()
	big.IntentChanges[0].Reason = strings.Repeat("r", MaxPlanTextBytes+1)
	if big.Validate() == nil {
		t.Fatal("oversized change reason admitted")
	}
	oversized := canonicalReassessmentReport()
	oversized.IntentChanges = nil
	oversized.Coverage = nil
	for i := 0; i < MaxReassessmentIntentChanges; i++ {
		intent := fmt.Sprintf("d%024x", i+1)
		idea := fmt.Sprintf("d%024x", 1000+i)
		task := fmt.Sprintf("d%024x", 2000+i)
		oversized.IntentChanges = append(oversized.IntentChanges, ReassessmentIntentChange{
			IntentID: intent, ExpectedRevision: reassessDecision('b'),
			ScopeDigest: reassessDigest('e'), Reason: strings.Repeat("r", MaxPlanTextBytes),
			Tasks: []IntentTask{canonicalReassessmentTask(task, idea)},
		})
		oversized.Coverage = append(oversized.Coverage, ReassessmentCoverage{
			IntentID: intent, OutcomeID: idea, TaskKeys: []string{task},
			DeliveredAcceptanceIDs: []string{reassessDecision('6')},
		})
	}
	if oversized.Validate() == nil {
		t.Fatal("oversized encoded report admitted")
	}
	attention := canonicalReassessmentReport()
	attention.Attention = []ReassessmentAttentionEntry{{IssueIndex: 7, Reason: "needs product review"}}
	if attention.AccountingHolds(3, []int64{2}) == nil {
		t.Fatal("attention on an unsealed issue admitted")
	}
}

// T35: the validation digest carries exact fields and sorted lists
// without narrative; the view carries gate state with its reason.
func TestReassessmentValidationAndView(t *testing.T) {
	digest := func(seed byte) string { return reassessDigest(seed) }
	valid := ReassessmentValidationReport{
		InventoryDigest: digest('1'), ProposalDigest: digest('2'), CoverageDigest: digest('3'),
		ScopeDigests:        []IntentDigest{{IntentID: reassessDecision('4'), Digest: digest('5')}},
		PacketDigests:       []IntentDigest{{IntentID: reassessDecision('4'), Digest: digest('6')}},
		MutableIssueIndexes: []int64{2}, ProtectedIssueIndexes: []int64{1, 3},
	}
	if err := valid.Validate(); err != nil {
		t.Fatal("valid validation report refused:", err)
	}
	unsorted := valid
	unsorted.MutableIssueIndexes = []int64{2, 2}
	if unsorted.Validate() == nil {
		t.Fatal("duplicated validation issue admitted")
	}
	view := ReassessmentView{
		CompletionID: reassessID('a'), CycleID: reassessID('d'),
		Repository: 7, Generation: 3, State: ReassessmentAttention,
		WaitReason: "proposal assessment unavailable; explicit retry may follow",
		Retryable:  true,
	}
	if err := view.Validate(); err != nil {
		t.Fatal("valid attention view refused:", err)
	}
	unreasoned := view
	unreasoned.WaitReason = ""
	if unreasoned.Validate() == nil {
		t.Fatal("reasonless attention view admitted")
	}
	readonly := view
	readonly.State = ReassessmentReleased
	readonly.Retryable = true
	if readonly.Validate() == nil {
		t.Fatal("released cycle waits for retry")
	}
}
