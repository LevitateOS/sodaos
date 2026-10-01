//go:build !linux

package terminal

import (
	"context"
	"errors"
)

// FactoryExportBundle is unavailable off Linux: export reads through the
// native container boundary.
func (s *Service) FactoryExportBundle(ctx context.Context, projectID, recorded, role, preparation, candidate string) ([]byte, error) {
	return nil, errors.New("factory export requires Linux")
}
