package control

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// ErrIneffectiveAuthority reports a dispatch request the current grants do
// not authorize. The caller restores the missing grant instead of retrying.
var ErrIneffectiveAuthority = errors.New("factory grants do not authorize dispatch")

func grantTarget(record string, repository int64) string {
	return "repository/" + strconv.FormatInt(repository, 10) + "/" + record
}

// ApplyPolicy records the owner's repository policy. Disabling or pausing
// closes dispatch and captures every outstanding ID in order. Each record
// carries its expected revision; the store's revision-guarded save refuses
// a stale one.
func (c *Coordinator) ApplyPolicy(ctx context.Context, commandID, principal string, policy factory.RepositoryPolicy) (factory.GrantReceipt, error) {
	if policy.Repository <= 0 {
		return factory.GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandPolicy, grantTarget("policy", policy.Repository), policy, policy.Repository)
}

// ApplyCapacity records the appliance capacity. Capacity bounds future
// reservations only; it never closes already registered dispatch.
func (c *Coordinator) ApplyCapacity(ctx context.Context, commandID, principal string, capacity factory.Capacity) (factory.GrantReceipt, error) {
	return c.applyGrant(ctx, commandID, principal, factory.CommandCapacity, "capacity", capacity, 0)
}

// ApplyOperatorGrant records the operator's repository permission.
// Withdrawing it closes dispatch for that repository.
func (c *Coordinator) ApplyOperatorGrant(ctx context.Context, commandID, principal string, grant factory.OperatorGrant) (factory.GrantReceipt, error) {
	if grant.Repository <= 0 {
		return factory.GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandOperatorGrant, grantTarget("operator-grant", grant.Repository), grant, grant.Repository)
}

// ApplySponsorship records one connection sponsorship. Withdrawing the last
// active sponsorship closes dispatch; other active sponsorships keep it open.
func (c *Coordinator) ApplySponsorship(ctx context.Context, commandID, principal string, sponsorship factory.Sponsorship) (factory.GrantReceipt, error) {
	if sponsorship.Repository <= 0 {
		return factory.GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandSponsorship, grantTarget("sponsorship/"+sponsorship.Connection, sponsorship.Repository), sponsorship, sponsorship.Repository)
}

// ApplyEnvironmentGrant records the owner's standing environment permission.
// Withdrawing it closes dispatch for that repository.
func (c *Coordinator) ApplyEnvironmentGrant(ctx context.Context, commandID, principal string, grant project.EnvironmentGrant) (factory.GrantReceipt, error) {
	if grant.Repository <= 0 {
		return factory.GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandEnvironmentGrant, grantTarget("environment-grant", grant.Repository), grant, grant.Repository)
}

// applyGrant is the shared settings path: normalize and record the command,
// CAS-save the grant, close dispatch on withdrawal, then finish with the
// visible receipt. A lost reply replays the durable outcome.
func (c *Coordinator) applyGrant(ctx context.Context, commandID, principal, typ, target string, record any, repository int64) (factory.GrantReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	payload, err := json.Marshal(record)
	if err != nil {
		return factory.GrantReceipt{}, err
	}
	cmd := factory.Command{
		ID: commandID, Type: typ, Target: target, Principal: principal,
		Payload: string(payload), Digest: factory.SettingsDigest(typ, target, string(payload)),
	}
	if err = cmd.Validate(); err != nil {
		return factory.GrantReceipt{}, err
	}
	stored, created, err := c.Store.ApplyFactoryGrantCommand(bounded, cmd, record, time.Now())
	if err != nil {
		return factory.GrantReceipt{}, err
	}
	receipt, err := replayGrant(stored)
	if err != nil {
		return factory.GrantReceipt{}, err
	}
	if created && repository > 0 && receipt.Withdrawn {
		// Only a withdrawal cancels work; saving a number never does. The
		// command's immutable receipt records requested/pending cancellation.
		// External delivery happens after the atomic local decision and updates
		// the publication and merge records used by status/recovery reads.
		c.cancelRepositoryPublications(bounded, repository)
		c.cancelRepositoryMerges(bounded, repository)
	}
	return receipt, nil
}

func replayGrant(stored factory.Command) (factory.GrantReceipt, error) {
	if stored.Finished == "" {
		return factory.GrantReceipt{}, ErrCommandRunning
	}
	var receipt factory.GrantReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return factory.GrantReceipt{}, err
	}
	return receipt, nil
}
