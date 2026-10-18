package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// FW06B intent-bound admission tests: pure, no Store/Host/Graph. The joints
// under test take none of those dependencies, which is itself the proof
// that intent-bound planning consumes planner outputs only — no
// accepted-graph or history prerequisite — and launches nothing by itself.

// Both planning operations enter through the same FW11 admission joint: a
// broad tracker graph through DecompositionOptions and a small single-issue
// graph through ProposeIntent.
func TestIntentBoundLaunchAdmitsBothOperationsEndToEnd(t *testing.T) {
	t.Run("broad options output", func(t *testing.T) {
		head := p4TrackerHead()
		in := p4PlannerInput(t, head, factory.IntentPlanningOptions)
		subject := planningSubject(head)
		if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningOptions, head, true, in); err != nil {
			t.Fatalf("authorized broad launch refused: %v", err)
		}
		stub := &stubPlanner{out: p4TrackerOutput(in)}
		out, err := PlanIntentDecomposition(context.Background(), subject, project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
		if err != nil {
			t.Fatalf("broad run refused: %v", err)
		}
		if err := AdmitIntentBoundResult(subject, project.ExecutionDecompose, in, out); err != nil {
			t.Fatalf("broad answer refused at the result boundary: %v", err)
		}
		if len(out.Candidates) != 1 || len(out.Candidates[0].Tasks) != 4 {
			t.Fatalf("broad answer lost its shape: %+v", out)
		}
	})

	t.Run("small propose output", func(t *testing.T) {
		head := p4TypoHead()
		in := p4PlannerInput(t, head, factory.IntentPlanningPropose)
		subject := planningSubject(head)
		if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningPropose, head, true, in); err != nil {
			t.Fatalf("authorized small launch refused: %v", err)
		}
		small := factory.SemanticPlannerOutput{
			Intent: head.Intent, Revision: head.ID, Operation: in.Operation,
			Candidates: []factory.SemanticCandidate{{Key: "typo-candidate", Rationale: "A wording correction is one cohesive issue.", Tasks: []factory.IntentTask{
				{Key: planningTaskKey(51), Objective: "Correct the approval confirmation wording", Outcomes: []string{"The status page shows the reviewed wording"}, IdeaIDs: []string{"d333333333333333333333335"}},
			}}},
		}
		if err := AdmitIntentBoundResult(subject, project.ExecutionDecompose, in, small); err != nil {
			t.Fatalf("small answer refused at the result boundary: %v", err)
		}
	})
}

func TestIntentBoundLaunchRefusesUnknownOperation(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	subject := planningSubject(head)
	if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleCoder, "decompose_extra", head, true, in); err == nil {
		t.Fatal("unknown operation launched")
	}
	mismatched := in
	mismatched.Operation = factory.IntentPlanningPropose
	if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningOptions, head, true, mismatched); err == nil {
		t.Fatal("operation-mismatched inputs launched")
	}
}

// Allowed-to-implemented transitions are gated: an intent subject names no
// phase but decompose, and no other subject kind enters this boundary.
func TestIntentBoundLaunchGatesPhaseTransitions(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	subject := planningSubject(head)
	for _, phase := range []string{project.ExecutionInvestigate, project.ExecutionImplement, project.ExecutionReview, project.ExecutionCorrect} {
		if err := AdmitIntentBoundLaunch(subject, phase, project.RoleCoder, factory.IntentPlanningOptions, head, true, in); !errors.Is(err, ErrIntentBoundTransition) {
			t.Fatalf("phase %s is not a gated transition: %v", phase, err)
		}
	}
	issue := factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 7, Acceptance: "d" + strings.Repeat("a", 24)}}
	if err := AdmitIntentBoundLaunch(issue, project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningOptions, head, true, in); err == nil {
		t.Fatal("issue subject entered intent-bound planning")
	}
	if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleReviewer, factory.IntentPlanningOptions, head, true, in); err == nil {
		t.Fatal("reviewer role entered intent-bound planning")
	}
}

func TestIntentBoundLaunchKeepsDraftAssociationAndAllowanceGates(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	subject := planningSubject(head)
	if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningOptions, head, false, in); !errors.Is(err, ErrIntentPlanningExhausted) {
		t.Fatalf("exhaustion is not visible: %v", err)
	}
	bare := head
	bare.Repository = 0
	if err := AdmitIntentBoundLaunch(planningSubject(bare), project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningOptions, bare, true, planningInput(t, bare, factory.IntentPlanningOptions)); err == nil {
		t.Fatal("repository-less draft entered the bound run")
	} else if !strings.Contains(err.Error(), "repository association") {
		t.Fatalf("refusal does not name the association gate: %v", err)
	}
	accepted := head
	accepted.State = "accepted"
	acceptedInput := planningInput(t, head, factory.IntentPlanningOptions)
	acceptedInput.Head = accepted
	if err := AdmitIntentBoundLaunch(planningSubject(accepted), project.ExecutionDecompose, project.RoleCoder, factory.IntentPlanningOptions, accepted, true, acceptedInput); err == nil {
		t.Fatal("accepted graph entered the pre-issue path")
	}
}

func TestIntentBoundResultAdmitsAttentionWithoutBatch(t *testing.T) {
	head := p4TrackerHead()
	in := p4PlannerInput(t, head, factory.IntentPlanningOptions)
	subject := planningSubject(head)
	attention := factory.SemanticPlannerOutput{Intent: head.Intent, Revision: head.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
	if err := AdmitIntentBoundResult(subject, project.ExecutionDecompose, in, attention); err != nil {
		t.Fatalf("attention refused at the result boundary: %v", err)
	}
}

func TestIntentBoundResultRefusesBadAnswersAndTransitions(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	subject := planningSubject(head)
	if err := AdmitIntentBoundResult(subject, project.ExecutionImplement, in, planningOutput(in)); !errors.Is(err, ErrIntentBoundTransition) {
		t.Fatalf("implement-phase result is not a gated transition: %v", err)
	}
	issue := factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 7, Acceptance: "d" + strings.Repeat("a", 24)}}
	if err := AdmitIntentBoundResult(issue, project.ExecutionDecompose, in, planningOutput(in)); !errors.Is(err, ErrIntentBoundTransition) {
		t.Fatalf("issue-subject result is not a gated transition: %v", err)
	}
	rebound := planningOutput(in)
	rebound.Operation = factory.IntentPlanningPropose
	if err := AdmitIntentBoundResult(subject, project.ExecutionDecompose, in, rebound); err == nil {
		t.Fatal("operation-mismatched answer admitted")
	}
	uncovered := planningOutput(in)
	uncovered.Candidates[0].Tasks = uncovered.Candidates[0].Tasks[:1]
	if err := AdmitIntentBoundResult(subject, project.ExecutionDecompose, in, uncovered); err == nil {
		t.Fatal("uncovered answer passed the result boundary")
	}
}

// D13 completion joints stay open: the completion subject is refused here
// and owned by FW30, never admitted as intent planning.
func TestIntentBoundJointsLeaveD13Open(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	completion := factory.ExecutionSubject{Kind: factory.SubjectCompletion, Completion: &factory.CompletionSubject{CompletionID: factory.NewID(), CycleID: factory.NewID(), InventoryDigest: strings.Repeat("e", 64)}}
	if err := AdmitIntentBoundLaunch(completion, project.ExecutionReassess, project.RoleCoder, factory.IntentPlanningOptions, head, true, in); !errors.Is(err, ErrIntentBoundTransition) {
		t.Fatalf("completion subject did not stay with FW30: %v", err)
	}
	if err := AdmitIntentBoundResult(completion, project.ExecutionReassess, in, planningOutput(in)); !errors.Is(err, ErrIntentBoundTransition) {
		t.Fatalf("completion result did not stay with FW30: %v", err)
	}
}

func intentBoundLaunchInput(prompt []byte) IntentBoundLaunchInput {
	return IntentBoundLaunchInput{
		Prompt: prompt, Deadline: time.Now().Add(time.Hour),
		RunID: strings.Repeat("a", 32), ProjectID: "p" + strings.Repeat("b", 24),
		PreparationID: "f" + strings.Repeat("c", 24),
		Harness:       project.FactoryHarnessCodex, HarnessVers: "1.2.3", Model: "test-model",
		PromptSHA: project.FactoryPromptDigest(prompt), SourceCommit: strings.Repeat("e", 40),
		Connection: "subscription", HarnessSHA256: strings.Repeat("d", 64), ActorID: 1001,
	}
}

// The staged launch rides the existing host path pinned to decompose with
// the coder role and no execution manifest: planning launches execute no
// code and mutate no native state.
func TestStageIntentBoundLaunchPinsDecomposeWithoutManifest(t *testing.T) {
	head := planningHead()
	subject := planningSubject(head)
	prompt := []byte("decompose the reviewed draft into candidate issue graphs")
	launch, err := StageIntentBoundLaunch(subject, intentBoundLaunchInput(prompt))
	if err != nil {
		t.Fatalf("admitted decompose launch refused: %v", err)
	}
	if launch.Run.ExecutionPhase != project.ExecutionDecompose || launch.Run.Role != project.RoleCoder {
		t.Fatalf("launch left the decompose/coder joint: %+v", launch.Run)
	}
	if len(launch.ExecutionManifest) != 0 || launch.Run.ManifestDigest != "" {
		t.Fatal("planning launch stages an execution manifest")
	}
	if err := launch.Validate(); err != nil {
		t.Fatalf("staged launch fails the host wire: %v", err)
	}
	issue := factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 7, Acceptance: "d" + strings.Repeat("a", 24)}}
	if _, err := StageIntentBoundLaunch(issue, intentBoundLaunchInput(prompt)); !errors.Is(err, ErrIntentBoundTransition) {
		t.Fatalf("non-intent launch staged as intent-bound: %v", err)
	}
	broken := intentBoundLaunchInput(prompt)
	broken.SourceCommit = "short"
	if _, err := StageIntentBoundLaunch(subject, broken); err == nil {
		t.Fatal("invalid launch identities staged for the host path")
	}
}

// TestStageIntentPlanningPromptPreservesCompleteInput proves the staged
// prompt carries the complete bounded typed input, not just the digest:
// a reviewed correction and a declined meaning absent from the
// objective, the authored alternatives for both operations, and the
// inspected sources all reach the host bytes.
func TestStageIntentPlanningPromptPreservesCompleteInput(t *testing.T) {
	head := factory.IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Repository: 7,
		Objective: "Decide vacation requests with manager review", State: "draft",
		MaxIssues: 10, CreatedUnix: 1792064400,
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333333", Description: "Capture leave requests", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
			{ID: "d444444444444444444444444", Description: "Sync to the legacy tracker", Consequence: "Legacy stays current", Decision: "edit", Correction: "Sync only on explicit manager request", Source: "requested", Provenance: "human", Actor: 7},
			{ID: "d555555555555555555555555", Description: "Delete requests automatically", Consequence: "Old requests vanish", Decision: "decline", Source: "requested", Provenance: "human", Actor: 7},
		},
	}
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatal(err)
	}
	set := factory.DescribeAlternatives(head.Objective)
	inspected := []factory.InspectedSource{
		{Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40), Evidence: "Draft head carries attributed effective scope with no tasks yet"},
	}
	for _, op := range []string{factory.IntentPlanningPropose, factory.IntentPlanningOptions} {
		in := factory.SemanticPlannerInput{
			Operation: op, Head: head, Content: content,
			Alternatives: set, Inspected: inspected,
		}
		prompt, err := StageIntentPlanningPrompt(head, in)
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if !set.Supported {
			t.Fatalf("objective unsupported by authored alternatives: %q", head.Objective)
		}
		text := string(prompt)
		for _, want := range []string{
			"Sync only on explicit manager request",
			"Delete requests automatically",
			"Old requests vanish",
			set.Pattern,
			"single_person",
			"Shared review workflow",
			"internal/factory/intent.go",
			head.Digest(),
			op,
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s prompt drops %q", op, want)
			}
		}
	}
}
