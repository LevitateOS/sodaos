package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestWorkClosureStates(t *testing.T) {
	if got := workClosureDTO(nil, nil); got.State != workClosureNone {
		t.Fatalf("empty closure: %+v", got)
	}
	pending := workClosureDTO(&factory.Publication{PRNumber: 12}, nil)
	if pending.State != workClosurePending {
		t.Fatalf("publication closure: %+v", pending)
	}
	pendingMerge := workClosureDTO(nil, &factory.Merge{Stage: factory.MergeOpen, PRNumber: 12})
	if pendingMerge.State != workClosurePending {
		t.Fatalf("merge closure: %+v", pendingMerge)
	}
	confirmed := workClosureDTO(
		&factory.Publication{PRNumber: 12},
		&factory.Merge{Stage: factory.MergeMerged, PRNumber: 12, OriginalClosingOccurrence: "34", OriginalClosedUnix: 42},
	)
	if confirmed.State != workClosureConfirmed || confirmed.Occurrence != "34" || confirmed.ClosedUnix != 42 {
		t.Fatalf("confirmed closure: %+v", confirmed)
	}
	unconfirmed := workClosureDTO(nil, &factory.Merge{Stage: factory.MergeMerged, PRNumber: 12})
	if unconfirmed.State != workClosurePending {
		t.Fatalf("merge without closure proof: %+v", unconfirmed)
	}
}

func TestWorkCandidateHistory(t *testing.T) {
	p := factory.Publication{
		Candidate: strings.Repeat("a", 40),
		Publish:   factory.PublicationOperation{Work: &factory.PublicationIntent{Candidate: strings.Repeat("b", 40)}},
		Corrections: []factory.PublicationOperation{
			{Effect: factory.OpEffectCommitted, Work: &factory.PublicationIntent{Candidate: strings.Repeat("c", 40)}},
			{Effect: factory.OpEffectNotCommitted, Work: &factory.PublicationIntent{Candidate: strings.Repeat("d", 40)}},
			{Effect: factory.OpEffectCommitted},
		},
	}
	history := workCandidateHistory(p)
	for _, want := range []string{strings.Repeat("b", 40), strings.Repeat("c", 40)} {
		if !history[want] {
			t.Fatalf("missing candidate %s: %v", want, history)
		}
	}
	for _, unwanted := range []string{strings.Repeat("a", 40), strings.Repeat("d", 40), ""} {
		if history[unwanted] {
			t.Fatalf("unexpected candidate %q: %v", unwanted, history)
		}
	}
	fallback := workCandidateHistory(factory.Publication{Candidate: strings.Repeat("e", 40)})
	if !fallback[strings.Repeat("e", 40)] || len(fallback) != 1 {
		t.Fatalf("fallback history: %v", fallback)
	}
}

func TestApplyWorkStaleness(t *testing.T) {
	current, old := strings.Repeat("c", 40), strings.Repeat("o", 40)
	publication := &factory.Publication{
		Publish: factory.PublicationOperation{Work: &factory.PublicationIntent{Candidate: current}},
	}
	fresh := &factoryIssueView{Checks: &checksView{HeadOID: current}, Merge: mergeViewDTO(factory.Merge{Stage: factory.MergeOpen, HeadOID: current})}
	applyWorkStaleness(fresh, publication, &factory.Merge{Stage: factory.MergeOpen, HeadOID: current})
	if fresh.Checks.Stale || fresh.Merge.Stale {
		t.Fatalf("current heads marked stale: %+v %+v", fresh.Checks, fresh.Merge)
	}
	stale := &factoryIssueView{Checks: &checksView{HeadOID: old}, Merge: mergeViewDTO(factory.Merge{Stage: factory.MergeOpen, HeadOID: old})}
	applyWorkStaleness(stale, publication, &factory.Merge{Stage: factory.MergeOpen, HeadOID: old})
	if !stale.Checks.Stale || !stale.Merge.Stale {
		t.Fatalf("superseded heads not marked stale: %+v %+v", stale.Checks, stale.Merge)
	}
	terminal := &factoryIssueView{Merge: mergeViewDTO(factory.Merge{Stage: factory.MergeMerged, HeadOID: old})}
	applyWorkStaleness(terminal, publication, &factory.Merge{Stage: factory.MergeMerged, HeadOID: old})
	if terminal.Merge.Stale {
		t.Fatalf("terminal merge marked stale: %+v", terminal.Merge)
	}
	unpublished := &factoryIssueView{Checks: &checksView{HeadOID: old}}
	applyWorkStaleness(unpublished, nil, nil)
	if unpublished.Checks.Stale {
		t.Fatalf("evidence without publication marked stale: %+v", unpublished.Checks)
	}
}

func TestWorkCapabilities(t *testing.T) {
	active := workCapabilityInput{SessionCurrent: true, HasRun: true, Member: true, DispatchOpen: true}
	caps := workCapabilitiesDTO(active)
	if !caps.Watch.Available || !caps.Stop.Available || !caps.Pause.Available {
		t.Fatalf("active capabilities: %+v", caps)
	}
	if caps.Retry.Available || caps.Retry.Reason != workCapUnsettled {
		t.Fatalf("active retry: %+v", caps.Retry)
	}
	if caps.Takeover.Available || caps.Takeover.Reason != workCapUnsettled {
		t.Fatalf("active takeover: %+v", caps.Takeover)
	}
	if caps.Resume.Available || caps.Resume.Reason != workCapDispatchOpen {
		t.Fatalf("active resume: %+v", caps.Resume)
	}

	settled := active
	settled.RunSettled = true
	caps = workCapabilitiesDTO(settled)
	if !caps.Retry.Available || !caps.Takeover.Available {
		t.Fatalf("settled capabilities: %+v", caps)
	}
	if caps.Stop.Available || caps.Stop.Reason != workCapSettled {
		t.Fatalf("settled stop: %+v", caps.Stop)
	}

	queued := settled
	queued.RetryQueued = true
	caps = workCapabilitiesDTO(queued)
	if caps.Retry.Available || caps.Retry.Reason != workCapRetryQueued {
		t.Fatalf("queued retry: %+v", caps.Retry)
	}
	if !caps.Takeover.Available {
		t.Fatalf("queued takeover: %+v", caps.Takeover)
	}

	outsider := settled
	outsider.Member = false
	caps = workCapabilitiesDTO(outsider)
	if caps.Takeover.Available || caps.Takeover.Reason != workCapMembership {
		t.Fatalf("outsider takeover: %+v", caps.Takeover)
	}

	idle := workCapabilityInput{SessionCurrent: true, DispatchOpen: true}
	caps = workCapabilitiesDTO(idle)
	for name, cap := range map[string]workCapabilityView{"stop": caps.Stop, "retry": caps.Retry, "takeover": caps.Takeover} {
		if cap.Available || cap.Reason != workCapNoRun {
			t.Fatalf("%s without run: %+v", name, cap)
		}
	}
	if !caps.Watch.Available || !caps.Pause.Available {
		t.Fatalf("idle watch/pause: %+v", caps)
	}

	paused := settled
	paused.DispatchOpen = false
	caps = workCapabilitiesDTO(paused)
	if caps.Pause.Available || caps.Pause.Reason != workCapDispatchShut {
		t.Fatalf("paused pause: %+v", caps.Pause)
	}
	if !caps.Resume.Available {
		t.Fatalf("paused resume: %+v", caps.Resume)
	}

	for _, reason := range []string{workCapWriteRequired, workCapNotVisible, workCapNoAuthority} {
		denied := settled
		denied.AuthorityReason = reason
		caps = workCapabilitiesDTO(denied)
		for name, cap := range map[string]workCapabilityView{
			"watch": caps.Watch, "stop": caps.Stop, "retry": caps.Retry,
			"takeover": caps.Takeover, "pause": caps.Pause, "resume": caps.Resume,
		} {
			if cap.Available || cap.Reason != reason {
				t.Fatalf("%s denied %s: %+v", name, reason, cap)
			}
		}
	}

	stale := settled
	stale.SessionCurrent = false
	caps = workCapabilitiesDTO(stale)
	if !caps.Watch.Available {
		t.Fatalf("stale session watch: %+v", caps.Watch)
	}
	for name, cap := range map[string]workCapabilityView{
		"stop": caps.Stop, "retry": caps.Retry, "takeover": caps.Takeover,
		"pause": caps.Pause, "resume": caps.Resume,
	} {
		if cap.Available || cap.Reason != workCapSession {
			t.Fatalf("%s stale session: %+v", name, cap)
		}
	}
}

func TestWorkPublicationBoundsHistory(t *testing.T) {
	p := factory.Publication{
		Candidate: strings.Repeat("a", 40), BaseSHA: strings.Repeat("b", 40),
		TargetBranch: "main",
		Publish:      factory.PublicationOperation{Effect: factory.OpEffectCommitted, Completion: "complete"},
	}
	for i := 0; i < maxWorkHistoryEntries+5; i++ {
		p.Corrections = append(p.Corrections, factory.PublicationOperation{
			Work:   &factory.PublicationIntent{Candidate: fmt.Sprintf("correction-%02d", i)},
			Effect: factory.OpEffectCommitted, Completion: "complete", UpdatedUnix: int64(100 + i),
		})
		p.ReviewOperations = append(p.ReviewOperations, factory.ReviewOperation{
			RunID:   fmt.Sprintf("run-%02d", i),
			Work:    factory.ReviewWork{HeadOID: fmt.Sprintf("head-%02d", i), Event: "APPROVED"},
			Outcome: factory.OperationOutcome{Effect: factory.OpEffectCommitted, Completion: "complete"},
		})
	}
	view := workPublicationDTO(factory.Assignment{}, p)
	if len(view.Corrections) != maxWorkHistoryEntries || view.CorrectionsTotal != maxWorkHistoryEntries+5 {
		t.Fatalf("corrections: %d of %d", len(view.Corrections), view.CorrectionsTotal)
	}
	if len(view.Reviews) != maxWorkHistoryEntries || view.ReviewsTotal != maxWorkHistoryEntries+5 {
		t.Fatalf("reviews: %d of %d", len(view.Reviews), view.ReviewsTotal)
	}
	if view.Corrections[0].Candidate != "correction-05" || view.Reviews[0].Run != "run-05" {
		t.Fatalf("oldest retained entries: %+v %+v", view.Corrections[0], view.Reviews[0])
	}
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"candidate", "base_sha", "publish_effect", "corrections", "reviews", "corrections_total", "reviews_total"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing publication key %s: %v", key, decoded)
		}
	}
}

func TestWorkPublicationOmitsProseAndReceipts(t *testing.T) {
	p := factory.Publication{
		Corrections: []factory.PublicationOperation{{
			Work:   &factory.PublicationIntent{Candidate: strings.Repeat("c", 40)},
			Effect: factory.OpEffectCommitted, Receipt: "secret-correction-receipt",
		}},
		ReviewOperations: []factory.ReviewOperation{{
			RunID:   strings.Repeat("r", 32),
			Work:    factory.ReviewWork{HeadOID: strings.Repeat("h", 40), Event: "REQUEST_CHANGES", Body: "secret review prose"},
			Outcome: factory.OperationOutcome{Effect: factory.OpEffectCommitted},
		}},
	}
	body, err := json.Marshal(workPublicationDTO(factory.Assignment{}, p))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "secret") {
		t.Fatalf("prose or receipt leaked: %s", body)
	}
}

func TestValidNativeLinkPart(t *testing.T) {
	for _, part := range []string{"owner", "repo.name", "a-b_c", "0"} {
		if !validNativeLinkPart(part) {
			t.Fatalf("rejected %q", part)
		}
	}
	for _, part := range []string{"", ".", "..", "a/b", `a\b`, "a?b", "a#b", "a b", "a\tb", "a%20b", "a\nb"} {
		if validNativeLinkPart(part) {
			t.Fatalf("accepted %q", part)
		}
	}
}

func TestWorkAttemptDTO(t *testing.T) {
	view := workAttemptDTO(factory.Assignment{
		ID: "attempt-1", Role: "coder", Stage: factory.AssignmentAssigned,
		Run: "run-1", Attempts: 2, CreatedUnix: 100,
	})
	if view.ID != "attempt-1" || view.Role != "coder" || view.Stage != factory.AssignmentAssigned ||
		view.Run != "run-1" || view.Attempts != 2 || view.CreatedUnix != 100 || view.FinishedUnix != 0 {
		t.Fatalf("attempt: %+v", view)
	}
}
