package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// EffectiveAuthority derives the visible verdict for one repository from
// the current grants, preparation readiness and dispatch state.
func (c *Coordinator) EffectiveAuthority(ctx context.Context, repository int64) (factory.EffectiveAuthority, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	var in factory.AuthorityInput
	if policy, err := c.Store.RepositoryPolicy(bounded, repository); err == nil {
		in.Policy = &policy
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.EffectiveAuthority{}, err
	}
	if grant, err := c.Store.OperatorGrant(bounded, repository); err == nil {
		in.Operator = &grant
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.EffectiveAuthority{}, err
	}
	if capacity, err := c.Store.Capacity(bounded); err == nil {
		in.Appliance = &capacity
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.EffectiveAuthority{}, err
	}
	sponsorships, err := c.Store.Sponsorships(bounded, repository)
	if err != nil {
		return factory.EffectiveAuthority{}, err
	}
	for i := range sponsorships {
		if sponsorships[i].Active {
			in.Sponsorship = &sponsorships[i]
			break
		}
	}
	if in.Sponsorship == nil && len(sponsorships) > 0 {
		in.Sponsorship = &sponsorships[0]
	}
	if grant, err := c.Store.EnvironmentGrant(bounded, repository); err == nil {
		in.Environment = &grant
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.EffectiveAuthority{}, err
	}
	projectID, ready, err := c.preparationReadiness(bounded, repository)
	if err != nil {
		return factory.EffectiveAuthority{}, err
	}
	in.ProjectExists = projectID != ""
	in.PreparationReady = ready
	open, _, _, err := c.Store.DispatchState(bounded, repository)
	if err != nil {
		return factory.EffectiveAuthority{}, err
	}
	in.DispatchOpen = open
	return factory.EvaluateAuthority(in), nil
}

// preparationReadiness reports whether the repository's Project has a ready
// preparation for both factory roles with no maintenance hold.
func (c *Coordinator) preparationReadiness(ctx context.Context, repository int64) (string, bool, error) {
	p, err := c.Store.ProjectByRepository(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	held, err := c.Store.MaintenanceHold(ctx, p.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return "", false, err
	}
	if err == nil && held.Hold {
		return p.ID, false, nil
	}
	records, err := c.Store.ProjectPreparations(ctx, p.ID)
	if err != nil {
		return "", false, err
	}
	ready := map[string]bool{}
	for _, record := range records {
		if record.State.Ready {
			ready[record.Preparation.Role] = true
		}
	}
	return p.ID, ready[project.RoleCoder] && ready[project.RoleReviewer], nil
}
