package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func canonicalStallAssignment(t *testing.T) Assignment {
	t.Helper()
	prompt := testPrompt(t)
	a := testAssignment(prompt)
	a.RunHistory = []string{a.Run}
	a.Subject = ExecutionSubject{
		Kind: SubjectStall,
		Stall: &StallSubject{
			StallID: strings.Repeat("7", 32), CycleID: strings.Repeat("8", 32),
			InventoryDigest: strings.Repeat("9", 64),
		},
	}
	a.Phase, a.Role = project.ExecutionRepair, project.RoleCoder
	a.ParentAssignment, a.PublicationAssignment = "", ""
	a.Issue, a.Acceptance = 0, ""
	return a
}

func canonicalDeliveryAssignment(t *testing.T) Assignment {
	t.Helper()
	prompt := testPrompt(t)
	a := testAssignment(prompt)
	a.RunHistory = []string{a.Run}
	a.Subject = ExecutionSubject{
		Kind: SubjectDelivery,
		Delivery: &DeliverySubject{
			DeliveryID: "8" + strings.Repeat("1", 31), IntentID: "d" + strings.Repeat("b", 24),
			AcceptedRevision: "d" + strings.Repeat("c", 24), ScopeDigest: strings.Repeat("e", 64),
			SourceCommit: strings.Repeat("f", 40), PreparationDigest: strings.Repeat("a", 64),
			VerificationManifestDigest: strings.Repeat("b", 64),
		},
	}
	a.Phase, a.Role = project.ExecutionVerify, project.RoleReviewer
	a.ParentAssignment, a.PublicationAssignment = "", ""
	a.Issue, a.Acceptance = 0, ""
	return a
}

func canonicalDeliveryRepairAssignment(t *testing.T) Assignment {
	t.Helper()
	prompt := testPrompt(t)
	a := testAssignment(prompt)
	a.RunHistory = []string{a.Run}
	a.Subject = ExecutionSubject{
		Kind: SubjectDeliveryRepair,
		DeliveryRepair: &DeliveryRepairSubject{
			DeliveryID: "8" + strings.Repeat("1", 31), FailedVerificationRunID: strings.Repeat("2", 32),
			FailureDigest: strings.Repeat("3", 64), InventoryDigest: strings.Repeat("4", 64),
			CycleID: strings.Repeat("5", 32),
		},
	}
	a.Phase, a.Role = project.ExecutionRepair, project.RoleCoder
	a.ParentAssignment, a.PublicationAssignment = "", ""
	a.Issue, a.Acceptance = 0, ""
	return a
}

// D15/D16 scope: stall, delivery and delivery-repair cycles validate as
// fresh repository roots with no parent, publication or issue authority,
// and their subject keys name the canonical key spaces.
func TestRepairVerifySubjectPhaseValid(t *testing.T) {
	stall := canonicalStallAssignment(t)
	if err := stall.Validate(); err != nil {
		t.Fatalf("valid stall repair refused: %v", err)
	}
	if key := stall.Subject.SubjectKey(7); key != "stall:"+strings.Repeat("7", 32) {
		t.Fatalf("stall subject key: %s", key)
	}
	repair := canonicalDeliveryRepairAssignment(t)
	if err := repair.Validate(); err != nil {
		t.Fatalf("valid delivery repair refused: %v", err)
	}
	if key := repair.Subject.SubjectKey(7); key != "delivery_repair:"+"8"+strings.Repeat("1", 31)+":"+strings.Repeat("2", 32) {
		t.Fatalf("delivery repair subject key: %s", key)
	}
	delivery := canonicalDeliveryAssignment(t)
	if err := delivery.Validate(); err != nil {
		t.Fatalf("valid delivery verification refused: %v", err)
	}
	if key := delivery.Subject.SubjectKey(7); key != "delivery:"+"8"+strings.Repeat("1", 31) {
		t.Fatalf("delivery subject key: %s", key)
	}
}

func TestRepairVerifySubjectPhaseRefusals(t *testing.T) {
	accept := "d" + strings.Repeat("a", 24)
	cases := map[string]func() Assignment{
		"stall in verify": func() Assignment {
			a := canonicalStallAssignment(t)
			a.Phase, a.Role = project.ExecutionVerify, project.RoleReviewer
			return a
		},
		"delivery in repair": func() Assignment {
			a := canonicalDeliveryAssignment(t)
			a.Phase, a.Role = project.ExecutionRepair, project.RoleCoder
			return a
		},
		"delivery repair in verify": func() Assignment {
			a := canonicalDeliveryRepairAssignment(t)
			a.Phase, a.Role = project.ExecutionVerify, project.RoleReviewer
			return a
		},
		"stall in implement": func() Assignment {
			a := canonicalStallAssignment(t)
			a.Phase = project.ExecutionImplement
			a.PublicationAssignment = NewID()
			return a
		},
		"repair reviewer": func() Assignment {
			a := canonicalStallAssignment(t)
			a.Role = project.RoleReviewer
			return a
		},
		"verify coder": func() Assignment {
			a := canonicalDeliveryAssignment(t)
			a.Role = project.RoleCoder
			return a
		},
		"stall with parent": func() Assignment {
			a := canonicalStallAssignment(t)
			a.ParentAssignment = NewID()
			return a
		},
		"delivery with publication": func() Assignment {
			a := canonicalDeliveryAssignment(t)
			a.PublicationAssignment = NewID()
			return a
		},
		"repair with issue": func() Assignment {
			a := canonicalDeliveryRepairAssignment(t)
			a.Issue, a.Acceptance = 3, accept
			return a
		},
		"stall dual payload": func() Assignment {
			a := canonicalStallAssignment(t)
			a.Subject.Delivery = &DeliverySubject{DeliveryID: "8" + strings.Repeat("1", 31)}
			return a
		},
		"stall bad digest": func() Assignment {
			a := canonicalStallAssignment(t)
			a.Subject.Stall.InventoryDigest = "short"
			return a
		},
		"delivery bad manifest": func() Assignment {
			a := canonicalDeliveryAssignment(t)
			a.Subject.Delivery.VerificationManifestDigest = "short"
			return a
		},
		"delivery repair bad run": func() Assignment {
			a := canonicalDeliveryRepairAssignment(t)
			a.Subject.DeliveryRepair.FailedVerificationRunID = "short"
			return a
		},
	}
	for name, build := range cases {
		if err := build().Validate(); err == nil {
			t.Fatalf("subject case %s admitted", name)
		}
	}
}

// D15/D16 scope: the 320 KiB prompt exception binds the repair phase to
// its stall or delivery-repair subject; verify keeps the standard bound.
func TestRepairPromptBound(t *testing.T) {
	oversize := func(a Assignment, size int) Assignment {
		a.Prompt = []byte(strings.Repeat("p", size))
		sum := sha256.Sum256(a.Prompt)
		a.PromptSHA = hex.EncodeToString(sum[:])
		return a
	}
	if err := oversize(canonicalStallAssignment(t), project.MaxReassessPrompt).Validate(); err != nil {
		t.Fatalf("bounded stall repair prompt refused: %v", err)
	}
	if err := oversize(canonicalDeliveryRepairAssignment(t), project.MaxReassessPrompt).Validate(); err != nil {
		t.Fatalf("bounded delivery repair prompt refused: %v", err)
	}
	for name, a := range map[string]Assignment{
		"repair over bound":      oversize(canonicalStallAssignment(t), project.MaxReassessPrompt+1),
		"verify at repair bound": oversize(canonicalDeliveryAssignment(t), project.MaxReassessPrompt),
	} {
		if err := a.Validate(); err == nil {
			t.Fatalf("prompt case %s admitted", name)
		}
	}
	mismatch := canonicalStallAssignment(t)
	mismatch.Subject = ExecutionSubject{Kind: SubjectDelivery, Delivery: &DeliverySubject{
		DeliveryID: "8" + strings.Repeat("1", 31), IntentID: "d" + strings.Repeat("b", 24),
		AcceptedRevision: "d" + strings.Repeat("c", 24), ScopeDigest: strings.Repeat("e", 64),
		SourceCommit: strings.Repeat("f", 40), PreparationDigest: strings.Repeat("a", 64),
		VerificationManifestDigest: strings.Repeat("b", 64),
	}}
	mismatch = oversize(mismatch, project.MaxFactoryPrompt+1)
	if err := mismatch.Validate(); err == nil {
		t.Fatal("repair-sized delivery prompt admitted")
	}
}

func TestRepairOutcomeVocabulary(t *testing.T) {
	for _, outcome := range []string{RepairOutcomeRepair, RepairOutcomeVerifiedWait, RepairOutcomeNoChange, RepairOutcomeIncomplete} {
		if !ValidRepairOutcome(outcome) {
			t.Fatalf("admitted repair outcome refused: %s", outcome)
		}
	}
	if ValidRepairOutcome("merged") || ValidRepairOutcome("") {
		t.Fatal("unknown repair outcome admitted")
	}
	for _, outcome := range []string{AppVerificationVerified, AppVerificationFailed, AppVerificationIncomplete} {
		if !ValidAppVerificationOutcome(outcome) {
			t.Fatalf("admitted verification outcome refused: %s", outcome)
		}
	}
	if ValidAppVerificationOutcome(RepairOutcomeRepair) || ValidAppVerificationOutcome("") {
		t.Fatal("unknown verification outcome admitted")
	}
}

func TestRepairReportValidation(t *testing.T) {
	goodStall := StallRepairReport{
		StallID: strings.Repeat("7", 32), CycleID: strings.Repeat("8", 32),
		InventoryDigest: strings.Repeat("9", 64), Outcome: RepairOutcomeRepair,
	}
	if err := goodStall.Validate(); err != nil {
		t.Fatalf("valid stall repair report refused: %v", err)
	}
	goodDeliveryRepair := DeliveryRepairReport{
		DeliveryID: strings.Repeat("7", 32), FailedVerificationRunID: strings.Repeat("8", 32),
		InputDigest: strings.Repeat("9", 64), InventoryDigest: strings.Repeat("a", 64),
		CycleID: strings.Repeat("b", 32), Outcome: RepairOutcomeNoChange,
	}
	if err := goodDeliveryRepair.Validate(); err != nil {
		t.Fatalf("valid delivery repair report refused: %v", err)
	}
	goodVerification := AppVerificationReport{
		DeliveryID: strings.Repeat("7", 32), InputDigest: strings.Repeat("9", 64),
		Outcome: AppVerificationFailed,
	}
	if err := goodVerification.Validate(); err != nil {
		t.Fatalf("valid app verification report refused: %v", err)
	}
	badStall := goodStall
	badStall.Outcome = "merged"
	badDeliveryRepair := goodDeliveryRepair
	badDeliveryRepair.FailedVerificationRunID = "short"
	badVerification := goodVerification
	badVerification.InputDigest = "short"
	for name, err := range map[string]error{
		"stall bad outcome":       badStall.Validate(),
		"delivery repair bad run": badDeliveryRepair.Validate(),
		"verification bad digest": badVerification.Validate(),
	} {
		if err == nil {
			t.Fatalf("report case %s admitted", name)
		}
	}
}
