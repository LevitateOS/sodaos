package forgejo

import (
	"context"
	"errors"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

var _ factoryservice.CollaborationCapabilitySource = HumanReads{}

func (s HumanReads) ReadCapabilities(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, q factoryservice.CapabilityQuery) (collaboration.Capabilities, error) {
	var empty collaboration.Capabilities
	if actor < 1 || repo < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrCapabilitiesQuery
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrCapabilitiesUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repo)
	if err != nil {
		return empty, humanCapabilitiesError(err)
	}
	index := ""
	if q.Number > 0 {
		index = strconv.FormatInt(q.Number, 10)
	}
	native, err := s.Authority.RepositoryCapabilities(ctx, nativeRepo, extensions.RepositoryCapabilitiesRequest{Subject: q.Subject, Index: index})
	if err != nil {
		return empty, humanCapabilitiesError(err)
	}
	r := collaboration.Capabilities{Repository: repo, Actor: actor, Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: time.Now().UTC()}, Actions: map[collaboration.Action]collaboration.Capability{}}
	if q.Subject == "issue" {
		r.Issue = &collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: repo, ExternalID: native.ResourceID, Number: q.Number}
	}
	if q.Subject == "pull" {
		r.Pull = &collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repo, ExternalID: native.ResourceID, Number: q.Number}
	}
	if native.Head != "" {
		head := collaboration.CommitID(native.Head)
		r.Head = &head
	}
	if native.Base != "" {
		base := collaboration.CommitID(native.Base)
		r.Base = &base
	}
	for key, fact := range native.Actions {
		r.Actions[collaboration.Action(key)] = collaboration.Capability{State: collaboration.Availability(fact.State), Reason: fact.Reason}
	}
	if native.MergeMethods != nil {
		r.MergeMethods = make(map[string]collaboration.Capability, len(native.MergeMethods))
		for method, fact := range native.MergeMethods {
			r.MergeMethods[method] = collaboration.Capability{State: collaboration.Availability(fact.State), Reason: fact.Reason}
		}
	}
	if q.Subject == "issue" {
		// The state operation has its own permission and blocker rules. The
		// general issue-edit capability does not describe Close or Reopen.
		snapshot, err := s.Authority.RepositoryStateSnapshot(ctx, nativeRepo, extensions.RepositoryStateSnapshotRequest{Subject: q.Subject, Index: index})
		if err == nil {
			if snapshot.SubjectID != native.ResourceID || snapshot.ActorID != nativeActor {
				return empty, factoryservice.ErrCapabilitiesChanged
			}
			r.StateActions = map[string]collaboration.Capability{
				"close":  {State: collaboration.Availability(snapshot.CloseAction.State), Reason: snapshot.CloseAction.Reason},
				"reopen": {State: collaboration.Availability(snapshot.ReopenAction.State), Reason: snapshot.ReopenAction.Reason},
			}
		}
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanCapabilitiesError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrCapabilitiesMissing
	}
	return r, nil
}
func humanCapabilitiesError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrCapabilitiesMissing
	case errors.Is(err, extensions.ErrRepositoryCapabilitiesQuery):
		return factoryservice.ErrCapabilitiesQuery
	case errors.Is(err, extensions.ErrRepositoryCapabilitiesChanged):
		return factoryservice.ErrCapabilitiesChanged
	default:
		return err
	}
}
