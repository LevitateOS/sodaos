package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type taskCheckpointHost interface {
	FactoryTaskCheckpoint(context.Context, project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error)
	FactoryFinalCandidate(context.Context, project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error)
}

type checkpointReceiptHost interface {
	FactoryFinalCandidateReceipt(context.Context, project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error)
}

var errCheckpointFailed = errors.New("trusted native checkpoint failed")

type taskOutputProposal struct {
	TaskID         string   `json:"task_id"`
	ManifestDigest string   `json:"manifest_digest"`
	OutputDigest   string   `json:"output_digest"`
	EvidenceRefs   []string `json:"evidence_refs"`
	NativeChildRef string   `json:"native_child_ref,omitempty"`
}

func decodeTaskOutput(submission factory.MCPSubmission) (taskOutputProposal, error) {
	var proposal taskOutputProposal
	decoder := json.NewDecoder(bytes.NewReader(submission.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proposal); err != nil {
		return proposal, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return proposal, errors.New("task proposal has trailing content")
	}
	if !factory.ValidPlanKey(proposal.TaskID) || !factory.ValidDigest(proposal.ManifestDigest) || !factory.ValidDigest(proposal.OutputDigest) {
		return proposal, errors.New("task proposal lacks its admitted output binding")
	}
	return proposal, nil
}

func checkpointRequest(run factory.Run, state project.FactoryState, head string) project.FactoryTaskCheckpoint {
	return project.FactoryTaskCheckpoint{
		Project: run.ProjectID, ID: run.ID, Container: state.Container, Invocation: state.Invocation,
		ManifestDigest: project.ExecutionManifestDigest(run.ExecutionManifest), ExpectedHead: head,
	}
}

// driveRunCheckpoints makes at most one native call per pass. The native
// journal is the sole task-state writer; the MCP receipt retains the exact
// request before effects so a lost reply never nominates a different parent.
func (c *Coordinator) driveRunCheckpoints(ctx context.Context, run factory.Run, state project.FactoryState) error {
	if !state.Live {
		return c.recoverFinalCheckpoint(ctx, run, state)
	}
	native, ok := c.Host.(taskCheckpointHost)
	if !ok || len(run.ExecutionManifest) == 0 {
		return nil
	}
	manifest, err := project.ParseExecutionManifest(run.ExecutionManifest)
	if err != nil {
		return err
	}
	var plan factory.InvestigationPlan
	if err := json.Unmarshal(manifest.Plan, &plan); err != nil {
		return err
	}
	receipts, err := c.Store.PlanTaskReceipts(ctx, plan.AttemptRoot, plan.ID)
	if err != nil {
		return err
	}
	owner, err := c.Store.AssignmentByRun(ctx, run.ID)
	if err != nil {
		return err
	}
	if owner.AttemptRoot != plan.AttemptRoot || owner.ProjectID != run.ProjectID || owner.Phase != manifest.Phase {
		return ErrTaskEventBinding
	}
	submissions, err := c.Store.MCPTaskOutputSubmissions(ctx, owner.ID)
	if err != nil {
		return err
	}
	byTask := map[string]store.PlanTaskReceipt{}
	head, lastCommit := plan.SourceCommit, int64(0)
	for _, receipt := range receipts {
		byTask[receipt.TaskID] = receipt
		for _, ref := range receipt.Events {
			sequence, _, valid := parseTaskEventRef(ref)
			if valid && sequence > lastCommit && receipt.Commit != "" && len(ref) > len(project.TaskEventTaskCommitted) && ref[:len(project.TaskEventTaskCommitted)] == project.TaskEventTaskCommitted {
				lastCommit, head = sequence, receipt.Commit
			}
		}
	}
	// Recover a previously reserved exact operation before admitting another.
	accepted := map[string]bool{}
	var activeCapability *factory.MCPCapability
	for _, submission := range submissions {
		proposal, err := decodeTaskOutput(submission)
		if err != nil {
			return err
		}
		receipt, err := c.Store.MCPReceipt(ctx, owner.ID, submission.SubmissionID)
		if err != nil {
			return err
		}
		capability, err := c.Store.GetMCPCapability(ctx, submission.CapabilityID)
		if err != nil || capability.AssignmentID != owner.ID || capability.RunID != run.ID ||
			capability.Project != run.ProjectID || factory.MCPPhaseOf(capability.Profile) != owner.Phase {
			return ErrTaskEventBinding
		}
		if capability.State == factory.MCPCapabilityActive && c.Store.CheckMCPWorkActive(ctx, capability) == nil {
			copied := capability
			activeCapability = &copied
		}
		if receipt.Checkpoint != nil && receipt.State == factory.MCPReceiptAccepted && receipt.Checkpoint.Status == "committed" {
			checkpoint := receipt.Checkpoint
			task, exists := byTask[proposal.TaskID]
			if !exists || checkpoint.Request.ID != run.ID || checkpoint.Request.ManifestDigest != project.ExecutionManifestDigest(run.ExecutionManifest) ||
				checkpoint.Request.TaskKey != proposal.TaskID || checkpoint.Request.OutputDigest != proposal.OutputDigest || task.OutputDigest != proposal.OutputDigest {
				return ErrTaskEventBinding
			}
			accepted[proposal.TaskID] = true
		}
		if receipt.State != factory.MCPReceiptVerifying || receipt.Checkpoint == nil {
			continue
		}
		task, exists := byTask[proposal.TaskID]
		if !exists || task.OutputDigest != proposal.OutputDigest || receipt.Checkpoint.Request.ID != run.ID {
			return ErrTaskEventBinding
		}
		// Only read back an already committed operation after withdrawal.
		// An unfinished operation waits for the ordinary stop owner.
		if !state.Live && task.State != store.PlanTaskCompleted || state.Live && (capability.State != factory.MCPCapabilityActive || c.Store.CheckMCPWorkActive(ctx, capability) != nil) {
			return nil
		}
		return c.advanceTaskCheckpoint(ctx, native, submission, receipt, receipt.Checkpoint.Request)
	}
	if !state.Live || activeCapability == nil {
		return nil
	}
	ready := make([]store.PlanTaskReceipt, 0, len(receipts))
	allCompleted := len(receipts) == len(plan.Tasks)
	for _, receipt := range receipts {
		if receipt.State == store.PlanTaskOutputReady {
			ready = append(ready, receipt)
		}
		allCompleted = allCompleted && receipt.State == store.PlanTaskCompleted && accepted[receipt.TaskID]
	}
	reportSequence := func(receipt store.PlanTaskReceipt) int64 {
		for _, ref := range receipt.Events {
			for _, eventType := range []string{project.TaskEventChildReported, project.TaskEventParentOutput} {
				if len(ref) > len(eventType) && ref[:len(eventType)] == eventType {
					sequence, _, valid := parseTaskEventRef(ref)
					if valid {
						return sequence
					}
				}
			}
		}
		return 0
	}
	sort.Slice(ready, func(i, j int) bool { return reportSequence(ready[i]) < reportSequence(ready[j]) })
	if len(ready) > 0 {
		task := ready[0]
		for _, submission := range submissions {
			proposal, err := decodeTaskOutput(submission)
			if err != nil {
				return err
			}
			if proposal.TaskID != task.TaskID || proposal.ManifestDigest != project.ExecutionManifestDigest(run.ExecutionManifest) ||
				proposal.OutputDigest != task.OutputDigest || proposal.NativeChildRef != task.Child {
				continue
			}
			receipt, err := c.Store.MCPReceipt(ctx, owner.ID, submission.SubmissionID)
			if err != nil {
				return err
			}
			if receipt.State != factory.MCPReceiptRecorded {
				continue
			}
			capability, err := c.Store.GetMCPCapability(ctx, submission.CapabilityID)
			if err != nil || capability.State != factory.MCPCapabilityActive || c.Store.CheckMCPWorkActive(ctx, capability) != nil {
				return nil
			}
			request := checkpointRequest(run, state, head)
			request.TaskKey, request.OutputDigest = task.TaskID, task.OutputDigest
			receipt.State = factory.MCPReceiptVerifying
			receipt.Checkpoint = &project.FactoryTaskCheckpointState{Request: request, Status: "pending"}
			if err := c.Store.AdvanceMCPReceipt(ctx, owner.ID, submission.SubmissionID, factory.MCPReceiptRecorded, receipt); err != nil {
				return err
			}
			return c.advanceTaskCheckpoint(ctx, native, submission, receipt, request)
		}
	}
	if allCompleted {
		if run.FinalCheckpoint != nil && (run.FinalCheckpoint.Status == "committed" || run.FinalCheckpoint.Status == "failed") {
			if run.FinalCheckpoint.Status == "failed" {
				return errCheckpointFailed
			}
			return nil
		}
		request := checkpointRequest(run, state, head)
		result, err := native.FactoryFinalCandidate(ctx, request)
		if err != nil {
			return err
		}
		if err := result.ValidateFor(request, true); err != nil {
			return err
		}
		if err := c.Store.SaveFactoryFinalCheckpoint(ctx, run, result); err != nil {
			return err
		}
		if result.Status == "failed" {
			return errCheckpointFailed
		}
	}
	return nil
}

// A native parent may finish its final checks and retire before the next Go
// poll. Adopt only an already retained final receipt through the read-only
// native route; this cannot revive the parent or launch another check.
func (c *Coordinator) recoverFinalCheckpoint(ctx context.Context, run factory.Run, state project.FactoryState) error {
	reader, ok := c.Host.(checkpointReceiptHost)
	if !ok || len(run.ExecutionManifest) == 0 || run.FinalCheckpoint != nil && (run.FinalCheckpoint.Status == "committed" || run.FinalCheckpoint.Status == "failed") {
		return nil
	}
	manifest, err := project.ParseExecutionManifest(run.ExecutionManifest)
	if err != nil {
		return err
	}
	var plan factory.InvestigationPlan
	if err := json.Unmarshal(manifest.Plan, &plan); err != nil {
		return err
	}
	receipts, err := c.Store.PlanTaskReceipts(ctx, plan.AttemptRoot, plan.ID)
	if err != nil {
		return err
	}
	if len(receipts) != len(plan.Tasks) {
		return nil
	}
	owner, err := c.Store.AssignmentByRun(ctx, run.ID)
	if err != nil {
		return err
	}
	submissions, err := c.Store.MCPTaskOutputSubmissions(ctx, owner.ID)
	if err != nil {
		return err
	}
	head, last := plan.SourceCommit, int64(0)
	for _, task := range receipts {
		if task.State != store.PlanTaskCompleted {
			return nil
		}
		matched := false
		for _, submission := range submissions {
			proposal, err := decodeTaskOutput(submission)
			if err != nil {
				return err
			}
			if proposal.TaskID != task.TaskID || proposal.ManifestDigest != project.ExecutionManifestDigest(run.ExecutionManifest) || proposal.OutputDigest != task.OutputDigest || proposal.NativeChildRef != task.Child {
				continue
			}
			capability, err := c.Store.GetMCPCapability(ctx, submission.CapabilityID)
			if err != nil {
				return err
			}
			if capability.AssignmentID != owner.ID || capability.RunID != run.ID || capability.Project != run.ProjectID || factory.MCPPhaseOf(capability.Profile) != manifest.Phase {
				return ErrTaskEventBinding
			}
			receipt, err := c.Store.MCPReceipt(ctx, owner.ID, submission.SubmissionID)
			if err != nil {
				return err
			}
			if receipt.Checkpoint != nil && (receipt.State == factory.MCPReceiptAccepted || receipt.State == factory.MCPReceiptVerifying) && receipt.Checkpoint.Request.TaskKey == task.TaskID && receipt.Checkpoint.Request.OutputDigest == task.OutputDigest && receipt.Checkpoint.Request.ID == run.ID && receipt.Checkpoint.Request.ManifestDigest == proposal.ManifestDigest {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
		for _, ref := range task.Events {
			if len(ref) > len(project.TaskEventTaskCommitted) && ref[:len(project.TaskEventTaskCommitted)] == project.TaskEventTaskCommitted && task.Commit != "" {
				sequence, _, valid := parseTaskEventRef(ref)
				if valid && sequence > last {
					last, head = sequence, task.Commit
				}
			}
		}
	}
	request := checkpointRequest(run, state, head)
	result, err := reader.FactoryFinalCandidateReceipt(ctx, request)
	if err != nil {
		return err
	}
	if err := result.ValidateFor(request, true); err != nil {
		return err
	}
	if result.Status != "committed" && result.Status != "failed" {
		return nil
	}
	return c.Store.SaveFactoryFinalCheckpoint(ctx, run, result)
}

func (c *Coordinator) advanceTaskCheckpoint(ctx context.Context, native taskCheckpointHost, submission factory.MCPSubmission, receipt factory.MCPReceipt, request project.FactoryTaskCheckpoint) error {
	result, err := native.FactoryTaskCheckpoint(ctx, request)
	if err != nil {
		return err
	}
	if err := result.ValidateFor(request, false); err != nil {
		return err
	}
	next := receipt
	next.Checkpoint = &result
	switch result.Status {
	case "committed":
		next.State, next.Reason = factory.MCPReceiptAccepted, "native_checkpoint:"+result.ReceiptDigest
	case "failed":
		next.State, next.Reason = factory.MCPReceiptRefused, result.Reason
	case "busy":
		next.State, next.Reason, next.Checkpoint = factory.MCPReceiptRecorded, "checkpoint_busy", nil
	}
	if !reflect.DeepEqual(receipt, next) {
		if err := c.Store.AdvanceMCPReceipt(ctx, submission.AssignmentID, submission.SubmissionID, receipt.State, next); err != nil {
			return err
		}
	}
	if result.Status == "failed" {
		return errCheckpointFailed
	}
	return nil
}
