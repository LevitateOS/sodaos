package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func (c *Coordinator) deliveryAdmission(ctx context.Context, candidate store.DeliveredIntentCandidate) (factory.ExecutionSubject, string, []byte, error) {
	var none factory.ExecutionSubject
	revision := candidate.Revision
	policy, err := c.Store.RepositoryPolicy(ctx, revision.Repository)
	if err != nil {
		return none, "", nil, err
	}
	proj, err := c.Store.ProjectByRepository(ctx, revision.Repository)
	if err != nil {
		return none, "", nil, err
	}
	native, err := c.DispatchReads.ReadDispatchInputs(ctx, strconv.FormatInt(revision.Repository, 10), strconv.FormatInt(candidate.FirstIssue, 10), nil, policy.TargetBranch)
	if err != nil {
		return none, "", nil, err
	}
	if native.TipRef != policy.TargetBranch || !factory.ValidCommit(native.Tip) {
		return none, "", nil, errors.New("integrated application source is unavailable")
	}
	preparations, err := c.Store.ProjectPreparations(ctx, proj.ID)
	if err != nil {
		return none, "", nil, err
	}
	requirements, err := c.Store.RequirementHead(ctx, proj.ID)
	if err != nil {
		return none, "", nil, err
	}
	approval, err := c.Store.ApprovalHead(ctx, proj.ID)
	if err != nil {
		return none, "", nil, err
	}
	for _, stored := range preparations {
		p := stored.Preparation
		if p.Role != project.RoleReviewer || p.SourceCommit != native.Tip || p.Requirements.ID != requirements || p.Approval.ID != approval || !stored.State.Ready {
			continue
		}
		request := project.FactoryPreparationContextRequest{Project: p.Project, ID: p.ID, SourceCommit: p.SourceCommit, ApprovedBase: p.Requirements.SourceCommit, DiffBase: p.SourceCommit, Candidate: p.SourceCommit, NotAfter: time.Now().Add(20 * time.Second)}
		material, err := c.Host.ReadPreparationContext(ctx, request, p.Role)
		if err != nil {
			continue
		}
		if material.ValidateFor(request, p.Role) != nil {
			continue
		}
		manifest, err := factory.ParseApprovedAppCheck(material.Check)
		if err != nil || !manifest.BindsIntent(revision) {
			continue
		}
		subject := factory.DeliverySubject{IntentID: revision.Intent, AcceptedRevision: revision.ID, ScopeDigest: revision.Digest(), SourceCommit: native.Tip, PreparationDigest: factory.DeliveryPreparationDigest(p), VerificationManifestDigest: manifest.Digest()}
		raw, _ := json.Marshal(subject)
		sum := sha256.Sum256(append([]byte("application-delivery\x00"), raw...))
		subject.DeliveryID = hex.EncodeToString(sum[:16])
		execution := factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &subject}
		// Repeat the native receipt join used by adoption before spending a worker
		// session. Missing/failed assertions remain an actionable Factory event.
		evidence, err := c.ApplicationVerificationEvidence(ctx, factory.Assignment{Repository: revision.Repository, ProjectID: proj.ID, Phase: project.ExecutionVerify, Role: project.RoleReviewer, Subject: execution, SourceCommit: native.Tip, Preparation: p.ID})
		if err != nil {
			return none, "", nil, err
		}
		payload, err := json.Marshal(struct {
			Objective string                  `json:"objective"`
			Revision  factory.IntentRevision  `json:"accepted_scope"`
			Subject   factory.DeliverySubject `json:"delivery"`
			Evidence  VerifiedAppEvidence     `json:"native_check_evidence"`
		}{revision.Objective, revision, subject, evidence})
		if err != nil {
			return none, "", nil, err
		}
		prompt := append([]byte("You are Test Pilot for this integrated application. Read factory.context and the recorded native application-check evidence below. Inspect whether the approved assertions cover every accepted outcome. Report findings with factory tools; use factory.finish_app_verification with the bound delivery ID, input digest from factory.context, outcome coverage and exact check receipt only when the evidence supports it. Missing or contradictory evidence must be reported as incomplete or failed. Do not change code or claim deployment.\n\n"), payload...)
		if len(prompt) > project.MaxFactoryPrompt {
			return none, "", nil, errors.New("application verification scope exceeds the worker prompt bound")
		}
		return execution, p.ID, prompt, nil
	}
	return none, "", nil, errors.New("A ready reviewer preparation at the integrated source needs an approved check.sh with soda-app-verification assertions for every accepted outcome.")
}

// Accepted and integrated scopes become one immutable delivery operation. The
// shared dispatch path records funding and its scoped MCP before native launch.
func (c *Coordinator) driveDeliveryVerifications(ctx context.Context) {
	if c == nil || c.Store == nil || c.Host == nil || c.DispatchReads == nil {
		return
	}
	candidates, err := c.Store.DeliveredIntentCandidates(ctx, 16)
	if err != nil {
		return
	}
	occupancy, err := snapshotOccupancy(ctx, c.Store)
	if err != nil {
		return
	}
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return
		}
		subject, preparation, prompt, err := c.deliveryAdmission(ctx, candidate)
		if err != nil {
			_ = c.Store.RecordDeliveryAdmissionWait(ctx, candidate.Revision, err.Error())
			continue
		}
		seen, err := c.Store.DeliveryAlreadyAssigned(ctx, candidate.Revision.Repository, subject.Delivery.DeliveryID)
		if err != nil || seen {
			continue
		}
		report := newDispatchReport()
		DispatchTimeOnlyRoot(ctx, c.dispatchDeps(), &occupancy, candidate.Revision.Repository, subject, subject.Delivery.SourceCommit, prompt, preparation, &report)
		for _, wait := range report.Waits {
			_ = c.Store.RecordDeliveryAdmissionWait(ctx, candidate.Revision, wait.Detail)
		}
		for _, failure := range report.Errors {
			_ = c.Store.RecordDeliveryAdmissionWait(ctx, candidate.Revision, failure.Detail)
		}
	}
}
