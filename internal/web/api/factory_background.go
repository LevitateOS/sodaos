package api

import (
	"github.com/levitateos/factory/pkg/factory/control"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

// FactoryApplication supplies this host's ports to the shared application.
// It references the same coordinator and store used by background continuation.
// Judgment availability is resolved on each pass so connect/withdraw needs no
// restart. Neither composition nor a read of collaborators makes a model call.
func (s *API) FactoryApplication() factoryservice.Application {
	configuration := s.Config
	application := factoryservice.Application{Store: s.Store, Coordinator: s.Coordinator, Chooser: func() control.ProductJudgment {
		if configuration == nil || !control.TypeSafeKeyReadable(configuration.TypeSafeAPIKeyFile) {
			return nil
		}
		return &control.TypeSafePriority{APIKeyFile: configuration.TypeSafeAPIKeyFile, Model: configuration.TypeSafeModel}
	}}
	if s.issueMakerNative() != nil {
		application.IssueEffects = s.capturedIssueEffects()
	}
	return application
}
