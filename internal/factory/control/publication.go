package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host/publish"
)

func (c *Controller) publish(ctx context.Context, a *factory.Attempt, r factory.Run, result factory.Result, bundle []byte) error {
	lease, err := c.lock(ctx, a.ID, "publication")
	if err != nil {
		return err
	}
	defer func() { _ = lease.Close() }()
	current, err := c.live(ctx, a.ID)
	if err != nil {
		return err
	}
	request := publish.Request{Attempt: current, Run: r, Commit: result.Candidate, Bundle: bundle}
	if err = c.Publisher.Candidate(ctx, request, func() error { _, err := c.live(ctx, a.ID); return err }); err != nil {
		return err
	}
	if err = c.recordCandidate(ctx, &current, r, result.Candidate); err != nil {
		return err
	}
	if err = c.ensurePull(ctx, &current, r, result); err != nil {
		return err
	}
	if err = c.checkPull(ctx, current); err != nil {
		return err
	}
	if err = current.BeginCI(time.Now()); err != nil {
		return err
	}
	if err = c.save(ctx, &current); err != nil {
		return err
	}
	*a = current
	return nil
}

func (c *Controller) ensurePull(ctx context.Context, current *factory.Attempt, r factory.Run, result factory.Result) error {
	if current.Pull == 0 {
		pull, err := c.Forgejo.CreateWorkPull(ctx, c.implementationToken, c.Repository, publish.Branch(current.ID), c.Repository.DefaultBranch, "Soda: issue "+fmt.Sprint(current.Work.Issue), fmt.Sprintf("Attempt `%s`, run `%s`.\n\n%s\n\nCloses #%d", current.ID, r.ID, result.Summary, current.Work.Issue))
		if err != nil {
			return err
		}
		current.Pull = pull.Number
	}
	return c.save(ctx, current)
}

func (c *Controller) checkPull(ctx context.Context, a factory.Attempt) error {
	pull, err := c.Forgejo.WorkPull(ctx, c.humanToken, c.Repository, a.Pull)
	if err != nil {
		return err
	}
	return c.validatePull(pull, a)
}

func (c *Controller) validatePull(pull forgejo.Pull, a factory.Attempt) error {
	if err := c.validateCandidatePull(pull, a); err != nil {
		return err
	}
	if pull.Head.Repo.ID != a.Work.RepositoryID || pull.Base.Repo.ID != a.Work.RepositoryID {
		return errors.New("PR repository differs from assigned target")
	}
	if pull.Base.Ref != c.Repository.DefaultBranch || pull.Base.SHA != a.Work.BaseSHA {
		return errors.New("PR base differs from admitted revision")
	}
	return nil
}

func (c *Controller) submitReview(ctx context.Context, a factory.Attempt, r factory.Run, result factory.Result) error {
	if result.Candidate != r.InputSHA {
		return errors.New("review result differs from assigned commit")
	}
	lease, err := c.lock(ctx, a.ID, "publication")
	if err != nil {
		return err
	}
	defer func() { _ = lease.Close() }()
	current, err := c.live(ctx, a.ID)
	if err != nil {
		return err
	}
	if err = r.Authority(a.ID, time.Now()); err != nil {
		return err
	}
	if err = c.checkPull(ctx, current); err != nil {
		return err
	}
	event := "REQUEST_CHANGES"
	if result.ReviewPassed {
		event = "APPROVED"
	}
	body := fmt.Sprintf("Soda attempt `%s`, fresh review run `%s`, commit `%s`.\n\n%s", a.ID, r.ID, r.InputSHA, result.Summary)
	if len(result.Findings) > 0 {
		body += "\n\n" + strings.Join(result.Findings, "\n\n")
	}
	if err = c.Forgejo.SubmitWorkReview(ctx, c.reviewToken, c.Repository, a.Pull, r.InputSHA, event, body); err != nil {
		return err
	}
	return c.checkPull(ctx, current)
}

func (c *Controller) recordCandidate(ctx context.Context, a *factory.Attempt, r factory.Run, commit string) error {
	if err := a.CandidateFrom(r, commit, time.Now()); err != nil {
		return err
	}
	return c.save(ctx, a)
}

func (c *Controller) validateCandidatePull(pull forgejo.Pull, a factory.Attempt) error {
	if pull.State != "open" || pull.Merged || pull.Head.SHA != a.Candidate || pull.Head.Ref != publish.Branch(a.ID) || pull.User.ID != c.Implementer.ID {
		return errors.New("PR candidate changed or closed")
	}
	return nil
}
