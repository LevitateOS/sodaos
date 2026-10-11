package forgejo

import (
	"context"
	"errors"

	extensions "forgejo.org/extension-sdk"
)

// ReadCreations resolves the enrolled native actor and shares its existing
// restricted PAT/background admission with ordinary readiness observations.
func (o *ServiceObserver) ReadCreations(ctx context.Context, repository, after string, limit int) (extensions.CreationRecoveryPage, error) {
	actor, err := o.ensureActor(ctx)
	if err != nil {
		return extensions.CreationRecoveryPage{}, err
	}
	client, err := o.ensureClient(ctx)
	if err != nil {
		return extensions.CreationRecoveryPage{}, err
	}
	return client.ReadCreations(ctx, o.Credential(), extensions.CreationRecoveryRequest{RepositoryID: repository, ActorID: actor, AfterID: after, Limit: limit})
}

// ReadCreations reads retained native creation provenance using the same
// credential-bound background admission as snapshots and operations. A page
// ending at retained history does not assert complete creation history.
func (b *ServiceBackground) ReadCreations(ctx context.Context, credential extensions.CredentialFile, req extensions.CreationRecoveryRequest) (extensions.CreationRecoveryPage, error) {
	if err := extensions.ValidateCreationRecoveryRequest(req); err != nil {
		return extensions.CreationRecoveryPage{}, err
	}
	secret, err := backgroundSecret(credential)
	if err != nil {
		return extensions.CreationRecoveryPage{}, err
	}
	request := struct {
		extensions.CreationRecoveryRequest
		Token string `json:"token"`
	}{CreationRecoveryRequest: req, Token: secret}
	var page extensions.CreationRecoveryPage
	if err := b.call(ctx, extensions.BackgroundCreationsPath, request, &page); err != nil {
		return extensions.CreationRecoveryPage{}, err
	}
	if page.RepositoryID != req.RepositoryID || len(page.Creations) > req.Limit {
		return extensions.CreationRecoveryPage{}, errors.New("invalid background response")
	}
	return page, nil
}
