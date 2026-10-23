package control

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// VerifiedAppEvidence is independently read from the native helper's protected
// approval snapshot and check receipt. It is also useful to Test Pilot when
// constructing its finish report; the controller repeats the read on adoption.
type VerifiedAppEvidence = factory.CurrentVerificationEvidence

// ApplicationVerificationEvidence never executes model-provided commands. The
// single local catalog entry is the exact approved check.sh already executed by
// the native preparation supervisor. An ordinary preparation without explicit
// application assertions does not qualify.
func (c *Coordinator) ApplicationVerificationEvidence(ctx context.Context, a factory.Assignment) (VerifiedAppEvidence, error) {
	var evidence VerifiedAppEvidence
	if c == nil || c.Store == nil || c.Host == nil || c.DispatchReads == nil {
		return evidence, errors.New("native application verification is unavailable")
	}
	if a.Subject.Delivery == nil || a.Phase != project.ExecutionVerify || a.Role != project.RoleReviewer {
		return evidence, errors.New("assignment is not application verification")
	}
	revision, issue, err := c.Store.ApplicationVerificationInputs(ctx, a)
	if err != nil {
		return evidence, err
	}
	preparation, err := c.Store.Preparation(ctx, a.Preparation)
	if err != nil {
		return evidence, err
	}
	p := preparation.Preparation
	if p.Project != a.ProjectID || p.Role != project.RoleReviewer || p.SourceCommit != a.SourceCommit || factory.DeliveryPreparationDigest(p) != a.Subject.Delivery.PreparationDigest {
		return evidence, errors.New("application preparation differs from admitted delivery")
	}
	state, err := c.Host.InspectPreparation(ctx, project.PrepareInspect{Project: a.ProjectID, ID: a.Preparation})
	if err != nil {
		return evidence, err
	}
	if state.ID != p.ID || state.Project != p.Project || state.Role != p.Role || state.SourceCommit != p.SourceCommit || state.SetupDigest != p.SetupDigest || !state.Ready || state.Stopped || state.Phase != project.PrepareReady || state.CheckExit == nil || *state.CheckExit != 0 {
		return evidence, errors.New("approved application check has no native passing receipt")
	}
	request := project.FactoryPreparationContextRequest{Project: p.Project, ID: p.ID, SourceCommit: p.SourceCommit, ApprovedBase: p.Requirements.SourceCommit, DiffBase: p.SourceCommit, Candidate: p.SourceCommit, NotAfter: time.Now().Add(20 * time.Second)}
	material, err := c.Host.ReadPreparationContext(ctx, request, p.Role)
	if err != nil {
		return evidence, err
	}
	if err = material.ValidateFor(request, p.Role); err != nil {
		return evidence, err
	}
	manifest, err := factory.ParseApprovedAppCheck(material.Check)
	if err != nil {
		return evidence, err
	}
	if !manifest.BindsIntent(revision) || manifest.Digest() != a.Subject.Delivery.VerificationManifestDigest {
		return evidence, errors.New("approved application check does not cover this accepted delivery")
	}
	policy, err := c.Store.RepositoryPolicy(ctx, a.Repository)
	if err != nil {
		return evidence, err
	}
	native, err := c.DispatchReads.ReadDispatchInputs(ctx, strconv.FormatInt(a.Repository, 10), strconv.FormatInt(issue, 10), nil, policy.TargetBranch)
	if err != nil {
		return evidence, err
	}
	if native.TipRef != policy.TargetBranch || native.Tip != a.SourceCommit {
		return evidence, errors.New("integrated application source changed since verification admission")
	}
	evidence = VerifiedAppEvidence{Manifest: manifest, ManifestDigest: manifest.Digest(), PreparationDigest: factory.DeliveryPreparationDigest(p), SourceCommit: a.SourceCommit, State: state,
		Receipt: factory.VerificationCheckReceipt{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: factory.MCPInputDigest([]byte(state.Output)), LogExcerpt: "Native approved application check passed"}}
	return evidence, nil
}

func (c *Coordinator) verifyApplicationReport(ctx context.Context, a factory.Assignment) error {
	if a.Stage != factory.AssignmentFinished || a.Outcome != factory.Succeeded || a.Result == nil || !a.Result.Reported || a.Result.PhaseResult == nil || a.Result.PhaseResult.Output == nil || a.Result.PhaseResult.Output.Verification == nil {
		return errors.New("application verification has no completed report")
	}
	report := a.Result.PhaseResult.Output.Verification
	if err := a.Result.PhaseResult.BindAssignment(a); err != nil {
		return err
	}
	evidence, err := c.ApplicationVerificationEvidence(ctx, a)
	if err != nil {
		return err
	}
	if report.Outcome != factory.AppVerificationVerified || len(report.CheckReceipts) != 1 || report.CheckReceipts[0].CheckID != factory.ApprovedAppCheckID || report.CheckReceipts[0].Outcome != factory.CheckReceiptPassed || report.CheckReceipts[0].ArtifactDigest != evidence.Receipt.ArtifactDigest {
		return errors.New("worker verification report differs from native check receipt")
	}
	covered := map[string]bool{}
	for _, coverage := range report.Coverage {
		if coverage.IntentID != evidence.Manifest.IntentID || covered[coverage.OutcomeID] {
			return errors.New("worker report does not cover the exact accepted application scope")
		}
		covered[coverage.OutcomeID] = true
	}
	if len(covered) != len(evidence.Manifest.Outcomes) {
		return errors.New("worker report does not cover the exact accepted application scope")
	}
	for _, outcome := range evidence.Manifest.Outcomes {
		if !covered[outcome] {
			return errors.New("worker report omits an accepted application outcome")
		}
	}
	return c.Store.RecordVerifiedApplication(ctx, a, evidence.Manifest, evidence.State)
}

// driveAppVerifications runs after MCP/run result settlement, including restart
// recovery. Missing approved assertions or stale/native evidence emits F09;
// only the complete independent join can produce the verified F10 milestone.
func (c *Coordinator) driveAppVerifications(ctx context.Context) {
	if c == nil || c.Store == nil {
		return
	}
	assignments, err := c.Store.PendingAppVerifications(ctx, 16)
	if err != nil {
		return
	}
	for _, a := range assignments {
		if ctx.Err() != nil {
			return
		}
		if err := c.verifyApplicationReport(ctx, a); err != nil {
			_ = c.Store.RecordAppVerificationWait(ctx, a, err.Error())
		}
	}
}
