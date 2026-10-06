package control

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func (c *Coordinator) exportCandidate(ctx context.Context, p factory.Publication, run factory.Run) ([]byte, *exportTerminal, *planWait) {
	state, err := c.Host.FactoryExport(ctx, project.FactoryExport{Project: p.ProjectID, ID: p.Run, Role: p.Role, Preparation: p.Preparation, Candidate: p.Candidate})
	if err != nil {
		if errors.Is(err, project.ErrFactoryExportCandidate) || errors.Is(err, project.ErrFactoryExportBounds) || isHostNotFound(err) || isHostStale(err) {
			return nil, &exportTerminal{Reason: factory.PublishReasonExportFailed}, nil
		}
		return nil, nil, waitFor("host_unavailable", "candidate export unconfirmed")
	}
	if state.ID != p.Run || state.Project != p.ProjectID || state.Candidate != p.Candidate {
		return nil, nil, waitFor("host_unavailable", "candidate export identity differs")
	}
	bundle, err := decodeExportBundle(state.Bundle)
	if err != nil {
		return nil, &exportTerminal{Reason: factory.PublishReasonExportFailed}, nil
	}
	return bundle, nil, nil
}
func decodeExportBundle(encoded string) ([]byte, error) {
	if encoded == "" || len(encoded) > 8<<20 {
		return nil, errors.New("candidate export exceeds bounds")
	}
	bundle, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(bundle) == 0 || len(bundle) > 4<<20 {
		return nil, errors.New("candidate export exceeds bounds")
	}
	return bundle, nil
}

const hostRunStale = "factory run incarnation changed"

func isHostStale(err error) bool { return err != nil && strings.Contains(err.Error(), hostRunStale) }
