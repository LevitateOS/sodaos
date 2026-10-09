package host

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/project"
)

// prepareResponseLimit bounds preparation observations, which carry bounded
// setup and check logs alongside the recorded state.
const prepareResponseLimit = 262144

func preparationIdentityConfirmed(in project.Preparation, out project.PrepareState) bool {
	return out.ID == in.ID && out.Project == in.Project && out.Role == in.Role &&
		project.ValidPreparePhase(out.Phase) && project.ValidContainerID(out.Container) &&
		out.SourceCommit == in.SourceCommit && out.SetupDigest == in.SetupDigest
}

func (c *Client) Prepare(ctx context.Context, in project.Prepare) (project.PrepareState, error) {
	var out project.PrepareState
	err := c.callLimit(ctx, "/prepare", in, &out, prepareResponseLimit)
	if err == nil && !preparationIdentityConfirmed(in.Preparation, out) {
		err = errors.New("native preparation result does not match its identity")
	}
	return out, err
}

func (c *Client) InspectPreparation(ctx context.Context, in project.PrepareInspect) (project.PrepareState, error) {
	var out project.PrepareState
	err := c.callLimit(ctx, "/prepare-inspect", in, &out, prepareResponseLimit)
	var httpErr nativeHTTPError
	if errors.As(err, &httpErr) && httpErr.status == http.StatusNotFound {
		return out, project.ErrPreparationNotFound
	}
	if err == nil && (out.ID != in.ID || out.Project != in.Project || !project.ValidPreparePhase(out.Phase) ||
		!project.ValidContainerID(out.Container)) {
		err = errors.New("native preparation observation does not match its identity")
	}
	return out, err
}

func (c *Client) StopPreparation(ctx context.Context, in project.PrepareStop) (project.PrepareState, error) {
	var out project.PrepareState
	err := c.callLimit(ctx, "/prepare-stop", in, &out, prepareResponseLimit)
	if err == nil && (out.ID != in.ID || out.Project != in.Project || !out.Stopped ||
		(out.Retirement != "confirmed" && out.Retirement != "uncertain")) {
		err = errors.New("native preparation stop was not confirmed")
	}
	return out, err
}

func (c *Client) HoldPreparation(ctx context.Context, in project.PrepareHold) (project.HoldState, error) {
	var out project.HoldState
	err := c.call(ctx, "/prepare-hold", in, &out)
	if err == nil && out.Active != in.Hold {
		err = errors.New("native maintenance hold outcome not confirmed")
	}
	return out, err
}
