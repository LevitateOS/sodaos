package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// F7 consumer joint, deterministic legs: verdict resolution admits exactly
// one source per assignment, the typed digest separates v2 rounds from v1,
// and the native prose renders deterministically. Store-backed binding
// (head/base/cycle/reviewer/carried) is proven in the submit tests below.

func phasedVerdictAssignment(phase string, output *factory.PhaseOutput, legacy *factory.ReviewReport) factory.Assignment {
	return factory.Assignment{
		Role: project.RoleReviewer, Phase: phase,
		Result: &factory.AssignmentResult{
			Status: "completed", Reported: true, Summary: "verdict recorded",
			PhaseResult: &factory.PhaseResult{
				Version: 2, Phase: phase, Status: "completed",
				Summary: "verdict recorded", Output: output,
			},
			Review: legacy,
		},
	}
}

func phasedCycleVerdict() *factory.ReviewCycleReport {
	return &factory.ReviewCycleReport{
		CycleID: strings.Repeat("c", 32),
		NewFindings: []factory.Finding{{
			Key: "00000000000000000000000000000000", OriginRun: strings.Repeat("1", 32),
			Severity: factory.FindingSeverityMajor, Defect: "missing bound",
			Evidence: "line 42 lacks a check", RequiredOutcome: "add the check",
			FixKind: factory.FindingFixCode, Actionable: true,
		}},
	}
}

func phasedCycleClear(head, base string) *factory.ClearReviewReport {
	return &factory.ClearReviewReport{
		CycleID: strings.Repeat("c", 32), Conclusion: factory.ClearReviewConclusion,
		Head: head, BaseOID: base, CoverageDigest: strings.Repeat("b", 64),
	}
}

func TestResolveAssignmentVerdictSources(t *testing.T) {
	legacy := &factory.ReviewReport{Verdict: "approve", Summary: "clean", Body: "checked", Findings: []string{}}
	typed := &factory.PhaseOutput{Review: phasedCycleVerdict()}
	clearOut := &factory.PhaseOutput{Clear: phasedCycleClear(strings.Repeat("e", 40), strings.Repeat("f", 40))}
	legacyAssignment := factory.Assignment{
		Result: &factory.AssignmentResult{Status: "completed", Reported: true, Summary: "clean", Review: legacy},
	}
	resolved, err := resolveAssignmentVerdict(legacyAssignment)
	if err != nil || resolved.Verdict != "approve" || resolved.Legacy == nil || resolved.Review != nil {
		t.Fatalf("legacy verdict not resolved: %+v %v", resolved, err)
	}
	resolved, err = resolveAssignmentVerdict(phasedVerdictAssignment(project.ExecutionReview, typed, nil))
	if err != nil || resolved.Verdict != "request-changes" || resolved.Review == nil || resolved.Legacy != nil {
		t.Fatalf("typed findings verdict not resolved: %+v %v", resolved, err)
	}
	resolved, err = resolveAssignmentVerdict(phasedVerdictAssignment(project.ExecutionReview, clearOut, nil))
	if err != nil || resolved.Verdict != "approve" || resolved.Clear == nil || resolved.Legacy != nil {
		t.Fatalf("typed clear not resolved: %+v %v", resolved, err)
	}
	for name, assignment := range map[string]factory.Assignment{
		"missing report":      {},
		"both sources":        phasedVerdictAssignment(project.ExecutionReview, typed, legacy),
		"neither source":      phasedVerdictAssignment(project.ExecutionReview, &factory.PhaseOutput{}, nil),
		"non-review phase":    phasedVerdictAssignment(project.ExecutionImplement, typed, nil),
		"legacy without rows": {Result: &factory.AssignmentResult{Status: "completed", Reported: true}},
	} {
		if _, err := resolveAssignmentVerdict(assignment); err == nil {
			t.Fatalf("%s resolved", name)
		}
	}
}

func TestRenderTypedReviewBodyDeterministic(t *testing.T) {
	head, base := strings.Repeat("e", 40), strings.Repeat("f", 40)
	review := ResolvedReviewVerdict{Verdict: "request-changes", Review: phasedCycleVerdict()}
	first := renderTypedReviewBody(head, base, review)
	if !strings.Contains(first, head) || !strings.Contains(first, base) ||
		!strings.Contains(first, "missing bound") || !strings.Contains(first, "line 42 lacks a check") {
		t.Fatalf("review body drops evidence: %q", first)
	}
	if second := renderTypedReviewBody(head, base, review); second != first {
		t.Fatal("review body replay diverged")
	}
	clear := ResolvedReviewVerdict{Verdict: "approve", Clear: phasedCycleClear(head, base)}
	body := renderTypedReviewBody(head, base, clear)
	if !strings.Contains(body, "Explicit clear") || !strings.Contains(body, head) {
		t.Fatalf("clear body drops identity: %q", body)
	}
}

// bindPhasedFixture crafts the exact values the future phased dispatch
// records: a first-round cycle bound to one reviewer assignment on the
// current head. The dispatch packet admits no phased review rows yet (the
// store refuses them for their owning boundary), so these tests prove the
// consumer binding directly — the same checks SubmitReviewForRun and
// currentReviewReport run before any native call or routing.
func bindPhasedFixture(head, base string, reviewerID string) (factory.Publication, factory.ReviewCycle, factory.Assignment) {
	publicationID := strings.Repeat("a", 32)
	attemptRoot := strings.Repeat("b", 32)
	p := factory.Publication{
		AssignmentID: strings.Repeat("1", 32),
		Candidate:    head,
		PRCreate:     factory.PublicationOperation{BaseOID: base},
	}
	cycle := factory.ReviewCycle{
		ID:            factory.ReviewCycleID(publicationID, attemptRoot, 1, "review"),
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		Round: 1, Phase: "review",
		TriggerDecision: "initial-review", TriggerReason: "first typed review",
		Head: head, BaseOID: base,
		FindingDigest:      reviewCycleInputDigest(head, base, nil, nil),
		ReviewerAssignment: reviewerID,
	}
	p.ID = publicationID
	p.ReviewCycles = []factory.ReviewCycle{cycle}
	assignment := factory.Assignment{
		ID: reviewerID, Role: project.RoleReviewer, SourceCommit: head,
		Phase: project.ExecutionReview,
		Result: &factory.AssignmentResult{
			Status: "completed", Reported: true, Summary: "verdict recorded",
		},
	}
	return p, cycle, assignment
}

func bindPhasedOutput(cycleID string) *factory.PhaseOutput {
	return &factory.PhaseOutput{Review: &factory.ReviewCycleReport{
		CycleID: cycleID, NewFindings: []factory.Finding{phasedFinding(0, true)},
	}}
}

// The binding admits the exact tuple only: current reviewer assignment,
// exact candidate and base, recorded cycle bound to this reviewer, and
// the complete carried findings. Each failure refuses with no event.
func TestBindPhasedReviewVerdict(t *testing.T) {
	head, base := strings.Repeat("e", 40), strings.Repeat("f", 40)
	reviewer := strings.Repeat("4", 32)
	coord := &Coordinator{}
	bind := func(p factory.Publication, a factory.Assignment, out *factory.PhaseOutput) (string, string, error) {
		resolved, err := resolveAssignmentVerdict(withPhasedResult(a, out))
		if err != nil {
			return "", "", err
		}
		return coord.bindPhasedReviewVerdict(context.Background(), p, a, head, resolved)
	}
	p, cycle, a := bindPhasedFixture(head, base, reviewer)
	event, body, err := bind(p, a, bindPhasedOutput(cycle.ID))
	if err != nil || event != "REQUEST_CHANGES" {
		t.Fatalf("bound verdict: event=%q err=%v", event, err)
	}
	if !strings.Contains(body, "line 42 lacks a check") || !strings.Contains(body, cycle.ID) {
		t.Fatalf("bound body drops evidence: %q", body)
	}
	clearVerdict := phasedCycleClear(head, base)
	clearVerdict.CycleID = cycle.ID
	clearOut := &factory.PhaseOutput{Clear: clearVerdict}
	event, _, err = bind(p, a, clearOut)
	if err != nil || event != "APPROVED" {
		t.Fatalf("bound clear: event=%q err=%v", event, err)
	}
	for name, mutate := range map[string]func(*factory.Publication, *factory.Assignment, *factory.PhaseOutput){
		"wrong base": func(p *factory.Publication, a *factory.Assignment, out *factory.PhaseOutput) {
			clear := phasedCycleClear(head, strings.Repeat("3", 40))
			clear.CycleID = out.Review.CycleID
			out.Clear = clear
			out.Review = nil
		},
		"unrecorded cycle": func(p *factory.Publication, a *factory.Assignment, out *factory.PhaseOutput) {
			out.Review.CycleID = strings.Repeat("d", 32)
		},
		"wrong reviewer": func(p *factory.Publication, a *factory.Assignment, out *factory.PhaseOutput) {
			a.ID = strings.Repeat("7", 32)
		},
		"missing carried assessment": func(p *factory.Publication, a *factory.Assignment, out *factory.PhaseOutput) {
			prior := phasedFinding(5, true)
			digest, err := factory.FindingSetDigest([]string{prior.Key})
			if err != nil {
				panic(err)
			}
			p.CorrectionJournals = []factory.CorrectionJournal{{
				CycleID: out.Review.CycleID, RunID: strings.Repeat("9", 32),
				Report: factory.CorrectionReport{
					SetDigest: digest, Findings: []factory.Finding{prior},
					Dispositions: []factory.CorrectionDisposition{{Key: prior.Key, Disposition: factory.DispositionFixed}},
				},
			}}
		},
	} {
		p, _, a := bindPhasedFixture(head, base, reviewer)
		out := bindPhasedOutput(p.ReviewCycles[0].ID)
		mutate(&p, &a, out)
		if _, _, err := bind(p, a, out); err == nil {
			t.Fatalf("%s bound", name)
		}
	}
}

// A same-head prior round that resolved legacy refuses the typed chain:
// string defects carry no stable key, so coverage cannot be proven and
// the legacy consumer keeps the chain. Rounds on superseded heads are
// history and never carried.
func TestBindPhasedReviewVerdictRefusesLegacyPrior(t *testing.T) {
	fx, owner, p, run := reviewRunSeed(t, true)
	prev := accountChildResult(t, fx, run, "notes\n```"+factory.ReviewReportFence+"\n"+`{"verdict":"request-changes","summary":"old","body":"old defect","findings":["path:1 evidence outcome"]}`+"\n```\n")
	head, base := p.Candidate, p.PRCreate.BaseOID
	stored, err := fx.db.PublicationByAssignment(context.Background(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	first := factory.ReviewCycle{
		ID:            factory.ReviewCycleID(stored.ID, owner.AttemptRoot, 1, "review"),
		PublicationID: stored.ID, AttemptRoot: owner.AttemptRoot,
		Round: 1, Phase: "review",
		TriggerDecision: "initial-review", TriggerReason: "legacy first round",
		Head: head, BaseOID: base,
		FindingDigest:      reviewCycleInputDigest(head, base, nil, nil),
		ReviewerAssignment: prev.ID,
	}
	second := first
	second.ID = factory.ReviewCycleID(stored.ID, owner.AttemptRoot, 2, "review")
	second.Round = 2
	second.TriggerDecision, second.TriggerReason = "re-review-same-head", "typed answer to legacy round"
	second.ReviewerAssignment = strings.Repeat("4", 32)
	stored.ReviewCycles = []factory.ReviewCycle{first, second}
	a := factory.Assignment{
		ID: second.ReviewerAssignment, Role: project.RoleReviewer, SourceCommit: head,
		Phase:  project.ExecutionReview,
		Result: &factory.AssignmentResult{Status: "completed", Reported: true, Summary: "verdict recorded"},
	}
	out := bindPhasedOutput(second.ID)
	resolved, err := resolveAssignmentVerdict(withPhasedResult(a, out))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := fx.coord.bindPhasedReviewVerdict(context.Background(), stored, a, head, resolved); err == nil {
		t.Fatal("typed verdict answered legacy strings")
	}
}

// withPhasedResult attaches one phased output to a crafted assignment
// result for consumer tests.
func withPhasedResult(a factory.Assignment, out *factory.PhaseOutput) factory.Assignment {
	a.Result.PhaseResult = &factory.PhaseResult{
		Version: 2, Phase: project.ExecutionReview, Status: "completed",
		Summary: "verdict recorded", Output: out,
	}
	return a
}

func TestReviewCycleInputDigestTypedSeparatesRounds(t *testing.T) {
	head, base := strings.Repeat("e", 40), strings.Repeat("f", 40)
	typed := reviewCycleInputDigestTyped(head, base, phasedCycleVerdict(), nil, nil)
	if !factory.ValidDigest(typed) {
		t.Fatalf("typed digest malformed: %q", typed)
	}
	if legacy := reviewCycleInputDigest(head, base, nil, nil); legacy == typed {
		t.Fatal("typed round collides with the legacy domain")
	}
	moved := *phasedCycleVerdict()
	moved.NewFindings[0].Evidence = "line 43 lacks a check"
	if changed := reviewCycleInputDigestTyped(head, base, &moved, nil, nil); changed == typed {
		t.Fatal("changed findings kept their round")
	}
	reordered := reviewCycleInputDigestTyped(head, base, phasedCycleVerdict(), nil, nil)
	if reordered != typed {
		t.Fatal("identical verdict replayed a new round")
	}
	clear := reviewCycleInputDigestTyped(head, base, nil, phasedCycleClear(head, base), nil)
	if clear == typed || !factory.ValidDigest(clear) {
		t.Fatal("clear verdict shares its round")
	}
}
