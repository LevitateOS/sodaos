package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func deliveryStoreAcceptance(publicationID, attemptRoot string, round int, kind, decision string, intents []string) factory.DeliveryAcceptance {
	head, base, diffBase := strings.Repeat("2", 40), strings.Repeat("1", 40), strings.Repeat("3", 40)
	return factory.DeliveryAcceptance{
		Tuple:         factory.CandidateTuple{Head: head, Base: base, DiffBase: diffBase},
		ID:            factory.DeliveryAcceptanceID(publicationID, attemptRoot, round, kind),
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ScopeDigest: strings.Repeat("a", 64), FindingDigest: strings.Repeat("b", 64),
		CoverageDigest: strings.Repeat("c", 64), CheckDigest: strings.Repeat("d", 64),
		PolicyRevision:    "policy-revision-9",
		DecisionID:        decision,
		OriginReceipt:     strings.Repeat("4", 32),
		ResolutionIntents: intents,
		Kind:              kind, Round: round,
	}
}

func deliveryStorePublication() factory.Publication {
	return factory.Publication{
		Authority:    factory.AuthorityRef{},
		ID:           strings.Repeat("7", 32),
		AssignmentID: strings.Repeat("8", 32),
		ProjectID:    "p" + strings.Repeat("d", 24),
		Role:         project.RoleCoder,
		Acceptance:   "d" + strings.Repeat("a", 24),
		Candidate:    strings.Repeat("2", 40),
		BaseSHA:      strings.Repeat("1", 40),
		TargetBranch: "main",
		Stage:        factory.PublicationOpen,
		Repository:   7,
		Issue:        3,
		Revision:     1,
		CreatedUnix:  1,
	}
}

func TestPublicationDeliveryAcceptancesAppendOnly(t *testing.T) {
	publicationID, attemptRoot := strings.Repeat("7", 32), strings.Repeat("b", 32)
	clean := deliveryStoreAcceptance(publicationID, attemptRoot, 1, factory.AcceptanceCleanReview, "", nil)
	if err := clean.Validate(); err != nil {
		t.Fatalf("clean acceptance fixture refused: %v", err)
	}
	old := deliveryStorePublication()
	next := deliveryStorePublication()
	next.DeliveryAcceptances = []factory.DeliveryAcceptance{clean}
	if err := publicationUpdateAllowed(old, next); err != nil {
		t.Fatalf("appending a delivery acceptance refused: %v", err)
	}
	jev := deliveryStoreAcceptance(publicationID, attemptRoot, 2, factory.AcceptanceJevAcceptance, "decision-1", []string{strings.Repeat("6", 32)})
	if err := jev.Validate(); err != nil {
		t.Fatalf("funded acceptance fixture refused: %v", err)
	}
	third := deliveryStorePublication()
	third.DeliveryAcceptances = []factory.DeliveryAcceptance{clean, jev}
	if err := publicationUpdateAllowed(next, third); err != nil {
		t.Fatalf("appending the funded variant refused: %v", err)
	}
	rewritten := deliveryStorePublication()
	mutated := clean
	mutated.FindingDigest = strings.Repeat("e", 64)
	rewritten.DeliveryAcceptances = []factory.DeliveryAcceptance{mutated}
	if err := publicationUpdateAllowed(next, rewritten); !errors.Is(err, ErrPublicationConflict) {
		t.Fatalf("rewritten finding digest allowed: %v", err)
	}
	dropped := deliveryStorePublication()
	if err := publicationUpdateAllowed(next, dropped); !errors.Is(err, ErrPublicationConflict) {
		t.Fatalf("dropped acceptance allowed: %v", err)
	}
	swapped := deliveryStorePublication()
	changed := jev
	changed.ResolutionIntents = []string{strings.Repeat("5", 32)}
	swapped.DeliveryAcceptances = []factory.DeliveryAcceptance{clean, changed}
	if err := publicationUpdateAllowed(third, swapped); !errors.Is(err, ErrPublicationConflict) {
		t.Fatalf("rewritten resolution intent allowed: %v", err)
	}
}
