package host

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/project"
)

// ErrRunNotFound reports a factory run the host never recorded. Callers
// fence unknown runs; absence of a receipt is not proof of retirement.
var ErrRunNotFound = errors.New("factory run not found")

// factoryResponseLimit bounds run observations, which carry bounded
// last-message output alongside the recorded state.
const factoryResponseLimit = 128 << 10

func factoryNotFound(err error) error {
	var httpErr nativeHTTPError
	if errors.As(err, &httpErr) && httpErr.status == http.StatusNotFound {
		return ErrRunNotFound
	}
	return err
}

func factoryStateConfirmed(in project.FactoryRun, out project.FactoryState) bool {
	return out.ID == in.ID && out.Project == in.Project && out.Role == in.Role && project.ValidFactoryPhase(out.Phase)
}

func (c *Client) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	var out project.FactoryState
	err := c.callLimit(ctx, "/factory-launch", in, &out, factoryResponseLimit)
	if err == nil && !factoryStateConfirmed(in.Run, out) {
		err = errors.New("native factory launch did not return the admitted run")
	}
	return out, err
}

func (c *Client) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	var out project.FactoryState
	err := c.callLimit(ctx, "/factory-inspect", in, &out, factoryResponseLimit)
	if err == nil && (out.ID != in.ID || out.Project != in.Project || !project.ValidFactoryPhase(out.Phase)) {
		err = errors.New("native factory observation does not match its identity")
	}
	return out, factoryNotFound(err)
}

func (c *Client) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	var out project.FactoryState
	err := c.callLimit(ctx, "/factory-stop", in, &out, factoryResponseLimit)
	if err == nil && (out.ID != in.ID || out.Project != in.Project || !project.ValidFactoryPhase(out.Phase) ||
		(out.Retirement != "" && out.Retirement != "confirmed" && out.Retirement != "uncertain")) {
		err = errors.New("native factory stop was not confirmed")
	}
	return out, err
}
