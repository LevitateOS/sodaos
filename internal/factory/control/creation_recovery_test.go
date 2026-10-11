package control

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

type firstCreationReadFails struct {
	evidence AcceptanceEvidence
	calls    atomic.Int32
}

func (s *firstCreationReadFails) ReadAcceptanceEvidence(context.Context, string, string, []string) (AcceptanceEvidence, error) {
	if s.calls.Add(1) == 1 {
		return AcceptanceEvidence{}, errors.New("temporary native read failure")
	}
	return s.evidence, nil
}

type retainedCreationSource struct {
	hint  IntakeHint
	calls atomic.Int32
}

func (s *retainedCreationSource) ReadCreations(_ context.Context, repository, after string, _ int) (extensions.CreationRecoveryPage, error) {
	s.calls.Add(1)
	page := extensions.CreationRecoveryPage{RepositoryID: repository, NextCursor: "9", Complete: true, ProvenanceComplete: true, RetentionLimited: true, OldestRetainedTaskID: "9"}
	if after == "" {
		page.Creations = []extensions.NativeCreationObservation{{TaskID: "9", DeliveryID: s.hint.Delivery, RepositoryID: repository, IssueID: "103", IssueIndex: "3", CreatorID: "5", CreatorWrite: s.hint.CreatorWrite, CreatedUnix: s.hint.CreatedUnix, TitleDigest: s.hint.TitleDigest, ContentDigest: s.hint.ContentDigest, Verified: true}}
	}
	return page, nil
}

func waitCreationHead(t *testing.T, c *Coordinator, want string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		head, err := c.Store.AcceptanceHead(context.Background(), 42, 3)
		seen, seenErr := c.Store.IntakeDeliverySeen(context.Background(), "creation-recovery")
		if err == nil && head == want && seenErr == nil && seen {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("serving owner failed to resume original creation: head=%q seen=%v errors=%v/%v", head, seen, err, seenErr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestServingResumesTransientInitialAdmissionAfterRestart(t *testing.T) {
	ctx := context.Background()
	evidence := creationEvidence()
	evidence.Issue.Verified, evidence.Issue.FirstCreated = false, false
	c := creationHarness(t, evidence)
	source := &firstCreationReadFails{evidence: evidence}
	c.AcceptanceReads = source
	hint := creationHint("creation-recovery")
	if _, _, err := c.ObserveIssueEvent(ctx, hint); err == nil {
		t.Fatal("temporary admission failure was discarded")
	}
	if seen, err := c.Store.IntakeDeliverySeen(ctx, hint.Delivery); err != nil || seen {
		t.Fatal("unfinished initial admission acknowledged:", seen, err)
	}
	work, err := c.Store.ReadinessWork(ctx, readinessDeliverySourceID(hint.Delivery))
	if err != nil || !work.CreationVerified || work.NativeIssueID != 103 {
		t.Fatal("restart lost original authority:", work, err)
	}
	resumed := NewCoordinator(c.Store, c.Host, c.Broker)
	resumed.AcceptanceReads = source
	if err := resumed.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resumed.Close() })
	waitCreationHead(t, resumed, factory.InitialAcceptanceID(42, "103"))
}

func TestServingRecoversIncompleteSameDeliveryWithoutFirstHistory(t *testing.T) {
	ctx := context.Background()
	evidence := creationEvidence()
	evidence.Issue.Verified, evidence.Issue.FirstCreated = false, false
	c := creationHarness(t, evidence)
	hint := creationHint("creation-recovery")
	partial := IntakeHint{Delivery: hint.Delivery, Repository: 42, Issue: 3, Created: true, Creator: 5}
	if _, _, err := c.ObserveIssueEvent(ctx, partial); err == nil {
		t.Fatal("incomplete creation was acknowledged")
	}
	if seen, err := c.Store.IntakeDeliverySeen(ctx, hint.Delivery); err != nil || seen {
		t.Fatal("incomplete hint escaped pending admission:", seen, err)
	}
	resumed := NewCoordinator(c.Store, c.Host, c.Broker)
	resumed.AcceptanceReads = c.AcceptanceReads
	resumed.Creations = &retainedCreationSource{hint: hint}
	if err := resumed.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resumed.Close() })
	waitCreationHead(t, resumed, factory.InitialAcceptanceID(42, "103"))
	if cursor, err := c.Store.CreationRecoveryCursor(ctx, 42); err != nil || cursor != 9 {
		t.Fatal(cursor, err)
	}
}

func TestCreationReplayRetainsLaterHumanAdoption(t *testing.T) {
	ctx := context.Background()
	c := creationHarness(t, creationEvidence())
	hint := creationHint("creation-recovery")
	partial := IntakeHint{Delivery: hint.Delivery, Repository: 42, Issue: 3, Created: true, Creator: 5}
	_, _, _ = c.ObserveIssueEvent(ctx, partial)
	decision := acceptanceDecision()
	decision.TitleDigest, decision.ContentDigest, decision.ContentVersion = hint.TitleDigest, hint.ContentDigest, 0
	decision.NativeRev = 41
	decision.Sources = nil
	decision.Resolutions = nil
	decision.Prerequisites = nil
	if err := c.Store.AdmitAcceptanceDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ObserveIssueEvent(ctx, hint); err != nil {
		t.Fatal(err)
	}
	head, err := c.Store.AcceptanceHead(ctx, 42, 3)
	if err != nil || head != decision.ID {
		t.Fatal("original replay replaced human adoption:", head, err)
	}
	if _, err := c.Store.AcceptanceDecision(ctx, factory.InitialAcceptanceID(42, "103")); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("replay created a competing initial decision:", err)
	}
}

func TestServingRecoversCreationMissedBeforeAnyDelivery(t *testing.T) {
	ctx := context.Background()
	evidence := creationEvidence()
	evidence.Issue.Verified, evidence.Issue.FirstCreated = false, false
	c := creationHarness(t, evidence)
	c.Creations = &retainedCreationSource{hint: creationHint("creation-recovery")}
	if err := c.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	waitCreationHead(t, c, factory.InitialAcceptanceID(42, "103"))
}

type incompleteCreationPage struct {
	retainedCreationSource
	provenanceComplete, complete bool
}

func (s *incompleteCreationPage) ReadCreations(ctx context.Context, repository, after string, limit int) (extensions.CreationRecoveryPage, error) {
	page, err := s.retainedCreationSource.ReadCreations(ctx, repository, after, limit)
	page.ProvenanceComplete, page.Complete = s.provenanceComplete, s.complete
	return page, err
}

func TestServingCreationRecoveryDistinguishesScanPrefixAndUnknownProvenance(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		provenanceComplete, complete bool
		cursor                       int64
	}{
		{"mixed-unknown-original-task", false, true, 0},
		{"known-bounded-prefix", true, false, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			c := creationHarness(t, creationEvidence())
			c.Creations = &incompleteCreationPage{retainedCreationSource: retainedCreationSource{hint: creationHint("creation-recovery")}, provenanceComplete: tc.provenanceComplete, complete: tc.complete}
			if err := c.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = c.Close() })
			waitCreationHead(t, c, factory.InitialAcceptanceID(42, "103"))
			if cursor, err := c.Store.CreationRecoveryCursor(ctx, 42); err != nil || cursor != tc.cursor {
				t.Fatal("scan end substituted for complete original provenance:", cursor, err)
			}
		})
	}
}
