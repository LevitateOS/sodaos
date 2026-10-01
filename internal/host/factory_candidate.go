package host

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/project"
)

func (c *Client) PrepareCandidate(ctx context.Context, in project.FactoryCandidate) (project.PrepareState, error) {
	var out project.PrepareState
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.callLimit(ctx, "/prepare-candidate", in, &out, prepareResponseLimit)
	if err == nil && !preparationIdentityConfirmed(in.Preparation, out) {
		err = errors.New("native candidate preparation does not match its identity")
	}
	return out, err
}

func (c *Client) FactoryInspectCandidate(ctx context.Context, in project.FactoryCandidateInspect) (project.FactoryCandidateState, error) {
	var out project.FactoryCandidateState
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.call(ctx, "/factory-candidate-inspect", in, &out)
	var httpErr nativeHTTPError
	if errors.As(err, &httpErr) && httpErr.status == http.StatusConflict {
		return out, ErrRunStale
	}
	err = factoryNotFound(err)
	if err == nil && (out.ID != in.ID || out.Project != in.Project || !project.ValidContainerID(out.Container) || !project.ValidCommit(out.Candidate)) {
		err = errors.New("native candidate observation does not match its identity")
	}
	return out, err
}
