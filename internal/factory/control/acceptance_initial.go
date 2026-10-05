package control

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// AdmitInitialAcceptance records the narrowly verified original-creation
// path: a visible issue whose creation provenance is verified and
// first-created, posted by the named creator, with creation content
// version, no lifecycle events and no prerequisites. Factory-posted,
// imported, edited or prerequisite-bearing issues require explicit human
// adoption, as do creations without standing policy or a currently
// code-write-authorized creator. The decision ID is deterministic, so a
// duplicate creation observation replays one decision; it can neither
// admit another nor overwrite a later decision. Coordinator-internal:
// intake calls this on authenticated creation observations.
func (c *Coordinator) AdmitInitialAcceptance(ctx context.Context, repository int64, issue, creatorID string, creatorWriteAuthorized bool) (factory.Acceptance, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if repository <= 0 || issue == "" || creatorID == "" {
		return factory.Acceptance{}, errors.New("invalid initial acceptance scope")
	}
	if c.AcceptanceReads == nil {
		return factory.Acceptance{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	evidence, err := c.AcceptanceReads.ReadAcceptanceEvidence(bounded, strconv.FormatInt(repository, 10), issue, nil)
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
	if !view.Verified || !view.FirstCreated || view.PosterID != creatorID {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnverified)
	}
	if view.ContentVer != 0 || view.Lifecycle != 0 {
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
	if !creatorWriteAuthorized {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnauthorized)
	}
	decision := factory.Acceptance{
		ID: factory.InitialAcceptanceID(repository, issue), Repository: repository, IssueIndex: issue,
		Approver: creator, NativeRev: evidence.Revision, Initial: true,
		TitleDigest: view.TitleDigest, ContentDigest: view.ContentDigest, ContentVersion: view.ContentVer,
	}
	if err := decision.Validate(); err != nil {
		return factory.Acceptance{}, err
	}
	if err := c.Store.AdmitAcceptanceDecision(bounded, decision); err != nil {
		return factory.Acceptance{}, err
	}
	return decision, nil
}
