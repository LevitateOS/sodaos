package control

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// Without a store the drive is a no-op: it never seals, decides or
// releases from missing state.
func TestDriveCompletionsWithoutStoreIsEmpty(t *testing.T) {
	coord := &Coordinator{}
	report := coord.DriveCompletions(context.Background(), CompletionDriveDeps{})
	if len(report.Sealed) != 0 || len(report.Decided) != 0 || len(report.Released) != 0 ||
		len(report.AwaitingCoder) != 0 || len(report.Held) != 0 {
		t.Fatalf("report=%+v, want no work without a store", report)
	}
}

// Retry without a store cannot reconcile: it refuses instead of
// inventing a successor.
func TestStartCompletionRetryWithoutStoreRefuses(t *testing.T) {
	coord := &Coordinator{}
	retry := factory.ReassessmentRetry{
		SuccessorCycleID: factory.NewID(), ExpectedPriorCycle: factory.NewID(),
		CommandID: factory.NewID(), EventID: factory.NewID(),
	}
	if _, err := coord.StartCompletionRetry(context.Background(), retry); err == nil {
		t.Fatal("retry admitted without reconciled state")
	}
}

// Attention and superseded cycles rest for explicit Retry: the drive
// never resumes them and never starts an automatic re-run.
func TestRouteCompletionCycleNeverAutoRetries(t *testing.T) {
	if routeCompletionCycle(factory.ReassessmentAttention) != driveHold {
		t.Fatal("attention routed for automatic driving")
	}
	if routeCompletionCycle(factory.ReassessmentSuperseded) != driveHold {
		t.Fatal("superseded routed for automatic driving")
	}
	if routeCompletionCycle(factory.ReassessmentReleased) != driveHold {
		t.Fatal("released routed for automatic driving")
	}
	if routeCompletionCycle(factory.ReassessmentCollecting) != driveCollect {
		t.Fatal("collecting not routed to seal")
	}
	if routeCompletionCycle(factory.ReassessmentDeciding) != driveDecide {
		t.Fatal("deciding not routed to decide")
	}
	for _, state := range []string{factory.ReassessmentReassessing, factory.ReassessmentAssessing, factory.ReassessmentApplying} {
		if routeCompletionCycle(state) != driveCoder {
			t.Fatalf("%s not held for the FW32 coder admission", state)
		}
	}
}
