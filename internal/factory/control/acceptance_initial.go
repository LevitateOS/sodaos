package control

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// AdmitInitialAcceptance admits only original authenticated native creation
// facts corroborated by the current exact issue identity and unchanged inputs.
// Normal native creation has no synthesized first-history row. The native
// issue ID determines replay identity; a later human head remains authoritative.
func (c *Coordinator) AdmitInitialAcceptance(ctx context.Context, hint IntakeHint) (factory.Acceptance, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if err := hint.Validate(); err != nil || !hint.Created || hint.NativeIssueID <= 0 {
		return factory.Acceptance{}, errors.New("invalid initial acceptance scope")
	}
	repository := hint.Repository
	issue := strconv.FormatInt(hint.Issue, 10)
	creatorID := strconv.FormatInt(hint.Creator, 10)
	decisionID := factory.InitialAcceptanceID(repository, strconv.FormatInt(hint.NativeIssueID, 10))
	if head, err := c.Store.AcceptanceHead(bounded, repository, hint.Issue); err == nil {
		if head != decisionID {
			return factory.Acceptance{}, store.ErrStaleRevision
		}
		return c.Store.AcceptanceDecision(bounded, head)
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.Acceptance{}, err
	}
	if c.AcceptanceReads == nil {
		return factory.Acceptance{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	evidence, err := c.readAcceptanceEvidence(bounded, factory.Acceptance{Repository: repository, IssueIndex: issue})
	if err != nil {
		var refusal *AcceptanceRefusal
		if errors.As(err, &refusal) {
			return factory.Acceptance{}, refusal
		}
		return factory.Acceptance{}, err
	}
	view := evidence.Issue
	if view.Index != issue {
		return factory.Acceptance{}, refuseAcceptance(RefusalIncompleteEvidence)
	}
	if !view.Visible {
		return factory.Acceptance{}, refuseAcceptance(RefusalIssueHidden)
	}
	if !hint.CreationVerified || view.ID != strconv.FormatInt(hint.NativeIssueID, 10) ||
		view.PosterID != creatorID || view.CreatedUnix != hint.CreatedUnix ||
		view.TitleDigest != hint.TitleDigest || view.ContentDigest != hint.ContentDigest {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnverified)
	}
	if view.ContentVer != 0 || view.LifecycleOccurrence != factory.OriginalOccurrence {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationEdited)
	}
	if len(evidence.Dependencies) != 0 {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationBlocked)
	}
	policy, err := c.Store.RepositoryPolicy(bounded, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnauthorized)
		}
		return factory.Acceptance{}, err
	}
	creator, err := strconv.ParseInt(creatorID, 10, 64)
	if err != nil || creator <= 0 {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnverified)
	}
	for _, ref := range []factory.ActorBindingRef{policy.Publish, policy.Create, policy.Review, policy.Merge} {
		if ref.ActorID == creator {
			return factory.Acceptance{}, refuseAcceptance(RefusalCreationFactory)
		}
	}
	if !hint.CreatorWrite {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnauthorized)
	}
	decision := factory.Acceptance{
		ID: decisionID, Repository: repository, IssueIndex: issue,
		Approver: creator, NativeRev: evidence.Revision, Initial: true,
		TitleDigest: view.TitleDigest, TitleOccurrence: view.TitleOccurrence, ContentDigest: view.ContentDigest, ContentVersion: view.ContentVer,
	}
	if err := decision.Validate(); err != nil {
		return factory.Acceptance{}, err
	}
	if err := c.Store.AdmitAcceptanceDecision(bounded, decision); err != nil {
		return factory.Acceptance{}, err
	}
	return decision, nil
}
