package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// Readiness sweep bounds. One sweep covers bounded repositories, pages
// and recorded failures; anything past a bound waits for a later sweep.
const (
	MaxReconcilePages = 4
	MaxSweepRepos     = 32
	MaxSweepFailures  = 32
)

// ReadinessSweepUnavailable is the bounded operational reason when native
// observation fails during a sweep or assessment cannot run now.
const ReadinessSweepUnavailable = "observation_unavailable"

// SweepFailure names one issue a sweep could not assess and why.
type SweepFailure struct {
	Issue  int64  `json:"issue"`
	Reason string `json:"reason"`
}

// ReadinessSweep is the bounded outcome of sweeping one repository:
// enumerated, assessed, changed and skipped issue counts, recorded
// failures, and flags for busy native state, truncation and skips.
type ReadinessSweep struct {
	Failures      []SweepFailure `json:"failures,omitempty"`
	Repository    int64          `json:"repository,string"`
	Seen          int64          `json:"seen"`
	Assessed      int64          `json:"assessed"`
	Changed       int64          `json:"changed"`
	Skipped       int64          `json:"skipped"`
	Failed        int64          `json:"failed"`
	Reason        string         `json:"reason,omitempty"`
	SkippedReason string         `json:"skipped_reason,omitempty"`
	Busy          bool           `json:"busy,omitempty"`
	Truncated     bool           `json:"truncated,omitempty"`
}

// ReadinessReport is the bounded outcome of sweeping every enabled
// repository: one sweep each, or a store-level error when the policy list
// itself is unreadable.
type ReadinessReport struct {
	Sweeps         []ReadinessSweep `json:"sweeps"`
	Error          string           `json:"error,omitempty"`
	TruncatedRepos bool             `json:"truncated_repos,omitempty"`
}

// ReconcileReadiness sweeps one factory-enabled repository with bounded
// work: it enumerates native issues a few pages at a time, assesses new
// and changed issues, and skips issues whose recorded inputs cannot have
// changed. Scans assess only: without an authenticated creation
// observation they never adopt initial acceptances. Per-issue failures
// are recorded and the sweep continues; the sweep revision advances only
// on a clean complete pass.
func (c *Coordinator) ReconcileReadiness(ctx context.Context, repository int64) (ReadinessSweep, error) {
	sweep := ReadinessSweep{Repository: repository}
	policy, err := c.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			sweep.SkippedReason = factory.MissingPolicy
			return sweep, nil
		}
		return sweep, err
	}
	switch {
	case !policy.Enabled:
		sweep.SkippedReason = factory.MissingPolicyDisabled
		return sweep, nil
	case policy.Paused:
		sweep.SkippedReason = factory.MissingPolicyPaused
		return sweep, nil
	}
	if c.Readiness == nil {
		return sweep, errors.New("readiness observation is not wired")
	}
	revision, idle, err := c.Readiness.ObserveNativeRevision(ctx)
	if err != nil {
		sweep.Reason = ReadinessSweepUnavailable
		return sweep, nil
	}
	if !idle {
		sweep.Busy = true
		return sweep, nil
	}
	authority, err := c.EffectiveAuthority(ctx, repository)
	if err != nil {
		return sweep, err
	}
	authorityHash := factory.AuthorityFingerprint(authority)
	visited := make(map[factory.DependenceRef]bool)
	for page := 1; page <= MaxReconcilePages; page++ {
		indexes, hasMore, err := c.Readiness.ListRepositoryIssues(ctx, repository, page)
		if err != nil {
			sweep.Reason = ReadinessSweepUnavailable
			return sweep, nil
		}
		for _, index := range indexes {
			sweep.Seen++
			skip, err := c.sweepPrecheck(ctx, repository, index, revision, authorityHash)
			if err != nil {
				c.sweepFailed(&sweep, index)
				continue
			}
			if skip {
				sweep.Skipped++
				continue
			}
			outcome, err := c.assessCascade(ctx, repository, index, visited)
			if err != nil {
				c.sweepFailed(&sweep, index)
				continue
			}
			sweep.Assessed++
			if outcome.changed {
				sweep.Changed++
			}
		}
		if !hasMore {
			break
		}
		if page == MaxReconcilePages {
			sweep.Truncated = true
		}
	}
	if !sweep.Truncated && sweep.Failed == 0 && sweep.Reason == "" {
		if err := c.Store.SaveReadinessSweep(ctx, repository, revision, time.Now()); err != nil {
			return sweep, err
		}
	}
	return sweep, nil
}

func (c *Coordinator) sweepFailed(sweep *ReadinessSweep, issue int64) {
	sweep.Failed++
	if len(sweep.Failures) < MaxSweepFailures {
		sweep.Failures = append(sweep.Failures, SweepFailure{Issue: issue, Reason: ReadinessSweepUnavailable})
	}
}

// sweepPrecheck reports whether an issue's recorded inputs cannot have
// changed: same acceptance head, same sweep revision, same authority
// verdict, same endpoint heads and no withdrawn code route. Everything
// the check reads is store-local; any difference forces a full
// assessment with fresh native evidence.
func (c *Coordinator) sweepPrecheck(ctx context.Context, repository, issue, revision int64, authorityHash string) (bool, error) {
	recorded, err := c.Store.IssueControl(ctx, repository, issue)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	head, herr := c.Store.AcceptanceHead(ctx, repository, issue)
	if herr != nil {
		if !errors.Is(herr, store.ErrNotFound) {
			return false, herr
		}
		head = ""
	}
	if recorded.Acceptance != head || recorded.NativeRev != revision || recorded.Authority != authorityHash {
		return false, nil
	}
	if head == "" {
		return len(recorded.EndpointHeads) == 0, nil
	}
	decision, err := c.Store.AcceptanceDecision(ctx, head)
	if err != nil {
		return false, err
	}
	if len(decision.Prerequisites) != len(recorded.EndpointHeads) {
		return false, nil
	}
	for _, prereq := range decision.Prerequisites {
		endpointHead, err := c.Store.AcceptanceHead(ctx, prereq.EndpointRepo, prereq.EndpointIssue)
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				return false, err
			}
			endpointHead = ""
		}
		if recorded.EndpointHeads[prereq.Occurrence] != endpointHead {
			return false, nil
		}
		if prereq.Outcome == factory.PrereqCode {
			withdrawn, _, err := c.Store.AcceptanceWithdrawn(ctx, prereq.EndpointRepo, prereq.EndpointIssue, prereq.PrereqAcceptance)
			if err != nil {
				return false, err
			}
			if withdrawn {
				return false, nil
			}
		}
	}
	return true, nil
}

// reconcileReadinessAll sweeps every enabled repository with bounded
// work for the operator reconcile command. Unwired observation skips
// silently: intake and browser flows already refuse as unavailable, and
// an unconfigured appliance must still settle its runs. Per-repository
// failures are recorded in their sweep, never raised.
func (c *Coordinator) reconcileReadinessAll(ctx context.Context) ReadinessReport {
	report := ReadinessReport{Sweeps: []ReadinessSweep{}}
	if c.Readiness == nil {
		return report
	}
	policies, err := c.Store.FactoryPolicies(ctx)
	if err != nil {
		report.Error = ReadinessSweepUnavailable
		return report
	}
	for _, policy := range policies {
		if !policy.Enabled || policy.Paused {
			continue
		}
		if len(report.Sweeps) >= MaxSweepRepos {
			report.TruncatedRepos = true
			break
		}
		sweep, err := c.ReconcileReadiness(ctx, policy.Repository)
		if err != nil {
			sweep = ReadinessSweep{Repository: policy.Repository, Reason: ReadinessSweepUnavailable}
		}
		report.Sweeps = append(report.Sweeps, sweep)
	}
	return report
}
