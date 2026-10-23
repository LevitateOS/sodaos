package host

import (
	"context"
	"errors"
	"github.com/levitateos/sodaos/internal/application"
)

func (c *Client) ApplicationSnapshots(ctx context.Context, request application.SnapshotRequest) (application.SnapshotInventory, error) {
	var out application.SnapshotInventory
	if err := request.Validate(); err != nil {
		return out, err
	}
	if err := c.callLimit(ctx, "/application/snapshots", request, &out, 128<<10); err != nil {
		return out, err
	}
	if len(out.Snapshots) > 64 {
		return out, errors.New("native snapshot inventory exceeded its bound")
	}
	seen := map[string]bool{}
	for _, snapshot := range out.Snapshots {
		if snapshot.Validate() != nil || snapshot.Application.ID != request.Application.ID || seen[snapshot.ID] {
			return out, errors.New("native snapshot inventory differs from the application")
		}
		seen[snapshot.ID] = true
	}
	if (request.Action == "backup" || request.Action == "inspect") && !seen[request.ID] {
		return out, errors.New("native backup was not confirmed")
	}
	return out, nil
}
