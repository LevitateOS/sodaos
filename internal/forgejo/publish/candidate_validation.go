package publish

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
)

// ValidatedRepo is one validated candidate workspace: the verified bundle
// imported into a fresh bare repository with its exact candidate,
// ancestry, protected paths and credential material checked. It performs
// no push; PushBranch publishes exactly its candidate under a registered
// operation.
type ValidatedRepo struct {
	repo    *repository
	cleanup func()
	request Request
}

// PrepareValidated imports and validates one candidate bundle, returning
// the workspace for a registered push. The caller closes it. Validation
// verdicts refuse terminally; transport failures stay errors for a later
// pass.
func (c Config) PrepareValidated(ctx context.Context, r Request) (*ValidatedRepo, error) {
	if err := c.Validate(); err != nil {
		return nil, &Refusal{Reason: "invalid_publisher"}
	}
	if err := r.Validate(); err != nil {
		return nil, &Refusal{Reason: "candidate_invalid"}
	}
	git, cleanup, err := c.prepare(ctx, r)
	if err != nil {
		return nil, err
	}
	validated := &ValidatedRepo{repo: git, cleanup: cleanup, request: r}
	if err = verifyCandidateBundle(ctx, git); err != nil {
		cleanup()
		return nil, err
	}
	if err = git.validateCandidate(ctx, r, c.ProtectedPaths); err != nil {
		cleanup()
		return nil, validationVerdict(err)
	}
	if err = git.checkCredentials(ctx, r); err != nil {
		cleanup()
		return nil, validationVerdict(err)
	}
	return validated, nil
}

// Bundle verification is a local Git verdict on the supplied bytes.
// Startup failures and cancellation are infrastructure errors instead.
func verifyCandidateBundle(ctx context.Context, git *repository) error {
	err := git.command(ctx, "bundle", "verify", filepath.Join(git.dir, "candidate.bundle")).Run()
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var exited *exec.ExitError
	if errors.As(err, &exited) && exited.ExitCode() == 1 {
		return &Refusal{Reason: "candidate_invalid"}
	}
	return errors.New("publication bundle verification unavailable")
}

// validationVerdict maps candidate checks to terminal refusals. Git
// transport failures and cancellation stay errors; every other check
// outcome is a verdict on the recorded bytes.
func validationVerdict(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err.Error() == "publication Git operation failed" || err.Error() == "publication credential scan failed" {
		return err
	}
	return &Refusal{Reason: "candidate_invalid"}
}

// Close removes the validated workspace.
func (v *ValidatedRepo) Close() {
	if v != nil && v.cleanup != nil {
		v.cleanup()
	}
}
