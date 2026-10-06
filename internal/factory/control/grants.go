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

// GrantReceipt is the durable outcome of one settings command: the stored
// revision, whether the change withdrew dispatch, the captured outstanding
// IDs in order, and the visible effective authority afterwards.
type GrantReceipt struct {
	Publications factory.PublicationWithdrawal `json:"publications"`
	Merges       factory.MergeWithdrawal       `json:"merges"`
	Effective    factory.EffectiveAuthority    `json:"effective"`
	Captured     []string                      `json:"captured,omitempty"`
	CommandID    string                        `json:"command_id"`
	Revision     int64                         `json:"revision"`
	Withdrawn    bool                          `json:"withdrawn"`
}

func grantTarget(record string, repository int64) string {
	return "repository/" + strconv.FormatInt(repository, 10) + "/" + record
}

// ApplyPolicy records the owner's repository policy. Disabling or pausing
// closes dispatch and captures every outstanding ID in order.
func (c *Coordinator) ApplyPolicy(ctx context.Context, commandID, principal string, expected int64, policy factory.RepositoryPolicy) (GrantReceipt, error) {
	if policy.Repository <= 0 || policy.Revision != expected {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	withdraw := !policy.Enabled || policy.Paused
	cause := "policy_disabled"
	if policy.Paused {
		cause = "policy_paused"
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandPolicy, grantTarget("policy", policy.Repository), policy,
		func(ctx context.Context) error { return c.Store.SaveRepositoryPolicy(ctx, policy) },
		func() int64 { return policy.Revision + 1 }, policy.Repository, withdraw, cause)
}

// ApplyCapacity records the appliance capacity. Capacity bounds future
// reservations only; it never closes already registered dispatch.
func (c *Coordinator) ApplyCapacity(ctx context.Context, commandID, principal string, expected int64, capacity factory.Capacity) (GrantReceipt, error) {
	if capacity.Revision != expected {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandCapacity, "capacity", capacity,
		func(ctx context.Context) error { return c.Store.SaveCapacity(ctx, capacity) },
		func() int64 { return capacity.Revision + 1 }, 0, false, "")
}

// ApplyOperatorGrant records the operator's repository permission.
// Withdrawing it closes dispatch for that repository.
func (c *Coordinator) ApplyOperatorGrant(ctx context.Context, commandID, principal string, expected int64, grant factory.OperatorGrant) (GrantReceipt, error) {
	if grant.Repository <= 0 || grant.Revision != expected {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandOperatorGrant, grantTarget("operator-grant", grant.Repository), grant,
		func(ctx context.Context) error { return c.Store.SaveOperatorGrant(ctx, grant) },
		func() int64 { return grant.Revision + 1 }, grant.Repository, !grant.Active, "operator_grant_withdrawn")
}

// ApplySponsorship records one connection sponsorship. Withdrawing the last
// active sponsorship closes dispatch; other active sponsorships keep it open.
func (c *Coordinator) ApplySponsorship(ctx context.Context, commandID, principal string, expected int64, sponsorship factory.Sponsorship) (GrantReceipt, error) {
	if sponsorship.Repository <= 0 || sponsorship.Revision != expected {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	withdraw := false
	if !sponsorship.Active {
		others, err := c.Store.Sponsorships(ctx, sponsorship.Repository)
		if err != nil {
			return GrantReceipt{}, err
		}
		withdraw = true
		for _, other := range others {
			if other.Connection != sponsorship.Connection && other.Active {
				withdraw = false
			}
		}
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandSponsorship, grantTarget("sponsorship/"+sponsorship.Connection, sponsorship.Repository), sponsorship,
		func(ctx context.Context) error { return c.Store.SaveSponsorship(ctx, sponsorship) },
		func() int64 { return sponsorship.Revision + 1 }, sponsorship.Repository, withdraw, "sponsorship_withdrawn")
}

// ApplyEnvironmentGrant records the owner's standing environment permission.
// Withdrawing it closes dispatch for that repository.
func (c *Coordinator) ApplyEnvironmentGrant(ctx context.Context, commandID, principal string, expected int64, grant project.EnvironmentGrant) (GrantReceipt, error) {
	if grant.Repository <= 0 || grant.Revision != expected {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	return c.applyGrant(ctx, commandID, principal, factory.CommandEnvironmentGrant, grantTarget("environment-grant", grant.Repository), grant,
		func(ctx context.Context) error { return c.Store.SaveEnvironmentGrant(ctx, grant) },
		func() int64 { return grant.Revision + 1 }, grant.Repository, !grant.Active, "environment_grant_withdrawn")
}

// applyGrant is the shared settings path: normalize and record the command,
// CAS-save the grant, close dispatch on withdrawal, then finish with the
// visible receipt. A lost reply replays the durable outcome.
func (c *Coordinator) applyGrant(ctx context.Context, commandID, principal, typ, target string, record any, save func(context.Context) error, revision func() int64, repository int64, withdraw bool, cause string) (GrantReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	payload, err := json.Marshal(record)
	if err != nil {
		return GrantReceipt{}, err
	}
	cmd := factory.Command{
		ID: commandID, Type: typ, Target: target, Principal: principal,
		Payload: string(payload), Digest: factory.SettingsDigest(typ, target, string(payload)),
	}
	if err = cmd.Validate(); err != nil {
		return GrantReceipt{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return GrantReceipt{}, err
	}
	if !created {
		return replayGrant(stored)
	}
	if err = save(bounded); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		}
		return GrantReceipt{}, err
	}
	receipt := GrantReceipt{CommandID: cmd.ID, Revision: revision(), Withdrawn: withdraw}
	if withdraw {
		withdrawal, err := c.Store.WithdrawDispatch(bounded, repository, cause, principal)
		if err != nil {
			return GrantReceipt{}, err
		}
		receipt.Captured = withdrawal.Captured
		if receipt.Captured == nil {
			receipt.Captured = []string{}
		}
	}
	if repository > 0 {
		receipt.Publications = c.cancelRepositoryPublications(bounded, repository)
		receipt.Merges = c.cancelRepositoryMerges(bounded, repository)
	}
	effective, err := c.EffectiveAuthority(bounded, repository)
	if err != nil {
		return GrantReceipt{}, err
	}
	receipt.Effective = effective
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return GrantReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return GrantReceipt{}, err
	}
	return receipt, nil
}

func replayGrant(stored factory.Command) (GrantReceipt, error) {
	if stored.Finished == "" {
		return GrantReceipt{}, ErrCommandRunning
	}
	var failure struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stored.Outcome), &failure); err != nil {
		return GrantReceipt{}, err
	}
	if failure.Error == "stale_revision" {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	var receipt GrantReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return GrantReceipt{}, err
	}
	return receipt, nil
}
