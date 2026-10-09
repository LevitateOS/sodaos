package store

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func readinessTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	s, _ := postgresFixture(t, nil)
	return s, context.Background()
}

func readinessCandidate() factory.IssueControl {
	return factory.IssueControl{
		Repository: 7, Issue: 3,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("a", 64), Authority: strings.Repeat("b", 64),
	}
}

func TestRecordIssueAssessment(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(500, 0)
	stored, changed, err := s.RecordIssueAssessment(ctx, readinessCandidate(), now)
	if err != nil || !changed {
		t.Fatal("first assessment must record:", stored, changed, err)
	}
	if stored.Revision != 1 || stored.FirstSeenUnix != 500 || stored.AssessedUnix != 500 {
		t.Fatal("first assessment identity wrong:", stored)
	}
	same, changed, err := s.RecordIssueAssessment(ctx, readinessCandidate(), now.Add(time.Hour))
	if err != nil || changed {
		t.Fatal("unchanged assessment must replay:", same, changed, err)
	}
	if same.Revision != 1 || same.AssessedUnix != 500 {
		t.Fatal("replay must keep the recorded assessment:", same)
	}
	blocked := readinessCandidate()
	blocked.Readiness = factory.ReadinessBlocked
	blocked.Reason = factory.BlockerCodePending
	blocked.Blockers = []factory.Blocker{{
		Code: factory.BlockerCodePending, EndpointRepo: 7, EndpointIssue: 9,
		Resolution: factory.BlockerResolution(factory.BlockerCodePending),
	}}
	blocked.Fingerprint = strings.Repeat("c", 64)
	advanced, changed, err := s.RecordIssueAssessment(ctx, blocked, now.Add(2*time.Hour))
	if err != nil || !changed {
		t.Fatal("changed assessment must record:", advanced, changed, err)
	}
	if advanced.Revision != 2 || advanced.FirstSeenUnix != 500 || advanced.AssessedUnix != 500+7200 {
		t.Fatal("advanced assessment identity wrong:", advanced)
	}
	got, err := s.IssueControl(ctx, 7, 3)
	if err != nil || got.Revision != 2 || got.Reason != factory.BlockerCodePending {
		t.Fatal("stored control mismatch:", got, err)
	}
	if _, err = s.IssueControl(ctx, 7, 4); !errors.Is(err, ErrNotFound) {
		t.Fatal("absent control must report not found:", err)
	}
	if _, _, err = s.RecordIssueAssessment(ctx, factory.IssueControl{}, now); err == nil {
		t.Fatal("invalid candidate recorded")
	}
}

func TestRecordIssueAssessmentRefreshesNativeObservationWithoutSemanticRevision(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(800, 0)
	candidate := readinessCandidate()
	candidate.NativeRev = 10
	first, changed, err := s.RecordIssueAssessment(ctx, candidate, now)
	if err != nil || !changed {
		t.Fatal("first assessment must record:", first, changed, err)
	}
	candidate.NativeRev = 11
	refreshed, changed, err := s.RecordIssueAssessment(ctx, candidate, now.Add(time.Minute))
	if err != nil || changed || refreshed.Revision != first.Revision || refreshed.NativeRev != 11 || refreshed.Fingerprint != first.Fingerprint {
		t.Fatalf("equal semantic fingerprint did not refresh its native observation: %+v changed=%v err=%v", refreshed, changed, err)
	}
	candidate.NativeRev = 9
	older, changed, err := s.RecordIssueAssessment(ctx, candidate, now.Add(2*time.Minute))
	if err != nil || changed || older.NativeRev != 11 {
		t.Fatalf("older observation downgraded the current marker: %+v changed=%v err=%v", older, changed, err)
	}
}

func TestIssueControlsListOldestFirst(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Now()
	for _, issue := range []int64{9, 3, 5} {
		candidate := readinessCandidate()
		candidate.Issue = issue
		if _, _, err := s.RecordIssueAssessment(ctx, candidate, now); err != nil {
			t.Fatal(err)
		}
	}
	controls, err := s.IssueControls(ctx, 7, 10)
	if err != nil || len(controls) != 3 {
		t.Fatal("control listing wrong:", controls, err)
	}
	for i, want := range []int64{3, 5, 9} {
		if controls[i].Issue != want {
			t.Fatal("controls not oldest first:", controls)
		}
	}
	one, err := s.IssueControls(ctx, 7, 1)
	if err != nil || len(one) != 1 || one[0].Issue != 3 {
		t.Fatal("bounded listing wrong:", one, err)
	}
	if _, err = s.IssueControls(ctx, 7, 0); err == nil {
		t.Fatal("unbounded listing accepted")
	}
}

func TestIntakeDeliveryDedup(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Now()
	seen, err := s.IntakeDeliverySeen(ctx, "delivery-1")
	if err != nil || seen {
		t.Fatal("fresh delivery seen:", seen, err)
	}
	duplicate, err := s.RecordIntakeDelivery(ctx, "delivery-1", 7, 3, "issues", now)
	if err != nil || duplicate {
		t.Fatal("first delivery must log:", duplicate, err)
	}
	seen, err = s.IntakeDeliverySeen(ctx, "delivery-1")
	if err != nil || !seen {
		t.Fatal("logged delivery unseen:", seen, err)
	}
	duplicate, err = s.RecordIntakeDelivery(ctx, "delivery-1", 7, 3, "issues", now)
	if err != nil || !duplicate {
		t.Fatal("redelivery must replay as duplicate:", duplicate, err)
	}
	if _, err = s.RecordIntakeDelivery(ctx, "", 7, 3, "issues", now); err == nil {
		t.Fatal("empty delivery logged")
	}
}

func TestAcceptanceDependants(t *testing.T) {
	s, ctx := readinessTestStore(t)
	digest := strings.Repeat("d", 64)
	dependent := factory.Acceptance{
		ID: "d" + strings.Repeat("1", 24), Repository: 7, IssueIndex: "3",
		Approver: 5, NativeRev: 9, TitleDigest: digest, ContentDigest: digest,
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "8", EndpointRepo: 7, EndpointIssue: 9,
			Outcome: factory.PrereqResult,
		}},
	}
	if err := s.AdmitAcceptanceDecision(ctx, dependent); err != nil {
		t.Fatal(err)
	}
	unrelated := dependent
	unrelated.ID = "d" + strings.Repeat("2", 24)
	unrelated.IssueIndex = "4"
	unrelated.Prerequisites = nil
	if err := s.AdmitAcceptanceDecision(ctx, unrelated); err != nil {
		t.Fatal(err)
	}
	var dependants []factory.DependenceRef
	err := s.VisitAcceptanceDependants(ctx, 7, 9, func(dependant factory.DependenceRef) error {
		dependants = append(dependants, dependant)
		return nil
	})
	if err != nil || len(dependants) != 1 || dependants[0].Issue != 3 {
		t.Fatal("dependant scan wrong:", dependants, err)
	}
	var none []factory.DependenceRef
	err = s.VisitAcceptanceDependants(ctx, 7, 3, func(dependant factory.DependenceRef) error {
		none = append(none, dependant)
		return nil
	})
	if err != nil || len(none) != 0 {
		t.Fatal("empty dependant scan wrong:", none, err)
	}
}

func TestVisitAcceptanceDependantsPagesAndPropagatesLateFailure(t *testing.T) {
	s, ctx := readinessTestStore(t)
	digest := strings.Repeat("d", 64)
	for issue := int64(1); issue <= acceptanceDependantsPageSize+1; issue++ {
		decision := factory.Acceptance{
			ID: "d" + fmt.Sprintf("%024d", issue), Repository: 7, IssueIndex: strconv.FormatInt(issue, 10),
			Approver: 5, NativeRev: 9, TitleDigest: digest, ContentDigest: digest,
		}
		if issue == 1 || issue == acceptanceDependantsPageSize+1 {
			decision.Prerequisites = append(decision.Prerequisites, factory.AcceptedPrerequisite{
				Occurrence: "21", DependsOn: "8", EndpointRepo: 7, EndpointIssue: 9,
				Outcome: factory.PrereqResult,
			})
		}
		if issue == acceptanceDependantsPageSize+1 {
			decision.Prerequisites = append(decision.Prerequisites, factory.AcceptedPrerequisite{
				Occurrence: "23", DependsOn: "8", EndpointRepo: 7, EndpointIssue: 11,
				Outcome: factory.PrereqResult,
			})
		}
		if issue <= acceptanceDependantsPageSize {
			decision.Prerequisites = append(decision.Prerequisites, factory.AcceptedPrerequisite{
				Occurrence: "22", DependsOn: "8", EndpointRepo: 7, EndpointIssue: 10,
				Outcome: factory.PrereqResult,
			})
		}
		if err := s.AdmitAcceptanceDecision(ctx, decision); err != nil {
			t.Fatal("admit head", issue, err)
		}
	}
	var found []int64
	if err := s.VisitAcceptanceDependants(ctx, 7, 9, func(dependant factory.DependenceRef) error {
		found = append(found, dependant.Issue)
		return nil
	}); err != nil {
		t.Fatal("paged scan:", err)
	}
	if !reflect.DeepEqual(found, []int64{1, acceptanceDependantsPageSize + 1}) {
		t.Fatal("sparse page scan lost or reordered matches:", found)
	}

	found = nil
	if err := s.VisitAcceptanceDependants(ctx, 7, 11, func(dependant factory.DependenceRef) error {
		found = append(found, dependant.Issue)
		return nil
	}); err != nil || !reflect.DeepEqual(found, []int64{acceptanceDependantsPageSize + 1}) {
		t.Fatal("empty matching page must not end the scan:", found, err)
	}

	lateCtx, cancel := context.WithCancel(ctx)
	visited := 0
	err := s.VisitAcceptanceDependants(lateCtx, 7, 10, func(factory.DependenceRef) error {
		visited++
		if visited == acceptanceDependantsPageSize {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || visited != acceptanceDependantsPageSize {
		t.Fatalf("late page failure must propagate after bounded progress: visited=%d err=%v", visited, err)
	}
}

func TestReadinessSweepState(t *testing.T) {
	s, ctx := readinessTestStore(t)
	if _, err := s.ReadinessSweepRevision(ctx, 7); !errors.Is(err, ErrNotFound) {
		t.Fatal("absent sweep state must report not found:", err)
	}
	if err := s.SaveReadinessSweep(ctx, 7, 12, time.Unix(600, 0)); err != nil {
		t.Fatal(err)
	}
	revision, err := s.ReadinessSweepRevision(ctx, 7)
	if err != nil || revision != 12 {
		t.Fatal("sweep revision wrong:", revision, err)
	}
	if err := s.SaveReadinessSweep(ctx, 7, 15, time.Unix(700, 0)); err != nil {
		t.Fatal(err)
	}
	revision, err = s.ReadinessSweepRevision(ctx, 7)
	if err != nil || revision != 15 {
		t.Fatal("sweep revision advance wrong:", revision, err)
	}
	if err := s.SaveReadinessSweep(ctx, 0, 15, time.Now()); err == nil {
		t.Fatal("invalid sweep state saved")
	}
}
