//go:build !linux

package terminal

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

var errFactoryUnavailable = errors.New("supervised factory execution requires linux")

func (s *Service) FactoryCodexReserve(context.Context, domain.FactoryRun, identity.Lease, string, int64) (identity.Binding, FactoryCodexPaths, error) {
	return identity.Binding{}, FactoryCodexPaths{}, errFactoryUnavailable
}

func (s *Service) FactoryCodexStart(context.Context, identity.Lease, []byte, []byte) error {
	return errFactoryUnavailable
}

func (s *Service) FactoryCodexWait(context.Context, identity.Lease) (int, string, error) {
	return -1, "", errFactoryUnavailable
}

func (s *Service) FactoryCodexValidate(context.Context, identity.Lease) error {
	return errFactoryUnavailable
}

func (s *Service) FactoryCodexStop(context.Context, identity.Lease) error {
	return errFactoryUnavailable
}

func (s *Service) FactoryCodexCapture(context.Context, identity.Lease) ([]byte, error) {
	return nil, errFactoryUnavailable
}

func (s *Service) FactoryCodexFinish(context.Context, identity.Lease) ([]byte, error) {
	return nil, errFactoryUnavailable
}

func (s *Service) FactoryCodexLive(context.Context, identity.Binding) bool { return false }

func (s *Service) FactoryCodexStopUnbound(context.Context, domain.FactoryRun) error {
	return errFactoryUnavailable
}
