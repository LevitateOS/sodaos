package host

import (
	"context"
	"errors"
	"github.com/levitateos/sodaos/internal/application"
	"net/url"
	"strings"
)

// FJ-874: maps the setup route's typed refusal to the existing
// committed/conflict/unconfirmed outcomes. The returned errors carry the
// same fixed literals the host gate emits, so the API layer's existing
// setup-claim mapping keeps working byte-identical. Every non-claim
// error (validation, transport, 200-shape) keeps its existing behavior
// byte-identical.
func mapSetupClaimError(err error) error {
	var claim setupClaimError
	if !errors.As(err, &claim) {
		return err
	}
	switch claim.code {
	case "committed":
		return errors.New("setup claim is committed; renewal refused with the retained credential kept")
	case "conflict":
		return errors.New("setup claim conflicts; renewal refused with the retained credential kept")
	default:
		return errors.New("setup claim is unconfirmed; renewal refused with the retained credential kept")
	}
}

func (c *Client) ApplicationSetup(ctx context.Context, request application.SetupRequest) (application.SetupResult, error) {
	var result application.SetupResult
	if err := request.Validate(); err != nil {
		return result, err
	}
	if err := c.callSetupLimit(ctx, "/application/setup", request, &result, 8192); err != nil {
		return result, mapSetupClaimError(err)
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
