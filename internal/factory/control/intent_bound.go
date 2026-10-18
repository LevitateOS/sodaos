package control

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// ErrIntentBoundTransition reports an intent-bound run attempting to leave
// its admitted decompose phase. An admitted intent plan never becomes
// implemented work here: native graph writes happen only through FW09
// adoption of an accepted revision, and D13 completion joints stay with
// FW30. The caller revises the product direction or adopts the accepted
// graph instead of transitioning this run.
var ErrIntentBoundTransition = errors.New("intent-bound planning cannot transition to implementation")

// AdmitIntentBoundLaunch is the FW06B FW11 admission joint for intent-bound
// pre-issue planning launches. It binds the planning operation to the FW11
// subject/phase/role triple before delegating to AdmitIntentPlanning for
// head binding, draft scope, repository association, investigation and
// allowance. Only the ProposeIntent and DecompositionOptions operations
// launch; no accepted graph or history is consulted at any step, and none
// is taken: the signature carries no Store, Host or Graph dependency, so
// this boundary launches no repository session and grants no publication
// capability. Recording the plan-intent command, the assignment and root
// allowance, the reservation and the run stays with the FW11 dispatch
// packet boundary; the staged launch rides the existing host path through
// StageIntentBoundLaunch.
func AdmitIntentBoundLaunch(subject factory.ExecutionSubject, phase, role, operation string, head factory.IntentRevision, allowanceAvailable bool, in factory.SemanticPlannerInput) error {
	if !factory.ValidIntentPlanningOperation(operation) {
		return errors.New("unknown intent planning operation")
	}
	if in.Operation != operation {
		return errors.New("planner inputs do not bind the launched operation")
	}
	if phase != project.ExecutionDecompose {
		return ErrIntentBoundTransition
	}
	return AdmitIntentPlanning(subject, phase, role, head, allowanceAvailable, in)
}

// AdmitIntentBoundResult gates one intent-bound decompose answer at the
// FW11 result boundary. The subject must still be the admitted intent kind
// and the phase still decompose: an allowed planning run never transitions
// to implemented work through its result. Semantic assessment stays
// planner-outputs-only through factory.AssessSemanticOutput, so broad
// multi-task graphs and small single-task graphs both admit, uncovered
// authorized outcomes and false dependencies refuse, and an attention
// answer admits with no candidates and triggers no further batch. Results
// spend no allowance; exhaustion and Retry stay with the launch boundary.
func AdmitIntentBoundResult(subject factory.ExecutionSubject, phase string, in factory.SemanticPlannerInput, out factory.SemanticPlannerOutput) error {
	if subject.Kind != factory.SubjectIntent || phase != project.ExecutionDecompose {
		return ErrIntentBoundTransition
	}
	if out.Operation != in.Operation {
		return errors.New("semantic output does not bind its planning input")
	}
	return factory.AssessSemanticOutput(in, out)
}

// StageIntentPlanningPrompt renders the caller-staged prompt for one
// admitted intent-bound decompose run from its complete planning input:
// the operation, the head digest the subject binds, and the full typed
// input — composed content with corrections and declined meanings,
// alternatives, effective scope and inspected sources — closed by the
// decompose report contract. A digest alone cannot communicate reviewed
// corrections or rejected behavior absent from the objective, so the
// prompt preserves the exact input serialization; the prompt carries
// evidence references, never credentials, and stays within the factory
// prompt bound. Anything larger refuses instead of truncating
// requirements.
func StageIntentPlanningPrompt(head factory.IntentRevision, in factory.SemanticPlannerInput) ([]byte, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if in.Head.Digest() != head.Digest() {
		return nil, errors.New("planning prompt answers another head")
	}
	encoded, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("## Intent planning\n\n")
	b.WriteString("Operation: " + in.Operation + "\n")
	b.WriteString("Intent: " + head.Intent + "\n")
	b.WriteString("Revision: " + head.ID + "\n")
	b.WriteString("Head digest: " + head.Digest() + "\n")
	b.WriteString("\n## Admitted planning input\n\n")
	b.WriteString(promptFenceGuard.Replace(string(encoded)) + "\n")
	b.WriteString(phasedReportContract(project.ExecutionDecompose))
	prompt := []byte(promptFenceGuard.Replace(b.String()))
	if len(prompt) == 0 || len(prompt) > project.MaxFactoryPrompt {
		return nil, errors.New("staged intent prompt is empty or over its bound")
	}
	return prompt, nil
}

// stagedPlanningInputMarker opens the exact typed planning input the
// staged prompt preserves; the shared phased report marker closes it.
const stagedPlanningInputMarker = "## Admitted planning input\n\n"

// ParseStagedPlanningInput recovers the exact typed input a staged
// prompt recorded. Prompts without the marker predate staged inputs
// and report unstaged; a marker without a parseable input reports
// corrupt. The JSON carries no fenced report block, so the fence guard
// cannot alter its structure.
func ParseStagedPlanningInput(prompt []byte) (in factory.SemanticPlannerInput, staged bool, err error) {
	text := string(prompt)
	start := strings.Index(text, stagedPlanningInputMarker)
	if start < 0 {
		return in, false, nil
	}
	rest := text[start+len(stagedPlanningInputMarker):]
	end := strings.Index(rest, phasedReportMarker)
	if end < 0 {
		return in, true, errors.New("staged planning input lacks its report close")
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(rest[:end])), &in); err != nil {
		return in, true, err
	}
	return in, true, nil
}

// IntentBoundLaunchInput carries the dispatch-verified identities one
// admitted intent-bound decompose run launches under. Every identity was
// verified at admission; this joint only binds them to the existing host
// launch shape.
type IntentBoundLaunchInput struct {
	Prompt        []byte
	Deadline      time.Time
	RunID         string
	ProjectID     string
	PreparationID string
	Harness       string
	HarnessVers   string
	Model         string
	PromptSHA     string
	SourceCommit  string
	Connection    string
	HarnessSHA256 string
	ActorID       int64
}

// StageIntentBoundLaunch binds one admitted intent-bound decompose run to
// the existing host launch path. The launch pins the coder role and the
// decompose phase with no execution manifest: planning phases carry no
// native publication or mutation capability, and implement runs stage
// their manifest through their own owning boundary. Validation here
// refuses construction drift before any host call.
func StageIntentBoundLaunch(subject factory.ExecutionSubject, in IntentBoundLaunchInput) (project.FactoryLaunch, error) {
	if subject.Kind != factory.SubjectIntent || subject.Intent == nil {
		return project.FactoryLaunch{}, ErrIntentBoundTransition
	}
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: in.RunID, Project: in.ProjectID, Role: project.RoleCoder,
			Preparation: in.PreparationID,
			Harness:     in.Harness, HarnessVers: in.HarnessVers,
			Model:      in.Model,
			Assignment: in.PromptSHA, SourceCommit: in.SourceCommit,
			Connection: in.Connection, Actor: in.ActorID,
			Deadline: in.Deadline, ExecutionPhase: project.ExecutionDecompose,
		},
		Prompt: in.Prompt, HarnessSHA256: in.HarnessSHA256,
	}
	if err := launch.Validate(); err != nil {
		return project.FactoryLaunch{}, err
	}
	if launch.Run.ExecutionPhase != project.ExecutionDecompose || launch.Run.Role != project.RoleCoder || len(launch.ExecutionManifest) != 0 {
		return project.FactoryLaunch{}, ErrIntentBoundTransition
	}
	return launch, nil
}
