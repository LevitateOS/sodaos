package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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

// exploreData carries the one consolidated exploration beside the revision.
// Older readers ignore it; the revision fields stay flat and unchanged.
type exploreData struct {
	Content        factory.ProposalContent       `json:"content"`
	Challenges     []factory.AssumptionChallenge `json:"challenges"`
	Settled        []string                      `json:"settled"`
	Recommendation *factory.ConsolidatedProposal `json:"recommendation,omitempty"`
	Consistency    *factory.ConsistencyReport    `json:"consistency,omitempty"`
}

type exploreResponse struct {
	factory.IntentRevision
	Exploration exploreData `json:"exploration"`
}

func attachExploration(proposal factory.IntentRevision, content factory.ProposalContent, challenges []factory.AssumptionChallenge, settled []string, recommendation *factory.ConsolidatedProposal, consistency *factory.ConsistencyReport) exploreResponse {
	if challenges == nil {
		challenges = []factory.AssumptionChallenge{}
	}
	if settled == nil {
		settled = []string{}
	}
	return exploreResponse{IntentRevision: proposal, Exploration: exploreData{Content: content, Challenges: challenges, Settled: settled, Recommendation: recommendation, Consistency: consistency}}
}

func settledIDs(review factory.AssumptionReview) []string {
	ids := make([]string, 0, len(review.Settled))
	for _, settlement := range review.Settled {
		ids = append(ids, settlement.Idea)
	}
	return ids
}

const designRubric = "Using the full supplied objective and reviewed choices, choose a bounded implementation approach. Respect an explicitly selected approach; choose clarify if a declined or corrected assumption contradicts the objective or material outcomes remain unresolved. Do not impose declined assumptions or invent product capabilities. This advice grants no acceptance, repository authority or spending permission."

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
	content, err := factory.ComposeProposalContent(current)
	if err != nil {
		factoryCommandError(w, err)
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
	chooser := control.TypeSafePriority{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel}
	coord := &control.Coordinator{Store: s.Store, Planning: &control.PlanningJudgments{Chooser: &chooser}}
	candidates := intentDesignCandidates(current)
	design := current
	design.ID = ""
	design.Predecessor = ""
	design.CreatedUnix = 0
	design.Tasks = nil
	available, err := s.Store.PlanningAllowanceAvailable(r.Context(), current.Intent, v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if !available {
		auth.JSONError(w, 409, "planning_funding_required", "Authorize a finite planning call allowance before exploration.")
		return
	}
	answer, _, callErr := coord.AdvisePurpose(r.Context(), control.PurposeAdvice{
		Purpose:    factory.JudgmentProductApproaches,
		State:      design,
		Question:   designRubric,
		Options:    intentDesignCriteria(),
		Outcome:    "product_design",
		Candidates: candidates,
		Reserve: func(ctx context.Context, snapshot string) (bool, error) {
			return s.Store.ReserveIntentJudgment(ctx, current, snapshot)
		},
	})
	if callErr != nil {
		if errors.Is(callErr, control.ErrPriorityUnavailable) {
			auth.JSONError(w, 503, "planning_unavailable", "Design consultation failed; the consumed allowance is retained.")
			return
		}
		factoryCommandError(w, callErr)
		return
	}
	proposal = applyIntentDesign(current, ids, proposal, answer)
	if answer.Selected == "clarify" || answer.Confidence < 0.25 {
		proposal.Tasks = []factory.IntentTask{}
		proposal = appendClarifyOnce(proposal, ids[103])
	}
	review, err := coord.JudgeAssumptions(r.Context(), current, content, proposal.Ideas)
	if err != nil {
		if errors.Is(err, control.ErrPriorityUnavailable) {
			auth.JSONError(w, 503, "planning_unavailable", "Assumption review failed; the consumed allowance is retained.")
			return
		}
		factoryCommandError(w, err)
		return
	}
	proposal = factory.SettleAssumptions(proposal, review)
	var recommendation *factory.ConsolidatedProposal
	if content.Supported {
		consolidated, err := coord.JudgeConsolidation(r.Context(), current, content)
		if err != nil {
			if errors.Is(err, control.ErrPriorityUnavailable) {
				auth.JSONError(w, 503, "planning_unavailable", "Consolidation failed; the consumed allowance is retained.")
				return
			}
			factoryCommandError(w, err)
			return
		}
		recommendation = &consolidated
	}
	var consistency *factory.ConsistencyReport
	if recommendation != nil {
		verified, err := coord.JudgeConsistency(r.Context(), current, content, *recommendation)
		if err != nil {
			if errors.Is(err, control.ErrPriorityUnavailable) {
				auth.JSONError(w, 503, "planning_unavailable", "Consistency review failed; the consumed allowance is retained.")
				return
			}
			factoryCommandError(w, err)
			return
		}
		consistency = &verified
	}
	if err = proposal.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_proposal", err.Error())
		return
	}
	if err = s.Store.SaveIntentRevision(r.Context(), proposal); err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 201, attachExploration(proposal, content, review.Challenges, settledIDs(review), recommendation, consistency))
}

// clarifyDescription and clarifyConsequence pin the one material product
// question blocking implementation. Matching reuses these exact texts so
// the carried-forward clarification keeps a stable identity and answers
// can settle instead of spawning a duplicate each round.
const (
	clarifyDescription = "Describe the first user action and the result that should count as success"
	clarifyConsequence = "Implementation waits until this material product question is resolved."
)

// clarifyIdea asks the one material product question blocking implementation.
// It carries autonomous assumption basis, never human approval.
func clarifyIdea(id string) factory.ProductIdea {
	return factory.ProductIdea{ID: id, Description: clarifyDescription, Consequence: clarifyConsequence, Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true}
}

// isClarification reports whether idea is the producer-owned material
// question, pending or already answered. It matches assumption-source
// ideas carrying the exact pinned texts; ordinary human ideas never
// match, so no human decision is coalesced or rewritten.
func isClarification(idea factory.ProductIdea) bool {
	return idea.Assumption && idea.Source == "assumption" && idea.Material &&
		idea.Description == clarifyDescription && idea.Consequence == clarifyConsequence
}

// appendClarifyOnce reuses the carried-forward producer-owned
// clarification for the unchanged material question. A pending one keeps
// its stable identity so challenges and answers settle on it; an
// answered one preserves the recorded answer. Either way no duplicate is
// appended.
func appendClarifyOnce(proposal factory.IntentRevision, id string) factory.IntentRevision {
	for _, idea := range proposal.Ideas {
		if isClarification(idea) {
			return proposal
		}
	}
	proposal.Ideas = append(proposal.Ideas, clarifyIdea(id))
	return proposal
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
