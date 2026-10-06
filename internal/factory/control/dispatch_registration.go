package control

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// RegisterDispatch records one outstanding dispatch under currently
// effective authority. Stale or ineffective authority is refused; a delayed
// registration after withdrawal cannot escape the captured set.
func (c *Coordinator) RegisterDispatch(ctx context.Context, d factory.DispatchRegistration) error {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if err := d.Validate(); err != nil {
		return err
	}
	effective, err := c.EffectiveAuthority(bounded, d.Repository)
	if err != nil {
		return err
	}
	if !effective.Effective || effective.Authority != d.Authority {
		if !effective.Effective {
			return ErrIneffectiveAuthority
		}
		return store.ErrStaleRevision
	}
	return c.Store.RegisterDispatch(bounded, d)
}

// ReopenDispatch reopens a withdrawn gate after an authorized control
// revalidated every grant. The expected revision must equal the recorded
// withdrawal revision.
func (c *Coordinator) ReopenDispatch(ctx context.Context, commandID, principal string, repository, expected int64) (GrantReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if repository <= 0 || expected < 0 {
		return GrantReceipt{}, errors.New("invalid dispatch reopen")
	}
	open, revision, _, err := c.Store.DispatchState(bounded, repository)
	if err != nil {
		return GrantReceipt{}, err
	}
	if open || revision != expected {
		return GrantReceipt{}, store.ErrStaleRevision
	}
	effective, err := c.EffectiveAuthority(bounded, repository)
	if err != nil {
		return GrantReceipt{}, err
	}
	for _, reason := range effective.Missing {
		if reason != factory.MissingDispatch {
			return GrantReceipt{}, ErrIneffectiveAuthority
		}
	}
	payload := `{"revision":` + strconv.FormatInt(expected, 10) + `}`
	target := grantTarget("dispatch", repository)
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandReopen, Target: target, Principal: principal,
		Payload: payload, Digest: factory.SettingsDigest(factory.CommandReopen, target, payload),
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
	if err = c.Store.ReopenDispatch(bounded, repository, expected); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		}
		return GrantReceipt{}, err
	}
	after, err := c.EffectiveAuthority(bounded, repository)
	if err != nil {
		return GrantReceipt{}, err
	}
	receipt := GrantReceipt{CommandID: cmd.ID, Revision: expected + 1, Effective: after}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return GrantReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return GrantReceipt{}, err
	}
	return receipt, nil
}
