package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Startup retains a recorded pre-launch hold when both native custody owners
// confirm that no execution exists. Stop/Pause still retire it explicitly.
func (c *Coordinator) preservePendingContext(ctx context.Context, run factory.Run) bool {
	if !run.ContextPending || run.Reconciled || run.Outcome != "" || run.CredentialDelegated || run.IdentityBinding != nil || run.Admission == nil || !time.Now().Before(run.Deadline) || len(c.MCPKey) == 0 {
		return false
	}
	a, err := c.Store.AssignmentByRun(ctx, run.ID)
	if err != nil || a.Stage != factory.AssignmentAssigned || a.Run != run.ID || a.ProjectID != run.ProjectID {
		return false
	}
	observed, err := c.Store.MCPRunWasObserved(ctx, run.ID, a.ID, a.ProjectID)
	if err != nil || observed {
		return false
	}
	if _, err = c.Host.FactoryInspect(ctx, project.FactoryInspect{Project: run.ProjectID, ID: run.ID}); !isHostNotFound(err) {
		return false
	}
	_, err = c.Broker.GetExecution(ctx, identity.Factory, run.ID)
	if !errors.Is(err, identity.ErrNotFound) {
		return false
	}
	_, _, terminal := pendingContextLaunch(ctx, c.dispatchDeps(), a, run)
	return !terminal
}

// A context hold is a pending first launch, not a failed native attempt.
// Recovery retains its run, capability and paid choice. The marker is cleared
// before any host call, so lost native replies retain ordinary fenced recovery.
func recoverPendingContext(ctx context.Context, deps DispatchDeps, a factory.Assignment, reservation factory.Reservation, run factory.Run, report *DispatchReport) bool {
	if run.Reconciled || run.Outcome != "" || !time.Now().Before(run.Deadline) || len(deps.MCPKey) == 0 {
		return false
	}
	launch, wait, terminal := pendingContextLaunch(ctx, deps, a, run)
	if terminal {
		return false
	}
	ensureHeld(ctx, deps, a, reservation, report)
	held, heldErr := deps.Store.Reservation(ctx, a.ID)
	if heldErr != nil || held.State != factory.ReservationHeld {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "pending launch reservation is not held"})
		return true
	}
	if wait != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: wait.Reason, Detail: wait.Detail})
		return true
	}
	launchTail(ctx, deps, a, run, launch, report)
	return true
}

func pendingContextLaunch(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run) (project.FactoryLaunch, *planWait, bool) {
	var launch project.FactoryLaunch
	if run.Admission == nil || run.ID != a.Run || run.InputSHA != a.SourceCommit {
		return launch, waitFor(DispatchErrStore, "pending launch lacks its recorded admission"), false
	}
	effective, err := deps.Authority(ctx, a.Repository)
	if err != nil {
		return launch, waitFor(DispatchErrStore, "pending launch authority unavailable"), false
	}
	if !effective.Effective || !effective.DispatchOpen {
		return launch, nil, true
	}
	requirement, requirementErr := deps.Store.RequirementHead(ctx, a.ProjectID)
	approval, approvalErr := deps.Store.ApprovalHead(ctx, a.ProjectID)
	if requirementErr != nil || approvalErr != nil {
		return launch, waitFor(WaitPreparation, "pending launch preparation decisions unavailable"), false
	}
	effective.Authority.RequirementsID, effective.Authority.ApprovalID = requirement, approval
	if !effective.Authority.SameGrants(a.Authority) || run.Admission.Authority != a.Authority {
		return launch, nil, true
	}
	capability, err := deps.Store.GetMCPCapability(ctx, run.ID)
	if errors.Is(err, store.ErrNotFound) {
		// A crash between recording the pending marker and staging its
		// capability resumes the same deterministic capability production.
		if err = ProduceMCPCapabilityForLaunch(ctx, deps.Store, a, run); err == nil {
			capability, err = deps.Store.GetMCPCapability(ctx, run.ID)
		}
	}
	if err != nil {
		return launch, waitFor("context_selection_hold", "pending capability unavailable"), false
	}
	if capability.State != factory.MCPCapabilityStaged {
		return launch, nil, true
	}
	current, err := deps.Store.FactoryCurrentContext(ctx, capability, time.Now())
	if err != nil {
		return launch, waitFor("context_selection_hold", "required current context unavailable"), false
	}
	if current.Allowance == nil || current.Allowance.Closed || current.Allowance.ActiveSecondsRemaining <= 0 {
		return launch, nil, true
	}
	if a.Issue > 0 {
		if wait, terminal := pendingIssueInputs(ctx, deps, a, run); wait != nil || terminal {
			return launch, wait, terminal
		}
	}
	preparation, stopped, err := recordedMCPPreparation(ctx, deps.Store, a.Preparation)
	if err != nil || stopped || !preparation.State.Ready {
		return launch, waitFor(WaitPreparation, "pending launch preparation is not ready"), false
	}
	// The pending launch binds the recorded admitted bundle exactly and never
	// consults the live host pin: rows carrying recorded tool proof resume
	// their adopted bundle byte-for-byte, so an update that moved the
	// current selection cannot strand or replace them. The host stages the
	// recorded bundle from its retained sidecar and fails closed when that
	// bundle is gone. Rows admitted before proof existed keep the legacy
	// live-pin comparison and hold when it differs; rows with missing or
	// incoherent recorded identity hold instead of falling back to current.
	selection, ok := run.Admission.Policy.Roles[a.Role]
	if !ok {
		return launch, waitFor(WaitHarness, "pending launch lacks its recorded role selection"), false
	}
	if selection.Model != a.Model || run.Model != a.Model {
		return launch, waitFor(WaitHarness, "pending launch model differs from its recorded admission"), false
	}
	adopted := run.Admission.Tool
	if adopted != (project.FactoryHarnessPin{}) {
		if err := adopted.Validate(); err != nil {
			return launch, waitFor(WaitHarness, "pending launch recorded tool proof is invalid"), false
		}
		if a.AdoptedHarness != adopted {
			return launch, waitFor(WaitHarness, "pending launch proof differs from its assignment"), false
		}
		if adopted.Harness != selection.Harness {
			return launch, waitFor(WaitHarness, "pending launch bundle is not its admitted family"), false
		}
		if a.Harness != adopted.Harness+"-"+adopted.Version || a.HarnessVers != adopted.Version ||
			run.Harness != a.Harness || run.Image != adopted.Image {
			return launch, waitFor(WaitHarness, "pending launch admission does not bind its recorded identity"), false
		}
		launch = project.FactoryLaunch{Run: project.FactoryRun{ID: run.ID, Project: a.ProjectID, Role: a.Role, Preparation: a.Preparation, Harness: adopted.Harness, HarnessVers: adopted.Version, Model: a.Model, Assignment: a.PromptSHA, SourceCommit: a.SourceCommit, Connection: a.Connection, Actor: a.ActorID, Deadline: run.Deadline, ExecutionPhase: a.Phase}, Prompt: a.Prompt, HarnessSHA256: adopted.SHA256, ExecutionManifest: run.ExecutionManifest}
	} else {
		if a.Harness != selection.Harness+"-"+a.HarnessVers || run.Harness != a.Harness {
			return launch, waitFor(WaitHarness, "pending launch admission does not bind its recorded identity"), false
		}
		pin, err := deps.Host.FactoryHarness(ctx, selection.Harness)
		if err != nil || pin.Validate() != nil {
			return launch, waitFor(WaitHarness, "pending launch harness unavailable"), false
		}
		if pin.Harness != selection.Harness || pin.Version != a.HarnessVers || pin.Image != run.Image {
			return launch, waitFor(WaitHarness, "recorded harness bundle is not staged; pending launch holds for its admitted identity"), false
		}
		launch = project.FactoryLaunch{Run: project.FactoryRun{ID: run.ID, Project: a.ProjectID, Role: a.Role, Preparation: a.Preparation, Harness: selection.Harness, HarnessVers: a.HarnessVers, Model: a.Model, Assignment: a.PromptSHA, SourceCommit: a.SourceCommit, Connection: a.Connection, Actor: a.ActorID, Deadline: run.Deadline, ExecutionPhase: a.Phase}, Prompt: a.Prompt, HarnessSHA256: pin.SHA256, ExecutionManifest: run.ExecutionManifest}
	}
	if len(run.ExecutionManifest) > 0 {
		launch.Run.ManifestDigest = project.ExecutionManifestDigest(run.ExecutionManifest)
	}
	launch.VerificationManifest = run.VerificationManifest
	if len(run.VerificationManifest) > 0 {
		launch.Run.VerificationManifestDigest = project.ExecutionManifestDigest(run.VerificationManifest)
	}
	if launch.Validate() != nil {
		return project.FactoryLaunch{}, waitFor(DispatchErrStore, "pending launch differs from its admitted packet"), false
	}
	return launch, nil, false
}

func pendingIssueInputs(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run) (*planWait, bool) {
	head, err := deps.Store.AcceptanceHead(ctx, a.Repository, a.Issue)
	if errors.Is(err, store.ErrNotFound) || (err == nil && head != a.Acceptance) {
		return nil, true
	}
	if err != nil {
		return waitFor(DispatchErrStore, "pending acceptance head unavailable"), false
	}
	acceptance, err := deps.Store.AcceptanceDecision(ctx, a.Acceptance)
	if err != nil {
		return waitFor(DispatchErrStore, "pending accepted inputs unavailable"), false
	}
	plan := &attemptPlan{acceptance: acceptance, policy: run.Admission.Policy}
	wait, failed := readAttemptInputs(ctx, deps, plan, a.Repository, a.Issue)
	if failed != nil {
		return failed, false
	}
	if wait != nil {
		switch wait.Reason {
		case WaitInputsChanged, WaitInputsHidden, WaitIssueClosed, WaitIssueLocked:
			return nil, true
		}
		return wait, false
	}
	if a.ParentAssignment == "" && a.Phase == project.ExecutionInvestigate && plan.inputs.Tip != a.SourceCommit {
		return nil, true
	}
	return nil, false
}
