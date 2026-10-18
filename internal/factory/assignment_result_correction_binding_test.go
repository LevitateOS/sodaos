package factory

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestCorrectionResultBindsRetainedCandidate(t *testing.T) {
	for _, status := range []string{"completed", "blocked", "failed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			phased := PhaseResult{
				Version: PhaseResultVersion, Phase: project.ExecutionCorrect,
				Status: status, Summary: "correction outcome",
			}
			candidate := ""
			if status == "completed" {
				phased.Output = correctionOutput()
				candidate = phased.Output.Correction.NewCandidate
			}
			result := AssignmentResult{
				AssignmentID: NewID(), RunID: NewID(), RecordedUnix: 1700000000,
				Phase: project.ExecutionCorrect, PhaseResult: &phased,
				Status: status, Summary: phased.Summary, Candidate: candidate, Reported: true,
			}
			if err := result.ValidatePhaseBinding(project.RoleCoder, project.ExecutionCorrect); err != nil {
				t.Fatalf("matching correction refused: %v", err)
			}
			t.Run("substituted candidate", func(t *testing.T) {
				changed := result
				changed.Candidate = strings.Repeat("a", 40)
				if err := changed.ValidatePhaseBinding(project.RoleCoder, project.ExecutionCorrect); err == nil {
					t.Fatal("candidate absent from the typed correction was admitted")
				}
			})
			t.Run("substituted status", func(t *testing.T) {
				changed := result
				changed.Status = "completed"
				if status == "completed" {
					changed.Status = "failed"
				}
				if err := changed.ValidatePhaseBinding(project.RoleCoder, project.ExecutionCorrect); err == nil {
					t.Fatal("retained status differs from the typed correction")
				}
			})
			if status == "completed" {
				t.Run("missing candidate", func(t *testing.T) {
					changed := result
					changed.Candidate = ""
					if err := changed.ValidatePhaseBinding(project.RoleCoder, project.ExecutionCorrect); err == nil {
						t.Fatal("completed correction lost its nominated candidate")
					}
				})
			}
		})
	}
}
