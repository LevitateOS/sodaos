package host

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/project"
)

// The wire envelope stays bounded at 3x the aggregate raw bound: 128 KiB
// of context bytes base64-expands past 170 KiB plus JSON framing, so the
// previous 96 KiB cap cannot carry one approved 32 KiB file with setup,
// check and instructions.
const factoryContextResponseLimit = 384 << 10

// IsPreparationContextBoundsExceeded reports whether err is the genuine
// typed explicit-bounds refusal for a /prepare-context read: the native
// route answers HTTP 413 (preparation context exceeds bounds) when and
// only when the selection cannot fit the file-count, per-file or
// aggregate context bounds. This function only ever calls
// /prepare-context, so a 413 here is that route's typed signal — never a
// sibling route's status, never an error-string guess. Every other error
// (generic 500s, malformed 200s, validation failures) fails closed.
func IsPreparationContextBoundsExceeded(err error) bool {
	var httpErr nativeHTTPError
	return errors.As(err, &httpErr) && httpErr.status == http.StatusRequestEntityTooLarge
}

// ReadPreparationContext reads bounded repository instructions, selected
// files and the exact candidate diff from the prepared native checkout.
func (c *Client) ReadPreparationContext(ctx context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
	var out project.FactoryPreparationContext
	if err := in.Validate(); err != nil || !project.ValidFactoryRole(role) {
		return out, errors.New("invalid factory preparation context selection")
	}
	ctx, cancel := context.WithDeadline(ctx, in.NotAfter)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	in.NotAfter, _ = ctx.Deadline()
	err := c.callLimit(ctx, "/prepare-context", in, &out, factoryContextResponseLimit)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return project.FactoryPreparationContext{}, ctxErr
	}
	if err == nil {
		err = out.ValidateFor(in, role)
	}
	if err != nil {
		return project.FactoryPreparationContext{}, err
	}
	return out, nil
}
