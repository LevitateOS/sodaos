package control

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/filelock"
	"golang.org/x/sys/unix"
)

var ErrClosed = errors.New("work item was closed")

func (c *Controller) lock(ctx context.Context, id, kind string) (*os.File, error) {
	if !factory.ValidID(id) {
		return nil, errors.New("invalid attempt identity")
	}
	file, err := os.OpenFile(filepath.Join(c.Config.Root, id+"."+kind+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = filelock.Acquire(ctx, file, unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func (c *Controller) save(ctx context.Context, a *factory.Attempt) error {
	saved, err := c.Store.SaveFactoryAttempt(ctx, a)
	if err != nil {
		return err
	}
	if !saved {
		return errors.New("attempt authority changed")
	}
	return nil
}

// live uses Forgejo's current permissions and issue state; labels and comments
// cannot admit, renew or extend an attempt.
func (c *Controller) live(ctx context.Context, id string) (factory.Attempt, error) {
	a, err := c.Store.FactoryAttempt(ctx, id)
	if err != nil {
		return a, err
	}
	if err = a.Authority(time.Now()); err != nil {
		return a, err
	}
	if a.Work.PolicySHA != c.PolicySHA || a.Work.HumanID != c.Human.ID {
		return a, errors.New("admitted policy or human changed")
	}
	return a, c.forgeAuthority(ctx, a)
}

func (c *Controller) forgeAuthority(ctx context.Context, a factory.Attempt) error {
	repo, err := c.Forgejo.RepositoryByID(ctx, c.humanToken, a.Work.RepositoryID)
	if err != nil {
		return err
	}
	if !repo.Private || repo.Permissions == nil || !repo.Permissions.Push || repo.FullName != c.Repository.FullName {
		return errors.New("repository authority changed")
	}
	issue, err := c.Forgejo.WorkIssue(ctx, c.humanToken, repo, a.Work.Issue)
	if err != nil {
		return err
	}
	if issue.State != "open" {
		return ErrClosed
	}
	return nil
}

func (c *Controller) Admit(ctx context.Context, issueNumber int64, delivery string) (factory.Attempt, bool, error) {
	if err := c.repository(ctx); err != nil {
		return factory.Attempt{}, false, err
	}
	issue, err := c.Forgejo.WorkIssue(ctx, c.humanToken, c.Repository, issueNumber)
	if err != nil {
		return factory.Attempt{}, false, err
	}
	if issue.State != "open" || issue.PullRequest != nil {
		return factory.Attempt{}, false, errors.New("admission requires an open issue")
	}
	base, err := c.Publisher.BranchRevision(ctx, c.Repository.DefaultBranch)
	if err != nil {
		return factory.Attempt{}, false, err
	}
	work := factory.WorkItem{RepositoryID: c.Repository.ID, Issue: issueNumber, HumanID: c.Human.ID, Objective: issue.Title + "\n\n" + issue.Body, BaseSHA: base, PolicySHA: c.PolicySHA}
	a, err := factory.New(work, delivery, time.Now())
	if err != nil {
		return a, false, err
	}
	return c.Store.AdmitFactory(ctx, a)
}

func (c *Controller) watch(ctx context.Context, id string, cancel context.CancelCauseFunc) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	polls := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a, err := c.Store.FactoryAttempt(ctx, id)
			if err == nil {
				err = a.Authority(time.Now())
			}
			polls++
			if err == nil && polls%5 == 0 {
				_, err = c.live(ctx, id)
			}
			if err != nil {
				cancel(err)
				return
			}
		}
	}
}
