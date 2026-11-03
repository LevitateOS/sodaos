package forgejo

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

var _ factoryservice.CollaborationCheckSource = HumanReads{}

func (s HumanReads) ReadChecks(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, number int64, q factoryservice.ChecksQuery) (collaboration.PullChecks, error) {
	var empty collaboration.PullChecks
	if actor < 1 || repo < 1 || number < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrChecksQuery
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrChecksUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repo)
	if err != nil {
		return empty, humanChecksError(err)
	}
	native, err := s.Authority.RepositoryChecks(ctx, nativeRepo, extensions.RepositoryChecksRequest{Kind: q.Kind, Index: strconv.FormatInt(number, 10), Head: string(q.Head), Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return empty, humanChecksError(err)
	}
	host, err := s.Identities.registry.FountainLedgerHost(ctx)
	if err != nil {
		return empty, err
	}
	if host.ID != s.Identities.host {
		return empty, ErrInvalidResponse
	}
	observed := time.Now().UTC()
	observation := func(v any) collaboration.Observation {
		return collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(v), ObservedAt: observed}
	}
	r := collaboration.PullChecks{Pull: collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repo, Number: number, ExternalID: native.PullID}, Commit: collaboration.CommitID(native.Head), Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: collaboration.Revision("fountain-checks:" + native.Revision), ObservedAt: observed}, Checks: collaboration.Page[collaboration.Check]{Observation: observation(native), Items: []collaboration.Check{}, Next: native.NextCursor, Complete: native.NextCursor == ""}, MissingRequired: native.MissingRequired, InvalidRequired: native.InvalidRequired, StatusGate: native.StatusGate}
	r.Kind = native.Kind
	if p := native.Protection; p != nil {
		r.Protection = &collaboration.BranchProtection{Repository: repo, Observation: observation(p), Branch: p.Branch, Protected: &p.Protected, StatusChecks: &p.StatusChecks, RequiredContexts: p.RequiredContexts, RequiredApprovals: &p.RequiredApprovals, BlockRejected: &p.BlockRejected, BlockReviewRequests: &p.BlockReviewRequests, RequireUpToDate: &p.RequireUpToDate, RequireSigned: &p.RequireSigned, DismissStale: &p.DismissStale, IgnoreStale: &p.IgnoreStale, ApplyToAdmins: &p.ApplyToAdmins, ProtectedFiles: p.ProtectedFiles, UnprotectedFiles: p.UnprotectedFiles}
	}
	for _, c := range native.Items {
		state, conclusion := "unknown", "unknown"
		switch c.State {
		case "pending":
			state = "pending"
		case "success":
			state, conclusion = "completed", "success"
		case "failure", "error":
			state, conclusion = "completed", "failure"
		case "warning":
			state = "completed"
		}
		var job *collaboration.CheckJob
		if j := c.Job; j != nil {
			state, conclusion = "unknown", "unknown"
			switch c.State {
			case "waiting", "blocked":
				state = "pending"
			case "running":
				state = "running"
			case "success", "failure", "skipped", "cancelled":
				state, conclusion = "completed", c.State
			}
			started, err := checkJobTimestamp(j.StartedAt)
			if err != nil {
				return empty, ErrInvalidResponse
			}
			stopped, err := checkJobTimestamp(j.StoppedAt)
			if err != nil {
				return empty, ErrInvalidResponse
			}
			job = &collaboration.CheckJob{RunID: j.RunID, RunNumber: j.RunNumber, Workflow: j.Workflow, Ref: j.Ref, Event: j.Event, TriggerEvent: j.TriggerEvent, RunState: j.RunState, NeedApproval: &j.NeedApproval, WorkflowCommit: collaboration.CommitID(j.WorkflowCommit), JobID: j.JobID, Attempt: j.Attempt, StartedAt: started, StoppedAt: stopped}
		}
		r.Checks.Items = append(r.Checks.Items, collaboration.Check{ExternalID: native.Kind + ":" + c.ID, Repository: repo, Observation: observation(c), Commit: r.Commit, Name: c.Name, Kind: native.Kind, Job: job, ProviderState: c.State, Description: c.Description, State: state, Conclusion: conclusion, Required: c.Required, DetailsURL: checksURL(host.APIBase, c.DetailsURL)})
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanChecksError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrChecksMissing
	}
	return r, nil
}
func checkJobTimestamp(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
func checksURL(base, raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return ""
	}
	if !u.IsAbs() {
		if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
			return ""
		}
		origin, err := url.Parse(base)
		if err != nil {
			return ""
		}
		u = origin.ResolveReference(u)
	}
	if u.Host == "" || u.Scheme != "https" && u.Scheme != "http" {
		return ""
	}
	return u.String()
}
func humanChecksError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrChecksMissing
	case errors.Is(err, extensions.ErrChecksQuery):
		return factoryservice.ErrChecksQuery
	case errors.Is(err, extensions.ErrChecksChanged):
		return factoryservice.ErrChecksChanged
	case errors.Is(err, extensions.ErrChecksTooLarge):
		return factoryservice.ErrChecksTooLarge
	default:
		return err
	}
}
