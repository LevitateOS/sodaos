package host

import (
	"context"
	"errors"
	"github.com/levitateos/sodaos/internal/application"
	"net/url"
	"strings"
)

func (c *Client) ApplicationSetup(ctx context.Context, request application.SetupRequest) (application.SetupResult, error) {
	var result application.SetupResult
	if err := request.Validate(); err != nil {
		return result, err
	}
	if err := c.callLimit(ctx, "/application/setup", request, &result, 8192); err != nil {
		return result, err
	}
	if result.Operation != request.Setup.ID || result.Observation.Validate() != nil {
		return application.SetupResult{}, errors.New("native setup identity differs")
	}
	if request.ExpectedContainer != "" && request.ExpectedContainer != result.Observation.Container {
		return application.SetupResult{}, errors.New("native setup generation differs")
	}
	if result.SetupURL != "" {
		u, err := url.Parse(result.SetupURL)
		if err != nil || strings.Split(result.SetupURL, "#")[0] != request.Setup.Application.Origin+"/setup" || u.Fragment == "" {
			return application.SetupResult{}, errors.New("invalid native setup link")
		}
	}
	if result.Receipt != nil && result.Receipt.Validate(request.Setup) != nil {
		return application.SetupResult{}, errors.New("native setup receipt differs")
	}
	if request.Action == "complete" && (result.Receipt == nil || result.Application == nil || result.Observation.Running || result.Observation.Ready) {
		return application.SetupResult{}, errors.New("native setup completion unconfirmed")
	}
	if request.Action == "abort" && (result.Observation.Running || result.Observation.Ready) {
		return application.SetupResult{}, errors.New("native setup writer did not stop")
	}
	return result, nil
}
