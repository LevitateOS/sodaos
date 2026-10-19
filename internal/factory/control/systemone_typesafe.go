package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// TypeSafeSystemOne exposes the three generic judgments. Results are advice;
// callers cannot substitute them for controller acceptance or native receipts.
type TypeSafeSystemOne struct {
	APIKeyFile string
	Model      string
	Client     *http.Client
}

func (p *TypeSafeSystemOne) Evaluate(ctx context.Context, kind string, state, instructions, criteria any) (map[string]any, error) {
	key, err := readPriorityKeyFile(p.APIKeyFile)
	if err != nil {
		return nil, errors.New("System One provider is not configured")
	}
	model := p.Model
	if model == "" {
		model = "jev-latest"
	}
	q := map[string]any{"type": kind, "instructions": instructions}
	if criteria != nil {
		q["criteria"] = criteria
	}
	body, err := json.Marshal(map[string]any{"model": model, "state": state, "questions": map[string]any{"judgment": q}})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.typesafe.ai/v1/systemone", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New("System One request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("System One provider refused the request")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil || len(raw) > 65536 {
		return nil, errors.New("System One response exceeds bound")
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return nil, errors.New("System One returned invalid JSON")
	}
	answers, _ := out["answers"].(map[string]any)
	answer, _ := answers["judgment"].(map[string]any)
	if answer["type"] != kind {
		return nil, errors.New("System One returned a different question type")
	}
	if err := validateSystemOneAnswer(kind, answer, criteria); err != nil {
		return nil, err
	}
	return map[string]any{"answer": answer, "model": out["model"], "usage": out["usage"]}, nil
}

func validateSystemOneAnswer(kind string, answer map[string]any, criteria any) error {
	invalid := errors.New("System One returned an invalid typed answer")
	switch kind {
	case "noul":
		value, ok := answer["noul"].(float64)
		if !ok || value < 0 || value > 1 {
			return invalid
		}
	case "choice":
		value, ok := answer["choice"].(string)
		var options map[string]json.RawMessage
		raw, err := json.Marshal(criteria)
		if err != nil || json.Unmarshal(raw, &options) != nil {
			return invalid
		}
		if _, present := options[value]; !ok || !present {
			return invalid
		}
	case "score":
		value, ok := answer["score"].(float64)
		var levels []json.RawMessage
		raw, err := json.Marshal(criteria)
		if err != nil || json.Unmarshal(raw, &levels) != nil || len(levels) < 2 || !ok || value < 0 || value > float64(len(levels)-1) {
			return invalid
		}
	default:
		return invalid
	}
	if confidence, present := answer["confidence"]; present {
		value, ok := confidence.(float64)
		if !ok || value < 0 || value > 1 {
			return invalid
		}
	}
	return nil
}
