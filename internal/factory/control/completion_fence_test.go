package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func fenceCandidate(repository, issue int64) factory.PriorityCandidate {
	return factory.PriorityCandidate{
		ID:      "cand",
		Control: factory.IssueControl{Repository: repository, Issue: issue},
	}
}

// A pending event excludes its own repository from selection while the
// rest still elect: no subset election inside the fence, no global
// stall outside it.
func TestSplitFencedCandidatesExcludesOnlyFencedRepos(t *testing.T) {
	candidates := []factory.PriorityCandidate{fenceCandidate(7, 31), fenceCandidate(7, 32), fenceCandidate(9, 41)}
	open, waits := splitFencedCandidates(candidates, map[int64]bool{7: true, 9: false})
	if len(open) != 1 || open[0].Control.Issue != 41 {
		t.Fatalf("open=%+v, want only repo 9", open)
	}
	if len(waits) != 2 {
		t.Fatalf("waits=%+v, want both fenced issues", waits)
	}
	for _, wait := range waits {
		if wait.Reason != WaitReassessmentPending || wait.Repository != 7 {
			t.Fatalf("wait=%+v, want reassessment_pending on repo 7", wait)
		}
	}
}

func TestSplitFencedCandidatesFullyFencedReportsWaitsOnly(t *testing.T) {
	candidates := []factory.PriorityCandidate{fenceCandidate(7, 31)}
	open, waits := splitFencedCandidates(candidates, map[int64]bool{7: true})
	if len(open) != 0 || len(waits) != 1 || waits[0].Reason != WaitReassessmentPending {
		t.Fatalf("open=%+v waits=%+v, want waits only", open, waits)
	}
}

func TestDispatchFencedMemoizesPerPass(t *testing.T) {
	calls := 0
	deps := DispatchDeps{
		FenceCache: map[int64]bool{},
		ReadCompletionFence: func(_ context.Context, repository int64) ([]int64, error) {
			calls++
			if repository != 7 {
				return nil, nil
			}
			return []int64{3}, nil
		},
	}
	for i := 0; i < 3; i++ {
		fenced, err := dispatchFenced(context.Background(), deps, 7)
		if err != nil || !fenced {
			t.Fatalf("fenced=%v err=%v, want fenced", fenced, err)
		}
	}
	if calls != 1 {
		t.Fatalf("fence reads=%d, want 1 memoized read", calls)
	}
	open, err := dispatchFenced(context.Background(), deps, 9)
	if err != nil || open {
		t.Fatalf("open=%v err=%v, want unfenced", open, err)
	}
}

func TestDispatchFencedReadsThroughWithoutCache(t *testing.T) {
	deps := DispatchDeps{
		ReadCompletionFence: func(context.Context, int64) ([]int64, error) { return nil, nil },
	}
	fenced, err := dispatchFenced(context.Background(), deps, 7)
	if err != nil || fenced {
		t.Fatalf("fenced=%v err=%v, want open", fenced, err)
	}
}

func TestAdmissionWaitMapsFenceToPending(t *testing.T) {
	wait := admissionWait(store.ErrCompletionFenced)
	if wait == nil || wait.Reason != WaitReassessmentPending {
		t.Fatalf("wait=%+v, want reassessment_pending", wait)
	}
}

func TestReportPacketRefusalMapsFenceToPending(t *testing.T) {
	report := newDispatchReport()
	reportPacketRefusal(context.Background(), DispatchDeps{}, nil, 7, 31, store.ErrCompletionFenced, &report)
	if len(report.Launched) != 0 || len(report.Errors) != 0 || len(report.Waits) != 1 {
		t.Fatalf("report=%+v, want one wait", report)
	}
	if report.Waits[0].Reason != WaitReassessmentPending {
		t.Fatalf("wait=%+v, want reassessment_pending", report.Waits[0])
	}
}

func TestCheckDispatchFenceWaitsBehindClosedFence(t *testing.T) {
	deps := DispatchDeps{
		FenceCache:          map[int64]bool{},
		ReadCompletionFence: func(context.Context, int64) ([]int64, error) { return []int64{1}, nil },
	}
	report := newDispatchReport()
	if !checkDispatchFence(context.Background(), deps, 7, 31, &report) {
		t.Fatal("open fence admitted dispatch")
	}
	if len(report.Waits) != 1 || report.Waits[0].Reason != WaitReassessmentPending {
		t.Fatalf("report=%+v, want reassessment_pending", report)
	}
}

func TestCheckDispatchFenceAdmitsThroughOpenFence(t *testing.T) {
	deps := DispatchDeps{
		FenceCache:          map[int64]bool{},
		ReadCompletionFence: func(context.Context, int64) ([]int64, error) { return nil, nil },
	}
	report := newDispatchReport()
	if checkDispatchFence(context.Background(), deps, 7, 31, &report) {
		t.Fatalf("closed fence refused dispatch: %+v", report)
	}
}

func TestCheckDispatchFenceUnreadableIsAnError(t *testing.T) {
	deps := DispatchDeps{
		ReadCompletionFence: func(context.Context, int64) ([]int64, error) {
			return nil, errors.New("fence unreadable")
		},
	}
	report := newDispatchReport()
	if !checkDispatchFence(context.Background(), deps, 7, 31, &report) {
		t.Fatal("unreadable fence admitted dispatch")
	}
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0].Detail, "fence unreadable") {
		t.Fatalf("report=%+v, want a fence error, never a launch", report)
	}
}
