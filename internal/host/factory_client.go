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

// ErrRunStale reports a recorded run whose container incarnation no longer
// resolves. Callers refuse the read rather than serving another
// execution's bytes.
var ErrRunStale = errors.New("factory run incarnation changed")

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

func factoryOutputError(in project.FactoryOutput, out project.FactoryOutputState, err error) error {
	var httpErr nativeHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.status == http.StatusNotFound {
			return ErrRunNotFound
		}
		if httpErr.status == http.StatusConflict {
			return ErrRunStale
		}
	}
	if err == nil && (out.ID != in.ID || out.Project != in.Project || !project.ValidFactoryPhase(out.Phase)) {
		return errors.New("native factory output does not match its identity")
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

func (c *Client) FactoryHarness(ctx context.Context) (project.FactoryHarnessPin, error) {
	var out project.FactoryHarnessPin
	err := c.callLimit(ctx, "/factory-harness", struct{}{}, &out, factoryResponseLimit)
	if err == nil && out.Validate() != nil {
		err = errors.New("native factory harness pin is not usable")
	}
	return out, err
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

func (c *Client) FactoryOutput(ctx context.Context, in project.FactoryOutput) (project.FactoryOutputState, error) {
	var out project.FactoryOutputState
	err := c.callLimit(ctx, "/factory-output", in, &out, factoryResponseLimit)
	return out, factoryOutputError(in, out, err)
}

func (c *Client) FactoryTakeover(ctx context.Context, in project.FactoryTakeover) (project.TakeoverResult, error) {
	var out project.TakeoverResult
	err := c.callLimit(ctx, "/factory-takeover", in, &out, factoryResponseLimit)
	if err == nil && (out.Validate() != nil || out.ID != in.ID || out.Project != in.Project || out.Member != in.Member) {
		err = errors.New("native factory takeover did not return the admitted destination")
	}
	return out, factoryNotFound(err)
}
