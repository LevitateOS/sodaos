package api

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func isNativeTaskTool(tool factory.MCPTool) bool {
	return tool == factory.MCPToolStartTask || tool == factory.MCPToolReleaseOutput
}

func (s *factoryCatalogServer) serveNativeTask(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	if args.ExpectedInputDigest != "" && args.ExpectedInputDigest != cap.ScopeDigest() {
		return nil, nil, errors.New("native task request has stale context")
	}
	taskID, ok := args.Payload["task_id"].(string)
	if !ok || !factory.ValidPlanKey(taskID) || len(args.Payload) != 1 || args.SubmissionID != "" {
		return nil, nil, errors.New("native task request requires only payload.task_id")
	}
	if _, frozen, err := s.deps.Submissions.TerminalMCPProposal(ctx, cap.AssignmentID); err != nil {
		return nil, nil, err
	} else if frozen {
		return nil, nil, errors.New("assignment already recorded a terminal proposal")
	}
	if s.deps.NativeTask == nil {
		return nil, nil, errors.New("live native task controller unavailable")
	}
	if err := s.requireCurrentWork(ctx, cap); err != nil {
		return nil, nil, err
	}
	action := "start_task"
	if tool == factory.MCPToolReleaseOutput {
		action = "release_output"
	}
	// Admission and output recovery belong to the live native task owner. Do
	// not create a second submission ledger or infer a grant from model prose.
	result, err := s.deps.NativeTask(ctx, cap, action, taskID)
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(result)
}

func (s *API) factoryNativeTask(ctx context.Context, cap factory.MCPCapability, action, taskID string) (map[string]any, error) {
	if s.Store == nil || s.Host == nil || cap.State != factory.MCPCapabilityActive || cap.Expired(time.Now()) || !factory.ValidPlanKey(taskID) {
		return nil, errors.New("live native task controller unavailable")
	}
	phase := project.ExecutionImplement
	if action == "release_output" && cap.Profile == factory.MCPProfileFixer {
		phase = project.ExecutionCorrect
	} else if action != "start_task" || cap.Profile != factory.MCPProfileBuilder {
		return nil, errors.New("native task action differs from assigned role")
	}
	if err := s.Store.CheckMCPWorkActive(ctx, cap); err != nil {
		return nil, err
	}
	a, err := s.Store.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return nil, err
	}
	if a.ID != cap.AssignmentID || a.Run != cap.RunID || a.ProjectID != cap.Project || a.Repository != cap.Repository || a.SourceCommit != cap.SourceDigest || a.Subject.SubjectKey(a.Repository) != cap.Subject || strconv.FormatInt(a.NativeRev, 10) != cap.AcceptedRevision || a.Stage != factory.AssignmentAssigned || a.Role != project.RoleCoder || a.Phase != phase {
		return nil, errors.New("native task assignment differs from capability")
	}
	run, err := s.Store.FactoryRun(ctx, cap.RunID)
	if err != nil {
		return nil, err
	}
	binding := run.IdentityBinding
	if run.ID != cap.RunID || run.ProjectID != cap.Project || run.Role != a.Role || run.InputSHA != a.SourceCommit || run.Reconciled || run.Outcome != "" || !time.Now().Before(run.Deadline) || binding == nil || binding.Validate() != nil || binding.ID != run.ID || binding.Project != run.ProjectID || binding.Kind != identity.Factory || binding.Generation != run.IdentityGeneration || run.Admission == nil || run.Admission.Tool.Harness != project.FactoryHarnessMuse || run.Harness != project.FactoryHarnessMuse+"-"+run.Admission.Tool.Version {
		return nil, errors.New("native task run has no current Muse binding")
	}
	manifest, err := project.ParseExecutionManifest(run.ExecutionManifest)
	if err != nil || manifest.Run != run.ID || manifest.Phase != phase || !manifest.Deadline.Equal(run.Deadline) {
		return nil, errors.New("native task manifest differs from assigned run")
	}
	var plan factory.InvestigationPlan
	if json.Unmarshal(manifest.Plan, &plan) != nil || plan.Validate() != nil || plan.AttemptRoot != a.AttemptRoot {
		return nil, errors.New("native task plan differs from assigned attempt")
	}
	found := false
	for _, task := range plan.Tasks {
		found = found || task.ID == taskID
	}
	if !found {
		return nil, errors.New("native task is absent from retained plan")
	}
	observeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	state, err := s.Host.FactoryInspect(observeCtx, project.FactoryInspect{Project: cap.Project, ID: cap.RunID})
	cancel()
	if err != nil || !state.Live || state.Phase != project.FactoryRunning || state.Role != a.Role || state.Generation != run.IdentityGeneration || state.Invocation != binding.InvocationID || state.Unit == "" || state.Container == "" {
		return nil, errors.New("native task process differs from retained binding")
	}
	requestCtx, finish := context.WithTimeout(ctx, 30*time.Second)
	defer finish()
	result, err := s.Host.FactoryNativeTask(requestCtx, project.FactoryNativeTask{
		Project: cap.Project, Run: cap.RunID, InvocationID: binding.InvocationID,
		ManifestDigest: project.ExecutionManifestDigest(run.ExecutionManifest), Action: action,
		Arguments: project.FactoryNativeTaskArguments{TaskID: taskID},
	})
	return result.Result, err
}
