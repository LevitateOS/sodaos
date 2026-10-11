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

	"github.com/levitateos/sodaos/internal/factory"
)

type priorityTransport func(*http.Request) (*http.Response, error)

func (f priorityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTypeSafePriorityUsesBoundedChoiceAndRejectsPartialAnswer(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "model.key")
	if err := os.WriteFile(secret, []byte("synthetic-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "partial"}[partial], func(t *testing.T) {
			calls := 0
			provider := &TypeSafePriority{APIKeyFile: secret, Client: &http.Client{Transport: priorityTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.String() != "https://api.typesafe.ai/v1/systemone" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer synthetic-secret" {
					t.Fatal("invalid provider boundary")
				}
				var body struct {
					State     factory.PriorityRequest `json:"state"`
					Questions map[string]struct {
						Type     string         `json:"type"`
						Criteria map[string]any `json:"criteria"`
					} `json:"questions"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Questions["priority"].Type != "choice" || len(body.Questions["priority"].Criteria) != 2 || len(body.State.Candidates) != 2 {
					t.Fatal("missing complete comparison state")
				}
				distribution := `"a":0,"b":1`
				if partial {
					distribution = `"b":1`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"model":"synthetic-model","answers":{"priority":{"type":"choice","choice":"b","confidence":1,"probabilities":{` + distribution + `}}},"usage":{"input_tokens":10,"output_tokens":3}}`))}, nil
			})}}
			a, err := provider.Choose(context.Background(), factory.PriorityRequest{Rubric: factory.PriorityRubric, Candidates: []factory.PriorityCandidate{{ID: "a"}, {ID: "b"}}})
			if calls != 1 || (!partial && (err != nil || a.Selected != "b" || a.InputTokens != 10)) || (partial && err == nil) {
				t.Fatalf("answer=%+v err=%v calls=%d", a, err, calls)
			}
		})
	}
}
