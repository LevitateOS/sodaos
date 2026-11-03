package forgejo

import (
	"context"
	"errors"
	"strconv"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

// ReadCreations resolves a canonical repository on the adopted host, reads as
// the enrolled native service actor, and maps original creators back to Factory
// actors. Native issue IDs, recovery positions and coverage facts stay intact.
func (o *ServiceObserver) ReadCreations(ctx context.Context, repository, after int64, limit int) (factory.CreationRecoveryBatch, error) {
	if o == nil || o.Identities == nil || repository <= 0 || after < 0 || limit < 1 || limit > extensions.CreationRecoveryPageLimit {
		return factory.CreationRecoveryBatch{}, ErrInvalidResponse
	}
	native, err := o.Identities.NativeRepository(ctx, collaboration.RepositoryID(repository))
	if err != nil {
		return factory.CreationRecoveryBatch{}, err
	}
	nativeRepository, err := originalCreationID(native)
	if err != nil {
		return factory.CreationRecoveryBatch{}, err
	}
	actor, err := o.ensureActor(ctx)
	if err != nil {
		return factory.CreationRecoveryBatch{}, err
	}
	client, err := o.ensureClient(ctx)
	if err != nil {
		return factory.CreationRecoveryBatch{}, err
	}
	afterID := ""
	if after > 0 {
		afterID = strconv.FormatInt(after, 10)
	}
	page, err := client.ReadCreations(ctx, o.Credential(), extensions.CreationRecoveryRequest{RepositoryID: native, ActorID: actor, AfterID: afterID, Limit: limit})
	if err != nil {
		return factory.CreationRecoveryBatch{}, err
	}
	batch, err := creationRecoveryBatch(page, nativeRepository, after, limit)
	if err != nil {
		return factory.CreationRecoveryBatch{}, err
	}
	batch.Repository = repository
	for i := range batch.Events {
		localActor, err := o.Identities.LocalActor(ctx, strconv.FormatInt(batch.Events[i].Creator, 10))
		if err != nil {
			return factory.CreationRecoveryBatch{}, err
		}
		batch.Events[i].Repository = repository
		batch.Events[i].Creator = int64(localActor)
	}
	return batch, nil
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
