package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
)

// PriorityCandidate contains observed evidence only. Empty impact, risk and
// effort mean unknown; the classifier must not manufacture estimates.
type PriorityCandidate struct {
	ID                  string         `json:"id"`
	Control             IssueControl   `json:"control"`
	Objective           string         `json:"objective"`
	Acceptance          Acceptance     `json:"acceptance"`
	AcceptedSections    []PromptSource `json:"accepted_sections,omitempty"`
	Authority           AuthorityRef   `json:"authority"`
	Preparation         string         `json:"preparation"`
	PreparationRevision int64          `json:"preparation_revision"`
	Source              string         `json:"source"`
	GateRevision        int64          `json:"gate_revision"`
	PlannedMinutes      int            `json:"planned_minutes"`
	PromptDigest        string         `json:"prompt_digest"`
	Impact              string         `json:"impact,omitempty"`
	Risk                string         `json:"risk,omitempty"`
	Effort              string         `json:"effort,omitempty"`
}

const PriorityRubric = "Choose the eligible issue whose next execution best advances its accepted objective. Prefer evidenced prerequisite leverage and material user impact, then reduce evidenced delivery risk and effort. Use only provided facts; missing impact, risk and effort are unknown, not zero. Queue age breaks equivalent preferences. Eligibility, acceptance, authority, spending and launch are decided by code, never by this preference."

type PriorityRequest struct {
	Snapshot   string              `json:"snapshot"`
	Candidates []PriorityCandidate `json:"candidates"`
	Rubric     string              `json:"rubric"`
}

type PriorityAnswer struct {
	Selected      string             `json:"selected"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
	Model         string             `json:"model"`
	InputTokens   int64              `json:"input_tokens"`
	OutputTokens  int64              `json:"output_tokens"`
}

type PriorityCall struct {
	Request PriorityRequest `json:"request"`
	Answer  PriorityAnswer  `json:"answer"`
	Failure string          `json:"failure,omitempty"`
}

type PriorityReceipt struct {
	Snapshot    string              `json:"snapshot"`
	Candidates  []PriorityCandidate `json:"candidates"`
	Calls       []PriorityCall      `json:"calls"`
	Selected    string              `json:"selected,omitempty"`
	Outcome     string              `json:"outcome"`
	CreatedUnix int64               `json:"created_unix"`
}

// PrioritySnapshot binds the complete candidate evidence and current rubric.
func PrioritySnapshot(candidates []PriorityCandidate) (string, error) {
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c.ID == "" || seen[c.ID] {
			return "", errors.New("invalid priority candidate identities")
		}
		seen[c.ID] = true
	}
	data, err := json.Marshal(struct {
		Rubric     string
		Candidates []PriorityCandidate
	}{PriorityRubric, candidates})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Validate checks coverage and normalization, not the model's factual accuracy.
func (a PriorityAnswer) Validate(request PriorityRequest) error {
	if len(request.Candidates) < 2 || len(request.Candidates) > 255 || a.Model == "" ||
		math.IsNaN(a.Confidence) || a.Confidence < 0 || a.Confidence > 1 ||
		a.InputTokens < 0 || a.OutputTokens < 0 || len(a.Probabilities) != len(request.Candidates) {
		return errors.New("invalid priority answer")
	}
	sum, selected := 0.0, false
	for _, c := range request.Candidates {
		p, ok := a.Probabilities[c.ID]
		if !ok || math.IsNaN(p) || p < 0 || p > 1 {
			return errors.New("incomplete priority distribution")
		}
		sum += p
		selected = selected || a.Selected == c.ID
	}
	if !selected || math.Abs(sum-1) > 0.001 {
		return errors.New("invalid priority selection")
	}
	for _, p := range a.Probabilities {
		if p > a.Probabilities[a.Selected]+0.000001 {
			return errors.New("priority choice is not maximal")
		}
	}
	return nil
}
