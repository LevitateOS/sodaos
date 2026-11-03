package forgejo

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

type FountainIdentityRegistry interface {
	FountainLedgerHost(context.Context) (collaboration.Host, error)
	CollaborationRepositoryOnHost(context.Context, collaboration.HostID, collaboration.RepositoryID) (collaboration.RepositoryRef, error)
	CollaborationRepositoryByExternalID(context.Context, collaboration.HostID, string) (collaboration.RepositoryRef, error)
	ObserveCollaborationRepository(context.Context, collaboration.HostID, string, string) (collaboration.RepositoryRef, error)
	CollaborationActorOnHost(context.Context, collaboration.HostID, collaboration.ActorID) (collaboration.ActorRef, error)
	CollaborationActorByExternalID(context.Context, collaboration.HostID, string) (collaboration.ActorRef, error)
	ObserveCollaborationActor(context.Context, collaboration.HostID, string, string) (collaboration.ActorRef, error)
	CollaborationTeamOnHost(context.Context, collaboration.HostID, collaboration.TeamID) (collaboration.TeamRef, error)
	CollaborationTeamByExternalID(context.Context, collaboration.HostID, string) (collaboration.TeamRef, error)
	ObserveCollaborationTeam(context.Context, collaboration.HostID, string, string) (collaboration.TeamRef, error)
}

// FountainIdentities binds an adapter to the explicitly adopted appliance host.
// It translates observations, never permissions, credentials or issue numbers.
// Outbound lookups cannot allocate or fall back to equal native/local numbers.
type FountainIdentities struct {
	registry FountainIdentityRegistry
	host     collaboration.HostID
}

// HostID is the immutable, adopted provider namespace, never an authority grant.
func (s *FountainIdentities) HostID() collaboration.HostID {
	if s == nil {
		return ""
	}
	return s.host
}

func NewFountainIdentities(ctx context.Context, registry FountainIdentityRegistry, host collaboration.HostID) (*FountainIdentities, error) {
	if registry == nil || !collaboration.ValidID(string(host)) {
		return nil, collaboration.ErrIdentityConflict
	}
	bound, err := registry.FountainLedgerHost(ctx)
	if err != nil {
		return nil, err
	}
	if bound.ID != host || bound.Provider != collaboration.Fountain {
		return nil, collaboration.ErrIdentityConflict
	}
	return &FountainIdentities{registry: registry, host: host}, nil
}

func (s *FountainIdentities) NativeRepository(ctx context.Context, local collaboration.RepositoryID) (string, error) {
	if s == nil || s.registry == nil || local < 1 {
		return "", collaboration.ErrIdentityConflict
	}
	ref, err := s.registry.CollaborationRepositoryOnHost(ctx, s.host, local)
	if err != nil {
		return "", err
	}
	if _, err := originalCreationID(ref.ExternalID); err != nil {
		return "", err
	}
	return ref.ExternalID, nil
}

func (s *FountainIdentities) NativeActor(ctx context.Context, local collaboration.ActorID) (string, error) {
	if s == nil || s.registry == nil || local < 1 {
		return "", collaboration.ErrIdentityConflict
	}
	ref, err := s.registry.CollaborationActorOnHost(ctx, s.host, local)
	if err != nil {
		return "", err
	}
	if _, err := originalCreationID(ref.ExternalID); err != nil {
		return "", err
	}
	return ref.ExternalID, nil
}

func (s *FountainIdentities) NativeTeam(ctx context.Context, local collaboration.TeamID) (string, error) {
	if s == nil || s.registry == nil || local < 1 {
		return "", collaboration.ErrIdentityConflict
	}
	ref, err := s.registry.CollaborationTeamOnHost(ctx, s.host, local)
	if err != nil {
		return "", err
	}
	if _, err := originalCreationID(ref.ExternalID); err != nil {
		return "", err
	}
	return ref.ExternalID, nil
}

// LocalRepository observes a subject named by an authenticated provider event.
// An absent display label remains a placeholder; existing metadata is retained.
func (s *FountainIdentities) LocalRepository(ctx context.Context, native int64) (collaboration.RepositoryID, error) {
	if s == nil || s.registry == nil || native < 1 {
		return 0, collaboration.ErrIdentityConflict
	}
	external := strconv.FormatInt(native, 10)
	ref, err := s.registry.CollaborationRepositoryByExternalID(ctx, s.host, external)
	if errors.Is(err, sql.ErrNoRows) {
		ref, err = s.registry.ObserveCollaborationRepository(ctx, s.host, external, "")
	}
	return ref.ID, err
}

func (s *FountainIdentities) LocalActor(ctx context.Context, native string) (collaboration.ActorID, error) {
	if s == nil || s.registry == nil {
		return 0, collaboration.ErrIdentityConflict
	}
	if _, err := originalCreationID(native); err != nil {
		return 0, err
	}
	ref, err := s.registry.CollaborationActorByExternalID(ctx, s.host, native)
	if errors.Is(err, sql.ErrNoRows) {
		ref, err = s.registry.ObserveCollaborationActor(ctx, s.host, native, "")
	}
	return ref.ID, err
}

// LocalTeam observes a subject supplied by an authenticated provider source.
// The numeric native namespace stays separate from users and grants no access.
func (s *FountainIdentities) LocalTeam(ctx context.Context, native string) (collaboration.TeamID, error) {
	if s == nil || s.registry == nil {
		return 0, collaboration.ErrIdentityConflict
	}
	if _, err := originalCreationID(native); err != nil {
		return 0, err
	}
	ref, err := s.registry.CollaborationTeamByExternalID(ctx, s.host, native)
	if errors.Is(err, sql.ErrNoRows) {
		ref, err = s.registry.ObserveCollaborationTeam(ctx, s.host, native, "")
	}
	return ref.ID, err
}

// LocalOperationOutcome projects known provider subjects without allocating
// identities from an opaque record. Receipt bytes and zero-scope absence or
// pre-submit cancellation tombstones retain their original meaning.
func (s *FountainIdentities) LocalOperationOutcome(ctx context.Context, outcome factory.OperationOutcome) (factory.OperationOutcome, error) {
	if s == nil || s.registry == nil {
		return factory.OperationOutcome{}, ErrUnavailable
	}
	if outcome.ActorID == 0 && outcome.RepositoryID == 0 && (outcome.NotObserved || outcome.Kind == "" && outcome.Effect == factory.OpEffectNotCommitted && outcome.Cancellation == factory.OpCancelCancelled) {
		return outcome, nil
	}
	actor, err := s.registry.CollaborationActorByExternalID(ctx, s.host, strconv.FormatInt(outcome.ActorID, 10))
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	repository, err := s.registry.CollaborationRepositoryByExternalID(ctx, s.host, strconv.FormatInt(outcome.RepositoryID, 10))
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	outcome.ActorID, outcome.RepositoryID = int64(actor.ID), int64(repository.ID)
	return outcome, nil
}
