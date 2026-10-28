package api

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type checkpointContextSource interface {
	PlanTaskReceipts(context.Context, string, string) ([]store.PlanTaskReceipt, error)
}

// addCheckpointContext projects bounded native observations and retained
// receipts. Reading context never starts a checkpoint or spends a model call.
func (s *factoryCatalogServer) addCheckpointContext(ctx context.Context, read *readContext, body map[string]any) error {
	phase := factory.MCPPhaseOf(read.capability.Profile)
	if phase != project.ExecutionImplement && phase != project.ExecutionCorrect {
		return nil
	}
	source, ok := s.deps.Submissions.(checkpointContextSource)
	if !ok || s.deps.Runs == nil {
		return nil
	}
	run, err := s.deps.Runs.FactoryRun(ctx, read.capability.RunID)
	if err != nil {
		return err
	}
	if len(run.ExecutionManifest) == 0 {
		return nil
	}
	manifest, err := project.ParseExecutionManifest(run.ExecutionManifest)
	if err != nil {
		return err
	}
	var plan factory.InvestigationPlan
	if json.Unmarshal(manifest.Plan, &plan) != nil || plan.Validate() != nil || plan.AttemptRoot != read.assignment.AttemptRoot || manifest.Run != run.ID || manifest.Phase != phase {
		return errors.New("native task context does not bind its assigned manifest")
	}
	receipts, err := source.PlanTaskReceipts(ctx, plan.AttemptRoot, plan.ID)
	if err != nil {
		return err
	}
	byTask := map[string]store.PlanTaskReceipt{}
	head, last := plan.SourceCommit, int64(0)
	for _, receipt := range receipts {
		byTask[receipt.TaskID] = receipt
		if receipt.Commit == "" {
			continue
		}
		for _, ref := range receipt.Events {
			suffix, found := strings.CutPrefix(ref, project.TaskEventTaskCommitted+"@")
			if !found {
				continue
			}
			sequence, _, _ := strings.Cut(suffix, ":")
			number, err := strconv.ParseInt(sequence, 10, 64)
			if err == nil && number > last {
				last, head = number, receipt.Commit
			}
		}
	}
	tasks := make([]map[string]any, 0, len(plan.Tasks))
	ready := make([]map[string]any, 0, 4)
	for _, task := range plan.Tasks {
		receipt, exists := byTask[task.ID]
		state := store.PlanTaskPending
		if exists {
			state = receipt.State
		}
		tasks = append(tasks, map[string]any{"task_id": task.ID, "state": state})
		if state != store.PlanTaskOutputReady {
			continue
		}
		if len(ready) >= 4 {
			return errors.New("native task output exceeds admitted concurrency")
		}
		output := map[string]any{"task_id": task.ID, "output_digest": receipt.OutputDigest}
		if receipt.Child != "" {
			output["native_child_ref"] = receipt.Child
		}
		ready = append(ready, output)
	}
	execution := map[string]any{
		"manifest_digest":  project.ExecutionManifestDigest(run.ExecutionManifest),
		"parent_only":      phase == project.ExecutionCorrect,
		"integration_head": head, "tasks": tasks, "ready_outputs": ready,
		"guidance": "Submit each ready output with factory.submit_task_output and its task_id; SodaOS fills the observed digest and native child. Poll the submission receipt; successors need a committed native checkpoint. Do not write Git metadata. Finish implementation only after integration.status is committed; SodaOS fills the candidate from it.",
	}
	if run.FinalCheckpoint != nil {
		execution["integration"] = run.FinalCheckpoint
	}
	body["execution"] = execution
	return nil
}
