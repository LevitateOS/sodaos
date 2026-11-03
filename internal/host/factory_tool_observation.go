package host

import (
	"context"
	"errors"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

var _ execution.ToolObserver = (*FactoryObserver)(nil)

func (c *Client) InspectTool(ctx context.Context, family string) (execution.ToolObservation, error) {
	return NewFactoryObserver(c, nil).InspectTool(ctx, family)
}

func (o *FactoryObserver) InspectTool(ctx context.Context, family string) (execution.ToolObservation, error) {
	if !project.ValidHarnessFamily(family) {
		return execution.ToolObservation{}, execution.ErrProvenance
	}
	if o == nil {
		return execution.ToolObservation{}, errors.New("appliance tool observer unavailable")
	}
	reader, ok := o.host.(interface {
		FactoryHarness(context.Context, string) (project.FactoryHarnessPin, error)
	})
	if !ok {
		return execution.ToolObservation{}, errors.New("appliance tool observer unavailable")
	}
	pin, err := reader.FactoryHarness(ctx, family)
	if err != nil {
		return execution.ToolObservation{}, err
	}
	observed := execution.ToolObservation{Kind: collaboration.Appliance, Appliance: &execution.ApplianceTool{Harness: pin.Harness, Version: pin.Version, SHA256: pin.SHA256, Image: pin.Image}}
	if err := observed.ValidateFor(family); err != nil {
		return execution.ToolObservation{}, err
	}
	return observed, nil
}
