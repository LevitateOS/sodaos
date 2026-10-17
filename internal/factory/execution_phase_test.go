package factory

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

// T03 (FW06A scope): every valid phase/role combination binds, every other
// refuses. Only review uses the reviewer account; phases are never inferred
// from roles.
func TestPhaseRoleBindingMatrix(t *testing.T) {
	valid := map[string]string{
		project.ExecutionDecompose:   project.RoleCoder,
		project.ExecutionInvestigate: project.RoleCoder,
		project.ExecutionImplement:   project.RoleCoder,
		project.ExecutionReview:      project.RoleReviewer,
		project.ExecutionCorrect:     project.RoleCoder,
		project.ExecutionReassess:    project.RoleCoder,
		project.ExecutionRepair:      project.RoleCoder,
		project.ExecutionVerify:      project.RoleReviewer,
	}
	for phase, role := range valid {
		if !ValidPhaseRole(phase, role) {
			t.Fatalf("valid phase/role refused: %s/%s", phase, role)
		}
		if got, err := PhaseRole(phase); err != nil || got != role {
			t.Fatalf("phase role mismatch: %s -> %s, %v", phase, got, err)
		}
	}
	invalid := [][2]string{
		{project.ExecutionDecompose, project.RoleReviewer},
		{project.ExecutionInvestigate, project.RoleReviewer},
		{project.ExecutionImplement, project.RoleReviewer},
		{project.ExecutionReview, project.RoleCoder},
		{project.ExecutionCorrect, project.RoleReviewer},
		{project.ExecutionReassess, project.RoleReviewer},
		{project.ExecutionRepair, project.RoleReviewer},
		{project.ExecutionVerify, project.RoleCoder},
		{"publish", project.RoleCoder},
		{"", project.RoleCoder},
	}
	for _, tc := range invalid {
		if ValidPhaseRole(tc[0], tc[1]) {
			t.Fatalf("invalid phase/role admitted: %s/%s", tc[0], tc[1])
		}
	}
	if _, err := PhaseRole("publish"); err == nil {
		t.Fatal("unknown phase resolved a role")
	}
}

// T03 (FW06A scope): only decompose accepts an intent subject; issue
// subjects never decompose or reassess; completion subjects enter only
// through reassess, which accepts nothing else.
func TestSubjectKindPhaseMatrix(t *testing.T) {
	for _, phase := range []string{project.ExecutionInvestigate, project.ExecutionImplement, project.ExecutionReview, project.ExecutionCorrect} {
		if !SubjectKindAllowed(SubjectIssue, phase) {
			t.Fatalf("issue subject refused for %s", phase)
		}
		if SubjectKindAllowed(SubjectIntent, phase) {
			t.Fatalf("intent subject admitted for %s", phase)
		}
		if SubjectKindAllowed(SubjectCompletion, phase) {
			t.Fatalf("completion subject admitted for %s", phase)
		}
	}
	if !SubjectKindAllowed(SubjectIntent, project.ExecutionDecompose) {
		t.Fatal("intent subject refused for decompose")
	}
	if SubjectKindAllowed(SubjectIssue, project.ExecutionDecompose) {
		t.Fatal("issue subject admitted for decompose")
	}
	if !SubjectKindAllowed(SubjectCompletion, project.ExecutionReassess) {
		t.Fatal("completion subject refused for reassess")
	}
	for _, kind := range []string{SubjectIssue, SubjectIntent} {
		if SubjectKindAllowed(kind, project.ExecutionReassess) {
			t.Fatalf("subject %s admitted for reassess", kind)
		}
	}
	if SubjectKindAllowed(SubjectCompletion, project.ExecutionDecompose) {
		t.Fatal("completion subject admitted for decompose")
	}
	// D15/D16 scope: repair admits only stall and delivery repair;
	// verify admits only delivery; nothing else enters either phase.
	for _, kind := range []string{SubjectStall, SubjectDeliveryRepair} {
		if !SubjectKindAllowed(kind, project.ExecutionRepair) {
			t.Fatalf("subject %s refused for repair", kind)
		}
		if SubjectKindAllowed(kind, project.ExecutionVerify) {
			t.Fatalf("subject %s admitted for verify", kind)
		}
	}
	if !SubjectKindAllowed(SubjectDelivery, project.ExecutionVerify) {
		t.Fatal("delivery subject refused for verify")
	}
	if SubjectKindAllowed(SubjectDelivery, project.ExecutionRepair) {
		t.Fatal("delivery subject admitted for repair")
	}
	for _, kind := range []string{SubjectIssue, SubjectIntent, SubjectCompletion} {
		if SubjectKindAllowed(kind, project.ExecutionRepair) {
			t.Fatalf("subject %s admitted for repair", kind)
		}
		if SubjectKindAllowed(kind, project.ExecutionVerify) {
			t.Fatalf("subject %s admitted for verify", kind)
		}
	}
	for _, phase := range []string{project.ExecutionInvestigate, project.ExecutionImplement, project.ExecutionReview, project.ExecutionCorrect} {
		for _, kind := range []string{SubjectStall, SubjectDelivery, SubjectDeliveryRepair} {
			if SubjectKindAllowed(kind, phase) {
				t.Fatalf("subject %s admitted for %s", kind, phase)
			}
		}
	}
	if SubjectKindAllowed("draft", project.ExecutionDecompose) || SubjectKindAllowed(SubjectIssue, "") {
		t.Fatal("unknown kind or empty phase admitted")
	}
}

// T03 (FW06A scope): automatic same-parent advancement is idempotent — the
// same completed parent replays to the same child identity — and the
// (parent, digest, phase) triple rejects garbage.
func TestAdvancementIdentityIsReplaySafe(t *testing.T) {
	parent := NewID()
	digest := strings.Repeat("a", 64)
	first, err := AdvancementID(parent, digest, project.ExecutionImplement)
	if err != nil || !ValidID(first) {
		t.Fatalf("advancement refused: %v %v", first, err)
	}
	second, err := AdvancementID(parent, digest, project.ExecutionImplement)
	if err != nil || second != first {
		t.Fatal("advancement replay diverged")
	}
	other, err := AdvancementID(parent, digest, project.ExecutionReview)
	if err != nil || other == first {
		t.Fatal("distinct next phase shared advancement identity")
	}
	for _, tc := range [][3]string{
		{"short", digest, project.ExecutionImplement},
		{parent, "short", project.ExecutionImplement},
		{parent, strings.Repeat("z", 64), project.ExecutionImplement},
		{parent, digest, "publish"},
		{parent, digest, ""},
	} {
		if _, err := AdvancementID(tc[0], tc[1], tc[2]); err == nil {
			t.Fatalf("invalid advancement admitted: %v", tc)
		}
	}
}

func TestExecutionProfileVocabularyAndMuseBinding(t *testing.T) {
	for _, effort := range []string{EffortLow, EffortMedium, EffortHigh, EffortXHigh} {
		if !ValidEffort(effort) {
			t.Fatalf("admitted effort refused: %s", effort)
		}
	}
	if ValidEffort("") || ValidEffort("turbo") {
		t.Fatal("unknown effort admitted")
	}
	crew := ExecutionProfile{Effort: EffortXHigh, Contributor: true}
	if err := crew.Validate(project.FactoryHarnessMuse); err != nil {
		t.Fatalf("qualified crew profile refused: %v", err)
	}
	if err := crew.Validate(project.FactoryHarnessCodex); err == nil {
		t.Fatal("contributor mode admitted for Codex")
	}
	if err := (ExecutionProfile{Effort: "turbo"}).Validate(project.FactoryHarnessMuse); err == nil {
		t.Fatal("unknown effort profile validated")
	}
	if got := EffectiveExecutionProfile(nil); got != (ExecutionProfile{Effort: EffortLow}) {
		t.Fatalf("profile defaults: %+v", got)
	}
	omitted := &ExecutionProfile{}
	if got := EffectiveExecutionProfile(omitted); got.Effort != EffortLow || got.Contributor {
		t.Fatalf("omitted profile misresolved: %+v", got)
	}
}
