package control

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// attemptPlan is everything one dispatch attempt verified, in check order:
// the bound authority and grants, the exact preparation, harness pin and
// source, the verified prompt bytes, and the deadline the reservation
// will hold.
type attemptPlan struct {
	effective   factory.EffectiveAuthority
	policy      factory.RepositoryPolicy
	sponsorship factory.Sponsorship
	prep        project.StoredPreparation
	pin         project.FactoryHarnessPin
	acceptance  factory.Acceptance
	inputs      DispatchInputs
	projectID   string
	prompt      []byte
	promptSHA   string
	deadline    time.Time
	requirement string
	approval    string
	planned     int
	gateRev     int64
}

// dispatchOne dispatches one queued issue or reports exactly why it
// waited. Checks run in fixed order so the same state always reports the
// same reason; waits hold nothing and release nothing.
func dispatchOne(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, report *DispatchReport) {
	repository, issue := control.Repository, control.Issue
	assignments, err := deps.Store.IssueAssignments(ctx, repository, issue)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "assignments unreadable"})
		return
	}
	for _, a := range assignments {
		if a.Stage != factory.AssignmentFinished {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned})
			return
		}
	}
	head, err := deps.Store.AcceptanceHead(ctx, repository, issue)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "acceptance head unreadable"})
		return
	}
	for _, a := range assignments {
		if a.Acceptance == head {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAttemptRecorded})
			return
		}
	}
	plan, wait, failed := planAttempt(ctx, deps, occupancy, repository, issue, head)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
		return
	}
	executeFreshAttempt(ctx, deps, occupancy, plan, repository, issue, report)
}

// planWait is one non-launch outcome: either a retryable wait or an
// unexpected failure. Exactly one of the dispatchOne results is set.
type planWait struct {
	Reason string
	Detail string
}

func waitFor(reason, detail string) *planWait { return &planWait{Reason: reason, Detail: detail} }

// admissionWait maps an atomic packet refusal to its wait. A refusal
// proves a concurrent admission consumed the room first; the issue waits
// for a later pass instead of launching.
func admissionWait(err error) *planWait {
	switch {
	case errors.Is(err, store.ErrCapacityFull):
		return waitFor(WaitCapacity, "appliance runs at its limit")
	case errors.Is(err, store.ErrRepositoryFull):
		return waitFor(WaitRepository, "repository runs at its limit")
	case errors.Is(err, store.ErrSponsorshipFull):
		return waitFor(WaitSponsorship, "sponsorship runs at its limit")
	case errors.Is(err, store.ErrAllowanceExhausted):
		return waitFor(WaitAllowance, "sponsorship allowance is exhausted")
	case errors.Is(err, store.ErrAdmissionChanged):
		return waitFor(WaitAuthority, "grants changed during dispatch")
	default:
		return nil
	}
}

// refreshOccupancy reloads the pass snapshot after a lost admission race
// so later issues plan against the winner's reservation instead of stale
// emptiness. The packet gate stays authoritative: if the reload itself
// fails, the pass continues and the gate refuses any over-admission.
func refreshOccupancy(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, report *DispatchReport) {
	fresh, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return
	}
	*occupancy = fresh
}

// planAttempt verifies one dispatch end to end without recording
// anything. Every check that fails reports its wait or failure; a full
// plan hands its exact bound inputs to the executor.
func planAttempt(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, head string) (*attemptPlan, *planWait, *planWait) {
	plan := &attemptPlan{}
	effective, err := deps.Authority(ctx, repository)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "authority unreadable")
	}
	plan.effective = effective
	if !effective.Effective {
		if onlyDispatchClosed(effective) {
			return nil, waitFor(WaitDispatchClosed, "dispatch gate is closed"), nil
		}
		return nil, waitFor(WaitAuthority, "grants do not authorize dispatch"), nil
	}
	p, err := deps.Store.ProjectByRepository(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, waitFor(WaitPreparation, "repository has no project"), nil
		}
		return nil, nil, waitFor(DispatchErrStore, "project unreadable")
	}
	plan.projectID = p.ID
	policy, err := deps.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, waitFor(WaitAuthority, "policy changed during dispatch"), nil
		}
		return nil, nil, waitFor(DispatchErrStore, "policy unreadable")
	}
	plan.policy = policy
	sponsorship, wait, failed := selectSponsorship(ctx, deps, repository, effective)
	if failed != nil || wait != nil {
		return nil, wait, failed
	}
	plan.sponsorship = sponsorship
	decision, err := deps.Store.AcceptanceDecision(ctx, head)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "acceptance decision unreadable")
	}
	plan.acceptance = decision
	if wait, failed := checkLimits(ctx, deps, occupancy, plan); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait, failed := selectPreparation(ctx, deps, plan); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait := checkHarness(ctx, deps, plan); wait != nil {
		return nil, wait, nil
	}
	if wait, failed := readAttemptInputs(ctx, deps, plan, repository, issue); failed != nil || wait != nil {
		return nil, wait, failed
	}
	prompt, err := factory.BuildDispatchPrompt(factory.PromptInputs{
		Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
		AcceptanceID: head, TargetBranch: policy.TargetBranch, SourceCommit: plan.inputs.Tip,
		Preparation: plan.prep.Preparation.ID,
		Harness:     project.FactoryHarnessCodex, Model: policy.Roles[project.RoleCoder].Model, Role: project.RoleCoder,
		Title: plan.inputs.Issue.Title, Body: plan.inputs.Issue.Body,
		Sources:     promptSections(plan.acceptance.Sources, plan.inputs.Comments),
		Resolutions: promptSections(plan.acceptance.Resolutions, plan.inputs.Comments),
	})
	if err != nil {
		return nil, waitFor(WaitPrompt, "accepted inputs exceed the prompt bound"), nil
	}
	plan.prompt = prompt
	plan.promptSHA = project.FactoryPromptDigest(prompt)
	open, gateRev, _, err := deps.Store.DispatchState(ctx, repository)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "dispatch gate unreadable")
	}
	if !open {
		return nil, waitFor(WaitDispatchClosed, "dispatch gate closed during dispatch"), nil
	}
	plan.gateRev = gateRev
	return plan, nil, nil
}

// selectSponsorship picks the authority-bound active sponsorship: the
// first active grant in connection order, at its bound revision, with
// the coder role. Any other connection is never a fallback.
func selectSponsorship(ctx context.Context, deps DispatchDeps, repository int64, effective factory.EffectiveAuthority) (factory.Sponsorship, *planWait, *planWait) {
	sponsorships, err := deps.Store.Sponsorships(ctx, repository)
	if err != nil {
		return factory.Sponsorship{}, nil, waitFor(DispatchErrStore, "sponsorships unreadable")
	}
	for _, sponsorship := range sponsorships {
		if !sponsorship.Active {
			continue
		}
		if sponsorship.Revision != effective.Authority.Sponsorship {
			return factory.Sponsorship{}, waitFor(WaitAuthority, "sponsorship changed during dispatch"), nil
		}
		codes := false
		for _, role := range sponsorship.Roles {
			codes = codes || role == project.RoleCoder
		}
		if !codes {
			return factory.Sponsorship{}, waitFor(WaitSponsorshipRole, "sponsorship permits no coding role"), nil
		}
		return sponsorship, nil, nil
	}
	return factory.Sponsorship{}, waitFor(WaitAuthority, "sponsorship changed during dispatch"), nil
}

// checkLimits enforces appliance, repository, sponsorship and allowance
// limits against held reservations, unattributed active runs and
// confirmed usage, then sizes the plan's deadline from the remaining
// allowance. The deadline never exceeds the host's supervised bound.
func checkLimits(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan) (*planWait, *planWait) {
	capacity, err := deps.Store.Capacity(ctx)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitAuthority, "capacity changed during dispatch"), nil
		}
		return nil, waitFor(DispatchErrStore, "capacity unreadable")
	}
	grant, err := deps.Store.OperatorGrant(ctx, plan.policy.Repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitAuthority, "operator grant changed during dispatch"), nil
		}
		return nil, waitFor(DispatchErrStore, "operator grant unreadable")
	}
	repository := plan.policy.Repository
	if occupancy.heldTotal()+occupancy.unattributed >= capacity.MaxConcurrentRuns {
		return waitFor(WaitCapacity, "appliance runs at its limit"), nil
	}
	repoLimit := plan.policy.MaxConcurrent
	if grant.MaxConcurrent < repoLimit {
		repoLimit = grant.MaxConcurrent
	}
	if occupancy.heldRepo(repository)+occupancy.byProject[plan.projectID] >= repoLimit {
		return waitFor(WaitRepository, "repository runs at its limit"), nil
	}
	slots, planned := occupancy.heldConn(repository, plan.sponsorship.Connection)
	if slots >= plan.sponsorship.MaxConcurrent {
		return waitFor(WaitSponsorship, "sponsorship runs at its limit"), nil
	}
	used, err := deps.Store.UsageTotal(ctx, repository, plan.sponsorship.Connection)
	if err != nil {
		return nil, waitFor(DispatchErrStore, "confirmed usage unreadable")
	}
	remaining := plan.sponsorship.AllowanceMinutes - used - planned
	if remaining < 1 {
		return waitFor(WaitAllowance, "sponsorship allowance is exhausted"), nil
	}
	if remaining > MaxDispatchDeadline {
		remaining = MaxDispatchDeadline
	}
	plan.planned = remaining
	plan.deadline = time.Now().Add(time.Duration(remaining) * time.Minute)
	return nil, nil
}

// selectPreparation binds the coder preparation the current heads
// authorize: ready, for the coding role, and built from the current
// requirement and approval decisions. Any other preparation waits.
func selectPreparation(ctx context.Context, deps DispatchDeps, plan *attemptPlan) (*planWait, *planWait) {
	requirement, err := deps.Store.RequirementHead(ctx, plan.projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitPreparation, "project has no accepted requirements"), nil
		}
		return nil, waitFor(DispatchErrStore, "requirement head unreadable")
	}
	approval, err := deps.Store.ApprovalHead(ctx, plan.projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitPreparation, "project has no approved setup"), nil
		}
		return nil, waitFor(DispatchErrStore, "approval head unreadable")
	}
	preparations, err := deps.Store.ProjectPreparations(ctx, plan.projectID)
	if err != nil {
		return nil, waitFor(DispatchErrStore, "preparations unreadable")
	}
	for _, prep := range preparations {
		if prep.Preparation.Role != project.RoleCoder || !prep.State.Ready {
			continue
		}
		if prep.Preparation.Requirements.ID != requirement || prep.Preparation.Approval.ID != approval {
			continue
		}
		plan.prep = prep
		plan.requirement, plan.approval = requirement, approval
		return nil, nil
	}
	return waitFor(WaitPreparation, "no ready coding preparation matches the current decisions"), nil
}

// checkHarness requires the host's staged pin to match the policy's
// coding selection exactly: same proved family and version, a valid
// content pin and a pinned execution image.
func checkHarness(ctx context.Context, deps DispatchDeps, plan *attemptPlan) *planWait {
	pin, err := deps.Host.FactoryHarness(ctx)
	if err != nil {
		return waitFor(WaitHarness, "staged harness pin unreadable")
	}
	if err := pin.Validate(); err != nil {
		return waitFor(WaitHarness, "staged harness is not pinned")
	}
	if want := plan.policy.Roles[project.RoleCoder].Harness; pin.Version != want {
		return waitFor(WaitHarness, "policy selects harness "+want)
	}
	plan.pin = pin
	return nil
}

// readAttemptInputs reads the accepted objective, every selected comment
// revision and the target tip through the protected path, then verifies
// every digest against the acceptance head. Anything changed, hidden,
// closed or locked waits; the stale acceptance never authorizes a run.
func readAttemptInputs(ctx context.Context, deps DispatchDeps, plan *attemptPlan, repository, issue int64) (*planWait, *planWait) {
	if deps.Reads == nil {
		return waitFor(WaitInputsUnavailable, "accepted-input reads are not wired"), nil
	}
	seen := make(map[string]bool)
	var commentIDs []string
	for _, section := range [][]factory.SelectedSource{plan.acceptance.Sources, plan.acceptance.Resolutions} {
		for _, source := range section {
			if !seen[source.ID] {
				seen[source.ID] = true
				commentIDs = append(commentIDs, source.ID)
			}
		}
	}
	inputs, err := deps.Reads.ReadDispatchInputs(ctx,
		strconv.FormatInt(repository, 10), strconv.FormatInt(issue, 10),
		commentIDs, plan.policy.TargetBranch)
	if err != nil {
		var refusal *AcceptanceRefusal
		if errors.As(err, &refusal) {
			switch refusal.Reason {
			case RefusalNativeBusy:
				return waitFor(WaitInputsBusy, "native writers are busy"), nil
			case RefusalStaleEvidence:
				return waitFor(WaitInputsStale, "native state changed during the read"), nil
			case RefusalIncompleteEvidence:
				return waitFor(WaitInputsIncomplete, "native evidence is incomplete"), nil
			case RefusalSnapshotUnavailable:
				return waitFor(WaitInputsUnavailable, "native evidence is unavailable"), nil
			case RefusalIssueHidden:
				return waitFor(WaitInputsHidden, "the bound actor cannot see the issue"), nil
			}
		}
		return nil, waitFor(DispatchErrHost, "accepted-input read failed")
	}
	if !inputs.Issue.Visible {
		return waitFor(WaitInputsHidden, "the bound actor cannot see the issue"), nil
	}
	if inputs.Issue.IsPull {
		return waitFor(WaitInputsChanged, "target is not an issue"), nil
	}
	if inputs.Issue.Closed {
		return waitFor(WaitIssueClosed, "issue is closed"), nil
	}
	if inputs.Issue.Locked {
		return waitFor(WaitIssueLocked, "issue is locked"), nil
	}
	decision := plan.acceptance
	if inputs.Issue.TitleDigest != decision.TitleDigest || inputs.Issue.ContentDigest != decision.ContentDigest ||
		inputs.Issue.ContentVer != decision.ContentVersion || inputs.Revision < 1 {
		return waitFor(WaitInputsChanged, "accepted objective changed"), nil
	}
	byID := make(map[string]DispatchComment, len(inputs.Comments))
	for _, comment := range inputs.Comments {
		byID[comment.ID] = comment
	}
	for _, section := range [][]factory.SelectedSource{decision.Sources, decision.Resolutions} {
		for _, source := range section {
			got, ok := byID[source.ID]
			if !ok || !got.Visible || got.Digest != source.Digest || got.ContentVer != source.ContentVersion {
				return waitFor(WaitInputsChanged, "selected comment "+source.ID+" changed"), nil
			}
		}
	}
	if inputs.TipRef != plan.policy.TargetBranch || !factory.ValidCommit(inputs.Tip) {
		return waitFor(WaitSource, "target branch tip is unavailable"), nil
	}
	plan.inputs = inputs
	return nil, nil
}

func promptSections(selected []factory.SelectedSource, comments []DispatchComment) []factory.PromptSource {
	byID := make(map[string]string, len(comments))
	for _, comment := range comments {
		byID[comment.ID] = comment.Content
	}
	var out []factory.PromptSource
	for _, source := range selected {
		content, ok := byID[source.ID]
		if !ok {
			continue
		}
		out = append(out, factory.PromptSource{ID: source.ID, Content: content})
	}
	return out
}

// executeFreshAttempt records one fresh assignment under the open gate and
// launches its first run. Registration, packet and the admission-gate
// limit recheck land in one transaction before any host call, so a
// concurrent pass either wins the issue or consumes the room first; the
// loser waits on a refreshed snapshot instead of admitting stale work.
func executeFreshAttempt(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan, repository, issue int64, report *DispatchReport) {
	now := time.Now()
	assignmentID, runID := factory.NewID(), factory.NewID()
	authority := plan.effective.Authority
	authority.RequirementsID, authority.ApprovalID = plan.requirement, plan.approval
	assignment := factory.Assignment{
		Authority: authority,
		ID:        assignmentID, ProjectID: plan.projectID, Role: project.RoleCoder,
		Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
		Acceptance: plan.acceptance.ID, Preparation: plan.prep.Preparation.ID,
		Harness: project.FactoryHarnessCodex + "-" + plan.pin.Version, HarnessVers: plan.pin.Version,
		Model: plan.policy.Roles[project.RoleCoder].Model, Connection: plan.sponsorship.Connection,
		SourceCommit: plan.inputs.Tip, Prompt: plan.prompt, PromptSHA: plan.promptSHA,
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(),
	}
	if err := assignment.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch assignment invalid"})
		return
	}
	registration := factory.DispatchRegistration{
		ID: assignmentID, Repository: repository, Revision: plan.gateRev, Authority: authority,
	}
	reservation := factory.Reservation{
		AssignmentID: assignmentID, Repository: repository,
		Connection: plan.sponsorship.Connection, State: factory.ReservationHeld, PlannedMinutes: plan.planned,
	}
	run := factory.Run{
		ID: runID, ProjectID: plan.projectID, Role: project.RoleCoder, InputSHA: plan.inputs.Tip,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: assignment.Harness, Model: assignment.Model,
	}
	view := factory.RunView{RunID: runID, Repository: repository, Issue: issue, Attempt: assignmentID}
	if err := deps.Store.RecordDispatchPacket(ctx, registration, assignment, reservation, run, view); err != nil {
		if errors.Is(err, store.ErrDispatchClosed) {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitDispatchClosed, Detail: "dispatch gate closed during dispatch"})
			return
		}
		if errors.Is(err, store.ErrAssignmentActive) {
			refreshOccupancy(ctx, deps, occupancy, report)
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned, Detail: "a concurrent dispatch won this issue"})
			return
		}
		if wait := admissionWait(err); wait != nil {
			refreshOccupancy(ctx, deps, occupancy, report)
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
			return
		}
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch packet unrecorded"})
		return
	}
	occupancy.hold(reservation)
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: plan.projectID, Role: project.RoleCoder,
			Preparation: plan.prep.Preparation.ID,
			Harness:     project.FactoryHarnessCodex, HarnessVers: plan.pin.Version,
			Model:      assignment.Model,
			Assignment: plan.promptSHA, SourceCommit: plan.inputs.Tip,
			Connection: plan.sponsorship.Connection, Actor: plan.sponsorship.GrantedBy,
			Deadline: plan.deadline,
		},
		Prompt: plan.prompt, HarnessSHA256: plan.pin.SHA256,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, assignment, run, launch, report)
}

// retryAttempt relaunches one released assignment after re-checking
// limits, preparation, harness and inputs. The stored prompt must equal
// the reverified rebuild; anything else finishes the attempt instead of
// running stale work. Limit and preparation waits defer quietly: the
// fresh visit already reports the issue as assigned.
func retryAttempt(ctx context.Context, deps DispatchDeps, a factory.Assignment, report *DispatchReport) {
	occupancy, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return
	}
	plan, wait, failed := planAttempt(ctx, deps, &occupancy, a.Repository, a.Issue, a.Acceptance)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		switch wait.Reason {
		case WaitAuthority, WaitDispatchClosed:
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonWithdrawn, "grants no longer authorize dispatch", report)
		case WaitInputsChanged, WaitInputsHidden, WaitIssueClosed, WaitIssueLocked:
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "accepted inputs no longer hold", report)
		}
		return
	}
	if !bytes.Equal(plan.prompt, a.Prompt) {
		finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "accepted inputs no longer hold", report)
		return
	}
	runID := factory.NewID()
	now := time.Now()
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: a.Role, InputSHA: a.SourceCommit,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: a.Harness, Model: a.Model,
	}
	view := factory.RunView{RunID: runID, Repository: a.Repository, Issue: a.Issue, Attempt: a.ID}
	next, err := deps.Store.RecordRetryPacket(ctx, a, run, view, plan.planned)
	if err != nil {
		// A limit refusal defers quietly: the fresh visit already
		// reports the issue as assigned, and a later pass retries.
		if admissionWait(err) != nil {
			return
		}
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "dispatch attempt unrecorded"})
		return
	}
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: a.ProjectID, Role: a.Role,
			Preparation: a.Preparation,
			Harness:     project.FactoryHarnessCodex, HarnessVers: a.HarnessVers,
			Model:      a.Model,
			Assignment: a.PromptSHA, SourceCommit: a.SourceCommit,
			Connection: a.Connection, Actor: plan.sponsorship.GrantedBy,
			Deadline: plan.deadline,
		},
		Prompt: a.Prompt, HarnessSHA256: plan.pin.SHA256,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, next, run, launch, report)
}

// launchTail drives one recorded attempt run: a detached launch bounded
// by the run deadline, then exactly one of launched, confirmed-unused
// settle-and-release, or fenced-and-held. A returned error never guesses
// the run's effects; the host and broker decide what is unused.
func launchTail(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run, launch project.FactoryLaunch, report *DispatchReport) {
	launchCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), run.Deadline)
	defer cancel()
	state, err := deps.Host.FactoryLaunch(launchCtx, launch)
	if err == nil {
		report.Launched = append(report.Launched, DispatchLaunch{
			Repository: a.Repository, Issue: a.Issue,
			AssignmentID: a.ID, RunID: run.ID, Phase: state.Phase,
		})
		return
	}
	unused, fenced := confirmUnused(ctx, deps, a, run)
	switch {
	case fenced:
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrHost, Detail: "launch outcome unconfirmed"})
		ensureHeldLaunch(ctx, deps, a, report)
	case !unused:
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrFence, Detail: "launch left effects for reconcile"})
		ensureHeldLaunch(ctx, deps, a, report)
	default:
		settleUnusedLaunch(ctx, deps, a, run, err, report)
	}
}

// ensureHeldLaunch re-holds the reservation behind a fenced launch so
// the uncertain attempt keeps counting against its limits.
func ensureHeldLaunch(ctx context.Context, deps DispatchDeps, a factory.Assignment, report *DispatchReport) {
	reservation, err := deps.Store.Reservation(ctx, a.ID)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment reservation unreadable"})
		return
	}
	ensureHeld(ctx, deps, a, reservation, report)
}

// settleUnusedLaunch settles a launch the host and broker never saw and
// releases its confirmed-unused capacity. Attempts remain retryable
// until exhausted; an exhausted assignment finishes failed.
func settleUnusedLaunch(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run, launchErr error, report *DispatchReport) {
	settled := run
	settled.Outcome, settled.Summary, settled.Reconciled = factory.Failed, boundSummary("dispatch launch refused: "+launchErr.Error(), "run failed"), true
	if err := deps.Store.SaveFactoryRun(ctx, settled); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "unlaunched run unsettled"})
		return
	}
	if err := deps.Store.ReleaseReservation(ctx, a.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation release failed"})
		return
	}
	if a.Attempts >= factory.MaxDispatchAttempts {
		finishTerminal(ctx, deps, a, factory.Failed, factory.AssignReasonExhausted, "dispatch launch refused", report)
		report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitLaunchExhausted, Detail: "dispatch launch refused"})
		return
	}
	report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitLaunchRefused, Detail: "dispatch launch refused; retries remain"})
}
