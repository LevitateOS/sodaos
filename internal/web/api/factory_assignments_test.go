package api

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestFactoryAssignmentDTORendersInputsAndResult(t *testing.T) {
	prompt := []byte("assignment prompt")
	assignmentID := "a" + strings.Repeat("0", 31)
	a := factory.Assignment{
		ID: assignmentID, AttemptRoot: assignmentID, PublicationAssignment: assignmentID,
		ProjectID: "p" + strings.Repeat("1", 24), Role: project.RoleCoder,
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
	view := factoryAssignmentDTO(a, &factory.Reservation{Connection: "conn", State: factory.ReservationConsumed, PlannedMinutes: 30}, &factory.Publication{
		Publish:  factory.PublicationOperation{Kind: factory.OpRefPublish, Effect: factory.OpEffectCommitted},
		PRCreate: factory.PublicationOperation{Kind: factory.OpPRCreate, Effect: factory.OpEffectCommitted},
		Stage:    factory.PublicationPublished, Reason: factory.PublishReasonLinked, PRNumber: 9, PRID: 8,
	})
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
	if view.Publication == nil || view.Publication.Stage != factory.PublicationPublished || view.Publication.PRNumber != "9" {
		t.Fatalf("publication = %+v", view.Publication)
	}
	plain := factoryAssignmentDTO(factory.Assignment{ID: a.ID, Repository: 7, Issue: 3}, nil, nil)
	if plain.Result != nil || plain.Reservation != nil || plain.Publication != nil {
		t.Fatalf("plain = %+v", plain)
	}
}

func TestFactoryAssignmentDTOPreservesIncompletePublication(t *testing.T) {
	view := factoryAssignmentDTO(factory.Assignment{}, nil, &factory.Publication{
		Publish:  factory.PublicationOperation{Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete, Cancellation: factory.OpCancelTooLate},
		PRCreate: factory.PublicationOperation{Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionNeedsIntervention, Cancellation: factory.OpCancelTooLate},
		Stage:    factory.PublicationFenced, WithdrawRequested: true, PRNumber: 9, PRID: 8,
	})
	p := view.Publication
	if p == nil || p.PRNumber != "9" || p.PublishCompletion != factory.OpCompletionComplete || p.PRCreateCompletion != factory.OpCompletionNeedsIntervention || p.PRCreateCancellation != factory.OpCancelTooLate || !p.WithdrawRequested {
		t.Fatalf("incomplete publication hidden: %+v", p)
	}
}
