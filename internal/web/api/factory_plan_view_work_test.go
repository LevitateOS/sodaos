package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func journeyCommitPair() (string, string) {
	return strings.Repeat("a", 40), strings.Repeat("b", 40)
}

func journeyDigest() string {
	return strings.Repeat("d", 64)
}

func journeyID(seed byte) string {
	return strings.Repeat(string([]byte{seed}), 32)
}

func TestJourneyPhaseRendersFrozenVocabulary(t *testing.T) {
	for _, phase := range []string{
		project.ExecutionDecompose, project.ExecutionInvestigate, project.ExecutionImplement,
		project.ExecutionReview, project.ExecutionCorrect, project.ExecutionReassess,
	} {
		view, err := journeyPhaseDTO(phase, journeyActive)
		if err != nil {
			t.Fatalf("phase %s: %v", phase, err)
		}
		if view.Phase != phase || view.State != journeyActive {
			t.Fatalf("phase render: %+v", view)
		}
	}
	if _, err := journeyPhaseDTO("publish", journeyActive); err == nil {
		t.Fatal("unknown phase accepted")
	}
	if _, err := journeyPhaseDTO(project.ExecutionReview, "done"); err == nil {
		t.Fatal("unknown phase state accepted")
	}
}

func TestJourneyTaskRendersLaneAndIssue(t *testing.T) {
	view, err := journeyTaskDTO("task-plan", journeyLaneB, "12", journeyComplete)
	if err != nil {
		t.Fatal(err)
	}
	if view.Key != "task-plan" || view.Lane != journeyLaneB || view.Issue != "12" {
		t.Fatalf("task render: %+v", view)
	}
	for _, lane := range []string{journeyLaneA, journeyLaneC, journeyLaneUnassigned} {
		if _, err := journeyTaskDTO("task-x", lane, "", journeyPending); err != nil {
			t.Fatalf("lane %s: %v", lane, err)
		}
	}
	for _, bad := range [][4]string{
		{"", journeyLaneA, "", journeyPending},
		{"task-x", "lane-d", "", journeyPending},
		{"task-x", journeyLaneA, "0", journeyPending},
		{"task-x", journeyLaneA, "", "done"},
	} {
		if _, err := journeyTaskDTO(bad[0], bad[1], bad[2], bad[3]); err == nil {
			t.Fatalf("task accepted: %+v", bad)
		}
	}
}

func TestJourneyCommitRendersPullLinkage(t *testing.T) {
	head, base := journeyCommitPair()
	view, err := journeyCommitDTO(head, base, "34", "56")
	if err != nil {
		t.Fatal(err)
	}
	if view.Commit != head || view.PRNumber != "34" {
		t.Fatalf("commit render: %+v", view)
	}
	if _, err := journeyCommitDTO(head, head, "34", "56"); err == nil {
		t.Fatal("identical head and base accepted")
	}
	if _, err := journeyCommitDTO("short", base, "34", "56"); err == nil {
		t.Fatal("short commit accepted")
	}
	if _, err := journeyCommitDTO(head, base, "0", "56"); err == nil {
		t.Fatal("zero pull number accepted")
	}
}

func TestJourneyFindingRendersFrozenContract(t *testing.T) {
	finding := factory.Finding{
		Key: journeyID('f'), OriginRun: journeyID('e'),
		Severity: factory.FindingSeverityMajor, Defect: "Missing bound check.",
		Evidence: "hunk 3 writes past the slice.", RequiredOutcome: "Bounds-checked write.",
		FixKind: factory.FindingFixCode, Actionable: true,
	}
	view, err := journeyFindingDTO(finding)
	if err != nil {
		t.Fatal(err)
	}
	if view.Severity != factory.FindingSeverityMajor || view.FixKind != factory.FindingFixCode {
		t.Fatalf("finding render: %+v", view)
	}
	if _, err := journeyFindingDTO(factory.Finding{}); err == nil {
		t.Fatal("empty finding accepted")
	}
}

func TestJourneyCorrectionRendersEffects(t *testing.T) {
	head, _ := journeyCommitPair()
	view, err := journeyCorrectionDTO(head, factory.OpEffectCommitted, factory.OpCompletionComplete, "")
	if err != nil {
		t.Fatal(err)
	}
	if view.Effect != factory.OpEffectCommitted || view.Completion != factory.OpCompletionComplete {
		t.Fatalf("correction render: %+v", view)
	}
	if _, err := journeyCorrectionDTO("", "", "", ""); err != nil {
		t.Fatalf("pending correction refused: %v", err)
	}
	if _, err := journeyCorrectionDTO(head, "merged", "", ""); err == nil {
		t.Fatal("unknown correction effect accepted")
	}
	if _, err := journeyCorrectionDTO(head, "", "done", ""); err == nil {
		t.Fatal("unknown correction completion accepted")
	}
}

func TestJourneyJudgmentRendersChoicesAndWaits(t *testing.T) {
	decided, err := journeyJudgmentDTO(journeyJudgmentReassessDecision, factory.ReassessmentOptionReassess, "jev-1.13.0", 0.8, 0, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if decided.Wait || decided.WaitReason != "" {
		t.Fatalf("decided judgment waits: %+v", decided)
	}
	waiting, err := journeyJudgmentDTO(journeyJudgmentDelivery, journeyJudgmentDeliveryReview, "jev-1.13.0", 0.4, 0.4, true, "low_confidence")
	if err != nil {
		t.Fatal(err)
	}
	if !waiting.Wait || waiting.WaitReason != "low_confidence" {
		t.Fatalf("waiting judgment: %+v", waiting)
	}
	if _, err := journeyJudgmentDTO(journeyJudgmentDelivery, "merge_now", "", 0, 0, false, ""); err == nil {
		t.Fatal("unknown delivery choice accepted")
	}
	if _, err := journeyJudgmentDTO(journeyJudgmentReassessDecision, factory.ReassessmentOptionContinue, "", 0.2, 0, false, ""); err == nil {
		t.Fatal("below-threshold reassessment decision accepted")
	}
	if _, err := journeyJudgmentDTO(journeyJudgmentReassessProposal, factory.ReassessmentOptionAccept, "", 0.9, 0.5, false, ""); err == nil {
		t.Fatal("below-threshold reassessment proposal accepted")
	}
	if _, err := journeyJudgmentDTO(journeyJudgmentPriority, "7/42", "", 0.7, 0, true, ""); err == nil {
		t.Fatal("wait without reason accepted")
	}
	if _, err := journeyJudgmentDTO(journeyJudgmentPriority, "7/42", "", 0.7, 0, false, "stale"); err == nil {
		t.Fatal("reason without wait accepted")
	}
	if _, err := journeyJudgmentDTO("oracle", "7/42", "", 0.7, 0, false, ""); err == nil {
		t.Fatal("unknown judgment kind accepted")
	}
}

func TestJourneyGateRendersD13Completion(t *testing.T) {
	gate, err := journeyGateDTO(factory.ReassessmentView{
		CompletionID: journeyID('c'), CycleID: journeyID('1'),
		State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gate.State != factory.ReassessmentDeciding || gate.Repository != 7 {
		t.Fatalf("gate render: %+v", gate)
	}
	waiting, err := journeyGateDTO(factory.ReassessmentView{
		CompletionID: journeyID('c'), CycleID: journeyID('1'),
		State:      factory.ReassessmentAttention,
		Decision:   factory.ReassessmentOptionReassess,
		Proposal:   factory.ReassessmentOptionNeedsAttention,
		Repository: 7, Generation: 2,
		WaitReason: "provider_unknown_response", Retryable: true,
		ProposalDigest: journeyDigest(), InventoryDigest: journeyDigest(),
		ReceiptDigests: []string{journeyDigest()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !waiting.Retryable || waiting.Proposal != factory.ReassessmentOptionNeedsAttention {
		t.Fatalf("waiting gate: %+v", waiting)
	}
	if _, err := journeyGateDTO(factory.ReassessmentView{}); err == nil {
		t.Fatal("empty gate accepted")
	}
}

func TestJourneyAdoptionRendersProposalPositions(t *testing.T) {
	intent, revision := "d"+"e"+"0123456789abcdef0123456", "d"+"f"+"0123456789abcdef0123456"
	for _, state := range []string{journeyAdoptionProposed, journeyAdoptionAdopted, journeyAdoptionReleased} {
		view, err := journeyAdoptionDTO(intent, revision, state)
		if err != nil {
			t.Fatalf("adoption %s: %v", state, err)
		}
		if view.State != state {
			t.Fatalf("adoption render: %+v", view)
		}
	}
	if _, err := journeyAdoptionDTO(intent, revision, "merged"); err == nil {
		t.Fatal("unknown adoption position accepted")
	}
	if _, err := journeyAdoptionDTO("intent-1", revision, journeyAdoptionProposed); err == nil {
		t.Fatal("non-decision intent accepted")
	}
}

func TestRenderWorkJourneyFixtureBoundsComposition(t *testing.T) {
	head, base := journeyCommitPair()
	phase, err := journeyPhaseDTO(project.ExecutionReview, journeyActive)
	if err != nil {
		t.Fatal(err)
	}
	task, err := journeyTaskDTO("task-plan", journeyLaneA, "12", journeyActive)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := journeyCommitDTO(head, base, "34", "56")
	if err != nil {
		t.Fatal(err)
	}
	gate, err := journeyGateDTO(factory.ReassessmentView{
		CompletionID: journeyID('c'), CycleID: journeyID('1'),
		State: factory.ReassessmentCollecting, Repository: 7, Generation: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err := renderWorkJourneyFixture(workJourneyFixture{
		Phases: []journeyPhaseView{phase}, Tasks: []journeyTaskView{task},
		Commits: []journeyCommitView{commit}, Gates: []journeyGateView{gate},
		Repository: "7", Issue: "12",
	}, 1720000300)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"phases", "tasks", "commits", "findings", "corrections", "judgments", "gates", "adoptions"} {
		items, ok := decoded[key].([]any)
		if !ok {
			t.Fatalf("%s missing: %v", key, decoded[key])
		}
		if key == "findings" || key == "corrections" || key == "judgments" || key == "adoptions" {
			if len(items) != 0 {
				t.Fatalf("%s not empty: %v", key, items)
			}
		} else if len(items) != 1 {
			t.Fatalf("%s: %v", key, items)
		}
	}
	overflow := workJourneyFixture{Repository: "7", Issue: "12"}
	for range maxJourneyTasks + 1 {
		overflow.Tasks = append(overflow.Tasks, task)
	}
	if _, err := renderWorkJourneyFixture(overflow, 1720000300); err == nil {
		t.Fatal("task overflow accepted")
	}
	if _, err := renderWorkJourneyFixture(workJourneyFixture{Repository: "7"}, 1720000300); err == nil {
		t.Fatal("fixture without issue accepted")
	}
}
