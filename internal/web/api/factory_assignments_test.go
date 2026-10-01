package api

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestFactoryAssignmentDTORendersInputsAndResult(t *testing.T) {
	prompt := []byte("assignment prompt")
	a := factory.Assignment{
		ID: "a" + strings.Repeat("0", 31), ProjectID: "p" + strings.Repeat("1", 24), Role: project.RoleCoder,
		Repository: 7, Issue: 3, Acceptance: "d" + strings.Repeat("2", 24),
		Preparation: "f" + strings.Repeat("3", 24), Harness: "codex-1.2.3", HarnessVers: "1.2.3",
		Model: "m", Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: strings.Repeat("4", 64),
		Run: strings.Repeat("5", 32), RunHistory: []string{strings.Repeat("5", 32)},
		Stage: factory.AssignmentFinished, Outcome: factory.Succeeded, Reason: factory.AssignReasonReported,
		NativeRev: 41, Attempts: 1, CreatedUnix: 1, FinishedUnix: 2,
		Authority: factory.AuthorityRef{Policy: 1, Sponsorship: 2},
		Result: &factory.AssignmentResult{
			AssignmentID: "a" + strings.Repeat("0", 31), RunID: strings.Repeat("5", 32),
			Status: "completed", Summary: "fixed", Candidate: strings.Repeat("d", 40),
			Findings: []string{}, Reported: true, RecordedUnix: 2,
		},
	}
	view := factoryAssignmentDTO(a, &factory.Reservation{Connection: "conn", State: factory.ReservationConsumed, PlannedMinutes: 30})
	if view.Repository != "7" || view.Issue != "3" || view.Prompt != "assignment prompt" {
		t.Fatalf("view = %+v", view)
	}
	if view.Result == nil || view.Result.Candidate != strings.Repeat("d", 40) || !view.Result.Reported {
		t.Fatalf("result = %+v", view.Result)
	}
	if view.Reservation == nil || view.Reservation.State != factory.ReservationConsumed || view.Reservation.PlannedMinutes != 30 {
		t.Fatalf("reservation = %+v", view.Reservation)
	}
	if view.Authority.Sponsorship != 2 {
		t.Fatalf("authority = %+v", view.Authority)
	}
	plain := factoryAssignmentDTO(factory.Assignment{ID: a.ID, Repository: 7, Issue: 3}, nil)
	if plain.Result != nil || plain.Reservation != nil {
		t.Fatalf("plain = %+v", plain)
	}
}

func TestCurrentIssueAssignmentSelectsLatest(t *testing.T) {
	if _, ok := currentIssueAssignment(nil); ok {
		t.Fatal("empty selection accepted")
	}
	first := factory.Assignment{ID: factory.NewID(), Stage: factory.AssignmentFinished}
	second := factory.Assignment{ID: factory.NewID(), Stage: factory.AssignmentAssigned}
	got, ok := currentIssueAssignment([]factory.Assignment{first, second})
	if !ok || got.ID != second.ID {
		t.Fatalf("selected = %+v %v", got, ok)
	}
}
