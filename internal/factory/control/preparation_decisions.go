package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

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
