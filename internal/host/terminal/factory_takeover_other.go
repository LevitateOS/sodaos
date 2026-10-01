//go:build !linux

package terminal

import (
	"context"
	"errors"
)

// FactoryTakeoverCopy is unavailable off Linux: takeover copies through
// the native container boundary.
func (s *Service) FactoryTakeoverCopy(ctx context.Context, projectID, recorded, role, preparation, member, run string) (string, bool, error) {
	return "", false, errors.New("factory takeover requires Linux")
}
