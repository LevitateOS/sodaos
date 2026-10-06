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

// DecisionReceipt is the durable outcome of one preparation decision: the
// admitted decision, the advanced head and its chain depth. Approvals also
// report whether the maintenance hold stayed active.
type DecisionReceipt struct {
	CommandID  string `json:"command_id"`
	DecisionID string `json:"decision_id"`
	Head       string `json:"head"`
	Depth      int64  `json:"depth"`
	HoldActive bool   `json:"hold_active,omitempty"`
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

// AdmitRequirement records the maintainer's requirement acceptance and
// advances the project head. It changes accepted inputs without closing
// dispatch: no dispatch binds requirement revisions yet.
func (c *Coordinator) AdmitRequirement(ctx context.Context, commandID, principal string, decision project.RequirementDecision) (DecisionReceipt, error) {
	return c.admitDecision(ctx, commandID, principal, factory.CommandRequirement, decision, func(ctx context.Context) error {
		return c.Store.AdmitRequirementDecision(ctx, decision)
	}, func(ctx context.Context) (string, int64, error) {
		head, err := c.Store.RequirementHead(ctx, decision.Project)
		if err != nil {
			return "", 0, err
		}
		depth, err := c.Store.RequirementDepth(ctx, decision.Project)
		return head, depth, err
	}, nil)
}

// AdmitApproval records the administrator's privileged-effect approval. The
// approval must bind the current requirement head; binding a superseded or
// missing requirement conflicts. The receipt reports whether the
// maintenance hold stayed active; releasing it stays an explicit hold
// control so the native marker sync keeps its existing owner.
func (c *Coordinator) AdmitApproval(ctx context.Context, commandID, principal string, decision project.ApprovalDecision) (DecisionReceipt, error) {
	requirement, err := c.Store.RequirementHead(ctx, decision.Project)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return DecisionReceipt{}, store.ErrCommandConflict
		}
		return DecisionReceipt{}, err
	}
	if decision.Requirement != requirement {
		return DecisionReceipt{}, store.ErrCommandConflict
	}
	return c.admitDecision(ctx, commandID, principal, factory.CommandApproval, decision, func(ctx context.Context) error {
		return c.Store.AdmitApprovalDecision(ctx, decision)
	}, func(ctx context.Context) (string, int64, error) {
		head, err := c.Store.ApprovalHead(ctx, decision.Project)
		if err != nil {
			return "", 0, err
		}
		depth, err := c.Store.ApprovalDepth(ctx, decision.Project)
		return head, depth, err
	}, func(ctx context.Context) (bool, error) {
		return c.holdActive(ctx, decision.Project)
	})
}

func (c *Coordinator) admitDecision(ctx context.Context, commandID, principal, typ string, decision any, save func(context.Context) error, head func(context.Context) (string, int64, error), after func(context.Context) (bool, error)) (DecisionReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	payload, err := json.Marshal(decision)
	if err != nil {
		return DecisionReceipt{}, err
	}
	var projectID, decisionID string
	switch d := decision.(type) {
	case project.RequirementDecision:
		projectID, decisionID = d.Project, d.ID
	case project.ApprovalDecision:
		projectID, decisionID = d.Project, d.ID
	default:
		return DecisionReceipt{}, errors.New("unknown preparation decision")
	}
	target := "project/" + projectID + "/" + typ
	cmd := factory.Command{
		ID: commandID, Type: typ, Target: target, Principal: principal,
		Payload: string(payload), Digest: factory.SettingsDigest(typ, target, string(payload)),
	}
	if err = cmd.Validate(); err != nil {
		return DecisionReceipt{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return DecisionReceipt{}, err
	}
	if !created {
		return replayDecision(stored)
	}
	if err = save(bounded); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		}
		return DecisionReceipt{}, err
	}
	current, depth, err := head(bounded)
	if err != nil {
		return DecisionReceipt{}, err
	}
	receipt := DecisionReceipt{CommandID: cmd.ID, DecisionID: decisionID, Head: current, Depth: depth}
	if after != nil {
		active, err := after(bounded)
		if err != nil {
			return DecisionReceipt{}, err
		}
		receipt.HoldActive = active
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return DecisionReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return DecisionReceipt{}, err
	}
	return receipt, nil
}

func replayDecision(stored factory.Command) (DecisionReceipt, error) {
	if stored.Finished == "" {
		return DecisionReceipt{}, ErrCommandRunning
	}
	var failure struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stored.Outcome), &failure); err != nil {
		return DecisionReceipt{}, err
	}
	if failure.Error == "stale_revision" {
		return DecisionReceipt{}, store.ErrStaleRevision
	}
	var receipt DecisionReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return DecisionReceipt{}, err
	}
	return receipt, nil
}

// holdActive reports whether the project maintenance hold is currently set.
func (c *Coordinator) holdActive(ctx context.Context, projectID string) (bool, error) {
	held, err := c.Store.MaintenanceHold(ctx, projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return held.Hold, nil
}
