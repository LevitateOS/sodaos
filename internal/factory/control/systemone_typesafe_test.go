package control

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenericSystemOneProviderUsesCurrentAPI(t *testing.T) {
	key := filepath.Join(t.TempDir(), "jev.key")
	if err := os.WriteFile(key, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"noul", "choice", "score"} {
		t.Run(kind, func(t *testing.T) {
			var criteria any
			value := `"noul":0.8`
			if kind == "choice" {
				criteria = map[string]any{"a": "First", "b": "Second"}
				value = `"choice":"a","confidence":0.8`
			}
			if kind == "score" {
				criteria = []string{"Low", "High"}
				value = `"score":0.8`
			}
			p := &TypeSafeSystemOne{APIKeyFile: key, Client: &http.Client{Transport: priorityTransport(func(r *http.Request) (*http.Response, error) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				questions := body["questions"].(map[string]any)
				q := questions["judgment"].(map[string]any)
				if r.URL.String() != "https://api.typesafe.ai/v1/systemone" || r.Header.Get("Authorization") != "Bearer synthetic" || q["type"] != kind || q["instructions"] != "Assess" {
					t.Fatal("wrong provider contract")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"model":"fixture","answers":{"judgment":{"type":"` + kind + `",` + value + `}},"usage":{"input_tokens":3}}`))}, nil
			})}}
			if _, err := p.Evaluate(context.Background(), kind, "known", "Assess", criteria); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGenericSystemOneRefusesInvalidTypedAnswers(t *testing.T) {
	for _, c := range []struct {
		kind     string
		answer   map[string]any
		criteria any
	}{
		{"noul", map[string]any{}, nil},
		{"noul", map[string]any{"noul": 1.2}, nil},
		{"choice", map[string]any{"choice": "invented"}, map[string]string{"a": "First", "b": "Second"}},
		{"choice", map[string]any{"choice": "a", "confidence": -0.1}, map[string]string{"a": "First", "b": "Second"}},
		{"score", map[string]any{"score": 2.0}, []string{"Low", "High"}},
		{"score", map[string]any{"score": "0.5"}, []string{"Low", "High"}},
	} {
		if err := validateSystemOneAnswer(c.kind, c.answer, c.criteria); err == nil {
			t.Fatalf("invalid %s answer admitted: %+v", c.kind, c.answer)
		}
	}
}
func TestLiveGenericSystemOnePrimitives(t *testing.T) {
	if os.Getenv("SODA_LIVE_SYSTEMONE") != "1" {
		t.Skip("live SystemOne not requested")
	}
	key := os.Getenv("TYPESAFE_API_KEY")
	if key == "" {
		t.Fatal("provider key unavailable")
	}
	path := filepath.Join(t.TempDir(), "jev.key")
	if err := os.WriteFile(path, []byte(key), 0600); err != nil {
		t.Fatal(err)
	}
	p := &TypeSafeSystemOne{APIKeyFile: path}
	cases := []struct {
		kind     string
		question string
		criteria any
	}{{"noul", "Does the supplied text explicitly say the check passed?", nil}, {"choice", "Which result is explicitly recorded?", map[string]any{"passed": "The check passed", "failed": "The check failed", "unknown": "No result"}}, {"score", "How many of the two checks are explicitly recorded as passed?", []string{"Neither check passed", "One check passed", "Both checks passed"}}}
	for _, c := range cases {
		result, err := p.Evaluate(context.Background(), c.kind, map[string]any{"text": "Check A passed. Check B passed."}, c.question, c.criteria)
		if err != nil {
			t.Fatal(err)
		}
		answer := result["answer"].(map[string]any)
		if answer["type"] != c.kind {
			t.Fatal("wrong type")
		}
		switch c.kind {
		case "noul":
			if n, ok := answer["noul"].(float64); !ok || n < 0 || n > 1 {
				t.Fatal("invalid noul")
			}
		case "choice":
			if answer["choice"] != "passed" {
				t.Fatal("wrong known choice")
			}
		case "score":
			if n, ok := answer["score"].(float64); !ok || n < 0 || n > 2 {
				t.Fatal("invalid score")
			}
		}
		t.Logf("%s returned typed answer", c.kind)
	}
}
