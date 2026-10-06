package publish

import (
	"context"
	"errors"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

// runWithEnv runs one Git command with additional environment entries,
// binding a registered operation to its push headers without placing
// secrets in arguments. Diagnostics stay redacted like run.
func (g *repository) runWithEnv(ctx context.Context, extra []string, args ...string) error {
	command := g.command(ctx, args...)
	command.Env = append(command.Env, extra...)
	var output limitedOutput
	command.Stdout = &output
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("publication Git operation failed")
	}
	return nil
}

// PushBranch pushes exactly the validated candidate to the registered
// target under its operation binding. The refspec carries one tuple and
// no options; anything else refuses before launching. A push failure
// reconciles by lookup: a committed, refused or still-pending operation
// is adopted, and only a pending one may push again under a rebound
// admission.
func (v *ValidatedRepo) PushBranch(ctx context.Context, c Config, ops BackgroundOperations, operationID, targetRef string) error {
	if v == nil || v.repo == nil {
		return errors.New("publication workspace is not validated")
	}
	if !factory.ValidPublicationOperationID(operationID) {
		return &Refusal{Reason: "invalid_operation"}
	}
	if !validPublishRef(targetRef) {
		return &Refusal{Reason: "invalid_refs"}
	}
	if ops == nil {
		return &Wait{Reason: "operations_unavailable"}
	}
	refspec, err := extensions.FormatPublishRefspec("refs/heads/candidate", targetRef)
	if err != nil {
		return &Refusal{Reason: "invalid_refs"}
	}
	env, err := ops.PublishPushEnv(ctx, operationID, false)
	if err != nil {
		return err
	}
	if err = v.repo.runWithEnv(ctx, env, "push", "--no-follow-tags", c.Remote, refspec); err != nil {
		return v.reconcilePush(ctx, c, ops, operationID, targetRef, refspec, err)
	}
	return nil
}

// reconcilePush adopts the operation behind a failed push. A still-pending
// operation retries once under a rebound admission: a revoked admission
// fails before launching, so the retry cannot duplicate a receiver. Any
// other state is adopted through the lookup.
func (v *ValidatedRepo) reconcilePush(ctx context.Context, c Config, ops BackgroundOperations, operationID, targetRef, refspec string, pushErr error) error {
	outcome, err := lookupOperation(ctx, ops, operationID)
	if err != nil {
		return pushErr
	}
	if outcome.Effect != factory.OpEffectPending || outcome.NotObserved {
		return &Adopted{Outcome: outcome}
	}
	env, err := ops.PublishPushEnv(ctx, operationID, true)
	if err != nil {
		return &Adopted{Outcome: outcome}
	}
	if err = v.repo.runWithEnv(ctx, env, "push", "--no-follow-tags", c.Remote, refspec); err != nil {
		reconciled, lookupErr := lookupOperation(ctx, ops, operationID)
		if lookupErr != nil {
			return pushErr
		}
		return &Adopted{Outcome: reconciled}
	}
	return nil
}
