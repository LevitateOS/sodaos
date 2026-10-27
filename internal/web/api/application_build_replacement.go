package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/factory"
)

func (s *API) resolveApplicationBuildReplacement(r *http.Request, current application.Record, input *applicationAction) error {
	if input.ReplacementBuild == "" {
		return nil
	}
	if input.Action != "replace" || input.Replacement != nil || !factory.ValidID(input.ReplacementBuild) {
		return errors.New("invalid build replacement")
	}
	repository, err := s.Store.ApplicationProject(r.Context(), current.Application.ID)
	if err != nil || repository <= 0 {
		return errors.New("application has no source project")
	}
	build, err := s.Store.ApplicationBuild(r.Context(), repository, input.ReplacementBuild)
	if err != nil {
		return err
	}
	request, delivery, err := s.Store.VerifiedFactoryBuildInput(r.Context(), repository, build.Delivery.DeliveryID, build.Artifact.Environment)
	if err != nil {
		return err
	}
	artifact := build.Artifact
	environment, envErr := s.Store.ApplicationEnvironment(r.Context(), current.Application.ID)
	if envErr != nil || artifact.Environment != environment || delivery != build.Delivery || artifact.ValidateFor(request) != nil || artifact.Name != current.Application.Name || artifact.Origin != current.Application.Origin || artifact.Image == current.Application.Image {
		return errors.New("build does not match current application and delivery")
	}
	next := current.Application
	next.Revision++
	next.ArtifactSHA256 = artifact.ArtifactSHA256
	next.BundleSHA256 = artifact.BundleSHA256
	next.ToolsSHA256 = artifact.ToolsSHA256
	next.RuntimeImage = artifact.RuntimeImage
	next.Image = artifact.Image
	input.Replacement = &next
	return current.Application.Replacement(next)
}
