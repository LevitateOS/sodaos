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

func factoryOutputError(err error) error {
	var httpErr nativeHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.status == http.StatusNotFound {
			return ErrRunNotFound
		}
		if httpErr.status == http.StatusConflict {
			return ErrRunStale
		}
	}
	return err
}

func (c *Client) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	var out project.FactoryState
	err := c.callLimit(ctx, "/factory-launch", in, &out, factoryResponseLimit)
	return out, err
}

func (c *Client) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	var out project.FactoryState
	err := c.callLimit(ctx, "/factory-inspect", in, &out, factoryResponseLimit)
	return out, factoryNotFound(err)
}

func (c *Client) FactoryHarness(ctx context.Context, family string) (project.FactoryHarnessPin, error) {
	var out project.FactoryHarnessPin
	err := c.callLimit(ctx, "/factory-harness", struct {
		Harness string `json:"harness"`
	}{Harness: family}, &out, factoryResponseLimit)
	return out, err
}

func (c *Client) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	var out project.FactoryState
	err := c.callLimit(ctx, "/factory-stop", in, &out, factoryResponseLimit)
	return out, err
}

func (c *Client) FactoryOutput(ctx context.Context, in project.FactoryOutput) (project.FactoryOutputState, error) {
	var out project.FactoryOutputState
	err := c.callLimit(ctx, "/factory-output", in, &out, factoryResponseLimit)
	return out, factoryOutputError(err)
}

// factoryTaskEventsResponseLimit bounds one paged task journal response:
// whole typed events under the existing 64 KiB response cap.
const factoryTaskEventsResponseLimit = 64 << 10

func (c *Client) FactoryTaskEvents(ctx context.Context, in project.FactoryTaskEvents) (project.FactoryTaskEventsState, error) {
	var out project.FactoryTaskEventsState
	err := c.callLimit(ctx, "/factory-task-events", in, &out, factoryTaskEventsResponseLimit)
	return out, factoryNotFound(err)
}

// factoryExportResponseLimit bounds one export response: the base64
// 4MB bundle plus its JSON envelope.
const factoryExportResponseLimit = 8 << 20

func (c *Client) FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	var out project.FactoryExportState
	err := c.callLimit(ctx, "/factory-export", in, &out, factoryExportResponseLimit)
	return out, factoryExportError(err)
}

func factoryExportError(err error) error {
	var httpErr nativeHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.status == http.StatusNotFound {
			return ErrRunNotFound
		}
		if httpErr.status == http.StatusConflict {
			return ErrRunStale
		}
		if httpErr.status == http.StatusUnprocessableEntity {
			return project.ErrFactoryExportCandidate
		}
		if httpErr.status == http.StatusRequestEntityTooLarge {
			return project.ErrFactoryExportBounds
		}
	}
	return err
}

func (c *Client) FactoryTakeover(ctx context.Context, in project.FactoryTakeover) (project.TakeoverResult, error) {
	var out project.TakeoverResult
	err := c.callLimit(ctx, "/factory-takeover", in, &out, factoryResponseLimit)
	return out, factoryNotFound(err)
}
