package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// TypeSafePriority uses a server-only secret file. It never retries requests:
// the finite journal reservation owns consultation count and attribution.
type TypeSafePriority struct {
	APIKeyFile string
	Model      string
	Client     *http.Client
}

func (p *TypeSafePriority) Choose(ctx context.Context, request factory.PriorityRequest) (factory.PriorityAnswer, error) {
	if len(request.Candidates) < 2 || len(request.Candidates) > 255 {
		return factory.PriorityAnswer{}, errors.New("invalid priority choice size")
	}
	key, err := os.ReadFile(p.APIKeyFile)
	if err != nil || len(bytes.TrimSpace(key)) == 0 {
		return factory.PriorityAnswer{}, ErrPriorityUnavailable
	}
	criteria := make(map[string]factory.PriorityCandidate, len(request.Candidates))
	for _, c := range request.Candidates {
		criteria[c.ID] = c
	}
	model := p.Model
	if model == "" {
		model = "jev-latest"
	}
	body, err := json.Marshal(map[string]any{"model": model, "state": request, "questions": map[string]any{"priority": map[string]any{"type": "choice", "instructions": request.Rubric, "criteria": criteria}}})
	if err != nil {
		return factory.PriorityAnswer{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.typesafe.ai/v1/systemone", bytes.NewReader(body))
	if err != nil {
		return factory.PriorityAnswer{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(key)))
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return factory.PriorityAnswer{}, ErrPriorityUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return factory.PriorityAnswer{}, ErrPriorityUnavailable
	}
	var out struct {
		Model   string `json:"model"`
		Answers map[string]struct {
			Type          string             `json:"type"`
			Choice        string             `json:"choice"`
			Confidence    float64            `json:"confidence"`
			Probabilities map[string]float64 `json:"probabilities"`
		} `json:"answers"`
		Usage struct {
			Input  int64 `json:"input_tokens"`
			Output int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&out); err != nil {
		return factory.PriorityAnswer{}, ErrPriorityUnavailable
	}
	a := out.Answers["priority"]
	answer := factory.PriorityAnswer{Selected: a.Choice, Confidence: a.Confidence, Probabilities: a.Probabilities, Model: out.Model, InputTokens: out.Usage.Input, OutputTokens: out.Usage.Output}
	if a.Type != "choice" {
		return answer, ErrPriorityUnavailable
	}
	return answer, answer.Validate(request)
}
