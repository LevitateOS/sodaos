package control

import (
	"context"
	"errors"
	"strconv"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/store"
)

// CreationRecoverySource observes retained original native issue-open tasks
// under an enrolled actor's credential. It grants no provider authority.
type CreationRecoverySource interface {
	ReadCreations(ctx context.Context, repository, after string, limit int) (extensions.CreationRecoveryPage, error)
}

const creationRecoveryPageSize = 16

var ErrCreationProvenanceIncomplete = errors.New("retained native creation provenance is incomplete")

// recoverCreations shares the serving owner and its absolute RPC budget. One
// repository page per pass keeps unavailable or full repositories from starving
// later repositories. Native retention is a boundary, not universal history.
func (c *Coordinator) recoverCreations(ctx context.Context) error {
	if c.Creations == nil {
		return nil
	}
	repositories, err := c.Store.CreationRecoveryRepositories(ctx, c.creationRepositoryCursor, 1)
	if err != nil {
		return err
	}
	if len(repositories) == 0 {
		c.creationRepositoryCursor = 0
		repositories, err = c.Store.CreationRecoveryRepositories(ctx, 0, 1)
		if err != nil || len(repositories) == 0 {
			return err
		}
	}
	repository := repositories[0]
	c.creationRepositoryCursor = repository
	after, err := c.Store.CreationRecoveryCursor(ctx, repository)
	if err != nil {
		return err
	}
	if !readinessBudgetFrom(ctx).reserveNativeRPC(1) {
		return ErrReadinessPassPending
	}
	afterID := ""
	if after > 0 {
		afterID = strconv.FormatInt(after, 10)
	}
	page, err := c.Creations.ReadCreations(ctx, strconv.FormatInt(repository, 10), afterID, creationRecoveryPageSize)
	if err != nil {
		return err
	}
	if page.RepositoryID != strconv.FormatInt(repository, 10) || len(page.Creations) > creationRecoveryPageSize || !page.RetentionLimited {
		return errors.New("invalid native creation recovery page")
	}
	next := after
	if page.NextCursor != "" {
		next, err = nativeCreationID(page.NextCursor)
		if err != nil || next < after {
			return errors.New("invalid native creation recovery cursor")
		}
	}
	if page.OldestRetainedTaskID != "" {
		if _, err = nativeCreationID(page.OldestRetainedTaskID); err != nil {
			return err
		}
	}
	works := make([]store.ReadinessWork, 0, len(page.Creations))
	last := after
	for _, observation := range page.Creations {
		task, err := nativeCreationID(observation.TaskID)
		if err != nil || task <= last || task > next || observation.RepositoryID != page.RepositoryID {
			return errors.New("invalid native creation task identity")
		}
		last = task
		issue, err := nativeCreationID(observation.IssueIndex)
		if err != nil {
			return err
		}
		issueID, err := nativeCreationID(observation.IssueID)
		if err != nil {
			return err
		}
		creator, err := nativeCreationID(observation.CreatorID)
		if err != nil {
			return err
		}
		hint := IntakeHint{
			Delivery: observation.DeliveryID, Repository: repository, Issue: issue,
			Created: true, Creator: creator, CreatorWrite: observation.CreatorWrite, CreationVerified: observation.Verified,
			NativeIssueID: issueID, CreatedUnix: observation.CreatedUnix, TitleDigest: observation.TitleDigest, ContentDigest: observation.ContentDigest,
		}
		if err = hint.Validate(); err != nil {
			return err
		}
		works = append(works, readinessCreationWork(hint))
	}
	// Unsupported or redacted tasks must not disappear behind the cursor.
	// Keep the known observations durable, but retain this bounded page until
	// its original provenance can be corroborated. Complete only names the
	// retained scan end and does not change this provenance requirement.
	checkpoint := next
	if !page.ProvenanceComplete {
		checkpoint = after
	}
	if err := c.Store.EnqueueReadinessCreationPage(ctx, repository, after, checkpoint, works); err != nil {
		return err
	}
	if !page.ProvenanceComplete {
		return ErrCreationProvenanceIncomplete
	}
	return nil
}

func nativeCreationID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
		return 0, errors.New("invalid native creation identity")
	}
	return id, nil
}
