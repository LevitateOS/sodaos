package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"

	"github.com/levitateos/sodaos/internal/web/auth"
)

func intentID() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "d" + hex.EncodeToString(raw[:]), nil
}
func (s *API) apiIntentExplore(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in struct {
		Revision string `json:"revision"`
	}
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	current, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if current.ID != in.Revision || current.State != "draft" {
		auth.JSONError(w, 412, "stale_revision", "Review inputs changed.")
		return
	}
	ids := make([]string, 104)
	for i := range ids {
		ids[i], err = intentID()
		if err != nil {
			factoryCommandError(w, err)
			return
		}
	}
	proposal := factory.ProposeIntent(current, ids)
	proposal.Predecessor = current.ID
	proposal.ID, err = intentID()
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	proposal.CreatedUnix = time.Now().Unix()
	// Product exploration uses explicitly bounded consultation credit, distinct
	// from execution funding. Failure remains visible and creates no acceptance.
	candidates := intentDesignCandidates(current)
	design := current
	design.ID = ""
	design.Predecessor = ""
	design.CreatedUnix = 0
	design.Tasks = nil
	rawDesign, _ := json.Marshal(design)
	sum := sha256.Sum256(rawDesign)
	for i := range candidates {
		candidates[i].Source = hex.EncodeToString(sum[:])
	}
	snapshot, err := factory.PrioritySnapshot(candidates)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if current.PlanningCalls < 1 {
		auth.JSONError(w, 409, "planning_funding_required", "Authorize a finite planning call allowance before exploration.")
		return
	}
	reserved, err := s.Store.ReserveIntentJudgment(r.Context(), current, snapshot)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if !reserved {
		auth.JSONError(w, 409, "planning_unavailable", "Planning credit is exhausted or these inputs have already been consulted.")
		return
	}
	request := factory.PriorityRequest{Snapshot: snapshot, Candidates: candidates, State: rawDesign, Criteria: intentDesignCriteria(), Rubric: "Using the full supplied objective and reviewed choices, choose a bounded implementation approach. Respect an explicitly selected approach; choose clarify if a declined or corrected assumption contradicts the objective or material outcomes remain unresolved. Do not impose declined assumptions or invent product capabilities. This advice grants no acceptance, repository authority or spending permission."}
	chooser := control.TypeSafePriority{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel}
	answer, callErr := chooser.Choose(r.Context(), request)
	if callErr == nil {
		callErr = answer.Validate(request)
	}
	receipt := factory.PriorityReceipt{Snapshot: snapshot, Candidates: candidates, CreatedUnix: time.Now().Unix(), Outcome: "product_design", Selected: answer.Selected, Calls: []factory.PriorityCall{{Request: request, Answer: answer}}}
	if callErr != nil {
		receipt.Calls[0].Failure = "consultation_unavailable"
		receipt.Outcome = "failed"
	}
	if err = s.Store.RecordPriorityJudgment(r.Context(), receipt); err != nil {
		factoryCommandError(w, err)
		return
	}
	if callErr != nil {
		auth.JSONError(w, 503, "planning_unavailable", "Design consultation failed; the consumed allowance is retained.")
		return
	}
	proposal = applyIntentDesign(current, ids, proposal, answer)
	if answer.Selected == "clarify" || answer.Confidence < 0.25 {
		proposal.Tasks = []factory.IntentTask{}
		proposal.Ideas = append(proposal.Ideas, factory.ProductIdea{ID: ids[103], Description: "Describe the first user action and the result that should count as success", Consequence: "Implementation waits until this material product question is resolved.", Assumption: true, Decision: "pending"})
	}
	if err = proposal.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_proposal", err.Error())
		return
	}
	if err = s.Store.SaveIntentRevision(r.Context(), proposal); err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, proposal)
}

func intentDesignCriteria() map[string]any {
	return map[string]any{"complete_first": "One task per reviewed product choice combines implementation and verification outcomes", "incremental": "Two tasks per reviewed product choice separate implementation and dependent verification", "clarify": "No work graph until consequential scope contradictions or missing product outcomes are reviewed"}
}
func intentDesignCandidates(r factory.IntentRevision) []factory.PriorityCandidate {
	result := []factory.PriorityCandidate{}
	for _, approach := range []string{"complete_first", "incremental"} {
		if r.Approach != "" && r.Approach != approach {
			continue
		}
		result = append(result, factory.PriorityCandidate{ID: approach, Objective: r.Objective, Impact: intentDesignCriteria()[approach].(string)})
	}
	return append(result, factory.PriorityCandidate{ID: "clarify", Objective: r.Objective, Risk: intentDesignCriteria()["clarify"].(string)})
}
func applyIntentDesign(current factory.IntentRevision, ids []string, proposal factory.IntentRevision, answer factory.PriorityAnswer) factory.IntentRevision {
	if answer.Confidence < .25 || answer.Selected == "clarify" {
		return proposal
	}
	approach := current.Approach
	if approach == "" {
		approach = answer.Selected
	}
	current.Approach = approach
	derived := factory.ProposeIntent(current, ids)
	proposal.Approach, proposal.Tasks = derived.Approach, derived.Tasks
	return proposal
}
